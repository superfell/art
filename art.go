package art

import "bytes"

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

type node interface {
	insert(key []byte, value interface{}) node
	get(key []byte) node
	nodeValue() (value interface{}, exists bool)
	walk(prefix []byte, callback ConsumerFn) WalkState
}

func newNode(key []byte, value interface{}) node {
	return &leaf{
		path:  key,
		value: value,
	}
}

type header struct {
	// path prefix to this node
	path []byte
	// number of populated children in this node
	childCount byte
	// if set, this node has a value associated with it, not just child nodes
	// how/where the value is kept is node type dependent. node4/16/48 keep
	// it in the last child, and have 1 less max children
	hasValue bool
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
		// ugh, we're full, this'll drop through to the grow at the bottom
	} else {
		for i := byte(0); i < n.header.childCount; i++ {
			if n.key[i] == key[0] {
				n.children[i] = n.children[i].insert(key[1:], value)
				return n
			}
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
	return newNode16(n).insert(key, value)
}

func (n *node4) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n4ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node4) get(key []byte) node {
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

type node16 struct {
	header
	key      [16]byte
	children [16]node
}

// constructs a new Node16 from a Node4
func newNode16(src *node4) *node16 {
	n := node16{header: src.header}
	maxSrcSlots := byte(4)
	if src.hasValue {
		maxSrcSlots--
	}
	for i := byte(0); i < maxSrcSlots; i++ {
		n.key[i] = src.key[i]
		n.children[i] = src.children[i]
	}
	if src.hasValue {
		n.children[n16ValueIdx] = src.children[n4ValueIdx]
	}
	return &n
}

func (n *node16) insert(key []byte, value interface{}) node {
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
		// ugh, we're full, this'll drop through to the grow at the bottom
	} else {
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
			idx := n.childCount
			n.key[idx] = key[0]
			n.children[idx] = newNode(key[1:], value)
			n.childCount++
			return n
		}
	}
	return newNode48(n).insert(key, value)
}

func (n *node16) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n16ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node16) get(key []byte) node {
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

type node48 struct {
	header
	key      [256]byte
	children [48]node
}

func newNode48(src *node16) node {
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
	} else {
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
			n.key[key[0]] = n.childCount + 1
			n.children[n.childCount] = newNode(key[1:], value)
			n.childCount++
			return n
		}
	}
	return newNode256(n).insert(key, value)
}

func (n *node48) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[n48ValueIdx].nodeValue()
	}
	return nil, false
}

func (n *node48) get(key []byte) node {
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

type node256 struct {
	children [256]node
	value    interface{}
	header
}

func newNode256(src *node48) node {
	n := &node256{header: src.header}
	if src.hasValue {
		var exists bool
		n.value, exists = src.children[n48ValueIdx].nodeValue()
		if !exists {
			panic("error, src node48 said it had a value, but does not")
		}
	}
	for i, k := range src.key {
		if k > 0 {
			n.children[i] = src.children[k-1]
		}
	}
	return n
}

func (n *node256) insert(key []byte, value interface{}) node {
	if len(key) == 0 {
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

type leaf struct {
	value interface{}
	path  []byte
}

func (l *leaf) insert(key []byte, value interface{}) node {
	if len(key) > 0 || len(l.path) > 0 {
		// if there's a key, then we need to change this item to a node that contains this leaf as a value
		// then pass the key down to that node
		n := &node4{}
		if len(l.path) == 0 {
			// leaf can become the node's value
			n.hasValue = true
			n.children[3] = l
		} else {
			// the leaf has a path, so should be inserted into the node as a child
			n.insert(l.path, l.value)
		}
		return n.insert(key, value)
	}
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
