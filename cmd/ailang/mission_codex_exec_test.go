package main

import (
	"strings"
	"testing"
)

func TestMissionCodexExecArgs(t *testing.T) {
	opts, jsonOutput, err := parseMissionCodexExecArgs([]string{"-c", `shell_environment_policy.set.MISSION_NAME="fleet"`, "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox", "--model", "test-model", "-C", "/work", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Model != "test-model" || opts.Workspace != "/work" || opts.Directive != "hello" || opts.Sandbox != "danger-full-access" || opts.ApprovalPolicy != "never" || jsonOutput {
		t.Fatalf("unexpected options: %+v json=%v", opts, jsonOutput)
	}
	if opts.Config["shell_environment_policy.set.MISSION_NAME"] != "fleet" {
		t.Fatal("role environment dropped")
	}
}

func TestMissionCodexExecRefusesUnsupportedArguments(t *testing.T) {
	for _, args := range [][]string{{"--model", "m", "--no-daemon", "p"}, {"--model", "m", "--output-schema", "f", "p"}, {"--model", "m", "p", "extra"}, {"--model"}, {"p"}, {"--model", "m", "-c", "x", "p"}, {"--model", "m", "--sandbox", "bad", "p"}} {
		if _, _, err := parseMissionCodexExecArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestMissionCodexExecJSONAndTOML(t *testing.T) {
	opts, j, err := parseMissionCodexExecArgs([]string{"--json", "-m", "m", "-c", "features.web_search_request=true", "-c", `model_reasoning_effort="high"`, "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if !j || opts.Config["features.web_search_request"] != true || opts.Effort != "high" {
		t.Fatalf("options %+v json %v", opts, j)
	}
}

func TestMissionCodexTextWriter(t *testing.T) {
	var out strings.Builder
	w := &codexTextWriter{out: &out}
	input := `{"type":"thread.started","thread_id":"t"}` + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}` + "\n" + `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"done"}}` + "\n"
	for _, part := range []string{input[:19], input[19:]} {
		if _, err := w.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(out.String(), "hello") || !strings.Contains(out.String(), "done") {
		t.Fatalf("lost output %q", out.String())
	}
}

// These tests call only the parser: no environment setup, daemon, or socket.
func TestMissionCodexExecRoleRefusal(t *testing.T) {
	for _, model := range []string{"gpt-6-astra", "gpt6-astra", "codex:gpt-6-sol", "gpt-5.6-luna", "gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			_, _, err := parseMissionCodexExecArgs([]string{"--model", model, "prompt"})
			if err == nil {
				t.Fatalf("accepted retired model %s before daemon validation", model)
			}
			for _, want := range []string{model, "gpt-6.1-sol", "D-FLEET-12"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal %q missing %q", err, want)
				}
			}
		})
	}
}

func TestMissionCodexExecForcedSpawnConfig(t *testing.T) {
	const key = "features.multi_agent_v2.expose_spawn_agent_model_overrides"
	for _, model := range []string{"gpt-6.1-sol", "gpt6-1-sol", "codex:gpt-6.1-sol", "openai/gpt-6.1-sol", "GPT-6.1-SOL", "claude-opus-5-5"} {
		t.Run(model, func(t *testing.T) {
			opts, _, err := parseMissionCodexExecArgs([]string{"--model", model, "prompt"})
			if err != nil {
				t.Fatal(err)
			}
			if value, ok := opts.Config[key].(bool); !ok || value {
				t.Fatalf("spawn override must be boolean false without -c, got %#v", opts.Config[key])
			}
		})
	}
}

func TestMissionCodexExecCallerSpawnOverride(t *testing.T) {
	const key = "features.multi_agent_v2.expose_spawn_agent_model_overrides"
	for _, raw := range []string{"true", `"false"`, "'false'", "0", "1", "[]", "{}", "true;false"} {
		t.Run(raw, func(t *testing.T) {
			args := []string{"--model", "gpt-6.1-sol", "-c", key + "=" + strings.Split(raw, ";")[0]}
			if strings.Contains(raw, ";") {
				args = append(args, "--config", key+"=false")
			}
			if _, _, err := parseMissionCodexExecArgs(append(args, "prompt")); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("caller override %s must be refused with key, got %v", raw, err)
			}
		})
	}
	opts, _, err := parseMissionCodexExecArgs([]string{"--model", "gpt-6.1-sol", "-c", key + "=false", "-c", `model_reasoning_effort="high"`, "-c", "other=true", "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := opts.Config[key].(bool); !ok || v || opts.Config["other"] != true || opts.Effort != "high" {
		t.Fatalf("explicit false and unrelated config must survive: %+v", opts)
	}
}
