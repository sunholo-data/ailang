package effects

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewBudgetReport(t *testing.T) {
	br := NewBudgetReport()

	if br == nil {
		t.Fatal("NewBudgetReport returned nil")
	}
	if br.FunctionUsage == nil {
		t.Error("FunctionUsage should be initialized")
	}
	if br.FunctionLimits == nil {
		t.Error("FunctionLimits should be initialized")
	}
	if br.TotalUsage == nil {
		t.Error("TotalUsage should be initialized")
	}
	if br.HasUsage() {
		t.Error("New report should not have usage")
	}
}

func TestBudgetReport_RecordUsage(t *testing.T) {
	br := NewBudgetReport()

	// Record usage without entering a function (uses <global>)
	br.RecordUsage("IO", 1)
	br.RecordUsage("IO", 2)
	br.RecordUsage("FS", 1)

	if br.TotalUsage["IO"] != 3 {
		t.Errorf("Expected IO total 3, got %d", br.TotalUsage["IO"])
	}
	if br.TotalUsage["FS"] != 1 {
		t.Errorf("Expected FS total 1, got %d", br.TotalUsage["FS"])
	}
	if br.FunctionUsage["<global>"]["IO"] != 3 {
		t.Errorf("Expected <global> IO 3, got %d", br.FunctionUsage["<global>"]["IO"])
	}
}

func TestBudgetReport_EnterExitFunction(t *testing.T) {
	br := NewBudgetReport()

	// Enter function with limits
	limits := map[string]int{"IO": 5, "FS": 3}
	br.EnterFunction("main", limits)

	if br.CurrentFunction != "main" {
		t.Errorf("Expected current function 'main', got '%s'", br.CurrentFunction)
	}

	// Check limits were recorded
	if br.FunctionLimits["main"]["IO"] == nil || *br.FunctionLimits["main"]["IO"] != 5 {
		t.Error("Expected IO limit 5 for main")
	}

	// Record usage
	br.RecordUsage("IO", 2)
	br.RecordUsage("FS", 1)

	if br.FunctionUsage["main"]["IO"] != 2 {
		t.Errorf("Expected main IO 2, got %d", br.FunctionUsage["main"]["IO"])
	}

	// Exit function
	bc := NewBudgetContext(nil)
	br.ExitFunction("main", bc)
}

func TestBudgetReport_HasUsage(t *testing.T) {
	br := NewBudgetReport()

	if br.HasUsage() {
		t.Error("Empty report should not have usage")
	}

	br.RecordUsage("IO", 1)

	if !br.HasUsage() {
		t.Error("Report with recorded usage should have usage")
	}
}

func TestFormatReport_Empty(t *testing.T) {
	br := NewBudgetReport()
	output := FormatReport(br)

	if output != "" {
		t.Errorf("Empty report should return empty string, got: %s", output)
	}
}

func TestFormatReport_WithUsage(t *testing.T) {
	br := NewBudgetReport()

	// Enter function and record usage
	br.EnterFunction("main", map[string]int{"IO": 5})
	br.RecordUsage("IO", 3)
	br.RecordUsage("FS", 2)

	output := FormatReport(br)

	// Check output contains expected elements
	if !strings.Contains(output, "Budget Report:") {
		t.Error("Output should contain 'Budget Report:'")
	}
	if !strings.Contains(output, "main:") {
		t.Error("Output should contain 'main:'")
	}
	if !strings.Contains(output, "IO") {
		t.Error("Output should contain 'IO'")
	}
	if !strings.Contains(output, "Total:") {
		t.Error("Output should contain 'Total:'")
	}
}

func TestFormatReportJSON_Empty(t *testing.T) {
	br := NewBudgetReport()
	data, err := FormatReportJSON(br)

	if err != nil {
		t.Fatalf("FormatReportJSON failed: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("Empty report JSON should be '{}', got: %s", string(data))
	}
}

func TestFormatReportJSON_WithUsage(t *testing.T) {
	br := NewBudgetReport()

	// Enter function and record usage
	br.EnterFunction("main", map[string]int{"IO": 5})
	br.RecordUsage("IO", 3)

	data, err := FormatReportJSON(br)
	if err != nil {
		t.Fatalf("FormatReportJSON failed: %v", err)
	}

	// Parse JSON to verify structure
	var result BudgetReportJSON
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check structure
	if result.Functions == nil {
		t.Error("Functions should not be nil")
	}
	if result.Total["IO"] != 3 {
		t.Errorf("Expected total IO 3, got %d", result.Total["IO"])
	}

	funcBudget, ok := result.Functions["main"]
	if !ok {
		t.Fatal("Expected 'main' in functions")
	}

	ioBudget, ok := funcBudget.Effects["IO"]
	if !ok {
		t.Fatal("Expected 'IO' in main's effects")
	}
	// M-DX25: Now tracks both semantic and physical
	if ioBudget.Semantic != 3 {
		t.Errorf("Expected IO semantic 3, got %d", ioBudget.Semantic)
	}
	if ioBudget.Physical != 3 {
		t.Errorf("Expected IO physical 3, got %d", ioBudget.Physical)
	}
	if ioBudget.Limit == nil || *ioBudget.Limit != 5 {
		t.Error("Expected IO limit 5")
	}
}

func TestFormatReportForError(t *testing.T) {
	br := NewBudgetReport()

	// Empty report
	output := FormatReportForError(br)
	if output != "" {
		t.Error("Empty report should return empty string for error format")
	}

	// With usage
	br.EnterFunction("main", map[string]int{"IO": 5})
	br.RecordUsage("IO", 3)

	output = FormatReportForError(br)
	if !strings.Contains(output, "Budget at failure:") {
		t.Error("Error format should contain 'Budget at failure:'")
	}
	if !strings.Contains(output, "main:") {
		t.Error("Error format should contain function name")
	}
}

func TestEffContext_RequireCapWithBudget_RecordsToReport(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))
	ctx.BudgetReport = NewBudgetReport()

	// Require cap should record to report
	err := ctx.RequireCapWithBudget("IO", "test.ail:1:1")
	if err != nil {
		t.Fatalf("RequireCapWithBudget failed: %v", err)
	}

	// Check report was updated
	if ctx.BudgetReport.TotalUsage["IO"] != 1 {
		t.Errorf("Expected IO total 1, got %d", ctx.BudgetReport.TotalUsage["IO"])
	}
}

func TestEffContext_WithBudget_PreservesReport(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.BudgetReport = NewBudgetReport()
	ctx.BudgetReport.RecordUsage("IO", 1)

	// Create new context with different budget
	limit := 5
	newBudget := NewBudgetContext(map[string]*int{"IO": &limit})
	newCtx := ctx.WithBudget(newBudget)

	// Report should be preserved
	if newCtx.BudgetReport != ctx.BudgetReport {
		t.Error("BudgetReport should be preserved across WithBudget")
	}
	if newCtx.BudgetReport.TotalUsage["IO"] != 1 {
		t.Errorf("Expected preserved IO total 1, got %d", newCtx.BudgetReport.TotalUsage["IO"])
	}
}
