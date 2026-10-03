package policytool

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

// #1559: the tools match fs_deny_write and `.git` case-insensitively — on
// APFS/NTFS `.CLAUDE/y.json` lands in `.claude/`.
func TestPolicyTool_DenyWriteCaseVariants(t *testing.T) {
	f := newFixture(t, "fs_deny_write = [\".claude/**\", \".ailang/**\", \"Makefile\"]\n")
	if err := os.MkdirAll(filepath.Join(f.sandbox, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.sandbox, ".claude", "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".CLAUDE/y.json", ".Claude/settings.json", ".AILANG/cache/z", "MAKEFILE", "sub/../.cLaUdE/x", filepath.Join(f.sandbox, ".CLAUDE", "y.json")} {
		if r := f.host.Dispatch(Request{Op: "write", Path: p, Content: "{\"hooks\":1}"}); r.OK || !strings.Contains(r.Refused, "fs_deny_write") {
			t.Errorf("write %s: %+v", filepath.ToSlash(p), r)
		}
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: ".CLAUDE/settings.json", OldText: "{}", NewText: "{\"hooks\":1}"}); r.OK || !strings.Contains(r.Refused, "fs_deny_write") {
		t.Errorf("edit .CLAUDE/settings.json: %+v", r)
	}
	for _, p := range []string{".GIT/config", ".Git/hooks/pre-commit"} {
		if r := f.host.Dispatch(Request{Op: "write", Path: p, Content: "x"}); r.OK || !strings.Contains(r.Refused, ".git") {
			t.Errorf("write %s: %+v", p, r)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(f.sandbox, ".claude", "settings.json")); string(b) != "{}" {
		t.Fatalf(".claude/settings.json changed: %q", b)
	}
	entries, _ := os.ReadDir(filepath.Join(f.sandbox, ".claude"))
	if len(entries) != 1 {
		t.Fatalf(".claude/ gained entries: %v", entries)
	}
}

// #1554 (1): `fmt` with the write flag rewrites its path in place, so it is a
// write and goes through the same protection as the write/edit ops — before
// anything executes.
func TestPolicyTool_FmtWriteHonoursDenyWrite(t *testing.T) {
	f := newFixture(t, "fs_deny_write = [\".claude/**\", \"*.gen.ail\"]\n")
	for _, d := range []string{".claude", ".git"} {
		if err := os.MkdirAll(filepath.Join(f.sandbox, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{".claude/x.ail", ".git/x.ail", "sub/y.gen.ail"} {
		if err := os.WriteFile(filepath.Join(f.sandbox, p), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ path, want string }{
		{".claude/x.ail", "fs_deny_write"},
		{".CLAUDE/x.ail", "fs_deny_write"},
		{"sub/Y.GEN.AIL", "fs_deny_write"},
		{".git/x.ail", ".git"},
		{".GIT/x.ail", ".git"},
	} {
		for _, val := range []string{"", "true"} {
			r := f.host.Dispatch(Request{Op: "fmt", Path: tc.path, Flags: map[string]string{"write": val}})
			if r.OK || !strings.Contains(r.Refused, tc.want) {
				t.Errorf("fmt --write=%q %s: want refusal naming %s, got %+v", val, tc.path, tc.want, r)
			}
		}
	}
	if len(f.runs) != 0 {
		t.Fatalf("a refused fmt --write executed: %v", f.runs)
	}
	// Reading a protected file through fmt (stdout) or --check writes nothing.
	if r := f.host.Dispatch(Request{Op: "fmt", Path: ".claude/x.ail"}); !r.OK {
		t.Errorf("fmt to stdout on a protected file is a read: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "fmt", Path: ".claude/x.ail", Flags: map[string]string{"check": ""}}); !r.OK {
		t.Errorf("fmt --check on a protected file is a read: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "fmt", Path: "in.ail", Flags: map[string]string{"write": ""}}); !r.OK {
		t.Errorf("fmt --write on an ordinary file: %+v", r)
	}
	if got := strings.Join(f.runs[len(f.runs)-1], " "); got != f.sandbox+" fmt --write in.ail" {
		t.Errorf("argv = %s", got)
	}
}

// #1554 (1): a boolean flag's value is not ignored — "false" used to mean
// --write. Only "" or "true" turn a boolean flag on; anything else is refused.
func TestPolicyTool_BooleanFlagValues(t *testing.T) {
	f := newFixture(t, "")
	for _, val := range []string{"false", "0", "no", "FALSE", "yes", "--evil"} {
		r := f.host.Dispatch(Request{Op: "fmt", Path: "in.ail", Flags: map[string]string{"write": val}})
		if r.OK || !strings.Contains(r.Refused, "boolean") {
			t.Errorf("fmt write=%q: want a boolean-flag refusal, got %+v", val, r)
		}
		r = f.host.Dispatch(Request{Op: "check", Path: "in.ail", Flags: map[string]string{"json": val}})
		if r.OK || !strings.Contains(r.Refused, "boolean") {
			t.Errorf("check json=%q: want a boolean-flag refusal, got %+v", val, r)
		}
	}
	if len(f.runs) != 0 {
		t.Fatalf("nothing may have executed: %v", f.runs)
	}
	if r := f.host.Dispatch(Request{Op: "check", Path: "in.ail", Flags: map[string]string{"json": "true"}}); !r.OK {
		t.Fatalf("json=true: %+v", r)
	}
	if got := strings.Join(f.runs[0], " "); got != f.sandbox+" check --json in.ail" {
		t.Fatalf("argv = %s", got)
	}
}

// Every CLI op that writes a known path inside the sandbox is checked:
// `lock` writes ailang.lock, `design_quorum` writes its artifact dir.
func TestPolicyTool_CLIWritesHonourDenyWrite(t *testing.T) {
	f := newFixture(t, "cli_allow = [\"lock\", \"design-quorum\"]\nfs_deny_write = [\".ailang/**\", \"AILANG.LOCK\"]\n")
	if err := os.WriteFile(filepath.Join(f.sandbox, "doc.md"), []byte("# d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := f.host.Dispatch(Request{Op: "lock"}); r.OK || !strings.Contains(r.Refused, "fs_deny_write") {
		t.Errorf("lock with ailang.lock deny-listed: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "design_quorum", Path: "doc.md"}); r.OK || !strings.Contains(r.Refused, "fs_deny_write") {
		t.Errorf("design_quorum with .ailang/** deny-listed: %+v", r)
	}
	if len(f.runs) != 0 {
		t.Fatalf("nothing may have executed: %v", f.runs)
	}
	// Without the deny entries both run.
	f = newFixture(t, "cli_allow = [\"lock\", \"design-quorum\"]\n")
	if err := os.WriteFile(filepath.Join(f.sandbox, "doc.md"), []byte("# d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := f.host.Dispatch(Request{Op: "lock"}); !r.OK {
		t.Errorf("lock: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "design_quorum", Path: "doc.md"}); !r.OK {
		t.Errorf("design_quorum: %+v", r)
	}
}

// The artifact dir the deny check assumes is design-quorum's real default.
func TestPolicyTool_QuorumArtifactDirInSync(t *testing.T) {
	if quorumArtifactDir != quorum.ArtifactDir {
		t.Fatalf("policytool assumes %q, design-quorum writes %q", quorumArtifactDir, quorum.ArtifactDir)
	}
}

// #1554 (1): request keys are exact. encoding/json matches keys
// case-insensitively and lets a later duplicate win (merging duplicate
// objects), so `FLAGS` aliased `flags` and a second `flags` object merged
// into the first. A strict decode refuses all of it.
func TestDecodeRequest_Strict(t *testing.T) {
	bad := map[string]string{
		"aliased flags":       `{"op":"fmt","path":"x.ail","FLAGS":{"write":""}}`,
		"aliased op":          `{"OP":"read","path":"x"}`,
		"aliased path":        `{"op":"read","Path":"x"}`,
		"duplicate flags":     `{"op":"fmt","path":"x.ail","flags":{"check":""},"flags":{"write":""}}`,
		"duplicate op":        `{"op":"read","op":"write","path":"x"}`,
		"duplicate flag name": `{"op":"fmt","path":"x.ail","flags":{"check":"","check":""}}`,
		"unknown field":       `{"op":"read","path":"x","argv":["rm"]}`,
		"trailing data":       `{"op":"read","path":"x"}{"op":"write"}`,
		"not an object":       `["read"]`,
		"flag not a string":   `{"op":"fmt","path":"x.ail","flags":{"write":true}}`,
		"not json":            `op=read`,
	}
	for name, raw := range bad {
		if _, err := DecodeRequest([]byte(raw)); err == nil {
			t.Errorf("%s: %s decoded without error", name, raw)
		}
	}
	req, err := DecodeRequest([]byte(`{"op":"fmt","path":"x.ail","flags":{"check":"","json":"true"}}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if req.Op != "fmt" || req.Path != "x.ail" || len(req.Flags) != 2 {
		t.Fatalf("decoded %+v", req)
	}
	req, err = DecodeRequest([]byte(`{"op":"edit","path":"a","old_text":"o","new_text":"n","content":"","module":"","query":"","package":""}`))
	if err != nil || req.OldText != "o" || req.NewText != "n" {
		t.Fatalf("every Request field must decode: %+v %v", req, err)
	}
}

// The strict decoder names its keys explicitly; every json name on Request
// must be one of them, so adding a field cannot silently make it undecodable.
func TestDecodeRequest_CoversEveryField(t *testing.T) {
	rt := reflect.TypeOf(Request{})
	for i := 0; i < rt.NumField(); i++ {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		val := `"v"`
		if name == "flags" {
			val = `{}`
		}
		if _, err := DecodeRequest([]byte(`{"` + name + `":` + val + `}`)); err != nil {
			t.Errorf("Request field %s (json %q) is not decoded: %v", rt.Field(i).Name, name, err)
		}
	}
}
