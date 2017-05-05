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

package introspection

import (
	"strings"

	"go.uber.org/yarpc/api/transport"
)

// Procedure represent a registered procedure on a dispatcher.
type Procedure struct {
	Name          string     `json:"name"`
	Encoding      string     `json:"encoding"`
	Signature     string     `json:"signature"`
	RPCType       string     `json:"rpcType"`
	IDLEntryPoint *IDLModule `json: "idlTree"`
}

// Procedures is a slice of Procedure.
type Procedures []Procedure

// IntrospectProcedures is a convenience function that translate a slice of
// transport.Procedure to a slice of introspection.Procedure. This output is
// used in debug and yarpcmeta.
func IntrospectProcedures(routerProcs []transport.Procedure) Procedures {
	procedures := make([]Procedure, 0, len(routerProcs))
	for _, p := range routerProcs {
		var spec interface{}
		switch p.HandlerSpec.Type() {
		case transport.Unary:
			spec = p.HandlerSpec.Unary()
		case transport.Oneway:
			spec = p.HandlerSpec.Oneway()
		}
		var IDLEntryPoint *IDLModule
		if spec != nil {
			if i, ok := spec.(IntrospectableHandler); ok {
				if i := i.Introspect(); i != nil {
					IDLEntryPoint = i.IDLEntryPoint
				}
			}
		}
		procedures = append(procedures, Procedure{
			Name:          p.Name,
			Encoding:      string(p.Encoding),
			Signature:     p.Signature,
			RPCType:       p.HandlerSpec.Type().String(),
			IDLEntryPoint: IDLEntryPoint,
		})
	}
	return procedures
}

// IDLModule is a generic IDL module. For example, a thrift file or a protobuf
// one.
type IDLModule struct {
	FilePath   string
	SHA1       string
	Includes   []IDLModule
	RawContent string
}

type IDLModules []IDLModule

// IDLModules returns a flat map of all IDLModules used across all procedures.
func (ps Procedures) IDLModules() IDLModules {
	seen := make(map[string]struct{})
	var r []IDLModule
	var collect func(m IDLModule)
	collect = func(m IDLModule) {
		if _, ok := seen[m.FilePath]; !ok {
			seen[m.FilePath] = struct{}{}
			r = append(r, m)
		}
		for _, i := range m.Includes {
			collect(i)
		}
	}
	for _, p := range ps {
		if p.IDLEntryPoint != nil {
			collect(*p.IDLEntryPoint)
		}
	}
	return r
}

func (ims IDLModules) Len() int {
	return len(ims)
}

func (ims IDLModules) Less(i int, j int) bool {
	return ims[i].FilePath < ims[j].FilePath
}

func (ims IDLModules) Swap(i int, j int) {
	ims[i], ims[j] = ims[j], ims[i]
}

type IDLTree struct {
	SubTrees map[string]*IDLTree
	Modules  IDLModules
}

func (ps Procedures) IDLTree() IDLTree {
	seen := make(map[string]struct{})
	var r IDLTree
	var collect func(m IDLModule)
	collect = func(m IDLModule) {
		if _, ok := seen[m.FilePath]; !ok {
			seen[m.FilePath] = struct{}{}
			n := &r
			parts := strings.Split(m.FilePath, "/")
			for i, part := range parts {
				if i == len(parts)-1 {
					continue
				}
				if n.SubTrees == nil {
					newNode := IDLTree{}
					n.SubTrees = map[string]*IDLTree{part: &newNode}
					n = &newNode
				} else {
					if subNode, ok := n.SubTrees[part]; ok {
						n = subNode
					} else {
						newNode := IDLTree{}
						n.SubTrees[part] = &newNode
						n = &newNode
					}
				}
			}
			n.Modules = append(n.Modules, m)
		}
		for _, i := range m.Includes {
			collect(i)
		}
	}
	for _, p := range ps {
		if p.IDLEntryPoint != nil {
			collect(*p.IDLEntryPoint)
		}
	}
	return r
}

func (it *IDLTree) Compact() {
	for _, l1tree := range it.SubTrees {
		l1tree.Compact()
	}
	for l1dir, l1tree := range it.SubTrees {
		if len(l1tree.SubTrees) == 1 && len(l1tree.Modules) == 0 {
			for l2dir, l2tree := range l1tree.SubTrees {
				compactedDir := l1dir + "/" + l2dir
				it.SubTrees = map[string]*IDLTree{compactedDir: l2tree}
				break
			}
		}
	}
}
