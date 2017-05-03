// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package yarpc

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"go.uber.org/yarpc/internal/introspection"
)

type dispatcherIdl struct {
	Name       string
	ID         string
	IDLModules []introspection.IDLModule
}

func renderIDLs(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/debug/yarpc/idl/")
	parts := strings.SplitN(path, "/", 2)
	var selectDispatcher string
	var selectIDL string
	if path != "" {
		if len(parts) != 2 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Invalid arguments")
			return
		}
		selectDispatcher = parts[0]
		selectIDL = parts[1]
	}

	data := struct {
		Dispatchers     []dispatcherIdl
		PackageVersions []introspection.PackageVersion
	}{
		PackageVersions: PackageVersions,
	}

	for _, disp := range dispatchers {
		if selectDispatcher != "" && disp.Name() != selectDispatcher {
			continue
		}
		procedures := introspection.IntrospectProcedures(disp.table.Procedures())
		idls := procedures.IDLModules()
		if selectIDL != "" {
			var selected []introspection.IDLModule
			for i, idl := range idls {
				if idl.FilePath == selectIDL {
					selected = idls[i : i+1]
					break
				}
			}
			if len(selected) != 1 {
				w.WriteHeader(404)
				fmt.Fprintf(w, "IDL not found: %q", selectIDL)
				return
			}
			idls = selected
		} else {
			sort.Sort(idls)
		}
		data.Dispatchers = append(data.Dispatchers, dispatcherIdl{
			Name:       disp.Name(),
			ID:         fmt.Sprintf("%p", disp),
			IDLModules: idls,
		})
	}

	if selectIDL != "" {
		if len(data.Dispatchers) == 0 {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Dispatcher %q not found", selectDispatcher)
			return
		}
		idl := data.Dispatchers[0].IDLModules[0]
		fmt.Fprint(w, idl.RawContent)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := idlsPageTmpl.Execute(w, data); err != nil {
		log.Printf("yarpc/debug/idl: Failed executing template: %v", err)
	}
}

var idlsPageTmpl = template.Must(
	template.New("idls").Funcs(template.FuncMap{}).Parse(idlsPageHTML))

const idlsPageHTML = `
<html>
	<head>
	<title>/debug/yarpc/idl</title>
	<style type="text/css">
		body {
			font-family: "Courier New", Courier, monospace;
		}
		table {
			color:#333333;
			border-width: 1px;
			border-color: #3A3A3A;
			border-collapse: collapse;
		}
		table th {
			border-width: 1px;
			padding: 8px;
			border-style: solid;
			border-color: #3A3A3A;
			background-color: #B3B3B3;
		}
		table td {
			border-width: 1px;
			padding: 8px;
			border-style: solid;
			border-color: #3A3A3A;
			background-color: #ffffff;
		}
		header::after {
			content: "";
			clear: both;
			display: table;
		}
		h1 {
			width: 40%;
			float: left;
			margin: 0;
		}
		div.dependencies {
			width: 60%;
			float: left;
			font-size: small;
			text-align: right;
		}
	</style>
	</head>
	<body>

<header>
<h1>/debug/yarpc/idl</h1>
<div class="dependencies">
	{{range .PackageVersions}}
	<span>{{.Name}}={{.Version}}</span>
	{{end}}
</div>
</header>

{{range .Dispatchers}}
	<hr />
	<h2>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h2>
	<table>
		<tr>
			<th>File</th>
			<th>SHA1</th>
			<th>Includes</th>
		</tr>
		{{range .IDLModules}}
		<tr>
			<td>{{.FilePath}}</td>
			<td>{{.SHA1}}</td>
			<td>
			{{ range .Includes }}
				{{ .FilePath }}
			{{ end }}
			</td>
		</tr>
		{{end}}
	</table>
{{end}}
	</body>
</html>
`
