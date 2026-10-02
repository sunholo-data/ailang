# M-CLAUDE-CODE-MODS — AILANG features as Claude Code mods (interactive, local agent, headless)

**Status**: Planned
**Target**: v0.52.0 (Phase 0 spike can land on any patch)
**Priority**: P2 — developer experience and fleet ergonomics; nothing is blocked without it
**Estimated**: Phase 0 ~0.5 session · Phase 1 ~1 session · Phase 2 ~1–2 sessions · Phase 3 ~1 session · Phase 4 ~1 session
**Dependencies**: None hard. First mod already exists: `ailang-lens` ([ailang_bootstrap#11](https://github.com/sunholo-data/ailang_bootstrap/pull/11)).
**Quorum**: REQUIRED — trigger 1 (design-freeze items) and trigger 4 (load-bearing premises about an external system we do not control: Claude Code's early-access mods API). Not yet run.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Harness/developer-experience tooling, not language surface. Same category as
[M-DX-PI-HARNESS](../implemented/v0_35_0/m-dx-pi-harness.md), scored the same way.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | +1 | The lens puts every function's effect row (`pure` / `!{IO,FS}`) in front of the person as the code is written — effect legibility made literal, at the moment it matters |
| A4: Explicit Authority | 0 | Deliberately none: mods are advisory UX and context, **never** an authority boundary (see Constraints). Scoring +1 here would be claiming exactly what CLAUDE.md principle 5 forbids |
| A5: Bounded Verification | +1 | Every mod subprocess runs under an explicit timeout (`$.process.run` `timeoutMs`, the pi Subprocess Contract carried over); a hung `ailang` degrades to a structured result, never a stuck session |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | Phase 2 feeds structured `ailang check` diagnostics straight into the model's tool result after a `.ail` edit — the agent sees the error without spending a turn asking for it |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No metered calls added; Phase 4 *removes* per-tool-call process spawns but that is not a cost the axiom tracks |
| A10: Composability | +1 | Every mod wraps an existing CLI (`ailang iface`, `check --json`, `messages list --json`) — never re-implements it |
| A11: Structured Failure | +1 | Diagnostics are shown and fed as `line:col cause`, parsed from the JSON contract, not scraped text |
| A12: System Boundary | 0 | No change to the .ail ↔ host boundary |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no −1 scores (A3, A7 strengthened; A4 neutral by design)

## Verification Log

Every load-bearing premise, with how it was checked. PENDING rows gate the phase named.

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Claude Code mods exist as plugins whose `hooks/hooks.json` names a `{ "modules": [...] }` entry exporting `register(on)`; hooks are middleware `($, e, next)` that can observe, rewrite or answer `tool.call`, `prompt.submit`, `prompt.compose`, `turn.complete`, `ui.render`, … | claude.dev blog "Getting started with Claude Code mods"; the engine's own `claude-code.d.ts` (Claude Code 2.1.287); `claude plugin validate` on `ailang-lens` lists the hooks it registered | Confirmed |
| V2 | A mod can draw a docked Pane, an AbovePrompt band, a status line and toasts; panes opened unasked seat only from 144 terminal columns | plugin-authoring reference + types; `ailang-lens` pane observed live by Mark 2026-10-02 | Confirmed |
| V3 | `ailang iface <file>` emits `{"schema":"ailang.iface/v1","module","types","funcs":[{name,type,effects,pure}]}` on stdout; a stdlib-version warning can precede it | Ran live on `examples/strict_fallbacks_demo.ail` (v0.50.0 binary, v0.51.0 stdlib) | Confirmed — parser must skip to the first `{` (lens does) |
| V4 | `ailang check --json --quiet <file>` emits `{file,passed,error_count,errors:[{code,message,file}]}`, exit ≠ 0 on failure | Ran live, passing and failing files | Confirmed |
| V5 | `ailang` resolves a module's expected name from its path relative to the working directory (MOD010), so an absolute path from Claude's Write tool fails a correct module | Live: `module lens_demo` at an absolute scratch path → MOD010; same file from its own dir → passes | Confirmed — mods must run `ailang` from the project root (lens: nearest `ailang.toml`, else `.git`, else the file's dir) |
| V6 | A `tool.call` hook only sees Claude's own tool calls; a file changed by Bash (`sed`) or by the person's editor is invisible to an Edit/Write hook | Live 2026-10-02: lens stayed ✗ after a `sed` restore | Confirmed — lens re-checks tracked files by `mtime` after every Bash call and at `turn.complete` |
| V7 | `ailang messages list --unread --json [--inbox X]` returns an array of `{id,from_agent,to_inbox,message_type,title,payload,status,created_at}`; `messages watch` exists (1 s SQLite poll, or `--pubsub`) | Ran live 2026-10-02; `ailang messages --help`, `messages watch --help` | Confirmed |
| V8 | `scripts/hooks/coordinator_hook.sh` is registered on 7 event types including PreToolUse **and** PostToolUse for `Bash\|Edit\|Write\|Read\|Agent\|WebFetch\|WebSearch\|mcp__.*` — at least two `bash`+`curl` processes per tool call | Parsed `.claude/settings.json` on `origin/dev` | Confirmed |
| V9 | The Claude executor already passes `--plugin-dir` per task (`internal/executor/claude/claude.go:228-231`), cloud mode can clone a `plugin_repo` (`internal/coordinator/agent_config.go:91-94`), and per-agent `plugins: {marketplaces, install}` already exists in `config.cloud.yaml` (website-builder installs `frontend-design`) | Read the code and config | Confirmed — headless delivery needs **no new plumbing** |
| V10 | `claude -p` loads `--plugin-dir` mods: `session.start` fires, `$.process.run` and `$.fs` work, `tool.call` fires per tool, and a result's `context` reaches the model | **Measured 2026-10-02** (Claude Code 2.1.287, local): probe mod under `claude -p --plugin-dir probe --max-turns 3`; marker file got `session.start ailang=AILANG v0.50.0…` and `tool.call Bash`; the model quoted back `tool.call hook additional context: PROBE-CONTEXT-SEEN`; empty stderr | **Confirmed locally.** Inside `Dockerfile.agent` on a Cloud Run job: not yet run (Phase 3 entry check) |
| V11 | Mods availability is server-gated per process. On 2026-10-02 `claude plugin test` refused with "hooks modules are turned off in this process: the rollout switch served off" ~20 min after the same command had run tests, while the interactive session kept its mods loaded | Observed this session | Confirmed — **nothing load-bearing may live in a mod** (Constraint C2) |
| V12 | Mod hooks run in a sandbox with no Node and no DOM; host access only via `$` (`$.process.run`, `$.fs`, `$.http`, `$.clock`); ~10 s budget per hook excluding time inside `$` calls | Types header + blog | Confirmed — pi extensions (Node) cannot be loaded as-is |
| V13 | No existing design doc covers Claude Code mods | `grep -rli "claude code mod\|function hooks\|hooks module" design_docs` → empty; neural search top match 0.46 (`M-CLAUDE-CODE-INTEGRATION`, v0.3.20 shell hooks + inbox) | Confirmed |
| V14 | No `ailang claude install` command exists; `ailang pi install` (`cmd/ailang/pi_setup.go`, `//go:embed all:pi_assets`) and `ailang editor install vscode` are the precedents | `grep` over `cmd/ailang/*.go`; `ls cmd/ailang` | Confirmed |

## Problem Statement

AILANG's harness work has three homes that each grew their own extension layer:
pi extensions (`.pi/extensions/`, 14 files), Motoko's compiled-in `motoko-ext-*`
packages, and Claude Code **shell hooks** (`.claude/settings.json`, 16 entries).
The Claude Code layer is the weakest of the three:

**Current State:**
- Shell hooks are stateless and spawn a process per event (V8: ≥2 per tool call
  just for telemetry), can only allow/block/add context, and cannot draw anything.
- Whatever a person writing AILANG with Claude wants to *see* — the module's
  types and effects, whether it type-checks, unread fleet messages — lives in a
  separate terminal (`ailang check`, `ailang messages list`) or the dashboard.
- pi has stateful gates (`unowned-dirty`, `session-protocol-gate`,
  `quality-monitor`) that Claude Code sessions in the same repo don't get.
- Claude Code now has **mods** (V1, V2): in-process, stateful, able to rewrite
  tool calls, register tools/commands, compose the system prompt and draw UI,
  hot-reloading while the agent edits them. `ailang-lens` proved the shape
  in an afternoon (V2–V6).

**Impact:** humans reviewing AILANG written by agents can't see effect rows
without running tools; agents spend turns re-running `ailang check`; the
Claude Code harness lags pi on the doctrine "a friction that bites twice gets
encoded" ([M-DX-PI-HARNESS](../implemented/v0_35_0/m-dx-pi-harness.md) §Doctrine).

## Goals

**Primary Goal:** Give AILANG a small, versioned set of Claude Code mods that make
AILANG visible to the person (interactive), feed structured AILANG feedback to the
agent (local agent), and carry the agent-facing subset into headless fleet runs —
without ever making a mod load-bearing.

**Success Metrics:**
- A person editing `.ail` with Claude sees types, effect rows and errors update within one tool call (lens; met for Edit/Write, V6 for the rest).
- Unread messages for the person's inboxes are visible above the prompt without running a command.
- In a measured A/B on one dev agent, check-on-edit feedback reduces explicit `ailang check` Bash calls per `.ail` task (direction measured, not assumed — Phase 3).
- `coordinator_hook.sh` process spawns per tool call go from ≥2 to 0 where the telemetry mod is loaded (Phase 4).
- Every mod degrades to "nothing happens" when the rollout switch is off (V11) — verified by running the same session with mods off.

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Change cost later |
|---|----------|---------|----------------|-------------|-------------------|
| D1 | **Source of truth** for mods | (a) `ailang_bootstrap/plugins/` (where `ailang-lens` is today) · (b) `ailang` repo `tools/claude-mods/`, copied to bootstrap by the existing `sync-ailang.yml` | **(b)** — mods call `ailang` flags and JSON schemas (V3, V4, V7) that change with the binary; versioning them with the binary is the same reasoning that put pi extensions in-tree. Bootstrap stays the *distribution* | Mark | Low (move a folder, add one sync step) |
| D2 | **Distribution** | (a) marketplace only (`/plugin install ailang-lens@ailang-marketplace`) · (b) also `ailang claude install` (embed, write `~/.claude/ailang-mods/`, add to `CLAUDE_CODE_PLUGIN_DIRS`), mirroring `ailang pi install` (V14) · (c) both | **(a) now, (b) deferred** until the API leaves early access — embedding an early-access surface in every release binary couples our release cadence to theirs | Mark | Low |
| D3 | **Shared logic with pi** | (a) separate implementations · (b) a pure-TS core (no Node, no `$`) per gate + two thin adapters (pi/Node, Claude/`$`) · (c) code-gen | **(b) for gates with real logic** (`unowned-dirty`, `prepush-gate`, check-on-edit parsing); **(a) for UI-only mods** (lens, inbox) — pi has no equivalent surface. V12 makes a shared core possible only if it takes `run(argv)` and `stat(path)` as injected functions | Agent-resolvable after D1 | Medium |
| D4 | **Headless scope** | (a) no mods headless · (b) agent-facing mods opt-in per agent via the existing `plugins.install` (V9) · (c) default-on fleet-wide | **(b)**, starting with one dev agent, measured (Phase 3) | Mark | Low (config only) |
| D5 | **Messages in model context** | (a) interactive UI only (band/pane/toast for the person) · (b) also inject unread-message summaries into the agent's system prompt via `prompt.compose` | **(a)**. Message bodies are external content (other agents, GitHub issues) — putting them in the prompt makes them instruction-shaped input. The person reads them; the agent reads one only when asked, via the existing CLI, as data | Mark | Medium — reversing means a trust review against [M-MESSAGE-PLANE-TRUST](m-message-plane-trust.md) |
| D6 | **Shell hooks vs mods** | (a) replace hooks with mods · (b) mods alongside hooks; retire a hook only after its mod is proven *and* the rollout question (V11) is settled | **(b)** — hooks keep working when the switch is off and in non-Claude harnesses | Agent-resolvable | Low |

### Design Freeze

- [ ] D1 source of truth
- [ ] D2 distribution (marketplace now)
- [ ] D4 headless scope (opt-in, one agent first)
- [ ] D5 messages never in model context

## Solution Design

### Overview

Three modes, one catalogue. Each mod is tagged with the modes it serves:

| Mod | Interactive (person sees) | Local agent (model gets) | Headless (`claude -p`) | Ports from |
|-----|:---:|:---:|:---:|---|
| **ailang-lens** — pane of types / effect rows / errors per edited module; status line | ✅ shipped | — | — (no UI) | new |
| **ailang-inbox** — AbovePrompt band "📬 3 unread · user", `/inbox` pane with Read / Ack, toast on arrival | ✅ | — (D5) | — | `session_start.sh` inbox check, `ailang-inbox` skill |
| **ailang-check-on-edit** — after a successful `.ail` Edit/Write: `ailang fmt --write`, then `ailang check --format agent`; diagnostics appended to the tool result | ✅ (via lens) | ✅ | ✅ (D4) | `format_ail.sh`, pi `ail-fmt-autolint`, pi `ailang-lsp-lite` |
| **unowned-dirty** — tracks files this session wrote (`$.state`); warns on `git add -A` / `commit -a` that would sweep others | ✅ | ✅ | ✅ | pi `unowned-dirty.ts` |
| **prepush-gate** — runs CI gates before `git push`; with a person present, shows what would go out (blog's "Blast Radius" pattern) | ✅ | ✅ | ✅ | pi `prepush-gate.ts` |
| **fleet-status** — status-line entry: provider quota, current sprint/mission step | ✅ | — | — | pi `provider-quota`, `builtin-sprint` |
| **coordinator-telemetry** — one in-process forwarder replacing the 7-event `coordinator_hook.sh` (V8), via `$.http`, non-blocking | — | — | ✅ | `coordinator_hook.sh` |

### Architecture

```
            ┌──────────── Claude Code session (interactive or -p) ────────────┐
 .ail edit →│ tool.call ─► check-on-edit ─► result + diagnostics ─► model    │
            │      └──────► lens ─► $.state ─► Pane / status line ─► person  │
 timer   →  │ $.clock ─► inbox ─► `ailang messages list --json` ─► band/toast │
            │ every hook: $.process.run(argv, {cwd: project root, timeoutMs})│
            └───────────────────────────────┬────────────────────────────────┘
                                            ▼
                         ailang CLI (iface · check · fmt · messages)
```

**Components:**
- **Mods** (`tools/claude-mods/<name>/`, D1): `.claude-plugin/plugin.json`, `hooks/hooks.json`, `hooks/register.tsx`, `types/index.d.ts` (state contract), `tests/*.test.tsx`.
- **Shared helpers** (copied per mod — mods cannot import across plugins): `projectRoot` (V5), `parseJson` (skip to first `{`, V3), `shortError`.
- **Shared gate cores** (D3b): `tools/agent-ext-core/<gate>.ts`, pure functions; adapters in `.pi/extensions/` and `tools/claude-mods/`.
- **Delivery**: marketplace entries in `ailang_bootstrap` (D2a); headless via per-agent `plugins.install` (V9, D4b).

### Implementation Plan

**Phase 0: Headless spike** (~0.5 session) — closes V10.
- `claude -p --plugin-dir <ailang-lens>` on a task that writes a `.ail` file; confirm the module loaded (no stderr refusal), `tool.call` fired, `$.process.run` worked, and output is unchanged when the switch is off.
- Same inside `Dockerfile.agent` on a dev Cloud Run job.
- Outcome recorded as V10 Confirmed/Refuted. **Refuted → Phases 3 and 4 are dropped**, Phases 1–2 still go ahead.

**Phase 1: Interactive** (~1 session)
- Move `ailang-lens` to `tools/claude-mods/` (D1), sync step to bootstrap.
- `ailang-inbox`: `$.clock` poll (60 s) of `ailang messages list --unread --json --inbox <configured>`; band shows count + newest title; pane lists messages with `Read` (opens `ailang messages read <id>` output) and `Ack` Buttons; toast on a new id. Inboxes from `userConfig` (default `user`).
- `fleet-status` status line.

**Phase 2: Local agent** (~1–2 sessions)
- `ailang-check-on-edit`: fmt + check after `.ail` Edit/Write, diagnostics appended to the tool result as `ailang check: ✗ 3:8 cannot unify int vs string` (compact, `--format agent`). The lens reads the same result instead of re-running check.
- Port `unowned-dirty` and `prepush-gate` on a shared core (D3b); pi adapters switch to the core in the same PR.

**Phase 3: Headless, measured** (~1 session) — only if V10 Confirmed.
- Enable `ailang-check-on-edit` + `unowned-dirty` on **one** dev agent via `plugins.install`.
- A/B against the same agent without: explicit `ailang check` Bash calls per `.ail` task, turns to green, task success. Ship fleet-wide only on a measured win.

**Phase 4: Telemetry consolidation** (~1 session) — only if V10 Confirmed.
- `coordinator-telemetry` posts the same payloads as `coordinator_hook.sh` with the same `X-Ailang-*` headers, fire-and-forget. Run both in parallel for one day, diff what the coordinator received, then remove the shell entries from `.claude/settings.json` (D6).

### Files to Modify/Create

**New files:**
- `tools/claude-mods/ailang-lens/` — moved from bootstrap (~260 LOC)
- `tools/claude-mods/ailang-inbox/` — band, pane, poll (~250 LOC + tests)
- `tools/claude-mods/ailang-check-on-edit/` — (~150 LOC + tests)
- `tools/claude-mods/unowned-dirty/`, `tools/claude-mods/prepush-gate/` — adapters (~80 LOC each)
- `tools/claude-mods/fleet-status/` — (~80 LOC)
- `tools/claude-mods/coordinator-telemetry/` — (~120 LOC)
- `tools/agent-ext-core/unowned-dirty.ts`, `tools/agent-ext-core/prepush-gate.ts` — pure cores (~150 LOC each)

**Modified files:**
- `.pi/extensions/unowned-dirty.ts`, `.pi/extensions/prepush-gate.ts` — call the core
- `ailang_bootstrap/.github/workflows/sync-ailang.yml` — copy `tools/claude-mods/*` into `plugins/`
- `ailang_bootstrap/.claude-plugin/marketplace.json` — one entry per mod
- `.claude/settings.json` — Phase 4 only: remove `coordinator_hook.sh` entries
- `ailang-multivac/config/config.cloud.yaml` — Phase 3: `plugins.install` on one dev agent

## Examples

### Example 1: The person writing AILANG (shipped)

**Before:** Claude edits `shop.ail`; to see what changed the person runs
`ailang check` and `ailang iface` in another terminal.

**After:** the lens pane, updated by the edit:
```
demo/shop  ✓ checks  212ms
types Order
total       pure
  (Order)->int
main        !{IO,FS}
  (())->()!{IO,FS}
```

### Example 2: Inbox above the prompt (Phase 1)

```
📬 2 unread · user — v0.51.0 strict VM: nested constructor patterns unbound …   /inbox
> _
```
`/inbox` opens a pane; `Ack` runs `ailang messages ack <id>`. Nothing reaches the model (D5).

### Example 3: The agent sees its own error (Phase 2)

Tool result the model receives after an Edit:
```
The file shop.ail has been updated successfully.
ailang check: ✗ 3:8 cannot unify type constructors: int vs string
```
Today the same information costs a Bash call and a turn.

### Example 4: Headless opt-in (Phase 3)

```yaml
# config.cloud.yaml, one dev agent
plugins:
  marketplaces: [sunholo-data/ailang_bootstrap]
  install: [ailang-check-on-edit@ailang-marketplace, unowned-dirty@ailang-marketplace]
```

## Success Criteria

- [ ] V10 resolved (Confirmed or Refuted) with a transcript in the Verification Log
- [ ] Each mod: `claude plugin validate` clean, `tsc` clean, `claude plugin test` covering its stated behaviour on `terminal` and `desktop` surfaces
- [ ] Each mod: same session with mods switched off behaves exactly as without the plugin (C2)
- [ ] Inbox: a message sent with `ailang messages send user …` appears in the band within one poll interval
- [ ] Check-on-edit: diagnostics present in the tool result for a failing edit, absent for a passing one
- [ ] Phase 3 A/B numbers recorded before any fleet-wide enablement
- [ ] Phase 4: one-day parallel run shows identical coordinator payload counts before hook removal
- [ ] README (bootstrap) and `skills/ailang/SKILL.md` updated per mod

## Testing Strategy

**Unit tests:** `claude plugin test` per mod with `process.run`, `fs.stat`, `ui.status`, `ui.open` answered by the test (pattern proven in `ailang-lens/tests/lens.test.tsx`). Shared cores: plain TS tests run by both adapters' test runners.

**Integration tests:** Phase 0 headless transcript; Phase 3 A/B on a dev agent.

**Manual testing:** hot-reload session (plugin-authoring dev folder), edit `.ail` files through Edit, Write and Bash; send and ack messages.

## Constraints

- **C1 — Not a security boundary.** Mods run inside the agent's process like hooks do. A gate mod is a convenience and a nudge; the boundary is per-lane IAM, executor SAs, the egress lock and `ailang run --policy` (CLAUDE.md principle 5, [m-git-guardrails-hookless-executors](m-git-guardrails-hookless-executors.md)). Docs and code comments must never cite a mod as a control.
- **C2 — Never load-bearing.** Availability is server-gated per process (V11). Every mod must leave the session correct when absent; nothing in the coordinator, CI or the fleet may depend on a mod having run.
- **C3 — Messages are data.** No mod puts message payloads into model context (D5).
- **C4 — Bounded subprocesses.** Every `$.process.run` sets `timeoutMs` (check 60 s, iface 20 s, messages 10 s); a timeout shows as a structured error in the pane/status line, never a retry loop.
- **C5 — Project root.** Every `ailang` call runs from the resolved project root with a root-relative path (V5).

## Deferred Decisions

- Exact poll interval and inbox list for `ailang-inbox` — agent picks, `userConfig`-overridable.
- Whether the lens shows `--verify` (Z3 contract) results — a toggle is fine; default off (slow).
- Pane layout details, colours, truncation — agent latitude.
- Whether `fleet-status` reads quota via CLI or `$.http` — whichever already has a JSON contract.

## Non-Goals

**Not attempted in this feature:**
- Porting Motoko's loop-level hooks (`on_build_system_prompt`, `on_response_intercept`, `on_solver_candidate`, budget planning). Claude Code owns its loop; mods do not expose response interception at that depth.
- Any mod that sends messages or completion reports on the agent's behalf — completion reporting belongs to the coordinator ([M-MESSAGE-PLANE-TRUST](m-message-plane-trust.md)).
- `ailang claude install` (D2b) — deferred until the mods API leaves early access.
- Replacing pi or Motoko extension layers.

## Timeline

**Week 1:** Phase 0 spike, Phase 1 (inbox, fleet-status, lens move).
**Week 2:** Phase 2 (check-on-edit, two gate ports on shared cores).
**Week 3:** Phase 3 A/B; Phase 4 parallel run if V10 Confirmed.

**Total: ~4.5–5.5 sessions across 3 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Mods API changes (early access) | Mods stop loading or validating | C2 makes that a UX regression only; mods pinned to a tested Claude Code version in the README; `claude plugin validate` in bootstrap CI |
| Rollout switch off for some users/processes (V11) | Feature silently absent | C2; README says so; no support burden beyond "it's optional" |
| Headless `-p` doesn't load modules (V10) | Phases 3–4 impossible | Phase 0 runs first and gates them |
| Message payloads leaking into prompts | Prompt injection via fleet messages | C3 / D5; reviewed in each PR touching `ailang-inbox` |
| Two implementations of the same gate drift (pi vs Claude) | Inconsistent behaviour | D3b shared cores with tests run by both adapters |
| Check-on-edit noise (fmt rewrites, slow checks) | Agents distracted, tool calls slower | Timeouts (C4); only diagnostics, never full output; A/B before fleet enablement |

## Related Documents

**Implemented (may inform design):**
- [M-DX-PI-HARNESS](../implemented/v0_35_0/m-dx-pi-harness.md) — pi extensions, Subprocess Contract, Distribution v2 (`ailang pi install`), the three-harness diffusion map this doc extends with a Claude Code mods column
- [M-CLAUDE-CODE-INTEGRATION](../implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION.md) / [hooks & inbox](../implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md) — the original shell hooks and user inbox (neural 0.46: same area, shell-hook era; this doc is the in-process successor)
- [agent-inbox-sessionstart-hook](../implemented/v0_3_14/agent-inbox-sessionstart-hook.md) — what `ailang-inbox` replaces visually
- [m-http-hooks-cloud-telemetry](../implemented/v0_9_0/m-http-hooks-cloud-telemetry-sprint-plan.md) — the telemetry `coordinator_hook.sh` carries; Phase 4 keeps its payloads
- [M-AILANG-LSP-FOR-AI](../implemented/v0_20_0/m-ailang-lsp-for-ai.md) — `ailang-lsp` plugin; the lens is its human-facing complement

**Planned (check for overlap):**
- [M-GIT-GUARDRAILS-HOOKLESS-EXECUTORS](m-git-guardrails-hookless-executors.md) — why gate mods are not controls (C1)
- [M-MESSAGE-PLANE-TRUST](m-message-plane-trust.md) — owner of completion reporting; source of D5
- [M-AILANG-NATIVE-HARNESS](m-ailang-native-harness.md) — Motoko-side harness; no overlap in mechanism

## References

- claude.dev blog: "Getting started with Claude Code mods"
- Claude Code 2.1.287 `claude-code.d.ts` (written by the engine beside each loaded mod)
- `ailang_bootstrap` PR #11 — `plugins/ailang-lens`

## Future Work

- `ailang claude install` once the API is stable (D2b).
- Lens v2: package-level view (`ailang check --package`), call graph once `iface` or `lsp` exposes one.
- Motoko diffusion: the same inbox band in Motoko's TUI.

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02
