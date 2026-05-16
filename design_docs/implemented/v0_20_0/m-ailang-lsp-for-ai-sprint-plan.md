# M-AILANG-LSP-FOR-AI Sprint Plan

**Sprint ID**: M-AILANG-LSP-FOR-AI
**Design doc**: [m-ailang-lsp-for-ai.md](m-ailang-lsp-for-ai.md)
**Target version**: v0.21.0
**Duration**: 5 days (~32 working hours)
**Risk level**: Medium (M3 position index is the only genuine unknown)
**Status**: Approved (committee decisions 1–5 locked in), ready to execute

## Sprint Goal

Ship a minimal AILANG LSP server (`ailang lsp`) exposing existing checker outputs over the standard LSP protocol, with a Claude Code plugin in the repo-local marketplace, so AI agents working on `.ail` codebases get diagnostics-after-save, go-to-def, find-references, hover types, and document symbols **structurally** instead of via grep + re-read + manual `ailang check`.

The milestone is also a **token-cost impact experiment** — M6 includes an eval harness that measures LSP-on vs LSP-off token consumption on a fixed task. We publish the delta.

## Locked-In Decisions (from design doc committee review)

1. **MVP capabilities**: `diagnostics`, `definition`, `references`, `hover`, `documentSymbol`. Everything else deferred.
2. **Packaging**: `ailang lsp [--stdio]` subcommand. No standalone binary. No flag on `serve-api`.
3. **Single surface**: one server for humans and AI, with LSP capability negotiation.
4. **Reuse boundary**: thin adapters on `internal/types`, `internal/elaborate`, `internal/iface`, `internal/pipeline`. New work: position-to-symbol index + subcommand wiring. MVP runs full pipeline on `didSave` (no incremental).
5. **Distribution**: ship `.claude/plugins/ailang-lsp/` in the existing `.claude-plugin/marketplace.json`. LSP config in `.lsp.json` (NOT inline — see commit `235a5afc` lesson).

## Velocity Calibration

Recent comparable sprints (single-author, multi-milestone sprints):

| Sprint | Date | LOC | Duration | Notes |
|--------|------|-----|----------|-------|
| M-SERVEAPI-SURFACE-DROPS | May 15 | ~250 | 1 day (3 commits) | Pure plumbing; low novelty |
| M-DX26 Phase 5 (whole) | May 14 | ~600 | ~2 days | 3 sub-phases, mid-novelty |
| M-PARSER-BLOCK-TR | May 14 | ~80 | ~4h | One file, one bug |
| M-STDLIB-XML-WALK-PERF | May 13 | ~400 | ~2 days | M1–M4, multi-milestone |
| M-STDLIB-URL-ENCODE | May 12 | ~120 | ~3h | Two-function add |

**Velocity baseline:** ~150 LOC/day sustained on plumbing; ~250–300 LOC/day on novel work where most code is mechanical adapters.

This sprint's estimated **~1,300 LOC** sits at the upper end. Most of it is mechanical adapters (LSP request → existing internal API → LSP response). The genuine new code is the position-to-symbol index in M3 (~250 LOC including walker + lookup), which is also the highest-risk milestone.

## Milestone Breakdown

### M1: LSP wire-protocol skeleton (~6h, ~250 LOC)

**Goal**: `ailang lsp --stdio` starts a JSON-RPC server, handles `initialize`/`initialized`/`shutdown`/`exit` correctly, and advertises the locked-in MVP capabilities. No analysis work yet — pure plumbing.

**Files:**
- New: `cmd/ailang/lsp.go` — subcommand entry point + flag parsing (~50 LOC)
- New: `internal/lsp/server.go` — `Server` struct, `NewServer`, `Run(ctx, stream)` (~100 LOC)
- New: `internal/lsp/lifecycle.go` — `initialize`, `initialized`, `shutdown`, `exit` handlers (~80 LOC)
- New: `internal/lsp/server_test.go` — handshake test using in-memory pipe (~80 LOC)
- Modified: `cmd/ailang/main.go` — register `lsp` subcommand (~5 LOC)
- Modified: `cmd/ailang/help.go` — document `ailang lsp` (~10 LOC)
- Modified: `go.mod` / `go.sum` — add `go.lsp.dev/protocol` + `go.lsp.dev/jsonrpc2`

