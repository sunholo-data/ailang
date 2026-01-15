// Package server provides the HTTP server for the Collaboration Hub.
// ailang_bridge.go provides AILANG integration for dashboard transforms.
package server

import (
	"log"
	"os"
	"sync"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/embed"
)

// AILANGBridge provides AILANG-based event formatting as an alternative to Go.
// Enable with AILANG_DASHBOARD=1 environment variable.
type AILANGBridge struct {
	engine  *embed.Engine
	enabled bool
	mu      sync.RWMutex
}

var (
	ailangBridge     *AILANGBridge
	ailangBridgeOnce sync.Once
)

// GetAILANGBridge returns the singleton AILANG bridge instance.
func GetAILANGBridge() *AILANGBridge {
	ailangBridgeOnce.Do(func() {
		enabled := os.Getenv("AILANG_DASHBOARD") == "1"
		ailangBridge = &AILANGBridge{
			enabled: enabled,
		}
		if enabled {
			// Find AILANG project root (assumes server runs from project root or near it)
			basePath := os.Getenv("AILANG_PROJECT_ROOT")
			if basePath == "" {
				// Default to current working directory
				basePath, _ = os.Getwd()
			}
			ailangBridge.engine = embed.New(basePath)
			log.Printf("[AILANG] Dashboard bridge enabled (basePath=%s)", basePath)
		}
	})
	return ailangBridge
}

// IsEnabled returns true if the AILANG bridge is enabled.
func (b *AILANGBridge) IsEnabled() bool {
	if b == nil {
		return false
	}
	return b.enabled
}

// SummarizeEvents calls the AILANG summarizeEvents function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) SummarizeEvents(events []*coordinator.TaskEventRecord) string {
	if !b.IsEnabled() || b.engine == nil {
		return coordinator.SummarizeEvents(events)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert events to AILANG-compatible format
	ailangEvents := convertEventsForAILANG(events)

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"summarizeEvents",
		ailangEvents,
	)
	if err != nil {
		log.Printf("[AILANG] SummarizeEvents failed, falling back to Go: %v", err)
		return coordinator.SummarizeEvents(events)
	}

	str, err := embed.ToString(result)
	if err != nil {
		log.Printf("[AILANG] SummarizeEvents result conversion failed: %v", err)
		return coordinator.SummarizeEvents(events)
	}

	return str
}

// CountTurns calls the AILANG countTurns function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) CountTurns(events []*coordinator.TaskEventRecord) int {
	if !b.IsEnabled() || b.engine == nil {
		return coordinator.CountTurns(events)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	ailangEvents := convertEventsForAILANG(events)

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"countTurns",
		ailangEvents,
	)
	if err != nil {
		log.Printf("[AILANG] CountTurns failed, falling back to Go: %v", err)
		return coordinator.CountTurns(events)
	}

	count, err := embed.ToInt(result)
	if err != nil {
		log.Printf("[AILANG] CountTurns result conversion failed: %v", err)
		return coordinator.CountTurns(events)
	}

	return count
}

// Truncate calls the AILANG truncate function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) Truncate(text string, maxLen int) string {
	if !b.IsEnabled() || b.engine == nil {
		return goTruncate(text, maxLen)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"truncate",
		text,
		maxLen,
	)
	if err != nil {
		log.Printf("[AILANG] Truncate failed, falling back to Go: %v", err)
		return goTruncate(text, maxLen)
	}

	str, err := embed.ToString(result)
	if err != nil {
		log.Printf("[AILANG] Truncate result conversion failed: %v", err)
		return goTruncate(text, maxLen)
	}

	return str
}

// goTruncate is the Go fallback for truncate.
func goTruncate(text string, maxLen int) string {
	if maxLen == 0 || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// ailangEvent is the struct format expected by the AILANG event_formatter module.
type ailangEvent struct {
	TurnNum    int    `json:"turnNum"`
	StreamType string `json:"streamType"`
	Text       string `json:"text"`
}

// convertEventsForAILANG converts coordinator events to AILANG-compatible format.
func convertEventsForAILANG(events []*coordinator.TaskEventRecord) []ailangEvent {
	result := make([]ailangEvent, len(events))
	for i, e := range events {
		result[i] = ailangEvent{
			TurnNum:    e.TurnNum,
			StreamType: e.StreamType,
			Text:       e.Text,
		}
	}
	return result
}

// Close shuts down the AILANG engine.
func (b *AILANGBridge) Close() error {
	if b.engine != nil {
		return b.engine.Close()
	}
	return nil
}
