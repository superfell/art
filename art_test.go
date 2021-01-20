package art

import (
	"fmt"
	"testing"

	"github.com/bruth/assert"
	"github.com/kr/pretty"
)

func Test_Empty(t *testing.T) {
	a := new(Art)
	notExists(t, a, "")
	notExists(t, a, "bob")
}

func Test_InsertGet(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), []byte("abc"))
	pretty.Print(a)
	contains(t, a, "123", "abc")
	notExists(t, a, "2")
	notExists(t, a, "12")
	notExists(t, a, "1234")
	notExists(t, a, "")
}

func Test_InsertOnLeaf(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), []byte("abc"))
	// now insert something that would add a child to the leaf above
	a.Insert([]byte("1234"), []byte("abcd"))
	pretty.Print(a)
	contains(t, a, "123", "abc")
	contains(t, a, "1234", "abcd")
}

func Test_MultipleInserts(t *testing.T) {
	a := new(Art)
	a.Insert([]byte("123"), []byte("abc"))
	a.Insert([]byte("456"), []byte("abcd"))
	a.Insert([]byte("1211"), []byte("abcde"))
	contains(t, a, "123", "abc")
	contains(t, a, "456", "abcd")
	contains(t, a, "1211", "abcde")
	notExists(t, a, "")
	notExists(t, a, "12")
	notExists(t, a, "1234")
	notExists(t, a, "5")
	notExists(t, a, "451")
	notExists(t, a, "4561")
	notExists(t, a, "1212")
	notExists(t, a, "121")
	fmt.Printf("%+v", *a)
}

func Test_Grow4to16(t *testing.T) {
	a := new(Art)
	k := []byte{65, 66}
	for i := byte(0); i < 10; i++ {
		a.Insert(append(k, i), []byte{i})
		for j := byte(0); j <= i; j++ {
			v, exists := a.Get(append(k, j))
			assert.True(t, exists, "expecting to find value for key", fmt.Sprintf("%+v", append(k, j)))
			assert.Equal(t, []byte{j}, v)
		}
	}
	a.Insert(append(k, 5, 10), []byte{100})
	v, exists := a.Get(append(k, 5, 10))
	assert.True(t, exists, "expecting to find value for key", fmt.Sprintf("%+v", append(k, 5, 10)))
	assert.Equal(t, []byte{100}, v)
}

func Test_GrowTo48(t *testing.T) {
	a := new(Art)
	k := []byte("12")
	a.Insert(k, k)
	contains(t, a, "12", "12")
	for i := byte(0); i < 30; i++ {
		a.Insert(append(k, i+'A'), append(k, i+'B'))
		for j := byte(0); j <= i; j++ {
			nk := append(k, j+'A')
			v, exists := a.Get(nk)
			assert.True(t, exists, "expecting to find value for key", nk)
			assert.Equal(t, append(k, j+'B'), v)
			contains(t, a, "12", "12")
		}
	}
	notExists(t, a, "12z")
	contains(t, a, "12A", "12B")
}

func notExists(t *testing.T, a *Art, k string) {
	t.Helper()
	act, exists := a.Get([]byte(k))
	assert.False(t, exists, "key shouldn't have value", k)
	assert.Nil(t, act)
}

func contains(t *testing.T, a *Art, k string, exp string) {
	t.Helper()
	act, exists := a.Get([]byte(k))
	assert.True(t, exists, "should contain key", k)
	assert.Equal(t, []byte(exp), act)
}
