package gatelint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGateLint_SelfTest(t *testing.T) {
	root := filepath.Join("testdata", "fixtures")
	violations, scanned := scan(root)
	if scanned != 4 {
		t.Fatalf("scanned %d candidate test files, want 4", scanned)
	}

	got := violationSet(violations)
	want := []string{
		"R1:internal/r1_test.go.fixture",
		"R2:cmd/r2_test.go.fixture",
		"R3:tests/r3_test.go.fixture",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("violations mismatch\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestGateLint_Repo(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	violations, scanned := scan(root)
	if scanned == 0 {
		t.Fatal("gatelint scanned zero *_test.go files; repository-root or walker scope is broken")
	}
	if len(violations) != 0 {
		lines := make([]string, len(violations))
		for i, violation := range violations {
			lines[i] = fmt.Sprintf("%s:%d: %s: %s", violation.Path, violation.Line, violation.Rule, violation.Message)
		}
		t.Fatalf("gatelint found %d violation(s) after scanning %d files:\n%s", len(violations), scanned, strings.Join(lines, "\n"))
	}
	t.Logf("scanned %d first-party test files", scanned)
}

func violationSet(violations []Violation) []string {
	set := make([]string, len(violations))
	for i, violation := range violations {
		set[i] = string(violation.Rule) + ":" + violation.Path
	}
	sort.Strings(set)
	return set
}
