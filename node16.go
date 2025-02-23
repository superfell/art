package art

import (
	"fmt"
)

// index into the children arrays for the node value leaf.
const n16ValueIdx = 15

type node16[V any] struct {
	nodeHeader
	key      [16]byte
	children [16]node[V]
}

// constructs a new node16 from another node
func newNode16[V any](src node[V]) *node16[V] {
	n := node16[V]{nodeHeader: src.header()}
	if n.hasValue {
		n.children[n16ValueIdx] = src.valueNode()
	}
	idx := 0
	// iterateChildren iterates in key order, which simplifies this
	src.iterateChildren(func(k byte, cn node[V]) WalkState {
		n.key[idx] = k
		n.children[idx] = cn
		idx++
		return Continue
	})
	return &n
}

func (n *node16[V]) header() nodeHeader {
	return n.nodeHeader
}

func (n *node16[V]) keyPath() *keyPath {
	return &n.path
}

func (n *node16[V]) grow() node[V] {
	return newNode48[V](n)
}

func (n *node16[V]) canAddChild() bool {
	max := int16(len(n.key))
	if n.hasValue {
		max--
	}
	return n.childCount < max
}

func (n *node16[V]) addChildNode(key byte, child node[V]) {
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

func (n *node16[V]) findInsertionPoint(key byte) (idx int, exists bool) {
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
func (n *node16[V]) canSetNodeValue() bool {
	return int(n.childCount) < len(n.key)
}

func (n *node16[V]) setNodeValue(v *leaf[V]) {
	n.children[n16ValueIdx] = v
	n.hasValue = true
}

func (n *node16[V]) valueNode() *leaf[V] {
	if n.hasValue {
		return n.children[n16ValueIdx].(*leaf[V])
	}
	return nil
}

func (n *node16[V]) iterateChildren(cb func(k byte, n node[V]) WalkState) WalkState {
	for i := 0; i < int(n.childCount); i++ {
		if cb(n.key[i], n.children[i]) == Stop {
			return Stop
		}
	}
	return Continue
}

func (n *node16[V]) iterateChildrenRange(start, end int, cb func(k byte, n node[V]) WalkState) WalkState {
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

func (n *node16[V]) removeValue() node[V] {
	n.children[n16ValueIdx] = nil
	n.hasValue = false
	return n
}

func (n *node16[V]) removeChild(k byte) {
	// keep key ordered
	idx, exists := n.findInsertionPoint(k)
	if !exists {
		panic(fmt.Sprintf("removeChild called on non-existing key %d, keys are %v", k, n.key[:n.childCount]))
	}
	copy(n.key[idx:], n.key[idx+1:int(n.childCount)])
	copy(n.children[idx:], n.children[idx+1:int(n.childCount)])
	n.childCount--
	n.key[int(n.childCount)] = 0
	n.children[int(n.childCount)] = nil
}

func (n *node16[V]) shrink() node[V] {
	if n.childCount <= 2 {
		return newNode4[V](n)
	}
	return n
}

func (n *node16[V]) getChildNode(key []byte) *node[V] {
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

func (n *node16[V]) pretty(indent int, w writer) {
	writeNode[V](n, "n16", indent, w)
}

func (n *node16[V]) stats(s *Stats) {
	s.Node16s++
	if n.hasValue {
		n.children[n16ValueIdx].stats(s)
	}
	for i := int16(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}
