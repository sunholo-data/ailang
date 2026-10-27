package eval_harness

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// This opt-in integration is also invoked explicitly by the sprint validation
// gate. Pure Go CI does not pretend that missing native tools prove semantics.
func TestMaintenanceNativeAnalysisControl(t *testing.T) {
	root := os.Getenv("AILANG_MAINTENANCE_PACKAGE_ROOT")
	if root == "" {
		t.Skip("set AILANG_MAINTENANCE_PACKAGE_ROOT for native source integration")
	}
	bin, err := exec.LookPath("ailang")
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(root, "assets", "fixtures", "ledger")
	output := filepath.Join(t.TempDir(), "result.json")
	p, err := RunMaintenanceNativeAnalysis(context.Background(), MaintenanceAnalysisInput{
		Binary: bin, PackageRoot: root, PlanPath: filepath.Join(fixture, "plan.json"), RunsPath: filepath.Join(fixture, "economic-runs.json"), UsagePath: filepath.Join(fixture, "economic-usage.json"), OutputPath: output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.CompilerSHA256 == "" || p.SourceSHA256 == "" || p.CompilerVersion == "" {
		t.Fatalf("missing external provenance: %+v", p)
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Data struct {
			Comparison struct {
				CostDelta int `json:"costPerSuccessDeltaMicros"`
				Fresh     int `json:"freshTokenDelta"`
				Input     int `json:"freshInputTokenDelta"`
			}
		}
	}
	if err = json.Unmarshal(b, &report); err != nil {
		t.Fatal(err)
	}
	if report.Data.Comparison.CostDelta != -22500 || report.Data.Comparison.Fresh != -540 || report.Data.Comparison.Input != -550 {
		t.Fatalf("native control mismatch: %s", b)
	}
	if _, err = RunMaintenanceNativeAnalysis(context.Background(), MaintenanceAnalysisInput{Binary: bin, PackageRoot: root, PlanPath: filepath.Join(fixture, "plan.json"), RunsPath: filepath.Join(fixture, "economic-runs.json"), UsagePath: filepath.Join(fixture, "economic-usage.json"), OutputPath: output}); err == nil {
		t.Fatal("overwrote evidence")
	}
}
