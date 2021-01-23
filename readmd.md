# Art

Art is a adaptive radix tree implementation in go. 

Art is described in detail in the [ART paper](https://db.in.tum.de/~leis/papers/ART.pdf).

This implementation is still a work in progress.

## Usage

The API might change as additional features are implemented.

```go
a := new(art.Art)
k := []byte{1, 2, 4, 0, 1}
k2 := []byte{1, 2, 4, 0, 2}
a.Insert(k, "bob")
v, exists := a.Get(k)
fmt.Printf("key %v exists %t with value %v\n", k, exists, v)
a.Insert(k2, "eve")
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
[n4]  [0x01 0x02 0x04 0x00]
  0x01: [leaf]  value:bob
  0x02: [leaf]  value:eve
```

## Implementation Notes

In order to try and pack as much into contiguous memory for each node, the nodes use fixed size arrays for keys & values rather than slices.

For simplicity the collapsed path is stored in a slice, but this could be improved.

node4 & node16 are currently identical other than the array sizes. node16 might benefit from the SIMD approach (if possible in go) or from
storing the keys sorted. It needs testing, but it not clear that sorting the keys in node4 is worth the effort.

Delete is not yet implemented.

Range scans (other than the entire tree) are not yet implemented.


## Differences vs paper

In addition to child nodes, a node can also contain a value leaf.
This stores the value associated with the key to that point in the path.
Node4/16/48 use one of the existing child slots for this when needed. Node256 has a dedicated field for it.
This allows for an arbitrary byte slice to be used as a key. 
The other common approach is to add a terminator to the key, but this then disallows the use of the terminator value inside the key.

Node16 doesn't use the SIMD approach for matching keys.
Node4/16 don't sort the keys, instead scan the contained keys linearly.

Collapsed inner nodes always use the pessimistic approach and store the collapsed key values in the node header.
