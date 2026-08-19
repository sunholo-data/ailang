package builtins

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Tests for the ailang#644 in-memory ZIP builders.
//
// The property under test throughout is that these produce a REAL archive with
// no filesystem anywhere near them — so every assertion here reads the bytes
// the builtin returned rather than re-deriving what it should have returned.

// ============================================================================
// helpers
// ============================================================================

func textEntries(pairs ...[2]string) []eval.Value {
	out := make([]eval.Value, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, &eval.RecordValue{Fields: map[string]eval.Value{
			"name":    &eval.StringValue{Value: p[0]},
			"content": &eval.StringValue{Value: p[1]},
		}})
	}
	return out
}

func byteEntries(pairs ...[2]string) []eval.Value {
	out := make([]eval.Value, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, &eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: p[0]},
			"data": &eval.StringValue{Value: base64.StdEncoding.EncodeToString([]byte(p[1]))},
		}})
	}
	return out
}

// buildArchiveBytes runs the builtin and decodes the base64 archive it returned.
// It deliberately goes through the Ok payload rather than calling the writer
// directly: the base64 boundary is part of the contract callers depend on.
func buildArchiveBytes(t *testing.T, impl func(entries []eval.Value) eval.Value, entries []eval.Value) []byte {
	t.Helper()
	inner := assertOk(t, impl(entries))
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue in Ok, got %T", inner)
	}
	raw, err := base64.StdEncoding.DecodeString(sv.Value)
	if err != nil {
		t.Fatalf("builtin returned a value that is not valid base64: %v", err)
	}
	return raw
}

func callBuildArchive(entries []eval.Value) eval.Value {
	// nil EffContext on purpose: a PURE builtin must never reach for one.
	v, err := zipBuildArchiveImpl(nil, []eval.Value{&eval.ListValue{Elements: entries}})
	if err != nil {
		panic(err)
	}
	return v
}

func callBuildArchiveWithBytes(entries []eval.Value) eval.Value {
	v, err := zipBuildArchiveWithBytesImpl(nil, []eval.Value{&eval.ListValue{Elements: entries}})
	if err != nil {
		panic(err)
	}
	return v
}

// readArchive parses archive bytes into name -> content.
func readArchive(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("builtin output is not a readable ZIP: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", f.Name, err)
		}
		var b bytes.Buffer
		if _, err := b.ReadFrom(rc); err != nil {
			t.Fatalf("read entry %s: %v", f.Name, err)
		}
		_ = rc.Close()
		out[f.Name] = b.String()
	}
	return out
}

// ============================================================================
// The headline property of ailang#644: these are PURE — no FS effect at all.
// ============================================================================

func TestZipBuildArchive_RegisteredPure(t *testing.T) {
	for _, name := range []string{"_zip_buildArchive", "_zip_buildArchiveWithBytes"} {
		spec, ok := GetSpec(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if !spec.IsPure {
			t.Errorf("%s: IsPure = false, want true — the whole point of #644 is that "+
				"building a ZIP needs no filesystem", name)
		}
		if spec.Effect != "" {
			t.Errorf("%s: Effect = %q, want \"\" — an effect here makes the builder "+
				"unusable in a WASM host, which is the case it exists for", name, spec.Effect)
		}
		if spec.NumArgs != 1 {
			t.Errorf("%s: NumArgs = %d, want 1 (entries only — no path)", name, spec.NumArgs)
		}
	}

	// Control: the FS-writing siblings must still be impure and FS-effecting,
	// so a registry-wide mistake cannot make this test pass vacuously.
	for _, name := range []string{"_zip_createArchive", "_zip_createArchiveWithBytes"} {
		spec, ok := GetSpec(name)
		if !ok {
			t.Fatalf("control %s is not registered", name)
		}
		if spec.IsPure || spec.Effect != "FS" {
			t.Errorf("control %s: IsPure=%v Effect=%q, want false/\"FS\"",
				name, spec.IsPure, spec.Effect)
		}
	}
}

// ============================================================================
// Round-trip: the returned bytes are a real archive with the right contents.
// ============================================================================

func TestZipBuildArchive_TextRoundTrip(t *testing.T) {
	raw := buildArchiveBytes(t, callBuildArchive, textEntries(
		[2]string{"[Content_Types].xml", "<Types/>"},
		[2]string{"word/document.xml", "<w:document><w:body/></w:document>"},
	))

	got := readArchive(t, raw)
	want := map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   "<w:document><w:body/></w:document>",
	}
	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d (%v)", len(got), len(want), got)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("entry %q = %q, want %q", name, got[name], content)
		}
	}
}

