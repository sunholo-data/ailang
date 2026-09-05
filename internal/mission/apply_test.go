package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeLaunchCtl models launchd well enough to exercise every bounded wait — including
// the ones that only fire when launchd misbehaves, which cannot be provoked on a real
// domain without breaking the fleet.
type fakeLaunchCtl struct {
	loaded        atomic.Bool
	bootoutHangs  bool // bootout "succeeds" but the job never disappears
	bootstrapHang time.Duration
	bootstrapErr  error
	neverRuns     bool // job loads but never reports a running state
	bootstraps    atomic.Int32
}

func (f *fakeLaunchCtl) Bootout(string) error {
	if !f.bootoutHangs {
		f.loaded.Store(false)
	}
	return nil
}

func (f *fakeLaunchCtl) Bootstrap(string) error {
	if f.bootstrapHang > 0 {
		time.Sleep(f.bootstrapHang)
	}
	if f.bootstrapErr != nil {
		return f.bootstrapErr
	}
	f.bootstraps.Add(1)
	f.loaded.Store(true)
	return nil
}

func (f *fakeLaunchCtl) Print(string) (string, error) {
	if !f.loaded.Load() {
		return "", errors.New("could not find service")
	}
	if f.neverRuns {
		return "state = not running\n", nil
	}
	return "state = running\npid = 123\n", nil
}

// applyFixture stages a mission ready to apply.
func applyFixture(t *testing.T, generated bool) (*Mission, Paths, *fakeLaunchCtl) {
	t.Helper()
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, env, "", true)
	if generated {
		// Mark the installed plist as one of ours, so no adoption is needed.
		body, _ := os.ReadFile(f.p.PlistPath(m))
		_ = os.WriteFile(f.p.PlistPath(m), append([]byte("<!-- "+generatedMarker+" alpha` -->\n"), body...), 0o600)
	}
	if _, err := RenderStaged(m, f.p.EnvPath(m.Name), f.p.PlistPath(m)); err != nil {
		t.Fatalf("RenderStaged: %v", err)
	}
	lc := &fakeLaunchCtl{}
	lc.loaded.Store(true)
	return m, f.p, lc
}

