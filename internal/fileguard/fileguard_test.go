package fileguard

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func tree(t *testing.T) (tmp string, r *Root) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows; restricted mode is refused there")
	}
	tmp = t.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	root := filepath.Join(tmp, "root")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "sub"), 0o755))
	must(os.WriteFile(filepath.Join(tmp, "outside.txt"), []byte("OUT"), 0o644))
	must(os.WriteFile(filepath.Join(root, "in.txt"), []byte("IN"), 0o644))
	must(os.WriteFile(filepath.Join(root, "sub", "deep.txt"), []byte("DEEP"), 0o644))
	must(os.Symlink("../outside.txt", filepath.Join(root, "out.lnk")))
	must(os.Symlink("sub/deep.txt", filepath.Join(root, "ok.lnk")))
	must(os.Symlink(filepath.Join(root, "in.txt"), filepath.Join(root, "abs.lnk")))
	r, err := Open(root)
	must(err)
	t.Cleanup(func() { _ = r.Close() })
	return tmp, r
}

func TestRel(t *testing.T) {
	tmp, r := tree(t)
	root := filepath.Join(tmp, "root")
	cases := []struct {
		in, want string
		escape   bool
	}{
		{"", ".", false},
		{"in.txt", "in.txt", false},
		{"./sub/../in.txt", "in.txt", false},
		{"../x", "../x", false}, // lexically kept; the kernel refuses it
		{root, ".", false},
		{filepath.Join(root, "sub", "deep.txt"), filepath.Join("sub", "deep.txt"), false},
		{filepath.Join(tmp, "outside.txt"), "", true},
		{root + "2/x", "", true}, // sibling prefix
		{"/etc/passwd", "", true},
		{`C:\x`, "", true},
	}
	for _, c := range cases {
		got, err := r.Rel(c.in)
		if c.escape {
			var ee *EscapeError
			if !errors.As(err, &ee) {
				t.Errorf("Rel(%q): want EscapeError, got %v (%q)", c.in, err, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Rel(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestOpenDenies(t *testing.T) {
	tmp, r := tree(t)
	for _, p := range []string{"../outside.txt", "sub/../../outside.txt", "out.lnk", "abs.lnk", filepath.Join(tmp, "outside.txt")} {
		f, err := r.Open(p)
		if err == nil {
			f.Close()
			t.Errorf("Open(%q) succeeded", p)
			continue
		}
		if !IsEscape(err) {
			t.Errorf("Open(%q): error is not an escape: %v", p, err)
		}
	}
}

func TestOpenAllows(t *testing.T) {
	tmp, r := tree(t)
	for p, want := range map[string]string{
		"in.txt":        "IN",
		"ok.lnk":        "DEEP",
		"sub/../in.txt": "IN",
		filepath.Join(tmp, "root", "sub", "deep.txt"): "DEEP",
	} {
		f, err := r.Open(p)
		if err != nil {
			t.Errorf("Open(%q): %v", p, err)
			continue
		}
		buf := make([]byte, 16)
		n, _ := f.Read(buf)
		f.Close()
		if string(buf[:n]) != want {
			t.Errorf("Open(%q) read %q, want %q", p, buf[:n], want)
		}
	}
}

func TestWriteMkdirRemoveRename(t *testing.T) {
	tmp, r := tree(t)
	// Writes outside are refused with no side effect.
	if _, err := r.OpenFile("../new.txt", os.O_CREATE|os.O_WRONLY, 0o644); !IsEscape(err) {
		t.Errorf("OpenFile(../new.txt): %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "new.txt")); err == nil {
		t.Error("../new.txt was created outside the root")
	}
	if err := r.Mkdir("../d", 0o755); !IsEscape(err) {
		t.Errorf("Mkdir(../d): %v", err)
	}
	if err := r.MkdirAll("out.lnk/x", 0o755); err == nil {
		t.Error("MkdirAll through an escaping symlink succeeded")
	}
	if err := r.MkdirAll("a/b/c", 0o755); err != nil {
		t.Errorf("MkdirAll(a/b/c): %v", err)
	}
	if err := r.Rename("in.txt", "../stolen.txt"); !IsEscape(err) {
		t.Errorf("Rename to outside: %v", err)
	}
	if err := r.Rename("../outside.txt", "stolen.txt"); !IsEscape(err) {
		t.Errorf("Rename from outside: %v", err)
	}
	if err := r.Rename("in.txt", "a/b/c/moved.txt"); err != nil {
		t.Errorf("in-root Rename: %v", err)
	}
	// Removing a symlink entry removes the link, never the target.
	if err := r.Remove("out.lnk"); err != nil {
		t.Errorf("Remove(out.lnk): %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "outside.txt")); err != nil {
		t.Errorf("outside.txt must survive: %v", err)
	}
	if err := r.Remove("../outside.txt"); !IsEscape(err) {
		t.Errorf("Remove(../outside.txt): %v", err)
	}
	// Lstat sees the link; Stat through an escaping link is refused.
	if fi, err := r.Lstat("abs.lnk"); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Lstat(abs.lnk) = %v, %v", fi, err)
	}
	if _, err := r.Stat("abs.lnk"); !IsEscape(err) {
		t.Errorf("Stat(abs.lnk): %v", err)
	}
}

func TestReadDirSorted(t *testing.T) {
	_, r := tree(t)
	entries, err := r.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Name() > entries[i].Name() {
			t.Fatalf("not sorted: %v", entries)
		}
	}
	if _, err := r.ReadDir(".."); !IsEscape(err) {
		t.Errorf("ReadDir(..): %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	_, r := tree(t)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	var nilRoot *Root
	if err := nilRoot.Close(); err != nil {
		t.Fatalf("nil Close: %v", err)
	}
}

func TestOpenRejectsMissingOrFile(t *testing.T) {
	tmp, _ := tree(t)
	if _, err := Open(filepath.Join(tmp, "nope")); err == nil {
		t.Error("Open(missing) succeeded")
	}
	if _, err := Open(filepath.Join(tmp, "outside.txt")); err == nil {
		t.Error("Open(file) succeeded")
	}
	if _, err := Open(""); err == nil {
		t.Error("Open(\"\") succeeded")
	}
}
