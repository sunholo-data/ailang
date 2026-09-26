# Sprint Plan — M-COORDINATOR-TEST-PARALLELISM (executor-ready)

- **Item id**: `m-coordinator-test-parallelism`
- **Design doc**: `design_docs/planned/v0_35_2/m-coordinator-test-parallelism.md` (revision 3, final; do not redesign — apply it literally)
- **Base commit**: `f3783c9765d1fcdca51533c401bea6a7e53d19b8` (origin/dev), isolated worktree
- **Milestones**: **M1** production injection points (defaults preserved) → **M2** four timer-bound tests updated + mandatory helpers → **M3** three behavioral regression guards (mandatory with M1/M2)
- **Total estimate**: 3–3.5 h (M1 ≈ 45 min · M2 ≈ 75 min · M3 ≈ 45 min · mutation + local measurement ≈ 45 min, both cuttable — see Cut set)
- **Rules of engagement**: work only inside this worktree; run no git write command; run the Gate List (§4) after each milestone; fix reds before proceeding; all line numbers below refer to the pristine base and will drift as edits land — the quoted code blocks, not the line numbers, are the anchor.

---

## M1 — Production injection points (defaults preserved)

**Purpose**: make the three hardcoded timers injectable as per-call config; preserve every production default exactly; keep the whole package green (including the four slow tests, which still run at production timings).

### Files touched

| File | Region (base `f3783c976`) |
|---|---|
| `internal/coordinator/provider.go` | `type ExecuteOptions struct` :54–82 (last field `Plugins *PluginsConfig` :81); `DefaultExecuteOptions()` :98–104 |
| `internal/coordinator/task_executor.go` | `func (te *TaskExecutor) ExecuteWithRetry` :152–188 (esp. :158, :161–169) |
| `internal/coordinator/daemon_tasks_exec_run.go` | opts literal :238–244; call context :333 (`ExecuteWithRetry(taskCtx, analyzed, opts, 2)`) — call itself unchanged |
| `internal/coordinator/approval_checkpoint.go` | `type StoreBackedApprovalCheckpoint struct` :327–331; `NewStoreBackedApprovalCheckpoint` :333–340; `RequestApproval` poll setup :386–394 |
| `internal/coordinator/event_handler.go` | `CoordinatorEventHandler` rate-limiting field block :40–45; option type + `NewCoordinatorEventHandler` :60–72; `checkRateLimit` :274–294 (`now := time.Now()` :279) |
| `internal/coordinator/integration_test.go` | `TestIntegration_TaskExecutorWithRetry` opts literal :216–220 — M1 only adds the two explicit production defaults (nil-`Wait` hazard; see E10 note) |
| `internal/coordinator/approval_checkpoint_edge_test.go` | constructor call sites :21 and :76 — forced by the signature change; M1 passes the production defaults explicitly |

### Edit steps

**E1 — `provider.go`: add two fields at the end of `ExecuteOptions`.**
Current (:79–82):
```go
	// Plugins for per-agent third-party plugin installation (M-CLOUD-PLUGIN-SKILLS, v0.9.1).
	Plugins *PluginsConfig
}
```
Replacement:
```go
	// Plugins for per-agent third-party plugin installation (M-CLOUD-PLUGIN-SKILLS, v0.9.1).
	Plugins *PluginsConfig

	// RetryBaseDelay is the base delay for ExecuteWithRetry's exponential backoff
	// (M-COORDINATOR-TEST-PARALLELISM). Set explicitly by DefaultExecuteOptions;
	// there is deliberately no zero-value fallback (FIX 2).
	RetryBaseDelay time.Duration

	// Wait is the backoff wait seam used by ExecuteWithRetry
	// (M-COORDINATOR-TEST-PARALLELISM). Set explicitly by DefaultExecuteOptions.
	Wait func(time.Duration)
}
```

**E2 — `provider.go`: extend `DefaultExecuteOptions()` (:99–104).**
Current:
```go
func DefaultExecuteOptions() *ExecuteOptions {
	return &ExecuteOptions{
		Timeout: 5 * time.Minute,
		DryRun:  false,
	}
}
```
Replacement:
```go
func DefaultExecuteOptions() *ExecuteOptions {
	return &ExecuteOptions{
		Timeout:        5 * time.Minute,
		DryRun:         false,
		RetryBaseDelay: time.Second,
		Wait:           time.Sleep,
	}
}
```

**E3 — `task_executor.go`: consume the injected values in `ExecuteWithRetry` (:157–169). No `0 → time.Second` fallback (FIX 2).**
Current:
```go
	var lastResult *ExecuteResult
	baseDelay := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
```
Replacement:
```go
	var lastResult *ExecuteResult
	baseDelay := opts.RetryBaseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff (injected wait seam: M-COORDINATOR-TEST-PARALLELISM)
			delay := baseDelay * time.Duration(1<<(attempt-1))
			opts.Wait(delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
```
Note: `opts.Wait(delay)` replaces `<-time.After(delay)` exactly as the design prescribes; ctx cancellation during a backoff is now observed when `Wait` returns. With the preserved defaults (1 s/2 s real sleeps, production caller rebased in E4) production timing semantics are unchanged. The `time` import stays live via `time.Duration` on the `delay :=` line (the only other `time.` uses in this file are the two being replaced — verified at base).

**E4 — `daemon_tasks_exec_run.go`: rebase the production caller on `DefaultExecuteOptions()` (:238–244, FIX 2).**
Current:
```go
	opts := &ExecuteOptions{
		Timeout:            agentConfig.GetEffectiveTimeout(),     // Hard ceiling (v0.8.1), default 60m
		IdleTimeout:        agentConfig.GetEffectiveIdleTimeout(), // Idle kill (v0.8.1), default 3m
		Workspace:          workspacePath,                         // Worktree path for AI agents, direct workspace for script agents
		ObservatoryContext: obsContext,
		AgentConfig:        agentConfig, // For system prompt construction (v0.8.0+)
	}
```
Replacement:
```go
	// Base on DefaultExecuteOptions so RetryBaseDelay/Wait are set explicitly
	// (M-COORDINATOR-TEST-PARALLELISM, FIX 2). Each override below wins over the
	// default, so the five fields are identical to today's literal.
	opts := DefaultExecuteOptions()
	opts.Timeout = agentConfig.GetEffectiveTimeout()         // Hard ceiling (v0.8.1), default 60m
	opts.IdleTimeout = agentConfig.GetEffectiveIdleTimeout() // Idle kill (v0.8.1), default 3m
	opts.Workspace = workspacePath                           // Worktree path for AI agents, direct workspace for script agents
	opts.ObservatoryContext = obsContext
	opts.AgentConfig = agentConfig // For system prompt construction (v0.8.0+)
```
All other fields (`Model`, `EventHandler`, `DryRun`, `InvokeConfig`, `Effort`, `PluginDirs`, `Plugins`) remain at the zero values they had in the literal; `DryRun` is `false` in both. The `ExecuteWithRetry(taskCtx, analyzed, opts, 2)` call at :333 is untouched.

