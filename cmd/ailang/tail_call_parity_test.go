package main

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// M-EVAL-TAIL-CALLS (#1486): the interpreter runs tail calls in constant depth,
// matching the bytecode VM, at the DEFAULT --max-recursion-depth (10,000).
// Before the change every interpreter row below failed RT_REC_003.

func runWithStdin(t *testing.T, bin, stdin string, args ...string) (string, string, int) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return out.String(), errb.String(), code
}

func TestTailCallParityWithVM(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "tailcall", "shapes.ail")
	const n = "200000"
	want := map[string]string{"ifShape": n, "matchShape": n, "letShape": n, "mutualShape": "true"}
	for entry, w := range want {
		interp, ierr, icode := runWithStdin(t, bin, "", "run", "--quiet", "--entry", entry, "--args-json", n, src)
		vm, verr, vcode := runWithStdin(t, bin, "", "run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", entry, "--args-json", n, src)
		if icode != 0 || vcode != 0 {
			t.Errorf("%s: exit interp=%d vm=%d\ninterp: %s\nvm: %s", entry, icode, vcode, ierr, verr)
			continue
		}
		if strings.TrimSpace(interp) != w || interp != vm {
			t.Errorf("%s: interpreter %q, VM %q, want %q", entry, strings.TrimSpace(interp), strings.TrimSpace(vm), w)
		}
	}
}

func TestTailCallStdinLoop1486(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "tailcall", "stdin_loop.ail")
	var in strings.Builder
	for i := 1; i <= 20000; i++ {
		in.WriteString("x\n")
	}
	out, stderr, code := runWithStdin(t, bin, in.String(), "run", "--quiet", "--caps", "IO", "--entry", "main", src)
	if code != 0 || strings.TrimSpace(out) != "20000" {
		t.Fatalf("exit=%d out=%q stderr=%s", code, strings.TrimSpace(out), stderr)
	}
}

// Non-tail recursion is still bounded by RT_REC_003 at the default depth.
func TestTailCallNonTailStillBounded(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join(t.TempDir(), "nontail.ail")
	writeFile(t, src, "module nontail\npure func sum(i: int) -> int = if i == 0 then 0 else 1 + sum(i - 1)\nexport pure func main(n: int) -> int = sum(n)\n")
	_, stderr, code := runWithStdin(t, bin, "", "run", "--quiet", "--entry", "main", "--args-json", "50000", src)
	if code == 0 || !strings.Contains(stderr, "RT_REC_003") {
		t.Fatalf("exit=%d stderr=%s, want RT_REC_003", code, stderr)
	}
}
