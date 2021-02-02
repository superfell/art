package art

import (
	"bytes"
	"testing"
)

func Test_KeyPathAssign(t *testing.T) {
	p := keyPath{}
	k := []byte{4, 5, 7, 8, 1, 10}
	p.assign(k)
	if !bytes.Equal(k, p.asSlice()) {
		t.Errorf("Assigned path %v, but got different path %v back", k, p)
	}
}

func Test_KeyPathTrim(t *testing.T) {
	p := keyPath{}
	k := []byte{1, 2, 3, 4, 5}
	p.assign(k)
	p.trimPathStart(2)
	exp := []byte{3, 4, 5}
	if !bytes.Equal(exp, p.asSlice()) {
		t.Errorf("Trimmed path expected to be %v but was %b", exp, p)
	}
}

func Test_KeyPathCanExtendBy(t *testing.T) {
	p := keyPath{}
	k := []byte{1, 2, 3}
	p.assign(k)
	if !p.canExtendBy(20) {
		t.Errorf("a 3 byte path should be extendable by another 20")
	}
	if p.canExtendBy(21) {
		t.Errorf("a 3 byte path shouldn't be extendable by another 21")
	}
}

func Test_KeyPathString(t *testing.T) {
	p := keyPath{}
	k := []byte{1, 0x42, 0, 0xFF}
	p.assign(k)
	exp := " 0x01 0x42 0x00 0xFF"
	if exp != p.String() {
		t.Errorf("Expecting '%s' for String() but got '%s'", exp, p.String())
	}
}

type prependCase struct {
	base     string
	prefix   string
	extra    string
	expected string
}

func Test_KeyPathPrepend(t *testing.T) {
	cases := []prependCase{
		{"qqq", "ppp", "x", "pppxqqq"},
		{"qqq", "ppp", "", "pppqqq"},
		{"qqq", "", "", "qqq"},
		{"", "", "", ""},
		{"", "", "x", "x"},
		{"", "xx", "y", "xxy"},
	}
	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			p := keyPath{}
			p.assign([]byte(c.base))
			p.prependPath([]byte(c.prefix), []byte(c.extra)...)
			if !bytes.Equal([]byte(c.expected), p.asSlice()) {
				t.Errorf("Expecting final path to be %s but was %s", c.expected, string(p.asSlice()))
			}
		})
	}
}
