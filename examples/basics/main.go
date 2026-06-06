// Command basics shows the fundamental seqs flow and how it sits between the
// standard library's container packages:
//
//	slices.Values  → onto the bus (container -> Seq)
//	seqs.Pipe      → along the bus (Seq -> Seq)
//	slices.Collect → off the bus   (Seq -> container)
//
//	go run ./examples/basics
package main

import (
	"cmp"
	"fmt"
	"iter"
	"slices"

	"github.com/tyarnell/seqs"
)

func main() {
	// --- Steps: define once, then read each pipeline below as a recipe. ---
	var (
		keepEven = seqs.Filter(func(n int) bool { return n%2 == 0 })
		square   = seqs.Map(func(n int) int { return n * n })
		cube     = seqs.Map(func(n int) int { return n * n * n })
		first3   = seqs.Limit[int](3)
		first5   = seqs.Limit[int](5)
	)

	// --- Recipes. ---
	nums := slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	fmt.Println("even squares, first 3:", slices.Collect(seqs.Pipe(nums, keepEven, square, first3)))

	// Merge two sorted streams: natural order, then a custom (descending) comparator.
	asc := seqs.Merge(slices.Values([]int{1, 3, 5, 7}), slices.Values([]int{2, 4, 6}))
	fmt.Println("merged (asc):         ", slices.Collect(asc))

	descending := func(x, y int) int { return cmp.Compare(y, x) }
	desc := seqs.MergeFunc(slices.Values([]int{7, 5, 3}), slices.Values([]int{8, 4}), descending)
	fmt.Println("merged (desc):        ", slices.Collect(desc))

	// Terminal fold.
	fmt.Println("sum:                  ", seqs.Reduce(slices.Values([]int{1, 2, 3, 4}), 0,
		func(acc, n int) int { return acc + n }))

	// Laziness: an unbounded source is fine because Limit stops it.
	fmt.Println("first 5 cubes:        ", slices.Collect(seqs.Pipe(naturals(), cube, first5)))
}

// naturals is an unbounded source: 1, 2, 3, … Any stopping step (Limit,
// TakeWhile, WithContext) bounds it.
func naturals() iter.Seq[int] {
	return func(yield func(int) bool) {
		for n := 1; ; n++ {
			if !yield(n) {
				return
			}
		}
	}
}
