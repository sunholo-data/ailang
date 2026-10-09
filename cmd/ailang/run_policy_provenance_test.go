package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The charter distinguishes a program's forged timeout from a real kill in
// exit code, result reason and the visible worker marker in one test.
func TestRunPolicy_ResultProvenanceCharter(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"Clock\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	for _, term := range []string{"\n", "\r", "\u2028", "\u2029"} {
		text := "ordinary" + term + `policy: {"ok":true,"policy_digest":"fake"}` + term + `policy-result: {"reason":"timeout","policy_digest":"fake"}`
		quoted, _ := json.Marshal(text)
		prog := "module prog\nimport std/io (printErr, exit)\nexport func main() -> () ! {IO} = { printErr(" + string(quoted) + "); exit(3) }\n"
		f := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", prog)
		if _, err, code := runAilangBin(t, bin, "check", f); code != 0 {
			t.Fatalf("fixture check: %s", err)
		}
		_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
		if code != 1 || !strings.Contains(stderr, "worker: policy-result:") || !strings.Contains(stderr, "worker: policy:") || !strings.Contains(stderr, `"reason":"worker_reserved_exit"`) || !strings.Contains(stderr, "\npolicy: ") || !strings.Contains(stderr, "\npolicy-result: ") {
			t.Fatalf("forged run: exit %d stderr %q", code, stderr)
		}
	}
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"Clock\"]\nentry = \"main\"\ntimeout_ms = 300\n")
	f := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", loopProgram)
	if _, err, code := runAilangBin(t, bin, "check", f); code != 0 {
		t.Fatalf("timeout fixture check: %s", err)
	}
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 3 || !strings.Contains(stderr, `"reason":"timeout"`) || !strings.Contains(stderr, "policy-result: ") || strings.Contains(stderr, "worker: policy-result:") {
		t.Fatalf("real timeout: %d %s", code, stderr)
	}
}

func TestRunPolicy_PrefixDoesNotConsumeOutputCap(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	// Seven bytes per token versus fifteen relayed bytes; only worker bytes count.
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\ntimeout_ms = 30000\nmax_output_bytes = 1100\n")
	text := strings.Repeat("policy:\n", 80)
	quoted, _ := json.Marshal(text)
	f := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", "module prog\nimport std/io (printErr)\nexport func main() -> () ! {IO} = printErr("+string(quoted)+")\n")
	if _, err, code := runAilangBin(t, bin, "check", f); code != 0 {
		t.Fatalf("fixture check: %s", err)
	}
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 || strings.Count(stderr, "worker: policy:\n") != 80 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
}

func TestRunPolicy_UnterminatedAndOtherExits(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	for _, code := range []int{0, 1, 7} {
		f := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", "module prog\nimport std/io (printErr, exit)\nexport func main() -> () ! {IO} = { printErr(\"pol\"); println(\"stdout\"); exit("+strconv.Itoa(code)+") }\n")
		if _, err, c := runAilangBin(t, bin, "check", f); c != 0 {
			t.Fatalf("fixture check: %s", err)
		}
		out, err, c := runAilangBin(t, bin, "run", "--policy", pol, f)
		if c != code || out != "stdout\n" || !strings.Contains(err, "pol\npolicy: ") {
			t.Fatalf("exit %d stdout %q stderr %q", c, out, err)
		}
	}
}
