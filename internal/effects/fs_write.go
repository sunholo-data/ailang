package effects

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/eval"
)

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

// ============================================================================
// M-AILANG-FS-RESULT (v0.16.0): Result-returning write/append variants
// ============================================================================

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
