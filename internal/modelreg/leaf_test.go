package modelreg_test

import (
	"os/exec"
	"strings"
	"testing"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M1 (decision D4(a), 2026-08-27).
//
// modelreg is a LEAF. It must never import internal/executor.
//
// This is not a style preference, it is the reason the package exists.
// internal/eval_harness imports internal/executor for ten symbols (Task,
// Result, EventHandler, the cost classes, ValidateTaskCapabilities), so the
// model registry could not stay in eval_harness: the executors have to resolve
// roles through it (M6/M7), and executor -> eval_harness -> executor is a cycle
// that does not compile.
//
// Once M6/M7 land, the Go compiler enforces this on its own — an import cycle
// is a build error. Until then nothing does, which is exactly the window this
// test covers. It stays afterwards because it names WHY, and a build error
// alone does not.
func TestModelregIsALeaf(t *testing.T) {
	const forbidden = "github.com/sunholo-data/ailang/internal/executor"

	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: prove the instrument resolved a REAL build before trusting an
	// absence. A first cut asserted "some dep starts with the module path" and
	// passed vacuously on an empty package — `go list -deps .` always lists the
	// package ITSELF, so that check could never fail. modelreg is a leaf and may
	// legitimately have no ailang deps at all, so the honest control is a
	// third-party dependency the registry genuinely has.
	const mustSee = "gopkg.in/yaml.v3"
	sawControl := false
	for _, d := range deps {
		if d == mustSee {
			sawControl = true
			break
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but %s absent; the registry "+
			"parses YAML, so this build did not resolve and the absence assertion below "+
			"would pass vacuously", len(deps), mustSee)
	}

	for _, d := range deps {
		if d == forbidden || strings.HasPrefix(d, forbidden+"/") {
			t.Errorf("modelreg must be a leaf but depends on %s.\n"+
				"This closes an import cycle: executor -> modelreg (M6/M7) plus "+
				"modelreg -> executor. Move the needed symbol DOWN into modelreg "+
				"instead of importing up.", d)
		}
	}
}
