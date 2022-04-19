package art

// key value for keys that have no child node.
const n48NoChildForKey byte = 255

// index into the children arrays for the node value leaf.
const n48ValueIdx = 47

type node48[V any] struct {
	nodeHeader
	key      [256]byte // index into children, 255 for no child
	children [48]node[V]
}

func newNode48[V any](src node[V]) *node48[V] {
	n := &node48[V]{nodeHeader: src.header()}
	for i := range n.key {
		n.key[i] = n48NoChildForKey
	}
	if n.hasValue {
		n.children[n48ValueIdx] = src.valueNode()
	}
	slot := byte(0)
	src.iterateChildren(func(k byte, cn node[V]) WalkState {
		n.key[k] = slot
		n.children[slot] = cn
		slot++
		return Continue
	})
	return n
}

func (n *node48[V]) header() nodeHeader {
	return n.nodeHeader
}

func (n *node48[V]) keyPath() *keyPath {
	return &n.path
}

func (n *node48[V]) grow() node[V] {
	return newNode256(n)
}

func (n *node48[V]) canAddChild() bool {
	max := int16(len(n.children))
	if n.hasValue {
		max--
	}
	return n.childCount < max
}

func (n *node48[V]) addChildNode(key byte, child node[V]) {
	n.key[key] = byte(n.childCount)
	n.children[n.childCount] = child
	n.childCount++
}

func (n *node48[V]) canSetNodeValue() bool {
	return n.childCount < int16(len(n.children))
}

func (n *node48[V]) setNodeValue(v *leaf[V]) {
	n.children[n48ValueIdx] = v
	n.hasValue = true
}

func (n *node48[V]) valueNode() *leaf[V] {
	if n.hasValue {
		return n.children[n48ValueIdx].(*leaf[V])
	}
	return nil
}

func (n *node48[V]) iterateChildren(cb func(k byte, n node[V]) WalkState) WalkState {
	return n.iterateChildrenRange(0, 256, cb)
}

func (n *node48[V]) iterateChildrenRange(start, end int, cb func(k byte, n node[V]) WalkState) WalkState {
	for k := start; k < end; k++ {
		slot := n.key[k]
		if slot != n48NoChildForKey {
			if cb(byte(k), n.children[slot]) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node48[V]) removeValue() node[V] {
	n.children[n48ValueIdx] = nil
	n.hasValue = false
	return n
}

// keyForSlot returns the key value for the supplied child slot number.
func (n *node48[V]) keyForSlot(slot byte) int {
	for k, s := range n.key {
		if s == slot {
			return k
		}
	}
	panic("n48.keyForSlot called with unused slot number")
}

func (n *node48[V]) removeChild(key byte) {
	lastSlot := byte(n.childCount - 1)
	keyOfLastSlot := n.keyForSlot(lastSlot)
	slot := n.key[key]

	n.children[slot] = n.children[lastSlot]
	n.key[keyOfLastSlot] = slot
	n.key[key] = n48NoChildForKey
	n.childCount--
}

func (n *node48[V]) shrink() node[V] {
	if n.childCount < 16*3/4 {
		return newNode16[V](n)
	}
	return n
}

func (n *node48[V]) getChildNode(key []byte) *node[V] {
	idx := n.key[key[0]]
	if idx == n48NoChildForKey {
		return nil
	}
	return &n.children[idx]
}

func (n *node48[V]) pretty(indent int, w writer) {
	writeNode[V](n, "n48", indent, w)
}

func (n *node48[V]) stats(s *Stats) {
	s.Node48s++
	if n.hasValue {
		n.children[n48ValueIdx].stats(s)
	}
	for i := int16(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}