**Dependencies to add:**
```
go get go.lsp.dev/protocol@latest
go get go.lsp.dev/jsonrpc2@latest
```

(Both are mature, license-compatible — same libraries gopls itself uses internally for protocol types.)

**Capability advertisement (post-`initialize`):**
```go
ServerCapabilities{
    TextDocumentSync:          &SyncKindFull, // didOpen/didSave full text
    DiagnosticProvider:        true,          // pull-mode or push-mode TBD in M2
    DefinitionProvider:        false,         // M4
    ReferencesProvider:        false,         // M4
    HoverProvider:             false,         // M3
    DocumentSymbolProvider:    false,         // M5
}
```

M1 advertises only `TextDocumentSync` so the client knows we accept documents but doesn't expect us to answer feature queries yet. Subsequent milestones flip each capability `true` as they ship.

**Acceptance criteria:**
- `ailang lsp --stdio` accepts a hand-rolled LSP `initialize` request and returns a valid `InitializeResult`
- Capability set in the response matches the M1 list above (only TextDocumentSync true)
- `shutdown` + `exit` shut the process down cleanly (exit code 0)
- `server_test.go` covers the full handshake using `net.Pipe()` for in-memory transport
- `make build` succeeds with the new dependencies; `go.sum` checked in
- `ailang help` includes `lsp` in the subcommand list

**Risk:** Low. `go.lsp.dev/protocol` provides all message types; `jsonrpc2` handles framing. The hardest part is matching the lifecycle protocol exactly (the spec is strict about `initialized` vs `shutdown` order).

---

### M2: Diagnostics-only MVP (~6h, ~200 LOC)

**Goal**: An AI agent saving a `.ail` file gets pipeline-checker diagnostics published back over LSP within budget (target: ≤500ms for a single-file 1k-LOC document). **This milestone alone closes the largest token-cost gap from the design doc.**

**Files:**
- New: `internal/lsp/diagnostics.go` — `runPipeline(uri) → []protocol.Diagnostic` adapter (~120 LOC)
- New: `internal/lsp/document_sync.go` — `didOpen`, `didChange` (no-op for now), `didSave` handlers (~60 LOC)
- New: `internal/lsp/diagnostics_test.go` — fixture-based test: known type error produces a diagnostic with correct range (~80 LOC)
- Modified: `internal/lsp/lifecycle.go` — flip `TextDocumentSync` to full + advertise diagnostics on initialize

**Implementation detail:**

`runPipeline(uri)` flow:
1. Parse `uri` → on-disk file path
2. Construct `pipeline.Config{}` matching `ailang check` defaults (caps off, no eval, type-check only)
3. Call `pipeline.Run(cfg, pipeline.Source{Filename: path, Source: readFile(path)})`
4. Map `Result.Errors` + `Result.Warnings` → `[]protocol.Diagnostic`:
   - Error position comes from the error's `core.Position` (most pipeline errors carry one); fall back to `{0,0}–{0,1}` with severity Error if no position
   - Severity: errors → `DiagnosticSeverityError`, exhaustiveness warnings → `DiagnosticSeverityWarning`
   - `source: "ailang"`, `code:` filled when the error type has a stable code
5. Send `textDocument/publishDiagnostics` notification with the URI + diagnostics array

**didSave** triggers `runPipeline`; **didOpen** also triggers it (so freshly-opened files get analyzed immediately); **didChange** stores the buffer text in an in-memory map but does NOT re-run the pipeline (defer to incremental milestone — see Out-of-Scope).

