package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestFSReadFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	// Create temp file
	tmpfile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	testContent := "Hello from file!"
	if _, err := tmpfile.WriteString(testContent); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test readFile
	args := []eval.Value{&eval.StringValue{Value: tmpfile.Name()}}
	result, err := Call(ctx, "FS", "readFile", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}

	if strVal.Value != testContent {
		t.Errorf("expected content %q, got %q", testContent, strVal.Value)
	}
}

func TestFSReadFile_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{}) // No FS capability

	args := []eval.Value{&eval.StringValue{Value: "/tmp/test.txt"}}
	_, err := Call(ctx, "FS", "readFile", args)

	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}

	if capErr.Effect != "FS" {
		t.Errorf("expected Effect='FS', got %q", capErr.Effect)
	}
}

func TestFSReadFile_NonexistentFile(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/file.txt"}}
	_, err := Call(ctx, "FS", "readFile", args)

	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "readFile") {
		t.Errorf("expected 'readFile' in error, got: %v", err)
	}
}

func TestFSReadFile_WrongArgCount(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	_, err := Call(ctx, "FS", "readFile", []eval.Value{})
	if err == nil {
		t.Error("expected error for wrong argument count (0 args)")
	}

	args := []eval.Value{
		&eval.StringValue{Value: "file1.txt"},
		&eval.StringValue{Value: "file2.txt"},
	}
	_, err = Call(ctx, "FS", "readFile", args)
	if err == nil {
		t.Error("expected error for wrong argument count (2 args)")
	}
}

func TestFSReadFile_WrongArgType(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := Call(ctx, "FS", "readFile", args)

	if err == nil {
		t.Fatal("expected error for wrong argument type")
	}

	if !strings.Contains(err.Error(), "expected String") {
		t.Errorf("expected 'expected String' in error, got: %v", err)
	}
}

func TestFSExists_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	// Create temp file
	tmpfile, err := os.CreateTemp("", "test-exists-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Test existing file
	args := []eval.Value{&eval.StringValue{Value: tmpfile.Name()}}
	result, err := Call(ctx, "FS", "exists", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for existing file")
	}

	// Test nonexistent file
	args = []eval.Value{&eval.StringValue{Value: "/nonexistent/file.txt"}}
	result, err = Call(ctx, "FS", "exists", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	boolVal, ok = result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}

	if boolVal.Value {
		t.Error("expected false for nonexistent file")
	}
}

func TestFSExists_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{}) // No FS capability

	args := []eval.Value{&eval.StringValue{Value: "/tmp/test.txt"}}
	_, err := Call(ctx, "FS", "exists", args)

	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}

	if capErr.Effect != "FS" {
		t.Errorf("expected Effect='FS', got %q", capErr.Effect)
	}
}

func TestFSSandbox_ReadFile(t *testing.T) {
	// Create temp sandbox directory
	sandbox, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	// Create file in sandbox
	testFile := filepath.Join(sandbox, "data.txt")
	testContent := "sandboxed content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create context with sandbox
	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	// Read using relative path (should be joined with sandbox)
	args := []eval.Value{&eval.StringValue{Value: "data.txt"}}
	result, err := Call(ctx, "FS", "readFile", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}

	if strVal.Value != testContent {
		t.Errorf("expected %q, got %q", testContent, strVal.Value)
	}
}

func TestFSSandbox_Exists(t *testing.T) {
	// Create temp sandbox directory
	sandbox, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	// Create file in sandbox
	testFile := filepath.Join(sandbox, "exists-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create context with sandbox
	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	// Check existence using relative path
	args := []eval.Value{&eval.StringValue{Value: "exists-test.txt"}}
	result, err := Call(ctx, "FS", "exists", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for existing sandboxed file")
	}
}

// TestFSSandbox_AbsolutePathWithinSandbox verifies that absolute paths pointing
// inside the sandbox are accepted and resolved correctly (regression for the
// "double-path" bug where filepath.Join(sandbox, absPath) produced
// /sandbox/sandbox/config.json instead of /sandbox/config.json).
func TestFSSandbox_AbsolutePathWithinSandbox(t *testing.T) {
	sandbox, err := os.MkdirTemp("", "sandbox-abspath-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	// Write a file using its absolute path
	absFile := filepath.Join(sandbox, "config.json")
	if err := os.WriteFile(absFile, []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	// exists(absolutePath) must return true, not false
	result, err := Call(ctx, "FS", "exists", []eval.Value{&eval.StringValue{Value: absFile}})
	if err != nil {
		t.Fatalf("exists: unexpected error: %v", err)
	}
	if !result.(*eval.BoolValue).Value {
		t.Error("exists: expected true for absolute path inside sandbox, got false")
	}

	// readFile(absolutePath) must return the content
	result2, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: absFile}})
	if err != nil {
		t.Fatalf("readFile: unexpected error: %v", err)
	}
	if result2.(*eval.StringValue).Value != `{"ok":true}` {
		t.Errorf("readFile: unexpected content: %q", result2.(*eval.StringValue).Value)
	}
}

// TestFSSandbox_AbsolutePathOutsideSandbox verifies that absolute paths
// pointing outside the sandbox are rejected (exists returns false; readFile
// returns an error).
func TestFSSandbox_AbsolutePathOutsideSandbox(t *testing.T) {
	sandbox, err := os.MkdirTemp("", "sandbox-escape-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	outsidePath := "/etc/hostname"

	// exists must return false (silently deny, not error)
	result, err := Call(ctx, "FS", "exists", []eval.Value{&eval.StringValue{Value: outsidePath}})
	if err != nil {
		t.Fatalf("exists: unexpected error for outside path: %v", err)
	}
	if result.(*eval.BoolValue).Value {
		t.Error("exists: expected false for path outside sandbox")
	}

	// readFile must return an error
	_, err = Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: outsidePath}})
	if err == nil {
		t.Error("readFile: expected error for path outside sandbox, got nil")
	}
}
