package art

import (
	"fmt"
)

// index into the children arrays for the node value leaf.
const n16ValueIdx = 15

type node16 struct {
	nodeHeader
	key      [16]byte
	children [16]node
}

// constructs a new node16 from another node
func newNode16(src node) *node16 {
	n := node16{nodeHeader: src.header()}
	if n.hasValue {
		n.children[n16ValueIdx] = src.valueNode()
	}
	idx := 0
	// iterateChildren iterates in key order, which simplifies this
	src.iterateChildren(func(k byte, cn node) WalkState {
		n.key[idx] = k
		n.children[idx] = cn
		idx++
		return Continue
	})
	return &n
}

func (n *node16) header() nodeHeader {
	return n.nodeHeader
}

func (n *node16) keyPath() *keyPath {
	return &n.path
}

func (n *node16) grow() node {
	return newNode48(n)
}

func (n *node16) canAddChild() bool {
	max := int16(len(n.key))
	if n.hasValue {
		max--
	}
	return n.childCount < max
}

func (n *node16) addChildNode(key byte, child node) {
	// keep key ordered
	slot, exists := n.findInsertionPoint(key)
	if exists {
		panic("addChildNode called with key that has an existing value")
	}
	copy(n.key[slot+1:], n.key[slot:int(n.childCount)])
	copy(n.children[slot+1:], n.children[slot:int(n.childCount)])
	n.key[slot] = key
	n.children[slot] = child
	n.nodeHeader.childCount++
}

func (n *node16) findInsertionPoint(key byte) (idx int, exists bool) {
	count := int(n.childCount)
	_ = n.key[count-1]
	for i := count - 1; i >= 0; i-- {
		if key == n.key[i] {
			return i, true
		}
		if key > n.key[i] {
			return i + 1, false
		}
	}
	return 0, false
}
func (n *node16) canSetNodeValue() bool {
	return int(n.childCount) < len(n.key)
}

func (n *node16) setNodeValue(v *leaf) {
	n.children[n16ValueIdx] = v
	n.hasValue = true
}

func (n *node16) valueNode() *leaf {
	if n.hasValue {
		return n.children[n16ValueIdx].(*leaf)
	}
	return nil
}

func (n *node16) iterateChildren(cb nodeConsumer) WalkState {
	for i := 0; i < int(n.childCount); i++ {
		if cb(n.key[i], n.children[i]) == Stop {
			return Stop
		}
	}
	return Continue
}

func (n *node16) iterateChildrenRange(start, end int, cb nodeConsumer) WalkState {
	for i := 0; i < int(n.childCount); i++ {
		k := int(n.key[i])
		if k < start {
			continue
		}
		if k >= end {
			return Continue
		}
		if cb(n.key[i], n.children[i]) == Stop {
			return Stop
		}
	}
	return Continue
}

func (n *node16) removeValue() node {
	n.children[n16ValueIdx] = nil
	n.hasValue = false
	return n
}

func (n *node16) removeChild(k byte) {
	// keep key ordered
	idx, exists := n.findInsertionPoint(k)
	if !exists {
		panic(fmt.Sprintf("removeChild called on non-existing key %d, keys are %v", k, n.key[:n.childCount]))
	}
	copy(n.key[idx:], n.key[idx+1:int(n.childCount)])
	copy(n.children[idx:], n.children[idx+1:int(n.childCount)])
	n.key[int(n.childCount)] = 0
	n.children[int(n.childCount)] = nil
	n.childCount--
}

func (n *node16) shrink() node {
	if n.childCount <= 2 {
		return newNode4(n)
	}
	return n
}

func (n *node16) getChildNode(key []byte) *node {
	// see https://www.superfell.com/weblog/2021/01/it-depends-episode-1
	// and https://www.superfell.com/weblog/2021/01/it-depends-episode-2
	// for a detailed discussion around looping vs binary search
	_ = n.key[n.childCount-1]
	for i := n.childCount - 1; i >= 0; i-- {
		if n.key[i] == key[0] {
			return &n.children[i]
		}
	}
	return nil
}

func (n *node16) pretty(indent int, w writer) {
	writeNode(n, "n16", indent, w)
}

func (n *node16) stats(s *Stats) {
	s.Node16s++
	if n.hasValue {
		n.children[n16ValueIdx].stats(s)
	}
	for i := int16(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}
