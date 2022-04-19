package art

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Tree is an Adaptive Radix Tree. Keys are arbitrary byte slices, and the path through the tree
// is the key. Values are stored on Leaves of the tree. The tree is organized in lexicographical
// order of the keys.
type Tree[V any] struct {
	root node[V]
}

// Put inserts or updates a value in the tree associated with the provided key. Value can be any
// interface value, including nil. key can be an arbitrary byte slice, including the empty slice.
func (a *Tree[V]) Put(key []byte, value V) {
	a.root = a.put(a.root, key, value)
}

func (a *Tree[V]) put(n node[V], key []byte, value V) node[V] {
	if n == nil {
		return newPathLeaf(key, value)
	}
	key, n = splitNodePath(key, n)
	if len(key) == 0 {
		vn := n.valueNode()
		if vn != nil {
			vn.value = value
		} else if n.canSetNodeValue() {
			n.setNodeValue(newLeaf(value))
		} else {
			n = n.grow()
			n.setNodeValue(newLeaf(value))
		}
		return n
	}
	child := n.getChildNode(key)
	if child != nil {
		*child = a.put(*child, key[1:], value)
		return n
	}
	if !n.canAddChild() {
		n = n.grow()
	}
	n.addChildNode(key[0], newPathLeaf(key[1:], value))
	return n
}

// Get the value for the provided key. exists is true if the key contains a value in the tree,
// false otherwise. The exists flag can be useful if you are storing nil values in the tree.
func (a *Tree[V]) Get(key []byte) (value V, exists bool) {
	var zero V
	if a.root == nil {
		return zero, false
	}
	curr := a.root
	for {
		h := curr.header()
		if !bytes.HasPrefix(key, h.path.asSlice()) {
			return zero, false
		}
		key = key[h.path.len:]
		if len(key) == 0 {
			leaf := curr.valueNode()
			if leaf != nil {
				return leaf.value, true
			}
			return zero, false
		}
		next := curr.getChildNode(key)
		if next == nil {
			return zero, false
		}
		curr = *next
		key = key[1:]
	}
}

// Delete removes the value associated with the supplied key if it exists. Its okay to
// call Delete with a key that doesn't exist.
func (a *Tree[V]) Delete(key []byte) {
	if a.root == nil {
		return
	}
	a.root = a.delete(a.root, key)
}

func (a *Tree[V]) delete(n node[V], key []byte) node[V] {
	h := n.header()
	if !bytes.HasPrefix(key, h.path.asSlice()) {
		return n
	}
	key = key[h.path.len:]
	if len(key) == 0 {
		return n.removeValue()
	}
	next := n.getChildNode(key)
	if next == nil {
		return n
	}
	*next = a.delete(*next, key[1:])
	if *next == nil {
		n.removeChild(key[0])
	}
	return n.shrink()
}

// WalkState describes how to proceed with an iteration of the tree (or partial tree).
type WalkState byte

const (
	// Continue will cause the tree iteration to continue on to the next key.
	Continue WalkState = iota
	// Stop will halt the iteration at this point.
	Stop
)

// ConsumerFn is the type of the callback function. It is called with key/value pairs in key order
// and the return value can be used to signal to continue or stop the iteration. The
// key value is only valid for the duration of the callback, and it should not be
// modified. If the callback needs access to the key after the callback returns, it
// must copy the key. The tree should not be modified during a callback.
//type ConsumerFn[V any] func[V](key []byte, value V) WalkState

// Walk will call the provided callback function with each key/value pair, in key order.
// The callback return value can be used to continue or stop the walk
func (a *Tree[V]) Walk(callback func(key []byte, value V) WalkState) {
	if a.root == nil {
		return
	}
	a.walk(a.root, make([]byte, 0, 32), callback)
}

func (a *Tree[V]) walk(n node[V], prefix []byte, callback func(key []byte, value V) WalkState) WalkState {
	h := n.header()
	prefix = append(prefix, h.path.asSlice()...)
	if h.hasValue {
		leaf := n.valueNode()
		if callback(prefix, leaf.value) == Stop {
			return Stop
		}
	}
	return n.iterateChildren(func(k byte, cn node[V]) WalkState {
		return a.walk(cn, append(prefix, k), callback)
	})
}

// WalkRange will call the provided callback function with each key/value pair, in key order.
// keys will be limited to those equal to or greater than start and less than end. So its inclusive
// of start, and exclusive of end.
// nil can used to mean no limit in that direction. e.g. WalkRange(nil,nil,cb) is the same as
// WalkRange(cb). WalkRange([]byte{1}, nil, cb) will wall all that are equal to or greater than [1]
// WalkRange([]byte{1}, []byte{2},cb) will walk all keys with a prefix of [1].
// The callback return value can be used to continue or stop the walk
func (a *Tree[V]) WalkRange(start []byte, end []byte, callback func(key []byte, value V) WalkState) {
	if a.root == nil {
		return
	}
	cmpEnd := keyLimit{end, 0}
	if len(end) == 0 {
		cmpEnd = keyLimit{end, -1}
	}
	a.walkStart(a.root, make([]byte, 0, 32), keyLimit{start, 0}, cmpEnd, callback)
}

