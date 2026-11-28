# M-WORLD-OS-APPLICATIONS: the applications layer for Ailang World — what runs on an OS whose files are typed world objects

**Status**: Planned — **r1 (2026-11-28)**: the World-side facts are now re-audited first-party in `sunholo-data/ailang-world` (local `9a40fe68`, origin `91befdab`; see W1–W16). r0's Phase 0 audit is done by this revision. The audit found that **World already owns, and has largely designed, its applications layer**. This doc is now the AILANG-side companion: it names the archetypes, maps each to World's landed or planned queue rows, records which of r0's D1–D6 World has already decided, and keeps only the AILANG-side work here.
**Target**: cross-repo. **World owns the applications layer** (World `DESIGN.md` §1, §11.2, milestone M6; `HUMAN-SURFACE.md` v0.3, ratified; World queue rows 151, 188, 243, 269, 272, 273; Human Decision Ledger `D-WORLD-64`, `-87`, `-92`, `-93`, `-94`). The parts that are still new route to World as a proposed queue row (274, `w-world-applications`). The AILANG side keeps the trace-reader example, a docs page and, later, the `ailang init` scaffold. **No AILANG core/stdlib change is proposed by this doc.**
**Priority**: P1
**Estimated**: AILANG side ~2 days (Phase 1), plus ~1 day for Phase 4 once World fixes the app manifest shape. World-side work is sized in World.
**Dependencies**: World v1.0.0-rc.1 (released 2026-10-16, "the 1.0 bar is met"). World row 243 V4 (browser start and finish of an episode; S9b landed 2026-11-28). World rows 151 and 188 (designed and approved). `m-a2ui-v1-platform-adoption` (only if `sunholo/a2ui` is to emit World's view catalog).

## Revision notes (r0 → r1)

r0 was written from the ailang repo without the World repo in the workspace. Its World claims came from the v1-mission ledger, and several were months out of date. The corrections:

| r0 said | First-party, 2026-11-28 | Evidence |
|---|---|---|
| World has "the files and most of the kernel"; agent floor (M4) PARKED; `w-mcp-projection` BLOCKED on `#764` | World 1.0's seven-clause bar is met and `v1.0.0-rc.1` is released. MCP dispatch (row 108), A2A card and AG-UI stream (row 148) have all landed | W1, W5 |
| The applications layer has no design ("Neither has a design") | World designed it: DESIGN §11.2 / M6 *generated projections*, `HUMAN-SURFACE.md` (ratified), row 188 `w-world-os-surface` (approved for build, D-WORLD-87), row 151 `w-agent-generated-projections` (D-WORLD-92/93) | W2, W9, W10 |
| file = task + its agent trace (`ailang chains`, observatory) | World's own analogy: *"Unix made everything a file. AILANG World makes everything a typed state transition."* Its "files" are content-addressed world objects, changed only by transitions, logged and replayable. World's Go code has **zero** references to `ailang chains`, the observatory or `ailang coordinator`: those are AILANG's task plane, not World's | W2, W4 |
| Launcher missing (Phase 2) | Agents run as **episodes**: `episode start --preset … --launch` starts a confined agent (Claude Code, codex, agy) in a worktree under typed grants, and `finish` runs checks and opens a PR. Browser start and finish are row 243 V4, in progress | W6, W7 |
| D2 (A2UI) open | Decided in World: AG-UI carries the events and A2UI is the payload (D-WORLD-64). Agent views are typed world objects over World's closed catalog (row 151) | W9 |
| World ≈ 100k lines of AILANG | 7,370 lines of tracked `.ail`, 90,193 lines of Go (non-test), 174,487 lines of Go tests | W12 |
| `serveapi/protocol` in 13 World files | 14 files; `go.mod` pins `github.com/sunholo-data/ailang v0.47.2` | W13 |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | World applications are transitions over a content-addressed store with proven deterministic replay (World bar clause 1). The AILANG-side probe (V12) is pure AILANG over stable stdlib |
| A2: Replayability | +1 | Every app action commits a log entry. `why` walks any result back through log entry, plan and effect records. Program traces replay via `ailang replay` |
| A3: Effect Legibility | +1 | Every app surface is a projection of typed, effect-annotated transitions (registry → MCP/A2A; views → AG-UI/A2UI). No hidden side effects |
| A4: Explicit Authority | +1 | Launch = an episode under typed grants. External apps enter only through broker effects or declared grants (row 272 pattern) |
| A5: Bounded Verification | 0 | No language-verification surface change |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Apps are AILANG packages and agent-authored views; the human sees projections, not hand-written UI per app |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Episodes carry budgets and receipts; an app over them can always show what it cost |
| A10: Composability | +1 | Episode outputs feed further episodes (row 188 "change from here"); typed messages are graph edges (World DESIGN §14, row 273) |
| A11: Structured Failure | +1 | Typed refusals, effect receipts (`indeterminate` fails closed) and typed task status |
| A12: System Boundary | +1 | External applications cross the boundary only via the broker, confinement and pinned grants |

**Net Score: +8** → **Decision: proceed with the AILANG-side Phase 1. The World-side work routes to the World queue. The decisions still open are Mark's.**

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (replay stays the contract)
- [x] A3: no hidden side effects (surfaces project declared transitions and views only)
- [x] A4: no ambient access (launch-time grants only)
- [x] A7: human convenience is the *product*, not the *authoring* model

## Problem Statement

**Origin**: Mark (2026-11-28, Daneel directive task-988c91fa): *"an operating system isn't just files... It's the applications that use those files. We need an equivalent for AILANG world os... How do we actually build on top of the AILANG world agent traces the same way Unix builds on files to offer applications."*

**Current state of World (first-party, 2026-11-28):**

- **World is a released semantic OS, not just a kernel.** `v1.0.0-rc.1` (2026-10-16) meets all seven bar clauses: deterministic kernel with bit-for-bit replay, local-first daemon `ailang-worldd`, an effect broker with explicit authority and a physical isolation floor, a non-inferiority floor for resident agents, the human surface with provenance walks, MCP and A2A at the boundary, and self-modification through extension packages (W1).
- **Its primitives are already the "files" and "syscalls".** The daemon serves objects (`GET /v1/objects/{ref}`), worlds (`/v1/worlds/{ref}`), the append-only log (`/v1/log`), effect receipts (`/v1/receipts/{id}`), the transition registry (`/v1/registry/{name}`), commits (`POST /v1/commit`) and episode reports and finish (`/v1/episodes/{episode}/…`) (W3).
- **Agent work is the episode.** `episode start --preset <p> --launch` starts a confined agent in its own worktree under typed grants. Every tool call commits an entry. `finish` re-runs the checks and opens a PR. `why` verifies the chain from a result back to its cause. Hosts: Claude Code, codex and agy (W6). The herdr plugin `plugins/ailang-world` launches World agents from the terminal multiplexer Mark works in (W8). Mark's standing goal is that **all** his agent work runs through World (D-WORLD-94, W8).
- **Its boundary already speaks the open protocols.** MCP at `POST /mcp/`, A2A at `/.well-known/agent.json` + `POST /a2a/`, and AG-UI at `POST /agui/` (W5). Tools come from **extension packages**: a package declares its transitions in `transitions.json` (id, title, description, AILANG `transitionFnFile`, JSON `inputSchema`). The registry serves them as MCP tools, filtered by the session's capabilities. `packages/se-tools` ships ten coding tools this way (W4).
- **It has a human surface, and the applications layer is designed in World.** `/surface` is the operator app: address-bar verbs, Needs-you, approve/decide/confirm with passkey or SSH signature, repos, presets and profiles, conversation, trajectory and stack. `/workbench` is the inspector (W3). Row 243 V4 moves episode start and finish into the browser (S9b landed 2026-11-28, W7). Row 188 `w-world-os-surface` (approved for build, D-WORLD-87) is Mark's "think operating system" surface: agents working on tasks as nodes you can play, replay, rewind and stitch together, with one episode's outputs becoming the next one's inputs (W10). Row 151 `w-agent-generated-projections` lets an agent show what it is doing as an A2UI view over a closed catalog when text isn't enough (W9). Rows 272 (skill-declared grants and host-mediated skill effects) and 273 (agent context and messages) are filed next after row 243 V4 (W11).
- **Two task planes exist, and they are different.** AILANG's coordinator, `ailang chains` and the observatory track the ailang repo's own agents (Daneel, sprint agents). World's episodes and log track work done *through World*. World does not read the AILANG plane (W2). r0 conflated them.

**Current state of the AILANG side (verified in this repo, unchanged from r0):** `ailang init web-app`, `ailang ops serve-api` (`serveapi/protocol` is the seam World imports), `ailang ops mcp` (the Parse and executor MCP services), `ailang serve` (observatory, port 1957), `ailang chains view/diff/journey/live`, `ailang replay`, `ailang dev export-training`, `ailang run --caps … --process-allowlist …`, and `docs/docs/guides/traces.md` (the program-trace JSONL schema). An AILANG program can consume a program trace like a file today (V12).

**Seed applications that exist today:** AILANG Parse (live MCP service), daneel (the directive router that delivered this task), docparse, email-parse, the Cognitive OS runtime (browser demo live), `sunholo/a2ui`, stapledons-godot (Godot owns rendering and input; the AILANG simulation is a child process over NDJSON stdio), and World's own `se-tools` package.

**What is actually missing (the gap this doc now addresses):**

1. **The word "application" as a packaged, launchable thing.** World has transitions (tools), presets (how to start an agent), grants (what it may do) and views (what it shows). Nothing ties them together as *one installable app* that appears in the surface and launches as an episode.
2. **Hosted external applications (A4).** Nothing in World's queue yet covers an agent driving Blender, Godot or similar through a declared grant, with the tool's results recorded as effects.
3. **An AILANG-side on-ramp.** An AILANG package author has no scaffold or docs page saying "this is how your package becomes a World app".
4. **One shared vocabulary across both repos**, so AILANG and World don't design the same layer twice. r0 was itself an instance of that risk.

## Goals

**Primary goal:** name the applications layer once, across both repos, and route each piece to its owner. World owns the layer and its contracts. AILANG supplies the language, the packages and the on-ramp.

**Success metrics:**
1. World's queue carries the missing pieces (gaps 1 and 2) as one row with a World design doc, and their decisions as `D-WORLD` asks.
2. An AILANG docs page names the four archetypes and points to World for the app contract.
3. The trace-reader probe (V12) is in `examples/`, with inline tests, and runs on the evaluator and `--bytecode` (engine parity).
4. After World fixes the app manifest shape, `ailang init world-app` scaffolds a package that World can install and launch.
5. No AILANG-side design re-decides anything World has already ratified.

## High-Impact Decisions

r0 posed D1–D6 as open. The audit found most of them already answered in World. Each remaining decision belongs to Mark and is recorded in World's ledger.

| Decision | Status after the World audit | Remaining question | Owner |
|---|---|---|---|
| D1: one descriptor → many projections | **Mostly answered.** The transition registry is the descriptor: one `transitions.json` per package becomes MCP tools and A2A skills, filtered by capability (clause 6, rows 107/108) | Should an app's launch preset and views come from the same package descriptor? | World (row 274) |
| D2: A2UI as the human-facing UI protocol | **Answered**: AG-UI carries events and A2UI is the payload (D-WORLD-64). Views are typed world objects over World's closed catalog (row 151, D-WORLD-92/93) | AILANG side only: should `sunholo/a2ui` v1.0 emit World's catalog? Belongs to `m-a2ui-v1-platform-adoption` | ailang (that doc) |
| D3: authority model for external apps (Blender-class) | **Mechanism exists**: broker effects with receipts, confined episodes, and row 272's skill-declared grants with host-mediated effects (`Host.TailnetPreview` is its first slice) | (a) record the tool's work as effects only, (b) a host-mediated effect per tool, or (c) a pinned process grant inside the confined episode? | **Mark**, as a `D-WORLD` ask |
| D4: first flagship app | Open | Recommendation: an A4 Blender/Godot asset app for stapledons. It is the archetype World doesn't cover yet, and Mark's directive names it | **Mark** |
| D5: where the contract lives | **Recommendation: World.** The episode record (`/v1/episodes/{e}/report`), the log and the registry are World's product data. World's DESIGN §14 sends only language gaps to the ailang repo | Confirm. If AILANG's coordinator plane wants a task-record schema of its own, that is a separate, optional ailang doc | **Mark** |
| D6: "everything is a task" | **Answered in practice.** Every agent run is an episode (rows 217/243, D-WORLD-94), and every human action is a transition (World DESIGN §11) | Confirm that an app is *package + preset (+ views)*, launched as an episode, with no separate runtime | World (row 274) |

### Design Freeze

Before any World app sprint:
- [ ] D3 (external-app authority) — Mark
- [ ] D5 confirmed — Mark
- [ ] World row 274 design doc ratified (it carries D1's and D6's remaining questions)

The AILANG-side Phase 1 waits on none of these.

## Solution Design

### Overview

Do not invent an application runtime, and do not design World's layer from this repo. World already has the store, log, broker, confinement, episodes, registry, protocol boundary and human surface, and it has designed generated views and an OS-style surface. The remaining work is small and has clear owners:

- **World:** an *app* = an extension package that declares its transitions, a launch preset and optional views, shown in the surface and launched as an episode. Plus one external-app slice (A4). Routed as proposed World row 274.
- **AILANG:** the trace-reader example, a docs page naming the archetypes, and later a scaffold for the manifest World fixes.

### The analogy, mapped onto World's actual primitives

| Unix | Ailang World | Status (World, 2026-11-28) |
|---|---|---|
| file | typed, content-addressed **world object** (`/v1/objects/{ref}`); a world snapshot (`/v1/worlds/{ref}`) | LANDED |
| file format | AILANG types (`packages/world-core/world/types.ail`) and transition input schemas | LANDED |
| `write` | a **transition**: propose → verify (types + Z3 contracts) → commit (`POST /v1/commit`) | LANDED |
| `read` / `stat` | object, log and receipt reads; `why` provenance walk | LANDED |
| journal | append-only transition log (`/v1/log`) and effect receipts | LANDED |
| syscall table | transition registry served as MCP tools (`POST /mcp/`), A2A agent card | LANDED (rows 107/108) |
| process / `exec` | **episode**: a confined agent in a worktree under typed grants | LANDED (CLI, herdr); browser in progress (row 243 V4) |
| shell | `/surface` address-bar verbs, CLI, herdr plugin | LANDED; browser start/finish row 243 V4 |
| `/proc`, `tail -f` | AG-UI event stream (`POST /agui/`), follow status, Needs-you | LANDED (row 148) |
| pipe | one episode's outputs as the next one's inputs (row 188, "change from here"); typed messages as edges (DESIGN §14, row 273) | DESIGNED |
| GUI toolkit | A2UI views as typed world objects over a closed catalog | DESIGNED (row 151) |
| package manager | AILANG registry extension packages (`world-publish`; clause 7 self-modification) | LANDED for transitions |
| **application** | package + launch preset + views, launched as an episode | **GAP → row 274** |
| `mount` (external device) | external tool via a host-mediated effect or declared grant | **GAP → row 274 (A4 slice)** |
| (AILANG side) program trace | `traces.md` JSONL; `ailang chains` for the coordinator's agents | AILANG; a different plane |

**Where the analogy breaks, and why the breaks are the product:**

1. **The "file" is typed and semantic.** Applications don't `cat` it. They query it, verify it, replay it and ask it *why*.
2. **It can be live.** An episode is running while you watch it (AG-UI). Unix files are closed objects.
3. **It can be talked to.** The episode an app displays can be resumed or redirected by a person or an agent ("change from here"). Editing a file becomes sending the work a follow-up.
4. **Composition is typed.** Pipes join byte streams. World joins typed outputs and authority, so composition comes with its own provenance.

### The application archetypes, each mapped to its owner

| Archetype | What it is | Today | Owner |
|---|---|---|---|
| **A1 — over history** (reads the "files") | dashboards, viewers, replay and verify UIs, exporters | World: `/workbench`, `why`, row 188 timeline and branes. AILANG: observatory, `ailang chains`, the V12 probe over program traces | World for World history; ailang for program traces and the coordinator plane |
| **A2 — composing work** (the "shell scripts") | launch, chain and supervise agents | World: episodes and presets (rows 217, 243, 269), herdr plugin, messages (row 273). AILANG: coordinator, daneel, the mission loop | Both, on their own planes |
| **A3 — the app is an agent's view** | UI the agent generates mid-task to show its work | World row 151 (D-WORLD-92/93); Cognitive OS runtime; `sunholo/a2ui` | World (row 151) |
| **A4 — hosted external apps** | existing software (Blender, Godot, CLIs) driven by an agent under authority | stapledons-godot NDJSON pattern; no World row yet | **World, new: row 274** |

### What a World application is (proposal for World row 274 to ratify)

An AILANG extension package that declares:

1. **Transitions**: its `transitions.json`, which World's registry already serves as MCP tools.
2. **A launch preset**: agent host, grants, checks and lifetime, in the preset shape World already lists at `/surface/presets.json` (row 269), with grants declared as in row 272.
3. **Views** (optional): A2UI views over World's catalog (row 151).

Installing the app publishes the package. Launching it starts an episode from its preset. Its work lands in the log like any other episode, so it is replayable, `why`-walkable and budgeted. Most of this is existing machinery. The new parts are a small manifest that ties 1–3 together, and an app list on the surface.

### The record applications read

r0's Phase 1 proposed a new normative "task record". In World, the record already exists: the episode report (`/v1/episodes/{episode}/report`), the log entries and the receipts. Whether World versions that report as the public contract for apps is World row 274's question (D5). On the AILANG side, a task-record schema for the coordinator/`chains` plane is optional. It needs its own doc and is out of scope here.

### Implementation Plan

**Phase 0: World-side audit** — **DONE in r1** (W1–W16).

**Phase 1: AILANG-side on-ramp** (~2 days, this repo). This is the scope a sprint planned from this doc may take.
- [ ] Promote the V12 probe to `examples/trace_apps/trace_summary.ail`, with inline tests (event counting, malformed-line tolerance). Run it on the evaluator and `--bytecode`.
- [ ] Docs page: "Applications on AILANG and Ailang World". It names the four archetypes, shows the program-trace reader, and links to World for the app contract. It states that World episodes and AILANG coordinator tasks are different planes.

**Phase 2: World app contract and the A4 slice** (World repo; proposed queue row 274, after row 243 V4)
- [ ] World design doc: app = package + preset + views; the app list on `/surface`; D1/D6 remaining questions; D5.
- [ ] One A4 slice under Mark's D3 ruling: a Blender or Godot asset task as an episode, with the tool's work recorded as effects and the asset as an output.

**Phase 3: seeds**, which mostly ride existing World rows
- [ ] A1: row 188 `w-world-os-surface` (already approved).
- [ ] A3: row 151 `w-agent-generated-projections` (already approved).
- [ ] A4: row 274's slice (above).

**Phase 4: distribution** (~1 day, this repo, **after** row 274 fixes the manifest)
- [ ] `ailang init world-app`, extending the `init web-app` table. It scaffolds the manifest World ratified, not a shape this repo guesses.

### Files to Modify/Create

**New (Phase 1, this repo):**
- `examples/trace_apps/trace_summary.ail`: the promoted probe, ~60 LOC with tests
- `docs/docs/guides/applications.md`: ~150 lines

**Modified (Phase 4, this repo):**
- the `ailang init` scaffold table (`cmd/ailang/init.go` region, +1 type)

**World repo:** sized in World under row 274.

No `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects` or `cmd/ailang/exec.go` changes, so **Conflict Surface analysis is not required**.

## Examples

### Example 1: an application over a program trace, today (probe V12)

A run produces a trace, and a pure-AILANG application reads it the way a Unix tool reads a file:

```bash
ailang run --caps Trace,IO --emit-trace jsonl tracedemo.ail > run_trace.jsonl   # the "file"
ailang run --caps FS,IO tracesummary.ail                                        # the "application"
```

`tracesummary.ail` reads the JSONL, decodes each line with `std/json`, counts events with `std/map`, and prints:

```
events: 4
by name:
  effect: 2
  module_end: 1
  module_start: 1
```

That is `wc` over a trace: an archetype-A1 application written in AILANG against the documented schema and stable stdlib only. Full source:

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

### Example 2: a World tool package today (the seed of an app)

`packages/se-tools/transitions.json` in World declares each tool once:

```json
{ "id": "ailang-read", "title": "Read (sandboxed)",
  "transitionFnFile": "packages/se-tools/se_tools/read.ail",
  "inputSchema": { "type": "object", "properties": { "path": { "type": "string" } } } }
```

World's registry serves it as an MCP tool. Each call costs one `Workspace.Read` effect, commits one log entry, and can be walked back with `why`. An *app* adds a launch preset and optional views to the same package.

### Example 3: launching an app (row 274 target shape)

```
/surface app list (derived from installed packages)
  └─ launch "asset-pipeline" = episode start --preset asset-pipeline
       ├─ confined agent in a worktree, grants from the package (e.g. a Blender grant per D3)
       ├─ every tool call → log entry + receipt; live on AG-UI; Needs-you for approvals
       ├─ optional A2UI view showing progress (row 151)
       └─ finish → checks, outputs (the .glb asset) as episode outputs, report, PR
```

### Example 4: mounting an existing application (A4, proven in stapledons-godot)

Godot owns rendering and input. The AILANG simulation runs as a child process (`ailang run --bytecode`) speaking NDJSON over stdin/stdout. A Blender asset app would follow the same shape inside a World episode, with D3 deciding how the tool's actions are authorized and recorded.

## Success Criteria

- [x] Phase 0 audit published (r1; World facts first-party, r0's cross-references corrected)
- [ ] World row 274 filed, with a World design doc and D3/D5 asks
- [ ] `trace_summary.ail` in `examples/`, passing `ailang check` and inline tests on the evaluator and `--bytecode`
- [ ] Applications docs page published, linking World's contract
- [ ] `ailang init world-app` scaffolds World's ratified manifest (after row 274)
- [ ] No new core/stdlib surface without its own design doc

## Testing Strategy

**Unit tests:** inline tests for the trace-summary example (event counting, malformed-line tolerance), following `examples/inline_tests_*`.

**Integration tests:** World-side, under row 274 (install → launch → episode → report round trip).

**Manual testing:** the A4 slice end-to-end on a real Blender or Godot asset (Mark's stapledons scenario is the acceptance scenario).

## Deferred Decisions

- A2UI catalog composition per app: World row 151 owns the catalog
- Naming ("world app", "app", "application"): World row 274 decides; this repo's docs follow it

## Non-Goals

- **No new AILANG core, stdlib or language surface.**
- **Not a second design of World's human surface.** HUMAN-SURFACE.md, rows 151 and 188 govern it.
- **No re-architecture of World's kernel, daemon or episode model.**
- **Not a general agent app store.** Distribution stays the registry.
- **No bespoke UI framework.** A2UI over World's catalog, or the Cognitive OS DOM path for in-tab apps.

## Timeline

| Phase | Effort | Gate |
|---|---|---|
| Phase 0 audit | done | — |
| Phase 1 AILANG on-ramp | ~2 days | none |
| Phase 2 World app contract + A4 | World-sized | row 243 V4, D3, D5 |
| Phase 3 seeds | rides rows 151, 188, 274 | those rows |
| Phase 4 scaffold | ~1 day | row 274's manifest |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| AILANG and World design the same layer twice (r0 was an instance) | High | World owns the layer. This doc keeps only AILANG-side work and routes the rest to World's queue |
| The two task planes (AILANG coordinator/`chains` vs World episodes) get treated as one | Med | Stated in the problem statement and on the docs page. Any shared record needs its own doc |
| A sprint planned from this doc reaches into World | Med | Phase 1 is the only in-repo sprint scope. Phases 2–3 are World rows |
| The analogy overreaches (World objects ≠ files) | Med | The mapping table cites a landed or designed World primitive for each row |
| The A4 slice widens authority (a GUI app can do anything) | High | D3 is Mark's. The default is effects-only recording or a host-mediated effect, never ambient access |

## Related Documents

AILANG repo:
- [m-wasm-reflective-runtime](../implemented/v0_21_0/m-wasm-reflective-runtime.md): the Cognitive OS umbrella
- [m-cog-runtime](../implemented/v0_21_0/m-cog-runtime.md): DOM/Msg/Cog/Trace effects and event-log replay
- [m-a2ui-v1-platform-adoption](m-a2ui-v1-platform-adoption.md): `sunholo/a2ui` v1.0 (D2's AILANG-side remainder)
- [m-game-engine-effects](v1_1_0/m-game-engine-effects.md): stapledons-godot architecture (A4 seed)
- [m-chess-engine](v0_64_2/m-chess-engine.md): a pending AILANG-native application
- [m-connector-channels-marketplaces](m-connector-channels-marketplaces.md): Parse and executor as live MCP services
- [m-task-status-truth](m-task-status-truth.md): typed task status (AILANG coordinator plane)
- [docs/docs/guides/traces.md](../../docs/docs/guides/traces.md): the program-trace schema

World repo (`sunholo-data/ailang-world`, `dev`):
- [DESIGN.md](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/DESIGN.md) §1, §11 (human and AI interfaces), §14, §17 (M6), §20
- [HUMAN-SURFACE.md](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/HUMAN-SURFACE.md): ratified UX grammar
- [w-world-os-surface](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-world-os-surface.md): row 188, the OS surface (D-WORLD-87)
- [w-agent-generated-projections](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-agent-generated-projections.md): row 151, agent views (D-WORLD-92/93)
- [w-world-address-bar-verbs-v4](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-world-address-bar-verbs-v4.md): row 243, browser start and finish
- [w-ui-profile-preset-onboarding](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-ui-profile-preset-onboarding.md): row 269, presets in the UI
- [w-skill-grants](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-skill-grants.md): row 272, declared grants and host-mediated effects (D3's mechanism)
- [w-agent-world-context-and-messages](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-agent-world-context-and-messages.md): row 273
- [w-all-agent-work-through-world](https://github.com/sunholo-data/ailang-world/blob/dev/design_docs/planned/w-all-agent-work-through-world.md): D-WORLD-94

## References

- Mark's directive (2026-11-28, task-988c91fa)
- [Design Axioms](/docs/references/axioms)
- World mission charter `design_docs/world-mission.md`: bar, queue, Human Decision Ledger

## Verification Log

### World, first-party (r1, 2026-11-28; `sunholo-data/ailang-world` local `9a40fe68`, origin `91befdab`, which adds only row 243 V4 S9/S9b)

| # | Claim | Verification |
|---|---|---|
| W1 | 1.0 bar met; `v1.0.0-rc.1` released | `gh release list`: "AILANG World v1.0.0-rc.1 — the 1.0 bar is met", 2026-10-16. Bar clauses 1–7 in `world-mission.md` "The bar" |
| W2 | World's analogy is "everything is a typed state transition"; World does not read AILANG's task plane | `README.md` and `DESIGN.md` §1 quote it. `git grep -l "ailang chains\|observatory" -- '*.go'` → 0 files; `git grep -l "ailang coordinator" -- '*.go'` → 0 |
| W3 | Daemon routes for objects, worlds, log, receipts, registry, commit, episodes, `/surface`, `/workbench` | `git grep` of `"GET /…"`/`"POST /…"` patterns under `host/` and `cmd/`: the list quoted in the problem statement |
| W4 | Extension packages declare transitions in `transitions.json`; `se-tools` ships ten tools | `packages/se-tools/transitions.json` (fields `id`, `title`, `description`, `transitionFnFile`, `inputSchema`); README "Serve World's ten coding tools" |
| W5 | MCP, A2A and AG-UI served | Routes `POST /mcp/`, `GET /.well-known/agent.json`, `POST /a2a/`, `POST /agui/`. Queue rows 108 and 148 LANDED |
| W6 | Episodes: preset launch, confined agent, finish → PR, `why`; hosts Claude Code, codex, agy | Queue row 217 LANDED (presets, headless and interactive launch, `--resume`, agy added 2026-11-24); attended exit run "`episode start --preset world --launch` … `finish` opened #455" |
| W7 | Browser start/finish in progress | Row 243 V4: S8 merge `85ed2234`, S9 merge `c1869819`, S9b merge `d9635675` (2026-11-28) |
| W8 | herdr plugin; D-WORLD-94 goal | `plugins/ailang-world/`, `integrations/herdr/`; ledger D-WORLD-94 and `planned/w-all-agent-work-through-world.md` |
| W9 | AG-UI events + A2UI payload; agent views over a closed catalog | Ledger D-WORLD-64 (RESOLVED 2026-10-06), D-WORLD-92 (RESOLVED); row 151 text "verification checks it against WORLD'S closed catalog" |
| W10 | OS surface approved | Row 188 re-scoped by D-WORLD-87 (attended 2026-10-20, "ok this looks great. lets build it"); `planned/w-world-os-surface.md` r2 |
| W11 | Rows 269, 272, 273 | Row 269 Phase 1 LANDED (PR #664, #665; `/surface/presets.json`, `/surface/profiles.json`); rows 272/273 filed attended 2026-11-27 (PRs #683, #684), "NEXT after row 243 V4" |
| W12 | Code size | `git ls-files '*.ail' \| xargs wc -l` → 7,370; non-test `*.go` → 90,193; `*_test.go` → 174,487 |
| W13 | `serveapi/protocol` imports and AILANG pin | `git grep -l serveapi/protocol -- '*.go' \| wc -l` → 14; `go.mod`: `github.com/sunholo-data/ailang v0.47.2` |
| W14 | DESIGN plans generated UI as M6 and treats games and documents as projections | `DESIGN.md` §11.2 ("domain UIs are *generated* projections rather than hand-built apps"), §17 M6, §20 |
| W15 | Self-modification via extension packages proven | Bar clause 7; DESIGN §14 |
| W16 | Effect receipts fail closed on `indeterminate` | `README.md` receipts paragraph |

### AILANG side (r0, 2026-11-28, this repo — unchanged)

| # | Claim | Verification |
|---|---|---|
| V1 | No ailang-repo doc covers the applications layer | create-script search on "world os applications": token coincidences only. **r1 note:** the layer is covered on the World side (W9, W10, W14) |
| V5 | `ailang init web-app` scaffolds an AILANG API backend + React frontend | `ailang init --help`: types `web-app`, `package` |
| V6 | `ailang ops serve-api`, `ailang serve` (1957), `ailang ops mcp` exist | `ailang ops --help`, `ailang --help` |
| V7 | Process launching is capability- and path-pinned | `ailang run --help`: `--process-allowlist <list>` |
| V8 | Two production MCP services (Parse, executor) | m-connector-channels-marketplaces.md Problem Statement table |
| V9 | Program-trace event schema documented | `docs/docs/guides/traces.md` |
| V10 | `ailang replay` exists | `ailang replay --help` |
| V11 | Cognitive OS runtime shipped; DOM/Msg/Cog/Trace caps exist | m-wasm-reflective-runtime.md; `ailang check examples/cognitive_os/single_agent_replay.ail` → clean |
| V12 | An AILANG program can consume a program trace like a file today | probe run: `--emit-trace jsonl` → 4 events; `ailang check tracesummary.ail` clean; run output as in Example 1 |
| V13 | stapledons-godot: Godot owns rendering and input, AILANG sim over NDJSON stdio | m-game-engine-effects.md "Reality check (2026-10-28)" |
| V14 | Blender interactions appear in stapledons traces | **operator report only**; the A4 slice confirms first-party |
| V16 | `sunholo/a2ui` emits v0.9.1; v1.0 migration planned | m-a2ui-v1-platform-adoption.md |
| V17 | Typed task status, unforgeable result line (AILANG coordinator plane) | m-task-status-truth.md; m-run-policy-result-unforgeable.md |
| V18 | Probe written per the current teaching prompt | `ailang prompt` read before authoring; `ailang check` clean |

**Superseded by r1:** V2, V3 and V4 (World status from the v1-mission ledger) are replaced by W1, W5 and W13. V15's "world 100k" is replaced by W12.

## Future Work

- A "coreutils of history": AILANG library functions over World's episode report and log, once World versions them (row 274/D5)
- An app-to-app composition gesture, which row 188's "change from here" and brane stitching already sketch
- Evaluation lane: "build a World app" as an agentic benchmark

---

**Document created**: 2026-11-28
**Last updated**: 2026-11-28 (r1: first-party World audit; the scope narrowed to the AILANG side)
**Quorum**: not run. The open decisions (D3, D4, D5) are Mark's, recorded in World's ledger. Run `ailang design-quorum` before planning Phase 4.
