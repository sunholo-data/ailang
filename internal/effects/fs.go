package effects

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo/ailang/internal/eval"
)

// init registers FS effect operations
func init() {
	RegisterOp("FS", "readFile", fsReadFile)
	RegisterOp("FS", "readFileBytes", fsReadFileBytes)
	RegisterOp("FS", "writeFile", fsWriteFile)
	RegisterOp("FS", "writeFileBytes", fsWriteFileBytes)
	RegisterOp("FS", "appendFile", fsAppendFile)
	RegisterOp("FS", "appendFileBytes", fsAppendFileBytes)
	RegisterOp("FS", "exists", fsExists)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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

	// Apply sandbox
	if ctx.Env.Sandbox != "" {
		path = filepath.Join(ctx.Env.Sandbox, path)
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
		path = filepath.Join(ctx.Env.Sandbox, path)
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
