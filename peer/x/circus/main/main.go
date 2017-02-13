// Copyright (c) 2016 Uber Technologies, Inc.
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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/peer/x/circus"
	"go.uber.org/yarpc/transport/http"
)

type Monitor struct{}

func (m *Monitor) NotifyStatusChanged(peer peer.Peer) {
	fmt.Println("status changed", peer.Identifier(), peer.Status())
}

func (m *Monitor) RetainPeer(addr string) {
	fmt.Println("retained peer", addr)
}

func (m *Monitor) Update() {
	fmt.Println("update")
}

func main() {
	x := http.NewTransport()

	// pl := peerheap.New(x)
	// pl := roundrobin.New(x)
	pl := circus.New(x)
	// pl.Monitor = &Monitor{}
	pl.Dump()

	pl.Start()
	defer pl.Stop()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	peertest.GenerateLoad(ctx, pl, 256)
	peertest.GenerateChaos(ctx, pl, 256)

	time.Sleep(10 * time.Millisecond)

	for n := 0; n < 300; n++ {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		peer, finish, err := pl.Choose(ctx, nil)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(peer.Identifier())
		pl.Dump()
		finish(nil)
		time.Sleep(500 * time.Nanosecond)

		pl.Dump()
	}

	pl.Dump()
}
