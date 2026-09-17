package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-PKG-QUALITY-LADDER M3 — the publisher-side command and the publish gate.
// runAilangBin executes from the project root; publish reads its cwd, so the
// dry-run arms use RunBounded with the fixture as working directory.

func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "internal", "pkg", "testdata", "quality", name))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestPkgQuality_JSONReportOnFlatFixture(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	bin := buildAilang(t)
	out, stderr, exit := runAilangBin(t, bin, "pkg", "quality", "--json", "--no-run", filepath.Join("internal", "pkg", "testdata", "quality", "flat_self_import"))
	if exit != 0 {
		t.Fatalf("pkg quality exited %d\nstderr:\n%s\nstdout:\n%s", exit, stderr, out)
	}
	var r pkg.QualityReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if r.Schema != pkg.QualitySchema || r.Mode != "publisher" {
		t.Errorf("schema/mode = %q/%q", r.Schema, r.Mode)
	}
	if !r.Compile.OK || r.Compile.Source != pkg.SourceServer {
		t.Errorf("compile = %+v", r.Compile)
	}
	if r.Contracts.Verified != 2 || r.Contracts.Total != 2 || r.Contracts.Counterexample != 0 {
		t.Errorf("contracts = %+v", r.Contracts)
	}
	if !strings.HasPrefix(r.Interface.HashV2, "sha256:ifacev2:") || r.Interface.Signatures == 0 {
		t.Errorf("interface = %+v", r.Interface)
	}
	if r.Style.ExportedFuncs != 4 || r.Style.PureExports != 4 {
		t.Errorf("style = %+v (fixture exports 4 pure funcs)", r.Style)
	}
	if len(r.Gates) != 0 {
		t.Errorf("unexpected gates: %v", r.Gates)
	}
	if r.Tests != nil {
		t.Errorf("--no-run must not attest tests: %+v", r.Tests)
	}
}

// --strict: the fixture has 2 uncontracted exports → PUB011 becomes a gate (exit 2).
func TestPkgQuality_StrictGatesUncontractedExports(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	bin := buildAilang(t)
	out, _, exit := runAilangBin(t, bin, "pkg", "quality", "--json", "--no-run", "--strict", filepath.Join("internal", "pkg", "testdata", "quality", "flat_self_import"))
	if exit != 2 {
		t.Fatalf("exit = %d, want 2\n%s", exit, out)
	}
	if !strings.Contains(out, `"code": "PUB011"`) {
		t.Errorf("expected PUB011 gate:\n%s", out)
	}
}

// publish --dry-run refuses a refuted contract with the same code the
// validator would return (PUB006), before any tarball is built.
func TestPublishDryRun_RefusesRefutedContract(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	bin := buildAilang(t)
	out, stderr, exit := testutil.RunBounded(t, fixtureDir(t, "refuted"), 90*time.Second, bin, "publish", "--dry-run")
	if exit == 0 {
		t.Fatalf("dry-run must fail on a refuted contract\n%s", out)
	}
	combined := out + stderr
	if !strings.Contains(combined, "PUB006") || !strings.Contains(combined, "quality gate") {
		t.Errorf("expected a PUB006 quality-gate refusal:\n%s", combined)
	}
}

func TestPublishDryRun_CleanFixturePasses(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("z3 not available (Windows CI has no z3)")
	}
	bin := buildAilang(t)
	out, stderr, exit := testutil.RunBounded(t, fixtureDir(t, "flat_self_import"), 90*time.Second, bin, "publish", "--dry-run")
	if exit != 0 {
		t.Fatalf("dry-run exited %d\n%s\n%s", exit, out, stderr)
	}
	if !strings.Contains(out, "Dry run complete") || !strings.Contains(out, "2/2 verified") {
		t.Errorf("dry-run output must carry the quality report and complete:\n%s", out)
	}
}
