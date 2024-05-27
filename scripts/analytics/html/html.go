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

package html

import (
	"bytes"
	"fmt"
)

func Div(contents ...[]byte) []byte {
	var b bytes.Buffer

	// start tag
	b.WriteString("<div class=\"container\">\n")
	b.WriteString("  <div class=\"item\" style=\"width:900px;font-family: Arial;\">\n")

	// content
	for _, content := range contents {
		b.Write(content)
	}

	// end tag
	b.WriteString("  </div>\n")
	b.WriteString("</div>\n")

	return b.Bytes()
}

func Text(tag string, formatted string) []byte {
	var b bytes.Buffer

	// start tag
	b.WriteString(fmt.Sprintf("<%s>\n", tag))

	// content
	b.WriteString(fmt.Sprintf("  %s", formatted))

	// end tag
	b.WriteString(fmt.Sprintf("</%s>\n", tag))

	return b.Bytes()
}

func P(paragraph string, format ...any) []byte {
	return Text("p", fmt.Sprintf(paragraph, format...))
}

func H1(paragraph string, format ...any) []byte {
	return Text("h1", fmt.Sprintf(paragraph, format...))
}

func H2(paragraph string, format ...any) []byte {
	return Text("h2", fmt.Sprintf(paragraph, format...))
}

func H3(paragraph string, format ...any) []byte {
	return Text("h3", fmt.Sprintf(paragraph, format...))
}

func Table(headers []string, values []map[string]string) []byte {
	var b bytes.Buffer

	// start tag
	b.WriteString("<table class=\"item\">\n")

	// header
	b.WriteString("<tr>\n")
	for _, header := range headers {
		b.WriteString(fmt.Sprintf("  <th>%s</th>\n", header))
	}
	b.WriteString("</tr>\n")

	// values
	for _, value := range values {
		b.WriteString("<tr>\n")
		for _, header := range headers {
			b.WriteString(fmt.Sprintf("  <td>%s</td>\n", value[header]))
		}
		b.WriteString("</tr>\n")
	}

	// end tag
	b.WriteString("</table>\n")

	return b.Bytes()
}

func List(items []string) []byte {
	var b bytes.Buffer

	// start tag
	b.WriteString("<ul>\n")

	// content
	for _, item := range items {
		b.WriteString(fmt.Sprintf("  <li>%s</li", item))
	}

	// end tag
	b.WriteString("</ul>\n")

	return b.Bytes()
}
