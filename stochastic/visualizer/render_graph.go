package visualizer

import (
	"bytes"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

// preGraphHtml is the preamble for an HTML page rending a dot graph.
const preGraphHtml = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>TITLE</title>

    <script>
        const dot = ` + "`"

// postGraphHtml is the postamble for an HTML page rending a dot graph.
const postGraphHtml = "`" + `;
    </script>
</head>

<body>
    <h1>TITLE</h1>
    <div id="graph"></div>
    <script type="module">
        import { Graphviz } from "https://cdn.jsdelivr.net/npm/@hpcc-js/wasm/dist/index.js";
        if (Graphviz) {
            const graphviz = await Graphviz.load();
            const svg = graphviz.layout(dot, "svg", "dot");
	    document.getElementById("graph").innerHTML = svg;
        } 
    </script>
</body>
</html>
`

// renderDotGraph renders a dotgraph as a HTML document.
func renderDotGraph(title string, g *graphviz.Graphviz, graph *cgraph.Graph) (string, error) {
	preamble := strings.Replace(preGraphHtml, "TITLE", title, -1)
	postamble := strings.Replace(postGraphHtml, "TITLE", title, -1)
	var buf bytes.Buffer
	if err := g.Render(graph, "dot", &buf); err != nil {
		return "", err
	}
	return preamble + buf.String() + postamble, nil
}
