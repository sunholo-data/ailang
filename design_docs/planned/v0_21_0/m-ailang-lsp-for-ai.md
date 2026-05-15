# M-AILANG-LSP-FOR-AI — AILANG LSP server targeting AI coding agents (and humans)

**Status**: Planned — awaiting design committee review
**Target**: v0.21.0 (MVP scope; sprint-planner sets the exact slot)
**Priority**: P1 — unblocks AI-agent productivity on large `.ail` codebases; not blocking a customer incident
**Estimated**: Deliberately unstated — sprint-planner to size after committee picks Decisions 1–5
**Dependencies**: None. Reuses existing `internal/types`, `internal/elaborate`, `internal/iface`, `internal/pipeline` outputs.

## Origin & Stance Reversal

The original AILANG position was: **"an AILANG LSP is for humans, not AI — AI consumers should use the type checker and tooling directly."** This doc revisits and **reverses** that stance.

The triggering observation: Claude Code (and other agentic coding tools) [now consume LSP servers directly via plugins](https://code.claude.com/docs/en/plugins-reference#lsp-servers). The advertised capabilities — `textDocument/diagnostics` after every edit, `definition`, `references`, `hover` — are exactly the operations that an AI agent navigating a large codebase otherwise performs by **grepping, re-reading whole files, and waiting for an explicit `ailang check` invocation**. Each of those is a token cost the agent pays for every navigation step.

We validated the workflow locally for the Go side of this repo: [`.claude-plugin/marketplace.json`](../../../.claude-plugin/marketplace.json) and [`.claude/plugins/ailang-go-lsp/`](../../../.claude/plugins/ailang-go-lsp/) ship a `gopls` plugin, and the documented capabilities — instant diagnostics, go-to-def, hover — are exactly what an LSP-blind AI agent on a 315k-LOC Go tree spends grep-tokens to approximate. That same gap will appear on the `.ail` side once codebases get large. Motoko-style `.ail` projects in the 50–100k-LOC range — [motoko_explorer](https://github.com/...) class — are where the failure mode becomes acute: an agent re-reading a 600-line file to recover an import binding is doing manually what the elaborator already did at module load.

The position-reversal is not a contradiction of AILANG's pitch. AILANG is positioned around **machine decidability and semantic transparency**. An LSP is the *natural exposure surface* for that promise to AI consumers — it is the wire protocol that turns "machine-decidable" into "machine-readable, on every keystroke". If anything, AILANG is unusually well-suited to ship an AI-targeted LSP because the underlying analysis is already total and deterministic; we don't need the layered approximations (best-effort completion, partial-parse error recovery) that human-IDE LSPs invest most of their engineering in.

## Axiom Compliance

**Canonical reference:** [Design Axioms](../../../docs/docs/references/axioms.md)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism; LSP responses are pure projections of existing deterministic analyses |
| A2: Replayability | 0 | No trace impact; an LSP session is not a program execution |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes; LSP server reads files at the same authority as `ailang check` |
| A5: Bounded Verification | +1 | Diagnostics-on-edit shifts verification from "agent remembers to run check" to "agent receives the verdict structurally" |
| A6: Safe Concurrency | 0 | No concurrency changes (LSP server is single-process; reuses existing re-typecheck path) |
| A7: Machines First | **+2** | Direct application: AILANG's analyses become machine-consumable via a standard protocol, not a one-off CLI output format |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No cost surface change |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +1 | Diagnostics surface elaborator/typechecker errors at edit time instead of compile time — same failure information, earlier and structured |
| A12: System Boundary | +1 | Defines an explicit, standard-protocol boundary between AILANG's analyses and consumer tooling — human or AI |

**Net Score: +5** → **Decision: Proceed (pending committee on Decisions 1–5)**

### Hard Violation Check

- [x] A1 (Determinism): LSP responses are projections of `internal/types` + `internal/elaborate` outputs which are already deterministic
- [x] A3 (Effects): No new effects; LSP server has no capabilities of its own
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Explicitly aligned — this is the machines-first surface for AILANG's existing analyses

## Problem Statement

**An AI coding agent working on a 50k-LOC AILANG codebase has no structural way to ask "what is the type of this binding?" or "where is this symbol defined?"** It can:

1. Read whole files and re-elaborate them in its head from raw `.ail` source. Expensive in tokens, error-prone, and the agent has weaker type-inference fidelity than the actual checker.
2. Run `ailang check <file>` as a shell command and parse the textual output. Works for diagnostics but doesn't answer navigation queries — and the agent has to *remember* to run it after every edit.
3. Grep for identifiers. Returns false positives (string literals, comments, unrelated rebindings) and gives no type information.

All three are token-expensive proxies for what an LSP server would answer in a single round trip with structured data. The cost compounds on every navigation step.

### Why an AI-targeted LSP looks different from a human one

A human IDE LSP needs hover-rendered docs, signature help, completion popups, code lens, refactor menus, formatting providers, rename-symbol — most of the engineering investment is in **UX rendering** and **partial-parse recovery** so the IDE stays usable while the user is mid-keystroke. An AI agent's needs are smaller and stricter:

- It accepts only fully-valid documents (or fully-invalid ones with structured errors) — it doesn't keystroke-by-keystroke.
- It doesn't need completion menus — its model produces completions.
- It needs **diagnostics, definitions, references, hover, document symbols**. Five capabilities. That's the entire MVP.

The asymmetry matters: an AI-targeted LSP is **substantially cheaper to build** than a human one because we get to defer ~70% of the surface area indefinitely. Most of what we'd ship is already computed inside the elaborator and type checker — the work is *exposure plumbing*, not new compiler work.

### Why this isn't urgent yet — and why we should ship it before it is

Today there is no `.ail` codebase large enough (in production AI-agent use) that the lack of an LSP is on fire. The motoko_explorer-class workloads are a near-future concern, not a today concern. The window to ship a tight MVP **before the lack of one becomes a token-cost incident** is now. Once a customer is paying the navigation tax, we lose the option to scope the MVP tightly — pressure forces us to bolt on human-IDE features too.

## Goals

**Primary Goal:** Ship a minimal AILANG LSP server (`ailang-lsp` or equivalent) that exposes existing checker outputs over the standard LSP protocol, with a packaged Claude Code plugin in the repo-local marketplace, so AI agents working on `.ail` codebases get diagnostics-after-edit, go-to-definition, find-references, hover types, and document symbols without bespoke tooling.

**Success Metrics:**
- An AI agent editing a `.ail` file receives diagnostics within 500ms of save without invoking `ailang check`
- `textDocument/definition` resolves cross-module imports correctly for at least 90% of cases in a benchmark `.ail` corpus
- `textDocument/hover` returns the inferred type and effect row for any well-typed binding
- The same server, started against the AILANG stdlib (`stdlib/*.ail`), returns sensible answers for all five MVP capabilities
- A Claude Code plugin shipping the server appears in `.claude-plugin/marketplace.json` alongside `ailang-go-lsp`, installable by the same `/plugin install` flow
- A documented eval comparing AI agent token cost on a fixed task with LSP-on vs LSP-off shows ≥30% token reduction

## High-Impact Decisions (for Design Committee)

This doc requests committee review on the following before any code lands, mirroring the [M-CODEGEN-STRATEGIC-REVIEW](../../implemented/v0_11_0/m-codegen-strategic-review.md) pattern of "ask the committee for scope decisions before sinking implementation cost into a wrong shape".

| # | Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|----------|-----------------|-----------|----------|-------------|
| 1 | **MVP capability set** | Picking too small leaves the LSP useless to AI agents; picking too large drags us into human-IDE polish indefinitely | committee | design | high |
| 2 | **Server packaging**: standalone `ailang-lsp` binary vs subcommand `ailang lsp` vs flag on `ailang serve` | Affects build, distribution, and Claude Code plugin shape | committee | design | medium |
| 3 | **One surface for humans + AI, or two parallel servers** | Forking later is cheaper than merging later, but two surfaces means double maintenance | committee | design | high |
| 4 | **Reuse audit boundary** — which internal packages get a thin LSP adapter, and where do we need genuinely new code (position indexing, incremental re-typecheck) | Determines whether this fits a single milestone or spawns a multi-mile effort | committee + author | design | medium |
| 5 | **Distribution channel** for the Claude Code plugin — same repo-local marketplace, or a published plugin alongside `pyright-lsp`/`rust-analyzer-lsp` | Affects discoverability and version-lockstep with the AILANG binary | committee | design | low |

### Author's Recommendation on Each Decision (for committee to accept, reject, or amend)

**Decision 1 — MVP capability set:**
- **Include:** `textDocument/diagnostics` (typechecker + effect inference + totality outputs), `textDocument/definition`, `textDocument/references`, `textDocument/hover` (type + effect row), `textDocument/documentSymbol`.
- **Defer:** `textDocument/completion`, `textDocument/signatureHelp`, `textDocument/codeAction`, `textDocument/formatting`, `textDocument/rename`, semantic tokens, code lens, inlay hints. These are human-IDE-shaped — an AI agent doesn't consume completion menus or hover-rendered Markdown.

**Decision 2 — Server packaging:**
- Recommend a **subcommand of the existing `ailang` binary**: `ailang lsp [--stdio|--socket :PORT]`. Rationale: single distributable, version-locked with the language, no second binary for users to track. The Claude Code plugin's `command` becomes `ailang lsp`, mirroring how `gopls` ships inside the Go toolchain.
- Reject standalone `ailang-lsp` binary (drift risk between LSP and ailang versions).
- Reject reusing `ailang serve-api` — that's an HTTP REST gateway with `@route` semantics, structurally orthogonal to LSP wire protocol.

**Decision 3 — Single surface vs two:**
- Recommend **single surface, capability-negotiated**. LSP `initialize` already includes `clientInfo` and capability negotiation — humans and AI agents can request the subset they want. The server holds a single source of truth for analyses.
- If a future feature is genuinely IDE-rendering-shaped (e.g., inlay hints with markdown), it can be opt-in via initialization options without forking the server.

**Decision 4 — Reuse audit:**
- **Reuse with thin adapter** (low cost):
  - `internal/types` — produces inferred types per binding; needs a position-keyed view
  - `internal/elaborate` — produces post-elaboration AST with resolved references; backs `definition` and `references`
  - `internal/iface` — exported interfaces per module; backs cross-module navigation
  - `internal/pipeline` — full compile-once-per-document driver; backs diagnostics
- **Genuinely new work** (must be sized honestly):
  - **Position-to-symbol index**: from a `(file, line, col)` cursor, find the AST node and its inferred type. Today the AST carries position info but there's no efficient lookup by cursor — this is the bulk of the new code.
  - **Incremental re-typecheck on `didChange`**: full re-pipeline per keystroke is too slow on 50k LOC. MVP can ship with full re-typecheck and `didSave`-only diagnostics — the smarter incremental path is a separate milestone.
  - **LSP wire-protocol plumbing**: JSON-RPC framing, request routing, lifecycle. Use an existing Go LSP library (e.g., `go.lsp.dev/protocol`) rather than rolling our own.

**Decision 5 — Distribution:**
- Ship the plugin in this repo's `.claude-plugin/marketplace.json` (the `ailang-tools` marketplace) alongside `ailang-go-lsp`. Plugin name: `ailang-lsp` (no namespace prefix needed inside our own marketplace). File-extension binding: `.ail` → language id `ailang`.
- Defer publication to a public marketplace until the server has shipped at least one milestone of bug-fix iteration internally.
- **Packaging gotcha learned from `ailang-go-lsp` (commit `235a5afc`):** the Claude Code docs list LSP config as valid either inline in `plugin.json` under `lspServers` OR in a separate `.lsp.json` at the plugin root. Empirically, the `.lsp.json` form is what Claude Code actually consumes — the inline form does not load reliably. Ship `ailang-lsp` with `.lsp.json` and a metadata-only `plugin.json`, mirroring the layout in `.claude/plugins/ailang-go-lsp/`.

## Solution Design (Sketch — committee may revise)

### Overview

`ailang lsp` subcommand starts a Language Server Protocol server on stdio (default) or a socket. On `initialize`, it negotiates capabilities; on `textDocument/didOpen` and `didSave`, it runs the existing pipeline on the document; analyses are stored in an in-memory map keyed by URI; capability handlers project from that map.

### Architecture (sketch — final shape pending committee on Decisions 2 & 4)

```
ailang lsp (stdio)
   │
   ├── JSON-RPC framing (go.lsp.dev/jsonrpc2)
   │
   ├── handler: initialize
   │     └── advertise: diagnostics, definition, references, hover, documentSymbol
   │
   ├── handler: textDocument/didOpen, didSave
   │     └── pipeline.Compile(uri.Path) ──► AnalysisResult
   │                                         │
   │                                         ├── typed AST   (from internal/elaborate)
   │                                         ├── type env    (from internal/types)
   │                                         ├── iface       (from internal/iface)
   │                                         └── diagnostics (from pipeline errors)
   │     └── publishDiagnostics(uri, analysis.diagnostics)
   │
   ├── handler: textDocument/definition
   │     └── positionIndex.lookup(uri, pos) → declaration node → file+range
   │
   ├── handler: textDocument/references
   │     └── for-each-uri-in-workspace: positionIndex.referencesOf(symbol)
   │
   ├── handler: textDocument/hover
   │     └── typeEnv.lookup(uri, pos) → "type: %s, effects: %s"
   │
   └── handler: textDocument/documentSymbol
         └── walk typed AST → symbols
```

### Implementation Plan (Sketch)

Concrete tasks deliberately unstated until committee picks Decisions 1–5; sprint-planner sizes after.

Indicative phases the committee should expect to budget for:
1. **Wire-protocol skeleton** — `ailang lsp` subcommand + JSON-RPC + `initialize`/`shutdown` lifecycle.
2. **Diagnostics-only MVP** — `didOpen`/`didSave` drive `pipeline.Compile`; publish typechecker errors as diagnostics. *This alone closes the largest token-cost gap.*
3. **Position index + hover** — first capability that needs the position-to-symbol indexing work.
4. **Definition + references** — reuses position index + iface.
5. **Document symbols** — straightforward AST walk.
6. **Claude Code plugin packaging** — add entry to `.claude-plugin/marketplace.json`, file-extension binding, install instructions in CLAUDE.md.

### Files Likely Modified/Created

**New files (likely):**
- `cmd/ailang/lsp.go` — subcommand entry point
- `internal/lsp/server.go` — LSP server core
- `internal/lsp/handlers.go` — per-capability handlers
- `internal/lsp/index.go` — position-to-symbol index
- `.claude/plugins/ailang-lsp/.claude-plugin/plugin.json` — Claude Code plugin **metadata** (name, version, description)
- `.claude/plugins/ailang-lsp/.lsp.json` — Claude Code plugin **LSP config** (command, args, extensionToLanguage). Two files, mirroring `ailang-go-lsp`'s layout — see Decision 5 for the reason this is split.

**Modified files (likely):**
- `cmd/ailang/main.go` — register `lsp` subcommand
- `cmd/ailang/help.go` — document the new subcommand (per [`.claude/rules/`](../../../.claude/rules/) CLI annotation discipline)
- `.claude-plugin/marketplace.json` — add `ailang-lsp` plugin entry
- `docs/docs/guides/lsp.md` — new user-facing guide (one for humans, with a section for AI-agent users)
- `CLAUDE.md` — install instructions appended to the existing LSP section
- `changelogs/v0.10-current.md` — `[Unreleased]` entry

## Examples

### Example 1: AI agent editing a `.ail` file gets diagnostics-on-save

**Before this milestone:**
```
[agent] Edit examples/billing.ail (introduces a type error)
[agent] -- has no way to know there's an error until --
[agent] Bash: ailang check examples/billing.ail
[agent] -- 2 seconds, 300 tokens of output to read --
```

**After this milestone:**
```
[agent] Edit examples/billing.ail (introduces a type error)
[harness] ailang-lsp publishes diagnostic via Claude Code's LSP integration
[agent] receives: { file: examples/billing.ail, line: 42, severity: error,
                    message: "expected Int, got String" }
[agent] -- structured, immediate, ~30 tokens --
```

### Example 2: Cross-module go-to-definition

```
[agent cursor at] examples/handler.ail:18:14   (inside `freePlanFromCatalog(...)`)
[agent issues]    textDocument/definition
[server returns]  pkg/sunholo/billing_entitlements/plan.ail:7:6
[agent reads]     just that one file, just that one range
```

vs. today's grep-and-read flow which reads ~3 candidate files to disambiguate.

### Example 3: Hover for type + effect row

```
[agent cursor at] examples/main.ail:12:8       (inside `read_config(path)`)
[agent issues]    textDocument/hover
[server returns]  type: string -> Result[Config, Error]
                  effects: {FS}
                  doc: (none)
```

The effect row is exposure that no other AILANG tool currently gives at navigation cost.

## Success Criteria

- [ ] `ailang lsp --stdio` accepts the LSP `initialize` handshake and advertises the chosen MVP capabilities
- [ ] `didOpen` + `didSave` on a well-typed `.ail` file produces zero diagnostics
- [ ] `didSave` on a `.ail` file with a known type error produces a structured diagnostic with file+range+message
- [ ] `textDocument/definition` resolves at least one cross-module identifier in a fixture project
- [ ] `textDocument/references` returns at least one cross-module reference list in the same fixture
- [ ] `textDocument/hover` returns the inferred type and effect row for an identifier with known type
- [ ] `textDocument/documentSymbol` lists top-level definitions in a `.ail` file
- [ ] `.claude-plugin/marketplace.json` has an `ailang-lsp` entry; `/plugin install ailang-lsp@ailang-tools` succeeds
- [ ] CLAUDE.md install instructions updated
- [ ] `docs/docs/guides/lsp.md` exists and documents both human and AI-agent use
- [ ] `changelogs/v0.10-current.md` `[Unreleased]` entry added
- [ ] Token-cost eval (LSP-on vs LSP-off, same task, same model) shows measurable reduction; baseline numbers captured in the implementation report

## Testing Strategy

**Unit tests** (`internal/lsp/*_test.go`):
- `initialize` advertises only the MVP capabilities (not the deferred ones)
- Position index correctly resolves cursor → AST node on a known fixture
- `hover` projects type + effect row from `internal/types` output
- Diagnostic mapping: pipeline error → LSP `Diagnostic` with correct severity, range, source

**Integration tests** (new — `cmd/ailang/lsp_test.go`):
- Launch `ailang lsp --stdio` as a subprocess, drive it with hand-rolled LSP messages
- Fixture: a 2-module project where `definition` must cross a module boundary
- Fixture: a `.ail` file with a deliberate type error → `publishDiagnostics` arrives within budget

**End-to-end token-cost eval:**
- Reuse the `ai-coding-lang-bench` harness
- One scenario: same task, same model, two configs (LSP plugin installed vs not)
- Compare median tokens-to-completion across N trials
- Report in implementation report, not as a gating criterion (effect size unknown until measured)

## Deferred Decisions

- **Completion, signature help, code actions, formatting, rename, semantic tokens, code lens, inlay hints.** All explicitly deferred. Track as separate milestones (`M-AILANG-LSP-HUMAN-IDE-*`) if a human-IDE use case materializes.
- **Incremental re-typecheck on `didChange`** (per-keystroke) — MVP runs full pipeline on `didSave` only. Incrementality is a separate optimization milestone.
- **Workspace-wide indexing** — references-across-workspace requires scanning all `.ail` files at startup; MVP can scope to "files opened in the session" with a documented limitation, and the workspace-scan path becomes its own follow-up.
- **Cross-process server caching** — multiple Claude Code sessions on the same workspace would each spawn their own `ailang lsp`. Sharing a server via socket is a future optimization.
- **Telemetry/tracing of LSP requests** — interesting for understanding agent navigation patterns, but not MVP.

## Non-Goals

- **This is NOT an AILANG language redesign.** Zero changes to syntax, type system, or runtime semantics. Pure exposure surface.
- **This is NOT human IDE polish.** No completion menus, no rename refactors, no code lens, no hover-Markdown rendering work. If a human IDE wants those, that's a follow-up milestone with its own design doc.
- **This is NOT a multi-milestone effort.** The MVP capability set is deliberately scoped to fit in one milestone. If sprint-planner's sizing comes back as multi-milestone, that's the signal to cut scope further, not to expand the milestone.
- **This is NOT re-doing serve-api.** `ailang serve-api` is the HTTP REST gateway with `@route` semantics. LSP is a separate wire protocol with separate concerns. Touching `serve-api` is out of scope (though see "Related Documents" below — both touch the broader question of "what surfaces does AILANG expose, to whom").

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Position-to-symbol index is harder than estimated (the AST may not carry sufficiently fine-grained positions for sub-token lookup) | Blows up milestone budget | Spike the index against the existing AST early; if positions are insufficient, surface as a separate "M-PARSER-POSITION-FIDELITY" milestone and ship the LSP without `definition`/`references` in the first cut |
| Full pipeline on every `didSave` is too slow on 50k+ LOC | UX regression for AI agents (saves blocking) | MVP ships with single-file scope; workspace-scale incrementality is a separate milestone. Document the limit. |
| Committee picks a packaging different from the author's recommendation (e.g., standalone binary) | Implementation re-plumb | Cheap to revise — packaging touches `cmd/ailang/` only, not internals. Accept whatever the committee chooses. |
| AI agents using the LSP develop reliance on it that we then can't break | Constraint on future LSP changes | Treat the LSP capability set as a stable public surface from day one. Document the contract explicitly. |
| LSP request rate from a session exceeds expected; CPU cost surprises us | Operator-visible | Add a `--max-rps` flag deferred to first observability pass; don't pre-optimize. |

## Best Practices Push

Once shipped, the AILANG agent-facing prompts (`prompts/*`) should be updated to mention that an LSP is available — agents that don't know to use it won't get the token savings. This is a docs/prompts push, not a code change in this milestone.

## Related Documents

- **[M-SERVEAPI-SURFACE-DROPS](m-serveapi-surface-drops.md)** (this version) — adjacent in concept: both define what surfaces AILANG exposes externally. Surface-drops is about *runtime HTTP surface*; this doc is about *static-analysis IDE surface*. The two share no implementation overlap but together define AILANG's "what does the outside world see" boundary. The [M-SERVEAPI-SURFACE-DROPS resolution principle](m-serveapi-surface-drops.md#problem-statement) — "no silent fallbacks at system boundaries" — applies here too: an LSP that silently returns empty results for a query it doesn't understand is the wrong shape; structured `null` with an error is the right shape.
- **[M-CODEGEN-STRATEGIC-REVIEW](../../implemented/v0_11_0/m-codegen-strategic-review.md)** (shipped, v0.11.0) — pattern reference: the codegen strategic review asked the design committee for scope decisions before code; this doc follows that pattern explicitly for Decisions 1–5.
- **[Claude Code LSP plugin spec](https://code.claude.com/docs/en/plugins-reference#lsp-servers)** — the external standard this milestone targets.
- **[`.claude/plugins/ailang-go-lsp/`](../../../.claude/plugins/ailang-go-lsp/)** (shipped, commit `235a5afc`) — the proof-of-workflow plugin that validated the Claude Code LSP integration end-to-end on the Go side of this repo. Its successful operation is part of the evidence this doc rests on. Empirical packaging finding (LSP config must live in `.lsp.json`, not inline in `plugin.json`) is folded into Decision 5 above.

## Conflict Surface

This milestone does **not** touch `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go` in a way that changes their semantics. It **reads** their outputs through the existing public APIs. No new conflict surface is introduced.

The one indirect risk: if `internal/types` or `internal/elaborate` change the shape of their public outputs in a future milestone, the LSP adapter breaks. Mitigation: treat the LSP adapter as a consumer with the same stability expectations as any other production consumer of those packages (i.e., the type checker public API stability discipline applies).

## DESIGN_DOC_PATH

`design_docs/planned/v0_21_0/m-ailang-lsp-for-ai.md`
