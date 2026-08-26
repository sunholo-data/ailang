package coordinator

import (
	"path/filepath"
	"strings"
)

// ExecutionLane names where an agent's work actually runs.
//
// M-MESSAGE-PLANE-FAIL-LOUD M3 (decision D3, ratified 2026-08-26).
//
// This exists because `workspace` was answering two questions at once. It is
// DECLARED as "Base directory for worktrees" and is chdir'd into by
// NewWorktreeManager; cloud dispatch ALSO read it, deriving a repo URL whenever
// it looked like org/repo. Satisfying either consumer broke the other:
//
//	workspace: sunholo-data/ailang        -> cloud dispatch works, the local
//	                                         worktree manager logs "chdir
//	                                         sunholo-data/ailang: no such file or
//	                                         directory" every 30s (measured: 3.5h)
//	workspace: /Users/.../ailang          -> worktrees work, and a Cloud Run job
//	                                         receives a Mac Studio filesystem path
//	                                         as its clone target (measured: 10
//	                                         consecutive dead-on-arrival jobs)
//
// Neither value is right, because the question "where does this run?" was being
// inferred from whether a string starts with a slash. Declare it instead.
type ExecutionLane string

const (
	// LaneCloud dispatches to a Cloud Run Job, which clones a repo coordinate.
	LaneCloud ExecutionLane = "cloud"
	// LaneLocal runs on a bare-metal worker against a local checkout.
	LaneLocal ExecutionLane = "local"
)

// isPathNotCoordinate reports whether s is a filesystem path rather than a
// GitHub coordinate.
//
// Deliberately NOT just filepath.IsAbs: that answers "is this absolute on the
// HOST I am running on", and the config is shared across hosts. A POSIX path
// like /Users/x/dev/ailang is not IsAbs on Windows, so a Windows reader would
// classify a bare-metal worker as cloud — measured on test-windows 2026-08-26,
// where TestExecutionLane_InfersLocalForAbsolutePath got "cloud".
//
// A coordinate is org/repo: exactly one slash, no leading separator. Anything
// with a leading "/" is a POSIX path on every platform, and IsAbs still catches
// Windows drive letters. The question is about the STRING's shape, not the
// reader's OS.
func isPathNotCoordinate(s string) bool {
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "/") || filepath.IsAbs(s)
}

// looksLikeOrgRepo reports whether s has the shape of a GitHub coordinate:
// exactly one slash and not an absolute path. This is the same test
// deriveRepoURL has always used, named so the intent is legible.
func looksLikeOrgRepo(s string) bool {
	return s != "" &&
		strings.Count(s, "/") == 1 &&
		!strings.HasPrefix(s, "/") &&
		!filepath.IsAbs(s)
}

// ResolveLane returns the agent's execution lane, preferring an explicit
// declaration and otherwise inferring from the workspace shape.
//
// The inference default is CLOUD, and that is load-bearing: all 39 agents in the
// live cloud config carry a bare org/repo workspace with no execution_lane. A
// local default would silently move the whole fleet onto a lane that does not
// exist on Cloud Run.
func (a *AgentConfig) ResolveLane() ExecutionLane {
	if a == nil {
		return LaneCloud
	}
	switch a.ExecutionLane {
	case LaneLocal, LaneCloud:
		return a.ExecutionLane
	}
	if isPathNotCoordinate(a.Workspace) {
		return LaneLocal
	}
	return LaneCloud
}

// ResolveRepo returns the GitHub coordinate (org/repo) for this agent, or "" if
// it has none.
//
// Prefers the explicit `repo` field, so `workspace` can go back to meaning only
// what it is declared to mean. Falls back to a bare org/repo workspace for
// back-compat — that is what every existing agent relies on.
//
// An ABSOLUTE workspace never resolves as a coordinate. Returning one would
// rebuild the original defect exactly: a Mac Studio path handed to a Cloud Run
// job as a clone target.
func (a *AgentConfig) ResolveRepo() string {
	if a == nil {
		return ""
	}
	if a.Repo != "" {
		return a.Repo
	}
	if looksLikeOrgRepo(a.Workspace) {
		return a.Workspace
	}
	return ""
}
