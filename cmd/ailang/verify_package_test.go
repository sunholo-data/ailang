package main

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
)

// M-PKG-QUALITY-LADDER M1: `ailang verify --package` on a FLAT package with an
// intra-package import verifies its contracts and emits the shared
// PackageVerifyReport shape. Before M1 the registry ran verify per file and
// died on MOD010 for every published package (0 contracts counted, 2026-09-17).
//
// Kills: (a) RelaxModules=false in verifyOneModule (MOD010 → compile error,
// Verified drops to 0); (b) emitting a bare results array instead of the
// report object (DecodePackageVerifyReport refuses it).
func TestVerifyPackage_FlatLayoutCountsContracts(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	bin := buildAilang(t)
	dir := filepath.Join("internal", "pkg", "testdata", "quality", "flat_self_import") // runAilangBin runs from the project root

	out, stderr, exit := runAilangBin(t, bin, "verify", "--package", dir, "--json", "--timeout", "5s")
	if exit != 0 {
		t.Fatalf("verify --package exited %d\nstderr:\n%s", exit, stderr)
	}
	report, err := pkg.DecodePackageVerifyReport([]byte(out))
	if err != nil {
		t.Fatalf("decode: %v\noutput:\n%s", err, out)
	}
	if report.Package != "test/flatself" {
		t.Errorf("package = %q", report.Package)
	}
	if report.Verified != 2 || report.Counterexample != 0 || report.Errors != 0 {
		t.Errorf("verified/counterexample/errors = %d/%d/%d, want 2/0/0\n%s", report.Verified, report.Counterexample, report.Errors, out)
	}
	if report.Total != 2 {
		t.Errorf("total = %d, want 2", report.Total)
	}
	if len(report.Modules) != 2 {
		t.Fatalf("modules = %d, want 2", len(report.Modules))
	}
	// Sorted module order and slash-normalised relative file paths (Windows).
	if report.Modules[0].Module != "test/flatself/settle" || report.Modules[0].File != "settle.ail" {
		t.Errorf("module[0] = %+v", report.Modules[0])
	}
}

func TestDecodePackageVerifyReport_RefusesBareArray(t *testing.T) {
	if _, err := pkg.DecodePackageVerifyReport([]byte(`[{"status":"verified"}]`)); err == nil {
		t.Fatal("bare array decoded without error — the pre-M1 validator shape must be refused")
	}
	if _, err := pkg.DecodePackageVerifyReport([]byte(`{"file":"x","verified":7,"results":[]}`)); err == nil {
		t.Fatal("single-file verify object decoded as a package report")
	}
}
