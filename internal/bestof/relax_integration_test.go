package bestof

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestRelaxModulesContractSelection runs the REAL `ailang` binary: ephemeral candidates
// (module decl ≠ temp path) must not fail MOD010 when RelaxModules is set, and the contract
// tier must reject a runs-but-WRONG candidate. Skips when no fresh ailang exists (CI w/o build).
func TestRelaxModulesContractSelection(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	bad := filepath.Join("testdata", "contract_demo", "cand_bad.ail")
	good := filepath.Join("testdata", "contract_demo", "cand_good.ail")
	v := AilangVerifier{Bin: bin, Caps: "IO", VerifyContracts: true, RelaxModules: true}

	if vb := v.Verify(bad); !vb.Runs {
		t.Fatalf("cand_bad should RUN under --relax-modules, got %+v", vb)
	} else if vb.ContractsPass {
		t.Errorf("cand_bad VIOLATES its ensures; ContractsPass must be false, got %+v", vb)
	}
	if vg := v.Verify(good); !vg.Runs || !vg.ContractsPass {
		t.Errorf("cand_good should run + satisfy contracts, got %+v", vg)
	}
	// bad listed first: plain selector keeps it (runs); contract tier flips to good (idx 1).
	if best, _ := SelectBest([]string{bad, good}, v); best != 1 {
		t.Errorf("contract selector must pick cand_good (idx 1), got %d", best)
	}
}

// TestRelaxModulesGuardsMOD010 documents the bug the fix closes: WITHOUT RelaxModules the
// ephemeral candidate fails typecheck (MOD010) → "neither"; WITH it, the candidate runs.
func TestRelaxModulesGuardsMOD010(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	good := filepath.Join("testdata", "contract_demo", "cand_good.ail")
	if v := (AilangVerifier{Bin: bin, Caps: "IO"}).Verify(good); v.TypeChecks {
		t.Skip("MOD010 not enforced in this environment; relax test still covers the fix")
	}
	if v := (AilangVerifier{Bin: bin, Caps: "IO", RelaxModules: true}).Verify(good); !v.Runs {
		t.Errorf("with RelaxModules the candidate must run, got %+v", v)
	}
}
