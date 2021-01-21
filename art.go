package art

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

type node interface {
	insert(key []byte, value interface{}) node
	get(key []byte) node
	nodeValue() (value interface{}, exists bool)
}

func newNode(key []byte, value interface{}) node {
	if len(key) == 0 {
		return &leaf{value: value}
	}
	n := &node4{}
	n.header.childCount = 1
	n.key[0] = key[0]
	n.children[0] = newNode(key[1:], value)
	return n
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

type node4 struct {
	key      [4]byte
	children [4]node
	header
}

func (n *node4) insert(key []byte, value interface{}) node {
	if len(key) == 0 {
		// we're trying to insert a value at this path, and this path
		// is the prefix of some other path.
		// if we already have a value, then just update it
		if n.header.hasValue {
			n.children[3].insert(key, value)
			return n
		}
		if n.header.childCount < 4 {
			n.children[3] = newNode(key, value)
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
		return n.children[3].nodeValue()
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

type node16 struct {
	key      [16]byte
	children [16]node
	header
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
		n.children[15] = src.children[3]
	}
	return &n
}

func (n *node16) insert(key []byte, value interface{}) node {
	if len(key) == 0 {
		// we're trying to insert a value at this path, and this path
		// is the prefix of some other path.
		// if we already have a value, then just update it
		if n.header.hasValue {
			n.children[15].insert(key, value)
			return n
		}
		if n.header.childCount < 16 {
			n.children[15] = newNode(key, value)
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
		return n.children[15].nodeValue()
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

type node48 struct {
	key      [256]byte
	children [48]node
	header
}

func newNode48(src *node16) node {
	n := &node48{header: src.header}
	if src.hasValue {
		n.children[47] = src.children[15]
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
			n.children[47].insert(key, value)
			return n
		}
		if n.header.childCount < 48 {
			n.header.hasValue = true
			n.children[47] = newNode(key, value)
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
	panic("need to grow to node256")
}

func (n *node48) nodeValue() (interface{}, bool) {
	if n.hasValue {
		return n.children[47].nodeValue()
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
	return n.children[idx-1]
}

type node256 struct {
	children [256]node
	header
}

type leaf struct {
	header
	value interface{}
}

func (l *leaf) insert(key []byte, value interface{}) node {
	if len(key) > 0 {
		// if there's a key, then we need to change this item to a node that contains this leaf as a value
		// then pass the key down to that node
		n := &node4{}
		n.hasValue = true
		n.children[3] = l
		return n.insert(key, value)
	}
	l.value = value
	return l
}

func (l *leaf) nodeValue() (interface{}, bool) {
	return l.value, true
}

func (l *leaf) get(key []byte) node {
	if len(key) > 0 {
		return nil
	}
	return l
}
