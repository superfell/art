package art

// index into the children arrays for the node value leaf.
const n4ValueIdx = 3

type node4 struct {
	nodeHeader
	key      [4]byte
	children [4]node
}

func newNode4(src node) *node4 {
	n := node4{nodeHeader: src.header()}
	if n.hasValue {
		n.children[n4ValueIdx] = src.valueNode()
	}
	slot := 0
	src.iterateChildren(func(k byte, cn node) WalkState {
		n.key[slot] = k
		n.children[slot] = cn
		slot++
		return Continue
	})
	return &n
}

func (n *node4) header() nodeHeader {
	return n.nodeHeader
}

func (n *node4) grow() node {
	return newNode16(n)
}

func (n *node4) canAddChild() bool {
	max := int16(len(n.key))
	if n.hasValue {
		max--
	}
	return n.childCount < max
}

func (n *node4) addChildNode(key byte, child node) {
	idx := n.childCount
	n.key[idx] = key
	n.children[idx] = child
	n.nodeHeader.childCount++
}

func (n *node4) canSetNodeValue() bool {
	return n.nodeHeader.childCount < 4
}

func (n *node4) setNodeValue(v *leaf) {
	n.children[n4ValueIdx] = v
	n.nodeHeader.hasValue = true
}

func (n *node4) valueNode() *leaf {
	if n.hasValue {
		return n.children[n4ValueIdx].(*leaf)
	}
	return nil
}

func (n *node4) iterateChildren(cb nodeConsumer) WalkState {
	done := byte(0)
	for i := byte(0); i < byte(n.childCount); i++ {
		next := byte(255)
		nextIdx := byte(255)
		for j := byte(0); j < byte(n.childCount); j++ {
			k := n.key[j]
			if k <= next && k >= done {
				next = k
				nextIdx = j
			}
		}
		if cb(next, n.children[nextIdx]) == Stop {
			return Stop
		}
		done = next + 1
	}
	return Continue
}

func (n *node4) removeValue() node {
	n.children[n4ValueIdx] = nil
	n.hasValue = false
	return n.shrinkMeMaybe()
}

func (n *node4) shrinkMeMaybe() node {
	if n.childCount == 1 && !n.hasValue {
		// when we're down to 1 child, we can add our path and the child's key to the start of its path
		// and return that instead
		c := n.children[0]
		c.prependPath(n.path, n.key[0])
		return c
	}
	if n.childCount == 0 && n.hasValue {
		// if all we have is our value, we can add our path to the values key, and return that
		v := n.children[n4ValueIdx]
		v.prependPath(n.path)
		return v
	}
	return n
}

func (n *node4) removeChild(k byte) node {
	lastIdx := n.childCount - 1
	for i := 0; i < int(n.childCount); i++ {
		if k == n.key[i] {
			n.children[i] = n.children[lastIdx]
			n.children[lastIdx] = nil
			n.key[i] = n.key[lastIdx]
			n.key[lastIdx] = 0
			n.childCount--
			return n.shrinkMeMaybe()
		}
	}
	return n
}

func (n *node4) getChildNode(key []byte) *node {
	for i := 0; i < int(n.childCount); i++ {
		if key[0] == n.key[i] {
			return &n.children[i]
		}
	}
	return nil
}

func (n *node4) pretty(indent int, w writer) {
	writeNode(n, "n4", indent, w)
}

func (n *node4) stats(s *Stats) {
	s.Node4s++
	if n.hasValue {
		n.children[n4ValueIdx].stats(s)
	}
	for i := int16(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}
