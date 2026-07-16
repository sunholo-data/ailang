package eval_harness

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTempGitRepo creates a temp git repo with one committed file so `git diff`
// has a HEAD to diff against, and returns the worktree path.
func newTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
			"GIT_CONFIG_NOSYSTEM=1",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	// Seed a committed file so the repo has a HEAD.
	writeFile(t, dir, "seed.txt", "seed\n")
	run("add", "seed.txt")
	run("commit", "-q", "-m", "seed")
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitAdd(t *testing.T, dir string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestBuildDiffBundle_IncludesDiffAndFiles(t *testing.T) {
	dir := newTempGitRepo(t)
	// Two committed .go files, then modify both (tracked changes).
	writeFile(t, dir, "alpha.go", "package a\n\nvar Alpha = 1\n")
	writeFile(t, dir, "beta.go", "package b\n\nvar Beta = 2\n")
	gitAdd(t, dir, "alpha.go", "beta.go")
	// commit them so they're tracked
	commit(t, dir, "add go files")
	// modify both
	writeFile(t, dir, "alpha.go", "package a\n\nvar Alpha = 100\n")
	writeFile(t, dir, "beta.go", "package b\n\nvar Beta = 200\n")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Unified diff present for both.
	if !strings.Contains(b.Text, "alpha.go") || !strings.Contains(b.Text, "beta.go") {
		t.Fatalf("bundle missing file names:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "diff --git") {
		t.Fatalf("bundle missing unified diff header:\n%s", b.Text)
	}
	// Full file contents present (the NEW values).
	if !strings.Contains(b.Text, "var Alpha = 100") {
		t.Errorf("bundle missing alpha full contents:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "var Beta = 200") {
		t.Errorf("bundle missing beta full contents:\n%s", b.Text)
	}
	if b.Truncated {
		t.Errorf("unexpected truncation: dropped=%v", b.DroppedFiles)
	}
}

func TestBuildDiffBundle_IncludesUntrackedNewFiles(t *testing.T) {
	dir := newTempGitRepo(t)
	// A brand-new untracked file — `git diff` alone would MISS this entirely.
	writeFile(t, dir, "newthing.go", "package n\n\nfunc Fresh() int { return 42 }\n")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Text, "+++ NEW FILE: newthing.go") {
		t.Fatalf("bundle missing NEW FILE header (git diff would have missed it):\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "func Fresh() int { return 42 }") {
		t.Fatalf("bundle missing untracked file contents:\n%s", b.Text)
	}
}

func TestBuildDiffBundle_DropsBinaryAndGenerated(t *testing.T) {
	dir := newTempGitRepo(t)
	// Generated file (*.pb.go) with real text content.
	writeFile(t, dir, "api.pb.go", "package api\n\n// generated\nvar X = 1\n")
	// Binary file: contains a NUL byte in the first bytes.
	writeFile(t, dir, "blob.bin", "abc\x00def-binary-content\n")
	gitAdd(t, dir, "api.pb.go", "blob.bin")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Truncated {
		t.Errorf("expected Truncated=true when dropping binary/generated")
	}
	// Both listed in DroppedFiles.
	joined := strings.Join(b.DroppedFiles, "\n")
	if !strings.Contains(joined, "api.pb.go") || !strings.Contains(joined, "blob.bin") {
		t.Errorf("DroppedFiles missing entries: %v", b.DroppedFiles)
	}
	// LOUD marker present for each.
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped api.pb.go") {
		t.Errorf("missing LOUD marker for generated file:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped blob.bin") {
		t.Errorf("missing LOUD marker for binary file:\n%s", b.Text)
	}
	// Their CONTENT must NOT appear as an inlined full-file body.
	if strings.Contains(b.Text, "----- FULL FILE: api.pb.go") {
		t.Errorf("generated file was inlined but should be dropped:\n%s", b.Text)
	}
	if strings.Contains(b.Text, "binary-content") {
		t.Errorf("binary content leaked into bundle:\n%s", b.Text)
	}
}

func TestBuildDiffBundle_TruncatesOverCeiling(t *testing.T) {
	dir := newTempGitRepo(t)
	// One large untracked file that blows a tiny ceiling.
	large := "package big\n\nvar Blob = \"" + strings.Repeat("x", 20*1024) + "\"\n"
	writeFile(t, dir, "big.go", large)
	// A small tracked change too, to prove the diff stays.
	writeFile(t, dir, "seed.txt", "seed-modified\n")

	b, err := BuildDiffBundle(dir, BundleOptions{MaxBytes: 2 * 1024})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Truncated {
		t.Fatalf("expected Truncated=true over ceiling")
	}
	joined := strings.Join(b.DroppedFiles, "\n")
	if !strings.Contains(joined, "big.go") {
		t.Errorf("big.go should be dropped: %v", b.DroppedFiles)
	}
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped big.go") {
		t.Errorf("missing over-ceiling marker:\n%s", firstN(b.Text, 800))
	}
	// The unified diff of the tracked change is STILL present (never dropped).
	if !strings.Contains(b.Text, "seed-modified") {
		t.Errorf("tracked diff was dropped — it must never be:\n%s", firstN(b.Text, 800))
	}
	// The dropped new file KEEPS its NEW FILE header (never silently invisible).
	if !strings.Contains(b.Text, "+++ NEW FILE: big.go") {
		t.Errorf("dropped new file lost its NEW FILE header:\n%s", firstN(b.Text, 800))
	}
	// The huge body must NOT be inlined.
	if strings.Contains(b.Text, strings.Repeat("x", 20*1024)) {
		t.Errorf("over-ceiling body was inlined")
	}
}

func TestBuildDiffBundle_Deterministic(t *testing.T) {
	dir := newTempGitRepo(t)
	writeFile(t, dir, "zeta.go", "package z\nvar Z = 1\n")
	writeFile(t, dir, "alpha.go", "package a\nvar A = 1\n")
	writeFile(t, dir, "mid.go", "package m\nvar M = 1\n")
	// Mixed: one tracked-modified, others untracked.
	writeFile(t, dir, "seed.txt", "changed\n")

	b1, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b1.Text != b2.Text {
		t.Errorf("BuildDiffBundle not deterministic:\n--- b1 ---\n%s\n--- b2 ---\n%s", b1.Text, b2.Text)
	}
}

// --- helpers ---

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-q", "-m", msg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ============================================================================
// M2 — directive + verdict parse/validate tests
// ============================================================================

func TestBuildEvaluatorDirective_ReasoningOnly(t *testing.T) {
	bundle := Bundle{Text: "=== SPRINT DIFF BUNDLE ===\nSOME-DIFF-CONTENT\n"}
	dir := BuildEvaluatorDirective("DESIGN-DOC-BODY", "SPRINT-PLAN-BODY", "CRIT-A; CRIT-B", bundle)

	// reasoning-only instruction present
	if !strings.Contains(dir, "Reasoning-only sprint evaluation") {
		t.Errorf("directive missing reasoning-only instruction:\n%s", dir)
	}
	// fenced-json verdict instruction present
	if !strings.Contains(dir, "fenced ```json block") {
		t.Errorf("directive missing fenced-json verdict instruction")
	}
	// bundle text present
	if !strings.Contains(dir, "SOME-DIFF-CONTENT") {
		t.Errorf("directive missing bundle text")
	}
	// design doc + plan + criteria present
	if !strings.Contains(dir, "DESIGN-DOC-BODY") || !strings.Contains(dir, "SPRINT-PLAN-BODY") || !strings.Contains(dir, "CRIT-A") {
		t.Errorf("directive missing design/plan/criteria")
	}
	// non-truncated bundle must NOT add the truncation note
	if strings.Contains(dir, "note in blockers") || strings.Contains(dir, "marked TRUNCATED") {
		t.Errorf("non-truncated bundle should not add truncation note")
	}

	// truncated bundle DOES add the note
	truncBundle := Bundle{Text: "x", Truncated: true, DroppedFiles: []string{"big.go (over-ceiling)"}}
	dir2 := BuildEvaluatorDirective("d", "p", "c", truncBundle)
	if !strings.Contains(dir2, "marked TRUNCATED") || !strings.Contains(dir2, "unseen files") {
		t.Errorf("truncated bundle missing the note-unseen-files line:\n%s", dir2)
	}
}

func TestParseGeminiVerdict_ExtractsFencedJSON(t *testing.T) {
	resp := "Let me reason about the diff.\n\n" +
		"Scratch verdict (ignore): ```json\n{\"score\": 10, \"pass\": false, \"blockers\": [\"draft\"]}\n```\n\n" +
		"Final answer:\n```json\n{\"score\": 85, \"pass\": true, \"blockers\": []}\n```\n"
	v, err := ParseGeminiVerdict(resp, DegradationInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if v.Score != 85 || !v.Pass {
		t.Errorf("parsed wrong (last) verdict: %+v", v)
	}
	if len(v.Blockers) != 0 {
		t.Errorf("expected no blockers, got %v", v.Blockers)
	}
	if v.VerificationDegraded {
		t.Errorf("clean bundle should not be degraded")
	}
}

func TestValidateGeminiVerdict_RejectsMalformed(t *testing.T) {
	// score out of range
	if err := ValidateGeminiVerdict(&GeminiVerdict{Score: 101}); err == nil {
		t.Errorf("expected error for score>100")
	}
	if err := ValidateGeminiVerdict(&GeminiVerdict{Score: -1}); err == nil {
		t.Errorf("expected error for score<0")
	}
	// blockers non-empty with pass==true
	if err := ValidateGeminiVerdict(&GeminiVerdict{Score: 90, Pass: true, Blockers: []string{"x"}}); err == nil {
		t.Errorf("expected error for pass==true with blockers")
	}
	// degraded with empty reason
	if err := ValidateGeminiVerdict(&GeminiVerdict{Score: 50, VerificationDegraded: true}); err == nil {
		t.Errorf("expected error for degraded with empty reason")
	}
	// valid case: no error
	if err := ValidateGeminiVerdict(&GeminiVerdict{Score: 80, Pass: true}); err != nil {
		t.Errorf("unexpected error on valid verdict: %v", err)
	}

	// ParseGeminiVerdict hard-errors on missing fence / non-JSON
	if _, err := ParseGeminiVerdict("no fences at all here", DegradationInfo{}); err == nil {
		t.Errorf("expected error for missing fence")
	}
	if _, err := ParseGeminiVerdict("```json\nnot json at all\n```", DegradationInfo{}); err == nil {
		t.Errorf("expected error for non-JSON fenced body")
	}
	// ParseGeminiVerdict must NOT coerce a malformed verdict to a pass
	if _, err := ParseGeminiVerdict("```json\n{\"score\": 999, \"pass\": true, \"blockers\": []}\n```", DegradationInfo{}); err == nil {
		t.Errorf("expected error for out-of-range score (never coerced pass)")
	}
}

func TestLastFencedBlock_UnchangedForExtractOut(t *testing.T) {
	// The exact extract-out cases from managed_agents_bridge_test.go must yield
	// identical results through the exported wrapper (regression guard for the
	// Conflict-Surface seam 4 sharing).
	cases := []string{
		"Here's my solution:\n\n```ailang\nmodule benchmark/solution\nexport func main() -> () = ()\n```\n\nDone.",
		"First attempt:\n```\nbroken\n```\n\nFinal:\n```ailang\nmodule benchmark/solution\nexport func main() -> () = ()\n```\n",
		"no fences here",
		"```\njust text\n```",
	}
	for i, c := range cases {
		if got, want := LastFencedBlock(c), lastFencedBlock(c); got != want {
			t.Errorf("case %d: LastFencedBlock diverged from lastFencedBlock:\n got=%q\nwant=%q", i, got, want)
		}
	}
}

// ============================================================================
// M3 — RunGeminiEvaluator caller-seam tests (stub runner only, no live Vertex)
// ============================================================================

func TestRunGeminiEvaluator_StubHappyPath(t *testing.T) {
	dir := newTempGitRepo(t)
	writeFile(t, dir, "feature.go", "package f\nvar F = 1\n")

	stub := func(_ context.Context, directive, systemPrompt string) (EvalRunnerOutput, error) {
		// Sanity: the directive must actually carry the bundle + reasoning note.
		if !strings.Contains(directive, "Reasoning-only sprint evaluation") {
			t.Errorf("stub: directive missing reasoning-only instruction")
		}
		if !strings.Contains(directive, "+++ NEW FILE: feature.go") {
			t.Errorf("stub: directive missing the new file")
		}
		return EvalRunnerOutput{
			Success: true,
			Output:  "Looks good.\n```json\n{\"score\": 88, \"pass\": true, \"blockers\": []}\n```\n",
		}, nil
	}

	v, err := RunGeminiEvaluator(context.Background(), dir, "design", "plan", EvalOptions{Runner: stub})
	if err != nil {
		t.Fatal(err)
	}
	if v.Score != 88 || !v.Pass {
		t.Errorf("wrong verdict: %+v", v)
	}
	if v.VerificationDegraded {
		t.Errorf("clean run should not be degraded: reason=%q", v.DegradedReason)
	}
}

func TestRunGeminiEvaluator_TruncationStampsDegraded(t *testing.T) {
	dir := newTempGitRepo(t)
	// Large new file blows a tiny ceiling → bundle Truncated.
	writeFile(t, dir, "huge.go", "package h\nvar H = \""+strings.Repeat("y", 20*1024)+"\"\n")

	// The stub LIES with pass:true — the caller must override to degraded.
	stub := func(_ context.Context, directive, systemPrompt string) (EvalRunnerOutput, error) {
		return EvalRunnerOutput{
			Success: true,
			Output:  "```json\n{\"score\": 95, \"pass\": true, \"blockers\": []}\n```",
		}, nil
	}

	v, err := RunGeminiEvaluator(context.Background(), dir, "design", "plan", EvalOptions{
		Runner: stub,
		Bundle: BundleOptions{MaxBytes: 2 * 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !v.VerificationDegraded {
		t.Fatalf("expected VerificationDegraded==true under truncation (stub said pass:true)")
	}
	if strings.TrimSpace(v.DegradedReason) == "" {
		t.Errorf("degraded verdict must carry a non-empty reason")
	}
	if !strings.Contains(v.DegradedReason, "huge.go") {
		t.Errorf("degraded reason should name the dropped file: %q", v.DegradedReason)
	}
}

func TestRunGeminiEvaluator_BackendErrorIsDegradedNotPass(t *testing.T) {
	dir := newTempGitRepo(t)
	writeFile(t, dir, "x.go", "package x\nvar X = 1\n")

	// Case 1: runner returns Success==false with error text.
	stubFail := func(_ context.Context, directive, systemPrompt string) (EvalRunnerOutput, error) {
		return EvalRunnerOutput{Success: false, Error: "vertex 503 unavailable"}, nil
	}
	v, err := RunGeminiEvaluator(context.Background(), dir, "design", "plan", EvalOptions{Runner: stubFail})
	if err != nil {
		t.Fatal(err)
	}
	if v.Pass {
		t.Errorf("backend failure must NOT be a pass: %+v", v)
	}
	if !v.VerificationDegraded || !strings.Contains(v.DegradedReason, "vertex 503") {
		t.Errorf("backend error not surfaced in degraded reason: %+v", v)
	}

	// Case 2: runner returns a hard error.
	stubErr := func(_ context.Context, directive, systemPrompt string) (EvalRunnerOutput, error) {
		return EvalRunnerOutput{}, errStub("network down")
	}
	v2, err := RunGeminiEvaluator(context.Background(), dir, "design", "plan", EvalOptions{Runner: stubErr})
	if err != nil {
		t.Fatal(err)
	}
	if v2.Pass || !v2.VerificationDegraded {
		t.Errorf("runner error must be degraded non-pass: %+v", v2)
	}
	if !strings.Contains(v2.DegradedReason, "network down") {
		t.Errorf("runner error text not surfaced: %q", v2.DegradedReason)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
