package eval

import (
	"fmt"
	"strings"
	"testing"
)

// truncateLikeCollector mirrors trace.truncateValue: the golden the bounded
// renderer must reproduce byte for byte, so the retained artifact does not
// change when the render site stops materialising the whole value first.
func truncateLikeCollector(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return fmt.Sprintf("%s…(+%d bytes elided)", s[:max], len(s)-max)
}

func boundedFixtures() map[string]Value {
	ints := make([]Value, 0, 300)
	for i := 0; i < 300; i++ {
		ints = append(ints, &IntValue{Value: i * 7})
	}
	rec := &RecordValue{Fields: map[string]Value{
		"name":  &StringValue{Value: strings.Repeat("n", 50)},
		"items": &ListValue{Elements: ints[:40]},
		"ok":    &BoolValue{Value: true},
		"f":     &FloatValue{Value: 2.5},
	}}
	m := &MapValue{Entries: map[string]*MapEntry{}}
	for i := 0; i < 30; i++ {
		k := &StringValue{Value: fmt.Sprintf("key%02d", i)}
		mk, _ := MapKey(k)
		m.Entries[mk] = &MapEntry{Key: k, Value: &TupleValue{Elements: []Value{&IntValue{Value: i}, rec}}}
	}
	cell := &RefCell{Val: rec, Init: true}
	return map[string]Value{
		"int":       &IntValue{Value: 42},
		"string":    &StringValue{Value: strings.Repeat("abc", 1000)},
		"bytes":     &BytesValue{Value: []byte(strings.Repeat("z", 100))},
		"list":      &ListValue{Elements: ints},
		"array":     &ArrayValue{Elements: ints},
		"tuple":     &TupleValue{Elements: []Value{&IntValue{Value: 1}, &StringValue{Value: "x"}, rec}},
		"record":    rec,
		"map":       m,
		"adt":       &TaggedValue{TypeName: "T", CtorName: "Node", Fields: []Value{rec, &ListValue{Elements: ints}}},
		"adt0":      &TaggedValue{TypeName: "T", CtorName: "Leaf"},
		"nested":    &ListValue{Elements: []Value{m, rec, &ListValue{Elements: []Value{&ListValue{Elements: ints}}}}},
		"indirect":  &IndirectValue{Cell: cell},
		"uninit":    &IndirectValue{Cell: &RefCell{}},
		"fn":        &FunctionValue{},
		"unit":      &UnitValue{},
		"emptylist": &ListValue{},
		"emptymap":  &MapValue{Entries: map[string]*MapEntry{}},
	}
}

func TestShowBoundedMatchesCollectorTruncation(t *testing.T) {
	for name, v := range boundedFixtures() {
		full := v.String()
		for _, budget := range []int{0, 1, 3, 16, 100, 1024, len(full), len(full) + 1, 1 << 20} {
			got := ShowBounded(v, budget)
			want := truncateLikeCollector(full, budget)
			if got != want {
				t.Errorf("%s @%d:\n got %q\nwant %q", name, budget, head(got), head(want))
			}
		}
	}
}

func head(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

func TestRenderedLenMatchesString(t *testing.T) {
	for name, v := range boundedFixtures() {
		if got, want := RenderedLen(v), len(v.String()); got != want {
			t.Errorf("%s: RenderedLen %d, len(String()) %d", name, got, want)
		}
	}
}

// The point of the renderer: a value far larger than the budget must not be
// materialised. 1M ints render to ~7 MB; at a 64-byte budget the allocation
// must stay in the low kilobytes (the prefix, the marker, and per-int scratch
// during the counting pass), not megabytes.
func TestShowBoundedDoesNotMaterialiseLargeValues(t *testing.T) {
	elems := make([]Value, 1_000_000)
	for i := range elems {
		elems[i] = &IntValue{Value: i}
	}
	big := &ListValue{Elements: elems}
	full := big.String()
	want := truncateLikeCollector(full, 64)

	bytesPerRun := testing.AllocsPerRun(1, func() {
		if got := ShowBounded(big, 64); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	_ = bytesPerRun
	var got string
	res := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			got = ShowBounded(big, 64)
		}
	})
	if got != want {
		t.Fatalf("mismatch")
	}
	if perOp := res.AllocedBytesPerOp(); perOp > 64<<10 {
		t.Fatalf("ShowBounded allocated %d bytes/op for a %d-byte value at budget 64; want < 64 KiB", perOp, len(full))
	}
}

func TestShowBoundedUnboundedDelegates(t *testing.T) {
	v := &ListValue{Elements: []Value{&IntValue{Value: 1}, &StringValue{Value: "two"}}}
	if got := ShowBounded(v, 0); got != v.String() {
		t.Fatalf("budget 0 must be unbounded: %q", got)
	}
	if got := ShowBounded(v, -1); got != v.String() {
		t.Fatalf("negative budget must be unbounded: %q", got)
	}
}
