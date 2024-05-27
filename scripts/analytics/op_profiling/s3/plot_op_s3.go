// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

//usr/bin/env go run $0; exit

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	xmath "github.com/Fantom-foundation/Aida/utils/math"

	// db
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	//echart
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const (
	// db
	first                        int    = 0
	last                         int    = 65_436_418
	logLevel                     string = "Debug"
	connection                   string = "/home/rapolt/dev/sqlite3/test.db"
	sqlite3_SelectFromOperations string = `
		SELECT start, end, opId, opName, count, sum, mean, min, max
		FROM operations 
		WHERE start=:start AND end=:end AND count > 0;
	`
	sqlite3_SelectDistinctOps string = ` 
		SELECT DISTINCT opId, opName
		FROM operations
		ORDER BY opId ASC;
	`

	workerCount int = 10
	bucketCount int = 654
	opCount     int = 50

	// report
	pHtml = "report_op_s3.html"
)

type query struct {
	Start  int `db:"start"`
	End    int `db:"end"`
	bucket int
}

type txResponse struct {
	Start  int     `db:"start"`
	End    int     `db:"end"`
	OpId   int     `db:"opId"`
	OpName string  `db:"opName"`
	Count  int     `db:"count"`
	Sum    float64 `db:"sum"`
	Mean   float64 `db:"mean"`
	Min    float64 `db:"min"`
	Max    float64 `db:"max"`
}

type opLookupResponse struct {
	OpId   int    `db:"opId"`
	OpName string `db:"opName"`
}

type bucketMsg struct {
	bucket int
	count  int
	time   float64
}

type opMsg struct {
	bucket int
	opid   int
	count  int
	time   float64
	avg    float64
	min    float64
	max    float64
}

func worker(id int, opCount int, qc <-chan query, bc chan<- bucketMsg, oc chan<- opMsg, ec chan<- error) {

	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		ec <- err
	}

	stmt, err := db.PrepareNamed(sqlite3_SelectFromOperations)
	if err != nil {
		ec <- err
	}

	log := logger.NewLogger(logLevel, fmt.Sprintf("Plot OP S3 Worker #%d", id))

	defer func() {
		stmt.Close()
		db.Close()

		log.Debugf("Worker #%d terminated.", id)
	}()

	for q := range qc {
		log.Debugf("Starting: %v", q)

		txs := []txResponse{}
		stmt.Select(&txs, q)

		var (
			count       int             = 0
			time        float64         = 0
			countByOpId map[int]int     = make(map[int]int, opCount)
			timeByOpId  map[int]float64 = make(map[int]float64, opCount)
			meanByOpId  map[int]float64 = make(map[int]float64, opCount)
			minByOpId   map[int]float64 = make(map[int]float64, opCount)
			maxByOpId   map[int]float64 = make(map[int]float64, opCount)
		)

		for _, tx := range txs {
			id := tx.OpId
			count += tx.Count
			time += tx.Sum
			countByOpId[id] += tx.Count
			timeByOpId[id] += tx.Sum
			meanByOpId[id] = tx.Mean
			minByOpId[id] = xmath.Min(tx.Min, minByOpId[id])
			maxByOpId[id] = xmath.Max(tx.Max, maxByOpId[id])
		}

		bc <- bucketMsg{q.bucket, count, time}

		for id := 0; id < opCount; id++ {
			oc <- opMsg{
				q.bucket,
				id,
				countByOpId[id],
				timeByOpId[id],
				meanByOpId[id],
				minByOpId[id],
				maxByOpId[id],
			}
		}

		log.Debugf("Done: %v", q)

	}
}

func lookupOperations(connection string, selectDistinct string) ([]int, map[int]string, error) {
	var (
		opIds        []int          = []int{}
		opNameByOpId map[int]string = map[int]string{}
	)

	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		return nil, nil, err
	}

	opls := []opLookupResponse{}
	err = db.Select(&opls, selectDistinct)
	if err != nil {
		return nil, nil, err
	}

	for _, opl := range opls {
		opIds = append(opIds, opl.OpId)
		opNameByOpId[opl.OpId] = opl.OpName
	}

	db.Close()
	return opIds, opNameByOpId, nil
}

