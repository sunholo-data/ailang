package main

import (
	"fmt"
	"strings"
)

// printUnifiedHierarchy prints the hierarchy as a tree
func printUnifiedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)

		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
		if h.Task.ParentTaskID != "" {
			fmt.Printf("  Parent: %s\n", h.Task.ParentTaskID)
		}
		fmt.Printf("  Cost: $%.4f | Tokens: %d in / %d out\n",
			h.Task.Cost, h.Task.TokensIn, h.Task.TokensOut)
	}

	// Print parent chain (ancestry)
	if len(h.ParentChain) > 0 {
		fmt.Printf("\n┌─ Parent Chain (%d levels):\n", len(h.ParentChain))
		for i, parent := range h.ParentChain {
			var prefix string
			if i == len(h.ParentChain)-1 {
				prefix = "└─ "
			} else {
				prefix = "├─ "
			}
			fmt.Printf("%s%s [%s]\n", prefix, parent.ID, parent.Status)
		}
	}

	// Print message info
	if h.Message != nil {
		fmt.Printf("\n📨 Message: %s\n", h.Message.ID)
		fmt.Printf("   From: %s → To: %s\n", h.Message.FromAgent, h.Message.ToInbox)
		if h.Message.ParentTaskID != "" {
			fmt.Printf("   Parent Task: %s\n", h.Message.ParentTaskID)
		}
	}

	// Print spans
	if len(h.Spans) > 0 {
		fmt.Printf("\n├─ Spans: %d total\n", h.Stats.TotalSpans)
		for i, span := range h.Spans {
			isLast := i == len(h.Spans)-1 && len(h.Handoffs) == 0
			printSpanTree(span, "│  ", isLast)
		}
	}

	// Print handoffs
	if len(h.Handoffs) > 0 {
		fmt.Printf("\n└─ Handoffs: %d\n", len(h.Handoffs))
		for i, handoff := range h.Handoffs {
			prefix := "   ├─ "
			if i == len(h.Handoffs)-1 {
				prefix = "   └─ "
			}
			agent := handoff.TargetAgent
			if agent == "" {
				agent = "?"
			}
			fmt.Printf("%s→ %s (%s) [%s]\n", prefix, handoff.TargetTaskID, agent, handoff.Status)
		}
	}

	// Print chat messages if available
	if len(h.ChatMessages) > 0 {
		fmt.Printf("\n💬 Conversation History: %d messages\n", len(h.ChatMessages))
		fmt.Println(strings.Repeat("─", 80))
		for _, msg := range h.ChatMessages {
			roleIcon := "👤"
			if msg.Role == "assistant" {
				roleIcon = "🤖"
			}
			fmt.Printf("\n%s [Turn %d] %s\n", roleIcon, msg.TurnNumber, msg.Role)
			if msg.ContentText != "" {
				// Truncate long messages for display
				content := msg.ContentText
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				fmt.Printf("   %s\n", strings.ReplaceAll(content, "\n", "\n   "))
			}
			if msg.ContentThinking != "" {
				thinking := msg.ContentThinking
				if len(thinking) > 200 {
					thinking = thinking[:200] + "..."
				}
				fmt.Printf("   💭 %s\n", strings.ReplaceAll(thinking, "\n", "\n   "))
			}
			if msg.TokensIn > 0 || msg.TokensOut > 0 {
				fmt.Printf("   📊 Tokens: %d in / %d out", msg.TokensIn, msg.TokensOut)
				if msg.Model != "" {
					fmt.Printf(" | Model: %s", msg.Model)
				}
				fmt.Println()
			}
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d spans | $%.4f | %d tokens | %s\n",
		h.Stats.TotalSpans,
		h.Stats.TotalCost,
		h.Stats.TotalTokens,
		formatHierarchyDuration(h.Stats.TotalDurationMs))
	if h.Stats.HandoffCount > 0 {
		fmt.Printf("       %d handoffs", h.Stats.HandoffCount)
		if h.Stats.PendingApprovals > 0 {
			fmt.Printf(" | %d pending approvals", h.Stats.PendingApprovals)
		}
		fmt.Println()
	}
}

