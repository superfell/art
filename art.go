package art

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Art ...
type Art struct {
	root node
}

// Insert ....
func (a *Art) Insert(key []byte, value interface{}) {
	if a.root == nil {
		a.root = newNode(key, value)
	} else {
		a.root = a.root.insert(key, value)
	}
}

// Get ...
func (a *Art) Get(key []byte) (value interface{}, exists bool) {
	if a.root == nil {
		return nil, false
	}
	n := a.root.get(key)
	if n == nil {
		return nil, false
	}
	return n.nodeValue()
}

type WalkState byte

const (
	Continue WalkState = iota
	Stop
)

// ConsumerFn ...
type ConsumerFn func(key []byte, value interface{}) WalkState

// Walk will call the provided callback function with each key/value pair, in key order.
// the callback return value can be used to continue or stop the walk
func (a *Art) Walk(callback ConsumerFn) {
	if a.root == nil {
		return
	}
	a.root.walk(nil, callback)
}

// PrettyPrint ...
func (a *Art) PrettyPrint(w io.Writer) {
	if a.root == nil {
		io.WriteString(w, "[empty]\n")
		return
	}
	bw := bufio.NewWriter(w)
	a.root.pretty(0, bw)
	bw.Flush()
}

type Stats struct {
	Node4s   int
	Node16s  int
	Node48s  int
	Node256s int
	Leafs    int
	Keys     int
}

// Stats returns current statistics about the nodes & keys in the tree.
func (a *Art) Stats() *Stats {
	s := &Stats{}
	if a.root == nil {
		return s
	}
	a.root.stats(s)
	return s
}

type node interface {
	insert(key []byte, value interface{}) node
	get(key []byte) node
	nodeValue() (value interface{}, exists bool)
	walk(prefix []byte, callback ConsumerFn) WalkState
	pretty(indent int, dest *bufio.Writer)
	stats(s *Stats)
	trimPathStart(amount int)
}

func newNode(key []byte, value interface{}) node {
	return &leaf{
		path:  key,
		value: value,
	}
}

type header struct {
	// additional key values to this node (for path compression, lazy expansion)
	path []byte
	// number of populated children in this node [not for node256]
	childCount byte
	// if set, this node has a value associated with it, not just child nodes
	// how/where the value is kept is node type dependent. node4/16/48 keep
	// it in the last child, and have 1 less max children
	hasValue bool
}

func (h *header) trimPathStart(amount int) {
	h.path = h.path[amount:]
}

const n4ValueIdx = 3
const n16ValueIdx = 15
const n48ValueIdx = 47

