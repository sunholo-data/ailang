package iteration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReviewPacketCandidateAndBounds(t *testing.T) {
	f := newArtifactFixture(t)
	s := &Service{RepositoryPath: f.repo, WorkspaceRoot: t.TempDir()}
	packet, path, err := s.reviewPacket(context.Background(), f.spec, f.spec.Stages[3], f.result.OutputRevision, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(packet, f.result.OutputRevision) || !strings.Contains(packet, "+new") {
		t.Fatal("candidate diff missing")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("packet stat: %v", err)
	}
	// Windows has no unix permission bits: a file created 0400 reports 0444, and there is no
	// mode that expresses "owner read-only" there. Assert the portable half — that the packet
	// is not writable — rather than a number the platform cannot represent.
	if info.Mode().Perm()&0222 != 0 {
		t.Fatalf("packet is writable: %v", info.Mode().Perm())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0400 {
		t.Fatalf("packet not read only: %v", info.Mode().Perm())
	}
	if within(f.repo, path) {
		t.Fatal("packet in author root")
	}
	writeTest(t, f.repo, "docs/result.md", strings.Repeat("large\n", 50000))
	gitTest(t, f.repo, "add", "docs/result.md")
	gitTest(t, f.repo, "commit", "-qm", "large")
	candidate := gitTest(t, f.repo, "rev-parse", "HEAD")
	packet, _, err = s.reviewPacket(context.Background(), f.spec, f.spec.Stages[3], candidate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet) > MaxReviewPacketBytes || !strings.Contains(packet, "INCOMPLETE") {
		t.Fatal("unbounded or silently truncated diff")
	}
}

func TestReviewPacketRejectsSymlinkPlacement(t *testing.T) {
	f := newArtifactFixture(t)
	root := t.TempDir()
	dir := filepath.Join(root, f.spec.MissionID, f.spec.WorkItemID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f.repo, filepath.Join(dir, "review-packets")); err != nil {
		t.Fatal(err)
	}
	s := &Service{RepositoryPath: f.repo, WorkspaceRoot: root}
	if _, _, err := s.reviewPacket(context.Background(), f.spec, f.spec.Stages[3], f.result.OutputRevision, nil); err == nil {
		t.Fatal("accepted redirected packet")
	}
}

func TestReviewRequestReplayDoesNotRebuildPacket(t *testing.T) {
	s, spec, _ := runtimeFixture(t)
	item, err := s.Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	stage := spec.Stages[1]
	first, err := s.request(context.Background(), spec, stage, 1, item)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Instructions, "Review packet SHA256:") {
		t.Fatal("packet not bound")
	}
	// Recovery must use the immutable saved request even if the materialized copy
	// is unavailable; it includes exact packet bytes, independent of this path.
	s.RepositoryPath = filepath.Join(t.TempDir(), "missing")
	again, err := s.request(context.Background(), spec, stage, 1, item)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != again.Digest() {
		t.Fatal("replay regenerated request")
	}
}

func TestOptionalReviewBaseline(t *testing.T) {
	spec := validSpec()
	body, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "review_base_revision") {
		t.Fatal("legacy digest changed")
	}
	decoded, err := Decode(strings.NewReader(string(body)))
	if err != nil || decoded.Digest() != spec.Digest() {
		t.Fatalf("legacy decode: %v", err)
	}
	spec.ReviewBaseRevision = "invalid"
	if spec.Validate() == nil {
		t.Fatal("invalid revision admitted")
	}
	spec.ReviewBaseRevision = spec.BaseRevision
	body, _ = json.Marshal(spec)
	if _, err = Decode(strings.NewReader(string(body))); err == nil {
		t.Fatal("author workflow admitted baseline override")
	}
}

func TestReviewPacketCannotHideAcceptedAuthorDiff(t *testing.T) {
	f := newArtifactFixture(t)
	s := &Service{RepositoryPath: f.repo, WorkspaceRoot: t.TempDir()}
	f.spec.ReviewBaseRevision = f.result.OutputRevision
	ev := &Evidence{Result: StageResult{Outcome: "produced", InputRevision: f.spec.BaseRevision}}
	if _, _, err := s.reviewPacket(context.Background(), f.spec, f.spec.Stages[3], f.result.OutputRevision, []*Evidence{ev}); err == nil {
		t.Fatal("declared baseline hid accepted author changes")
	}
}
