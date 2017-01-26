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

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

const (
	freeHeadIndex = iota
	unusedHeadIndex
	connectingHeadIndex
	circusHeadIndex

	initialCapacity = 32
)

// TODO connection warming (new peer becomes temporarily unavailable while
// ramping up throughput)
// TODO choose to retain added peers randomly

var maxInt = int(^uint(0) >> 1)

// Circus is a load balancing peer list and peer chooser.
// An n-ring circus has that many circular doubly-linked lists, one for each
// non-empty class of peers, sharing a certain number of pending requests.
// The rings themselves are co-allocated in a single array for tight memory
// locality.
type Circus struct {
	sync.Mutex

	transport peer.Transport
	Monitor   Monitor

	// The number of desired retained peers.  The default is to connect all
	// peers.
	goal int
	// The number of peers that are currently unused (not retained, no peer
	// obtained from transport, not connecting, not available).
	unused int
	// The number of peers that are attempting to connect.
	// The Peer is responsible for informing each subscribing List
	// when it connects. Connecting moves the peer to the least pending
	// available list.
	connecting int
	// The number of peers that are available to take requests.
	// The Peer is responsible for informing each subscribing List
	// when it disconnects.  Disconnecting moves the peer to the
	// connecting peers list.  Peers in the connecting peers list must have a
	// ringIndex of -1 to indicate that Finish messages must be ignored.
	// Peers that are unused must have a peer reference of nil.
	available int

	// all peers have a node, some nodes are head nodes of lists,
	// some are free. node 0 is the head of the free list for nodes.
	// node 1 is the head of the free list for nodes for unretained peers
	// (peers that are neither connecting or available).
	nodes []node
	// locator is a map from peer id to the index of the peer's node.
	locator map[string]int // TODO use identifier as key

	// onPeerAvailable is a channel with room for one notification that at least one
	// peer has become available, allowing Choose to resume and check
	// for an available peer. This event gets emitted each time a peer
	// sends a NotifyStatusChanged message that promotes a peer from connecting
	// to available.
	onPeerAvailable chan struct{}
	onStop          chan struct{}
}

type node struct {
	// available:  peer != nil, ringIndex != -1, retained
	// connecting: peer != nil, ringIndex == -1, retained
	// unused:     peer == nil, ringIndex == -1, released

	id peer.Identifier
	// nil indicates that the peer is unused (not connecting and not available)
	peer peer.Peer
	// -1 indicates that the peer is unavailable (either connecting or unused)
	ringIndex int
	nextIndex int
	prevIndex int
	pending   int

	subscriber *subscriber
}

func (n node) available() bool {
	return n.peer != nil && n.ringIndex != -1
}

func (n node) connecting() bool {
	return n.peer != nil && n.ringIndex == -1
}

func (n node) unused() bool {
	return n.peer == nil && n.ringIndex == -1
}

func (n node) connectionStatus() peer.ConnectionStatus {
	if n.peer == nil {
		return peer.Available
	} else if n.ringIndex == -1 {
		return peer.Connecting
	} else {
		return peer.Unavailable
	}
}

func (n node) String() string {
	r := ""
	if n.peer != nil {
		r = " " + n.peer.Status().String()
	}
	return fmt.Sprintf("{%d <- %v%s ring:%d -> %d}", n.prevIndex, n.id, r, n.ringIndex, n.nextIndex)
}

type ring struct {
	headIndex int
	pending   int
}

func (r ring) String() string {
	return fmt.Sprintf("{Pending:%d Head:%d}", r.pending, r.headIndex)
}

func New(transport peer.Transport) *Circus {
	pl := newCircus()
	pl.transport = transport
	// This creates the head of the unused peer list, implicitly at index 1 as
	// reflected by the unusedHeadIndex constant.
	pl.retainNode()
	// This creates the head of the connecting peer list, implicitly at index 2
	// as reflected by the connectingHeadIndex constant.
	// This list exists so we can release peers while they attempt to connect
	// if the goal connections state changes.
	pl.retainNode()
	// This creates the head of the connected rings list (the circus).
	pl.retainNode()
	return pl
}

// As a convenience for tests, this creates a bare circus peer list, without
// the head nodes for unused and connecting peers.
func newCircus() *Circus {
	return &Circus{
		goal:       maxInt,
		connecting: 0,
		available:  0,
		// The zero value of a node serves as the head of the free list
		nodes: make([]node, 1, initialCapacity),
		// Initially empty locator for peers by identifier as string
		locator:         make(map[string]int),
		onPeerAvailable: make(chan struct{}, 1),
		onStop:          make(chan struct{}, 0),
	}
}

