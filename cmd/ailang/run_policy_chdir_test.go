package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A caller that cannot set a child's working directory (an AILANG program:
// std/process.exec takes no cwd) runs a sandboxed program with --chdir. Module
// names resolve from the --chdir directory, so `module app/prog` in
// <sandbox>/app/prog.ail runs from any caller cwd; under --policy the --chdir
// directory must itself be inside fs_sandbox (it is the module root).
func TestRunPolicy_Chdir(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	// FS admitted: without it the policy resolves no sandbox root (fs_sandbox
	// is "" when FS is not admitted), so neither the entry-file rule nor the
	// --chdir rule has a root to check against. Lane policies always admit FS.
	pol := writePolicyFixture(t, dir, `"IO","FS"`)
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(filepath.Join(sandbox, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeAil(t, filepath.Join(sandbox, "app"), "prog.ail", "module app/prog\nexport func main() -> () ! {IO} = println(\"chdir-ran\")\n")

	// Inside the sandbox: the module path resolves from --chdir.
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, "--chdir", sandbox, "app/prog.ail")
	if code != 0 || !strings.Contains(stdout, "chdir-ran") {
		t.Fatalf("--chdir inside the sandbox: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	// Outside the sandbox: refused before anything runs.
	stdout, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, "--chdir", dir, filepath.Join(sandbox, "app", "prog.ail"))
	if code == 0 || strings.Contains(stdout, "chdir-ran") {
		t.Fatalf("--chdir outside the sandbox was not refused: exit %d\nstdout: %s", code, stdout)
	}
	if !strings.Contains(stderr, "--chdir") || !strings.Contains(stderr, "outside fs_sandbox") {
		t.Fatalf("refusal does not name --chdir and the sandbox:\n%s", stderr)
	}
}

// policy-tool reads its request from --request-file when the caller cannot
// pipe stdin; the policy still decides (a path escaping the sandbox is refused).
func TestPolicyTool_RequestFile(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO","FS"`)
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.WriteFile(filepath.Join(sandbox, "note.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	ask := func(req string) map[string]any {
		t.Helper()
		f := filepath.Join(dir, "req.json")
		if err := os.WriteFile(f, []byte(req), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runAilangBin(t, bin, "policy-tool", "--policy", pol, "--request-file", f)
		if code != 0 {
			t.Fatalf("policy-tool exit %d\nstderr: %s", code, stderr)
		}
		var resp map[string]any
		if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
			t.Fatalf("response is not JSON: %v\n%s", err, stdout)
		}
		return resp
	}
	if r := ask(`{"op":"read","path":"note.txt"}`); r["ok"] != true || r["content"] != "inside" {
		t.Fatalf("read inside the sandbox via --request-file: %v", r)
	}
	if r := ask(`{"op":"read","path":"../policy.toml"}`); r["ok"] == true {
		t.Fatalf("a path escaping the sandbox was read via --request-file: %v", r)
	}
}
