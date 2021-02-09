package art

import "fmt"

var emptyPath [24]byte

type leaf struct {
	value interface{}
	path  keyPath
}

func newLeaf(value interface{}) *leaf {
	return &leaf{value: value}
}

func newPathLeaf(key []byte, value interface{}) node {
	l := &leaf{value: value}
	kend := len(key)
	kst := max(0, kend-len(l.path.key))
	l.path.assign(key[kst:kend])
	key = key[:kst]
	var curr node = l
	for len(key) > 0 {
		n := &node4{}
		kend := len(key)
		n.addChildNode(key[kend-1], curr)
		kend--
		kst := max(0, kend-len(l.path.key))
		n.path.assign(key[kst:kend])
		key = key[:kst]
		curr = n
	}
	return curr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (l *leaf) header() nodeHeader {
	return nodeHeader{
		path:     l.path,
		hasValue: true,
	}
}

func (l *leaf) keyPath() *keyPath {
	return &l.path
}

func (l *leaf) grow() node {
	// we need to promote this leaf to a node with a contained value
	n := &node4{}
	n.path = l.path
	l.path.assign(nil)
	n.setNodeValue(l)
	return n
}

func (l *leaf) shrink() node {
	return l
}

func (l *leaf) canAddChild() bool {
	return false
}

func (l *leaf) addChildNode(key byte, child node) {
	panic("Can't add a childNode to a leaf")
}

func (l *leaf) canSetNodeValue() bool {
	return true
}

func (l *leaf) setNodeValue(v *leaf) {
	l.value = v.value
}

func (l *leaf) valueNode() *leaf {
	return l
}

func (l *leaf) iterateChildren(cb nodeConsumer) WalkState {
	return Continue
}

func (l *leaf) iterateChildrenRange(start, end int, cb nodeConsumer) WalkState {
	return Continue
}

func (l *leaf) removeValue() node {
	return nil
}

func (l *leaf) removeChild(key byte) {
	panic("removeChild called on leaf")
}

func (l *leaf) getChildNode(key []byte) *node {
	return nil
}

func (l *leaf) pretty(indent int, w writer) {
	w.WriteString("[leaf")
	writePath(l.path.asSlice(), w)
	fmt.Fprintf(w, "] value:%v\n", l.value)
}

func (l *leaf) stats(s *Stats) {
	s.Keys++
}
