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

// func (pl *Circus) adjustRing(index, ringIndex, pending int) error {
// 	// Assertion: the pending request count for the current ring is not the
// 	// same as this ring.
// 	// Assertion: the node already exists on some ring, therefore there must
// 	// already be at least one ring.
// 	// Assertion: ringIndex therefore is the index of an existing ring.

// 	// If there is exactly one ring,
// 	if len(pl.rings) == 1 {
// 		// Assertion: the node must already be on this ring.
// 		// If the node is alone on the only ring,
// 		leastPendingRing := &pl.rings[leastPendingRingIndex]
// 		if pl.alone(leastPendingRingIndex) && pending == leastPendingRing.pending {
// 			// We will just keep that one ring and adjust the pending count.
// 			leastPendingRing.pending++
// 			return nil
// 		}
// 	}

// 	// Assertion: there is more than one ring (as is expected to be the common
// 	// case).
// 	currentRing := &pl.rings[ringIndex]

// 	// If the node is alone on its current ring,
// 	if pl.alone(currentRing.headIndex) {
// 		// There is some chance that that we can update the ring's pending
// 		// count instead of moving the node to a neighboring ring.
// 		// If the neighboring ring is for the desired pending request count, or
// 		// if the desired pending request ring is beyond the neighbor, we will
// 		// have to move the node and empty out this ring (causing a shift).
// 		inPlace := false
// 		if pending < currentRing.pending {
// 			if ringIndex > 0 {
// 				fewerRing := pl.rings[ringIndex-1]
// 				// == accounts for merges, > accounts for jumps.
// 				// < indicates that we can adjust the ring instead of the node.
// 				if fewerRing.pending < pending {
// 					inPlace = true
// 				}
// 			}
// 		} else if pending > currentRing.pending {
// 			if ringIndex < len(pl.rings)-1 {
// 				moreRing := pl.rings[ringIndex+1]
// 				// == accounts for merges, < accounts for jumps.
// 				// > indicates that we can adjust the ring instead of the node.
// 				if moreRing.pending > pending {
// 					inPlace = true
// 				}
// 			}
// 		}
// 		if inPlace {
// 			// We can just move the ring's pending count in place.
// 			// This strategy limits the number of rings to the number of unique
// 			// pending request counts, as opposed to having a ring for every
// 			// possible pending request count.
// 			// The bulk of requests will exist in the two most pending request rings,
// 			// with peers occasionally entering from the zero-pending-request-ring
// 			// and getting rapidly promoted (carrying that singleton ring) until
// 			// they merge with the second most pending request ring.
// 			// By chance, a new least pending request ring may form under the two
// 			// most-pending-request-rings when multiple requests complete in rappid
// 			// succession.
// 			// As such, the costly management of rings may increase with the number
// 			// of peers and the variance in latency, but at time of writing, that
// 			// cost is not known and expected to be less than continuous O(log n)
// 			// sifting of peers in a heap.
// 			currentRing.pending = pending
// 		}
// 	}

// 	// In all other cases, which we expect to be common, we move the node to
// 	// the appropriate ring.
// 	// Removing the node from the current ring will often be indexpensive since
// 	// the ring has more than one node, avoiding an O(len(rings-ringIndex))
// 	// shift.
// 	if err := pl.popFromRing(index, ringIndex); err != nil {
// 		return err
// 	}
// 	// It may be necessary to create a new ring for the node, causing a similar
// 	// shift, but that is also unusual.
// 	if err := pl.pushToRing(index, ringIndex, pending); err != nil {
// 		return err
// 	}

// 	return nil
// }

// // ringIndex is advice on where to begin scanning for the ring with the desired
// // pending request count.
// func (pl *Circus) pushToRing(index, ringIndex, pending int) error {
// 	// Assertion: the node does not exist on any ring.
// 	node := &pl.nodes[index]

// 	// If there are no rings, we create one and bail.
// 	// (uncommon)
// 	if len(pl.rings) == 0 {
// 		onlyRing := ring{
// 			headIndex: pl.retainNode(),
// 			pending:   pending,
// 		}
// 		pl.rings = append(pl.rings, onlyRing)
// 		pl.push(index, onlyRing.headIndex)
// 		node.ringIndex = 0
// 		return nil
// 	}

// 	// Assertion: there is now at least one ring.
// 	// Clip the ringIndex to the least pending ring.
// 	// Hereafter, we are guaranteed that there is a ring at ringIndex.
// 	if ringIndex >= len(pl.rings) {
// 		ringIndex = len(pl.rings) - 1
// 	} else if ringIndex < 0 {
// 		// This should not be possible, but would cause a panic.
// 		ringIndex = 0
// 	}

// 	givenRing := pl.rings[ringIndex]
// 	// Scan backward (for rings with fewer pending requests)
// 	for givenRing.pending > pending && ringIndex > 0 {
// 		ringIndex--
// 		givenRing = pl.rings[ringIndex]
// 	}
// 	// Scan forward (for rings with more pending reqests)
// 	for givenRing.pending < pending && ringIndex < len(pl.rings)-1 {
// 		ringIndex++
// 		givenRing = pl.rings[ringIndex]
// 	}
// 	// Create a new ring at the end of the rings if we have yet more pending
// 	// requests than the most pending request ring.
// 	if givenRing.pending < pending {
// 		ringIndex++
// 		givenRing = ring{
// 			headIndex: pl.retainNode(),
// 			pending:   pending,
// 		}
// 		pl.rings = append(pl.rings, givenRing)
// 	}

// 	// Inject in a ring, if necessary
// 	if givenRing.pending != pending {
// 		pl.rings = append(pl.rings, ring{})
// 		// Shift existing rings right
// 		copy(pl.rings[ringIndex+1:], pl.rings[ringIndex:])
// 		// Overwrite ring with the new ring to accommodate this peer's
// 		// pending request count.
// 		pl.rings[ringIndex] = ring{
// 			headIndex: pl.retainNode(),
// 			pending:   pending,
// 		}
// 	}

// 	// All paths that lead here guarantee that the found ring has the
// 	// appropriate pending request count for the node.

// 	// Add the node to the ring
// 	pl.push(index, givenRing.headIndex)
// 	node.ringIndex = ringIndex

// 	return nil
// }

// func (pl *Circus) popFromRing(index, ringIndex int) error {
// 	node := &pl.nodes[index]
// 	pl.pop(index)
// 	if node.ringIndex == -1 {
// 		return nil
// 	}
// 	node.ringIndex = -1
// 	ring := pl.rings[ringIndex]
// 	// If the ring has become empty as a consequence of removing the final node,
// 	if pl.empty(ring.headIndex) {
// 		// Eliminate the ring:
// 		// Shift and truncate.
// 		copy(pl.rings[ringIndex:], pl.rings[ringIndex+1:])
// 		pl.rings = pl.rings[:len(pl.rings)-1]
// 		// Release the ring's head node to the free list.
// 		// The head node never has id, peer, or ringIndex set, so we don't need
// 		// to clear them out.
// 		if err := pl.releaseNode(ring.headIndex); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }
