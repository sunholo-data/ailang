package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseFuncEffects is a small helper that parses a single-function input and
// returns the effect annotations on the (sole) function declaration. It also
// returns the parser's accumulated errors so tests can assert on structured
// error codes.
func parseFuncEffects(t *testing.T, input string) ([]ast.EffectAnnotation, []*ParserError) {
	t.Helper()
	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	var perrs []*ParserError
	for _, err := range p.Errors() {
		if pe, ok := err.(*ParserError); ok {
			perrs = append(perrs, pe)
		}
	}

	if program == nil {
		return nil, perrs
	}
	if program.File == nil {
		return nil, perrs
	}
	if len(program.File.Funcs) == 0 || program.File.Funcs[0] == nil {
		return nil, perrs
	}
	return program.File.Funcs[0].Effects, perrs
}

// hasErrorCode returns true if any of the parser errors has the given code.
func hasErrorCode(errs []*ParserError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

// =========================================================================
// Positive cases
// =========================================================================

func TestEffectParams_SinglePair(t *testing.T) {
	effects, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os]} { 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	if effects[0].Name != "Rand" {
		t.Errorf("effect name: want Rand, got %s", effects[0].Name)
	}
	if len(effects[0].Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(effects[0].Params))
	}
	if effects[0].Params[0].Key != "mode" || effects[0].Params[0].Value != "os" {
		t.Errorf("param: want mode=os, got %s=%s",
			effects[0].Params[0].Key, effects[0].Params[0].Value)
	}
}

func TestEffectParams_MultiplePairs(t *testing.T) {
	effects, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os, scope=identity]} { 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	if len(effects[0].Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(effects[0].Params))
	}
	// Params preserved in source order; sorting happens at print time.
	if effects[0].Params[0].Key != "mode" || effects[0].Params[0].Value != "os" {
		t.Errorf("param[0]: want mode=os, got %s=%s",
			effects[0].Params[0].Key, effects[0].Params[0].Value)
	}
	if effects[0].Params[1].Key != "scope" || effects[0].Params[1].Value != "identity" {
		t.Errorf("param[1]: want scope=identity, got %s=%s",
			effects[0].Params[1].Key, effects[0].Params[1].Value)
	}
}

func TestEffectParams_WithBudget(t *testing.T) {
	effects, errs := parseFuncEffects(t, "func f() -> int ! {AI[mode=routeable] @limit=10} { 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	if effects[0].Name != "AI" {
		t.Errorf("effect name: want AI, got %s", effects[0].Name)
	}
	if len(effects[0].Params) != 1 || effects[0].Params[0].Key != "mode" || effects[0].Params[0].Value != "routeable" {
		t.Errorf("params: want [mode=routeable], got %+v", effects[0].Params)
	}
	if effects[0].Budget == nil || *effects[0].Budget != 10 {
		t.Errorf("budget: want 10, got %v", effects[0].Budget)
	}
}

func TestEffectParams_MixedParameterisedAndBare(t *testing.T) {
	effects, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os], FS} { 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(effects))
	}
	if effects[0].Name != "Rand" || len(effects[0].Params) != 1 {
		t.Errorf("effect[0]: want Rand[mode=os], got %s%+v",
			effects[0].Name, effects[0].Params)
	}
	if effects[1].Name != "FS" || len(effects[1].Params) != 0 {
		t.Errorf("effect[1]: want bare FS, got %s%+v",
			effects[1].Name, effects[1].Params)
	}
}

func TestEffectParams_MultipleParameterisedEffects(t *testing.T) {
	effects, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os], Net[mode=live]} { 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(effects))
	}
	if effects[0].Name != "Rand" || effects[0].Params[0].Value != "os" {
		t.Errorf("effect[0]: want Rand[mode=os], got %s%+v",
			effects[0].Name, effects[0].Params)
	}
	if effects[1].Name != "Net" || effects[1].Params[0].Value != "live" {
		t.Errorf("effect[1]: want Net[mode=live], got %s%+v",
			effects[1].Name, effects[1].Params)
	}
}

// =========================================================================
// Error cases
// =========================================================================

func TestEffectParams_ErrorMissingValue(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for missing value, got none")
	}
	if !hasErrorCode(errs, "PAR_EFF013_PARAM_VAL") {
		t.Errorf("expected PAR_EFF013_PARAM_VAL, got: %v", errs)
	}
}

func TestEffectParams_ErrorMissingKey(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[=os]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for missing key, got none")
	}
	if !hasErrorCode(errs, "PAR_EFF011_PARAM_KEY") {
		t.Errorf("expected PAR_EFF011_PARAM_KEY, got: %v", errs)
	}
}

