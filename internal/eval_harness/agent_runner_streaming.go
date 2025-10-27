package eval_harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// runHeadlessSessionStreaming executes Claude in headless mode with real-time message streaming
// This is used when DEBUG_AGENT=1 to provide visibility into what Claude is doing
func runHeadlessSessionStreaming(prompt, workspace string, config AgentBenchmarkConfig) (*ClaudeHeadlessResult, error) {
	// Generate UUID for session ID (Claude CLI requires valid UUID)
	sessionID := uuid.New().String()

	// Build command with stream-json for real-time NDJSON events
	// --output-format stream-json: Get structured JSON events as they happen
	// --include-partial-messages: Show thinking process token by token
	// --permission-mode bypassPermissions: Skip approval prompts for automated execution
	// --verbose: Show detailed execution information
	cmd := exec.Command(config.ClaudePath, "-p", prompt,
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--verbose",
		"--permission-mode", "bypassPermissions",
		"--model", config.ClaudeModel,
		"--session-id", sessionID,
		"--add-dir", workspace,
		"--allowedTools", strings.Join(config.AllowedTools, ","),
	)

	cmd.Dir = workspace

	// Claude's stream-json output goes to stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Stderr for system errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] ========== CLAUDE SESSION START ==========\n")
	fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace: %s\n", workspace)
	fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Session ID: %s\n", sessionID)
	fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Using stream-json for real-time visibility\n\n")

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Set up timeout
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Parse stream-json output
	done := make(chan error, 1)
	var finalResult *ClaudeHeadlessResult
	var currentMessage strings.Builder
	var turnNum int

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Read stderr in background (for system errors)
		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				fmt.Fprintf(os.Stderr, "[STDERR] %s\n", line)
			}
		}()

		// Parse NDJSON from stdout
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			if line == "" {
				continue
			}

			// Parse JSON event
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				// Skip malformed lines
				continue
			}

			eventType, _ := event["type"].(string)

			switch eventType {
			case "system":
				// System init event - already logged

			case "stream_event":
				// Real-time streaming events from API
				streamEvent, _ := event["event"].(map[string]interface{})
				if streamEvent == nil {
					continue
				}

				streamType, _ := streamEvent["type"].(string)
				switch streamType {
				case "message_start":
					turnNum++
					fmt.Fprintf(os.Stderr, "\n[TURN %d] ==========================================\n", turnNum)
					currentMessage.Reset()

				case "content_block_start":
					// New content block (text or tool use)
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						blockType, _ := contentBlock["type"].(string)
						if blockType == "tool_use" {
							toolName, _ := contentBlock["name"].(string)
							fmt.Fprintf(os.Stderr, "[TOOL USE] %s\n", toolName)
						}
					}

				case "content_block_delta":
					// Incremental content (text tokens or tool input)
					delta, _ := streamEvent["delta"].(map[string]interface{})
					if delta != nil {
						deltaType, _ := delta["type"].(string)
						if deltaType == "text_delta" {
							text, _ := delta["text"].(string)
							currentMessage.WriteString(text)
							// Print token by token for visibility
							fmt.Fprintf(os.Stderr, "%s", text)
						}
					}

				case "content_block_stop":
					// Content block finished
					if currentMessage.Len() > 0 {
						fmt.Fprintf(os.Stderr, "\n")
					}

				case "message_stop":
					// Turn finished
					if currentMessage.Len() > 0 {
						fmt.Fprintf(os.Stderr, "\n")
					}
				}

			case "assistant":
				// Full assistant message (for logging)
				// Already shown via stream_events, so skip

			case "user":
				// Tool result being sent back to Claude
				// Already visible via tool execution

			case "result":
				// Final result with metrics
				if err := json.Unmarshal([]byte(line), &finalResult); err != nil {
					done <- fmt.Errorf("failed to parse final result: %w", err)
					return
				}
				fmt.Fprintf(os.Stderr, "\n[DEBUG_AGENT] ========== SESSION COMPLETE ==========\n")
				fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Turns: %d\n", finalResult.NumTurns)
				fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Duration: %dms\n", finalResult.DurationMS)
				fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Cost: $%.4f\n", finalResult.TotalCostUSD)
			}
		}

		if err := stdoutScanner.Err(); err != nil {
			done <- fmt.Errorf("stdout scanner error: %w", err)
			return
		}

		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case <-timer.C:
		_ = cmd.Process.Kill()
		fmt.Fprintf(os.Stderr, "\n[ERROR] Claude session timed out after %d seconds\n", config.TimeoutSeconds)
		return nil, fmt.Errorf("claude session timed out after %d seconds", config.TimeoutSeconds)

	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[ERROR] Claude failed: %v\n", err)
			return nil, fmt.Errorf("claude failed: %w", err)
		}

		if finalResult == nil {
			// No structured result, but session completed
			return &ClaudeHeadlessResult{
				Type:     "result",
				IsError:  false,
				Result:   "Session completed (see workspace for solution.ail)",
				NumTurns: turnNum,
			}, nil
		}

		return finalResult, nil
	}
}
