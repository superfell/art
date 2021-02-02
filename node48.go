package art

// key value for keys that have no child node.
const n48NoChildForKey byte = 255

// index into the children arrays for the node value leaf.
const n48ValueIdx = 47

type node48 struct {
	nodeHeader
	key      [256]byte // index into children, 255 for no child
	children [48]node
}

func newNode48(src node) *node48 {
	n := &node48{nodeHeader: src.header()}
	for i := range n.key {
		n.key[i] = n48NoChildForKey
	}
	if n.hasValue {
		n.children[n48ValueIdx] = src.valueNode()
	}
	slot := byte(0)
	src.iterateChildren(func(k byte, cn node) WalkState {
		n.key[k] = slot
		n.children[slot] = cn
		slot++
		return Continue
	})
	return n
}

func (n *node48) header() nodeHeader {
	return n.nodeHeader
}

func (n *node48) keyPath() *keyPath {
	return &n.path
}

func (n *node48) grow() node {
	return newNode256(n)
}

func (n *node48) canAddChild() bool {
	max := int16(len(n.children))
	if n.hasValue {
		max--
	}
	return n.childCount < max
}

func (n *node48) addChildNode(key byte, child node) {
	n.key[key] = byte(n.childCount)
	n.children[n.childCount] = child
	n.childCount++
}

func (n *node48) canSetNodeValue() bool {
	return n.childCount < int16(len(n.children))
}

func (n *node48) setNodeValue(v *leaf) {
	n.children[n48ValueIdx] = v
	n.hasValue = true
}

func (n *node48) valueNode() *leaf {
	if n.hasValue {
		return n.children[n48ValueIdx].(*leaf)
	}
	return nil
}

func (n *node48) iterateChildren(cb nodeConsumer) WalkState {
	for k, slot := range n.key {
		if slot != n48NoChildForKey {
			if cb(byte(k), n.children[slot]) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node48) removeValue() node {
	n.children[n48ValueIdx] = nil
	n.hasValue = false
	return n
}

// keyForSlot returns the key value for the supplied child slot number.
func (n *node48) keyForSlot(slot byte) int {
	for k, s := range n.key {
		if s == slot {
			return k
		}
	}
	panic("n48.keyForSlot called with unused slot number")
}

func (n *node48) removeChild(key byte) {
	lastSlot := byte(n.childCount - 1)
	keyOfLastSlot := n.keyForSlot(lastSlot)
	slot := n.key[key]

	n.children[slot] = n.children[lastSlot]
	n.key[keyOfLastSlot] = slot
	n.key[key] = n48NoChildForKey
	n.childCount--
}

func (n *node48) shrink() node {
	if n.childCount < 16*3/4 {
		return newNode16(n)
	}
	return n
}

func (n *node48) getChildNode(key []byte) *node {
	idx := n.key[key[0]]
	if idx == n48NoChildForKey {
		return nil
	}
	return &n.children[idx]
}

func (n *node48) pretty(indent int, w writer) {
	writeNode(n, "n48", indent, w)
}

func (n *node48) stats(s *Stats) {
	s.Node48s++
	if n.hasValue {
		n.children[n48ValueIdx].stats(s)
	}
	for i := int16(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}
