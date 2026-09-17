package coordinator

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-AGENT-AILANG-ONLY-EXECUTION M4: the registry's tool_policy is what the
// operator can READ; its default is the fleet's (full, D6).

func TestAgentConfig_ToolPolicyDefaultsToFull(t *testing.T) {
	a := &AgentConfig{ID: "x"}
	if got := a.GetEffectiveToolPolicy(); got != executor.ToolProfileFull {
		t.Fatalf("undeclared tool_policy = %q, want full", got)
	}
	a.ToolPolicy = " ailang_only "
	if got := a.GetEffectiveToolPolicy(); got != executor.ToolProfileAILANGOnly {
		t.Fatalf("declared = %q", got)
	}
}

func TestQuestionTools_PerExecutor(t *testing.T) {
	// pi has no Grep/Glob/WebFetch/WebSearch: the Claude list would have run
	// with zero tools, silently (V4). pi gets Read + the type-checker.
	if got := questionTools("pi"); !reflect.DeepEqual(got, []string{"Read", "AilangCheck"}) {
		t.Fatalf("pi question tools = %v", got)
	}
	if got := questionTools("claude"); !reflect.DeepEqual(got, []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"}) {
		t.Fatalf("claude question tools = %v", got)
	}
}