func (a *Tree[V]) walkStart(n node[V], current []byte, start, end keyLimit, callback func(key []byte, value V) WalkState) WalkState {
	h := n.header()
	for _, k := range h.path.asSlice() {
		start.cmpSegment(k)
		end.cmpSegment(k)
	}
	if end.eqOrGreaterThan() {
		return Stop
	}
	current = append(current, h.path.asSlice()...)
	if start.eqOrGreaterThan() && h.hasValue {
		leaf := n.valueNode()
		if callback(current, leaf.value) == Stop {
			return Stop
		}
	}
	return n.iterateChildrenRange(start.minNextKey(), end.stopKey(), func(k byte, cn node[V]) WalkState {
		nextStart, nextEnd := start, end
		nextStart.cmpSegment(k)
		nextEnd.cmpSegment(k)
		return a.walkStart(cn, append(current, k), nextStart, nextEnd, callback)
	})
}

type keyLimit struct {
	path []byte
	cmp  int
}

// eqOrGreaterThan will return true if the current key is greater than or equal to the limit key
func (l *keyLimit) eqOrGreaterThan() bool {
	return l.cmp > 0 || (l.cmp == 0 && len(l.path) == 0)
}

func (l *keyLimit) minNextKey() int {
	if len(l.path) > 0 && l.cmp == 0 {
		return int(l.path[0])
	}
	return 0
}
func (l *keyLimit) stopKey() int {
	if len(l.path) > 0 && l.cmp == 0 {
		return int(l.path[0]) + 1
	}
	return 256
}

// cmpSegment will update our state based on the provided segment of the key path.
func (l *keyLimit) cmpSegment(k byte) {
	if l.cmp != 0 {
		return
	}
	if len(l.path) > 0 {
		l.cmp = compare(k, l.path[0])
		l.path = l.path[1:]
	} else {
		l.cmp = 1
	}
}

func compare(a, b byte) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// PrettyPrint will generate a compact representation of the state of the tree. Its primary
// use is in diagnostics, or helping to understand how the tree is constructed.
func (a *Tree[V]) PrettyPrint(w io.Writer) {
	if a.root == nil {
		io.WriteString(w, "[empty]\n")
		return
	}
	bw := bufio.NewWriter(w)
	a.root.pretty(0, bw)
	bw.Flush()
}

// Stats contains counts of items in the tree
type Stats struct {
	Node4s   int
	Node16s  int
	Node48s  int
	Node256s int
	Keys     int
}

// Stats returns current statistics about the nodes & keys in the tree.
func (a *Tree[V]) Stats() *Stats {
	s := &Stats{}
	if a.root == nil {
		return s
	}
	a.root.stats(s)
	return s
}

type writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
}

//type nodeConsumer func[V any](k byte, n node[V]) WalkState

type node[V any] interface {
	header() nodeHeader
	keyPath() *keyPath

	canAddChild() bool
	addChildNode(key byte, child node[V])
	getChildNode(key []byte) *node[V]
	iterateChildren(cb func(k byte, n node[V]) WalkState) WalkState
	// iterateChildrenRange a potential subset of children where start >= key < end
	iterateChildrenRange(start, end int, cb func(k byte, n node[V]) WalkState) WalkState

	canSetNodeValue() bool
	setNodeValue(n *leaf[V])
	valueNode() *leaf[V]

	// remove the value (or child) for this node, the node can be removed from the tree
	// if it returns nil, or it can return a different node instance and the
	// tree will be updated to that one (i.e. so that nodes can shrink to
	// a smaller type)
	removeValue() node[V]
	removeChild(key byte)

	grow() node[V]
	shrink() node[V]

	pretty(indent int, dest writer)
	stats(s *Stats)
}

type nodeHeader struct {
	// number of populated children in this node
	childCount int16
	// if set, this node has a value associated with it, not just child nodes
	// how/where the value is kept is node type dependent. node4/16/48 keep
	// it in the last child, and have 1 less max children
	hasValue bool
	// additional key values to this node (for path compression, lazy expansion)
	path keyPath
}

// splitNodePath will if needed split the supplied node into 2 based on the
// overlap of the key and the nodes compressed path. If the key and the path are the
// same then there's no need to split and the node is returned unaltered.
func splitNodePath[V any](key []byte, n node[V]) (remainingKey []byte, out node[V]) {
	h := n.header()
	path := h.path.asSlice()
	prefixLen := prefixSize(key, path)
	if prefixLen < len(path) {
		// need to split into 2
		parent := &node4[V]{}
		parent.path.assign(path[:prefixLen])
		parent.addChildNode(path[prefixLen], n)
		// +1 because we consumed a byte for the child key
		n.keyPath().trimPathStart(prefixLen + 1)
		return key[prefixLen:], parent
	}
	return key[prefixLen:], n
}

func writePath(p []byte, w io.Writer) {
	for _, k := range p {
		fmt.Fprintf(w, " 0x%02X", k)
	}
}

var spaces = bytes.Repeat([]byte{' '}, 16)

func writeIndent(indent int, w io.Writer) {
	if indent > len(spaces) {
		spaces = bytes.Repeat([]byte{' '}, indent*2)
	}
	w.Write(spaces[:indent])
}

func writeNode[V any](n node[V], name string, indent int, w writer) {
	w.WriteByte('[')
	w.WriteString(name)
	h := n.header()
	writePath(h.path.asSlice(), w)
	w.WriteString("] ")
	if h.hasValue {
		w.WriteString(" value: ")
		n.valueNode().pretty(indent, w)
	} else {
		w.WriteByte('\n')
	}
	n.iterateChildren(func(k byte, child node[V]) WalkState {
		writeIndent(indent+2, w)
		fmt.Fprintf(w, "0x%02X: ", k)
		child.pretty(indent+8, w)
		return Continue
	})
}

// prefixSize returns the length of the common prefix of the 2 slices.
func prefixSize(a, b []byte) int {
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return i
}
