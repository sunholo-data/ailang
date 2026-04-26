package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// emptyStdin is the stdin we hand the resolver in tests where no stdin read
// should happen. ReadAll on an empty bytes.Reader returns immediately.
func emptyStdin() io.Reader { return strings.NewReader("") }

func TestResolveArgsJSON_DefaultPassthrough(t *testing.T) {
	got, err := resolveArgsJSON("null", "", emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "null" {
		t.Errorf("expected %q, got %q", "null", got)
	}
}

func TestResolveArgsJSON_InlineJSONPassthrough(t *testing.T) {
	in := `["robot"]`
	got, err := resolveArgsJSON(in, "", emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != in {
		t.Errorf("expected %q, got %q", in, got)
	}
}

func TestResolveArgsJSON_StdinSentinel(t *testing.T) {
	stdin := strings.NewReader(`{"name":"Alice"}` + "\n")
	got, err := resolveArgsJSON("-", "", stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"name":"Alice"}` {
		t.Errorf("expected trimmed JSON, got %q", got)
	}
}

func TestResolveArgsJSON_StdinEmptyIsError(t *testing.T) {
	_, err := resolveArgsJSON("-", "", emptyStdin())
	if err == nil {
		t.Fatal("expected error for empty stdin, got nil")
	}
	if !strings.Contains(err.Error(), "empty stdin") {
		t.Errorf("expected 'empty stdin' in error, got %q", err.Error())
	}
}

func TestResolveArgsJSON_StdinWhitespaceOnlyIsError(t *testing.T) {
	_, err := resolveArgsJSON("-", "", strings.NewReader("\n  \r\n  "))
	if err == nil {
		t.Fatal("expected error for whitespace-only stdin, got nil")
	}
}

func TestResolveArgsJSON_FileBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`["robot"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveArgsJSON("null", path, emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `["robot"]` {
		t.Errorf("expected %q, got %q", `["robot"]`, got)
	}
}

func TestResolveArgsJSON_FileWithTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`{"x":1}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveArgsJSON("null", path, emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"x":1}` {
		t.Errorf("expected newline trimmed, got %q", got)
	}
}

func TestResolveArgsJSON_FileWithBOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	// Windows PowerShell Set-Content commonly prepends UTF-8 BOM.
	bom := "\ufeff"
	if err := os.WriteFile(path, []byte(bom+`{"x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveArgsJSON("null", path, emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"x":1}` {
		t.Errorf("expected BOM stripped, got %q", got)
	}
}

func TestResolveArgsJSON_FileNotFound(t *testing.T) {
	_, err := resolveArgsJSON("null", "/no/such/path/args.json", emptyStdin())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "-args-file") {
		t.Errorf("expected error to name -args-file, got %q", err.Error())
	}
	// Wrapped os.PathError should be unwrappable.
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected wrapped *os.PathError, got %T: %v", err, err)
	}
}

func TestResolveArgsJSON_FileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveArgsJSON("null", path, emptyStdin())
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if !strings.Contains(err.Error(), "empty -args-file") {
		t.Errorf("expected 'empty -args-file' in error, got %q", err.Error())
	}
}

func TestResolveArgsJSON_FileWhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ws.json")
	if err := os.WriteFile(path, []byte("\n  \t\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveArgsJSON("null", path, emptyStdin())
	if err == nil {
		t.Fatal("expected error for whitespace-only file, got nil")
	}
}

func TestResolveArgsJSON_MutualExclusivity_InlineAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`[1]`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveArgsJSON(`[2]`, path, emptyStdin())
	if err == nil {
		t.Fatal("expected mutual-exclusivity error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "-args-json") || !strings.Contains(msg, "-args-file") {
		t.Errorf("expected error to name both flags, got %q", msg)
	}
}

func TestResolveArgsJSON_MutualExclusivity_StdinAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`[1]`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveArgsJSON("-", path, strings.NewReader(`[2]`))
	if err == nil {
		t.Fatal("expected mutual-exclusivity error, got nil")
	}
}

func TestResolveArgsJSON_DefaultPlusFile(t *testing.T) {
	// Default -args-json + -args-file must NOT trigger mutual-exclusivity.
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	if err := os.WriteFile(path, []byte(`true`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveArgsJSON("null", path, emptyStdin())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "true" {
		t.Errorf("expected file value, got %q", got)
	}
}
