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
	"net/http"
	"sort"
	"strings"

	"go.uber.org/yarpc/internal/introspection"
)

type dispatcherIdl struct {
	Name       string
	ID         string
	IDLModules []introspection.IDLModule
	IDLTree    introspection.IDLTree
}

type IDLTreeHelper struct {
	DispatcherName string
	Indent         int
	Tree           *introspection.IDLTree
}

func wrapIDLTree(dname string, i int, t *introspection.IDLTree) IDLTreeHelper {
	sort.Sort(t.Modules)
	return IDLTreeHelper{dname, i + 1, t}
}

var idlPage = page{
	path: "/debug/yarpc/idl/",
	handler: func(w http.ResponseWriter, req *http.Request, is IntrospectionProvider) interface{} {
		path := strings.TrimPrefix(req.URL.Path, "/debug/yarpc/idl/")
		parts := strings.SplitN(path, "/", 2)
		var selectDispatcher string
		var selectIDL string
		if path != "" {
			if len(parts) != 2 {
				w.WriteHeader(400)
				fmt.Fprintf(w, "Invalid arguments")
				return nil
			}
			selectDispatcher = parts[0]
			selectIDL = parts[1]
		}

		var dispatchers []introspection.DispatcherStatus
		if selectDispatcher != "" {
			dispatchers = is.DispatchersByName(selectDispatcher)

			if len(dispatchers) == 0 {
				w.WriteHeader(404)
				fmt.Fprintf(w, "dispatcher %q not found", selectDispatcher)
				return nil
			}
		} else {
			dispatchers = is.Dispatchers()
		}

		data := struct {
			Dispatchers     []dispatcherIdl
			PackageVersions []introspection.PackageVersion
		}{
			PackageVersions: is.PackageVersions(),
		}

		for _, d := range dispatchers {
			idls := d.Procedures.IDLModules()
			if selectIDL != "" {
				var selected []introspection.IDLModule
				for i, idl := range idls {
					if idl.FilePath == selectIDL {
						selected = idls[i : i+1]
						break
					}
				}
				idls = selected
			} else {
				sort.Sort(idls)
			}
			idltree := d.Procedures.IDLTree()
			idltree.Compact()
			data.Dispatchers = append(data.Dispatchers, dispatcherIdl{
				Name:       d.Name,
				ID:         d.ID,
				IDLModules: idls,
				IDLTree:    idltree,
			})
		}

		if selectDispatcher == "" {
			return data
		}

		for idx, d := range data.Dispatchers {
			if len(data.Dispatchers) > 1 {
				fmt.Fprintf(w, "Dispatcher %q #%d:\n", d.Name, idx)
			}
			if len(d.IDLModules) != 1 {
				if len(data.Dispatchers) == 1 {
					w.WriteHeader(404)
				}
				fmt.Fprintf(w, "IDL not found: %q\n", selectIDL)
				continue
			}
			fmt.Fprintf(w, d.IDLModules[0].RawContent)
		}
		return nil
	},
	html: `
{{ define "title"}}/debug/yarpc/idl{{ end }}
{{ define "body" }}
{{range .Dispatchers}}
	<hr />
	<h2>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h2>
	<table class="tree">
		<tr>
			<th>File</th>
			<th>SHA1</th>
			<th>Includes</th>
		</tr>
		{{ template "idltree" (wrapIDLtree .Name -1 .IDLTree) }}
	</div>
{{end}}
{{end}}
{{ define "idltree" }}
{{ $dname := .DispatcherName }}
{{ $indent := .Indent }}
{{ with .Tree }}
	{{range .Modules}}
		<tr>
			<td style="padding-left: {{ $indent }}em">
				<div class="filename" id="{{$dname}}/{{.FilePath}}">
					<a href="{{$dname}}/{{.FilePath}}">{{pathBase .FilePath}}</a>
				<div>
			</td>
			<td class="sha1">
				{{ .SHA1 }}
			</td>
			<td class="includes">
				{{ range .Includes }}
					<a class="anchor" href="#{{$dname}}/{{.FilePath}}">{{pathBase .FilePath}}</a>
				{{ end }}
			</td>
		</tr>
	{{end}}
	{{range $dir, $subTree := .SubTrees}}
		<tr>
			<td style="padding-left: {{ $indent }}em">
				<div class="filename">
					{{ $dir }}/
				</div>
			</td>
		</tr>
		{{ template "idltree" (wrapIDLtree $dname $indent $subTree) }}
	{{end}}
</tr>
{{ end }}
{{ end }}
`,
}
