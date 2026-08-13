// Dual-write routing for `ailang chains post-iteration`
// (M-MISSION-LOOP-UNIFIED-TELEMETRY M3).
//
// The ratified requirement these tests exist for is NEVER BLOCK: when the remote
// observatory cannot be reached, the iteration is buffered and the loop continues.
// Asserted against a REAL broken config and a REAL failing backend, not by reading
// the code.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func testPost(source string) *observatory.IterationPost {
	return &observatory.IterationPost{
		Source: source,
		Stages: []observatory.IterationStage{
			{Role: "controller", QuotaBucket: "opus", Status: "completed"},
			{Role: "quorum-r1", Provider: "openrouter", Model: "gpt-5", CostUSD: 0.057,
				TokensIn: 18432, TokensOut: 2101, Status: "completed"},
		},
	}
}

// TestOpenPostTargets_NoCloudIsUnchanged: with no remote named, the command has
// exactly one target and behaves as it did before dual-write existed.
func TestOpenPostTargets_NoCloudIsUnchanged(t *testing.T) {
	t.Setenv("AILANG_CHAINS_CLOUD", "")
	spool := filepath.Join(t.TempDir(), "spool.jsonl")

	targets := openPostTargets(context.Background(), spool, "")
	for _, tgt := range targets {
		if tgt.close != nil {
			tgt.close()
		}
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target with no cloud configured, got %d", len(targets))
	}
	if targets[0].name != "local" {
		t.Errorf("target name = %q, want \"local\"", targets[0].name)
	}
}

// TestOpenPostTargets_BrokenCloudConfigStillReturnsTarget: a remote that cannot be
// opened must still come back as a target carrying its error. Dropping it here
// would silently discard the post instead of spooling it — the failure mode the
// never-block decision exists to prevent.
func TestOpenPostTargets_BrokenCloudConfigStillReturnsTarget(t *testing.T) {
	// A genuinely broken config: gcp mode with no project set.
	t.Setenv("AILANG_CLOUD_PROJECT", "")
	spool := filepath.Join(t.TempDir(), "spool.jsonl")

	targets := openPostTargets(context.Background(), spool, "gcp")
	for _, tgt := range targets {
		if tgt.close != nil {
			tgt.close()
		}
	}
	if len(targets) != 2 {
		t.Fatalf("expected local + cloud targets, got %d", len(targets))
	}
	cloud := targets[1]
	if cloud.name != "cloud" {
		t.Fatalf("second target = %q, want \"cloud\"", cloud.name)
	}
	if cloud.connErr == nil {
		t.Fatal("broken cloud config produced no connErr — the failure would be invisible")
	}
	if cloud.backend != nil {
		t.Error("broken cloud target must not carry a usable backend")
	}
}

// TestOpenPostTargets_UnknownModeIsAnError: an unknown storage mode is rejected by
// the shared selector rather than silently falling back to local — a silent
// fallback here would report success while writing nowhere near the cloud.
func TestOpenPostTargets_UnknownModeIsAnError(t *testing.T) {
	spool := filepath.Join(t.TempDir(), "spool.jsonl")
	targets := openPostTargets(context.Background(), spool, "not-a-mode")
	for _, tgt := range targets {
		if tgt.close != nil {
			tgt.close()
		}
	}
	if len(targets) != 2 || targets[1].connErr == nil {
		t.Fatal("unknown storage mode must surface as a cloud connErr")
	}
}

// TestWriteToTarget_UnreachableTargetSpoolsAndContinues is the never-block
// assertion at the write site: a broken target buffers the post, says so, and
// reports failure without panicking or aborting.
func TestWriteToTarget_UnreachableTargetSpoolsAndContinues(t *testing.T) {
	dir := t.TempDir()
	spoolPath := filepath.Join(dir, "cloud-spool.jsonl")
	target := &postTarget{
		name:    "cloud",
		spool:   observatory.NewSpool(spoolPath),
		connErr: os.ErrNotExist, // stands in for "cloud is not reachable right now"
	}
	target.spool.SetWarnWriter(os.Stderr)

	if writeToTarget(context.Background(), target, testPost("mission:v1/iter-200")) {
		t.Fatal("writeToTarget reported success for an unreachable target")
	}
	if n := target.spool.Len(); n != 1 {
		t.Fatalf("spool holds %d entries, want 1 — the post was lost", n)
	}
}

// TestWriteToTarget_FailingBackendSpools uses a REAL backend that fails writes (an
// opened-then-closed SQLite store), covering the case where the target connects
// and the write itself fails.
func TestWriteToTarget_FailingBackendSpools(t *testing.T) {
	backend, err := observatory.NewSQLiteBackendFromPath(filepath.Join(t.TempDir(), "obs.db"))
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}
	_ = backend.Close() // every subsequent write now fails for real

	spoolPath := filepath.Join(t.TempDir(), "spool.jsonl")
	target := &postTarget{name: "cloud", backend: backend, spool: observatory.NewSpool(spoolPath)}

	if writeToTarget(context.Background(), target, testPost("mission:v1/iter-201")) {
		t.Fatal("writeToTarget reported success against a closed store")
	}
	if n := target.spool.Len(); n != 1 {
		t.Fatalf("spool holds %d entries, want 1", n)
	}
}