func TestZipBuildArchiveWithBytes_Base64RoundTrip(t *testing.T) {
	// A payload that is NOT valid UTF-8, which is the case base64 exists for.
	binary := string([]byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0xFF, 0xFE})
	raw := buildArchiveBytes(t, callBuildArchiveWithBytes, byteEntries(
		[2]string{"word/media/image1.png", binary},
	))

	got := readArchive(t, raw)
	if got["word/media/image1.png"] != binary {
		t.Errorf("binary entry round-tripped as %q, want %q",
			got["word/media/image1.png"], binary)
	}
}

func TestZipBuildArchive_EmptyEntryListIsAValidArchive(t *testing.T) {
	// An empty part list is a legitimate request, not an error — and the result
	// must still be a parseable archive (i.e. the central directory was
	// flushed), not an empty string.
	raw := buildArchiveBytes(t, callBuildArchive, nil)
	if len(raw) == 0 {
		t.Fatal("empty entry list produced zero bytes; expected an empty but valid archive")
	}
	if got := readArchive(t, raw); len(got) != 0 {
		t.Errorf("empty entry list produced %d entries: %v", len(got), got)
	}
}

// ============================================================================
// Purity is a claim about the OUTPUT, not just about the signature.
// ============================================================================

func TestZipBuildArchive_Deterministic(t *testing.T) {
	entries := textEntries([2]string{"a.txt", "alpha"}, [2]string{"b/c.txt", "beta"})

	first := buildArchiveBytes(t, callBuildArchive, entries)
	// MS-DOS timestamps have 2-second resolution, so a wall-clock stamp would
	// have to cross a 2s boundary to differ. Sleep past it: without this the
	// test passes whether or not the builder is deterministic.
	time.Sleep(2100 * time.Millisecond)
	second := buildArchiveBytes(t, callBuildArchive, entries)

	if !bytes.Equal(first, second) {
		t.Errorf("two builds of identical entries differ (%d vs %d bytes) — the builtin "+
			"is registered PURE, so a wall-clock timestamp in the output is a soundness bug",
			len(first), len(second))
	}
}

// TestZipBuildArchive_MatchesFSArchive pins that the in-memory and FS builders
// share one serialisation path. If they ever diverge, a document that opens
// from disk could fail to open from the browser (or vice versa) for reasons no
// caller could see.
func TestZipBuildArchive_MatchesFSArchive(t *testing.T) {
	entries := textEntries(
		[2]string{"[Content_Types].xml", "<Types/>"},
		[2]string{"word/document.xml", "<w:document><w:body/></w:document>"},
	)

	inMemory := buildArchiveBytes(t, callBuildArchive, entries)

	dir := t.TempDir()
	path := filepath.Join(dir, "fs.zip")
	ctx := makeTestCtx(t)
	res, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: path},
		&eval.ListValue{Elements: entries},
	})
	if err != nil {
		t.Fatalf("_zip_createArchive: %v", err)
	}
	assertOk(t, res)

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fs archive: %v", err)
	}

	if !bytes.Equal(inMemory, onDisk) {
		t.Errorf("in-memory archive (%d bytes) differs from the FS archive (%d bytes) "+
			"for identical entries", len(inMemory), len(onDisk))
	}
}

// ============================================================================
// Refusals. One arm per refusal branch (skill rule 3j).
// ============================================================================

func TestZipBuildArchive_RejectsPathTraversal(t *testing.T) {
	assertErr(t, callBuildArchive(textEntries([2]string{"../escape.txt", "x"})),
		"path traversal rejected")
	assertErr(t, callBuildArchiveWithBytes(byteEntries([2]string{"../escape.bin", "x"})),
		"path traversal rejected")
}

func TestZipBuildArchiveWithBytes_RejectsInvalidBase64(t *testing.T) {
	entries := []eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{
		"name": &eval.StringValue{Value: "bad.bin"},
		"data": &eval.StringValue{Value: "!!! not base64 !!!"},
	}}}
	assertErr(t, callBuildArchiveWithBytes(entries), "invalid base64")
}

