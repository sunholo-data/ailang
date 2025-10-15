package effects

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestIOPrint_Success(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args := []eval.Value{&eval.StringValue{Value: "Hello"}}
	result, err := Call(ctx, "IO", "print", args)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if output != "Hello" {
		t.Errorf("expected output 'Hello', got %q", output)
	}
}

func TestIOPrintln_Success(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args := []eval.Value{&eval.StringValue{Value: "Hello"}}
	result, err := Call(ctx, "IO", "println", args)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Type() != "unit" {
		t.Errorf("expected unit type, got %s", result.Type())
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if output != "Hello\n" {
		t.Errorf("expected output 'Hello\\n', got %q", output)
	}
}

func TestIOPrint_MissingCapability(t *testing.T) {
	ctx := NewEffContext() // No IO capability granted

	args := []eval.Value{&eval.StringValue{Value: "Hello"}}
	_, err := Call(ctx, "IO", "print", args)

	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}

	if capErr.Effect != "IO" {
		t.Errorf("expected Effect='IO', got %q", capErr.Effect)
	}
}

func TestIOPrint_WrongArgCount(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	// No arguments
	_, err := Call(ctx, "IO", "print", []eval.Value{})
	if err == nil {
		t.Error("expected error for wrong argument count (0 args)")
	}

	// Too many arguments
	args := []eval.Value{
		&eval.StringValue{Value: "Hello"},
		&eval.StringValue{Value: "World"},
	}
	_, err = Call(ctx, "IO", "print", args)
	if err == nil {
		t.Error("expected error for wrong argument count (2 args)")
	}
}

func TestIOPrint_WrongArgType(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := Call(ctx, "IO", "print", args)

	if err == nil {
		t.Fatal("expected error for wrong argument type")
	}

	if !strings.Contains(err.Error(), "expected String") {
		t.Errorf("expected 'expected String' in error, got: %v", err)
	}
}

func TestIOReadLine_Success(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	// Mock stdin
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write test input
	go func() {
		w.Write([]byte("test input\n"))
		w.Close()
	}()

	result, err := Call(ctx, "IO", "readLine", []eval.Value{})

	os.Stdin = old

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}

	if strVal.Value != "test input" {
		t.Errorf("expected 'test input', got %q", strVal.Value)
	}
}

func TestIOReadLine_EOF(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	// Mock stdin with empty input (EOF)
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close() // Immediate EOF

	result, err := Call(ctx, "IO", "readLine", []eval.Value{})

	os.Stdin = old

	if err != nil {
		t.Fatalf("expected no error on EOF, got: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}

	if strVal.Value != "" {
		t.Errorf("expected empty string on EOF, got %q", strVal.Value)
	}
}

func TestIOReadLine_MissingCapability(t *testing.T) {
	ctx := NewEffContext() // No IO capability

	_, err := Call(ctx, "IO", "readLine", []eval.Value{})

	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}

	if capErr.Effect != "IO" {
		t.Errorf("expected Effect='IO', got %q", capErr.Effect)
	}
}

func TestIOReadLine_WrongArgCount(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	args := []eval.Value{&eval.StringValue{Value: "unexpected"}}
	_, err := Call(ctx, "IO", "readLine", args)

	if err == nil {
		t.Fatal("expected error for wrong argument count")
	}

	if !strings.Contains(err.Error(), "expected 0 arguments") {
		t.Errorf("expected 'expected 0 arguments' in error, got: %v", err)
	}
}

func TestCall_UnknownEffect(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Unknown"))

	_, err := Call(ctx, "Unknown", "operation", []eval.Value{})

	if err == nil {
		t.Fatal("expected error for unknown effect")
	}

	if !strings.Contains(err.Error(), "unknown effect: Unknown") {
		t.Errorf("expected 'unknown effect' in error, got: %v", err)
	}
}

func TestCall_UnknownOperation(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	_, err := Call(ctx, "IO", "unknownOp", []eval.Value{})

	if err == nil {
		t.Fatal("expected error for unknown operation")
	}

	if !strings.Contains(err.Error(), "unknown operation unknownOp") {
		t.Errorf("expected 'unknown operation' in error, got: %v", err)
	}
}

func TestRegisterOp(t *testing.T) {
	// Test that operations are registered
	if Registry["IO"] == nil {
		t.Fatal("IO effect not registered")
	}

	ops := []string{"print", "println", "readLine"}
	for _, op := range ops {
		if Registry["IO"][op] == nil {
			t.Errorf("IO.%s not registered", op)
		}
	}
}
