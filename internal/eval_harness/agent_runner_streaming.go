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

// RunHeadlessSessionStreaming executes Claude in headless mode with real-time message streaming
// This is used when DEBUG_AGENT=1 to provide visibility into what Claude is doing
// systemPrompt contains language knowledge (loaded from prompts/versions.json)
// taskPrompt contains the benchmark task description
//
// Exported for use by internal/agent package for directive execution
func RunHeadlessSessionStreaming(spec *BenchmarkSpec, systemPrompt, taskPrompt, workspace string, config AgentBenchmarkConfig) (*ClaudeHeadlessResult, error) {
	// Generate UUID for session ID (Claude CLI requires valid UUID)
	sessionID := uuid.New().String()

	// Build command with stream-json for real-time NDJSON events
	// --system-prompt: Language knowledge (AILANG/Python syntax reference)
	// -p: Task prompt (benchmark description and expected output)
	// --output-format stream-json: Get structured JSON events as they happen
	// --include-partial-messages: Show thinking process token by token
	// --permission-mode bypassPermissions: Skip approval prompts for automated execution
	// --verbose: Show detailed execution information
	cmd := exec.Command(config.ClaudePath,
		"--system-prompt", systemPrompt,
		"-p", taskPrompt,
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

	// Set up timeout - use spec timeout if set, otherwise use config default
	timeoutSeconds := config.TimeoutSeconds
	if spec.Timeout > 0 {
		timeoutSeconds = spec.Timeout
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Parse stream-json output
	done := make(chan error, 1)
	var finalResult *ClaudeHeadlessResult
	var currentMessage strings.Builder
	var transcriptBuf strings.Builder // Accumulate full conversation for session log
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
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d] ==========================================\n", turnNum))
					currentMessage.Reset()

				case "content_block_start":
					// New content block (text or tool use)
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						blockType, _ := contentBlock["type"].(string)
						if blockType == "tool_use" {
							toolName, _ := contentBlock["name"].(string)
							fmt.Fprintf(os.Stderr, "[TOOL USE] %s\n", toolName)
							transcriptBuf.WriteString(fmt.Sprintf("[TOOL USE] %s\n", toolName))
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
							transcriptBuf.WriteString(text) // Also accumulate in transcript
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
		fmt.Fprintf(os.Stderr, "\n[ERROR] Claude session timed out after %d seconds\n", timeoutSeconds)

		// Build transcript for return (file write happens in caller, before workspace cleanup)
		transcript := fmt.Sprintf("=== Claude Session Log ===\n\nTask Prompt:\n%s\n\nTranscript:\n%s\n\nTimeout after %d seconds\n",
			taskPrompt, transcriptBuf.String(), timeoutSeconds)

		// Return partial result with what we captured before timeout
		return &ClaudeHeadlessResult{
			Type:       "result",
			Subtype:    "timeout",
			IsError:    true,
			Result:     fmt.Sprintf("Session timed out after %d seconds", timeoutSeconds),
			NumTurns:   turnNum,
			DurationMS: timeoutSeconds * 1000,
			SessionID:  sessionID,
			Transcript: transcript,
		}, nil // Return result, not error, so caller can still log partial data

	case err := <-done:
		// Build transcript for return (file write happens in caller, before workspace cleanup)
		transcript := fmt.Sprintf("=== Claude Session Log ===\n\nTask Prompt:\n%s\n\nTranscript:\n%s\n",
			taskPrompt, transcriptBuf.String())

		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[ERROR] Claude failed: %v\n", err)
			// Return partial result even on error, so we can capture transcript
			result := &ClaudeHeadlessResult{
				Type:       "result",
				Subtype:    "error",
				IsError:    true,
				Result:     fmt.Sprintf("Claude execution error: %v", err),
				NumTurns:   turnNum,
				DurationMS: int(time.Since(time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)).Milliseconds()),
				SessionID:  sessionID,
				Transcript: transcript,
			}
			return result, nil // Return result, not error, so caller can still log partial data
		}

		if finalResult == nil {
			// No structured result, but session completed
			return &ClaudeHeadlessResult{
				Type:       "result",
				IsError:    false,
				Result:     "Session completed (see workspace for solution.ail)",
				NumTurns:   turnNum,
				Transcript: transcript,
			}, nil
		}

		// Attach transcript to finalResult
		finalResult.Transcript = transcript
		return finalResult, nil
	}
}
