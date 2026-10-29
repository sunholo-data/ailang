package effects

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// AILANG World 2026-10-17 (v0.54.11): fsCheckMutation matched the spelled
// path, so a program wrote a deny-listed file through an in-root symlink.
func TestFSDenyWrite_ThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	sandbox := t.TempDir()
	if real, err := filepath.EvalSymlinks(sandbox); err == nil {
		sandbox = real
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(sandbox, "tests"), 0o755))
	must(os.MkdirAll(filepath.Join(sandbox, "pkg"), 0o755))
	must(os.WriteFile(filepath.Join(sandbox, "tests", "test_x.py"), []byte("orig"), 0o644))
	must(os.WriteFile(filepath.Join(sandbox, "pkg", "a_test.go"), []byte("orig"), 0o644))
	must(os.Symlink("tests", filepath.Join(sandbox, "lnk")))
	must(os.Symlink("pkg/a_test.go", filepath.Join(sandbox, "alias.txt")))
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = sandbox
	ctx.Env.DenyWrite = []string{"tests/**", "*_test.go"}
	ctx.Grant(NewCapability("FS"))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })

	for _, p := range []string{"lnk/new5.py", "lnk/test_x.py", "alias.txt"} {
		for _, op := range []string{"writeFile", "appendFile"} {
			if _, err := Call(ctx, "FS", op, []eval.Value{&eval.StringValue{Value: p}, &eval.StringValue{Value: "x"}}); err == nil || !strings.Contains(err.Error(), "E_FS_PROTECTED") {
				t.Errorf("%s(%s) must be E_FS_PROTECTED, got %v", op, p, err)
			}
		}
	}
	if _, err := Call(ctx, "FS", "removeFile", []eval.Value{&eval.StringValue{Value: "lnk/test_x.py"}}); err == nil || !strings.Contains(err.Error(), "E_FS_PROTECTED") {
		t.Errorf("removeFile through a linked parent: %v", err)
	}
	for _, p := range []string{"tests/test_x.py", "pkg/a_test.go"} {
		if b, _ := os.ReadFile(filepath.Join(sandbox, p)); string(b) != "orig" {
			t.Errorf("%s changed: %q", p, b)
		}
	}
	if _, err := os.Lstat(filepath.Join(sandbox, "tests", "new5.py")); err == nil {
		t.Error("tests/new5.py was created through the link")
	}
	if _, err := Call(ctx, "FS", "writeFile", []eval.Value{&eval.StringValue{Value: "ok.txt"}, &eval.StringValue{Value: "x"}}); err != nil {
		t.Fatalf("ordinary write: %v", err)
	}
}
