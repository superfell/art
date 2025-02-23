package art

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

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
	testArt(t, []keyVal[string]{}, &Stats{})
}

func Test_OverwriteWithSameKey(t *testing.T) {
	testArt(t, []keyVal[string]{
		kvs("one", "one"),
		kvs("two", "two"),
		kvs("one", "three"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_InsertOnLeaf(t *testing.T) {
	testArt(t, []keyVal[string]{
		kvs("123", "abc"),
		// now insert something that would add a child to the leaf above
		kvs("1234", "abcd"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_LeafPathToNToLeafPath(t *testing.T) {
	testArt(t, []keyVal[string]{
		kvs("123", "1"),
		kvs("12345678", "2"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_SimpleMultipleInserts(t *testing.T) {
	testArt(t, []keyVal[string]{
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
		{3, &Stats{Node4s: 1, Keys: 3}},
		{4, &Stats{Node4s: 1, Keys: 4}},
		{5, &Stats{Node16s: 1, Keys: 5}},
		{14, &Stats{Node16s: 1, Keys: 14}},
		{15, &Stats{Node16s: 1, Keys: 15}},
		{16, &Stats{Node16s: 1, Keys: 16}},
		{17, &Stats{Node48s: 1, Keys: 17}},
		{40, &Stats{Node48s: 1, Keys: 40}},
		{48, &Stats{Node48s: 1, Keys: 48}},
		{49, &Stats{Node256s: 1, Keys: 49}},
		{200, &Stats{Node256s: 1, Keys: 200}},
		{256, &Stats{Node256s: 1, Keys: 256}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("children %d", tc.children), func(t *testing.T) {
			inserts := []keyVal[int]{}
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
			inserts := []keyVal[string]{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, byte(i)}, strconv.Itoa(i)))
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
			inserts := []keyVal[string]{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, byte(i)}, strconv.Itoa(i)))
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
			inserts := []keyVal[int]{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, 2, 3, 4, 5, 6, 7, byte(i + 10)}, i))
			}
			inserts = append(inserts, kv([]byte{1, 2, 3}, 123))
			testArt(t, inserts, tc.stats)
		})
	}
}

func Test_CompressedPathLargerThan24(t *testing.T) {
	cases := []simpleCase{
		{2, &Stats{Node4s: 5, Keys: 4}},
		{12, &Stats{Node4s: 4, Node16s: 1, Keys: 14}},
		{40, &Stats{Node4s: 4, Node48s: 1, Keys: 42}},
		{200, &Stats{Node4s: 4, Node256s: 1, Keys: 202}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("c_%d", tc.children), func(t *testing.T) {
			inserts := []keyVal[string]{}
			for i := 0; i < tc.children; i++ {
				inserts = append(inserts, kv([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 24, 25, 26, 27, 28, 29, 30, byte(i + 10)}, strconv.Itoa(i)))
			}
			inserts = append(inserts, kv([]byte{1, 2, 3}, "123"))
			inserts = append(inserts, kv([]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}, "234..."))
			testArt(t, inserts, tc.stats)
		})
	}
}

func Test_GrowWithPrefixValue(t *testing.T) {
	keyVals := []keyVal[int]{
		kvs("BBB", 1010),
		kvs("B", 505),
		kvs("BBx", 5555),
	}
	for i := 0; i < 256; i++ {
		keyVals = append(keyVals, kv([]byte{'B', byte(i)}, i))
	}
	testArt(t, keyVals, &Stats{Node256s: 1, Node4s: 1, Keys: 259})
}

func Test_KeyWithZeros(t *testing.T) {
	// any arbitrary byte array should be a valid key, even those with embedded nulls.
	testArt(t, []keyVal[string]{
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
		testArt(t, []keyVal[string]{
			kv(nil, "k1"),
			kv([]byte{0}, "k2"),
		}, nil)
	})
	t.Run("empty key", func(t *testing.T) {
		testArt(t, []keyVal[string]{
			kv([]byte{}, "k1"),
			kv([]byte{0}, "k2"),
		}, nil)
	})
}

func Test_NilValue(t *testing.T) {
	three := "3"
	testArt(t, []keyVal[*string]{
		kv[*string]([]byte{0, 0, 0}, nil),
		kv([]byte{0, 0, 0, 1}, &three),
		kv[*string]([]byte{10}, nil),
	}, nil)
}

func Test_NodeCompression(t *testing.T) {
	testArt(t, []keyVal[string]{
		kvs("1234567", "1"),
		kvs("1239000", "2"),
	}, &Stats{Node4s: 1, Keys: 2})
}

func Test_LeafLazyExpansion(t *testing.T) {
	testArt(t, []keyVal[string]{
		kvs("aaa", "foo"),
		kvs("aaattt", "bar"),
		kvs("aaatttxxx", "baz"),
	}, &Stats{Node4s: 2, Keys: 3})
}

func Test_Walk(t *testing.T) {
	testArt(t, []keyVal[string]{
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
			a := new(Tree[int])
			baseK := []byte{'A'}
			for i := 0; i < sz; i++ {
				a.Put(append(baseK, byte(i)), i)
			}
			t.Run("Full Walk", func(t *testing.T) {
				i := 0
				a.Walk(func(k []byte, v int) WalkState {
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
				a.Walk(func(k []byte, v int) WalkState {
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
				a.Walk(func(k []byte, v int) WalkState {
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
				a.Walk(func(k []byte, v int) WalkState {
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

type keyRange struct {
	start []byte
	end   []byte
}

func (r *keyRange) String() string {
	b := strings.Builder{}
	if len(r.start) > 0 {
		fmt.Fprintf(&b, "0x%0X", r.start)
	}
	b.WriteByte('-')
	if len(r.end) > 0 {
		fmt.Fprintf(&b, "0x%0X", r.end)
	}
	return b.String()
}

func Test_WalkRangeCompressedPath(t *testing.T) {
	a := new(Tree[string])
	s := kvStore[string]{}
	keyVals := []keyVal[string]{
		kv([]byte{2, 3, 4}, "1"),
		kv([]byte{2, 3, 4, 5, 6, 7, 8}, "2"),
		kv([]byte{2, 3, 4, 5, 6, 7, 9}, "3"),
	}
	for _, kv := range keyVals {
		a.Put(kv.key, kv.val)
		s.put(kv)
	}
	testWalkRange(t, a, &s, nil, nil)
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 5}, nil)
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 6}, nil)
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 7}, nil)
	testWalkRange(t, a, &s, []byte{2}, []byte{3})
	testWalkRange(t, a, &s, []byte{2, 3, 4}, []byte{2, 3, 4, 5, 6, 7, 9})
	testWalkRange(t, a, &s, []byte{2, 3, 4}, []byte{2, 3, 4, 5, 6, 7, 10})
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 5}, []byte{2, 3, 4, 5, 6})
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 6}, []byte{2, 3, 4, 5, 7})
	testWalkRange(t, a, &s, []byte{2, 3, 4, 5, 7}, []byte{2, 3, 4, 5, 6, 7, 9, 1, 2})
	testWalkRange(t, a, &s, rndKey(), rndKey())
}

func Test_WalkRange(t *testing.T) {
	a := new(Tree[int])
	s := kvStore[int]{}
	for i := 1; i < 5; i++ {
		for j := 1; j < 5; j++ {
			e := kv([]byte{byte(i * 2), byte(1 + j*2), byte(2 + j*3)}, i*j*j)
			a.Put(e.key, e.val)
			s.put(e)
		}
	}
	cases := []keyRange{
		{[]byte{6}, []byte{8, 5, 8}},
		{[]byte{5}, []byte{8, 5, 8}},
		{[]byte{6}, []byte{8, 5, 9}},
		{[]byte{4}, []byte{5}},
		{[]byte{4}, []byte{6}},
		{[]byte{3}, []byte{6}},
		{nil, []byte{6}},
		{[]byte{3}, nil},
		{[]byte{4, 3, 5, 1}, []byte{6, 3, 5, 1}},
	}
	for _, tc := range cases {
		testWalkRange(t, a, &s, tc.start, tc.end)
	}
}

func testArt[V comparable](t *testing.T, inserts []keyVal[V], expectedStats *Stats) {
	deleters := []func([]keyVal[V]) []keyVal[V]{randDeleteOrder[V], deleteLongestFirst[V], deleteShortestFirst[V]}
	names := []string{"random", "longest to shortest", "shortest to longest"}
	for i := 0; i < len(deleters); i++ {
		t.Run(fmt.Sprintf("normal/deletes %s", names[i]), func(t *testing.T) {
			testArtOne(t, inserts, deleters[i], expectedStats)
		})
		t.Run(fmt.Sprintf("reverse insertion order/deletes %s", names[i]), func(t *testing.T) {
			testArtOne(t, reverse(inserts), deleters[i], expectedStats)
		})
		t.Run(fmt.Sprintf("write twice/deletes %s", names[i]), func(t *testing.T) {
			testArtOne(t, append(inserts, inserts...), deleters[i], expectedStats)
		})
		t.Run(fmt.Sprintf("write twice in reverse/deletes %s", names[i]), func(t *testing.T) {
			testArtOne(t, reverse(append(inserts, inserts...)), deleters[i], expectedStats)
		})
	}
}

func testArtOne[V comparable](t *testing.T, inserts []keyVal[V], deleteOrderer func([]keyVal[V]) []keyVal[V], expectedStats *Stats) {
	a := new(Tree[V])
	defer func() {
		if t.Failed() {
			t.Logf("tree\n%v", pretty(a))
		}
	}()

	store := kvStore[V]{}
	for i := 0; i < len(inserts); i++ {
		a.Put(inserts[i].key, inserts[i].val)
		store.put(inserts[i])
		hasKeyVals(t, a, store.ordered())
		if t.Failed() {
			t.Logf("inserted %d keys, last inserted key %v", i+1, inserts[i].key)
			t.FailNow() // no point to keep going
		}
	}
	orderd := store.ordered()
	hasKeyVals(t, a, store.ordered())
	testWalkRange(t, a, &store, nil, nil)
	if len(orderd) > 0 {
		testWalkRange(t, a, &store, orderd[rnd.Intn(len(orderd))].key, nil)
		testWalkRange(t, a, &store, nil, orderd[rnd.Intn(len(orderd))].key)
		rStart := orderd[rnd.Intn(len(orderd))].key
		rEnd := orderd[rnd.Intn(len(orderd))].key
		if bytes.Compare(rStart, rEnd) > 0 {
			rStart, rEnd = rEnd, rStart
		}
		testWalkRange(t, a, &store, rStart, rEnd)
		testWalkRange(t, a, &store, rStart[:len(rStart)/2], rEnd[:len(rEnd)/2])
		testWalkRange(t, a, &store, addBytes(rStart, 0x05), addBytes(rEnd, 0x10))
	}

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

	deletes := deleteOrderer(inserts)

	for _, kv := range deletes {
		before := pretty(a)
		//t.Logf("About to delete key %s", hexPath(kv.key))
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

func testWalkRange[V comparable](t *testing.T, a *Tree[V], s *kvStore[V], start, end []byte) {
	kr := keyRange{start, end}
	t.Run(kr.String(), func(t *testing.T) {
		exp := s.orderedRange(start, end)
		idx := 0
		a.WalkRange(start, end, func(k []byte, v V) WalkState {
			if len(exp) == 0 {
				t.Errorf("received more keys than expecting, additional key/val is %v : %v", k, v)
			} else {
				bc := bytes.Compare(k, exp[0].key)
				if bc != 0 {
					t.Errorf("key %d expecting %v but got %v", idx, exp[0].key, k)
				} else if v != exp[0].val {
					t.Errorf("key %v expecting value %v but got %v", k, exp[0].val, v)
				}
				if bc >= 0 {
					exp = exp[1:]
				}
			}
			idx++
			return Continue
		})
		if len(exp) != 0 {
			t.Errorf("received %d less keys than expected, missing kvs are\n%v", len(exp), kvList(exp))
		}
		if t.Failed() {
			t.Logf("Tree is \n%v", pretty(a))
		}
	})
}

func addBytes(v []byte, add byte) []byte {
	res := append([]byte(nil), v...)
	idx := len(res) - 1
	for idx >= 0 {
		l := res[idx]
		n := l + add
		res[idx] = n
		if n > l {
			return res
		}
	}
	return append([]byte{1}, res...)
}

func randDeleteOrder[V any](i []keyVal[V]) []keyVal[V] {
	deletes := append([]keyVal[V]{}, i...)
	rnd.Shuffle(len(deletes), func(i, j int) {
		deletes[i], deletes[j] = deletes[j], deletes[i]
	})
	return deletes
}

func deleteLongestFirst[V any](i []keyVal[V]) []keyVal[V] {
	deletes := append([]keyVal[V]{}, i...)
	sort.Slice(deletes, func(i, j int) bool {
		return len(deletes[j].key) < len(deletes[i].key)
	})
	return deletes
}

func deleteShortestFirst[V any](i []keyVal[V]) []keyVal[V] {
	deletes := append([]keyVal[V]{}, i...)
	sort.Slice(deletes, func(i, j int) bool {
		return len(deletes[i].key) < len(deletes[j].key)
	})
	return deletes
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

type keyVal[V any] struct {
	key []byte
	val V
}

func kvList[V any](l []keyVal[V]) string {
	b := &strings.Builder{}
	for _, x := range l {
		b.WriteString(x.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func (kv keyVal[V]) String() string {
	return fmt.Sprintf("[k:%s v:%v]", hexPath(kv.key), kv.val)
}

func kv[V any](k []byte, v V) keyVal[V] {
	return keyVal[V]{key: k, val: v}
}
func kvs[V any](k string, v V) keyVal[V] {
	return keyVal[V]{key: []byte(k), val: v}
}

func reverse[V any](kv []keyVal[V]) []keyVal[V] {
	c := make([]keyVal[V], len(kv))
	j := len(kv) - 1
	for i := 0; i < len(kv); i++ {
		c[j] = kv[i]
		j--
	}
	return c
}

func hasKeyVals[V comparable](t *testing.T, a *Tree[V], exp []keyVal[V]) {
	t.Helper()
	// verifies that the tree matches the supplied set of kv's by using the Walk fn
	i := 0
	a.Walk(func(k []byte, v V) WalkState {
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
type kvStore[V any] struct {
	kvs []keyVal[V]
}

func (s *kvStore[V]) put(kv keyVal[V]) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(kv.key, s.kvs[i].key) {
			s.kvs[i].val = kv.val
			return
		}
	}
	s.kvs = append(s.kvs, kv)
}

func (s *kvStore[V]) delete(k []byte) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(k, s.kvs[i].key) {
			s.kvs[i] = s.kvs[len(s.kvs)-1]
			s.kvs = s.kvs[:len(s.kvs)-1]
			return
		}
	}
}

func (s *kvStore[V]) get(k []byte) (val V, exists bool) {
	for i := 0; i < len(s.kvs); i++ {
		if bytes.Equal(k, s.kvs[i].key) {
			return s.kvs[i].val, true
		}
	}
	var zero V
	return zero, false
}

// ordered returns the contents of the store in key order
func (s *kvStore[V]) ordered() []keyVal[V] {
	sort.Slice(s.kvs, func(i, j int) bool {
		return bytes.Compare(s.kvs[i].key, s.kvs[j].key) == -1
	})
	return s.kvs
}

// orderedRange returns the keyVals that are between the supplied start,end
// values, using the same semantics as WalkRange.
func (s *kvStore[V]) orderedRange(start, end []byte) []keyVal[V] {
	sorted := s.ordered()
	rng := make([]keyVal[V], 0, 10)
	for _, kv := range sorted {
		if (len(start) == 0 || bytes.Compare(kv.key, start) >= 0) && (len(end) == 0 || bytes.Compare(kv.key, end) == -1) {
			rng = append(rng, kv)
		}
	}
	return rng
}

func pretty[V any](a *Tree[V]) string {
	tree := &strings.Builder{}
	a.PrettyPrint(tree)
	return tree.String()
}

func hexPath(p []byte) string {
	w := &strings.Builder{}
	writePath(p, w)
	return w.String()
}
