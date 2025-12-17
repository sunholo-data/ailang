package gemini

import (
	"testing"

	"github.com/sunholo/ailang/internal/executor"
)

func TestNewGeminiExecutor(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if exec.Name() != "gemini" {
		t.Errorf("expected name 'gemini', got '%s'", exec.Name())
	}
}

func TestGeminiDefaultModel(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Should use Gemini 3 Flash preview as default (from DefaultConfig)
	if exec.model != "gemini-3-flash-preview" {
		t.Errorf("expected default model 'gemini-3-flash-preview', got '%s'", exec.model)
	}
}

func TestGeminiCapabilities(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	caps := exec.Capabilities()
	if len(caps) == 0 {
		t.Error("expected at least one capability")
	}

	hasStreaming := false
	for _, cap := range caps {
		if cap == executor.CapStreaming {
			hasStreaming = true
			break
		}
	}
	if !hasStreaming {
		t.Error("expected streaming capability")
	}
}

func TestGeminiCostModel(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	costModel := exec.CostModel()

	// Gemini 3 Flash pricing: $0.50/$3.00 per 1M
	expectedInput := 0.0005 // $0.50 per 1M = $0.0005 per 1K
	expectedOutput := 0.003 // $3.00 per 1M = $0.003 per 1K

	if costModel.InputTokenCost != expectedInput {
		t.Errorf("expected input cost %.4f, got %.4f", expectedInput, costModel.InputTokenCost)
	}
	if costModel.OutputTokenCost != expectedOutput {
		t.Errorf("expected output cost %.4f, got %.4f", expectedOutput, costModel.OutputTokenCost)
	}
}

func TestGeminiFactoryRegistration(t *testing.T) {
	// Gemini executor should be registered via init()
	factory := executor.GlobalFactory()
	available := factory.ListAvailable()

	hasGemini := false
	for _, name := range available {
		if name == "gemini" {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Error("expected 'gemini' to be registered in global factory")
	}
}
