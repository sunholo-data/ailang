# M-MOTOKO-EDITDECL-ASTEDIT: AST-span decl-replace tool for motoko (kill large-file rewrite-thrash)

**Status**: Planned (trigger met — see Problem)
**Target**: v0.26.0
**Priority**: P3 → elevated (docx P0 surfaced the rewrite-thrash this addresses)
**Estimated**: 3–5 days (cross-layer: motoko TS + .ail core; ailang CLI already exists)
**Dependencies**: `ailang ast-edit replace` CLI (DONE, 51eff8c92 — verified working). Complements the /v1
timeout fix (motoko_agent#65) and P2 context-compression.

## Axiom Compliance

| Axiom | Score | Rationale |
|---|---|---|
| A1: Determinism | +1 | Span-based decl replace is deterministic; no fragile string-match ambiguity. |
| A2: Replayability | +1 | Edit traces become "replaced decl X" not a 500-line diff → cleaner replay. |
| A3: Effect Legibility | 0 | No effect changes. |
| A4: Explicit Authority | 0 | Same FS authority as existing edit tools. |
| A5: Bounded Verification | +1 | Each edit re-parses one decl; the rest of the file is preserved exactly (re-typecheck is local). |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The model emits ONE decl per edit instead of re-emitting the whole file — far fewer output tokens. |
| A8: Minimal Syntax | 0 | No language syntax; a new tool schema only. |
| A9: Cost Visibility | +1 | Edit cost ∝ decl size, not file size — visible + bounded. |
| A10: Composability | 0 | Coexists with EditFile (intra-decl) / WriteFile (new files). |
| A11: Structured Failure | +1 | ast-edit returns a typed "decl not found" error vs a silent wrong string-replace. |
| A12: System Boundary | 0 | Same ailang-subprocess boundary as other tools. |

**Net Score: +6** → **Decision: Move forward (after the docx grade quantifies the need)**

### Hard Violation Check
- [x] A1 / A3 / A4 / A7 all ≥ 0.

## Problem Statement

motoko's edit tools are line/string based: `WriteFile` (whole-file), `EditFile` (string old/new), `hashline`
(hash-addressed lines). On a LARGE file the model re-emits the entire file (WriteFile) or does fragile
string edits — costly in output tokens and prone to thrash.

**Current State (evidence):** the docx_reimplement P0 run had motoko reimplement a stubbed parser by writing
a **526-line file** (from a 31-line stub). Every subsequent fix to that file risks another large rewrite.
This rewrite-volume compounds the context-bloat that drives the /v1 timeout (each turn re-emits a big file →
big prompt next turn). Task #8 deferred this "until P0 shows rewrite-thrash on long tasks" — **P0 now shows it.**

**Impact:** large-context / long-file tasks (the regime where AILANG-native harnesses should win) are exactly
where full-file rewriting is most expensive. A span-based decl-replace tool is an AILANG-native lever pi's
generic line-edit tools don't have (AILANG decls have clean parse spans + names the model already knows).

## Goals

**Primary Goal:** let motoko replace one top-level declaration at a time by name (parsed span), so editing a
function in a 500-line file costs ~one decl of output, not the whole file.

**Success Metrics:**
- On docx (or a synthetic large-file edit task), median edit output tokens drop ≥3× vs the WriteFile/replace baseline.
- No regression in pass rate (EditDecl edits apply correctly; the rest of the file is byte-preserved).
- Fewer turns-to-success on long-file tasks (less re-read/re-write churn).

## High-Impact Decisions

| Decision | Why it matters | Who | When | Cost |
|---|---|---|---|---|
| EditDecl tool schema: `{path, decl, new_body}` | The model contract; must be obvious to use | human/agent | design | med |
| When the model picks EditDecl vs EditFile vs WriteFile | Prompt guidance — EditDecl for whole-decl rewrites, EditFile for intra-decl, WriteFile for new files | human | design | med |
| New EditMode `astedit` vs always-available tool | Gating vs always-on (auto) | human | design | low |
| Cross-layer wiring: TS dispatcher + .ail tool registry | Tool must appear in the model's schema AND dispatch | agent (impl) | runtime | high |

### Design Freeze
- [ ] EditDecl schema field names.
- [ ] EditMode value (`astedit`) + whether `auto` offers it.
- [ ] Prompt rule for tool selection.

## Solution Design

**New tool `EditDecl`** (replace a top-level decl by name):
- **Schema:** `{ path: string, decl: string, new_body: string }`.
- **Dispatch (TS, `src/tui/src/ohMyPi/dispatcher.ts`):** add an `if (tool === "EditDecl")` branch (~25 LOC,
  matching the existing `spawn`-based pattern): write `new_body` to a tmp file, run
  `ailang ast-edit replace --file <path> --decl <decl> --new <tmp> --in-place`, return the CLI's
  typed result (replaced / decl-not-found).
- **.ail core wiring (`src/core`):** add `EditDeclResult` to `types.ail`; render it in `tool_dispatch_adapter.ail`
  (so the model sees a structured result); register the tool schema where the tool list is built for the model
  (`agent_loop_v2.ail` / `tool_runtime.ail`).
- **EditMode (`session-adapter.ts`):** add `"astedit"`; offer EditDecl when `editMode === "astedit"` (and
  optionally `auto`).
- **Prompt:** teach "to rewrite an entire function/type, use EditDecl(path, decl, new_body); for a small
  in-function change use EditFile; for a brand-new file use WriteFile."

**Files:** motoko fork — `src/tui/src/ohMyPi/{dispatcher,session-adapter}.ts`, `src/core/{types,tool_dispatch_adapter,agent_loop_v2}.ail`, prompt. ailang side: `ailang ast-edit` (already done).

**Non-goals:** intra-decl edits (keep EditFile); multi-decl atomic edits (one decl per call); non-AILANG files.

## Risks / Conflict Surface
- **Cross-layer consistency** (TS schema ↔ .ail registry ↔ dispatcher) — the main implementation risk; mirror an existing tool (EditFile) end-to-end as the template.
- **Model adoption** — the model must choose EditDecl appropriately; needs prompt teaching + the docx trial to confirm it actually uses it (not just that it exists).
- **Whole-decl granularity** — ast-edit replaces the entire decl; a one-line change still re-emits the decl (still far cheaper than the file). EditFile remains for sub-decl edits.

## Success Criteria
- [ ] EditDecl appears in the model's tool schema and dispatches to `ailang ast-edit`.
- [ ] Round-trip: replacing decl X preserves all other decls byte-for-byte (golden test).
- [ ] docx trial (EDIT_MODE=astedit) completes and shows ≥3× lower median edit-output-tokens vs baseline.
- [ ] No pass-rate regression on the core eval tier.
- [ ] Tests + `make verify-examples` green; DRAFT PR to arniwesth/motoko_agent.

## Timeline
- Day 1–2: TS dispatcher EditDecl branch + .ail types/adapter/registry (mirror EditFile).
- Day 3: EditMode + prompt; tsc build; golden round-trip test.
- Day 4–5: docx trial on the rig (after the timeout-fixed baseline grade exists), A/B by edit-token-volume + pass rate; DRAFT PR with evidence.

## Related Documents
- [m-ollama-v1-streaming-idle-timeout](m-ollama-v1-streaming-idle-timeout.md) — sibling large-context lever (timeout); EditDecl shrinks WRITES, that shrinks the per-turn prompt growth, easing the same timeout pressure.
- P2 (context_mode compression) — shrinks READS/context; EditDecl shrinks WRITES. Complementary halves of large-context efficiency.
- `ailang ast-edit` CLI (51eff8c92, internal/astedit) — the engine this tool drives.