func main() {

	start := time.Now()

	var (
		interval int           = 100_000
		buckets  []int         = make([]int, bucketCount)
		log      logger.Logger = logger.NewLogger(logLevel, "Plot F1")

		opIds        []int          = []int{}
		opNameByOpId map[int]string = map[int]string{}

		countTotal          int                     = 0
		timeTotal           float64                 = 0
		countByBucket       map[int]float64         = make(map[int]float64, bucketCount)
		timeByBucket        map[int]float64         = make(map[int]float64, bucketCount)
		countByOpId         map[int]float64         = make(map[int]float64, bucketCount)
		timeByOpId          map[int]float64         = make(map[int]float64, bucketCount)
		countByBucketByOpId map[int]map[int]float64 = map[int]map[int]float64{}
		timeByBucketByOpId  map[int]map[int]float64 = map[int]map[int]float64{}
		meanByBucketByOpId  map[int]map[int]float64 = map[int]map[int]float64{}
		minByBucketByOpId   map[int]map[int]float64 = map[int]map[int]float64{}
		maxByBucketByOpId   map[int]map[int]float64 = map[int]map[int]float64{}
	)

	for b := range buckets {
		countByBucketByOpId[b] = map[int]float64{}
		timeByBucketByOpId[b] = map[int]float64{}
		meanByBucketByOpId[b] = map[int]float64{}
		minByBucketByOpId[b] = map[int]float64{}
		maxByBucketByOpId[b] = map[int]float64{}
	}

	log.Infof("Bucket: %d, Interval: %d, Worker: %d", bucketCount, interval, workerCount)

	opIds, opNameByOpId, err := lookupOperations(connection, sqlite3_SelectDistinctOps)

	log.Infof("opCount: %d", len(opIds))

	qc := make(chan query, bucketCount)
	bc := make(chan bucketMsg, bucketCount)
	oc := make(chan opMsg, opCount*bucketCount)
	ec := make(chan error, 1)

	var (
		qWg sync.WaitGroup
		bWg sync.WaitGroup
		oWg sync.WaitGroup
		eWg sync.WaitGroup
	)

	// monitor for error when querying db, close all channels + terminate if found.
	go func() {
		for e := range ec {
			fmt.Println("Received an error: ", e)

			close(qc)
			close(bc)
			close(oc)
			close(ec)

			qWg.Wait()
			bWg.Wait()
			oWg.Wait()
			eWg.Wait()

			os.Exit(1)
		}
	}()

	// start multiple threads to query DB
	for w := 0; w < workerCount; w++ {
		qWg.Add(1)
		go func(id int) {
			defer qWg.Done()
			worker(id, 50, qc, bc, oc, ec)
		}(w)
	}

	// start a thread to digest bucket-wise response from DB
	for w := 0; w < 1; w++ { // just in case this becomes a bottleneck
		bWg.Add(1)
		go func() {
			defer bWg.Done()
			for m := range bc {
				countTotal += m.count
				countByBucket[m.bucket] += float64(m.count)
				timeTotal += m.time
				timeByBucket[m.bucket] += m.time
			}
		}()
	}

	// start a thread to digest operation-wise response from DB
	for w := 0; w < 1; w++ { // just in case this becomes a bottleneck
		oWg.Add(1)
		go func() {
			defer oWg.Done()
			for m := range oc {
				countByOpId[m.opid] += float64(m.count)
				timeByOpId[m.opid] += m.time
				countByBucketByOpId[m.opid][m.bucket] += float64(m.count)
				timeByBucketByOpId[m.opid][m.bucket] += m.time
				minByBucketByOpId[m.opid][m.bucket] = xmath.Min(m.min, minByBucketByOpId[m.opid][m.bucket])
				maxByBucketByOpId[m.opid][m.bucket] = xmath.Max(m.max, maxByBucketByOpId[m.opid][m.bucket])

				if m.count > 0 {
					meanByBucketByOpId[m.opid][m.bucket] = timeByBucketByOpId[m.opid][m.bucket] / countByBucketByOpId[m.opid][m.bucket]
				}
			}
		}()
	}

	// generate queries here
	itv := utils.NewInterval(uint64(first), uint64(last), uint64(interval))

	qc <- query{int(0), int(100000), 0} // first insert was wrong on the DB side
	itv.Next()

	for b := 1; b < bucketCount; b, itv = b+1, itv.Next() {
		q := query{int(itv.Start() + 1), int(itv.End() + 1), b}
		buckets[b] = int(itv.Start())
		qc <- q
	}

	close(qc)
	qWg.Wait()

	close(ec) // no more error
	eWg.Wait()

	log.Infof("queries - time taken: %f s", time.Since(start).Seconds())

	close(bc)
	bWg.Wait()

	close(oc)
	oWg.Wait()

	log.Infof("postprocessing - time taken: %f s", time.Since(start).Seconds())

	// Charts start here
	page := components.NewPage().AddCharts(
		PieWithTitle(
			pie("Count By OpId", opIds, countByOpId, opNameByOpId),
			"Percentage By Count", "",
		),
		BarWithTitle(
			BarWithCustomXy(
				bar("Count By Interval", buckets, countByBucket),
				"Block Height",
				"Count", "",
			),
			"Total Op Count / 100,000 Blocks", "",
		),
		stackedBar("Percentage", buckets, countByBucket, opIds, countByOpId, countByBucketByOpId, opNameByOpId),
		PieWithTitle(
			pie("Time By OpId", opIds, timeByOpId, opNameByOpId),
			"Percentage By Runtime", "",
		),
		BarWithTitle(
			BarWithCustomXy(
				bar("Time By Interval", buckets, timeByBucket),
				"Block Height",
				"Time", "μs",
			),
			"Total Op Runtime  / 100,000 Blocks", "",
		),
		stackedBar("Percentage", buckets, timeByBucket, opIds, timeByOpId, timeByBucketByOpId, opNameByOpId),
	)

	for _, op := range opIds {
		if countByOpId[op] == 0 {
			continue
		}

		page.AddCharts(
			ScatterWithTitle(
				ScatterWithCustomXy(
					scatter("Count", buckets, countByBucketByOpId[op]),
					"Block Height",
					"Count", "",
				),
				fmt.Sprintf("[%d]%s Total Call Count", op, opNameByOpId[op]), "",
			),
			ScatterWithTitle(
				ScatterWithCustomXy(
					scatter("Total Time", buckets, timeByBucketByOpId[op]),
					"Block Height",
					"Time", "μs",
				),
				fmt.Sprintf("[%d]%s Total Runtime", op, opNameByOpId[op]), "",
			),
			ScatterWithTitle(
				ScatterWithCustomXy(
					scatter("Avg. Time", buckets, meanByBucketByOpId[op]),
					"Block Height",
					"Time", "μs",
				),
				fmt.Sprintf("[%d]%s Average Time / 1 Call", op, opNameByOpId[op]), "",
			),
		)
	}

	f, err := os.Create(pHtml)
	if err != nil {
		fmt.Println("Rendered to ", pHtml)
		os.Exit(1)
	}

	page.Render(io.MultiWriter(f))
	log.Infof("Rendered to %s", pHtml)
}

