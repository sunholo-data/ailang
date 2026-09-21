package policytool

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// M-EXECUTOR-POLICY-HARDENING M4 (AC5): direct tool invocations, no LLM.

type fixture struct {
	tmp, sandbox, policyPath, marker string
	host                             *Host
	runs                             [][]string
}

func newFixture(t *testing.T, extraPolicy string) *fixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	tmp := t.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	f := &fixture{tmp: tmp, sandbox: filepath.Join(tmp, "sandbox")}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(f.sandbox, "sub"), 0o755))
	must(os.MkdirAll(filepath.Join(tmp, "ext"), 0o755))
	f.marker = filepath.Join(tmp, "marker.txt")
	must(os.WriteFile(f.marker, []byte("OUTSIDE-SENTINEL"), 0o644))
	must(os.WriteFile(filepath.Join(tmp, "ext", "ailang-exec.ts"), []byte("// extension"), 0o644))
	must(os.WriteFile(filepath.Join(f.sandbox, "in.ail"), []byte("module in\nexport func main() -> () ! {} = ()\n"), 0o644))
	must(os.WriteFile(filepath.Join(f.sandbox, "sub", "deep.txt"), []byte("deep"), 0o644))
	must(os.Symlink("../marker.txt", filepath.Join(f.sandbox, "link.txt")))
	must(os.Symlink(tmp, filepath.Join(f.sandbox, "linkdir")))
	// The policy lives OUTSIDE the sandbox (D4) and is not writable through it.
	f.policyPath = filepath.Join(tmp, "policy.toml")
	must(os.WriteFile(f.policyPath, []byte("allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \""+filepath.ToSlash(f.sandbox)+"\"\nentry = \"main\"\n"+extraPolicy), 0o644))
	h, err := Open(f.policyPath)
	must(err)
	h.run = func(dir string, argv []string) (string, string, int) {
		f.runs = append(f.runs, append([]string{dir}, argv...))
		return "ran", "", 0
	}
	f.host = h
	t.Cleanup(func() { _ = h.Close() })
	return f
}

