package circus

import "fmt"

func (pl *Circus) inspectNodes() string {
	res := ""
	for index, node := range pl.nodes {
		if index != 0 {
			res += ", "
		}
		res += fmt.Sprintf("%d<-%d->%d", node.prevIndex, index, node.nextIndex)
	}
	return res
}

func (pl *Circus) inspectRings() string {
	res := "["
	// for index, ring := range pl.rings {
	// 	if index != 0 {
	// 		res += ", "
	// 	}
	// 	length := ""
	// 	if pl.alone(ring.headIndex) {
	// 		length = " (0)"
	// 	} else if pl.empty(ring.headIndex) {
	// 		length = " (1)"
	// 	}
	// 	res += fmt.Sprintf("%d%s", ring.pending, length)
	// }
	res += "]"
	return res
}
