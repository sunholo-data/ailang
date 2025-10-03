package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestFSReadFile_Success(t *testing.T) {
	ctx := NewEffContext()
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
	ctx := NewEffContext() // No FS capability

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
	ctx := NewEffContext()
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
	ctx := NewEffContext()
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
	ctx := NewEffContext()
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

func TestFSWriteFile_Success(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("FS"))

	// Create temp file path (use simple name, no wildcards which break on Windows)
	tmpfile := filepath.Join(os.TempDir(), "ailang-test-write.txt")
	defer os.Remove(tmpfile)

	testContent := "Test content"
	args := []eval.Value{
		&eval.StringValue{Value: tmpfile},
		&eval.StringValue{Value: testContent},
	}

	result, err := Call(ctx, "FS", "writeFile", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	// Verify file was written
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}
}

func TestFSWriteFile_MissingCapability(t *testing.T) {
	ctx := NewEffContext() // No FS capability

	args := []eval.Value{
		&eval.StringValue{Value: "/tmp/test.txt"},
		&eval.StringValue{Value: "content"},
	}
	_, err := Call(ctx, "FS", "writeFile", args)

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

func TestFSWriteFile_WrongArgCount(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("FS"))

	// 0 args
	_, err := Call(ctx, "FS", "writeFile", []eval.Value{})
	if err == nil {
		t.Error("expected error for wrong argument count (0 args)")
	}

	// 1 arg
	args := []eval.Value{&eval.StringValue{Value: "file.txt"}}
	_, err = Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Error("expected error for wrong argument count (1 arg)")
	}

	// 3 args
	args = []eval.Value{
		&eval.StringValue{Value: "file.txt"},
		&eval.StringValue{Value: "content"},
		&eval.StringValue{Value: "extra"},
	}
	_, err = Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Error("expected error for wrong argument count (3 args)")
	}
}

func TestFSWriteFile_WrongArgType(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("FS"))

	// Wrong path type
	args := []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.StringValue{Value: "content"},
	}
	_, err := Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Fatal("expected error for wrong path type")
	}

	// Wrong content type
	args = []eval.Value{
		&eval.StringValue{Value: "file.txt"},
		&eval.IntValue{Value: 42},
	}
	_, err = Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Fatal("expected error for wrong content type")
	}
}

func TestFSExists_Success(t *testing.T) {
	ctx := NewEffContext()
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
	ctx := NewEffContext() // No FS capability

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
	ctx := NewEffContext()
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

func TestFSSandbox_WriteFile(t *testing.T) {
	// Create temp sandbox directory
	sandbox, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	// Create context with sandbox
	ctx := NewEffContext()
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	// Write using relative path
	testContent := "sandboxed write"
	args := []eval.Value{
		&eval.StringValue{Value: "output.txt"},
		&eval.StringValue{Value: testContent},
	}

	_, err = Call(ctx, "FS", "writeFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was written to sandbox
	content, err := os.ReadFile(filepath.Join(sandbox, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read sandboxed file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(content))
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
	ctx := NewEffContext()
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
