package art

import "fmt"

type node16 struct {
	nodeHeader
	key      [16]byte
	children [16]node
}

func (n *node16) header() nodeHeader {
	return n.nodeHeader
}

// constructs a new node16 from a node4.
func newNode16(src *node4) *node16 {
	n := node16{nodeHeader: src.nodeHeader}
	for i := 0; i < int(src.nodeHeader.childCount); i++ {
		n.key[i] = src.key[i]
		n.children[i] = src.children[i]
	}
	if src.hasValue {
		n.children[n16ValueIdx] = src.children[n4ValueIdx]
	}
	return &n
}

func (n *node16) insert(key []byte, value interface{}) node {
	splitN, replaced, prefixLen := splitNodePath(key, n.path, n)
	if replaced {
		return splitN.insert(key, value)
	}
	key = key[prefixLen:]
	if len(key) == 0 {
		// we're trying to insert a value at this path, and this path
		// is the prefix of some other path.
		// if we already have a value, then just update it
		if n.nodeHeader.hasValue {
			n.children[n16ValueIdx].insert(key, value)
			return n
		}
		if n.nodeHeader.childCount < 16 {
			n.children[n16ValueIdx] = newNode(key, value)
			n.nodeHeader.hasValue = true
			return n
		}
		// we're full, need to grow
		n48 := newNode48(n)
		n48.children[n48ValueIdx] = newNode(key, value)
		n48.hasValue = true
		return n48
	}
	for i := int16(0); i < n.childCount; i++ {
		if n.key[i] == key[0] {
			n.children[i] = n.children[i].insert(key[1:], value)
			return n
		}
	}
	maxChildren := int16(len(n.children))
	if n.hasValue {
		maxChildren--
	}
	if n.childCount < maxChildren {
		n.addChildLeaf(key, value)
		return n
	}
	n48 := newNode48(n)
	n48.addChildLeaf(key, value)
	return n48
}

func (n *node16) addChildLeaf(key []byte, val interface{}) {
	idx := n.childCount
	n.key[idx] = key[0]
	n.children[idx] = newNode(key[1:], val)
	n.nodeHeader.childCount++
}

func (n *node16) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n16ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node16) removeValue() bool {
	n.children[n16ValueIdx] = nil
	n.hasValue = false
	return n.childCount == 0
}

func (n *node16) removeChild(k byte) bool {
	lastIdx := n.childCount - 1
	for i := 0; i < int(n.childCount); i++ {
		if k == n.key[i] {
			n.children[i] = n.children[lastIdx]
			n.children[lastIdx] = nil
			n.key[i] = n.key[lastIdx]
			n.key[lastIdx] = 0
			n.childCount--
			return n.childCount == 0 && !n.hasValue
		}
	}
	return false
}

func (n *node16) getNextNode(key []byte) (next node, remainingKey []byte) {
	for i := 0; i < int(n.childCount); i++ {
		if key[0] == n.key[i] {
			return n.children[i], key[1:]
		}
	}
	return nil, nil
}

func (n *node16) walk(prefix []byte, cb ConsumerFn) WalkState {
	prefix = append(prefix, n.path...)
	val, exists := n.nodeValue()
	if exists {
		if cb(prefix, val) == Stop {
			return Stop
		}
	}
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
		if n.children[nextIdx].walk(append(prefix, next), cb) == Stop {
			return Stop
		}
		done = next + 1
	}
	return Continue
}

func (n *node16) pretty(indent int, w writer) {
	w.WriteString("[n16] ")
	writePath(n.path, w)
	if n.hasValue {
		w.WriteString(" value: ")
		n.children[n16ValueIdx].pretty(indent, w)
	} else {
		w.WriteByte('\n')
	}
	for i := 0; i < int(n.childCount); i++ {
		writeIndent(indent+2, w)
		fmt.Fprintf(w, "0x%02X: ", n.key[i])
		n.children[i].pretty(indent+8, w)
	}
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
