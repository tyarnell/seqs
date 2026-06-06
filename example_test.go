package seqs_test

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/tyarnell/seqs"
)

// The fundamental pattern: a pipeline is just function composition over
// iter.Seq[T]. slices.Values gets you onto the bus, seqs.Pipe moves you along
// it, slices.Collect gets you off. Nothing runs until the result is ranged.
func Example_pipe() {
	input := slices.Values([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	result := seqs.Pipe(input,
		seqs.Filter(func(n int) bool { return n%2 == 0 }),
		seqs.Map(func(n int) int { return n * n }),
		seqs.Limit[int](3),
	)

	fmt.Println(slices.Collect(result))
	// Output: [4 16 36]
}

// Map transforms each element, preserving the element type.
func ExampleMap() {
	doubled := seqs.Map(func(n int) int { return n * 2 })
	fmt.Println(slices.Collect(doubled(slices.Values([]int{1, 2, 3}))))
	// Output: [2 4 6]
}

// Filter keeps only elements matching the predicate.
func ExampleFilter() {
	evens := seqs.Filter(func(n int) bool { return n%2 == 0 })
	fmt.Println(slices.Collect(evens(slices.Values([]int{1, 2, 3, 4, 5, 6}))))
	// Output: [2 4 6]
}

// FlatMap expands each element into many.
func ExampleFlatMap() {
	repeat := seqs.FlatMap(func(n int) iter.Seq[int] {
		return seqs.Repeat(n, n)
	})
	fmt.Println(slices.Collect(repeat(slices.Values([]int{1, 2, 3}))))
	// Output: [1 2 2 3 3 3]
}

// MapTo crosses type boundaries (here int -> string), unlike Map which
// preserves the element type.
func ExampleMapTo() {
	toStr := seqs.MapTo(func(n int) string { return "#" + strconv.Itoa(n) })
	fmt.Println(slices.Collect(toStr(slices.Values([]int{1, 2, 3}))))
	// Output: [#1 #2 #3]
}

// TakeWhile stops before the first element that fails the predicate.
func ExampleTakeWhile() {
	small := seqs.TakeWhile(func(n int) bool { return n < 4 })
	fmt.Println(slices.Collect(small(slices.Values([]int{1, 2, 3, 4, 1, 2}))))
	// Output: [1 2 3]
}

// Batch groups elements into fixed-size slices; the final batch may be short.
func ExampleBatch() {
	for group := range seqs.Batch[int](2)(slices.Values([]int{1, 2, 3, 4, 5})) {
		fmt.Println(group)
	}
	// Output:
	// [1 2]
	// [3 4]
	// [5]
}

// Merge interleaves two sorted sequences into one sorted sequence using the
// natural ordering. It uses iter.Pull internally — the canonical case for
// pull-style iterators.
func ExampleMerge() {
	a := slices.Values([]int{1, 3, 5, 7})
	b := slices.Values([]int{2, 4, 6})
	fmt.Println(slices.Collect(seqs.Merge(a, b)))
	// Output: [1 2 3 4 5 6 7]
}

// Siphon pulls matching elements out of the stream into a side channel while
// everything else passes through. This is the "errors as data" pattern: keep
// the pipeline flowing while collecting special cases.
func ExampleSiphon() {
	var negatives []int
	clean := seqs.Pipe(slices.Values([]int{1, -2, 3, -4, 5}),
		seqs.Siphon(func(n int) bool { return n < 0 }, func(n int) {
			negatives = append(negatives, n)
		}),
	)
	fmt.Println("passed:", slices.Collect(clean))
	fmt.Println("siphoned:", negatives)
	// Output:
	// passed: [1 3 5]
	// siphoned: [-2 -4]
}

// DedupeByFunc removes elements with duplicate keys using a key function —
// handy when you don't control the element type.
func ExampleDedupeByFunc() {
	words := slices.Values([]string{"Apple", "apple", "Banana", "BANANA", "cherry"})
	unique := seqs.DedupeByFunc(func(s string) string {
		return strings.ToLower(s)
	})
	fmt.Println(slices.Collect(unique(words)))
	// Output: [Apple Banana cherry]
}

// Reduce folds a sequence into a single value.
func ExampleReduce() {
	sum := seqs.Reduce(slices.Values([]int{1, 2, 3, 4}), 0, func(acc, n int) int {
		return acc + n
	})
	fmt.Println(sum)
	// Output: 10
}

// The Seq2 side: maps.All yields key/value pairs, the *2 combinators transform
// them, and the bridges (Keys/Values) project back down to a plain Seq when
// you only need one side. Sorting keeps the output deterministic.
func Example_seq2() {
	scores := map[string]int{"alice": 7, "bob": 3, "carol": 9, "dave": 1}

	// Stay in Seq2 to keep names and scores paired; bump every score by 1.
	passing := seqs.Pipe2(maps.All(scores),
		seqs.Filter2(func(_ string, score int) bool { return score >= 5 }),
		seqs.Map2(func(name string, score int) (string, int) {
			return name, score + 1
		}),
	)

	// Bridge down to just the names, then sort for stable output.
	names := slices.Sorted(seqs.Keys(passing))
	fmt.Println(names)
	// Output: [alice carol]
}