func BarWithTitle(b *charts.Bar, title string, subtitle string) *charts.Bar {
	b.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}),
	)
	return b
}

func BarWithCustomXy(b *charts.Bar, x string, y string, yu string) *charts.Bar {
	b.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: x,
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    "{value}",
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
			SplitLine: &opts.SplitLine{
				Show: true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: y,
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    fmt.Sprintf("{value} %s", yu),
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
		}),
	)
	return b
}

func bar(title string, buckets []int, byBucket map[int]float64) *charts.Bar {
	var y []opts.BarData = make([]opts.BarData, len(buckets))

	for b := range buckets {
		y[b] = opts.BarData{
			Value: byBucket[b],
		}
	}

	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)

	bar.SetXAxis(buckets).AddSeries(title, y)

	return bar
}

func stackedBar(title string, buckets []int, byBucket map[int]float64, opIds []int, byOpId map[int]float64, byBucketByOpIds map[int]map[int]float64, opNameByOpId map[int]string) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Block Height",
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    "{value}",
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
			SplitLine: &opts.SplitLine{
				Show: true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Percentage",
			Max:  1.0,
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    fmt.Sprintf("{value}"),
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:         true,
			SelectedMode: "false",
			Orient:       "vertical",
			X:            "right",
			Y:            "center",
		}),
		charts.WithGridOpts(opts.Grid{
			Right: "18%",
		}),
	)

	var sortedOpIds []int
	for _, id := range opIds {
		sortedOpIds = append(sortedOpIds, id)
	}

	sort.Slice(sortedOpIds, func(i, j int) bool {
		return byOpId[sortedOpIds[i]] < byOpId[sortedOpIds[j]]
	})

	var x int = len(opIds) - 10
	bar.SetXAxis(buckets)

	var others []opts.BarData = make([]opts.BarData, len(buckets))
	for b := range buckets {
		var val float64 = 0
		for _, id := range sortedOpIds[:x] {
			val += float64(byBucketByOpIds[id][b])
		}
		others[b] = opts.BarData{
			Value: val / byBucket[b],
		}
	}
	bar.AddSeries("Others", others)

	for _, id := range sortedOpIds[x:] {
		var y []opts.BarData = make([]opts.BarData, len(buckets))
		for b := range buckets {
			y[b] = opts.BarData{
				Value: float64(byBucketByOpIds[id][b]) / byBucket[b],
			}
		}
		bar.AddSeries(opNameByOpId[id], y)
	}

	bar.SetSeriesOptions(
		charts.WithBarChartOpts(opts.BarChart{
			Stack: title,
		}),
	)

	return bar
}

