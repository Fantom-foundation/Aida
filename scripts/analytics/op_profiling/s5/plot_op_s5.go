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
	"github.com/Fantom-foundation/Aida/scripts/analytics/html"
	"github.com/Fantom-foundation/Aida/tracer/operation"
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

// all configuration goes here
const (
	first      int    = 0
	last       int    = 65_436_418
	logLevel   string = "Debug"
	connection string = "/var/opera/Aida/tmp-rapolt/op-profiling/s5-op-profiling.db"

	queryWorkerCount     int = 10
	bucketMsgWorkerCount int = 1
	opMsgWorkerCount     int = 2

	bucketCount int = 654
	opCount     int = 50

	// report
	pHtml = "report_op_s5.html"
)

// DB-related const
const (
	sqlite3SelectFromOperations string = `
		SELECT blockId, txId, opId, opName, count, sum, mean, min, max
		FROM ops_transaction
		WHERE blockId>=:start AND blockId<=:end AND count > 0;
	`
)

type query struct {
	Start  int `db:"start"`
	End    int `db:"end"`
	bucket int
}

type txResponse struct {
	BlockId int     `db:"blockId"`
	TxId    int     `db:"txId"`
	OpId    int     `db:"opId"`
	OpName  string  `db:"opName"`
	Count   int     `db:"count"`
	Sum     float64 `db:"sum"`
	Mean    float64 `db:"mean"`
	Min     float64 `db:"min"`
	Max     float64 `db:"max"`
}

type bucketMsg struct {
	bucket int
	count  int
	time   float64
}

type opMsg struct {
	bucket int
	opid   int
	opname string
	count  int
	time   float64
	avg    float64
	min    float64
	max    float64
}

func worker(id int, opCount int,
	queries <-chan query,
	bc chan<- bucketMsg,
	oc chan<- opMsg,
	ec chan<- error) {

	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		ec <- err
	}

	stmt, err := db.PrepareNamed(sqlite3SelectFromOperations)
	if err != nil {
		ec <- err
	}

	log := logger.NewLogger(logLevel, fmt.Sprintf("Plot Worker #%d", id))

	defer func() {
		stmt.Close()
		db.Close()

		log.Debugf("Worker #%d terminated.", id)
	}()

	for q := range queries {
		log.Debugf("Starting: %v", q)

		txs := []txResponse{}
		stmt.Select(&txs, q)

		var (
			count        int             = 0
			time         float64         = 0
			opNameByOpId map[int]string  = make(map[int]string, opCount)
			countByOpId  map[int]int     = make(map[int]int, opCount)
			timeByOpId   map[int]float64 = make(map[int]float64, opCount)
			meanByOpId   map[int]float64 = make(map[int]float64, opCount)
			minByOpId    map[int]float64 = make(map[int]float64, opCount)
			maxByOpId    map[int]float64 = make(map[int]float64, opCount)
		)

		for _, tx := range txs {
			id := tx.OpId
			count += tx.Count
			time += tx.Sum
			opNameByOpId[id] = tx.OpName
			countByOpId[id] += tx.Count
			timeByOpId[id] += tx.Sum
			meanByOpId[id] = tx.Mean
			minByOpId[id] = xmath.Min(tx.Min, minByOpId[id])
			maxByOpId[id] = xmath.Max(tx.Max, maxByOpId[id])
		}

		bc <- bucketMsg{q.bucket, count, time}

		for id := 0; id < opCount; id++ {
			if q.bucket == 0 {
				fmt.Println(opMsg{
					q.bucket,
					id,
					opNameByOpId[id],
					countByOpId[id],
					timeByOpId[id],
					meanByOpId[id],
					minByOpId[id],
					maxByOpId[id],
				})
			}
			if countByOpId[id] > 0 {
				oc <- opMsg{
					q.bucket,
					id,
					opNameByOpId[id],
					countByOpId[id],
					timeByOpId[id],
					meanByOpId[id],
					minByOpId[id],
					maxByOpId[id],
				}
			}
		}

		log.Debugf("Done: %v", q)
	}
}

