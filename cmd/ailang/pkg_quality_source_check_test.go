package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// The quality subprocess launches os.Executable. In this test that is this
// test executable: intercept only its test-command invocation before flag parsing.
func init() {
	if dest := os.Getenv("AILANG_TEST_QUALITY_ARGS"); dest != "" && len(os.Args) > 1 && os.Args[1] == "test" {
		data, err := json.Marshal(os.Args[1:])
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			panic(err)
		}
		fmt.Println(`{"passed_tests":2,"failed_tests":0,"skipped_tests":1}`)
		os.Exit(0)
	}
}

func TestQualitySourceCheckOptOut(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.json")
	t.Setenv("AILANG_TEST_QUALITY_ARGS", argsFile)
	// Discovery only needs a filename; the helper subprocess executes no AILANG.
	if err := os.WriteFile(filepath.Join(dir, "good_test.ail"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	result := runAttestedChecks(dir, &pkg.PackageManifest{})
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(args, "--no-source-check") {
		t.Fatalf("quality repeats source check: %v", args)
	}
	if result.Tests.Files != 1 || result.Tests.Passed != 2 || result.Tests.Failed != 0 || result.Tests.Skipped != 1 || result.Tests.Notes != "" {
		t.Fatalf("quality test summary changed: %+v", result.Tests)
	}
}
