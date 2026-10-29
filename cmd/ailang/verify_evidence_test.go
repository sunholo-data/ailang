package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

func TestPackageVerifyEvidenceProjection(t *testing.T) {
	r := smt.VerifyResult{Function: "f", Status: "skipped", ReasonCode: verification.UnsupportedEncoding, Reason: "unsupported", BoundedDepth: 4, Rejections: []smt.SMTRejectionReason{{Code: smt.RejectUnencodable, Message: "m", Hint: "h", Location: "x:1"}}}
	got := packageVerifyItem(r)
	if got.BoundedDepth != 4 || got.ReasonCode != r.ReasonCode || len(got.Rejections) != 1 || got.Rejections[0].Hint != "h" || got.Rejections[0].Location != "x:1" {
		t.Fatalf("projection lost evidence: %+v", got)
	}
}

func TestVerifyEvidenceHumanAvailableAndUnavailable(t *testing.T) {
	for _, e := range []verification.Evidence{{}, verification.EncodedEvidence()} {
		r := &pkg.PackageVerifyReport{Evidence: e, Package: "test/p", Modules: []pkg.ModuleVerifyReport{{Module: "m", Results: []pkg.FunctionVerifyItem{{Function: "f", Status: "verified", BoundedDepth: 3}}}}}
		out := captureStdout(t, func() { printPackageVerifyHuman(r) })
		if !strings.Contains(out, "runtime contracts") || !strings.Contains(out, "bounded: depth 3") {
			t.Fatalf("missing scope:\n%s", out)
		}
		if e.NumericModel == nil && !strings.Contains(out, "numeric model unavailable") {
			t.Fatalf("old report invented model:\n%s", out)
		}
		if e.NumericModel != nil && (!strings.Contains(out, "float=Real") || !strings.Contains(out, "not an IEEE")) {
			t.Fatalf("missing numeric disclaimer:\n%s", out)
		}
	}
}

func TestVerifyEvidenceQualityHumanRetainsPropertySkips(t *testing.T) {
	r := &pkg.QualityReport{Contracts: pkg.ContractsSection{Evidence: verification.EncodedEvidence()}, Tests: &pkg.TestsSection{Passed: 2, Skipped: 3}}
	out := captureStdout(t, func() { printQualityHuman(r) })
	if !strings.Contains(out, "3 skipped") || !strings.Contains(out, "float=Real") || !strings.Contains(out, "runtime contracts not_measured") {
		t.Fatalf("missing independent evidence:\n%s", out)
	}
}

func TestVerifyEvidenceModuleBudgetPreservesSentinelAccounting(t *testing.T) {
	r := &pkg.PackageVerifyReport{}
	r.Add(moduleBudgetReport("m", time.Second))
	if r.Skipped != 1 || r.Errors != 0 {
		t.Fatalf("legacy sentinel changed: %+v", r)
	}
	m := r.Modules[0]
	if len(m.Results) != 0 || len(m.Issues) != 1 || m.Issues[0].Scope != "module" || m.Issues[0].ReasonCode != verification.ModuleBudgetExhausted {
		t.Fatalf("fabricated function outcome: %+v", m)
	}
}

func TestVerifyEvidenceLiveReasonsAndCounts(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	source := `module evidence_fixture
export pure func identity(x: int) -> int
ensures { result == x } { x }
export pure func requiresOnly(x: int) -> int
requires { x >= 0 } { x }
export pure func noContract(x: int) -> int { x }
`
	file := filepath.Join(t.TempDir(), "evidence_fixture.ail")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true}, pipeline.Source{Code: source, Filename: file})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatal(result.Errors)
	}
	vr := smt.Verify(result.Artifacts.Core, result.Artifacts.AST, verifyModulesFromPipeline(result), smt.VerifyOptions{Timeout: time.Second})
	if vr.Verified != 1 || vr.Skipped != 1 || vr.Errors != 0 || vr.Uncontracted != 1 {
		t.Fatalf("legacy counts changed: %+v", vr)
	}
	reasons := map[string]verification.ReasonCode{}
	for _, r := range vr.Results {
		reasons[r.Function] = r.ReasonCode
	}
	if reasons["requiresOnly"] != verification.MissingPostcondition || reasons["noContract"] != verification.Uncontracted {
		t.Fatalf("emission lost code: %+v", reasons)
	}
	if vr.NumericModel == nil || vr.NumericModel.RuntimeEquivalence != "not_established" {
		t.Fatal("numeric scope absent")
	}
}

func TestVerifyEvidenceHumanDoesNotHideContractedSkipsOrErrors(t *testing.T) {
	for _, tc := range []struct {
		name            string
		skipped, errors int
		want            string
	}{
		{name: "skipped", skipped: 1, want: "0/1 verified"},
		{name: "errored", errors: 2, want: "0/2 verified"},
		{name: "only_uncontracted", want: "no contracts (3 exported)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &pkg.PackageVerifyReport{Package: "test/p"}
			r.Add(pkg.ModuleVerifyReport{Module: "m", Uncontracted: 3, Skipped: tc.skipped, Errors: tc.errors})
			out := captureStdout(t, func() { printPackageVerifyHuman(r) })
			if !strings.Contains(out, tc.want) {
				t.Fatalf("module outcome hidden; want %q:\n%s", tc.want, out)
			}
			if tc.skipped+tc.errors > 0 && strings.Contains(out, "no contracts") {
				t.Fatalf("contract-bearing functions called uncontracted:\n%s", out)
			}
			if r.Skipped != tc.skipped || r.Errors != tc.errors || r.Uncontracted != 3 {
				t.Fatal("printer changed counters")
			}
		})
	}
}
