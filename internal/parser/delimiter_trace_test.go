package parser

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

func TestDelimiterTracerDisabled(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Ensure disabled
	os.Setenv("DEBUG_DELIMITERS", "0")

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Operations should be no-ops
	tracer.push(delimCtxMatch, 1, 5)
	if len(tracer.stack) != 0 {
		t.Errorf("Expected empty stack when disabled, got %d frames", len(tracer.stack))
	}

	tracer.pop(delimCtxMatch, 1, 10)
	// Should not panic
}

func TestDelimiterTracerEnabled(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Push a frame
	tracer.push(delimCtxMatch, 1, 5)
	if len(tracer.stack) != 1 {
		t.Fatalf("Expected 1 frame after push, got %d", len(tracer.stack))
	}

	frame := tracer.stack[0]
	if frame.context != delimCtxMatch {
		t.Errorf("Expected context=match, got %s", frame.context)
	}
	if frame.line != 1 {
		t.Errorf("Expected line=1, got %d", frame.line)
	}
	if frame.col != 5 {
		t.Errorf("Expected col=5, got %d", frame.col)
	}
	if frame.depth != 0 {
		t.Errorf("Expected depth=0, got %d", frame.depth)
	}

	// Pop the frame
	tracer.pop(delimCtxMatch, 1, 10)
	if len(tracer.stack) != 0 {
		t.Errorf("Expected empty stack after pop, got %d frames", len(tracer.stack))
	}
}

func TestDelimiterTracerNestedContexts(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Push match context
	tracer.push(delimCtxMatch, 1, 5)
	if len(tracer.stack) != 1 || tracer.stack[0].depth != 0 {
		t.Errorf("Expected depth=0 for first frame")
	}

	// Push block context (nested)
	tracer.push(delimCtxBlock, 2, 10)
	if len(tracer.stack) != 2 || tracer.stack[1].depth != 1 {
		t.Errorf("Expected depth=1 for second frame")
	}

	// Push case context (deeply nested)
	tracer.push(delimCtxCase, 3, 15)
	if len(tracer.stack) != 3 || tracer.stack[2].depth != 2 {
		t.Errorf("Expected depth=2 for third frame")
	}

	// Pop in correct order
	tracer.pop(delimCtxCase, 3, 20)
	if len(tracer.stack) != 2 {
		t.Errorf("Expected 2 frames after first pop, got %d", len(tracer.stack))
	}

	tracer.pop(delimCtxBlock, 2, 25)
	if len(tracer.stack) != 1 {
		t.Errorf("Expected 1 frame after second pop, got %d", len(tracer.stack))
	}

	tracer.pop(delimCtxMatch, 1, 30)
	if len(tracer.stack) != 0 {
		t.Errorf("Expected empty stack after final pop, got %d", len(tracer.stack))
	}
}

func TestDelimiterTracerMismatch(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Push match context
	tracer.push(delimCtxMatch, 1, 5)

	// Pop with wrong context (should detect mismatch)
	tracer.pop(delimCtxBlock, 1, 10)

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for mismatch message
	if !strings.Contains(output, "DELIM_MISMATCH") {
		t.Errorf("Expected DELIM_MISMATCH in output, got: %s", output)
	}
	if !strings.Contains(output, "match") && !strings.Contains(output, "block") {
		t.Errorf("Expected both 'match' and 'block' in mismatch message, got: %s", output)
	}
}

func TestDelimiterTracerPopEmptyStack(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Pop from empty stack
	tracer.pop(delimCtxMatch, 1, 5)

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for error message
	if !strings.Contains(output, "DELIM_ERROR") {
		t.Errorf("Expected DELIM_ERROR in output, got: %s", output)
	}
	if !strings.Contains(output, "stack is empty") {
		t.Errorf("Expected 'stack is empty' message, got: %s", output)
	}
}

func TestDelimiterTracerShowStack(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Push multiple contexts
	tracer.push(delimCtxMatch, 1, 5)
	tracer.push(delimCtxBlock, 2, 10)
	tracer.push(delimCtxCase, 3, 15)

	// Show stack
	tracer.showStack()

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for stack display
	if !strings.Contains(output, "DELIM_STACK") {
		t.Errorf("Expected DELIM_STACK in output, got: %s", output)
	}
	if !strings.Contains(output, "depth: 3") {
		t.Errorf("Expected 'depth: 3' in output, got: %s", output)
	}
	// Should show all contexts
	if !strings.Contains(output, "match") || !strings.Contains(output, "block") || !strings.Contains(output, "case") {
		t.Errorf("Expected all contexts (match, block, case) in stack output, got: %s", output)
	}
}

