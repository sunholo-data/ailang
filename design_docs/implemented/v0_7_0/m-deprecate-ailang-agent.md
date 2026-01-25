# M-DEPRECATE-AILANG-AGENT: Deprecate Standalone Agent Binary

**Status:** PLANNED
**Version:** v0.6.3
**Priority:** MEDIUM
**Estimated Effort:** 1-2 days
**Prerequisites:** v0.6.2 (Coordinator with GitHub Auto-Routing)

## Summary

Deprecate and remove the standalone `ailang-agent` binary (`cmd/ailang-agent/`) and its supporting package (`internal/agent/`). The coordinator daemon (`ailang coordinator`) now provides all agent functionality with significant improvements.

## Background

The `ailang-agent` binary was introduced in v0.4.5 as an experimental background agent. Since then, the coordinator system (v0.5.0+) has evolved to be the primary agent execution mechanism with features that supersede the standalone binary.

### Current State

| Component | Version | Status | Purpose |
|-----------|---------|--------|---------|
| `cmd/ailang-agent/` | v0.4.5-dev | **FROZEN** | Standalone polling agent |
| `internal/agent/` | v0.4.5 | **ORPHANED** | Only used by ailang-agent |
| `ailang coordinator` | v0.6.2+ | **ACTIVE** | Integrated daemon with advanced features |

### Why Deprecate?

1. **Duplicate Functionality**: Both systems poll messages and execute tasks
2. **Maintenance Burden**: Two codepaths doing similar things
3. **Feature Gap**: Coordinator has features agent never got (worktrees, GitHub, task chains)
4. **Confusion**: Users unsure which to use

## Feature Comparison

### Execution Architecture

| Aspect | ailang-agent | coordinator |
|--------|-------------|-------------|
| **Binary** | Standalone (`ailang-agent`) | Integrated (`ailang coordinator`) |
| **Executor** | `eval_harness` wrapper | `internal/executor/` factory |
| **Provider Support** | Claude only | Claude + Gemini (extensible) |
| **Session Resume** | No | Yes (`--resume` / `--conversation-id`) |

### Message Handling

| Aspect | ailang-agent | coordinator |
|--------|-------------|-------------|
| **Polling** | `messaging.Client` | `InboxMessageAdapter` |
| **Message Types** | directive, question | directive, question, research |
| **Claiming** | DB-based atomic claim | Same + GitHub sync |
| **Multi-inbox** | Single instance | Config-based agent registry |

### Isolation & Workspace

| Aspect | ailang-agent | coordinator |
|--------|-------------|-------------|
| **Isolation** | Temp directories | Git worktrees |
| **Cleanup** | Manual (`CLEANUP_AGENT_WORKSPACES=1`) | Automatic with retention policy |
| **Context Sharing** | None | `.claude/` settings carried over |

### Approval Workflow

| Aspect | ailang-agent | coordinator |
|--------|-------------|-------------|
| **Approval Source** | Database polling | GitHub PR labels |
| **Timeout** | 24 hours | Configurable |
| **Visibility** | Collaboration Hub only | GitHub + Collaboration Hub |
| **Multi-stage** | No | Yes (design → sprint → implement → merge) |

### Advanced Features (Coordinator Only)

- **Agent Registry**: YAML-configured agents with capabilities
- **Task Chaining**: Automatic handoffs between agents
- **Stage Execution**: Multi-phase workflows
- **HTTP Broadcasting**: Real-time dashboard streaming
- **Resource Tracking**: Memory/process monitoring
- **Task Deduplication**: SimHash-based duplicate detection
- **Priority Calculation**: Keyword-based priority scoring

## Feature Gap Analysis

### Features in ailang-agent NOT in coordinator

| Feature | ailang-agent Location | Recommendation |
|---------|----------------------|----------------|
| **Capability Detection** (FS/Net/Shell/Budget) | `internal/agent/capabilities.go` | **MIGRATE** to coordinator |
| **Impact Classification** (low/medium/high) | `internal/agent/capabilities.go` | **MIGRATE** to coordinator |
| **Pre-execution Cost Estimation** | `internal/agent/capabilities.go` | **MIGRATE** to coordinator |
| **Result Formatting** (markdown) | `internal/agent/formatter.go` | Already have `templates.go` - **SKIP** |