type node4 struct {
	header
	key      [4]byte
	children [4]node
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
		if n.header.hasValue {
			n.children[n4ValueIdx].insert(key, value)
			return n
		}
		if n.header.childCount < 4 {
			n.children[n4ValueIdx] = newNode(key, value)
			n.header.hasValue = true
			return n
		}
		// we're full, need to grow
		n16 := newNode16(n)
		n16.children[n16ValueIdx] = newNode(key, value)
		n16.hasValue = true
		return n16
	}
	for i := byte(0); i < n.header.childCount; i++ {
		if n.key[i] == key[0] {
			n.children[i] = n.children[i].insert(key[1:], value)
			return n
		}
	}
	maxChildren := byte(4)
	if n.hasValue {
		maxChildren--
	}
	if n.childCount < maxChildren {
		idx := n.childCount
		n.key[idx] = key[0]
		n.children[idx] = newNode(key[1:], value)
		n.header.childCount++
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

func (n *node4) get(key []byte) node {
	if !bytes.HasPrefix(key, n.path) {
		return nil
	}
	key = key[len(n.path):]
	if len(key) == 0 {
		return n
	}
	for i := byte(0); i < n.childCount; i++ {
		if key[0] == n.key[i] {
			return n.children[i].get(key[1:])
		}
	}
	return nil
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
	for i := byte(0); i < n.childCount; i++ {
		next := byte(255)
		nextIdx := byte(255)
		for j := byte(0); j < n.childCount; j++ {
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

func (n *node4) pretty(indent int, w *bufio.Writer) {
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
	for i := byte(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}

type node16 struct {
	header
	key      [16]byte
	children [16]node
}

// constructs a new Node16 from a Node4 and adds the additional child value
func newNode16(src *node4) *node16 {
	n := node16{header: src.header}
	for i := 0; i < int(src.header.childCount); i++ {
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
		if n.header.hasValue {
			n.children[n16ValueIdx].insert(key, value)
			return n
		}
		if n.header.childCount < 16 {
			n.children[n16ValueIdx] = newNode(key, value)
			n.header.hasValue = true
			return n
		}
		// we're full, need to grow
		n48 := newNode48(n)
		n48.children[n48ValueIdx] = newNode(key, value)
		n48.hasValue = true
		return n48
	}
	for i := byte(0); i < n.childCount; i++ {
		if n.key[i] == key[0] {
			n.children[i] = n.children[i].insert(key[1:], value)
			return n
		}
	}
	maxChildren := byte(16)
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
	n.header.childCount++
}

func (n *node16) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n16ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node16) get(key []byte) node {
	if !bytes.HasPrefix(key, n.path) {
		return nil
	}
	key = key[len(n.path):]
	if len(key) == 0 {
		return n
	}
	for i := byte(0); i < n.childCount; i++ {
		if key[0] == n.key[i] {
			return n.children[i].get(key[1:])
		}
	}
	return nil
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
	for i := byte(0); i < n.childCount; i++ {
		next := byte(255)
		nextIdx := byte(255)
		for j := byte(0); j < n.childCount; j++ {
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

func (n *node16) pretty(indent int, w *bufio.Writer) {
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
	for i := byte(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}

type node48 struct {
	header
	key      [256]byte
	children [48]node
}

func newNode48(src *node16) *node48 {
	n := &node48{header: src.header}
	if src.hasValue {
		n.children[n48ValueIdx] = src.children[n16ValueIdx]
	}
	for i := byte(0); i < src.childCount; i++ {
		n.key[src.key[i]] = i + 1
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
		if n.header.hasValue {
			n.children[n48ValueIdx].insert(key, value)
			return n
		}
		if n.header.childCount < 48 {
			n.header.hasValue = true
			n.children[n48ValueIdx] = newNode(key, value)
			return n
		}
		// ugh, we're full, this'll drop through to the grow at the bottom
		n256 := newNode256(n)
		n256.value = value
		n256.hasValue = true
		return n256
	}
	slot := n.key[key[0]]
	if slot > 0 {
		slot = slot - 1
		n.children[slot] = n.children[slot].insert(key[1:], value)
		return n
	}
	maxSlots := byte(48)
	if n.hasValue {
		maxSlots--
	}
	if n.childCount < maxSlots {
		n.addChildLeaf(key, value)
		return n
	}
	n256 := newNode256(n)
	n256.children[key[0]] = newNode(key[1:], value)
	return n256
}

func (n *node48) addChildLeaf(key []byte, val interface{}) {
	n.key[key[0]] = n.childCount + 1
	n.children[n.childCount] = newNode(key[1:], val)
	n.childCount++
}

func (n *node48) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n48ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node48) get(key []byte) node {
	if !bytes.HasPrefix(key, n.path) {
		return nil
	}
	key = key[len(n.path):]
	if len(key) == 0 {
		return n
	}
	idx := n.key[key[0]]
	if idx == 0 {
		return nil
	}
	return n.children[idx-1].get(key[1:])
}

func (n *node48) walk(prefix []byte, cb ConsumerFn) WalkState {
	prefix = append(prefix, n.path...)
	v, exists := n.nodeValue()
	if exists && cb(prefix, v) == Stop {
		return Stop
	}
	for idx, slot := range n.key {
		if slot > 0 {
			if n.children[slot-1].walk(append(prefix, byte(idx)), cb) == Stop {
				return Stop
			}
		}
	}
	return Continue
}

func (n *node48) pretty(indent int, w *bufio.Writer) {
	w.WriteString("[n48] ")
	writePath(n.path, w)
	if n.hasValue {
		w.WriteString(" value: ")
		n.children[n48ValueIdx].pretty(indent, w)
	} else {
		w.WriteByte('\n')
	}
	for k, slot := range n.key {
		if slot > 0 {
			writeIndent(indent+2, w)
			fmt.Fprintf(w, "0x%02X: ", k)
			n.children[slot-1].pretty(indent+8, w)
		}
	}
}

func (n *node48) stats(s *Stats) {
	s.Node48s++
	if n.hasValue {
		n.children[n48ValueIdx].stats(s)
	}
	for i := byte(0); i < n.childCount; i++ {
		n.children[i].stats(s)
	}
}

type node256 struct {
	children [256]node
	value    interface{}
	header
}

func newNode256(src *node48) *node256 {
	n := &node256{header: src.header}
	if src.hasValue {
		var exists bool
		n.value, exists = src.children[n48ValueIdx].nodeValue()
		if !exists {
			panic("error, src node48 said it had a value, but does not")
		}
	}
	for k, slot := range src.key {
		if slot > 0 {
			n.children[k] = src.children[slot-1]
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
	} else {
		n.children[key[0]] = c.insert(key[1:], value)
	}
	return n
}

func (n *node256) get(key []byte) node {
	if !bytes.HasPrefix(key, n.path) {
		return nil
	}
	key = key[len(n.path):]
	if len(key) == 0 {
		if n.hasValue {
			return n
		}
		return nil
	}
	c := n.children[key[0]]
	if c == nil {
		return nil
	}
	return c.get(key[1:])
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

func (n *node256) pretty(indent int, w *bufio.Writer) {
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
		// there's no actual leaf instance, but its logically a leaf
		s.Leafs++
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
	if len(key) > 0 {
		// we need to "promote" this leaf to a node with a contained value
		n := &node4{}
		n.path = l.path
		l.path = nil
		n.hasValue = true
		n.children[3] = l
		return n.insert(key, value)
	}
	// key & path both empty update ourselves
	l.value = value
	return l
}

func (l *leaf) nodeValue() (interface{}, bool) {
	return l.value, true
}

func (l *leaf) get(key []byte) node {
	if bytes.Equal(key, l.path) {
		return l
	}
	return nil
}

func (l *leaf) walk(prefix []byte, cb ConsumerFn) WalkState {
	return cb(append(prefix, l.path...), l.value)
}

func (l *leaf) pretty(indent int, w *bufio.Writer) {
	w.WriteString("[leaf] ")
	writePath(l.path, w)
	fmt.Fprintf(w, " value:%v\n", l.value)
}

func (l *leaf) stats(s *Stats) {
	s.Leafs++
	s.Keys++
}

func writePath(p []byte, w *bufio.Writer) {
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

func writeIndent(indent int, w *bufio.Writer) {
	if indent > len(spaces) {
		spaces = append(spaces, bytes.Repeat([]byte{' '}, indent-len(spaces)+4)...)
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
