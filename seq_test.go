package seqs

import (
	"context"
	"iter"
	"slices"
	"testing"
)

// of is a terse producer for tests. Public API users reach for slices.Values
// (of(vs...) is exactly slices.Values(vs)); this just saves the []T literal.
func of[T any](vs ...T) iter.Seq[T] { return slices.Values(vs) }

func TestPipe(t *testing.T) {
	input := of(1, 2, 3, 4, 5)

	double := Map(func(n int) int { return n * 2 })
	keepEven := Filter(func(n int) bool { return n%2 == 0 })

	result := slices.Collect(Pipe(input, double, keepEven))
	expected := []int{2, 4, 6, 8, 10}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestFilter(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	result := slices.Collect(Filter(func(n int) bool { return n > 3 })(input))
	expected := []int{4, 5}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMap(t *testing.T) {
	input := of("a", "b", "c")
	result := slices.Collect(Map(func(s string) string { return s + s })(input))
	expected := []string{"aa", "bb", "cc"}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestLimit(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	result := slices.Collect(Limit[int](3)(input))
	expected := []int{1, 2, 3}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDrop(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	result := slices.Collect(Drop[int](2)(input))
	expected := []int{3, 4, 5}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestConcat(t *testing.T) {
	a := of(1, 2)
	b := of(3, 4)
	result := slices.Collect(Concat(a, b))
	expected := []int{1, 2, 3, 4}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestFirst(t *testing.T) {
	val, ok := First(of(10, 20, 30))
	if !ok || val != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", val, ok)
	}

	val, ok = First(Empty[int]())
	if ok {
		t.Errorf("expected (_, false), got (%d, %v)", val, ok)
	}
}

func TestCount(t *testing.T) {
	n := Count(of(1, 2, 3, 4))
	if n != 4 {
		t.Errorf("expected 4, got %d", n)
	}
}

func TestReduce(t *testing.T) {
	sum := Reduce(of(1, 2, 3, 4), 0, func(acc, n int) int { return acc + n })
	if sum != 10 {
		t.Errorf("expected 10, got %d", sum)
	}
}

func TestAny(t *testing.T) {
	if !Any(of(1, 2, 3), func(n int) bool { return n == 2 }) {
		t.Error("expected Any to return true")
	}
	if Any(of(1, 2, 3), func(n int) bool { return n == 5 }) {
		t.Error("expected Any to return false")
	}
}

func TestAll(t *testing.T) {
	if !All(of(2, 4, 6), func(n int) bool { return n%2 == 0 }) {
		t.Error("expected All to return true")
	}
	if All(of(2, 3, 4), func(n int) bool { return n%2 == 0 }) {
		t.Error("expected All to return false")
	}
}

func TestMerge(t *testing.T) {
	// Two sorted sequences, natural ordering.
	s1 := of(1, 3, 5, 7)
	s2 := of(2, 4, 6, 8)

	result := slices.Collect(Merge(s1, s2))
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeUneven(t *testing.T) {
	// One sequence longer than the other.
	s1 := of(1, 5, 9)
	s2 := of(2, 3, 4, 6, 7, 8, 10)

	result := slices.Collect(Merge(s1, s2))
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeEmpty(t *testing.T) {
	s1 := of(1, 2, 3)
	s2 := Empty[int]()

	result := slices.Collect(Merge(s1, s2))
	expected := []int{1, 2, 3}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeFunc(t *testing.T) {
	// Descending sequences merged with a reversed comparator.
	s1 := of(7, 5, 3, 1)
	s2 := of(8, 6, 4, 2)
	desc := func(a, b int) int { return b - a }

	result := slices.Collect(MergeFunc(s1, s2, desc))
	expected := []int{8, 7, 6, 5, 4, 3, 2, 1}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestTakeWhile(t *testing.T) {
	input := of(1, 2, 3, 4, 5, 1, 2)
	result := slices.Collect(TakeWhile(func(n int) bool { return n < 4 })(input))
	expected := []int{1, 2, 3}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestTakeUntil(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	// Stop after finding 3 (include 3 in output).
	result := slices.Collect(TakeUntil(func(n int) bool { return n == 3 })(input))
	expected := []int{1, 2, 3}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestTakeUntilNoMatch(t *testing.T) {
	input := of(1, 2, 3)
	// Predicate never matches — should yield all elements.
	result := slices.Collect(TakeUntil(func(n int) bool { return n > 10 })(input))
	expected := []int{1, 2, 3}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDropWhile(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	result := slices.Collect(DropWhile(func(n int) bool { return n < 3 })(input))
	expected := []int{3, 4, 5}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a sequence that will be cancelled mid-iteration.
	var yielded []int
	input := of(1, 2, 3, 4, 5)
	result := WithContext[int](ctx)(input)

	for v := range result {
		yielded = append(yielded, v)
		if v == 3 {
			cancel() // Cancel after receiving 3.
		}
	}

	// Should have stopped at or shortly after 3.
	if len(yielded) > 4 {
		t.Errorf("expected at most 4 elements, got %d: %v", len(yielded), yielded)
	}
	if len(yielded) < 3 {
		t.Errorf("expected at least 3 elements, got %d: %v", len(yielded), yielded)
	}
}

func TestWithContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	input := of(1, 2, 3, 4, 5)
	result := slices.Collect(WithContext[int](ctx)(input))

	// Should yield nothing or very few elements.
	if len(result) > 1 {
		t.Errorf("expected at most 1 element from cancelled context, got %d", len(result))
	}
}

func TestFlatMap(t *testing.T) {
	input := of(1, 2, 3)
	result := slices.Collect(FlatMap(func(n int) iter.Seq[int] {
		return of(n, n*10)
	})(input))
	expected := []int{1, 10, 2, 20, 3, 30}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDedupe(t *testing.T) {
	input := of(1, 1, 2, 2, 2, 3, 1, 1)
	result := slices.Collect(Dedupe(func(a, b int) bool { return a == b })(input))
	expected := []int{1, 2, 3, 1}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBatch(t *testing.T) {
	input := of(1, 2, 3, 4, 5)
	batches := slices.Collect(Batch[int](2)(input))

	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if !slices.Equal(batches[0], []int{1, 2}) {
		t.Errorf("expected [1,2], got %v", batches[0])
	}
	if !slices.Equal(batches[1], []int{3, 4}) {
		t.Errorf("expected [3,4], got %v", batches[1])
	}
	if !slices.Equal(batches[2], []int{5}) {
		t.Errorf("expected [5], got %v", batches[2])
	}
}

func TestRepeat(t *testing.T) {
	result := slices.Collect(Repeat("x", 3))
	expected := []string{"x", "x", "x"}

	if !slices.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestForEach(t *testing.T) {
	var sum int
	ForEach(of(1, 2, 3), func(n int) { sum += n })

	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}