### Migration Plan for Capability Detection

The coordinator's `TaskAnalyzer` classifies task TYPE (bug-fix, feature, refactor) but doesn't detect CAPABILITIES (FS, Net, Shell). This is useful for:

1. **Risk Assessment**: Warn users before shell execution
2. **Cost Budgeting**: Estimate cost before expensive operations
3. **Approval Routing**: High-risk → human approval, low-risk → auto-approve

**Recommendation**: Create `internal/coordinator/capability_detector.go` by adapting `internal/agent/capabilities.go`:

```go
// capability_detector.go - Adapted from internal/agent/capabilities.go

type CapabilityDetector struct{}

func (cd *CapabilityDetector) DetectCapabilities(content string) []Capability {
    // Detect FS, Net, Shell, Budget capabilities
}

func (cd *CapabilityDetector) ClassifyImpact(caps []Capability) string {
    // Return "low", "medium", or "high"
}

func (cd *CapabilityDetector) EstimateCost(content string) float64 {
    // Pre-execution cost estimation
}
```

**Integration point**: Call from `TaskAnalyzer.Analyze()` to augment `AnalyzedTask` with capability info.

## Implementation Plan

### Phase 1: Migrate Capability Detection (Day 1)

1. **Create** `internal/coordinator/capability_detector.go`
   - Adapt logic from `internal/agent/capabilities.go`
   - Add to `AnalyzedTask` struct:
     ```go
     type AnalyzedTask struct {
         // ... existing fields ...
         Capabilities    []Capability `json:"capabilities"`
         ImpactLevel     string       `json:"impact_level"` // low, medium, high
         EstimatedCost   float64      `json:"estimated_cost"`
     }
     ```

2. **Integrate** capability detection into task analysis
   - Call from `TaskAnalyzer.Analyze()`
   - Use in approval routing (high-risk → require approval)

3. **Update** GitHub comment templates
   - Show detected capabilities in comments
   - Display impact level badges

### Phase 2: Remove Code (Day 1-2)

1. **Delete source code**:
   - `cmd/ailang-agent/` directory (3 files)
   - `internal/agent/` directory (6 files)
   - `cmd/ailang/agent.go.bak` backup file

2. **Remove Makefile targets**:
   ```makefile
   # REMOVE these targets from Makefile:
   # - build-agent (line 43-48)
   # - install-agent (line 50-54)
   # - Update build-all to not reference ailang-agent
   # - Update help text (lines 568-571)
   ```

### Phase 3: Update Documentation (Day 2)

#### Files to Update

1. **`docs/docs/guides/collaboration-hub.md`** - Major updates needed:
   - Line 16-27: Remove `make build-agent`, `make install-agent` references
   - Line 32-34: Remove `ailang-agent --version` verification
   - Line 208-214: Remove ailang-agent CLI usage examples
   - Replace with `ailang coordinator` equivalents

2. **`CHANGELOG.md`** - Add deprecation notice:
   - Add to v0.6.3 section: "Deprecated: `ailang-agent` binary removed"
   - Reference this design doc
   - Note: Historical entries (v0.4.5) remain for record

3. **`Makefile`** - Remove targets:
   - `build-agent` target
   - `install-agent` target
   - Update `build-all` target
   - Update `help` target text

4. **External: `ailang_bootstrap/skills/ailang/resources/cli_reference.md`**
   - Currently does NOT mention ailang-agent (no changes needed)
   - Verify after removal that no skills reference it

#### Documentation Changes Detail

**collaboration-hub.md before:**
```markdown
# Build ailang-agent only
make build-agent

# Or build all binaries (ailang + ailang-agent)
make build-all

ailang-agent --version
# ailang-agent version 0.4.5-dev
```

**collaboration-hub.md after:**
```markdown
# Build ailang
make build

# Start the coordinator daemon
ailang coordinator start

# Check status
ailang coordinator status
```

