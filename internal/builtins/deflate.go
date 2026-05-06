package builtins

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Deflate builtins for AILANG
// These provide raw deflate (RFC 1951) and zlib-wrapped (RFC 1950) compression
// primitives, complementing std/gzip (RFC 1952) and std/zip (archive container).
// Part of M-STD-DEFLATE-ZLIB (v0.16.0)
//
// Naming: inflate/deflate operate on raw RFC 1951 streams (no header, no trailer).
// inflateZlib/deflateZlib operate on RFC 1950 streams (2-byte zlib header + adler32).
// Use inflateZlib for PDF FlateDecode, HTTP Content-Encoding: deflate, PNG IDAT,
// and WebSocket permessage-deflate. Use inflate for raw deflate streams (e.g., the
// body of a gzip or zip entry once the wrapper has been stripped).
//
// All binary data crosses the AILANG boundary as base64-encoded strings, mirroring
// std/gzip and std/zip.readEntryBytes.

const (
	deflateMaxDecompressedSize = 100 * 1024 * 1024 // 100 MB — matches std/gzip
)

func init() {
	registerDeflateInflate()
	registerDeflateInflateZlib()
	registerDeflateDeflate()
	registerDeflateDeflateZlib()
}

// Result helpers — distinct from gzip's to avoid coupling, but semantically identical.

func deflateMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func deflateMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// readBounded copies from r into memory, enforcing the 100MB cap via io.LimitReader.
// Wraps any read error with errPrefix for friendlier messages.
func readBounded(r io.Reader, errPrefix string) ([]byte, error) {
	// +1 so ReadAll detects a cap breach even when the stream is exactly the cap.
	limited := io.LimitReader(r, int64(deflateMaxDecompressedSize)+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", errPrefix, err)
	}
	if len(out) > deflateMaxDecompressedSize {
		return nil, fmt.Errorf("decompressed size exceeds %d bytes", deflateMaxDecompressedSize)
	}
	return out, nil
}

// validateDeflateLevel rejects compression levels outside [-1, 9].
// -1 is flate.DefaultCompression; 0-9 are the standard deflate levels.
func validateDeflateLevel(level int) error {
	if level < -1 || level > 9 {
		return fmt.Errorf("compression level out of range [-1..9]: %d", level)
	}
	return nil
}

// ============================================================================
// _deflate_inflate: string -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded raw deflate (RFC 1951) bytes — no header, no trailer.
// Output: base64-encoded decompressed bytes.
// Use for raw deflate streams. For PDF FlateDecode / HTTP / PNG, use _deflate_inflateZlib.

func registerDeflateInflate() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/deflate",
		Name:    "_deflate_inflate",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDeflateInflateType,
		Impl:    deflateInflateImpl,
		Metadata: &BuiltinMetadata{
			Description: "Inflate raw deflate stream (base64 in, base64 out)",
			LongDesc:    "Decompresses a base64-encoded raw deflate (RFC 1951) stream — no zlib header, no adler32 trailer. For PDF FlateDecode, HTTP Content-Encoding: deflate, PNG IDAT, or WebSocket permessage-deflate, use _deflate_inflateZlib instead. Rejects streams whose decompressed size exceeds 100MB.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded raw deflate bytes"},
			},
			Returns: "Result[string, string] — Ok(base64 of decompressed bytes) or Err(error message)",
			Examples: []Example{
				{Code: `_deflate_inflate(b64_of_raw_deflate)`, Description: "Returns Ok(base64-encoded decompressed bytes)"},
			},
			SeeAlso:   []string{"_deflate_deflate", "_deflate_inflateZlib", "_gzip_decompress"},
			Since:     "v0.16.0",
			Stability: StabilityStable,
			Tags:      []string{"deflate", "decompress", "pure"},
			Category:  "deflate",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _deflate_inflate: %v", err))
	}
}

func makeDeflateInflateType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func deflateInflateImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_inflate: expected String, got %T", args[0])
	}

	compressed, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	fr := flate.NewReader(bytes.NewReader(compressed))
	defer fr.Close()

	out, err := readBounded(fr, "deflate read error")
	if err != nil {
		return deflateMakeErr(err.Error()), nil
	}

	return deflateMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(out)}), nil
}

// ============================================================================
// _deflate_inflateZlib: string -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded zlib-wrapped (RFC 1950) bytes — 2-byte header + deflate body + adler32.
// Output: base64-encoded decompressed bytes.
// Use this for PDF FlateDecode, HTTP Content-Encoding: deflate, PNG IDAT, WebSocket permessage-deflate.

func registerDeflateInflateZlib() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/deflate",
		Name:    "_deflate_inflateZlib",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDeflateInflateZlibType,
		Impl:    deflateInflateZlibImpl,
		Metadata: &BuiltinMetadata{
			Description: "Inflate zlib-wrapped (RFC 1950) stream (base64 in, base64 out)",
			LongDesc:    "Decompresses a base64-encoded zlib (RFC 1950) stream — 2-byte header + deflate body + adler32 trailer. Use for PDF FlateDecode, HTTP Content-Encoding: deflate, PNG IDAT chunks, and WebSocket permessage-deflate. For raw deflate (no header/trailer), use _deflate_inflate. Rejects streams whose decompressed size exceeds 100MB.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded zlib-wrapped bytes"},
			},
			Returns: "Result[string, string] — Ok(base64 of decompressed bytes) or Err(error message)",
			Examples: []Example{
				{Code: `_deflate_inflateZlib(b64_of_pdf_objstm)`, Description: "Decompress a PDF /ObjStm FlateDecode payload"},
			},
			SeeAlso:   []string{"_deflate_deflateZlib", "_deflate_inflate", "_gzip_decompress"},
			Since:     "v0.16.0",
			Stability: StabilityStable,
			Tags:      []string{"deflate", "zlib", "decompress", "pure"},
			Category:  "deflate",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _deflate_inflateZlib: %v", err))
	}
}