**Acceptance criteria:**
- A well-typed `.ail` file on `didSave` produces a `publishDiagnostics` with an empty array
- A `.ail` file with a known type error (test fixture) produces a `publishDiagnostics` with at least one diagnostic at the correct range
- An exhaustiveness warning produces a `Warning`-severity diagnostic, not Error
- Diagnostic round-trip latency on a 1k-LOC fixture stays under 500ms median across 10 runs (asserted in test as a soft check)
- `make ci` passes

**Risk:** Medium-Low. Risk is in *position mapping* — pipeline errors don't all carry positions of the same fidelity. Mitigation: any error without a position lands at `{0,0}–{0,1}` with the file-level message; document the limitation in the implementation report. Don't block M2 on perfect positions.

---

### M3: Position-to-symbol index + hover (~7h, ~350 LOC)

**Goal**: Given a cursor `(uri, line, col)`, identify the AST node under it and return its inferred type + effect row. **This is the highest-risk milestone — it's the only one with genuinely new code, and it underpins M4.**

**Files:**
- New: `internal/lsp/index.go` — position-to-AST-node index built from `*ast.File` (~180 LOC)
- New: `internal/lsp/index_test.go` — table-driven tests for index lookup on a small fixture (~100 LOC)
- New: `internal/lsp/hover.go` — `textDocument/hover` handler (~60 LOC)
- Modified: `internal/lsp/lifecycle.go` — flip `HoverProvider: true`
- Modified: `internal/lsp/document_sync.go` — on successful `didSave`, store the `Result` + index in a session-scoped map keyed by URI

**Position index design:**

```go
type PositionIndex struct {
    // Sorted by start position; binary search to find smallest enclosing node
    spans []nodeSpan
}

type nodeSpan struct {
    start, end token.Position // line+col, 1-indexed
    node       ast.Node       // expression, binding, type ref, ...
    kind       NodeKind       // for hover-relevance filtering
}

func BuildPositionIndex(file *ast.File) *PositionIndex
func (px *PositionIndex) Lookup(line, col int) (ast.Node, NodeKind, bool)
```

Walker visits every `*ast.Ident`, `*ast.FuncDecl`, `*ast.LetBinding`, `*ast.Apply`, `*ast.TypeRef`, etc., recording its position span. Lookup does a binary search for "smallest span containing the cursor".

**Hover handler:**
1. Look up the node at cursor via `PositionIndex.Lookup`
2. If node is an `*ast.Ident`, resolve it via the stored `Result.TypeChecker` to get its `Scheme` and effect row
3. Format as: `type: %s\neffects: {%s}\n` (Markdown content kind)
4. Return `protocol.Hover{Contents: ..., Range: nodeSpan.toLSPRange()}`

**Acceptance criteria:**
- `Lookup(line, col)` returns the smallest enclosing node for a cursor inside an identifier
- `Lookup` returns `(nil, NodeKindNone, false)` for cursors outside any node (whitespace, comments)
- Hover on an `Ident` with known type returns a `Hover{Contents}` matching `type: <T>\neffects: {<labels>}\n`
- Hover on a position with no resolvable type returns `nil, nil` (LSP spec: no hover info)
- Fixture: hover at line 5, col 14 of a known `.ail` file returns the expected type-string
- `make ci` passes; M2 diagnostics still work

**Risk:** Medium. Two unknowns:
1. **AST position fidelity** — if `*ast.Ident` nodes don't carry sub-token start/end positions, the lookup will be ambiguous for nested expressions. Spike this on day-3 morning: write a 30-line test that asserts position info on `examples/` files; if it fails, we either (a) backfill positions in the AST in a parallel mini-milestone, or (b) ship M3 with coarser-granularity hover (function-level only) and file the position-fidelity work as M-PARSER-POSITION-FIDELITY.
2. **TypeChecker reuse** — the `Result.TypeChecker` is a debug-oriented surface, not a clean public API. Worst case we adapt it; expected case it answers `LookupType(node) → Scheme, Row` directly.

If either unknown lands worst-case, M3 expands to 1.5 days and M4+M5 compress into Day 4.

---

### M4: Definition + references (~5h, ~250 LOC)

**Goal**: Cross-module go-to-definition and find-references using the stored `Result.Modules` (per-module Core) and `Result.Interface` (exported decls).

