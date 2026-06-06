# BYOC — bring your own container

`seqs` never asks for a slice or a map. Every combinator is defined over
`iter.Seq[T]` (or `iter.Seq2[K, V]`), so **any type that can produce a sequence
works** — your own collections included. There's nothing to register and no
interface to implement.

## The minimum: one producer

If a value can be ranged, it can feed a pipeline. The smallest case is a single
function that yields:

```go
func (t *Tree[T]) InOrder() iter.Seq[T] {
    return func(yield func(T) bool) {
        // ... walk the tree, yield each value ...
    }
}

clean := slices.Collect(seqs.Pipe(tree.InOrder(),
    seqs.Filter(keep),
    seqs.Map(normalize),
))
```

That's it — `seqs` does the middle, your type owns the production.

## The idiom: speak the adapter vocabulary

`slices` and `maps` are the standard library's *adapter layer* (see
[concepts.md](concepts.md)): they convert a concrete container to and from
`iter.Seq` using a small, consistent vocabulary. Give your own container the
same names and it becomes a first-class citizen alongside them:

| Direction | Standard name | What it returns |
|-----------|---------------|-----------------|
| container → seq | `All()` | `iter.Seq2[K, V]` — the *primary*, position/key-preserving iterator |
| container → seq | `Values()` | `iter.Seq[V]` — value-only projection |
| container → seq | `Keys()` | `iter.Seq[K]` — key-only projection (for keyed containers) |
| seq → container | `CollectXxx(seq)` | a new container (a package function, since you can't add methods to construct) |

Implement those and the whole ecosystem lines up: your `Values()` gets onto the
bus, `seqs` transforms along it, and `slices.Collect` / `slices.Sorted` /
`maps.Collect` / your own `CollectXxx` get off.

```go
out := CollectRing(seqs.Pipe(ring.Values(), cleanup, classify), 64)
```

### Worked example

[`examples/byoc`](../examples/byoc) implements a fixed-capacity ring buffer with
exactly this shape:

```go
func (r *Ring[T]) All() iter.Seq2[int, T]      // (age, value), oldest → newest
func (r *Ring[T]) Values() iter.Seq[T]         // = seqs.Values(r.All())
func CollectRing[T any](seq iter.Seq[T], capacity int) *Ring[T]
```

Run it:

```bash
go run ./examples/byoc
```

Notice how little is bespoke: once `All()`/`Values()` exist, the bridges
(`seqs.Values`, `seqs.Keys`, `seqs.Enumerate`) and every transform work for free
— `Ring` never grew its own `Map`/`Filter`.

## Why not put `Map`/`Filter` on the container?

Tempting, but it's the wrong layer. A method-per-container approach means every
collection re-implements the same transforms, and pipelines can't span types
(your `Ring.Map` can't feed a `Tree`). Keeping transforms in `seqs` and only
adapters on the container is exactly how `slices`/`maps` are factored — one set
of combinators, many containers, all meeting at `iter.Seq`.
