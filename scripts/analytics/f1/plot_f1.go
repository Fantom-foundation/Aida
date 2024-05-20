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

	"golang.org/x/exp/constraints"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/scripts/analytics/html"
	"github.com/Fantom-foundation/Aida/utils"

	// db
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	//echart
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type Number interface {
	constraints.Integer | constraints.Float
}

const (
	// db
	first      int    = 479_327
	last       int    = 22_832_168
	logLevel   string = "Debug"
	connection string = "/var/opera/Aida/tmp-rapolt/register/s5-f1.db"

	workerCount int = 10
	bucketCount int = 223

	// report
	pHtml = "report_f1.html"
)

// DB-related const
const (
	sqlite3SelectFromStats string = `
		SELECT start, end, memory, disk, tx_rate, gas_rate, overall_tx_rate, overall_gas_rate
		FROM stats
		WHERE start>=:start AND end<=:end;
	`
)

type query struct {
	Start  int `db:"start"`
	End    int `db:"end"`
	bucket int
}

type statsResponse struct {
	Start          int     `db:"start"`
	End            int     `db:"end"`
	Memory         int     `db:"memory"`
	Disk           int     `db:"disk"`
	TxRate         float64 `db:"tx_rate"`
	GasRate        float64 `db:"gas_rate"`
	OverallTxRate  float64 `db:"overall_tx_rate"`
	OverallGasRate float64 `db:"overall_gas_rate"`
}

type bucketMsg struct {
	bucket         int
	memory         int
	disk           int
	txRate         float64
	gasRate        float64
	overallTxRate  float64
	overallGasRate float64
}

func worker(id int, qc <-chan query, bc chan<- bucketMsg, ec chan<- error) {
	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		ec <- err
	}

	stmt, err := db.PrepareNamed(sqlite3SelectFromStats)
	if err != nil {
		ec <- err
	}

	log := logger.NewLogger(logLevel, fmt.Sprintf("Plot F1 Worker #%d", id))

	defer func() {
		stmt.Close()
		db.Close()

		log.Debugf("Worker #%d terminated.", id)
	}()

	for q := range qc {
		log.Debugf("Starting: %v", q)

		stats := []statsResponse{}
		stmt.Select(&stats, q)

		// for some reason, cannot loop over stats if length == 1
		// complains that Memory, Disk, etc. is undefined.
		bc <- bucketMsg{
			q.bucket,
			stats[0].Memory,
			stats[0].Disk,
			stats[0].TxRate,
			stats[0].GasRate,
			stats[0].OverallTxRate,
			stats[0].OverallGasRate,
		}

		log.Debugf("Done: %v", q)
	}
}