func TestApply_PromotesBothAndReloads(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	res, err := Apply(m, p, lc, ApplyOpts{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Promoted) != 2 {
		t.Errorf("expected 2 promotions, got %v", res.Promoted)
	}
	if !res.Reloaded {
		t.Error("expected a reload")
	}
	if lc.bootstraps.Load() != 1 {
		t.Errorf("bootstraps = %d, want 1", lc.bootstraps.Load())
	}
	// Staged files are consumed by the rename, not left behind to be re-applied.
	for _, s := range []string{p.EnvPath(m.Name) + StagedSuffix, p.PlistPath(m) + StagedSuffix} {
		if _, err := os.Stat(s); !os.IsNotExist(err) {
			t.Errorf("staged file %s survived the promotion", s)
		}
	}
}

// HD-4(b). Reloading kills the running iteration, so a busy mission is refused.
func TestApply_RefusesWhileMidIteration_ForceOverrides(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	pidDir := filepath.Join(p.Home, ".ailang", "state")
	if err := os.MkdirAll(pidDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Our own pid is unquestionably alive.
	if err := os.WriteFile(filepath.Join(pidDir, "mission-alpha.pid"), []byte(fmt.Sprint(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Apply(m, p, lc, ApplyOpts{})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
	if _, statErr := os.Stat(p.EnvPath(m.Name) + StagedSuffix); statErr != nil {
		t.Error("a refusal must leave the staged files in place for a later apply")
	}

	res, err := Apply(m, p, lc, ApplyOpts{Force: true})
	if err != nil {
		t.Fatalf("--force must override: %v", err)
	}
	if len(res.Notes) == 0 || !strings.Contains(strings.Join(res.Notes, " "), "killed") {
		t.Errorf("--force must say plainly that the running iteration dies; notes=%v", res.Notes)
	}
}

// HD-3(c). A plist we did not generate may hold anything — v1's carries ~40 lines of
// ratified cadence rationale — so the first adoption is one deliberate acknowledgement.
func TestApply_RefusesFirstAdoptionWithoutFlag(t *testing.T) {
	m, p, lc := applyFixture(t, false) // installed plist has no generated marker

	_, err := Apply(m, p, lc, ApplyOpts{})
	if !errors.Is(err, ErrNeedsAdopt) {
		t.Fatalf("expected ErrNeedsAdopt, got %v", err)
	}
	if !strings.Contains(err.Error(), "missions/alpha.toml") {
		t.Errorf("the refusal must say where to put anything worth keeping; got: %v", err)
	}
	if lc.bootstraps.Load() != 0 {
		t.Error("a refused apply must not touch launchd")
	}

	if _, err := Apply(m, p, lc, ApplyOpts{Adopt: true}); err != nil {
		t.Fatalf("--adopt must proceed: %v", err)
	}
}

// Second and later applies need no acknowledgement: the marker is now present, so the
// scheme converges to plain generation exactly as HD-3(c) intended.
func TestApply_ConvergesAfterFirstAdoption(t *testing.T) {
	m, p, lc := applyFixture(t, false)
	if _, err := Apply(m, p, lc, ApplyOpts{Adopt: true}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if _, err := RenderStaged(m, p.EnvPath(m.Name), p.PlistPath(m)); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(m, p, lc, ApplyOpts{}); err != nil {
		t.Fatalf("a second apply must need no --adopt: %v", err)
	}
}

// The promotion ORDER is load-bearing and was untested until a mutation survived.
// The plist is inert until launchd reloads it; the env file is live on the very next
// fire. So the plist goes first and the env last — if anything fails between them, the
// fleet is left with old config and an unloaded new plist, not new config running
// under an old schedule.
func TestApply_PromotesPlistBeforeEnv(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	res, err := Apply(m, p, lc, ApplyOpts{SkipReload: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Promoted) != 2 {
		t.Fatalf("expected 2 promotions, got %v", res.Promoted)
	}
	if res.Promoted[0] != p.PlistPath(m) {
		t.Errorf("plist must be promoted FIRST (it is inert until reload); order was %v", res.Promoted)
	}
	if res.Promoted[1] != p.EnvPath(m.Name) {
		t.Errorf("env must be promoted LAST (it is live on the next fire); order was %v", res.Promoted)
	}
}

func TestApply_RefusesWhenNothingStaged(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, env, "", true)
	_, err := Apply(m, f.p, &fakeLaunchCtl{}, ApplyOpts{})
	if !errors.Is(err, ErrNotStaged) {
		t.Fatalf("expected ErrNotStaged, got %v", err)
	}
	if !strings.Contains(err.Error(), "mission install") {
		t.Error("the error should name the command that fixes it")
	}
}

// ── the bounded waits ────────────────────────────────────────────────────────
// Each of these is a launchd misbehaviour that cannot be provoked on a real domain
// without breaking the fleet, which is precisely why launchctl is injected.

func TestApply_BootoutThatNeverSettlesIsATypedTimeout(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	lc.bootoutHangs = true
	start := time.Now()
	_, err := Apply(m, p, lc, ApplyOpts{})
	if !errors.Is(err, ErrBootoutTimeout) {
		t.Fatalf("expected ErrBootoutTimeout, got %v", err)
	}
	if el := time.Since(start); el > bootoutDeadline+3*time.Second {
		t.Errorf("bootout wait was not bounded: %v", el)
	}
}

func TestApply_BootstrapFailureRetainsNothingStagedButReportsClearly(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	lc.bootstrapErr = errors.New("Bootstrap failed: 5: Input/output error")
	_, err := Apply(m, p, lc, ApplyOpts{})
	if err == nil {
		t.Fatal("a failed bootstrap must be reported")
	}
	if !strings.Contains(err.Error(), "Input/output error") {
		t.Errorf("launchctl's own message must survive; got: %v", err)
	}
}

func TestApply_JobThatNeverRunsIsATypedVerifyTimeout(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	lc.neverRuns = true
	start := time.Now()
	_, err := Apply(m, p, lc, ApplyOpts{})
	if !errors.Is(err, ErrVerifyTimeout) {
		t.Fatalf("expected ErrVerifyTimeout, got %v", err)
	}
	if el := time.Since(start); el > verifyDeadline+5*time.Second {
		t.Errorf("verify wait was not bounded: %v", el)
	}
}

// The whole operation is bounded, not just its parts.
func TestApply_WorstCaseIsBounded(t *testing.T) {
	if bootoutDeadline+bootstrapDeadline+verifyDeadline > 40*time.Second {
		t.Errorf("worst case %v exceeds the 35s the design commits to",
			bootoutDeadline+bootstrapDeadline+verifyDeadline)
	}
}

func TestApply_SkipReloadPromotesWithoutTouchingLaunchd(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	res, err := Apply(m, p, lc, ApplyOpts{SkipReload: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Reloaded || lc.bootstraps.Load() != 0 {
		t.Error("SkipReload must not touch launchd")
	}
	if len(res.Promoted) != 2 {
		t.Error("SkipReload must still promote both artifacts")
	}
	if !strings.Contains(strings.Join(res.Notes, " "), "next fire") {
		t.Errorf("must warn that the env is live on the next fire anyway; notes=%v", res.Notes)
	}
}

func TestApply_BacksUpWhatItReplaces(t *testing.T) {
	m, p, lc := applyFixture(t, true)
	backups := filepath.Join(t.TempDir(), "bak")
	res, err := Apply(m, p, lc, ApplyOpts{BackupDir: backups})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Backups) != 2 {
		t.Fatalf("expected 2 backups, got %v", res.Backups)
	}
	for _, b := range res.Backups {
		if _, err := os.Stat(b); err != nil {
			t.Errorf("reported backup %s does not exist", b)
		}
	}
}
