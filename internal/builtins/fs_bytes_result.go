package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Raw-bytes, Result-returning FS builtins (std/fs readFileRaw /
// writeFileBytesResult). _fs_readFileBytes returns base64 TEXT, and
// _fs_writeFileBytes panics on failure; these are the bytes-typed,
// Err-reporting forms. Both dispatch to the FS effect ops in
// internal/effects/fs_bytes_result.go, so capability, sandbox, size-cap and
// fs_deny_write checks are the same as for every other FS builtin.

func init() {
	registerFSBytesResult()
}

func registerFSBytesResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_readFileRaw", NumArgs: 1, IsPure: false, Effect: "FS",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).Returns(T.App("Result", T.Bytes(), T.String())).Effects("FS")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "FS", "readFileRaw", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Read a file's raw bytes, returning Result instead of panicking on failure",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the file to read"},
			},
			Returns: "Result[bytes, string] - Ok(file bytes, unchanged) on success, Err(message) on failure",
			Examples: []Example{
				{Code: `_fs_readFileRaw("image.png")`, Description: "Returns Ok(bytes) or Err(\"cannot read file: ...\")"},
			},
			LongDesc:  "Returns the file's bytes as-is: no base64 (unlike _fs_readFileBytes, which returns base64 text) and no UTF-8 decoding (unlike _fs_readFileResult). Respects AILANG_FS_SANDBOX and the FS read size cap.",
			SeeAlso:   []string{"_fs_writeFileBytesResult", "_fs_readFileBytes", "_fs_readFileResult"},
			Since:     "v0.51.1",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "read", "binary", "bytes", "result", "safe"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_readFileRaw: %v", err))
	}

	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_writeFileBytesResult", NumArgs: 2, IsPure: false, Effect: "FS",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String(), T.Bytes()).Returns(T.App("Result", T.Unit(), T.String())).Effects("FS")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "FS", "writeFileBytesResult", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Write raw bytes to a file (truncates if exists), returning Result instead of panicking on failure",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to the file to write"},
				{Name: "data", Description: "Bytes to write"},
			},
			Returns: "Result[(), string] - Ok(()) on success, Err(message) on failure",
			Examples: []Example{
				{Code: `_fs_writeFileBytesResult("out.bin", _bytes_from_string("hi"))`, Description: "Returns Ok(()) or Err(\"cannot write file: ...\")"},
			},
			LongDesc:  "Result-returning variant of _fs_writeFileBytes. Returns Err on missing parent dir, permission denied, sandbox violation or a protected path, where _fs_writeFileBytes panics. File permissions: 0644.",
			SeeAlso:   []string{"_fs_writeFileBytes", "_fs_readFileRaw", "_fs_writeFileResult"},
			Since:     "v0.51.1",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "write", "binary", "bytes", "result", "safe"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_writeFileBytesResult: %v", err))
	}
}
