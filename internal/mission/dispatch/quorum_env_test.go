package dispatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

func TestQuorumEnvironmentHandoff(t *testing.T) {
	for _, arm := range []string{"clean", "inherited_marker", "task_policy"} {
		t.Run(arm, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			override := filepath.Join(home, " evidence with spaces ")
			if err := os.WriteFile(filepath.Join(home, "task-policy"), []byte("allowed_caps = [\"IO\"]\nentry = \"main\"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv(config.EnvMissionQuorumArtifactDir, override)
			marker := ""
			if arm == "inherited_marker" {
				marker = " "
			}
			t.Setenv(config.EnvAgentPolicy, marker)
			for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
				for _, lane := range []struct{ executor, model string }{{"claude", ""}, {"codex", ""}, {"pi", "anthropic/claude-opus-5-5"}, {"pi", "openai/gpt-6.1-sol"}, {"pi", "google/gemini-4-argon"}, {"pi", "openrouter/z-ai/glm-5.3"}} {
					t.Run(role+"/"+lane.executor+"/"+lane.model, func(t *testing.T) {
						task := taskFor(Request{Role: role, Workspace: home}, testCandidate())
						wantMarker := marker
						if arm == "task_policy" {
							task.PolicyPath = filepath.Join(home, "task-policy")
							wantMarker = task.PolicyPath
						}
						env, err := executor.BuildEnvironment(executor.EnvironmentOptions{Task: task, Executor: lane.executor, Model: lane.model})
						if err != nil {
							t.Fatal(err)
						}
						cmd := exec.Command("/bin/bash", "-c", `printf '%s\n' "$MISSION_QUORUM_ARTIFACT_DIR"; if [ -n "${AILANG_AGENT_POLICY:-}" ]; then echo present; else echo absent; fi`)
						cmd.Env = env
						cmd.Dir = home
						out, err := cmd.CombinedOutput()
						presence := "absent"
						if wantMarker != "" {
							presence = "present"
						}
						if err != nil || string(out) != override+"\n"+presence+"\n" {
							t.Fatalf("fake CLI %q %v", out, err)
						}
						// Compare exact bytes internally; print only presence on a mismatch.
						found := ""
						for _, kv := range env {
							if strings.HasPrefix(kv, config.EnvAgentPolicy+"=") {
								found = strings.TrimPrefix(kv, config.EnvAgentPolicy+"=")
							}
						}
						if found != wantMarker {
							t.Fatal("marker handoff differs")
						}
					})
				}
			}
		})
	}
}
