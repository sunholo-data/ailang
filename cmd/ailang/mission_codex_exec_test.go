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
