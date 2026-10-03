package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// policyToolDenyFixture is a sandbox under a policy that deny-lists the
// editor config, the AILANG state dir and a library directory, with a test
// module in the deny-listed library that imports a sibling.
func policyToolDenyFixture(t *testing.T) (bin, dir, sandbox, pol string) {
	t.Helper()
	bin = buildAilang(t)
	dir = t.TempDir()
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	pol = writePolicyFixture(t, dir, `"IO","FS"`)
	f, err := os.OpenFile(pol, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("fs_deny_write = [\".claude/**\", \".ailang/**\", \"lib/**\"]\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	sandbox = filepath.Join(dir, "sandbox")
	for _, d := range []string{".claude", "lib"} {
		if err := os.MkdirAll(filepath.Join(sandbox, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeAil(t, filepath.Join(sandbox, ".claude"), "x.ail", "module x\nexport func   f() -> int = 1\n")
	// A package in the deny-listed dir: the test module imports a sibling,
	// which resolves through the package manifest next to it.
	if err := os.WriteFile(filepath.Join(sandbox, "lib", "ailang.toml"), []byte("[package]\nname = \"x/leak\"\nversion = \"0.1.0\"\nedition = \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Not deny-listed: the boolean-flag refusal must come from the flag rule.
	writeAil(t, sandbox, "free.ail", "module x\nexport func   f() -> int = 1\n")
	writeAil(t, filepath.Join(sandbox, "lib"), "util.ail", "module x/leak/util\n\nexport pure func inc(x: int) -> int = x + 1\n")
	writeAil(t, filepath.Join(sandbox, "lib"), "t_test.ail", "module x/leak/t_test\n\nimport x/leak/util (inc)\n\ntest \"sibling import\" { inc(1) == 2 }\n")
	return bin, dir, sandbox, pol
}

func askPolicyTool(t *testing.T, bin, dir, pol, req string) map[string]any {
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

// treeListing is every path under root, slash-separated, sorted.
func treeListing(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// #1559 + #1554 end to end through the binary: case variants of a
// deny-listed path, `fmt --write` on one, aliased/duplicate request keys —
// all refused, and nothing in the sandbox changes.
func TestPolicyTool_DenyWriteEndToEnd(t *testing.T) {
	bin, dir, sandbox, pol := policyToolDenyFixture(t)
	before := treeListing(t, sandbox)
	orig, _ := os.ReadFile(filepath.Join(sandbox, ".claude", "x.ail"))

	refusals := map[string]string{
		`{"op":"write","path":".CLAUDE/y.json","content":"{}"}`:                            "fs_deny_write",
		`{"op":"write","path":".AILANG/cache/z","content":"z"}`:                            "fs_deny_write",
		`{"op":"fmt","path":".claude/x.ail","flags":{"write":""}}`:                         "fs_deny_write",
		`{"op":"fmt","path":".CLAUDE/x.ail","flags":{"write":"true"}}`:                     "fs_deny_write",
		`{"op":"fmt","path":"free.ail","flags":{"write":"false"}}`:                         "boolean",
		`{"op":"fmt","path":".claude/x.ail","FLAGS":{"write":""}}`:                         "malformed request",
		`{"op":"fmt","path":".claude/x.ail","flags":{"check":""},"flags":{"write":""}}`:    "malformed request",
		`{"op":"fmt","path":".claude/x.ail","flags":{"write":""},"Op":"fmt"}`:              "malformed request",
		`{"op":"write","path":".GIT/config","content":"[core]\n\tfsmonitor = evil\n"}`:     ".git",
		`{"op":"edit","path":".Claude/x.ail","old_text":"module x","new_text":"module y"}`: "fs_deny_write",
	}
	for req, want := range refusals {
		r := askPolicyTool(t, bin, dir, pol, req)
		refused, _ := r["refused"].(string)
		if r["ok"] == true || !strings.Contains(refused, want) {
			t.Errorf("%s: want a refusal naming %q, got %v", req, want, r)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(sandbox, ".claude", "x.ail")); string(b) != string(orig) {
		t.Errorf(".claude/x.ail was rewritten: %q", b)
	}
	if after := treeListing(t, sandbox); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Errorf("sandbox changed:\nbefore %v\nafter  %v", before, after)
	}
}

// #1554 (2) + the child's compile cache: `test` on a module in a deny-listed
// directory resolves its sibling import and passes, while writing nothing in
// the sandbox — no `_namedtest_body_*` scaffold next to the module (it lives
// in a private temp dir, #1502) and no `.ailang/cache` (policy-tool points
// the child's cache outside the sandbox). `check` likewise.
func TestPolicyTool_CLIOpsWriteNothingInSandbox(t *testing.T) {
	bin, dir, sandbox, pol := policyToolDenyFixture(t)
	writeAil(t, sandbox, "prog.ail", "module prog\n\nimport std/list (length)\n\nexport pure func n() -> int = length([1, 2])\n")
	before := treeListing(t, sandbox)

	r := askPolicyTool(t, bin, dir, pol, `{"op":"test","path":"lib/t_test.ail"}`)
	out, _ := r["stdout"].(string)
	errOut, _ := r["stderr"].(string)
	if r["ok"] != true || !strings.Contains(out+errOut, "sibling import") {
		t.Fatalf("test op on a module in a deny-listed dir must run and pass: %v", r)
	}
	if r = askPolicyTool(t, bin, dir, pol, `{"op":"check","path":"prog.ail"}`); r["ok"] != true {
		t.Fatalf("check: %v", r)
	}
	if after := treeListing(t, sandbox); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Errorf("a read-only CLI op wrote into the sandbox:\nbefore %v\nafter  %v", before, after)
	}
}
