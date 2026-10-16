package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageSourceCheckDiscovery(t *testing.T) {
	bin := buildAilang(t)
	for _, tc := range []struct {
		name, file, source string
		broken             bool
	}{
		{"non-exported", "bad.ail", "module probe/p/bad\nexport pure func bad() -> int = \"bad\"\n", true},
		{"test-module", "bad_test.ail", "module probe/p/bad_test\nexport pure func bad() -> int = \"bad\"\n", true},
		{"orphan", "orphan.ail", "this is not a module or valid source", false},
		{"hidden", ".hidden/bad.ail", "module probe/p/bad\nexport pure func bad() -> int = \"bad\"\n", false},
		{"node_modules", "node_modules/bad.ail", "module probe/p/bad\nexport pure func bad() -> int = \"bad\"\n", false},
		{"vendor", "_vendor/bad.ail", "module probe/p/bad\nexport pure func bad() -> int = \"bad\"\n", false},
		{"named-test-debris", "_namedtest_body_0.ail", "module probe/p/bad\nexport pure func bad() -> int = \"bad\"\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := sourceCheckPackage(t, true)
			writeSourceCheckFile(t, dir, tc.file, tc.source)
			_, checkErr, checkExit := runAilangBin(t, bin, "check", "--package", dir)
			out, stderr, exit := runAilangBin(t, bin, "test", "--package", dir, "--json")
			wantExit := 0
			if tc.broken {
				wantExit = 1
			}
			if checkExit != wantExit || exit != checkExit {
				t.Fatalf("check=%d test=%d want=%d\n%s\n%s\n%s", checkExit, exit, wantExit, checkErr, out, stderr)
			}
			var doc struct {
				Total  int `json:"total_tests"`
				Source struct {
					Passed int `json:"passed"`
					Failed int `json:"failed"`
				} `json:"source_check"`
			}
			if err := json.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatal(err, out)
			}
			if doc.Source.Passed != 2 || (doc.Source.Failed > 0) != tc.broken || (doc.Total == 0) != tc.broken {
				t.Fatalf("unexpected discovery: %s", out)
			}
		})
	}
}

func TestPackageSourceCheckStrictFallback(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	writeStrictFallbackPackage(t, dir, strictFallbackViolationSrc)
	writeSourceCheckFile(t, dir, "client_test.ail", "module sfvendor/sfpkg/client_test\ntest \"must not run\" { true }\n")
	out, stderr, exit := runAilangBin(t, bin, "test", "--package", dir, "--json", "--allow-skips")
	var doc struct {
		Total   int  `json:"total_tests"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err, out, stderr)
	}
	if exit != 1 || doc.Total != 0 || doc.Success || !strings.Contains(out, "STRICT_FALLBACK_001") {
		t.Fatalf("exit=%d %s %s", exit, out, stderr)
	}
}

func TestPackageSourceCheckEnginesAndFileMode(t *testing.T) {
	bin := buildAilang(t)
	dir := sourceCheckPackage(t, true)
	for _, flag := range []string{"--bytecode", "--strict-bytecode"} {
		out, stderr, exit := runAilangBin(t, bin, "test", "--package", dir, "--json", flag)
		if exit != 0 || !strings.Contains(out, `"source_check"`) {
			t.Fatalf("%s exit=%d %s %s", flag, exit, out, stderr)
		}
	}
	file := filepath.Join(dir, "good_test.ail")
	for _, args := range [][]string{{"test", file, "--json"}, {"test", file, "--json", "--no-source-check"}} {
		out, stderr, exit := runAilangBin(t, bin, args...)
		if exit != 0 || strings.Contains(out, `"source_check"`) {
			t.Fatalf("file mode exit=%d %s %s", exit, out, stderr)
		}
	}
	out, stderr, exit := runAilangBin(t, bin, "test", "--package", filepath.Join("examples", "intra_package_imports"), "--json")
	if exit != 0 || !json.Valid([]byte(out)) {
		t.Fatalf("zero-test example exit=%d %s %s", exit, out, stderr)
	}
}
