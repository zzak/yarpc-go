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

import "fmt"

func (pl *Circus) empty(headIndex int) bool {
	head := pl.nodes[headIndex]
	return head.nextIndex == headIndex
}

func (pl *Circus) alone(headIndex int) bool {
	head := pl.nodes[headIndex]
	// bail if empty
	if head.nextIndex == headIndex {
		return false
	}
	next := &pl.nodes[head.nextIndex]
	return next.nextIndex == headIndex
}

func (pl *Circus) pop(index int) {
	node := &pl.nodes[index]
	next := &pl.nodes[node.nextIndex]
	prev := &pl.nodes[node.prevIndex]
	next.prevIndex = node.prevIndex
	prev.nextIndex = node.nextIndex
	node.prevIndex = index
	node.nextIndex = index
}

func (pl *Circus) push(index int, nextIndex int) {
	node := &pl.nodes[index]
	next := &pl.nodes[nextIndex]
	prev := &pl.nodes[next.prevIndex]
	prev.nextIndex = index
	node.nextIndex = nextIndex
	node.prevIndex = next.prevIndex
	next.prevIndex = index
}

// retainNode returns the index of an unused node.
// The returned node has meaningless nextIndex and prevIndex
// values and is not a member of any list, not even the free list.
// The receiver of the retained node must add the node to a ring.
func (pl *Circus) retainNode() int {
	if pl.empty(freeHeadIndex) {
		// Grow the collection of nodes
		index := len(pl.nodes)
		subscriber := &subscriber{
			circus: pl,
			index:  index,
		}
		node := node{
			ringIndex:  -1,
			nextIndex:  index,
			prevIndex:  index,
			subscriber: subscriber,
		}
		// Allocate a finish closure once
		subscriber.boundFinish = subscriber.finish
		pl.nodes = append(pl.nodes, node)
		return index
	}
	freeListHead := &pl.nodes[freeHeadIndex]
	index := freeListHead.nextIndex
	pl.pop(index)

	// Reset the node so that it is an empty ring.
	// TODO assert that the node is an empty ring with no data
	return index
}

func (pl *Circus) releaseNode(index int) error {
	node := &pl.nodes[index]

	if node.id != nil {
		return fmt.Errorf("node at %d expected to have no id when released", index)
	}
	if node.peer != nil {
		return fmt.Errorf("node at %d expected to have no peer reference when released", index)
	}
	if node.nextIndex != index {
		return fmt.Errorf("node at %d expected to be an empty ring (next index is own index)", index)
	}
	if node.prevIndex != index {
		return fmt.Errorf("node at %d expected to be an empty ring (prev index is own index)", index)
	}
	if node.ringIndex != -1 {
		return fmt.Errorf("node at %d expected to have ring index of -1 indicating that it is not on any ring", index)
	}

	// Add the node to the free list
	pl.push(index, freeHeadIndex)
	return nil
}

func (pl *Circus) walk(headIndex int, f func(int, *node)) {
	head := &pl.nodes[headIndex]
	index := head.nextIndex
	node := &pl.nodes[index]
	for node != head {
		f(index, node)
		index = node.nextIndex
		node = &pl.nodes[index]
	}
}
