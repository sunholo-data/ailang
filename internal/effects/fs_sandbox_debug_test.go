package effects

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// captureSandboxRejectLog temporarily sets AILANG_FS_SANDBOX_DEBUG=1, redirects
// stderr to a buffer, runs f, then restores both. Returns what was written to stderr.
func captureSandboxRejectLog(t *testing.T, f func()) string {
	t.Helper()

	// Set the debug env var.
	t.Setenv("AILANG_FS_SANDBOX_DEBUG", "1")

	// Redirect stderr.
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func sandboxCtx(sandbox string) *EffContext {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))
	ctx.Env.Sandbox = sandbox
	return ctx
}

// TestSandboxReject_Exists verifies exists() logs to stderr when path escapes sandbox.
func TestSandboxReject_Exists(t *testing.T) {
	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)
	escapingPath := "/etc/passwd"

	log := captureSandboxRejectLog(t, func() {
		args := []eval.Value{&eval.StringValue{Value: escapingPath}}
		val, err := fsExists(ctx, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b := val.(*eval.BoolValue); b.Value {
			t.Error("exists: want false for sandbox-escaping path, got true")
		}
	})

	if !strings.Contains(log, "[ailang/sandbox] REJECT exists") {
		t.Errorf("expected sandbox reject log for exists, got: %q", log)
	}
	if !strings.Contains(log, escapingPath) {
		t.Errorf("log should include attempted path %q, got: %q", escapingPath, log)
	}
	// The log formats the sandbox via %q, which Go-escapes backslashes. On
	// Windows the raw tempdir path contains `\` but the log emits `\\`, so a
	// naive Contains(log, sandbox) misses. Compare against the same quoted
	// form the logger produces.
	if !strings.Contains(log, strconv.Quote(sandbox)) {
		t.Errorf("log should include sandbox %q (formatted as %s), got: %q",
			sandbox, strconv.Quote(sandbox), log)
	}
}

// TestSandboxReject_IsDir verifies isDir() logs to stderr when path escapes sandbox.
func TestSandboxReject_IsDir(t *testing.T) {
	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)
	escapingPath := "/tmp/some-other-dir"

	log := captureSandboxRejectLog(t, func() {
		args := []eval.Value{&eval.StringValue{Value: escapingPath}}
		val, err := fsIsDir(ctx, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b := val.(*eval.BoolValue); b.Value {
			t.Error("isDir: want false for sandbox-escaping path, got true")
		}
	})

	if !strings.Contains(log, "[ailang/sandbox] REJECT isDir") {
		t.Errorf("expected sandbox reject log for isDir, got: %q", log)
	}
}

// TestSandboxReject_IsFile verifies isFile() logs to stderr when path escapes sandbox.
func TestSandboxReject_IsFile(t *testing.T) {
	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)
	escapingPath := "/etc/hosts"

	log := captureSandboxRejectLog(t, func() {
		args := []eval.Value{&eval.StringValue{Value: escapingPath}}
		val, err := fsIsFile(ctx, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b := val.(*eval.BoolValue); b.Value {
			t.Error("isFile: want false for sandbox-escaping path, got true")
		}
	})

	if !strings.Contains(log, "[ailang/sandbox] REJECT isFile") {
		t.Errorf("expected sandbox reject log for isFile, got: %q", log)
	}
}

// TestSandboxReject_NoLogWhenDebugUnset verifies no log is emitted when
// AILANG_FS_SANDBOX_DEBUG is not set (zero-cost in production).
func TestSandboxReject_NoLogWhenDebugUnset(t *testing.T) {
	os.Unsetenv("AILANG_FS_SANDBOX_DEBUG")

	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	args := []eval.Value{&eval.StringValue{Value: "/etc/passwd"}}
	fsExists(ctx, args)

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	if buf.Len() > 0 {
		t.Errorf("expected no stderr output without AILANG_FS_SANDBOX_DEBUG=1, got: %q", buf.String())
	}
}

// TestSandboxReject_WithinSandboxNoLog verifies no log for paths inside the sandbox.
func TestSandboxReject_WithinSandboxNoLog(t *testing.T) {
	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)

	t.Setenv("AILANG_FS_SANDBOX_DEBUG", "1")

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	// This path is inside the sandbox — no rejection should occur.
	args := []eval.Value{&eval.StringValue{Value: sandbox + "/nonexistent.txt"}}
	fsExists(ctx, args)

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	if buf.Len() > 0 {
		t.Errorf("expected no reject log for in-sandbox path, got: %q", buf.String())
	}
}

// TestSandboxReject_TraceEventDeepTier verifies that under AILANG_TRACE=deep
// a sandbox rejection records a FS.exists.sandbox.reject event in the trace collector.
func TestSandboxReject_TraceEventDeepTier(t *testing.T) {
	t.Setenv("AILANG_TRACE", "deep")

	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)
	ctx.Trace = trace.NewCollector()

	args := []eval.Value{&eval.StringValue{Value: "/etc/passwd"}}
	fsExists(ctx, args)

	events := ctx.Trace.Events()
	var found bool
	for _, ev := range events {
		if ev.Effect != nil && ev.Effect.EffectName == "FS" && strings.Contains(ev.Effect.OpName, "sandbox.reject") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected FS.exists.sandbox.reject trace event under deep tier; got %d events: %v", len(events), events)
	}
}

// TestSandboxReject_NoTraceEventStandardTier verifies no sandbox.reject trace event
// is emitted under the default standard tier (only deep tier opts in).
func TestSandboxReject_NoTraceEventStandardTier(t *testing.T) {
	t.Setenv("AILANG_TRACE", "standard")

	sandbox := t.TempDir()
	ctx := sandboxCtx(sandbox)
	ctx.Trace = trace.NewCollector()

	args := []eval.Value{&eval.StringValue{Value: "/etc/passwd"}}
	fsExists(ctx, args)

	for _, ev := range ctx.Trace.Events() {
		if ev.Effect != nil && strings.Contains(ev.Effect.OpName, "sandbox.reject") {
			t.Errorf("unexpected sandbox.reject trace event under standard tier: %v", ev)
		}
	}
}
