# M-WORLD-OS-APPLICATIONS: the applications layer for Ailang World — Unix apps over files, for an OS whose files are agent traces

**Status**: Planned — investigation / program-level design doc (no single sprint; routes decisions to Mark, execution to per-seed sprints)
**Target**: cross-repo — primary consumer `sunholo-data/ailang-world`; AILANG-side work routes through the existing lanes (serve-api/MCP surface, `std/*`, registry packages). **No AILANG core/stdlib change is proposed by this doc.**
**Priority**: P1
**Estimated**: Phase 0 audit ~1 day; Phase 1 contract ~3 days; Phases 2–4 sized after D1–D6 (below)
**Dependencies**: Ailang World M1 kernel, M2 daemon `ailang-worldd`, M3 effect broker (LANDED, per v1-mission records); `w-mcp-projection` (world item 5, blocked on `serveapi/protocol` weight — #764, discharged by the v0.33.2 tag per ledger `D-34`); `w-agent-floor-m4` (world item 6, PARKED — this doc is the proposal for what sits on top of that floor)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Application logic over traces inherits the trace's determinism guarantees; seed probe (V12) is pure AILANG over stable stdlib |
| A2: Replayability | +1 | The core application operation on a trace is replay (`ailang replay` exists); replayable apps are a new property no desktop OS has |
| A3: Effect Legibility | +1 | Every proposed app surface (serve-api REST, MCP tools, A2UI surfaces) is a projection of typed, effect-annotated exports; no hidden side effects |
| A4: Explicit Authority | +1 | Launch = capability grant at spawn time (`--caps`, `--process-allowlist` path-pinned); external apps (Blender/Godot) enter only through named process grants |
| A5: Bounded Verification | 0 | No language-verification surface change |
| A6: Safe Concurrency | 0 | No concurrency change; tasks are daemon-scheduled |
| A7: Machines First | +1 | Applications are AILANG-authored (machine-first authorship); the human sees the A2UI/trace projection, not hand-written UI code. This doc is deliberately the one layer where *human interaction* is the goal — but the authoring stays machine-first |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Traces carry cost/usage (chains stats, budget_delta events); an application over traces can always show what it cost |
| A10: Composability | +1 | The pipe equivalent (task A's artifacts feed task B) composes via the existing message/task plane, not bespoke glue |
| A11: Structured Failure | +1 | Task status is typed (`m-task-status-truth`, run-policy unforgeable result lines) |
| A12: System Boundary | +1 | External applications cross the boundary only via pinned process allowlists and trace contracts, never ambient access |

**Net Score: +8** → **Decision: proceed to Phase 0/1; D1–D6 below are Mark's, not the agent's**

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (replay gate stays the contract)
- [x] A3: no hidden side effects (surfaces project declared exports only)
- [x] A4: no ambient access (launch-time capability grants only)
- [x] A7: human convenience is the *product*, not the *authoring* model

## Problem Statement

**Origin**: Mark (2026-11-28, Daneel directive task-988c91fa): *"an operating system isn't just files... It's the applications that use those files. We need an equivalent for AILANG world os... How do we actually build on top of the AILANG world agent traces the same way Unix builds on files to offer applications."*

**Current state (verified in this repo; world-repo facts are from cross-references, not a first-party audit — see Phase 0):**

- **Ailang World has the "files" and most of the "kernel".** Per the v1-mission ledger: world items 2–4 are LANDED (M1 kernel, M2 daemon `ailang-worldd`, M3 effect broker); item 5 `w-mcp-projection` was BLOCKED on `serveapi/protocol` (#764), which shipped as a plain package at **v0.33.2** (ledger `D-34`); item 6 `w-agent-floor-m4` is PARKED pending the value gate. World is ~**100k lines** of frontier-agent-authored AILANG (m-eval-finetuning-data-pipeline G1 census).
- **The AILANG side already has every ingredient of an application surface, but nothing names them as one thing.** Verified CLI surface: `ailang init web-app` (AILANG API backend + React frontend), `ailang ops serve-api` (serve AILANG exports as REST endpoints; `serveapi/protocol` is the seam World imports in 13 files per ledger `D-67`), `ailang ops mcp` (two production MCP services: **AILANG Parse** at `docparse.ailang.sunholo.com/mcp/connect/`, the **AILANG executor** at `executor.ailang.sunholo.com/mcp/connect/`), `ailang serve` (Observatory dashboard, port 1957), `ailang chains view/diff/journey/live` (trace viewers), `ailang replay` / `ailang dev export-training` (trace consumers), `ailang run --caps ... --process-allowlist git:status,...` (capability- and path-pinned process launching).
- **The trace format is documented as a de-facto contract.** `docs/docs/guides/traces.md` specifies the program-trace JSONL event schema (`module_start`, `function_enter/exit`, `effect`, `contract_check`, `budget_delta`, `error`, `module_end`), replay semantics, and the two-level model: **program traces** (JSONL files) vs **agent traces** (`ailang chains`, observatory). An AILANG program that *consumes a trace like a file* compiles and runs today against stable stdlib only (probe V12 below).
- **Seed applications already exist and are the proof the layer is buildable**: AILANG Parse (31k lines, live MCP service), daneel (21k lines, the directive router that delivered this very task), docparse (13k), email-parse (12k), the Cognitive OS runtime (DOM/Msg/Cog/Trace effects, browser demo live at `ailang.sunholo.com/demos/cognitive-os-runtime/`), `sunholo/a2ui` (A2UI protocol emitter package), stapledons-godot (Godot owns rendering/input; the AILANG simulation runs as a child process over NDJSON stdin/stdout).

**Impact / why this gap matters:**

- The stated premise of the whole program is that **the human meets AI at the applications layer**. World's agent floor (M4) is the first moment *any agent* can use World; the applications layer is the first moment *a human* can. Neither has a design.
- Without a named application contract, every new app (chess engine, game, parse service, dashboard) re-decides its surface ad hoc — the fragmentation pattern this program treats as a defect everywhere else.
- Unix won not because files were clever but because **every application could assume one narrow interface**. The equivalent interface for World is currently implicit, scattered across `traces.md`, the chains schema, `serveapi/protocol`, and world's daemon.

## Goals

**Primary goal:** define the applications layer for Ailang World — a named, typed contract between World's agent tasks/traces and things a human uses — so that AILANG-native applications are as routine to build as Unix applications over files.

**Success metrics:**
1. One normative document (the "task record" contract) that any application can program against: task identity, status, trace pointer, artifacts, cost, replay verdict. Phase 1 deliverable.
2. A launcher contract in the World UI: launching an app = spawning a task with declared caps and entrypoint, surfaced as a trace. Phase 2 deliverable (world-repo sprint).
3. Three seed applications, one per archetype (§Archetypes), each landing end-to-end: a trace-consuming app (built in AILANG, probe already works), a hosted-external app (stapledons-godot pattern generalised to Blender), and an agent-trace-in-itself app (A2UI-based).
4. Each seed's surface projects from ONE descriptor to REST (serve-api) + MCP (`w-mcp-projection`) + A2UI — no bespoke wire formats per app.
5. Replay remains meaningful across the layer: a launched app task replays (`ailang replay`) or reports why not (external-app boundary), in the app's own trace.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: One descriptor → many projections. App surface = a single declared export set projected to REST (serve-api), MCP (`w-mcp-projection`), and A2UI — vs. per-app bespoke surfaces. The `m-mcp-exact-tool-surface` Lane B ask ("caller-supplied exact descriptors... nothing built-in unless the caller supplies it") is the same decision one level down | This IS the "file interface" of the analogy; get it wrong and every app re-sets it | **human (Mark)** | design | high |
| D2: A2UI v1.0 as the standard human-facing UI protocol for World applications (upgrade `sunholo/a2ui` off v0.9.1 per m-a2ui-v1-platform-adoption) vs. ad-hoc UIs | Determines whether app UIs are generated-from-traces (the AILANG-native property) or hand-built | **human (Mark)** | design | med |
| D3: External-app integration model for Blender-class tools: (a) trace-only (record tool interactions as task events, drive externally), (b) launch-and-drive (Process cap with path-pinned allowlist from within a World task), or (c) both, staged (a) first | "It's ok to use existing applications" — but each option has different authority exposure (A4) | **human (Mark)** | design | med |
| D4: First flagship seed application (candidates: AILANG Parse as the MCP-first app; the stapledons-godot/Blender asset pipeline; a World trace-workbench dashboard in AILANG) | Sequencing decides which archetype gets hardened first | **human (Mark)** | design | low |
| D5: Where the task-record contract lives: world repo DESIGN.md (World owns its "POSIX") vs. ailang repo docs (both repos consume it; world imports `serveapi/protocol` in 13 files) | Determines who maintains the contract and who reviews changes | **human (Mark)** | design | med |
| D6: "Everything is a task" unification — the World UI launcher spawns every application as a task with caps (Unix's "everything is a file"), vs. apps as a separate citizen class | If yes, the launcher is ~free (daemon already spawns tasks); if no, a second app-runtime must be invented | **human (Mark)** | design | high |

### Design Freeze

Before any seed sprint begins, these must be resolved:

- [ ] D1 (surface contract)
- [ ] D3 (external-app authority model)
- [ ] D6 (launcher unification)

(D2, D4, D5 may ride one phase behind; they gate their own phases, not Phase 1.)

## Solution Design

### Overview

Do NOT invent an application runtime. World already has the kernel, daemon, effect broker, task plane, and trace store; AILANG already has the serving surfaces, the trace schema, the replay gate, and five real applications proving the archetypes. What is missing is **the contract layer that makes these one thing**: a normative task-record schema (the "file format"), a launcher (the "shell"), and named archetypes (the "application classes"). This doc proposes exactly that, plus the seed plan that exercises it.

### The analogy, made precise

| Unix primitive | Ailang World equivalent | Status today (verified where cited) |
|---|---|---|
| file | **task + its trace** (agent trace, `ailang chains`; program trace JSONL, `traces.md`) | exists, two levels documented |
| file format | **trace event schema** (`module_start`…`module_end`, `traces.md` table) | documented for program traces; agent-trace/observatory schema not yet normative — Phase 1 |
| `open/read/write` | query/extend/replay a task (`ailang chains view/tree/diff/journey`, `ailang replay`, coordinator) | exists as CLI, not as a programmatic contract |
| pipe | task A's artifacts → task B's prompt (chains → stages; message plane; `ailang ops exec`) | exists (chains), unbranded as composition primitive |
| `exec` | daemon task spawn with caps (`ailang-worldd` M2; `ailang run --caps`, `--policy`) | exists |
| shell | **the World UI launcher** (the thing Mark's message asks to launch from) | missing — Phase 2 |
| `/proc` | `ailang chains live/active` | exists |
| `mount` (external device) | external app via pinned Process cap (`--process-allowlist`, path-pinned; stapledons-godot NDJSON child-process pattern) | exists as mechanism (V7), unused as an *app* pattern — Phase 3 |
| package manager | `ailang pkg` + registry (`sunholo/*` packages) | exists |
| `stdin/stdout` | NDJSON over stdio (stapledons-godot: AILANG sim as child of Godot, JSONL trace on stdout) | proven in one repo |

**Where the analogy breaks — and the breaks are the product:**

1. A trace is **typed and semantic**, not a byte array. Applications don't `cat` it; they *query* it (`chains journey`), *score* it (`export-training --score`), *replay* it (`ailang replay`), and *diff* it (`chains diff`). Every one of those already exists as a CLI — the investigation's headline finding is that the "coreutils of traces" largely exist but are unbranded as an application surface.
2. A trace can be **live** (task still running). Unix files are closed objects; World "files" need subscribe (`chains live`, `--spans`). This is an application capability Unix never had.
3. A trace can be **conversed with**: the same record that an app displays can be resumed/extended by an agent (A2UI's bidirectional function RPC; `ailang ops exec`). "Editing a file" becomes "sending the task a follow-up".
4. **Composition is not concatenation**: pipes compose byte streams; World composes *typed artifacts + prompts*, so the composition contract is the task-record schema (Phase 1), not a delimiter.

### The application archetypes (each with today's seed)

**A1 — Trace-consuming applications** (read the "files"): dashboards, viewers, replay/verify UIs, training-data exporters, eval analytics. *Seeds:* Observatory dashboard (`ailang serve`), `ailang chains view/diff/journey`, `ailang export-training`, and this session's probe (V12: a pure-AILANG trace summariser — see Examples).

**A2 — Task-composing applications** (the "shell scripts"): launch, chain, and supervise agents over the task plane. *Seeds:* the mission loop itself, the coordinator, **daneel** — which routed this very directive — and `ailang ops exec`. These are the Unix shell-script equivalents and they are already the most production-hardened AILANG applications in the fleet.

**A3 — Applications with agent traces inside themselves** (the app IS a trace): the UI is generated by an agent mid-flight. *Seeds:* the Cognitive OS runtime (DOM/Msg/Cog/Trace effects; browser demo live), `sunholo/a2ui` (A2UI envelope emitter, consumed by `sunholo/ailang_parse`), the Aitana platform's A2UI surfaces. This archetype is Mark's "perhaps involve ai agent traces in themselves" — it is the most AILANG-native: the application's runtime state is a replayable trace.

**A4 — Hosted external applications** (existing apps, mounted): Godot (stapledons-godot: engine owns rendering/input, AILANG owns logic, NDJSON bridge, `--strict-bytecode` gating), Blender (per Mark's report, the godot traces show Blender interactions producing 3D assets — D3 decides whether such interactions stay trace-recorded or become launch-and-drive), plus any CLI tool reachable through the pinned process allowlist. This archetype is where a human most directly *sees* AI using real software.

### What an AILANG-world-native application is (proposed definition)

An application is an AILANG package (`ailang.toml`) that additionally declares:

1. **Surfaces** — which exports are exposed, projected from one descriptor (D1) to REST (`serve-api`), MCP tools (`w-mcp-projection`), and/or A2UI surfaces (D2).
2. **Caps** — the effect set + process allowlist it needs at launch (A4 explicit authority; already the `--caps`/`--process-allowlist` model).
3. **Task record participation** — it runs as a World task (D6): its trace lands in the task store, queryable/replayable like every other task.

Nothing here is new machinery — `ailang.toml` already has `[exports]`/`[effects]` blocks, serve-api already serves exports, `w-mcp-projection` already projects them, the registry already ships such packages. The definition is deliberately a **branding of existing seams** so the next app doesn't re-decide them.

### The contract that makes it Unix: the task record (Phase 1 deliverable)

`traces.md` is the de-facto "file format" for *program* traces. Phase 1 writes its sibling for *agent tasks* — the normative task-record schema a World UI, a dashboard, a re-prompter, and a training exporter can all assume: identity (task/chain/trace ids), typed status (`m-task-status-truth`), trace pointer + event envelope, artifacts (diff pointers), cost (chains stats fields), provenance (parent task, directive source), replay verdict. Where D5 says it lives is Mark's; the content is not novel — it consolidates what `ailang chains`, the coordinator, and the observatory already store, so applications stop coupling to those internals.

### Implementation Plan

**Phase 0: World-side audit** (~1 day, world repo)
- [ ] First-party audit of ailang-world DESIGN.md items 1–6, the daemon's task spawn surface, and what its UI currently launches. (This doc's world-repo claims are all cross-referenced, not audited — every such claim is marked in the Verification Log.)
- [ ] Map every existing world "application-like" artifact against the four archetypes; anything that fits none is scope-creep evidence against the archetype set.

**Phase 1: task-record contract** (~3 days, location per D5)
- [ ] Write the task-record schema doc (the "POSIX.1 of World"), consolidating chains/coordinator/observatory fields; version it like the trace schema (`"version":"1.0"` precedent in every JSONL line).
- [ ] Ship one programmatic reader of the contract: the probe below promoted to `examples/`/a package, extended from program traces to chain records.
- [ ] Contract test: an AILANG program reading a task record and printing a summary must run on evaluator + `--bytecode` both (engine parity rule).

**Phase 2: launcher contract** (world-repo sprint, gated on D6)
- [ ] World UI "launch app" = spawn task with the package's declared caps + entrypoint; the launch itself is a task with a trace (everything-is-a-task).
- [ ] Launch surface projection per D1; app card auto-generated from the descriptor (machines-first: the card is derived, not hand-authored).

**Phase 3: seed applications** (sized per seed after D4)
- [ ] Seed 1 (A1): trace workbench — query/journey/diff/replay over World tasks, written in AILANG, served via serve-api, A2UI-surfaced.
- [ ] Seed 2 (A4): Blender asset pipeline on the stapledons-godot pattern — AILANG orchestrator task, tool interactions recorded as trace events (D3 option a), artifacts as task outputs; launch-and-drive (option b) only if D3 rules it in.
- [ ] Seed 3 (A3): an A2UI v1.0 application whose entire UI is agent-generated during the task (upgrades `sunholo/a2ui` per m-a2ui-v1-platform-adoption; that doc's migration work is the dependency, not duplicated here).

**Phase 4: distribution** (~2 days)
- [ ] `ailang init world-app` scaffold (extends the existing `init web-app`); docs page naming the archetypes; registry listing for seeds 1–3.

### Files to Modify/Create

**New (Phase 1, location per D5):**
- `docs/.../task-record.md` (or world-repo equivalent) — the contract, ~200 LOC of prose + schema tables
- `examples/world_apps/trace_summary.ail` — the promoted probe, ~60 LOC (already written and verified this session)

**Modified:**
- Phase 2: world repo UI + daemon launch path (world-repo sprint; sized there)
- Phase 4: `ailang init` scaffold table (`cmd/ailang/init.go` region, +1 type), docs site

No `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects` or `cmd/ailang/exec.go` changes → **Conflict Surface analysis not required** (per skill gate: this doc touches none of those directories; the code-facing items are CLI scaffold + docs only).

## Examples

### Example 1: an application over a trace, today (probe V12, run this session)

A trace is produced by any run, then consumed by a pure-AILANG application exactly the way a Unix tool consumes a file:

```bash
ailang run --caps Trace,IO --emit-trace jsonl tracedemo.ail > run_trace.jsonl   # the "file"
ailang run --caps FS,IO tracesummary.ail                                        # the "application"
```

`tracesummary.ail` (full source in this session's transcript; ~50 LOC, `ailang check` clean) reads the JSONL, decodes each line with `std/json`, aggregates event counts with `std/map`, and prints:

```
events: 4
by name:
  effect: 2
  module_end: 1
  module_start: 1
```

That is `wc`-over-a-trace: the archetype-A1 application is *already writable in AILANG against the documented trace schema and stable stdlib only* — no new runtime, no new language surface. Full source, preserved so nothing depends on a session log:

```ailang
-- tracesummary: a minimal "application over an agent trace" probe.
-- Reads an AILANG JSONL execution trace and prints a summary:
--   total events, events by name. Unix `wc`-over-a-file, applied to a trace.

module tracesummary
import std/fs (readFile)
import std/json (decode, get, JString)
import std/io (println)
import std/list (foldl, filter, length)
import std/string (split)
import std/map as M

func eventName(j: Json) -> string = {
  match get(j, "event") {
    Some(JString(s)) => s,
    _ => "unknown"
  }
}

func summarize(m: Map[string,int], j: Json) -> Map[string,int] = {
  M.update(m, eventName(j), 1, \v. v + 1)
}

func printCounts(pairs: list[(string, int)]) -> () ! {IO} = {
  match pairs {
    [] => (),
    (name, n) :: rest => { println("  ${name}: ${n}"); printCounts(rest) }
  }
}

export func main() -> () ! {FS, IO} = {
  let raw = readFile("run_trace.jsonl");
  let lines = filter(\l. l != "", split(raw, "\n"));
  let decoded = foldl(\acc l. match decode(l) { Ok(j) => j :: acc, Err(_) => acc }, [], lines);
  let counts = foldl(summarize, M.empty(), decoded);
  println("events: ${length(decoded)}");
  println("by name:");
  printCounts(M.toList(counts))
}
```

### Example 2: mounting an existing application (archetype A4, proven in stapledons-godot)

Godot hosts rendering/input; the AILANG simulation is a child process (`ailang run --bytecode`) speaking NDJSON over stdin/stdout, its Makefile gating modules under `--strict-bytecode`. The Blender interactions Mark cites ride the same shape: tool calls recorded as trace events, artifacts as task outputs — the trace becomes the reviewable "session file" a human (or a game designer agent) can replay and extend.

### Example 3: launching from the World UI (Phase 2 target shape)

```
World UI app card (auto-derived from the package descriptor)
  └─ launch = spawn task {package: sunholo/parse, caps: FS,Net, entry: convert}
       ├─ trace lands in the task store (queryable: chains view/diff/journey)
       ├─ surface live at serve-api/MCP projection while it runs (A2UI for humans)
       └─ completion artifact + cost recorded in the task record (Phase 1 schema)
```

## Success Criteria

- [ ] Phase 0 audit published (world-repo facts first-party, every cross-reference in this doc either confirmed or corrected)
- [ ] Task-record contract doc exists, versioned, and one AILANG reader passes `ailang check` + runs on evaluator and `--bytecode` (engine parity)
- [ ] Launcher spawns a seed app as a task with declared caps; the launch is itself in the trace store
- [ ] Three seeds land, one per archetype, each projecting from one descriptor to at least two surfaces
- [ ] The probe example (`trace_summary.ail`) is in-repo and in the docs corpus
- [ ] All tests passing; documentation updated; no new core/stdlib surface without its own design doc

## Testing Strategy

**Unit tests:** the promoted trace-summary package gets inline tests (event counting, malformed-line tolerance) following `examples/inline_tests_*` conventions.

**Integration tests:** Phase 2 launcher: spawn→trace→query round-trip in the world repo's test suite.

**Manual testing:** seed 2 end-to-end on a real Blender/Godot asset (Mark's stapledon scenario is the acceptance scenario); seed 3 A2UI surface in a browser.

## Deferred Decisions

- A2UI catalog composition per app — agent may choose (per m-a2ui-v1-platform-adoption's mapping table)
- Which chains/observatory fields enter the v1 task record beyond the minimum — implementer may choose, additive versioning
- Naming: "world app" vs "task app" vs plain "application" — agent may choose in docs pass

## Non-Goals

- **No new AILANG core/stdlib/language surface.** Every capability cited already ships; this doc only names and contracts them.
- **No re-architecture of World's kernel/daemon** — the application layer rides the landed M1–M3 substrate.
- **Not a general "agent app store"** — distribution stays the existing registry; only a scaffold + docs page are added.
- **No bespoke UI framework** — A2UI (per D2) or serve-api projections; the Cognitive OS DOM path stays for in-tab apps.
- **Not** the v1.1 game-engine-effects doc (that remains the game's own design; this doc only cites its architecture as the A4 seed pattern).

## Timeline

| Phase | Effort | Gate |
|---|---|---|
| Phase 0 audit | ~1 day | none (read-only) |
| Phase 1 contract | ~3 days | D5 |
| Phase 2 launcher | world sprint (~1 wk) | D1, D6 |
| Phase 3 seeds | ~1–2 wk per seed | D3, D4 |
| Phase 4 distribution | ~2 days | Phase 3 |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| World-repo facts are cross-referenced, not audited (repo not in this workspace) | Med | Phase 0 is the first deliverable; every world claim is flagged in the Verification Log |
| The analogy overreaches (traces ≠ files; apps expect query/subscribe, not read) | Med | The contract (Phase 1) defines the operations apps may assume; the four archetypes only use operations that exist today |
| D1 one-descriptor projection could duplicate `m-mcp-exact-tool-surface` Lane B | Med | This doc treats Lane B as the dependency decision, not a competing design; D1 routes to that lane if Mark ratifies |
| A2UI v0.9.1→v1.0 migration slips (m-a2ui doc is itself pending) | Low | Seed 3 is gated on that doc's migration, explicitly not duplicated |
| "Applications" scope-creeps into a UI framework project | High | Non-Goals forbid it; A2UI/Cognitive-OS/DOM are the only sanctioned UI paths |

## Related Documents

Auto-search on "world os applications" returned no doc above the noise threshold (top matches were token coincidences: `m-verify-stdlib-stale-path`, `m-comments-for-ai-authors`) — the duplicate/coverage gate passes; no existing doc covers the applications layer. The genuinely related docs, found by directed reading:

- [m-wasm-reflective-runtime](../implemented/v0_21_0/m-wasm-reflective-runtime.md) — the Cognitive OS umbrella; its OS-primitive table is the in-language sibling of this doc's world-level table
- [m-cog-runtime](../implemented/v0_21_0/m-cog-runtime.md) — DOM/Msg/Cog/Trace effects + event-log replay (archetype A3 substrate)
- [m-a2ui-v1-platform-adoption](m-a2ui-v1-platform-adoption.md) — D2's dependency: A2UI v1.0 migration for the platform + `sunholo/a2ui`
- [m-game-engine-effects](v1_1_0/m-game-engine-effects.md) — reality-checked architecture of stapledons-godot (archetype A4 seed)
- [m-chess-engine](v0_64_2/m-chess-engine.md) — a pending AILANG-native application; candidate seed
- [m-connector-channels-marketplaces](m-connector-channels-marketplaces.md) — AILANG Parse + executor as live MCP services; distribution reach
- [m-mcp-exact-tool-surface-lane-b](../implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md) — the caller-supplied-descriptor ask that D1 generalises
- [m-task-status-truth](m-task-status-truth.md) — typed task status feeding the Phase 1 contract
- [m-eval-finetuning-data-pipeline](v1_1_0/m-eval-finetuning-data-pipeline.md) — the corpus census (world 100k, ailang-parse 31k, daneel 21k...) cited above
- [docs/docs/guides/traces.md](../../docs/docs/guides/traces.md) — the program-trace schema; Phase 1's sibling document
- [m-ailang-native-harness](m-ailang-native-harness.md) — the machine-convenience doctrine this layer extends to human-facing surfaces

## References

- Mark's directive (2026-11-28, this task) — problem statement and the stapledons/Blender observation
- [Design Axioms](/docs/references/axioms)
- v1-mission ledger rows D-34/D-35/D-67 and the `#764` chain (world M1–M3 landed, serveapi/protocol delivery)

## Verification Log

| # | Claim | Verification (command + result, run 2026-11-28 in this workspace) |
|---|---|---|
| V1 | No existing design doc covers the world-applications layer | create-script search on "world os applications": all matches were token coincidences (m-verify-stdlib-stale-path 1.00 etc.); directed reading found the Related Documents set above — none covers this layer |
| V2 | World items 2–4 LANDED (M1 kernel, M2 daemon `ailang-worldd`, M3 effect broker); item 5 `w-mcp-projection` blocked on `#764`; item 6 `w-agent-floor-m4` PARKED | `grep` in design_docs/v1-mission.md:1920 ("World items 2, 3 and 4 are LANDED... item 5 is the only one of 2–5 still open"); **not first-party for the world repo** — Phase 0 audits |
| V3 | `serveapi/protocol` shipped at v0.33.2, unblocking `#764` | v1-mission ledger `D-34` row: "serveapi/protocol ships at v0.33.2... tag published 2026-08-24"; world consumes via pinned releases |
| V4 | World imports `serveapi/protocol` in 13 files | ledger `D-67` row 10 (v1-mission.md:136); **cross-reference, Phase 0 confirms** |
| V5 | `ailang init web-app` scaffolds AILANG API backend + React frontend | `ailang init --help`: types `web-app`, `package` (live CLI output quoted in Problem Statement) |
| V6 | `ailang ops serve-api` serves AILANG exports as REST; `ailang serve` starts the Observatory dashboard (1957); `ailang ops mcp` exists | `ailang ops --help` and `ailang --help`, live output |
| V7 | Process launching is capability- and path-pinned | `ailang run --help`: `--process-allowlist <list>` "...only these binaries may be exec'd (path-pinned at startup); cmd:sub narrows to a subcommand chain (git:status,gh:pr:list)" |
| V8 | Two production MCP services exist (Parse at docparse..., executor at executor.ailang.sunholo.com) | m-connector-channels-marketplaces.md Problem Statement table (docparse v0.46.0, AILANG v0.57.4, dated 2026-11-10) |
| V9 | Trace event schema is documented (8 event types; replay; scoring) | docs/docs/guides/traces.md, read in full (132 lines): event table + `ailang replay` exit codes + `export-training` weights |
| V10 | `ailang replay` exists and auto-resolves module + caps | `ailang replay --help`, live: flags `-caps`, `-entry`, `-file`, `-json` |
| V11 | Cognitive OS runtime shipped; demo live; DOM/Msg/Cog/Trace caps exist | design_docs/implemented/v0_21_0/m-wasm-reflective-runtime.md (demo URL quoted) + `ailang run --help` caps list "...Secret,Trace,DOM,Msg,Cog"; `ailang check examples/cognitive_os/single_agent_replay.ail` → "✓ No errors found!" |
| V12 | An AILANG application can consume a trace like a file, today, stable stdlib only | **Probe run this session**: `ailang run --caps Trace,IO --emit-trace jsonl tracedemo.ail > run_trace.jsonl` (4 JSONL events, schema exactly as traces.md documents); `ailang check tracesummary.ail` → clean; `ailang run --caps FS,IO tracesummary.ail` → the output quoted in Example 1. Uses std/fs, std/json, std/list, std/map, std/string only |
| V13 | stapledons-godot: Godot owns rendering/input, AILANG sim is a child over NDJSON stdio, `--strict-bytecode` gating; Go port dead since 2026-03-16 | m-game-engine-effects.md "Reality check (2026-10-28)" block, read verbatim; **repo not in this workspace** |
| V14 | Blender interactions appear in stapledon traces | **Operator report only (Mark's directive)** — not verifiable from this repo; Phase 0/seed-2 confirms first-party |
| V15 | Corpus sizes (world 100k, ailang-parse 31k, packages 22k, daneel 21k...) | m-eval-finetuning-data-pipeline.md G1 table (v1_1_0), quoted verbatim |
| V16 | `sunholo/a2ui` package emits v0.9.1 envelopes; consumed by `sunholo/ailang_parse`; v1.0 migration planned | m-a2ui-v1-platform-adoption.md header + Problem Statement (registry v0.2.0, repo HEAD 0.3.0) |
| V17 | Task status is typed / result line unforgeable (feeds Phase 1) | m-task-status-truth.md (planned/) and design_docs/implemented/v0_53_0/m-run-policy-result-unforgeable.md exist as named |
| V18 | AILANG programs are "everything is an expression, no loops"; probe written per current teaching prompt | `ailang prompt` read before authoring; probe compiled first try modulo documented syntax facts (lambda `\a b.`, `::` cons, `${}` interpolation), all corrected from `ailang check` diagnostics — the iteration transcript is itself a small demonstration of the machine-first authoring loop |

## Future Work

- A "coreutils of traces" package: canonical AILANG implementations of `chains view/journey/diff` as library functions over the Phase 1 contract, so World apps compose them instead of shelling out.
- App-to-app composition sugar on the task plane (named pipes of tasks).
- A2UI catalog for World app cards derived automatically from package descriptors.
- Evaluation lane: "build a World app" as an agentic benchmark (composes existing eval harness with the Phase 1 contract).

---

**Document created**: 2026-11-28
**Last updated**: 2026-11-28
**Quorum**: not run — this is an investigation doc whose load-bearing decisions (D1–D6) are Mark's by design; run `ailang design-quorum` on it if a seed sprint is planned directly from it.
