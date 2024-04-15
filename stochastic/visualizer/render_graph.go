// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
