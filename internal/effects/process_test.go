//go:build !js

package effects

import (
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Helper to create a Process-capable context
func newProcessCtx() *EffContext {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Process"))
	ctx.Process = NewProcessContext()
	return ctx
}

// Helper to extract Result[ProcessOutput, ProcessError] fields
func unwrapOk(t *testing.T, result eval.Value) *eval.RecordValue {
	t.Helper()
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	if len(tagged.Fields) != 1 {
		t.Fatalf("Ok expected 1 field, got %d", len(tagged.Fields))
	}
	rec, ok := tagged.Fields[0].(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue inside Ok, got %T", tagged.Fields[0])
	}
	return rec
}

func unwrapErr(t *testing.T, result eval.Value) (string, []eval.Value) {
	t.Helper()
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	if len(tagged.Fields) != 1 {
		t.Fatalf("Err expected 1 field, got %d", len(tagged.Fields))
	}
	inner, ok := tagged.Fields[0].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue inside Err, got %T", tagged.Fields[0])
	}
	return inner.CtorName, inner.Fields
}

func TestProcessExec_EchoSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test requires unix")
	}
	ctx := newProcessCtx()

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "hello"},
			&eval.StringValue{Value: "world"},
		}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := unwrapOk(t, result)

	// Check stdout
	stdout, ok := rec.Fields["stdout"].(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected BytesValue for stdout, got %T", rec.Fields["stdout"])
	}
	if string(stdout.Value) != "hello world\n" {
		t.Errorf("stdout = %q, want %q", string(stdout.Value), "hello world\n")
	}

	// Check exitCode
	exitCode, ok := rec.Fields["exitCode"].(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue for exitCode, got %T", rec.Fields["exitCode"])
	}
	if exitCode.Value != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode.Value)
	}

	// Check truncated
	truncated, ok := rec.Fields["truncated"].(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue for truncated, got %T", rec.Fields["truncated"])
	}
	if truncated.Value {
		t.Error("truncated = true, want false")
	}
}

func TestProcessExec_NonZeroExit_IsOk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("false command test requires unix")
	}
	ctx := newProcessCtx()

	// "false" always exits with code 1
	args := []eval.Value{
		&eval.StringValue{Value: "false"},
		&eval.ListValue{Elements: []eval.Value{}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-zero exit is still Ok (completion semantics)
	rec := unwrapOk(t, result)

	exitCode := rec.Fields["exitCode"].(*eval.IntValue)
	if exitCode.Value != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode.Value)
	}
}

func TestProcessExec_NotFound(t *testing.T) {
	ctx := newProcessCtx()

	args := []eval.Value{
		&eval.StringValue{Value: "nonexistent_command_xyz"},
		&eval.ListValue{Elements: []eval.Value{}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctorName, fields := unwrapErr(t, result)
	if ctorName != "NotFound" {
		t.Errorf("error constructor = %q, want NotFound", ctorName)
	}
	if len(fields) != 1 {
		t.Fatalf("NotFound expected 1 field, got %d", len(fields))
	}
	msg := fields[0].(*eval.StringValue)
	if msg.Value != "nonexistent_command_xyz" {
		t.Errorf("NotFound message = %q, want %q", msg.Value, "nonexistent_command_xyz")
	}
}

func TestProcessExec_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{}) // No Process capability

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "test"}}},
	}

	_, err := Call(ctx, "Process", "exec", args)
	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Fatalf("expected *CapabilityError, got %T: %v", err, err)
	}
	if capErr.Effect != "Process" {
		t.Errorf("error effect = %q, want Process", capErr.Effect)
	}
}

func TestProcessExec_Allowlist_Allowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test requires unix")
	}
	ctx := newProcessCtx()

	// Set up allowlist with echo
	if err := ctx.Process.ResolveAllowlist("echo"); err != nil {
		t.Fatalf("ResolveAllowlist failed: %v", err)
	}

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "allowed"}}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := unwrapOk(t, result)
	stdout := rec.Fields["stdout"].(*eval.BytesValue)
	if string(stdout.Value) != "allowed\n" {
		t.Errorf("stdout = %q, want %q", string(stdout.Value), "allowed\n")
	}
}

func TestProcessExec_Allowlist_NotAllowed(t *testing.T) {
	ctx := newProcessCtx()

	// Set up allowlist with only "echo"
	if err := ctx.Process.ResolveAllowlist("echo"); err != nil {
		t.Fatalf("ResolveAllowlist failed: %v", err)
	}

	// Try to run "ls" (not in allowlist)
	args := []eval.Value{
		&eval.StringValue{Value: "ls"},
		&eval.ListValue{Elements: []eval.Value{}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctorName, fields := unwrapErr(t, result)
	if ctorName != "NotAllowed" {
		t.Errorf("error constructor = %q, want NotAllowed", ctorName)
	}
	if len(fields) != 1 {
		t.Fatalf("NotAllowed expected 1 field, got %d", len(fields))
	}
	msg := fields[0].(*eval.StringValue)
	if msg.Value != "ls" {
		t.Errorf("NotAllowed message = %q, want %q", msg.Value, "ls")
	}
}

func TestProcessExec_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep test requires unix")
	}
	ctx := newProcessCtx()
	ctx.Process.Timeout = 100 * 1e6 // 100ms (time.Duration is nanoseconds)

	args := []eval.Value{
		&eval.StringValue{Value: "sleep"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "10"}}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctorName, _ := unwrapErr(t, result)
	if ctorName != "Timeout" {
		t.Errorf("error constructor = %q, want Timeout", ctorName)
	}
}

