# M-COORD-THINKING: Extended Thinking Levels for Coordinator Skills

**Status:** Planned
**Target:** v0.6.4
**Priority:** P1 (Medium)
**Estimated:** 2-3 days
**Dependencies:** None (coordinator already exists)

---

## Problem Statement

Currently, AILANG coordinator agents execute with a fixed reasoning approach. Some tasks benefit from lightweight processing (quick decisions, simple edits), while others require deeper analysis and extended thinking (complex bug fixes, architecture design, systemic analysis).

**Current limitations:**
- All agents use the same model reasoning depth
- No way to request extended thinking for complex tasks
- Message types don't control reasoning behavior
- Message labels could provide better task routing but aren't used for thinking level selection

**Impact:**
- Complex tasks may rush through analysis, missing systemic issues
- Simple tasks waste tokens on unnecessary deep thinking
- Cannot prioritize expensive reasoning for critical work
- Message metadata isn't fully utilized

---

## Goals

**Primary Goal:** Add configurable thinking levels to coordinator skills, allowing fine-grained control over reasoning depth based on task complexity and message metadata.

**Success Metrics:**
1. Agents can be configured with `thinking_level` in `~/.ailang/config.yaml`
2. Message types (bug, feature, research, docs) can specify override thinking levels
3. Message labels can force specific thinking levels
4. All 4 thinking levels work: `off`, `think`, `think_hard`, `ultrathink`
5. Usage tracking shows which thinking level is used per task
6. Token consumption metrics correlate with thinking level

---

## Solution Design

### 1. Overview

The solution adds three layers of thinking level configuration:

1. **Agent-Level Default**: Each agent (design-doc-creator, sprint-executor, etc.) has a default thinking level
2. **Message Type Overrides**: Message types (bug, feature, research) can override agent defaults
3. **Label Overrides**: Message labels like `thinking:ultrathink` force specific levels

**Priority order (highest to lowest):**
```
Label override (thinking:*) > Message type override > Agent default > Global default
```

### 2. Architecture

#### 2.1 Configuration Structure

**In `~/.ailang/config.yaml`:**

```yaml
coordinator:
  # Global default for all agents
  default_thinking: think

  agents:
    # Design doc creator - uses research for systematic analysis
    - id: design-doc-creator
      thinking_level: think_hard
      thinking_overrides:
        research: ultrathink
        bug: think
        feature: think_hard
        docs: think
      label_overrides:
        - label: complexity:high
          thinking_level: ultrathink
        - label: complexity:low
          thinking_level: off

    # Sprint executor - more conservative with thinking
    - id: sprint-executor
      thinking_level: think
      thinking_overrides:
        bug: think_hard          # Bugs need deep analysis
        feature: think           # Features can use standard thinking
        research: think          # Don't waste tokens on research
        docs: off                # Docs don't need extended thinking

    # Sprint planner - requires analysis
    - id: sprint-planner
      thinking_level: think_hard
      thinking_overrides:
        research: think_hard
        bug: think_hard
        feature: think_hard
        docs: off

    # General coordinator (catch-all)
    - id: coordinator
      thinking_level: think
      thinking_overrides:
        bug: think_hard
        feature: think
        research: ultrathink
        docs: off
```

#### 2.2 Message Type Structure

Extend message schema to support thinking overrides:

```go
// internal/messaging/message.go
type Message struct {
    ID            string
    Inbox         string
    Title         string
    Content       string
    Type          MessageType // bug, feature, research, docs, custom
    Status        string      // unread, read, archived, deleted
    ThinkingLevel *string     // optional override: off, think, think_hard, ultrathink
    Labels        []string    // custom labels (e.g., "complexity:high", "thinking:ultrathink")
    // ... existing fields
}

type MessageType string

const (
    TypeBug      MessageType = "bug"
    TypeFeature  MessageType = "feature"
    TypeResearch MessageType = "research"
    TypeDocs     MessageType = "docs"
    TypeCustom   MessageType = "custom"
)

// ThinkingLevel represents extended thinking configuration
type ThinkingLevel string

const (
    ThinkingOff       ThinkingLevel = "off"
    ThinkingStandard  ThinkingLevel = "think"
    ThinkingHard      ThinkingLevel = "think_hard"
    ThinkingUltra     ThinkingLevel = "ultrathink"
)
```

#### 2.3 Agent Configuration Update

