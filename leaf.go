package art

import "fmt"

type leaf struct {
	value interface{}
	path  []byte
}

func (l *leaf) header() nodeHeader {
	return nodeHeader{
		path:     l.path,
		hasValue: true,
	}
}

func (l *leaf) trimPathStart(amount int) {
	l.path = l.path[amount:]
}

func (l *leaf) insert(key []byte, value interface{}) node {
	// may need to split this so that the child nodes can be added
	splitN, replaced, prefixLen := splitNodePath(key, l.path, l)
	if replaced {
		return splitN.insert(key, value)
	}
	if prefixLen == len(key) && prefixLen == len(l.path) {
		// this is our stop
		l.value = value
		return l
	}
	// we need to promote this leaf to a node with a contained value
	n := &node4{}
	n.path = l.path
	l.path = nil
	n.hasValue = true
	n.children[3] = l
	return n.insert(key, value)
}

func (l *leaf) nodeValue() (interface{}, bool) {
	return l.value, true
}

func (l *leaf) removeValue() node {
	return nil
}

func (l *leaf) removeChild(key byte) node {
	panic("removeChild called on leaf")
}

func (l *leaf) getNextNode(key []byte) (next *node, remainingKey []byte) {
	return nil, nil
}

func (l *leaf) walk(prefix []byte, cb ConsumerFn) WalkState {
	return cb(append(prefix, l.path...), l.value)
}

func (l *leaf) pretty(indent int, w writer) {
	w.WriteString("[leaf] ")
	writePath(l.path, w)
	fmt.Fprintf(w, " value:%v\n", l.value)
}

func (l *leaf) stats(s *Stats) {
	s.Keys++
}
