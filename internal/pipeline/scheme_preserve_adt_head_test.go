package pipeline

// M-SCHEME-IMPORT-PRESERVE-ADT-HEAD (v0.22.0) structural regression test.
//
// Verifies the EXPORTED SCHEME of a function whose return type is a concrete
// ADT (e.g. Option[float]) preserves that ADT head through generalization.
// This is the structural counterpart to the behavioral tests in
// match_foreign_constructor_function_call_test.go (same package).
//
// (Test lives in internal/pipeline/ rather than internal/iface/ — the
// design doc accepted either location, but internal/iface/ can't import
// internal/pipeline due to an import cycle, and the structural assert
// needs to go through the full pipeline to see realistic schemes.)
//
// Pre-fix, std/json.getNumber's stored scheme was forall α. (Json, string) -> α
// (Option head lost). Post-fix the scheme MUST be (Json, string) -> Option[α'].
//
// Why this test in addition to the behavioral pipeline tests:
//   - Behavioral tests catch the symptom (pattern xcheck fires correctly).
//   - This structural test catches the cause directly — if a future
//     refactor breaks scheme generalization the same way again, this test
//     fails immediately with a precise diagnostic naming the corrupted
//     scheme structure, rather than producing a downstream pattern-check
//     symptom that takes investigation to localize.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestSchemeImport_PreservesADTHead_OptionReturn builds a minimal stdlib
// (Option + a Json-shaped helper returning Option[float]) and asserts the
// exported scheme of the helper preserves the Option TApp head.
//
// Test acts on Result.Interface (root module's iface) rather than walking
// the dep graph — to test getNumber's scheme directly we load the
// json-stub as the root module.
func TestSchemeImport_PreservesADTHead_OptionReturn(t *testing.T) {
	tempDir := t.TempDir()
	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("mkdir std: %v", err)
	}

	// Minimal Option (Some / None) — getNumber's return-type carrier.
	if err := os.WriteFile(filepath.Join(stdDir, "option.ail"),
		[]byte(`module std/option
export type Option[a] = Some(a) | None
`), 0644); err != nil {
		t.Fatalf("write option.ail: %v", err)
	}

	// std/json-shaped helper. The function returns Option[float] from a body
	// that pattern-matches and returns Some(...) / None — exactly the shape
	// that triggered the original bug (msg_20260520_111521_44c38751).
	jsonAil := `module std/json

import std/option (Option, Some, None)

export type Json = JNum(float) | JStr(string) | JNull

export pure func getNumber(j: Json, k: string) -> Option[float] =
  match j {
    JNum(n) => Some(n),
    _       => None
  }
`
	jsonPath := filepath.Join(stdDir, "json.ail")
	if err := os.WriteFile(jsonPath, []byte(jsonAil), 0644); err != nil {
		t.Fatalf("write json.ail: %v", err)
	}

	// Root module imports std/json and uses getNumber so the iface gets compiled.
	rootAil := `module test
import std/json (Json, getNumber, JNum)

export pure func main() -> bool =
  match getNumber(JNum(1.0), "x") {
    _ => true
  }
`
	rootPath := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(rootPath, []byte(rootAil), 0644); err != nil {
		t.Fatalf("write test.ail: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir tempDir: %v", err)
	}
	defer os.Chdir(origDir)

	src := Source{Filename: "test.ail"}
	cfg := Config{DryLink: true, RelaxModules: true}
	result, err := Run(cfg, src)
	if err != nil {
		t.Fatalf("pipeline.Run failed: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("pipeline errors: %v", result.Errors)
	}

	// Find std/json in the compiled modules and inspect its iface.
	jsonMod, ok := result.Modules["std/json"]
	if !ok {
		var keys []string
		for k := range result.Modules {
			keys = append(keys, k)
		}
		t.Fatalf("std/json not in result.Modules: keys=%v", keys)
	}
	if jsonMod.Iface == nil {
		t.Fatal("std/json LoadedModule has nil Iface; expected populated after type-checking")
	}

	item, ok := jsonMod.Iface.Exports["getNumber"]
	if !ok {
		t.Fatalf("getNumber not in std/json exports: keys=%v", keysOf(jsonMod.Iface.Exports))
	}
	if item.Type == nil {
		t.Fatal("getNumber's export item has nil Type (scheme)")
	}

	// Structural assert: scheme.Type must be TFunc2 with Return = TApp{TCon{Option}, ...}.
	// If pre-fix bug returns, scheme.Type.Return will be a *TVar2 (the over-quantified
	// fresh variable) and this test fails with the precise diagnostic.
	fn, ok := item.Type.Type.(*types.TFunc2)
	if !ok {
		t.Fatalf("getNumber scheme.Type is %T; expected *types.TFunc2", item.Type.Type)
	}
	if fn.Return == nil {
		t.Fatal("getNumber return type is nil")
	}

	tapp, ok := fn.Return.(*types.TApp)
	if !ok {
		t.Fatalf("getNumber return type is %T (%v); expected *types.TApp with Option head.\n"+
			"This is the M-SCHEME-IMPORT-PRESERVE-ADT-HEAD regression: the scheme has\n"+
			"lost its Option[float] head and degraded to forall α. ... -> α.\n"+
			"See internal/types/typechecker_functions.go (apply sub before generalize).",
			fn.Return, fn.Return)
	}

	con, ok := tapp.Constructor.(*types.TCon)
	if !ok {
		t.Fatalf("TApp constructor is %T; expected *types.TCon{Name: \"Option\"}", tapp.Constructor)
	}
	if con.Name != "Option" {
		t.Fatalf("getNumber return TApp constructor is %q; expected %q", con.Name, "Option")
	}
	if len(tapp.Args) != 1 {
		t.Fatalf("Option TApp has %d args; expected exactly 1 (the element type)", len(tapp.Args))
	}
}

func keysOf(m map[string]*iface.IfaceItem) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