func TestDelimiterTracerShowStackEmpty(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Show empty stack (should not output anything)
	tracer.showStack()

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should be empty or minimal output
	if strings.Contains(output, "DELIM_STACK") {
		t.Errorf("Expected no DELIM_STACK output for empty stack, got: %s", output)
	}
}

func TestParserDelimiterTraceMethods(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Reset global tracer
	globalDelimiterTracer = &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	input := `func test() -> int { 42 }`
	l := lexer.New(input, "test.ail")
	p := New(l)

	// Test parser methods
	p.traceDelimiterOpen(delimCtxFunction)
	if len(globalDelimiterTracer.stack) != 1 {
		t.Errorf("Expected 1 frame after traceDelimiterOpen, got %d", len(globalDelimiterTracer.stack))
	}

	p.traceDelimiterClose(delimCtxFunction)
	if len(globalDelimiterTracer.stack) != 0 {
		t.Errorf("Expected empty stack after traceDelimiterClose, got %d", len(globalDelimiterTracer.stack))
	}

	// Test traceDelimiterStack (should not panic)
	p.traceDelimiterOpen(delimCtxMatch)
	p.traceDelimiterStack()
	p.traceDelimiterClose(delimCtxMatch)
}

func TestDelimiterTracerAllContextTypes(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Test all context types
	contexts := []delimiterContext{
		delimCtxMatch,
		delimCtxBlock,
		delimCtxFunction,
		delimCtxLambda,
		delimCtxRecord,
		delimCtxList,
		delimCtxCase,
	}

	for i, ctx := range contexts {
		tracer.push(ctx, i+1, 5)
		if len(tracer.stack) != i+1 {
			t.Errorf("Expected %d frames after pushing %s, got %d", i+1, ctx, len(tracer.stack))
		}
		if tracer.stack[i].context != ctx {
			t.Errorf("Expected context=%s, got %s", ctx, tracer.stack[i].context)
		}
	}

	// Pop all in reverse order
	for i := len(contexts) - 1; i >= 0; i-- {
		ctx := contexts[i]
		tracer.pop(ctx, i+1, 10)
		if len(tracer.stack) != i {
			t.Errorf("Expected %d frames after popping %s, got %d", i, ctx, len(tracer.stack))
		}
	}
}

func TestDelimiterTracerTokenTracing(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Reset global tracer
	globalDelimiterTracer = &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	input := `func test() -> int { 42 }`
	l := lexer.New(input, "test.ail")
	p := New(l)

	// Advance to LBRACE token
	for !p.curTokenIs(lexer.LBRACE) && !p.curTokenIs(lexer.EOF) {
		p.nextToken()
	}

	// Trace LBRACE token
	p.traceDelimiterToken(lexer.LBRACE, "consume")

	// Advance to RBRACE
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		p.nextToken()
	}

	// Trace RBRACE token
	p.traceDelimiterToken(lexer.RBRACE, "consume")

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for token trace messages
	if !strings.Contains(output, "DELIM_TOKEN") {
		t.Errorf("Expected DELIM_TOKEN in output, got: %s", output)
	}
	if !strings.Contains(output, "consume {") || !strings.Contains(output, "consume }") {
		t.Errorf("Expected both '{' and '}' token traces, got: %s", output)
	}
}

func TestDelimiterTracerTokenTracingNonBrace(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Reset global tracer
	globalDelimiterTracer = &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	input := `func test() -> int { 42 }`
	l := lexer.New(input, "test.ail")
	p := New(l)

	// Trace non-brace token (should not output anything)
	p.traceDelimiterToken(lexer.INT, "consume")
	p.traceDelimiterToken(lexer.IDENT, "consume")
	p.traceDelimiterToken(lexer.ARROW, "consume")

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should not contain token traces for non-brace tokens
	if strings.Contains(output, "DELIM_TOKEN") {
		t.Errorf("Expected no DELIM_TOKEN output for non-brace tokens, got: %s", output)
	}
}

