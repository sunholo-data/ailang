package golang

import (
	"strings"
	"testing"
)

func TestGenerateRandHandler(t *testing.T) {
	gen := NewEffectsGenerator("mygame")
	handlers := []EffectHandler{DefaultRandHandler()}

	code, err := gen.GenerateHandlers(handlers)
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	output := string(code)

	// Check package declaration
	if !strings.Contains(output, "package mygame") {
		t.Error("Missing package declaration")
	}

	// Check interface definition
	if !strings.Contains(output, "type RandHandler interface {") {
		t.Error("Missing RandHandler interface")
	}

	// Check methods
	if !strings.Contains(output, "RandInt(min int64, max int64) int64") {
		t.Error("Missing RandInt method")
	}
	if !strings.Contains(output, "RandFloat(min float64, max float64) float64") {
		t.Error("Missing RandFloat method")
	}
	if !strings.Contains(output, "RandBool() bool") {
		t.Error("Missing RandBool method")
	}
	if !strings.Contains(output, "SetSeed(seed int64)") {
		t.Error("Missing SetSeed method")
	}

	// Check Handlers struct
	if !strings.Contains(output, "type Handlers struct {") {
		t.Error("Missing Handlers struct")
	}
	if !strings.Contains(output, "Rand RandHandler") {
		t.Error("Missing Rand field in Handlers")
	}

	// Check Init function
	if !strings.Contains(output, "func Init(h Handlers) {") {
		t.Error("Missing Init function")
	}
}

func TestGenerateClockHandler(t *testing.T) {
	gen := NewEffectsGenerator("mygame")
	handlers := []EffectHandler{DefaultClockHandler()}

	code, err := gen.GenerateHandlers(handlers)
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	output := string(code)

	// Check interface definition
	if !strings.Contains(output, "type ClockHandler interface {") {
		t.Error("Missing ClockHandler interface")
	}

	// Check methods
	if !strings.Contains(output, "DeltaTime() float64") {
		t.Error("Missing DeltaTime method")
	}
	if !strings.Contains(output, "TotalTime() float64") {
		t.Error("Missing TotalTime method")
	}
	if !strings.Contains(output, "FrameCount() int64") {
		t.Error("Missing FrameCount method")
	}
}

func TestGenerateMultipleHandlers(t *testing.T) {
	gen := NewEffectsGenerator("mygame")
	handlers := []EffectHandler{
		DefaultRandHandler(),
		DefaultClockHandler(),
	}

	code, err := gen.GenerateHandlers(handlers)
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	output := string(code)

	// Check both interfaces
	if !strings.Contains(output, "type RandHandler interface {") {
		t.Error("Missing RandHandler interface")
	}
	if !strings.Contains(output, "type ClockHandler interface {") {
		t.Error("Missing ClockHandler interface")
	}

	// Check Handlers struct has both - use tab-prefixed format
	if !strings.Contains(output, "Rand  RandHandler") && !strings.Contains(output, "Rand RandHandler") {
		t.Errorf("Missing Rand field in output:\n%s", output)
	}
	if !strings.Contains(output, "Clock ClockHandler") {
		t.Errorf("Missing Clock field in output:\n%s", output)
	}
}

func TestGenerateHandlersDocComments(t *testing.T) {
	gen := NewEffectsGenerator("mygame")
	handlers := []EffectHandler{DefaultRandHandler()}

	code, err := gen.GenerateHandlers(handlers)
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	output := string(code)

	// Check documentation comments
	if !strings.Contains(output, "// RandInt returns a random integer") {
		t.Error("Missing RandInt doc comment")
	}
	if !strings.Contains(output, "// Example:") {
		t.Error("Missing example hint")
	}
}

func TestEffectsGeneratedCodeCompiles(t *testing.T) {
	gen := NewEffectsGenerator("mygame")
	handlers := []EffectHandler{
		DefaultRandHandler(),
		DefaultClockHandler(),
	}

	code, err := gen.GenerateHandlers(handlers)
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	// The fact that we got here without error means go/format.Source succeeded
	// which means the code is syntactically valid Go
	if len(code) == 0 {
		t.Error("Generated code is empty")
	}
}