**Files:**
- New: `internal/lsp/definition.go` — `textDocument/definition` handler (~80 LOC)
- New: `internal/lsp/references.go` — `textDocument/references` handler (~100 LOC)
- New: `internal/lsp/xref_test.go` — fixture: 2-module project; definition crosses module boundary; references return all use-sites (~120 LOC)
- New: `examples/lsp_xref_fixture/` — 2-module `.ail` fixture used by the test (~30 LOC of `.ail`)
- Modified: `internal/lsp/lifecycle.go` — flip `DefinitionProvider: true`, `ReferencesProvider: true`

**Implementation detail:**

`definition`:
1. Look up node at cursor via `PositionIndex`
2. If node is an `*ast.Ident` with a resolved binding (typechecker knows the declaration site), return `Location{URI: declFile, Range: declRange}`
3. For cross-module references, follow the import resolution chain via `Result.Modules[moduleID].File` to find the actual decl

`references`:
1. Look up node at cursor → symbol name + declaring module
2. Walk all open documents' `PositionIndex` entries; collect `*ast.Ident`s with the same canonical name *and* resolved to the same declaring module
3. Return `[]Location` — LSP spec says references should include the definition unless `includeDeclaration: false` in the request

Workspace-scope for references: MVP scopes to "documents that have been `didOpen`-ed or `didSave`-ed in this session". Document the limitation; full-workspace scan is deferred (see Out-of-Scope).

**Acceptance criteria:**
- Definition on `freePlanFromCatalog(...)` inside `examples/lsp_xref_fixture/a.ail` returns a `Location` pointing to `examples/lsp_xref_fixture/b.ail` (the decl site)
- References on the same identifier from `b.ail` returns at least two `Location`s (the decl + one use in `a.ail`)
- Definition on a stdlib identifier (e.g. `std/list.map`) returns a location inside `stdlib/std/list.ail`
- `make ci` passes

**Risk:** Low-Medium. Risk shape is the same as M3 (depends on AST + typechecker giving us declaration sites). If M3 ships, M4 is mechanical.

---

### M5: Document symbols (~3h, ~100 LOC)

**Goal**: `textDocument/documentSymbol` returns the top-level definitions of an open `.ail` file, hierarchically nested where applicable (e.g., type with constructors).

**Files:**
- New: `internal/lsp/symbols.go` — AST walker producing `[]protocol.DocumentSymbol` (~70 LOC)
- New: `internal/lsp/symbols_test.go` — fixture test (~50 LOC)
- Modified: `internal/lsp/lifecycle.go` — flip `DocumentSymbolProvider: true`

**Symbol kinds:**
- `FuncDecl` → `SymbolKindFunction`
- `TypeDecl` (ADT) → `SymbolKindClass` (LSP has no `SymbolKindADT`); children = constructors as `SymbolKindConstructor`
- `LetBinding` at top level → `SymbolKindVariable`
- `TypeClassDecl` → `SymbolKindInterface`
- `Import` → not emitted (clutter)

**Acceptance criteria:**
- A `.ail` file with one func, one ADT (3 constructors), and one let returns 3 top-level symbols, where the ADT has 3 children
- Each symbol's `range` and `selectionRange` are valid LSP positions (start ≤ end, inside file bounds)
- `make ci` passes

**Risk:** Low. Pure AST traversal, no new types or libraries.

---

### M6: Claude Code plugin packaging + docs + token-cost eval (~5h, ~200 LOC + plugin JSON + docs)

**Goal**: Ship the plugin so AI agents can install + use it, document the user surface, and **publish a number** on the LSP-on vs LSP-off token-cost delta.

