//usr/bin/env go run $0; exit

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"slices"
	"sort"

	"github.com/Fantom-foundation/Aida/utils"
	md "github.com/go-spectest/markdown"
	"github.com/jmoiron/sqlx"
	mp "github.com/mandolyte/mdtopdf"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const (
	// db
	first                        uint64 = 0
	last                         uint64 = 65_436_418
	interval                     uint64 = 100_000
	connection                   string = "/home/rapolt/dev/sqlite3/test.db"
	sqlite3_SelectFromOperations        = `SELECT * FROM operations WHERE start=:start AND end=:end`

	// report
	pPngs = "png"
	pMd   = "report.md"
	pPdf  = "report.pdf"
)

type query struct {
	Start int `db:"start"`
	End   int `db:"end"`
}

type statistics struct {
	Start    int     `db:"start"`
	End      int     `db:"end"`
	OpId     int     `db:"opId"`
	OpName   string  `db:"opName"`
	Count    int     `db:"count"`
	Sum      float64 `db:"sum"`
	Mean     float64 `db:"mean"`
	Std      float64 `db:"std"`
	Variance float64 `db:"variance"`
	Skewness float64 `db:"skewness"`
	Kurtosis float64 `db:"kurtosis"`
	Min      float64 `db:"min"`
	Max      float64 `db:"max"`
}

func main() {

	var (
		byOpId  map[int][]statistics = map[int][]statistics{}
		byStart map[int][]statistics = map[int][]statistics{}

		starts []uint64 = []uint64{}
	)

	// color
	clr := map[int]drawing.Color{
		20: drawing.ColorFromHex("92CEA8"), //seafoam
		37: drawing.ColorFromHex("EEE4E1"), //Egg
		30: drawing.ColorFromHex("E5F9FE"), // Oyster Bay
		13: drawing.ColorFromHex("FFBBDA"), //Cotton Candy
		28: drawing.ColorFromHex("F6C6C7"), //Flamingo
		15: drawing.ColorFromHex("916848"), //Leather
		0:  drawing.ColorFromHex("D4C8BE"), //Dusty Rose
		17: drawing.ColorFromHex("A8D1E7"), // Light Blue
		7:  drawing.ColorFromHex("8BD2EC"), //skay
		10: drawing.ColorFromHex("B3DBD8"), //Scandal
		9:  drawing.ColorFromHex("577460"), //Ebony
		16: drawing.ColorFromHex("E2E3DE"), //Stone White
		35: drawing.ColorFromHex("BFCED6"), //Misty
		38: drawing.ColorFromHex("9DA2AE"), //Winter sea

		999: drawing.ColorFromHex("808080"), //other: grey
	}

	// db
	db, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		panic(err)
	}

	stmt, err := db.PrepareNamed(sqlite3_SelectFromOperations)
	if err != nil {
		panic(err)
	}

	stats := []statistics{}
	q := query{int(0), int(100000)}
	stmt.Select(&stats, q)
	for _, stat := range stats {
		byOpId[stat.OpId] = append(byOpId[stat.OpId], stat)
		byStart[1] = append(byStart[1], stat)
	}

	for i := utils.NewInterval(first, last, interval); i.End() < last; i.Next() {
		starts = append(starts, i.Start()+1)

		q := query{int(i.Start()) + 1, int(i.End()) + 1}
		stmt.Select(&stats, q)
		for _, s := range stats {
			byOpId[s.OpId] = append(byOpId[s.OpId], s)
			byStart[s.Start] = append(byStart[s.Start], s)
		}
	}

	stmt.Close()
	db.Close()

	// calculate total op count per interval
	var (
		opIds              []int                   = []int{}
		opNameByOpId       map[int]string          = map[int]string{}
		totalCount         uint64                  = 0
		countByStart       map[int]float64         = map[int]float64{}
		timeByStart        map[int]float64         = map[int]float64{}
		countByOpId        map[int]float64         = map[int]float64{}
		timeByOpId         map[int]float64         = map[int]float64{}
		avgByOpId          map[int]float64         = map[int]float64{}
		countByStartByOpId map[int]map[int]float64 = map[int]map[int]float64{}
		timeByStartByOpId  map[int]map[int]float64 = map[int]map[int]float64{}
	)

	for id, stats := range byOpId {
		opIds = append(opIds, id)
		opNameByOpId[id] = stats[0].OpName
	}

	sort.Slice(opIds, func(i, j int) bool { return opIds[i] < opIds[j] })

	target := float64(1_000_000)
	for _, start := range starts {
		s := int(float64(start) / target)
		countByStartByOpId[s] = map[int]float64{}
		timeByStartByOpId[s] = map[int]float64{}
	}

	for _, start := range starts {
		stats := byStart[int(start)]
		for _, stat := range stats {
			s := int(float64(start) / target)
			id := int(stat.OpId)
			totalCount += uint64(stat.Count)
			countByStart[s] += float64(stat.Count)
			timeByStart[s] += float64(stat.Sum)
			countByOpId[id] += float64(stat.Count)
			timeByOpId[id] += float64(stat.Sum)
			countByStartByOpId[s][id] += float64(stat.Count)
			timeByStartByOpId[s][id] += float64(stat.Sum)
			avgByOpId[id] = float64(timeByOpId[id]) / float64(countByOpId[id])
		}
	}

	// plot

	makePieChart("./pngs/count.pie.png", countByOpId, opNameByOpId, clr)
	makePieChart("./pngs/time.pie.png", timeByOpId, opNameByOpId, clr)
	makePieChart("./pngs/avg.pie.png", avgByOpId, opNameByOpId, clr)

	makeBarChart("./pngs/count.bar.png", countByStart, clr, 5000_000_000)
	makeBarChart("./pngs/time.bar.png", timeByStart, clr, 8000_000_000)
	makeBarChart("./pngs/avg.bar.png", avgByOpId, clr, 200)

	makePercentageTrend("./pngs/time.bars.png", timeByStartByOpId, opNameByOpId, clr)

	for _, id := range opIds {
		makeGraph(
			fmt.Sprintf("./pngs/%d.total.png", id),
			fmt.Sprintf("./pngs/%d.avg.png", id),
			byOpId[id],
		)
	}

	// package as md

	f, err := os.Create(pMd)
	if err != nil {
		panic(err)
	}

	mdf := md.NewMarkdown(f).
		H2("Overall Count").
		PlainTextf(md.Image("", "./pngs/count.pie.png")).
		PlainTextf(md.Image("", "./pngs/count.bar.png")).
		H2("Overall Time Taken").
		PlainTextf(md.Image("", "./pngs/time.pie.png")).
		PlainTextf(md.Image("", "./pngs/time.bar.png")).
		H2("Overall Time Taken / call").
		PlainTextf(md.Image("", "./pngs/avg.pie.png")).
		PlainTextf(md.Image("", "./pngs/avg.bar.png"))

	for _, id := range opIds {
		mdf = mdf.
			H2(fmt.Sprintf("[%d] %s", id, opNameByOpId[id])).
			H3("Total Time / Block (s)").
			PlainTextf(md.Image("", fmt.Sprintf("./pngs/%d.total.png", id))).
			H3("Avg Time / Block (ms)").
			PlainTextf(md.Image("", fmt.Sprintf("./pngs/%d.avg.png", id)))
	}

	mdf.Build()

	// md to pdf

	var r *mp.PdfRenderer = mp.NewPdfRenderer("", "", pPdf, "report.log", []mp.RenderOption{}, mp.DARK)

	content, _ := ioutil.ReadFile(pMd)
	r.Process(content)
}

