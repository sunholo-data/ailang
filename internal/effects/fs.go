package effects

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// resolveSandboxPath resolves a path against the sandbox root.
//
// Relative paths are joined with the sandbox (existing behaviour).
// Absolute paths that fall within the sandbox are returned as-is — this
// allows programs to use absolute paths for files they already know live
// inside the sandbox (e.g. config files resolved from an absolute workdir).
// Absolute paths that escape the sandbox are rejected with an error.
//
// Before this fix, filepath.Join(sandbox, "/abs/path") produced
// "/sandbox/abs/path" (a doubled path that never exists on disk), causing
// all FS operations with absolute sandbox-relative paths to fail silently.
func resolveSandboxPath(sandbox, path string) (string, error) {
	if !filepath.IsAbs(path) {
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

// fsWriteFile implements FS.writeFile(path: String, content: String) -> ()
//
// Writes a string to a file, creating it if it doesn't exist.
// If the file exists, it will be truncated.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue, StringValue] - file path and content
//
// Returns:
//   - UnitValue on success
//   - Error if write fails or wrong arguments
//
// Example AILANG code:
//
//	writeFile("output.txt", "Hello, World!")
//
// File permissions: 0644 (owner: rw, group: r, others: r)
func fsWriteFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("writeFile: expected 2 arguments, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("writeFile: expected String for path, got %T", args[0])
	}

	contentVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("writeFile: expected String for content, got %T", args[1])
	}

	path := pathVal.Value
	content := contentVal.Value

	// Apply sandbox
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	// Write file (0644 permissions)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// fsWriteFileBytes implements FS.writeFileBytes(path: String, data: Bytes) -> ()
//
// Writes raw bytes to a file, creating it if it doesn't exist.
// If the file exists, it will be truncated.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue, BytesValue] - file path and binary data
//
// Returns:
//   - UnitValue on success
//   - Error if write fails or wrong arguments
//
// Example AILANG code:
//
//	import std/bytes (fromString)
//	writeFileBytes("output.bin", fromString("binary data"))
//
// File permissions: 0644 (owner: rw, group: r, others: r)
func fsWriteFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("writeFileBytes: expected 2 arguments, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("writeFileBytes: expected String for path, got %T", args[0])
	}

	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("writeFileBytes: expected Bytes for data, got %T", args[1])
	}

	path := pathVal.Value

	// Apply sandbox
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	// Write file (0644 permissions)
	err := os.WriteFile(path, bytesVal.Value, 0644)
	if err != nil {
		return nil, fmt.Errorf("writeFileBytes: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// fsAppendFile implements FS.appendFile(path: String, content: String) -> ()
//
// Appends a string to a file, creating it if it doesn't exist.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue, StringValue] - file path and content to append
//
// Returns:
//   - UnitValue on success
//   - Error if write fails or wrong arguments
//
// Example AILANG code:
//
//	appendFile("log.txt", "new line\n")
//
// File permissions: 0644 (owner: rw, group: r, others: r)
func fsAppendFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("appendFile: expected 2 arguments, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("appendFile: expected String for path, got %T", args[0])
	}

	contentVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("appendFile: expected String for content, got %T", args[1])
	}

	path := pathVal.Value

	// Apply sandbox
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("appendFile: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(contentVal.Value); err != nil {
		return nil, fmt.Errorf("appendFile: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// fsAppendFileBytes implements FS.appendFileBytes(path: String, data: Bytes) -> ()
//
// Appends raw bytes to a file, creating it if it doesn't exist.
// Ideal for streaming binary data to disk (e.g., accumulating PCM audio frames).
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
//
// Parameters:
//   - ctx: Effect context (with optional Sandbox configuration)
//   - args: [StringValue, BytesValue] - file path and binary data to append
//
// Returns:
//   - UnitValue on success
//   - Error if write fails or wrong arguments
//
// Example AILANG code:
//
//	import std/bytes (fromBase64)
//	import std/option (Option, Some, None)
//	match fromBase64(audioChunk) {
//	  Some(pcm) => appendFileBytes("output.pcm", pcm),
//	  None => ()
//	}
//
// File permissions: 0644 (owner: rw, group: r, others: r)
func fsAppendFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("appendFileBytes: expected 2 arguments, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("appendFileBytes: expected String for path, got %T", args[0])
	}

	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("appendFileBytes: expected Bytes for data, got %T", args[1])
	}

	path := pathVal.Value

	// Apply sandbox
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("appendFileBytes: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(bytesVal.Value); err != nil {
		return nil, fmt.Errorf("appendFileBytes: %w", err)
	}

	return &eval.UnitValue{}, nil
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

// fsMkdir implements FS.mkdir(path: String) -> ()
// Creates a single directory. Parent directories must already exist.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsMkdir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("mkdir: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("mkdir: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.Mkdir(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// fsMkdirAll implements FS.mkdirAll(path: String) -> ()
// Creates a directory and all parent directories (like mkdir -p).
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsMkdirAll(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("mkdirAll: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("mkdirAll: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdirAll: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// fsIsDir implements FS.isDir(path: String) -> Bool
// Returns true if the path exists and is a directory.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsIsDir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("isDir: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("isDir: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return &eval.BoolValue{Value: false}, nil
		}
		path = resolved
	}

	info, err := os.Stat(path)
	if err != nil {
		return &eval.BoolValue{Value: false}, nil
	}

	return &eval.BoolValue{Value: info.IsDir()}, nil
}

// fsIsFile implements FS.isFile(path: String) -> Bool
// Returns true if the path exists and is a regular file.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsIsFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("isFile: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("isFile: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return &eval.BoolValue{Value: false}, nil
		}
		path = resolved
	}

	info, err := os.Stat(path)
	if err != nil {
		return &eval.BoolValue{Value: false}, nil
	}

	return &eval.BoolValue{Value: info.Mode().IsRegular()}, nil
}

