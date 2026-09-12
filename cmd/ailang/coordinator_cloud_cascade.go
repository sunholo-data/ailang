package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The deterministic cascade path: a dependency bump the wrapper can apply and
// verify itself, with no AI agent invoked at all.
//
// Split out of coordinator_cloud.go when it crossed the 800-line ceiling
// (make check-file-sizes). Nothing here is used by the ordinary agent path.

// DispatchPath classifies how a cascade task should be handled.
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
type DispatchPath string

const (
	// DispatchDeterministic — wrapper applies the toml bump + lock + check + test
	// and only commits if all green. No AI agent is invoked.
	DispatchDeterministic DispatchPath = "deterministic"
	// DispatchAI — interface change with removed exports or widened effects;
	// the AI agent is invoked with full hash context to repair consumers.
	DispatchAI DispatchPath = "ai"
)

// classifyDispatchPath chooses how to handle a cascade task based on the
// change_class from the publisher. This mirrors classifyChange in
// internal/messaging/pkg_events.go but lives wrapper-side so the cloud job
// can decide without re-invoking the classifier.
//
// Mapping:
//
//	A (content-only)  → Deterministic
//	B (additive)      → Deterministic (consumer code keeps working — no exports removed, no effects widened)
//	C (interface)     → AI (something was removed OR effects widened — needs interpretation)
//	(unknown / empty) → AI (conservative default)
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func classifyDispatchPath(changeClass string) DispatchPath {
	switch changeClass {
	case "A", "B":
		return DispatchDeterministic
	default:
		return DispatchAI
	}
}

// deterministicCascadeBump performs the routine cascade work without invoking
// an AI agent: edit ailang.toml to point the dependency at the new version,
// regenerate ailang.lock, run check + test. If any step fails the function
// returns the error and the wrapper falls back to the AI escalation path.
//
// On success it stages and commits the changes; the wrapper's existing
// push + open-PR steps then run unchanged. The commit message tags the work
// as deterministic so the PR shows clearly that no AI was involved.
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func deterministicCascadeBump(ctx context.Context, workDir, rootPackage, toVersion, taskID, agentID string) error {
	// rootPackage format: "vendor/name@x.y.z" — strip the version since we
	// already have it as toVersion (and the toml maps to vendor/name only).
	parts := strings.SplitN(rootPackage, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid root package format %q (expected vendor/name@version)", rootPackage)
	}
	pkgName := parts[0]
	if toVersion == "" {
		toVersion = parts[1]
	}

	// 1. Read ailang.toml from the consumer package directory.
	tomlPath := filepath.Join(workDir, "ailang.toml")
	contentBytes, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("read ailang.toml at %s: %w", tomlPath, err)
	}

	// 2. Find and replace the dep version. ailang.toml uses TOML syntax
	// like:  "vendor/name" = "0.1.2"
	// We match the line with the package name and rewrite the version.
	pattern := fmt.Sprintf(`("%s"\s*=\s*)"[^"]+"`, regexp.QuoteMeta(pkgName))
	re := regexp.MustCompile(pattern)
	replacement := fmt.Sprintf(`${1}"%s"`, toVersion)
	newContent := re.ReplaceAllString(string(contentBytes), replacement)
	if newContent == string(contentBytes) {
		return fmt.Errorf("dep %q not found in ailang.toml at %s (or already at %s)", pkgName, tomlPath, toVersion)
	}
	if err := os.WriteFile(tomlPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write ailang.toml: %w", err)
	}
	fmt.Printf("execute-job: deterministic bump %s → %s in %s\n", pkgName, toVersion, tomlPath)

	// 3. Regenerate ailang.lock against the bumped dep.
	lockCmd := exec.CommandContext(ctx, "ailang", "lock")
	lockCmd.Dir = workDir
	if out, lockErr := lockCmd.CombinedOutput(); lockErr != nil {
		return fmt.Errorf("ailang lock failed: %w (output: %s)", lockErr, strings.TrimSpace(string(out)))
	}
	fmt.Println("execute-job: deterministic ailang lock regenerated")

	// 4. ailang check — this MUST pass for a class A or B bump. If it fails,
	// the change_class was misclassified at publish time or there's a real
	// breakage we didn't anticipate; either way, escalate to AI.
	checkCmd := exec.CommandContext(ctx, "ailang", "check", "--package", ".")
	checkCmd.Dir = workDir
	if out, checkErr := checkCmd.CombinedOutput(); checkErr != nil {
		return fmt.Errorf("ailang check failed (escalating to AI): %w (output: %s)", checkErr, strings.TrimSpace(string(out)))
	}
	fmt.Println("execute-job: deterministic ailang check passed")

	// 5. ailang test — only run if a *_test.ail exists in the package.
	testFiles, _ := filepath.Glob(filepath.Join(workDir, "*_test.ail"))
	if len(testFiles) > 0 {
		testCmd := exec.CommandContext(ctx, "ailang", "test", "--package", ".")
		testCmd.Dir = workDir
		if out, testErr := testCmd.CombinedOutput(); testErr != nil {
			return fmt.Errorf("ailang test failed (escalating to AI): %w (output: %s)", testErr, strings.TrimSpace(string(out)))
		}
		fmt.Printf("execute-job: deterministic ailang test passed (%d test files)\n", len(testFiles))
	}

	// 6. Stage + commit. The wrapper's existing Step 4 detects this commit
	// and the existing Step 5 does the push + PR-open.
	addCmd := exec.CommandContext(ctx, "git", "-C", workDir, "add", "-A")
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	commitMsg := fmt.Sprintf(
		"[cascade] bump %s to %s\n\n"+
			"Deterministic cascade bump by AILANG coordinator wrapper.\n"+
			"No AI agent was invoked — change_class permitted direct\n"+
			"toml bump + lock + check + test. All steps passed.\n\n"+
			"Task: %s\n"+
			"Agent: %s\n"+
			"Timestamp: %s",
		pkgName, toVersion, taskID, agentID, time.Now().UTC().Format(time.RFC3339),
	)
	commitCmd := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "-m", commitMsg)
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}
