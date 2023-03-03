package stochastic

import (
	"fmt"
	"net/http"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

// HTML references for the rendered pages.
const countingRef = "counting-stats"
const queuingRef = "queuing-stats"
const snapshotRef = "snapshot-stats"
const operationRef = "operation-stats"
const txoperationRef = "tx-operation-stats"
const simplifiedMarkovRef = "simplified-markov-stats"
const markovRef = "markov-stats"

// MainHtml is the index page.
const MainHtml = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Aida: Stochastic Estimator</title>
    <link rel="stylesheet" href="style.css">
    <script src="script.js"></script>
  </head>
  <body>
    <h1>Aida: Stochastic Estimator</h1>
    <ul>
    <li> <h3> <a href="/` + countingRef + `"> Counting Statistics </a> </h3> </li>
    <li> <h3> <a href="/` + queuingRef + `"> Queuing Statistics </a> </h3> </li>
    <li> <h3> <a href="/` + snapshotRef + `"> Snapshot Statistics </a> </h3> </li>
    <li> <h3> <a href="/` + txoperationRef + `"> Transactional Operation Statistics  </a> </h3> </li>
    <li> <h3> <a href="/` + operationRef + `"> Operation Statistics  </a> </h3> </li>
    <li> <h3> <a href="/` + simplifiedMarkovRef + `"> Simplified Markov Chain </a> </h3> </li>
    <li> <h3> <a href="/` + markovRef + `"> Markov Chain </a> </h3> </li>
    </ul>
</body>
</html>
`

// renderMain renders the main menu.
func renderMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, MainHtml)
}

// convertCountingData converts CDF points to chart points.
func convertCountingData(data [][2]float64) []opts.LineData {
	items := []opts.LineData{}
	for _, pair := range data {
		items = append(items, opts.LineData{Value: pair})
	}
	return items
}

// newCountingChart creates a line chart for a counting statistic.
func newCountingChart(title string, subtitle string, lambda float64, ecdf [][2]float64, cdf [][2]float64) *charts.Line {
	chart := charts.NewLine()
	chart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{
		Theme: types.ThemeChalk,
	}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Title: "Save",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: subtitle,
		}))
	sLambda := fmt.Sprintf("%v", lambda)
	chart.AddSeries("eCDF", convertCountingData(ecdf)).AddSeries("CDF, λ="+sLambda, convertCountingData(cdf))
	return chart
}

// renderCounting renders counting statistics.
func renderCounting(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	contracts := newCountingChart("Counting Statistics", "for Contract-Addresses",
		events.Contracts.Lambda,
		events.Contracts.ECdf,
		events.Contracts.Cdf)
	keys := newCountingChart("Counting Statistics", "for Storage-Keys",
		events.Keys.Lambda,
		events.Keys.ECdf,
		events.Keys.Cdf)
	values := newCountingChart("Counting Statistics", "for Storage-Values",
		events.Values.Lambda,
		events.Values.ECdf,
		events.Values.Cdf)

	// TODO: Set HTML title via GlobalOption
	page := components.NewPage()
	page.AddCharts(contracts, keys, values)
	page.Render(w)
}

// renderSnapshotStast renders a line chart for a snapshot statistics
func renderSnapshotStats(w http.ResponseWriter, r *http.Request) {
	chart := charts.NewLine()
	chart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{
		Theme: types.ThemeChalk,
	}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Title: "Save",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Snapshot Statistics",
			Subtitle: "Delta Distribution",
		}))
	events := GetEventsData()
	sLambda := fmt.Sprintf("%v", events.Snapshot.Lambda)
	chart.AddSeries("eCDF", convertCountingData(events.Snapshot.ECdf)).AddSeries("CDF, λ="+sLambda, convertCountingData(events.Snapshot.Cdf))
	chart.Render(w)
}

// convertQueuingData rendering plot data for the queuing statistics.
func convertQueuingData(data []float64) []opts.ScatterData {
	items := []opts.ScatterData{}
	for x, p := range data {
		items = append(items, opts.ScatterData{Value: [2]float64{float64(x), p}, SymbolSize: 5})
	}
	return items
}

// renderQueuing renders a queuing statistics.
func renderQueuing(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{
		Theme:     types.ThemeChalk,
		PageTitle: "Queuing Probabilities",
	}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Title: "Save",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Queuing Probabilities",
			Subtitle: "for contract-addresses, storage-keys, and storage-values",
		}))
	scatter.AddSeries("Contract", convertQueuingData(events.Contracts.QPdf)).AddSeries("Keys", convertQueuingData(events.Keys.QPdf)).AddSeries("Values", convertQueuingData(events.Values.QPdf))
	scatter.Render(w)
}

// convertOperationData produces the data series for the sationary distribution.
func convertOperationData(data []OpData) []opts.BarData {
	items := []opts.BarData{}
	for i := 0; i < len(data); i++ {
		items = append(items, opts.BarData{Value: data[i].value})
	}
	return items
}

// convertOperationLabel produces operations' labels.
func convertOperationLabel(data []OpData) []string {
	items := []string{}
	for i := 0; i < len(data); i++ {
		items = append(items, data[i].label)
	}
	return items
}

// renderOperationStats renders the stationary distribution.
func renderOperationStats(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{
		Theme:     types.ThemeChalk,
		PageTitle: "StateDB Operations",
		Height:    "1300px",
	}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Title: "Save",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTitleOpts(opts.Title{
			Title: "StateDB Operations",
		}))
	bar.SetXAxis(convertOperationLabel(events.Stationary)).AddSeries("Stationary Distribution", convertOperationData(events.Stationary))
	bar.XYReversal()
	bar.Render(w)
}

// renderTransactionalOperationStats renders the average number of operations per transaction.
func renderTransactionalOperationStats(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	title := fmt.Sprintf("Average %.1f Tx/Bl; %.1f Bl/Ep", events.TxPerBlock, events.BlocksPerEpoch)
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{
		Theme:     types.ThemeChalk,
		PageTitle: title,
		Height:    "1300px",
	}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Title: "Save",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show: true,
				},
			},
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTitleOpts(opts.Title{
			Title: title,
		}))
	bar.SetXAxis(convertOperationLabel(events.TxOperation)).AddSeries("Ops/Tx", convertOperationData(events.TxOperation))
	bar.Render(w)
}

// renderSimplifiedMarkovChain renders a reduced markov chain whose nodes have no argument classes.
func renderSimplifiedMarkovChain(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	g := graphviz.New()
	graph, _ := g.Graph()
	defer func() {
		graph.Close()
		g.Close()
	}()
	nodes := make([]*cgraph.Node, numOps)
	for op := 0; op < numOps; op++ {
		nodes[op], _ = graph.CreateNode(opMnemo[op])
		nodes[op].SetLabel(opMnemo[op])
	}
	for i := 0; i < numOps; i++ {
		for j := 0; j < numOps; j++ {
			p := events.SimplifiedMatrix[i][j]
			if p > 0.0 {
				txt := fmt.Sprintf("%.2f", p)
				e, _ := graph.CreateEdge("", nodes[i], nodes[j])
				e.SetLabel(txt)
				var color string
				switch int(4 * p) {
				case 0:
					color = "gray"
				case 1:
					color = "green"
				case 3:
					color = "indianred"
				case 4:
					color = "red"
				}
				e.SetColor(color)
			}
		}
	}
	txt, _ := renderDotGraph("StateDB Simplified Markov-Chain", g, graph)
	fmt.Fprint(w, txt)
}

// renderMarkovChain renders a markov chain.
func renderMarkovChain(w http.ResponseWriter, r *http.Request) {
	events := GetEventsData()
	g := graphviz.New()
	graph, _ := g.Graph()
	defer func() {
		graph.Close()
		g.Close()
	}()
	n := len(events.OperationLabel)
	nodes := make([]*cgraph.Node, n)
	for i := 0; i < n; i++ {
		nodes[i], _ = graph.CreateNode(events.OperationLabel[i])
		nodes[i].SetLabel(events.OperationLabel[i])
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			p := events.StochasticMatrix[i][j]
			if p > 0.0 {
				txt := fmt.Sprintf("%.2f", p)
				e, _ := graph.CreateEdge("", nodes[i], nodes[j])
				e.SetLabel(txt)
				var color string
				switch int(4 * p) {
				case 0:
					color = "gray"
				case 1:
					color = "green"
				case 3:
					color = "indianred"
				case 4:
					color = "red"
				}
				e.SetColor(color)
			}
		}
	}
	txt, _ := renderDotGraph("StateDB Markov-Chain", g, graph)
	fmt.Fprint(w, txt)
}

// FireUpWeb produces a data model for the recorded events and
// visualizes with a local web-server.
func FireUpWeb(eventRegistry *EventRegistryJSON, addr string) {

	// create data model (as a singleton) for visualization
	eventModel := GetEventsData()
	eventModel.PopulateEventData(eventRegistry)

	// create web server
	http.HandleFunc("/", renderMain)
	http.HandleFunc("/"+countingRef, renderCounting)
	http.HandleFunc("/"+queuingRef, renderQueuing)
	http.HandleFunc("/"+snapshotRef, renderSnapshotStats)
	http.HandleFunc("/"+operationRef, renderOperationStats)
	http.HandleFunc("/"+txoperationRef, renderTransactionalOperationStats)
	http.HandleFunc("/"+simplifiedMarkovRef, renderSimplifiedMarkovChain)
	http.HandleFunc("/"+markovRef, renderMarkovChain)
	http.ListenAndServe(":"+addr, nil)
}
