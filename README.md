# seqs — Generic Combinators for `iter.Seq` (Go 1.23+)

The missing sibling of `slices` and `maps`:

```
slices : []T            ← operations on slices
maps   : map[K]V        ← operations on maps
seqs   : iter.Seq[T]    ← operations on sequences  (this package)
```

The standard library ships two of the three iterator layers:

1. **Type layer** — `iter`: `Seq[T]`, `Seq2[K,V]`, `Pull`/`Pull2`.
2. **Adapter layer** — `slices` / `maps`: bridge a concrete container ↔ `iter.Seq`
   (`slices.Values`, `slices.Collect`, `slices.Sorted`, `maps.Keys`, `maps.Values`, `maps.All`).
3. **Combinator layer** — *the transforms* (`Map`, `Filter`, `Limit`, `Merge`…).

Layer 3 was deliberately left out of std (the [`xiter` proposal](https://go.dev/issue/61898)
is still on hold). **`seqs` is that layer.** So it stays in its lane: get *onto* the bus with
`slices.Values` / `maps.Values`, move *along* it with `seqs`, get *off* with `slices.Collect` /
`slices.Sorted`. `seqs` never reimplements the adapters.

## Installation

```bash
go get github.com/tyarnell/seqs
```

## Quick Start

```go
import (
    "slices"
    "github.com/tyarnell/seqs"
)

// slices.Values → onto the bus | seqs.Pipe → along it | slices.Collect → off
result := slices.Collect(seqs.Pipe(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
    seqs.Filter(func(n int) bool { return n%2 == 0 }),
    seqs.Map(func(n int) int { return n * n }),
    seqs.Limit[int](3),
))
// result == [4 16 36]
```

## Core Concept

`Transform[T]` is just a type alias:

```go
type Transform[T any] func(iter.Seq[T]) iter.Seq[T]
```

This gives you:

- **Lazy evaluation** — nothing runs until you range over the result
- **Composability** — transforms compose with `Pipe` or plain function calls
- **Standard library alignment** — names and semantics match `slices`/`maps`/`iter`; you bring your own `slices.Collect`, `iter.Pull`, etc.

## Combinators

### Core

| Function | Description |
|----------|-------------|
| `Pipe` | Chain transforms left-to-right |
| `Map` | Transform each element |
| `Filter` | Keep elements matching a predicate |
| `FlatMap` | One-to-many expansion |

### Limiting & Stopping

| Function | Description |
|----------|-------------|
| `Limit(n)` | Yield first n elements (xiter spelling; was `Take`) |
| `Drop(n)` | Skip first n elements |
| `TakeWhile(pred)` | Yield while predicate is true |
| `TakeUntil(pred)` | Yield until predicate matches (inclusive) |
| `DropWhile(pred)` | Skip while predicate is true |
| `WithContext(ctx)` | Stop when context is cancelled |

### Combining & Grouping

| Function | Description |
|----------|-------------|
| `Concat` | Chain sequences end-to-end |
| `Merge` | Interleave two sequences sorted by natural order |
| `MergeFunc(cmp)` | Interleave two sequences sorted by a `cmp.Compare`-style func |
| `Dedupe(eq)` | Remove consecutive duplicates |
| `Batch(n)` | Group into fixed-size slices |
| `Window(n)` | Sliding windows of size n |

### Side Effects

| Function | Description |
|----------|-------------|
| `Tee(fn)` | Call fn for each element, pass all through |
| `Siphon(pred, fn)` | Remove matching elements, send to fn |

### Producers & Consumers

For the container ↔ sequence boundary, **use the standard library** — `seqs` does not
duplicate it:

| Need | Use |
|------|-----|
| slice/map → seq | `slices.Values`, `maps.Keys`, `maps.Values`, `maps.All`, `slices.All` |
| seq → slice/map | `slices.Collect`, `slices.Sorted`, `maps.Collect` |

`seqs` adds only the producers/consumers with **no** std equivalent: `Empty[T]()`,
`Repeat(val, n)`; and terminal ops `ForEach`, `Reduce`, `Count`, `First`, `Any`, `All`.

### Seq2 (`iter.Seq2[K, V]`)

Following the std `2`-suffix convention, every Seq combinator with a key/value analogue
has a `2` twin, plus bridges between the two worlds:

| Function | Description |
|----------|-------------|
| `Pipe2` / `Map2` / `Filter2` | Seq2 versions of `Pipe`/`Map`/`Filter` |
| `Limit2` / `Drop2` / `Tee2` / `Concat2` / `WithContext2` | …the rest of the parallels |
| `Enumerate(seq)` | `Seq[T]` → `Seq2[int, T]` (add indices) |
| `Zip(a, b)` | `Seq[A] + Seq[B]` → `Seq2[A, B]` (positional pairing) |
| `Keys(seq2)` / `Values(seq2)` | `Seq2[K,V]` → `Seq[K]` / `Seq[V]` (projections) |
| `Swap(seq2)` | `Seq2[K,V]` → `Seq2[V,K]` |

```go
// maps.All → Seq2 | stay in Seq2 to keep pairs | Keys → drop to Seq | slices.Sorted → off
passing := seqs.Pipe2(maps.All(scores),
    seqs.Filter2(func(_ string, s int) bool { return s >= 5 }),
)
names := slices.Sorted(seqs.Keys(passing))
```

Rule of thumb: **one `Pipe` per element type.** Stay in `Seq2` while you must keep keys and
values paired; drop to `Seq` with `Keys`/`Values` as soon as a transform needs only one side.

## Cross-Type Transforms

Transform between types:

```go
// int -> string
texts := seqs.MapTo(func(n int) string {
    return fmt.Sprintf("%d", n)
})(numbers)

// string -> []rune (flattened)
runes := seqs.FlatMapTo(func(s string) iter.Seq[rune] {
    return slices.Values([]rune(s))
})(strings)

// Filter and transform in one pass
valid := seqs.FilterMapTo(func(s string) (int, bool) {
    n, err := strconv.Atoi(s)
    return n, err == nil
})(strings)
```

## Generic Interface Pattern

For types that can extract a key:

```go
type User struct { ID string; Name string }
func (u User) Key() string { return u.ID }

// Deduplicate by ID
unique := seqs.DedupeByKey[User, string](users)

// Group by ID
groups := seqs.GroupBy[User, string](users)
```

Or use function-based versions when you don't control the type:

```go
unique := seqs.DedupeByFunc(func(u User) string { return u.ID })(users)
groups := seqs.GroupByFunc(func(u User) string { return u.ID }, users)
```

## Stopping Strategies

Three ways to control when iteration stops:

```go
// TakeWhile: stop BEFORE first failure
result := seqs.TakeWhile(func(n int) bool { return n < 10 })(input)

// TakeUntil: stop AFTER first match (inclusive)
result := seqs.TakeUntil(func(n int) bool { return n == sentinel })(input)

// WithContext: stop on cancellation
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
result := seqs.WithContext[int](ctx)(input)
```

## Merge (Pull Iterator)

`Merge`/`MergeFunc` use `iter.Pull` internally — the canonical case for pull iterators,
since merging needs to compare the *heads* of both sequences:

```go
merged := seqs.Merge(seq1, seq2)                       // natural order
merged := seqs.MergeFunc(seq1, seq2, cmp.Compare[int]) // custom comparator
```

## Examples

**Runnable demos** — one program per subfolder under `examples/`:

```bash
go run ./examples/basics   # Pipe/Filter/Map/Limit, Merge/MergeFunc, Reduce, a lazy infinite source
go run ./examples/etl      # ETL: parse → siphon errors → filter → dedupe → batch (Seq[T])
go run ./examples/seq2     # Seq2: Pipe2/Filter2/Map2/Tee2 + bridges (Enumerate, Zip, Keys, Values)
```

**Testable examples** — `example_test.go` has `Example*` functions (also rendered on pkg.go.dev) covering `Pipe`, `Map`, `Filter`, `FlatMap`, `MapTo`, `TakeWhile`, `Batch`, `Merge`, `Siphon`, `DedupeByFunc`, `Reduce`, and a `Seq2` example (`maps.All` → `Pipe2`/`Filter2`/`Map2` → `Keys` → `slices.Sorted`). They run as part of the test suite:

```bash
go test ./...
```

## Design Principles

**Functions over interfaces** — `Transform[T]` is a type alias, not an interface.

**Lazy by default** — Nothing runs until you range over the result.

**Generic** — All combinators work on any `iter.Seq[T]` / `iter.Seq2[K,V]`.

**Stays in its lane** — `seqs` is the combinator layer only. It leans on `slices`/`maps`
for the container boundaries and never reimplements them. Names track the std libraries and
the `xiter` proposal (`Limit`, `Merge`/`MergeFunc`, the `2`-suffix twins), so if `xiter`
ever lands, migration is a rename — or a delete.

---

*Extracted as a standalone module from the `docproc` document-processing project, where these combinators power the parsing/transform pipeline.*
