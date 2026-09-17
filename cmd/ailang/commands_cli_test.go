package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cliTestBin returns the binary the CLI-surface tests drive.
//
// AILANG_CLI_TEST_BIN points them at a PREBUILT binary instead, which is how
// TestCLI_HelpExitsZeroEverywhere is run as a negative control:
//
//	go build -ldflags "-X .../version.Version=vSNAP -X .../version.Commit=SNAPSHOT \
//	  -X .../version.BuildTime=SNAPSHOT" -o bin/ailang-before ./cmd/ailang   # unedited tree
//	AILANG_CLI_TEST_BIN=$PWD/bin/ailang-before go test ./cmd/ailang -run HelpExitsZero -count=1
//
// Against the pre-M1 binary it FAILS, naming every command that rejected
// --help. That is the measurement the milestone is judged on.
func cliTestBin(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("AILANG_CLI_TEST_BIN"); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			t.Fatalf("AILANG_CLI_TEST_BIN: %v", err)
		}
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("AILANG_CLI_TEST_BIN=%s: %v", abs, err)
		}
		return abs
	}
	return buildAilang(t)
}

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
	timedOut bool
}

// runCLIIsolated runs the binary with a private HOME, a private state
// directory and a private working directory, so a run reads no fleet state and
// writes none. Anything that outlives the bound is killed and reported.
func runCLIIsolated(t *testing.T, bin string, args ...string) cliResult {
	t.Helper()
	return runCLIIn(t, bin, t.TempDir(), t.TempDir(), args...)
}

// runCLIIn is runCLIIsolated with the HOME and working directory supplied by
// the caller, so two invocations can be compared byte for byte.
//
// Two commands need it: `ailang budget` and `ailang lock` print their working
// directory, so a fresh t.TempDir() per run makes the SAME route differ from
// itself (.../002 versus .../004). That is a property of the instrument, not
// of the routes, and the pair test in commands_groups_test.go would otherwise
// report it as a real difference.
func runCLIIn(t *testing.T, bin, home, work string, args ...string) cliResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = work
	// cmd.Env, not a HOME override in this process: the gate on hand-rolled
	// HOME overrides (make check-home-isolation) exists because an in-process
	// override leaks between tests. A child environment does not.
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
		"AILANG_STATE_DIR="+filepath.Join(home, ".ailang", "state"),
		"NO_COLOR=1",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader("")

	err := cmd.Run()
	res := cliResult{stdout: stdout.String(), stderr: stderr.String()}
	if ctx.Err() != nil {
		res.timedOut = true
	}
	var ee *exec.ExitError
	switch {
	case err == nil:
		res.exitCode = 0
	case errors.As(err, &ee):
		res.exitCode = ee.ExitCode()
	default:
		t.Fatalf("running %s %v: %v", bin, args, err)
	}
	return res
}

// TestCLI_HelpExitsZeroEverywhere is the M1 acceptance gate: --help is answered
// at every level. Before M1 twenty top-level commands rejected it — eight of
// them ("chains", "dashboard", "daemon", "eval-chains", "observatory", "pi",
// "pkg", "trace") with an "unknown subcommand" error, the rest by treating
// --help as a filename, a results directory or an unknown flag.
func TestCLI_HelpExitsZeroEverywhere(t *testing.T) {
	bin := cliTestBin(t)

	var routes []string
	for i := range allCommands {
		c := &allCommands[i]
		routes = append(routes, c.Name)
		routes = append(routes, c.Aliases...)
	}
	for _, sub := range pkgSubcommands() {
		routes = append(routes, "pkg "+sub.Name)
	}
	routes = append(routes, "") // the top level itself

	for _, route := range routes {
		route := route
		name := route
		if name == "" {
			name = "(top level)"
		}
		t.Run(name, func(t *testing.T) {
			args := append(strings.Fields(route), "--help")
			res := runCLIIsolated(t, bin, args...)
			if res.timedOut {
				t.Fatalf("`ailang %s --help` did not finish inside the bound", route)
			}
			if res.exitCode != 0 {
				t.Errorf("`ailang %s --help` exited %d, want 0\nstdout: %s\nstderr: %s",
					route, res.exitCode, firstLines(res.stdout, 3), firstLines(res.stderr, 3))
			}
			if strings.TrimSpace(res.stdout) == "" && strings.TrimSpace(res.stderr) == "" {
				t.Errorf("`ailang %s --help` printed nothing at all", route)
			}
		})
	}
}

