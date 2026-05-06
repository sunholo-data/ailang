package builtins

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/base64"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Note: unwrapOkString, assertErrContains, b64, mustB64Decode are defined in gzip_test.go
// (same package). We reuse them here.

// ============================================================================
// Round-trip tests — raw deflate
// ============================================================================

func TestDeflateRawRoundTrip(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	cases := []struct {
		name  string
		input []byte
		level int
	}{
		{"empty", []byte{}, -1},
		{"small ascii", []byte("hello, deflate"), 6},
		{"repeated text (good compression)", bytes.Repeat([]byte("ailang "), 200), 9},
		{"level 0 (no compression)", []byte("pass through"), 0},
		{"level 1 (fastest)", []byte("small payload"), 1},
		{"level 9 (best)", []byte("small payload"), 9},
		{"unicode utf-8", []byte("日本語 arxiv τέχνη"), 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := deflateDeflateImpl(ctx, []eval.Value{
				&eval.StringValue{Value: b64(tc.input)},
				&eval.IntValue{Value: tc.level},
			})
			if err != nil {
				t.Fatalf("deflate err: %v", err)
			}
			deflatedB64 := unwrapOkString(t, compressed)

			decompressed, err := deflateInflateImpl(ctx, []eval.Value{
				&eval.StringValue{Value: deflatedB64},
			})
			if err != nil {
				t.Fatalf("inflate err: %v", err)
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
// Round-trip tests — zlib-wrapped (RFC 1950)
// ============================================================================

func TestDeflateZlibRoundTrip(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	cases := []struct {
		name  string
		input []byte
		level int
	}{
		{"empty", []byte{}, -1},
		{"small ascii", []byte("hello, zlib"), 6},
		{"repeated text", bytes.Repeat([]byte("PDF "), 500), 9},
		{"level 0 (no compression)", []byte("pass through"), 0},
		{"level 9 (best)", []byte("small payload"), 9},
		{"unicode utf-8", []byte("日本語 PDF FlateDecode"), 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := deflateDeflateZlibImpl(ctx, []eval.Value{
				&eval.StringValue{Value: b64(tc.input)},
				&eval.IntValue{Value: tc.level},
			})
			if err != nil {
				t.Fatalf("deflateZlib err: %v", err)
			}
			zlibB64 := unwrapOkString(t, compressed)

			decompressed, err := deflateInflateZlibImpl(ctx, []eval.Value{
				&eval.StringValue{Value: zlibB64},
			})
			if err != nil {
				t.Fatalf("inflateZlib err: %v", err)
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
// Cross-format negative tests
// ============================================================================

// Feeding a zlib-wrapped stream to raw inflate should not return the original bytes.
// The raw deflate decoder treats the 2-byte zlib header as part of the deflate body,
// so it either errors out or produces garbage — but it must NOT round-trip cleanly.
func TestDeflate_CrossFormat_ZlibToRawInflate(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	original := []byte("PDF FlateDecode payload")
	zlibCompressed, err := deflateDeflateZlibImpl(ctx, []eval.Value{
		&eval.StringValue{Value: b64(original)},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("deflateZlib err: %v", err)
	}
	zlibB64 := unwrapOkString(t, zlibCompressed)

	// Feed zlib-wrapped data to raw inflate — should either Err, or decode to non-original garbage.
	res, err := deflateInflateImpl(ctx, []eval.Value{&eval.StringValue{Value: zlibB64}})
	if err != nil {
		t.Fatalf("inflate impl err: %v", err)
	}

	tv, ok := res.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", res)
	}
	if tv.CtorName == "Ok" {
		// If it decoded successfully, it must NOT equal the original — that would mean
		// the raw inflater silently swallowed the zlib header, which would be a bug.
		out := mustB64Decode(t, tv.Fields[0].(*eval.StringValue).Value)
		if bytes.Equal(out, original) {
			t.Errorf("raw inflate should not round-trip a zlib-wrapped stream, but got the original back")
		}
	}
	// If Err, that's expected — both outcomes are acceptable.
}

// Feeding raw deflate to inflateZlib MUST error — the zlib reader requires the 2-byte header.
func TestDeflate_CrossFormat_RawToZlibInflate(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	original := []byte("raw deflate payload")
	rawCompressed, err := deflateDeflateImpl(ctx, []eval.Value{
		&eval.StringValue{Value: b64(original)},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("deflate err: %v", err)
	}
	rawB64 := unwrapOkString(t, rawCompressed)

	res, err := deflateInflateZlibImpl(ctx, []eval.Value{&eval.StringValue{Value: rawB64}})
	if err != nil {
		t.Fatalf("inflateZlib impl err: %v", err)
	}
	assertErrContains(t, res, "invalid zlib stream")
}

// ============================================================================
// Malformed input tests
// ============================================================================

func TestDeflateInflate_InvalidBase64(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	res, err := deflateInflateImpl(ctx, []eval.Value{&eval.StringValue{Value: "not@@valid!!base64"}})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "invalid base64")
}

func TestDeflateInflateZlib_InvalidBase64(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	res, err := deflateInflateZlibImpl(ctx, []eval.Value{&eval.StringValue{Value: "not@@valid!!"}})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "invalid base64")
}

func TestDeflateInflateZlib_NotZlib(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	// Valid base64, but the bytes aren't a zlib stream.
	res, err := deflateInflateZlibImpl(ctx, []eval.Value{
		&eval.StringValue{Value: b64([]byte("plain text, no zlib header"))},
	})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "invalid zlib stream")
}

func TestDeflateDeflate_InvalidBase64(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	res, err := deflateDeflateImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "@@invalid@@"},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "invalid base64")
}

func TestDeflateDeflate_BadLevel(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	for _, lvl := range []int{-2, 10, 99} {
		res, err := deflateDeflateImpl(ctx, []eval.Value{
			&eval.StringValue{Value: b64([]byte("hi"))},
			&eval.IntValue{Value: lvl},
		})
		if err != nil {
			t.Fatalf("impl err: %v", err)
		}
		assertErrContains(t, res, "compression level")
	}
}

func TestDeflateDeflateZlib_BadLevel(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	for _, lvl := range []int{-2, 10, 99} {
		res, err := deflateDeflateZlibImpl(ctx, []eval.Value{
			&eval.StringValue{Value: b64([]byte("hi"))},
			&eval.IntValue{Value: lvl},
		})
		if err != nil {
			t.Fatalf("impl err: %v", err)
		}
		assertErrContains(t, res, "compression level")
	}
}

// ============================================================================
// Output-cap tests (100 MB decompressed limit)
// ============================================================================

// Build a tiny raw-deflate stream representing ~150MB of zeros.
// Compressed size is KB; decompressed would blow past the cap.
func TestDeflateInflate_BombRejected(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	payloadSize := deflateMaxDecompressedSize + 50*1024*1024 // 150 MB
	chunk := make([]byte, 1024*1024)
	written := 0
	for written < payloadSize {
		n := payloadSize - written
		if n > len(chunk) {
			n = len(chunk)
		}
		if _, err := fw.Write(chunk[:n]); err != nil {
			t.Fatalf("write: %v", err)
		}
		written += n
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	res, err := deflateInflateImpl(ctx, []eval.Value{&eval.StringValue{Value: b64(buf.Bytes())}})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "exceeds")
}

// Same test for the zlib-wrapped variant.
func TestDeflateInflateZlib_BombRejected(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	payloadSize := deflateMaxDecompressedSize + 50*1024*1024
	chunk := make([]byte, 1024*1024)
	written := 0
	for written < payloadSize {
		n := payloadSize - written
		if n > len(chunk) {
			n = len(chunk)
		}
		if _, err := zw.Write(chunk[:n]); err != nil {
			t.Fatalf("write: %v", err)
		}
		written += n
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	res, err := deflateInflateZlibImpl(ctx, []eval.Value{&eval.StringValue{Value: b64(buf.Bytes())}})
	if err != nil {
		t.Fatalf("impl err: %v", err)
	}
	assertErrContains(t, res, "exceeds")
}

// ============================================================================
// Level monotonicity (sanity check on compression behavior)
// ============================================================================

// For a non-trivial repeated payload, level 9 must produce output no larger than level 1.
// This catches a level-validation bug where the level argument is silently ignored.
func TestDeflate_LevelMonotonicity(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	// A repeated payload compresses well at any non-zero level.
	input := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog "), 100)

	compressAt := func(level int) []byte {
		res, err := deflateDeflateImpl(ctx, []eval.Value{
			&eval.StringValue{Value: b64(input)},
			&eval.IntValue{Value: level},
		})
		if err != nil {
			t.Fatalf("deflate level=%d err: %v", level, err)
		}
		return mustB64Decode(t, unwrapOkString(t, res))
	}

	level1 := compressAt(1)
	level9 := compressAt(9)

	if len(level9) > len(level1) {
		t.Errorf("level 9 (%d bytes) larger than level 1 (%d bytes); compression level appears not to be honored",
			len(level9), len(level1))
	}
	// Sanity: both should be smaller than the original.
	if len(level1) >= len(input) || len(level9) >= len(input) {
		t.Errorf("compressed sizes (level1=%d, level9=%d) should be smaller than input (%d) for repeated payload",
			len(level1), len(level9), len(input))
	}
}

// ============================================================================
// Determinism (pure builtins must be deterministic — Go map iteration sanity)
// ============================================================================

func TestDeflate_Deterministic(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	input := []byte("deterministic payload")
	inputB64 := b64(input)

	// Run each variant 20 times — outputs must be identical.
	variants := []struct {
		name string
		fn   func() string
	}{
		{"deflate", func() string {
			res, _ := deflateDeflateImpl(ctx, []eval.Value{
				&eval.StringValue{Value: inputB64},
				&eval.IntValue{Value: 6},
			})
			return unwrapOkString(t, res)
		}},
		{"deflateZlib", func() string {
			res, _ := deflateDeflateZlibImpl(ctx, []eval.Value{
				&eval.StringValue{Value: inputB64},
				&eval.IntValue{Value: 6},
			})
			return unwrapOkString(t, res)
		}},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			first := v.fn()
			for i := 1; i < 20; i++ {
				if got := v.fn(); got != first {
					t.Fatalf("non-deterministic output on iteration %d: first=%q got=%q", i, first, got)
				}
			}
		})
	}
}

// ============================================================================
// Cross-verification: gzip body (header/trailer stripped) must inflate via raw inflate
// ============================================================================

// A gzip stream is: 10-byte header + raw deflate body + 8-byte trailer (CRC32 + ISIZE).
// Stripping the wrapper should yield a raw deflate stream that _deflate_inflate can decode.
// This proves _deflate_inflate is genuinely RFC 1951 compatible, not just self-consistent.
func TestDeflate_GzipBodyCrossDecode(t *testing.T) {
	ctx := effects.NewEffContext(nil)

	original := []byte("the deflate body inside gzip is plain RFC 1951")

	// Build a gzip stream with the existing gzip impl.
	gzipped, err := gzipCompressImpl(ctx, []eval.Value{
		&eval.StringValue{Value: b64(original)},
		&eval.IntValue{Value: 6},
	})
	if err != nil {
		t.Fatalf("gzipCompress err: %v", err)
	}
	gzipB64 := unwrapOkString(t, gzipped)
	gzipBytes := mustB64Decode(t, gzipB64)

	// Strip the 10-byte gzip header and 8-byte trailer (CRC32 + ISIZE).
	// gzip.compress with no extras emits exactly this layout.
	if len(gzipBytes) < 18 {
		t.Fatalf("gzip output too short: %d bytes", len(gzipBytes))
	}
	deflateBody := gzipBytes[10 : len(gzipBytes)-8]

	// Feed the raw deflate body to _deflate_inflate.
	res, err := deflateInflateImpl(ctx, []eval.Value{
		&eval.StringValue{Value: base64.StdEncoding.EncodeToString(deflateBody)},
	})
	if err != nil {
		t.Fatalf("inflate err: %v", err)
	}
	got := mustB64Decode(t, unwrapOkString(t, res))
	if !bytes.Equal(got, original) {
		t.Errorf("cross-decode mismatch: got %q, want %q", got, original)
	}
}