// printTurnGroupedHierarchy prints the hierarchy organized by conversation turns.
func printTurnGroupedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Turn-Grouped Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info if available
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)
		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
	}

	tg := h.TurnGrouped
	if tg == nil {
		fmt.Println("\n(No turn data available)")
		return
	}

	// Print session info
	if tg.Session != nil {
		fmt.Printf("\n┌─ Session: %s\n", tg.Session.Name)
		if tg.Session.Provider != "" || tg.Session.Model != "" {
			fmt.Printf("│  Provider: %s | Model: %s\n", tg.Session.Provider, tg.Session.Model)
		}
		fmt.Printf("│  Duration: %s | Cost: $%.4f | Tokens: %d in / %d out\n",
			formatHierarchyDuration(tg.Session.DurationMs),
			tg.Session.Cost,
			tg.Session.TokensIn,
			tg.Session.TokensOut)
	}

	// Print turns
	if len(tg.Turns) > 0 {
		fmt.Printf("│\n")
		for i, turn := range tg.Turns {
			isLast := i == len(tg.Turns)-1

			// Turn header
			prefix := "├"
			nextPrefix := "│"
			if isLast {
				prefix = "└"
				nextPrefix = " "
			}

			fmt.Printf("%s─ Turn %d [%s] $%.4f [%d→%d tokens]\n",
				prefix,
				turn.TurnNumber,
				formatHierarchyDuration(turn.DurationMs),
				turn.Cost,
				turn.TokensIn,
				turn.TokensOut)

			// Print tools within turn
			if len(turn.Tools) > 0 {
				for j, tool := range turn.Tools {
					toolIsLast := j == len(turn.Tools)-1
					toolPrefix := "├─"
					if toolIsLast {
						toolPrefix = "└─"
					}

					statusIcon := "✓"
					if tool.Status == "error" {
						statusIcon = "✗"
					}

					toolName := tool.ToolName
					if toolName == "" {
						toolName = tool.Name
					}

					fmt.Printf("%s  %s %s %s [%s]\n",
						nextPrefix,
						toolPrefix,
						toolName,
						statusIcon,
						formatHierarchyDuration(tool.DurationMs))
				}
			}

			// Add spacing between turns
			if !isLast {
				fmt.Printf("%s\n", nextPrefix)
			}
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d turns | %d tools | $%.4f | %d tokens | %s\n",
		tg.Stats.TotalTurns,
		tg.Stats.TotalTools,
		tg.Stats.TotalCost,
		tg.Stats.TotalTokens,
		formatHierarchyDuration(tg.Stats.DurationMs))

	// Print handoffs if any
	if len(h.Handoffs) > 0 {
		fmt.Printf("       %d handoffs\n", len(h.Handoffs))
	}
}

// printSpanTree prints a span and its children as a tree
func printSpanTree(span *unifiedSpan, prefix string, isLast bool) {
	if span == nil {
		return
	}

	connector := "├─ "
	if isLast {
		connector = "└─ "
	}

	// Format metrics
	metrics := ""
	if span.DurationMs > 0 {
		metrics += fmt.Sprintf(" %s", formatHierarchyDuration(span.DurationMs))
	}
	if span.Cost > 0 {
		metrics += fmt.Sprintf(" $%.4f", span.Cost)
	}
	if span.TokensIn > 0 || span.TokensOut > 0 {
		metrics += fmt.Sprintf(" [%d→%d]", span.TokensIn, span.TokensOut)
	}

	fmt.Printf("%s%s%s%s\n", prefix, connector, span.Name, metrics)

	// Print children
	childPrefix := prefix
	if isLast {
		childPrefix += "   "
	} else {
		childPrefix += "│  "
	}

	for i, child := range span.Children {
		printSpanTree(child, childPrefix, i == len(span.Children)-1)
	}
}

// formatHierarchyDuration formats milliseconds as human-readable duration for hierarchy output
func formatHierarchyDuration(ms int64) string {
	if ms >= 60000 {
		return fmt.Sprintf("%.1fm", float64(ms)/60000)
	} else if ms >= 1000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%dms", ms)
}
