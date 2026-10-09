package config

import "testing"

func TestCloudRunExecution(t *testing.T) {
	t.Setenv("CLOUD_RUN_EXECUTION", "")
	if got := CloudRunExecution(); got != "" {
		t.Fatalf("unset execution: %q", got)
	}
	t.Setenv("CLOUD_RUN_EXECUTION", "codex-go-execution")
	if got := CloudRunExecution(); got != "codex-go-execution" {
		t.Fatalf("execution: %q", got)
	}
}
