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
| A4: Explicit Authority | +1 | Mutating tools unlock only behind `session_protocol_ack`, which enforces verifiable prerequisites (human `ctx.ui.confirm` in TUI, V11; observable protocol steps in session history in headless mode, V10) — not bare self-attestation. Residual: superficial compliance remains possible (F8); full feature authorization stays with the design-doc approval flow |
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
| V1 | `tool_call` fires before a tool executes and can block. Exact contract (quoted): "Return values from `tool_call` control blocking via `{ block: true, reason?: string, terminate?: boolean }`" | extensions.md "Tool Events" § `tool_call`, line 774 (read this session) | Confirmed — contract quoted verbatim, closing the unquoted-assumption objection |
| V2 | Project-local extensions auto-discover from `.pi/extensions/` (hot-reloadable via `/reload`) | extensions.md "Placement for /reload" note (line 9) | Confirmed |
| V3 | `.pi/` project resources are trust-gated; project extensions load only after trust resolves | extensions.md `project_trust` event (lines 352–369) | Confirmed — all machines that work this repo already trust it |
| V4 | `session_start` event fires on startup/new/resume/fork with `event.reason` | extensions.md "Session Events" (lines 389–397) | Confirmed |
| V5 | Extensions can persist state and register custom tools via `pi.registerTool()` | extensions.md lines 15, 78, 1453 | Confirmed |
| V10 | An extension can READ BACK prior-session state at `session_start` (the resume-disarm premise) | extensions.md "State Management" (line 1859+) gives the canonical reconstruction pattern: `pi.on("session_start", …)` iterating `ctx.sessionManager.getBranch()` and reading entries incl. tool-result `details` ("Reconstruct state from session"). Design updated to use exactly this pattern — the ack tool returns `details: { acked: true }` and `session_start` scans for it — replacing the unverified `appendEntry`-read-back premise | Confirmed — persistence mechanism now verified, not assumed |
| V11 | `ctx.ui.confirm` exists and blocks for a human decision | extensions.md `project_trust` handler example: `const confirmed = await ctx.ui.confirm("Trust project?", event.cwd)` — same API used by the ack tool in interactive mode | Confirmed |
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

**Primary Goal (scoped):** In pi sessions of this repo **where project extensions are loaded** — the default for trusted projects, interactive and `-p` alike — the edit/write built-in tools and common bash write patterns are mechanically blocked until the session protocol is acknowledged. Best-effort bash coverage and known fail-open paths are enumerated in the Fail-open register below; the gate strengthens the advisory layer, it does not claim to be a sandbox.

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
| Ack state persists across `/resume` of the same session (via the documented state-reconstruction pattern, V10) but re-arms for genuinely new sessions | A resumed session already passed the gate; a fresh session has not — matches how humans re-onboard | agent | design | low |
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

1. **Gate state** — in-memory per session; `session_start` (`reason: startup|new|resume|fork`) disarms ONLY if the session branch contains the ack tool's tool-result with `details: { acked: true }` (verified pattern, V10); otherwise arms. Fail-closed: if state reconstruction is unavailable or ambiguous, the gate arms.
2. **`session_protocol_ack` custom tool** — `pi.registerTool` with a description that enumerates the
   protocol: (a) read `CLAUDE.md`; (b) run `ailang messages list --unread` and summarize to the user
   before acking; (c) classify work via the AGENTS.md work-routing table — feature/semantics work
   additionally requires design doc → user approval → sprint plan → "execute sprint". **The tool
   enforces verifiable prerequisites before disarming** (round-2 quorum fix — closes the bare
   self-attestation hole):
   - Interactive (`ctx.hasUI`): the handler calls `ctx.ui.confirm("Complete the session protocol?")` —
     a real human keypress is required (pattern verified: `project_trust` example uses `ctx.ui.confirm`).
   - Headless (`-p`/RPC, no UI): the handler scans `getBranch()` (V10 pattern) for the protocol's
     observable steps — a tool call touching `CLAUDE.md` AND a bash call running `ailang messages` —
     and refuses with the missing steps listed if absent.
   On success it returns `details: { acked: true, at: <iso> }` (the state-reconstruction payload, V10)
   and disarms. Residual: a model can still comply superficially (run the steps without engaging) —
   that is F8 in the Fail-open register, accepted by the human; the gate converts silent skips into
   audited, prerequisite-checked steps.
