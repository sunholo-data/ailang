//go:build !js

package effects

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

func TestManagedProcess_SpawnWriteClose(t *testing.T) {
	// Spawn cat, write data, close stdin, verify cat exits cleanly
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	mp, err := NewManagedProcess(t.Context(), catPath, []string{})
	if err != nil {
		t.Fatalf("NewManagedProcess: %v", err)
	}

	// Write some data
	if err := mp.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := mp.Write([]byte("world\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Close stdin — cat should exit
	mp.CloseStdin()

	// Wait for subprocess to exit
	select {
	case <-mp.done:
		// Good — subprocess exited
	case <-time.After(5 * time.Second):
		t.Fatal("subprocess did not exit within 5 seconds after stdin close")
	}
}

func TestManagedProcess_WriteAfterClose(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	mp, err := NewManagedProcess(t.Context(), catPath, []string{})
	if err != nil {
		t.Fatalf("NewManagedProcess: %v", err)
	}

	mp.CloseStdin()

	// Wait for subprocess to exit
	select {
	case <-mp.done:
	case <-time.After(5 * time.Second):
		t.Fatal("subprocess did not exit")
	}

	// Write after close should fail
	err = mp.Write([]byte("should fail"))
	if err == nil {
		t.Fatal("expected error writing after close, got nil")
	}
	if !strings.Contains(err.Error(), "stdin already closed") {
		t.Fatalf("expected 'stdin already closed' error, got: %v", err)
	}
}

func TestManagedProcess_CloseIdempotent(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	mp, err := NewManagedProcess(t.Context(), catPath, []string{})
	if err != nil {
		t.Fatalf("NewManagedProcess: %v", err)
	}

	// CloseStdin should be safe to call multiple times
	mp.CloseStdin()
	mp.CloseStdin() // Should not panic

	// Close should also be safe
	mp.Close()
	mp.Close() // Should not panic
}

func TestManagedProcess_KillOnClose(t *testing.T) {
	// Use sleep to create a long-running process, then Close() should kill it
	sleepPath, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not found in PATH")
	}

	mp, err := NewManagedProcess(t.Context(), sleepPath, []string{"60"})
	if err != nil {
		t.Fatalf("NewManagedProcess: %v", err)
	}

	// Close should kill the subprocess
	mp.Close()

	// Verify subprocess is no longer running
	select {
	case <-mp.done:
		// Good — subprocess killed
	case <-time.After(10 * time.Second):
		t.Fatal("subprocess not killed within 10 seconds after Close()")
	}
}

func TestManagedProcess_CommandNotFound(t *testing.T) {
	_, err := NewManagedProcess(t.Context(), "/nonexistent/binary", []string{})
	if err == nil {
		t.Fatal("expected error for nonexistent command, got nil")
	}
	if !strings.Contains(err.Error(), "start process") {
		t.Fatalf("expected 'start process' error, got: %v", err)
	}
}

func TestProcessContext_AcquireGetRelease(t *testing.T) {
	pc := NewProcessContext()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	mp, err := NewManagedProcess(t.Context(), catPath, []string{})
	if err != nil {
		t.Fatalf("NewManagedProcess: %v", err)
	}
	defer mp.Close()

	// Acquire
	id := pc.AcquireManagedProcess(mp)
	if id != 0 {
		t.Fatalf("expected first ID to be 0, got %d", id)
	}

	// Get
	got, ok := pc.GetManagedProcess(id)
	if !ok {
		t.Fatal("expected to find managed process by ID")
	}
	if got != mp {
		t.Fatal("returned managed process doesn't match original")
	}

	// Get nonexistent
	_, ok = pc.GetManagedProcess(999)
	if ok {
		t.Fatal("expected not to find nonexistent managed process")
	}

	// Release
	pc.ReleaseManagedProcess(id)

	_, ok = pc.GetManagedProcess(id)
	if ok {
		t.Fatal("expected process to be removed after release")
	}
}

func TestProcessContext_CloseAllManaged(t *testing.T) {
	pc := NewProcessContext()

	sleepPath, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not found in PATH")
	}

	// Spawn two long-running processes
	mp1, err := NewManagedProcess(t.Context(), sleepPath, []string{"60"})
	if err != nil {
		t.Fatalf("NewManagedProcess mp1: %v", err)
	}
	mp2, err := NewManagedProcess(t.Context(), sleepPath, []string{"60"})
	if err != nil {
		t.Fatalf("NewManagedProcess mp2: %v", err)
	}

	pc.AcquireManagedProcess(mp1)
	pc.AcquireManagedProcess(mp2)

	// CloseAllManaged should kill both
	pc.CloseAllManaged()

	// Verify both exited
	for i, mp := range []*managedProcess{mp1, mp2} {
		select {
		case <-mp.done:
			// Good
		case <-time.After(10 * time.Second):
			t.Fatalf("process %d not killed within 10 seconds after CloseAllManaged()", i)
		}
	}
}

func TestProcessSpawn_AllowlistBlocking(t *testing.T) {
	pc := NewProcessContext()
	pc.HasAllowlist = true
	pc.Allowlist = map[string]string{
		"echo": "/bin/echo",
	}

	// Allowed command should resolve
	path, err := resolveCommand(pc, "echo")
	if err != nil {
		t.Fatalf("resolveCommand(echo): %v", err)
	}
	if path != "/bin/echo" {
		t.Fatalf("expected /bin/echo, got %s", path)
	}

	// Blocked command should fail
	_, err = resolveCommand(pc, "cat")
	if err == nil {
		t.Fatal("expected error for blocked command, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected 'not allowed' error, got: %v", err)
	}
}

func TestProcessSpawn_EffectHandler(t *testing.T) {
	// Test the full ProcessSpawn effect handler
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	pc := NewProcessContext()
	pc.HasAllowlist = true
	pc.Allowlist = map[string]string{
		"cat": catPath,
	}

	ctx := &EffContext{
		Caps:    map[string]Capability{"Process": NewCapability("Process")},
		Process: pc,
	}

	// Spawn via effect handler
	result, err := ProcessSpawn(ctx, []eval.Value{
		&eval.StringValue{Value: "cat"},
		&eval.ListValue{Elements: []eval.Value{}},
	})
	if err != nil {
		t.Fatalf("ProcessSpawn: %v", err)
	}

	// Verify it returned a ProcessHandle ADT
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "ProcessHandle" {
		t.Fatalf("expected ProcessHandle, got %s", tagged.CtorName)
	}

	// Write to it
	writeResult, err := ProcessWriteStdin(ctx, []eval.Value{
		result,
		&eval.BytesValue{Value: []byte("test\n")},
	})
	if err != nil {
		t.Fatalf("ProcessWriteStdin: %v", err)
	}

	// Verify Ok(())
	okTagged, ok := writeResult.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", writeResult)
	}
	if okTagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", okTagged.CtorName)
	}

	// Close stdin
	_, err = ProcessCloseStdin(ctx, []eval.Value{result})
	if err != nil {
		t.Fatalf("ProcessCloseStdin: %v", err)
	}

	// Verify cleanup — handle should be released
	handleID, _ := extractProcessHandleID(result)
	_, found := pc.GetManagedProcess(handleID)
	if found {
		t.Fatal("expected managed process to be released after close")
	}
}