func makeDeflateInflateZlibType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func deflateInflateZlibImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_inflateZlib: expected String, got %T", args[0])
	}

	compressed, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		// zlib.NewReader fails if the header is malformed (e.g., raw deflate input).
		return deflateMakeErr(fmt.Sprintf("invalid zlib stream: %v (input may be raw deflate — try _deflate_inflate)", err)), nil
	}
	defer zr.Close()

	out, err := readBounded(zr, "zlib read error")
	if err != nil {
		return deflateMakeErr(err.Error()), nil
	}

	return deflateMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(out)}), nil
}

// ============================================================================
// _deflate_deflate: string -> int -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded raw bytes + compression level (0..9, -1 for default).
// Output: base64-encoded raw deflate (RFC 1951) bytes — no header, no trailer.

func registerDeflateDeflate() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/deflate",
		Name:    "_deflate_deflate",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeDeflateDeflateType,
		Impl:    deflateDeflateImpl,
		Metadata: &BuiltinMetadata{
			Description: "Compress to raw deflate stream (base64 in, base64 out)",
			LongDesc:    "Compresses base64-encoded bytes with raw deflate (RFC 1951) at the given level (0-9, or -1 for default). No zlib header, no adler32 trailer. For zlib-wrapped output (used by PDF/HTTP/PNG), use _deflate_deflateZlib.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded raw bytes to compress"},
				{Name: "level", Description: "Compression level: 0 (none) to 9 (best). Use -1 for default (6)."},
			},
			Returns: "Result[string, string] — Ok(base64 of raw deflate stream) or Err(error message)",
			Examples: []Example{
				{Code: `_deflate_deflate(b64_of_text, 6)`, Description: "Returns Ok(base64-encoded raw deflate stream)"},
			},
			SeeAlso:   []string{"_deflate_inflate", "_deflate_deflateZlib", "_gzip_compress"},
			Since:     "v0.16.0",
			Stability: StabilityStable,
			Tags:      []string{"deflate", "compress", "pure"},
			Category:  "deflate",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _deflate_deflate: %v", err))
	}
}

func makeDeflateDeflateType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func deflateDeflateImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_deflate: expected String for input, got %T", args[0])
	}
	levelVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_deflate: expected Int for level, got %T", args[1])
	}

	raw, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	if err := validateDeflateLevel(levelVal.Value); err != nil {
		return deflateMakeErr(err.Error()), nil
	}

	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, levelVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("deflate writer error: %v", err)), nil
	}
	if _, err := fw.Write(raw); err != nil {
		_ = fw.Close()
		return deflateMakeErr(fmt.Sprintf("deflate write error: %v", err)), nil
	}
	if err := fw.Close(); err != nil {
		return deflateMakeErr(fmt.Sprintf("deflate close error: %v", err)), nil
	}

	return deflateMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}), nil
}

// ============================================================================
// _deflate_deflateZlib: string -> int -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded raw bytes + compression level (0..9, -1 for default).
// Output: base64-encoded zlib-wrapped (RFC 1950) bytes.

func registerDeflateDeflateZlib() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/deflate",
		Name:    "_deflate_deflateZlib",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeDeflateDeflateZlibType,
		Impl:    deflateDeflateZlibImpl,
		Metadata: &BuiltinMetadata{
			Description: "Compress to zlib-wrapped (RFC 1950) stream (base64 in, base64 out)",
			LongDesc:    "Compresses base64-encoded bytes with zlib (RFC 1950) framing at the given level (0-9, or -1 for default). Output includes the 2-byte zlib header and adler32 trailer. Suitable for HTTP Content-Encoding: deflate output, PDF /ObjStm authoring, etc. For raw deflate (no wrapper), use _deflate_deflate.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded raw bytes to compress"},
				{Name: "level", Description: "Compression level: 0 (none) to 9 (best). Use -1 for default (6)."},
			},
			Returns: "Result[string, string] — Ok(base64 of zlib-wrapped stream) or Err(error message)",
			Examples: []Example{
				{Code: `_deflate_deflateZlib(b64_of_text, 6)`, Description: "Returns Ok(base64-encoded zlib-wrapped stream)"},
			},
			SeeAlso:   []string{"_deflate_inflateZlib", "_deflate_deflate", "_gzip_compress"},
			Since:     "v0.16.0",
			Stability: StabilityStable,
			Tags:      []string{"deflate", "zlib", "compress", "pure"},
			Category:  "deflate",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _deflate_deflateZlib: %v", err))
	}
}

func makeDeflateDeflateZlibType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func deflateDeflateZlibImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_deflateZlib: expected String for input, got %T", args[0])
	}
	levelVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_deflate_deflateZlib: expected Int for level, got %T", args[1])
	}

	raw, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	if err := validateDeflateLevel(levelVal.Value); err != nil {
		return deflateMakeErr(err.Error()), nil
	}

	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, levelVal.Value)
	if err != nil {
		return deflateMakeErr(fmt.Sprintf("zlib writer error: %v", err)), nil
	}
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return deflateMakeErr(fmt.Sprintf("zlib write error: %v", err)), nil
	}
	if err := zw.Close(); err != nil {
		return deflateMakeErr(fmt.Sprintf("zlib close error: %v", err)), nil
	}

	return deflateMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}), nil
}
