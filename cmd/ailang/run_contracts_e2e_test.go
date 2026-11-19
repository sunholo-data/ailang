package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-V1-VERIFICATION-HONESTY M2 — runtime contracts, end to end.
//
// half(5) returns -95 and violates `ensures { result >= 0 }`. Before M2:
//   - the evaluator raised "ensures failed in  at …" (no function name: the
//     name was in scope where contracts are attached and dropped there);
//   - `--bytecode --verify-contracts` printed -95 and exited 0 (#1956).

const runContractsSrc = `module k1

import std/io (println)

export pure func half(x: int) -> int
  requires { x >= 0 }
  ensures { result >= 0 }
{
  x - 100
}

export func main() -> () ! {IO} = println(show(half(5)))
`

func writeRunContractsFixture(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "k1.ail")
	if err := os.WriteFile(f, []byte(runContractsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestRunContracts_ViolationNamesTheFunction. Red mutation: drop the
// `Function: name` assignment in runtime.attachContracts — this fails.
func TestRunContracts_ViolationNamesTheFunction(t *testing.T) {
	bin := buildAilang(t)
	f := writeRunContractsFixture(t)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--verify-contracts", "--caps", "IO", "--entry", "main", f)
	if code != 1 {
		t.Fatalf("exit = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "contract violation: ensures failed in half at ") {
		t.Fatalf("violation does not name the function:\n%s", stderr)
	}
	if strings.Contains(stdout, "-95") {
		t.Fatalf("the violating value was printed:\n%s", stdout)
	}
}

// TestRunContracts_DefaultIsOff pins the documented default: `run` does not
// check contracts unless --verify-contracts is given (D3, frozen for 1.0).
func TestRunContracts_DefaultIsOff(t *testing.T) {
	bin := buildAilang(t)
	f := writeRunContractsFixture(t)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--caps", "IO", "--entry", "main", f)
	if code != 0 || strings.TrimSpace(stdout) != "-95" {
		t.Fatalf("default run: exit %d, stdout %q (stderr %q); want 0 and -95", code, stdout, stderr)
	}
}

// TestRunContracts_BytecodeIsRefused (#1956): every VM spelling with
// --verify-contracts is refused with CTR004 and exit 2 before the program
// prints anything.
func TestRunContracts_BytecodeIsRefused(t *testing.T) {
	bin := buildAilang(t)
	f := writeRunContractsFixture(t)
	for _, vmFlag := range []string{"--bytecode", "--strict-bytecode"} {
		stdout, stderr, code := runAilangBin(t, bin, "run", vmFlag, "--verify-contracts", "--caps", "IO", "--entry", "main", f)
		if code != 2 {
			t.Fatalf("%s: exit = %d, want 2\nstdout:\n%s\nstderr:\n%s", vmFlag, code, stdout, stderr)
		}
		if !strings.Contains(stderr, "CTR004") || !strings.Contains(stderr, "Drop "+vmFlag) {
			t.Fatalf("%s: refusal does not name CTR004 and the flag to drop:\n%s", vmFlag, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Fatalf("%s: the program ran before the refusal:\n%s", vmFlag, stdout)
		}
	}
}
