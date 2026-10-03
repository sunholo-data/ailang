package main

// `ailang test FILE1 FILE2` used to run FILE1 only and silently drop the rest
// (reported by stapledons_godot): runTestCommand read testFlags.Arg(0) and
// ignored every later positional — including flags placed after the first
// path, which Go's flag package stops parsing at.

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseTestArgs_FlagsAnywhereAndAllPaths(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantPaths []string
		wantJSON  bool
	}{
		{"none", nil, nil, false},
		{"flags then paths", []string{"--json", "a.ail", "b.ail"}, []string{"a.ail", "b.ail"}, true},
		{"flag between paths", []string{"a.ail", "--json", "b.ail"}, []string{"a.ail", "b.ail"}, true},
		{"flag after paths", []string{"a.ail", "b.ail", "--json"}, []string{"a.ail", "b.ail"}, true},
		{"double dash makes the rest paths", []string{"a.ail", "--", "--json"}, []string{"a.ail", "--json"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			jsonFlag := fs.Bool("json", false, "")
			paths, err := parseTestArgs(fs, tc.args)
			if err != nil {
				t.Fatalf("parseTestArgs(%q): %v", tc.args, err)
			}
			if !reflect.DeepEqual(paths, tc.wantPaths) {
				t.Errorf("paths = %q, want %q", paths, tc.wantPaths)
			}
			if *jsonFlag != tc.wantJSON {
				t.Errorf("--json = %v, want %v", *jsonFlag, tc.wantJSON)
			}
		})
	}
}

// End to end: every file given runs, the summary aggregates them, and one
// failing file makes the whole run exit non-zero — whichever position it has.
func TestTestCommandRunsEveryFileGiven(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pass1 := filepath.Join(dir, "one_test.ail")
	pass2 := filepath.Join(dir, "two_test.ail")
	fail := filepath.Join(dir, "bad_test.ail")
	for path, body := range map[string]string{
		pass1: "module one_test\n\ntest \"one\" { 1 + 1 == 2 }\n",
		pass2: "module two_test\n\ntest \"two\" { 2 + 2 == 4 }\n",
		fail:  "module bad_test\n\ntest \"bad\" { 1 == 2 }\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	type summary struct {
		Total  int `json:"total_tests"`
		Passed int `json:"passed_tests"`
		Failed int `json:"failed_tests"`
	}
	decode := func(stdout string) summary {
		t.Helper()
		var got summary
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("decode JSON report: %v\n%s", err, stdout)
		}
		return got
	}

	// Two passing files, --json AFTER the paths: both run, flag honoured.
	stdout, stderr, exit := runAilangBin(t, bin, "test", "--no-color", pass1, pass2, "--json")
	if exit != 0 {
		t.Fatalf("two passing files: exit %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if got := decode(stdout); got.Total != 2 || got.Passed != 2 {
		t.Fatalf("two passing files: summary %+v, want 2 total / 2 passed", got)
	}

	// A failing file in SECOND position must fail the run.
	stdout, stderr, exit = runAilangBin(t, bin, "test", "--json", "--no-color", pass1, fail, pass2)
	if exit == 0 {
		t.Fatalf("failing second file: exit 0, want non-zero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if got := decode(stdout); got.Total != 3 || got.Passed != 2 || got.Failed != 1 {
		t.Fatalf("failing second file: summary %+v, want 3 total / 2 passed / 1 failed", got)
	}

	// --package names ONE package directory; extra paths are refused loudly.
	_, stderr, exit = runAilangBin(t, bin, "test", "--package", dir, dir)
	if exit == 0 || !strings.Contains(stderr, "--package") {
		t.Fatalf("--package with two dirs: exit %d, stderr %q; want a non-zero refusal naming --package", exit, stderr)
	}
}

// A `_namedtest_body_<n>.ail` left in a directory by an interrupted run of an
// older binary (#1502) is a copy of a real test module; walking it used to run
// that module's tests twice.
func TestTestCommandSkipsLeftoverNamedTestBodies(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	src := "module one_test\n\ntest \"one\" { 1 + 1 == 2 }\n"
	for _, name := range []string{"one_test.ail", "_namedtest_body_607890236.ail"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	stdout, stderr, exit := runAilangBin(t, bin, "test", "--json", "--no-color", dir)
	if exit != 0 {
		t.Fatalf("exit %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	var got struct {
		Total int `json:"total_tests"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v\n%s", err, stdout)
	}
	if got.Total != 1 {
		t.Errorf("total_tests = %d, want 1 (the leftover body copy must not run)", got.Total)
	}
	if !strings.Contains(stderr, "_namedtest_body_607890236.ail") {
		t.Errorf("stderr does not name the skipped leftover file:\n%s", stderr)
	}
}
