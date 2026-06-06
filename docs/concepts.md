# Concepts — where `seqs` fits

`seqs` is the missing sibling of the standard library's container packages:

```
slices : []T            ← operations on slices
maps   : map[K]V        ← operations on maps
seqs   : iter.Seq[T]    ← operations on sequences  (this package)
```

## The three iterator layers

Go's iterator story splits into three layers. The standard library ships the
first two; `seqs` is the third.

1. **Type layer — `iter`.** The lingua franca: `Seq[V] = func(yield func(V) bool)`,
   `Seq2[K, V]`, and `Pull`/`Pull2`. No operations, just the contract.

2. **Adapter layer — `slices`, `maps`, `strings`, …** Bridges a *concrete
   container* to and from `iter.Seq`:
   - out of a container: `slices.Values`, `slices.All`, `maps.Keys`,
     `maps.Values`, `maps.All`, `strings.Lines`, `strings.SplitSeq`
   - back into one: `slices.Collect`, `slices.Sorted`, `maps.Collect`,
     `slices.AppendSeq`

3. **Combinator layer — the transforms.** `Map`, `Filter`, `FlatMap`, `Limit`,
   `Merge`, `Batch`, … — operations that go `Seq → Seq` and *don't care what
   container the data came from*.

Layer 3 was **deliberately** left out of the standard library: the
[`xiter` proposal](https://go.dev/issue/61898) that would add `Map`/`Filter`/
`Concat`/`Merge`/etc. is still on hold while the Go team gathers experience
with `iter`. **`seqs` is a local implementation of that layer.**

## Get on the bus, move along it, get off

Because `seqs` owns only the middle of a pipeline, the three layers collaborate
and nobody overlaps:

```go
import (
    "maps"
    "slices"
    "github.com/tyarnell/seqs"
)

m := map[string]int{"a": 3, "b": 8, "c": 1, "d": 9}

top := slices.Sorted(                          // slices → OFF the bus
    seqs.Pipe(maps.Values(m),                  // maps   → ONTO the bus
        seqs.Filter(func(n int) bool { return n > 2 }),
        seqs.Map(func(n int) int { return n * 10 }),
    ),                                         // seqs   → ALONG the bus
)
// top == [30 80 90]
```

Your own container types plug into the very same picture — see
[byoc.md](byoc.md).

## Design principles

- **Functions over interfaces.** `Transform[T]` is a type alias
  (`func(iter.Seq[T]) iter.Seq[T]`), not an interface. Pipelines are plain
  function composition; there is no `Pipeline` type, `Step` interface, or
  registry.

- **Lazy by default.** Nothing runs until you range over the result. An
  infinite source is fine as long as something downstream stops it (`Limit`,
  `TakeWhile`, `WithContext`).

- **Generic.** Every combinator works on any `iter.Seq[T]` /
  `iter.Seq2[K, V]`.

- **Stays in its lane.** `seqs` is the combinator layer only. It leans on
  `slices`/`maps` for the container boundaries and never reimplements them. The
  core transforms don't even import `slices`/`maps`.

- **Aligned with std and `xiter`.** Names track the standard libraries and the
  proposal: `Limit` (not `Take`), `Merge`/`MergeFunc`, and the `2`-suffix twins
  (`Map2`, `Filter2`, `Concat2`, …). If `xiter` ever lands, migration is a
  rename — or a delete.

  Two intentional divergences: `Reduce` keeps the sequence as its first
  argument (consistent with `Count`/`First`/`Any`/`All`), and `Map` stays
  same-type with a separate `MapTo` for type changes (so `Map` fits
  `Transform[T]` and chains inside `Pipe`).
