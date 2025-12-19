# M-EVAL-AGENT-QUEUE: Queue-Based Agent Evaluation Architecture

**Status**: Planned (Follow-up to M-EVAL-AGENT)
**Priority**: Medium (Phase 2 enhancement)
**Estimated Effort**: 4-6 hours
**Depends On**: M-EVAL-AGENT (Milestone 1.1-1.3 complete)

## Context

M-EVAL-AGENT Milestone 1.3 implemented **direct spawning** of headless Claude sessions from the eval harness. This works well and provides:
- ✅ Agent-based evaluation with multi-turn repair
- ✅ Parallel execution (10 concurrent sessions)
- ✅ Rate limiting (API quota management)
- ✅ Unique workspace isolation per session
- ✅ Multi-language support (AILANG, Python)

**What's missing**: The original design doc specified using **AILANG agent protocol as a task queue** for robustness and crash recovery.

## Problem Statement

Current implementation has limitations:
1. **No crash recovery**: If eval harness crashes mid-run, in-progress benchmarks are lost
2. **No state persistence**: Can't resume interrupted eval runs
3. **Limited observability**: Can't inspect queue state or task progress
4. **Doesn't dogfood**: Not using AILANG's own agent messaging system

## Proposed Solution

Upgrade to **queue-based architecture** using AILANG agent inbox as persistent task queue.

### Architecture

```
Eval Harness (ailang eval-suite --agent --queue)
    ↓
1. Send all benchmarks to eval-agent inbox as tasks
    ↓
2. Poll inbox for unread tasks (rate-limited)
    ↓
3. For each task, spawn headless Claude session
    │
    ├─→ Session 1: list_map      → posts result to inbox → ack task
    ├─→ Session 2: tree_traversal → posts result to inbox → ack task
    ├─→ ...
    └─→ Session 10: factorial    → posts result to inbox → ack task
    ↓
4. Collect all results from inbox
    ↓
5. Generate eval report
```

### Key Components

#### 1. Task Queue (.ailang/state/messages/eval-agent/)

**Task Message Format** (sent to eval-agent inbox):
```json
{
  "type": "eval_task",
  "task_id": "hello_world_claude-sonnet-4-5_ailang_1698765432",
  "benchmark_id": "hello_world",
  "model": "claude-sonnet-4-5",
  "language": "ailang",
  "spec": {
    "description": "Print hello world",
    "caps": ["IO"],
    "expected_output": "Hello, World!\n"
  },
  "config": {
    "timeout_seconds": 300,
    "max_iterations": 10,
    "workspace_dir": "/tmp/ailang_eval/hello_world_..."
  },
  "timestamp": "2025-10-27T19:40:00Z"
}
```

**Result Message Format** (posted back to inbox):
```json
{
  "type": "eval_result",
  "task_id": "hello_world_claude-sonnet-4-5_ailang_1698765432",
  "success": true,
  "duration_ms": 45123,
  "num_turns": 3,
  "cost": 0.0234,
  "usage": {
    "input_tokens": 1234,
    "output_tokens": 567,
    "cache_read_input_tokens": 890
  },
  "error": "",
  "session_id": "session_abc123",
  "timestamp": "2025-10-27T19:45:00Z"
}
```

#### 2. Queue Manager (internal/eval_harness/queue_manager.go)

