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

package circus

import "go.uber.org/yarpc/api/peer"

type subscriber struct {
	circus      *Circus
	index       int
	boundFinish func(error)
}

func newSubscriber(c *Circus, index int) *subscriber {
	s := &subscriber{
		circus: c,
		index:  index,
	}
	s.boundFinish = s.finish
	return s
}

// NotifyStatusChanged makes a node in a peer list suitable as
// a peer Subscriber, so it can receive notifications from a peer
// when it becomes available, unavailable, or when its pending request count
// changes.
func (s *subscriber) NotifyStatusChanged(_ peer.Identifier) {
	s.circus.lockNotifyStatusChanged(s.index)
}

func (s *subscriber) finish(err error) {
	s.circus.finish(s, s.index, err)
}

func (pl *Circus) finish(s *subscriber, index int, err error) {
	node := pl.nodes[index]
	peer := node.peer
	// TODO decrement pending request count and adjust ring position (again, incrementally)
	pending := node.pending
	pl.popFromCircus(index)
	pl.pushToCircus(index, pending-1)
	peer.EndRequest()
}
