package main

import (
	"github.com/sunholo-data/ailang/internal/mission/comms"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// M3 of M-MISSION-COMMS-P1.
//
// The load-bearing property is --dry-run making ZERO network calls. That is what
// lets the later shell cutover be rehearsed against the live bookkeeping thread
// without posting to it — and the cutover is the risky half, because three live
// loops run pinned driver copies that re-sync from origin/dev at every fire.

type failingPoster struct{ t *testing.T }

func (p failingPoster) AddComment(repo string, number int, body string) error {
	p.t.Fatalf("network call made during --dry-run: AddComment(%q, %d, %d chars)", repo, number, len(body))
	return nil
}

type recordingPoster struct {
	repo   string
	number int
	body   string
	calls  int
}

func (p *recordingPoster) AddComment(repo string, number int, body string) error {
	p.calls++
	p.repo, p.number, p.body = repo, number, body
	return nil
}

func withMissionState(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mission-v1-gh-issue"), []byte("972\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STATE_DIR", dir)
	t.Setenv("MISSION_GH_ISSUE", "")
	return dir
}

func bodyFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissionReport_DryRunMakesNoNetworkCall(t *testing.T) {
	withMissionState(t)
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	// If --dry-run constructs a poster at all, this fails the test.
	newMissionPoster = func() (missionPoster, error) {
		t.Fatal("--dry-run constructed a poster — it must make no network call and need no gh auth")
		return failingPoster{t}, nil
	}

	rc := runMissionReport([]string{"--mission", "v1", "--body-file", bodyFile(t, "iter 325 landed"), "--dry-run"})
	if rc != 0 {
		t.Fatalf("dry-run rc = %d, want 0", rc)
	}
}

func TestMissionReport_PostsResolvedIdentity(t *testing.T) {
	withMissionState(t)
	rec := &recordingPoster{}
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) { return rec, nil }

	rc := runMissionReport([]string{"--mission", "v1", "--body-file", bodyFile(t, "iter 325 landed\n")})
	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if rec.calls != 1 {
		t.Fatalf("AddComment called %d times, want 1", rec.calls)
	}
	if rec.repo != "sunholo-data/ailang" || rec.number != 972 {
		t.Fatalf("posted to %s#%d, want sunholo-data/ailang#972", rec.repo, rec.number)
	}
	if rec.body != "iter 325 landed" {
		t.Fatalf("body = %q, want trailing newline trimmed", rec.body)
	}
}

func TestMissionReport_UnknownMissionFailsBeforeAnyPoster(t *testing.T) {
	withMissionState(t)
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) {
		t.Fatal("a poster was constructed for an unknown mission — identity must resolve first")
		return nil, nil
	}
	if rc := runMissionReport([]string{"--mission", "nope", "--body-file", bodyFile(t, "x")}); rc == 0 {
		t.Fatal("unknown mission returned rc=0 — it must fail loudly")
	}
}

func TestMissionReport_EmptyBodyRefused(t *testing.T) {
	withMissionState(t)
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) {
		t.Fatal("attempted to post an empty comment")
		return nil, nil
	}
	if rc := runMissionReport([]string{"--mission", "v1", "--body-file", bodyFile(t, "   \n\n")}); rc == 0 {
		t.Fatal("empty body returned rc=0 — an empty comment is noise on a thread this doc exists to quieten")
	}
}

func TestMissionReport_RequiredFlags(t *testing.T) {
	withMissionState(t)
	if rc := runMissionReport([]string{"--body-file", bodyFile(t, "x")}); rc == 0 {
		t.Error("missing --mission returned rc=0")
	}
	if rc := runMissionReport([]string{"--mission", "v1"}); rc == 0 {
		t.Error("missing --body-file returned rc=0")
	}
}

func TestMissionReport_PostFailureIsLoud(t *testing.T) {
	withMissionState(t)
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) { return failOnPost{}, nil }
	if rc := runMissionReport([]string{"--mission", "v1", "--body-file", bodyFile(t, "body")}); rc == 0 {
		t.Fatal("a failed post returned rc=0 — Critical Principle 2: fail loudly")
	}
}

type failOnPost struct{}

func (failOnPost) AddComment(string, int, string) error { return os.ErrPermission }

func TestMissionReportSharedDispatcher(t *testing.T) {
	withMissionState(t)
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) { t.Fatal("report help/dry-run constructed a poster"); return nil, nil }
	if err := missionCommand([]string{"report", "--mission", "v1", "--body-file", bodyFile(t, "dispatcher smoke"), "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if err := missionCommand([]string{"report", "--help"}); err != nil {
		t.Fatal(err)
	}
	if code := missionErrorExitCode(missionCommand([]string{"report"})); code != 2 {
		t.Fatalf("invalid report exit = %d, want 2", code)
	}
	if code := missionErrorExitCode(missionCommand([]string{"status"})); code != 2 {
		t.Fatalf("existing iteration dispatch exit = %d, want 2", code)
	}
}

func TestMissionReportSuppliedBodyUsesSharedPayloadCap(t *testing.T) {
	withMissionState(t)
	rec := &recordingPoster{}
	orig := newMissionPoster
	t.Cleanup(func() { newMissionPoster = orig })
	newMissionPoster = func() (missionPoster, error) { return rec, nil }
	err := missionCommand([]string{"report", "--mission", "v1", "--body-file", bodyFile(t, strings.Repeat("本", 200))})
	if err != nil {
		t.Fatal(err)
	}
	if rec.calls != 1 || len(rec.body) > comms.MaxReportChars || !utf8.ValidString(rec.body) || !strings.HasSuffix(rec.body, "…[truncated]") {
		t.Fatalf("unbounded or invalid report: calls=%d bytes=%d body=%q", rec.calls, len(rec.body), rec.body)
	}
}
