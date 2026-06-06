// Command byoc shows how to use seqs with your own container type.
//
// seqs only speaks iter.Seq[T], so any type that can produce one plugs in. The
// idiomatic move (mirroring slices/maps) is to give your container the standard
// adapter vocabulary — All()/Values() methods to get a sequence out, and a
// Collect function to build one back — then let seqs do the transforms between.
//
//	go run ./examples/byoc
package main

import (
	"fmt"
	"iter"
	"slices"

	"github.com/tyarnell/seqs"
)

// Ring is a fixed-capacity ring buffer: a custom container with a non-trivial
// iteration order (oldest → newest), so it can't just be a slice.
type Ring[T any] struct {
	buf   []T
	start int
	len   int
}

func NewRing[T any](capacity int) *Ring[T] { return &Ring[T]{buf: make([]T, capacity)} }

// Push appends v, overwriting the oldest element once the ring is full.
func (r *Ring[T]) Push(v T) {
	end := (r.start + r.len) % len(r.buf)
	r.buf[end] = v
	if r.len == len(r.buf) {
		r.start = (r.start + 1) % len(r.buf) // full: advance past the overwritten oldest
	} else {
		r.len++
	}
}

// All is the primary iterator (position-preserving), mirroring slices.All: it
// yields (age, value) from oldest to newest.
func (r *Ring[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i := range r.len {
			if !yield(i, r.buf[(r.start+i)%len(r.buf)]) {
				return
			}
		}
	}
}

// Values is the value-only projection, mirroring slices.Values. It reuses the
// seqs.Values bridge to drop the keys from All().
func (r *Ring[T]) Values() iter.Seq[T] { return seqs.Values(r.All()) }

// CollectRing builds a Ring from a sequence — the way back in, mirroring
// slices.Collect.
func CollectRing[T any](seq iter.Seq[T], capacity int) *Ring[T] {
	r := NewRing[T](capacity)
	for v := range seq {
		r.Push(v)
	}
	return r
}

func main() {
	r := NewRing[int](5)
	for i := 1; i <= 8; i++ { // 8 pushes into a cap-5 ring keeps the last 5: 4..8
		r.Push(i)
	}
	fmt.Println("ring (oldest→newest):", slices.Collect(r.Values()))

	// --- Steps, then a recipe — a custom container is no different from any
	// other source: its Values() gets us onto the bus, seqs does the middle. ---
	var (
		keepEven = seqs.Filter(func(n int) bool { return n%2 == 0 })
		double   = seqs.Map(func(n int) int { return n * 2 })
		tenfold  = seqs.Map(func(n int) int { return n * 10 })
	)

	fmt.Println("evens × 2:           ", slices.Collect(seqs.Pipe(r.Values(), keepEven, double)))

	// All() carries position — range it as a Seq2 (or transform with the *2 set).
	fmt.Println("by age:")
	for age, v := range r.All() {
		fmt.Printf("  age %d → %d\n", age, v)
	}

	// The way back: transform a sequence, then Collect into a new container.
	r2 := CollectRing(seqs.Pipe(r.Values(), tenfold), 3)
	fmt.Println("collected back (cap 3):", slices.Collect(r2.Values()))
}
