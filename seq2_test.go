package seqs

import (
	"maps"
	"slices"
	"testing"
)

// pairs collects a Seq2 into parallel key/value slices for easy assertions.
func pairs[K, V any](s func(yield func(K, V) bool)) ([]K, []V) {
	var ks []K
	var vs []V
	for k, v := range s {
		ks = append(ks, k)
		vs = append(vs, v)
	}
	return ks, vs
}

func TestPipe2(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}

	got := Pipe2(maps.All(m),
		Filter2(func(_ string, v int) bool { return v%2 == 0 }),
		Map2(func(k string, v int) (string, int) { return k, v * 10 }),
	)

	// Order from maps.All is nondeterministic; compare as a rebuilt map.
	out := maps.Collect(got)
	want := map[string]int{"b": 20, "d": 40}
	if !maps.Equal(out, want) {
		t.Errorf("expected %v, got %v", want, out)
	}
}

func TestLimit2(t *testing.T) {
	n := 0
	for range Limit2[int, int](2)(Enumerate(of(10, 20, 30, 40))) {
		n++
	}
	if n != 2 {
		t.Errorf("expected 2 pairs, got %d", n)
	}
}

func TestEnumerate(t *testing.T) {
	ks, vs := pairs(Enumerate(of("x", "y", "z")))
	if !slices.Equal(ks, []int{0, 1, 2}) {
		t.Errorf("expected indices [0 1 2], got %v", ks)
	}
	if !slices.Equal(vs, []string{"x", "y", "z"}) {
		t.Errorf("expected values [x y z], got %v", vs)
	}
}

func TestZip(t *testing.T) {
	// Zip stops at the shorter sequence.
	ks, vs := pairs(Zip(of("a", "b", "c"), of(1, 2)))
	if !slices.Equal(ks, []string{"a", "b"}) {
		t.Errorf("expected keys [a b], got %v", ks)
	}
	if !slices.Equal(vs, []int{1, 2}) {
		t.Errorf("expected values [1 2], got %v", vs)
	}
}

func TestKeysValues(t *testing.T) {
	s := Enumerate(of("a", "b", "c")) // Seq2[int, string]

	if got := slices.Collect(Keys(s)); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("Keys: expected [0 1 2], got %v", got)
	}
	if got := slices.Collect(Values(s)); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("Values: expected [a b c], got %v", got)
	}
}

func TestSwap(t *testing.T) {
	ks, vs := pairs(Swap(Enumerate(of("a", "b"))))
	if !slices.Equal(ks, []string{"a", "b"}) {
		t.Errorf("expected keys [a b], got %v", ks)
	}
	if !slices.Equal(vs, []int{0, 1}) {
		t.Errorf("expected values [0 1], got %v", vs)
	}
}

func TestConcat2(t *testing.T) {
	n := Count(Keys(Concat2(Enumerate(of("a", "b")), Enumerate(of("c")))))
	if n != 3 {
		t.Errorf("expected 3 pairs, got %d", n)
	}
}

func TestTee2(t *testing.T) {
	var seen []string
	tap := Tee2(func(_ int, v string) { seen = append(seen, v) })
	count := 0
	for range tap(Enumerate(of("a", "b", "c"))) {
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 pass-through pairs, got %d", count)
	}
	if !slices.Equal(seen, []string{"a", "b", "c"}) {
		t.Errorf("expected side effect [a b c], got %v", seen)
	}
}