func main() {

	start := time.Now()

	var (
		interval int           = 100_000
		buckets  []int         = make([]int, bucketCount)
		log      logger.Logger = logger.NewLogger(logLevel, "Operational Profiler S5 Plot")

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

	log.Noticef("Bucket: %d, Interval: %d, Worker: %d", bucketCount, interval, queryWorkerCount)

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

	// start a thread to monitor for error when querying db, close all channels + terminate if found.
	eWg.Add(1)
	go func() {
		defer eWg.Done()
		for e := range ec {
			log.Errorf("Received an error: ", e)

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
	for w := 0; w < queryWorkerCount; w++ {
		qWg.Add(1)
		go func(id int) {
			defer qWg.Done()
			worker(id, 50, qc, bc, oc, ec)
		}(w)
	}

	// start a thread to digest bucket-wise response from DB
	for w := 0; w < bucketMsgWorkerCount; w++ {
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

	// start a thread to digest op-wise response from DB
	var ol sync.Mutex
	for w := 0; w < opMsgWorkerCount; w++ {
		oWg.Add(1)
		go func() {
			defer oWg.Done()
			for m := range oc {
				ol.Lock()
				countByOpId[m.opid] += float64(m.count)
				timeByOpId[m.opid] += m.time
				countByBucketByOpId[m.opid][m.bucket] += float64(m.count)
				timeByBucketByOpId[m.opid][m.bucket] += m.time
				minByBucketByOpId[m.opid][m.bucket] = xmath.Min(m.min, minByBucketByOpId[m.opid][m.bucket])
				maxByBucketByOpId[m.opid][m.bucket] = xmath.Max(m.max, maxByBucketByOpId[m.opid][m.bucket])

				if m.count > 0 {
					meanByBucketByOpId[m.opid][m.bucket] = timeByBucketByOpId[m.opid][m.bucket] / countByBucketByOpId[m.opid][m.bucket]
				}

				// Defect in DB
				/*
					if _, ok := opNameByOpId[m.opid]; !ok {
						fmt.Println(m)
						opIds = append(opIds, m.opid)
						opNameByOpId[m.opid] = m.opname
					}
				*/
				ol.Unlock()
			}
		}()
	}

	// generate queries here
	itv := utils.NewInterval(uint64(first), uint64(last), uint64(interval))
	for b := 0; b < bucketCount; b, itv = b+1, itv.Next() {
		q := query{int(itv.Start() + 1), int(itv.End() + 1), b}
		buckets[b] = int(itv.Start())
		qc <- q
	}

	close(qc)
	qWg.Wait()

	close(ec) // no more error
	eWg.Wait()

	log.Noticef("queries - time taken: %d s", time.Since(start).Seconds())

	close(bc)
	bWg.Wait()

	close(oc)
	oWg.Wait()

	// get opIds, opNameByOpId
	ops := operation.CreateIdLabelMap() //byte->string
	for opId, count := range countByOpId {
		if count > 0 {
			opIds = append(opIds, opId)
			opNameByOpId[opId] = ops[byte(opId)]
		}
	}

	sort.Slice(opIds, func(i, j int) bool {
		return opIds[i] < opIds[j]
	})

	log.Noticef("postprocessing - time taken: %d s", time.Since(start).Seconds())
	log.Noticef("total: %d, time total: %f", countTotal, timeTotal)

	// generate report

	f, err := os.Create(pHtml)
	if err != nil {
		log.Errorf("Unable to create html at %s.", pHtml)
	}

	writer := io.MultiWriter(f)

	// style for table
	writer.Write([]byte(`
		<style> 
			table {border: 1px solid #54585d; border-collapse: collapse;} 
			tr {border: 1px solid #54585d; border-collapse: collapse;} 
			th {border: 1px solid #54585d; border-collapse: collapse;} 
			td {border: 1px solid #54585d; border-collapse: collapse;} 
		</style>
	`))

	//warning
	writer.Write(html.Div(
		html.H1("<FONT COLOR\"FFFF99\">Warning: Intermediate Results with Known Issues<FONT COLOR>"),
		html.H2("The following report contains results with known issues - a small amount of operations (~10k) are errorneously excluded from the analysis. The issue is being corrected."),
		html.H2("The report has been made available nonetheless, as the broader picture of the analysis is still preserved."),
	))

	// header
	writer.Write(html.Div(
		html.H1("Operation Profiling Report"),
		html.P(time.Now().Format("2006-01-02")),
	))

	// experimental setup
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

	// total operation count
	writer.Write(html.Div(
		html.H2("2. Total Operation Count"),
		html.P(`The experiment was conducted for the block range from <b>%d</b> to <b>%d</b>.`, first, last),
		html.P(`The block range contains <b>%d transactions</b>. The accumulated operation processing time is <b>%f hours</b>.`, countTotal, float64(timeTotal/3600_000_000)),
		html.P(`The top seven operations called are the following:`),
		tableFromTopX([]string{"Op Name", "Number of Calls Made"}, countByOpId, opNameByOpId, 7),
	))

	// charts: op counts
	components.NewPage().AddCharts(
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
	).Render(writer)

	// total operation processing time
	writer.Write(html.Div(
		html.H2("3. Total Operation Processing Time"),
		html.P(`The experiment was conducted for the block range from <b>%d</b> to <b>%d</b>.`, first, last),
		html.P(`The block range contains <b>%d transactions</b>. The accumulated operation processing time is <b>%f hours</b>.`, countTotal, float64(timeTotal/3600_000_000)),
		tableFromTopX([]string{"Op Name", "Total Processing Time"}, timeByOpId, opNameByOpId, 7),
	))

	// charts: op time
	components.NewPage().AddCharts(
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
	).Render(writer)

	// Glossary
	writer.Write(html.Div(
		html.H2("4. Glossary"),
		html.P("Total <b>%d</b> operation types are profiled. Each type is shown here profiled with the following charts:", len(opIds)),
		html.List([]string{
			"Call per 100,000 Blocks",
			"Total Processing Time per 100,000 Blocks",
			"Average Processing Time per call",
		}),
	))

	for _, op := range opIds {
		if countByOpId[op] == 0 {
			continue
		}

		// add glossary tag
		writer.Write(html.Div(
			html.H3("[%d] %s", op, opNameByOpId[op]),
		))

		// add charts for glossary
		components.NewPage().AddCharts(
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
		).Render(writer)
	}

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

func tableFromTopX(headers []string, byOpId map[int]float64, opNameByOpId map[int]string, x int) []byte {
	var values []map[string]string

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

	for _, kv := range kvs[:x] {
		m := make(map[string]string, len(headers))
		m[headers[0]] = opNameByOpId[kv.k]
		m[headers[1]] = fmt.Sprintf("%f", kv.v)
		values = append(values, m)
	}

	return html.Table(headers, values)
}
