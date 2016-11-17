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
	"fmt"
	"time"

	"go.uber.org/yarpc/transport"
	tInternal "go.uber.org/yarpc/transport/internal"
	"go.uber.org/yarpc/transport/redis/internal"
	redis5 "gopkg.in/redis.v5"
)

type handler struct {
	registry transport.Registry
	deps     transport.Deps

	client *redis5.Client
	stop   chan struct{}
}

func newHandler(
	service transport.ServiceDetail,
	deps transport.Deps,
	client *redis5.Client,
) *handler {
	return &handler{
		registry: service.Registry,
		deps:     deps,
		client:   client,
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
	cmd := h.client.BRPopLPush(queueKey, queueProcessingKey, time.Second)
	if res, err := cmd.Result(); res == "" || err != nil {
		return
	}

	item, err := cmd.Bytes()
	if err != redis5.Nil && err != nil {
		fmt.Println(err, "one")
		return
	}

	req, err := internal.UnmarshalRequest(item)
	if err != nil {
		fmt.Println(err, "two", item, cmd.String())
		return
	}

	ctx := context.Background()

	spec, err := h.registry.Choose(ctx, req)
	if err != nil || spec.Oneway() == nil {
		fmt.Println(err, "three")
		h.remove(item)
		return
	}

	err = tInternal.SafelyCallOnewayHandler(ctx, spec.Oneway(), req)
	if err != nil {
		fmt.Println(err, "four")
	}
	h.remove(item)
}

func (h *handler) remove(item []byte) {
	h.client.LRem(queueProcessingKey, 1, item)
}

func (h *handler) Stop() error {
	h.stop <- struct{}{}
	return h.client.Close()
}
