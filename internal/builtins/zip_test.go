package builtins

import (
	"archive/zip"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Test helpers
// ============================================================================

func createTestZip(t *testing.T, dir string, entries map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	w := zip.NewWriter(f)
	for name, data := range entries {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
	return path
}

func makeTestCtx(t *testing.T) *effects.EffContext {
	t.Helper()
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("FS"))
	return ctx
}

func assertOk(t *testing.T, result eval.Value) eval.Value {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Ok" {
		errMsg := ""
		if len(tv.Fields) > 0 {
			if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
				errMsg = sv.Value
			}
		}
		t.Fatalf("expected Ok, got Err(%q)", errMsg)
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field in Ok, got %d", len(tv.Fields))
	}
	return tv.Fields[0]
}

func assertErr(t *testing.T, result eval.Value, contains string) {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err, got Ok")
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field in Err, got %d", len(tv.Fields))
	}
	sv, ok := tv.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue in Err, got %T", tv.Fields[0])
	}
	if !strings.Contains(sv.Value, contains) {
		t.Fatalf("error message %q does not contain %q", sv.Value, contains)
	}
}

// ============================================================================
// _zip_listEntries tests
// ============================================================================

func TestZipListEntries_HappyPath(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"hello.txt":       []byte("Hello, World!"),
		"data/config.xml": []byte("<config/>"),
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: zipPath}}
	result, err := zipListEntriesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	list, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list.Elements))
	}

	names := make([]string, len(list.Elements))
	for i, e := range list.Elements {
		sv, ok := e.(*eval.StringValue)
		if !ok {
			t.Fatalf("expected StringValue at index %d, got %T", i, e)
		}
		names[i] = sv.Value
	}

	// Check both entries present (order may vary due to map iteration in createTestZip)
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["hello.txt"] || !found["data/config.xml"] {
		t.Fatalf("expected hello.txt and data/config.xml, got %v", names)
	}
}

func TestZipListEntries_EmptyZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{})

	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: zipPath}}
	result, err := zipListEntriesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	list, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	if len(list.Elements) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(list.Elements))
	}
}

func TestZipListEntries_MissingFile(t *testing.T) {
	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/file.zip"}}
	result, err := zipListEntriesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "cannot open ZIP")
}

// ============================================================================
// _zip_readEntry tests
// ============================================================================

func TestZipReadEntry_HappyPath(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"hello.txt":       []byte("Hello, World!"),
		"data/config.xml": []byte("<config><name>test</name></config>"),
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "hello.txt"},
	}
	result, err := zipReadEntryImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}
	if sv.Value != "Hello, World!" {
		t.Fatalf("expected 'Hello, World!', got %q", sv.Value)
	}
}

func TestZipReadEntry_XMLContent(t *testing.T) {
	dir := t.TempDir()
	xmlContent := `<?xml version="1.0"?><w:document><w:body><w:p>Hello</w:p></w:body></w:document>`
	zipPath := createTestZip(t, dir, map[string][]byte{
		"word/document.xml": []byte(xmlContent),
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "word/document.xml"},
	}
	result, err := zipReadEntryImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}
	if sv.Value != xmlContent {
		t.Fatalf("expected XML content, got %q", sv.Value)
	}
}

func TestZipReadEntry_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"hello.txt": []byte("Hello"),
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "nonexistent.txt"},
	}
	result, err := zipReadEntryImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "entry not found")
}

func TestZipReadEntry_MissingFile(t *testing.T) {
	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: "/nonexistent/file.zip"},
		&eval.StringValue{Value: "hello.txt"},
	}
	result, err := zipReadEntryImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "cannot open ZIP")
}

func TestZipReadEntry_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"safe.txt": []byte("safe"),
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "../../etc/passwd"},
	}
	result, err := zipReadEntryImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "path traversal rejected")
}

// ============================================================================
// _zip_readEntryBytes tests
// ============================================================================

