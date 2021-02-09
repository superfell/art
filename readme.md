# Art

![Go](https://github.com/superfell/art/workflows/Go/badge.svg?branch=main) 
[![GoDoc](https://godoc.org/github.com/superfell/art?status.svg)](https://godoc.org/github.com/superfell/art)

Art is a adaptive radix tree implementation in go. 

Art is described in detail in the [ART paper](https://db.in.tum.de/~leis/papers/ART.pdf).

## Usage

```go
a := new(art.Tree)
k := []byte{1, 2, 4, 0, 1}
k2 := []byte{1, 2, 4, 0, 2}
a.Put(k, "bob")
v, exists := a.Get(k)
fmt.Printf("key %v exists %t with value %v\n", k, exists, v)
a.Put(k2, "eve")
a.Walk(func(k []byte, v interface{}) art.WalkState {
  fmt.Printf("%v : %v\n", k, v)
  return art.Continue
})
a.PrettyPrint(os.Stdout)
```

```
key [1 2 4 0 1] exists true with value bob
[1 2 4 0 1] : bob
[1 2 4 0 2] : eve
[n4 0x01 0x02 0x04 0x00] 
  0x01: [leaf] value:bob
  0x02: [leaf] value:eve
```

## Implementation Notes

In order to try and pack as much into contiguous memory for each node, the nodes use fixed size arrays for keys, values
and compressed key path rather than slices.

node4 stores the keys unordered and loops to find entries.

node16 stores the keys ordered, but uses a loop to find entries as that's faster for a small array. Having the keys sorted makes the
child iteration simpler and faster.
https://www.superfell.com/weblog/2021/01/it-depends-episode-1 and https://www.superfell.com/weblog/2021/01/it-depends-episode-2 have
a detailed deep dive into the best way to search the keys for node16.

During delete nodes will be shrunk once they contain less then 75% entries than the next smallest size can support. e.g. a node256 will
shrink to a node48 once the node256 only has 36 children. This should help reduce grow/shrink/grow around the node size boundaries.

The compressed path is stored in a fixed sized array that's part of the node header. If the compressed path is longer than will fit
in there intermediate node4s will be created. The array is 23 bytes, so this is unlikely to be an issue unless you have very long sparse
keys.

## Differences vs paper

In addition to child nodes, a node can also contain a value leaf.
This stores the value associated with the key to that point in the path.
Node4/16/48 use one of the existing child slots for this when needed. Node256 has a dedicated field for it.
This allows for an arbitrary byte slice to be used as a key. 
The other common approach is to add a terminator to the key, but this then disallows the use of the terminator value inside the key.

Node16 doesn't use the SIMD approach for matching keys.

Collapsed inner nodes always use the pessimistic approach and store the collapsed key values in the node header.
