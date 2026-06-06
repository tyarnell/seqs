# seqs — generic combinators for `iter.Seq` (Go 1.23+)

The missing sibling of `slices` and `maps`:

```
slices : []T            ← operations on slices
maps   : map[K]V        ← operations on maps
seqs   : iter.Seq[T]    ← operations on sequences  (this package)
```

`slices` and `maps` bridge a container to and from `iter.Seq`. `seqs` is the
combinator layer in between — `Map`, `Filter`, `Limit`, `Merge`, `Batch`, … —
that the standard library left out (the [`xiter` proposal](https://go.dev/issue/61898)
is still on hold).

## Install

```bash
go get github.com/tyarnell/seqs
```

## Quick start

The pattern is always: **`slices.Values` onto the bus → `seqs.Pipe` along it → `slices.Collect` off.**

```go
import (
    "slices"
    "github.com/tyarnell/seqs"
)

result := slices.Collect(seqs.Pipe(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
    seqs.Filter(func(n int) bool { return n%2 == 0 }),
    seqs.Map(func(n int) int { return n * n }),
    seqs.Limit[int](3),
))
// result == [4 16 36]
```

`Transform[T]` is just `func(iter.Seq[T]) iter.Seq[T]`, so a pipeline is plain
function composition — lazy, generic, no pipeline type or registry. `seqs` leans
on `slices`/`maps` at the boundaries and never reimplements them.

## Examples

Runnable demos, one program per subfolder:

```bash
go run ./examples/basics   # core flow: Pipe/Filter/Map/Limit, Merge, Reduce, a lazy infinite source
go run ./examples/etl      # ETL on Seq[T]: parse → siphon errors → filter → dedupe → batch
go run ./examples/seq2     # Seq2: Pipe2/Filter2/Map2/Tee2 + bridges (Enumerate, Zip, Keys, Values)
go run ./examples/byoc     # use seqs with your own container type
```

`example_test.go` also has runnable, output-verified `Example*` functions
(rendered on pkg.go.dev).

## Docs

- **[docs/concepts.md](docs/concepts.md)** — where `seqs` fits: the three iterator layers, and the design principles.
- **[docs/combinators.md](docs/combinators.md)** — full reference: `Seq[T]`, `Seq2[K,V]`, cross-type, and the generic-interface pattern.
- **[docs/byoc.md](docs/byoc.md)** — bring your own container.

---

*Standalone module extracted from the `docproc` document-processing project,
where these combinators power the parsing/transform pipeline.*
