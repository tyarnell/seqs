// Command seq2 demonstrates the Seq2 side of seqs: iter.Seq2[K, V] pipelines
// with the *2 combinators, and the bridges to and from plain Seq[T].
//
//	Seq[T]        --Enumerate-->  Seq2[int, T]
//	Seq[A],Seq[B] --Zip-------->  Seq2[A, B]
//	Seq2[K, V]    --Keys-------->  Seq[K]
//	Seq2[K, V]    --Values------>  Seq[V]
//
// Rule of thumb: stay in Seq2 while keys and values must travel together; drop
// to Seq with Keys/Values the moment a transform only needs one side.
//
//	go run ./examples/seq2
package main

import (
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/tyarnell/seqs"
)

type Order struct {
	ID     string
	Region string
	Amount int
}

func orders() iter.Seq[Order] {
	return slices.Values([]Order{
		{"A-1001", "us-east", 250},
		{"A-1002", "us-west", 90},
		{"A-1003", "eu", 40},
		{"A-1004", "us-east", 500},
		{"A-1005", "ap-south", 30},
		{"A-1006", "eu", 120},
	})
}

// byRegion bridges UP from Seq[Order] to Seq2[region, Order] by keying each
// order on its region. (The key comes from the value, so this is a small
// inline producer rather than seqs.Enumerate/Zip.)
func byRegion(s iter.Seq[Order]) iter.Seq2[string, Order] {
	return func(yield func(string, Order) bool) {
		for o := range s {
			if !yield(o.Region, o) {
				return
			}
		}
	}
}

func main() {
	// One Pipe2, staying in Seq2 so region and order stay paired: Filter2 keeps
	// the big orders, Map2 normalises the key to upper-case. big is pure (no
	// side effects), so it can be re-ranged freely below.
	big := seqs.Pipe2(byRegion(orders()),
		seqs.Filter2(func(_ string, o Order) bool { return o.Amount >= 50 }),
		seqs.Map2(func(region string, o Order) (string, Order) {
			return strings.ToUpper(region), o
		}),
	)

	fmt.Println("=== Pipe2: orders >= $50, region upper-cased ===")
	for region, o := range big { // ranging a Seq2 directly
		fmt.Printf("    %-9s %s ($%d)\n", region, o.ID, o.Amount)
	}

	// Tee2 taps a Seq2 for side effects. We drain it exactly once so the trace
	// prints once — re-ranging a lazy seq re-runs everything, side effects too.
	fmt.Println("\n=== Tee2: drain once for a side-effect trace ===")
	traced := seqs.Pipe2(big, seqs.Tee2(func(region string, o Order) {
		fmt.Printf("    · audit %s in %s\n", o.ID, region)
	}))
	seqs.ForEach2(traced, func(string, Order) {}) // Tee2 does the printing

	// --- Bridge DOWN: project the same Seq2 onto one side. ---
	fmt.Println("\n=== Keys / Values projections ===")
	regions := slices.Compact(slices.Sorted(seqs.Keys(big)))
	ids := slices.Collect(seqs.MapTo(func(o Order) string { return o.ID })(seqs.Values(big)))
	fmt.Println("    regions:", regions)
	fmt.Println("    ids:    ", ids)

	// --- Bridge UP from a plain Seq: Enumerate adds an index. ---
	fmt.Println("\n=== Enumerate: Seq[Order] -> Seq2[int, Order] (rank) ===")
	for rank, o := range seqs.Enumerate(seqs.Values(big)) {
		fmt.Printf("    #%d %s ($%d)\n", rank, o.ID, o.Amount)
	}

	// --- Zip pairs two Seqs positionally, stopping at the shorter one. ---
	fmt.Println("\n=== Zip: medals × amounts (stops at shorter) ===")
	medals := slices.Values([]string{"gold", "silver", "bronze"})
	amounts := seqs.MapTo(func(o Order) int { return o.Amount })(seqs.Values(big))
	for medal, amount := range seqs.Zip(medals, amounts) {
		fmt.Printf("    %-7s $%d\n", medal, amount)
	}
}