Extend agent registry to support thinking levels:

```go
// internal/coordinator/agent_registry.go
type AgentConfig struct {
    ID                  string
    Inbox               string
    ThinkingLevel       ThinkingLevel // default for this agent
    ThinkingOverrides   map[MessageType]ThinkingLevel
    LabelOverrides      []LabelOverride // regex patterns to override thinking
    // ... existing fields
}

type LabelOverride struct {
    Label         string        // e.g., "complexity:high" or "urgent:.*"
    ThinkingLevel ThinkingLevel
}
```

#### 2.4 Thinking Level Resolution

Create resolver to determine final thinking level:

```go
// internal/coordinator/thinking_resolver.go
package coordinator

func ResolvethinkingLevel(
    msg *messaging.Message,
    agent *AgentConfig,
    globalDefault ThinkingLevel,
) ThinkingLevel {
    // Priority: Label > MessageThinkingLevel > MessageType > Agent > Global

    // 1. Check label overrides (highest priority)
    for _, label := range msg.Labels {
        if matches := regex(label); matches {
            for _, override := range agent.LabelOverrides {
                if matches.MatchString(override.Label) {
                    return override.ThinkingLevel
                }
            }
        }
    }

    // 2. Check explicit message thinking level
    if msg.ThinkingLevel != nil {
        return *msg.ThinkingLevel
    }

    // 3. Check message type overrides
    if override, ok := agent.ThinkingOverrides[msg.Type]; ok {
        return override
    }

    // 4. Use agent default
    if agent.ThinkingLevel != "" {
        return agent.ThinkingLevel
    }

    // 5. Fall back to global default
    return globalDefault
}
```

### 3. Implementation Plan

**Phase 1: Core Infrastructure** (~1 day)
- [ ] Extend Message struct with `ThinkingLevel` field
- [ ] Add MessageType enum and constants
- [ ] Create ThinkingLevel enum and resolver
- [ ] Update agent configuration schema
- [ ] Add database migration for message schema
- [ ] Write unit tests for resolver logic

**Phase 2: CLI & Configuration** (~1 day)
- [ ] Update `ailang messages send` to accept `--thinking` flag
- [ ] Add `--type` flag support (bug, feature, research, docs)
- [ ] Update config.yaml parsing for thinking_overrides
- [ ] Add validation for thinking level values
- [ ] Test with all 4 thinking levels
- [ ] Update documentation and examples

**Phase 3: Agent Integration** (~0.5 days)
- [ ] Update executor factory to pass thinking level to Claude CLI
- [ ] Update executor factory for Gemini CLI (if supported)
- [ ] Ensure thinking level is logged in task execution
- [ ] Add metrics for thinking level usage
- [ ] Write integration tests

**Phase 4: Dashboard & Monitoring** (~0.5 days)
- [ ] Display thinking level in task history
- [ ] Show usage breakdown by thinking level
- [ ] Add token consumption estimates per level
- [ ] Create dashboard card for thinking metrics

### 4. Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/messaging/message.go` | Add ThinkingLevel field, MessageType enum | +40 |
| `internal/coordinator/agent_registry.go` | Extend AgentConfig with thinking config | +30 |
| `internal/coordinator/thinking_resolver.go` | NEW: Resolution logic | +80 |
| `internal/coordinator/store_sqlite.go` | Add thinking_level column to messages table | +15 |
| `internal/executor/executor.go` | Pass thinking level to CLI tools | +20 |
| `cmd/ailang/messages.go` | Add --thinking and --type flags | +35 |
| `~/.ailang/config.yaml` | Add thinking configuration example | +25 |
| Tests (thinking_resolver_test.go, etc.) | Full test coverage | +200 |
| **Total** | | **~445** |

### 5. Examples

#### Example 1: Bug Fix Requiring Extended Analysis

```bash
# Send bug with ultrathink to catch systemic issues
ailang messages send coordinator \
  "Multiple slice type conversions fail at codegen" \
  --title "Bug: Slice conversion panics" \
  --type bug \
  --label "complexity:high,systemic:true"

# Agent config for coordinator with thinking overrides:
# - Bug messages use think_hard by default
# - complexity:high labels force ultrathink
# Agent will use ultrathink for this message
```

Result: Extended thinking analyzes all slice types, finds unified solution (v0.6.4 pattern).

#### Example 2: Simple Documentation Update

