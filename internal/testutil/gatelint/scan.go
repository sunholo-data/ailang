// Package gatelint enforces the repository's explicit test-gating conventions.
package gatelint

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Rule identifies a gatelint rule.
type Rule string

const (
	RuleR1 Rule = "R1"
	RuleR2 Rule = "R2"
	RuleR3 Rule = "R3"
)

// Violation describes one rule violation in a first-party test file.
type Violation struct {
	Rule    Rule
	Path    string
	Line    int
	Message string
}

var scanRoots = []string{"internal", "cmd", "runtime", "std", "tests"}

// Scan checks first-party test files below root and returns violations in stable order.
func Scan(root string) []Violation {
	violations, _ := scan(root)
	return violations
}

func scan(root string) ([]Violation, int) {
	var violations []Violation
	scanned := 0
	for _, scanRoot := range scanRoots {
		base := filepath.Join(root, scanRoot)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != base && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "testdata") {
					return filepath.SkipDir
				}
				return nil
			}

			logicalName, fixture := logicalFileName(entry.Name())
			if !strings.HasSuffix(logicalName, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "internal/testutil/gatelint/") && !fixture {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			violations = append(violations, inspect(rel, logicalName, string(contents))...)
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			violations = append(violations, Violation{Rule: "WALK", Path: filepath.ToSlash(scanRoot), Message: err.Error()})
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		return violations[i].Rule < violations[j].Rule
	})
	return violations, scanned
}

func logicalFileName(name string) (string, bool) {
	const fixtureSuffix = ".fixture"
	if strings.HasSuffix(name, fixtureSuffix) {
		return strings.TrimSuffix(name, fixtureSuffix), true
	}
	return name, false
}

func inspect(path, name, contents string) []Violation {
	var violations []Violation
	if line := tokenLine(contents, "testing.Short("); line != 0 && !isAllowlisted(RuleR1, path) {
		violations = append(violations, Violation{Rule: RuleR1, Path: path, Line: line, Message: "testing.Short is inert in CI; delete the gate or use an explicit testutil opt-in helper"})
	}
	if line := firstTokenLine(contents, `Getenv("CI")`, `Getenv("GITHUB_ACTIONS")`); line != 0 && !isAllowlisted(RuleR2, path) {
		violations = append(violations, Violation{Rule: RuleR2, Path: path, Line: line, Message: "CI environment opt-out gate; use the explicit testutil gating convention"})
	}
	if !strings.HasSuffix(name, "_live_test.go") &&
		!strings.Contains(contents, "testutil.RequiresLiveNetwork(") &&
		!isAllowlisted(RuleR3, path) {
		if line := firstTokenLine(contents, "httpbin.org", "ailang-packages"); line != 0 {
			violations = append(violations, Violation{Rule: RuleR3, Path: path, Line: line, Message: "known third-party token outside a live-network gate or documented allowlist"})
		}
	}
	return violations
}

func firstTokenLine(contents string, tokens ...string) int {
	first := 0
	for _, token := range tokens {
		line := tokenLine(contents, token)
		if line != 0 && (first == 0 || line < first) {
			first = line
		}
	}
	return first
}

func tokenLine(contents, token string) int {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for line := 1; scanner.Scan(); line++ {
		if strings.Contains(scanner.Text(), token) {
			return line
		}
	}
	return 0
}