func TestZipBuildArchive_RejectsMalformedEntryRecord(t *testing.T) {
	// Wrong payload field: a {name, data} record handed to the TEXT builder.
	assertErr(t, callBuildArchive([]eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{
		"name": &eval.StringValue{Value: "a.txt"},
		"data": &eval.StringValue{Value: "x"},
	}}}), "'content' field must be string")

	// And the mirror: a {name, content} record handed to the BYTES builder.
	assertErr(t, callBuildArchiveWithBytes([]eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{
		"name":    &eval.StringValue{Value: "a.bin"},
		"content": &eval.StringValue{Value: "x"},
	}}}), "'data' field must be string")

	// Non-string name.
	assertErr(t, callBuildArchive([]eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{
		"name":    &eval.IntValue{Value: 1},
		"content": &eval.StringValue{Value: "x"},
	}}}), "'name' field must be string")

	// Not a record at all.
	assertErr(t, callBuildArchive([]eval.Value{&eval.StringValue{Value: "nope"}}),
		"expected record")
}

func TestZipBuildArchive_RejectsTooManyEntries(t *testing.T) {
	entries := make([]eval.Value, zipMaxEntries+1)
	for i := range entries {
		entries[i] = &eval.RecordValue{Fields: map[string]eval.Value{
			"name":    &eval.StringValue{Value: fmt.Sprintf("f%d.txt", i)},
			"content": &eval.StringValue{Value: "x"},
		}}
	}
	assertErr(t, callBuildArchive(entries), "too many entries")
}

// TestZipBuildArchive_RejectsOversizeTotal exercises the budget that only the
// in-memory builders carry. It is reachable because the cap is a var: at its
// production value the only way to red this would be to allocate 100MB, i.e.
// the guard would never be exercised at all.
func TestZipBuildArchive_RejectsOversizeTotal(t *testing.T) {
	saved := zipMaxArchiveContentSize
	zipMaxArchiveContentSize = 16
	defer func() { zipMaxArchiveContentSize = saved }()

	// Each entry is under the cap; the TOTAL is not. A per-entry-only check
	// passes this and is exactly the bug the budget exists to prevent.
	entries := textEntries(
		[2]string{"a.txt", "0123456789"},
		[2]string{"b.txt", "0123456789"},
	)
	assertErr(t, callBuildArchive(entries), "archive contents too large")

	// Same for the base64 arm, measured on DECODED length.
	assertErr(t, callBuildArchiveWithBytes(byteEntries(
		[2]string{"a.bin", "0123456789"},
		[2]string{"b.bin", "0123456789"},
	)), "archive contents too large")

	// Control: comfortably under the cap still succeeds, so the arm above is
	// not passing because the builder is broken outright.
	assertOk(t, callBuildArchive(textEntries([2]string{"a.txt", "tiny"})))
}

// TestZipCreateArchive_StaysUnboundedInAggregate is the negative control for
// the budget: the FS builders share the serialiser but pass budget=0, and must
// not inherit a cap they never had.
func TestZipCreateArchive_StaysUnboundedInAggregate(t *testing.T) {
	saved := zipMaxArchiveContentSize
	zipMaxArchiveContentSize = 16
	defer func() { zipMaxArchiveContentSize = saved }()

	dir := t.TempDir()
	ctx := makeTestCtx(t)
	res, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: filepath.Join(dir, "big.zip")},
		&eval.ListValue{Elements: textEntries(
			[2]string{"a.txt", "0123456789"},
			[2]string{"b.txt", "0123456789"},
		)},
	})
	if err != nil {
		t.Fatalf("_zip_createArchive: %v", err)
	}
	assertOk(t, res)
}

// ============================================================================
// Type surface
// ============================================================================

func TestZipBuildArchive_TypeIsPureAndReturnsString(t *testing.T) {
	for name, mk := range map[string]func() interface{ String() string }{
		"_zip_buildArchive":          func() interface{ String() string } { return makeZipBuildArchiveType() },
		"_zip_buildArchiveWithBytes": func() interface{ String() string } { return makeZipBuildArchiveWithBytesType() },
	} {
		got := mk().String()
		if want := "Result[string, string]"; !contains(got, want) {
			t.Errorf("%s type = %s, want it to return %s", name, got, want)
		}
		// The FS siblings carry `! {FS}`; these must not.
		if contains(got, "FS") {
			t.Errorf("%s type = %s, want no FS effect", name, got)
		}
	}
}

func contains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
