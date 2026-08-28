# M-DX-SESSION-PROTOCOL-GATE — mechanical enforcement of the session protocol for pi sessions

**Status**: Planned
**Target**: v0.35.0
**Priority**: P0 (High — this repo's agent workflow is the product)
**Estimated**: ~3 hours (single sprint session)
**Dependencies**: None. Complements the AGENTS.md work-routing gate added 2026-08-28 (`007184b7b`).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This feature is repo developer-experience tooling (a pi extension), not AILANG language surface. It changes no language semantics; it scores on the axioms that concern agent workflow and authority.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect surface change |
| A4: Explicit Authority | +1 | Turns "please follow the protocol" into an explicit, inspectable authorization step (`session_protocol_ack`) before mutating tools are granted |
| A5: Bounded Verification | +1 | Gate state is locally checkable (`ailang doctor`-style: extension loads, tool registered) |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | Removes a recurring silent-failure mode of agent sessions (skipped protocol) — a machine-analysis win, not human convenience |
| A8: Minimal Syntax | +1 | No language syntax |
| A9: Cost Visibility | 0 | Gate adds one tool call per session (~100 tokens) — negligible, visible |
| A10: Composability | +1 | Composes with existing repo hooks (`.claude/` fmt hooks); does not replace the skills pipeline, enforces its entry |
| A11: Structured Failure | +1 | Block reason is a structured, actionable message, not a crash |
| A12: System Boundary | +1 | The harness/agent boundary is exactly where the gate lives |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations (A4 is strengthened, not weakened: authority is granted explicitly via the ack tool)

## Verification Log

All pi-platform claims verified against pi v0.84.3 docs at
`/Users/mark/.nvm/versions/node/v25.5.0/lib/node_modules/@earendil-works/pi-coding-agent/docs/extensions.md` (read in full sections cited):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `tool_call` event fires before a tool executes and can block with `{ block: true, reason? }` | extensions.md "Tool Events" § `tool_call` (lines 760–791) | Confirmed |
| V2 | Project-local extensions auto-discover from `.pi/extensions/` (hot-reloadable via `/reload`) | extensions.md "Placement for /reload" note (line 9) | Confirmed |
| V3 | `.pi/` project resources are trust-gated; project extensions load only after trust resolves | extensions.md `project_trust` event (lines 352–369) | Confirmed — all machines that work this repo already trust it |
| V4 | `session_start` event fires on startup/new/resume/fork with `event.reason` | extensions.md "Session Events" (lines 389–397) | Confirmed |
| V5 | Extensions can persist state via `pi.appendEntry()` and register custom tools via `pi.registerTool()` | extensions.md lines 15, 78, 1453 | Confirmed |
| V6 | Headless `pi -p` runs load project extensions by default (only `--no-extensions` disables) | extensions.md + pi `--help` (`-ne` flag text) | Confirmed — the 2026-08-28 incident was itself a `pi -p` sub-run, so the gate must cover print mode |
| V7 | No existing extension in this repo gates session protocol | `ls .pi/extensions/ 2>/dev/null` → empty; no `.pi/extensions` tracked in git | Confirmed (negative-existence) |
| V8 | AGENTS.md is auto-loaded into every pi session in this repo (context-file discovery) | pi `--help` (`--no-context-files` disables "AGENTS.md and CLAUDE.md discovery"); observed live 2026-08-28 — this session's system prompt contains AGENTS.md | Confirmed |
| V9 | Advisory text demonstrably fails: a prior session was told "Read CLAUDE.md first" in AGENTS.md, skipped it, and began implementing a feature | This session, 2026-08-28 (recorded in AGENTS.md work-routing section) | Confirmed — the reason this doc exists |

## Problem Statement

The repo's agent workflow (read CLAUDE.md → check messages → route work by scope) is documented
in AGENTS.md/CLAUDE.md, but both are **advisory**: nothing stops a model from picking an issue and
editing code. This is not hypothetical — on 2026-08-28 a session skipped the "read CLAUDE.md first"
pointer in AGENTS.md and began implementing issue #897 with no design doc or approval (V9). The
same failure mode applies to any model this repo routes to (GLM, Ollama, frontier APIs).

**Current State:** enforcement is prose in context files; compliance depends on model judgment.

**Impact:** Every pi invocation in this repo — interactive, headless, sub-agent — can silently
bypass human approval on feature work. The cost of one violation was ~45 minutes of unpark/review
overhead; the tail risk is unreviewed language-semantics changes landing on `dev`.

## Goals

**Primary Goal:** No pi session in this repo can execute a file-mutating tool until it has completed (or explicitly acknowledged) the session protocol.

**Success Metrics:**
- Fresh pi session in a trusted checkout: `edit`/`write` blocked with an actionable reason until the ack tool is called
- After `session_protocol_ack`: tools unblocked for the session's lifetime; no repeated nagging
- Headless `pi -p` behavior identical (V6)
- Extension ships in-repo (`.pi/extensions/`) — zero per-machine setup; arrives via `git pull`
- Gate overhead per session ≤ 1 tool call + ~150 prompt tokens

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Block set v1 = `edit` + `write` (absolute) + bash patterns (best-effort) | edit/write are the mutation primitives the harness controls fully; bash is a wide surface — cover `>`, `tee`, `sed -i`, `git apply/push/reset/checkout --` patterns v1, widen later | agent | design | low |
| Disarm = calling the `session_protocol_ack` custom tool once per session | One explicit, logged acknowledgment; the tool description *is* the protocol, so acknowledging requires engaging with it | human (confirm) | design | low |
| Ack state persists across `/resume` of the same session (via `pi.appendEntry`) but re-arms for genuinely new sessions | A resumed session already passed the gate; a fresh session has not — matches how humans re-onboard | agent | design | low |
| Read-only tools never blocked | The protocol itself requires reading CLAUDE.md, checking messages, reading issues — the gate must not block its own remedy | agent | design | low |

### Design Freeze

- [ ] Confirm v1 block set (edit/write absolute + bash pattern list) and the ack-based disarm

## Solution Design

### Overview

One project-scoped pi extension, `.pi/extensions/session-protocol-gate.ts`, committed to the repo
(distribution = git; V2, V3). On every session start it arms a gate; the agent disarms by calling a
custom tool whose description spells out the protocol. While armed, mutating tool calls are blocked
with a structured reason (V1).

### Components

1. **Gate state** — in-memory per session; `session_start` (`reason: startup|new|resume|fork`) arms
   the gate unless the session file already contains a prior ack entry (`pi.appendEntry("session_protocol_ack", …)`; V4, V5).
2. **`session_protocol_ack` custom tool** — `pi.registerTool` with a description that enumerates the
   protocol: (a) read `CLAUDE.md`; (b) run `ailang messages list --unread` and summarize to the user
   before acking; (c) classify work via the AGENTS.md work-routing table — feature/semantics work
   additionally requires design doc → user approval → sprint plan → "execute sprint". Calling the
   tool appends the ack entry and disarms. The tool validates nothing about *what the agent read* —
   it is an engagement gate plus an audit trail entry, not a comprehension test (the advisory layer
   remains AGENTS.md; see Non-Goals).
3. **`tool_call` interceptor** — while armed: block `edit`/`write` unconditionally; block `bash` calls
   matching write patterns (redirect `>`/`>>`, `tee`, `sed -i`, `git apply|push|reset|clean|checkout --`,
   `rm -rf` outside tmp). Block reason (V1):
   `"Session protocol not completed — call session_protocol_ack after reading CLAUDE.md and checking ailang messages. Feature/semantics work additionally requires an approved design doc and sprint plan."`
4. **Failure mode when UI absent** — headless/print mode has no interactive UI; the gate never needs
   `ctx.ui` (block reasons are text returned to the model), so behavior is identical (V6).

### Implementation Plan

**Phase 1: Extension core** (~1.5h)
- [ ] `.pi/extensions/session-protocol-gate.ts` — state, ack tool, tool_call interceptor
- [ ] Unit-testable pure predicate: `shouldBlock(toolName, bashCommand, acked) -> string | null`

**Phase 2: Session persistence + edge cases** (~0.5h)
- [ ] Ack entry append + resume detection
- [ ] Trust-not-yet-resolved startup: gate is inert until extensions load (V3) — document as known gap in README comment
- [ ] `/reload` safety: reload re-runs `session_start`; ack state re-reads from session entries

**Phase 3: Validation** (~1h)
- [ ] Fresh interactive session: edit blocked → ack → edit allowed
- [ ] `pi -p "edit a file"` headless: blocked with reason (V6)
- [ ] Resumed session with prior ack: not blocked
- [ ] Write a short `.pi/extensions/README.md` note (what the gate does, how to disarm, `--no-extensions` escape hatch and why the repo workflow forbids it)

### Files to Modify/Create

**New files:**
- `.pi/extensions/session-protocol-gate.ts` — the gate (~120 LOC)
- `.pi/extensions/README.md` — operational notes (~30 LOC)

**Modified files:**
- `AGENTS.md` — one line under Work Routing pointing at the mechanical gate (post-merge note)
- `CHANGELOG` (v0.35.0 section) — entry

## Examples

### Example 1: Fresh session, model wants to implement immediately

```
model:  [calls edit on internal/effects/fs.go]
gate:   BLOCKED — "Session protocol not completed — call session_protocol_ack after reading
        CLAUDE.md and checking ailang messages. Feature/semantics work additionally requires
        an approved design doc and sprint plan."
model:  [reads CLAUDE.md, runs ailang messages list --unread, summarizes to user]
model:  [calls session_protocol_ack]
gate:   ack recorded (session entry), tools unlocked
```

### Example 2: Resumed session

Session file already contains `session_protocol_ack` from yesterday → gate inert at `session_start`
(`reason: resume`), agent works immediately. New session next morning → armed again.

## Success Criteria

- [ ] Fresh pi session in this repo: `edit` blocked with actionable reason until ack (acceptance: manual TUI test)
- [ ] `pi -p` headless: same block, same reason (acceptance: scripted run)
- [ ] After ack: zero further blocks for the session (acceptance: manual test)
- [ ] Resumed session with prior ack: no block (acceptance: manual test)
- [ ] Read-only tools (read, grep, messages) never blocked (acceptance: scripted run)
- [ ] Extension hot-reloads with `/reload` without duplicating gates (acceptance: manual test)
- [ ] No regression in existing repo hooks (`make fmt` hook events unaffected)

## Testing Strategy

**Unit tests:** the `shouldBlock` predicate as a table-driven test (tool name × bash pattern × acked) —
plain vitest/node:test against the TS module, no pi runtime needed.

**Manual testing:** the Phase 3 checklist against the real TUI and `-p` mode on this machine.

**Manual testing (other machines):** Studio + cloud pull and repeat the fresh-session test once.

## Deferred Decisions

- Exact ack-tool name (`session_protocol_ack` proposed) — agent may adjust before implementation.
- Whether `bash` gating later graduates to full command parsing — agent may propose v2 doc.
- Global (cross-repo) distribution as a pi package — deferred; see Future Work.

## Non-Goals

- **Blocking reads** — the gate must never obstruct its own remedy.
- **Content inspection of *what* the agent read** — v1 proves engagement + leaves an audit trail; deeper verification (e.g., quizzing) is over-engineering for now.
- **Claude Code / other-harness parity** — separate follow-up if wanted (`.claude/` PreToolUse hook); not this doc.
- **Coordinator/daemon sessions** — coordinator spawns its own sessions; wiring the gate there is future work if the coordinator doesn't inherit project extensions.

## Timeline

**Single session** (~3h): core + persistence + manual test pass. Same-day deploy to all machines via git pull.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gate blocks the remedy (agent can't read docs to complete protocol) | High | Read-only tools never blocked; block reason names the exact remedy |
| Model calls ack without doing the protocol | Med | Ack description requires the steps; AGENTS.md + audit trail make skips visible in session files; acceptable residual (harness cannot verify comprehension) |
| Gate annoys quick one-line fixes | Low | Trivial-fix scope per AGENTS.md still passes the gate in seconds (read CLAUDE.md, ack, state change, act) |
| `--no-extensions` bypass | Low | Documented as forbidden in repo workflow; extension is advisory+mechanical belt-and-braces with AGENTS.md, which survives |
| pi API drift between versions | Low | Pin expectation in README (tested against pi 0.84.3); `pi update` runs are attended |

## Related Documents

<!-- Auto-populated by Ollama neural search on "dx session protocol gate"; duplicate gate passed (max neural 0.46, unrelated) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_0/m-session-workspace-hooks.md](design_docs/implemented/v0_7_0/m-session-workspace-hooks.md) (0.37) — prior session-hook work, adjacent surface
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](design_docs/implemented/v0_6_0/semantic-caching-complete.md) (0.38) — unrelated (embedding cache)

**Planned (check for overlap):**
- [design_docs/planned/v1_1_0/m-csp-session-types.md](design_docs/planned/v1_1_0/m-csp-session-types.md) (0.46) — unrelated (language session types, not agent sessions)

## References

- pi v0.84.3 `docs/extensions.md` — `tool_call` blocking (V1), `.pi/extensions/` discovery (V2), trust (V3), `session_start` (V4), `appendEntry`/`registerTool` (V5)
- AGENTS.md "Work Routing (do not self-approve)" — the advisory layer this gate mechanically backs
- CLAUDE.md — session-start routine and critical principles the gate enforces

## Future Work

- Promote to a global pi package (`pi install <source>`) for cross-repo enforcement on all machines
- Claude Code PreToolUse hook parity for Claude-driven shells
- Coordinator-spawned session coverage, if not inherited automatically

---

**Document created**: 2026-08-28
**Last updated**: 2026-08-28