package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestFSWriteFile_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

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

	content, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}
}

func TestFSWriteFile_MissingCapability(t *testing.T) {
	ctx := NewEffContext([]string{})

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

	_, err := Call(ctx, "FS", "writeFile", []eval.Value{})
	if err == nil {
		t.Error("expected error for wrong argument count (0 args)")
	}

	args := []eval.Value{&eval.StringValue{Value: "file.txt"}}
	_, err = Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Error("expected error for wrong argument count (1 arg)")
	}

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

	args := []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.StringValue{Value: "content"},
	}
	_, err := Call(ctx, "FS", "writeFile", args)
	if err == nil {
		t.Fatal("expected error for wrong path type")
	}

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

	testData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0xFF, 0xFE}
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

	args[1] = &eval.StringValue{Value: "line2\n"}
	_, err = Call(ctx, "FS", "appendFile", args)
	if err != nil {
		t.Fatalf("second append: expected no error, got: %v", err)
	}

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

	args := []eval.Value{
		&eval.StringValue{Value: tmpfile},
		&eval.BytesValue{Value: frame1},
	}
	_, err := Call(ctx, "FS", "appendFileBytes", args)
	if err != nil {
		t.Fatalf("first append: expected no error, got: %v", err)
	}

	args[1] = &eval.BytesValue{Value: frame2}
	_, err = Call(ctx, "FS", "appendFileBytes", args)
	if err != nil {
		t.Fatalf("second append: expected no error, got: %v", err)
	}

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

func TestFSSandbox_WriteFile(t *testing.T) {
	sandbox, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandbox)

	ctx := NewEffContext([]string{})
	ctx.Env.Sandbox = sandbox
	ctx.Grant(NewCapability("FS"))

	testContent := "sandboxed write"
	args := []eval.Value{
		&eval.StringValue{Value: "output.txt"},
		&eval.StringValue{Value: testContent},
	}

	_, err = Call(ctx, "FS", "writeFile", args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(sandbox, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read sandboxed file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(content))
	}
}