3. **`tool_call` interceptor** — while armed: block `edit`/`write` unconditionally; for `bash`, **block
   any command that does not match a read-only allowlist** (fail-closed, round-2 quorum fix: a
   denylist silently passes unlisted writes — `python -c 'open(…)'`, `dd`, `cp` — which violates
   no-silent-fallbacks). Allowlist: `ls, cat, head, tail, grep, rg, find, wc, stat, file, git
   status/log/diff/show/branch, ailang check/messages list/read/doctor/builtins`. Everything else →
   blocked with the standard reason. Block reason (V1, contract quoted there):
   `"Session protocol not completed — call session_protocol_ack after reading CLAUDE.md and checking ailang messages. Feature/semantics work additionally requires an approved design doc and sprint plan."`
4. **Failure mode when UI absent** — headless/print mode has no interactive UI; the ack falls back to
   the verifiable-prerequisites path (above), and block reasons are text returned to the model, so
   gating behavior is identical (V6).

### Implementation Plan

**Phase 1: Extension core** (~1.5h)
- [ ] `.pi/extensions/session-protocol-gate.ts` — state, ack tool, tool_call interceptor
- [ ] Unit-testable pure predicate: `shouldBlock(toolName, bashCommand, acked) -> string | null`

**Phase 2: Session persistence + edge cases** (~0.5h)
- [ ] Ack reconstruction per the documented State Management pattern: `session_start` scans `getBranch()` for the ack tool's tool-result `details` (V10); fail-closed on ambiguity
- [ ] Trust-not-yet-resolved startup: gate is inert until extensions load (V3) — recorded in the Fail-open register
- [ ] `/reload` safety: reload re-runs `session_start`; ack state re-reads from the session branch (V10 pattern)

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

## Conflict Surface

**Formal trigger check:** this feature touches none of the compiler packages (`internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects`, `cmd/ailang/exec.go`) — it is a pi project extension, so the mandatory Conflict Surface for language surfaces does not apply. The related-work analysis below is included voluntarily, closing the round-2 quorum objection that prior session-hook work was cited without reconciliation:

### Related machinery: `m-session-workspace-hooks` (implemented, v0.7.0)

**What it actually is (verified by reading the doc, not the 0.37 similarity score):** Claude Code **OTEL telemetry enrichment** — hooks that attach session→workspace metadata to observability spans so dashboards can build hierarchy/filtering. Its surface is telemetry attributes; its harness is Claude Code.

| Dimension | m-session-workspace-hooks (v0.7.0) | This design |
|---|---|---|
| Harness | Claude Code (OTEL hooks) | pi (extension API) |
| Mechanism | Span metadata enrichment | `tool_call` interception + custom tool |
| Purpose | Observability (attribution, cost rollups) | Governance (gate mutation until protocol ack) |
| Shared code | None — different runtime, different hook points | — |

**Reuse verdict:** no machinery to reuse or extend — the overlap is the phrase "session hooks", not code or semantics. V7 stands: this repo has no `.pi/extensions/` today; the v0.7.0 work lives in Claude Code's hook configuration, orthogonal to pi extension discovery (V2).

### Positions touched (pi surfaces)

- Adds one file to `.pi/extensions/` (auto-discovered; V2) — a namespace this repo has never used (V7)
- Registers one custom tool (`session_protocol_ack`) — new tool name, no collision with built-in or extension tools (V7)
- Subscribes to `session_start` + `tool_call` — no other extension in this repo subscribes (V7), so no event-ordering interactions

### Programs/flows that MUST still work

1. Existing repo hooks: `make fmt` `.claude/` hook — unaffected (different harness)
2. `@ollama/pi-web-search` package extension — unaffected (its tools are not in the block set; its bash-free tool surface unaffected)
3. Headless `pi -p` runs (coordinator sub-sessions) — must still function; gate behavior identical per V6, ack via prerequisites path
4. `/reload`, `/resume`, `/fork` — session_start re-derivation per V10

### Deliberately changes

Nothing pre-existing. First session in this repo: the gate arms — that is the feature.

## Fail-open register