func ScatterWithTitle(s *charts.Scatter, title string, subtitle string) *charts.Scatter {
	s.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}),
	)
	return s
}

func ScatterWithCustomXy(s *charts.Scatter, x string, y string, yu string) *charts.Scatter {
	s.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: x,
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    "{value}",
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
			SplitLine: &opts.SplitLine{
				Show: true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: y,
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Formatter:    fmt.Sprintf("{value} %s", yu),
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
		}),
	)
	return s
}

func scatter(title string, buckets []int, byBucket map[int]float64) *charts.Scatter {
	var y []opts.ScatterData = make([]opts.ScatterData, len(buckets))

	for b := range buckets {
		y[b] = opts.ScatterData{
			Value:      byBucket[b],
			Symbol:     "circle",
			SymbolSize: 5,
		}
	}

	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)
	scatter.SetXAxis(buckets).AddSeries(title, y)

	return scatter
}

func line(title string, buckets []int, byBucket map[int]float64) *charts.Line {
	var y []opts.LineData = make([]opts.LineData, len(buckets))

	for b := range buckets {
		y[b] = opts.LineData{Value: byBucket[b]}
	}

	line := charts.NewLine()
	line.SetXAxis(buckets).AddSeries(title, y)

	return line
}

func pie(title string, opIds []int, byOpId map[int]float64, opNameByOpId map[int]string) *charts.Pie {
	var items []opts.PieData = make([]opts.PieData, len(opIds))

	for ix, opId := range opIds {
		if byOpId[opId] == 0 {
			continue
		}
		items[ix] = opts.PieData{
			Value: byOpId[opId],
			Name:  opNameByOpId[opId],
		}
	}

	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)
	pie.AddSeries(title, items).SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:      true,
			Formatter: "{b} {d}%",
		}),
	)
	return pie
}

func PieWithTitle(p *charts.Pie, title string, subtitle string) *charts.Pie {
	p.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}),
	)
	return p
}