type val struct {
	id int
	v  float64
}

type byVal []val

func (a byVal) Len() int {
	return len(a)
}

func (a byVal) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byVal) Less(i, j int) bool {
	return a[i].v > a[j].v
}

type byId []val

func (b byId) Len() int {
	return len(b)
}

func (b byId) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byId) Less(i, j int) bool {
	return b[i].id < b[j].id
}

func printLabel(i int) string {
	if i%10_000_000 == 1 {
		if i == 1 {
			return "0"
		}
		return fmt.Sprintf("%d", i/10_000_000)
	}
	return ""
}

func makeGraph(pTotalPng string, pAvgPng string, stats []statistics) string {
	var (
		sums   []float64 = []float64{}
		means  []float64 = []float64{}
		starts []float64 = []float64{}
	)

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Start < stats[j].Start
	})

	for _, stat := range stats {
		sums = append(sums, float64(stat.Sum/1_000_000))
		means = append(means, float64(stat.Mean))
		starts = append(starts, float64(stat.Start/100_000))
	}

	meanSeries := chart.ContinuousSeries{
		Name: "Mean",
		Style: chart.Style{
			StrokeColor: chart.GetDefaultColor(0),
		},
		XValues: starts,
		YValues: means,
	}

	graph := chart.Chart{
		Width:  750,
		Height: 325,
		Series: []chart.Series{
			meanSeries,
		},
	}

	f, err := os.Create(pAvgPng)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	graph.Render(chart.PNG, f)

	totalSeries := chart.ContinuousSeries{
		Name: "Total",
		Style: chart.Style{
			StrokeColor: chart.GetDefaultColor(0),
		},
		XValues: starts,
		YValues: sums,
	}

	graph2 := chart.Chart{
		Width:  750,
		Height: 325,
		Series: []chart.Series{
			totalSeries,
		},
	}

	f2, err := os.Create(pTotalPng)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	graph2.Render(chart.PNG, f2)

	return pTotalPng
}

