package effects

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// logSandboxReject emits diagnostics when an FS operation silently returns false
// because the requested path escapes the sandbox (exists/isDir/isFile contract).
//
// Two diagnostic channels, both zero-cost when inactive:
//  1. AILANG_FS_SANDBOX_DEBUG=1 → stderr line:
//     [ailang/sandbox] REJECT <op>(<path>) → escapes sandbox "<sandbox>" (returns <result>)
//  2. AILANG_TRACE=deep + active trace collector → RecordEffect event tagged sandbox.reject
func logSandboxReject(ctx *EffContext, op, attemptedPath, result string) {
	if os.Getenv("AILANG_FS_SANDBOX_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[ailang/sandbox] REJECT %s(%q) → escapes sandbox %q (returns %s)\n",
			op, attemptedPath, ctx.Env.Sandbox, result)
	}

	if ctx.Trace != nil && ctx.Trace.Enabled() {
		if tier, err := trace.TierFromEnv(); err == nil && tier == trace.TierDeep {
			ctx.RecordEffect("FS", op+".sandbox.reject",
				[]string{attemptedPath, ctx.Env.Sandbox},
				result)
		}
	}
}

// resolveSandboxPath resolves a path against the sandbox root.
//
// Relative paths are joined with the sandbox (existing behaviour).
// Absolute paths that fall within the sandbox are returned as-is — this
// allows programs to use absolute paths for files they already know live
// inside the sandbox (e.g. config files resolved from an absolute workdir).
// Absolute paths that escape the sandbox are rejected with an error.
//
// "Absolute" here means absolute on *any* mainstream host, not just the
// current one. A .ail program saying fs.exists("/etc/passwd") must trigger
// the sandbox reject path consistently on Linux, macOS, and Windows — the
// security model can't depend on what filepath.IsAbs says about the same
// string on different hosts. Without this, on Windows the leading-slash
// path would be treated as relative, joined into the sandbox, and silently
// resolve to a missing file with no diagnostic.
//
// Before this fix, filepath.Join(sandbox, "/abs/path") produced
// "/sandbox/abs/path" (a doubled path that never exists on disk), causing
// all FS operations with absolute sandbox-relative paths to fail silently.
func resolveSandboxPath(sandbox, path string) (string, error) {
	if !isAbsoluteCrossPlatform(path) {
		return filepath.Join(sandbox, path), nil
	}
	clean := filepath.Clean(path)
	sandboxClean := filepath.Clean(sandbox)
	if clean != sandboxClean && !strings.HasPrefix(clean, sandboxClean+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes sandbox %q", path, sandbox)
	}
	return clean, nil
}

// init registers FS effect operations
func init() {
	RegisterOp("FS", "readFile", fsReadFile)
	RegisterOp("FS", "readFileBytes", fsReadFileBytes)
	RegisterOp("FS", "writeFile", fsWriteFile)
	RegisterOp("FS", "writeFileBytes", fsWriteFileBytes)
	RegisterOp("FS", "appendFile", fsAppendFile)
	RegisterOp("FS", "appendFileBytes", fsAppendFileBytes)
	RegisterOp("FS", "exists", fsExists)
	RegisterOp("FS", "listDir", fsListDir)
	RegisterOp("FS", "mkdir", fsMkdir)
	RegisterOp("FS", "mkdirAll", fsMkdirAll)
	RegisterOp("FS", "isDir", fsIsDir)
	RegisterOp("FS", "isFile", fsIsFile)
	RegisterOp("FS", "removeFile", fsRemoveFile)

	// M-AILANG-FS-RESULT (v0.16.0): Result-returning variants for agent
	// runtimes that need to recover from fs syscall failures without
	// crashing. The void-returning variants above panic through the effect
	// system; these wrap errors as Err(string).
	RegisterOp("FS", "readFileResult", fsReadFileResult)
	RegisterOp("FS", "writeFileResult", fsWriteFileResult)
	RegisterOp("FS", "appendFileResult", fsAppendFileResult)
	RegisterOp("FS", "removeFileResult", fsRemoveFileResult)
	RegisterOp("FS", "mkdirAllResult", fsMkdirAllResult)
}

// fsReadFile implements FS.readFile(path: String) -> String
//
// Reads the entire contents of a file and returns it as a string.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue] - the file path
//
// Returns:
//   - StringValue with file contents
//   - Error if file doesn't exist, permission denied, or wrong arguments
//
// Example AILANG code:
//
//	let config = readFile("config.yaml")
//
// With sandbox:
//
//	AILANG_FS_SANDBOX=/tmp ailang run app.ail --caps FS
//	-- readFile("data.txt") reads "/tmp/data.txt"
func fsReadFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("readFile: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("readFile: expected String, got %T", args[0])
	}

	path := pathVal.Value

	// Apply sandbox if configured
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("readFile: %w", err)
	}

	return &eval.StringValue{Value: string(content)}, nil
}

// fsExists implements FS.exists(path: String) -> Bool
//
// Checks if a file or directory exists at the given path.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue] - the file path
//
// Returns:
//   - BoolValue true if file/directory exists, false otherwise
//   - Error if wrong arguments
//
// Example AILANG code:
//
//	if exists("config.yaml") then
//	    readFile("config.yaml")
//	else
//	    "default config"
func fsExists(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("exists: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("exists: expected String, got %T", args[0])
	}

	path := pathVal.Value

	// Apply sandbox — paths outside sandbox are treated as non-existent.
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			logSandboxReject(ctx, "exists", path, "false")
			return &eval.BoolValue{Value: false}, nil
		}
		path = resolved
	}

	// Check existence
	_, err := os.Stat(path)
	exists := err == nil

	return &eval.BoolValue{Value: exists}, nil
}

// Result helpers for Ok/Err return values

func fsMakeOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{val},
	}
}

func fsMakeErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// fsReadFileBytes implements FS.readFileBytes(path: String) -> Result[string, string]
//
// Reads the entire contents of a file and returns it as a base64-encoded string.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue] - the file path
//
// Returns:
//   - Ok(base64-encoded string) on success
//   - Err(error message) if file doesn't exist, permission denied, etc.
func fsReadFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("readFileBytes: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("readFileBytes: expected String, got %T", args[0])
	}

	path := pathVal.Value

	// Apply sandbox if configured
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return fsMakeErr(fmt.Sprintf("cannot read file: %v", err)), nil
	}

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(content)
	return fsMakeOk(&eval.StringValue{Value: encoded}), nil
}

// fsListDir implements FS.listDir(path: String) -> [String]
// M-DOCPARSE-DX M3: Returns sorted list of entry names in a directory.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsListDir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("listDir: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("listDir: expected String, got %T", args[0])
	}

	path := pathVal.Value

	// Apply sandbox if configured
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listDir: %w", err)
	}

	// os.ReadDir returns entries sorted by name
	result := make([]eval.Value, 0, len(entries))
	for _, entry := range entries {
		result = append(result, &eval.StringValue{Value: entry.Name()})
	}

	return &eval.ListValue{Elements: result}, nil
}

// fsReadFileResult implements FS.readFileResult(path: String) -> Result[String, String]
//
// M-AILANG-FS-RESULT (v0.16.0): wraps syscall failures as Err(message) instead
// of returning a Go error, so agent runtimes can recover without crashing.
func fsReadFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("readFileResult: expected 1 argument, got %d", len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("readFileResult: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fsMakeErr(fmt.Sprintf("cannot read file: %v", err)), nil
	}
	return fsMakeOk(&eval.StringValue{Value: string(content)}), nil
}
