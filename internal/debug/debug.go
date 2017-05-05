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

package debug

import (
	"fmt"
	"html/template"
	"net/http"
	"path"

	"go.uber.org/yarpc/internal/introspection"
)

type HTTPMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type IntrospectionProvider interface {
	Dispatchers() []introspection.DispatcherStatus
	DispatchersByName(name string) []introspection.DispatcherStatus
	PackageVersions() []introspection.PackageVersion
}

type page struct {
	path    string
	handler func(w http.ResponseWriter, req *http.Request,
		is IntrospectionProvider) interface{}
	html string
	tmpl *template.Template
}

type DebugPages struct {
	is    IntrospectionProvider
	pages map[string]*page
}

func (dp *DebugPages) registerPage(page *page) {
	funcmap := map[string]interface{}{
		"pathBase":    path.Base,
		"wrapIDLtree": wrapIDLTree,
	}

	// We do not .Clone() the base template, but reparse it every time. Because
	// of a race condition/memory leak when template.Clone() is used in
	// conjunction with template blocks on Go<=1.8.
	base := template.Must(template.New("base").Funcs(funcmap).Parse(baseHTML))
	page.tmpl = template.Must(base.Parse(page.html))
	dp.pages[page.path] = page
}

func (dp *DebugPages) executePage(w http.ResponseWriter, req *http.Request, page *page) {
	data := page.handler(w, req, dp.is)
	if data == nil {
		return
	}
	if err := page.tmpl.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Failed executing template: %v", err)
		return
	}
}

func (dp *DebugPages) newPageHandler(page *page) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		dp.executePage(w, req, page)
	}
}

func NewDebugPages(is IntrospectionProvider) *DebugPages {
	r := DebugPages{
		is:    is,
		pages: make(map[string]*page),
	}
	r.registerPage(&rootPage)
	r.registerPage(&idlPage)
	return &r
}

func (dp *DebugPages) RegisterOn(mux HTTPMux) {
	for _, page := range dp.pages {
		mux.HandleFunc(page.path, dp.newPageHandler(page))
	}
}

const baseHTML = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>{{ block "title" . }}yarpc{{ end }}</title>
		<style type="text/css">
			body {
				font-family: "Courier New", Courier, monospace;
			}
			table.spreadsheet {
				color:#333333;
				border-width: 1px;
				border-color: #3A3A3A;
				border-collapse: collapse;
			}
			table.spreadsheet th {
				border-width: 1px;
				padding: 8px;
				border-style: solid;
				border-color: #3A3A3A;
				background-color: #B3B3B3;
			}
			table.spreadsheet td {
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
			table.tree td {
				padding-left: 1em;
				padding-right: 1em;
			}
			:target {
				background-color: #ffa;
			}
		</style>
	</head>
	<body>
		<header>
		<h1>{{ template "title" . }}</h1>
		<div class="dependencies">
			{{range .PackageVersions}}
			<span>{{.Name}}={{.Version}}</span>
			{{end}}
		</div>
		</header>
		{{ block "body" . }}Something here{{ end }}
	</body>
</html>
`
