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
	first                   int    = 479_327
	last                    int    = 22_832_168
	logLevel                string = "Debug"
	connection              string = "/var/opera/Aida/tmp-rapolt/register/s5-f1.db"
	sqlite3_SelectFromStats string = `
		SELECT start, end, memory, disk, tx_rate, gas_rate, overall_tx_rate, overall_gas_rate
		FROM stats
		WHERE start>=:start AND end<=:end;
	`

	workerCount int = 10
	bucketCount int = 223

	// report
	pHtml = "report_f1.html"
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

	stmt, err := db.PrepareNamed(sqlite3_SelectFromStats)
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

	log.Infof("Bucket: %d, Interval: %d, Worker: %d", bucketCount, interval, workerCount)

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

	log.Infof("queries - time taken: %f s", time.Since(start).Seconds())

	close(bc)
	bWg.Wait()

	log.Infof("postprocessing - time taken: %f s", time.Since(start).Seconds())

	// Charts start here
	page := components.NewPage().AddCharts(
		ScatterWithTitle(scatter("Memory", buckets, memoryByBucket), "Memory", ""),
		ScatterWithTitle(scatter("Disk", buckets, diskByBucket), "Disk", ""),
		ScatterWithTitle(scatter("Tx Rate", buckets, txRateByBucket), "Tx Rate", ""),
		ScatterWithTitle(scatter("Gas Rate", buckets, gasRateByBucket), "Gas Rate", ""),
		ScatterWithTitle(scatter("Overall Tx Rate", buckets, overallTxRateByBucket), "Overall Tx Rate", ""),
		ScatterWithTitle(scatter("Overall Gas Rate", buckets, overallGasRateByBucket), "Overall Gas Rate", ""),
	)

	f, err := os.Create(pHtml)
	if err != nil {
		log.Errorf("Unable to create html documents at %s", pHtml)
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
