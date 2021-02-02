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

func Test_JoinSlices(t *testing.T) {
	kEmpty := []byte(nil)
	k1 := []byte{15}
	k5 := []byte{1, 2, 3, 4, 5}
	test := func(a, b, c, exp []byte) {
		t.Helper()
		act := joinSlices(a, b, c)
		if !bytes.Equal(act, exp) {
			t.Errorf("joinSlice(%v,%v,%v) expected to generate %v but was %v", a, b, c, exp, act)
		}
	}
	test(kEmpty, kEmpty, kEmpty, kEmpty)
	test(k1, kEmpty, kEmpty, k1)
	test(kEmpty, k5, kEmpty, k5)
	test(kEmpty, kEmpty, k1, k1)
	test(kEmpty, k1, k5, []byte{15, 1, 2, 3, 4, 5})
	test(k1, kEmpty, k5, []byte{15, 1, 2, 3, 4, 5})
	test(k1, k5, kEmpty, []byte{15, 1, 2, 3, 4, 5})
	test(k1, k5, []byte{22, 23}, []byte{15, 1, 2, 3, 4, 5, 22, 23})
}

func Test_WriteIndent(t *testing.T) {
	for i := 2; i < 40; i++ {
		t.Run(fmt.Sprintf("length %d", i), func(t *testing.T) {
			w := strings.Builder{}
			writeIndent(i, &w)
			if len(w.String()) != i {
				t.Errorf("writeIndent(%d) wrote an indent of length %d", i, len(w.String()))
			}
		})
	}
}

func Test_Empty(t *testing.T) {
	testArt(t, []keyVal{}, &Stats{})
}

func Test_OverwriteWithSameKey(t *testing.T) {
	testArt(t, []keyVal{
		kvs("one", "one"),
		kvs("two", "two"),
		kvs("one", "three"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_InsertOnLeaf(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "abc"),
		// now insert something that would add a child to the leaf above
		kvs("1234", "abcd"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_LeafPathToNToLeafPath(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "1"),
		kvs("12345678", "2"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_SimpleMultipleInserts(t *testing.T) {
	testArt(t, []keyVal{
		kvs("123", "abc"),
		kvs("456", "abcd"),
		kvs("1211", "def"),
	}, &Stats{Node4s: 2, Keys: 3})
}

type simpleCase struct {
	children int
	stats    *Stats
}

func Test_GrowNode(t *testing.T) {
	cases := []simpleCase{
		// test that the node is grown to the relevant type
		{1, &Stats{Keys: 1}},
		{2, &Stats{Node4s: 1, Keys: 2}},
		{12, &Stats{Node16s: 1, Keys: 12}},
		{40, &Stats{Node48s: 1, Keys: 40}},
		{200, &Stats{Node256s: 1, Keys: 200}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("children %d", tc.children), func(t *testing.T) {
			inserts := []keyVal{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, byte(i)}, i))
			}
			testArt(t, inserts, tc.stats)
		})
	}
}

func Test_GrowNodeWithMixedChildren(t *testing.T) {
	cases := []simpleCase{
		// test that the node is grown to the relevant type, while the node contains
		// a mixed of leafs & nodes
		{2, &Stats{Node4s: 2, Keys: 4}},
		{12, &Stats{Node4s: 2, Node16s: 1, Keys: 14}},
		{40, &Stats{Node4s: 2, Node48s: 1, Keys: 42}},
		{200, &Stats{Node4s: 2, Node256s: 1, Keys: 202}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("children %d", tc.children), func(t *testing.T) {
			inserts := []keyVal{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, byte(i)}, i))
			}
			inserts = append(inserts, kv([]byte{1, 1, 10, 4}, "a"))
			inserts = append(inserts, kv([]byte{1, 11, 10, 4}, "b"))
			testArt(t, inserts, tc.stats)
		})
	}
}

func Test_SetValueOnExistingNode(t *testing.T) {
	cases := []simpleCase{
		// set value on an existing node with space for it
		{2, &Stats{Node4s: 1, Keys: 3}},
		{12, &Stats{Node16s: 1, Keys: 13}},
		{40, &Stats{Node48s: 1, Keys: 41}},
		{200, &Stats{Node256s: 1, Keys: 201}},

		// set value on an existing node that is already full and should grow
		{4, &Stats{Node16s: 1, Keys: 5}},
		{16, &Stats{Node48s: 1, Keys: 17}},
		{48, &Stats{Node256s: 1, Keys: 49}},
		{256, &Stats{Node256s: 1, Keys: 257}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("children %d", tc.children), func(t *testing.T) {
			inserts := []keyVal{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, byte(i)}, i))
			}
			inserts = append(inserts, kv([]byte{1}, "value"))
			testArt(t, inserts, tc.stats)
		})
	}
}

