package config

import "strings"

// The mission outer loop and the identity of whoever is acting.
const (
	EnvMissionName          = "MISSION_NAME"
	EnvMissionGHIssue       = "MISSION_GH_ISSUE"
	EnvMissionControlActive = "MISSION_CONTROL_ACTIVE"
	EnvControllerID         = "CONTROLLER_ID"
	EnvMissionRole          = "MISSION_ROLE"
	EnvMissionMeteredBudget = "MISSION_METERED_BUDGET_USD"
	EnvMissionDesignAuthor  = "MISSION_DESIGN_AUTHOR"
	EnvAnthropicRation      = "AILANG_ANTHROPIC_RATION"
	EnvMissionRegistry      = "AILANG_MISSION_REGISTRY"
	EnvCodexHome            = "CODEX_HOME"
	EnvClaudeCode           = "CLAUDECODE"
	EnvClaudeCodeSessionID  = "CLAUDE_CODE_SESSION_ID"
	EnvUser                 = "USER"
)

// MissionMessagePlaneProject is the Firestore project every mission stage
// execution files its messages to, regardless of the parent environment.
// It is a fixed value, deliberately NOT a Var and NOT overridable by any
// variable: the point of the pin is that a dev shell (AILANG_STORAGE=local,
// a scratch AILANG_MESSAGES_PROJECT) cannot re-route a stage's
// cross-mission messages away from the canonical inbox. Ruling D8 of
// design_docs/planned/m-v1-simplification-program.md (2026-09-15): keep the
// pin, make it explicit — it is the one production literal D3 does not
// retire. internal/mission/dispatch sets AILANG_STORAGE_MESSAGING=gcp and
// AILANG_MESSAGES_PROJECT to this on every stage.
const MissionMessagePlaneProject = "ailang-multivac"

var missionVars = []Var{
	{EnvMissionName, "", AreaMission, "Name of the mission whose fire this process belongs to (v1, docs, motoko, world, fleet); set by the mission driver. Default filer for `ailang mission ticket file`; the push scope guard keys on it."},
	{EnvMissionGHIssue, "", AreaMission, "GitHub issue number that overrides the mission's directive channel, so a cutover can be rehearsed against a scratch issue."},
	{EnvMissionControlActive, "0", AreaMission, "1 marks the process as running inside a mission-control iteration."},
	{EnvControllerID, "", AreaMission, "Identity of the mission controller acting in this process; used as the approval identity label."},
	{EnvMissionRole, "", AreaMission, "Role the mission loop pinned this process to (controller, designer, executor, ...)."},
	{EnvMissionMeteredBudget, "", AreaMission, "Metered spend budget in USD for the mission's chain statistics; unset means no budget line."},
	{EnvMissionDesignAuthor, "", AreaMission, "The designer lane that wrote a doc (e.g. codex:gpt-6-astra); design-quorum benches that vendor. Unset assumes a Claude author."},
	{EnvAnthropicRation, "1", AreaMission, "0 turns off Anthropic subscription rationing for one attended process; unset means rationed."},
	{EnvMissionRegistry, "", AreaMission, "Absolute directory holding mission definitions, instead of the built-in registry."},
	{EnvCodexHome, "", AreaMission, "Codex CLI home whose auth and quota files are observed; unset means ~/.codex."},
	{EnvClaudeCode, "", AreaMission, "Set to 1 by Claude Code in the sessions it runs; used only to label an attended identity."},
	{EnvClaudeCodeSessionID, "", AreaMission, "Claude Code session id, appended to the attended identity label."},
	{EnvUser, "", AreaMission, "The login user, the last-resort identity label and operator principal."},
}

// MissionName returns the trimmed MISSION_NAME, "" when unset (an attended session).
func MissionName() string { return strings.TrimSpace(get(EnvMissionName)) }

// MissionGHIssue returns the trimmed MISSION_GH_ISSUE, "" when unset.
func MissionGHIssue() string { return strings.TrimSpace(get(EnvMissionGHIssue)) }

// MissionControlActive reports MISSION_CONTROL_ACTIVE=1.
func MissionControlActive() bool { return getOr(EnvMissionControlActive) == "1" }

// ControllerID returns the trimmed CONTROLLER_ID, "" when unset.
func ControllerID() string { return strings.TrimSpace(get(EnvControllerID)) }

// MissionDesignAuthor returns the trimmed MISSION_DESIGN_AUTHOR, "" when unset.
func MissionDesignAuthor() string { return strings.TrimSpace(get(EnvMissionDesignAuthor)) }

// MissionRole returns the trimmed MISSION_ROLE, "" when unset.
func MissionRole() string { return strings.TrimSpace(get(EnvMissionRole)) }

// MissionMeteredBudgetUSD returns MISSION_METERED_BUDGET_USD verbatim, ""
// when unset; the caller parses it so it can distinguish unset from 0.
func MissionMeteredBudgetUSD() string { return get(EnvMissionMeteredBudget) }

// AnthropicRation reports whether rationing is on: anything but
// AILANG_ANTHROPIC_RATION=0.
func AnthropicRation() bool { return getOr(EnvAnthropicRation) != "0" }

// MissionRegistry returns AILANG_MISSION_REGISTRY, "" when unset.
func MissionRegistry() string { return get(EnvMissionRegistry) }

// CodexHome returns CODEX_HOME, "" when unset.
func CodexHome() string { return get(EnvCodexHome) }

// InClaudeCode reports CLAUDECODE=1.
func InClaudeCode() bool { return get(EnvClaudeCode) == "1" }

// ClaudeCodeSessionID returns the trimmed CLAUDE_CODE_SESSION_ID, "" when unset.
func ClaudeCodeSessionID() string { return strings.TrimSpace(get(EnvClaudeCodeSessionID)) }

// User returns the trimmed USER, "" when unset.
func User() string { return strings.TrimSpace(get(EnvUser)) }
