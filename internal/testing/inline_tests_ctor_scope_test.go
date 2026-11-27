package testing

// INLINE-TESTS-SCOPE: the expected side of an inline `tests [...]` row is
// evaluated in the module's scope, so it sees the implicit-prelude
// constructors (Some/None, Ok/Err) and every constructor the module imports,
// under the name it imports it as — not only the module's own ADTs.

import (
	"os"
	"path/filepath"
	"testing"
)

// TST-RC1-08
func TestInlineTestsSeePreludeConstructors(t *testing.T) {
	source := `module ctor_prelude

export pure func safeDiv(a: int, b: int) -> Option[int]
  tests [
    ((10, 2), Some(5)),
    ((10, 0), None)
  ]
{
  if b == 0 then None else Some(a / b)
}

export pure func checked(n: int) -> Result[int, string]
  tests [
    (3, Ok(3)),
    (-1, Err("negative"))
  ]
{
  if n < 0 then Err("negative") else Ok(n)
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ctor_prelude.ail"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	result := runTestsOnFile(t, "ctor_prelude.ail", source)
	if result.FailedTests > 0 {
		t.Fatalf("expected 0 failures, got %d; first error: %s", result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 4 {
		t.Fatalf("expected 4 passing inline test rows, got %d", result.PassedTests)
	}
}

// TST-RC1-08
func TestInlineTestsSeeImportedAndAliasedConstructors(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"ailang.toml": "[package]\nname = \"x/pkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n",
		"shapes.ail":  "module x/pkg/shapes\n\nexport type Shape = Circle(int) | Dot\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	testSrc := `module x/pkg/t_test

import x/pkg/shapes (Shape, Circle as Round, Dot)

export pure func mk(n: int) -> Shape
  tests [
    (0, Dot),
    (2, Round(2))
  ]
{
  if n == 0 then Dot else Round(n)
}
`
	path := filepath.Join(dir, "t_test.ail")
	if err := os.WriteFile(path, []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runTestsOnFile(t, path, testSrc)
	if result.FailedTests > 0 {
		t.Fatalf("expected 0 failures, got %d; first error: %s", result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 2 {
		t.Fatalf("expected 2 passing inline test rows, got %d", result.PassedTests)
	}
}

// TST-RC1-08
func TestInlineTestsLocalConstructorShadowsPrelude(t *testing.T) {
	source := `module ctor_shadow

type Maybe = Some(int) | None

export pure func wrap(n: int) -> Maybe
  tests [
    (0, None),
    (4, Some(4))
  ]
{
  if n == 0 then None else Some(n)
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ctor_shadow.ail"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	result := runTestsOnFile(t, "ctor_shadow.ail", source)
	if result.FailedTests > 0 {
		t.Fatalf("expected 0 failures, got %d; first error: %s", result.FailedTests, firstFailureError(result))
	}
}