**Files:**
- New: `.claude/plugins/ailang-lsp/.claude-plugin/plugin.json` — metadata-only (per commit `235a5afc` lesson)
- New: `.claude/plugins/ailang-lsp/.lsp.json` — LSP config: `command: "ailang"`, `args: ["lsp", "--stdio"]`, `extensionToLanguage: {".ail": "ailang"}`, `restartOnCrash: true`
- Modified: `.claude-plugin/marketplace.json` — add `ailang-lsp` plugin entry alongside `ailang-go-lsp`
- New: `docs/docs/guides/lsp.md` — user-facing guide (~150 lines prose) with sections for human users (VSCode setup) and AI-agent users (Claude Code plugin install)
- Modified: `CLAUDE.md` — append `ailang-lsp` to the existing LSP plugin section
- Modified: `cmd/ailang/help.go` — `ailang lsp` subcommand help line
- Modified: `changelogs/v0.10-current.md` — `[Unreleased]` entry under `### Added`
- New: `bench/lsp_token_cost/` — eval harness (~120 LOC)
  - `bench/lsp_token_cost/task.md` — fixed agent task: "add a function `daysBetween(a, b: Date) → Int` to examples/calendar.ail and update its call sites"
  - `bench/lsp_token_cost/run.sh` — driver: run task once with the LSP plugin enabled, once with it disabled, capture tokens
  - `bench/lsp_token_cost/README.md` — how to interpret results

**Eval harness design:**

The harness uses `ai-coding-lang-bench` plumbing if available; otherwise it's a thin wrapper around `claude --print` (the headless mode) with two runs:
1. **LSP-on**: Claude Code session with `ailang-lsp` plugin installed
2. **LSP-off**: same session with the plugin disabled

Both runs execute the same fixed task. We capture:
- Total tokens (input + output)
- Wall-clock duration
- Number of `Bash`, `Read`, `Grep` tool calls (proxy for "grepping in lieu of LSP")
- Final correctness (did the function get added with correct type?)

The result is **single-trial, single-task** — not a statistically robust eval, but enough to publish a number with appropriate caveats. The implementation report includes the raw numbers and a "what surprised us" section.

**Acceptance criteria:**
- `/plugin install ailang-lsp@ailang-tools` succeeds against the local marketplace
- `/plugin` shows `ailang-lsp` under Installed with no entry under Errors
- Opening a `.ail` file in a Claude Code session shows real-time diagnostics on save
- `docs/docs/guides/lsp.md` documents both the human IDE path (VSCode + a generic LSP client) and the Claude Code plugin path
- `bench/lsp_token_cost/run.sh` produces a `results.json` with both LSP-on and LSP-off numbers
- A summary line is added to the implementation report: "LSP reduced tokens by X% on the fixed task (N=1 trial)"
- CHANGELOG entry added
- `make ci` passes

**Risk:** Medium on the eval harness — `claude --print` headless mode has its own surface area to learn (see `.claude/skills/headless-runner/`). If the harness slips, we ship M6 without the published number and file it as M-LSP-EVAL-FOLLOWUP.

---

## Day-by-Day Plan

**Day 1 (~6h)** — M1: LSP wire-protocol skeleton
- 09:00–10:00: `go get` deps; scaffold `internal/lsp/` package; `cmd/ailang/lsp.go` subcommand registration
- 10:00–12:00: `server.go` + `lifecycle.go` — initialize/shutdown/exit handlers
- 13:00–14:30: `server_test.go` — in-memory pipe handshake test
- 14:30–15:30: `make ci`, fix lint, commit

**Day 2 (~6h)** — M2: Diagnostics-only MVP
- 09:00–11:00: `diagnostics.go` — pipeline.Run adapter + error→Diagnostic mapping
- 11:00–12:00: `document_sync.go` — didOpen/didChange/didSave plumbing
- 13:00–14:30: `diagnostics_test.go` — fixture with known type error
- 14:30–15:30: Latency budget test (1k-LOC fixture, ≤500ms)
- 15:30–16:00: Commit

**Day 3 (~7h)** — M3: Position index + hover (highest-risk milestone)
- 09:00–09:30: **Spike**: write a 30-line position-fidelity probe on `examples/` files. Decision gate — if fidelity is insufficient, pause and decide between path (a) backfill positions or (b) coarser-granularity M3
- 09:30–12:00: `index.go` — node-span walker + binary-search lookup
- 13:00–14:30: `hover.go` — handler + TypeChecker integration
- 14:30–16:30: `index_test.go` — table-driven tests; iterate on the hard cases
- 16:30–17:00: Commit

