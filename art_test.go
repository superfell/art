package art

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func Test_Empty(t *testing.T) {
	testArt(t, []keyVal{}, &Stats{})
}

func Test_OverwriteWithSameKey(t *testing.T) {
	testArt(t, []keyVal{
		kvs("one", "one"),
		kvs("two", "two"),
		kvs("one", "three"),
	}, nil)
}

func Test_InsertOnLeaf(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "abc"),
		// now insert something that would add a child to the leaf above
		kvs("1234", "abcd"),
	}, nil)
}

func Test_LeafPathToNToLeafPath(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "1"),
		kvs("12345678", "2"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_MultipleInserts(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "abc"),
		kvs("456", "abcd"),
		kvs("1211", "def"),
	}, nil)
}

func Test_Grow4to16(t *testing.T) {
	keyVals := []keyVal{}
	k := []byte{65, 66}
	for i := byte(0); i < 10; i++ {
		keyVals = append(keyVals, kv(append(k, i), i))
	}
	keyVals = append(keyVals, kv(append(k, 5, 10), 100))
	testArt(t, keyVals, &Stats{Node4s: 1, Node16s: 1, Keys: 11})
}

func Test_Node4FullAddValue(t *testing.T) {
	testArt(t, []keyVal{
		kvs("11", "1"),
		kvs("12", "2"),
		kvs("13", "3"),
		kvs("14", "4"),
		kvs("1", "5"),
	}, &Stats{Node16s: 1, Keys: 5})
}
func Test_GrowTo48(t *testing.T) {
	keyVals := []keyVal{}
	k := []byte{65, 66}
	for i := byte(0); i < 40; i++ {
		keyVals = append(keyVals, kv(append(k, i), i))
	}
	keyVals = append(keyVals, kv(append(k, 5, 10), 100))
	testArt(t, keyVals, &Stats{Node48s: 1, Node4s: 1, Keys: 41})
}

func Test_GrowTo256(t *testing.T) {
	keyVals := []keyVal{}
	k := []byte{65, 66, 67}
	for i := 0; i < 256; i++ {
		keyVals = append(keyVals, kv(append(k, byte(i)), i))
	}
	keyVals = append(keyVals, kv(append(k, 5, 10), 100))
	testArt(t, keyVals, &Stats{Node256s: 1, Node4s: 1, Keys: 257})
}

func Test_GrowWithPrefixValue(t *testing.T) {
	keyVals := []keyVal{
		kvs("BBB", "kk"),
		kvs("B", "k"),
		kvs("BBx", 100),
	}
	for i := 0; i < 256; i++ {
		keyVals = append(keyVals, kv([]byte{'B', byte(i)}, i))
	}
	testArt(t, keyVals, nil)
}

func Test_KeyWithZeros(t *testing.T) {
	// any arbitrary byte array should be a valid key, even those with embedded nulls.
	testArt(t, []keyVal{
		kv([]byte{0, 0, 0}, "k1"),
		kv([]byte{0, 0, 0, 0}, "k2"),
		kv([]byte{0, 0, 0, 1}, "k3"),
		kv([]byte{0, 1, 0}, "k4"),
		kv([]byte{0, 1, 0, 1}, "k5"),
	}, nil)
}

func Test_EmptyKey(t *testing.T) {
	// an empty byte array is also a valid key
	t.Run("nil key", func(t *testing.T) {
		testArt(t, []keyVal{
			kv(nil, "k1"),
			kv([]byte{0}, "k2"),
		}, nil)
	})
	t.Run("empty key", func(t *testing.T) {
		testArt(t, []keyVal{
			kv([]byte{}, "k1"),
			kv([]byte{0}, "k2"),
		}, nil)
	})
}

func Test_NilValue(t *testing.T) {
	testArt(t, []keyVal{
		kv([]byte{0, 0, 0}, nil),
		kv([]byte{0, 0, 0, 1}, "3"),
		kv([]byte{10}, nil),
	}, nil)
}