The gate is a governance control, not a sandbox. Every known bypass is enumerated here at design
time — each with an owner who accepts it and a mitigation or an explicit scope boundary (resolves
the quorum's fail-open objection by making the residual risk explicit and signed-for, not silent):

| # | Bypass path | Scope | Accepted by | Mitigation / disposition |
|---|---|---|---|---|
| F1 | `--no-extensions` / `-ne` CLI flag | All project extensions, incl. this gate | human (Mark) | Documented as forbidden in this repo's workflow (`.pi/extensions/README.md`, AGENTS.md note); mechanical gate is one layer of a two-layer defense whose other layer (AGENTS.md) survives |
| F2 | Project trust not yet resolved → project extensions don't load | First-ever session on a new machine | human (Mark) | Trust prompt is interactive at startup; all working machines already trust this repo; onboarding doc mentions the gate |
| F3 | Bash commands outside the read-only allowlist (e.g. `make`, `go build`, `go test`) | bash only, while armed; `edit`/`write` remain absolute | agent | Round-2 fix inverted the gate: bash is now **fail-closed allowlist** (block unless recognized read-only), so unlisted write vectors can no longer pass silently — the remaining cost is convenience false-positives while armed, resolved by completing the protocol |
| F3b | Allowlist false-positives block legitimate read-only-ish commands while armed | bash convenience | agent | Ack resolves immediately; blocked output names the remedy; `ailang` and git read-only commands are allowlisted so the protocol itself never self-blocks |
| F4 | Tools registered by OTHER extensions that mutate files | custom tool surface | agent | Out of v1 scope; `tool_call` hook still fires for them, so a v2 can extend the block set; recorded here so the gap is visible |
| F5 | Coordinator/daemon-spawned sessions | Unknown whether they inherit project extensions | human (Mark) | Verify during Phase 3; if not inherited, coverage is a follow-up doc rather than a silent assumption |
| F6 | Non-pi harnesses (Claude Code, raw API clients) | Different harness, no pi hooks | human (Mark) | Out of scope; `.claude/` PreToolUse parity listed as Future Work |
| F7 | `/reload` mid-session | Extension reload | agent | Reload re-runs `session_start`; ack state re-reads from the session branch (V10) — no duplicate gate, no disarm-by-reload |
| F8 | Superficial compliance: model runs the protocol's observable steps without genuinely engaging | Ack prerequisite check | human (Mark) | Accepted residual — a harness cannot verify comprehension; the gate converts silent skips into audited, prerequisite-checked steps and leaves a session trail; feature-level authorization remains the design-doc approval flow (human) |

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
| `--no-extensions` bypass | Low | Fail-open register F1: documented as forbidden in repo workflow; two-layer defense with AGENTS.md, which survives |
| pi API drift between versions | Low | Pin expectation in README (tested against pi 0.84.3); `pi update` runs are attended |

## Related Documents

<!-- Auto-populated by Ollama neural search on "dx session protocol gate"; duplicate gate passed (max neural 0.46, unrelated) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_0/m-session-workspace-hooks.md](design_docs/implemented/v0_7_0/m-session-workspace-hooks.md) (0.37) — analyzed in Conflict Surface: Claude Code OTEL telemetry enrichment; no shared machinery with pi tool-call interception
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](design_docs/implemented/v0_6_0/semantic-caching-complete.md) (0.38) — unrelated (embedding cache)

**Planned (check for overlap):**
- [design_docs/planned/v1_1_0/m-csp-session-types.md](design_docs/planned/v1_1_0/m-csp-session-types.md) (0.46) — unrelated (language session types, not agent sessions)

## References

- pi v0.84.3 `docs/extensions.md` — `tool_call` block contract, quoted in V1; `.pi/extensions/` discovery (V2), trust (V3), `session_start` (V4), `registerTool` (V5), State Management reconstruction pattern (V10)
- AGENTS.md "Work Routing (do not self-approve)" — the advisory layer this gate mechanically backs
- CLAUDE.md — session-start routine and critical principles the gate enforces

## Future Work

- Promote to a global pi package (`pi install <source>`) for cross-repo enforcement on all machines
- Claude Code PreToolUse hook parity for Claude-driven shells
- Coordinator-spawned session coverage, if not inherited automatically

---

**Document created**: 2026-08-28
**Last updated**: 2026-08-28