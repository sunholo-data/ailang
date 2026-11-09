# Sprint Plan: M-DX-SESSION-GATE — mechanical session-protocol gate for pi

## Summary
Ship `.pi/extensions/session-protocol-gate.ts`: pi sessions in this repo cannot execute `edit`/`write` (and fail-closed bash) until `session_protocol_ack` succeeds — with verifiable prerequisites — mechanically backing the AGENTS.md work-routing gate.

**Design doc:** [m-dx-session-protocol-gate.md](m-dx-session-protocol-gate.md) (ratified 2026-08-28; quorum 2 rounds, Fail-open register F1–F8 signed)
**Duration:** 1 session (~3h)
**Dependencies:** None (complements AGENTS.md gate already on dev)
**Risk Level:** Medium-low — new integration surface (pi extension API), but every platform claim is doc-verified (V1–V12)

## Current Status Analysis

### Completed Recently
- ✅ AGENTS.md advisory gate live on `origin/dev` (`007184b7b`) — this sprint adds the mechanical layer
- ✅ All pi API premises verified against v0.84.3 docs (tool_call block contract, State Management reconstruction, ui.confirm, trust gating)

### Velocity
- Extension scope ≈ 210 LOC total (extension ~120 + predicate tests ~60 + README ~30)
- Single-session comparable: fmt-hook plumbing landed same-day

### Remaining from Design Doc
- ⏳ Extension core (state + ack tool + interceptor)
- ⏳ Persistence via documented reconstruction pattern; fail-closed on ambiguity
- ⏳ Validation matrix + operational README

## Proposed Milestones

### M1: Extension core (~1.5h)
**Estimated:** ~120 LOC

**Tasks:**
- `.pi/extensions/session-protocol-gate.ts`: session_start arming (V4), `session_protocol_ack` tool with prerequisite enforcement (V11 ui.confirm in TUI; V10 getBranch scan for CLAUDE.md-touch + `ailang messages` bash call in headless), `tool_call` interceptor with quoted block contract (V1)
- Pure predicate `shouldBlock(toolName, bashCommand, acked)` kept unit-testable

**Acceptance criteria:**
- [ ] Fresh interactive session: `edit`/`write` blocked with the standard reason; bash non-allowlisted commands blocked
- [ ] Ack refused while prerequisites unmet (lists missing steps)

### M2: Persistence + edge cases (~0.5h)
**Estimated:** ~30 LOC

**Tasks:**
- Ack reconstruction: `session_start` scans `getBranch()` for ack tool-result `details: { acked: true }` (V10 canonical pattern); **fail-closed** (re-arm) on ambiguity
- `/reload` and `/resume` re-derivation; read-only tools never blocked

**Acceptance criteria:**
- [ ] Resumed session with prior ack: not blocked
- [ ] Read tools (`read`, grep, `ailang messages list`) never blocked

### M3: Validation + docs (~1h)
**Estimated:** ~60 test LOC + 30 LOC README

**Tasks:**
- Table-driven tests for `shouldBlock` (tool × bash command × acked)
- Phase-3 validation matrix from design doc: fresh-session block → ack → unblocked; `pi -p` identical; resumed-not-blocked; F5 coordinator-session inheritance verified
- `.pi/extensions/README.md`: what the gate does, disarm steps, F1 (`--no-extensions`) documented as forbidden in this repo's workflow
- CHANGELOG entry under v0.35.0 (DX section)

**Acceptance criteria:**
- [ ] All design-doc Success Criteria boxes pass
- [ ] `make fmt` hook unaffected; web-search package extension unaffected
- [ ] README covers disarm + escape hatch

## Success Metrics
- Gate demonstrably blocks the 2026-08-28 incident pattern (unacked session editing files) in both TUI and `-p` modes
- Zero false blocks on read-only work
- Ships via git — other machines inherit on pull

## Out of Scope (per design doc)
Claude Code PreToolUse parity, global pi-package distribution, coordinator-session coverage if not inherited (F5 → follow-up if needed).