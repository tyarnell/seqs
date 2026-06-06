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
	"slices"

	"github.com/tyarnell/seqs"
)

func main() {
	// Filter -> Map -> Limit, composed in one Pipe.
	evenSquares := slices.Collect(seqs.Pipe(slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		seqs.Filter(func(n int) bool { return n%2 == 0 }),
		seqs.Map(func(n int) int { return n * n }),
		seqs.Limit[int](3),
	))
	fmt.Println("even squares, first 3:", evenSquares)

	// Merge two sorted streams (Merge for natural order, MergeFunc for custom).
	a := slices.Values([]int{1, 3, 5, 7})
	b := slices.Values([]int{2, 4, 6})
	fmt.Println("merged (asc):  ", slices.Collect(seqs.Merge(a, b)))

	desc1 := slices.Values([]int{7, 5, 3})
	desc2 := slices.Values([]int{8, 4})
	rev := func(x, y int) int { return cmp.Compare(y, x) }
	fmt.Println("merged (desc): ", slices.Collect(seqs.MergeFunc(desc1, desc2, rev)))

	// Terminal fold.
	sum := seqs.Reduce(slices.Values([]int{1, 2, 3, 4}), 0, func(acc, n int) int { return acc + n })
	fmt.Println("sum:           ", sum)

	// Laziness: an infinite source is fine because Limit stops it.
	naturals := func(yield func(int) bool) {
		for n := 1; ; n++ {
			if !yield(n) {
				return
			}
		}
	}
	fmt.Println("first 5 cubes: ", slices.Collect(seqs.Pipe(naturals,
		seqs.Map(func(n int) int { return n * n * n }),
		seqs.Limit[int](5),
	)))
}
