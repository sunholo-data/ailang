package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func chainsChatCommand() {
	fs := flag.NewFlagSet("chains chat", flag.ExitOnError)
	stageNum := fs.Int("stage", 0, "Show chat for specific stage number (1-indexed)")
	compact := fs.Bool("compact", false, "Compact one-line-per-turn view")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	limit := fs.Int("limit", 0, "Limit number of turns shown")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains chat <chain-id> [options]")
		fmt.Println()
		fmt.Println("Show turn-by-turn conversation for a chain stage.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --stage N     Show chat for specific stage (1-indexed)")
		fmt.Println("  --compact     One-line summary per turn")
		fmt.Println("  --json        JSON output")
		fmt.Println("  --limit N     Limit number of turns shown")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang chains chat 47bef0bc               # All stages")
		fmt.Println("  ailang chains chat 47bef0bc --stage 3     # Stage 3 only")
		fmt.Println("  ailang chains chat 47bef0bc --compact     # One-line summaries")
		os.Exit(1)
	}

	chainPrefix := fs.Arg(0)

	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve chain ID from prefix
	chainID, err := resolveChainID(backend, ctx, chainPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get stages to find session IDs
	stages, err := backend.GetChainStages(ctx, chainID, observatory.ChainReadOptions{IncludeStages: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain stages: %v\n", err)
		os.Exit(1)
	}

	if len(stages) == 0 {
		fmt.Fprintln(os.Stderr, "No stages found for this chain.")
		os.Exit(1)
	}

	// Filter by stage number if specified
	if *stageNum > 0 {
		if *stageNum > len(stages) {
			fmt.Fprintf(os.Stderr, "Error: stage %d does not exist (chain has %d stages)\n", *stageNum, len(stages))
			os.Exit(1)
		}
		stages = []*observatory.ChainStage{stages[*stageNum-1]}
	}

	// Collect and display chat messages for each stage
	for i, stage := range stages {
		if stage.SessionID == "" {
			continue
		}

		messages, err := backend.GetChatMessagesBySession(ctx, stage.SessionID, time.Time{}, time.Time{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get chat for session %s: %v\n", stage.SessionID[:12], err)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		if *jsonOutput {
			outputChatJSON(stage, messages)
			continue
		}

		// Header
		if i > 0 {
			fmt.Println()
		}
		stageLabel := ""
		if stage.EvalAssessment != nil && stage.EvalAssessment.BenchmarkID != "" {
			stageLabel = stage.EvalAssessment.BenchmarkID
		}
		if stageLabel == "" {
			stageLabel = stage.AgentID
		}

		sessionShort := stage.SessionID
		if len(sessionShort) > 12 {
			sessionShort = sessionShort[:12] + "..."
		}
		fmt.Printf("Session: %s (Stage %d: %s)\n", sessionShort, stage.StageNumber, stageLabel)
		fmt.Printf("%d messages, %d turns, %d tool calls\n", len(messages), stage.Turns, stage.ToolCalls)
		fmt.Println()

		if *compact {
			chatOutputCompact(messages, *limit)
		} else {
			chatOutputFull(messages, *limit)
		}
	}
}

func outputChatJSON(stage *observatory.ChainStage, messages []*observatory.ChatMessage) {
	type chatOutput struct {
		SessionID string                     `json:"session_id"`
		Stage     int                        `json:"stage"`
		Messages  []*observatory.ChatMessage `json:"messages"`
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(chatOutput{
		SessionID: stage.SessionID,
		Stage:     stage.StageNumber,
		Messages:  messages,
	})
}

func chatOutputCompact(messages []*observatory.ChatMessage, limit int) {
	shown := 0
	for _, msg := range messages {
		if limit > 0 && shown >= limit {
			fmt.Printf("  ... (%d more messages)\n", len(messages)-shown)
			break
		}

		turnStr := fmt.Sprintf("T%-3d", msg.TurnNumber)
		roleStr := fmt.Sprintf("%-9s", msg.Role)

		summary := chatSummaryLine(msg)
		if summary == "" {
			continue // skip empty/null messages
		}

		fmt.Printf("  %s %s %s\n", turnStr, roleStr, summary)
		shown++
	}
}

func chatOutputFull(messages []*observatory.ChatMessage, limit int) {
	shown := 0
	lastTurn := -1

	for _, msg := range messages {
		if limit > 0 && shown >= limit {
			fmt.Printf("\n  ... (%d more messages)\n", len(messages)-shown)
			break
		}

		// Parse content
		blocks := parseContentJSON(msg.ContentJSON)
		if len(blocks) == 0 {
			continue
		}

		// Turn header (only on new turn)
		if msg.TurnNumber != lastTurn {
			if lastTurn >= 0 {
				fmt.Println()
			}
			fmt.Printf("─── Turn %d (%s) ───\n", msg.TurnNumber, msg.Role)
			lastTurn = msg.TurnNumber
		}

		for _, block := range blocks {
			switch block.Type {
			case "text":
				text := block.Text
				if len(text) > 300 {
					text = text[:300] + "..."
				}
				fmt.Println(text)
			case "tool_use":
				if block.ToolUse != nil {
					inputPreview := chatToolInputPreview(block.ToolUse.Input, 100)
					if inputPreview != "" {
						fmt.Printf("  [tool] %s: %s\n", block.ToolUse.Name, inputPreview)
					} else {
						fmt.Printf("  [tool] %s\n", block.ToolUse.Name)
					}
				}
			case "tool_result":
				if block.ToolResult != nil {
					resultPreview := block.ToolResult.Content
					if len(resultPreview) > 200 {
						resultPreview = resultPreview[:200] + "..."
					}
					if block.ToolResult.IsError {
						fmt.Printf("  [result:ERROR] %s\n", resultPreview)
					} else {
						fmt.Printf("  [result] %s\n", resultPreview)
					}
				}
			}
		}

		shown++
	}
}

func chatSummaryLine(msg *observatory.ChatMessage) string {
	blocks := parseContentJSON(msg.ContentJSON)
	if len(blocks) == 0 {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			text := b.Text
			if len(text) > 80 {
				text = text[:80] + "..."
			}
			text = strings.ReplaceAll(text, "\n", " ")
			parts = append(parts, text)
		case "tool_use":
			if b.ToolUse != nil {
				input := chatToolInputPreview(b.ToolUse.Input, 60)
				if input != "" {
					parts = append(parts, fmt.Sprintf("[%s: %s]", b.ToolUse.Name, input))
				} else {
					parts = append(parts, fmt.Sprintf("[%s]", b.ToolUse.Name))
				}
			}
		case "tool_result":
			if b.ToolResult != nil {
				if b.ToolResult.IsError {
					errMsg := b.ToolResult.Content
					if len(errMsg) > 80 {
						errMsg = errMsg[:80] + "..."
					}
					errMsg = strings.ReplaceAll(errMsg, "\n", " ")
					parts = append(parts, fmt.Sprintf("→ ERROR: %s", errMsg))
				} else {
					content := b.ToolResult.Content
					if len(content) > 60 {
						content = content[:60] + "..."
					}
					content = strings.ReplaceAll(content, "\n", " ")
					parts = append(parts, fmt.Sprintf("→ %s", content))
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

func chatToolInputPreview(input interface{}, maxLen int) string {
	m, ok := input.(map[string]interface{})
	if !ok {
		return ""
	}

	// For common tools, extract the most useful field
	if cmd, ok := m["command"]; ok {
		s := fmt.Sprintf("%v", cmd)
		if len(s) > maxLen {
			s = s[:maxLen] + "..."
		}
		return s
	}
	if fp, ok := m["file_path"]; ok {
		s := fmt.Sprintf("%v", fp)
		// Show just filename
		parts := strings.Split(s, "/")
		return parts[len(parts)-1]
	}
	if content, ok := m["content"]; ok {
		s := fmt.Sprintf("%v", content)
		if len(s) > maxLen {
			return fmt.Sprintf("(%d chars)", len(s))
		}
		return s
	}

	return ""
}
