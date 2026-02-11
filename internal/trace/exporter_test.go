package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportTrainingData_SingleFile(t *testing.T) {
	// Create temp trace file
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "test_trace.jsonl")

	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "test", Caps: []string{"IO"}}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventEffect, Depth: 1, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "()"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "test"}},
	}

	f, err := os.Create(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	WriteJSONL(f, events)
	f.Close()

	var buf strings.Builder
	exported, skipped, err := ExportTrainingData(&buf, []string{tracePath}, ExportConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if exported != 1 {
		t.Errorf("expected 1 exported, got %d", exported)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}

	// Parse output
	var example TrainingExample
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &example); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if example.Score <= 0 {
		t.Errorf("expected positive score, got %f", example.Score)
	}
	if example.Metadata.Module != "test" {
		t.Errorf("expected module 'test', got %q", example.Metadata.Module)
	}
	if example.Metadata.Events != 5 {
		t.Errorf("expected 5 events, got %d", example.Metadata.Events)
	}
}

func TestExportTrainingData_MinScore(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "low_score.jsonl")

	// Minimal trace — low score
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "f"}},
	}

	f, err := os.Create(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	WriteJSONL(f, events)
	f.Close()

	var buf strings.Builder
	exported, skipped, err := ExportTrainingData(&buf, []string{tracePath}, ExportConfig{MinScore: 0.9})
	if err != nil {
		t.Fatal(err)
	}

	if exported != 0 {
		t.Errorf("expected 0 exported (below min score), got %d", exported)
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", skipped)
	}
}

func TestExportTrainingData_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	// Good trace
	goodPath := filepath.Join(dir, "good.jsonl")
	goodEvents := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "good"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventEffect, Depth: 1, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "()"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "good"}},
	}
	writeTraceFile(t, goodPath, goodEvents)

	// Bad trace (error)
	badPath := filepath.Join(dir, "bad.jsonl")
	badEvents := []TraceEvent{
		{Version: "1.0", Event: EventError, Error: &ErrorEvent{Message: "crash"}},
	}
	writeTraceFile(t, badPath, badEvents)

	var buf strings.Builder
	exported, skipped, err := ExportTrainingData(&buf, []string{goodPath, badPath}, ExportConfig{MinScore: 0.3})
	if err != nil {
		t.Fatal(err)
	}

	if exported != 1 {
		t.Errorf("expected 1 exported, got %d", exported)
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", skipped)
	}
}

func TestExportTrainingData_WithSource(t *testing.T) {
	dir := t.TempDir()

	// Create source file
	sourceContent := "module test\n\nexport func main() -> () ! {IO} {\n  println(\"hello\")\n}\n"
	sourcePath := filepath.Join(dir, "test.ail")
	os.WriteFile(sourcePath, []byte(sourceContent), 0644)

	// Create trace
	tracePath := filepath.Join(dir, "trace.jsonl")
	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "test"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "()"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "test"}},
	}
	writeTraceFile(t, tracePath, events)

	var buf strings.Builder
	_, _, err := ExportTrainingData(&buf, []string{tracePath}, ExportConfig{})
	if err != nil {
		t.Fatal(err)
	}

	var example TrainingExample
	json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &example)

	if example.Source != sourceContent {
		t.Errorf("expected source content, got %q", example.Source)
	}
}

func TestExportTrainingData_MissingFile(t *testing.T) {
	var buf strings.Builder
	exported, skipped, err := ExportTrainingData(&buf, []string{"/nonexistent/trace.jsonl"}, ExportConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if exported != 0 {
		t.Errorf("expected 0 exported, got %d", exported)
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", skipped)
	}
}

func TestExportTrainingData_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(tracePath, []byte(""), 0644)

	var buf strings.Builder
	exported, skipped, err := ExportTrainingData(&buf, []string{tracePath}, ExportConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if exported != 0 {
		t.Errorf("expected 0 exported, got %d", exported)
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", skipped)
	}
}

func TestScoreTraceFile(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "score_test.jsonl")

	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "test"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "test"}},
	}
	writeTraceFile(t, tracePath, events)

	score, err := ScoreTraceFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}

	if score.Total <= 0 {
		t.Errorf("expected positive score, got %f", score.Total)
	}
	if score.Completion != 1.0 {
		t.Errorf("expected completion 1.0, got %f", score.Completion)
	}
}

func TestScoreTraceFile_NotFound(t *testing.T) {
	_, err := ScoreTraceFile("/nonexistent.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func writeTraceFile(t *testing.T, path string, events []TraceEvent) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := WriteJSONL(f, events); err != nil {
		t.Fatal(err)
	}
}
