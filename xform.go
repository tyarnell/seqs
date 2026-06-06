// xform.go provides cross-type transforms where input and output
// types differ, and generic interface constraints for pluggable behavior.
//
// The generic interface pattern from https://go.dev/blog/generic-interfaces
// is used here to express constraints like "a type that can compare itself
// to another of its kind" or "a type that can extract a key from itself".
package seqs

import (
	"iter"
)

// --- Cross-type transforms ---

// MapTo transforms a sequence of T into a sequence of U.
// Unlike Map which preserves the type, this crosses type boundaries.
func MapTo[T, U any](fn func(T) U) func(iter.Seq[T]) iter.Seq[U] {
	return func(s iter.Seq[T]) iter.Seq[U] {
		return func(yield func(U) bool) {
			for v := range s {
				if !yield(fn(v)) {
					return
				}
			}
		}
	}
}

// FlatMapTo transforms each T into a sequence of U, then flattens.
func FlatMapTo[T, U any](fn func(T) iter.Seq[U]) func(iter.Seq[T]) iter.Seq[U] {
	return func(s iter.Seq[T]) iter.Seq[U] {
		return func(yield func(U) bool) {
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

// FilterMapTo applies fn to each element; if it returns a value and true,
// that value is yielded. This combines filter + map in one pass.
func FilterMapTo[T, U any](fn func(T) (U, bool)) func(iter.Seq[T]) iter.Seq[U] {
	return func(s iter.Seq[T]) iter.Seq[U] {
		return func(yield func(U) bool) {
			for v := range s {
				if u, ok := fn(v); ok {
					if !yield(u) {
						return
					}
				}
			}
		}
	}
}

// --- Generic interface constraints ---

// Keyer is a generic interface: a type that can extract a key from itself.
// The self-referential pattern from the blog post: T implements Keyer[T, K]
// means "I can produce a key of type K from myself."
//
// Usage:
//
//	type MyRecord struct { ID string; Data string }
//	func (r MyRecord) Key() string { return r.ID }
//	// MyRecord implements Keyer[MyRecord, string]
type Keyer[T any, K comparable] interface {
	Key() K
}

// DedupeByKey removes elements with duplicate keys.
// Uses the Keyer[T, K] generic interface so any type that knows
// how to produce a comparable key can be deduplicated.
func DedupeByKey[T Keyer[T, K], K comparable](s iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(map[K]struct{})
		for v := range s {
			k := v.Key()
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				if !yield(v) {
					return
				}
			}
		}
	}
}

// GroupBy collects elements by key into a map.
// Uses the Keyer[T, K] constraint.
func GroupBy[T Keyer[T, K], K comparable](s iter.Seq[T]) map[K][]T {
	groups := make(map[K][]T)
	for v := range s {
		k := v.Key()
		groups[k] = append(groups[k], v)
	}
	return groups
}

// --- Function-based alternatives (no interface required) ---

// DedupeByFunc removes elements with duplicate keys, using a key extraction function.
// This is the function-based complement to DedupeByKey — no interface needed.
func DedupeByFunc[T any, K comparable](keyFn func(T) K) func(iter.Seq[T]) iter.Seq[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			seen := make(map[K]struct{})
			for v := range s {
				k := keyFn(v)
				if _, ok := seen[k]; !ok {
					seen[k] = struct{}{}
					if !yield(v) {
						return
					}
				}
			}
		}
	}
}

// GroupByFunc collects elements by a key function.
func GroupByFunc[T any, K comparable](keyFn func(T) K, s iter.Seq[T]) map[K][]T {
	groups := make(map[K][]T)
	for v := range s {
		k := keyFn(v)
		groups[k] = append(groups[k], v)
	}
	return groups
}

// Partition splits a sequence into two: elements matching pred, and those that don't.
// Returns two slices to avoid consuming the sequence twice.
func Partition[T any](pred func(T) bool, s iter.Seq[T]) (matched, unmatched []T) {
	for v := range s {
		if pred(v) {
			matched = append(matched, v)
		} else {
			unmatched = append(unmatched, v)
		}
	}
	return
}

// Tee sends each element to a side-effect function while passing it through.
// Useful for logging, metrics, or debugging without breaking the pipeline.
func Tee[T any](fn func(T)) func(iter.Seq[T]) iter.Seq[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				fn(v)
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Siphon removes elements matching pred from the stream and sends them to fn.
// Non-matching elements pass through normally. This is useful for extracting
// special cases from a pipeline without stopping it.
//
// Example: extract errors while processing
//
//	var errors []Error
//	result := Pipe(input,
//	    Siphon(isError, func(e Error) { errors = append(errors, e) }),
//	    Process(),
//	)
func Siphon[T any](pred func(T) bool, fn func(T)) func(iter.Seq[T]) iter.Seq[T] {
	return func(s iter.Seq[T]) iter.Seq[T] {
		return func(yield func(T) bool) {
			for v := range s {
				if pred(v) {
					fn(v)
					// Don't yield — element is siphoned off
				} else {
					if !yield(v) {
						return
					}
				}
			}
		}
	}
}
