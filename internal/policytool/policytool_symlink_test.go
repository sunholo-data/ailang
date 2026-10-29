package policytool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AILANG World 2026-10-17 (v0.54.11): fs_deny_write was matched against the
// spelled path only, so an in-root symlink reached a protected file.
func TestPolicyTool_DenyWriteThroughSymlink(t *testing.T) {
	f := newFixture(t, "fs_deny_write = [\"tests/**\", \"*_test.go\", \"**/*.test.ts\"]\n")
	for _, d := range []string{"tests", "pkg", "src"} {
		if err := os.MkdirAll(filepath.Join(f.sandbox, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeSandbox(t, f, "tests/test_x.py", "orig")
	writeSandbox(t, f, "pkg/a_test.go", "orig")
	for name, target := range map[string]string{"lnk": "tests", "alias.txt": "pkg/a_test.go", "dangling.py": "tests/new.py"} {
		if err := os.Symlink(target, filepath.Join(f.sandbox, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"tests/new.py", "lnk/new5.py", "lnk/test_x.py", "alias.txt", "dangling.py", "src/foo.test.ts", "src/deep/x.test.ts", "a.test.ts"} {
		if r := f.host.Dispatch(Request{Op: "write", Path: p, Content: "clobbered"}); r.OK || !strings.Contains(r.Refused, "fs_deny_write") {
			t.Errorf("write %s: %+v", p, r)
		}
	}
	if r := f.host.Dispatch(Request{Op: "edit", Path: "lnk/test_x.py", OldText: "orig", NewText: "x"}); r.OK || !strings.Contains(r.Refused, "resolves to tests/test_x.py") {
		t.Errorf("edit through a link must name the resolved path: %+v", r)
	}
	for _, p := range []string{"tests/test_x.py", "pkg/a_test.go"} {
		if b, _ := os.ReadFile(filepath.Join(f.sandbox, p)); string(b) != "orig" {
			t.Errorf("%s changed: %q", p, b)
		}
	}
	for _, p := range []string{"tests/new.py", "tests/new5.py"} {
		if _, err := os.Lstat(filepath.Join(f.sandbox, p)); err == nil {
			t.Errorf("%s was created", p)
		}
	}
	if r := f.host.Dispatch(Request{Op: "write", Path: "src/ok.txt", Content: "fine"}); !r.OK {
		t.Errorf("ordinary write: %+v", r)
	}
	f.outsideIntact(t)
}

func TestPolicyTool_DenyWriteHardLink(t *testing.T) {
	f := newFixture(t, "fs_deny_write = [\"*_test.go\"]\n")
	writeSandbox(t, f, "a_test.go", "orig")
	if err := os.Link(filepath.Join(f.sandbox, "a_test.go"), filepath.Join(f.sandbox, "hard.txt")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	if r := f.host.Dispatch(Request{Op: "write", Path: "hard.txt", Content: "x"}); r.OK || !strings.Contains(r.Refused, "hard link") {
		t.Errorf("write through a hard link: %+v", r)
	}
	if b, _ := os.ReadFile(filepath.Join(f.sandbox, "a_test.go")); string(b) != "orig" {
		t.Errorf("a_test.go changed: %q", b)
	}
}

func TestPolicyTool_Delete(t *testing.T) {
	f := newFixture(t, "fs_deny_write = [\"tests/**\", \"Makefile\"]\n")
	if err := os.MkdirAll(filepath.Join(f.sandbox, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSandbox(t, f, "tests/t.py", "orig")
	writeSandbox(t, f, "Makefile", "orig")
	writeSandbox(t, f, "helper.ail", "tmp")
	if err := os.Symlink("tests", filepath.Join(f.sandbox, "lnk")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("Makefile", filepath.Join(f.sandbox, "mk")); err != nil {
		t.Fatal(err)
	}
	if r := f.host.Dispatch(Request{Op: "delete", Path: "helper.ail"}); !r.OK {
		t.Fatalf("delete: %+v", r)
	}
	if _, err := os.Lstat(filepath.Join(f.sandbox, "helper.ail")); !os.IsNotExist(err) {
		t.Fatalf("helper.ail still there: %v", err)
	}
	// The link entry goes; its protected target stays.
	if r := f.host.Dispatch(Request{Op: "delete", Path: "mk"}); !r.OK {
		t.Fatalf("delete a symlink entry: %+v", r)
	}
	for p, want := range map[string]string{
		"tests/t.py": "fs_deny_write", "lnk/t.py": "fs_deny_write", "Makefile": "fs_deny_write",
		"tests": "fs_deny_write", "sub": "directory", ".git/config": ".git", "nope.txt": "no such file",
		"../marker.txt": "escapes", "link.txt/x": "",
	} {
		r := f.host.Dispatch(Request{Op: "delete", Path: p})
		if r.OK || !strings.Contains(r.Refused, want) {
			t.Errorf("delete %s: want refusal containing %q, got %+v", p, want, r)
		}
	}
	for _, p := range []string{"tests/t.py", "Makefile", "sub/deep.txt"} {
		if _, err := os.Stat(filepath.Join(f.sandbox, p)); err != nil {
			t.Errorf("%s removed: %v", p, err)
		}
	}
	// An EMPTY directory is refused too: delete removes files only.
	if err := os.Mkdir(filepath.Join(f.sandbox, "emptyd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r := f.host.Dispatch(Request{Op: "delete", Path: "emptyd"}); r.OK || !strings.Contains(r.Refused, "is a directory") {
		t.Errorf("delete of an empty directory: %+v", r)
	}
	if fi, err := os.Stat(filepath.Join(f.sandbox, "emptyd")); err != nil || !fi.IsDir() {
		t.Errorf("emptyd removed: %v", err)
	}
	if r := f.host.Dispatch(Request{Op: "delete"}); r.OK {
		t.Error("delete without a path must be refused")
	}
	f.outsideIntact(t)
}
