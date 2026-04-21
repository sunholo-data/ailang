package builtins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Fixture builders
// ============================================================================

type tarFixtureEntry struct {
	name     string
	body     []byte
	isDir    bool
	typeflag byte   // optional: override typeflag (e.g. TypeSymlink)
	linkname string // for symlinks/hardlinks
}

// writeTarFixture builds a tar file at path containing the given entries.
func writeTarFixture(t *testing.T, path string, entries []tarFixtureEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	writeFixtureTo(t, tw, entries)
}

// writeTarGzFixture builds a gzipped tar file at path.
func writeTarGzFixture(t *testing.T, path string, entries []tarFixtureEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	writeFixtureTo(t, tw, entries)
}

func writeFixtureTo(t *testing.T, tw *tar.Writer, entries []tarFixtureEntry) {
	t.Helper()
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0o644,
			Size:     int64(len(e.body)),
			Typeflag: tar.TypeReg,
		}
		if e.isDir {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0o755
			hdr.Size = 0
		}
		if e.typeflag != 0 {
			hdr.Typeflag = e.typeflag
			hdr.Linkname = e.linkname
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %s: %v", e.name, err)
		}
		if hdr.Typeflag == tar.TypeReg && len(e.body) > 0 {
			if _, err := tw.Write(e.body); err != nil {
				t.Fatalf("write body %s: %v", e.name, err)
			}
		}
	}
}

// ============================================================================
// listEntries
// ============================================================================

func TestTarListEntries_Happy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.tar")
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "README.md", body: []byte("hello arxiv")},
		{name: "src/", isDir: true},
		{name: "src/main.tex", body: []byte("\\documentclass{article}")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarListEntriesImpl(ctx, []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv, ok := res.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %+v", res)
	}
	list, ok := tv.Fields[0].(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", tv.Fields[0])
	}
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(list.Elements))
	}
	// First entry: README.md, size=11 ("hello arxiv"), not dir
	rec := list.Elements[0].(*eval.RecordValue)
	if rec.Fields["name"].(*eval.StringValue).Value != "README.md" {
		t.Errorf("wrong name: %v", rec.Fields["name"])
	}
	if rec.Fields["size"].(*eval.IntValue).Value != 11 {
		t.Errorf("wrong size: %v", rec.Fields["size"])
	}
	if rec.Fields["isDir"].(*eval.BoolValue).Value {
		t.Errorf("README.md should not be isDir")
	}
	// src/ is a directory
	dirRec := list.Elements[1].(*eval.RecordValue)
	if !dirRec.Fields["isDir"].(*eval.BoolValue).Value {
		t.Errorf("src/ should be isDir")
	}
}

func TestTarListEntries_Unicode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unicode.tar")
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "日本語/文書.txt", body: []byte("unicode content")},
		{name: "τέχνη.tex", body: []byte("\\title{τέχνη}")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarListEntriesImpl(ctx, []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv := res.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s: %v", tv.CtorName, tv.Fields[0])
	}
	list := tv.Fields[0].(*eval.ListValue)
	gotName := list.Elements[0].(*eval.RecordValue).Fields["name"].(*eval.StringValue).Value
	if gotName != "日本語/文書.txt" {
		t.Errorf("unicode name mangled: got %q", gotName)
	}
}

func TestTarListEntries_MissingFile(t *testing.T) {
	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarListEntriesImpl(ctx, []eval.Value{&eval.StringValue{Value: "/nonexistent/foo.tar"}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "cannot open")
}

func TestTarListEntries_TooManyEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.tar")

	// Build a tar with > 10k entries. Each entry has an empty body so the file
	// stays reasonably small (a few hundred KB).
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tw := tar.NewWriter(f)
	for i := 0; i < tarMaxEntries+5; i++ {
		hdr := &tar.Header{
			Name:     "file" + strings.Repeat("x", 4) + string(rune('a'+(i%26))) + ".txt",
			Mode:     0o644,
			Size:     0,
			Typeflag: tar.TypeReg,
		}
		// Force unique names so tar doesn't dedupe
		hdr.Name = "entry_" + itoa(i) + ".txt"
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("hdr %d: %v", i, err)
		}
	}
	_ = tw.Close()
	_ = f.Close()

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarListEntriesImpl(ctx, []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "too many entries")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}

// ============================================================================
// readEntry / readEntryBytes
// ============================================================================

func TestTarReadEntry_Happy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mix.tar")
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "notes.txt", body: []byte("just some notes\n")},
		{name: "data/x.tex", body: []byte("\\begin{document}")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadEntryImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "data/x.tex"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	got := unwrapOkString(t, res)
	if got != "\\begin{document}" {
		t.Errorf("got %q", got)
	}
}

func TestTarReadEntry_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tar")
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "a.txt", body: []byte("a")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadEntryImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "does-not-exist"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "entry not found")
}

func TestTarReadEntry_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tar")
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "somedir/", isDir: true},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadEntryImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "somedir/"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "directory")
}

func TestTarReadEntry_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tar")
	writeTarFixture(t, path, []tarFixtureEntry{{name: "a.txt", body: []byte("a")}})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadEntryImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "../../etc/passwd"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "path traversal")
}

