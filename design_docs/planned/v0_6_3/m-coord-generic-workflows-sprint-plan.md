# Sprint Plan: M-COORD-GENERIC-WORKFLOWS

## Summary
Make the coordinator workflow fully configurable so different projects can define their own skills, stages, and approval rules. This removes all AILANG-specific hardcoding from coordinator code.

**Duration:** 2 days (~8-10 hours)
**Dependencies:** M-COORD-TASKCHAIN-TESTS (COMPLETE - provides regression safety)
**Risk Level:** Medium (refactoring core pipeline logic)

## Current Status Analysis

### Completed Recently
- M-COORD-TASKCHAIN-TESTS: 897 LOC in 0.5 days (25 test functions)
- M-COORD-APPROVALWATCHER-OBSERVABILITY: 150 LOC in 0.5 days

### Velocity
- Recent average: ~300-400 LOC/day for coordinator work
- Test coverage: ~65-70% for TaskChain after recent tests
- Estimated capacity: ~400-500 LOC for this sprint

### Current Hardcoded Elements
1. **Skill names** in `stage_execution.go`: "design-doc-creator", "sprint-planner", "sprint-executor"
2. **Output markers**: `DESIGN_DOC_PATH:`, `SPRINT_PLAN_PATH:`, `IMPLEMENTATION_COMPLETE:`
3. **Approval labels** in `approval_watcher.go`: "needs-design-approval", "design-approved", etc.
4. **Stage transitions** in `task_chain.go`: Design → Sprint → Implementation → Merge

## Proposed Milestones

### Milestone 1: Add InvokeConfig and ApprovalConfig
**Goal:** Extend AgentConfig with invoke and approval configuration
**Estimated:** 100 LOC implementation + 50 LOC tests
**Duration:** 1 hour

**Tasks:**
- Add InvokeConfig struct (type, name, template fields)
- Add ApprovalConfig struct (needs_label, approved_label, github_comment_template)
- Add OutputMarkers field to AgentConfig
- Write unit tests for config parsing

**Acceptance Criteria:**
- [ ] InvokeConfig struct supports: skill, agent, prompt types
- [ ] ApprovalConfig struct supports configurable labels
- [ ] OutputMarkers is []string on AgentConfig
- [ ] Config parsing tests pass

**Files:**
- `internal/coordinator/agent_registry.go` - Add structs (~60 LOC)
- `internal/coordinator/agent_registry_test.go` - Tests (~50 LOC)

### Milestone 2: Config-Driven Directive Building
**Goal:** Replace hardcoded `buildDesignDirective` etc. with config-driven approach
**Estimated:** 120 LOC implementation + 60 LOC tests
**Duration:** 1.5 hours

**Tasks:**
- Create `BuildDirectiveFromConfig(task, agent)` function
- Support all three invoke types (skill, agent, prompt)
- Template expansion for prompt type
- Remove or deprecate legacy build*Directive functions

**Acceptance Criteria:**
- [ ] BuildDirectiveFromConfig generates correct directive for skill type
- [ ] BuildDirectiveFromConfig generates correct directive for agent type
- [ ] BuildDirectiveFromConfig generates correct directive for prompt type
- [ ] Legacy functions have deprecation comments

**Files:**
- `internal/coordinator/stage_execution.go` - Refactor (~80 LOC)
- `internal/coordinator/stage_execution_test.go` - Tests (~60 LOC)

### Milestone 3: Generic Output Marker Parsing
**Goal:** Replace hardcoded ParseStageOutput with config-driven marker extraction
**Estimated:** 80 LOC implementation + 40 LOC tests
**Duration:** 1 hour

**Tasks:**
- Create `ParseOutputMarkers(output, markers)` function
- Return map[string]string of marker -> value
- Support any configured markers, not just built-in ones
- Update ProcessStageCompletion to use generic parsing

**Acceptance Criteria:**
- [ ] ParseOutputMarkers extracts any configured marker
- [ ] Returns empty map when no markers found
- [ ] Works with multi-value markers (comma-separated)
- [ ] ProcessStageCompletion uses agent.OutputMarkers

**Files:**
- `internal/coordinator/stage_execution.go` - Refactor (~50 LOC)
- `internal/coordinator/stage_execution_test.go` - Tests (~40 LOC)

### Milestone 4: Config-Driven Approval Flow
**Goal:** Make ApprovalWatcher and TaskChain use agent config for labels
**Estimated:** 180 LOC implementation + 80 LOC tests
**Duration:** 2 hours

**Tasks:**
- Add agent registration to ApprovalWatcher (map approved_label -> agent)
- Update checkIssueLabels to use registered labels
- Create generic OnAgentComplete in TaskChain
- Create generic OnApprovalDetected in TaskChain
- Update TriggerNextAgents to use agent.TriggerOnComplete

**Acceptance Criteria:**
- [ ] ApprovalWatcher.RegisterAgentApproval() works
- [ ] checkIssueLabels detects config-driven labels
- [ ] OnAgentComplete posts correct labels from config
- [ ] OnApprovalDetected triggers correct next agents
- [ ] needs-revision still works (universal)

**Files:**
- `internal/coordinator/approval_watcher.go` - Add registration (~60 LOC)
- `internal/coordinator/task_chain.go` - Generic handlers (~80 LOC)
- `internal/coordinator/task_chain_test.go` - Tests (~80 LOC)

### Milestone 5: Default Config and Backwards Compatibility
**Goal:** Provide default config for existing AILANG workflow
**Estimated:** 100 LOC implementation + 40 LOC tests
**Duration:** 1 hour

**Tasks:**
- Create DefaultInvokeConfig(agentID) for legacy support
- Create DefaultApprovalConfig(agentID) for legacy support
- Apply defaults when config fields are nil
- Add deprecation warning when using defaults
- Update coordinator config loading

**Acceptance Criteria:**
- [ ] Existing AILANG config continues to work
- [ ] Warning logged when using default configs
- [ ] New agents with explicit config work correctly
- [ ] Documentation updated with new config format

**Files:**
- `internal/coordinator/agent_registry.go` - Default functions (~60 LOC)
- `internal/coordinator/daemon.go` - Config loading (~30 LOC)
- `internal/coordinator/agent_registry_test.go` - Tests (~40 LOC)

### Milestone 6: E2E Integration Test
**Goal:** Verify full pipeline works with new config
**Estimated:** 0 LOC implementation (E2E only)
**Duration:** 30 minutes

**Tasks:**
- Test AILANG workflow with explicit invoke config
- Test custom project simulation (different skill names)
- Verify backwards compatibility with existing config

**Acceptance Criteria:**
- [ ] AILANG workflow: design → sprint → implement works
- [ ] Custom skill names work in directive
- [ ] Custom approval labels work
- [ ] All existing coordinator tests pass

## Success Metrics
- Hardcoded skill names removed: 3 → 0
- Hardcoded label constants: 8 → 1 (only needs-revision)
- Config-driven agent setup: 100%
- Test coverage maintained: >65% for affected files
- All tests passing
- All linting passing

## Dependencies
- TaskChain tests (COMPLETE - M-COORD-TASKCHAIN-TESTS)
- Existing AgentRegistry (stable)
- Existing ApprovalWatcher (stable)

## Risks
1. **Breaking existing config** - Mitigated by backwards compatibility layer
2. **Complex config migrations** - Mitigated by optional fields with defaults
3. **Test coverage gaps** - Mitigated by existing TaskChain tests

## Notes
- Phase 7 (Comment Templates) deferred - lower priority, can be separate sprint
- Focus on core generic workflow first
- Estimated total: ~580 LOC new code + tests
