//usr/bin/env go run $0; exit

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

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
	first        int = 0
	last         int = 65_436_418
	worker_count int = 10
	bucket_count int = 654
	op_count     int = 50

	connection                   string = "/home/rapolt/dev/sqlite3/test.db"
	sqlite3_SelectFromOperations string = `
	SELECT start, end, opId, opName, count, sum, mean, min, max
	FROM operations 
	WHERE start=:start AND end=:end AND count > 0;`
	sqlite3_SelectDistinctOps string = ` 
	SELECT DISTINCT opId, opName
	FROM operations
	ORDER BY opId ASC;`

	// report
	pHtml = "report.html"
)

type query struct {
	Start  int `db:"start"`
	End    int `db:"end"`
	bucket int
}

type tx_op struct {
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

type op_lookup struct {
	OpId   int    `db:"opId"`
	OpName string `db:"opName"`
}

type done_msg struct {
	q        query
	bucket   int
	tx_count int
}

type bucket_msg struct {
	bucket int
	count  int
	time   float64
}

type op_msg struct {
	bucket int
	opid   int
	count  int
	time   float64
	avg    float64
	min    float64
	max    float64
}

// TODO: make sure worker returns the thread properly
func worker(id int, opCount int, queries <-chan query, done chan<- done_msg,
	bc chan<- bucket_msg, oc chan<- op_msg) {

	for q := range queries {
		db, err := sqlx.Open("sqlite3", connection)
		if err != nil {
			panic(err)
		}

		stmt, err := db.PrepareNamed(sqlite3_SelectFromOperations)
		if err != nil {
			panic(err)
		}

		txs := []tx_op{}
		stmt.Select(&txs, q)
		stmt.Close()
		db.Close()

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

		bc <- bucket_msg{q.bucket, count, time}

		for id := 0; id < opCount; id++ {
			oc <- op_msg{
				q.bucket,
				id,
				countByOpId[id],
				timeByOpId[id],
				meanByOpId[id],
				minByOpId[id],
				maxByOpId[id],
			}
		}

		done <- done_msg{q, id, len(txs)}
	}
}

func main() {

	start := time.Now()

	var (
		//interval int = (last-first) / bucket_count
		//buckets []int = make([]int, bucket_count)
		interval int   = 100_000
		buckets  []int = make([]int, bucket_count)

		opIds        []int          = []int{}
		opNameByOpId map[int]string = map[int]string{}

		countTotal          int                     = 0
		timeTotal           float64                 = 0
		countByBucket       map[int]float64         = make(map[int]float64, bucket_count)
		timeByBucket        map[int]float64         = make(map[int]float64, bucket_count)
		countByOpId         map[int]float64         = make(map[int]float64, bucket_count)
		timeByOpId          map[int]float64         = make(map[int]float64, bucket_count)
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

	fmt.Println("Bucket: ", bucket_count, "Interval: ", interval, "Worker: ", worker_count)

	/////////

	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		panic(err)
	}

	opls := []op_lookup{}
	err = db.Select(&opls, sqlite3_SelectDistinctOps)
	if err != nil {
		panic(err)
	}

	for _, opl := range opls {
		opIds = append(opIds, opl.OpId)
		opNameByOpId[opl.OpId] = opl.OpName
	}

	db.Close()

	fmt.Println("op_count: ", len(opIds))

	/////////

	queries := make(chan query, bucket_count)
	done := make(chan done_msg, bucket_count)
	bc := make(chan bucket_msg, bucket_count)
	oc := make(chan op_msg, op_count*bucket_count)

	for w := 0; w < worker_count; w++ {
		go worker(w, 50, queries, done, bc, oc)
	}

	itv := utils.NewInterval(uint64(first), uint64(last), uint64(interval))

	queries <- query{int(0), int(100000), 0}
	itv.Next()

	for b := 1; b < bucket_count; b, itv = b+1, itv.Next() {
		q := query{int(itv.Start() + 1), int(itv.End() + 1), b}
		buckets[b] = int(itv.Start())
		queries <- q
	}
	close(queries)

	for b := 0; b < bucket_count; b++ {
		fmt.Println(<-done)
	}
	close(done)

	elapsed := time.Since(start)
	fmt.Println("time taken: ", elapsed)

	for b := 0; b < bucket_count; b++ {
		m := <-bc

		countTotal += m.count
		countByBucket[b] += float64(m.count)
		timeTotal += m.time
		timeByBucket[b] += m.time
	}
	close(bc)

	for bo := 0; bo < bucket_count*op_count; bo++ {
		m := <-oc

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
	close(oc)

	fmt.Println("Metadata")
	fmt.Println("count: ", countTotal, "time: ", timeTotal, "opCount: ", len(opIds))
	printTopX(countByOpId, opNameByOpId, 7)
	printTopX(timeByOpId, opNameByOpId, 7)
	for _, opid := range opIds {
		if countByOpId[opid] == 0 {
			fmt.Println(opid, opNameByOpId[opid])
		}
	}

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
		panic(err)
	}

	page.Render(io.MultiWriter(f))
	fmt.Println("Rendered to ", pHtml)
}

func printTopX(byOpId map[int]float64, opNameByOpId map[int]string, x int) {
	type kv struct {
		k int
		v float64
	}

	var kvs []kv
	for k, v := range byOpId {
		kvs = append(kvs, kv{k, v})
	}

	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].v > kvs[j].v
	})

	for ix, kv := range kvs[:x] {
		fmt.Println("Rank ", ix, "[", kv.k, "]", opNameByOpId[kv.k], " has value ", kv.v)
	}
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
