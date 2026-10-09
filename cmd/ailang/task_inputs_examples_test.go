package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestTaskInputsExamples(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "task_inputs")
	inputs, err := loadTaskInputsFile(filepath.Join(root, "inputs.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, declared, err := coordinator.LoadAgentRegistryFromDeclared(filepath.Join(root, "agent-config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !declared {
		t.Fatal("example is not a declared coordinator config")
	}
	agent := registry.GetAgentByID("site-builder")
	if agent == nil {
		t.Fatal("missing example agent")
	}
	if err := coordinator.ValidateInputsForAgent(inputs, agent); err != nil {
		t.Fatal(err)
	}
	directive := coordinator.BuildDirectiveFromConfig(&coordinator.TaskRecord{Content: "Publish the autumn event page"}, agent)
	if !strings.Contains(directive, "Publish the autumn event page") {
		t.Fatal("example prompt lost the sender request")
	}
	if agent.ToolPolicy != "ailang_only" || agent.ExecutionLane != "cloud" {
		t.Fatalf("example lane/policy: %+v", agent)
	}
}
