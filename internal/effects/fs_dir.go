package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// fsMkdir implements FS.mkdir(path: String) -> ()
// Creates a single directory. Parent directories must already exist.
func fsMkdir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("mkdir", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Mkdir(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsMkdirAll implements FS.mkdirAll(path: String) -> ()
// Creates a directory and all parent directories (like mkdir -p).
func fsMkdirAll(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("mkdirAll", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdirAll: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsIsDir implements FS.isDir(path: String) -> Bool
// Returns true if the path exists and is a directory. A path the sandbox
// refuses is false (with the optional reject diagnostics).
func fsIsDir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("isDir", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	info, err := b.Stat(path)
	if err != nil {
		fsRejectProbe(ctx, "isDir", path, err)
		return &eval.BoolValue{Value: false}, nil
	}
	return &eval.BoolValue{Value: info.IsDir()}, nil
}

// fsIsFile implements FS.isFile(path: String) -> Bool
// Returns true if the path exists and is a regular file. A path the sandbox
// refuses is false (with the optional reject diagnostics).
func fsIsFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("isFile", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	info, err := b.Stat(path)
	if err != nil {
		fsRejectProbe(ctx, "isFile", path, err)
		return &eval.BoolValue{Value: false}, nil
	}
	return &eval.BoolValue{Value: info.Mode().IsRegular()}, nil
}

// fsRemoveFile implements FS.removeFile(path: String) -> ()
// Removes a file, an empty directory, or a symlink entry (never its target).
func fsRemoveFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("removeFile", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Remove(path); err != nil {
		return nil, fmt.Errorf("removeFile: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// fsRenameArgs unpacks (oldPath, newPath).
func fsRenameArgs(op string, args []eval.Value) (string, string, error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("%s: expected 2 arguments, got %d", op, len(args))
	}
	oldPathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return "", "", fmt.Errorf("%s: expected String oldPath, got %T", op, args[0])
	}
	newPathVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return "", "", fmt.Errorf("%s: expected String newPath, got %T", op, args[1])
	}
	return oldPathVal.Value, newPathVal.Value, nil
}

// fsRenameFile implements FS.renameFile(oldPath, newPath: String) -> ()
// Atomically renames (moves) a file or directory on the same filesystem.
// With a sandbox BOTH operands go through the root — renaming across the
// boundary would otherwise be a sandbox-escape primitive. There is no
// copy/delete fallback across roots.
func fsRenameFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	oldPath, newPath, err := fsRenameArgs("renameFile", args)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("renameFile: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// ============================================================================
// M-AILANG-FS-RESULT (v0.16.0): Result-returning dir/remove variants
// ============================================================================

// fsRemoveFileResult implements FS.removeFileResult(path: String) -> Result[(), String]
func fsRemoveFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("removeFileResult", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Remove(path); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot remove file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsRenameFileResult implements FS.renameFileResult(oldPath, newPath: String) -> Result[(), String]
// Result-returning variant of fsRenameFile: sandbox applies to BOTH paths.
func fsRenameFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	oldPath, newPath, err := fsRenameArgs("renameFileResult", args)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Rename(oldPath, newPath); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot rename file: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsMkdirAllResult implements FS.mkdirAllResult(path: String) -> Result[(), String]
func fsMkdirAllResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("mkdirAllResult", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.MkdirAll(path, 0755); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot create directory: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsMkdirResult creates ONE directory and reports an existing one as Err —
// the atomic try-lock primitive (`mkdir lock.d` succeeds for exactly one
// caller). mkdirAllResult is the wrong tool for that: it is Ok on an existing
// directory. Daneel's rig-lock client had to exec("mkdir") to get this.
func fsMkdirResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("mkdirResult", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if err := b.Mkdir(path, 0755); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot create directory: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}

// fsRemoveDirResult removes an EMPTY directory (the lock release). A
// non-empty directory or a non-directory is Err, never a recursive delete.
// The check is Lstat so a symlink to a directory is "not a directory": the
// entry is never dereferenced, and its target is never removed.
func fsRemoveDirResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("removeDirResult", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	info, err := b.Lstat(path)
	if err != nil {
		return fsMakeErr(fmt.Sprintf("cannot remove directory: %v", err)), nil
	}
	if !info.IsDir() {
		return fsMakeErr(fmt.Sprintf("cannot remove directory: %s is not a directory", path)), nil
	}
	if err := b.Remove(path); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot remove directory: %v", err)), nil
	}
	return fsMakeOk(&eval.UnitValue{}), nil
}
