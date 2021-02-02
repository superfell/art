package art

import "fmt"

type node256 struct {
	children [256]node
	value    *leaf
	nodeHeader
}

func newNode256(src *node48) *node256 {
	n := &node256{nodeHeader: src.nodeHeader}
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

func (n *node256) header() nodeHeader {
	return n.nodeHeader
}

func (n *node256) grow() node {
	panic("Can't grow a node256")
}

func (n *node256) canAddChild() bool {
	return true
}

func (n *node256) addChildNode(key byte, child node) {
	n.children[key] = child
	n.childCount++
}

func (n *node256) canSetNodeValue() bool {
	return true
}

func (n *node256) setNodeValue(v *leaf) {
	n.value = v
	n.hasValue = true
}

func (n *node256) valueNode() *leaf {
	return n.value
}

func (n *node256) iterateChildren(cb nodeConsumer) WalkState {
	for k, n := range n.children {
		if n != nil {
			if cb(byte(k), n) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node256) removeValue() node {
	n.hasValue = false
	n.value = nil
	return n
}

func (n *node256) removeChild(k byte) node {
	n.children[k] = nil
	n.childCount--
	if n.childCount < 48*3/4 {
		return newNode48(n)
	}
	return n
}

func (n *node256) getNextNode(key []byte) *node {
	c := n.children[key[0]]
	if c == nil {
		return nil
	}
	return &n.children[key[0]]
}

func (n *node256) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.value.nodeValue()
	}
	return nil, false
}

func (n *node256) pretty(indent int, w writer) {
	w.WriteString("[n256] ")
	writePath(n.path, w)
	if n.hasValue {
		fmt.Fprintf(w, "value: %v", n.value)
	}
	w.WriteByte('\n')
	for idx, c := range n.children {
		if c != nil {
			writeIndent(indent+2, w)
			fmt.Fprintf(w, "0x%02X: ", idx)
			c.pretty(indent+8, w)
		}
	}
}

func (n *node256) stats(s *Stats) {
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
