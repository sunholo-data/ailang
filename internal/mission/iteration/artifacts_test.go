package iteration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

func gitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.test", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.test", "GIT_CONFIG_NOSYSTEM=1")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}
func writeTest(t *testing.T, dir, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, path)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, path), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

type artifactFixture struct {
	repo, verify string
	spec         Spec
	stage        Stage
	request      dispatch.Request
	report       *dispatch.Report
	result       StageResult
}

func newArtifactFixture(t *testing.T) artifactFixture {
	t.Helper()
	repo := t.TempDir()
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "remote", "add", "origin", "https://github.com/example/project.git")
	writeTest(t, repo, "docs/result.md", "old\n")
	gitTest(t, repo, "add", ".")
	gitTest(t, repo, "commit", "-qm", "base")
	base := gitTest(t, repo, "rev-parse", "HEAD")
	s := validSpec()
	s.BaseRevision = base
	s.Verification = []Verification{{ID: "read", Argv: []string{"git", "diff", "--check", base}, Cwd: ".", TimeoutSeconds: 5}}
	stage := s.Stages[2]
	req := dispatch.Request{Version: 1, MissionID: s.MissionID, WorkItemID: s.WorkItemID, StageID: stage.ID, AttemptID: "attempt-1", Role: stage.Role, Workspace: repo, InputRevision: base, Instructions: stage.Instructions, Models: []string{"fixture-author"}, TimeoutSeconds: 30, MaxTokens: 1000, MaxCostUSD: 0.1}
	writeTest(t, repo, "docs/result.md", "new\n")
	gitTest(t, repo, "add", ".")
	gitTest(t, repo, "commit", "-qm", "result")
	out := gitTest(t, repo, "rev-parse", "HEAD")
	r := StageResult{Version: 1, RequestDigest: req.Digest(), InputRevision: base, OutputRevision: out, ArtifactPaths: []string{"docs/result.md"}, Outcome: "produced"}
	b, _ := json.Marshal(r)
	writeTest(t, repo, ResultFile, string(b))
	report := &dispatch.Report{Version: 1, AttemptID: req.AttemptID, RequestDigest: req.Digest(), Status: "execution_completed", Attempts: []dispatch.Attempt{{Status: "completed", Candidate: dispatch.Candidate{Model: "fixture-author", Vendor: "fixture", Executor: "fake"}}}}
	return artifactFixture{repo: repo, verify: t.TempDir(), spec: s, stage: stage, request: req, report: report, result: r}
}
func (f artifactFixture) validate() (*Evidence, error) {
	return ValidateArtifacts(context.Background(), f.spec, f.stage, f.request, f.report, f.repo, f.verify)
}
func TestArtifactAcceptance(t *testing.T) {
	f := newArtifactFixture(t)
	e, err := f.validate()
	if err != nil {
		t.Fatal(err)
	}
	if e.OutputRevision != f.result.OutputRevision || len(e.Artifacts) != 1 || len(e.Checks) != 1 || e.Checks[0].ExitCode != 0 || e.ResultSHA256 == "" || e.Digest() == "" || len(e.AuthorModels) != 1 {
		t.Fatalf("incomplete evidence: %+v", e)
	}
	if string(e.ResultBytes) == "" {
		t.Fatal("exact protocol bytes missing")
	}
}
func TestArtifactsRejectBoundaryViolations(t *testing.T) {
	cases := map[string]func(*artifactFixture){
		"wrong_origin": func(f *artifactFixture) {
			gitTest(t, f.repo, "remote", "set-url", "origin", "https://github.com/wrong/repo")
		},
		"dirty":         func(f *artifactFixture) { writeTest(t, f.repo, "docs/result.md", "dirty") },
		"untracked":     func(f *artifactFixture) { writeTest(t, f.repo, "other", "untracked") },
		"wrong_request": func(f *artifactFixture) { f.report.RequestDigest = strings.Repeat("b", 64) },
		"missing_route": func(f *artifactFixture) { f.report.Attempts = nil },
		"foreign_route": func(f *artifactFixture) { f.report.Attempts[0].Candidate.Model = "unapproved" },
		"wrong_input": func(f *artifactFixture) {
			f.result.InputRevision = strings.Repeat("c", 40)
			b, _ := json.Marshal(f.result)
			writeTest(t, f.repo, ResultFile, string(b))
		},
		"missing_artifact": func(f *artifactFixture) { f.stage.RequiredArtifacts = []string{"docs/missing.md"} },
		"protocol_symlink": func(f *artifactFixture) {
			os.Remove(filepath.Join(f.repo, ResultFile))
			os.Symlink("docs/result.md", filepath.Join(f.repo, ResultFile))
		},
		"verification_inside_author": func(f *artifactFixture) { f.verify = filepath.Join(f.repo, "verify") },
		"failed_check": func(f *artifactFixture) {
			f.spec.Verification[0].Argv = []string{"git", "rev-parse", "--verify", "refs/heads/nonexistent"}
		},
		"mutating_check": func(f *artifactFixture) {
			f.spec.Verification[0].Argv = []string{"sh", "-c", "printf tampered > docs/result.md"}
		},
		"mutating_then_restore_check": func(f *artifactFixture) {
			f.spec.Verification = append(f.spec.Verification, Verification{ID: "restore", Argv: []string{"git", "checkout", "--", "docs/result.md"}, Cwd: ".", TimeoutSeconds: 5})
			f.spec.Verification[0].Argv = []string{"sh", "-c", "printf tampered > docs/result.md"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newArtifactFixture(t)
			mutate(&f)
			if _, err := f.validate(); err == nil {
				t.Fatal("invalid artifact accepted")
			}
		})
	}
}
func TestChangedPathsAndReservedProtocol(t *testing.T) {
	for _, name := range []string{"outside", "delete_outside", "symlink", "tracked_protocol", "rename_outside"} {
		t.Run(name, func(t *testing.T) {
			f := newArtifactFixture(t)
			os.Remove(filepath.Join(f.repo, ResultFile))
			switch name {
			case "outside":
				writeTest(t, f.repo, "src/code.go", "bad")
			case "symlink":
				os.Remove(filepath.Join(f.repo, "docs/result.md"))
				if err := os.Symlink("../../outside", filepath.Join(f.repo, "docs/result.md")); err != nil {
					t.Fatal(err)
				}
			case "tracked_protocol":
				writeTest(t, f.repo, ResultFile, "{}")
			case "delete_outside", "rename_outside":
				// Establish an out-of-scope source in the recorded stage base.
				writeTest(t, f.repo, "outside.md", "base")
				gitTest(t, f.repo, "add", ".")
				gitTest(t, f.repo, "commit", "-qm", "base outside")
				f.request.InputRevision = gitTest(t, f.repo, "rev-parse", "HEAD")
				f.result.InputRevision = f.request.InputRevision
				f.report.RequestDigest = f.request.Digest()
				f.result.RequestDigest = f.request.Digest()
				if name == "delete_outside" {
					os.Remove(filepath.Join(f.repo, "outside.md"))
				} else {
					os.Rename(filepath.Join(f.repo, "outside.md"), filepath.Join(f.repo, "docs/moved.md"))
				}
			}
			gitTest(t, f.repo, "add", "-A")
			gitTest(t, f.repo, "commit", "-qm", "invalid candidate")
			f.result.OutputRevision = gitTest(t, f.repo, "rev-parse", "HEAD")
			b, _ := json.Marshal(f.result)
			writeTest(t, f.repo, ResultFile, string(b))
			if _, err := f.validate(); err == nil {
				t.Fatal("invalid changed path accepted")
			}
		})
	}
}
func TestCheckOutputBoundedAndExitPreserved(t *testing.T) {
	f := newArtifactFixture(t)
	f.spec.Verification[0].Argv = []string{"sh", "-c", "head -c 1100000 /dev/zero; exit 7"}
	e, err := f.validate()
	if err == nil || e == nil || len(e.Checks) != 1 {
		t.Fatalf("missing failed receipt: %v %+v", err, e)
	}
	c := e.Checks[0]
	if c.ExitCode != 7 || !c.Truncated || len(c.Output) > 1024*1024 {
		t.Fatalf("bad bounded receipt: exit=%d len=%d truncated=%v", c.ExitCode, len(c.Output), c.Truncated)
	}
}

