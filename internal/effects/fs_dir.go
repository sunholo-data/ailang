package effects

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/eval"
)

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
			logSandboxReject(ctx, "isDir", path, "false")
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
			logSandboxReject(ctx, "isFile", path, "false")
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

// fsRenameFile implements FS.renameFile(oldPath, newPath: String) -> ()
// Atomically renames (moves) a file or directory on the same filesystem.
// If AILANG_FS_SANDBOX is set, BOTH paths are restricted to the sandbox —
// renaming across the boundary would otherwise be a sandbox-escape primitive.
func fsRenameFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("renameFile: expected 2 arguments, got %d", len(args))
	}

	oldPathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("renameFile: expected String oldPath, got %T", args[0])
	}
	newPathVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("renameFile: expected String newPath, got %T", args[1])
	}

	oldPath := oldPathVal.Value
	newPath := newPathVal.Value
	if ctx.Env.Sandbox != "" {
		resolvedOld, err := resolveSandboxPath(ctx.Env.Sandbox, oldPath)
		if err != nil {
			return nil, err
		}
		resolvedNew, err := resolveSandboxPath(ctx.Env.Sandbox, newPath)
		if err != nil {
			return nil, err
		}
		oldPath = resolvedOld
		newPath = resolvedNew
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("renameFile: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// ============================================================================
// M-AILANG-FS-RESULT (v0.16.0): Result-returning dir/remove variants
// ============================================================================

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

// fsRenameFileResult implements FS.renameFileResult(oldPath, newPath: String) -> Result[(), String]
// Result-returning variant of fsRenameFile: sandbox applies to BOTH paths.
func fsRenameFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("renameFileResult: expected 2 arguments, got %d", len(args))
	}
	oldPathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("renameFileResult: expected String oldPath, got %T", args[0])
	}
	newPathVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("renameFileResult: expected String newPath, got %T", args[1])
	}

	oldPath := oldPathVal.Value
	newPath := newPathVal.Value
	if ctx.Env.Sandbox != "" {
		resolvedOld, err := resolveSandboxPath(ctx.Env.Sandbox, oldPath)
		if err != nil {
			return nil, err
		}
		resolvedNew, err := resolveSandboxPath(ctx.Env.Sandbox, newPath)
		if err != nil {
			return nil, err
		}
		oldPath = resolvedOld
		newPath = resolvedNew
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fsMakeErr(fmt.Sprintf("cannot rename file: %v", err)), nil
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
