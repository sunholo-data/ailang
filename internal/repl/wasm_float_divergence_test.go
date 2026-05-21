package repl

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWasmFloatDivergence reproduces M-WASM-TYPECHECK-FLOAT-DIVERGENCE.
//
// The native CLI type-checker accepts cognitive_commons unchanged. The WASM
// module-loading path (ModuleRegistry.LoadModule called in sequence, same as
// browser boot) rejects commons_browser.ail with "float vs int" when ANY
// float-typed helper is added to citizen.ail — even one that is never
// referenced by commons_browser.
//
// Trigger pattern in commons_browser.ail:
//
//	let sx = match score_result { Ok(s) => s.x, Err(_) => 0.0 };
//	...
//	kv("score", jo([kv("x", jnum(sx)), ...]))
//
// where s.x is float (from JudgeScore = {x: float, y: float} in persuasion).
// On CLI: sx: float. On WASM-path: sx: int → jnum(sx) fails to unify.
//
// See design_docs/planned/v0_22_0/m-wasm-typecheck-float-divergence.md
func TestWasmFloatDivergence(t *testing.T) {
	testdata := filepath.Join("..", "types", "testdata", "wasm_float_divergence")

	// loadStdlibs loads the same stdlibs in the same order as the WASM
	// loadEmbeddedStdlib path. ai/dom/cognition are needed by citizen.ail.
	loadStdlibs := func(reg *ModuleRegistry) error {
		// Order matters: dependencies first.
		for _, modName := range []string{"option", "result", "list", "math", "string", "json", "ai", "dom", "cognition", "io"} {
			path := filepath.Join("..", "..", "std", modName+".ail")
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := reg.LoadModule("std/"+modName, string(content)); err != nil {
				return err
			}
		}
		return nil
	}

	loadModuleFile := func(reg *ModuleRegistry, name, file string) error {
		content, err := os.ReadFile(filepath.Join(testdata, file))
		if err != nil {
			return err
		}
		_, err = reg.LoadModule(name, string(content))
		return err
	}

	// Loads the 4-then-commons_browser chain. citizenFile selects which
	// variant of citizen.ail to use (baseline vs with-float-helper).
	loadCognitiveCommons := func(t *testing.T, citizenFile string) error {
		t.Helper()
		reg := NewModuleRegistry()
		if err := loadStdlibs(reg); err != nil {
			t.Fatalf("stdlib load: %v", err)
		}
		// Order matches demos/scripts/wasm-loadmodule-harness.js
		seq := []struct{ name, file string }{
			{"cognitive_commons/types/personas", "personas.ail"},
			{"cognitive_commons/services/consensus", "consensus.ail"},
			{"cognitive_commons/services/persuasion", "persuasion.ail"},
			{"cognitive_commons/services/citizen", citizenFile},
			{"cognitive_commons/services/commons_browser", "commons_browser.ail"},
		}
		for _, m := range seq {
			if err := loadModuleFile(reg, m.name, m.file); err != nil {
				return err
			}
		}
		return nil
	}

	t.Run("baseline_no_float_helper_loads_clean", func(t *testing.T) {
		if err := loadCognitiveCommons(t, "citizen.ail"); err != nil {
			t.Fatalf("baseline citizen.ail should load cleanly, but got: %v", err)
		}
	})

	// Regression: prior to M-WASM-TYPECHECK-FLOAT-DIVERGENCE, adding any
	// float-typed helper to citizen.ail (here `gap_force(g: float) -> string`
	// with `if g > 0.5 ...` in the body) caused commons_browser.ail to fail
	// at `jnum(sx)` with "cannot unify type constructors: float vs <X>" where
	// X was a non-deterministic concrete type (int in browser, string in this
	// Go test). The fix propagates imported type aliases and parameter/return
	// annotations on the WASM ModuleRegistry path so declared signatures like
	// `st: CommonsState` actually constrain inference.
	t.Run("with_float_helper_now_loads_clean", func(t *testing.T) {
		if err := loadCognitiveCommons(t, "citizen_with_float_helper.ail"); err != nil {
			t.Fatalf("commons_browser should load cleanly after adding gap_force to citizen.ail, but failed: %v", err)
		}
	})
}