func Test_NodeCompression(t *testing.T) {
	testArt(t, []keyVal{
		kvs("1234567", "1"),
		kvs("1239000", "2"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_LeafLazyExpansion(t *testing.T) {
	testArt(t, []keyVal{
		kvs("aaa", "foo"),
		kvs("aaattt", "bar"),
		kvs("aaatttxxx", "baz"),
	}, &Stats{Node4s: 2, Keys: 3})
}

func Test_Walk(t *testing.T) {
	testArt(t, []keyVal{
		kvs("C", "c"),
		kvs("A", "a"),
		kvs("AA", "aa"),
		kvs("B", "b"),
	}, &Stats{Node4s: 2, Keys: 4})
}

func Test_MoreWalk(t *testing.T) {
	sizes := []int{2, 4, 5, 16, 17, 47, 48, 49, 50, 120, 255, 256}
	for _, sz := range sizes {
		t.Run(fmt.Sprintf("Walk size %d", sz), func(t *testing.T) {
			a := new(Tree)
			baseK := []byte{'A'}
			for i := 0; i < sz; i++ {
				a.Insert(append(baseK, byte(i)), i)
			}
			t.Run("Full Walk", func(t *testing.T) {
				i := 0
				a.Walk(func(k []byte, v interface{}) WalkState {
					exp := append(baseK, byte(i))
					if !bytes.Equal(k, exp) {
						t.Errorf("Expecting key %v, but got %v", exp, k)
					}
					if v != i {
						t.Errorf("Expecting value %d for key %v but got %v", i, k, v)
					}
					i++
					return Continue
				})
				if i != sz {
					t.Errorf("Unexpected number of callbacks from walk, got %d, expecting %d", i, sz)
				}
			})
			t.Run("Early Stop", func(t *testing.T) {
				i := 0
				a.Walk(func(k []byte, v interface{}) WalkState {
					i++
					if i >= sz-1 {
						return Stop
					}
					return Continue
				})
				if i != sz-1 {
					t.Errorf("Unexpected number of callbacks with early termination, got %d, expecting %d", i, sz-1)
				}
			})
			t.Run("With NodeValues", func(t *testing.T) {
				for i := 0; i < sz; i++ {
					a.Insert(append(baseK, byte(i), byte(i)), i*i)
				}
				calls := 0
				prevKey := make([]byte, 0, 5)
				a.Walk(func(k []byte, v interface{}) WalkState {
					calls++
					if bytes.Compare(prevKey, k) != -1 {
						t.Errorf("Key %v received out of order, prevKey was %v", k, prevKey)
					}
					if len(k) == 2 && int(k[1]) != v {
						t.Errorf("Unexpected value %v for key %v, was expecting %v", v, k, k[1])
					}
					if len(k) == 3 {
						expV := int(k[2]) * int(k[2])
						if expV != v {
							t.Errorf("Unexpected value %v for key %v, was expecting %v", v, k, expV)
						}
					}
					prevKey = append(prevKey[:0], k...)
					return Continue
				})
				if calls != sz*2 {
					t.Errorf("Unexpected number of callbacks %d, expecting %d", calls, sz*2)
				}
			})
		})
	}
}

func testArt(t *testing.T, inserts []keyVal, expectedStats *Stats) {
	t.Run("normal", func(t *testing.T) {
		testArtOne(t, inserts, expectedStats)
	})
	t.Run("reverse insertion order", func(t *testing.T) {
		testArtOne(t, reverse(inserts), expectedStats)
	})
	t.Run("write twice", func(t *testing.T) {
		testArtOne(t, append(inserts, inserts...), expectedStats)
	})
	t.Run("write twice in reverse", func(t *testing.T) {
		testArtOne(t, reverse(append(inserts, inserts...)), expectedStats)
	})
}

func testArtOne(t *testing.T, inserts []keyVal, expectedStats *Stats) {
	a := new(Tree)
	defer func() {
		if t.Failed() {
			tree := &strings.Builder{}
			a.PrettyPrint(tree)
			t.Logf("tree\n%v", tree.String())
		}
	}()

	store := kvStore{}
	for i := 0; i < len(inserts); i++ {
		a.Insert(inserts[i].key, inserts[i].val)
		store.put(inserts[i])
		hasKeyVals(t, a, store.ordered())
		if t.Failed() {
			t.Logf("inserted %d keys, last inserted key %v", i+1, inserts[i].key)
			t.FailNow() // no point to keep going
		}
	}
	hasKeyVals(t, a, store.ordered())

	for i := 0; i < len(inserts)*2+4; i++ {
		k := rndKey()
		act, exists := a.Get(k)
		exp, shouldExist := store.get(k)
		if exists != shouldExist {
			t.Errorf("key %v expected to exist:%t actual:%t", k, shouldExist, exists)
		}
		if act != exp {
			t.Errorf("key %v expected value %v, actual value %v", k, exp, act)
		}
	}

	if expectedStats != nil {
		act := a.Stats()
		if !reflect.DeepEqual(*expectedStats, *act) {
			t.Errorf("Unexpected stats of %#v, expecting %#v", *act, *expectedStats)
		}
	}
}

// rndKey returns a random generated key
func rndKey() []byte {
	k := make([]byte, int(rnd.Int31n(15)))
	for i := 0; i < len(k); i++ {
		k[i] = byte(rnd.Int31n(256))
	}
	return k
}

var rnd = rand.New(rand.NewSource(42))

type keyVal struct {
	key []byte
	val interface{}
}

func kv(k []byte, v interface{}) keyVal {
	return keyVal{key: k, val: v}
}
func kvs(k string, v interface{}) keyVal {
	return keyVal{key: []byte(k), val: v}
}

func reverse(kv []keyVal) []keyVal {
	c := make([]keyVal, len(kv))
	j := len(kv) - 1
	for i := 0; i < len(kv); i++ {
		c[j] = kv[i]
		j--
	}
	return c
}

func hasKeyVals(t *testing.T, a *Tree, exp []keyVal) {
	// verifies that the tree matches the supplied set of kv's by using the Walk fn
	i := 0
	a.Walk(func(k []byte, v interface{}) WalkState {
		if !bytes.Equal(exp[i].key, k) {
			t.Errorf("key %d was %v but expecting %v", i, k, exp[i].key)
		}
		if v != exp[i].val {
			t.Errorf("key %v expecting value %v but was %v", exp[i].key, exp[i].val, v)
		}
		i++
		return Continue
	})
	if i < len(exp) {
		t.Errorf("Expecting %d keys to be walked, but only got %d", len(exp), i)
	}
	// check that the values are available via the Get fn as well
	for _, kv := range exp {
		actual, exists := a.Get(kv.key)
		if !exists {
			t.Errorf("key %v should have a value, but Get() says it doesn't", exists)
		}
		if actual != kv.val {
			t.Errorf("value %v for key %v is not the expected value %v", actual, kv.key, kv.val)
		}
	}
}

// kvStore is a really simple store that tracks keys & values. Its used to
// generate the expected key/values in the tree during tests.
type kvStore struct {
	kvs []keyVal
}

func (s *kvStore) put(kv keyVal) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(kv.key, s.kvs[i].key) {
			s.kvs[i].val = kv.val
			return
		}
	}
	s.kvs = append(s.kvs, kv)
}

func (s *kvStore) get(k []byte) (val interface{}, exists bool) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(k, s.kvs[i].key) {
			return s.kvs[i].val, true
		}
	}
	return nil, false
}

// ordered returns the contents of the store in key order
func (s *kvStore) ordered() []keyVal {
	sort.Slice(s.kvs, func(i, j int) bool {
		return bytes.Compare(s.kvs[i].key, s.kvs[j].key) == -1
	})
	return s.kvs
}
