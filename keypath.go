package art

import "strings"

// keyPath contains up to 23 bytes of key that has been compressed into a node for path compression/lazy expansion.
// Compressed paths longer than 23 bytes will need intermediate node4s to extend the path. Unless you have very long
// sparse keys its unlikely that a compressed path will need more than 23 bytes.
type keyPath struct {
	key [23]byte
	len byte
}

func (k *keyPath) asSlice() []byte {
	return k.key[:k.len]
}

func (k *keyPath) assign(p []byte) {
	if len(p) > len(k.key) {
		panic("Tried to assign a compressed path longer than supported")
	}
	copy(k.key[:], p)
	k.len = byte(len(p))
}

func (k *keyPath) trimPathStart(amount int) {
	copy(k.key[:], k.key[amount:k.len])
	k.len -= byte(amount)
}

func (k *keyPath) canExtendBy(additional byte) bool {
	return k.len+additional <= byte(len(k.key))
}

func (k *keyPath) prependPath(prefix []byte, extra ...byte) {
	additionalLen := len(prefix) + len(extra)
	if int(k.len)+additionalLen > len(k.key) {
		panic("Attempt to extend path outside of supported size.")
	}
	// [prefix] [extra] [existing path]
	copy(k.key[additionalLen:], k.key[:k.len])
	copy(k.key[:], prefix)
	copy(k.key[len(prefix):], extra)
	k.len += byte(additionalLen)
}

func (k *keyPath) String() string {
	b := strings.Builder{}
	writePath(k.asSlice(), &b)
	return b.String()
}
