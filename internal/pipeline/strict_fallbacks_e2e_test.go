package pipeline

// M-CHECK-STRICT-FALLBACKS end-to-end suite.
//
// These tests run the REAL module pipeline (pipeline.Run) on fixtures, NOT
// hand-built Core. This is load-bearing: only the module pipeline resolves an
// imported `jo` to App{VarGlobal{std/json.jo}} and ANF-normalizes constructor
// args to enclosing `let` bindings. A synthetic Core tree would not exercise
// either, so a synthetic-only test would be a false green for the two hardest
// cases (Ok([]) via ANF Var, Ok(jo([])) via the registry).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStrictFallbackStdlib writes a minimal std/result + std/json into
// tempDir/std. std/json.jo mirrors the real signature so an imported `jo([])`
// elaborates to App{VarGlobal{std/json.jo}, Args:[[]]} — the registry key.
func writeStrictFallbackStdlib(t *testing.T, stdDir string) {
	t.Helper()
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}
	files := map[string]string{
		"result.ail": `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`,
		"json.ail": `module std/json
export type Json =
  | JNull
  | JObject(List[{key: string, value: Json}])

export func jo(kvs: List[{key: string, value: Json}]) -> Json {
  JObject(kvs)
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(stdDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
}

// runStrictCheck runs the module pipeline against testContent (written to
// test.ail in a temp dir with the mini stdlib) and returns the collected
// warnings as rendered strings plus any hard error.
func runStrictCheck(t *testing.T, testContent string) ([]string, error) {
	t.Helper()
	tempDir := t.TempDir()
	writeStrictFallbackStdlib(t, filepath.Join(tempDir, "std"))

	testFile := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test.ail: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "test.ail"}
	cfg := Config{Mode: ModeCheck}
	result, runErr := Run(cfg, src)

	var warns []string
	for _, w := range result.Warnings {
		warns = append(warns, w.String())
	}
	return warns, runErr
}

// countStrictFallbacks returns how many STRICT_FALLBACK_001 warnings appeared.
func countStrictFallbacks(warns []string) int {
	n := 0
	for _, w := range warns {
		if strings.Contains(w, "STRICT_FALLBACK_001") {
			n++
		}
	}
	return n
}

// TestStrictFallback_E2E_OkEmptyString proves the direct-atomic literal case.
func TestStrictFallback_E2E_OkEmptyString(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)

export func getName(x: string) -> Result[string, string] =
  Ok("")
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 1 {
		t.Fatalf("expected exactly 1 STRICT_FALLBACK_001 for Ok(\"\"), got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_OkEmptyList proves the ANF-Var case: Ok([]) becomes
// `let t = [] in make_Result_Ok(t)`, so the detector MUST resolve Args[0]
// (a Var) back through the enclosing Let. THIS is the case a synthetic tree
// would false-green on.
func TestStrictFallback_E2E_OkEmptyList(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)

export func getItems(x: string) -> Result[[int], string] =
  Ok([])
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 1 {
		t.Fatalf("expected exactly 1 STRICT_FALLBACK_001 for Ok([]), got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_OkJoEmpty proves the MOTIVATING case: Ok(jo([]))
// becomes `let t = jo([]) in make_Result_Ok(t)` where jo is
// App{VarGlobal{std/json.jo}}. Requires BOTH the ANF let-resolver AND the
// module-qualified registry. Synthetic Core lacks the VarGlobal — only a real
// module-pipeline run exercises it.
func TestStrictFallback_E2E_OkJoEmpty(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)
import std/json (Json, jo)

export func getDoc(x: string) -> Result[Json, string] =
  Ok(jo([]))
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 1 {
		t.Fatalf("expected exactly 1 STRICT_FALLBACK_001 for Ok(jo([])), got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_OkAllZeroRecord proves the Pattern B all-zero record.
func TestStrictFallback_E2E_OkAllZeroRecord(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)

export func getUser(x: string) -> Result[{name: string, age: int}, string] =
  Ok({name: "", age: 0})
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 1 {
		t.Fatalf("expected exactly 1 STRICT_FALLBACK_001 for Ok({name:\"\",age:0}), got %d\nwarnings: %v", got, warns)
	}
}

// --- negatives --------------------------------------------------------------

// TestStrictFallback_E2E_OkRealValueNoFlag: Ok("real") must NOT flag.
func TestStrictFallback_E2E_OkRealValueNoFlag(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)

export func getName(x: string) -> Result[string, string] =
  Ok("real")
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 0 {
		t.Fatalf("expected NO STRICT_FALLBACK_001 for Ok(\"real\"), got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_UserLocalJoNoFlag: a user-defined LOCAL `jo` is a
// plain core.Var head, never the std/json VarGlobal, so Ok(jo([])) must NOT
// flag. This is the soundness case (gpt5-6-sol objection).
func TestStrictFallback_E2E_UserLocalJoNoFlag(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)

func jo(xs: [int]) -> [int] = xs

export func getItems(x: string) -> Result[[int], string] =
  Ok(jo([]))
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	// A user-local jo([]) resolves to a call whose result is not a literal
	// empty collection we can see structurally, and the head is not the
	// std/json VarGlobal — so it must NOT flag.
	if got := countStrictFallbacks(warns); got != 0 {
		t.Fatalf("expected NO STRICT_FALLBACK_001 for user-local jo([]), got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_NonResultNoFlag: Ok(...) patterns in a
// non-Result-returning function must NOT flag (return-type filter). We use a
// helper that returns Json directly.
func TestStrictFallback_E2E_NonResultNoFlag(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)
import std/json (Json, jo)

-- Returns Json, not Result — the return-type filter must skip it even though
-- jo([]) appears in its body.
export func emptyDoc(x: string) -> Json =
  jo([])
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 0 {
		t.Fatalf("expected NO STRICT_FALLBACK_001 in a non-Result function, got %d\nwarnings: %v", got, warns)
	}
}

// TestStrictFallback_E2E_AnnotationSuppresses: @allow_empty_ok("...") silences.
func TestStrictFallback_E2E_AnnotationSuppresses(t *testing.T) {
	src := `module test
import std/result (Result, Ok, Err)
import std/json (Json, jo)

@allow_empty_ok("missing key legitimately means an empty collection")
export func listDocs(x: string) -> Result[Json, string] =
  Ok(jo([]))
`
	warns, err := runStrictCheck(t, src)
	if err != nil {
		t.Fatalf("unexpected pipeline error: %v", err)
	}
	if got := countStrictFallbacks(warns); got != 0 {
		t.Fatalf("expected @allow_empty_ok to suppress, got %d\nwarnings: %v", got, warns)
	}
}