func TestProcessExec_StderrCapture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh test requires unix")
	}
	ctx := newProcessCtx()

	// Use sh -c to write to stderr
	args := []eval.Value{
		&eval.StringValue{Value: "sh"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "-c"},
			&eval.StringValue{Value: "echo err_msg >&2"},
		}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := unwrapOk(t, result)
	stderr := rec.Fields["stderr"].(*eval.BytesValue)
	if string(stderr.Value) != "err_msg\n" {
		t.Errorf("stderr = %q, want %q", string(stderr.Value), "err_msg\n")
	}
}

func TestProcessExec_ResolvedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test requires unix")
	}
	ctx := newProcessCtx()

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "test"}}},
	}

	result, err := Call(ctx, "Process", "exec", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := unwrapOk(t, result)
	resolvedPath := rec.Fields["resolvedPath"].(*eval.StringValue)
	// Should be an absolute path like /bin/echo or /usr/bin/echo
	if resolvedPath.Value == "" || resolvedPath.Value[0] != '/' {
		t.Errorf("resolvedPath = %q, want absolute path", resolvedPath.Value)
	}
}

func TestProcessContext_ResolveAllowlist(t *testing.T) {
	pc := NewProcessContext()

	// Empty allowlist should not set HasAllowlist
	if err := pc.ResolveAllowlist(""); err != nil {
		t.Fatalf("empty allowlist failed: %v", err)
	}
	if pc.HasAllowlist {
		t.Error("empty allowlist set HasAllowlist=true")
	}

	// Resolve with known and unknown commands
	if err := pc.ResolveAllowlist("echo,nonexistent_xyz"); err != nil {
		t.Fatalf("ResolveAllowlist failed: %v", err)
	}
	if !pc.HasAllowlist {
		t.Error("allowlist not set after ResolveAllowlist")
	}

	// echo should have a resolved path
	echoPath, ok := pc.Allowlist["echo"]
	if !ok {
		t.Fatal("echo not in allowlist")
	}
	if echoPath == "" {
		t.Error("echo resolved to empty path")
	}

	// nonexistent should have empty resolved path
	unknownPath, ok := pc.Allowlist["nonexistent_xyz"]
	if !ok {
		t.Fatal("nonexistent_xyz not in allowlist")
	}
	if unknownPath != "" {
		t.Errorf("nonexistent_xyz resolved to %q, want empty", unknownPath)
	}
}

func TestProcessContext_AbsolutePath(t *testing.T) {
	pc := NewProcessContext()

	if err := pc.ResolveAllowlist("/usr/bin/echo"); err != nil {
		t.Fatalf("ResolveAllowlist failed: %v", err)
	}

	path, ok := pc.Allowlist["/usr/bin/echo"]
	if !ok {
		t.Fatal("/usr/bin/echo not in allowlist")
	}
	if path != "/usr/bin/echo" {
		t.Errorf("absolute path resolved to %q, want %q", path, "/usr/bin/echo")
	}
}

func TestLimitedWriter(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		var buf [32]byte
		w := &limitedWriter{w: nil, max: 100}
		// Use a real writer
		var actual []byte
		w.w = writerFunc(func(p []byte) (int, error) {
			actual = append(actual, p...)
			return len(p), nil
		})
		n, err := w.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("wrote %d bytes, want 5", n)
		}
		if w.exceeded {
			t.Error("exceeded flag set prematurely")
		}
		_ = buf
	})

	t.Run("exceeds limit", func(t *testing.T) {
		var actual []byte
		w := &limitedWriter{
			w: writerFunc(func(p []byte) (int, error) {
				actual = append(actual, p...)
				return len(p), nil
			}),
			max: 5,
		}
		n, err := w.Write([]byte("hello world"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 11 {
			t.Errorf("Write returned %d, want 11 (reports full write)", n)
		}
		if !w.exceeded {
			t.Error("exceeded flag not set")
		}
		// Should have written first 5 bytes
		if string(actual) != "hello" {
			t.Errorf("actual written = %q, want %q", string(actual), "hello")
		}
	})
}

// writerFunc adapts a function to io.Writer
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}


// TestProcessExec_WaitDelay_OrphanGrandchildNoHang is the regression for the 2026-06-22 motoko
// BashExec 7h hang: a `find / | head` pipeline whose orphaned `find` outlived the SIGKILLed bash
// and held the stdout pipe open, so cmd.Run() blocked forever despite the 30s timeout. Here bash
// backgrounds `sleep 60` (which inherits + holds the stdout pipe) then exits immediately. Without
// cmd.WaitDelay, cmd.Run() blocks ~60s waiting for the pipe to close; with it, Go force-closes the
// pipe after WaitDelay and the call returns promptly.
func TestProcessExec_WaitDelay_OrphanGrandchildNoHang(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash test requires unix")
	}
	if testing.Short() {
		t.Skip("WaitDelay timing test (~5s)")
	}
	ctx := newProcessCtx()
	args := []eval.Value{
		&eval.StringValue{Value: "bash"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "-c"},
			&eval.StringValue{Value: "sleep 60 & echo hi"},
		}},
	}
	start := time.Now()
	_, _ = Call(ctx, "Process", "exec", args)
	elapsed := time.Since(start)
	if elapsed > 20*time.Second {
		t.Fatalf("exec hung %v on an orphaned grandchild holding the stdout pipe — cmd.WaitDelay not effective (want <20s, would be ~60s unfixed)", elapsed)
	}
}
