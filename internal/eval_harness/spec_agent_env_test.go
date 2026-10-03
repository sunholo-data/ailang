package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-EXECUTOR-ENV-HARDENING D3: a benchmark's agent_env is semi-trusted data,
// so a loader/routing name fails at spec load, naming the variable, while the
// names the live corpus uses (MOTOKO_AST_*) keep loading.
func TestLoadSpec_AgentEnvDenyList(t *testing.T) {
	base := `id: "env_probe"
languages: ["ailang"]
prompt: "x"
agent_env:
`
	write := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "spec.yml")
		if err := os.WriteFile(p, []byte(base+body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	if _, err := LoadSpec(write(t, "  MOTOKO_AST_AUTOREAD: \"1\"\n  MOTOKO_AST_READ_FULL: \"${WORKSPACE}/a.ail\"\n")); err != nil {
		t.Fatalf("live agent_env names refused: %v", err)
	}
	for _, name := range []string{"LD_PRELOAD", "OTEL_EXPORTER_OTLP_ENDPOINT", "GIT_SSH_COMMAND", "AILANG_AGENT_POLICY", "PATH"} {
		_, err := LoadSpec(write(t, "  "+name+": \"x\"\n"))
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("agent_env %s: LoadSpec = %v, want an error naming it", name, err)
		}
	}
}
