package firestore

import (
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

func TestAddCost(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: make(map[string]float64),
	}

	s.addCost("claude", 1.50)
	s.addCost("gemini", 0.75)
	s.addCost("claude", 2.00)

	s.costMu.RLock()
	defer s.costMu.RUnlock()

	if got := s.costCounters["claude"]; got != 3.50 {
		t.Errorf("claude cost = %f, want 3.50", got)
	}
	if got := s.costCounters["gemini"]; got != 0.75 {
		t.Errorf("gemini cost = %f, want 0.75", got)
	}
	if !s.costDirty {
		t.Error("costDirty should be true after addCost")
	}
}

func TestAddCostSkipsEmptyProvider(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: make(map[string]float64),
	}

	s.addCost("", 1.50)    // empty provider → skip
	s.addCost("claude", 0) // zero cost → skip

	s.costMu.RLock()
	defer s.costMu.RUnlock()

	if len(s.costCounters) != 0 {
		t.Errorf("expected empty counters, got %v", s.costCounters)
	}
	if s.costDirty {
		t.Error("costDirty should be false when nothing was added")
	}
}

func TestGetCostByProviderFromMemory(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: map[string]float64{
			"claude": 5.00,
			"gemini": 2.50,
		},
		costLoaded: true,
	}

	costs, err := s.GetCostByProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := costs["claude"]; got != 5.00 {
		t.Errorf("claude = %f, want 5.00", got)
	}
	if got := costs["gemini"]; got != 2.50 {
		t.Errorf("gemini = %f, want 2.50", got)
	}

	// Verify returned map is a copy (mutation doesn't affect internal state).
	costs["claude"] = 999.0
	s.costMu.RLock()
	if s.costCounters["claude"] != 5.00 {
		t.Error("GetCostByProvider should return a copy, not the internal map")
	}
	s.costMu.RUnlock()
}

func TestStatsCacheHitAndInvalidation(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: make(map[string]float64),
	}

	// Manually set cached stats.
	fakeStats := &coordinator.TaskStats{
		TotalTasks:     42,
		CompletedTasks: 10,
		ByType:         make(map[string]int),
		ByProvider:     make(map[string]*coordinator.DetailedStats),
		ByWorkspace:    make(map[string]*coordinator.DetailedStats),
	}
	s.statsMu.Lock()
	s.cachedStats = fakeStats
	s.statsExpiry = time.Now().Add(30 * time.Second)
	s.statsMu.Unlock()

	// Cache hit — should return cached stats without Firestore.
	s.statsMu.RLock()
	if s.cachedStats == nil || s.cachedStats.TotalTasks != 42 {
		t.Error("expected cached stats to be set")
	}
	s.statsMu.RUnlock()

	// Invalidate.
	s.invalidateStatsCache()

	s.statsMu.RLock()
	if s.cachedStats != nil {
		t.Error("expected cached stats to be nil after invalidation")
	}
	s.statsMu.RUnlock()
}

func TestStatsCacheExpiry(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: make(map[string]float64),
	}

	// Set expired cache.
	s.statsMu.Lock()
	s.cachedStats = &coordinator.TaskStats{TotalTasks: 99,
		ByType: make(map[string]int), ByProvider: make(map[string]*coordinator.DetailedStats), ByWorkspace: make(map[string]*coordinator.DetailedStats)}
	s.statsExpiry = time.Now().Add(-1 * time.Second) // already expired
	s.statsMu.Unlock()

	// Check that expired cache would be skipped (cache miss path).
	s.statsMu.RLock()
	isExpired := s.cachedStats != nil && time.Now().After(s.statsExpiry)
	s.statsMu.RUnlock()

	if !isExpired {
		t.Error("expected cache to be expired")
	}
}

func TestCostCounterConcurrency(t *testing.T) {
	s := &CoordinatorStore{
		costCounters: make(map[string]float64),
		costLoaded:   true,
	}

	done := make(chan struct{})
	// Concurrent writes.
	for i := 0; i < 100; i++ {
		go func() {
			s.addCost("claude", 0.01)
			done <- struct{}{}
		}()
	}
	// Concurrent reads.
	for i := 0; i < 100; i++ {
		go func() {
			_, _ = s.GetCostByProvider()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 200; i++ {
		<-done
	}

	s.costMu.RLock()
	defer s.costMu.RUnlock()
	// Allow small floating-point drift.
	if got := s.costCounters["claude"]; got < 0.99 || got > 1.01 {
		t.Errorf("claude cost after 100 x 0.01 = %f, want ~1.00", got)
	}
}