// TestWriteToTarget_SpoolStaysBoundedUnderCloudOutage: a persistent outage cannot
// grow the spool without limit. 120 failed posts, cap 100.
func TestWriteToTarget_SpoolStaysBoundedUnderCloudOutage(t *testing.T) {
	dir := t.TempDir()
	spoolPath := filepath.Join(dir, "cloud-spool.jsonl")
	spool := observatory.NewSpool(spoolPath)
	spool.SetWarnWriter(os.NewFile(0, os.DevNull)) // 120 loud notices would drown the log
	target := &postTarget{name: "cloud", spool: spool, connErr: os.ErrNotExist}

	for i := 0; i < 120; i++ {
		writeToTarget(context.Background(), target, testPost("mission:v1/iter-"+string(rune('a'+i%26))))
	}

	if n := spool.Len(); n != observatory.DefaultSpoolMaxEntries {
		t.Errorf("spool holds %d entries after 120 failures, want the %d cap",
			n, observatory.DefaultSpoolMaxEntries)
	}
	info, err := os.Stat(spoolPath)
	if err != nil {
		t.Fatalf("stat spool: %v", err)
	}
	if info.Size() > observatory.DefaultSpoolMaxBytes {
		t.Errorf("spool grew to %d bytes, past the %d cap", info.Size(), observatory.DefaultSpoolMaxBytes)
	}
}

// TestCloudSpoolPath_IsSeparate: the targets must not share a spool. A shared one
// would let a long cloud outage evict local posts that were only waiting on a
// locked database.
func TestCloudSpoolPath_IsSeparate(t *testing.T) {
	local := "/x/.ailang/state/chains-iteration-spool.jsonl"
	cloud := cloudSpoolPath(local)
	if cloud == local {
		t.Fatal("cloud spool path equals the local one — an outage on one would evict the other")
	}
	if filepath.Dir(cloud) != filepath.Dir(local) {
		t.Errorf("cloud spool %q left the local spool's directory", cloud)
	}
	if filepath.Ext(cloud) != ".jsonl" {
		t.Errorf("cloud spool %q lost its .jsonl extension", cloud)
	}
}

// setHomeDir points os.UserHomeDir() at dir (or makes it fail, when dir is "")
// on every platform the CI matrix builds.
//
// os.UserHomeDir reads a DIFFERENT variable per GOOS — USERPROFILE on Windows,
// $home on plan9, HOME elsewhere — so a test that sets only HOME silently has no
// effect on Windows: the runner's real profile resolves, the guard under test
// never sees the input the test believes it supplied, and the assertion fails for
// the platform rather than for the code. Setting all three keeps the arm honest
// wherever it runs.
func setHomeDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)        // unix, darwin
	t.Setenv("USERPROFILE", dir) // windows
	t.Setenv("home", dir)        // plan9
}

func TestCheckRemoteIsElsewhere_UnresolvableHomeIsAnError(t *testing.T) {
	t.Setenv("AILANG_STATE_DIR", "")
	setHomeDir(t, "")

	err := checkRemoteIsElsewhere("local")
	if err == nil || !strings.Contains(err.Error(), "cannot resolve") {
		t.Fatalf("unresolvable home error = %v, want message containing %q", err, "cannot resolve")
	}
}

func TestCheckRemoteIsElsewhere_SelfTargetIsRejected(t *testing.T) {
	home := t.TempDir()
	setHomeDir(t, home)
	t.Setenv("AILANG_STATE_DIR", filepath.Join(home, ".ailang", "state"))

	err := checkRemoteIsElsewhere("local")
	if err == nil || !strings.Contains(err.Error(), "resolves to this node's own observatory") {
		t.Fatalf("self-target error = %v, want self-observatory rejection", err)
	}
}

func TestCheckRemoteIsElsewhere_PositiveControls(t *testing.T) {
	t.Run("non-local mode short-circuits", func(t *testing.T) {
		setHomeDir(t, "")
		t.Setenv("AILANG_STATE_DIR", "")
		if err := checkRemoteIsElsewhere("gcp"); err != nil {
			t.Fatalf("gcp target rejected: %v", err)
		}
	})

	t.Run("different local directory is accepted", func(t *testing.T) {
		home := t.TempDir()
		setHomeDir(t, home)
		t.Setenv("AILANG_STATE_DIR", filepath.Join(t.TempDir(), "remote-state"))
		if err := checkRemoteIsElsewhere("local"); err != nil {
			t.Fatalf("distinct local target rejected: %v", err)
		}
	})
}
