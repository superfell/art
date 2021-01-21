package art

import (
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
