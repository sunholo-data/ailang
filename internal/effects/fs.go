package effects

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// FS effect — every operation is anchored to the context's filesystem
// backend (fs_root.go): the confined os.Root handle when AILANG_FS_SANDBOX /
// the policy's fs_sandbox is set, the host filesystem otherwise. Paths are
// passed to the backend as the program spelled them; the root handle, not a
// string check, decides containment (M-EXECUTOR-POLICY-HARDENING M1).

// logSandboxReject emits diagnostics when an FS operation silently returns false
// because the requested path escapes the sandbox (exists/isDir/isFile contract).
//
// Two diagnostic channels, both zero-cost when inactive:
//  1. AILANG_FS_SANDBOX_DEBUG=1 → stderr line:
//     [ailang/sandbox] REJECT <op>(<path>) → escapes sandbox "<sandbox>" (returns <result>)
//  2. AILANG_TRACE=deep + active trace collector → RecordEffect event tagged sandbox.reject
func logSandboxReject(ctx *EffContext, op, attemptedPath, result string) {
	if config.FSSandboxDebug() {
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
	RegisterOp("FS", "renameFile", fsRenameFile)
	RegisterOp("FS", "renameFileResult", fsRenameFileResult)

	// M-AILANG-FS-RESULT (v0.16.0): Result-returning variants for agent
	// runtimes that need to recover from fs syscall failures without
	// crashing. The void-returning variants above panic through the effect
	// system; these wrap errors as Err(string).
	RegisterOp("FS", "readFileResult", fsReadFileResult)
	RegisterOp("FS", "writeFileResult", fsWriteFileResult)
	RegisterOp("FS", "appendFileResult", fsAppendFileResult)
	RegisterOp("FS", "removeFileResult", fsRemoveFileResult)
	RegisterOp("FS", "mkdirAllResult", fsMkdirAllResult)
	RegisterOp("FS", "mkdirResult", fsMkdirResult)
	RegisterOp("FS", "removeDirResult", fsRemoveDirResult)
}

// fsPathArg unpacks args[i] as the String path argument of op.
func fsPathArg(op string, args []eval.Value, i, want int) (string, error) {
	if len(args) != want {
		return "", fmt.Errorf("%s: expected %d argument%s, got %d", op, want, plural(want), len(args))
	}
	v, ok := args[i].(*eval.StringValue)
	if !ok {
		return "", fmt.Errorf("%s: expected String, got %T", op, args[i])
	}
	return v.Value, nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// fsReadFile implements FS.readFile(path: String) -> String
//
// Reads the entire contents of a file and returns it as a string. With a
// sandbox the read goes through the confined root: `readFile("data.txt")`
// reads "<sandbox>/data.txt", and any path that leaves the root fails.
//
// Example AILANG code:
//
//	let config = readFile("config.yaml")
func fsReadFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("readFile", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	content, err := readCapped(b, path, ctx.Env.FSMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("readFile: %w", err)
	}
	return &eval.StringValue{Value: string(content)}, nil
}

// fsExists implements FS.exists(path: String) -> Bool
//
// Checks if a file or directory exists. A path the sandbox refuses is
// reported as non-existent (false), with the optional reject diagnostics.
//
// Example AILANG code:
//
//	if exists("config.yaml") then
//	    readFile("config.yaml")
//	else
//	    "default config"
func fsExists(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("exists", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	if _, err := b.Stat(path); err != nil {
		fsRejectProbe(ctx, "exists", path, err)
		return &eval.BoolValue{Value: false}, nil
	}
	return &eval.BoolValue{Value: true}, nil
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
//
// Returns:
//   - Ok(base64-encoded string) on success
//   - Err(error message) if file doesn't exist, permission denied, escapes the sandbox, etc.
func fsReadFileBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("readFileBytes", args, 0, 1)
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
	encoded := base64.StdEncoding.EncodeToString(content)
	return fsMakeOk(&eval.StringValue{Value: encoded}), nil
}

// fsListDir implements FS.listDir(path: String) -> [String]
// M-DOCPARSE-DX M3: Returns sorted list of entry names in a directory.
func fsListDir(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	path, err := fsPathArg("listDir", args, 0, 1)
	if err != nil {
		return nil, err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	entries, err := b.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listDir: %w", err)
	}
	// Both backends return entries sorted by name.
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
	path, err := fsPathArg("readFileResult", args, 0, 1)
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
	return fsMakeOk(&eval.StringValue{Value: string(content)}), nil
}
