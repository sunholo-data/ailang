package executor

import (
	"strings"
	"testing"
)

// M-EXECUTOR-ENV-HARDENING D3 / AC3.
func TestValidateExtraEnv_DenyList(t *testing.T) {
	denied := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
		"PATH", "HOME", "SHELL", "BASH_ENV", "ENV", "NODE_OPTIONS", "PYTHONPATH",
		"GIT_CONFIG_GLOBAL", "GIT_SSH_COMMAND", "GIT_ASKPASS", "GIT_CONFIG_COUNT",
		"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_RESOURCE_ATTRIBUTES",
		"AILANG_AGENT_POLICY", "AILANG_AGENT_POLICY_TOML",
		"HTTP_PROXY", "https_proxy", "ALL_PROXY", "NO_PROXY",
		"TRACEPARENT", "AILANG_TASK_ID", "AILANG_PARENT_TASK_ID", "AILANG_RIG_LEASE",
		"AILANG_STDLIB_PATH", "PWD", "GOOGLE_CLOUD_PROJECT",
		"lower case ok?", "1LEADING_DIGIT", "HAS-DASH", "",
	}
	for _, name := range denied {
		err := ValidateExtraEnv(map[string]string{name: "x"})
		if err == nil {
			t.Errorf("ExtraEnv %q accepted, want refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), "ExtraEnv") {
			t.Errorf("error for %q does not name the rule source: %v", name, err)
		}
	}
}

// What the harness and the corpus legitimately set today must keep working.
func TestValidateExtraEnv_LiveProducersAllowed(t *testing.T) {
	ok := map[string]string{
		"MOTOKO_AST_AUTOREAD":         "1",                      // benchmarks/*.yml agent_env
		"MOTOKO_AST_READ_FULL":        "a.ail:b.ail",            // benchmarks/*.yml agent_env
		"AILANG_MISSION_STAGE":        "1",                      // mission dispatch
		"AILANG_STORAGE_MESSAGING":    "gcp",                    // mission dispatch
		"AILANG_MESSAGES_PROJECT":     "ailang-multivac",        // mission dispatch
		"PI_WORKSPACE_TRUST_REMOTES":  "sunholo-data/ailang",    // coordinator + cloud job
		"PLAYWRIGHT_MCP_CDP_ENDPOINT": "wss://example/devtools", // browser lane
	}
	if err := ValidateExtraEnv(ok); err != nil {
		t.Fatalf("live ExtraEnv producers refused: %v", err)
	}
}

func TestValidateExtraEnv_NULValue(t *testing.T) {
	if err := ValidateExtraEnv(map[string]string{"MOTOKO_X": "a\x00b"}); err == nil {
		t.Fatal("NUL byte in an ExtraEnv value accepted")
	}
}

func TestValidateTaskCapabilities_RefusesDeniedExtraEnv(t *testing.T) {
	task := &Task{ExtraEnv: map[string]string{"LD_PRELOAD": "/tmp/x.so"}}
	err := ValidateTaskCapabilities(task, &capExecutor{name: "pi"})
	if err == nil || !strings.Contains(err.Error(), `"LD_PRELOAD"`) {
		t.Fatalf("ValidateTaskCapabilities = %v, want a pre-dispatch error naming LD_PRELOAD", err)
	}
}