func TestTarReadEntryBytes_Binary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin.tar")
	// Use bytes that are NOT valid UTF-8 to prove base64 preserves them.
	binary := []byte{0xff, 0xfe, 0x00, 0x42, 0xaa, 0xbb, 0xcc}
	writeTarFixture(t, path, []tarFixtureEntry{
		{name: "blob.bin", body: binary},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadEntryBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "blob.bin"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	got := mustB64Decode(t, unwrapOkString(t, res))
	if !bytes.Equal(got, binary) {
		t.Errorf("binary round-trip: got %x want %x", got, binary)
	}
}

// ============================================================================
// extractAll + tarbomb defence
// ============================================================================

func TestTarExtractAll_Happy(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "src.tar")
	destDir := filepath.Join(dir, "out")

	writeTarFixture(t, tarPath, []tarFixtureEntry{
		{name: "README.md", body: []byte("# Paper")},
		{name: "src/", isDir: true},
		{name: "src/main.tex", body: []byte("\\begin{document}hello\\end{document}")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarExtractAllImpl(ctx, []eval.Value{
		&eval.StringValue{Value: tarPath},
		&eval.StringValue{Value: destDir},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv := res.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got Err(%v)", tv.Fields[0])
	}
	writtenList := tv.Fields[0].(*eval.ListValue)
	if len(writtenList.Elements) != 2 { // README.md + src/main.tex (dirs don't count)
		t.Errorf("expected 2 written files, got %d", len(writtenList.Elements))
	}

	// Verify files actually exist on disk
	mainTexPath := filepath.Join(destDir, "src", "main.tex")
	data, err := os.ReadFile(mainTexPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "\\begin{document}hello\\end{document}" {
		t.Errorf("wrong content: %q", data)
	}
}

func TestTarExtractAll_TarbombRejected(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "evil.tar")
	destDir := filepath.Join(dir, "sandbox")

	writeTarFixture(t, tarPath, []tarFixtureEntry{
		{name: "../../etc/passwd", body: []byte("pwned")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarExtractAllImpl(ctx, []eval.Value{
		&eval.StringValue{Value: tarPath},
		&eval.StringValue{Value: destDir},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "path traversal")
	// And crucially — verify the tarbomb's target does not exist on disk
	bomb := "/etc/passwd-tarbomb-marker"
	if _, err := os.Stat(bomb); err == nil {
		t.Errorf("SECURITY: tarbomb marker was written")
	}
}

func TestTarExtractAll_SymlinkRejected(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "symlink.tar")
	destDir := filepath.Join(dir, "sandbox")

	writeTarFixture(t, tarPath, []tarFixtureEntry{
		{
			name:     "escape",
			typeflag: tar.TypeSymlink,
			linkname: "/etc/passwd",
		},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarExtractAllImpl(ctx, []eval.Value{
		&eval.StringValue{Value: tarPath},
		&eval.StringValue{Value: destDir},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "symlink rejected")
}

func TestTarExtractAll_AbsolutePathRejected(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "abs.tar")
	destDir := filepath.Join(dir, "sandbox")

	writeTarFixture(t, tarPath, []tarFixtureEntry{
		{name: "/tmp/absolute.txt", body: []byte("bad")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarExtractAllImpl(ctx, []eval.Value{
		&eval.StringValue{Value: tarPath},
		&eval.StringValue{Value: destDir},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "path traversal")
}

// ============================================================================
// readFromGzip
// ============================================================================

func TestTarReadFromGzip_Happy(t *testing.T) {
	// Simulate an arxiv source bundle: paper.tar.gz containing main.tex + refs.bib.
	dir := t.TempDir()
	path := filepath.Join(dir, "paper.tar.gz")
	mainTex := "\\documentclass{article}\n\\begin{document}\nHello, arxiv.\n\\end{document}\n"
	writeTarGzFixture(t, path, []tarFixtureEntry{
		{name: "main.tex", body: []byte(mainTex)},
		{name: "refs.bib", body: []byte("@article{foo, title={Foo}}")},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadFromGzipImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "main.tex"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	got := unwrapOkString(t, res)
	if got != mainTex {
		t.Errorf("got %q", got)
	}
}

func TestTarReadFromGzip_NotGzip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.tar.gz")
	// This is an uncompressed tar file being read as if it were .tar.gz.
	writeTarFixture(t, path, []tarFixtureEntry{{name: "a.txt", body: []byte("a")}})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadFromGzipImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "a.txt"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "invalid gzip")
}

func TestTarReadFromGzipBytes_Binary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.tar.gz")
	binary := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0xff}
	writeTarGzFixture(t, path, []tarFixtureEntry{
		{name: "image.png", body: binary},
	})

	ctx := effects.NewEffContext([]string{"FS"})
	res, err := tarReadFromGzipBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.StringValue{Value: "image.png"},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	got := mustB64Decode(t, unwrapOkString(t, res))
	if !bytes.Equal(got, binary) {
		t.Errorf("binary roundtrip: got %x want %x", got, binary)
	}
}

// ============================================================================
// Sandbox check
// ============================================================================

func TestTarListEntries_SandboxRespected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.tar")
	writeTarFixture(t, path, []tarFixtureEntry{{name: "a.txt", body: []byte("a")}})

	// Caller passes only "s.tar" but sandbox prepends dir.
	ctx := effects.NewEffContext([]string{"FS"})
	ctx.Env.Sandbox = dir
	res, err := tarListEntriesImpl(ctx, []eval.Value{&eval.StringValue{Value: "s.tar"}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv := res.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got Err(%v)", tv.Fields[0])
	}
}

// ============================================================================
// Silence unused-import lints if any helper becomes unused
// ============================================================================

var _ = base64.StdEncoding
var _ = io.EOF
