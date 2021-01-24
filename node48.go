package art

import "fmt"

const n48NoChildForKey byte = 255

type node48 struct {
	nodeHeader
	key      [256]byte // index into children, 255 for no child
	children [48]node
}

func (n *node48) header() nodeHeader {
	return n.nodeHeader
}

func newNode48(src *node16) *node48 {
	n := &node48{nodeHeader: src.nodeHeader}
	for i := range n.key {
		n.key[i] = n48NoChildForKey
	}
	if src.hasValue {
		n.children[n48ValueIdx] = src.children[n16ValueIdx]
	}
	for i := byte(0); i < byte(src.childCount); i++ {
		n.key[src.key[i]] = i
		n.children[i] = src.children[i]
	}
	return n
}

func (n *node48) insert(key []byte, value interface{}) node {
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
			n.children[n48ValueIdx].insert(key, value)
			return n
		}
		if n.nodeHeader.childCount < 48 {
			n.nodeHeader.hasValue = true
			n.children[n48ValueIdx] = newNode(key, value)
			return n
		}
		// We're full, need to grow to a larger node size first
		n256 := newNode256(n)
		n256.value = newNode(key, value)
		n256.hasValue = true
		return n256
	}
	slot := n.key[key[0]]
	if slot < n48NoChildForKey {
		n.children[slot] = n.children[slot].insert(key[1:], value)
		return n
	}
	maxSlots := int16(len(n.children))
	if n.hasValue {
		maxSlots--
	}
	if n.childCount < maxSlots {
		n.addChildLeaf(key, value)
		return n
	}
	n256 := newNode256(n)
	n256.children[key[0]] = newNode(key[1:], value)
	n256.childCount++
	return n256
}

func (n *node48) addChildLeaf(key []byte, val interface{}) {
	n.key[key[0]] = byte(n.childCount)
	n.children[n.childCount] = newNode(key[1:], val)
	n.childCount++
}

func (n *node48) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n48ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node48) removeValue() bool {
	n.children[n48ValueIdx] = nil
	n.hasValue = false
	return n.childCount == 0
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

func (n *node48) removeChild(key byte) bool {
	lastSlot := byte(n.childCount - 1)
	keyOfLastSlot := n.keyForSlot(lastSlot)
	slot := n.key[key]

	n.children[slot] = n.children[lastSlot]
	n.key[keyOfLastSlot] = slot
	n.key[key] = n48NoChildForKey
	n.childCount--
	return n.childCount == 0 && !n.hasValue
}

func (n *node48) getNextNode(key []byte) (next node, remainingKey []byte) {
	idx := n.key[key[0]]
	if idx == n48NoChildForKey {
		return nil, nil
	}
	return n.children[idx], key[1:]
}

func (n *node48) walk(prefix []byte, cb ConsumerFn) WalkState {
	prefix = append(prefix, n.path...)
	v, exists := n.nodeValue()
	if exists && cb(prefix, v) == Stop {
		return Stop
	}
	for idx, slot := range n.key {
		if slot != n48NoChildForKey {
			if n.children[slot].walk(append(prefix, byte(idx)), cb) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node48) pretty(indent int, w writer) {
	w.WriteString("[n48] ")
	writePath(n.path, w)
	if n.hasValue {
		w.WriteString(" value: ")
		n.children[n48ValueIdx].pretty(indent, w)
	} else {
		w.WriteByte('\n')
	}
	for k, slot := range n.key {
		if slot != n48NoChildForKey {
			writeIndent(indent+2, w)
			fmt.Fprintf(w, "0x%02X: ", k)
			n.children[slot].pretty(indent+8, w)
		}
	}
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
