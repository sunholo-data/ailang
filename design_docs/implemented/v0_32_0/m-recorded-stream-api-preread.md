# PRE-READ: m-recorded-stream-api (ailang#546) — attended evaluation of the offered implementation

**Status**: Pre-read banked 2026-07-31 (attended, Fable session + zero-model mechanical checks)
for the Monday 2026-08-03 iteration that owns this item. NOT a quorum verdict — evidence to
start from, per the provenance rule: everything below states HOW it was verified.
**Reference implementation**: https://github.com/arniwesth/ailang/pull/2
(branch `arniwesth/spike/motoko-009-prototype-v031`, fetched as remote `arniwesth`,
worktree `.claude/worktrees/arni-546-eval` left in place for the iteration).

## Mechanical verification (run first-party on the rig, 2026-07-31)

- `go build ./...` — **OK** (the `cmd/wasm` "main is undeclared" error reproduces IDENTICALLY
  on our dev HEAD → pre-existing build-tag artifact, NOT the branch's).
- `go test ./internal/builtins/ ./internal/effects/` — **all 4 new tests PASS**
  (success path · error path · delta-concat parity · `stepWithStream` unchanged-by-variant).
  The one failure seen (`TestNetHttpPost`, 0.37s fast-fail) **passed on -count=1 re-run** and
  passes on dev — transient network flake (eval suite was running concurrently), not the branch.
- Diff shape: +452 lines, 5 files, **purely additive** — no existing line modified.

## Design read (Fable review of the full diff)

**API**: sibling `stepWithStreamRecorded` returning `RecordedStream = { chunks: [StreamChunk],
outcome: Result[StepResult, AIError] }`; callback keeps `(StreamChunk) -> () ! {IO}` — no
effect-row widening; existing `stepWithStream` untouched (and tested untouched).

**Judged RIGHT, adopt as-is:**
1. Additive sibling, not a return-shape change to the stable surface — honors the 1.x promise.
2. `{chunks, outcome}` rather than `Result[{result, chunks}, err]` — the rejected shape would
   discard chunks observed before a mid-stream failure, which is the case DST replay most
   depends on. Arni's comment articulates this correctly; the error-path test proves it.
3. Handler mirrors the house patterns: arg-validation/decode error classes, `RecordAIEffect`
   tracing (with chunk counts), `classifyOpError`, fail-soft callback errors recorded to trace.
4. Record-append happens BEFORE the callback call → record order = arrival order regardless of
   callback behavior.

**Productionization asks for the adoption sprint (none blocking the design):**
1. **Dedupe** (the real work): ~80 lines of arg-validation + decode are copy-pasted from
   `aiStepWithStream`. Refactor both onto one shared core (recorded variant wraps it) or the
   two will drift. This is why the spike is labelled "never merges" — honor that label by
   productionizing, not cherry-picking.
2. Repo completeness per house rules: `examples/ai_streaming_recorded.ail` (every feature needs
   an example), teaching-prompt + μRAG entry (verify claims with `ailang check` first),
   CHANGELOG, docs site, `Since: prototype` → real version (keep `StabilityExperimental`).
3. LongDesc memory note: retention is unbounded, ∝ stream length (fine for agent turns; say so).
4. Minor: `chunkCount` increments on encode-nil skips while `recorded` doesn't — the two can
   diverge in trace text only. Cosmetic; align or document.
5. Check the bytecode-VM bridge story for the new op matches `stepWithStream`'s (same effects
   registry path — expected uniform, verify not assumed).

**Routing call for the quorum**: this is a `std/ai` + builtins + effects change → core surface,
NOT an extension — but it is additive, experimental-marked, and demanded by a real external
consumer with the strongest evidence class. Recommend: ADOPT with the productionization list,
authorship credited to @arniwesth in the commit.

**Design context the quorum must read** (Arni, Discord 2026-07-31): the two DST ADRs —
Project 009 (deterministic test-world architecture) and Project 007 (DST definition/taxonomy),
links in the charter queue row and ailang#546.