func (pl *Circus) Start() error {
	pl.Lock()
	defer pl.Unlock()
	pl.satisfyGoal()
	return nil
}

func (pl *Circus) Stop() error {
	pl.Lock()
	defer pl.Unlock()
	pl.goal = 0
	pl.satisfyGoal()
	return nil
}

func (pl *Circus) IsRunning() bool {
	// TODO
	return true
}

func (pl *Circus) SetGoal(goal int) {
	pl.Lock()
	defer pl.Unlock()
	pl.goal = goal
	pl.satisfyGoal()
}

func (pl *Circus) Update(updates peer.ListUpdates) error {
	pl.Lock()
	defer pl.Unlock()

	add := updates.Additions
	remove := updates.Removals

	if pl.Monitor != nil {
		pl.Monitor.Update()
	}

	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	// TODO Remove
	// Add
	for _, pid := range add {
		index := pl.retainNode()
		node := &pl.nodes[index]
		node.id = pid
		pl.push(index, unusedHeadIndex)
		pl.locator[pid.Identifier()] = index
		pl.unused++
	}

	pl.satisfyGoal()
	return nil
}

func (pl *Circus) Choose(ctx context.Context, _ *transport.Request) (peer.Peer, func(error), error) {
	for {
		pl.Lock()
		if pl.available > 0 {

			// We may have consumed a peer changed message that was intended
			// for multiple subscribers.  We must warn the others.
			pl.notifyPeerAvailable()

			node := pl.getLeastPendingNode()
			pl.Unlock()
			node.peer.StartRequest()
			return node.peer, node.subscriber.boundFinish, nil
		}
		pl.Unlock()

		select {
		case <-pl.onPeerAvailable:
		case <-pl.onStop:
			// TODO error type
			return nil, nil, fmt.Errorf("server stopped while waiting for an available peer")
		case <-ctx.Done():
			// TODO wrapped error type maybe, consider behaviors
			return nil, nil, ctx.Err()
		}
	}
}

func (pl *Circus) dump() {
	// TODO lol
	// fmt.Printf("circus: unused:%d connecting:%d available:%d\n", pl.unused, pl.connecting, pl.available)
	// fmt.Printf("rings:")
	// for i, ring := range pl.rings {
	// 	if i != 0 {
	// 		fmt.Printf(",")
	// 	}
	// 	fmt.Printf(" {%d pending at %d}", ring.pending, ring.headIndex)
	// }
	// fmt.Printf("\n")
	// for i, node := range pl.nodes {
	// 	fmt.Printf("%d. %v", i, node)
	// 	if i == freeHeadIndex {
	// 		fmt.Printf(" (free)")
	// 	} else if i == unusedHeadIndex {
	// 		fmt.Printf(" (unused)")
	// 	} else if i == connectingHeadIndex {
	// 		fmt.Printf(" (connecting)")
	// 	} else {
	// 		for _, ring := range pl.rings {
	// 			if i == ring.headIndex {
	// 				fmt.Printf(" (%d pending)", ring.pending)
	// 			}
	// 		}
	// 	}
	// 	fmt.Printf("\n")
	// }
}

