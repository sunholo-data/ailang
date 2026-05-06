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

func TestFSWriteFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
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
	ctx := NewEffContext([]string{}) // No FS capability

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
	ctx := NewEffContext([]string{})
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
	ctx := NewEffContext([]string{})
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

func TestFSWriteFileBytes_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile := filepath.Join(os.TempDir(), "ailang-test-write-bytes.bin")
	defer os.Remove(tmpfile)

	testData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0xFF, 0xFE} // "Hello" + null + binary
	args := []eval.Value{
		&eval.StringValue{Value: tmpfile},
		&eval.BytesValue{Value: testData},
	}

	result, err := Call(ctx, "FS", "writeFileBytes", args)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	// Verify file was written with exact bytes
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if len(content) != len(testData) {
		t.Fatalf("expected %d bytes, got %d", len(testData), len(content))
	}

	for i, b := range content {
		if b != testData[i] {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, testData[i], b)
		}
	}
}

func TestFSWriteFileBytes_WrongArgType(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	// String instead of bytes for data arg
	args := []eval.Value{
		&eval.StringValue{Value: "/tmp/test.bin"},
		&eval.StringValue{Value: "not bytes"},
	}
	_, err := Call(ctx, "FS", "writeFileBytes", args)
	if err == nil {
		t.Fatal("expected error for wrong data type")
	}
	if !strings.Contains(err.Error(), "expected Bytes") {
		t.Errorf("expected 'expected Bytes' in error, got: %v", err)
	}
}

func TestFSAppendFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile := filepath.Join(os.TempDir(), "ailang-test-append.txt")
	defer os.Remove(tmpfile)

	// First append creates file
	args := []eval.Value{
		&eval.StringValue{Value: tmpfile},
		&eval.StringValue{Value: "line1\n"},
	}
	result, err := Call(ctx, "FS", "appendFile", args)
	if err != nil {
		t.Fatalf("first append: expected no error, got: %v", err)
	}
	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	// Second append adds to file
	args[1] = &eval.StringValue{Value: "line2\n"}
	_, err = Call(ctx, "FS", "appendFile", args)
	if err != nil {
		t.Fatalf("second append: expected no error, got: %v", err)
	}

	// Verify both lines present
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expected := "line1\nline2\n"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}

func TestFSAppendFileBytes_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	tmpfile := filepath.Join(os.TempDir(), "ailang-test-append-bytes.bin")
	defer os.Remove(tmpfile)

	frame1 := []byte{0x01, 0x02, 0x03, 0x04}
	frame2 := []byte{0x05, 0x06, 0x07, 0x08}

	// Append first frame
	args := []eval.Value{
		&eval.StringValue{Value: tmpfile},
		&eval.BytesValue{Value: frame1},
	}
	_, err := Call(ctx, "FS", "appendFileBytes", args)
	if err != nil {
		t.Fatalf("first append: expected no error, got: %v", err)
	}

	// Append second frame
	args[1] = &eval.BytesValue{Value: frame2}
	_, err = Call(ctx, "FS", "appendFileBytes", args)
	if err != nil {
		t.Fatalf("second append: expected no error, got: %v", err)
	}

	// Verify concatenated bytes
	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expected := append(frame1, frame2...)
	if len(content) != len(expected) {
		t.Fatalf("expected %d bytes, got %d", len(expected), len(content))
	}
	for i, b := range content {
		if b != expected[i] {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, expected[i], b)
		}
	}
}

func TestFSAppendFileBytes_WrongArgType(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	// String instead of bytes for data arg
	args := []eval.Value{
		&eval.StringValue{Value: "/tmp/test.bin"},
		&eval.StringValue{Value: "not bytes"},
	}
	_, err := Call(ctx, "FS", "appendFileBytes", args)
	if err == nil {
		t.Fatal("expected error for wrong data type")
	}
	if !strings.Contains(err.Error(), "expected Bytes") {
		t.Errorf("expected 'expected Bytes' in error, got: %v", err)
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

func TestFSSandbox_WriteFile(t *testing.T) {
	// Create temp sandbox directory
	sandbox, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	// Create context with sandbox
	ctx := NewEffContext([]string{})
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

// ============================================================================
// mkdir / mkdirAll tests
// ============================================================================

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

	// Parent doesn't exist — should fail
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

	// /tmp always exists — mkdirAll should be a no-op
	args := []eval.Value{&eval.StringValue{Value: os.TempDir()}}
	_, err := Call(ctx, "FS", "mkdirAll", args)
	if err != nil {
		t.Fatalf("mkdirAll on existing dir should not error, got: %v", err)
	}
}

func TestFSMkdir_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{}) // No FS capability

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

// ============================================================================
// isDir / isFile tests
// ============================================================================

func TestFSIsDir_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	// /tmp is a directory
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

// ============================================================================
// removeFile tests
// ============================================================================

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

	// Verify file is gone
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
	ctx := NewEffContext([]string{}) // No FS capability

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

// ============================================================================
// Sandbox tests for new builtins
// ============================================================================

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

	// Create subdir in sandbox
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