func main() {

	start := time.Now()

	var (
		interval int           = 100_000
		buckets  []int         = make([]int, bucketCount)
		log      logger.Logger = logger.NewLogger(logLevel, "Plot F1")

		memoryByBucket         map[int]int     = make(map[int]int, bucketCount)
		diskByBucket           map[int]int     = make(map[int]int, bucketCount)
		txRateByBucket         map[int]float64 = make(map[int]float64, bucketCount)
		gasRateByBucket        map[int]float64 = make(map[int]float64, bucketCount)
		overallTxRateByBucket  map[int]float64 = make(map[int]float64, bucketCount)
		overallGasRateByBucket map[int]float64 = make(map[int]float64, bucketCount)
	)

	log.Noticef("Bucket: %d, Interval: %d, Worker: %d", bucketCount, interval, workerCount)

	qc := make(chan query, bucketCount)
	bc := make(chan bucketMsg, bucketCount)
	ec := make(chan error, 1)

	var (
		qWg sync.WaitGroup
		bWg sync.WaitGroup
		eWg sync.WaitGroup
	)

	// start a thread to monitor for error when querying db, close all channels + terminate if found.
	eWg.Add(1)
	go func() {
		defer eWg.Done()
		for e := range ec {
			log.Errorf("Received an error: %v", e)

			close(qc)
			close(bc)
			close(ec)

			qWg.Wait()
			bWg.Wait()
			eWg.Wait()

			os.Exit(1)
		}
	}()

	// start multiple threads to query DB
	for w := 0; w < workerCount; w++ {
		qWg.Add(1)
		go func(id int) {
			defer qWg.Done()
			worker(id, qc, bc, ec)
		}(w)
	}

	// start a thread to digest bucket-wise response from DB
	for w := 0; w < 1; w++ { // just in case this becomes a bottleneck
		bWg.Add(1)
		go func() {
			defer bWg.Done()
			for m := range bc {
				memoryByBucket[m.bucket] += m.memory
				diskByBucket[m.bucket] += m.disk
				txRateByBucket[m.bucket] += m.txRate
				gasRateByBucket[m.bucket] += m.gasRate
				overallTxRateByBucket[m.bucket] += m.overallTxRate
				overallGasRateByBucket[m.bucket] += m.overallGasRate
			}
		}()
	}

	// generate queries here
	itv := utils.NewInterval(uint64(first), uint64(last), uint64(interval))
	for b := 0; b < bucketCount; b, itv = b+1, itv.Next() {
		q := query{int(itv.Start()), int(itv.End()), b}
		buckets[b] = int(itv.Start())
		qc <- q
	}

	close(qc)
	qWg.Wait()

	close(ec) // no more error
	eWg.Wait()

	log.Noticef("queries - time taken: %f s", time.Since(start).Seconds())

	close(bc)
	bWg.Wait()

	log.Noticef("postprocessing - time taken: %f s", time.Since(start).Seconds())

	// Charts start here
	f, err := os.Create(pHtml)
	if err != nil {
		log.Errorf("Unable to create html documents at %s", pHtml)
		os.Exit(1)
	}

	writer := io.MultiWriter(f)

	// Report header
	writer.Write(html.Div(
		html.H1("F1 - Functional Correctness of LiveDB using Testnet"),
		html.P(time.Now().Format("2006-01-02")),
	))

	// Experimental setup
	var (
		machine     string = "wasuwee-x249(65.109.70.227)"
		cpu         string = "AMD Ryzen 9 5950X 16-Core Processor"
		ram         string = "125GB RAM"
		disk        string = "Samsung Electronics Disk, WDC WUH721816AL, Samsung Electronics Disk, WDC WUH721816AL"
		os          string = "Agent pid 1400011 Ubuntu 22.04.2 LTS"
		goVersion   string = "go1.21.1 linux/amd64"
		aidaVersion string = "81703de9537bb746c1e4e67c51b9fcae3f89e1e8"
		stateDbType string = "carmen(go-file 5)"
		vmType      string = "lfvm"
		dbPath      string = connection
	)

	writer.Write(html.Div(
		html.H2("1. Experimental Setup"),
		html.P(`The experiment is run on the machine <b>%s</b> - CPU: <b>%s</b>, Ram: <b>%s</b>, Disk: <b>%s</b>.`, machine, cpu, ram, disk),
		html.P(`The operating system is <b>%s</b>. The system has installed go version <b>%s</b>`, os, goVersion),
		html.P(`The github hash of the Aida repository is <b>%s</b>. For this experiment, we use <b>%s</b> as a StateDB and <b>%s</b> as a virtual machine. The profiling result for this experiment is stored in the database <b>%s</b>.`, aidaVersion, stateDbType, vmType, dbPath),
	))

	// Tx Rate

	var (
		maxTxRate            float64 = 23965.1
		maxTxRateBlockHeight int     = 700000
	)

	writer.Write(html.Div(
		html.H2("2. Transaction Rate"),
		html.P(`The experiment was conducted for the block range from <b>%d</b> to <b>%d</b>.`, first, last),
		html.P(`For the entire run, the max transaction rate is <b>%f</b> TPM, at block height <b>%d</b> `, maxTxRate, maxTxRateBlockHeight),
	))

	components.NewPage().AddCharts(
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("Tx Rate", buckets, txRateByBucket),
				"Block Height", "Transactions", "TPM",
			), "Tx Rate", "",
		),
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("Overall Tx Rate", buckets, overallTxRateByBucket),
				"Block Height", "Transactions", "TPM",
			), "Overall Tx Rate", "",
		),
	).Render(writer)

	// Gas Rate

	var (
		maxGasRate            float64 = 1575026886
		maxGasRateBlockHeight int     = 800000
	)

	writer.Write(html.Div(
		html.H2("3. Gas Rate"),
		html.P(`The experiment was conducted for the block range from <b>%d</b> to <b>%d</b>.`, first, last),
		html.P(`For the entire run, the max gas rate is <b>%f</b> TPM, at block height <b>%d</b> `, maxGasRate, maxGasRateBlockHeight),
	))

	components.NewPage().AddCharts(
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("Gas Rate", buckets, gasRateByBucket),
				"Block Height", "Gas", "GPM",
			), "Gas Rate", "",
		),
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("Overall Gas Rate", buckets, overallGasRateByBucket),
				"Block Height", "Gas", "GPM",
			), "Overall Gas Rate", "",
		),
	).Render(writer)
	// memory and disk usage
	var (
		maxRam             float64 = 6.687
		maxRamBlockHeight  int     = 22600000
		maxDisk            float64 = 4.215
		maxDiskBlockHeight int     = 17900000
	)

	writer.Write(html.Div(
		html.H2("4. Memory and Disk Usage"),
		html.P(`The experiment was conducted for the block range from <b>%d</b> to <b>%d</b>.`, first, last),
		html.P(`For the entire run, max RAM Usage the run is <b>%f</b> Gigabytes at block height <b>%d</b>.`, maxRam, maxRamBlockHeight),
		html.P(`For the entire run, max Disk Usage throughout the run is <b>%f</b> Gigabytes at block height <b>%d</b>.`, maxDisk, maxDiskBlockHeight),
	))

	// memory and disk chart
	components.NewPage().AddCharts(
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("RAM", buckets, memoryByBucket),
				"Block Height", "RAM Consumption", "Byte",
			), "Memory Usage", "",
		),
		ScatterWithTitle(
			ScatterWithCustomXy(
				scatter("Disk", buckets, diskByBucket),
				"Block Height", "Disk Consumption", "Byte",
			), "Disk", "",
		),
	).Render(writer)

	log.Noticef("Rendered to %s", pHtml)
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

func scatter[T Number](title string, buckets []int, byBucket map[int]T) *charts.Scatter {
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
