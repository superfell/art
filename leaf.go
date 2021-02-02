package art

import "fmt"

type leaf struct {
	value interface{}
	path  []byte
}

func newLeaf(key []byte, value interface{}) *leaf {
	return &leaf{
		path:  key,
		value: value,
	}
}

func (l *leaf) header() nodeHeader {
	return nodeHeader{
		path:     l.path,
		hasValue: true,
	}
}

func (l *leaf) grow() node {
	// we need to promote this leaf to a node with a contained value
	n := &node4{}
	n.path = l.path
	l.path = nil
	n.setNodeValue(l)
	return n
}

func (l *leaf) trimPathStart(amount int) {
	l.path = l.path[amount:]
}

func (l *leaf) prependPath(prefix []byte, k ...byte) {
	l.path = joinSlices(prefix, k, l.path)
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

func (l *leaf) removeValue() node {
	return nil
}

func (l *leaf) removeChild(key byte) node {
	panic("removeChild called on leaf")
}

func (l *leaf) getChildNode(key []byte) *node {
	return nil
}

func (l *leaf) pretty(indent int, w writer) {
	w.WriteString("[leaf] ")
	writePath(l.path, w)
	fmt.Fprintf(w, " value:%v\n", l.value)
}

func (l *leaf) stats(s *Stats) {
	s.Keys++
}