**Day 4 (~5h)** — M4: Definition + references; **M5: Document symbols**
- 09:00–11:00: M4 — `definition.go` + cross-module resolution
- 11:00–12:30: M4 — `references.go` + 2-module xref fixture
- 13:30–14:30: M5 — `symbols.go` walker
- 14:30–15:00: M5 — `symbols_test.go`
- 15:00–16:00: Commit M4 + M5 separately

**Day 5 (~5h)** — M6: Plugin packaging + docs + eval
- 09:00–10:00: Plugin scaffolding — `.claude/plugins/ailang-lsp/{.claude-plugin/plugin.json, .lsp.json}` + marketplace entry
- 10:00–11:30: Live-test the plugin end-to-end in a Claude Code session against `examples/`
- 11:30–13:00: `docs/docs/guides/lsp.md` + CLAUDE.md append + help.go + CHANGELOG
- 14:00–16:00: `bench/lsp_token_cost/` — eval harness + first run
- 16:00–17:00: Implementation report with the published delta; `make ci`; commit

**Buffer**: ~3 hours total across the week for M3 overruns, integration-test flakes, or CI surprises.

---

## Pause Points for Human Input

**Day 3, 09:30 — Position fidelity gate** (the only mandatory pause).

After running the 30-line probe on `examples/`, surface the result to the user:
- **Path A** (positions are fine): proceed with M3 as planned
- **Path B** (positions are insufficient): pause, file a parallel mini-milestone M-PARSER-POSITION-FIDELITY, ship M3 with function-level-only hover

User decides which path; both are pre-thought-out in the design doc.

**Other potential pauses (not pre-scheduled):**
- If `go.lsp.dev/protocol` has a breaking change between scaffolding and M3, surface before proceeding
- If `Result.TypeChecker` doesn't expose enough to answer hover queries, surface before patching internal APIs
- If the M6 eval harness produces a *negative* delta (LSP-off cheaper than LSP-on), surface — that's a surprising result worth investigating before publishing

---

## Success Metrics

- [ ] All 6 milestones merged
- [ ] `ailang lsp --stdio` handles full LSP lifecycle + 5 MVP capabilities
- [ ] Plugin installable from local marketplace; visible to Claude Code without errors
- [ ] Diagnostic round-trip latency ≤500ms median on 1k-LOC fixture
- [ ] Hover, definition, references, documentSymbol all answer correctly on the `examples/lsp_xref_fixture/` test corpus
- [ ] `make ci` passes throughout (no test regressions in `internal/pipeline`, `internal/types`, `internal/elaborate`)
- [ ] `docs/docs/guides/lsp.md` published with both human and AI-agent install paths
- [ ] CHANGELOG entry under `[Unreleased]` in `changelogs/v0.10-current.md`
- [ ] Token-cost eval harness produces a number; implementation report quotes it with caveats
- [ ] Total LOC: ~1,300 ±20%

## Example Files

- `examples/lsp_xref_fixture/a.ail` + `b.ail` — 2-module project used by M4 xref tests (and useful as a docs example for "what AILANG modules look like to an LSP")
- `examples/calendar.ail` — used by M6 eval harness as the agent's task target

No top-level `examples/lsp_*.ail` showcase file is required — the LSP doesn't change the language surface, only the tooling surface. The plugin-install instructions in `docs/docs/guides/lsp.md` are the user-facing showcase.

## Dependencies

**External (new):**
- `go.lsp.dev/protocol` — LSP message types
- `go.lsp.dev/jsonrpc2` — JSON-RPC framing

