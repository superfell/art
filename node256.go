package art

type node256[V any] struct {
	children [256]node[V]
	value    *leaf[V]
	nodeHeader
}

func newNode256[V any](src *node48[V]) *node256[V] {
	n := &node256[V]{nodeHeader: src.nodeHeader}
	if src.hasValue {
		n.value = src.valueNode()
	}
	for k, slot := range src.key {
		if slot != n48NoChildForKey {
			n.children[k] = src.children[slot]
		}
	}
	return n
}

func (n *node256[V]) header() nodeHeader {
	return n.nodeHeader
}

func (n *node256[V]) keyPath() *keyPath {
	return &n.path
}

func (n *node256[V]) grow() node[V] {
	panic("Can't grow a node256")
}

func (n *node256[V]) canAddChild() bool {
	return true
}

func (n *node256[V]) addChildNode(key byte, child node[V]) {
	n.children[key] = child
	n.childCount++
}

func (n *node256[V]) canSetNodeValue() bool {
	return true
}

func (n *node256[V]) setNodeValue(v *leaf[V]) {
	n.value = v
	n.hasValue = true
}

func (n *node256[V]) valueNode() *leaf[V] {
	return n.value
}

func (n *node256[V]) iterateChildren(cb func(k byte, n node[V]) WalkState) WalkState {
	return n.iterateChildrenRange(0, 256, cb)
}

func (n *node256[V]) iterateChildrenRange(start, end int, cb func(k byte, n node[V]) WalkState) WalkState {
	for k := start; k < end; k++ {
		c := n.children[k]
		if c != nil {
			if cb(byte(k), c) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node256[V]) removeValue() node[V] {
	n.hasValue = false
	n.value = nil
	return n
}

func (n *node256[V]) removeChild(k byte) {
	n.children[k] = nil
	n.childCount--
}

func (n *node256[V]) shrink() node[V] {
	if n.childCount < 48*3/4 {
		return newNode48[V](n)
	}
	return n
}

func (n *node256[V]) getChildNode(key []byte) *node[V] {
	c := n.children[key[0]]
	if c == nil {
		return nil
	}
	return &n.children[key[0]]
}

func (n *node256[V]) pretty(indent int, w writer) {
	writeNode[V](n, "n256", indent, w)
}

func (n *node256[V]) stats(s *Stats) {
	s.Node256s++
	if n.hasValue {
		s.Keys++
	}
	for _, c := range n.children {
		if c != nil {
			c.stats(s)
		}
	}
}
