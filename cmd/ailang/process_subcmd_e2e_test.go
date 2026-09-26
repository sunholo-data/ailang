package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// M-PROCESS-SUBCMD M3: the flag string reaches the runtime. `--process-allowlist
// echo:hello` lets `echo hello` run and refuses `echo bye` with the subcommand
// named — through the real binary, so setupProcessHandler is on the path.
func TestRun_ProcessAllowlist_Subcommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo as a binary requires unix")
	}
	bin := buildAilang(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "sub.ail")
	prog := `module sub
import std/io (println)
import std/process (exec)
import std/bytes (toString)

export func main() -> () ! {IO, Process} {
  match exec("echo", ["hello"]) {
    Ok(out) => println("ok: ${toString(out.stdout)}"),
    Err(e) => println("err: ${show(e)}")
  };
  match exec("echo", ["bye"]) {
    Ok(_) => println("LEAK: bye ran"),
    Err(e) => println("refused: ${show(e)}")
  }
}
`
	if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runAilangBin(t, bin, "run", "--relax-modules", "--caps", "IO,Process",
		"--process-allowlist", "echo:hello", "--entry", "main", src)
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "ok: hello") {
		t.Errorf("echo hello should run under echo:hello\nstdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "LEAK") {
		t.Errorf("echo bye ran under echo:hello\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "refused: NotAllowed(echo bye)") {
		t.Errorf("refusal should name the subcommand\nstdout:\n%s", stdout)
	}
}

// A malformed entry must stop the run at startup, not silently widen the grant.
func TestRun_ProcessAllowlist_MalformedEntryFails(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "m.ail")
	if err := os.WriteFile(src, []byte("module m\nexport func main() -> int ! {Process} = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runAilangBin(t, bin, "run", "--relax-modules", "--caps", "Process",
		"--process-allowlist", "echo:", "--entry", "main", src)
	if code == 0 {
		t.Fatalf("expected non-zero exit for --process-allowlist echo:\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stdout+stderr, "echo:") {
		t.Errorf("error should name the offending entry\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}
