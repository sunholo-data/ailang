package config

import "strings"

// The eval harness (internal/eval_harness, `ailang eval-*`): where the
// language toolchains are, the memory cap on generated code, the prompt
// delivery knobs and the observatory the suite files its task under.
const (
	EnvAILANGBin           = "AILANG_BIN"
	EnvEvalMaxRSS          = "AILANG_EVAL_MAX_RSS"
	EnvEvalTrapsCard       = "AILANG_EVAL_TRAPS_CARD"
	EnvEvalPersistPrompt   = "AILANG_EVAL_PERSIST_PROMPT"
	EnvAgentOutputDelivery = "AILANG_AGENT_OUTPUT_DELIVERY"
	EnvUV                  = "AILANG_UV"
	EnvAver                = "AILANG_AVER"
	EnvCargoHome           = "CARGO_HOME"
	EnvMoon                = "AILANG_MOON"
	EnvMoonHome            = "MOON_HOME"
	EnvObservatoryEndpoint = "OBSERVATORY_ENDPOINT"
)

// DefaultObservatoryEndpoint is the local observatory the eval suite files
// its task with when OBSERVATORY_ENDPOINT is unset.
const DefaultObservatoryEndpoint = "http://localhost:1957"

var evalVars = []Var{
	{EnvAILANGBin, "", AreaEval, "The ailang binary the agent-mode grade probe runs; unset resolves `ailang` on PATH and serves it as a deprecated default (the stale-binary trap), refused under AILANG_STRICT_CONFIG=1."},
	{EnvEvalMaxRSS, "8G", AreaEval, "Resident-memory cap per generated-code run: a byte count or an integer with a K/M/G/T suffix; 0 or off disables the watchdog, and a malformed value is an error."},
	{EnvEvalTrapsCard, "", AreaEval, "Path of the traps card prepended to every agent-mode directive; unset loads the built-in card, and off (or 0/false/no/none) disables it."},
	{EnvEvalPersistPrompt, "0", AreaEval, "1, true, on or yes delivers the full teaching prompt through a persistent system-prompt channel re-injected every turn (measured worse; for A/B only)."},
	{EnvAgentOutputDelivery, "1", AreaEval, "0 drops the agent-mode output-delivery override from the system prompt (a clean A/B control arm)."},
	{EnvUV, "", AreaEval, "Path of the uv binary for Python benchmarks; unset looks it up on PATH."},
	{EnvAver, "", AreaEval, "Path of the aver binary; unset looks on PATH, then $CARGO_HOME/bin/aver."},
	{EnvCargoHome, "", AreaEval, "Cargo home searched for aver when it is not on PATH; unset means ~/.cargo."},
	{EnvMoon, "", AreaEval, "Path of the moon (MoonBit) binary; unset looks on PATH, then $MOON_HOME/bin/moon."},
	{EnvMoonHome, "", AreaEval, "MoonBit home searched for moon when it is not on PATH; unset means ~/.moon."},
	{EnvObservatoryEndpoint, DefaultObservatoryEndpoint, AreaEval, "Observatory API the eval suite creates and completes its task in."},
}

// AILANGBin returns the trimmed AILANG_BIN, "" when unset.
func AILANGBin() string { return strings.TrimSpace(get(EnvAILANGBin)) }

// EvalMaxRSS returns the trimmed AILANG_EVAL_MAX_RSS, or the registered 8G
// when unset; the memory watchdog parses it so a malformed cap fails loudly
// there.
func EvalMaxRSS() string { return strings.TrimSpace(getOr(EnvEvalMaxRSS)) }

// EvalTrapsCard returns the trimmed AILANG_EVAL_TRAPS_CARD, "" when unset.
func EvalTrapsCard() string { return strings.TrimSpace(get(EnvEvalTrapsCard)) }

// EvalPersistPrompt reports whether AILANG_EVAL_PERSIST_PROMPT is 1, true,
// on or yes (case-insensitive).
func EvalPersistPrompt() bool {
	switch strings.ToLower(strings.TrimSpace(get(EnvEvalPersistPrompt))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// AgentOutputDelivery reports whether the agent-mode output-delivery
// override is on: anything but AILANG_AGENT_OUTPUT_DELIVERY=0.
func AgentOutputDelivery() bool { return getOr(EnvAgentOutputDelivery) != "0" }

// UV returns the trimmed AILANG_UV, "" when unset.
func UV() string { return strings.TrimSpace(get(EnvUV)) }

// Aver returns the trimmed AILANG_AVER, "" when unset.
func Aver() string { return strings.TrimSpace(get(EnvAver)) }

// CargoHome returns the trimmed CARGO_HOME, "" when unset.
func CargoHome() string { return strings.TrimSpace(get(EnvCargoHome)) }

// Moon returns the trimmed AILANG_MOON, "" when unset.
func Moon() string { return strings.TrimSpace(get(EnvMoon)) }

// MoonHome returns the trimmed MOON_HOME, "" when unset.
func MoonHome() string { return strings.TrimSpace(get(EnvMoonHome)) }

// ObservatoryEndpoint returns OBSERVATORY_ENDPOINT, default the local observatory.
func ObservatoryEndpoint() string { return getOr(EnvObservatoryEndpoint) }