**E5 — `approval_checkpoint.go`: struct field + 4-arg constructor + default ticker helper (:327–340).**
Current:
```go
type StoreBackedApprovalCheckpoint struct {
	*ApprovalCheckpoint
	store        ApprovalStore
	pollInterval time.Duration
}

// NewStoreBackedApprovalCheckpoint creates a store-backed approval checkpoint
func NewStoreBackedApprovalCheckpoint(store ApprovalStore, defaultTimeout time.Duration) *StoreBackedApprovalCheckpoint {
	return &StoreBackedApprovalCheckpoint{
		ApprovalCheckpoint: NewApprovalCheckpoint(defaultTimeout),
		store:              store,
		pollInterval:       2 * time.Second,
	}
}
```
Replacement:
```go
type StoreBackedApprovalCheckpoint struct {
	*ApprovalCheckpoint
	store        ApprovalStore
	pollInterval time.Duration
	tick         func(time.Duration) <-chan time.Time
}

// NewStoreBackedApprovalCheckpoint creates a store-backed approval checkpoint.
// pollInterval and tick are per-call injection points (M-COORDINATOR-TEST-PARALLELISM,
// FIX 1). Callers pass the production defaults 2*time.Second and defaultPollTick to
// preserve pre-injection behavior exactly. There are no production callers.
func NewStoreBackedApprovalCheckpoint(store ApprovalStore, defaultTimeout, pollInterval time.Duration, tick func(time.Duration) <-chan time.Time) *StoreBackedApprovalCheckpoint {
	return &StoreBackedApprovalCheckpoint{
		ApprovalCheckpoint: NewApprovalCheckpoint(defaultTimeout),
		store:              store,
		pollInterval:       pollInterval,
		tick:               tick,
	}
}

// defaultPollTick is the production-default tick source: a real ticker channel.
// Unreferenced tickers are garbage-collected since Go 1.23; this module targets
// Go 1.26, so dropping pollTicker.Stop() leaks nothing.
func defaultPollTick(interval time.Duration) <-chan time.Time {
	return time.NewTicker(interval).C
}
```

**E6 — `approval_checkpoint.go`: use the tick seam in `RequestApproval` (:386–394).**
Current:
```go
	// Start polling for store changes
	pollTicker := time.NewTicker(sac.pollInterval)
	defer pollTicker.Stop()
```
Replacement:
```go
	// Start polling for store changes (injected tick seam: M-COORDINATOR-TEST-PARALLELISM)
	tickCh := sac.tick(sac.pollInterval)
```
And current:
```go
		case <-pollTicker.C:
			// Poll store for status changes
```
Replacement:
```go
		case <-tickCh:
			// Poll store for status changes
```

**E7 — `event_handler.go`: add the `now` field to the rate-limiting block (:40–45).**
Current:
```go
	// Rate limiting
	mu              sync.Mutex
	lastEventTime   time.Time
	eventCount      int
	maxEventsPerSec int
	throttled       bool
```
Replacement:
```go
	// Rate limiting
	mu              sync.Mutex
	lastEventTime   time.Time
	eventCount      int
	maxEventsPerSec int
	throttled       bool
	now             func() time.Time // Injected clock (M-COORDINATOR-TEST-PARALLELISM); default time.Now
```

**E8 — `event_handler.go`: option type + variadic constructor (:60–72).**
Current:
```go
// NewCoordinatorEventHandler creates a new event handler for a task.
func NewCoordinatorEventHandler(taskID, threadID string, broadcast EventBroadcaster) *CoordinatorEventHandler {
	return &CoordinatorEventHandler{
		taskID:          taskID,
		threadID:        threadID,
		broadcast:       broadcast,
		maxEventsPerSec: 10,  // Rate limit: max 10 events per second
		maxBufferSize:   100, // Keep last 100 events for replay
		eventBuffer:     make([]*websocket.TaskStreamEvent, 0, 100),
		startTime:       time.Now(),
	}
}
```
Replacement:
```go
// CoordinatorEventHandlerOption is a functional option for NewCoordinatorEventHandler.
type CoordinatorEventHandlerOption func(*CoordinatorEventHandler)

// WithClock injects the clock used by the rate-limit window check
// (M-COORDINATOR-TEST-PARALLELISM). Default: time.Now.
func WithClock(now func() time.Time) CoordinatorEventHandlerOption {
	return func(h *CoordinatorEventHandler) {
		h.now = now
	}
}

// NewCoordinatorEventHandler creates a new event handler for a task.
func NewCoordinatorEventHandler(taskID, threadID string, broadcast EventBroadcaster, opts ...CoordinatorEventHandlerOption) *CoordinatorEventHandler {
	h := &CoordinatorEventHandler{
		taskID:          taskID,
		threadID:        threadID,
		broadcast:       broadcast,
		maxEventsPerSec: 10,  // Rate limit: max 10 events per second
		maxBufferSize:   100, // Keep last 100 events for replay
		eventBuffer:     make([]*websocket.TaskStreamEvent, 0, 100),
		startTime:       time.Now(),
		now:             time.Now,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
```
The variadic parameter keeps all 8 existing call sites (1 production at `daemon_tasks_exec_run.go:292`, 7 test) source-compatible — none is edited.

**E9 — `event_handler.go`: use the injected clock in `checkRateLimit` (:279).**
Current:
```go
	now := time.Now()
```
Replacement:
```go
	now := h.now()
```
(Uniqueness verified: this is the only `now := time.Now()` in the file at base.)

**E10 — `integration_test.go`: set the two fields to the production defaults in the retry test's literal (:216–220).**
Current:
```go
	opts := &ExecuteOptions{
		Timeout:   30 * time.Second,
		Workspace: t.TempDir(),
	}
```
Replacement:
```go
	opts := &ExecuteOptions{
		Timeout:        30 * time.Second,
		Workspace:      t.TempDir(),
		RetryBaseDelay: time.Second, // production default, set explicitly (M1; shortened in M2)
		Wait:           time.Sleep,  // production default, set explicitly (M1; seamed in M2)
	}
```
This step is load-bearing: after E3, `ExecuteWithRetry` calls `opts.Wait(delay)` unconditionally — a literal without `Wait` is a nil-func panic. Every `ExecuteWithRetry` caller must therefore have both fields set explicitly (FIX 2, no silent coercion): the production caller via E4, this test via E10. These are the only two callers (census in the Plan Verification Log).

**E11 — `approval_checkpoint_edge_test.go`: update the two forced call sites with production defaults.**
Site 1 (:21, followed by `// Request approval in background`). Current:
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour)

	// Request approval in background
