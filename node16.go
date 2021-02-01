package art

import (
	"fmt"
)

type node16 struct {
	nodeHeader
	key      [16]byte
	children [16]node
}

func (n *node16) header() nodeHeader {
	return n.nodeHeader
}

// constructs a new node16 from a node4.
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
	next, remainingKey := n.getNextNode(key)
	if next != nil {
		*next = (*next).insert(remainingKey, value)
		return n
	}
	maxChildren := int16(len(n.children))
	if n.hasValue {
		maxChildren--
	}
	if n.childCount < maxChildren {
		n.addChildNode(key[0], newNode(key[1:], value))
		return n
	}
	n48 := newNode48(n)
	n48.addChildLeaf(key, value)
	return n48
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

func (n *node16) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n16ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node16) valueNode() node {
	if n.hasValue {
		return n.children[n16ValueIdx]
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

func (n *node16) removeValue() node {
	n.children[n16ValueIdx] = nil
	n.hasValue = false
	return n
}

func (n *node16) removeChild(k byte) node {
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
	if n.childCount <= 2 {
		return newNode4(n)
	}
	return n
}

func (n *node16) getNextNode(key []byte) (next *node, remainingKey []byte) {
	// see https://www.superfell.com/weblog/2021/01/it-depends-episode-1
	// and https://www.superfell.com/weblog/2021/01/it-depends-episode-2
	// for a detailed discussion around looping vs binary search
	_ = n.key[n.childCount-1]
	for i := n.childCount - 1; i >= 0; i-- {
		if n.key[i] == key[0] {
			return &n.children[i], key[1:]
		}
	}
	return nil, nil
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
