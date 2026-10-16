package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// #1845: `ailang test --package` must fail on a type error in a module that no
// test imports, as `ailang check --package` does.
func TestPackageTestTypeChecksUntestedModules(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ailang.toml"), `[package]
name = "probe/p"
version = "0.1.0"
edition = "1"

[exports]
modules = ["probe/p/good", "probe/p/bad"]
`)
	writeFile(t, filepath.Join(dir, "good.ail"), "module probe/p/good\nexport pure func one() -> int = 1\n")
	writeFile(t, filepath.Join(dir, "good_test.ail"), "module probe/p/good_test\nimport probe/p/good (one)\ntest \"one is one\" { one() == 1 }\n")
	bad := filepath.Join(dir, "bad.ail")
	writeFile(t, bad, "module probe/p/bad\nexport pure func oops() -> int = \"not an int\"\n")

	stdout, stderr, exit := runAilangBin(t, bin, "test", "--package", "--format", "json", dir)
	if exit == 0 || !strings.Contains(stderr, "bad.ail") {
		t.Fatalf("untested type error: exit=%d stderr=%q; want non-zero naming bad.ail", exit, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\n%s", err, stdout)
	}

	writeFile(t, bad, "module probe/p/bad\nexport pure func oops() -> int = 2\n")
	if _, stderr, exit := runAilangBin(t, bin, "test", "--package", dir); exit != 0 {
		t.Fatalf("clean package: exit=%d stderr=%q; want 0", exit, stderr)
	}
}
