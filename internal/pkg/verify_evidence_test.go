package pkg

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

func TestVerifyEvidenceLegacyAndNewReaders(t *testing.T) {
	old := []byte(`{"schema":"ailang.package-verify/v1","package":"test/p","version":"1","verified":2,"total":4,"skipped":2,"errors":0,"counterexample":0,"uncontracted":3,"modules":[]}`)
	r, err := DecodePackageVerifyReport(old)
	if err != nil {
		t.Fatal(err)
	}
	if r.NumericModel != nil || r.ReasonCounts != nil {
		t.Fatal("old report invented numeric/reason evidence")
	}
	r.Evidence = verification.EncodedEvidence()
	r.ReasonCounts = map[verification.ReasonCode]int{verification.MissingPostcondition: 1, verification.UnsupportedEncoding: 1}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// An old consumer ignores details but must retain exactly the original counts.
	var legacy struct{ Verified, Total, Skipped, Errors, Counterexample, Uncontracted int }
	if err = json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Verified != 2 || legacy.Total != 4 || legacy.Skipped != 2 || legacy.Errors != 0 || legacy.Uncontracted != 3 {
		t.Fatalf("legacy counts changed: %+v", legacy)
	}
	got, err := DecodePackageVerifyReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.NumericModel == nil || got.NumericModel.FloatSort != "Real" || got.NumericModel.IntSort != "Int" || got.NumericModel.RuntimeEquivalence != "not_established" {
		t.Fatalf("numeric scope lost: %+v", got)
	}
}

func TestQualityVerifyEvidenceProjectionPreservesScope(t *testing.T) {
	in := cleanInputs()
	in.Verify.Evidence = verification.EncodedEvidence()
	in.Verify.ReasonCounts = map[verification.ReasonCode]int{verification.MissingPostcondition: 1}
	in.Verify.Modules = []ModuleVerifyReport{{Module: "m", Issues: []verification.ModuleIssue{{Scope: "module", ReasonCode: verification.ModuleBudgetExhausted, Reason: "wall cap"}}}}
	in.Attested = &AttestedBlock{AttestedBy: "test", Tests: &TestsSection{Passed: 2, Skipped: 1}}
	r := BuildQualityReport(qualityManifest("experimental", nil), ModePublisher, in, false)
	if r.Contracts.NumericModel == nil || r.Contracts.ReasonCounts[verification.MissingPostcondition] != 1 {
		t.Fatalf("scope/reasons dropped: %+v", r.Contracts)
	}
	if len(r.Contracts.ModuleIssues) != 1 || r.Contracts.ModuleIssues[0].Scope != "module" {
		t.Fatalf("module budget lost: %+v", r.Contracts)
	}
	if r.Tests == nil || r.Tests.Skipped != 1 {
		t.Fatal("property skip not retained")
	}
}
