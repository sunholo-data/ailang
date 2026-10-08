package main

// `ailang test --bytecode` / `--strict-bytecode` (#1487), end to end: the JSON
// report on stdout is the evaluator's, and stderr names where bodies ran.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTestCommandBytecodeFlags(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "engine_test.ail")
	src := `module engine_test

pure func sq(x: float) -> float = x * x

test "passes" { sq(3.0) == 9.0 }

test "fails" { sq(2.0) == 5.0 }

test "non-bool body" { sq(1.0) }

test "runtime error" { 1 / 0 == 0 }
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	// Durations differ run to run; everything else must match.
	report := func(stdout string) map[string]any {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal([]byte(stdout), &m); err != nil {
			t.Fatalf("decode JSON report: %v\n%s", err, stdout)
		}
		delete(m, "total_duration")
		delete(m, "total_duration_ms")
		if tests, ok := m["tests"].([]any); ok {
			for _, tc := range tests {
				if tm, ok := tc.(map[string]any); ok {
					delete(tm, "duration")
					delete(tm, "duration_ms")
				}
			}
		}
		return m
	}

	evOut, evErr, evExit := runAilangBin(t, bin, "test", "--json", "--no-color", file)
	// Flags after the path, as parseTestArgs allows.
	bcOut, bcErr, bcExit := runAilangBin(t, bin, "test", "--json", "--no-color", file, "--bytecode")
	if evExit == 0 || bcExit != evExit {
		t.Fatalf("exit codes: evaluator %d, bytecode %d; want the same non-zero code\nstderr:\n%s\n%s", evExit, bcExit, evErr, bcErr)
	}
	if got, want := report(bcOut), report(evOut); !reflect.DeepEqual(got, want) {
		t.Errorf("--bytecode JSON report differs\nbytecode:  %v\nevaluator: %v", got, want)
	}
	if strings.Contains(evErr, "bytecode:") {
		t.Errorf("evaluator run printed an engine summary:\n%s", evErr)
	}
	// The non-bool body runs on the VM (its batched entry carries no `-> bool`,
	// M-TEST-RUNNER-COMPILE-ONCE); the runtime error falls back.
	if !strings.Contains(bcErr, "bytecode: 3 named-test bodies ran on the VM, 1 fell back to the evaluator") {
		t.Errorf("stderr does not report where bodies ran:\n%s", bcErr)
	}

	stOut, stErr, _ := runAilangBin(t, bin, "test", "--strict-bytecode", "--json", "--no-color", file)
	if !strings.Contains(stOut, "--strict-bytecode") || !strings.Contains(stErr, "1 failed under --strict-bytecode") {
		t.Errorf("--strict-bytecode did not fail the non-VM body loudly\nstdout:\n%s\nstderr:\n%s", stOut, stErr)
	}
}
