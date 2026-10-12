package executor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// Task.ExtraEnv validation (M-EXECUTOR-ENV-HARDENING D3).
//
// ExtraEnv reaches the model-facing child with the highest precedence of any
// layer, and some of its sources are semi-trusted: a benchmark's `agent_env`
// is a data file anyone with a PR can edit. So it may set ordinary
// configuration (MOTOKO_AST_AUTOREAD, AILANG_MISSION_STAGE, a browser lane's
// PLAYWRIGHT_MCP_CDP_ENDPOINT) but never a name that gives it control over how
// the child loads code, where its traffic or telemetry goes, which git config
// it trusts, or the values the harness itself injects (trace context,
// correlation IDs, the program policy). A violation is a loud error naming the
// variable and the rule — never a silent drop.

var extraEnvNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// extraEnvDeniedExact are refused by exact (case-insensitive) name.
var extraEnvDeniedExact = map[string]string{
	"PATH":  "executable search path",
	"HOME":  "home directory (config, credentials and hooks resolve from it)",
	"SHELL": "login shell",
	// Shell and interpreter start-up hooks: each makes the next
	// non-interactive shell or interpreter run attacker-chosen code.
	"BASH_ENV":       "shell start-up file",
	"ENV":            "shell start-up file",
	"PROMPT_COMMAND": "shell start-up hook",
	"IFS":            "shell word splitting",
	"NODE_OPTIONS":   "node loader options (--require/--import)",
	"PYTHONSTARTUP":  "python start-up file",
	"PYTHONPATH":     "python module search path",
	"PERL5OPT":       "perl loader options",
	"RUBYOPT":        "ruby loader options",
	// Proxies: route every request, credentials included, through a host
	// the task chose.
	"HTTP_PROXY":  "proxy routing",
	"HTTPS_PROXY": "proxy routing",
	"ALL_PROXY":   "proxy routing",
	"NO_PROXY":    "proxy routing",
	// Harness-injected values: ExtraEnv outranks the harness layer, so these
	// are refused outright rather than silently overwritten either way.
	"PWD":                       "harness-injected working directory",
	config.EnvCodexHome:         "harness-owned Codex credential home",
	config.EnvCodexRuntime:      "harness-owned Codex refresh owner runtime",
	"TRACEPARENT":               "harness-injected trace context",
	"TRACESTATE":                "harness-injected trace context",
	"AILANG_TASK_ID":            "harness-injected correlation ID",
	"AILANG_SESSION_ID":         "harness-injected correlation ID",
	"AILANG_PARENT_TASK_ID":     "harness-injected correlation ID",
	"AILANG_CHAIN_ID":           "harness-injected correlation ID",
	"AILANG_STAGE_ID":           "harness-injected correlation ID",
	"AILANG_MESSAGE_ID":         "harness-injected correlation ID",
	"AILANG_STDLIB_PATH":        "harness-injected stdlib pin",
	config.EnvRigLease:          "harness-injected rig lease",
	"GOOGLE_CLOUD_PROJECT":      "harness-injected GCP project (use Task.GCPProject)",
	"GOOGLE_CLOUD_LOCATION":     "harness-injected GCP location (use Task.GCPLocation)",
	"OTLP_GOOGLE_CLOUD_PROJECT": "harness-injected trace project",
}

// extraEnvDeniedPrefixes are refused by (case-insensitive) prefix.
var extraEnvDeniedPrefixes = []struct{ prefix, rule string }{
	{"LD_", "dynamic loader (LD_PRELOAD and friends)"},
	{"DYLD_", "dynamic loader (macOS)"},
	{"GIT_", "git configuration and transport (GIT_CONFIG*, GIT_SSH_COMMAND, GIT_ASKPASS)"},
	{"OTEL_", "telemetry routing (an exporter endpoint is an exfiltration channel)"},
	{config.EnvAgentPolicy, "program policy (harness-injected from Task.PolicyPath)"},
}

// ExtraEnvDenyRule returns the rule that refuses name as a Task.ExtraEnv key,
// or "" when the name is allowed. Exported so a spec loader can fail at load
// time with the same verdict the builder enforces.
func ExtraEnvDenyRule(name string) string {
	if !extraEnvNameRE.MatchString(name) {
		return "not a portable environment variable name ([A-Za-z_][A-Za-z0-9_]*)"
	}
	upper := strings.ToUpper(name)
	if rule, ok := extraEnvDeniedExact[upper]; ok {
		return rule
	}
	for _, d := range extraEnvDeniedPrefixes {
		if strings.HasPrefix(upper, d.prefix) {
			return d.rule
		}
	}
	return ""
}

// ValidateExtraEnv checks every Task.ExtraEnv entry, in sorted order so the
// first error is deterministic. It is enforced by BuildEnvironment (every
// executor, every dispatch path) and, earlier, by ValidateTaskCapabilities.
func ValidateExtraEnv(extra map[string]string) error {
	names := make([]string, 0, len(extra))
	for k := range extra {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		if rule := ExtraEnvDenyRule(name); rule != "" {
			return fmt.Errorf("ExtraEnv %q is refused: %s; remove it from the task (benchmark agent_env, dispatcher) — the agent child may not receive it from a task", name, rule)
		}
		if strings.ContainsRune(extra[name], 0) {
			return fmt.Errorf("ExtraEnv %q is refused: value contains a NUL byte", name)
		}
	}
	return nil
}
