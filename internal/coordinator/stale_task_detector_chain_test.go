package coordinator

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// chainClosingBackend records UpdateChainStatus calls. It embeds
// observatory.Backend so the unimplemented methods panic loudly if the code
// under test ever reaches for one we did not expect.
type chainClosingBackend struct {
	observatory.Backend
	mu     sync.Mutex
	closed map[string]observatory.ChainStatus
	err    error
}

func newChainClosingBackend() *chainClosingBackend {
	return &chainClosingBackend{closed: map[string]observatory.ChainStatus{}}
}

func (b *chainClosingBackend) UpdateChainStatus(_ context.Context, chainID string, status observatory.ChainStatus) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	b.closed[chainID] = status
	return nil
}

func (b *chainClosingBackend) statusFor(chainID string) (observatory.ChainStatus, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s, ok := b.closed[chainID]
	return s, ok
}

// staleDetectorForTest builds a detector over a store holding one task that is
// already far past any plausible timeout.
func staleDetectorForTest(t *testing.T, chainID string) (*StaleTaskDetector, *MockStore, *chainClosingBackend) {
	t.Helper()
	store := NewMockStore()
	if err := store.CreateTask(context.Background(), &TaskRecord{
		ID:        "task-a0628a5f",
		Status:    TaskStatusRunning,
		AgentID:   "sprint-planner",
		ChainID:   chainID,
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	obs := newChainClosingBackend()
	d := NewStaleTaskDetector(store, nil, nil, log.New(io.Discard, "", 0)).WithObservatory(obs)
	return d, store, obs
}

// TestStaleTaskClosesItsChain pins the fix for the 92 chains found `active` in
// prod, the oldest dating to 2026-04-27. MarkTaskFailed moved only the task;
// chain closure otherwise happens solely on the approval path, which a job that
// died before completing never reaches. A chain that stays `active` makes
// "running now" and "died in April" the same reading.
func TestStaleTaskClosesItsChain(t *testing.T) {
	d, store, obs := staleDetectorForTest(t, "chain-123")

	d.detectAndMarkStale(context.Background())

	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Fatalf("MarkTaskFailed called %d times, want 1", got)
	}
	status, ok := obs.statusFor("chain-123")
	if !ok {
		t.Fatal("timed-out task did not close its chain — this is the stuck-active defect")
	}
	if status != observatory.ChainStatusFailed {
		t.Errorf("chain closed as %q, want %q", status, observatory.ChainStatusFailed)
	}
}

// TestStaleTaskWithoutChainIDIsSafe: not every task carries a chain, and a
// missing chain id must not produce a bogus closure or a panic.
func TestStaleTaskWithoutChainIDIsSafe(t *testing.T) {
	d, store, obs := staleDetectorForTest(t, "")

	d.detectAndMarkStale(context.Background())

	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Fatalf("MarkTaskFailed called %d times, want 1", got)
	}
	obs.mu.Lock()
	n := len(obs.closed)
	obs.mu.Unlock()
	if n != 0 {
		t.Errorf("closed %d chains for a task with no chain id, want 0", n)
	}
}

// TestChainCloseFailureDoesNotBlockTaskFailure: a chain left open is a reporting
// defect, not a lost outcome. The task must still be marked failed.
func TestChainCloseFailureDoesNotBlockTaskFailure(t *testing.T) {
	d, store, obs := staleDetectorForTest(t, "chain-err")
	obs.err = context.DeadlineExceeded

	d.detectAndMarkStale(context.Background())

	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Errorf("MarkTaskFailed called %d times, want 1 — chain trouble must not swallow the task outcome", got)
	}
}

// TestNilObservatoryIsTolerated preserves the pre-existing behaviour for any
// caller that has no observatory attached.
func TestNilObservatoryIsTolerated(t *testing.T) {
	store := NewMockStore()
	if err := store.CreateTask(context.Background(), &TaskRecord{
		ID: "task-nil-obs", Status: TaskStatusRunning, ChainID: "chain-x",
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	d := NewStaleTaskDetector(store, nil, nil, log.New(io.Discard, "", 0))
	d.detectAndMarkStale(context.Background()) // must not panic
	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Errorf("MarkTaskFailed called %d times, want 1", got)
	}
}
