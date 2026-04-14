package builtins

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// Helper: unwrap an Ok(string) TaggedValue or fail the test.
func unwrapOkString(t *testing.T, v eval.Value) string {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", v)
	}
	if tv.CtorName != "Ok" {
		// Surface the error message if it's an Err
		if tv.CtorName == "Err" && len(tv.Fields) == 1 {
			if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
				t.Fatalf("expected Ok, got Err(%q)", sv.Value)
			}
		}
		t.Fatalf("expected Ok, got %s", tv.CtorName)
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(tv.Fields))
	}
	sv, ok := tv.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue inside Ok, got %T", tv.Fields[0])
	}
	return sv.Value
}

// Helper: assert Err(msg) where msg contains substr.
func assertErrContains(t *testing.T, v eval.Value, substr string) {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", v)
	}
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tv.CtorName)
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("expected 1 field in Err, got %d", len(tv.Fields))
	}
	sv, ok := tv.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue inside Err, got %T", tv.Fields[0])
	}
	if !strings.Contains(sv.Value, substr) {
		t.Errorf("Err(%q) does not contain %q", sv.Value, substr)
	}
}

func b64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func mustB64Decode(t *testing.T, s string) []byte {
	t.Helper()
	out, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	return out
}

// ============================================================================
// Round-trip tests
// ============================================================================

func TestGzipRoundTrip(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	cases := []struct {
		name  string
		input []byte
		level int
	}{
		{"empty", []byte{}, -1},
		{"small ascii", []byte("hello, gzip"), 6},
		{"repeated text (good compression)", bytes.Repeat([]byte("ailang "), 200), 9},
		{"level 0 (no compression)", []byte("pass through"), 0},
		{"level 9 (best)", []byte("small payload"), 9},
		{"unicode utf-8", []byte("日本語 arxiv τέχνη"), 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// compress
			compressArgs := []eval.Value{
				&eval.StringValue{Value: b64(tc.input)},
				&eval.IntValue{Value: tc.level},
			}
			compressed, err := gzipCompressImpl(ctx, compressArgs)
			if err != nil {
				t.Fatalf("compress err: %v", err)
			}
			gzippedB64 := unwrapOkString(t, compressed)

			// decompress
			decompressed, err := gzipDecompressImpl(ctx, []eval.Value{
				&eval.StringValue{Value: gzippedB64},
			})
			if err != nil {
				t.Fatalf("decompress err: %v", err)
			}
			outB64 := unwrapOkString(t, decompressed)

			got := mustB64Decode(t, outB64)
			if !bytes.Equal(got, tc.input) {
				t.Errorf("round-trip mismatch: got %q, want %q", got, tc.input)
			}
		})
	}
}

// ============================================================================
// Decompress error cases
// ============================================================================

func TestGzipDecompress_InvalidBase64(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	res, err := gzipDecompressImpl(ctx, []eval.Value{&eval.StringValue{Value: "not@@valid!!base64"}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "invalid base64")
}

func TestGzipDecompress_NotGzip(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	// Valid base64, but the decoded bytes aren't a gzip stream.
	res, err := gzipDecompressImpl(ctx, []eval.Value{&eval.StringValue{Value: b64([]byte("plain text, no magic"))}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "invalid gzip")
}

// TestGzipDecompress_BombRejected verifies the 100MB decompressed cap.
// We build a tiny gzip stream of ~150MB of zeros — it compresses to KBs but
// would blow past the cap on decompression.
func TestGzipDecompress_BombRejected(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	// 150 MB of zeros → highly compressible
	payloadSize := gzipMaxDecompressedSize + 50*1024*1024
	// Write in 1 MB chunks to avoid a huge allocation.
	chunk := make([]byte, 1024*1024)
	written := 0
	for written < payloadSize {
		n := payloadSize - written
		if n > len(chunk) {
			n = len(chunk)
		}
		if _, err := gw.Write(chunk[:n]); err != nil {
			t.Fatalf("write: %v", err)
		}
		written += n
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	res, err := gzipDecompressImpl(ctx, []eval.Value{&eval.StringValue{Value: b64(buf.Bytes())}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "exceeds")
}

// ============================================================================
// Compress error cases
// ============================================================================

func TestGzipCompress_InvalidBase64(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	res, err := gzipCompressImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "@@invalid@@"},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "invalid base64")
}

func TestGzipCompress_BadLevel(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	for _, lvl := range []int{-2, 10, 99} {
		res, err := gzipCompressImpl(ctx, []eval.Value{
			&eval.StringValue{Value: b64([]byte("hi"))},
			&eval.IntValue{Value: lvl},
		})
		if err != nil {
			t.Fatalf("impl error: %v", err)
		}
		assertErrContains(t, res, "compression level")
	}
}

// ============================================================================
// decompressFile
// ============================================================================

func TestGzipDecompressFile_HappyPath(t *testing.T) {
	ctx := effects.NewEffContext([]string{"FS"})

	dir := t.TempDir()
	path := filepath.Join(dir, "greeting.txt.gz")

	original := []byte("hello from a real gzip file\n")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write(original); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("f close: %v", err)
	}

	res, err := gzipDecompressFileImpl(ctx, []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	got := mustB64Decode(t, unwrapOkString(t, res))
	if !bytes.Equal(got, original) {
		t.Errorf("got %q, want %q", got, original)
	}
}

func TestGzipDecompressFile_MissingFile(t *testing.T) {
	ctx := effects.NewEffContext([]string{"FS"})
	res, err := gzipDecompressFileImpl(ctx, []eval.Value{&eval.StringValue{Value: "/nonexistent/does-not-exist.gz"}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "cannot open")
}

func TestGzipDecompressFile_NotGzip(t *testing.T) {
	ctx := effects.NewEffContext([]string{"FS"})
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.gz")
	if err := os.WriteFile(path, []byte("this is not gzip"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	res, err := gzipDecompressFileImpl(ctx, []eval.Value{&eval.StringValue{Value: path}})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	assertErrContains(t, res, "invalid gzip")
}
