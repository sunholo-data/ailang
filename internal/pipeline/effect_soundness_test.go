package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

// compileAndValidateEffects parses, elaborates, type-checks, and runs effect
// validation on the given AILANG code. Returns nil if everything passes, or
// the first error encountered.
func compileAndValidateEffects(t *testing.T, code string) error {
	t.Helper()

	l := lexer.New(code, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(program)
	if err != nil {
		return err
	}

	tc := types.NewCoreTypeChecker()
	_, err = tc.CheckCoreProgram(coreProg)
	if err != nil {
		return err
	}

	return ValidateEffects(program.File, coreProg, tc.CoreTI)
}

// TestEffectSoundness_FS_Does_Not_Absorb_Env is an integration-level regression test
// ensuring that declaring ! {FS} on a function does NOT satisfy an Env requirement.
//
// Background: A local gcp_auth.ail declared ! {FS} while using getEnvOr (which
// requires Env). The effect system should treat each effect as independent —
// FS and Env are distinct, and both must be explicitly declared.
func TestEffectSoundness_FS_Does_Not_Absorb_Env(t *testing.T) {
	err := compileAndValidateEffects(t, `
module test/effect_soundness

export func getHome() -> String ! {FS} =
  getEnvOr("HOME", "/tmp")
`)
	if err == nil {
		t.Fatal("Expected effect checking error: function declares ! {FS} but uses Env effect (getEnvOr). Compilation should fail.")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Env") && !strings.Contains(errMsg, "getEnvOr") {
		t.Errorf("Expected error mentioning Env or getEnvOr, got: %s", errMsg)
	}
}

// TestEffectSoundness_Both_FS_And_Env_Required verifies that a function using
// both FS and Env operations must declare both effects — FS alone is not enough.
func TestEffectSoundness_Both_FS_And_Env_Required(t *testing.T) {
	err := compileAndValidateEffects(t, `
module test/effect_both

export func readConfig() -> String ! {FS} =
  let home = getEnvOr("HOME", "/tmp")
  readFile(home ++ "/config.txt")
`)
	if err == nil {
		t.Fatal("Expected effect checking error: function declares only ! {FS} but also uses Env effect")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Env") && !strings.Contains(errMsg, "getEnvOr") {
		t.Errorf("Expected error mentioning Env or getEnvOr, got: %s", errMsg)
	}
}

// TestEffectSoundness_IO_Does_Not_Absorb_Env verifies IO and Env are independent.
func TestEffectSoundness_IO_Does_Not_Absorb_Env(t *testing.T) {
	err := compileAndValidateEffects(t, `
module test/io_env

export func getHome() -> String ! {IO} =
  getEnvOr("HOME", "/tmp")
`)
	if err == nil {
		t.Fatal("Expected error: ! {IO} should not satisfy Env requirement")
	}
}

// TestEffectSoundness_Env_Does_Not_Absorb_FS verifies Env does not cover FS.
func TestEffectSoundness_Env_Does_Not_Absorb_FS(t *testing.T) {
	err := compileAndValidateEffects(t, `
module test/env_fs

export func readIt() -> String ! {Env} =
  readFile("/etc/hostname")
`)
	if err == nil {
		t.Fatal("Expected error: ! {Env} should not satisfy FS requirement")
	}
}

// TestEffectSoundness_Pure_Rejects_IO verifies that a pure function
// (no effect annotation) cannot call any effectful builtin.
func TestEffectSoundness_Pure_Rejects_IO(t *testing.T) {
	err := compileAndValidateEffects(t, `
module test/pure_io

export func greet() -> () =
  println("hello")
`)
	if err == nil {
		t.Fatal("Expected error: pure function should not be able to call println (IO)")
	}
}
