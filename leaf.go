package art

import "fmt"

type leaf[V any] struct {
	value V
	path  keyPath
}

func newLeaf[V any](value V) *leaf[V] {
	return &leaf[V]{value: value}
}

func newPathLeaf[V any](key []byte, value V) node[V] {
	l := &leaf[V]{value: value}
	kend := len(key)
	kst := max(0, kend-len(l.path.key))
	l.path.assign(key[kst:kend])
	key = key[:kst]
	var curr node[V] = l
	for len(key) > 0 {
		n := &node4[V]{}
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

func (l *leaf[V]) header() nodeHeader {
	return nodeHeader{
		path:     l.path,
		hasValue: true,
	}
}

func (l *leaf[V]) keyPath() *keyPath {
	return &l.path
}

func (l *leaf[V]) grow() node[V] {
	// we need to promote this leaf to a node with a contained value
	n := &node4[V]{}
	n.path = l.path
	l.path.assign(nil)
	n.setNodeValue(l)
	return n
}

func (l *leaf[V]) shrink() node[V] {
	return l
}

func (l *leaf[V]) canAddChild() bool {
	return false
}

func (l *leaf[V]) addChildNode(key byte, child node[V]) {
	panic("Can't add a childNode to a leaf")
}

func (l *leaf[V]) canSetNodeValue() bool {
	return true
}

func (l *leaf[V]) setNodeValue(v *leaf[V]) {
	l.value = v.value
}

func (l *leaf[V]) valueNode() *leaf[V] {
	return l
}

func (l *leaf[V]) iterateChildren(cb func(k byte, n node[V]) WalkState) WalkState {
	return Continue
}

func (l *leaf[V]) iterateChildrenRange(start, end int, cb func(k byte, n node[V]) WalkState) WalkState {
	return Continue
}

func (l *leaf[V]) removeValue() node[V] {
	return nil
}

func (l *leaf[V]) removeChild(key byte) {
	panic("removeChild called on leaf")
}

func (l *leaf[V]) getChildNode(key []byte) *node[V] {
	return nil
}

func (l *leaf[V]) pretty(indent int, w writer) {
	w.WriteString("[leaf")
	writePath(l.path.asSlice(), w)
	fmt.Fprintf(w, "] value:%v\n", l.value)
}

func (l *leaf[V]) stats(s *Stats) {
	s.Keys++
}