```go
// QueueManager handles agent inbox-based task queue
type QueueManager struct {
    agentID      string // "eval-agent"
    stateDir     string // ".ailang/state"
    rateLimiter  *time.Ticker
    maxConcurrent int
}

// EnqueueBenchmarks sends all benchmarks to agent inbox
func (qm *QueueManager) EnqueueBenchmarks(specs []*BenchmarkSpec, config AgentBenchmarkConfig) error {
    for _, spec := range specs {
        task := createTaskMessage(spec, config)
        if err := qm.sendTask(task); err != nil {
            return fmt.Errorf("failed to enqueue %s: %w", spec.ID, err)
        }
    }
    return nil
}

// ProcessQueue polls inbox and spawns headless sessions
func (qm *QueueManager) ProcessQueue(ctx context.Context) ([]*AgentBenchmarkResult, error) {
    results := []*AgentBenchmarkResult{}
    sem := make(chan struct{}, qm.maxConcurrent)

    for {
        // Poll for unread tasks
        tasks, err := qm.pollUnreadTasks()
        if err != nil {
            return nil, err
        }

        if len(tasks) == 0 {
            break // All tasks processed
        }

        for _, task := range tasks {
            select {
            case <-ctx.Done():
                return results, ctx.Err()
            case <-qm.rateLimiter.C:
                sem <- struct{}{} // Acquire semaphore

                go func(t *TaskMessage) {
                    defer func() { <-sem }() // Release semaphore

                    // Spawn headless Claude session
                    result := qm.processTask(t)

                    // Post result back to inbox
                    qm.postResult(result)

                    // Acknowledge task (moves to _processed)
                    qm.ackTask(t.TaskID)

                    results = append(results, result)
                }(task)
            }
        }
    }

    // Wait for all goroutines to finish
    for i := 0; i < qm.maxConcurrent; i++ {
        sem <- struct{}{}
    }

    return results, nil
}

// pollUnreadTasks reads unread messages from eval-agent inbox
func (qm *QueueManager) pollUnreadTasks() ([]*TaskMessage, error) {
    // Use: ailang agent inbox eval-agent --unread-only
    // Or call internal/agent/inbox.go directly
}

// postResult sends result back to inbox
func (qm *QueueManager) postResult(result *AgentBenchmarkResult) error {
    // Use: ailang agent send eval-agent '{"type": "eval_result", ...}'
    // Or call internal/agent/send.go directly
}

// ackTask marks task as processed
func (qm *QueueManager) ackTask(taskID string) error {
    // Use: ailang agent ack <message-id>
    // Moves from _unread to _processed
}
```

#### 3. CLI Integration (cmd/ailang/eval_suite.go)

Add `--queue` flag to enable queue-based architecture:

```go
// Agent mode flags
agent := fs.Bool("agent", false, "Use agent-based evaluation (Claude Code headless mode)")
agentQueue := fs.Bool("queue", false, "Use agent inbox as task queue (enables crash recovery)")
// ... other flags

if *agent {
    if *agentQueue {
        // Queue-based architecture
        queueManager := eval_harness.NewQueueManager("eval-agent", agentConfig)

        // Enqueue all benchmarks
        if err := queueManager.EnqueueBenchmarks(specs, *agentConfig); err != nil {
            log.Fatal(err)
        }

        // Process queue with parallelism + rate limiting
        results, err := queueManager.ProcessQueue(context.Background())
        if err != nil {
            log.Fatal(err)
        }

        // Save results
        for _, result := range results {
            logger.Log(convertToRunMetrics(result))
        }
    } else {
        // Direct spawning (current implementation)
        results := runBenchmarksParallel(jobs, ..., agentConfig)
    }
}
```

### Benefits

1. **Crash Recovery**:
   - Unacked tasks remain in `_unread` directory
   - Re-run `ailang eval-suite --agent --queue` to resume
   - No lost work, no duplicate runs

2. **State Persistence**:
   - Task queue survives process crashes
   - Results persist in inbox until collected
   - Easy to inspect: `ailang agent inbox eval-agent`

3. **Observability**:
   - Check progress: `ailang agent inbox eval-agent --unread-only | wc -l`
   - Inspect tasks: `ailang agent inbox eval-agent`
   - Retry failed tasks: `ailang agent unack <msg-id>`

4. **Dogfooding**:
   - Uses AILANG's own agent protocol
   - Tests message passing at scale (264 tasks)
   - Validates inbox scalability

5. **Distributed Execution** (future):
   - Multiple workers can poll same queue
   - Scale horizontally across machines
   - Cloud-native architecture

### Implementation Plan

**Phase 1: Core Queue Manager** (~2 hours)
- [ ] Create `internal/eval_harness/queue_manager.go`
- [ ] Implement task serialization/deserialization
- [ ] Add `EnqueueBenchmarks()` method
- [ ] Add `pollUnreadTasks()` method
- [ ] Add `postResult()` method
- [ ] Add `ackTask()` method

