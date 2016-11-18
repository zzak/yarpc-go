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

package redis

import (
	"context"
	"time"

	"go.uber.org/yarpc/transport"
	tInternal "go.uber.org/yarpc/transport/internal"
	"go.uber.org/yarpc/transport/redis/internal"
)

type handler struct {
	registry transport.Registry
	deps     transport.Deps

	server QueueServer
	stop   chan struct{}
}

func newHandler(
	service transport.ServiceDetail,
	deps transport.Deps,
	server QueueServer,
) *handler {
	return &handler{
		registry: service.Registry,
		deps:     deps,
		server:   server,
		stop:     make(chan struct{}),
	}
}

func (h *handler) StartBackground() {
	go h.start()
}

func (h *handler) start() {
	for {
		select {
		case <-h.stop:
			return
		default:
			h.handle()
		}
	}
}

func (h *handler) handle() {
	item, err := h.server.BRPopLPush(time.Second)
	if err != nil {
		return
	}

	req, err := internal.UnmarshalRequest(item)
	if err != nil {
		h.server.LRem(item)
		return
	}

	ctx := context.Background()

	spec, err := h.registry.Choose(ctx, req)
	if err != nil || spec.Type() != transport.Oneway {
		h.server.LRem(item)
		return
	}

	tInternal.SafelyCallOnewayHandler(ctx, spec.Oneway(), req)
	h.server.LRem(item)
}

func (h *handler) Stop() {
	h.stop <- struct{}{}
}
