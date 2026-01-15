# Sprint Plan: M-UNIFIED-AI-CONTROL-PLANE

## Summary
Implement `ailang exec` as the unified AI execution command, consolidate environment setup, add message hierarchy support, and migrate eval suite to message-based coordination.

**Duration:** 4-5 days (~18-20 hours)
**Dependencies:** M-CONTROL-PLANE-V4-INTEGRATION (partially complete)
**Risk Level:** Medium

## Current Status Analysis

### Completed Recently
- Claude executor with OTEL tracing (~500 LOC) - fully operational
- Gemini executor with streaming NDJSON (~350 LOC) - fully operational
- Task hierarchy via resource attributes - implemented in claude.go
- Messaging system with correlation_id support - operational
- Observatory integration - functional

### Velocity
- Recent average: ~150 LOC/day (based on executor implementations)
- Estimated capacity: ~800 LOC for this sprint

### Remaining from Design Doc
- Phase 1: `ailang exec` command - ~250 LOC
- Phase 2: Environment consolidation - ~100 LOC (net -50 after dedup)
- Phase 3: Coordinator integration - ~50 LOC
- Phase 4: API provider mode - ~100 LOC
- Phase 5: Message hierarchy - ~100 LOC
- Phase 6: Eval suite migration - ~-300 LOC (deletion)

**Net change: ~400 LOC reduction**

## Proposed Milestones

### Milestone 1: Unified `ailang exec` Command
**Goal:** Create the central entry point for all AI executions with provider selection and NDJSON streaming output.
**Estimated:** 200 LOC implementation + 50 LOC tests = 250 LOC
**Duration:** 1 day

**Files to Create/Modify:**
- Create `cmd/ailang/exec.go` (~200 LOC)
- Modify `cmd/ailang/main.go` - register exec subcommand (~10 LOC)

**Tasks:**
- Create exec.go with subcommand parsing
- Add provider selection: `ailang exec claude|gemini|openai|ollama`
- Add flags: `--workspace`, `--model`, `--timeout`, `--task-id`, `--system-prompt`
- Implement NDJSON streaming output format
- Create root span with resource attributes

**Acceptance Criteria:**
- [ ] `ailang exec claude "hello"` works with streaming output
- [ ] `ailang exec gemini "hello"` works with streaming output
- [ ] `--workspace` flag sets working directory
- [ ] `--task-id` flag sets span attributes
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Provider path resolution edge cases - Mitigation: Use existing executor factory

---

### Milestone 2: Environment Consolidation
**Goal:** Extract shared environment setup into a single function to eliminate ~100 LOC of duplication.
**Estimated:** 100 LOC new, -100 LOC removed = net 0
**Duration:** 0.5 days

**Files to Create/Modify:**
- Create `internal/executor/environment.go` (~100 LOC)
- Modify `internal/executor/claude/claude.go` - use shared function (~-50 LOC)
- Modify `internal/executor/gemini/gemini.go` - use shared function (~-50 LOC)

**Tasks:**
- Create `BuildEnvironment(task *Task, sessionID string) []string` function
- Extract OTEL endpoint configuration
- Extract stdlib path configuration
- Extract resource attribute building
- Update Claude executor to use shared function
- Update Gemini executor to use shared function

**Acceptance Criteria:**
- [ ] Both executors use shared `BuildEnvironment()`
- [ ] Environment includes OTEL endpoint, stdlib path, resource attrs
- [ ] No functional regression in coordinator tasks
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Environment variable ordering matters - Mitigation: Test with both providers

---

### Milestone 3: Coordinator Integration
**Goal:** Update coordinator to use `ailang exec` internally for all task execution.
**Estimated:** 50 LOC
**Duration:** 0.5 days

**Files to Modify:**
- `internal/coordinator/daemon_tasks.go` (~50 LOC)

**Tasks:**
- Add `--register-task` flag handling
- Wire task_id from coordinator → exec → Observatory
- Update executeTask to pass parent context

**Acceptance Criteria:**
- [ ] Coordinator tasks create linked spans in Observatory
- [ ] Task hierarchy visible in Control Plane dashboard
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Breaking existing coordinator flows - Mitigation: Gradual migration