```
Replacement:
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 2*time.Second, defaultPollTick)

	// Request approval in background
```
Site 2 (:76, followed by `var wg sync.WaitGroup`). Current:
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour)

	var wg sync.WaitGroup
```
Replacement:
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 2*time.Second, defaultPollTick)

	var wg sync.WaitGroup
```

**E12 — run `gofmt -w internal/coordinator`, then the full Gate List (§4). M1 ends green.**

### Acceptance criteria

| command | expected observation |
|---|---|
| `go build ./internal/coordinator/...` | exit 0 (base: exit 0) |
| `grep -n 'RetryBaseDelay' internal/coordinator/provider.go internal/coordinator/task_executor.go` | matches: field decl + `RetryBaseDelay: time.Second` in provider.go; `baseDelay := opts.RetryBaseDelay` in task_executor.go (base: no output, exit 1; control `grep -c 'Timeout' internal/coordinator/provider.go` > 0) |
| `grep -n 'baseDelay == 0' internal/coordinator/task_executor.go` | no output, exit 1 — no zero-value fallback (base: also no output; control `grep -c 'baseDelay' internal/coordinator/task_executor.go` ≥ 1 at base, ≥ 2 after) |
| `grep -n 'time.After' internal/coordinator/task_executor.go` | no output, exit 1 (base: one match at :165) |
| `grep -n 'tick func\|sac.tick\|defaultPollTick' internal/coordinator/approval_checkpoint.go` | matches: field type, ctor param, `sac.tick(sac.pollInterval)`, helper (base: no output, exit 1; control `grep -c 'pollInterval' internal/coordinator/approval_checkpoint.go` = 3 at base) |
| `grep -n 'now func() time.Time\|WithClock\|h.now()' internal/coordinator/event_handler.go` | matches: field, option func, `now: time.Now` is allowed — specifically `now := h.now()` present (base: no output, exit 1; control `grep -c 'func (h \*CoordinatorEventHandler)' internal/coordinator/event_handler.go` > 0) |
| `grep -n 'pollInterval:' internal/coordinator/approval_checkpoint.go` | no output — the hardcoded `pollInterval: 2 * time.Second` literal is gone (base: one match at :338) |
| `grep -n 'opts := DefaultExecuteOptions()' internal/coordinator/daemon_tasks_exec_run.go` | exactly one match (base: no output, exit 1; control `grep -c 'GetEffectiveTimeout' internal/coordinator/daemon_tasks_exec_run.go` = 2 both sides — both assignments survive the rebase) |
| `grep -rn 'ExecuteWithRetry(taskCtx' internal/coordinator/daemon_tasks_exec_run.go` | unchanged single call with `opts, 2` |
| `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1` | exit 0; all four PASS. **No duration assertion** — they legitimately still sleep at production timings pre-M2 (base: exit 0) |
| Gate List (§4), all four commands | all green |

### Anti-vacuity note (M1)

M1 adds no new tests; its tripwires are the grep rows above: the `pollInterval:` no-output row and the `opts := DefaultExecuteOptions()` row die if E5/E4 are reverted, and the M1 four-test run dies loudly (nil-`Wait` panic) if E3 lands without E10/E4 — so the milestone cannot pass with a half-applied injection.

---

## M2 — Update the four timer-bound tests (inject short intervals + seams)

**Purpose**: remove the 8.14 s of real sleep/poll time. Adds three shared test helpers (`waitRecorder`, `tickRecorder`, `manualClock`) that M3's guards also use — that is deliberate sequencing so the package compiles and gates green after M2 alone.

### Files touched

| File | Region (base `f3783c976`) |
|---|---|
| `internal/coordinator/integration_test.go` | import block :3–7; helper added before `TestIntegration_TaskExecutorWithRetry` :186; literal (post-M1) :216–222; tail of test :223–230 |
| `internal/coordinator/approval_checkpoint_edge_test.go` | helper added before `TestStoreBackedApprovalCheckpoint` :11; full rewrite of test 1 :11–65 and test 2 :67–106 (bodies below quote the **post-M1** text) |
| `internal/coordinator/event_handler_test.go` | helper added before `TestCoordinatorEventHandler_RateLimitReset` :123; full rewrite of that test :123–153 |

### Edit steps

**E1 — `integration_test.go`: add `sync` to the import block.**
Current:
```go
import (
	"context"
	"testing"
	"time"
)
```
Replacement:
```go
import (
	"context"
	"sync"
	"testing"
	"time"
)
```

**E2 — `integration_test.go`: add the wait seam helper directly above `TestIntegration_TaskExecutorWithRetry`.**
Current:
```go
// TestIntegration_TaskExecutorWithRetry tests retry behavior
func TestIntegration_TaskExecutorWithRetry(t *testing.T) {
```
Replacement:
```go
// waitRecorder is a test-controlled Wait seam for ExecuteWithRetry
// (M-COORDINATOR-TEST-PARALLELISM). It records each requested backoff delay
// and returns immediately, so tests never sleep on the backoff timer.
type waitRecorder struct {
	mu     sync.Mutex
	delays []time.Duration
}

func (w *waitRecorder) wait(d time.Duration) {
	w.mu.Lock()
	w.delays = append(w.delays, d)
	w.mu.Unlock()
}

func (w *waitRecorder) recorded() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]time.Duration(nil), w.delays...)
}

// TestIntegration_TaskExecutorWithRetry tests retry behavior
func TestIntegration_TaskExecutorWithRetry(t *testing.T) {
```

**E3 — `integration_test.go`: shorten the backoff and seam the wait in the literal (post-M1 text).**
Current (as produced by M1-E10):
```go
	opts := &ExecuteOptions{
		Timeout:        30 * time.Second,
		Workspace:      t.TempDir(),
		RetryBaseDelay: time.Second, // production default, set explicitly (M1; shortened in M2)
		Wait:           time.Sleep,  // production default, set explicitly (M1; seamed in M2)
	}
```
Replacement:
```go
	waitRec := &waitRecorder{}
	opts := &ExecuteOptions{
		Timeout:        30 * time.Second,
		Workspace:      t.TempDir(),
		RetryBaseDelay: 10 * time.Millisecond, // M2: 1s+2s backoff becomes 10ms+20ms
		Wait:           waitRec.wait,          // M2: recorded, returns immediately — no real sleep
	}
```

