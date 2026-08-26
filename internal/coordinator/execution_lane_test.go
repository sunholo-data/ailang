package coordinator

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// M-MESSAGE-PLANE-FAIL-LOUD M3 (decision D3: explicit execution_lane).
//
// `workspace` carried two incompatible meanings. It is DECLARED as "Base
// directory for worktrees" (AgentConfig.Workspace) and is handed to
// NewWorktreeManager, which chdir's into it. Cloud dispatch ALSO read it,
// deriving a repo URL whenever it looked like org/repo.
//
// So satisfying one consumer broke the other. Measured 2026-08-26: setting
// eval-rig's workspace to "sunholo-data/ailang" fixed cloud dispatch and made the
// rig log "chdir sunholo-data/ailang: no such file or directory" every 30s for
// ~3.5 hours; setting it back to the local path restored worktrees and re-broke
// cloud dispatch. Neither value is correct because the field is answering two
// questions.
//
// D3: declare the lane instead of inferring intent from whether a string starts
// with a slash.
func TestExecutionLane_Parse(t *testing.T) {
	var cfg AgentConfig
	src := `
id: eval-rig
inbox: eval-rig
workspace: /Users/x/dev/ailang
execution_lane: local
repo: sunholo-data/ailang
`
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.ExecutionLane != LaneLocal {
		t.Errorf("expected lane %q, got %q", LaneLocal, cfg.ExecutionLane)
	}
	if cfg.Repo != "sunholo-data/ailang" {
		t.Errorf("expected repo coordinate preserved, got %q", cfg.Repo)
	}
}

// BACK-COMPAT IS LOAD-BEARING. All 39 agents in the live cloud config carry a
// bare org/repo workspace and no execution_lane. If inference defaulted them to
// local, this milestone would silently move the entire fleet onto a lane that
// does not exist on Cloud Run.
func TestExecutionLane_InfersCloudForBareOrgRepo(t *testing.T) {
	cases := []string{
		"sunholo-data/ailang",
		"sunholo-data/ailang-packages",
		"sunholo-data/ailang-parse",
	}
	for _, ws := range cases {
		a := &AgentConfig{ID: "x", Workspace: ws}
		if got := a.ResolveLane(); got != LaneCloud {
			t.Errorf("workspace %q must infer cloud (fleet default), got %q", ws, got)
		}
	}
}

// An absolute path is a bare-metal worker: a Cloud Run job cannot use it.
func TestExecutionLane_InfersLocalForAbsolutePath(t *testing.T) {
	a := &AgentConfig{ID: "eval-rig", Workspace: "/Users/voightkampff/dev/sunholo-data/ailang"}
	if got := a.ResolveLane(); got != LaneLocal {
		t.Errorf("an absolute workspace must infer local, got %q", got)
	}
}

// An explicit declaration always wins over inference — that is the entire point.
func TestExecutionLane_ExplicitBeatsInference(t *testing.T) {
	a := &AgentConfig{ID: "eval-rig", Workspace: "sunholo-data/ailang", ExecutionLane: LaneLocal}
	if got := a.ResolveLane(); got != LaneLocal {
		t.Errorf("explicit execution_lane must win over inference, got %q", got)
	}
	b := &AgentConfig{ID: "x", Workspace: "/abs/path", ExecutionLane: LaneCloud}
	if got := b.ResolveLane(); got != LaneCloud {
		t.Errorf("explicit execution_lane must win over inference, got %q", got)
	}
}

// The repo coordinate must come from `repo` when set, so `workspace` can go back
// to meaning only what it is declared to mean.
func TestResolveRepoCoordinate_PrefersExplicitRepo(t *testing.T) {
	a := &AgentConfig{ID: "eval-rig", Workspace: "/Users/x/dev/ailang", Repo: "sunholo-data/ailang"}
	if got := a.ResolveRepo(); got != "sunholo-data/ailang" {
		t.Errorf("explicit repo must be used, got %q", got)
	}
}

// Back-compat: with no `repo`, a bare org/repo workspace is still usable as the
// coordinate (that is what all 39 agents rely on today).
func TestResolveRepoCoordinate_FallsBackToOrgRepoWorkspace(t *testing.T) {
	a := &AgentConfig{ID: "x", Workspace: "sunholo-data/ailang-packages"}
	if got := a.ResolveRepo(); got != "sunholo-data/ailang-packages" {
		t.Errorf("bare org/repo workspace must still resolve as the coordinate, got %q", got)
	}
}

// An absolute workspace is NOT a repo coordinate. Returning it would rebuild the
// exact bug: a Mac Studio path handed to a Cloud Run job as a clone target.
func TestResolveRepoCoordinate_AbsolutePathIsNotACoordinate(t *testing.T) {
	a := &AgentConfig{ID: "eval-rig", Workspace: "/Users/voightkampff/dev/sunholo-data/ailang"}
	if got := a.ResolveRepo(); got != "" {
		t.Errorf("an absolute path must not be offered as a repo coordinate, got %q", got)
	}
}

// FLEET REGRESSION — the single most important test in M3.
//
// Fixture is the LIVE cloud config as of 2026-08-26 (39 agents, every one with a
// bare org/repo workspace and no execution_lane). If lane inference ever defaults
// these to local, the entire fleet silently stops being cloud-dispatched onto a
// lane that does not exist on Cloud Run — a failure that would look exactly like
// "the coordinator went quiet", which is the class of bug this whole milestone
// exists to eliminate.
func TestExecutionLane_LiveFleetStillInfersCloud(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "live_cloud_config_20260826.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var file struct {
		Coordinator CoordinatorConfig `yaml:"coordinator"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	agents := file.Coordinator.Agents
	if len(agents) == 0 {
		t.Fatal("fixture parsed to zero agents — the test would vacuously pass")
	}
	// Pin the count so a fixture that silently shrinks is caught too.
	if len(agents) != 39 {
		t.Errorf("fixture should hold the 39 agents measured 2026-08-26, got %d", len(agents))
	}

	for _, a := range agents {
		if got := a.ResolveLane(); got != LaneCloud {
			t.Errorf("agent %q (workspace %q) inferred %q — the live fleet MUST infer cloud", a.ID, a.Workspace, got)
		}
		if a.ResolveRepo() == "" {
			t.Errorf("agent %q (workspace %q) resolved no repo coordinate; a cloud job would have nothing to clone", a.ID, a.Workspace)
		}
	}
}
