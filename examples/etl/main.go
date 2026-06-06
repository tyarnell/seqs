// Command etl demonstrates the seqs framework as a small, realistic ETL
// pipeline: parse raw CSV-ish lines into typed records, clean and validate
// them, route bad rows to a side channel ("errors as data"), dedupe, then
// batch the survivors for a simulated bulk write — all lazily, one element
// at a time.
//
//	go run ./examples/etl
package main

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/tyarnell/seqs"
)

// Order is our domain record. It implements seqs.Keyer[Order, string] by
// exposing Key(), so it works with DedupeByKey / GroupBy out of the box.
type Order struct {
	ID     string
	Region string
	Amount int
}

func (o Order) Key() string { return o.ID }

// rawOrders simulates an unbounded source (a file, a Kafka topic, an HTTP
// stream). Because it's an iter.Seq, nothing is read until the pipeline pulls.
func rawOrders() iter.Seq[string] {
	lines := []string{
		"A-1001, us-east, 250",
		"A-1002, us-west , 90",
		"  ",                   // blank -> skipped
		"A-1003, eu , notanum", // bad amount -> siphoned as an error
		"A-1001, us-east, 250", // duplicate ID -> deduped
		"A-1004, us-east, 500",
		"A-1005, ap-south, 30",
	}
	return slices.Values(lines)
}

// parse turns "id, region, amount" into an Order. On any problem it returns a
// non-nil error so the pipeline can route it instead of panicking.
func parse(line string) (Order, error) {
	parts := strings.Split(line, ",")
	if len(parts) != 3 {
		return Order{}, fmt.Errorf("want 3 fields, got %d: %q", len(parts), line)
	}
	amount, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return Order{}, fmt.Errorf("bad amount in %q: %w", line, err)
	}
	return Order{
		ID:     strings.TrimSpace(parts[0]),
		Region: strings.TrimSpace(parts[1]),
		Amount: amount,
	}, nil
}

func main() {
	// result is the carrier type that lets one Pipe span the whole chain: keep
	// every stage in the result domain and they're all Transform[result], so
	// they slot straight into a single seqs.Pipe.
	type result struct {
		order Order
		err   error
	}

	var failures []error

	// --- Stage definitions: each is one named transform, defined once. ---
	var (
		// Edge steps (cross-type, live outside the Pipe):
		nonBlank = seqs.Filter(func(s string) bool { // string -> string
			return strings.TrimSpace(s) != ""
		})
		asResult = seqs.MapTo(func(line string) result { // string -> result
			o, err := parse(line)
			return result{o, err}
		})

		// In-domain steps (Transform[result], chain inside one Pipe):
		siphonErrors = seqs.Siphon( // route parse failures to the side
			func(r result) bool { return r.err != nil },
			func(r result) { failures = append(failures, r.err) },
		)
		dropTiny = seqs.Filter(func(r result) bool { // keep orders >= $50
			return r.order.Amount >= 50
		})
		trace = seqs.Tee(func(r result) { // peek, pass everything through
			fmt.Printf("  · accepted %s\n", r.order.ID)
		})
		dedupeByID = seqs.DedupeByFunc(func(r result) string { // dedupe by ID
			return r.order.ID
		})
	)

	// --- The flow: read it top to bottom. ---
	source := asResult(seqs.Pipe(rawOrders(), nonBlank))
	clean := seqs.Pipe(source,
		siphonErrors,
		dropTiny,
		trace,
		dedupeByID,
	)

	// Batch can't live inside the Pipe (it returns iter.Seq[[]result], a
	// different type), so it runs as a plain call on the result.
	fmt.Println("=== pipeline trace ===")
	batches := slices.Collect(seqs.Batch[result](2)(clean))

	fmt.Println("\n=== bulk writes (batch size 2) ===")
	for i, batch := range batches {
		fmt.Printf("batch %d:\n", i+1)
		for _, r := range batch {
			fmt.Printf("  %-7s %-9s $%d\n", r.order.ID, r.order.Region, r.order.Amount)
		}
	}

	fmt.Println("\n=== siphoned errors ===")
	for _, e := range failures {
		fmt.Println("  -", e)
	}

	// --- Bonus: laziness. An infinite source is fine because Limit stops it. ---
	fmt.Println("\n=== lazy: first 5 squares from an infinite source ===")
	naturals := func(yield func(int) bool) {
		for n := 1; ; n++ {
			if !yield(n) {
				return
			}
		}
	}
	squares := seqs.Pipe(
		naturals,
		seqs.Map(func(n int) int { return n * n }),
		seqs.Limit[int](5),
	)
	fmt.Println(" ", slices.Collect(squares))
}