**E4 — `integration_test.go`: assert the recorded backoff at the end of the test (after the existing `attemptCount` check).**
Current:
```go
	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}
```
Replacement:
```go
	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	delays := waitRec.recorded()
	if len(delays) != 2 || delays[0] != 10*time.Millisecond || delays[1] != 20*time.Millisecond {
		t.Errorf("expected recorded backoff waits [10ms 20ms], got %v", delays)
	}
}
```
(The `oldText` `if attemptCount != 3` block appears twice in the file — anchor on the one inside `TestIntegration_TaskExecutorWithRetry` by including the preceding `if !result.Success` block in the search region if your editor requires uniqueness; the M3 guard's equivalent text lives in a different file so there is no cross-file ambiguity.)

**E5 — `approval_checkpoint_edge_test.go`: add the tick seam helper directly above the first test.**
Current:
```go
// TestStoreBackedApprovalCheckpoint tests the store-backed version
func TestStoreBackedApprovalCheckpoint(t *testing.T) {
```
Replacement:
```go
// tickRecorder is a test-controlled tick seam for StoreBackedApprovalCheckpoint
// (M-COORDINATOR-TEST-PARALLELISM, FIX 1). It records the requested poll interval
// and returns ch, which the test drives explicitly. ch is unbuffered: one release
// is consumed by exactly one poll iteration.
type tickRecorder struct {
	mu       sync.Mutex
	interval time.Duration
	ch       chan time.Time
}

func newTickRecorder() *tickRecorder {
	return &tickRecorder{ch: make(chan time.Time)}
}

func (tr *tickRecorder) tick(d time.Duration) <-chan time.Time {
	tr.mu.Lock()
	tr.interval = d
	tr.mu.Unlock()
	return tr.ch
}

func (tr *tickRecorder) recordedInterval() time.Duration {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.interval
}

// TestStoreBackedApprovalCheckpoint tests the store-backed version
func TestStoreBackedApprovalCheckpoint(t *testing.T) {
```

**E6 — `approval_checkpoint_edge_test.go`: rewrite `TestStoreBackedApprovalCheckpoint`'s body after the `defer store.Close()` line.**
Current (post-M1):
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 2*time.Second, defaultPollTick)

	// Request approval in background
	var wg sync.WaitGroup
	var status ApprovalStatus
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		status, err = sac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:          "test-store-1",
			TaskID:      "task-store-1",
			Type:        ApprovalTypeMerge,
			Description: "Test approval",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	// Wait for request to be stored
	time.Sleep(100 * time.Millisecond)

	// Verify it's in the store
	pending, err := store.ListPendingApprovals(context.Background())
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	// Approve via store (simulates CLI)
	err = store.ResolveApprovalRequest(context.Background(), "test-store-1", "approved", "test-user")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	// Wait for polling to detect the change
	wg.Wait()

	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
}
```
Replacement:
```go
	tr := newTickRecorder()
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 10*time.Millisecond, tr.tick)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Request approval in background
	var status ApprovalStatus
	done := make(chan struct{})
	go func() {
		defer close(done)
		var err error
		status, err = sac.RequestApproval(ctx, &ApprovalRequest{
			ID:          "test-store-1",
			TaskID:      "task-store-1",
			Type:        ApprovalTypeMerge,
			Description: "Test approval",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	// Wait for request to be stored
	time.Sleep(100 * time.Millisecond)

	// Verify it's in the store
	pending, err := store.ListPendingApprovals(context.Background())
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	// Approve via store (simulates CLI)
	err = store.ResolveApprovalRequest(context.Background(), "test-store-1", "approved", "test-user")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	// Explicitly release ticks until the poll observes the resolution (FIX 1).
	// The deadline is a LIVENESS safeguard, not a timing-correctness assertion.
	liveness := time.After(5 * time.Second)
releaseLoop:
	for {
		select {
		case <-done:
			break releaseLoop
		case <-liveness:
			t.Fatal("poll did not detect the store resolution (liveness safeguard)")
		case tr.ch <- time.Now():
		}
	}

	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
}
```
(`done` closing establishes happens-before for the `status` read; once `RequestApproval` returns, `tr.ch <- time.Now()` is never ready again, so only the `<-done` case can fire — no deadlock, no busy spin.)

**E7 — `approval_checkpoint_edge_test.go`: rewrite `TestStoreBackedApprovalCheckpoint_Rejection`'s body after the `defer store.Close()` line.**
Current (post-M1):
```go
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 2*time.Second, defaultPollTick)

	var wg sync.WaitGroup
	var status ApprovalStatus
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		status, err = sac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:          "test-reject-1",
			TaskID:      "task-reject-1",
			Type:        ApprovalTypeDestroy,
			Description: "Destroy worktree",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Reject via store
	err = store.ResolveApprovalRequest(context.Background(), "test-reject-1", "rejected", "test-user")
	if err != nil {
		t.Fatalf("failed to reject: %v", err)
	}

	wg.Wait()

	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected, got %s", status)
	}
}
```
Replacement:
```go
	tr := newTickRecorder()
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 10*time.Millisecond, tr.tick)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var status ApprovalStatus
	done := make(chan struct{})
	go func() {
		defer close(done)
		var err error
		status, err = sac.RequestApproval(ctx, &ApprovalRequest{
			ID:          "test-reject-1",
			TaskID:      "task-reject-1",
			Type:        ApprovalTypeDestroy,
			Description: "Destroy worktree",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Reject via store
	err = store.ResolveApprovalRequest(context.Background(), "test-reject-1", "rejected", "test-user")
	if err != nil {
		t.Fatalf("failed to reject: %v", err)
	}

	// Explicitly release ticks until the poll observes the rejection (FIX 1).
	// The deadline is a LIVENESS safeguard, not a timing-correctness assertion.
	liveness := time.After(5 * time.Second)
releaseLoop:
	for {
		select {
		case <-done:
			break releaseLoop
		case <-liveness:
			t.Fatal("poll did not detect the store rejection (liveness safeguard)")
		case tr.ch <- time.Now():
		}
	}

	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected, got %s", status)
	}
}
```
The remaining tests in this file keep using `sync` and `fmt` (verified at base: `sync.WaitGroup` at :115/:154/:199/:249, `fmt.Sprintf` at :203/:208/:260) — no import changes needed.

**E8 — `event_handler_test.go`: add the clock helper above `TestCoordinatorEventHandler_RateLimitReset`.**
Current:
```go
func TestCoordinatorEventHandler_RateLimitReset(t *testing.T) {
```
(anchor additionally on the preceding closing of `TestCoordinatorEventHandler_RateLimiting` — the line `func TestCoordinatorEventHandler_RateLimitReset(t *testing.T) {` is unique at base.) Replacement:
```go
// manualClock is a test-controlled clock for the rate-limit window
// (M-COORDINATOR-TEST-PARALLELISM). Advancing it never touches wall time.
type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestCoordinatorEventHandler_RateLimitReset(t *testing.T) {
```

**E9 — `event_handler_test.go`: rewrite the test body (no `time.Sleep`; deterministic clock).**
Current:
```go
	handler := NewCoordinatorEventHandler("task-123", "", broadcaster)

	// Send burst to trigger throttling
	for i := 0; i < 15; i++ {
		handler.OnText("message")
	}

	// Wait for rate limit to reset
	time.Sleep(1100 * time.Millisecond)

	// Should be able to send more events now
	mu.Lock()
	countBefore := len(events)
	mu.Unlock()

	handler.OnText("after reset")

	mu.Lock()
	countAfter := len(events)
	mu.Unlock()

	if countAfter <= countBefore {
		t.Error("expected event to be sent after rate limit reset")
	}
}
```
Replacement:
```go
	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	handler := NewCoordinatorEventHandler("task-123", "", broadcaster, WithClock(clock.Now))

	// Send 15 events at a frozen time: exactly maxEventsPerSec (10) are broadcast,
	// the rest are throttled. This "engaged" assertion keeps the test non-vacuous.
	for i := 0; i < 15; i++ {
		handler.OnText("message")
	}

	mu.Lock()
	count := len(events)
	mu.Unlock()
	if count != 10 {
		t.Errorf("expected exactly 10 events broadcast before reset, got %d", count)
	}
	if !handler.IsThrottled() {
		t.Error("expected handler to be throttled after burst")
	}

	// Advance the injected clock past the 1s window — no wall-clock sleep.
	clock.Advance(1100 * time.Millisecond)

	handler.OnText("after reset")

	mu.Lock()
	countAfter := len(events)
	mu.Unlock()

	if countAfter != 11 {
		t.Errorf("expected 11 events after clock-driven reset, got %d", countAfter)
	}
}
```
(The "exactly 10"/"exactly 11" numbers are exact by construction: with a frozen clock, calls 1–10 pass `checkRateLimit` and broadcast, calls 11–15 set `throttled` and do not broadcast; after `Advance(1100ms)` the next call resets the window and broadcasts. This matches `checkRateLimit` + `OnText` as read at base.)

**E10 — run `gofmt -w internal/coordinator`, then the full Gate List (§4). M2 ends green.**

### Acceptance criteria

| command | expected observation |
|---|---|
| `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1` | exit 0; all four PASS (base: exit 0). No duration assertion — the win is recorded once under Local Measurement (cut set), never asserted |
| `go test ./internal/coordinator/ -count=1` | exit 0 (base: 13.12 s, exit 0) |
| `grep -n 'time.Sleep(1100' internal/coordinator/event_handler_test.go` | no output, exit 1 (base: one match at :141) |
| `grep -n 'WithClock(clock.Now)\|manualClock' internal/coordinator/event_handler_test.go` | matches (base: no output, exit 1; control `grep -c 'NewCoordinatorEventHandler' internal/coordinator/event_handler_test.go` = 7) |
| `grep -n 'waitRec\|waitRecorder' internal/coordinator/integration_test.go` | matches (base: no output; control `grep -c 'IntegrationMockProvider' internal/coordinator/integration_test.go` > 0) |
| `grep -n 'tickRecorder\|10\*time.Millisecond, tr.tick' internal/coordinator/approval_checkpoint_edge_test.go` | helper + two 4-arg call sites present (base: no output; control `grep -c 'NewStoreBackedApprovalCheckpoint' internal/coordinator/approval_checkpoint_edge_test.go` = 2) |
| `grep -n '2\*time.Second, defaultPollTick' internal/coordinator/approval_checkpoint_edge_test.go` | no output, exit 1 — the M1 stopgap defaults are gone from the test file (base/M1: two matches) |
| `go test ./internal/coordinator/ -run 'TestRetryBaseDelayInjected\|TestStoreBackedApprovalCheckpoint_PollIntervalInjected\|TestCoordinatorEventHandler_RateLimitWindowInjected' -count=1` | exit 0 with `testing: warning: no tests to run` — guards land in M3; M2 and M3 are accepted together (FIX 1) |
| Gate List (§4), all four commands | all green |

### Anti-vacuity note (M2)

The rewritten `TestCoordinatorEventHandler_RateLimitReset` cannot pass vacuously: if throttling never engages, the "exactly 10" and `IsThrottled()` assertions fail, and if the injected clock is ignored (real `time.Now()` in `checkRateLimit`), the post-advance event is still throttled so the "11" assertion fails — the same reversion the M3 guard kills in the mutation table.

---

## M3 — Behavioral regression guards (MANDATORY with M1/M2)

**Purpose**: three white-box tests (package `coordinator`, unexported access) that observe each injected value's **effect on production behavior** through the test-controlled seam — never the field, never elapsed wall time (FIX 1).

### Files touched

| File | Region |
|---|---|
| `internal/coordinator/timer_injection_guard_test.go` | **new file**, entire content below |

### Edit steps

**E1 — create `internal/coordinator/timer_injection_guard_test.go` with exactly this content:**
```go
package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/websocket"
)

// Behavioral regression guards for M-COORDINATOR-TEST-PARALLELISM (M3, FIX 1).
// Each guard asserts that production BEHAVIOR changes when the injected value
// changes, observed through a test-controlled seam (wait recorder, tick
// recorder, or manual clock). None asserts elapsed wall time; every deadline
// below is a liveness safeguard on a failure path, not a timing assertion.

// TestRetryBaseDelayInjected proves ExecuteWithRetry honors the injected
// RetryBaseDelay: a reversion to hardcoded backoff records [1s 2s], not [5ms 10ms].
func TestRetryBaseDelayInjected(t *testing.T) {
	attemptCount := 0
	mockProvider := NewIntegrationMockProvider("guard-retry-mock")
	mockProvider.SetExecuteFunc(func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
		attemptCount++
		if attemptCount < 3 {
			return &ExecuteResult{
				Success: false,
				Error:   "rate limit exceeded - 429",
			}, nil
		}
		return &ExecuteResult{
			Success:  true,
			Output:   "Success after retries",
			Provider: "guard-retry-mock",
		}, nil
	})

	executor := NewTaskExecutor(mockProvider)

	task := &AnalyzedTask{
		Task: &Task{
			ID:      "guard-retry-task",
			Title:   "Guard Retry Task",
			Content: "Guard retry injection",
		},
		Type: TaskTypeBugFix,
	}

	waitRec := &waitRecorder{}
	opts := DefaultExecuteOptions()
	opts.Workspace = t.TempDir()
	opts.RetryBaseDelay = 5 * time.Millisecond
	opts.Wait = waitRec.wait

	result, err := executor.ExecuteWithRetry(context.Background(), task, opts, 3)
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success after retries, got failure: %s", result.Error)
	}
	if attemptCount != 3 {
		t.Fatalf("expected 3 attempts, got %d", attemptCount)
	}

	delays := waitRec.recorded()
	if len(delays) != 2 {
		t.Fatalf("expected 2 recorded backoff waits, got %d (%v) — Wait seam not consumed", len(delays), delays)
	}
	if delays[0] != 5*time.Millisecond || delays[1] != 10*time.Millisecond {
		t.Errorf("expected recorded backoff waits [5ms 10ms], got %v — injected RetryBaseDelay not honored", delays)
	}
}

