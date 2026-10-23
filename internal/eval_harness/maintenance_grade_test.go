package eval_harness

import (
	"context"
	"strings"
	"testing"
)

func TestMaintenanceNativeGraderRequiresAdmission(t *testing.T) {
	e := &MaintenanceExecution{}
	_, err := NewMaintenanceNativeGrader(e, MaintenanceNativeGradeOptions{EvidenceRoot: t.TempDir(), CommandPrefix: []string{"/trusted/wrapper"}, ContainmentDigest: strings.Repeat("a", 64)})
	if err == nil {
		t.Fatal("self-declared digest admitted protected grading")
	}
}

func TestMaintenanceGradeProtocolRejectsSkippedAssertions(t *testing.T) {
	n := 1
	h := strings.Repeat("a", 64)
	r := maintenanceGradeProtocol{Schema: "maintenance.candidate-grade/1", Family: "text-fidelity", Language: "python", Stage: 1, Status: "passed", Compiled: true, SourceDigest: h, CandidateDigestAfter: &h, RuntimeDigest: h, OracleDigest: h, ProtocolDigest: h, Cases: []maintenanceGradeCase{{ID: "s1", InputDigest: h, OutputDigest: h, ExpectedAssertions: 8, Assertions: &n, Passed: true, MismatchIDs: []string{}}}}
	if _, err := maintenanceGradeVerdict(r, MaintenanceTrajectoryPlan{Family: "text-fidelity", Language: "python"}, MaintenanceStagePlan{ID: "S1", OracleDigest: h}, MaintenanceCheckpoint{Digest: h}); err == nil {
		t.Fatal("accepted skipped assertions")
	}
}

func TestMaintenanceGradeProtocolClassifiesArtifactFailures(t *testing.T) {
	h := strings.Repeat("a", 64)
	r := maintenanceGradeProtocol{Schema: "maintenance.candidate-grade/1", Family: "text-fidelity", Language: "python", Stage: 1, Status: "artifact_failure", SourceDigest: h, Failure: "native_compile_failed"}
	g, err := maintenanceGradeVerdict(r, MaintenanceTrajectoryPlan{Family: "text-fidelity", Language: "python"}, MaintenanceStagePlan{ID: "S1", OracleDigest: h}, MaintenanceCheckpoint{Digest: h})
	if err != nil || g.Visible != "fail" || g.Oracle != "not_run" || g.OracleDigest != "" {
		t.Fatalf("compile error became oracle mutant or setup error: %+v %v", g, err)
	}
}

func TestMaintenanceNativeGraderRejectsConflictingMode(t *testing.T) {
	_, err := NewMaintenanceNativeGrader(&MaintenanceExecution{}, MaintenanceNativeGradeOptions{EvidenceRoot: t.TempDir(), DevelopmentUnconfined: true, Admission: func(context.Context) error { return nil }})
	if err == nil {
		t.Fatal("development mode silently accepted protected admission")
	}
}

func TestMaintenanceGradeRejectsImplicitZeroOrMissingEvidence(t *testing.T) {
	for _, raw := range []string{`{"schema":"maintenance.candidate-grade/1"}`, `{"schema":"maintenance.candidate-grade/1","family":"text-fidelity","language":"python","stage":1,"status":"artifact_failure","compiled":null,"cases":[],"publicCheckPassed":null,"sourceDigest":null,"candidateDigestAfter":null,"containment":{},"modelRun":false}`} {
		if err := maintenanceGradeRequired([]byte(raw)); err == nil {
			t.Fatal("missing/null measurement was accepted")
		}
	}
}

func TestMaintenanceGradeNeverInfersFailurePhaseFromLanguage(t *testing.T) {
	h := strings.Repeat("a", 64)
	for _, language := range []string{"ailang", "python"} {
		for _, reason := range []string{"native_compile_failed", "candidate_timeout", "candidate_process_failed", "invalid_candidate_output"} {
			r := maintenanceGradeProtocol{Schema: "maintenance.candidate-grade/1", Family: "text-fidelity", Language: language, Stage: 1, Status: "artifact_failure", SourceDigest: h, Failure: reason}
			got, err := maintenanceGradeVerdict(r, MaintenanceTrajectoryPlan{Family: "text-fidelity", Language: language}, MaintenanceStagePlan{ID: "S1", OracleDigest: h}, MaintenanceCheckpoint{Digest: h})
			if err != nil {
				t.Fatal(err)
			}
			if len(got.CheckKinds) != 1 || got.CheckKinds[0] != "public" {
				t.Fatalf("%s/%s fabricated diagnostic phase: %v", language, reason, got.CheckKinds)
			}
		}
	}
}