### Phase 4: Archive Design Docs

Move to archive (already done for some):
- `design_docs/archive/v0_5_1_m-background-agent-daemon.md` ✓ (already archived)
- `design_docs/archive/v0_4_6_agent-execution-enhancements.md` ✓ (already archived)
- Keep `design_docs/implemented/v0_4_5/agent-execution-integration.md` for historical reference

## Files to Delete

| Path | Lines | Purpose |
|------|-------|---------|
| `cmd/ailang-agent/main.go` | 69 | CLI entry point |
| `cmd/ailang-agent/agent.go` | 359 | Agent implementation |
| `cmd/ailang-agent/agent_test.go` | ~100 | Tests |
| `internal/agent/capabilities.go` | 266 | Capability detection (MIGRATE FIRST) |
| `internal/agent/capabilities_test.go` | ~200 | Tests |
| `internal/agent/executor.go` | 311 | Directive executor |
| `internal/agent/executor_test.go` | ~150 | Tests |
| `internal/agent/formatter.go` | 155 | Result formatting |
| `internal/agent/formatter_test.go` | ~100 | Tests |
| `cmd/ailang/agent.go.bak` | ~50 | Backup file |

**Total: ~1,760 lines to remove**

## Files to Modify

| Path | Changes |
|------|---------|
| `Makefile` | Remove build-agent, install-agent targets |
| `docs/docs/guides/collaboration-hub.md` | Replace ailang-agent with coordinator |
| `CHANGELOG.md` | Add deprecation notice |

## Testing

### Before Deprecation

1. **Verify capability detection works in coordinator**:
   ```bash
   # Create task with shell keywords
   ailang messages send coordinator "run npm install in the project" --from test

   # Verify impact level detected
   ailang coordinator status  # Should show high-risk task
   ```

2. **Verify all ailang-agent features have coordinator equivalents**:
   - [x] Message polling
   - [x] Message claiming
   - [x] Directive execution
   - [x] Question handling
   - [ ] Capability detection (migrate first)
   - [ ] Impact classification (migrate first)
   - [ ] Cost estimation (migrate first)

### After Deprecation

1. **Run full coordinator test suite**:
   ```bash
   go test ./internal/coordinator/... -v
   ```

2. **Verify no imports of internal/agent remain**:
   ```bash
   grep -r "internal/agent" --include="*.go" | grep -v "_test.go"
   # Should return nothing
   ```

3. **Verify build succeeds**:
   ```bash
   make build
   make test
   ```

4. **Verify docs build**:
   ```bash
   cd docs && npm run build
   ```

## Success Criteria

- [ ] Capability detection migrated to coordinator
- [ ] Impact classification migrated to coordinator
- [ ] Cost estimation migrated to coordinator
- [ ] `cmd/ailang-agent/` directory deleted
- [ ] `internal/agent/` package deleted
- [ ] `cmd/ailang/agent.go.bak` deleted
- [ ] Makefile targets removed
- [ ] `collaboration-hub.md` updated
- [ ] `CHANGELOG.md` updated
- [ ] All tests pass
- [ ] Docs build successfully
- [ ] No regressions in coordinator functionality

## Rollback Plan

If issues arise:

1. Code is in git history - can restore from:
   - `cmd/ailang-agent/` from commit before deletion
   - `internal/agent/` from commit before deletion

2. Capability detection can be disabled without removing coordinator

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Users depending on ailang-agent | Deprecation warning + documentation |
| Missing feature in migration | Thorough feature comparison (this doc) |
| Coordinator bugs exposed | Keep code in git history for rollback |
| External tools referencing ailang-agent | Search external repos, update if needed |

## References

- Original agent design: `design_docs/archive/v0_5_1_m-background-agent-daemon.md`
- Agent execution integration: `design_docs/implemented/v0_4_5/agent-execution-integration.md`
- Coordinator design: `design_docs/implemented/v0_6_1/m-exec-multi-executor-support.md`
- Agent protocol: `design_docs/implemented/v0_5_0/M-AGENT-PROTOCOL.md`
