package pcoll

import (
	"errors"
	"math"
	"sort"
	"strings"
	"testing"
)

func key(t *testing.T, build func(w *KeyWriter)) string {
	t.Helper()
	var w KeyWriter
	build(&w)
	k, err := w.Key()
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	return k
}

func keyErr(build func(w *KeyWriter)) error {
	var w KeyWriter
	build(&w)
	_, err := w.Key()
	return err
}

// The v0.11.0 encodings are frozen byte for byte (Decision D2): every map
// that runs today keeps its keys, its iteration order and its show output.
func TestLegacyAtomsByteIdentical(t *testing.T) {
	cases := []struct {
		name  string
		build func(w *KeyWriter)
		want  string
	}{
		{"int", func(w *KeyWriter) { w.Int(42) }, "i:42"},
		{"negative int", func(w *KeyWriter) { w.Int(-7) }, "i:-7"},
		{"min int", func(w *KeyWriter) { w.Int(math.MinInt64) }, "i:-9223372036854775808"},
		{"string", func(w *KeyWriter) { w.Str("hello") }, "s:hello"},
		{"empty string", func(w *KeyWriter) { w.Str("") }, "s:"},
		{"string with NUL stays raw at top level", func(w *KeyWriter) { w.Str("a\x00b") }, "s:a\x00b"},
		{"true", func(w *KeyWriter) { w.Bool(true) }, "b:true"},
		{"false", func(w *KeyWriter) { w.Bool(false) }, "b:false"},
	}
	for _, c := range cases {
		if got := key(t, c.build); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestNewKindsGolden(t *testing.T) {
	cases := []struct {
		name  string
		build func(w *KeyWriter)
		want  string
	}{
		{"unit", func(w *KeyWriter) { w.Unit() }, "u:"},
		{"float 1.5", func(w *KeyWriter) { w.Float(1.5) }, "f:bff8000000000000"},
		{"float -1.5", func(w *KeyWriter) { w.Float(-1.5) }, "f:4007ffffffffffff"},
		{"float 0", func(w *KeyWriter) { w.Float(0) }, "f:8000000000000000"},
		{"tuple (1, 2)", func(w *KeyWriter) { w.BeginTuple(); w.Int(1); w.Int(2); w.End() }, "t:i:1\x00i:2\x00"},
		{"empty tuple", func(w *KeyWriter) { w.BeginTuple(); w.End() }, "t:"},
		{"tuple with NUL string", func(w *KeyWriter) { w.BeginTuple(); w.Str("a\x00"); w.End() }, "t:s:a\x00\xff\x00"},
		{"record {x: 1, y: true}", func(w *KeyWriter) {
			w.BeginRecord()
			w.Field("x")
			w.Int(1)
			w.Field("y")
			w.Bool(true)
			w.End()
		}, "r:x\x00i:1\x00y\x00b:true\x00"},
		{"nullary ctor", func(w *KeyWriter) { w.BeginCtor("North"); w.End() }, "c:North\x00"},
		{"ctor with field", func(w *KeyWriter) { w.BeginCtor("Some"); w.Int(3); w.End() }, "c:Some\x00i:3\x00"},
		{"nested tuple", func(w *KeyWriter) {
			w.BeginTuple()
			w.BeginTuple()
			w.Int(1)
			w.End()
			w.Int(2)
			w.End()
		}, "t:t:i:1\x00\xff\x00i:2\x00"},
	}
	for _, c := range cases {
		if got := key(t, c.build); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFloatNaNRejectedAndNegativeZeroFolded(t *testing.T) {
	err := keyErr(func(w *KeyWriter) { w.Float(math.NaN()) })
	if err == nil || !strings.Contains(err.Error(), "NaN") {
		t.Fatalf("NaN key: want error naming NaN, got %v", err)
	}
	// NaN nested inside a tuple fails the whole key, not just the element.
	err = keyErr(func(w *KeyWriter) { w.BeginTuple(); w.Int(1); w.Float(math.NaN()); w.End() })
	if err == nil {
		t.Fatal("nested NaN key: want error")
	}
	pos := key(t, func(w *KeyWriter) { w.Float(0) })
	neg := key(t, func(w *KeyWriter) { w.Float(math.Copysign(0, -1)) })
	if pos != neg {
		t.Fatalf("-0.0 and +0.0 compare equal and must share a key: %q vs %q", neg, pos)
	}
}

// Float keys sort numerically, unlike legacy int keys (which stay lexical).
func TestFloatKeysSortNumerically(t *testing.T) {
	xs := []float64{3.5, -1, math.Inf(1), 0, -math.MaxFloat64, 1e-300, -2.25, 100, math.Inf(-1)}
	keys := make([]string, len(xs))
	byKey := map[string]float64{}
	for i, x := range xs {
		keys[i] = key(t, func(w *KeyWriter) { w.Float(x) })
		byKey[keys[i]] = x
	}
	sort.Strings(keys)
	for i := 1; i < len(keys); i++ {
		if !(byKey[keys[i-1]] < byKey[keys[i]]) {
			t.Fatalf("float key order not numeric: %v before %v", byKey[keys[i-1]], byKey[keys[i]])
		}
	}
}

// Composite keys sort element by element, each element ordered exactly as it
// would be as a top-level key (so int elements are lexical, like int keys).
func TestTupleOrderIsElementwise(t *testing.T) {
	type pair struct {
		a int64
		b string
	}
	ps := []pair{{2, "a"}, {10, "z"}, {1, "b"}, {1, "a\x00"}, {1, "a"}, {1, ""}, {-3, "q"}}
	enc := func(p pair) string {
		return key(t, func(w *KeyWriter) { w.BeginTuple(); w.Int(p.a); w.Str(p.b); w.End() })
	}
	atom := func(p pair) (string, string) {
		return key(t, func(w *KeyWriter) { w.Int(p.a) }), key(t, func(w *KeyWriter) { w.Str(p.b) })
	}
	sorted := append([]pair(nil), ps...)
	sort.Slice(sorted, func(i, j int) bool { return enc(sorted[i]) < enc(sorted[j]) })
	for i := 1; i < len(sorted); i++ {
		a1, b1 := atom(sorted[i-1])
		a2, b2 := atom(sorted[i])
		if a1 > a2 || (a1 == a2 && b1 >= b2) {
			t.Fatalf("tuple order not elementwise: %v before %v", sorted[i-1], sorted[i])
		}
	}
}

// Distinct values must never share a key. These are the shapes a naive
// separator grammar collides on.
func TestInjectiveOnTrickyShapes(t *testing.T) {
	shapes := map[string]func(w *KeyWriter){
		`("a", "b")`:   func(w *KeyWriter) { w.BeginTuple(); w.Str("a"); w.Str("b"); w.End() },
		`("a\0", "b")`: func(w *KeyWriter) { w.BeginTuple(); w.Str("a\x00"); w.Str("b"); w.End() },
		`("a\0s:b")`:   func(w *KeyWriter) { w.BeginTuple(); w.Str("a\x00s:b"); w.End() },
		`("a\0\xffb")`: func(w *KeyWriter) { w.BeginTuple(); w.Str("a\x00\xffb"); w.End() },
		`(("a"), "b")`: func(w *KeyWriter) {
			w.BeginTuple()
			w.BeginTuple()
			w.Str("a")
			w.End()
			w.Str("b")
			w.End()
		},
		`(("a", "b"))`: func(w *KeyWriter) {
			w.BeginTuple()
			w.BeginTuple()
			w.Str("a")
			w.Str("b")
			w.End()
			w.End()
		},
		`()`:         func(w *KeyWriter) { w.BeginTuple(); w.End() },
		`unit`:       func(w *KeyWriter) { w.Unit() },
		`"t:"`:       func(w *KeyWriter) { w.Str("t:") },
		`A`:          func(w *KeyWriter) { w.BeginCtor("A"); w.End() },
		`A(())`:      func(w *KeyWriter) { w.BeginCtor("A"); w.Unit(); w.End() },
		`{a: ()}`:    func(w *KeyWriter) { w.BeginRecord(); w.Field("a"); w.Unit(); w.End() },
		`{}`:         func(w *KeyWriter) { w.BeginRecord(); w.End() },
		`(1)`:        func(w *KeyWriter) { w.BeginTuple(); w.Int(1); w.End() },
		`1`:          func(w *KeyWriter) { w.Int(1) },
		`1.0`:        func(w *KeyWriter) { w.Float(1) },
		`"i:1"`:      func(w *KeyWriter) { w.Str("i:1") },
		`(true)`:     func(w *KeyWriter) { w.BeginTuple(); w.Bool(true); w.End() },
		`("b:true")`: func(w *KeyWriter) { w.BeginTuple(); w.Str("b:true"); w.End() },
	}
	seen := map[string]string{}
	for name, build := range shapes {
		k := key(t, build)
		if other, dup := seen[k]; dup {
			t.Fatalf("collision: %s and %s both encode to %q", name, other, k)
		}
		seen[k] = name
	}
}

func TestRecordFieldsMustBeSortedAndPaired(t *testing.T) {
	if err := keyErr(func(w *KeyWriter) {
		w.BeginRecord()
		w.Field("y")
		w.Int(1)
		w.Field("x")
		w.Int(2)
		w.End()
	}); err == nil {
		t.Fatal("unsorted record fields: want error")
	}
	if err := keyErr(func(w *KeyWriter) { w.BeginRecord(); w.Int(1); w.End() }); err == nil {
		t.Fatal("record value without field name: want error")
	}
	if err := keyErr(func(w *KeyWriter) { w.BeginRecord(); w.Field("x"); w.End() }); err == nil {
		t.Fatal("record field without value: want error")
	}
}

func TestUnsupportedNamesTheType(t *testing.T) {
	err := keyErr(func(w *KeyWriter) { w.BeginTuple(); w.Int(1); w.Unsupported("function"); w.End() })
	var uk *UnsupportedKeyError
	if !errors.As(err, &uk) || uk.Kind != "function" {
		t.Fatalf("want UnsupportedKeyError{function}, got %v", err)
	}
	if !strings.Contains(err.Error(), "unsupported map key type: function") {
		t.Fatalf("message must name the type: %v", err)
	}
}

func TestMisuseIsAnError(t *testing.T) {
	for name, build := range map[string]func(w *KeyWriter){
		"empty":       func(w *KeyWriter) {},
		"unclosed":    func(w *KeyWriter) { w.BeginTuple(); w.Int(1) },
		"extra End":   func(w *KeyWriter) { w.Int(1); w.End() },
		"two values":  func(w *KeyWriter) { w.Int(1); w.Int(2) },
		"stray Field": func(w *KeyWriter) { w.BeginTuple(); w.Field("x"); w.End() },
	} {
		if keyErr(build) == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
