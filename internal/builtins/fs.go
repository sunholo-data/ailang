package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// FS effect builtins for AILANG
// These provide filesystem operations with capability-based security

func init() {
	registerFS()
}

// ============================================================================
// FS Effect Builtins (_fs_readFile, _fs_readFileBytes, _fs_writeFile, _fs_writeFileBytes, _fs_exists)
// ============================================================================

func registerFS() {
	// _fs_readFile
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFile", args)
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.String()).Effects("FS")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_readFile", NumArgs: 1, IsPure: false, Effect: "FS", Type: type1, Impl: impl1,

		Metadata: &BuiltinMetadata{
			Description: "Read entire file contents as a string",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to read"},
			},
			Returns: "String containing file contents",
			Examples: []Example{
				{Code: `_fs_readFile("config.yaml")`, Description: "Reads config.yaml contents"},
			},
			LongDesc:  "Respects AILANG_FS_SANDBOX environment variable for sandboxed execution.",
			SeeAlso:   []string{"_fs_writeFile", "_fs_exists"},
			Since:     "v0.2.0",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "read", "io"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_readFile: %v", err))
	}

	// _fs_readFileBytes
	impl1b := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}
	type1b := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(
			T.App("Result", T.String(), T.String()),
		).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_readFileBytes", NumArgs: 1, IsPure: false, Effect: "FS", Type: type1b, Impl: impl1b,

		Metadata: &BuiltinMetadata{
			Description: "Read entire file contents as a base64-encoded string",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to read"},
			},
			Returns: "Result[string, string] - Ok(base64 content) or Err(error message)",
			Examples: []Example{
				{Code: `_fs_readFileBytes("image.png")`, Description: "Returns Ok(base64-encoded content)"},
			},
			LongDesc:  "Reads binary file contents and returns as base64. Use _bytes_from_base64 to decode. Respects AILANG_FS_SANDBOX.",
			SeeAlso:   []string{"_fs_readFile", "_bytes_from_base64", "_zip_readEntryBytes"},
			Since:     "v0.8.0",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "read", "binary", "base64"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_readFileBytes: %v", err))
	}

	// _fs_writeFile
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "writeFile", args)
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.Unit()).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_writeFile", NumArgs: 2, IsPure: false, Effect: "FS", Type: type2, Impl: impl2,

		Metadata: &BuiltinMetadata{
			Description: "Write string content to a file (truncates if exists)",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to write"},
				{Name: "content", Description: "String content to write"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_fs_writeFile("output.txt", "Hello, World!")`, Description: "Writes string to output.txt"},
			},
			LongDesc:  "Respects AILANG_FS_SANDBOX environment variable. File permissions: 0644 (owner: rw, group: r, others: r).",
			SeeAlso:   []string{"_fs_readFile"},
			Since:     "v0.2.0",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "write", "io"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_writeFile: %v", err))
	}

	// _fs_writeFileBytes
	impl2b := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "writeFileBytes", args)
	}
	type2b := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.Bytes()).Returns(T.Unit()).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_writeFileBytes", NumArgs: 2, IsPure: false, Effect: "FS", Type: type2b, Impl: impl2b,

		Metadata: &BuiltinMetadata{
			Description: "Write raw bytes to a file (truncates if exists)",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to write"},
				{Name: "data", Description: "Bytes to write"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_fs_writeFileBytes("output.bin", _bytes_from_string("hello"))`, Description: "Writes bytes to output.bin"},
			},
			LongDesc:  "Writes raw byte data to disk. Use for binary files (PCM audio, images, WAV). Respects AILANG_FS_SANDBOX. File permissions: 0644.",
			SeeAlso:   []string{"_fs_readFileBytes", "_fs_writeFile", "_bytes_from_string"},
			Since:     "v0.8.2",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "write", "binary", "bytes"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_writeFileBytes: %v", err))
	}

	// _fs_exists
	impl3 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "exists", args)
	}
	type3 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Bool()).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_exists", NumArgs: 1, IsPure: false, Effect: "FS", Type: type3, Impl: impl3,

		Metadata: &BuiltinMetadata{
			Description: "Check if file or directory exists",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to check"},
			},
			Returns: "True if file/directory exists, false otherwise",
			Examples: []Example{
				{Code: `_fs_exists("/tmp")`, Description: "Returns true if /tmp exists"},
				{Code: `if _fs_exists("config.yaml") then _fs_readFile("config.yaml") else "default"`, Description: "Conditional file reading"},
			},
			LongDesc:  "Respects AILANG_FS_SANDBOX environment variable for sandboxed execution.",
			SeeAlso:   []string{"_fs_readFile", "_fs_writeFile"},
			Since:     "v0.3.25",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "exists", "check"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_exists: %v", err))
	}

	// _fs_appendFile
	impl4 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "appendFile", args)
	}
	type4 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.Unit()).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_appendFile", NumArgs: 2, IsPure: false, Effect: "FS", Type: type4, Impl: impl4,

		Metadata: &BuiltinMetadata{
			Description: "Append string content to a file (creates if not exists)",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to append to"},
				{Name: "content", Description: "String content to append"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_fs_appendFile("log.txt", "new line\n")`, Description: "Appends a line to log.txt"},
			},
			LongDesc:  "Appends to file without reading entire contents. O(1) per call. Respects AILANG_FS_SANDBOX. File permissions: 0644.",
			SeeAlso:   []string{"_fs_writeFile", "_fs_appendFileBytes"},
			Since:     "v0.8.0",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "append", "io"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_appendFile: %v", err))
	}

	// _fs_appendFileBytes
	impl4b := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "appendFileBytes", args)
	}
	type4b := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.Bytes()).Returns(T.Unit()).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_appendFileBytes", NumArgs: 2, IsPure: false, Effect: "FS", Type: type4b, Impl: impl4b,

		Metadata: &BuiltinMetadata{
			Description: "Append raw bytes to a file (creates if not exists)",
			Params: []ParamDoc{
				{Name: "path", Description: "Path to file to append to"},
				{Name: "data", Description: "Bytes to append"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_fs_appendFileBytes("output.pcm", pcmFrame)`, Description: "Appends PCM audio frame to file"},
			},
			LongDesc:  "Appends raw bytes without reading entire file. O(1) per call — ideal for streaming binary data to disk (e.g., PCM audio frames). Respects AILANG_FS_SANDBOX. File permissions: 0644.",
			SeeAlso:   []string{"_fs_writeFileBytes", "_fs_appendFile"},
			Since:     "v0.8.0",
			Stability: StabilityStable,
			Tags:      []string{"fs", "file", "append", "binary", "bytes", "streaming"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_appendFileBytes: %v", err))
	}

	// _fs_listDir — M-DOCPARSE-DX M3
	implListDir := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "listDir", args)
	}
	typeListDir := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.List(T.String())).Effects("FS")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/fs", Name: "_fs_listDir", NumArgs: 1, IsPure: false, Effect: "FS", Type: typeListDir, Impl: implListDir,
		Metadata: &BuiltinMetadata{
			Description: "List directory entries sorted by name",
			Params: []ParamDoc{
				{Name: "path", Description: "Directory path to list"},
			},
			Returns:   "Sorted list of entry names (files and directories)",
			Examples:  []Example{{Code: `_fs_listDir("examples/")`, Description: "Returns sorted list of filenames"}},
			LongDesc:  "Uses os.ReadDir which returns entries sorted by name. Respects AILANG_FS_SANDBOX.",
			SeeAlso:   []string{"_fs_readFile", "_fs_exists"},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"fs", "directory", "list", "readdir"},
			Category:  "fs",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _fs_listDir: %v", err))
	}
}
