// Package seqs provides generic combinators for iter.Seq[T] — the
// transformation layer the standard library deliberately left out.
//
// Think of it as the missing sibling of slices and maps:
//
//	slices : []T            ← operations on slices
//	maps   : map[K]V        ← operations on maps
//	seqs   : iter.Seq[T]    ← operations on sequences (this package)
//
// The standard library ships the *type* layer (package iter) and the
// *adapter* layer (slices.Values/All/Collect/Sorted, maps.Keys/Values/All)
// that bridge a concrete container to and from iter.Seq. seqs fills the
// *combinator* layer: container-agnostic transforms (Map, Filter, FlatMap,
// Limit, Merge, Batch, …) that go Seq → Seq, plus the Seq2 parallels and the
// Seq↔Seq2 bridges (see seq2.go).
//
// Because seqs owns only the middle of a pipeline, it never imports slices or
// maps in its transforms: get onto the bus with slices.Values / maps.Values,
// move along it with seqs, and get off with slices.Collect / slices.Sorted /
// maps.Collect.
//
//	import (
//	    "slices"
//	    "github.com/tyarnell/seqs"
//	)
//
//	result := slices.Collect(seqs.Pipe(slices.Values([]int{1, 2, 3, 4, 5}),
//	    seqs.Filter(func(n int) bool { return n%2 == 0 }),
//	    seqs.Map(func(n int) int { return n * 2 }),
//	))
//
// A pipeline is just function composition over iter.Seq[T]: there is no
// Pipeline type, no Step interface, no registry of transforms. Just functions
// that take a sequence and return a sequence.
package seqs

import (
	"cmp"
	"context"
	"iter"
)

// --- Core Combinators ---

// Transform is a function that transforms one sequence into another.
// This is the unit of composition. Pipelines are built by composing Transforms.
type Transform[T any] func(iter.Seq[T]) iter.Seq[T]

// Pipe applies a chain of transforms to a sequence, left to right.
// This is the primary way to build processing pipelines.
//
//	result := Pipe(input, step1, step2, step3)
//
// is equivalent to step3(step2(step1(input))).
func Pipe[T any](input iter.Seq[T], transforms ...Transform[T]) iter.Seq[T] {
	s := input
	for _, t := range transforms {
		s = t(s)
	}
	return s
}

// Map applies fn to each element in the sequence, yielding the results.
//
// Map preserves the element type (T → T) so it fits Transform[T] and chains
// inside Pipe. To change the element type (T → U), use MapTo.
func Map[T any](fn func(T) T) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				if !yield(fn(v)) {
					return
				}
			}
		}
	}
}

// Filter returns a transform that keeps only elements for which pred returns true.
func Filter[T any](pred func(T) bool) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				if pred(v) {
					if !yield(v) {
						return
					}
				}
			}
		}
	}
}

// FlatMap applies fn to each element, yielding all elements from the returned sequence.
// This is how one element can expand into many.
func FlatMap[T any](fn func(T) iter.Seq[T]) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				for inner := range fn(v) {
					if !yield(inner) {
						return
					}
				}
			}
		}
	}
}

// Limit yields at most n elements from the sequence, then stops.
//
// The name mirrors the xiter proposal (golang.org/issue/61898), where this
// operation is Limit rather than Take.
func Limit[T any](n int) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			count := 0
			for v := range s {
				if count >= n {
					return
				}
				if !yield(v) {
					return
				}
				count++
			}
		}
	}
}

// Drop skips the first n elements, then yields the rest.
func Drop[T any](n int) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			count := 0
			for v := range s {
				if count < n {
					count++
					continue
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}

// TakeWhile yields elements as long as pred returns true, then stops.
// The element that causes pred to return false is NOT yielded.
func TakeWhile[T any](pred func(T) bool) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				if !pred(v) {
					return
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}

// TakeUntil yields elements until pred returns true, then yields that element and stops.
// The element that causes pred to return true IS yielded (unlike TakeWhile).
//
//	// Stop after finding a sentinel value, including it in output
//	result := TakeUntil(func(v int) bool { return v == -1 })(input)
func TakeUntil[T any](pred func(T) bool) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				if pred(v) {
					yield(v) // Yield the matching element, then stop
					return
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}

// DropWhile skips elements as long as pred returns true, then yields the rest.
func DropWhile[T any](pred func(T) bool) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			dropping := true
			for v := range s {
				if dropping {
					if pred(v) {
						continue
					}
					dropping = false
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}

// WithContext stops iteration when the context is cancelled.
// This allows pipelines to respect cancellation signals for graceful shutdown.
//
//	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
//	defer cancel()
//	result := Pipe(input, WithContext[int](ctx), Map(process))
func WithContext[T any](ctx context.Context) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				select {
				case <-ctx.Done():
					return
				default:
					if !yield(v) {
						return
					}
				}
			}
		}
	}
}

// --- Producers ---
//
// To turn a concrete container into a sequence, prefer the standard library:
// slices.Values(slice), maps.Values(m), maps.Keys(m). seqs only adds producers
// that have no standard-library equivalent.

// Empty returns a sequence with no elements.
func Empty[T any]() iter.Seq[T] {
	return func(yield func(T) bool) {}
}