**Internal (consumed read-only):**
- `internal/pipeline.Run` — drives diagnostics in M2
- `internal/types.TypeChecker` (via `pipeline.Result.TypeChecker`) — hover/definition type info in M3+M4
- `internal/elaborate.*` (via `pipeline.Result.Modules[].Core`) — resolved AST for index walks
- `internal/iface.Iface` (via `pipeline.Result.Interface`) — cross-module identifier resolution in M4
- `internal/ast.*` — position-bearing nodes for the M3 index

**Plugin tooling (already shipped, commit `235a5afc`):**
- `.claude-plugin/marketplace.json` — extended in M6
- `.claude/plugins/ailang-go-lsp/` — referenced as the packaging pattern

## Out of Scope (Tracked for Follow-ups)

From design doc "Deferred Decisions":
- **Completion, signatureHelp, codeAction, formatting, rename, semantic tokens, code lens, inlay hints** — track as `M-AILANG-LSP-HUMAN-IDE-*` if a human-IDE use case materializes
- **Incremental re-typecheck on `didChange`** (per-keystroke) — track as `M-AILANG-LSP-INCREMENTAL`. MVP is `didSave`-only.
- **Workspace-wide indexing for `references`** — MVP scopes to session-opened documents only. Track as `M-AILANG-LSP-WORKSPACE-SCAN`.
- **Cross-session server caching** — each Claude Code session spawns its own `ailang lsp`. Track as `M-AILANG-LSP-SHARED-SERVER`.
- **LSP request telemetry** — interesting for analyzing AI navigation patterns. Track as `M-AILANG-LSP-TELEMETRY`.
- **Multi-trial statistical eval** — M6 publishes a single-trial number. A proper N=10+ eval across multiple tasks/models is `M-LSP-EVAL-FOLLOWUP`.

## Commit Plan

6 commits, one per milestone:

1. `feat(lsp): M-AILANG-LSP-FOR-AI M1 — ailang lsp subcommand + LSP protocol skeleton`
2. `feat(lsp): M-AILANG-LSP-FOR-AI M2 — pipeline-driven diagnostics on didSave`
3. `feat(lsp): M-AILANG-LSP-FOR-AI M3 — position-to-symbol index + textDocument/hover`
4. `feat(lsp): M-AILANG-LSP-FOR-AI M4 — textDocument/definition + references`
5. `feat(lsp): M-AILANG-LSP-FOR-AI M5 — textDocument/documentSymbol`
6. `feat(lsp): M-AILANG-LSP-FOR-AI M6 — Claude Code plugin + docs + token-cost eval`

All commits reference the design doc path in the body. No `Fixes #N` since no GitHub issue is associated — this is author-initiated, not bug-driven.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|-----------|
| AST position fidelity insufficient for sub-token hover | Medium | M3 expands; M4 affected | Day-3 09:30 spike + pre-thought-out fallback (function-level hover, file M-PARSER-POSITION-FIDELITY) |
| `Result.TypeChecker` doesn't expose hover-needed queries | Low | Patch internal API; adds ~50 LOC | Adapter wraps debug-oriented surface in a clean read-only view |
| `pipeline.Run` on every didSave is too slow on real-world files | Low | UX regression for AI agents | Latency test in M2 (≤500ms budget); if exceeded, defer M3+ and ship M2 only with a known-limitation note |
| `go.lsp.dev/*` libraries are stale or have breaking changes | Low | Re-pick library; ~1 day setback | Use latest stable; alternative: `creachadair/jrpc2` + hand-rolled LSP types (gopls's approach pre-`go.lsp.dev/*`) |
| M6 eval harness slips | Medium | Ship M6 without published number | File as M-LSP-EVAL-FOLLOWUP; ship the plugin alone in M6 |
| Plugin install path differs across Claude Code versions | Low | Doc fragility | Test against the user's installed version (CLAUDE.md cites the marketplace + install command) |

## Best Practices Push (Deferred to Post-Sprint)

Once shipped, AILANG agent-facing prompts (`prompts/*`) should mention LSP availability — agents that don't know to use it won't get token savings. Tracked as a docs/prompts push, not part of this sprint.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_21_0/m-ailang-lsp-for-ai-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-AILANG-LSP-FOR-AI.json`
