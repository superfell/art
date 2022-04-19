package main

import (
	"fmt"
	"os"

	"github.com/superfell/art"
)

func main() {
	a := new(art.Tree[string])
	k := []byte{1, 2, 4, 0, 1}
	k2 := []byte{1, 2, 4, 0, 2}
	a.Put(k, "bob")
	v, exists := a.Get(k)
	fmt.Printf("key %v exists %t with value %v\n", k, exists, v)
	a.Put(k2, "eve")
	a.Walk(func(k []byte, v string) art.WalkState {
		fmt.Printf("%v : %v\n", k, v)
		return art.Continue
	})
	a.PrettyPrint(os.Stdout)
}
