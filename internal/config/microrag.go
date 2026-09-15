package config

import (
	"strconv"
	"strings"
)

// MicroRAG (internal/microrag): the retrieval hook on agent prompts.
const (
	EnvMicroRAGEnabled         = "AILANG_MICRORAG_ENABLED"
	EnvMicroRAGRoutes          = "AILANG_MICRORAG_ROUTES"
	EnvMicroRAGDryrun          = "AILANG_MICRORAG_DRYRUN"
	EnvMicroRAGSession         = "AILANG_MICRORAG_SESSION"
	EnvMicroRAGUserPromptFloor = "AILANG_MICRORAG_USERPROMPT_FLOOR"
)

// DefaultMicroRAGUserPromptFloor is the relevance floor for the user-prompt
// path; a widened gate is the failure mode the hook's token cost cannot afford.
const DefaultMicroRAGUserPromptFloor = 0.70

var microragVars = []Var{
	{EnvMicroRAGEnabled, "true", AreaMicroRAG, "0 or false turns the engine off; anything else (including an invalid value) leaves it on — it does not fail closed."},
	{EnvMicroRAGRoutes, "", AreaMicroRAG, "Comma-separated allowlist of routes the engine serves; unset allows all."},
	{EnvMicroRAGDryrun, "false", AreaMicroRAG, "1 or true retrieves and logs without injecting."},
	{EnvMicroRAGSession, "", AreaMicroRAG, "Session id that names the ledger directory; unset uses pid-<pid> so concurrent sessions do not share one."},
	{EnvMicroRAGUserPromptFloor, "0.70", AreaMicroRAG, "Relevance floor in (0, 1] for the user-prompt path; out of range or malformed keeps the default rather than widening the gate."},
}

// MicroRAGEnabled returns the trimmed AILANG_MICRORAG_ENABLED, "" when unset;
// the engine applies its "invalid means on" rule.
func MicroRAGEnabled() string { return strings.TrimSpace(get(EnvMicroRAGEnabled)) }

// MicroRAGRoutes returns the trimmed AILANG_MICRORAG_ROUTES, "" when unset.
func MicroRAGRoutes() string { return strings.TrimSpace(get(EnvMicroRAGRoutes)) }

// MicroRAGDryrun reports AILANG_MICRORAG_DRYRUN=1 or true (case-insensitive).
func MicroRAGDryrun() bool {
	v := strings.TrimSpace(get(EnvMicroRAGDryrun))
	return v == "1" || strings.ToLower(v) == "true"
}

// MicroRAGSession returns the trimmed AILANG_MICRORAG_SESSION, "" when unset.
func MicroRAGSession() string { return strings.TrimSpace(get(EnvMicroRAGSession)) }

// MicroRAGUserPromptFloor returns AILANG_MICRORAG_USERPROMPT_FLOOR when it
// parses to a float in (0, 1], else the default.
func MicroRAGUserPromptFloor() float64 {
	v := strings.TrimSpace(get(EnvMicroRAGUserPromptFloor))
	if v == "" {
		return DefaultMicroRAGUserPromptFloor
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 || f > 1 {
		return DefaultMicroRAGUserPromptFloor
	}
	return f
}
