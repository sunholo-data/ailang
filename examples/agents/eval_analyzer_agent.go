package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
	"github.com/sunholo/ailang/internal/agentrunner"
)

// EvalAnalyzerAgent demonstrates bridging a Claude agent concept with the protocol.
//
// This agent:
// 1. Receives messages requesting eval analysis
// 2. Reads eval results (simulated)
// 3. Analyzes failures
// 4. Creates design docs (simulated)
// 5. Sends response
//
// Usage:
//   go run examples/agents/eval_analyzer_agent.go
//
// Send it a message:
//   go run examples/agents/send_message.go eval-analyzer '{
//     "action": "analyze_failures",
//     "eval_results": "eval_results/latest.json"
//   }'
func main() {
	stateDir := ".ailang/state"
	if len(os.Args) > 1 {
		stateDir = os.Args[1]
	}

	fmt.Println("🔍 Eval Analyzer Agent Starting...")
	fmt.Printf("   State dir: %s\n", stateDir)
	fmt.Printf("   Agent ID: eval-analyzer\n")
	fmt.Printf("   Capabilities: analyze failures, create design docs\n\n")

	// Create handler (simulates what ClaudeAgentHandler will do)
	handler := agentrunner.NewFunctionHandler(evalAnalyzerHandler)

	// Configure runner
	config := &agentrunner.AgentConfig{
		AgentID:       "eval-analyzer",
		StateDir:      stateDir,
		PollInterval:  3 * time.Second,
		LeaseDuration: 120,
		Handler:       handler,
		OnError: func(err error) {
			log.Printf("❌ Error: %v", err)
		},
	}

	// Create and start runner
	runner, err := agentrunner.NewRunner(config)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	defer runner.Stop()

	log.Println("✓ Eval analyzer started. Press Ctrl+C to stop.\n")
	log.Println("Capabilities:")
	log.Println("  - analyze_failures: Analyze eval benchmark failures")
	log.Println("  - report_dx_friction: Report developer experience issues")
	log.Println()

	// Run
	if err := runner.Run(); err != nil {
		log.Fatalf("Runner failed: %v", err)
	}
}

// evalAnalyzerHandler processes messages for the eval-analyzer agent.
// This simulates what the ClaudeAgentHandler will eventually do.
func evalAnalyzerHandler(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	log.Printf("📨 Received message from %s", msg.FromAgent)
	log.Printf("   Action: %v", msg.Payload["action"])

	action, ok := msg.Payload["action"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'action' in payload")
	}

	switch action {
	case "analyze_failures":
		return analyzeFailures(msg)
	case "report_dx_friction":
		return reportDXFriction(msg)
	default:
		return map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("unknown action: %s", action),
		}, nil
	}
}

// analyzeFailures simulates analyzing eval failures and creating design docs.
func analyzeFailures(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	log.Println("🔍 Analyzing eval failures...")

	// Simulate reading eval results
	evalResultsPath, _ := msg.Payload["eval_results"].(string)
	if evalResultsPath == "" {
		evalResultsPath = "eval_results/latest.json"
	}

	log.Printf("   Reading: %s", evalResultsPath)
	time.Sleep(500 * time.Millisecond) // Simulate work

	// Simulate finding failures
	failures := []map[string]interface{}{
		{
			"benchmark": "list_comprehension",
			"error":     "missing builtin: list.map",
			"priority":  "high",
		},
		{
			"benchmark": "import_resolution",
			"error":     "module not found",
			"priority":  "medium",
		},
		{
			"benchmark": "type_inference",
			"error":     "type variable not resolved",
			"priority":  "high",
		},
	}

	log.Printf("   Found %d failures", len(failures))

	// Simulate creating design doc
	designDocPath := "design_docs/planned/M-DX9-fix-eval-failures.md"
	log.Printf("   Creating design doc: %s", designDocPath)

	designDoc := fmt.Sprintf(`# M-DX9: Fix Eval Failures

**Generated**: %s
**Failures Analyzed**: %d

## High Priority Issues

1. **list_comprehension** - Missing builtin: list.map
   - Add list.map, list.filter, list.reduce builtins
   - Estimated effort: 2 hours

2. **type_inference** - Type variable not resolved
   - Improve type inference for nested expressions
   - Estimated effort: 4 hours

## Medium Priority Issues

1. **import_resolution** - Module not found
   - Fix module path resolution
   - Estimated effort: 1 hour

## Recommendation

Implement high-priority fixes first for maximum impact.
`, time.Now().Format(time.RFC3339), len(failures))

	// Simulate writing file
	time.Sleep(300 * time.Millisecond)
	log.Printf("✅ Design doc created")

	// Return response
	return map[string]interface{}{
		"status":            "completed",
		"design_doc":        designDocPath,
		"design_doc_content": designDoc,
		"failures_analyzed": len(failures),
		"failures":          failures,
		"high_priority":     2,
		"medium_priority":   1,
		"recommendation":    "Implement high-priority fixes first",
	}, nil
}

// reportDXFriction simulates reporting developer experience friction.
func reportDXFriction(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	log.Println("📝 Reporting DX friction...")

	frictionType, _ := msg.Payload["friction_type"].(string)
	description, _ := msg.Payload["description"].(string)
	severity, _ := msg.Payload["severity"].(string)

	if frictionType == "" {
		frictionType = "unknown"
	}
	if severity == "" {
		severity = "medium"
	}

	log.Printf("   Type: %s", frictionType)
	log.Printf("   Severity: %s", severity)
	log.Printf("   Description: %s", description)

	// Simulate recording friction in database
	time.Sleep(200 * time.Millisecond)

	frictionReport := map[string]interface{}{
		"id":            fmt.Sprintf("friction_%d", time.Now().Unix()),
		"friction_type": frictionType,
		"severity":      severity,
		"description":   description,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"ailang_version": "v0.3.18",
	}

	reportJSON, _ := json.MarshalIndent(frictionReport, "", "  ")
	log.Printf("✅ Friction report created:\n%s", string(reportJSON))

	return map[string]interface{}{
		"status": "recorded",
		"report": frictionReport,
		"message": "DX friction recorded successfully",
	}, nil
}