func TestEffectParams_ErrorMissingEquals(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode os]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for missing '=', got none")
	}
	if !hasErrorCode(errs, "PAR_EFF012_PARAM_EQ") {
		t.Errorf("expected PAR_EFF012_PARAM_EQ, got: %v", errs)
	}
}

func TestEffectParams_ErrorWrongSeparator(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode:os]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for ':' instead of '=', got none")
	}
	if !hasErrorCode(errs, "PAR_EFF012_PARAM_EQ") {
		t.Errorf("expected PAR_EFF012_PARAM_EQ, got: %v", errs)
	}
}

func TestEffectParams_ErrorEmptyBracket(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for empty params, got none")
	}
	if !hasErrorCode(errs, "PAR_EFF010_EMPTY_PARAMS") {
		t.Errorf("expected PAR_EFF010_EMPTY_PARAMS, got: %v", errs)
	}
}

func TestEffectParams_ErrorDuplicateKey(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os, mode=crypto]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate key, got none")
	}
	if !hasErrorCode(errs, "PAR_EFF014_PARAM_DUP") {
		t.Errorf("expected PAR_EFF014_PARAM_DUP, got: %v", errs)
	}
}

func TestEffectParams_ErrorMissingComma(t *testing.T) {
	_, errs := parseFuncEffects(t, "func f() -> int ! {Rand[mode=os scope=id]} { 42 }")
	if len(errs) == 0 {
		t.Fatal("expected error for missing comma, got none")
	}
	// We accept either PAR_UNEXPECTED_TOKEN (from reportExpected on COMMA) or
	// the explicit PAR_EFF012/013 path that some recovery paths might take.
	saw := false
	for _, e := range errs {
		if e.Code == "PAR_UNEXPECTED_TOKEN" || e.Code == "PAR_EFF012_PARAM_EQ" || e.Code == "PAR_EFF013_PARAM_VAL" {
			saw = true
			break
		}
	}
	if !saw {
		t.Errorf("expected an error indicating a missing comma, got: %v", errs)
	}
}

// =========================================================================
// Round-trip
// =========================================================================

// TestEffectParams_RoundTrip parses an effect annotation, calls .String(),
// reparses, and asserts equality of the structural shape.
func TestEffectParams_RoundTrip(t *testing.T) {
	cases := []string{
		`func f() -> int ! {Rand[mode=os]} { 0 }`,
		`func f() -> int ! {Rand[mode=os, scope=identity]} { 0 }`,
		`func f() -> int ! {AI[mode=routeable] @limit=10} { 0 }`,
		`func f() -> int ! {Rand[mode=os], FS} { 0 }`,
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			first, errs := parseFuncEffects(t, src)
			if len(errs) > 0 {
				t.Fatalf("first parse errors: %v", errs)
			}
			// Format: rebuild a function source from the parsed effects' .String()
			formatted := "func f() -> int " + ast.FormatEffects(first) + " { 0 }"

			second, errs2 := parseFuncEffects(t, formatted)
			if len(errs2) > 0 {
				t.Fatalf("reparse errors for %q: %v", formatted, errs2)
			}
			if len(first) != len(second) {
				t.Fatalf("effect count differs after round-trip: first=%d, second=%d (%q)",
					len(first), len(second), formatted)
			}
			for i := range first {
				if first[i].Name != second[i].Name {
					t.Errorf("effect[%d] name differs: %s vs %s", i, first[i].Name, second[i].Name)
				}
				// Compare params: alphabetical-by-key, since the printer sorts.
				fp := sortedParamMap(first[i].Params)
				sp := sortedParamMap(second[i].Params)
				if len(fp) != len(sp) {
					t.Errorf("effect[%d] param count differs: %d vs %d", i, len(fp), len(sp))
					continue
				}
				for k, v := range fp {
					if sp[k] != v {
						t.Errorf("effect[%d] param %q: %q vs %q", i, k, v, sp[k])
					}
				}
				// Budget round-trip
				switch {
				case first[i].Budget == nil && second[i].Budget != nil,
					first[i].Budget != nil && second[i].Budget == nil:
					t.Errorf("effect[%d] budget presence differs", i)
				case first[i].Budget != nil && second[i].Budget != nil && *first[i].Budget != *second[i].Budget:
					t.Errorf("effect[%d] budget value differs: %d vs %d",
						i, *first[i].Budget, *second[i].Budget)
				}
			}
		})
	}
}

func sortedParamMap(ps []ast.EffectParam) map[string]string {
	m := make(map[string]string, len(ps))
	for _, p := range ps {
		m[p.Key] = p.Value
	}
	return m
}
