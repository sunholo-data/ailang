package builtins

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Gzip builtins for AILANG
// These provide gzip compression/decompression with the same 100MB safety cap as std/zip.
// Part of M-STDLIB-TAR-GZIP (v0.12.0)
//
// All binary data crosses the AILANG boundary as base64-encoded strings, mirroring
// std/zip.readEntryBytes. This keeps the Go <-> AILANG representation consistent
// and avoids assuming compressed/decompressed bytes are valid UTF-8.

const (
	gzipMaxDecompressedSize = 100 * 1024 * 1024 // 100 MB — matches zipMaxDecompressedSize
)

func init() {
	registerGzipDecompress()
	registerGzipCompress()
	registerGzipDecompressFile()
}

// Result helpers — distinct from zip's to avoid coupling, but semantically identical.

func gzipMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func gzipMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// gunzipBounded decompresses gzip data with the 100MB cap enforced via io.LimitReader.
// Returns the raw decompressed bytes or an error if the cap is exceeded.
func gunzipBounded(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip: %v", err)
	}
	defer gr.Close()

	// +1 so ReadAll detects a cap breach even when the stream is exactly the cap.
	limited := io.LimitReader(gr, int64(gzipMaxDecompressedSize)+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("gzip read error: %v", err)
	}
	if len(out) > gzipMaxDecompressedSize {
		return nil, fmt.Errorf("decompressed size exceeds %d bytes", gzipMaxDecompressedSize)
	}
	return out, nil
}

// ============================================================================
// _gzip_decompress: string -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded gzipped bytes.
// Output: base64-encoded raw (decompressed) bytes.

func registerGzipDecompress() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/gzip",
		Name:    "_gzip_decompress",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeGzipDecompressType,
		Impl:    gzipDecompressImpl,
		Metadata: &BuiltinMetadata{
			Description: "Decompress gzip data (base64 in, base64 out)",
			LongDesc:    "Decompresses a base64-encoded gzip stream and returns the raw decompressed bytes as a base64-encoded string. Use _bytes_from_base64 to decode and _bytes_to_string to convert to UTF-8. Rejects streams whose decompressed size exceeds 100MB.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded gzip data"},
			},
			Returns: "Result[string, string] — Ok(base64 of decompressed bytes) or Err(error message)",
			Examples: []Example{
				{Code: `_gzip_decompress(b64_of_gzipped_text)`, Description: "Returns Ok(base64-encoded decompressed bytes)"},
			},
			SeeAlso:   []string{"_gzip_compress", "_gzip_decompressFile", "_bytes_from_base64"},
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"gzip", "decompress", "pure"},
			Category:  "gzip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _gzip_decompress: %v", err))
	}
}

func makeGzipDecompressType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func gzipDecompressImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_gzip_decompress: expected String, got %T", args[0])
	}

	compressed, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	out, err := gunzipBounded(compressed)
	if err != nil {
		return gzipMakeErr(err.Error()), nil
	}

	return gzipMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(out)}), nil
}

// ============================================================================
// _gzip_compress: string -> int -> Result[string, string]   (pure)
// ============================================================================
// Input: base64-encoded raw bytes + compression level (0..9, -1 for default).
// Output: base64-encoded gzipped bytes.

func registerGzipCompress() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/gzip",
		Name:    "_gzip_compress",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeGzipCompressType,
		Impl:    gzipCompressImpl,
		Metadata: &BuiltinMetadata{
			Description: "Gzip-compress bytes (base64 in, base64 out)",
			LongDesc:    "Compresses base64-encoded bytes with gzip at the given level (0-9, or -1 for default level 6). Returns the gzipped bytes as a base64-encoded string.",
			Params: []ParamDoc{
				{Name: "input", Description: "Base64-encoded raw bytes to compress"},
				{Name: "level", Description: "Compression level: 0 (none) to 9 (best). Use -1 for default (6)."},
			},
			Returns: "Result[string, string] — Ok(base64 of gzipped bytes) or Err(error message)",
			Examples: []Example{
				{Code: `_gzip_compress(b64_of_text, 6)`, Description: "Returns Ok(base64-encoded gzip stream)"},
			},
			SeeAlso:   []string{"_gzip_decompress", "_bytes_to_base64"},
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"gzip", "compress", "pure"},
			Category:  "gzip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _gzip_compress: %v", err))
	}
}

func makeGzipCompressType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int()).Returns(
		T.App("Result", T.String(), T.String()),
	).Build()
}

func gzipCompressImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	inVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_gzip_compress: expected String for input, got %T", args[0])
	}
	levelVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_gzip_compress: expected Int for level, got %T", args[1])
	}

	raw, err := base64.StdEncoding.DecodeString(inVal.Value)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("invalid base64: %v", err)), nil
	}

	level := levelVal.Value
	// Accept -1 (DefaultCompression) through 9 (BestCompression). Any other value is an error.
	if level < -1 || level > 9 {
		return gzipMakeErr(fmt.Sprintf("compression level out of range [-1..9]: %d", level)), nil
	}

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("gzip writer error: %v", err)), nil
	}
	if _, err := gw.Write(raw); err != nil {
		_ = gw.Close()
		return gzipMakeErr(fmt.Sprintf("gzip write error: %v", err)), nil
	}
	if err := gw.Close(); err != nil {
		return gzipMakeErr(fmt.Sprintf("gzip close error: %v", err)), nil
	}

	return gzipMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}), nil
}

// ============================================================================
// _gzip_decompressFile: string -> Result[string, string] ! {FS}
// ============================================================================
// Reads a .gz file from disk and returns the decompressed bytes as base64.

func registerGzipDecompressFile() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/gzip",
		Name:    "_gzip_decompressFile",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeGzipDecompressFileType,
		Impl:    gzipDecompressFileImpl,
		Metadata: &BuiltinMetadata{
			Description: "Decompress a .gz file and return its contents as base64",
			LongDesc:    "Opens a gzip file on disk and streams its decompressed contents, returning the raw bytes as a base64-encoded string. Respects AILANG_FS_SANDBOX. Rejects decompressed payloads exceeding 100MB.",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the .gz file"},
			},
			Returns: "Result[string, string] — Ok(base64 of decompressed bytes) or Err(error message)",
			Examples: []Example{
				{Code: `_gzip_decompressFile("data.txt.gz")`, Description: "Returns Ok(base64 of original data.txt contents)"},
			},
			SeeAlso:   []string{"_gzip_decompress", "_tar_readFromGzip"},
			Since:     "v0.12.0",
			Stability: StabilityStable,
			Tags:      []string{"gzip", "decompress", "file", "fs"},
			Category:  "gzip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _gzip_decompressFile: %v", err))
	}
}

func makeGzipDecompressFileType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", T.String(), T.String()),
	).Effects("FS")
}

func gzipDecompressFileImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_gzip_decompressFile: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
	}

	f, err := os.Open(path)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("cannot open file: %v", err)), nil
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("invalid gzip: %v", err)), nil
	}
	defer gr.Close()

	limited := io.LimitReader(gr, int64(gzipMaxDecompressedSize)+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return gzipMakeErr(fmt.Sprintf("gzip read error: %v", err)), nil
	}
	if len(out) > gzipMaxDecompressedSize {
		return gzipMakeErr(fmt.Sprintf("decompressed size exceeds %d bytes", gzipMaxDecompressedSize)), nil
	}

	return gzipMakeOk(&eval.StringValue{Value: base64.StdEncoding.EncodeToString(out)}), nil
}