func (pl *Circus) satisfyGoal() {
	for pl.goal > pl.connecting+pl.available && pl.unused > 0 {
		err := pl.retainPeer()
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	if pl.available > 0 {
		pl.notifyPeerAvailable()
	}
	// TODO clamp connections when goal state lowered
	// for pl.goal < pl.connecting+pl.available && pl.connecting > 0 {
	// }
	// for pl.goal < pl.connecting+pl.available {
	// }
}

func (pl *Circus) retainPeer() error {
	head := &pl.nodes[unusedHeadIndex]
	index := head.nextIndex
	node := &pl.nodes[index]
	subscriber := newSubscriber(pl, index)
	peer, err := pl.transport.RetainPeer(node.id, subscriber)
	if pl.Monitor != nil {
		pl.Monitor.RetainPeer(peer.Identifier())
	}
	// TODO handle the assertion error cases (e.g., resubscription) (probably just logging)
	node.peer = peer

	pl.pop(index)
	pl.push(index, connectingHeadIndex)

	pl.unused--
	pl.connecting++
	pl.notifyStatusChanged(index)

	return err
}

func (pl *Circus) getLeastPendingNode() *node {
	circusHead := &pl.nodes[circusHeadIndex]
	leastPendingCircusIndex := circusHead.prevIndex
	leastPendingCircus := &pl.nodes[leastPendingCircusIndex]
	headIndex := leastPendingCircus.ringIndex
	head := &pl.nodes[headIndex]
	index := head.nextIndex
	pl.pop(index)
	// TODO promote the used node to the next pending ring instead of recycling
	pl.push(index, headIndex)
	return &pl.nodes[index]
}

// returns the index of the head node of the ring with the least pending requests.
func (pl *Circus) getLeastPendingRingHeadIndex(pending int) int {
	circusHead := &pl.nodes[circusHeadIndex]
	if pl.empty(circusHeadIndex) {
		leastPendingCircusIndex := pl.retainNode()
		pl.push(leastPendingCircusIndex, circusHeadIndex)
		leastPendingCircus := &pl.nodes[leastPendingCircusIndex]
		leastPendingCircus.pending = pending
		ringHeadIndex := pl.retainNode()
		ringHead := &pl.nodes[ringHeadIndex]
		leastPendingCircus.ringIndex = ringHeadIndex
		ringHead.ringIndex = leastPendingCircusIndex
		return ringHeadIndex
	}
	leastPendingCircusIndex := circusHead.prevIndex
	leastPendingCircus := &pl.nodes[leastPendingCircusIndex]
	return leastPendingCircus.ringIndex
}

func (pl *Circus) getPendingRingHeadIndex(pending, nearRingHeadIndex int) int {
	// TODO search from a starting ring head index for the head of the pending
	// request ring with the given pending request count, or insert a head node
	// in the right position and return its index with the expectation that
	// ring will be populated.
	return 0
}

func (pl *Circus) lockNotifyStatusChanged(index int) {
	pl.Lock()
	defer pl.Unlock()
	pl.notifyStatusChanged(index)
}

func (pl *Circus) notifyPeerAvailable() {
	select {
	case pl.onPeerAvailable <- struct{}{}:
	default:
	}
}

func (pl *Circus) notifyStatusChanged(index int) {
	node := &pl.nodes[index]
	p := node.peer

	status := p.Status()

	// if pl.Monitor != nil {
	// 	fmt.Printf("status change %v %v %v\n", p.Identifier(), status, node)
	// 	fmt.Printf("before\n")
	// 	pl.dump()
	// }

	// A peer has become available.
	if status.ConnectionStatus == peer.Available && !node.available() {
		// 	if pl.Monitor != nil {
		// 		fmt.Printf("%v became available\n", node)
		// 	}
		if node.connecting() {
			pl.connecting--
		} else if node.unused() {
			pl.unused--
		}
		pl.available++

		// Remove the node from the connecting peer list
		pl.pop(index)
		// Add to the least pending ring
		pl.push(index, pl.getLeastPendingRingHeadIndex(status.PendingRequestCount))
		// TODO search for the correct ring for the given pending request count

		// Non-blocking notification to goroutines blocked on Choose that
		// they may resume and check for an available peer.
		pl.notifyPeerAvailable()

		// 	if pl.Monitor != nil {
		// 		fmt.Printf("after\n")
		// 		pl.dump()
		// 	}

		return
	}

	// A peer has become unavailable.
	// If the peer is no longer connected, move it to the connecting list,
	// awaiting connection notification (peer is obliged to attempt to
	// reconnect until we release the node)
	// TODO consider ranking peers by number of failed connection attempts,
	// release bad peers, retain good ones
	// if status.ConnectionStatus != peer.Available && node.available() {
	// 	if pl.Monitor != nil {
	// 		fmt.Printf("%v became unavailable\n", node)
	// 	}

	// 	pl.available--
	// 	pl.popFromRing(index, node.ringIndex)
	// 	if node.unused() {
	// 		pl.unused++
	// 		pl.push(index, unusedHeadIndex)
	// 	} else if node.connecting() {
	// 		pl.connecting++
	// 		pl.pop(index)
	// 		pl.push(index, connectingHeadIndex)
	// 	}

	// }

	// // TODO handle ConnectionFailed status (release and retain a different
	// // peer)

	// // If the peer is connected and available, consider adjusting its ring for
	// // its current pending request count.
	// if node.ringIndex != -1 {
	// 	ring := pl.rings[node.ringIndex]
	// 	if status.PendingRequestCount != ring.pending {
	// 		if pl.Monitor != nil {
	// 			fmt.Printf("adjusted pending request count\n")
	// 		}
	// 		pl.adjustRing(index, node.ringIndex, status.PendingRequestCount)
	// 	}
	// }

	// if pl.Monitor != nil {
	// 	fmt.Printf("after\n")
	// 	pl.dump()
	// }
}
