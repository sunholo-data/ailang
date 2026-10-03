package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Raw-bytes, Result-returning FS ops (std/fs readFileRaw / writeFileBytesResult).
//
// readFileBytes predates the bytes type and returns base64 TEXT in a
// Result[string, string]. A caller that wants the file's bytes had to decode
// it again. writeFileBytes panics on failure, unlike every other *Result
// write. These two ops close both gaps. They go through the same backend,
// size caps, sandbox and fs_deny_write checks as their siblings, and report
// every failure as Err(message), never as a Go error.

func init() {
	RegisterOp("FS", "readFileRaw", fsReadFileRaw)
	RegisterOp("FS", "writeFileBytesResult", fsWriteFileBytesResult)
}

// fsReadFileRaw implements FS.readFileRaw(path: String) -> Result[Bytes, String].
// It returns the file's bytes unchanged (no base64, no UTF-8 decoding).
func fsReadFileRaw(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("readFileRaw", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	content, err := readCapped(b, path, ctx.Env.FSMaxBytes)
	if err != nil {
		return fsMakeErr(fmt.Sprintf("cannot read file: %v", err)), nil
	}
	return fsMakeOk(&eval.BytesValue{Value: content}), nil
}

// fsWriteFileBytesResult implements
// FS.writeFileBytesResult(path: String, data: Bytes) -> Result[(), String]:
// the Result-returning twin of writeFileBytes (truncates, 0644).
func fsWriteFileBytesResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, data, err := fsWriteBytesArgs("writeFileBytesResult", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot write file: %v", err)), nil
	}
	if err := ctx.fsCheckMutation(path); err != nil {
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
