package art

import "fmt"

type node256 struct {
	children [256]node
	value    node
	nodeHeader
}

func (n *node256) header() nodeHeader {
	return n.nodeHeader
}

func newNode256(src *node48) *node256 {
	n := &node256{nodeHeader: src.nodeHeader}
	if src.hasValue {
		n.value = src.children[n48ValueIdx]
	}
	for k, slot := range src.key {
		if slot != n48NoChildForKey {
			n.children[k] = src.children[slot]
		}
	}
	return n
}

func (n *node256) insert(key []byte, value interface{}) node {
	splitN, replaced, prefixLen := splitNodePath(key, n.path, n)
	if replaced {
		return splitN.insert(key, value)
	}
	key = key[prefixLen:]
	if len(key) == 0 {
		if n.hasValue {
			n.value.insert(key, value)
		} else {
			n.hasValue = true
			n.value = newNode(key, value)
		}
		return n
	}
	c := n.children[key[0]]
	if c == nil {
		n.children[key[0]] = newNode(key[1:], value)
		n.childCount++
	} else {
		n.children[key[0]] = c.insert(key[1:], value)
	}
	return n
}

func (n *node256) valueNode() node {
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

func (n *node256) getNextNode(key []byte) (next *node, remainingKey []byte) {
	c := n.children[key[0]]
	if c == nil {
		return nil, nil
	}
	return &n.children[key[0]], key[1:]
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
