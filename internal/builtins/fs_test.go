package builtins

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// _fs_readFileBytes tests
// ============================================================================

func TestFSReadFileBytes_BinaryContent(t *testing.T) {
	dir := t.TempDir()
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	path := filepath.Join(dir, "image.png")
	if err := os.WriteFile(path, binaryData, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: path}}

	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}

	result, err := impl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}

	// Verify base64 encoding
	expected := base64.StdEncoding.EncodeToString(binaryData)
	if sv.Value != expected {
		t.Fatalf("expected base64 %q, got %q", expected, sv.Value)
	}

	// Verify roundtrip
	decoded, err := base64.StdEncoding.DecodeString(sv.Value)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if string(decoded) != string(binaryData) {
		t.Fatalf("roundtrip failed: got %v, want %v", decoded, binaryData)
	}
}

func TestFSReadFileBytes_TextContent(t *testing.T) {
	dir := t.TempDir()
	textData := []byte("Hello, World!")
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, textData, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: path}}

	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}

	result, err := impl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}

	expected := base64.StdEncoding.EncodeToString(textData)
	if sv.Value != expected {
		t.Fatalf("expected base64 %q, got %q", expected, sv.Value)
	}
}

func TestFSReadFileBytes_MissingFile(t *testing.T) {
	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: "/nonexistent/file.bin"}}

	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}

	result, err := impl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertErr(t, result, "cannot read file")
}

func TestFSReadFileBytes_Sandbox(t *testing.T) {
	dir := t.TempDir()
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), data, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := makeTestCtx(t)
	ctx.Env.Sandbox = dir

	// Use relative path — sandbox should resolve it
	args := []eval.Value{&eval.StringValue{Value: "data.bin"}}

	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}

	result, err := impl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}

	expected := base64.StdEncoding.EncodeToString(data)
	if sv.Value != expected {
		t.Fatalf("expected base64 %q, got %q", expected, sv.Value)
	}
}

func TestFSReadFileBytes_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ctx := makeTestCtx(t)
	args := []eval.Value{&eval.StringValue{Value: path}}

	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "readFileBytes", args)
	}

	result, err := impl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	sv, ok := inner.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", inner)
	}

	// Empty file should return empty base64 string
	if sv.Value != "" {
		t.Fatalf("expected empty string for empty file, got %q", sv.Value)
	}
}