---

### Milestone 4: API Provider Mode
**Goal:** Add `--api-only` flag for direct API calls without CLI tools.
**Estimated:** 100 LOC
**Duration:** 0.5 days

**Files to Modify:**
- `cmd/ailang/exec.go` (~100 LOC)

**Tasks:**
- Add `--api-only` flag
- Integrate `internal/ai/` providers (openai, anthropic, gemini, ollama)
- Ensure same tracing/metadata for API vs CLI modes
- Stream API responses as NDJSON events

**Acceptance Criteria:**
- [ ] `ailang exec openai "prompt" --api-only` works
- [ ] `ailang exec anthropic "prompt" --api-only` works
- [ ] Traces appear with same resource attributes as CLI mode
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Different response formats per provider - Mitigation: Normalize in event handler

---

### Milestone 5: Message Hierarchy Integration
**Goal:** Add `--parent-task` and `--correlation-id` flags to messages for task hierarchy.
**Estimated:** 80 LOC changes + 20 LOC schema = 100 LOC
**Duration:** 0.5 days

**Files to Modify:**
- `cmd/ailang/messages_send.go` - add flags (~20 LOC)
- `internal/messaging/inbox.go` - add ParentTaskID field (~10 LOC)
- `internal/messaging/schema.go` - add column (~20 LOC)
- `internal/coordinator/daemon_tasks.go` - extract hierarchy (~30 LOC)

**Tasks:**
- Add `--parent-task` flag to `ailang messages send`
- Add `parent_task_id` column to `inbox_messages` table
- Store hierarchy metadata on message creation
- Update coordinator to extract and pass hierarchy to exec
- Add `--parent-task-id` flag to `ailang exec` for span linking

**Acceptance Criteria:**
- [ ] `ailang messages send ... --parent-task=X` stores parent reference
- [ ] Child tasks inherit parent context automatically
- [ ] Observatory shows task hierarchy tree
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Schema migration for existing databases - Mitigation: Use nullable column

---

### Milestone 6: Eval Suite Migration to Messages
**Goal:** Replace custom parallel job management with message-based coordination.
**Estimated:** ~-300 LOC (deletion of custom code)
**Duration:** 1 day

**Files to Modify:**
- `cmd/ailang/eval_suite.go` - major refactor (~-300 LOC)
- Create `eval-runner` inbox configuration

**Tasks:**
- Create `eval-runner` inbox in coordinator config
- Refactor `runBenchmarksParallel()` to use `ailang messages send` per benchmark
- Delete custom job queue management (~100 LOC)
- Delete custom result aggregation (~50 LOC)
- Add `--queue` flag to enable message-based coordination
- Implement result collection via message acknowledgement
- Add crash recovery via unacked messages
- Test with full suite (264 benchmarks)

**Acceptance Criteria:**
- [ ] `ailang eval-suite --queue` runs benchmarks via messages
- [ ] Crash recovery works (unacked messages resume)
- [ ] Results appear in Observatory with hierarchy
- [ ] Performance comparable to direct execution
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Performance regression from message overhead - Mitigation: Keep direct mode as default
- Partial failure handling - Mitigation: Transaction-like message batching

---

## Implementation Order

```
Day 1: M1 (exec command) - Foundation for everything else
Day 2: M2 (environment) + M3 (coordinator) - Build on exec
Day 3: M4 (API mode) + M5 (hierarchy) - Extend capabilities
Day 4: M6 (eval migration) - Major refactor requiring prior pieces
Day 5: Integration testing, documentation, cleanup
```

## Success Metrics
- Test coverage: >80% on new code
- `make test` passing
- `make lint` passing
- Examples: `ailang exec claude "hello"` works
- Documentation: Update `docs/guides/coordinator.md`

## Dependencies
- Existing executor implementations (Claude, Gemini)
- Existing messaging system with correlation_id
- Observatory integration

## Open Questions
- Should `--api-only` be the default for some providers?
- Should eval migration be opt-in (`--queue`) or default?

## Notes
- Prioritize CLI executors over API mode (more capable)
- Keep direct eval execution as default for backwards compatibility
- Message hierarchy enables future cost aggregation features