// Repeat yields the same value n times. (slices.Repeat repeats a slice, not a
// single value, so this has no direct standard-library equivalent.)
func Repeat[T any](val T, n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		for range n {
			if !yield(val) {
				return
			}
		}
	}
}

// --- Consumers ---
//
// To gather a sequence back into a container, prefer the standard library:
// slices.Collect(seq), slices.Sorted(seq), maps.Collect(seq2). seqs only adds
// terminal operations that have no standard-library equivalent.

// ForEach calls fn for every element in the sequence.
func ForEach[T any](s iter.Seq[T], fn func(T)) {
	for v := range s {
		fn(v)
	}
}

// Reduce folds a sequence into a single value.
//
// The sequence comes first to match the other seqs consumers (Count, First,
// Any, All); the xiter proposal orders the arguments differently.
func Reduce[T, R any](s iter.Seq[T], initial R, fn func(R, T) R) R {
	acc := initial
	for v := range s {
		acc = fn(acc, v)
	}
	return acc
}

// Count returns the number of elements in the sequence.
func Count[T any](s iter.Seq[T]) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// First returns the first element and true, or the zero value and false.
func First[T any](s iter.Seq[T]) (T, bool) {
	for v := range s {
		return v, true
	}
	var zero T
	return zero, false
}

// Any returns true if pred returns true for any element.
func Any[T any](s iter.Seq[T], pred func(T) bool) bool {
	for v := range s {
		if pred(v) {
			return true
		}
	}
	return false
}

// All returns true if pred returns true for every element.
func All[T any](s iter.Seq[T], pred func(T) bool) bool {
	for v := range s {
		if !pred(v) {
			return false
		}
	}
	return true
}

// --- Sequence Combinators ---

// Concat chains multiple sequences into one.
func Concat[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, s := range seqs {
			for v := range s {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Dedupe removes consecutive duplicate elements using eq for comparison.
func Dedupe[T any](eq func(T, T) bool) Transform[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			first := true
			var prev T
			for v := range s {
				if first || !eq(prev, v) {
					if !yield(v) {
						return
					}
					prev = v
					first = false
				}
			}
		}
	}
}

// Window yields sliding windows of size n over the sequence.
// Each window is itself a sequence (not a slice), keeping things lazy.
func Window[T any](n int) func(iter.Seq[T]) iter.Seq[iter.Seq[T]] {
	return func(s iter.Seq[T]) iter.Seq[iter.Seq[T]] {
		return func(yield func(iter.Seq[T]) bool) {
			// Windows need buffering, so we materialize here.
			buf := make([]T, 0, n)
			for v := range s {
				buf = append(buf, v)
				if len(buf) > n {
					buf = buf[1:]
				}
				if len(buf) == n {
					// Copy to avoid mutation.
					window := make([]T, n)
					copy(window, buf)
					if !yield(slicesValues(window)) {
						return
					}
				}
			}
		}
	}
}

// Batch collects elements into groups of at most n, yielding each group as a slice.
func Batch[T any](n int) func(iter.Seq[T]) iter.Seq[[]T] {
	return func(s iter.Seq[T]) iter.Seq[[]T] {
		return func(yield func([]T) bool) {
			batch := make([]T, 0, n)
			for v := range s {
				batch = append(batch, v)
				if len(batch) >= n {
					if !yield(batch) {
						return
					}
					batch = make([]T, 0, n)
				}
			}
			if len(batch) > 0 {
				yield(batch)
			}
		}
	}
}

// Merge interleaves two sorted sequences into one sorted sequence, comparing
// elements with the natural ordering. For a custom comparison, use MergeFunc.
//
//	merged := Merge(seq1, seq2) // both ascending
func Merge[T cmp.Ordered](s1, s2 iter.Seq[T]) iter.Seq[T] {
	return MergeFunc(s1, s2, cmp.Compare[T])
}

// MergeFunc interleaves two sequences sorted by compare into one sorted
// sequence. compare reports whether a is less than (<0), equal to (0), or
// greater than (>0) b — the same shape as cmp.Compare.
//
// It uses iter.Pull internally because merge requires comparing the heads of
// both sequences — something push iterators can't do alone.
//
//	merged := MergeFunc(seq1, seq2, func(a, b int) int { return a - b })
func MergeFunc[T any](s1, s2 iter.Seq[T], compare func(T, T) int) iter.Seq[T] {
	return func(yield func(T) bool) {
		next1, stop1 := iter.Pull(s1)
		defer stop1()
		next2, stop2 := iter.Pull(s2)
		defer stop2()

		v1, ok1 := next1()
		v2, ok2 := next2()

		for ok1 && ok2 {
			if compare(v1, v2) <= 0 {
				if !yield(v1) {
					return
				}
				v1, ok1 = next1()
			} else {
				if !yield(v2) {
					return
				}
				v2, ok2 = next2()
			}
		}

		// Drain remaining from s1
		for ok1 {
			if !yield(v1) {
				return
			}
			v1, ok1 = next1()
		}

		// Drain remaining from s2
		for ok2 {
			if !yield(v2) {
				return
			}
			v2, ok2 = next2()
		}
	}
}

// slicesValues is a tiny local copy of slices.Values, used so the core
// transforms stay free of a slices import. Window is the only caller.
func slicesValues[T any](s []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s {
			if !yield(v) {
				return
			}
		}
	}
}
