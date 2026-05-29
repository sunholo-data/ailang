package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestFSMkdir_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpDir := filepath.Join(os.TempDir(), "ailang-test-mkdir")
	defer os.RemoveAll(tmpDir)

	args := []eval.Value{&eval.StringValue{Value: tmpDir}}
	result, err := Call(ctx, "FS", "mkdir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestFSMkdir_ParentMissing(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "/tmp/ailang-nonexistent-parent/child"}}
	_, err := Call(ctx, "FS", "mkdir", args)
	if err == nil {
		t.Fatal("expected error when parent directory doesn't exist")
	}
}

func TestFSMkdirAll_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpDir := filepath.Join(os.TempDir(), "ailang-test-mkdirall", "a", "b", "c")
	defer os.RemoveAll(filepath.Join(os.TempDir(), "ailang-test-mkdirall"))

	args := []eval.Value{&eval.StringValue{Value: tmpDir}}
	result, err := Call(ctx, "FS", "mkdirAll", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("directory tree not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestFSMkdirAll_AlreadyExists(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: os.TempDir()}}
	_, err := Call(ctx, "FS", "mkdirAll", args)
	if err != nil {
		t.Fatalf("mkdirAll on existing dir should not error, got: %v", err)
	}
}

func TestFSMkdir_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{})

	args := []eval.Value{&eval.StringValue{Value: "/tmp/test"}}
	_, err := Call(ctx, "FS", "mkdir", args)
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

func TestFSMkdir_WrongArgType(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := Call(ctx, "FS", "mkdir", args)
	if err == nil {
		t.Fatal("expected error for wrong argument type")
	}
	if !strings.Contains(err.Error(), "expected String") {
		t.Errorf("expected 'expected String' in error, got: %v", err)
	}
}

func TestFSIsDir_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: os.TempDir()}}
	result, err := Call(ctx, "FS", "isDir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true for /tmp directory")
	}
}

func TestFSIsDir_FileReturnsFalse(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile, err := os.CreateTemp("", "test-isdir-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	args := []eval.Value{&eval.StringValue{Value: tmpfile.Name()}}
	result, err := Call(ctx, "FS", "isDir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal := result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for a regular file")
	}
}

func TestFSIsDir_NonexistentReturnsFalse(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/path"}}
	result, err := Call(ctx, "FS", "isDir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal := result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for nonexistent path")
	}
}

func TestFSIsFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile, err := os.CreateTemp("", "test-isfile-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	args := []eval.Value{&eval.StringValue{Value: tmpfile.Name()}}
	result, err := Call(ctx, "FS", "isFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true for a regular file")
	}
}

func TestFSIsFile_DirReturnsFalse(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: os.TempDir()}}
	result, err := Call(ctx, "FS", "isFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal := result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for a directory")
	}
}

func TestFSIsFile_NonexistentReturnsFalse(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/file.txt"}}
	result, err := Call(ctx, "FS", "isFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal := result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for nonexistent path")
	}
}

func TestFSRemoveFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile, err := os.CreateTemp("", "test-remove-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	args := []eval.Value{&eval.StringValue{Value: tmpfile.Name()}}
	result, err := Call(ctx, "FS", "removeFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	if _, err := os.Stat(tmpfile.Name()); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestFSRemoveFile_EmptyDir(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpDir := filepath.Join(os.TempDir(), "ailang-test-removedir")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	args := []eval.Value{&eval.StringValue{Value: tmpDir}}
	_, err := Call(ctx, "FS", "removeFile", args)
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}

	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("expected directory to be deleted")
	}
}

func TestFSRemoveFile_Nonexistent(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/file.txt"}}
	_, err := Call(ctx, "FS", "removeFile", args)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFSRemoveFile_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{})

	args := []eval.Value{&eval.StringValue{Value: "/tmp/test.txt"}}
	_, err := Call(ctx, "FS", "removeFile", args)
	if err == nil {
		t.Fatal("expected error for missing capability")
	}
	_, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}
}

func TestFSSandbox_Mkdir(t *testing.T) {
	sandbox, err := os.MkdirTemp("", "sandbox-mkdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "newdir"}}
	_, err = Call(ctx, "FS", "mkdir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	info, err := os.Stat(filepath.Join(sandbox, "newdir"))
	if err != nil {
		t.Fatalf("directory not created in sandbox: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestFSSandbox_IsDir(t *testing.T) {
	sandbox, err := os.MkdirTemp("", "sandbox-isdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	if err := os.Mkdir(filepath.Join(sandbox, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	args := []eval.Value{&eval.StringValue{Value: "subdir"}}
	result, err := Call(ctx, "FS", "isDir", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	boolVal := result.(*eval.BoolValue)
	if !boolVal.Value {
		t.Error("expected true for sandboxed directory")
	}
}
