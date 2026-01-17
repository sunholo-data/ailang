package dashboard_transforms

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/embed"
)

// Event represents a task event for benchmarking
type Event struct {
	TurnNum    int    `json:"turnNum"`
	StreamType string `json:"streamType"`
	Text       string `json:"text"`
}

// Go implementations (baseline)

func goTruncate(text string, maxLen int) string {
	if maxLen == 0 || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func goCountTurns(events []Event) int {
	maxTurn := 0
	for _, e := range events {
		if e.TurnNum > maxTurn {
			maxTurn = e.TurnNum
		}
	}
	return maxTurn
}

//nolint:unused // Test helper for future filter tests
func goFilterByType(events []Event, typeFilter []string) []Event {
	if len(typeFilter) == 0 {
		return events
	}
	typeSet := make(map[string]bool)
	for _, t := range typeFilter {
		typeSet[t] = true
	}
	result := make([]Event, 0, len(events))
	for _, e := range events {
		if typeSet[e.StreamType] {
			result = append(result, e)
		}
	}
	return result
}

func goSummarize(events []Event) string {
	turns := goCountTurns(events)
	toolCount := 0
	textLen := 0
	for _, e := range events {
		if e.StreamType == "tool_use" {
			toolCount++
		}
		if e.StreamType == "text" {
			textLen += len(e.Text)
		}
	}
	return strings.Join([]string{
		itoa(turns), " turns, ",
		itoa(toolCount), " tool calls, ",
		itoa(textLen), " chars of text",
	}, "")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Test data generators

func generateEvents(n int) []Event {
	events := make([]Event, n)
	types := []string{"text", "tool_use", "tool_result", "turn_start"}
	for i := 0; i < n; i++ {
		events[i] = Event{
			TurnNum:    (i / 10) + 1,
			StreamType: types[i%4],
			Text:       strings.Repeat("x", 50),
		}
	}
	return events
}

// Benchmarks

func BenchmarkTruncate_Go(b *testing.B) {
	text := strings.Repeat("hello world ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goTruncate(text, 50)
	}
}

func BenchmarkTruncate_AILANG(b *testing.B) {
	engine := embed.New("../..")
	defer engine.Close()

	text := strings.Repeat("hello world ", 100)

	// Warm up (load module)
	_, err := engine.Call("internal/dashboard_transforms/event_formatter", "truncate", text, 50)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/event_formatter", "truncate", text, 50)
	}
}

func BenchmarkCountTurns_Go_10(b *testing.B) {
	events := generateEvents(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goCountTurns(events)
	}
}

func BenchmarkCountTurns_AILANG_10(b *testing.B) {
	engine := embed.New("../..")
	defer engine.Close()

	events := generateEvents(10)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/event_formatter", "countTurns", events)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/event_formatter", "countTurns", events)
	}
}

func BenchmarkCountTurns_Go_100(b *testing.B) {
	events := generateEvents(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goCountTurns(events)
	}
}

func BenchmarkCountTurns_AILANG_100(b *testing.B) {
	engine := embed.New("../..")
	defer engine.Close()

	events := generateEvents(100)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/event_formatter", "countTurns", events)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/event_formatter", "countTurns", events)
	}
}

func BenchmarkSummarize_Go_10(b *testing.B) {
	events := generateEvents(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goSummarize(events)
	}
}

func BenchmarkSummarize_AILANG_10(b *testing.B) {
	engine := embed.New("../..")
	defer engine.Close()

	events := generateEvents(10)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/event_formatter", "summarizeEvents", events)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/event_formatter", "summarizeEvents", events)
	}
}

func BenchmarkSummarize_Go_100(b *testing.B) {
	events := generateEvents(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goSummarize(events)
	}
}

func BenchmarkSummarize_AILANG_100(b *testing.B) {
	engine := embed.New("../..")
	defer engine.Close()

	events := generateEvents(100)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/event_formatter", "summarizeEvents", events)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/event_formatter", "summarizeEvents", events)
	}
}

// Functional correctness test
func TestAILANGMatchesGo(t *testing.T) {
	engine := embed.New("../..")
	defer engine.Close()

	// Test truncate
	text := "hello world this is a long string"
	goResult := goTruncate(text, 10)

	ailangResult, err := engine.Call("internal/dashboard_transforms/event_formatter", "truncate", text, 10)
	if err != nil {
		t.Fatalf("AILANG truncate failed: %v", err)
	}
	ailangStr, _ := embed.ToString(ailangResult)

	if goResult != ailangStr {
		t.Errorf("truncate mismatch: Go=%q, AILANG=%q", goResult, ailangStr)
	}

	// Test countTurns
	events := generateEvents(25)
	goTurns := goCountTurns(events)

	ailangTurnsVal, err := engine.Call("internal/dashboard_transforms/event_formatter", "countTurns", events)
	if err != nil {
		t.Fatalf("AILANG countTurns failed: %v", err)
	}
	ailangTurns, _ := embed.ToInt(ailangTurnsVal)

	if goTurns != ailangTurns {
		t.Errorf("countTurns mismatch: Go=%d, AILANG=%d", goTurns, ailangTurns)
	}

	t.Logf("truncate: %q", ailangStr)
	t.Logf("countTurns: %d (expected %d)", ailangTurns, goTurns)
}
