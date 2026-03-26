package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeSafety_StringNotJson verifies that passing a string value where a
// Json ADT is expected produces a compile-time type error.
//
// Background: device_auth.ail passed encode(jo([...])) (a string) to
// setDoc(fields: Json). The type checker must reject this — string and Json
// are distinct types that cannot unify.
func TestTypeSafety_StringNotJson(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-type-safety-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create minimal std/json.ail with Json type and encode function
	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	jsonStdlib := `module std/json
export type Json = JNull | JString(string) | JObject(List[{key: string, value: Json}])

export pure func encode(obj: Json) -> string =
  "encoded"

export pure func jo(kvs: List[{key: string, value: Json}]) -> Json =
  JObject(kvs)

export pure func kv(k: string, v: Json) -> {key: string, value: Json} =
  {key: k, value: v}

export pure func js(s: string) -> Json =
  JString(s)
`
	if err := os.WriteFile(filepath.Join(stdDir, "json.ail"), []byte(jsonStdlib), 0644); err != nil {
		t.Fatalf("failed to write json.ail: %v", err)
	}

	// Test case: passing encode() result (string) where Json is expected
	testContent := `module test
import std/json (Json, encode, jo, kv, js)

pure func acceptJson(x: Json) -> Json = x

export pure func main() -> Json =
  acceptJson(encode(jo([kv("name", js("Alice"))])))
`
	testFile := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test.ail: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "test.ail"}
	cfg := Config{Mode: ModeCheck}

	_, err = Run(cfg, src)
	if err == nil {
		t.Fatal("Expected type error: encode() returns string, but acceptJson expects Json")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "string") {
		t.Errorf("Expected error to mention 'string', got: %s", errMsg)
	}
}

// TestTypeSafety_StringNotJson_Positive verifies the correct usage compiles.
func TestTypeSafety_StringNotJson_Positive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-type-safety-ok-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	jsonStdlib := `module std/json
export type Json = JNull | JString(string) | JObject(List[{key: string, value: Json}])

export pure func jo(kvs: List[{key: string, value: Json}]) -> Json =
  JObject(kvs)

export pure func kv(k: string, v: Json) -> {key: string, value: Json} =
  {key: k, value: v}

export pure func js(s: string) -> Json =
  JString(s)
`
	if err := os.WriteFile(filepath.Join(stdDir, "json.ail"), []byte(jsonStdlib), 0644); err != nil {
		t.Fatalf("failed to write json.ail: %v", err)
	}

	testContent := `module test
import std/json (Json, jo, kv, js)

pure func acceptJson(x: Json) -> Json = x

export pure func main() -> Json =
  acceptJson(jo([kv("name", js("hello"))]))
`
	testFile := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test.ail: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "test.ail"}
	cfg := Config{Mode: ModeCheck}

	_, err = Run(cfg, src)
	if err != nil {
		t.Fatalf("Expected no error for valid Json usage, got: %v", err)
	}
}
