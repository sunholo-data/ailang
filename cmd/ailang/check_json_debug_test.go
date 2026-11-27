package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TST-RC1-17
func TestCheckDebugCompileJSON_TimingsWriterPurity(t *testing.T) {
	var buf bytes.Buffer
	timings := map[string]int64{
		"parse":     5,
		"elaborate": 7,
		"typecheck": 12,
	}
	printPhaseTimingsTo(&buf, timings)
	got := buf.String()
	for _, want := range []string{"Compilation Phase Timings", "Parsing:", "Type Checking:", "Total:"} {
		if !strings.Contains(got, want) {
			t.Errorf("printPhaseTimingsTo output missing %q:\n%s", want, got)
		}
	}
}

// TST-RC1-17
func TestCheckDebugCompileJSON_StdoutPurity(t *testing.T) {
	bin := cliTestBin(t)
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "ok.ail")
	src := "module ok\nexport pure func add(a: int, b: int) -> int = a + b\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	res := runCLIIn(t, bin, t.TempDir(), dir, "check", "--debug-compile", "--json", srcPath)
	if res.exitCode != 0 {
		t.Fatalf("ailang check --debug-compile --json failed (exit %d): stdout=%q stderr=%q", res.exitCode, res.stdout, res.stderr)
	}

	var parsed checkJSONOutput
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("stdout is not valid JSON (corrupted by debug output?): %v\nstdout=%q\nstderr=%q", err, res.stdout, res.stderr)
	}
	if !parsed.Passed || parsed.ErrorCount != 0 {
		t.Fatalf("unexpected JSON check payload: %+v", parsed)
	}
	if strings.Contains(res.stdout, "Compilation Phase Timings") || strings.Contains(res.stdout, "[DEBUG") {
		t.Fatalf("debug output leaked to stdout: %q", res.stdout)
	}
	if !strings.Contains(res.stderr, "Compilation Phase Timings") {
		t.Fatalf("expected phase timings on stderr, got stderr=%q", res.stderr)
	}
}