func makeBarChart(pPng string, valByStart map[int]float64, clr map[int]drawing.Color, max int) string {
	var (
		vals  []val        = []val{}
		ticks []chart.Tick = []chart.Tick{}
	)

	for id, v := range valByStart {
		vals = append(vals, val{id, v})
	}

	sort.Sort(byId(vals))

	values := []chart.Value{}
	for _, val := range vals {
		values = append(values, chart.Value{
			Value: val.v,
			Label: printLabel(val.id),
			Style: chart.Style{
				FillColor:   clr[999],
				StrokeColor: drawing.ColorFromHex("000000"),
				StrokeWidth: 0,
			},
		})
	}

	for i := 0; i <= max; i += max / 5 {
		ticks = append(ticks, chart.Tick{
			Value: float64(i),
			Label: fmt.Sprintf("%d", i),
		})
	}

	bars := chart.BarChart{
		Width:  750,
		Height: 400,
		Background: chart.Style{
			Padding: chart.Box{
				Top: 50,
			},
		},
		Bars:  values,
		YAxis: chart.YAxis{Ticks: ticks},
	}

	f, err := os.Create(pPng)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	bars.Render(chart.PNG, f)

	return pPng
}

func makePieChart(pPng string, valByOpId map[int]float64, nameById map[int]string, clr map[int]drawing.Color) string {
	var (
		vals      []val   = []val{}
		total     float64 = 0.0
		now       float64 = 0.0
		remaining float64 = 0.0
	)

	for id, v := range valByOpId {
		vals = append(vals, val{id, v})
		total += v
	}

	sort.Sort(byVal(vals))

	values := []chart.Value{}
	for _, val := range vals {
		now += val.v
		if now/total < 0.93 {
			values = append(values, chart.Value{
				Value: val.v,
				Label: fmt.Sprintf("%s %.1f%%", nameById[val.id], val.v/total*100),
				Style: chart.Style{
					FillColor: clr[val.id],
				},
			})
		} else {
			remaining += val.v
		}
	}

	values = append(values, chart.Value{
		Value: remaining,
		Label: fmt.Sprintf("Others %.1f%%", remaining/total*100),
		Style: chart.Style{
			FillColor: clr[999],
		},
	})

	pie := chart.DonutChart{
		Width:  750,
		Height: 400,
		Values: values,
	}

	f, err := os.Create(pPng)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	pie.Render(chart.PNG, f)

	return pPng
}

func makePercentageTrendByStart(start int, timeByOpId map[int]float64, nameById map[int]string, clr map[int]drawing.Color) chart.StackedBar {

	var (
		tracks    []int   = []int{20, 7, 13, 9, 28, 37, 10}
		total     float64 = 0.0
		remainder float64 = 0.0
	)

	for _, v := range timeByOpId {
		total += v
	}

	values := []chart.Value{}
	for _, id := range tracks {
		values = append(values, chart.Value{
			Value: timeByOpId[id],
			Label: fmt.Sprintf("%d", id),
			Style: chart.Style{
				FillColor:   clr[id],
				StrokeColor: clr[id],
				FontSize:    10,
			},
		})
	}

	for id, v := range timeByOpId {
		if slices.Contains(tracks, id) {
			continue
		}
		remainder += v
	}

	values = append(values, chart.Value{
		Value: remainder,
		Label: "Others",
		Style: chart.Style{
			FillColor:   clr[999],
			StrokeColor: clr[999],
			FontSize:    10,
		},
	})

	slices.Reverse(values)

	return chart.StackedBar{
		Name:   fmt.Sprintf("%d", start),
		Width:  20,
		Values: values,
	}

}

func makePercentageTrend(pPng string, timeByStartByOpId map[int]map[int]float64, nameById map[int]string, clr map[int]drawing.Color) string {

	var (
		bars   []chart.StackedBar = []chart.StackedBar{}
		starts []int              = []int{}
	)

	for start := range timeByStartByOpId {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })

	for _, start := range starts {
		bars = append(bars, makePercentageTrendByStart(start, timeByStartByOpId[start], nameById, clr))
	}

	stackedBarChart := chart.StackedBarChart{
		TitleStyle:   chart.Shown(),
		Width:        1200,
		Height:       1800,
		XAxis:        chart.Shown(),
		YAxis:        chart.Shown(),
		BarSpacing:   1,
		IsHorizontal: true,
		Bars:         bars,
	}

	f, err := os.Create(pPng)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	stackedBarChart.Render(chart.PNG, f)

	return pPng
}

func HashGeneric[T comparable](a, b []T) []T {
	set := make([]T, 0)
	hash := make(map[T]struct{})

	for _, v := range a {
		hash[v] = struct{}{}
	}

	for _, v := range b {
		if _, ok := hash[v]; ok {
			set = append(set, v)
		}
	}

	return set
}
