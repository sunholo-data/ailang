package pipeline

import (
	"os"
	"strings"
	"testing"
)

// M-EFFECT-CEILING-REACHABLE: the [effects].max ceiling is computed over what
// the package's OWN code can reach — its functions' declared rows plus every
// effect carried by the type of any expression it contains — never over the
// bodies of imported std/dependency modules it merely imports from.
//
// The soundness controls (a package that DOES reach asyncExecProcess, by any
// route) matter more than the false-positive fix: under-approximating
// reachability would let a package exceed its ceiling silently.

const ceilingManifest = `[package]
name = "t/a"
version = "0.1.0"
edition = "1"

[effects]
max = ["Stream"]
`

// checkCeilingPkg writes a package (manifest + files) into a temp dir, chdirs
// there and type-checks entry. Returns the pipeline error (nil on success).
func checkCeilingPkg(t *testing.T, files map[string]string, entry string) error {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	mustWrite(t, dir, "ailang.toml", ceilingManifest)
	for name, src := range files {
		mustWrite(t, dir, name, src)
	}
	_, err = Run(Config{Mode: ModeCheck, RelaxModules: true}, Source{Filename: entry})
	return err
}

// The exact Daneel repro (2026-09-26): importing only disconnect must not
// charge std/stream's asyncExecProcess (Process) to the package.
func TestEffectCeiling_UnreferencedStdFunctionsNotCharged(t *testing.T) {
	err := checkCeilingPkg(t, map[string]string{
		"m.ail": `module m

import std/stream (disconnect, StreamConn)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`,
	}, "m.ail")
	if err != nil {
		t.Fatalf("package reaching only Stream must pass max=[Stream]; got: %v", err)
	}
}

// Importing a name is not reaching it: only a reference in the package's own
// code charges the imported function's effects.
func TestEffectCeiling_ImportWithoutReferenceNotCharged(t *testing.T) {
	err := checkCeilingPkg(t, map[string]string{
		"m.ail": `module m

import std/stream (asyncExecProcess, disconnect, StreamConn)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`,
	}, "m.ail")
	if err != nil {
		t.Fatalf("an imported-but-unreferenced function must not be charged; got: %v", err)
	}
}

// Precision controls: effects that only appear in the TYPES of the package's
// own functions or of caller-supplied callbacks are not reached authority.
// This is the motoko-ext hook shape: a hook must carry the ABI's fixed row
// even when its body uses less.
func TestEffectCeiling_OwnTypesAndCallbacksNotCharged(t *testing.T) {
	cases := map[string]string{
		"hook typed wider than its body": `module m

export func mk() -> (string) -> int ! {Stream, Process} = \c. 1
`,
		"caller-supplied callback stored, never called": `module m

type Box = { run: (string) -> int ! {Process} }

export func keep(f: (string) -> int ! {Process}) -> Box = { run: f }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if err := checkCeilingPkg(t, map[string]string{"m.ail": src}, "m.ail"); err != nil {
				t.Fatalf("no authority entry point reaches Process; must pass max=[Stream]; got: %v", err)
			}
		})
	}

	// A reference to an OWN sibling module's function is not an authority
	// entry point: that function's body is checked when its module is. This
	// is the motoko-ext _smoke.ail shape (a sibling references make_hooks,
	// whose type carries the ABI's hook rows).
	t.Run("reference to own sibling whose type carries wider rows", func(t *testing.T) {
		err := checkCeilingPkg(t, map[string]string{
			"hooks.ail": `module t/a/hooks

export func mkHook() -> (string) -> int ! {Stream, Process} = \c. 1
`,
			"m.ail": `module t/a/m

import ./hooks (mkHook)

export func hook() -> (string) -> int ! {Stream, Process} = mkHook()
`,
		}, "m.ail")
		if err != nil {
			t.Fatalf("own-module references must not be charged by type; got: %v", err)
		}
	})
}

func TestIsOwnPackageModule(t *testing.T) {
	cases := []struct {
		modID string
		want  bool
	}{
		{"m", true},
		{"sub/x", true},
		{"t/a/m", true},
		{"pkg/t/a/exec", true},
		{"pkg/t/ab/exec", false}, // name prefix is not a package match
		{"pkg/other/dep/x", false},
		{"std/stream", false},
		{"std", false},
	}
	for _, tc := range cases {
		if got := isOwnPackageModule(tc.modID, "t/a"); got != tc.want {
			t.Errorf("isOwnPackageModule(%q, t/a) = %v, want %v", tc.modID, got, tc.want)
		}
	}
}

// Soundness controls: every route by which the package's own code can reach
// asyncExecProcess must still violate max=[Stream].
func TestEffectCeiling_ReachedProcessStillViolates(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		entry string
	}{
		{
			name: "direct call",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess, StreamSource)

export func run() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)
`},
			entry: "m.ail",
		},
		{
			name: "private helper never called by an export",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess, StreamSource, disconnect, StreamConn)

