package builtins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M1 — the archive builtins (std/zip, std/gzip,
// std/tar, zip_xml) used to join ctx.Env.Sandbox themselves and hand the
// joined path to os.*: the F1/F2 escape by another door. They now go through
// the same confined root as the FS ops; these tests hold that.

type archiveTree struct {
	tmp, sandbox, sentinel string
}

// newArchiveTree lays out <tmp>/{marker.zip, marker.tar, marker.gz, sandbox/}
// with sandbox/link.* -> ../marker.* and sandbox/linkdir -> <tmp>.
func newArchiveTree(t *testing.T) *archiveTree {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	tmp := t.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	a := &archiveTree{tmp: tmp, sandbox: filepath.Join(tmp, "sandbox"), sentinel: "ARCHIVE-SENTINEL-" + filepath.Base(tmp)}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(a.sandbox, 0o755))
	// zip with one entry
	createTestZip(t, tmp, map[string][]byte{"file.txt": []byte(a.sentinel)})
	must(os.Rename(filepath.Join(tmp, "test.zip"), filepath.Join(tmp, "marker.zip")))
	// tar with one entry
	var tb bytes.Buffer
	tw := tar.NewWriter(&tb)
	must(tw.WriteHeader(&tar.Header{Name: "file.txt", Mode: 0o644, Size: int64(len(a.sentinel))}))
	_, err := tw.Write([]byte(a.sentinel))
	must(err)
	must(tw.Close())
	must(os.WriteFile(filepath.Join(tmp, "marker.tar"), tb.Bytes(), 0o644))
	// gzip
	var gb bytes.Buffer
	gw := gzip.NewWriter(&gb)
	_, err = gw.Write([]byte(a.sentinel))
	must(err)
	must(gw.Close())
	must(os.WriteFile(filepath.Join(tmp, "marker.gz"), gb.Bytes(), 0o644))
	must(os.Symlink("../marker.zip", filepath.Join(a.sandbox, "link.zip")))
	must(os.Symlink("../marker.tar", filepath.Join(a.sandbox, "link.tar")))
	must(os.Symlink("../marker.gz", filepath.Join(a.sandbox, "link.gz")))
	must(os.Symlink(tmp, filepath.Join(a.sandbox, "linkdir")))
	return a
}

func (a *archiveTree) ctx(t *testing.T) *effects.EffContext {
	t.Helper()
	ctx := makeTestCtx(t)
	ctx.Env.Sandbox = a.sandbox
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	return ctx
}

func sv(s string) eval.Value { return &eval.StringValue{Value: s} }

// mustDenyResult: result is Err and carries no sentinel; a Go error is a
// denial too.
func mustDenyResult(t *testing.T, what, sentinel string, result eval.Value, err error) {
	t.Helper()
	if err != nil {
		return
	}
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("%s: expected TaggedValue, got %T", what, result)
	}
	if tv.CtorName != "Err" {
		t.Errorf("CONTAINMENT FAILURE: %s returned %s", what, tv.CtorName)
	}
	for _, f := range tv.Fields {
		if s, ok := f.(*eval.StringValue); ok && strings.Contains(s.Value, sentinel) {
			t.Errorf("CONTAINMENT FAILURE: %s leaked the sentinel", what)
		}
		if l, ok := f.(*eval.ListValue); ok {
			for _, e := range l.Elements {
				if s, ok := e.(*eval.StringValue); ok && strings.Contains(s.Value, sentinel) {
					t.Errorf("CONTAINMENT FAILURE: %s leaked the sentinel", what)
				}
			}
		}
	}
}