// fsRemoveFile implements FS.removeFile(path: String) -> ()
// Removes a file or empty directory.
// If AILANG_FS_SANDBOX is set, the path is restricted to the sandbox directory.
func fsRemoveFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("removeFile: expected 1 argument, got %d", len(args))
	}

	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("removeFile: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("removeFile: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// ============================================================================
// M-AILANG-FS-RESULT (v0.16.0): Result-returning variants
// ============================================================================
//
// Each handler below mirrors its void-returning twin above but wraps syscall
// failures as Err(message) using fsMakeErr/fsMakeOk instead of returning a
// Go error (which AILANG turns into a runtime panic that escapes the effect
// system). Agent runtimes that dispatch fs operations against user-supplied
// paths should prefer these over the panicking variants.

// fsReadFileResult implements FS.readFileResult(path: String) -> Result[String, String]
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

// fsWriteFileResult implements FS.writeFileResult(path: String, content: String) -> Result[(), String]
func fsWriteFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("writeFileResult: expected 2 arguments, got %d", len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("writeFileResult: expected String for path, got %T", args[0])
	}
	contentVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("writeFileResult: expected String for content, got %T", args[1])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.WriteFile(path, []byte(contentVal.Value), 0644); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot write file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsAppendFileResult implements FS.appendFileResult(path: String, content: String) -> Result[(), String]
func fsAppendFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("appendFileResult: expected 2 arguments, got %d", len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("appendFileResult: expected String for path, got %T", args[0])
	}
	contentVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("appendFileResult: expected String for content, got %T", args[1])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fsMakeErr(fmt.Sprintf("cannot append to file: %v", err)), nil
	}
	defer f.Close()
	if _, err := f.WriteString(contentVal.Value); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot append to file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsRemoveFileResult implements FS.removeFileResult(path: String) -> Result[(), String]
func fsRemoveFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("removeFileResult: expected 1 argument, got %d", len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("removeFileResult: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.Remove(path); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot remove file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsMkdirAllResult implements FS.mkdirAllResult(path: String) -> Result[(), String]
func fsMkdirAllResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("mkdirAllResult: expected 1 argument, got %d", len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("mkdirAllResult: expected String, got %T", args[0])
	}

	path := pathVal.Value
	if ctx.Env.Sandbox != "" {
		resolved, sandboxErr := resolveSandboxPath(ctx.Env.Sandbox, path)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		path = resolved
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot create directory: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}
