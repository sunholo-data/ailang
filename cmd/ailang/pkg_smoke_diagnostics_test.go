package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func smokeDiagnosticPackage(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"ailang.toml": "[package]\nname = \"test/diag\"\nversion = \"0.1.0\"\nedition = \"1\"\n[exports]\nmodules = [\"test/diag/model\"]\n[effects]\nmax = [\"IO\",\"FS\",\"Clock\"]\n[smoke]\ntimeout_seconds = 10\n",
		"model.ail":   "module test/diag/model\nexport pure func ping() -> int = 1\n",
		"_smoke.ail":  body,
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPkgSmokeDiagnostics_StagingErrorIsReported(t *testing.T) {
	dir := smokeDiagnosticPackage(t, "module test/diag/_smoke\nexport func main() -> () = ()\n")
	if err := os.Mkdir(filepath.Join(dir, pkg.LockFileName), 0700); err != nil {
		t.Fatal(err)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	report := runAttestedChecks(dir, manifest)
	if report.Smoke == nil || report.Smoke.Passed || report.Smoke.NotRun || !strings.Contains(report.Smoke.Notes, pkg.LockFileName) {
		t.Fatalf("staging rErr disappeared: %+v", report.Smoke)
	}
}

func TestPkgSmokeDiagnostics_RealQualityNotes(t *testing.T) {
	bin := buildAilang(t)
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	t.Setenv("AILANG_REGISTRY", "http://127.0.0.1:1")
	cases := []struct{ name, source, want string }{
		{"long_output", `module test/diag/_smoke
import std/io (println, exit)
export func main() -> () ! {IO} { println("INFO: warming"); println("Error: SOURCE_DIAGNOSTIC"); println(` + fmt.Sprintf("%q", strings.Repeat("TRACE: 界 trailing\n", 1600)) + `); exit(23) }`, "SOURCE_DIAGNOSTIC"},
		{"timeout", `module test/diag/_smoke
import std/clock (sleep)
export func main() -> () ! {Clock} = sleep(5000)`, "timed out"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dir := smokeDiagnosticPackage(t, tt.source)
			if tt.name == "timeout" {
				path := filepath.Join(dir, pkg.ManifestFile)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(strings.Replace(string(data), "timeout_seconds = 10", "timeout_seconds = 1", 1)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			home := t.TempDir()
			result := runCLIIn(t, bin, home, dir, "pkg", "quality", "--json", ".")
			if result.exitCode != 2 {
				t.Fatalf("expected PUB015 gate: %+v", result)
			}
			var report pkg.QualityReport
			if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result.stdout)
			}
			if report.Smoke == nil || report.Smoke.Passed || !strings.Contains(report.Smoke.Notes, tt.want) || len(report.Smoke.Notes) > 2048 || !utf8.ValidString(report.Smoke.Notes) {
				t.Fatalf("wrong bounded cause: %+v", report.Smoke)
			}
			found := false
			for _, gate := range report.Gates {
				if gate.Code == "PUB015" {
					found = strings.Contains(gate.Msg, tt.want)
				}
			}
			if !found {
				t.Fatalf("human/JSON finding lost same cause: %+v", report.Gates)
			}
		})
	}
}
