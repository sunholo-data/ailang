package ast

import "testing"

// TestEffectAnnotation_BareString verifies that an effect with no params
// preserves the legacy bare form.
func TestEffectAnnotation_BareString(t *testing.T) {
	e := EffectAnnotation{Name: "Rand"}
	got := e.String()
	want := "Rand"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_SingleParam verifies single-param formatting.
func TestEffectAnnotation_SingleParam(t *testing.T) {
	e := EffectAnnotation{
		Name:   "Rand",
		Params: []EffectParam{{Key: "mode", Value: "os"}},
	}
	got := e.String()
	want := "Rand[mode=os]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_MultipleParamsSorted verifies that multiple params
// emit in alphabetical-by-key order regardless of insertion order.
func TestEffectAnnotation_MultipleParamsSorted(t *testing.T) {
	e := EffectAnnotation{
		Name: "Rand",
		Params: []EffectParam{
			{Key: "scope", Value: "id"},
			{Key: "mode", Value: "os"},
		},
	}
	got := e.String()
	want := "Rand[mode=os, scope=id]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_ParamsWithBudget verifies params combine with @limit.
func TestEffectAnnotation_ParamsWithBudget(t *testing.T) {
	five := 5
	e := EffectAnnotation{
		Name:   "AI",
		Params: []EffectParam{{Key: "mode", Value: "routeable"}},
		Budget: &five,
	}
	got := e.String()
	want := "AI[mode=routeable] @limit=5"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_ParamsWithMin verifies params combine with @min.
func TestEffectAnnotation_ParamsWithMin(t *testing.T) {
	one := 1
	e := EffectAnnotation{
		Name:   "Rand",
		Params: []EffectParam{{Key: "mode", Value: "crypto"}},
		Min:    &one,
	}
	got := e.String()
	want := "Rand[mode=crypto] @min=1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_EmptyParamsSliceTreatedAsBare verifies that an empty
// (non-nil) params slice still produces the bare form.
func TestEffectAnnotation_EmptyParamsSliceTreatedAsBare(t *testing.T) {
	e := EffectAnnotation{
		Name:   "Rand",
		Params: []EffectParam{},
	}
	got := e.String()
	want := "Rand"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEffectAnnotation_DeterministicParamOrdering runs the formatter
// repeatedly to verify alphabetical ordering is stable across iterations.
// (We sort a slice, not a map, so map-iteration nondeterminism doesn't apply
// directly — but this guards against regressions if the implementation
// changes.)
func TestEffectAnnotation_DeterministicParamOrdering(t *testing.T) {
	for i := 0; i < 20; i++ {
		e := EffectAnnotation{
			Name: "Rand",
			Params: []EffectParam{
				{Key: "scope", Value: "id"},
				{Key: "mode", Value: "os"},
			},
		}
		got := e.String()
		want := "Rand[mode=os, scope=id]"
		if got != want {
			t.Errorf("iter %d: got %q, want %q", i, got, want)
		}
	}
}

// TestEffectAnnotation_StringDoesNotMutateParams verifies that calling
// String() does not reorder the caller's Params slice (we sort a copy).
func TestEffectAnnotation_StringDoesNotMutateParams(t *testing.T) {
	e := EffectAnnotation{
		Name: "Rand",
		Params: []EffectParam{
			{Key: "scope", Value: "id"},
			{Key: "mode", Value: "os"},
		},
	}
	_ = e.String()
	if e.Params[0].Key != "scope" || e.Params[1].Key != "mode" {
		t.Errorf("String() mutated input params; got order [%s, %s]",
			e.Params[0].Key, e.Params[1].Key)
	}
}

// TestFormatEffects_WithParams verifies the top-level FormatEffects helper
// renders parameterised effects inline.
func TestFormatEffects_WithParams(t *testing.T) {
	effects := []EffectAnnotation{
		{
			Name:   "Rand",
			Params: []EffectParam{{Key: "mode", Value: "os"}},
		},
		{Name: "FS"},
	}
	got := FormatEffects(effects)
	want := "! {Rand[mode=os], FS}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
