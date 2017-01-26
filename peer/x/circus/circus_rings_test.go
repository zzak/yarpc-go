package circus

// func TestRetainReleaseRetain(t *testing.T) {
// 	pl := newCircus()

// 	index := pl.retainNode()
// 	assert.Equal(t, 1, index, "retaining a node should create a node")

// 	assert.Equal(t, true, pl.empty(freeHeadIndex), "free list should be empty initially")
// 	assert.Equal(t, false, pl.alone(freeHeadIndex), "free list should not contain exactly one node")
// 	pl.releaseNode(index)
// 	assert.Equal(t, false, pl.empty(freeHeadIndex), "free list should not be empty after release")
// 	assert.Equal(t, true, pl.alone(freeHeadIndex), "free list should contain exactly one node")

// 	index = pl.retainNode()
// 	assert.Equal(t, 1, index, "retaining after release should reuse a node")
// }

// func TestPushPop(t *testing.T) {
// 	pl := newCircus()
// 	pl.retainNode() // TODO remove this artifact

// 	headIndex := pl.retainNode()

// 	aIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 2<-2->2, 3<-3->3", pl.inspectNodes())
// 	assert.Equal(t, true, pl.empty(headIndex))

// 	pl.push(aIndex, headIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 3<-2->3, 2<-3->2", pl.inspectNodes())
// 	assert.Equal(t, false, pl.empty(headIndex))
// 	assert.Equal(t, true, pl.alone(headIndex))

// 	bIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 3<-2->3, 2<-3->2, 4<-4->4", pl.inspectNodes())

// 	pl.push(bIndex, headIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 4<-2->3, 2<-3->4, 3<-4->2", pl.inspectNodes())
// 	assert.Equal(t, false, pl.alone(headIndex))
// 	assert.Equal(t, false, pl.empty(4))

// 	pl.pop(bIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 3<-2->3, 2<-3->2, 4<-4->4", pl.inspectNodes())
// 	assert.Equal(t, true, pl.empty(4))

// 	pl.push(bIndex, headIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 4<-2->3, 2<-3->4, 3<-4->2", pl.inspectNodes())
// }

// func TestRetainRelease(t *testing.T) {
// 	pl := newCircus()
// 	pl.retainNode() // TODO remove this artifact

// 	headIndex := pl.retainNode()

// 	aIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 2<-2->2, 3<-3->3", pl.inspectNodes())
// 	pl.push(aIndex, headIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 3<-2->3, 2<-3->2", pl.inspectNodes())
// 	bIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 3<-2->3, 2<-3->2, 4<-4->4", pl.inspectNodes())
// 	pl.push(bIndex, headIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 4<-2->3, 2<-3->4, 3<-4->2", pl.inspectNodes())
// 	pl.pop(aIndex)
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 4<-2->4, 3<-3->3, 2<-4->2", pl.inspectNodes())
// 	assert.NoError(t, pl.releaseNode(aIndex))
// 	assert.Equal(t, "3<-0->3, 1<-1->1, 4<-2->4, 0<-3->0, 2<-4->2", pl.inspectNodes())
// 	cIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 4<-2->4, 3<-3->3, 2<-4->2", pl.inspectNodes())
// 	assert.NoError(t, pl.releaseNode(cIndex))
// 	assert.Equal(t, "3<-0->3, 1<-1->1, 4<-2->4, 0<-3->0, 2<-4->2", pl.inspectNodes())
// 	pl.pop(bIndex)
// 	assert.Equal(t, "3<-0->3, 1<-1->1, 2<-2->2, 0<-3->0, 4<-4->4", pl.inspectNodes())
// 	assert.NoError(t, pl.releaseNode(bIndex))
// 	assert.Equal(t, "4<-0->3, 1<-1->1, 2<-2->2, 0<-3->4, 3<-4->0", pl.inspectNodes())
// }

// func TestRetainRetainReleaseRelease(t *testing.T) {
// 	pl := newCircus()
// 	pl.retainNode() // TODO remove this artifact

// 	assert.Equal(t, "0<-0->0, 1<-1->1", pl.inspectNodes())

// 	aIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 2<-2->2", pl.inspectNodes())
// 	assert.Equal(t, 2, aIndex, "retaining a node should create a node")
// 	assert.Equal(t, true, pl.empty(freeHeadIndex), "free list should be empty initially")

// 	bIndex := pl.retainNode()
// 	assert.Equal(t, "0<-0->0, 1<-1->1, 2<-2->2, 3<-3->3", pl.inspectNodes())
// 	assert.Equal(t, 3, bIndex, "retaining a node should create a node")
// 	assert.Equal(t, true, pl.empty(freeHeadIndex), "free list should still be empty")

// 	pl.releaseNode(bIndex)
// 	assert.Equal(t, "3<-0->3, 1<-1->1, 2<-2->2, 0<-3->0", pl.inspectNodes())
// 	assert.Equal(t, false, pl.empty(freeHeadIndex), "free list should not be empty after release")
// 	assert.Equal(t, true, pl.alone(freeHeadIndex), "free list should contain exactly one node")

// 	pl.releaseNode(aIndex)
// 	assert.Equal(t, "2<-0->3, 1<-1->1, 3<-2->0, 0<-3->2", pl.inspectNodes())
// 	assert.Equal(t, false, pl.empty(freeHeadIndex), "free list should not be empty after release")
// 	assert.Equal(t, false, pl.alone(freeHeadIndex), "free list should now contain more than just one node")

// 	cIndex := pl.retainNode()
// 	assert.Equal(t, "2<-0->2, 1<-1->1, 0<-2->0, 3<-3->3", pl.inspectNodes())
// 	assert.Equal(t, 3, cIndex, "retaining after release should reuse a node")
// }

// // This test effectively verifies that the default goal is the maximum integer.
// func TestDefaultGoal(t *testing.T) {
// 	pl := newCircus()

// 	assert.Equal(t, true, pl.goal > 0)
// 	// Verify overflow
// 	assert.Equal(t, true, pl.goal+1 < 0)
// }