func TestEvaluationBindsExactCandidate(t *testing.T) {
	f := newArtifactFixture(t)
	f.stage = f.spec.Stages[3]
	f.request.StageID = f.stage.ID
	f.request.Role = "evaluator"
	f.request.InputRevision = f.result.OutputRevision
	f.request.AuthorModels = []string{"prior-author"}
	f.result.InputRevision = f.request.InputRevision
	f.result.Outcome = "pass"
	f.result.Criteria = map[string]CriterionResult{"clear": {Outcome: "pass", Evidence: "docs/result.md"}}
	f.result.RequestDigest = f.request.Digest()
	f.report.RequestDigest = f.request.Digest()
	b, _ := json.Marshal(f.result)
	writeTest(t, f.repo, ResultFile, string(b))
	if _, err := f.validate(); err != nil {
		t.Fatal(err)
	}
	f.result.OutputRevision = f.spec.BaseRevision
	b, _ = json.Marshal(f.result)
	writeTest(t, f.repo, ResultFile, string(b))
	if _, err := f.validate(); err == nil {
		t.Fatal("stale evaluation accepted")
	}
}
func TestCheckTimeoutAndIgnoredOutput(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		f := newArtifactFixture(t)
		f.spec.Verification[0].Argv = []string{"sh", "-c", "sleep 30 & wait"}
		f.spec.Verification[0].TimeoutSeconds = 1
		start := time.Now()
		e, err := f.validate()
		if err == nil || e == nil || len(e.Checks) != 1 || time.Since(start) > 5*time.Second {
			t.Fatalf("timeout not bounded: %v", err)
		}
	})
	t.Run("ignored_build_output", func(t *testing.T) {
		f := newArtifactFixture(t)
		os.Remove(filepath.Join(f.repo, ResultFile))
		writeTest(t, f.repo, "docs/.gitignore", "build/\n")
		gitTest(t, f.repo, "add", ".")
		gitTest(t, f.repo, "commit", "-qm", "ignore builds")
		f.result.OutputRevision = gitTest(t, f.repo, "rev-parse", "HEAD")
		b, _ := json.Marshal(f.result)
		writeTest(t, f.repo, ResultFile, string(b))
		f.spec.Verification[0].Argv = []string{"sh", "-c", "mkdir docs/build; printf generated > docs/build/output"}
		if _, err := f.validate(); err != nil {
			t.Fatal(err)
		}
	})
}
func TestNoChangeCommitRejected(t *testing.T) {
	f := newArtifactFixture(t)
	os.Remove(filepath.Join(f.repo, ResultFile))
	f.request.InputRevision = f.result.OutputRevision
	f.result.InputRevision = f.result.OutputRevision
	gitTest(t, f.repo, "commit", "--allow-empty", "-qm", "empty")
	f.result.OutputRevision = gitTest(t, f.repo, "rev-parse", "HEAD")
	f.result.RequestDigest = f.request.Digest()
	f.report.RequestDigest = f.request.Digest()
	b, _ := json.Marshal(f.result)
	writeTest(t, f.repo, ResultFile, string(b))
	if _, err := f.validate(); err == nil {
		t.Fatal("empty author commit accepted")
	}
}

