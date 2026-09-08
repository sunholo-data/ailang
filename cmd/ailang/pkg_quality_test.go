package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func qualityFixture(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"ailang.toml": "[package]\nname = \"test/evidence\"\nversion = \"0.1.0\"\nedition = \"1\"\n[exports]\nmodules = [\"test/evidence/core\"]\n[effects]\nmax = []\n",
		"core.ail":    "module test/evidence/core\n" + source,
		"AGENT.md":    "Usage and effects documented here.",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPackageQualityDoesNotCountCommentsOrTestNames(t *testing.T) {
	dir := qualityFixture(t, `-- tests [(1,1)] ensures { true }
export pure func test_identity(x: int) -> int { x }
`)
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.NativeTests != 0 || report.ContractClauses != 0 || len(report.Gaps) != 2 {
		t.Fatalf("%+v", report)
	}
	var out, errs bytes.Buffer
	if err := runPackageQuality([]string{"--strict", "--json", dir}, &out, &errs); err == nil {
		t.Fatal("strict accepted missing evidence")
	}
	var decoded packageQualityReport
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Execution != "not_run" {
		t.Fatal(decoded.Execution)
	}
}

func TestPackageQualityNativeEvidenceAndPrivateHelpers(t *testing.T) {
	dir := qualityFixture(t, `export pure func identity(x: int) -> int
ensures { result == x }
tests [(1, 1), (0, 0)]
{ x }
pure func helper(x: int) -> int { x }
`)
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.NativeTests != 2 || report.ContractClauses != 1 || report.Properties != 0 {
		t.Fatalf("%+v", report)
	}
	if len(report.Gaps) != 1 || !strings.Contains(report.Gaps[0], "helper") {
		t.Fatalf("%+v", report.Gaps)
	}
	if len(report.TestSources) != 1 || report.TestSources[0] != "core.ail" {
		t.Fatal(report.TestSources)
	}
}

func TestPackageQualityContractAloneIsNotTest(t *testing.T) {
	dir := qualityFixture(t, `export pure func identity(x: int) -> int
ensures { result == x }
{ x }
`)
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.NativeTests != 0 || report.Properties != 0 || len(report.Gaps) != 1 {
		t.Fatalf("%+v", report)
	}
}

func TestPackageQualityBudgetsAndTestFile(t *testing.T) {
	dir := qualityFixture(t, `export func bounded() -> () ! {IO @limit=1} { () }
export func unbounded() -> () ! {Net} { () }
`)
	testSource := `module test/evidence/core_test
pure func specimen(x: int) -> int tests [(1,1)] { x }
`
	if err := os.WriteFile(filepath.Join(dir, "core_test.ail"), []byte(testSource), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.NativeTests != 1 || len(report.Gaps) != 1 || !strings.Contains(report.Gaps[0], "budget for Net") {
		t.Fatalf("%+v", report)
	}
}

func TestPackageQualityParseFailureAndEmptyTests(t *testing.T) {
	for _, source := range []string{"export pure func broken(", "export pure func identity(x: int) -> int tests [] { x }"} {
		dir := qualityFixture(t, source)
		report, err := inspectPackageQuality(dir)
		if strings.Contains(source, "broken") {
			if err == nil {
				t.Fatal("accepted malformed source")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if report.NativeTests != 0 {
			t.Fatal(report)
		}
	}
}

func TestPackageQualityStrictPassAndHelp(t *testing.T) {
	dir := qualityFixture(t, `export pure func identity(x: int) -> int
ensures { result == x }
tests [(1,1)]
{ x }
`)
	var out, errs bytes.Buffer
	if err := runPackageQuality([]string{"--strict", dir}, &out, &errs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "NOT ASSESSED") {
		t.Fatal(out.String())
	}
	if err := runPackageQuality([]string{"--help"}, &out, &errs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errs.String(), "never executes") {
		t.Fatal(errs.String())
	}
	if err := runPackageQuality([]string{dir, "extra"}, &out, &errs); err == nil {
		t.Fatal("accepted extra arg")
	}
}

func TestPackageQualityOrphanHelperAndMissingDocs(t *testing.T) {
	dir := qualityFixture(t, `export pure func identity(x: int) -> int
ensures { result == x }
tests [(1,1)]
{ x }
`)
	if err := os.Remove(filepath.Join(dir, "AGENT.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "private.ail"), []byte("module test/evidence/private\npure func helper(x: int) -> int { x }"), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Gaps) != 2 || report.Files != 2 {
		t.Fatalf("%+v", report)
	}
}

func TestPackageQualityPropertyDeclaration(t *testing.T) {
	dir := qualityFixture(t, `export pure func identity(x: int) -> int
ensures { result == x }
{ x }
property "identity" { forall (x: int) => identity(x) == x }
`)
	report, err := inspectPackageQuality(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Properties != 1 || report.NativeTests != 0 || len(report.Gaps) != 0 {
		t.Fatalf("%+v", report)
	}
}
