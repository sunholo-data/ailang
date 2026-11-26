package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestInspect_SingleFileAgentAndJSON verifies TST-IN-01:
// `ailang inspect <file> --format=agent` and `--json` emit exact types, `! {Effects}`,
// `requires`/`ensures` contracts, line numbers, and direct `calls` for both exported
// and local functions, plus `--exports-only` / `--exported-only` filtering.
func TestInspect_SingleFileAgentAndJSON(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.ail")
	src := `module sample

import std/io (println)
import std/list (length)

export type Item = { name: string, count: int } deriving (Eq)
type Status = Idle | Busy(int)

export func summarize(items: [Item]) -> int ! {IO}
  requires { length(items) >= 0 }
  ensures { result >= 0 }
{
  let n = countLocal(items)
  println(show(n))
  n
}

pure func countLocal(items: [Item]) -> int
  ensures { result >= 0 }
{
  length(items)
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.ail: %v", err)
	}

	// 1. Default (--format=agent) includes both exported and local symbols + types.
	var agentBuf bytes.Buffer
	if err := runInspectCommandTo([]string{srcPath, "--relax-modules"}, &agentBuf); err != nil {
		t.Fatalf("inspect agent failed: %v", err)
	}
	agentOut := agentBuf.String()
	for _, want := range []string{
		"module sample (" + srcPath + ")",
		"type Item = { name: string, count: int } deriving (Eq) [export, line 6]",
		"type Status = Idle | Busy(int) [line 7]",
		"export func summarize(",
		"! {IO}",
		"requires {",
		"ensures {",
		"calls: countLocal, std/io.println",
		"func countLocal(",
		"! {}",
		"std/list.length",
	} {
		if !strings.Contains(agentOut, want) {
			t.Errorf("agent output missing %q\nfull output:\n%s", want, agentOut)
		}
	}

	// 2. --json emits structured JSON matching D4=a schema (with unescaped `->`).
	var jsonBuf bytes.Buffer
	if err := runInspectCommandTo([]string{"--json", "--relax-modules", srcPath}, &jsonBuf); err != nil {
		t.Fatalf("inspect json failed: %v", err)
	}
	jsonOut := jsonBuf.String()
	if strings.Contains(jsonOut, `\u003e`) {
		t.Errorf("JSON output escaped '->' as \\u003e:\n%s", jsonOut)
	}

	var report inspectReport
	if err := json.Unmarshal([]byte(jsonOut), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, jsonOut)
	}
	if len(report.Files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(report.Files))
	}
	f := report.Files[0]
	if f.Module != "sample" || f.File != srcPath {
		t.Errorf("file header = (%q, %q), want (%q, %q)", f.Module, f.File, "sample", srcPath)
	}
	if len(f.Types) != 2 {
		t.Fatalf("len(types) = %d, want 2: %+v", len(f.Types), f.Types)
	}
	if f.Types[0].Name != "Item" || f.Types[0].Kind != "record" || !f.Types[0].Exported || f.Types[0].Line != 6 {
		t.Errorf("types[0] = %+v, want exported record Item at line 6", f.Types[0])
	}
	if len(f.Types[0].Deriving) != 1 || f.Types[0].Deriving[0] != "Eq" {
		t.Errorf("types[0].Deriving = %v, want [Eq]", f.Types[0].Deriving)
	}
	if f.Types[1].Name != "Status" || f.Types[1].Kind != "adt" || f.Types[1].Exported || f.Types[1].Line != 7 {
		t.Errorf("types[1] = %+v, want non-exported adt Status at line 7", f.Types[1])
	}

	if len(f.Symbols) != 2 {
		t.Fatalf("len(symbols) = %d, want 2: %+v", len(f.Symbols), f.Symbols)
	}
	sumSym := f.Symbols[0]
	if sumSym.Name != "summarize" || !sumSym.Exported || sumSym.Pure || sumSym.Line != 9 {
		t.Errorf("symbols[0] = %+v", sumSym)
	}
	if len(sumSym.Effects) != 1 || sumSym.Effects[0] != "IO" {
		t.Errorf("summarize.Effects = %v, want [IO]", sumSym.Effects)
	}
	if len(sumSym.Contracts) != 2 || sumSym.Contracts[0].Kind != "requires" || sumSym.Contracts[1].Kind != "ensures" {
		t.Errorf("summarize.Contracts = %+v, want [requires, ensures]", sumSym.Contracts)
	}
	if !containsString(sumSym.Calls, "countLocal") || !containsString(sumSym.Calls, "std/io.println") {
		t.Errorf("summarize.Calls = %v, want to contain countLocal and std/io.println", sumSym.Calls)
	}

	locSym := f.Symbols[1]
	if locSym.Name != "countLocal" || locSym.Exported || !locSym.Pure || locSym.Line != 18 {
		t.Errorf("symbols[1] = %+v", locSym)
	}
	if len(locSym.Effects) != 0 {
		t.Errorf("countLocal.Effects = %v, want []", locSym.Effects)
	}
	if !containsString(locSym.Calls, "std/list.length") {
		t.Errorf("countLocal.Calls = %v, want to contain std/list.length", locSym.Calls)
	}

	// 3. --exports-only and --exported-only filter out non-exported types & functions.
	for _, flagName := range []string{"--exports-only", "--exported-only"} {
		var expBuf bytes.Buffer
		if err := runInspectCommandTo([]string{srcPath, "--json", "--relax-modules", flagName}, &expBuf); err != nil {
			t.Fatalf("inspect %s failed: %v", flagName, err)
		}
		var expRep inspectReport
		if err := json.Unmarshal(expBuf.Bytes(), &expRep); err != nil {
			t.Fatalf("unmarshal %s: %v", flagName, err)
		}
		if len(expRep.Files[0].Types) != 1 || expRep.Files[0].Types[0].Name != "Item" {
			t.Errorf("%s types = %+v, want only [Item]", flagName, expRep.Files[0].Types)
		}
		if len(expRep.Files[0].Symbols) != 1 || expRep.Files[0].Symbols[0].Name != "summarize" {
			t.Errorf("%s symbols = %+v, want only [summarize]", flagName, expRep.Files[0].Symbols)
		}
	}
}

// TestInspect_DirectoryDeterministicOrder verifies directory walking emits sorted .ail modules.
func TestInspect_DirectoryDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"b_mod.ail": "module b_mod\nexport pure func beta(x: int) -> int = x + 1\n",
		"a_mod.ail": "module a_mod\nexport pure func alpha(x: int) -> int = x * 2\n",
		"notes.txt": "ignored\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	var buf bytes.Buffer
	if err := runInspectCommandTo([]string{dir, "--json", "--relax-modules"}, &buf); err != nil {
		t.Fatalf("inspect dir failed: %v", err)
	}
	var report inspectReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(report.Files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(report.Files))
	}
	if report.Files[0].Module != "a_mod" || report.Files[1].Module != "b_mod" {
		t.Errorf("module order = [%s, %s], want [a_mod, b_mod]", report.Files[0].Module, report.Files[1].Module)
	}
}

// TestInspect_StdStringLatency verifies TST-IN-02: single-module inspection on std/string.ail
// completes quickly (<100ms steady-state; 250ms generous CI ceiling) and emits expected exports.
func TestInspect_StdStringLatency(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "std", "string.ail")); err != nil {
		t.Skipf("std/string.ail not found at %s: %v", repoRoot, err)
	}
	t.Chdir(repoRoot)

	// Warm stdlib cache once, then measure single-module inspect latency.
	var warm bytes.Buffer
	if err := runInspectCommandTo([]string{"std/string.ail", "--relax-modules"}, &warm); err != nil {
		t.Fatalf("warm inspect std/string.ail failed: %v", err)
	}

	start := time.Now()
	var buf bytes.Buffer
	if err := runInspectCommandTo([]string{"std/string.ail", "--format=agent", "--relax-modules"}, &buf); err != nil {
		t.Fatalf("inspect std/string.ail failed: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("inspect std/string.ail elapsed: %v", elapsed)

	if elapsed > 250*time.Millisecond {
		t.Errorf("inspect std/string.ail took %v, want <250ms (target <100ms)", elapsed)
	}
	out := buf.String()
	for _, want := range []string{
		"module std/string",
		"export func length(s: string) -> int ! {}",
		"export func formatFixed(",
		"requires {",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("std/string inspect output missing %q", want)
		}
	}
}

// TestInspect_HelpAndErrorHandling verifies --help exits nil and invalid flags/missing args error cleanly.
func TestInspect_HelpAndErrorHandling(t *testing.T) {
	var helpBuf bytes.Buffer
	if err := runInspectCommandTo([]string{"--help"}, &helpBuf); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if !strings.Contains(helpBuf.String(), "Usage: ailang inspect") {
		t.Errorf("--help missing Usage line:\n%s", helpBuf.String())
	}

	var errBuf bytes.Buffer
	if err := runInspectCommandTo([]string{}, &errBuf); err == nil {
		t.Error("expected error when no path argument is provided")
	}
	if err := runInspectCommandTo([]string{"--format=xml", "foo.ail"}, &errBuf); err == nil {
		t.Error("expected error for unsupported --format")
	}
}

func containsString(xs []string, want string) bool {
	for _, s := range xs {
		if s == want {
			return true
		}
	}
	return false
}