func TestChangedFrozenAuthorityRejected(t *testing.T) {
	f := newArtifactFixture(t)
	// Bind a frozen authority document in the base, then edit it in the candidate.
	f.spec.Stages[2].AuthorityRefs = []AuthorityRef{{Revision: f.spec.BaseRevision, Path: "docs/result.md", Locator: "old", SHA256: shaBytes([]byte("old\n")), ArtifactDigest: shaBytes([]byte("old\n"))}}
	f.stage = f.spec.Stages[2]
	if _, err := f.validate(); err == nil {
		t.Fatal("candidate changed frozen approval content")
	}
}
func TestUnfrozenStagePolicyRejected(t *testing.T) {
	f := newArtifactFixture(t)
	f.stage.RequiredArtifacts = nil
	if _, err := f.validate(); err == nil {
		t.Fatal("stage changed frozen artifact requirements")
	}
}

func TestDecisionResultRetainedWithoutAcceptance(t *testing.T) {
	f := newArtifactFixture(t)
	f.result.Outcome = "needs_decision"
	f.result.BlockingFindings = []string{"Approve a new dependency"}
	f.result.ArtifactPaths = nil
	body, _ := json.Marshal(f.result)
	writeTest(t, f.repo, ResultFile, string(body))
	// This check must never run for a decision request.
	f.spec.Verification[0].Argv = []string{"sh", "-c", "exit 99"}
	e, err := f.validate()
	if !errors.Is(err, ErrNeedsDecision) || e == nil || !bytes.Equal(e.ResultBytes, body) || e.ResultSHA256 != shaBytes(body) || len(e.Checks) != 0 {
		t.Fatalf("decision not retained: %v %+v", err, e)
	}
	for _, field := range []string{"digest", "input", "head"} {
		t.Run(field, func(t *testing.T) {
			bad := f.result
			switch field {
			case "digest":
				bad.RequestDigest = strings.Repeat("c", 64)
			case "input":
				bad.InputRevision = strings.Repeat("c", 40)
			case "head":
				bad.OutputRevision = bad.InputRevision
			}
			body, _ := json.Marshal(bad)
			writeTest(t, f.repo, ResultFile, string(body))
			_, err := f.validate()
			if err == nil || errors.Is(err, ErrNeedsDecision) {
				t.Fatalf("unbound decision accepted: %v", err)
			}
		})
	}
}
func TestAggregateVerificationOutputFitsAcceptance(t *testing.T) {
	f := newArtifactFixture(t)
	f.spec.Verification = nil
	for _, id := range []string{"one", "two", "three"} {
		f.spec.Verification = append(f.spec.Verification, Verification{ID: id, Argv: []string{"sh", "-c", "head -c 1048576 /dev/zero"}, Cwd: ".", TimeoutSeconds: 5})
	}
	e, err := f.validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Checks) != 3 {
		t.Fatal("checks were skipped")
	}
	total := 0
	for i, c := range e.Checks {
		total += len(c.Output)
		if c.ExitCode != 0 || c.OutputSHA256 != shaBytes([]byte(c.Output)) {
			t.Fatalf("lost exit/hash: %+v", c)
		}
		if i > 0 && !c.Truncated {
			t.Fatal("discarded output must be explicitly marked truncated")
		}
	}
	if total > 1024*1024 {
		t.Fatalf("aggregate output unbounded: %d", total)
	}
	encoded, err := json.Marshal(struct {
		Evidence *Evidence
		Report   *dispatch.Report
	}{Evidence: e, Report: f.report})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) >= 16*1024*1024 {
		t.Fatalf("acceptance exceeds store limit: %d", len(encoded))
	}
}

func TestAggregateTruncationPreservesLaterFailure(t *testing.T) {
	f := newArtifactFixture(t)
	f.spec.Verification = []Verification{
		{ID: "fill", Argv: []string{"sh", "-c", "head -c 1048576 /dev/zero"}, Cwd: ".", TimeoutSeconds: 5},
		{ID: "fail", Argv: []string{"sh", "-c", "printf failure; exit 7"}, Cwd: ".", TimeoutSeconds: 5},
	}
	e, err := f.validate()
	if err == nil || e == nil || len(e.Checks) != 2 {
		t.Fatalf("failed check receipt lost: %v", err)
	}
	c := e.Checks[1]
	if c.ExitCode != 7 || !c.Truncated || c.Output != "" || c.OutputSHA256 != shaBytes(nil) {
		t.Fatalf("failed receipt lost exit/hash/truncation: %+v", c)
	}
}
