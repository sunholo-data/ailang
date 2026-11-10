// Package server provides the HTTP server for the Collaboration Hub.
// ailang_bridge.go provides AILANG integration for dashboard transforms.
package server

import (
	"fmt"
	"log"
	"sync"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/embed"
)

// AILANGBridge runs the dashboard's transforms, which are written in AILANG.
//
// It used to be a SWITCH between two implementations: AILANG when
// AILANG_DASHBOARD=1, a Go copy otherwise, plus a silent fall back to that copy
// whenever an AILANG call errored. Removed 2026-09-08 on Mark's instruction,
// because a second implementation of a rule is not a safety net:
//
//   - the copies were free to drift, and nothing compared them;
//   - the substitution was visible only in a log line nobody reads, so a broken
//     AILANG path looked exactly like a working one;
//   - `budget_checker` decides SPENDING. CLAUDE.md Principle 2 puts business
//     logic explicitly outside what may be quietly defaulted;
//   - and the flag was off in production while the deployed image shipped no
//     `.ail` files at all — so the AILANG path had never run there, and the
//     "fallback" was in fact the only thing that had ever executed.
//
// Now there is one implementation. Every method returns an error the caller must
// handle, and a transform that cannot run says so.
type AILANGBridge struct {
	engine *embed.Engine
	err    error
	mu     sync.RWMutex
}

var (
	ailangBridge     *AILANGBridge
	ailangBridgeOnce sync.Once
)

// GetAILANGBridge returns the singleton AILANG bridge instance.
func GetAILANGBridge() *AILANGBridge {
	ailangBridgeOnce.Do(func() {
		// Resolve against a module we actually call, so a tree that is present
		// but missing the transforms fails here rather than on first request.
		eng, err := embed.NewForModule(moduleEventFormatter)
		ailangBridge = &AILANGBridge{engine: eng, err: err}
		if err != nil {
			log.Printf("[AILANG] dashboard transforms UNAVAILABLE: %v", err)
			return
		}
		log.Printf("[AILANG] dashboard transforms ready")
	})
	return ailangBridge
}

// AILANG module paths, named once so a typo is a compile error rather than a
// runtime miss that used to be absorbed by a fallback.
// The heatmap and budget transforms the UI's panels used are no longer called
// from here; budget_checker.ail is still `ailang budget`'s (cmd/ailang/budget.go).
const moduleEventFormatter = "internal/dashboard_transforms/event_formatter"

// Ready reports whether the transforms can run, and why not if they cannot.
//
// Replaces IsEnabled. The old name asked whether a FEATURE was switched on; the
// question that matters now is whether the only implementation is reachable.
func (b *AILANGBridge) Ready() error {
	if b == nil {
		return fmt.Errorf("AILANG bridge not initialised")
	}
	if b.err != nil {
		return b.err
	}
	if b.engine == nil {
		return fmt.Errorf("AILANG engine not loaded")
	}
	return nil
}

// SummarizeEvents calls the AILANG summarizeEvents function.
func (b *AILANGBridge) SummarizeEvents(events []*coordinator.TaskEventRecord) (string, error) {
	if err := b.Ready(); err != nil {
		return "", err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "summarizeEvents", convertEventsForAILANG(events))
	if err != nil {
		return "", fmt.Errorf("summarizeEvents: %w", err)
	}
	str, err := embed.ToString(result)
	if err != nil {
		return "", fmt.Errorf("summarizeEvents result: %w", err)
	}
	return str, nil
}

// CountTurns calls the AILANG countTurns function.
func (b *AILANGBridge) CountTurns(events []*coordinator.TaskEventRecord) (int, error) {
	if err := b.Ready(); err != nil {
		return 0, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "countTurns", convertEventsForAILANG(events))
	if err != nil {
		return 0, fmt.Errorf("countTurns: %w", err)
	}
	count, err := embed.ToInt(result)
	if err != nil {
		return 0, fmt.Errorf("countTurns result: %w", err)
	}
	return count, nil
}

// Truncate calls the AILANG truncate function.
func (b *AILANGBridge) Truncate(text string, maxLen int) (string, error) {
	if err := b.Ready(); err != nil {
		return "", err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "truncate", text, maxLen)
	if err != nil {
		return "", fmt.Errorf("truncate: %w", err)
	}
	str, err := embed.ToString(result)
	if err != nil {
		return "", fmt.Errorf("truncate result: %w", err)
	}
	return str, nil
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
