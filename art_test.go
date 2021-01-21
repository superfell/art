package art

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_Empty(t *testing.T) {
	a := new(Art)
	noStringKey(t, a, "")
	noStringKey(t, a, "bob")
}

func Test_OverwriteWithSameKey(t *testing.T) {
	a := new(Art)
	k := []byte("QWE")
	a.Insert(k, "one")
	hasStringKey(t, a, "QWE", "one")
	a.Insert(k, "two")
	hasStringKey(t, a, "QWE", "two")
}

func Test_InsertGet(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), "abc")
	hasStringKey(t, a, "123", "abc")
	noStringKey(t, a, "2")
	noStringKey(t, a, "12")
	noStringKey(t, a, "1234")
	noStringKey(t, a, "")
}

func Test_InsertOnLeaf(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), "abc")
	// now insert something that would add a child to the leaf above
	a.Insert([]byte("1234"), "abcd")
	hasStringKey(t, a, "123", "abc")
	hasStringKey(t, a, "1234", "abcd")
}

func Test_MultipleInserts(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), "abc")
	a.Insert([]byte("456"), "abcd")
	a.Insert([]byte("1211"), "abcde")
	hasStringKey(t, a, "123", "abc")
	hasStringKey(t, a, "456", "abcd")
	hasStringKey(t, a, "1211", "abcde")
	noStringKey(t, a, "")
	noStringKey(t, a, "12")
	noStringKey(t, a, "1234")
	noStringKey(t, a, "5")
	noStringKey(t, a, "451")
	noStringKey(t, a, "4561")
	noStringKey(t, a, "1212")
	noStringKey(t, a, "121")
}

func Test_Grow4to16(t *testing.T) {
	a := new(Art)
	k := []byte{65, 66}
	for i := byte(0); i < 10; i++ {
		a.Insert(append(k, i), i)
		for j := byte(0); j <= i; j++ {
			hasByteKey(t, a, append(k, j), j)
		}
	}
	a.Insert(append(k, 5, 10), 100)
	hasByteKey(t, a, append(k, 5, 10), 100)
}

func Test_GrowTo48(t *testing.T) {
	a := new(Art)
	k := []byte("12")
	a.Insert(k, "12")
	hasStringKey(t, a, "12", "12")
	for i := byte(0); i < 30; i++ {
		a.Insert(append(k, i+'A'), fmt.Sprintf("val_%d", i))
		for j := byte(0); j <= i; j++ {
			nk := append(k, j+'A')
			hasByteKey(t, a, nk, fmt.Sprintf("val_%d", j))
			hasStringKey(t, a, "12", "12")
		}
	}
	noStringKey(t, a, "12z")
	noStringKey(t, a, "12Bzasd")
	noStringKey(t, a, "12Bz")
}

func Test_GrowTo256(t *testing.T) {
	a := new(Art)
	for i := 0; i < 256; i++ {
		a.Insert([]byte{byte(i)}, i)
	}
	for i := 0; i < 256; i++ {
		hasByteKey(t, a, []byte{byte(i)}, i)
	}
}

func Test_GrowWithPrefixValue(t *testing.T) {
	a := new(Art)
	k := []byte("B")
	a.Insert([]byte("BBB"), "kk")
	a.Insert(k, "k")
	for i := 0; i < 256; i++ {
		ck := append(k, byte(i))
		a.Insert(ck, i)
		hasByteKey(t, a, k, "k")
		hasStringKey(t, a, "BBB", "kk")
		for j := 0; j < i; j++ {
			hasByteKey(t, a, append(k, byte(j)), j)
		}
	}
}

func Test_KeyWithZeros(t *testing.T) {
	// any arbitrary byte array should be a valid key, even those with embedded nulls.
	a := new(Art)
	k1 := []byte{0, 0, 0}
	k2 := []byte{0, 0, 0, 0}
	k3 := []byte{0, 0, 0, 1}
	a.Insert(k1, "k1")
	a.Insert(k2, "k2")
	a.Insert(k3, "k3")
	hasByteKey(t, a, k1, "k1")
	hasByteKey(t, a, k2, "k2")
	hasByteKey(t, a, k3, "k3")
}