```bash
# Send docs task with thinking disabled
ailang messages send coordinator \
  "Update README examples for v0.6.3" \
  --title "Docs: Update README" \
  --type docs \
  --thinking off

# Agent config:
# - Docs type overrides to thinking: off
# - Explicit --thinking flag also sets off
# Agent uses no extended thinking (saves tokens)
```

Result: Quick edit without wasted computation.

#### Example 3: Complex Feature Design

```bash
# Send feature with research thinking
ailang messages send design-doc-creator \
  "Semantic caching for AI providers" \
  --title "Feature: Semantic Caching" \
  --type feature \
  --label "complexity:very-high"

# Agent config for design-doc-creator:
# - complexity:very-high label forces ultrathink
# Agent uses ultrathink for systematic analysis
```

Result: Thorough design doc considering all implications.

### 6. Success Criteria

- [ ] All 4 thinking levels configurable
- [ ] Message type overrides work correctly
- [ ] Label overrides function properly
- [ ] Resolver prioritization is correct
- [ ] CLI flags `--thinking` and `--type` work
- [ ] Config validation rejects invalid values
- [ ] Database migration adds thinking_level column
- [ ] Thinking level passed to executor tools
- [ ] Metrics show usage by level
- [ ] Dashboard displays thinking metadata
- [ ] Documentation covers all 4 levels and config
- [ ] Integration tests verify full workflow
- [ ] Token consumption correlates with thinking level

### 7. Testing Strategy

**Unit Tests:**
- Resolver logic with all priority combinations
- Config parsing and validation
- Message type enum handling
- Label pattern matching

**Integration Tests:**
- Send message with each thinking level
- Verify executor receives correct level
- Test all override combinations
- Verify CLI flags work correctly

**Dashboard Tests:**
- Metrics display correctly
- Usage breakdown accurate
- Historical tracking works

---

## Timeline

**Week 1:**
- Day 1-2: Core infrastructure (Phase 1)
- Day 3-4: CLI & configuration (Phase 2)

**Week 2:**
- Day 5: Agent integration (Phase 3)
- Day 6-7: Dashboard & monitoring (Phase 4)

**Week 3:**
- Final testing and documentation
- User feedback integration

---

## Related Documents

- [M-COORD-TASKCHAIN-TESTS](m-coord-taskchain-tests.md) - Related coordinator testing
- [M-ARCH4: Executor Stream Processor](m-arch4-executor-stream-processor.md) - Executor architecture
- [M-COORD-MESSAGING: Message System v0.6.2](../v0_6_2/m-coord-messaging.md) - Message infrastructure
- [Design: Coordinator Daemon](../../../docs/docs/guides/coordinator.md) - Current coordinator guide

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Configuration-driven behavior is deterministic |
| A3: Effect Legibility | +1 | Thinking level is explicit in task metadata |
| A7: Machines First | +1 | Enables AI agents to request appropriate reasoning |
| A8: Minimal Syntax | 0 | Reuses existing config system |
| A9: Cost Visibility | +1 | Thinking level directly correlates to token cost |
| A10: Composability | +1 | Overrides compose cleanly (label > type > agent) |
| **Net Score** | **+5** | ✅ Approved |

---

## Risk Analysis

**Risk:** Thinking levels disabled but tokens still consumed
**Mitigation:** Add validation that thinking level "off" actually disables extended thinking in executors

**Risk:** Label patterns too complex for users
**Mitigation:** Provide pre-built label templates and documentation

**Risk:** Token costs explode with ultrathink
**Mitigation:** Add warnings for ultrathink usage, track metrics per task

**Risk:** Config changes don't take effect immediately
**Mitigation:** Coordinator reload config on each task pickup

---

## Future Enhancements

1. **Per-Skill Thinking Defaults** - Different defaults for each skill
2. **Budget-Based Thinking** - Allocate thinking tokens per sprint/week
3. **Adaptive Thinking** - Automatically increase thinking level if task fails
4. **Thinking Profiles** - Pre-built thinking configurations for common workflows
5. **Cost Projection** - Estimate token cost based on thinking level before execution

---

## Notes

This feature integrates with the existing message system and coordinator infrastructure. No breaking changes to the core architecture.

The resolver can later be extended to support custom thinking levels or continuous values (0.0-1.0) if needed.

Message labels provide extensibility for future features beyond thinking levels.
