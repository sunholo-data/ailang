//go:build !js

package effects

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// ResolveAllowlist resolves command names to absolute paths via exec.LookPath.
// This pins paths at startup to prevent TOCTOU attacks.
func (pc *ProcessContext) ResolveAllowlist(allowlistStr string) error {
	if allowlistStr == "" {
		return nil
	}

	pc.HasAllowlist = true
	pc.Allowlist = make(map[string]string)

	for _, cmd := range strings.Split(allowlistStr, ",") {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		if strings.HasPrefix(cmd, "/") {
			// Absolute path — use directly
			pc.Allowlist[cmd] = cmd
		} else {
			// Name — resolve via LookPath
			resolved, err := exec.LookPath(cmd)
			if err != nil {
				// Command not found at startup — still add to allowlist
				// but with empty resolved path (will fail at exec time with NotFound)
				pc.Allowlist[cmd] = ""
			} else {
				pc.Allowlist[cmd] = resolved
			}
		}
	}
	return nil
}

func init() {
	RegisterOp("Process", "exec", processExec)
}

// processExec implements Process.exec(cmd: String, args: List[String])
//
//	-> Result[ProcessOutput, ProcessError] ! {Process}
//
// Executes an external command synchronously with timeout and output limits.
// No shell expansion — uses os/exec.Command directly.
//
// Completion semantics:
//   - Ok for all completed processes (even non-zero exit)
//   - Err only for infra/spawn failures
func processExec(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("exec: expected 2 arguments (cmd, args), got %d", len(args))
	}

	cmdVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("exec: expected String for cmd, got %T", args[0])
	}
	cmdName := cmdVal.Value

	argsVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("exec: expected List for args, got %T", args[1])
	}

	cmdArgs := make([]string, len(argsVal.Elements))
	for i, elem := range argsVal.Elements {
		strVal, ok := elem.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("exec: args[%d] expected String, got %T", i, elem)
		}
		cmdArgs[i] = strVal.Value
	}

	// Get process context (may be nil if not configured)
	pc := ctx.Process
	if pc == nil {
		pc = NewProcessContext()
	}

	// Step 1: Allowlist check
	var resolvedPath string
	if pc.HasAllowlist {
		resolved, allowed := pc.Allowlist[cmdName]
		if !allowed {
			return makeProcessResultErr("NotAllowed", cmdName), nil
		}
		if resolved == "" {
			return makeProcessResultErr("NotFound", cmdName), nil
		}
		resolvedPath = resolved
	} else {
		// No allowlist — resolve via LookPath
		resolved, err := exec.LookPath(cmdName)
		if err != nil {
			return makeProcessResultErr("NotFound", cmdName), nil
		}
		resolvedPath = resolved
	}

	// Step 2: Set up command with timeout
	execCtx, cancel := context.WithTimeout(context.Background(), pc.Timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, resolvedPath, cmdArgs...)

	// Set working directory from sandbox if configured
	if ctx.Env.Sandbox != "" {
		cmd.Dir = ctx.Env.Sandbox
	}

	// Step 3: Capture stdout/stderr with output limits
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutLimited := &limitedWriter{w: &stdoutBuf, max: pc.MaxOutput}
	stderrLimited := &limitedWriter{w: &stderrBuf, max: pc.MaxOutput - int64(stdoutBuf.Len())}

	cmd.Stdout = stdoutLimited
	cmd.Stderr = stderrLimited

	// Step 4: Execute
	startTime := time.Now()
	err := cmd.Run()
	durationMs := time.Since(startTime).Milliseconds()

	// Step 5: Check for output limit exceeded
	if stdoutLimited.exceeded || stderrLimited.exceeded {
		totalBytes := stdoutBuf.Len() + stderrBuf.Len()
		return makeProcessResultErrInt("OutputLimitExceeded", int64(totalBytes)), nil
	}

	// Step 6: Handle errors
	if err != nil {
		// Timeout?
		if execCtx.Err() == context.DeadlineExceeded {
			return makeProcessResultErrInt("Timeout", durationMs), nil
		}

		// Exit error with signal (abnormal termination)?
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Check for signal kill
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				sig := status.Signal()
				return makeProcessResultErrSignal(int(sig), sig.String()), nil
			}
			// Non-zero exit — this is Ok (completion semantics)
			return makeProcessResultOk(
				stdoutBuf.Bytes(),
				stderrBuf.Bytes(),
				exitErr.ExitCode(),
				false,
				resolvedPath,
			), nil
		}

		// Permission denied?
		if errors.Is(err, exec.ErrNotFound) {
			return makeProcessResultErr("NotFound", cmdName), nil
		}
		if strings.Contains(err.Error(), "permission denied") {
			return makeProcessResultErr("PermissionDenied", cmdName), nil
		}

		// Other spawn failure
		return makeProcessResultErr("SpawnFailed", err.Error()), nil
	}

	// Step 7: Success — process completed with exit code 0
	return makeProcessResultOk(
		stdoutBuf.Bytes(),
		stderrBuf.Bytes(),
		0,
		false,
		resolvedPath,
	), nil
}

// limitedWriter wraps a writer with a byte limit.
// When exceeded, further writes are silently discarded and exceeded is set.
type limitedWriter struct {
	w        io.Writer
	max      int64
	written  int64
	exceeded bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.exceeded {
		return len(p), nil // Discard but don't error
	}
	remaining := lw.max - lw.written
	if int64(len(p)) > remaining {
		lw.exceeded = true
		// Write what we can, then discard the rest
		if remaining > 0 {
			_, err := lw.w.Write(p[:remaining])
			if err != nil {
				return 0, err
			}
			lw.written += remaining
		}
		return len(p), nil
	}
	n, err := lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}

// makeProcessResultOk wraps ProcessOutput in Result's Ok constructor
func makeProcessResultOk(stdout, stderr []byte, exitCode int, truncated bool, resolvedPath string) eval.Value {
	output := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"stdout":       &eval.BytesValue{Value: stdout},
			"stderr":       &eval.BytesValue{Value: stderr},
			"exitCode":     &eval.IntValue{Value: exitCode},
			"truncated":    &eval.BoolValue{Value: truncated},
			"resolvedPath": &eval.StringValue{Value: resolvedPath},
		},
	}
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{output},
	}
}

// makeProcessError constructs a ProcessError ADT value
func makeProcessError(ctorName string, fields []eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/process",
		TypeName:   "ProcessError",
		CtorName:   ctorName,
		Fields:     fields,
	}
}

// makeProcessResultErr wraps a ProcessError(string) in Result's Err constructor
func makeProcessResultErr(ctorName, message string) eval.Value {
	procErr := makeProcessError(ctorName, []eval.Value{&eval.StringValue{Value: message}})
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{procErr},
	}
}

// makeProcessResultErrInt wraps a ProcessError(int) in Result's Err constructor
func makeProcessResultErrInt(ctorName string, value int64) eval.Value {
	procErr := makeProcessError(ctorName, []eval.Value{&eval.IntValue{Value: int(value)}})
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{procErr},
	}
}

// makeProcessResultErrSignal wraps AbnormalExit(int, string) in Result's Err constructor
func makeProcessResultErrSignal(sigNum int, sigName string) eval.Value {
	procErr := makeProcessError("AbnormalExit", []eval.Value{
		&eval.IntValue{Value: sigNum},
		&eval.StringValue{Value: sigName},
	})
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{procErr},
	}
}
