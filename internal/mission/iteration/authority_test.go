package iteration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func prerequisiteFixture(t *testing.T) (string, Spec) {
	t.Helper()
	repo := t.TempDir()
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "remote", "add", "origin", "git@github.com:example/project.git")
	writeTest(t, repo, "docs/plan.md", "approved artifact\n")
	writeTest(t, repo, "docs/approval.md", "decision-1: approved\n")
	gitTest(t, repo, "add", ".")
	gitTest(t, repo, "commit", "-qm", "approval")
	s := validSpec()
	s.BaseRevision = gitTest(t, repo, "rev-parse", "HEAD")
	artifactHash := shaBytes([]byte("approved artifact\n"))
	approvalHash := shaBytes([]byte("decision-1: approved\n"))
	a := AuthorityRef{Revision: s.BaseRevision, Path: "docs/approval.md", Locator: "decision-1", SHA256: approvalHash, ArtifactDigest: artifactHash}
	for _, role := range []string{"designer", "planner"} {
		s.Prerequisites = append(s.Prerequisites, Prerequisite{Role: role, Artifact: ArtifactRef{Commit: s.BaseRevision, Path: "docs/plan.md", SHA256: artifactHash}, AuthorityRefs: []AuthorityRef{a}, AuthorModels: []string{"author"}})
	}
	s.Stages = s.Stages[2:]
	return repo, s
}
func TestFrozenPrerequisiteAuthority(t *testing.T) {
	repo, s := prerequisiteFixture(t)
	if err := VerifyPrerequisites(context.Background(), repo, s); err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, repo, "status", "--porcelain")
	if err := VerifyPrerequisites(context.Background(), repo, s); err != nil {
		t.Fatal(err)
	}
	if after := gitTest(t, repo, "status", "--porcelain"); after != before {
		t.Fatal("read-only authority check mutated repository")
	}
	for name, mutate := range map[string]func(*Spec){"content_hash": func(s *Spec) { s.Prerequisites[0].AuthorityRefs[0].SHA256 = s.Prerequisites[0].Artifact.SHA256 }, "locator": func(s *Spec) { s.Prerequisites[0].AuthorityRefs[0].Locator = "missing" }, "artifact_hash": func(s *Spec) { s.Prerequisites[0].Artifact.SHA256 = s.Prerequisites[0].AuthorityRefs[0].SHA256 }, "binding": func(s *Spec) {
		s.Prerequisites[0].AuthorityRefs[0].ArtifactDigest = s.Prerequisites[0].AuthorityRefs[0].SHA256
	}} {
		t.Run(name, func(t *testing.T) {
			repo, s := prerequisiteFixture(t)
			mutate(&s)
			if err := VerifyPrerequisites(context.Background(), repo, s); err == nil {
				t.Fatal("tampered prerequisite accepted")
			}
		})
	}
}
func TestPrerequisiteSymlinkRejected(t *testing.T) {
	repo, s := prerequisiteFixture(t)
	os.Remove(filepath.Join(repo, "docs/plan.md"))
	if err := os.Symlink("approval.md", filepath.Join(repo, "docs/plan.md")); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", ".")
	gitTest(t, repo, "commit", "-qm", "symlink")
	s.BaseRevision = gitTest(t, repo, "rev-parse", "HEAD")
	s.Prerequisites[0].Artifact.Commit = s.BaseRevision
	if err := VerifyPrerequisites(context.Background(), repo, s); err == nil {
		t.Fatal("symlink prerequisite accepted")
	}
}