**Phase 2: CLI Integration** (~1 hour)
- [ ] Add `--queue` flag to `eval-suite` command
- [ ] Wire queue manager to CLI
- [ ] Add queue status reporting

**Phase 3: Testing** (~1-2 hours)
- [ ] Unit tests for queue manager
- [ ] Integration test: enqueue → process → collect
- [ ] Crash recovery test: kill process mid-run, resume
- [ ] Parallel execution test: 10 concurrent sessions

**Phase 4: Documentation** (~30 min)
- [ ] Update README with `--queue` flag usage
- [ ] Add crash recovery guide
- [ ] Document queue inspection commands

### Usage Examples

**Queue-based evaluation:**
```bash
# Enqueue all benchmarks and process with queue
ailang eval-suite --agent --queue \
  --models claude-sonnet-4-5 \
  --benchmarks hello_world,fizzbuzz,factorial \
  --langs ailang \
  --agent-parallel 10 \
  --output eval_results/queue_test
```

**Resume interrupted run:**
```bash
# Same command - automatically resumes from queue
ailang eval-suite --agent --queue \
  --models claude-sonnet-4-5 \
  --benchmarks hello_world,fizzbuzz,factorial \
  --langs ailang \
  --agent-parallel 10 \
  --output eval_results/queue_test
```

**Inspect queue state:**
```bash
# Check remaining tasks
ailang agent inbox eval-agent --unread-only

# Check completed results
ailang agent inbox eval-agent --read-only

# Retry failed task
ailang agent unack msg_20251027_194523_a5f3e77ee975
```

### Migration Path

1. **v0.4.0** (now): Direct spawning works (Milestone 1.3 complete)
2. **v0.4.1** (optional): Add queue-based architecture with `--queue` flag
3. **v0.4.2** (optional): Make queue default, keep direct as `--no-queue` fallback

### Risks & Mitigations

**Risk 1: Agent inbox scalability**
- Concern: Can inbox handle 264 messages efficiently?
- Mitigation: Test with full suite, measure read/write performance
- Fallback: Optimize inbox implementation if needed

**Risk 2: Message serialization overhead**
- Concern: JSON encoding/decoding adds latency
- Mitigation: Use efficient JSON encoding, benchmark overhead
- Fallback: Use binary format (protobuf) if needed

**Risk 3: Queue complexity**
- Concern: More complex than direct spawning
- Mitigation: Keep direct spawning as fallback (`--no-queue`)
- Fallback: Revert to direct spawning if issues arise

### Success Metrics

- [ ] Queue-based eval matches direct spawning results (correctness)
- [ ] Crash recovery works: kill process mid-run, resume successfully
- [ ] Performance: <5% overhead vs direct spawning
- [ ] Scalability: Handles 264 tasks without degradation
- [ ] Observability: Can inspect queue state at any time

### References

- Original design: [M-EVAL-AGENT.md](M-EVAL-AGENT.md)
- Agent protocol: `internal/agent/` (inbox, send, ack commands)
- Current implementation: `internal/eval_harness/agent_runner.go`
- CLI integration: `cmd/ailang/eval_suite.go`

### Decision Log

**2025-10-27**: Deferred to Phase 2
- Reason: Direct spawning (Milestone 1.3) provides 80% value with 20% complexity
- Decision: Ship direct spawning first, validate concept, then upgrade to queue
- Timeline: Queue architecture optional for v0.4.1+ if crash recovery becomes critical

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Persistent queue enables deterministic crash recovery |
| A2: Replayability | +1 | Task queue preserves full eval run history |
| A3: Effect Legibility | 0 | No change to effect visibility |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Task status verifiable per-benchmark |
| A6: Safe Concurrency | +1 | Ack-based queue prevents duplicate processing |
| A7: Machines First | +1 | Uses AILANG's agent messaging for dogfooding |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-task cost tracking in results |
| A10: Composability | +1 | Extends existing M-EVAL-AGENT infrastructure |
| A11: Structured Failure | +1 | Task failures isolated and retryable |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Queue order is deterministic
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Dogfoods AILANG's own agent protocol
