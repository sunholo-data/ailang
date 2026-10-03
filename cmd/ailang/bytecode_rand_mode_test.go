package main

// #1545 (security): `ailang run --bytecode` must honour the dynamic frame
// modes the evaluator enforces — Rand[mode=seeded|crypto] and @limit/@min
// budgets. Before the fix the VM ran moded functions with the mode dropped:
// seeded and crypto draws silently came from the os (math/rand) source and
// per-function budgets were not enforced.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

const seededProbeSrc = `module modeprobe
import std/rand (rand_int)
import std/io (println)

func draw() -> int ! {Rand} = rand_int(0, 1000000)

func seededOuter() -> int ! {Rand[mode=seeded]} = draw()

export func main() -> unit ! {IO, Rand[mode=seeded]} {
  println("start")
  println(show(seededOuter()))
}
`

const cryptoProbeSrc = `module cryptoprobe
import std/rand (rand_int)
import std/io (println)

func draw() -> int ! {Rand} = rand_int(0, 1000000)

func cryptoOuter() -> int ! {Rand[mode=crypto]} = draw()

export func main() -> unit ! {IO, Rand[mode=crypto]} {
  println(show(draw() >= 0))
  println(show(cryptoOuter() >= 0))
}
`

func writeProbe(t *testing.T, name, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runWithSeed runs the binary from the project root with AILANG_SEED removed
// from the environment, or set to seed when non-empty.
func runWithSeed(t *testing.T, bin, seed string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := testutil.HangGuardContext(t, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	cmd.Env = removeEnv(append([]string{}, os.Environ()...), "AILANG_SEED")
	if seed != "" {
		cmd.Env = append(cmd.Env, "AILANG_SEED="+seed)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	runErr := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		return out.String(), errb.String(), 0
	case errors.As(runErr, &exitErr):
		return out.String(), errb.String(), exitErr.ExitCode()
	}
	t.Fatalf("run %v: %v", args, runErr)
	return "", "", -1
}

func TestBytecodeRandMode_SeededHonoured(t *testing.T) {
	bin := buildAilang(t)
	probe := writeProbe(t, "modeprobe.ail", seededProbeSrc)
	base := []string{"run", "--caps", "IO,Rand", "--entry", "main", "--quiet"}

	for _, engine := range [][]string{nil, {"--bytecode"}} {
		args := append(append(append([]string{}, base...), engine...), probe)
		out, errOut, code := runWithSeed(t, bin, "", args...)
		if code == 0 || !strings.Contains(errOut, "RAND_SEEDED_NO_SEED") {
			t.Errorf("%v without a seed: want RAND_SEEDED_NO_SEED, got exit %d\nstdout:\n%s\nstderr:\n%s", engine, code, out, errOut)
		}
		// The entry runs exactly once: a failed evaluator-only entry must not
		// be re-run by the --bytecode fallback (its side effects would repeat).
		if n := strings.Count(out, "start"); n != 1 {
			t.Errorf("%v: entry side effect ran %d times, want 1\nstdout:\n%s", engine, n, out)
		}
	}

	evArgs := append(append([]string{}, base...), probe)
	bcArgs := append(append(append([]string{}, base...), "--bytecode"), probe)
	evOut, evErr, evCode := runWithSeed(t, bin, "42", evArgs...)
	bcOut, bcErr, bcCode := runWithSeed(t, bin, "42", bcArgs...)
	if evCode != 0 || bcCode != 0 {
		t.Fatalf("seeded runs failed: evaluator %d, bytecode %d\n%s\n%s", evCode, bcCode, evErr, bcErr)
	}
	if evOut != bcOut {
		t.Errorf("same AILANG_SEED, different draws:\nevaluator:\n%s\nbytecode:\n%s", evOut, bcOut)
	}
}

// The replay trace records the Rand mode observed at draw time — the same
// instrument TestModalRandEntrypoints uses to assert crypto dispatch.
func TestBytecodeRandMode_CryptoDrawsFromCrypto(t *testing.T) {
	bin := buildAilang(t)
	probe := writeProbe(t, "cryptoprobe.ail", cryptoProbeSrc)
	out, errOut, code := runWithSeed(t, bin, "", "run", "--bytecode", "--caps", "IO,Rand", "--entry", "main",
		"--quiet", "--emit-trace", "jsonl", probe)
	if code != 0 {
		t.Fatalf("crypto probe failed (exit %d)\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	all := out + errOut
	if n := strings.Count(all, `"mode":"crypto","contract":"opaque"`); n != 2 {
		t.Errorf("want 2 crypto-mode draws, got %d\n%s", n, all)
	}
	if strings.Contains(all, `"mode":"os"`) || strings.Contains(all, `"mode":"seeded"`) {
		t.Errorf("a Rand[mode=crypto] draw ran under another mode:\n%s", all)
	}
}

func TestBytecodeRandMode_StrictFailsLoudly(t *testing.T) {
	bin := buildAilang(t)
	probe := writeProbe(t, "modeprobe.ail", seededProbeSrc)
	out, errOut, code := runWithSeed(t, bin, "42", "run", "--bytecode", "--strict-bytecode", "--caps", "IO,Rand",
		"--entry", "main", "--quiet", probe)
	if code == 0 || !strings.Contains(errOut, "Rand[mode=seeded]") {
		t.Errorf("--strict-bytecode must refuse a Rand[mode=seeded] function naming the mode, got exit %d\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
}

// A per-function @limit is a frame too: under --bytecode the VM ran the
// function without pushing it, so the budget was never enforced.
func TestBytecodeBudget_LimitEnforced(t *testing.T) {
	bin := buildAilang(t)
	example := filepath.Join("examples", "tests", "test_capability_budget_exhausted.ail")
	out, errOut, code := runWithSeed(t, bin, "", "run", "--bytecode", "--caps", "IO", "--entry", "main", "--quiet", example)
	if code == 0 || !strings.Contains(errOut, "budget exhausted") {
		t.Errorf("@limit=2 not enforced under --bytecode (exit %d)\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if strings.Contains(out, "Call 3") {
		t.Errorf("the third IO call ran past @limit=2:\n%s", out)
	}
}