// TestCLI_UnknownCommand pins the replacement for the old default arm, which
// printed 16 KB of printHelp() to STDOUT before exiting 1.
func TestCLI_UnknownCommand(t *testing.T) {
	bin := cliTestBin(t)
	res := runCLIIsolated(t, bin, "chian")

	if res.exitCode != 1 {
		t.Errorf("exit code = %d, want 1", res.exitCode)
	}
	if res.stdout != "" {
		t.Errorf("stdout is not empty (%d bytes); an unknown command must not pollute stdout:\n%s",
			len(res.stdout), firstLines(res.stdout, 5))
	}
	lines := nonEmptyLines(res.stderr)
	if len(lines) != 2 {
		t.Fatalf("stderr has %d non-empty lines, want 2 (the error and the suggestion):\n%s", len(lines), res.stderr)
	}
	if !strings.Contains(lines[0], "unknown command 'chian'") {
		t.Errorf("first stderr line = %q, want it to name the unknown command", lines[0])
	}
	if !strings.Contains(lines[1], "chains") {
		t.Errorf("second stderr line = %q, want a did-you-mean suggestion", lines[1])
	}
}

// TestCLI_DaemonWithNoArgsDoesNotStart pins the most dangerous default in the
// pre-M1 CLI: `ailang daemon` defaulted its subcommand to "run" and started the
// daemon in the foreground. The 30s bound in runCLIIsolated is what makes this
// test unable to hang if the fix is reverted — a started daemon is killed and
// reported as a timeout, which fails the test.
func TestCLI_DaemonWithNoArgsDoesNotStart(t *testing.T) {
	bin := cliTestBin(t)
	res := runCLIIsolated(t, bin, "daemon")

	if res.timedOut {
		t.Fatalf("`ailang daemon` did not exit — it is still starting the daemon")
	}
	if res.exitCode == 0 {
		t.Errorf("exit code = 0, want non-zero: a bare `ailang daemon` is a usage error")
	}
	out := res.stdout + res.stderr
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "ailang daemon <subcommand>") {
		t.Errorf("no usage block printed:\nstdout: %s\nstderr: %s", res.stdout, res.stderr)
	}
	for _, sub := range []string{"run", "install", "uninstall", "status"} {
		if !strings.Contains(out, sub) {
			t.Errorf("usage does not name the %q subcommand", sub)
		}
	}
}

// TestCLI_StartupProbesAreObservable is the END-TO-END control on the probe
// contract, independent of the in-process counting fake in
// commands_table_test.go. Both startup probes are made observable from outside
// the process — an oversized observatory DB makes the health check log, and a
// source tree newer than the binary makes the stale-binary probe warn — and
// then counted: a platform command emits both, a language command neither.
func TestCLI_StartupProbesAreObservable(t *testing.T) {
	if testing.Short() {
		t.Skip("creates a 2.1 GB sparse file")
	}
	bin := cliTestBin(t)

	home := t.TempDir()
	stateDir := filepath.Join(home, ".ailang", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Sparse: no blocks are allocated, only the size the health check stats.
	db, err := os.Create(filepath.Join(stateDir, "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Truncate(2200 * 1024 * 1024); err != nil {
		_ = db.Close()
		t.Skipf("cannot create a sparse 2.1 GB file here: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// A working directory that looks like a checkout whose sources are newer
	// than the binary, which is what the stale-binary probe samples.
	work := t.TempDir()
	src := filepath.Join(work, "internal", "parser")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(src, "parser.go")
	if err := os.WriteFile(srcFile, []byte("package parser\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(srcFile, future, future); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(),
			"HOME="+home,
			"USERPROFILE="+home,
			"AILANG_STATE_DIR="+stateDir,
			"NO_COLOR=1",
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stdin = strings.NewReader("")
		_ = cmd.Run() // exit code is not the signal here; stderr is
		return stderr.String()
	}

	count := func(stderr string) (obs, stale int) {
		return strings.Count(stderr, "Observatory:"), strings.Count(stderr, "Binary may be stale")
	}

	// Positive control FIRST: without it, zero probes on the language command
	// would only prove the instrument is blind.
	obs, stale := count(run("chains"))
	if obs < 1 || stale < 1 {
		t.Fatalf("platform command emitted observatory=%d stale=%d, want >=1 of each — "+
			"the instrument cannot see the probes, so the negative result below would mean nothing", obs, stale)
	}

	for _, lang := range []string{"version", "check", "fmt", "docs", "builtins"} {
		obs, stale := count(run(lang))
		if obs != 0 || stale != 0 {
			t.Errorf("language command %q emitted observatory=%d stale=%d, want 0 of each", lang, obs, stale)
		}
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
