package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestContractGenerator_GenerateContractTypes(t *testing.T) {
	g := NewContractGenerator("testpkg")
	code, err := g.GenerateContractTypes()
	if err != nil {
		t.Fatalf("GenerateContractTypes failed: %v", err)
	}

	codeStr := string(code)

	// Check for expected types
	if !strings.Contains(codeStr, "type ContractMode int") {
		t.Error("missing ContractMode type")
	}
	if !strings.Contains(codeStr, "type ContractCheck struct") {
		t.Error("missing ContractCheck struct")
	}
	if !strings.Contains(codeStr, "type ContractContext struct") {
		t.Error("missing ContractContext struct")
	}
	if !strings.Contains(codeStr, "ContractModePanic") {
		t.Error("missing ContractModePanic constant")
	}
	if !strings.Contains(codeStr, "ContractModeReport") {
		t.Error("missing ContractModeReport constant")
	}
	if !strings.Contains(codeStr, "ContractModeOff") {
		t.Error("missing ContractModeOff constant")
	}
}

func TestGenerateRequiresChecks(t *testing.T) {
	contracts := []*core.Contract{
		{
			Kind:     core.RequiresKind,
			Message:  "x >= 0",
			Location: "test.ail:10:1",
		},
		{
			Kind:     core.EnsuresKind,
			Message:  "result > x",
			Location: "test.ail:11:1",
		},
	}

	code := GenerateRequiresChecks(contracts, "foo")

	// Should only contain requires
	if !strings.Contains(code, "Requires: x >= 0") {
		t.Error("missing requires comment")
	}
	// Should NOT contain ensures
	if strings.Contains(code, "Ensures") {
		t.Error("should not contain ensures")
	}
}

func TestGenerateEnsuresChecks(t *testing.T) {
	contracts := []*core.Contract{
		{
			Kind:     core.RequiresKind,
			Message:  "x >= 0",
			Location: "test.ail:10:1",
		},
		{
			Kind:     core.EnsuresKind,
			Message:  "result > x",
			Location: "test.ail:11:1",
		},
	}

	code := GenerateEnsuresChecks(contracts, "foo")

	// Should only contain ensures
	if !strings.Contains(code, "Ensures: result > x") {
		t.Error("missing ensures comment")
	}
	// Should NOT contain requires
	if strings.Contains(code, "Requires") {
		t.Error("should not contain requires")
	}
}

func TestGenerateChecks_EmptyContracts(t *testing.T) {
	var contracts []*core.Contract

	reqCode := GenerateRequiresChecks(contracts, "foo")
	ensCode := GenerateEnsuresChecks(contracts, "foo")

	if reqCode != "" {
		t.Errorf("expected empty requires code, got %q", reqCode)
	}
	if ensCode != "" {
		t.Errorf("expected empty ensures code, got %q", ensCode)
	}
}

func TestDefaultContractHandler(t *testing.T) {
	handler := DefaultContractHandler()

	if handler.Name != "Contract" {
		t.Errorf("expected name 'Contract', got %q", handler.Name)
	}

	if len(handler.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(handler.Methods))
	}

	// Check CheckRequires method
	if handler.Methods[0].Name != "CheckRequires" {
		t.Errorf("expected first method 'CheckRequires', got %q", handler.Methods[0].Name)
	}

	// Check CheckEnsures method
	if handler.Methods[1].Name != "CheckEnsures" {
		t.Errorf("expected second method 'CheckEnsures', got %q", handler.Methods[1].Name)
	}
}
