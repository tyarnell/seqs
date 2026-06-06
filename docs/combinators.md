# Combinator reference

All combinators operate on `iter.Seq[T]` (or `iter.Seq2[K, V]`). For the
container ↔ sequence boundary, use the standard library — `seqs` does not
duplicate it (see [Producers & consumers](#producers--consumers)).

## Writing readable pipelines (recipe style)

Each combinator returns a value (a `Transform[T]`), so you can name your steps
once and let the `Pipe` read as a recipe — a list of step names instead of a
wall of inline closures:

```go
// Steps: define once.
var (
    keepEven = seqs.Filter(func(n int) bool { return n%2 == 0 })
    square   = seqs.Map(func(n int) int { return n * n })
    first3   = seqs.Limit[int](3)
)

// Recipe: read top to bottom.
out := slices.Collect(seqs.Pipe(nums, keepEven, square, first3))
```

The same idiom works for `Pipe2` with named `Transform2` steps. One rule keeps
it honest: **one `Pipe` per element type** — the type-crossing steps (`MapTo`,
`Batch`, `Window`) sit outside the `Pipe` as plain calls. The runnable
[`examples/`](../examples) all follow this shape.

## Seq[T]

### Core

| Function | Description |
|----------|-------------|
| `Pipe(input, …t)` | Chain transforms left-to-right |
| `Map(fn)` | Transform each element (same type; use `MapTo` to change type) |
| `Filter(pred)` | Keep elements matching a predicate |
| `FlatMap(fn)` | One-to-many expansion |

### Limiting & stopping

| Function | Description |
|----------|-------------|
| `Limit(n)` | Yield first n elements (xiter spelling; was `Take`) |
| `Drop(n)` | Skip first n elements |
| `TakeWhile(pred)` | Yield while predicate is true (stops *before* first failure) |
| `TakeUntil(pred)` | Yield until predicate matches, inclusive (stops *after* first match) |
| `DropWhile(pred)` | Skip while predicate is true |
| `WithContext(ctx)` | Stop when the context is cancelled |

```go
result := seqs.TakeWhile(func(n int) bool { return n < 10 })(input)
result := seqs.TakeUntil(func(n int) bool { return n == sentinel })(input)

ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
result := seqs.WithContext[int](ctx)(input)
```

### Combining & grouping

| Function | Description |
|----------|-------------|
| `Concat(…seqs)` | Chain sequences end-to-end |
| `Merge(s1, s2)` | Interleave two sequences sorted by natural order |
| `MergeFunc(s1, s2, cmp)` | Interleave two sequences sorted by a `cmp.Compare`-style func |
| `Dedupe(eq)` | Remove consecutive duplicates |
| `Batch(n)` | Group into fixed-size slices (`iter.Seq[[]T]`) |
| `Window(n)` | Sliding windows of size n (`iter.Seq[iter.Seq[T]]`) |

`Merge`/`MergeFunc` use `iter.Pull` internally — the canonical case for pull
iterators, since merging must compare the *heads* of both sequences:

```go
merged := seqs.Merge(seq1, seq2)                       // natural order
merged := seqs.MergeFunc(seq1, seq2, cmp.Compare[int]) // custom comparator
```

### Side effects

| Function | Description |
|----------|-------------|
| `Tee(fn)` | Call fn for each element, pass all through |
| `Siphon(pred, fn)` | Remove matching elements from the stream, send them to fn |

`Siphon` is the "errors as data" pattern: route special cases off to the side
while the pipeline keeps flowing.

## Producers & consumers

For the container boundary, **use the standard library** — `seqs` does not wrap
it:

| Need | Use |
|------|-----|
| slice/map → seq | `slices.Values`, `slices.All`, `maps.Keys`, `maps.Values`, `maps.All` |
| seq → slice/map | `slices.Collect`, `slices.Sorted`, `maps.Collect` |

`seqs` adds only the producers/consumers with **no** standard-library
equivalent:

- Producers: `Empty[T]()`, `Repeat(val, n)`
- Terminal ops: `ForEach`, `Reduce`, `Count`, `First`, `Any`, `All`

## Seq2[K, V]

Following the standard `2`-suffix convention (`iter.Pull2`, `slices.All`,
`maps.All`), every Seq combinator with a key/value analogue has a `2` twin,
plus bridges between the two worlds:

| Function | Description |
|----------|-------------|
| `Pipe2` / `Map2` / `Filter2` | Seq2 versions of `Pipe`/`Map`/`Filter` |
| `Limit2` / `Drop2` / `Tee2` / `Concat2` / `WithContext2` | …the rest of the parallels |
| `ForEach2` | Terminal: call fn for each pair |
| `Enumerate(seq)` | `Seq[T]` → `Seq2[int, T]` (add indices) |
| `Zip(a, b)` | `Seq[A] + Seq[B]` → `Seq2[A, B]` (positional pairing, stops at the shorter) |
| `Keys(seq2)` / `Values(seq2)` | `Seq2[K,V]` → `Seq[K]` / `Seq[V]` (projections) |
| `Swap(seq2)` | `Seq2[K,V]` → `Seq2[V,K]` |

```go
passing := seqs.Pipe2(maps.All(scores),
    seqs.Filter2(func(_ string, s int) bool { return s >= 5 }),
    seqs.Map2(func(name string, s int) (string, int) { return name, s + 1 }),
)
names := slices.Sorted(seqs.Keys(passing))
```

**Rule of thumb: one `Pipe` per element type.** Stay in `Seq2` while keys and
values must travel together; drop to `Seq` with `Keys`/`Values` the moment a
transform needs only one side.

## Cross-type transforms

`Map` preserves the element type so it fits `Transform[T]`. To change the type,
use the `…To` family (these can't live inside `Pipe[T]`, so call them directly):

```go
// int -> string
texts := seqs.MapTo(func(n int) string { return strconv.Itoa(n) })(numbers)

// string -> rune (flattened)
runes := seqs.FlatMapTo(func(s string) iter.Seq[rune] {
    return slices.Values([]rune(s))
})(words)

// filter and transform in one pass
valid := seqs.FilterMapTo(func(s string) (int, bool) {
    n, err := strconv.Atoi(s)
    return n, err == nil
})(words)
```

## Generic interface pattern

For types that can extract a comparable key, implement `Keyer[T, K]`:

```go
type User struct { ID, Name string }
func (u User) Key() string { return u.ID }

unique := seqs.DedupeByKey(users) // type args inferred from User.Key()
groups := seqs.GroupBy(users)     // map[string][]User
```

Or use the function-based versions when you don't control the type:

```go
unique := seqs.DedupeByFunc(func(u User) string { return u.ID })(users)
groups := seqs.GroupByFunc(func(u User) string { return u.ID }, users)
```

`Partition(pred, seq)` splits a sequence into matched/unmatched slices in one
pass.
