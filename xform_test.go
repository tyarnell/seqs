package seqs

import (
	"iter"
	"slices"
	"testing"
)

func TestMapTo(t *testing.T) {
	input := of(1, 2, 3)
	result := slices.Collect(MapTo(func(n int) string {
		return string(rune('a' + n - 1))
	})(input))

	expected := []string{"a", "b", "c"}
	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestFlatMapTo(t *testing.T) {
	input := of("ab", "cd")
	result := slices.Collect(FlatMapTo(func(s string) iter.Seq[rune] {
		return func(yield func(rune) bool) {
			for _, r := range s {
				if !yield(r) {
					return
				}
			}
		}
	})(input))

	expected := []rune{'a', 'b', 'c', 'd'}
	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestFilterMapTo(t *testing.T) {
	input := of("1", "abc", "2", "def", "3")
	result := slices.Collect(FilterMapTo(func(s string) (int, bool) {
		if len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
			return int(s[0] - '0'), true
		}
		return 0, false
	})(input))

	expected := []int{1, 2, 3}
	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// Test type for Keyer interface
type keyedItem struct {
	id   string
	name string
}

func (k keyedItem) Key() string { return k.id }

func TestDedupeByKey(t *testing.T) {
	input := of(
		keyedItem{"a", "first"},
		keyedItem{"b", "second"},
		keyedItem{"a", "duplicate"},
		keyedItem{"c", "third"},
	)

	result := slices.Collect(DedupeByKey[keyedItem, string](input))

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	if result[0].name != "first" {
		t.Errorf("expected 'first', got %q", result[0].name)
	}
}

func TestGroupBy(t *testing.T) {
	input := of(
		keyedItem{"a", "one"},
		keyedItem{"b", "two"},
		keyedItem{"a", "three"},
	)

	groups := GroupBy[keyedItem, string](input)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["a"]) != 2 {
		t.Errorf("expected 2 items in group 'a', got %d", len(groups["a"]))
	}
	if len(groups["b"]) != 1 {
		t.Errorf("expected 1 item in group 'b', got %d", len(groups["b"]))
	}
}

func TestDedupeByFunc(t *testing.T) {
	input := of("apple", "apricot", "banana", "avocado")
	// Dedupe by first letter
	result := slices.Collect(DedupeByFunc(func(s string) byte { return s[0] })(input))

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0] != "apple" || result[1] != "banana" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestGroupByFunc(t *testing.T) {
	input := of(1, 2, 3, 4, 5, 6)
	groups := GroupByFunc(func(n int) string {
		if n%2 == 0 {
			return "even"
		}
		return "odd"
	}, input)

	if len(groups["even"]) != 3 {
		t.Errorf("expected 3 even numbers, got %d", len(groups["even"]))
	}
	if len(groups["odd"]) != 3 {
		t.Errorf("expected 3 odd numbers, got %d", len(groups["odd"]))
	}
}

func TestPartition(t *testing.T) {
	input := of(1, 2, 3, 4, 5, 6)
	even, odd := Partition(func(n int) bool { return n%2 == 0 }, input)

	if !slices.Equal(even, []int{2, 4, 6}) {
		t.Errorf("expected [2,4,6], got %v", even)
	}
	if !slices.Equal(odd, []int{1, 3, 5}) {
		t.Errorf("expected [1,3,5], got %v", odd)
	}
}

func TestTee(t *testing.T) {
	input := of(1, 2, 3)

	var sideEffect []int
	result := slices.Collect(Tee(func(n int) {
		sideEffect = append(sideEffect, n*10)
	})(input))

	if !slices.Equal(result, []int{1, 2, 3}) {
		t.Errorf("expected pass-through [1,2,3], got %v", result)
	}
	if !slices.Equal(sideEffect, []int{10, 20, 30}) {
		t.Errorf("expected side effect [10,20,30], got %v", sideEffect)
	}
}

func TestSiphon(t *testing.T) {
	input := of(1, 2, 3, 4, 5, 6)

	// Siphon off even numbers
	var evens []int
	result := slices.Collect(Siphon(
		func(n int) bool { return n%2 == 0 },
		func(n int) { evens = append(evens, n) },
	)(input))

	// Result should only have odd numbers
	if !slices.Equal(result, []int{1, 3, 5}) {
		t.Errorf("expected [1,3,5], got %v", result)
	}
	// Evens should be siphoned
	if !slices.Equal(evens, []int{2, 4, 6}) {
		t.Errorf("expected siphoned [2,4,6], got %v", evens)
	}
}
