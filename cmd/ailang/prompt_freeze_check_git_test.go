package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const freezeCheckVersion = "v1.0.0"

func TestFreezeCheck_MergeBaseEditPlusRegenIsRed(t *testing.T) {
	root := newFreezeCheckRepo(t)
	changed := []byte("changed prompt\n")
	writeFreezeCheckFile(t, root, "prompts/v1.0.0.md", changed)
	writeFreezeCheckFile(t, root, "cmd/ailang/prompts/v1.0.0.md", changed)
	writeFreezeCheckRegistries(t, root, hashBytes(changed), hashBytes(changed), true)

	violations, rc := runFreezeCheck(t, root)
	if rc != 1 || len(violations) != 1 || !strings.Contains(violations[0], freezeCheckVersion) || !strings.Contains(violations[0], "immutability") {
		t.Fatalf("want only the discriminating immutability violation for %s, got %v", freezeCheckVersion, violations)
	}
}

func TestFreezeCheck_FrozenPlaceholderIsRed(t *testing.T) {
	root := newFreezeCheckRepo(t)
	writeFreezeCheckRegistries(t, root, "PLACEHOLDER", "PLACEHOLDER", true)

	violations, rc := runFreezeCheck(t, root)
	if rc != 1 || !hasViolation(violations, freezeCheckVersion, "unenforceable freeze") {
		t.Fatalf("want frozen placeholder violation naming %s, got %v", freezeCheckVersion, violations)
	}
}

func TestFreezeCheck_MirrorMdDivergenceIsRed(t *testing.T) {
	root := newFreezeCheckRepo(t)
	writeFreezeCheckFile(t, root, "cmd/ailang/prompts/v1.0.0.md", []byte("different mirror\n"))

	violations, rc := runFreezeCheck(t, root)
	if rc != 1 || !hasViolation(violations, freezeCheckVersion, "cmd/ailang/prompts/v1.0.0.md") {
		t.Fatalf("want mirror-byte violation naming its path and %s, got %v", freezeCheckVersion, violations)
	}
}

func TestFreezeCheck_MirrorRegistryDivergenceIsRed(t *testing.T) {
	root := newFreezeCheckRepo(t)
	base := []byte("base prompt\n")
	writeFreezeCheckRegistries(t, root, hashBytes(base), strings.Repeat("a", 64), true)

	violations, rc := runFreezeCheck(t, root)
	if rc != 1 || len(violations) != 1 || !hasViolation(violations, freezeCheckVersion, "cmd/ailang/prompts/versions.json") {
		t.Fatalf("want only mirror-registry violation naming path and %s, got %v", freezeCheckVersion, violations)
	}
}

func TestFreezeCheck_UnmodifiedTreeIsGreen(t *testing.T) {
	root := newFreezeCheckRepo(t)
	violations, rc := runFreezeCheck(t, root)
	if rc != 0 || len(violations) != 0 {
		t.Fatalf("unmodified tree must have zero violations, got %v", violations)
	}
}

func TestFreezeCheckHelperProcess(t *testing.T) {
	if os.Getenv("AILANG_FREEZE_CHECK_HELPER") != "1" {
		return
	}
	runPromptFreeze([]string{"--check", "--repo", os.Getenv("AILANG_FREEZE_CHECK_REPO")})
}

func newFreezeCheckRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := []byte("base prompt\n")
	writeFreezeCheckFile(t, root, "prompts/v1.0.0.md", base)
	writeFreezeCheckFile(t, root, "cmd/ailang/prompts/v1.0.0.md", base)
	writeFreezeCheckRegistries(t, root, hashBytes(base), hashBytes(base), true)
	runFreezeCheckGit(t, root, "init", "-q")
	runFreezeCheckGit(t, root, "config", "user.email", "freeze-check@example.invalid")
	runFreezeCheckGit(t, root, "config", "user.name", "Freeze Check")
	runFreezeCheckGit(t, root, "add", ".")
	runFreezeCheckGit(t, root, "commit", "-q", "-m", "base")
	runFreezeCheckGit(t, root, "update-ref", "refs/remotes/origin/dev", "HEAD")
	runFreezeCheckGit(t, root, "switch", "-q", "-c", "feature")
	return root
}

func writeFreezeCheckRegistries(t *testing.T, root, sourceHash, mirrorHash string, frozen bool) {
	t.Helper()
	write := func(path, hash string) {
		entry := map[string]any{"file": "prompts/v1.0.0.md", "hash": hash}
		if frozen {
			entry["frozen"] = map[string]any{"at": "2026-08-27", "reason": "banked", "evidence_count": 1, "evidence_example": "eval_results/baselines/one.json"}
		}
		data, err := json.MarshalIndent(map[string]any{"schema_version": "1.1", "versions": map[string]any{freezeCheckVersion: entry}, "active": freezeCheckVersion, "notes": []string{}}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		writeFreezeCheckFile(t, root, path, append(data, '\n'))
	}
	write("prompts/versions.json", sourceHash)
	write("cmd/ailang/prompts/versions.json", mirrorHash)
}

func writeFreezeCheckFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runFreezeCheckGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func runFreezeCheck(t *testing.T, root string) ([]string, int) {
	t.Helper()
	oldScanner := corpusScanner
	corpusScanner = func(string) (map[string]corpusEvidence, error) { return map[string]corpusEvidence{}, nil }
	t.Cleanup(func() { corpusScanner = oldScanner })
	violations, err := checkRegistries(root)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFreezeCheckHelperProcess$")
	cmd.Env = append(os.Environ(), "AILANG_FREEZE_CHECK_HELPER=1", "AILANG_FREEZE_CHECK_REPO="+root)
	err = cmd.Run()
	rc := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		rc = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return violations, rc
}

func hasViolation(violations []string, parts ...string) bool {
	for _, violation := range violations {
		matched := true
		for _, part := range parts {
			matched = matched && strings.Contains(filepath.ToSlash(violation), part)
		}
		if matched {
			return true
		}
	}
	return false
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
