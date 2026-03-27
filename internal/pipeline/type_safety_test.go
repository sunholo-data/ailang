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

// TestTypeSafety_DecodeJsonNotString verifies that passing a Json value to
// decode() (which expects string) produces a compile-time type error.
//
// Background: device_auth.ail called decode(docJson) where docJson was Json
// from getDoc(). decode expects string, not Json. The type checker must reject
// this — it was silently accepted, causing runtime garbage.
func TestTypeSafety_DecodeJsonNotString(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-decode-type-safety-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	resultContent := `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`
	if err := os.WriteFile(filepath.Join(stdDir, "result.ail"), []byte(resultContent), 0644); err != nil {
		t.Fatalf("failed to write result.ail: %v", err)
	}

	jsonStdlib := `module std/json
import std/result (Result, Ok, Err)

export type Json = JNull | JString(string) | JObject(List[{key: string, value: Json}])

export pure func decode(s: string) -> Result[Json, string] =
  Ok(JNull)

export pure func jo(kvs: List[{key: string, value: Json}]) -> Json =
  JObject(kvs)
`
	if err := os.WriteFile(filepath.Join(stdDir, "json.ail"), []byte(jsonStdlib), 0644); err != nil {
		t.Fatalf("failed to write json.ail: %v", err)
	}

	// Test case: passing Json to decode() which expects string
	testContent := `module test
import std/json (Json, decode, jo)
import std/result (Result)

export pure func main() -> Result[Json, string] =
  decode(jo([]))
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
		t.Fatal("Expected type error: decode() expects string, but was passed Json")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "string") && !strings.Contains(errMsg, "Json") {
		t.Errorf("Expected error to mention 'string' or 'Json', got: %s", errMsg)
	}
}

// TestTypeSafety_CrossModule_JsonToString verifies that passing Json to an
// imported function expecting string is caught across module boundaries.
func TestTypeSafety_CrossModule_JsonToString(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-json-to-string-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	// Helper with a simple string->string function
	helperContent := `module std/helper
export pure func needsString(s: string) -> string = s
`
	if err := os.WriteFile(filepath.Join(stdDir, "helper.ail"), []byte(helperContent), 0644); err != nil {
		t.Fatalf("failed to write helper.ail: %v", err)
	}

	resultContent := `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`
	if err := os.WriteFile(filepath.Join(stdDir, "result.ail"), []byte(resultContent), 0644); err != nil {
		t.Fatalf("failed to write result.ail: %v", err)
	}

	jsonContent := `module std/json
import std/result (Result, Ok, Err)
export type Json = JNull | JString(string) | JObject(List[{key: string, value: Json}])

export pure func jo(kvs: List[{key: string, value: Json}]) -> Json =
  JObject(kvs)
`
	if err := os.WriteFile(filepath.Join(stdDir, "json.ail"), []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write json.ail: %v", err)
	}

	// Test: pass Json (from jo) to imported needsString(string)
	testContent := `module test
import std/helper (needsString)
import std/json (jo)

export pure func main() -> string =
  needsString(jo([]))
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
		t.Fatal("Expected type error: needsString expects string, but was passed Json from jo()")
	}
	t.Logf("Json-to-string type mismatch caught: %s", err)
}

// TestTypeSafety_CrossModule_StringParam verifies that calling an imported function
// expecting string with a non-string type is caught.
func TestTypeSafety_CrossModule_StringParam(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-cross-module-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	// Minimal helper module with a string-accepting function
	helperContent := `module std/helper
export pure func needsString(s: string) -> string = s
`
	if err := os.WriteFile(filepath.Join(stdDir, "helper.ail"), []byte(helperContent), 0644); err != nil {
		t.Fatalf("failed to write helper.ail: %v", err)
	}

	// Test: pass an int to an imported function expecting string
	testContent := `module test
import std/helper (needsString)

export pure func main() -> string =
  needsString(42)
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
		t.Fatal("Expected type error: imported needsString expects string, but was passed int")
	}
	t.Logf("Cross-module type mismatch correctly caught: %s", err)
}

// TestTypeSafety_WithinModule_ImportedADTChain verifies that a local function
// wrapping an imported ADT constructor has its return type preserved.
// This was the wider manifestation of M-TYPEENV-SUB: even within a single module,
// functions returning imported ADTs had their return types erased to type variables.
func TestTypeSafety_WithinModule_ImportedADTChain(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-within-module-adt-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	resultContent := `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`
	if err := os.WriteFile(filepath.Join(stdDir, "result.ail"), []byte(resultContent), 0644); err != nil {
		t.Fatalf("failed to write result.ail: %v", err)
	}

	// Local func wrapping imported ADT, then used with wrong type
	testContent := `module test
import std/result (Result, Ok, Err)

pure func wrap() -> Result[int, string] = Ok(42)

export pure func main() -> string = wrap()
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
		t.Fatal("Expected type error: wrap() returns Result[int, string], but main expects string")
	}
	t.Logf("Within-module ADT chain type mismatch caught: %s", err)
}

// TestTypeSafety_WithinModule_ImportedADTChain_Positive verifies that correct
// usage of imported ADTs across local function chains still compiles.
func TestTypeSafety_WithinModule_ImportedADTChain_Positive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-within-module-adt-ok-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	resultContent := `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`
	if err := os.WriteFile(filepath.Join(stdDir, "result.ail"), []byte(resultContent), 0644); err != nil {
		t.Fatalf("failed to write result.ail: %v", err)
	}

	// Correct usage: wrap returns Result, main also returns Result
	testContent := `module test
import std/result (Result, Ok, Err)

pure func wrap() -> Result[int, string] = Ok(42)

export pure func main() -> Result[int, string] = wrap()
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
		t.Fatalf("Expected no error for valid Result chain, got: %v", err)
	}
}