func Test_NodeInsertSplitsCompressedPath(t *testing.T) {
	cases := []simpleCase{
		{2, &Stats{Node4s: 2, Keys: 3}},
		{12, &Stats{Node4s: 1, Node16s: 1, Keys: 13}},
		{40, &Stats{Node4s: 1, Node48s: 1, Keys: 41}},
		{200, &Stats{Node4s: 1, Node256s: 1, Keys: 201}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("children %d", tc.children), func(t *testing.T) {
			inserts := []keyVal{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, 2, 3, 4, 5, 6, 7, byte(i + 10)}, i))
			}
			inserts = append(inserts, kv([]byte{1, 2, 3}, "123"))
			testArt(t, inserts, tc.stats)
		})
	}
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
	testArt(t, keyVals, &Stats{Node256s: 1, Node4s: 1, Keys: 259})
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
				a.Put(append(baseK, byte(i)), i)
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
			t.Run("Stop After First Key", func(t *testing.T) {
				i := 0
				a.Walk(func(k []byte, v interface{}) WalkState {
					i++
					return Stop
				})
				if i != 1 {
					t.Errorf("Unexpected number of callbacks with early termination, got %d, expecting %d", i, 1)
				}
			})
			t.Run("With NodeValues", func(t *testing.T) {
				for i := 0; i < sz; i++ {
					a.Put(append(baseK, byte(i), byte(i)), i*i)
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
			t.Logf("tree\n%v", pretty(a))
		}
	}()

	store := kvStore{}
	for i := 0; i < len(inserts); i++ {
		a.Put(inserts[i].key, inserts[i].val)
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
	if t.Failed() {
		t.FailNow() // no point to keep going
	}

	deletes := append([]keyVal{}, inserts...)
	rnd.Shuffle(len(deletes), func(i, j int) {
		deletes[i], deletes[j] = deletes[j], deletes[i]
	})

	for _, kv := range deletes {
		before := pretty(a)
		// t.Logf("About to delete key %s", hexPath(kv.key))
		a.Delete(kv.key)
		store.delete(kv.key)
		hasKeyVals(t, a, store.ordered())
		if t.Failed() {
			t.Logf("just deleted key %v", hexPath(kv.key))
			t.Logf("tree before delete\n%v", before)
			t.FailNow() // no point to keep going
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

func kvList(l []keyVal) string {
	b := &strings.Builder{}
	for _, x := range l {
		b.WriteString(x.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func (kv keyVal) String() string {
	return fmt.Sprintf("[k:%s v:%v]", hexPath(kv.key), kv.val)
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
	t.Helper()
	// verifies that the tree matches the supplied set of kv's by using the Walk fn
	i := 0
	a.Walk(func(k []byte, v interface{}) WalkState {
		if i >= len(exp) {
			t.Errorf("Got more callbacks than expected, additional k/v is %v / %v", k, v)
		} else {
			if !bytes.Equal(exp[i].key, k) {
				t.Errorf("key %d was %v but expecting %v", i, k, exp[i].key)
			}
			if v != exp[i].val {
				t.Errorf("key %v expecting value %v but was %v", exp[i].key, exp[i].val, v)
			}
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
			t.Errorf("key %v should have a value, but Get() says it doesn't", kv.key)
		}
		if actual != kv.val {
			t.Errorf("value %v for key %v is not the expected value of %v", actual, kv.key, kv.val)
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

func (s *kvStore) delete(k []byte) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(k, s.kvs[i].key) {
			s.kvs[i] = s.kvs[len(s.kvs)-1]
			s.kvs = s.kvs[:len(s.kvs)-1]
			return
		}
	}
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

func pretty(a *Tree) string {
	tree := &strings.Builder{}
	a.PrettyPrint(tree)
	return tree.String()
}

func hexPath(p []byte) string {
	w := &strings.Builder{}
	writePath(p, w)
	return w.String()
}
