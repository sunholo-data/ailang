# Agent Client Protocol (ACP) Evaluation for AILANG

**Feature Name:** ACP Integration Assessment
**Status:** Rejected (re-confirmed 2026-05-23)
**Created:** 2026-02-03
**Updated:** 2026-05-23 — added May 2026 reassessment
**Target:** N/A (not adopted)
**Priority:** P2 (Low - investigative only)
**Estimated:** 0 days (evaluation only)
**Dependencies:** None
**Related (downstream):** [motoko_agent ACP integration](https://github.com/arniwesth/motoko_agent/blob/main/design_docs/planned/m-motoko-acp-integration.md) — the case is much stronger one layer up, where motoko_agent is an agent harness rather than a language.

## Problem Statement

AILANG currently has a custom architecture for integrating AI coding agents:
- **Executor interface** (`internal/executor/`) for agentic coding (Claude Code CLI, Gemini CLI)
- **Provider interface** (`internal/ai/`) for text generation (OpenAI, Anthropic, Gemini APIs)
- **Factory pattern** with static registration via `init()` functions
- **Custom JSONL parsing** for each CLI tool's output format

The Agent Client Protocol (ACP) claims to standardize agent-client communication, similar to how LSP standardized language servers. GitHub issue #129 asks whether AILANG should adopt ACP.

**Key questions:**
1. Would ACP simplify adding new agents to AILANG?
2. Does ACP's protocol align with AILANG's architecture?
3. What would migration cost vs. benefit?

## Goals

**Primary Goal:** Evaluate whether adopting ACP would benefit AILANG's agent integration architecture

**Success Metrics:**
- [ ] Clear recommendation on ACP adoption (YES/NO with rationale)
- [ ] Identified gaps between ACP and AILANG requirements
- [ ] Cost-benefit analysis of migration
- [ ] Alternative improvements identified if not adopting ACP

## Research Findings

### What is ACP?

The **Agent Client Protocol** is a standardized protocol for AI agent communication:
- **Protocol:** JSON-RPC over stdio (local) or HTTP/WebSocket (remote)
- **Purpose:** Enable any ACP-compatible agent to work with any ACP-compatible client
- **Registry:** Curated collection at `https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json`
- **Status:** Under active development (format may change)

### ACP's Agent Registry (as of February 2026)

Currently lists 10 agents including:
- **Claude Code** v0.13.2 (AILANG already integrates)
- **Gemini CLI** v0.26.0 (AILANG already integrates)
- **Codex CLI** v0.9.0 (AILANG has placeholder for)
- **GitHub Copilot** v1.418.0 (new opportunity)
- Others: Auggie CLI, Factory Droid, Mistral Vibe, OpenCode, Qwen Code

### ACP vs AILANG Current Architecture

| Aspect | AILANG Current | ACP Approach | Gap Analysis |
|--------|---------------|--------------|--------------|
| **Protocol** | Custom CLI invocation + JSONL parsing | JSON-RPC over stdio/HTTP/WebSocket | Would require protocol adapter layer |
| **Discovery** | Hardcoded in `init()` functions | Dynamic registry fetch | More flexible but adds network dependency |
| **Authentication** | Per-provider API keys/CLI auth | Standardized auth requirement | Need to map auth models |
| **Tool Calling** | Provider-specific (Bash, Read, Write, etc.) | Unknown (not documented yet) | Critical gap - need tool standardization |
| **Streaming** | Custom EventHandler interface | Unknown protocol details | Need to verify streaming support |
| **Session Continuity** | `--resume` flags, SessionID tracking | Unknown | Critical for coordinator handoffs |
| **Cost Tracking** | TokenUsage, CostModel interfaces | Unknown | Need cost visibility |

### AILANG's Unique Requirements

AILANG has specific needs beyond simple text generation:

1. **Coordinator Integration**
   - Task hierarchy tracking (`parent_task_id`)
   - Approval workflows
   - Agent-to-agent handoffs with session continuity
   - Git worktree management

2. **Telemetry & Observability**
   - OTEL span injection
   - Task/trace correlation
   - Event streaming to dashboard
   - Timestamp-based span re-parenting

3. **Hybrid Execution**
   - Script agents (deterministic, $0 cost)
   - AI agents (Claude, Gemini)
   - Mixed pipelines (AI → Script → AI)

4. **AILANG-Specific Context**
   - Meta-prompts with AILANG syntax
   - Project conventions (CLAUDE.md)
   - Design axioms compliance

## Analysis

### Benefits of Adopting ACP

1. **Standardization**
   - Consistent interface across all agents
   - No custom JSONL parsers per agent
   - Community-maintained protocol

2. **Broader Agent Support**
   - Access to GitHub Copilot, Mistral, Qwen, etc.
   - Future agents automatically compatible
   - Reduced integration effort per agent

3. **Dynamic Discovery**
   - Registry-based agent discovery
   - Version management
   - Automatic updates to agent capabilities

### Challenges with ACP Adoption

1. **Protocol Mismatch**
   - AILANG uses direct CLI execution
   - ACP uses JSON-RPC (requires protocol bridge)
   - Existing streaming/event handlers would need rewrite

2. **Missing Critical Features**
   - No documented tool calling standard
   - Session continuity model unclear
   - Cost tracking not specified
   - Telemetry integration unknown

3. **Architectural Impact**
   - Would need adapter layer between Executor interface and ACP
   - Factory pattern would need overhaul for dynamic registration
   - Coordinator assumptions about executors would break

4. **Maturity Concerns**
   - ACP is "under active development"
   - Protocol format may change
   - Limited documentation available

### Cost-Benefit Analysis

**Migration Cost (High):**
- Rewrite executor abstraction (~2,000 LOC)
- Create ACP protocol adapter (~1,000 LOC)
- Update coordinator integration (~500 LOC)
- Rewrite streaming/event handling (~500 LOC)
- Update all tests (~1,000 LOC)
- **Total: ~5,000 LOC, 2-3 weeks**

**Benefit (Low-Medium):**
- AILANG already supports main agents (Claude, Gemini)
- Other agents in registry not critical for AILANG use cases
- Protocol standardization nice but not blocking
- Dynamic discovery adds complexity without clear value

## Recommendation

### **DO NOT ADOPT ACP (at this time)**

**Rationale:**

1. **Insufficient Value**: AILANG already integrates the two most important agents (Claude Code, Gemini CLI). Adding GitHub Copilot might be valuable, but not worth a major architectural change.

2. **Feature Gaps**: ACP lacks critical features AILANG needs:
   - Session continuity for agent handoffs
   - Tool calling standardization
   - Cost tracking
   - Telemetry integration

3. **Architectural Mismatch**: AILANG's direct CLI execution model is simpler and more reliable than JSON-RPC. Adding a protocol layer increases complexity without clear benefits.

4. **Immaturity**: ACP is still under development. Adopting now risks breaking changes and missing features.

5. **Unique Requirements**: AILANG's coordinator, approval workflows, and AILANG-specific context aren't addressed by ACP.

### Alternative Improvements (Recommended)

Instead of adopting ACP, enhance AILANG's current architecture:

1. **Modularize JSONL Parsers**
   ```go
   // internal/executor/parsers/
   type StreamParser interface {
       ParseLine([]byte) (*StreamEvent, error)
   }

   // Reusable parsers for common formats
   var Parsers = map[string]StreamParser{
       "claude": &ClaudeParser{},
       "gemini": &GeminiParser{},
       "openai": &OpenAIParser{}, // If Codex uses OpenAI format
   }
   ```

2. **Improve Factory Registration**
   ```go
   // Config-driven registration instead of init()
   executors:
     claude:
       binary: claude
       parser: claude
       capabilities: [streaming, session_resume]
     copilot:
       binary: copilot
       parser: openai  # Reuse parser
       capabilities: [streaming]
   ```

3. **Add GitHub Copilot Directly**
   - Check if Copilot CLI has similar JSONL format to other tools
   - Implement `internal/executor/copilot/` if valuable
   - Reuse existing patterns, no protocol change needed

4. **Monitor ACP Evolution**
   - Revisit in 6 months when protocol stabilizes
   - Contribute AILANG requirements to ACP spec
   - Consider adapter-based integration if ACP matures

## Implementation Plan

Since we're **not adopting ACP**, no implementation needed. However, if reconsidered:

### Phase 0: Monitor & Contribute (Ongoing)
- [ ] Watch ACP development
- [ ] Submit issues for missing features
- [ ] Evaluate when v1.0 releases

### Future Phase 1: Adapter Prototype (If Revisited)
- [ ] Create `internal/acp/` adapter package
- [ ] Implement JSON-RPC client
- [ ] Bridge to Executor interface
- [ ] Test with one agent

### Future Phase 2: Migration (If Proven Valuable)
- [ ] Migrate all executors
- [ ] Update coordinator
- [ ] Deprecate old system

## Success Criteria

For this evaluation:
- [x] Analyzed ACP protocol and registry
- [x] Compared with AILANG architecture
- [x] Identified critical gaps
- [x] Made clear recommendation
- [x] Proposed alternative improvements

## Related Documents

- [internal/executor/](../../internal/executor/) - Current executor implementation
- [internal/ai/](../../internal/ai/) - Provider abstraction
- [internal/coordinator/](../../internal/coordinator/) - Task orchestration
- [GitHub Issue #129](https://github.com/sunholo-data/ailang/issues/129) - Original request

## Timeline

**Evaluation Complete:** 2026-02-03

No implementation timeline as recommendation is not to adopt.

## Decision Record

**Decision:** Do not adopt ACP at this time

**Reasons:**
1. Insufficient value for migration cost
2. Critical features missing
3. Protocol immaturity
4. Architectural mismatch

**Alternative:** Enhance current architecture with modular parsers and config-driven registration

**Review Date:** 2026-08-01 (6 months) to reassess if ACP has matured

---

## 2026-05-23 Reassessment (early review)

Triggered by a question about whether AILANG could standardize its *agent monitoring* layer on ACP. Re-confirming rejection, but updating context.

### What changed since Feb 2026

| Signal | Then | Now (2026-05-23) | Direction |
|---|---|---|---|
| Wire-protocol version | "Under active development" | **Stable v1** | ✅ matured |
| Crate release | — | v0.13.3 (released 2026-05-22) | active dev |
| Agents in registry | ~10 | **25+** (incl. GitHub Copilot CLI, Codex, Qwen, OpenCode, Auggie, Factory Droid, Mistral Vibe) | ✅ broader |
| Editor support | Zed only | **Zed + JetBrains + marimo + Eclipse prototype + Toad** | ✅ broader |
| ACP Registry | Not yet | Live since Jan 2026 (https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json) | ✅ usable |
| Repository activity | New | 3.2k stars, 1,419 commits, 45 releases | active |

The three weakest pillars of the Feb 2026 rejection — *insufficient value*, *protocol immaturity*, *feature gaps* — have all weakened.

### What still holds — the monitoring distinction

The trigger question was: *"can we standardize agent monitoring on ACP?"* The answer is **no, and the framing is wrong**:

ACP is an **editor ↔ agent integration protocol** (two-party, JSON-RPC over stdio for local, HTTP/WebSocket WIP for remote). It defines:

- Session lifecycle: `session/list`, `session/resume`, `session/close`, `session/prompt`, `session/cancel`
- Streaming notifications during a turn (`session/update`):
  - `plan`, `agent_message_chunk`, `tool_call`, `tool_call_update`
- Tool-use approval: `session/request_permission`
- Stop reasons: `end_turn`, `max_tokens`, `max_turn_requests`, `refusal`, `cancelled`

It does **NOT** define:

- A third-party observer role (no way for an external monitor to attach to a running session — `ailang chains live` couldn't be implemented as an ACP observer)
- Token-usage telemetry in the wire format
- Cost tracking
- Cross-session correlation / task hierarchy / `parent_task_id`
- Stable remote transport (HTTP/WebSocket explicitly marked WIP)

AILANG's monitoring stack — OTEL spans → otelcol → `observatory.db` consumed by `ailang chains live` — is already on the right standard for monitoring. OTEL is the vendor-neutral telemetry protocol with native support for spans, traces, attributes, and observer fan-out. OpenCode (which we use in the local-Ollama eval harness) happens to also be an ACP agent, but we're consuming its **OTEL spans** (the right layer for monitoring), not its **ACP messages** (the right layer for *driving* it interactively).

**Rolling AILANG monitoring on ACP would be a downgrade**, not a consolidation: we'd lose the third-party observer role, lose token/cost telemetry primitives, lose the OTEL span correlation that already powers chains/dashboard, and gain nothing — because ACP doesn't address this layer.

### What's the right framing then

Two separate concerns, which the Feb 2026 doc already implicitly conflated:

1. **Monitoring concern** (the trigger for this reassessment): **stay on OTEL.** ACP is wrong layer.
2. **Executor integration concern** (what the Feb 2026 doc actually evaluated): the case has strengthened, but the ~5,000 LOC migration cost and architectural-mismatch concerns from Feb 2026 still hold. The right move at the 2026-08-01 review is to scope a *minimal* single-agent ACP adapter (likely OpenCode, since it's already in our path) parallel to the existing JSONL path — not a full executor rewrite. Validate the value at small cost before committing.

### Where ACP fits naturally — not here, one layer up

The trigger for this reassessment surfaced a key insight: **motoko_agent** (the agent harness built on AILANG, in [arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent)) is a much more natural home for ACP than AILANG itself. Motoko IS an agent — making it an ACP-compatible agent would let it plug into Zed, JetBrains, marimo, and the rest of the ACP ecosystem for free. AILANG is a *language* used to build agents; the protocol that connects editors to agents is the wrong abstraction layer for it.

A planned design doc lives in motoko's repo: see `arniwesth/motoko_agent/design_docs/planned/m-motoko-acp-integration.md`.

### Re-confirmed decision

**Still rejected for AILANG.** The Aug 2026 review point remains for re-examining the executor-integration concern (a separate question from monitoring). If that review concludes positively, the minimal-adapter approach is the recommended first move — not a full migration.