// TestStoreBackedApprovalCheckpoint_PollIntervalInjected proves RequestApproval
// polls at the injected interval via the injected tick: a reversion to the
// hardcoded 2s poll records 2s (or never records), not 5ms.
func TestStoreBackedApprovalCheckpoint_PollIntervalInjected(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	tr := newTickRecorder()
	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour, 5*time.Millisecond, tr.tick)

	// Deadline is a LIVENESS safeguard only: it bounds a hung test.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var status ApprovalStatus
	done := make(chan struct{})
	go func() {
		defer close(done)
		var err error
		status, err = sac.RequestApproval(ctx, &ApprovalRequest{
			ID:          "test-guard-poll",
			TaskID:      "task-guard-poll",
			Type:        ApprovalTypeMerge,
			Description: "Guard poll injection",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	// Wait until the request is persisted. Bounded liveness loop, not a timing assertion.
	persistDeadline := time.Now().Add(2 * time.Second)
	for {
		pending, lerr := store.ListPendingApprovals(context.Background())
		if lerr == nil && len(pending) == 1 {
			break
		}
		if time.Now().After(persistDeadline) {
			t.Fatal("approval request was not persisted (liveness safeguard)")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Resolve via the store (simulates CLI).
	if err := store.ResolveApprovalRequest(context.Background(), "test-guard-poll", "approved", "test-user"); err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	// Explicitly release ticks until the poll observes the resolution (FIX 1).
	liveness := time.After(5 * time.Second)
releaseLoop:
	for {
		select {
		case <-done:
			break releaseLoop
		case <-liveness:
			t.Fatal("poll did not detect the store resolution (liveness safeguard)")
		case tr.ch <- time.Now():
		}
	}

	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
	if got := tr.recordedInterval(); got != 5*time.Millisecond {
		t.Errorf("expected recorded poll interval 5ms, got %s — injected pollInterval not honored", got)
	}
}

// TestCoordinatorEventHandler_RateLimitWindowInjected proves checkRateLimit
// consults the injected clock: a reversion to time.Now() leaves the
// post-advance event throttled, so the 11th broadcast never happens.
func TestCoordinatorEventHandler_RateLimitWindowInjected(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	var mu sync.Mutex

	broadcaster := func(event *websocket.TaskStreamEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	handler := NewCoordinatorEventHandler("task-guard-rl", "", broadcaster, WithClock(clock.Now))

	// Step 1 — burst at a frozen time; throttling MUST engage (anti-vacuity).
	for i := 0; i < 15; i++ {
		handler.OnText("message")
	}
	if !handler.IsThrottled() {
		t.Fatal("expected throttling to engage after 15 frozen-time events — guard would be vacuous")
	}
	mu.Lock()
	got := len(events)
	mu.Unlock()
	if got != 10 {
		t.Fatalf("expected exactly 10 broadcasts during throttling, got %d", got)
	}

	// Step 2 — advance the injected clock past the window; the next event must be broadcast.
	clock.Advance(1100 * time.Millisecond)
	handler.OnText("after reset")

	mu.Lock()
	got = len(events)
	mu.Unlock()
	if got != 11 {
		t.Errorf("expected 11 broadcasts after clock-driven window reset, got %d — injected clock not honored", got)
	}
}
```
All three helpers (`waitRecorder`, `tickRecorder`, `manualClock`) and `NewIntegrationMockProvider`/`NewTaskExecutor` are in the same package — defined in M2 / `integration_test.go` — so no further edits.

**E2 — run `gofmt -w internal/coordinator`, then the full Gate List (§4). M3 ends green.**

**E3 — (deferrable, see Cut set) Mutation-table execution per §3 procedure, and Local Measurement**: record once on this machine, append two rows to the design doc's `## Verification Log`:
| Measurement | Command | Base `f3783c976` | After M2 (record once) |
|---|---|---|---|
| Four-test combined duration | `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1 -json \| awk '/"Action":"pass"/ && /"Test":"Test(Integration_TaskExecutorWithRetry\|StoreBackedApprovalCheckpoint\|StoreBackedApprovalCheckpoint_Rejection\|CoordinatorEventHandler_RateLimitReset)"/ { match($0, /"Elapsed":[0-9.]+/); sum += substr($0, RSTART+10, RLENGTH-10) } END { printf "four-test sum: %.2fs\n", sum }'` | 8.14 s | expect < 0.5 s (target ~0.25 s) — record the observed value |
| Full-package wall | `time go test ./internal/coordinator/ -count=1` | 13.12 s | expect < 6 s — record the observed value |

These are local evidence, written only into the design doc's verification log (inside the worktree). They are **not** CI assertions and must never become gates.

### Acceptance criteria

| command | expected observation |
|---|---|
| `go test ./internal/coordinator/ -run 'TestRetryBaseDelayInjected\|TestStoreBackedApprovalCheckpoint_PollIntervalInjected\|TestCoordinatorEventHandler_RateLimitWindowInjected' -count=1 -v` | exit 0; exactly three `--- PASS:` lines, one per guard (base: `testing: warning: no tests to run`, exit 0) |
| `go test ./internal/coordinator/ -count=1` | exit 0 (base: 13.12 s, exit 0) |
| Mutation 1 red-check (procedure §3; deferrable per Cut set) | `TestRetryBaseDelayInjected` FAILS inside the sandbox, passes clean afterward |
| `git status --porcelain` | empty apart from the files this plan touches (and the design-doc verification-log rows if E3 landed) |
| Gate List (§4), all four commands | all green |

### Anti-vacuity note (per new guard)

- `TestRetryBaseDelayInjected`: kills the backoff reversion — with `baseDelay := time.Second` restored, the recorded waits are `[1s 2s]`, failing the `[5ms 10ms]` assertion; with the `Wait` seam bypassed, zero delays are recorded, failing the length assertion.
- `TestStoreBackedApprovalCheckpoint_PollIntervalInjected`: kills the poll reversion — a constructor ignoring the `pollInterval` param records `2s`, and a body bypassing `sac.tick` records nothing; both fail the `got != 5ms` assertion while the status assertion still passes (so the interval assertion is provably the killer).
- `TestCoordinatorEventHandler_RateLimitWindowInjected`: kills the clock reversion — with real `time.Now()` the window never resets at the frozen test instant, so the final event stays throttled and `got != 11`; the step-1 `IsThrottled()`/exactly-10 assertions guarantee throttling engaged, so the guard cannot pass vacuously under a shortened or broken window.

---

## 3. Mutation table (every mutant compiles — a build failure proves nothing)

**Execution procedure (per mutant; deferrable per Cut set; everything stays inside the worktree):**
```
cd <worktree root>
rm -rf .mutant-sandbox && mkdir .mutant-sandbox
tar -cf - --exclude=.git --exclude=.mutant-sandbox . | tar -xf - -C .mutant-sandbox
# apply exactly ONE mutant's edit inside .mutant-sandbox, then:
cd .mutant-sandbox && go build ./internal/coordinator/... && go vet ./internal/coordinator/...   # MUST pass — else the mutant is invalid, not caught
go test ./internal/coordinator/ -run '<killing test>' -count=1                                  # MUST FAIL (red)
cd .. && rm -rf .mutant-sandbox
git status --porcelain                                                                           # empty: the real tree is untouched
```
One mutant per sandbox; delete the sandbox after each; total budget 30 min. (`go` writes only to its standard build/module caches; all fixture writes stay inside `.mutant-sandbox/`.)

Expected-red rows reference the post-plan code (the text after this plan's edits):

| # | File · region (post-plan) | Mutant — exact change | Compiles? | Single test that MUST go RED | Killing assertion |
|---|---|---|---|---|---|
| 1 | `task_executor.go` · `ExecuteWithRetry` head | `baseDelay := opts.RetryBaseDelay` → `baseDelay := time.Second` (field and default kept) | yes (`opts.RetryBaseDelay` simply unread) | `TestRetryBaseDelayInjected` | recorded-delay assertion: records `[1s 2s]`, fails `delays[0] != 5ms \|\| delays[1] != 10ms` |
| 2 | `task_executor.go` · backoff block | `opts.Wait(delay)` → `time.Sleep(delay)` (seam bypass) | yes (all vars still used) | `TestRetryBaseDelayInjected` | length assertion: zero delays recorded → `len(delays) != 2` fatal |
| 3 | `approval_checkpoint.go` · constructor literal | `pollInterval: pollInterval,` → `pollInterval: 2 * time.Second,` (param ignored) | yes (unused function params are legal) | `TestStoreBackedApprovalCheckpoint_PollIntervalInjected` | recorded-interval assertion: `recordedInterval()` = `2s` ≠ `5ms` (status assertion still passes — the interval assertion is the killer) |
| 4 | `approval_checkpoint.go` · `RequestApproval` poll setup | `tickCh := sac.tick(sac.pollInterval)` → `tickCh := time.NewTicker(sac.pollInterval).C` (tick seam bypass) | yes | `TestStoreBackedApprovalCheckpoint_PollIntervalInjected` | recorded-interval assertion: `sac.tick` never called → `recordedInterval()` = `0` ≠ `5ms` (real 5ms ticker still resolves status, so again the interval assertion is the killer) |
| 5 | `event_handler.go` · `checkRateLimit` | `now := h.now()` → `now := time.Now()` (injected clock ignored) | yes | `TestCoordinatorEventHandler_RateLimitWindowInjected` | step-2 assertion: real clock never crosses the window at the frozen instant → still throttled → `got != 11` |
| 6 | `event_handler.go` · constructor | delete the option-application loop `for _, opt := range opts { opt(h) }` | yes (unused variadic param is legal) | `TestCoordinatorEventHandler_RateLimitWindowInjected` | same as #5 — `WithClock` never applied, real clock ticks, `got != 11` (M2's `RateLimitReset` also goes red; the guard is the required named test) |
| 7 | `provider.go` · `DefaultExecuteOptions` | `RetryBaseDelay: time.Second,` → `RetryBaseDelay: 2 * time.Second,` (production default change) | yes | **none of the three guards** — the guards override the field per call; this mutant is caught by M1's "defaults preserved" grep row: `grep -n 'RetryBaseDelay:.*time\.Second' internal/coordinator/provider.go` loses its match | M1 acceptance row (grep), not a `go test` |

Notes for the executor: mutants 3/4 and 5/6 are pairs attacking the same injection at different diff lines — run both members; the pair must be killed by the *same named assertion*, which is what proves the guard observes behavior through the seam rather than the field. Mutant 7 documents the design's last row verbatim: an accidental production-default change is a grep-tripwire catch, not a guard catch.

---

## 4. Gate List (run after EACH milestone; nothing else)

```
go build ./internal/coordinator/...
go vet ./internal/coordinator/...
gofmt -l internal/coordinator
go test ./internal/coordinator/ -count=1
```

- All four were verified green on the pristine base by the controller.
- `gofmt -l internal/coordinator` must print **nothing**.
- Do **not** run `go build ./...` (rc=1 on pristine dev for unrelated reasons) or `make test` (far outside this diff's blast radius). Adding gates is out of scope.

## 5. Cut set (if time runs out — in this order)

1. **First cut: Local Measurement recording** (M3-E3's two measurements + the design-doc verification-log rows). The latency win and the tripwire are unaffected; the numbers can be recorded by any follow-up.
2. **Second cut: mutation-table execution** (§3 sandbox red-checks). The table itself stays in this plan; executing it is deferrable.
3. **The three M3 behavioral guards are MANDATORY with M1/M2 (FIX 1) — they are never the cut.** If only M1 can land, the *next* cut is M2 itself (M1 alone changes nothing observable: defaults are preserved and the package gates stay green).

## 6. Out of scope (explicit)

- **No `t.Parallel()` is added anywhere** — not in the four named tests, not elsewhere in the package.
- **No test outside the four named tests is modified.** The only other test-file changes are *additions*: three package-level helper types (`waitRecorder`, `tickRecorder`, `manualClock`) used solely by the four tests and the three new guards, plus the new guard file.
- **No production behavior changes.** Defaults are preserved exactly: retry backoff base `1s` (now set explicitly by `DefaultExecuteOptions()` and inherited by the one production caller via rebase — each override wins); poll interval `2s` + real ticker (now passed explicitly by the only two callers, both tests — zero production callers); rate-limit window `1s` and `maxEventsPerSec: 10` unchanged; `WithClock` is opt-in via an unused-by-default variadic option, so the one production constructor call is source-unchanged. Within the design's settled direction (FIX 1/FIX 2) two mechanics are noted, not decided: ctx cancellation during backoff is observed when `opts.Wait` returns instead of interrupting `time.After`, and the unreferenced poll ticker is GC-collected (Go ≥ 1.23; module targets Go 1.26).
- No changes to the Windows CI timeout situation, the other 125 fast tests, the 13 `exec.Command` / 5 `os.MkdirTemp` tests, or anything outside `internal/coordinator`.

---

## Plan Verification Log (run by the planner at base `f3783c9765d1fcdca51533c401bea6a7e53d19b8`)

Every load-bearing "the code currently says X" claim in this plan is backed by one of these rows, all executed in this session in the worktree. Negative grep readings are paired with a same-call positive control.

| # | Command | Observed output (verbatim key lines) |
|---|---|---|
| P1 | `grep -n 'baseDelay := time.Second' internal/coordinator/task_executor.go` | `158:	baseDelay := time.Second` (1 match). Control same call `grep -n 'baseDelay' …` → also `163: delay := baseDelay * time.Duration(1<<(attempt-1))` |
| P2 | `grep -n 'pollInterval' internal/coordinator/approval_checkpoint.go` | `330: pollInterval time.Duration` · `338: pollInterval:       2 * time.Second,` · `387: pollTicker := time.NewTicker(sac.pollInterval)` (3 matches) |
| P3 | `grep -n 'func (h \*CoordinatorEventHandler) IsThrottled' internal/coordinator/event_handler.go` | `306:func (h *CoordinatorEventHandler) IsThrottled() bool {` — exists at base; nothing to add (FIX 2 check) |
| P4 | `grep -n 'DefaultExecuteOptions' internal/coordinator/provider.go` | `98:// DefaultExecuteOptions returns sensible defaults` · `99:func DefaultExecuteOptions() *ExecuteOptions {` |
| P5 | `sed -n '230,250p' internal/coordinator/daemon_tasks_exec_run.go` | literal at :238–244: `opts := &ExecuteOptions{ Timeout: agentConfig.GetEffectiveTimeout() …, IdleTimeout: …, Workspace: workspacePath, ObservatoryContext: obsContext, AgentConfig: agentConfig }` — five fields, exactly as quoted in M1-E4 |
| P6 | `sed -n '325,345p' internal/coordinator/approval_checkpoint.go` | struct (:327–331) with `pollInterval time.Duration`; ctor (:334) `func NewStoreBackedApprovalCheckpoint(store ApprovalStore, defaultTimeout time.Duration) *StoreBackedApprovalCheckpoint {` with `pollInterval: 2 * time.Second` — exactly as quoted in M1-E5 |
| P7 | `sed -n '150,175p' internal/coordinator/task_executor.go` | `ExecuteWithRetry` :152; nil-opts default :153–155; `baseDelay := time.Second` :158; backoff `select { case <-time.After(delay): case <-ctx.Done(): }` :164–168 — exactly as quoted in M1-E3 |
| P8 | `sed -n '270,315p' internal/coordinator/event_handler.go` | `checkRateLimit` :275 with `now := time.Now()` :279 and `>= time.Second` window :282; `IsThrottled` body :306–310 — exactly as quoted in M1-E9 |
| P9 | call-site census: `grep -rn 'NewStoreBackedApprovalCheckpoint\|ExecuteWithRetry\|NewCoordinatorEventHandler' --include='*.go' internal/ cmd/` | checkpoint ctor: **only** `approval_checkpoint_edge_test.go:21,76` (zero production); `ExecuteWithRetry`: definition + `daemon_tasks_exec_run.go:333` + `integration_test.go:221` (the only two callers); `NewCoordinatorEventHandler`: `daemon_tasks_exec_run.go:292` + 7 test call sites |
| P10 | `sed -n '45,110p' internal/coordinator/provider.go` | `ExecuteOptions` :54–82 (`Plugins *PluginsConfig` :81); `DefaultExecuteOptions` body returns only `{Timeout: 5 * time.Minute, DryRun: false}` — must gain both new fields (FIX 2) |
| P11 | `sed -n '343,430p' internal/coordinator/approval_checkpoint.go` | `RequestApproval` poll setup :386–389 (`pollTicker := time.NewTicker(sac.pollInterval)`, `defer pollTicker.Stop()`), `case <-pollTicker.C:` — exactly as quoted in M1-E6 |
| P12 | `sed -n '99,145p' + '209,273p' internal/coordinator/event_handler.go` | `OnText` broadcasts iff `checkRateLimit()` true; `checkRateLimit` increments `eventCount`, drops when `> maxEventsPerSec (10)`; ctor sets `maxEventsPerSec: 10` — grounds the "exactly 10 / exactly 11" assertions |
| P13 | `sed -n '40,80p' + '1,40p' internal/coordinator/event_handler.go` | rate-limit field block :40–45; ctor :61–71 non-variadic — exactly as quoted in M1-E7/E8; package imports `sync`,`time`,`websocket` |
| P14 | `sed -n '175,235p' internal/coordinator/integration_test.go` + `sed -n '1,17p'` | `TestIntegration_TaskExecutorWithRetry` :186; opts literal :216–219 (`Timeout: 30 * time.Second`, `Workspace: t.TempDir()`); imports `context, testing, time` (no `sync`) — M2-E1 required |
| P15 | `sed -n '1,120p' internal/coordinator/approval_checkpoint_edge_test.go` (531 lines) | full text of both target tests as quoted in M2-E6/E7; ctor calls at :21 and :76; `sync`/`fmt` used by later tests (:115, :154, :199, :203, :208, :249, :260) so imports survive the rewrite |
| P16 | `sed -n '100,170p' internal/coordinator/event_handler_test.go` | `TestCoordinatorEventHandler_RateLimitReset` :123–153 verbatim as quoted in M2-E9, incl. `time.Sleep(1100 * time.Millisecond)`; imports `sync, testing, time, websocket` already sufficient |
| P17 | name-collision check `grep -rn 'waitRecorder\|tickRecorder\|manualClock\|WithClock\|defaultPollTick\|RetryBaseDelay' internal/coordinator/` | **no matches** (negative); paired control `grep -rn 'DefaultExecuteOptions' internal/coordinator/` → 4 hits (provider.go, provider_executor.go:67, task_executor.go:82,154) — instrument works |
| P18 | `grep -n 'func NewIntegrationMockProvider\|func NewTaskExecutor' internal/coordinator/*.go` + `grep -n 'func (m \*IntegrationMockProvider)' …` | `integration_test.go:18`, `task_executor.go:23` (variadic `...Provider`); `SetExecuteFunc` at :52 — the M3 retry guard's mock pattern is valid same-package reuse |
| P19 | `grep -n 'time\.' internal/coordinator/task_executor.go` | only :158, :163, :165 — after M1-E3 `time.Duration` at :163 keeps the import alive (no unused-import break) |
| P20 | `head -5 go.mod` | `go 1.26.6` — unreferenced tickers are GC-collected (≥ 1.23), so the tick seam dropping `pollTicker.Stop()` leaks nothing |
| P21 | `grep -n 'now := time.Now()' internal/coordinator/event_handler.go` | single match at :279 — the M1-E9 anchor is unique |
| P22 | `grep -n 'type ApprovalCheckpoint struct' internal/coordinator/approval_checkpoint.go` (+12 lines) | :69; `defaultTimeout` lives on the embedded base type — untouched by this plan |

*Planner: mission V1 iteration 351 (sprint-planner, headless). Design direction per revision 3 is settled; this plan adds no scope and proposes no alternatives.*
