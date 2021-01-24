package art

import "fmt"

type node4 struct {
	nodeHeader
	key      [4]byte
	children [4]node
}

func (n *node4) header() nodeHeader {
	return n.nodeHeader
}

func (n *node4) insert(key []byte, value interface{}) node {
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
			n.children[n4ValueIdx].insert(key, value)
			return n
		}
		if n.nodeHeader.childCount < 4 {
			n.children[n4ValueIdx] = newNode(key, value)
			n.nodeHeader.hasValue = true
			return n
		}
		// we're full, need to grow
		n16 := newNode16(n)
		n16.children[n16ValueIdx] = newNode(key, value)
		n16.hasValue = true
		return n16
	}
	for i := int16(0); i < n.nodeHeader.childCount; i++ {
		if n.key[i] == key[0] {
			n.children[i] = n.children[i].insert(key[1:], value)
			return n
		}
	}
	maxChildren := len(n.children)
	if n.hasValue {
		maxChildren--
	}
	if int(n.childCount) < maxChildren {
		idx := n.childCount
		n.key[idx] = key[0]
		n.children[idx] = newNode(key[1:], value)
		n.nodeHeader.childCount++
		return n
	}
	n16 := newNode16(n)
	n16.addChildLeaf(key, value)
	return n16
}

func (n *node4) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n4ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node4) removeValue() node {
	n.children[n4ValueIdx] = nil
	n.hasValue = false
	if n.childCount == 0 {
		return nil
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
			if n.childCount == 0 && !n.hasValue {
				return nil
			}
			return n
		}
	}
	return n
}

func (n *node4) getNextNode(key []byte) (next *node, remainingKey []byte) {
	for i := 0; i < int(n.childCount); i++ {
		if key[0] == n.key[i] {
			return &n.children[i], key[1:]
		}
	}
	return nil, nil
}

func (n *node4) walk(prefix []byte, cb ConsumerFn) WalkState {
	prefix = append(prefix, n.path...)
	val, exists := n.nodeValue()
	if exists {
		if cb(prefix, val) == Stop {
			return Stop
		}
	}
	done := byte(0)
	for i := int16(0); i < n.childCount; i++ {
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

func (n *node4) pretty(indent int, w writer) {
	w.WriteString("[n4] ")
	writePath(n.path, w)
	if n.hasValue {
		w.WriteString(" value: ")
		n.children[n4ValueIdx].pretty(indent, w)
	} else {
		w.WriteByte('\n')
	}
	for i := 0; i < int(n.childCount); i++ {
		writeIndent(indent+2, w)
		fmt.Fprintf(w, "0x%02X: ", n.key[i])
		n.children[i].pretty(indent+8, w)
	}
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