func go() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`},
			entry: "m.ail",
		},
		{
			name: "returned closure from a pure function",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess, StreamSource)

export func mk() -> (string) -> StreamSource ! {Stream, Process} = \c. asyncExecProcess(c, [], "n", 0, 10)
`},
			entry: "m.ail",
		},
		{
			name: "function value passed through a higher-order helper",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess, StreamSource)

func pass[a](f: a) -> a = f

export func get() -> (string, [string], string, int, int) -> StreamSource ! {Stream, Process} = pass(asyncExecProcess)
`},
			entry: "m.ail",
		},
		{
			name: "stored in a record field",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess, StreamSource)

type Hooks = { start: (string) -> StreamSource ! {Stream, Process} }

export func hooks() -> Hooks = { start: \c. asyncExecProcess(c, [], "n", 0, 10) }
`},
			entry: "m.ail",
		},
		{
			name: "closure stored in a local, never called, no Process in any annotation",
			files: map[string]string{"m.ail": `module m

import std/stream (asyncExecProcess)

export func f() -> int {
  let g = \c. asyncExecProcess(c, [], "n", 0, 10);
  1
}
`},
			entry: "m.ail",
		},
		{
			name: "sibling module via relative import",
			files: map[string]string{
				"exec.ail": `module t/a/exec

import std/stream (asyncExecProcess, StreamSource)

export func execP() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)
`,
				"m.ail": `module t/a/m

import ./exec (execP)
import std/stream (disconnect, StreamConn)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`,
			},
			entry: "m.ail",
		},
		{
			name: "sibling module via bare canonical import",
			files: map[string]string{
				"exec.ail": `module t/a/exec

import std/stream (asyncExecProcess, StreamSource)

export func execP() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)
`,
				"m.ail": `module t/a/m

import t/a/exec (execP)
import std/stream (disconnect, StreamConn)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`,
			},
			entry: "m.ail",
		},
		{
			name: "sibling module via pkg/<self>/ wrapper (re-export shape)",
			files: map[string]string{
				"exec.ail": `module t/a/exec

import std/stream (asyncExecProcess, StreamSource)

export func execP() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)
`,
				"m.ail": `module t/a/m

import pkg/t/a/exec (execP)
import std/stream (disconnect, StreamConn)

export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
`,
			},
			entry: "m.ail",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCeilingPkg(t, tc.files, tc.entry)
			if err == nil {
				t.Fatalf("package reaching Process must violate max=[Stream]; got nil")
			}
			t.Logf("violation: %v", err)
			msg := err.Error()
			if !strings.Contains(msg, "effect ceiling violation") || !strings.Contains(msg, "Process") {
				t.Fatalf("want a ceiling violation naming Process; got: %v", err)
			}
			if strings.Contains(msg, "std/stream:") || strings.Contains(msg, "in function asyncExecProcess") {
				t.Fatalf("violation must be attributed to the package's own code, not std/stream: %v", err)
			}
		})
	}
}

// The compile cache must not serve a module that passed a WIDER ceiling after
// [effects].max is narrowed: the cache key did not include the ceiling, so a
// narrowed manifest was silently accepted until the source changed.
func TestEffectCeiling_NarrowedCeilingNotServedFromCache(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "")
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	wide := strings.Replace(ceilingManifest, `max = ["Stream"]`, `max = ["Stream", "Process"]`, 1)
	mustWrite(t, dir, "ailang.toml", wide)
	mustWrite(t, dir, "m.ail", `module m

import std/stream (asyncExecProcess, StreamSource)

export func run() -> StreamSource ! {Stream, Process} = asyncExecProcess("ls", [], "n", 0, 10)
`)
	// Prove the cache actually serves m under the wide ceiling before
	// narrowing it; otherwise this test would pass vacuously. The module's
	// key is not perfectly stable run-to-run (an occasional spurious MISS,
	// pre-existing and unrelated to the ceiling), so allow a few attempts.
	cfg := Config{Mode: ModeCheck, RelaxModules: true, DebugCompile: true}
	served := false
	for i := 0; i < 6 && !served; i++ {
		var runErr error
		out := capturePipelineStderrWith(nil, func() { _, runErr = Run(cfg, Source{Filename: "m.ail"}) })
		if runErr != nil {
			t.Fatalf("wide ceiling must pass (run %d): %v", i, runErr)
		}
		served = strings.Contains(out, "[CACHE] m: SKIP")
	}
	if !served {
		t.Fatalf("precondition: module m was never served from the compile cache")
	}

	// Narrow the ceiling. Every attempt must re-check m; three attempts keep
	// the regression visible despite the occasional spurious miss above.
	mustWrite(t, dir, "ailang.toml", ceilingManifest)
	for i := 0; i < 3; i++ {
		var runErr error
		_ = capturePipelineStderrWith(nil, func() { _, runErr = Run(cfg, Source{Filename: "m.ail"}) })
		if runErr == nil || !strings.Contains(runErr.Error(), "effect ceiling violation") {
			t.Fatalf("narrowed ceiling must be re-checked, not served from cache (attempt %d); got: %v", i, runErr)
		}
	}
}