func (f *fixture) outsideIntact(t *testing.T) {
	t.Helper()
	if b, err := os.ReadFile(f.marker); err != nil || string(b) != "OUTSIDE-SENTINEL" {
		t.Errorf("outside marker changed: %q %v", b, err)
	}
	if b, err := os.ReadFile(f.policyPath); err != nil || !strings.Contains(string(b), "allowed_caps") {
		t.Errorf("policy file changed: %q %v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(f.tmp, "ext", "ailang-exec.ts")); err != nil || string(b) != "// extension" {
		t.Errorf("extension changed: %q %v", b, err)
	}
}

func TestPolicyTool_FileOpsDenyOutside(t *testing.T) {
	f := newFixture(t, "")
	outside := []string{"../marker.txt", "link.txt", "linkdir/marker.txt", f.marker, f.policyPath, "../policy.toml", "../ext/ailang-exec.ts", "linkdir/ext/ailang-exec.ts", "sub/../../marker.txt"}
	for _, p := range outside {
		if r := f.host.Dispatch(Request{Op: "read", Path: p}); r.OK || strings.Contains(r.Content, "SENTINEL") || strings.Contains(r.Content, "allowed_caps") {
			t.Errorf("read %s: %+v", p, r)
		}
		if r := f.host.Dispatch(Request{Op: "write", Path: p, Content: "clobbered"}); r.OK {
			t.Errorf("write %s succeeded", p)
		}
		if r := f.host.Dispatch(Request{Op: "edit", Path: p, OldText: "a", NewText: "b"}); r.OK {
			t.Errorf("edit %s succeeded", p)
		}
	}
	f.outsideIntact(t)
}

func TestPolicyTool_FileOpsPositive(t *testing.T) {
	f := newFixture(t, "")
	if r := f.host.Dispatch(Request{Op: "read", Path: "sub/deep.txt"}); !r.OK || r.Content != "deep" {
		t.Fatalf("read: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "read", Path: filepath.Join(f.sandbox, "sub", "deep.txt")}); !r.OK || r.Content != "deep" {
		t.Fatalf("absolute in-root read: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "write", Path: "new.txt", Content: "hello world"}); !r.OK {
		t.Fatalf("write: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: "new.txt", OldText: "world", NewText: "there"}); !r.OK {
		t.Fatalf("edit: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "read", Path: "new.txt"}); r.Content != "hello there" {
		t.Fatalf("after edit: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: "new.txt", OldText: "zzz", NewText: "y"}); r.OK || !strings.Contains(r.Refused, "not found") {
		t.Fatalf("edit miss: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: "new.txt", OldText: "e", NewText: "y"}); r.OK || !strings.Contains(r.Refused, "occurs") {
		t.Fatalf("ambiguous edit: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: "new.txt", OldText: "", NewText: "y"}); r.OK {
		t.Fatalf("empty old_text must be refused: %+v", r)
	}
}

func TestPolicyTool_TransferCap(t *testing.T) {
	f := newFixture(t, "max_fs_transfer_bytes = 16\n")
	if r := f.host.Dispatch(Request{Op: "write", Path: "big.txt", Content: strings.Repeat("x", 17)}); r.OK {
		t.Fatal("write over the cap must be refused")
	}
	if err := os.WriteFile(filepath.Join(f.sandbox, "big.txt"), []byte(strings.Repeat("y", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := f.host.Dispatch(Request{Op: "read", Path: "big.txt"}); r.OK || !strings.Contains(r.Refused, "cap") {
		t.Fatalf("read over the cap: %+v", r)
	}
}

func TestPolicyTool_CLIPathBearingOptionsRefused(t *testing.T) {
	f := newFixture(t, "")
	cases := []Request{
		{Op: "check", Path: "../marker.txt"},
		{Op: "check", Path: f.policyPath},
		{Op: "check", Path: "link.txt"},
		{Op: "check", Path: "in.ail", Flags: map[string]string{"stdlib-path": "/etc"}},
		{Op: "check", Path: "in.ail", Flags: map[string]string{"package-dir": "/tmp"}},
		{Op: "check", Path: "in.ail", Flags: map[string]string{"o": "/tmp/x"}},
		{Op: "check", Path: "-o"},
		{Op: "fmt", Path: "../ext/ailang-exec.ts", Flags: map[string]string{"write": ""}},
		{Op: "test", Package: "../"},
		{Op: "test", Package: "linkdir"},
		{Op: "iface", Module: "../../etc/passwd"},
		{Op: "iface", Module: "/etc/passwd"},
		{Op: "policy_check", Path: "in.ail", Flags: map[string]string{"policy": "/tmp/evil.toml"}},
		{Op: "docs_search", Query: "--limit 1"},
		{Op: "check", Path: "in.ail", Content: "smuggled"},
		{Op: "run", Path: "in.ail"},
		{Op: "exec", Path: "in.ail"},
		{Op: "bash", Query: "rm -rf /"},
	}
	for _, req := range cases {
		r := f.host.Dispatch(req)
		if r.OK || r.Refused == "" {
			t.Errorf("%+v: expected a named refusal, got %+v", req, r)
		}
	}
	if len(f.runs) != 0 {
		t.Fatalf("nothing may have been executed: %v", f.runs)
	}
	f.outsideIntact(t)
}

func TestPolicyTool_CLIPositive(t *testing.T) {
	f := newFixture(t, "")
	r := f.host.Dispatch(Request{Op: "check", Path: "in.ail", Flags: map[string]string{"json": ""}})
	if !r.OK || r.Stdout != "ran" {
		t.Fatalf("check: %+v", r)
	}
	want := []string{f.sandbox, "check", "--json", "in.ail"}
	if strings.Join(f.runs[0], " ") != strings.Join(want, " ") {
		t.Fatalf("argv = %v, want %v", f.runs[0], want)
	}
	r = f.host.Dispatch(Request{Op: "docs_search", Query: "read a file", Flags: map[string]string{"limit": "3"}})
	if !r.OK || strings.Join(f.runs[1], " ") != " docs search --limit 3 read a file" {
		t.Fatalf("docs_search: %+v %v", r, f.runs[1])
	}
	r = f.host.Dispatch(Request{Op: "test", Package: "sub"})
	if !r.OK || strings.Join(f.runs[2], " ") != f.sandbox+" test --package sub" {
		t.Fatalf("test: %+v %v", r, f.runs[2])
	}
	r = f.host.Dispatch(Request{Op: "policy_check", Path: "in.ail"})
	if !r.OK || strings.Join(f.runs[3], " ") != f.sandbox+" policy-check --policy "+f.policyPath+" in.ail" {
		t.Fatalf("policy_check: %+v %v", r, f.runs[3])
	}
	r = f.host.Dispatch(Request{Op: "iface", Module: "std/fs"})
	if !r.OK || strings.Join(f.runs[4], " ") != f.sandbox+" iface std/fs" {
		t.Fatalf("iface: %+v %v", r, f.runs[4])
	}
}

func TestPolicyTool_CLIAllowFromPolicy(t *testing.T) {
	f := newFixture(t, "cli_allow = [\"version\", \"docs:search\"]\n")
	if r := f.host.Dispatch(Request{Op: "check", Path: "in.ail"}); r.OK || !strings.Contains(r.Refused, "cli_allow") {
		t.Fatalf("check outside cli_allow: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "version"}); !r.OK {
		t.Fatalf("version: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "docs_search", Query: "x"}); !r.OK {
		t.Fatalf("docs:search: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "examples_search", Query: "x"}); r.OK {
		t.Fatalf("examples not listed: %+v", r)
	}
	s := f.host.Dispatch(Request{Op: "summary"}).Summary
	if strings.Join(s.CLI, ",") != "docs_search,version" {
		t.Fatalf("summary.cli = %v", s.CLI)
	}
	// A gate-only command listed in cli_allow is still refused.
	f = newFixture(t, "cli_allow = [\"run\", \"check\"]\n")
	if r := f.host.Dispatch(Request{Op: "run", Path: "in.ail"}); r.OK {
		t.Fatalf("run is gate-only: %+v", r)
	}
}

func TestPolicyTool_SummaryIsGoProduced(t *testing.T) {
	f := newFixture(t, "")
	r := f.host.Dispatch(Request{Op: "summary"})
	if !r.OK || r.Summary == nil {
		t.Fatalf("%+v", r)
	}
	s := r.Summary
	if s.Mode != "restricted" || s.Sandbox != f.sandbox || strings.Join(s.Caps, ",") != "FS,IO" || s.PolicyDigest == "" || s.TimeoutMs != 5000 {
		t.Fatalf("summary: %+v", s)
	}
	for _, op := range []string{"read", "write", "edit", "check", "docs_search"} {
		found := false
		for _, o := range s.Ops {
			if o == op {
				found = true
			}
		}
		if !found {
			t.Errorf("summary.ops lacks %s: %v", op, s.Ops)
		}
	}
}

func TestPolicyTool_NoFSPolicyRefusesFileOps(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "p.toml")
	if err := os.WriteFile(p, []byte("allowed_caps = [\"IO\"]\nentry = \"main\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if r := h.Dispatch(Request{Op: "read", Path: "x"}); r.OK || !strings.Contains(r.Refused, "no FS") {
		t.Fatalf("%+v", r)
	}
	if r := h.Dispatch(Request{Op: "check", Path: "x"}); r.OK {
		t.Fatalf("%+v", r)
	}
	// version needs no root; with no binary configured the library refuses
	// to exec anything (never itself) and says so.
	if r := h.Dispatch(Request{Op: "version"}); r.OK || !strings.Contains(r.Stderr, "no ailang binary configured") {
		t.Fatalf("%+v", r)
	}
}

// M6: the repository metadata is read-only to the lane's tools.
func TestPolicyTool_DotGitIsReadOnly(t *testing.T) {
	f := newFixture(t, "")
	if err := os.MkdirAll(filepath.Join(f.sandbox, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.sandbox, ".git", "config"), []byte("[core]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".git/config", ".git/hooks/pre-commit", "sub/../.git/config", filepath.Join(f.sandbox, ".git", "config")} {
		if r := f.host.Dispatch(Request{Op: "write", Path: p, Content: "[core]\n\tfsmonitor = evil\n"}); r.OK || !strings.Contains(r.Refused, ".git") {
			t.Errorf("write %s: %+v", p, r)
		}
		if r := f.host.Dispatch(Request{Op: "edit", Path: p, OldText: "[core]", NewText: "[core]\n\tfsmonitor = evil"}); r.OK || !strings.Contains(r.Refused, ".git") {
			t.Errorf("edit %s: %+v", p, r)
		}
	}
	if r := f.host.Dispatch(Request{Op: "read", Path: ".git/config"}); !r.OK {
		t.Errorf("reading .git is fine: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "write", Path: ".gitignore", Content: "x\n"}); !r.OK {
		t.Errorf(".gitignore is an ordinary file: %+v", r)
	}
	if b, _ := os.ReadFile(filepath.Join(f.sandbox, ".git", "config")); string(b) != "[core]\n" {
		t.Fatal(".git/config changed")
	}
}
