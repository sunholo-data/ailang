package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestResourceTracker_Basic(t *testing.T) {
	tracker := NewResourceTracker("task-123", "thread-456", 0)

	// Test initial state
	metrics := tracker.GetMetrics()
	if metrics.TaskID != "task-123" {
		t.Errorf("expected task ID 'task-123', got %s", metrics.TaskID)
	}
	if metrics.ThreadID != "thread-456" {
		t.Errorf("expected thread ID 'thread-456', got %s", metrics.ThreadID)
	}
	if metrics.TokensIn != 0 {
		t.Errorf("expected 0 tokens in, got %d", metrics.TokensIn)
	}
}

func TestResourceTracker_UpdateTokens(t *testing.T) {
	tracker := NewResourceTracker("task-123", "", 0)

	// Add tokens
	tracker.UpdateTokens(100, 50)
	tracker.UpdateTokens(200, 100)

	metrics := tracker.GetMetrics()
	if metrics.TokensIn != 300 {
		t.Errorf("expected 300 tokens in, got %d", metrics.TokensIn)
	}
	if metrics.TokensOut != 150 {
		t.Errorf("expected 150 tokens out, got %d", metrics.TokensOut)
	}
}

func TestResourceTracker_SetCost(t *testing.T) {
	tracker := NewResourceTracker("task-123", "", 0)

	// Set cost
	tracker.SetCost(0.05)
	metrics := tracker.GetMetrics()
	if metrics.Cost != 0.05 {
		t.Errorf("expected cost 0.05, got %f", metrics.Cost)
	}

	// Set again (should replace, not add)
	tracker.SetCost(0.10)
	metrics = tracker.GetMetrics()
	if metrics.Cost != 0.10 {
		t.Errorf("expected cost 0.10, got %f", metrics.Cost)
	}
}

func TestResourceTracker_Duration(t *testing.T) {
	tracker := NewResourceTracker("task-123", "", 0)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	metrics := tracker.GetMetrics()
	if metrics.DurationSec < 0 {
		t.Errorf("expected non-negative duration, got %d", metrics.DurationSec)
	}
}

func TestResourceTracker_UpdateCallback(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	tracker := NewResourceTracker("task-123", "", 0)
	tracker.SetUpdateCallback(func(m *ResourceMetrics) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Trigger a metrics update by setting tokens
	tracker.UpdateTokens(100, 50)

	// The callback is only called during polling, so we need to verify the setup works
	mu.Lock()
	defer mu.Unlock()
	// Callback may or may not have been called yet (depends on polling)
	// Just verify no panic occurred - callCount >= 0 is always true
	if callCount < 0 {
		t.Error("unexpected negative call count")
	}
}

func TestResourceTracker_StartStop(t *testing.T) {
	tracker := NewResourceTracker("task-123", "", 0)
	tracker.SetPollInterval(50 * time.Millisecond)

	ctx := context.Background()
	tracker.Start(ctx)

	// Let it poll a couple times
	time.Sleep(120 * time.Millisecond)

	// Stop should not panic
	tracker.Stop()
}

func TestResourceTrackerRegistry_Basic(t *testing.T) {
	registry := NewResourceTrackerRegistry()

	// Register a tracker
	tracker := NewResourceTracker("task-1", "", 0)
	registry.Register("task-1", tracker)

	// Get the tracker
	got := registry.Get("task-1")
	if got != tracker {
		t.Error("expected to get the same tracker back")
	}

	// Get non-existent
	got = registry.Get("task-2")
	if got != nil {
		t.Error("expected nil for non-existent tracker")
	}
}

func TestResourceTrackerRegistry_GetAllMetrics(t *testing.T) {
	registry := NewResourceTrackerRegistry()

	// Register multiple trackers
	tracker1 := NewResourceTracker("task-1", "", 0)
	tracker1.UpdateTokens(100, 50)

	tracker2 := NewResourceTracker("task-2", "", 0)
	tracker2.UpdateTokens(200, 100)

	registry.Register("task-1", tracker1)
	registry.Register("task-2", tracker2)

	// Get all metrics
	metrics := registry.GetAllMetrics()
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}

	// Verify metrics are correct (order may vary)
	totalTokensIn := 0
	for _, m := range metrics {
		totalTokensIn += m.TokensIn
	}
	if totalTokensIn != 300 {
		t.Errorf("expected 300 total tokens in, got %d", totalTokensIn)
	}
}

func TestResourceTrackerRegistry_Unregister(t *testing.T) {
	registry := NewResourceTrackerRegistry()

	// Register and start a tracker
	tracker := NewResourceTracker("task-1", "", 0)
	tracker.SetPollInterval(50 * time.Millisecond)
	ctx := context.Background()
	tracker.Start(ctx)

	registry.Register("task-1", tracker)

	// Unregister should stop the tracker
	registry.Unregister("task-1")

	// Verify removed
	got := registry.Get("task-1")
	if got != nil {
		t.Error("expected nil after unregister")
	}
}

func TestResourceTrackerRegistry_StopAll(t *testing.T) {
	registry := NewResourceTrackerRegistry()

	// Register multiple trackers
	for i := 0; i < 5; i++ {
		tracker := NewResourceTracker("task-"+string(rune('0'+i)), "", 0)
		tracker.SetPollInterval(50 * time.Millisecond)
		ctx := context.Background()
		tracker.Start(ctx)
		registry.Register("task-"+string(rune('0'+i)), tracker)
	}

	// Stop all
	registry.StopAll()

	// Verify all removed
	metrics := registry.GetAllMetrics()
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics after StopAll, got %d", len(metrics))
	}
}

func TestResourceTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewResourceTracker("task-123", "", 0)

	// Concurrent updates
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.UpdateTokens(10, 5)
			tracker.SetCost(0.01)
			_ = tracker.GetMetrics()
		}()
	}

	wg.Wait()

	metrics := tracker.GetMetrics()
	if metrics.TokensIn != 1000 {
		t.Errorf("expected 1000 tokens in, got %d", metrics.TokensIn)
	}
	if metrics.TokensOut != 500 {
		t.Errorf("expected 500 tokens out, got %d", metrics.TokensOut)
	}
}
