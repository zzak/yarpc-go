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
	"gopkg.in/redis.v5"
)

type outbound struct {
	host string
	port int

	client *redis.Client
}

// NewOutbound creates a redis transport.OnewayOutbound
func NewOutbound(host string, port int) transport.OnewayOutbound {
	return &outbound{
		host: host,
		port: port,
	}
}

func (o *outbound) Start(deps transport.Deps) error {
	client, err := NewRedis5Client(o.host, o.port)
	if err != nil {
		return err
	}

	o.client = client
	_, err = o.client.Ping().Result()
	return err
}

func (o *outbound) Stop() error {
	return o.client.Close()
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

	cmd := o.client.LPush(queueKey, marshalledReq)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	ack := time.Now()
	return ack, nil
}
