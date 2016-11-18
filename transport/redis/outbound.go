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
	"go.uber.org/yarpc/transport/redis/internal"
)

type outbound struct {
	host string
	port int

	client QueueClient
}

// NewOnewayOutbound creates a redis transport.OnewayOutbound
func NewOnewayOutbound(client QueueClient) transport.OnewayOutbound {
	return &outbound{client: client}
}

func (o *outbound) Start(deps transport.Deps) error {
	return o.client.Start()
}

func (o *outbound) Stop() error {
	return o.client.Stop()
}

type ack time.Time

func (a ack) String() string {
	return a.String()
}

func (o *outbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	marshalledReq, err := internal.MarshalRequest(req)
	if err != nil {
		return nil, err
	}

	err = o.client.LPush(marshalledReq)
	ack := time.Now()

	if err != nil {
		return nil, err
	}

	return ack, nil
}