func TestDelimiterTracerIntegrationWithParser(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Reset global tracer
	globalDelimiterTracer = &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Parse a nested match expression
	input := `
module test

func example(x: int) -> int {
  match x {
    0 => {
      match x {
        0 => 0
      }
    }
    _ => 1
  }
}
`
	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.ParseFile()

	// Check parser succeeded
	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("Parser error: %s", err)
		}
		t.Fatalf("Parser had %d errors", len(p.Errors()))
	}

	if program == nil {
		t.Fatal("Expected program, got nil")
	}

	// Check global tracer stack is empty (all delimiters matched)
	if len(globalDelimiterTracer.stack) != 0 {
		t.Errorf("Expected empty delimiter stack after parsing, got %d frames", len(globalDelimiterTracer.stack))
		for i, frame := range globalDelimiterTracer.stack {
			t.Errorf("  Frame %d: %s at %d:%d (depth=%d)", i, frame.context, frame.line, frame.col, frame.depth)
		}
	}
}

func TestDelimiterTracerDepthCalculation(t *testing.T) {
	// Save original state
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Verify depth calculation at each level
	for depth := 0; depth < 10; depth++ {
		tracer.push(delimCtxMatch, depth+1, 5)

		if len(tracer.stack) != depth+1 {
			t.Errorf("Expected stack length %d, got %d", depth+1, len(tracer.stack))
		}

		frame := tracer.stack[depth]
		if frame.depth != depth {
			t.Errorf("Expected depth=%d for frame %d, got %d", depth, depth, frame.depth)
		}
	}

	// Pop all and verify depth decreases correctly
	for depth := 9; depth >= 0; depth-- {
		if len(tracer.stack) == 0 {
			t.Fatalf("Unexpected empty stack at depth %d", depth)
		}

		topFrame := tracer.stack[len(tracer.stack)-1]
		if topFrame.depth != depth {
			t.Errorf("Expected top frame depth=%d, got %d", depth, topFrame.depth)
		}

		tracer.pop(delimCtxMatch, depth+1, 10)
	}

	if len(tracer.stack) != 0 {
		t.Errorf("Expected empty stack after popping all, got %d frames", len(tracer.stack))
	}
}

func TestDelimiterTracerOutputFormat(t *testing.T) {
	// Save original state and stderr
	origEnv := os.Getenv("DEBUG_DELIMITERS")
	defer os.Setenv("DEBUG_DELIMITERS", origEnv)
	origStderr := os.Stderr

	// Enable tracer
	os.Setenv("DEBUG_DELIMITERS", "1")

	// Create pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create new tracer
	tracer := &delimiterTracer{
		enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
		stack:   []delimiterFrame{},
	}

	// Push and pop to generate output
	tracer.push(delimCtxMatch, 5, 10)
	tracer.push(delimCtxBlock, 6, 15)
	tracer.pop(delimCtxBlock, 6, 20)
	tracer.pop(delimCtxMatch, 5, 25)

	// Close writer and restore stderr
	w.Close()
	os.Stderr = origStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output format contains expected patterns
	expectedPatterns := []string{
		"[DELIM_OPEN match]",
		"match { at 5:10",
		"[DELIM_OPEN block]",
		"block { at 6:15",
		"[DELIM_CLOSE block]",
		"block } at 6:20",
		"[DELIM_CLOSE match]",
		"match } at 5:25",
		"depth=0",
		"depth=1",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected output to contain %q, got:\n%s", pattern, output)
		}
	}

	// Verify indentation for nested context (depth=1 should have 2 spaces)
	lines := strings.Split(output, "\n")
	foundIndentedBlock := false
	for _, line := range lines {
		if strings.Contains(line, "DELIM_OPEN block") || strings.Contains(line, "DELIM_CLOSE block") {
			// Block is at depth 1, should have "  block" (2 spaces)
			if strings.Contains(line, "  block") {
				foundIndentedBlock = true
				break
			}
		}
	}
	if !foundIndentedBlock {
		t.Errorf("Expected to find indented block output (depth=1 with 2 spaces), got:\n%s", output)
	}
}

func ExampleDelimiterTracer() {
	// Enable delimiter tracing
	os.Setenv("DEBUG_DELIMITERS", "1")
	defer os.Setenv("DEBUG_DELIMITERS", "0")

	// Reset global tracer
	globalDelimiterTracer = &delimiterTracer{
		enabled: true,
		stack:   []delimiterFrame{},
	}

	// Parse a simple match expression
	input := `
module test

func example(x: int) -> int {
  match x {
    0 => 0
    _ => 1
  }
}
`
	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.ParseFile()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Println("  ", err)
		}
	} else if program != nil {
		fmt.Println("Parse succeeded")
		fmt.Printf("Delimiter stack empty: %v\n", len(globalDelimiterTracer.stack) == 0)
	}

	// Output:
	// Parse succeeded
	// Delimiter stack empty: true
}
