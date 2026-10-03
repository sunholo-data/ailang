package pipeline

// M-CTOR-PATTERN-ALIAS-AND-SCOPE (#1478) check-mode regression suite.
// Runtime parity (evaluator vs strict VM) lives in
// cmd/ailang/ctor_alias_parity_test.go; these cases pin the typechecker gate.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ctorScopeDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	stdDir := filepath.Join(tempDir, "std")
	writeMiniStdlib(t, stdDir)
	// std/wrap exposes Option (pick's return type) but uses Result only
	// internally, so Result's constructors are loaded but not in scope for a
	// module that imports just `std/wrap (pick)`.
	wrap := `module std/wrap
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)
pure func verify(n: int) -> Result[int, string] = if n > 0 then Ok(n) else Err("neg")
export pure func pick(n: int) -> Option[int] = match verify(n) { Ok(v) => Some(v), Err(_) => None }
`
	if err := os.WriteFile(filepath.Join(stdDir, "wrap.ail"), []byte(wrap), 0644); err != nil {
		t.Fatal(err)
	}
	return tempDir
}

func TestCtorPattern_UnknownNullaryRejected(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/option (Option, Some, None)
export pure func main() -> int =
  let x: Option[int] = None in
  match x { Bogus => 1, _ => 0 }
`)
	if err == nil {
		t.Fatal("expected TC_MATCH_001 for unknown nullary constructor pattern, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "TC_MATCH_001") || !strings.Contains(msg, "Bogus") {
		t.Errorf("want TC_MATCH_001 naming Bogus, got: %s", msg)
	}
	if !strings.Contains(msg, "None") || !strings.Contains(msg, "Some") {
		t.Errorf("want suggestion listing the scrutinee's constructors, got: %s", msg)
	}
}

func TestCtorPattern_UnknownWithArgsRejected(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/option (Option, Some, None)
export pure func main() -> int =
  let x: Option[int] = Some(1) in
  match x { Bogus(v) => v, _ => 0 }
`)
	if err == nil || !strings.Contains(err.Error(), "TC_MATCH_001") || !strings.Contains(err.Error(), "Bogus") {
		t.Fatalf("want TC_MATCH_001 naming Bogus, got: %v", err)
	}
}

// A typo of a LOCAL constructor is also caught (no imports involved).
func TestCtorPattern_UnknownLocalTypoRejected(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
type Phase = Idle | Running
export pure func main() -> int = match Idle { Idel => 1, _ => 0 }
`)
	if err == nil || !strings.Contains(err.Error(), "TC_MATCH_001") || !strings.Contains(err.Error(), "Idle") {
		t.Fatalf("want TC_MATCH_001 suggesting Idle, got: %v", err)
	}
}

func TestCtorPattern_AliasAccepted(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/option (Option, Some as S, None as Nada)
export pure func main() -> int =
  let x: Option[int] = Nada in
  let y: Option[int] = S(2) in
  match x { Nada => match y { S(v) => v, Nada => 0 }, S(_) => 0 }
`)
	if err != nil {
		t.Fatalf("aliased constructors should type-check, got: %v", err)
	}
}

// Bare non-nullary alias without arguments stays the existing loud error.
func TestCtorPattern_BareNonNullaryAliasRejected(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/option (Option, Some as S, None)
export pure func main() -> int =
  let x: Option[int] = None in
  match x { S => 1, _ => 0 }
`)
	if err == nil || !strings.Contains(err.Error(), "expects 1 argument") {
		t.Fatalf("want arity error for bare S, got: %v", err)
	}
}

// #323 family: constructors known only transitively keep matching by name.
func TestCtorPattern_TransitiveConstructorsAccepted(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/wrap (pick)
export pure func main() -> int = match pick(3) { None => 0, Some(v) => v }
`)
	if err != nil {
		t.Fatalf("transitively-known constructors should type-check, got: %v", err)
	}
}

// A transitively-known constructor from the WRONG ADT is now the same loud
// foreign-constructor error users get for directly-imported constructors.
func TestCtorPattern_TransitiveForeignRejected(t *testing.T) {
	err := runCheck(t, ctorScopeDir(t), `module test
import std/wrap (pick)
export pure func main() -> int = match pick(3) { Err(e) => 0, _ => 1 }
`)
	if err == nil {
		t.Fatal("expected foreign-constructor error for Err arm on Option scrutinee, got nil")
	}
	msg := err.Error()
	t.Logf("err: %s", msg)
	if !strings.Contains(msg, "Err") || !strings.Contains(msg, "Result") || !strings.Contains(msg, "Option") {
		t.Errorf("want foreign-constructor error naming Err, Result and Option, got: %s", msg)
	}
}