func Test_EmptyKey(t *testing.T) {
	// an empty byte array is also a valid key
	a := new(Art)
	a.Insert(nil, "n")
	hasByteKey(t, a, nil, "n")
	a.Insert([]byte{}, "b")
	hasByteKey(t, a, []byte{}, "b")
	a.Insert([]byte{0}, "0")
	hasByteKey(t, a, []byte{0}, "0")
	hasByteKey(t, a, []byte{}, "b")
}

func Test_Walk(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("C"), "c")
	a.Insert([]byte("A"), "a")
	a.Insert([]byte("AA"), "aa")
	a.Insert([]byte("B"), "b")
	expKeys := [][]byte{[]byte("A"), []byte("AA"), []byte("B"), []byte("C")}
	expVals := []string{"a", "aa", "b", "c"}
	idx := 0
	a.Walk(func(k []byte, v interface{}) WalkState {
		if !bytes.Equal(expKeys[idx], k) {
			t.Errorf("At iteration %d expecting key %v but got %v", idx, expKeys[idx], k)
		}
		if v != expVals[idx] {
			t.Errorf("At iteration %d expected value %v but got %v", idx, expVals[idx], v)
		}
		idx++
		return Continue
	})
	if idx != 4 {
		t.Errorf("Expected 4 callbacks during walk, but only got %d", idx)
	}
}

func Test_MoreWalk(t *testing.T) {
	sizes := []byte{2, 4, 5, 16, 17, 47, 48, 49, 50, 255}
	for _, sz := range sizes {
		t.Run(fmt.Sprintf("Walk size %d", sz), func(t *testing.T) {
			a := new(Art)
			baseK := []byte{'A'}
			for i := byte(0); i < sz; i++ {
				a.Insert(append(baseK, i), i)
			}
			t.Run("Full Walk", func(t *testing.T) {
				i := byte(0)
				a.Walk(func(k []byte, v interface{}) WalkState {
					exp := append(baseK, i)
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
				i := byte(0)
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
				for i := byte(0); i < sz; i++ {
					a.Insert(append(baseK, i, i), int(i)^2)
				}
				calls := 0
				prevKey := make([]byte, 0, 5)
				a.Walk(func(k []byte, v interface{}) WalkState {
					calls++
					if bytes.Compare(prevKey, k) != -1 {
						t.Errorf("Key %v received out of order, prevKey was %v", k, prevKey)
					}
					if len(k) == 2 && k[1] != v {
						t.Errorf("Unexpected value %v for key %v, was expecting %v", v, k, k[1])
					}
					if len(k) == 3 {
						expV := int(k[2]) ^ 2
						if expV != v {
							t.Errorf("Unexpected value %v for key %v, was expecting %v", v, k, expV)
						}
					}
					return Continue
				})
				if calls != int(sz)*2 {
					t.Errorf("Unexpected number of callbacks %d, expecting %d", calls, sz*2)
				}
			})
		})
	}
}

func noStringKey(t *testing.T, a *Art, k string) {
	t.Helper()
	noByteKey(t, a, []byte(k))
}

func noByteKey(t *testing.T, a *Art, k []byte) {
	t.Helper()
	act, exists := a.Get(k)
	if exists {
		t.Errorf("Unexpected value of %#v exists for key %v", act, k)
	}
	if act != nil {
		t.Errorf("Unexpected value of %#v exists for key %v", act, k)
	}
}

func hasStringKey(t *testing.T, a *Art, k string, exp interface{}) {
	t.Helper()
	hasByteKey(t, a, []byte(k), exp)
}

func hasByteKey(t *testing.T, a *Art, k []byte, exp interface{}) {
	t.Helper()
	act, exists := a.Get([]byte(k))
	if !exists {
		t.Errorf("Should contain value for key %#v but does not", k)
	}
	if act != exp {
		t.Errorf("Unexpected value of %#v for key %#v, expecting %#v", act, k, exp)
	}
}
