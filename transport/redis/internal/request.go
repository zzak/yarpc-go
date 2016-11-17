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

package internal

import (
	"bytes"
	"io/ioutil"

	"go.uber.org/yarpc/transport"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

// MarshalRequest encodes a transport.Request into bytes
func MarshalRequest(treq *transport.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return nil, err
	}

	req := Request{
		Caller:          treq.Caller,
		ServiceName:     treq.Service,
		Encoding:        string(treq.Encoding),
		Procedure:       treq.Procedure,
		Headers:         treq.Headers.Items(),
		ShardKey:        &treq.ShardKey,
		RoutingKey:      &treq.RoutingKey,
		RoutingDelegate: &treq.RoutingDelegate,
		Body:            body,
	}

	wireValue, err := req.ToWire()
	if err != nil {
		return nil, err
	}

	writer := bytes.NewBuffer([]byte{})
	err = protocol.Binary.Encode(wireValue, writer)
	return writer.Bytes(), err
}

// UnmarshalRequest decodes bytes into a transport.Request
func UnmarshalRequest(request []byte) (*transport.Request, error) {
	reader := bytes.NewReader(request)
	wireValue, err := protocol.Binary.Decode(reader, wire.TStruct)
	if err != nil {
		return nil, err
	}

	req := Request{}
	err = req.FromWire(wireValue)
	if err != nil {
		return nil, err
	}

	treq := transport.Request{
		Caller:          req.Caller,
		Service:         req.ServiceName,
		Encoding:        transport.Encoding(req.Encoding),
		Procedure:       req.Procedure,
		Headers:         transport.HeadersFromMap(req.Headers),
		ShardKey:        *req.ShardKey,
		RoutingKey:      *req.RoutingKey,
		RoutingDelegate: *req.RoutingDelegate,
		Body:            bytes.NewBuffer(req.Body),
	}

	return &treq, nil
}
