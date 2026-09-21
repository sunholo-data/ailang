package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// fsWriteArgs unpacks (path, String content) for the string write/append ops.
func fsWriteArgs(op string, args []eval.Value) (string, []byte, error) {
	if len(args) != 2 {
		return "", nil, fmt.Errorf("%s: expected 2 arguments, got %d", op, len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return "", nil, fmt.Errorf("%s: expected String for path, got %T", op, args[0])
	}
	contentVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return "", nil, fmt.Errorf("%s: expected String for content, got %T", op, args[1])
	}
	return pathVal.Value, []byte(contentVal.Value), nil
}

// fsWriteBytesArgs unpacks (path, Bytes data) for the binary write/append ops.
func fsWriteBytesArgs(op string, args []eval.Value) (string, []byte, error) {
	if len(args) != 2 {
		return "", nil, fmt.Errorf("%s: expected 2 arguments, got %d", op, len(args))
	}
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return "", nil, fmt.Errorf("%s: expected String for path, got %T", op, args[0])
	}
	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return "", nil, fmt.Errorf("%s: expected Bytes for data, got %T", op, args[1])
	}
	return pathVal.Value, bytesVal.Value, nil
}

// fsWriteFile implements FS.writeFile(path: String, content: String) -> ()
//
// Writes a string to a file, creating it if it doesn't exist and truncating
// it otherwise. File permissions: 0644.
//
// Example AILANG code:
//
//	writeFile("output.txt", "Hello, World!")
func fsWriteFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteArgs("writeFile", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsWriteAll(b, path, data); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsWriteFileBytes implements FS.writeFileBytes(path: String, data: Bytes) -> ()
//
// Writes raw bytes to a file, creating it if it doesn't exist and truncating
// it otherwise. File permissions: 0644.
//
// Example AILANG code:
//
//	import std/bytes (fromString)
//	writeFileBytes("output.bin", fromString("binary data"))
func fsWriteFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteBytesArgs("writeFileBytes", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsWriteAll(b, path, data); err != nil {
		return nil, fmt.Errorf("writeFileBytes: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsAppendFile implements FS.appendFile(path: String, content: String) -> ()
//
// Appends a string to a file, creating it if it doesn't exist. Permissions 0644.
//
// Example AILANG code:
//
//	appendFile("log.txt", "new line\n")
func fsAppendFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteArgs("appendFile", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsAppendAll(b, path, data); err != nil {
		return nil, fmt.Errorf("appendFile: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsAppendFileBytes implements FS.appendFileBytes(path: String, data: Bytes) -> ()
//
// Appends raw bytes to a file, creating it if it doesn't exist. Ideal for
// streaming binary data to disk (e.g., accumulating PCM audio frames).
//
// Example AILANG code:
//
//	import std/bytes (fromBase64)
//	import std/option (Option, Some, None)
//	match fromBase64(audioChunk) {
//	  Some(pcm) => appendFileBytes("output.pcm", pcm),
//	  None => ()
//	}
func fsAppendFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteBytesArgs("appendFileBytes", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsAppendAll(b, path, data); err != nil {
		return nil, fmt.Errorf("appendFileBytes: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// ============================================================================
// M-AILANG-FS-RESULT (v0.16.0): Result-returning write/append variants
// ============================================================================

// fsWriteFileResult implements FS.writeFileResult(path: String, content: String) -> Result[(), String]
func fsWriteFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteArgs("writeFileResult", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot write file: %v", err)), nil
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsWriteAll(b, path, data); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot write file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsAppendFileResult implements FS.appendFileResult(path: String, content: String) -> Result[(), String]
func fsAppendFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteArgs("appendFileResult", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot append to file: %v", err)), nil
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := fsAppendAll(b, path, data); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot append to file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}
