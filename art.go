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
		next, remainingKey, _ := curr.getNextNode(key)
		if next == nil {
			return nil, false
		}
		if next == curr {
			return next.nodeValue()
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
	next, remainingKey, remover := n.getNextNode(key)
	if next == nil {
		return false
	}
	if next == n {
		return remover()
	}
	if a.delete(next, remainingKey) {
		return remover()
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

	getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool)

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

func (n *node4) removeValue() bool {
	n.children[n4ValueIdx] = nil
	n.hasValue = false
	return n.childCount == 0
}

func (n *node4) childRemover(i int) func() bool {
	return func() bool {
		lastIdx := n.childCount - 1
		n.children[i] = n.children[lastIdx]
		n.children[lastIdx] = nil
		n.key[i] = n.key[lastIdx]
		n.key[lastIdx] = 0
		n.childCount--
		return n.childCount == 0 && !n.hasValue
	}
}

func (n *node4) getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool) {
	if len(key) == 0 {
		return n, key, n.removeValue
	}
	for i := 0; i < int(n.childCount); i++ {
		if key[0] == n.key[i] {
			return n.children[i], key[1:], n.childRemover(i)
		}
	}
	return nil, nil, nil
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

func (n *node16) childRemover(i int) func() bool {
	return func() bool {
		lastIdx := n.childCount - 1
		n.children[i] = n.children[lastIdx]
		n.children[lastIdx] = nil
		n.key[i] = n.key[lastIdx]
		n.key[lastIdx] = 0
		n.childCount--
		return n.childCount == 0 && !n.hasValue
	}
}

func (n *node16) getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool) {
	if len(key) == 0 {
		return n, key, n.removeValue
	}
	for i := 0; i < int(n.childCount); i++ {
		if key[0] == n.key[i] {
			return n.children[i], key[1:], n.childRemover(i)
		}
	}
	return nil, nil, nil
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
		n256.value = value
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

func (n *node48) childRemover(key byte, slot byte) func() bool {
	return func() bool {
		lastSlot := byte(n.childCount - 1)
		n.children[slot] = n.children[lastSlot]
		n.key[key] = n48NoChildForKey
		for key, slot := range n.key {
			if slot == lastSlot {
				n.key[key] = slot
				break
			}
		}
		return n.childCount == 0 && !n.hasValue
	}
}

func (n *node48) getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool) {
	if len(key) == 0 {
		return n, key, n.removeValue
	}
	idx := n.key[key[0]]
	if idx == n48NoChildForKey {
		return nil, nil, nil
	}
	return n.children[idx], key[1:], n.childRemover(key[0], idx)
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

type node256 struct {
	children [256]node
	value    interface{}
	nodeHeader
}

func (n *node256) header() nodeHeader {
	return n.nodeHeader
}

func newNode256(src *node48) *node256 {
	n := &node256{nodeHeader: src.nodeHeader}
	if src.hasValue {
		var exists bool
		n.value, exists = src.children[n48ValueIdx].nodeValue()
		if !exists {
			panic("error, src node48 said it had a value, but does not")
		}
	}
	for k, slot := range src.key {
		if slot != n48NoChildForKey {
			n.children[k] = src.children[slot]
		}
	}
	return n
}

func (n *node256) insert(key []byte, value interface{}) node {
	splitN, replaced, prefixLen := splitNodePath(key, n.path, n)
	if replaced {
		return splitN.insert(key, value)
	}
	key = key[prefixLen:]
	if len(key) == 0 {
		n.hasValue = true
		n.value = value
		return n
	}
	c := n.children[key[0]]
	if c == nil {
		n.children[key[0]] = newNode(key[1:], value)
		n.childCount++
	} else {
		n.children[key[0]] = c.insert(key[1:], value)
	}
	return n
}

func (n *node256) removeValue() bool {
	n.hasValue = false
	n.value = nil
	return n.childCount == 0
}

func (n *node256) childRemover(k byte) func() bool {
	return func() bool {
		n.children[k] = nil
		n.childCount--
		return n.childCount == 0 && !n.hasValue
	}
}

func (n *node256) getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool) {
	if len(key) == 0 {
		if n.hasValue {
			return n, key, n.removeValue
		}
		return nil, nil, nil
	}
	c := n.children[key[0]]
	if c == nil {
		return nil, nil, nil
	}
	return c, key[1:], n.childRemover(key[0])
}

func (n *node256) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.value, true
	}
	return nil, false
}

func (n *node256) walk(prefix []byte, cb ConsumerFn) WalkState {
	prefix = append(prefix, n.path...)
	v, exists := n.nodeValue()
	if exists && cb(prefix, v) == Stop {
		return Stop
	}
	for idx, c := range n.children {
		if c != nil && c.walk(append(prefix, byte(idx)), cb) == Stop {
			return Stop
		}
	}
	return Continue
}

func (n *node256) pretty(indent int, w writer) {
	w.WriteString("[n256] ")
	writePath(n.path, w)
	if n.hasValue {
		fmt.Fprintf(w, "value: %v", n.value)
	}
	w.WriteByte('\n')
	for idx, c := range n.children {
		if c != nil {
			writeIndent(indent+2, w)
			fmt.Fprintf(w, "0x%02X: ", idx)
			c.pretty(indent+8, w)
		}
	}
}

func (n *node256) stats(s *Stats) {
	s.Node256s++
	if n.hasValue {
		s.Keys++
	}
	for _, c := range n.children {
		if c != nil {
			c.stats(s)
		}
	}
}

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

func (l *leaf) getNextNode(key []byte) (next node, remainingKey []byte, remover func() bool) {
	if len(key) == 0 {
		return l, []byte{}, func() bool { return true }
	}
	return nil, nil, nil
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