func TestZipReadEntryBytes_HappyPath(t *testing.T) {
	dir := t.TempDir()
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	zipPath := createTestZip(t, dir, map[string][]byte{
		"image.png": binaryData,
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "image.png"},
	}
	result, err := zipReadEntryBytesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}

	// Verify base64 encoding
	expected := base64.StdEncoding.EncodeToString(binaryData)
	if sv.Value != expected {
		t.Fatalf("expected base64 %q, got %q", expected, sv.Value)
	}

	// Verify roundtrip
	decoded, err := base64.StdEncoding.DecodeString(sv.Value)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if string(decoded) != string(binaryData) {
		t.Fatalf("roundtrip failed: got %v", decoded)
	}
}

func TestZipReadEntryBytes_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"safe.bin": []byte{0x00},
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "../secret.bin"},
	}
	result, err := zipReadEntryBytesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "path traversal rejected")
}

func TestZipReadEntryBytes_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"data.bin": []byte{0x01, 0x02},
	})

	ctx := makeTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "missing.bin"},
	}
	result, err := zipReadEntryBytesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "entry not found")
}

// ============================================================================
// Sandbox tests
// ============================================================================

func TestZipListEntries_Sandbox(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"file.txt": []byte("content"),
	})

	ctx := makeTestCtx(t)
	ctx.Env.Sandbox = dir

	// Use relative path — sandbox should resolve it
	args := []eval.Value{&eval.StringValue{Value: "test.zip"}}
	result, err := zipListEntriesImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should work because sandbox + "test.zip" = the actual path
	_ = zipPath // used to create the file
	inner := assertOk(t, result)
	list, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list.Elements))
	}
}

// ============================================================================
// _zip_createArchiveWithBytes tests
// ============================================================================

func TestZipCreateArchiveWithBytes_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")

	// PNG header bytes
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	b64 := base64.StdEncoding.EncodeToString(binaryData)

	ctx := makeTestCtx(t)
	entries := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "image.png"},
			"data": &eval.StringValue{Value: b64},
		}},
	}}

	// Write
	result, err := zipCreateArchiveWithBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
		entries,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOk(t, result)

	// Read back with readEntryBytes
	readResult, err := zipReadEntryBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
		&eval.StringValue{Value: "image.png"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := assertOk(t, readResult)
	sv := inner.(*eval.StringValue)
	if sv.Value != b64 {
		t.Fatalf("round-trip failed: expected %q, got %q", b64, sv.Value)
	}
}

func TestZipCreateArchiveWithBytes_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")

	ctx := makeTestCtx(t)
	entries := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "../evil.bin"},
			"data": &eval.StringValue{Value: base64.StdEncoding.EncodeToString([]byte("bad"))},
		}},
	}}

	result, err := zipCreateArchiveWithBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
		entries,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "path traversal rejected")
}

func TestZipCreateArchiveWithBytes_InvalidBase64(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")

	ctx := makeTestCtx(t)
	entries := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "file.bin"},
			"data": &eval.StringValue{Value: "not-valid-base64!!!"},
		}},
	}}

	result, err := zipCreateArchiveWithBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
		entries,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertErr(t, result, "invalid base64")
}

func TestZipCreateArchiveWithBytes_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")

	data1 := []byte{0x01, 0x02, 0x03}
	data2 := []byte{0x04, 0x05, 0x06}

	ctx := makeTestCtx(t)
	entries := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "a.bin"},
			"data": &eval.StringValue{Value: base64.StdEncoding.EncodeToString(data1)},
		}},
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "b.bin"},
			"data": &eval.StringValue{Value: base64.StdEncoding.EncodeToString(data2)},
		}},
	}}

	result, err := zipCreateArchiveWithBytesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
		entries,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOk(t, result)

	// Verify both entries exist
	listResult, err := zipListEntriesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: outPath},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := assertOk(t, listResult)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list.Elements))
	}
}
