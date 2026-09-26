package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M7 — fs_deny_write: operator-protected paths
// inside the sandbox (the artifact's own supply chain: workflows, Makefile,
// extensions) are read-only to every mutating FS op.
func TestFSDenyWrite_PatternsProtectInsideRoot(t *testing.T) {
	sandbox := t.TempDir()
	if real, err := filepath.EvalSymlinks(sandbox); err == nil {
		sandbox = real
	}
	for _, d := range []string{".github/workflows", ".pi/extensions", "src"} {
		if err := os.MkdirAll(filepath.Join(sandbox, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{".github/workflows/ci.yml", ".pi/extensions/x.ts", "Makefile", "src/a.ail", "deploy.yml"} {
		if err := os.WriteFile(filepath.Join(sandbox, f), []byte("orig"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = sandbox
	ctx.Env.DenyWrite = []string{".github/**", ".pi/**", "Makefile", "*.yml"}
	ctx.Grant(NewCapability("FS"))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })

	denied := []string{".github/workflows/ci.yml", ".github/new.yml", ".pi/extensions/x.ts", ".pi/new.ts", "Makefile", "deploy.yml", "src/../Makefile", filepath.Join(sandbox, "Makefile")}
	for _, p := range denied {
		for _, op := range []string{"writeFile", "appendFile"} {
			if _, err := Call(ctx, "FS", op, []eval.Value{&eval.StringValue{Value: p}, &eval.StringValue{Value: "x"}}); err == nil || !strings.Contains(err.Error(), "E_FS_PROTECTED") {
				t.Errorf("%s(%s) must be E_FS_PROTECTED, got %v", op, p, err)
			}
		}
		if v, _ := Call(ctx, "FS", "writeFileResult", []eval.Value{&eval.StringValue{Value: p}, &eval.StringValue{Value: "x"}}); v == nil || v.(*eval.TaggedValue).CtorName != "Err" {
			t.Errorf("writeFileResult(%s) must be Err, got %v", p, v)
		}
		if _, err := Call(ctx, "FS", "removeFile", []eval.Value{&eval.StringValue{Value: p}}); err == nil || !strings.Contains(err.Error(), "E_FS_PROTECTED") {
			t.Errorf("removeFile(%s): %v", p, err)
		}
	}
	if _, err := Call(ctx, "FS", "mkdirAll", []eval.Value{&eval.StringValue{Value: ".github/evil"}}); err == nil {
		t.Error("mkdirAll under a protected prefix must be refused")
	}
	if _, err := Call(ctx, "FS", "renameFile", []eval.Value{&eval.StringValue{Value: "src/a.ail"}, &eval.StringValue{Value: ".github/workflows/x.yml"}}); err == nil {
		t.Error("rename INTO a protected path must be refused")
	}
	if _, err := Call(ctx, "FS", "renameFile", []eval.Value{&eval.StringValue{Value: "Makefile"}, &eval.StringValue{Value: "src/m"}}); err == nil {
		t.Error("rename OUT of a protected path must be refused")
	}
	// Allowed: ordinary files, and reading protected ones.
	if _, err := Call(ctx, "FS", "writeFile", []eval.Value{&eval.StringValue{Value: "src/a.ail"}, &eval.StringValue{Value: "new"}}); err != nil {
		t.Fatalf("ordinary write: %v", err)
	}
	if _, err := Call(ctx, "FS", "writeFile", []eval.Value{&eval.StringValue{Value: "notes.md"}, &eval.StringValue{Value: "x"}}); err != nil {
		t.Fatalf("*.yml must not match notes.md: %v", err)
	}
	if v, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: "Makefile"}}); err != nil || v.(*eval.StringValue).Value != "orig" {
		t.Fatalf("reading a protected file: %v %v", v, err)
	}
	for _, f := range []string{".github/workflows/ci.yml", ".pi/extensions/x.ts", "Makefile", "deploy.yml"} {
		if b, _ := os.ReadFile(filepath.Join(sandbox, f)); string(b) != "orig" {
			t.Errorf("%s changed: %q", f, b)
		}
	}
}