func TestArchiveContainment_Reads(t *testing.T) {
	type call struct {
		name string
		fn   func(*effects.EffContext, []eval.Value) (eval.Value, error)
		args func(path string) []eval.Value
		ext  string
	}
	calls := []call{
		{"zipListEntries", zipListEntriesImpl, func(p string) []eval.Value { return []eval.Value{sv(p)} }, "zip"},
		{"zipReadEntry", zipReadEntryImpl, func(p string) []eval.Value { return []eval.Value{sv(p), sv("file.txt")} }, "zip"},
		{"zipReadEntryBytes", zipReadEntryBytesImpl, func(p string) []eval.Value { return []eval.Value{sv(p), sv("file.txt")} }, "zip"},
		{"tarListEntries", tarListEntriesImpl, func(p string) []eval.Value { return []eval.Value{sv(p)} }, "tar"},
		{"tarReadEntry", tarReadEntryImpl, func(p string) []eval.Value { return []eval.Value{sv(p), sv("file.txt")} }, "tar"},
		{"tarReadEntryBytes", tarReadEntryBytesImpl, func(p string) []eval.Value { return []eval.Value{sv(p), sv("file.txt")} }, "tar"},
		{"gzipDecompressFile", gzipDecompressFileImpl, func(p string) []eval.Value { return []eval.Value{sv(p)} }, "gz"},
	}
	for _, c := range calls {
		for _, route := range []struct{ name, tpl string }{
			{"traversal", "../marker.%s"},
			{"symlink-file", "link.%s"},
			{"symlink-dir", "linkdir/marker.%s"},
			{"absolute", "%%ABS%%/marker.%s"},
		} {
			t.Run(c.name+"/"+route.name, func(t *testing.T) {
				a := newArchiveTree(t)
				p := strings.Replace(strings.Replace(route.tpl, "%s", c.ext, 1), "%ABS%", a.tmp, 1)
				result, err := c.fn(a.ctx(t), c.args(p))
				mustDenyResult(t, c.name+"("+p+")", a.sentinel, result, err)
			})
		}
	}
}

func TestArchiveContainment_Writes(t *testing.T) {
	entries := &eval.ListValue{Elements: []eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{
		"name": sv("x.txt"), "content": sv("x"),
	}}}}
	for _, route := range []string{"../out.zip", "linkdir/out.zip"} {
		t.Run("zipCreateArchive/"+route, func(t *testing.T) {
			a := newArchiveTree(t)
			result, err := zipCreateArchiveImpl(a.ctx(t), []eval.Value{sv(route), entries})
			mustDenyResult(t, "zipCreateArchive", a.sentinel, result, err)
			if _, err := os.Stat(filepath.Join(a.tmp, "out.zip")); err == nil {
				t.Error("CONTAINMENT FAILURE: out.zip was created outside the sandbox")
			}
		})
	}
	for _, route := range []string{"..", "linkdir/extracted", "../extracted"} {
		t.Run("tarExtractAll/"+route, func(t *testing.T) {
			a := newArchiveTree(t)
			// The archive itself is inside the sandbox; only the destination escapes.
			data, _ := os.ReadFile(filepath.Join(a.tmp, "marker.tar"))
			if err := os.WriteFile(filepath.Join(a.sandbox, "in.tar"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			result, err := tarExtractAllImpl(a.ctx(t), []eval.Value{sv("in.tar"), sv(route)})
			mustDenyResult(t, "tarExtractAll", a.sentinel, result, err)
			for _, p := range []string{filepath.Join(a.tmp, "file.txt"), filepath.Join(a.tmp, "extracted", "file.txt")} {
				if _, err := os.Stat(p); err == nil {
					t.Errorf("CONTAINMENT FAILURE: %s was written outside the sandbox", p)
				}
			}
		})
	}
}

// Positive control: the same builtins work on in-root archives, and the
// reported extraction paths keep their sandbox-prefixed shape.
func TestArchiveContainment_PositiveControl(t *testing.T) {
	a := newArchiveTree(t)
	ctx := a.ctx(t)
	data, _ := os.ReadFile(filepath.Join(a.tmp, "marker.tar"))
	if err := os.WriteFile(filepath.Join(a.sandbox, "in.tar"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := tarExtractAllImpl(ctx, []eval.Value{sv("in.tar"), sv("out")})
	if err != nil {
		t.Fatal(err)
	}
	list := assertOk(t, result).(*eval.ListValue)
	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 written path, got %v", list.Elements)
	}
	want := filepath.Join(a.sandbox, "out", "file.txt")
	if got := list.Elements[0].(*eval.StringValue).Value; got != want {
		t.Errorf("written path = %q, want %q", got, want)
	}
	if b, err := os.ReadFile(want); err != nil || string(b) != a.sentinel {
		t.Errorf("extracted content: %q %v", b, err)
	}
	zdata, _ := os.ReadFile(filepath.Join(a.tmp, "marker.zip"))
	if err := os.WriteFile(filepath.Join(a.sandbox, "in.zip"), zdata, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := zipReadEntryImpl(ctx, []eval.Value{sv("in.zip"), sv("file.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if got := assertOk(t, r).(*eval.StringValue).Value; got != a.sentinel {
		t.Errorf("zip read = %q", got)
	}
}
