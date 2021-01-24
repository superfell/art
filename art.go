package art

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Tree is an Adaptive Radix Tree. Keys are arbitrary byte slices, and the path through the tree
// is the key. values are stored on Leafs of the tree. The tree is organized in lexicographical
// order of the keys.
type Tree struct {
	root node
}

// Put inserts or updates a value in the tree associated with the provided key. value can be any
// interface value, including nil. key can be an arbitrary byte slice, including the empty slice.
func (a *Tree) Put(key []byte, value interface{}) {
	if a.root == nil {
		a.root = newNode(key, value)
	} else {
		a.root = a.root.insert(key, value)
	}
}

// Get the value for the provided key. exists is true if the key contains a value in the tree,
// false otherwise. This can be useful if you are storing nil values in the tree.
func (a *Tree) Get(key []byte) (value interface{}, exists bool) {
	if a.root == nil {
		return nil, false
	}
	curr := a.root
	for {
		h := curr.header()
		if !bytes.HasPrefix(key, h.path) {
			return nil, false
		}
		key = key[len(h.path):]
		if len(key) == 0 {
			return curr.nodeValue()
		}
		next, remainingKey := curr.getNextNode(key)
		if next == nil {
			return nil, false
		}
		curr = next
		key = remainingKey
	}
}

// Delete removes the value associated with the supplied key if it exists. Its okay to
// call Delete with a key that doesn't exist.
func (a *Tree) Delete(key []byte) {
	if a.root == nil {
		return
	}
	if a.delete(a.root, key) {
		a.root = nil
	}
}

func (a *Tree) delete(n node, key []byte) bool {
	h := n.header()
	if !bytes.HasPrefix(key, h.path) {
		return false
	}
	key = key[len(h.path):]
	if len(key) == 0 {
		return n.removeValue()
	}
	next, remainingKey := n.getNextNode(key)
	if next == nil {
		return false
	}
	if a.delete(next, remainingKey) {
		return n.removeChild(key[0])
	}
	return false
}

// WalkState describes how to proceed with an iteration of the tree (or partial tree).
type WalkState byte

const (
	// Continue will cause the tree iteration to continue on to the next key.
	Continue WalkState = iota
	// Stop will halt the iteration at this point.
	Stop
)

// ConsumerFn is the type of the callback function. It is called with key/value pairs
// and the return value can be used to signal to continue or stop the iteration.
type ConsumerFn func(key []byte, value interface{}) WalkState

// Walk will call the provided callback function with each key/value pair, in key order.
// The callback return value can be used to continue or stop the walk
func (a *Tree) Walk(callback ConsumerFn) {
	if a.root == nil {
		return
	}
	a.root.walk(nil, callback)
}

// PrettyPrint will generate a compact representation of the state of the tree. Its primary
// use is in diagnostics, or helping to understand how the tree is constructed.
func (a *Tree) PrettyPrint(w io.Writer) {
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
func (a *Tree) Stats() *Stats {
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

type node interface {
	header() nodeHeader
	insert(key []byte, value interface{}) node
	trimPathStart(amount int)

	getNextNode(key []byte) (next node, remainingKey []byte)

	removeValue() bool
	removeChild(key byte) bool

	nodeValue() (value interface{}, exists bool)
	walk(prefix []byte, callback ConsumerFn) WalkState

	pretty(indent int, dest writer)
	stats(s *Stats)
}

func newNode(key []byte, value interface{}) node {
	return &leaf{
		path:  key,
		value: value,
	}
}

type nodeHeader struct {
	// additional key values to this node (for path compression, lazy expansion)
	path []byte
	// number of populated children in this node
	childCount int16
	// if set, this node has a value associated with it, not just child nodes
	// how/where the value is kept is node type dependent. node4/16/48 keep
	// it in the last child, and have 1 less max children
	hasValue bool
}

func (h *nodeHeader) trimPathStart(amount int) {
	h.path = h.path[amount:]
}

// index into the children arrays for the node value leaf.
const n4ValueIdx = 3
const n16ValueIdx = 15
const n48ValueIdx = 47

// splitNodePath will if needed split the supplied node into 2 based on the
// overlap of the key and the nodes compressed path. If the key and the path are the
// same then there's no need to split and the node is returned unaltered.
func splitNodePath(key []byte, path []byte, n node) (out node, replaced bool, prefixLen int) {
	prefixLen = prefixSize(key, path)
	if prefixLen < len(path) {
		// need to split into 2
		parent := &node4{}
		parent.path = path[:prefixLen]
		parent.childCount = 1
		parent.key[0] = path[prefixLen]
		parent.children[0] = n
		// +1 because we consumed a byte for the child key
		n.trimPathStart(prefixLen + 1)
		return parent, true, prefixLen
	}
	return n, false, prefixLen
}

func writePath(p []byte, w writer) {
	if len(p) > 0 {
		w.WriteString(" [")
		for i, k := range p {
			if i > 0 {
				w.WriteByte(' ')
			}
			fmt.Fprintf(w, "0x%02X", k)
		}
		w.WriteByte(']')
	}
}

var spaces = bytes.Repeat([]byte{' '}, 16)

func writeIndent(indent int, w io.Writer) {
	if indent > len(spaces) {
		spaces = bytes.Repeat([]byte{' '}, indent*2)
	}
	w.Write(spaces[:indent])
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
