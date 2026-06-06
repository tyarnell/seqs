// seq2.go covers iter.Seq2[K, V] — the key/value (or index/value) sequence.
//
// The standard library uses the "2" suffix to name the Seq2 form of a thing
// (iter.Pull2, slices.All, maps.All, Concat2 in the xiter proposal). seqs
// follows that convention: every Seq combinator that has a Seq2 analogue is
// spelled with a trailing 2 (Map2, Filter2, Limit2, …), and the file also
// provides the bridges that convert between the two worlds:
//
//	Seq[T]      --Enumerate-->  Seq2[int, T]
//	Seq[A],Seq[B] --Zip------>  Seq2[A, B]
//	Seq2[K, V]  --Keys------->  Seq[K]
//	Seq2[K, V]  --Values----->  Seq[V]
//	Seq2[K, V]  --Swap------->  Seq2[V, K]
//
// Drop to Seq[T] (via Keys/Values) whenever a transform only needs one side;
// stay in Seq2 when you must keep keys and values paired.
package seqs

import (
	"context"
	"iter"
)

// --- Core Seq2 combinators (parallel to seq.go) ---

// Transform2 is the Seq2 analogue of Transform: a function that rewrites one
// key/value sequence into another. Compose them with Pipe2.
type Transform2[K, V any] func(iter.Seq2[K, V]) iter.Seq2[K, V]

// Pipe2 applies a chain of Transform2 to a Seq2, left to right.
func Pipe2[K, V any](input iter.Seq2[K, V], transforms ...Transform2[K, V]) iter.Seq2[K, V] {
	s := input
	for _, t := range transforms {
		s = t(s)
	}
	return s
}

// Map2 applies fn to each pair, yielding the rewritten pair. Like Map, it
// preserves the key and value types so it fits Transform2.
func Map2[K, V any](fn func(K, V) (K, V)) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			for k, v := range s {
				if !yield(fn(k, v)) {
					return
				}
			}
		}
	}
}

// Filter2 keeps only pairs for which pred returns true.
func Filter2[K, V any](pred func(K, V) bool) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			for k, v := range s {
				if pred(k, v) {
					if !yield(k, v) {
						return
					}
				}
			}
		}
	}
}

// Limit2 yields at most n pairs, then stops.
func Limit2[K, V any](n int) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			count := 0
			for k, v := range s {
				if count >= n {
					return
				}
				if !yield(k, v) {
					return
				}
				count++
			}
		}
	}
}

// Drop2 skips the first n pairs, then yields the rest.
func Drop2[K, V any](n int) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			count := 0
			for k, v := range s {
				if count < n {
					count++
					continue
				}
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// Tee2 calls fn for each pair, passing every pair through unchanged.
func Tee2[K, V any](fn func(K, V)) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			for k, v := range s {
				fn(k, v)
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// WithContext2 stops iteration when the context is cancelled.
func WithContext2[K, V any](ctx context.Context) Transform2[K, V] {
	return func(s iter.Seq2[K, V]) iter.Seq2[K, V] {
		return func(yield func(K, V) bool) {
			for k, v := range s {
				select {
				case <-ctx.Done():
					return
				default:
					if !yield(k, v) {
						return
					}
				}
			}
		}
	}
}

// Concat2 chains multiple Seq2 into one.
func Concat2[K, V any](seqs ...iter.Seq2[K, V]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, s := range seqs {
			for k, v := range s {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// ForEach2 calls fn for every pair in the sequence.
func ForEach2[K, V any](s iter.Seq2[K, V], fn func(K, V)) {
	for k, v := range s {
		fn(k, v)
	}
}

// --- Bridges between Seq[T] and Seq2[K, V] ---

// Enumerate wraps each element with its index, turning a Seq into a Seq2.
func Enumerate[T any](s iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		i := 0
		for v := range s {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}

// Zip pairs elements from a and b positionally into a Seq2, stopping when
// either sequence is exhausted. It uses iter.Pull to advance both in lockstep.
func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextA, stopA := iter.Pull(a)
		defer stopA()
		nextB, stopB := iter.Pull(b)
		defer stopB()
		for {
			va, oka := nextA()
			vb, okb := nextB()
			if !oka || !okb {
				return
			}
			if !yield(va, vb) {
				return
			}
		}
	}
}

// Keys projects a Seq2 onto its keys, discarding the values. (Generalises
// maps.Keys from a map to any key/value sequence.)
func Keys[K, V any](s iter.Seq2[K, V]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range s {
			if !yield(k) {
				return
			}
		}
	}
}

// Values projects a Seq2 onto its values, discarding the keys. (Generalises
// maps.Values from a map to any key/value sequence.)
func Values[K, V any](s iter.Seq2[K, V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range s {
			if !yield(v) {
				return
			}
		}
	}
}

// Swap exchanges the key and value of each pair.
func Swap[K, V any](s iter.Seq2[K, V]) iter.Seq2[V, K] {
	return func(yield func(V, K) bool) {
		for k, v := range s {
			if !yield(v, k) {
				return
			}
		}
	}
}
