# M-EVAL-TRUST-SIGNALS: External Trust Signals for AILANG Eval Results

**Status**: Planned
**Target**: v0.15.x (HumanEval port + receipts) → v0.16.x (limitations page)
**Priority**: P1 (Medium-High — gating credibility for AILANG adoption)
**Estimated**: ~3 weeks total (1 week HumanEval port, 1 week receipts + MCP, 1 week limitations page)
**Dependencies**: Existing eval suite (M-EVAL-SUITE-PREP, v0.14.0), MCP server (M-AGENT-MCP, v0.15.0), benchmark dashboard

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is meta-tooling about *evidence*, not language semantics. Most axioms are neutral; the ones that score positive concern reproducibility (A1) and machine-readable provenance (A7).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Reproducibility receipts make eval runs deterministic and re-executable from a manifest |
| A2: Replayability | +1 | Public transcript publishing means any third party can replay a run end-to-end |
| A3: Effect Legibility | 0 | No language-level effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Pre-registered spec + provenance hash is locally verifiable without trusting our infra |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | MCP responses gain `methodology_url`, `prompt_hash`, `runner_version`, `reproducible` fields agents can weight automatically |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Receipts include exact token/cost ledger per benchmark, anchored to model+temp+seed |
| A10: Composability | 0 | Composes with existing eval/MCP infra; no new abstractions |
| A11: Structured Failure | +1 | "What AILANG can't do" page is a typed, machine-readable failure catalog (not free-form prose) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Receipts strengthen determinism, no implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects (publishing is explicit `IO+Net` in the runner)
- [x] A4 (Authority): Transcript publishing requires explicit credentials; no ambient access
- [x] A7 (Machines First): Designed for machine consumption first (MCP fields, JSON manifests)

## Problem Statement

AILANG's in-house eval suite is **technically stronger** than most external benchmarks for AILANG's actual goals — it measures MCP-attached vs unattached behaviour, token reduction, agent vs single-shot harnesses, and runs against 4 frontier models on 22+ benchmarks. The in-house suite **remains the primary signal**. But it suffers from a **provenance gap**: when an LLM agent or a human reviewer encounters a claim like "AILANG passes 61% of benchmarks on claude-sonnet-4-6", they have no independent way to weight it, and there's nothing to anchor that number to a benchmark they already trust.

The fix is **not** to convert or replace the in-house eval. It's to:
1. Add an externally-recognisable benchmark (HumanEval-164) that runs **in parallel** with the existing suite, giving a small comparable headline.
2. Add reproducibility/provenance machinery to **all** benchmarks (in-house + HumanEval) so any reader can independently verify any claim.

**Current State:**
- Eval results live in `eval_results/` and the dashboard, but the methodology is implicit (encoded in code, not a citable spec).
- No externally-recognisable comparable number — readers can't anchor AILANG's numbers to anything they already trust.
- Transcripts are not consistently public; reproducing a run requires our infrastructure.
- The MCP `benchmarks_*` tools return scores but no provenance metadata, so a downstream agent can't decide how much to trust them.
- Failure modes are scattered across `docs/LIMITATIONS.md`, `prompts/`, and individual benchmark notes — there's no single canonical "what AILANG cannot currently do" page derived from data.

**Impact:**
- **AI agents** (including Claude Code itself when reasoning about AILANG) discount in-house claims because they cannot verify them. This is correct behaviour from the agent — and it caps how confidently any agent can recommend AILANG.
- **Human reviewers** (researchers, prospective users, eval-curious developers) bounce off because there's no apples-to-apples number against Python/Rust/etc.
- **Sales/positioning** lacks a one-line credibility anchor ("AILANG: X% on HumanEval; Python: Y%").
- The asymmetry between in-house quality and external legibility means the project's actual eval rigour is invisible.

## Goals

**Primary Goal:** Make AILANG's existing high-quality eval data externally verifiable, and add **one additional** externally-recognisable benchmark that runs alongside it — without disturbing the in-house eval suite, which remains the primary signal.

**Success Metrics:**
- HumanEval-164 runs as a **parallel** benchmark in the eval suite (additive, not replacing anything) — yields one externally-comparable number per release.
- Any third party can run `ailang eval reproduce v0.15.x` against published manifests and reach within ±2pp of the published result on the same model+temperature, **for any benchmark** (in-house or HumanEval).
- MCP `benchmarks_*` responses ship provenance metadata (`methodology_url`, `prompt_hash`, `runner_version`, `reproducible: true`) for all benchmarks within the same release.
- A canonical `docs/limitations` page exists, **derived from all eval data** (in-house + HumanEval), and updates automatically when a new release is evaluated.
- An external researcher (paid engagement, ~$2-5k) reruns one release's eval suite and publishes a 2-page report.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| HumanEval translation methodology (mechanical vs idiomatic) | Affects comparability — too literal hurts AILANG, too idiomatic invites "you cherry-picked" | human | design | high |
| Transcript hosting (HuggingFace Datasets vs GCS bucket vs git-LFS) | Affects discoverability and long-term cost; HF gives free academic surface area | human | design | med |
| What "consistent failure" threshold defines a documented limitation (3+ runs? all models? 2 of 4?) | Determines what lands on the "can't do" page; too loose = noise, too tight = missing real gaps | human | design | med |
| Pre-registered spec format (YAML schema) | Spec must be diffable, hashable, and round-trippable through the runner | agent | design | med |
| Whether to translate HumanEval prompts at all (vs only translate signatures + tests) | Pure-test translation is more defensible academically; full prompt translation matches actual usage | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] HumanEval translation methodology decided and documented
- [ ] Transcript hosting target chosen (recommendation: HuggingFace Datasets — free, citable, academic credibility)
- [ ] "Consistent failure" threshold defined (recommendation: failure on ≥3 of 4 frontier models across ≥2 runs)
- [ ] Pre-registered spec YAML schema reviewed and frozen

## Solution Design

### Overview

Four components, deliberately decoupled so each can ship independently. The in-house eval suite is **untouched** — these are additions, not replacements:

1. **HumanEval-AILANG port** — 164 problems translated, added as one more benchmark suite the runner can execute (`ailang eval --suite humaneval`). Runs in parallel with the existing 22 benchmarks, doesn't displace them.
2. **Reproducibility receipts** — applies to **all** benchmarks (in-house + HumanEval). `ailang eval reproduce <manifest>` command + public manifest publishing + MCP provenance fields.
3. **Pre-registered eval specs** — `eval_results/v<version>/spec.yml` committed before any model run; covers whatever suite is being run that release.
4. **"What AILANG can't do" page** — auto-generated from consistent failures across the **whole eval corpus** (in-house + HumanEval), lives at `docs/limitations/`.

### Architecture

**Components:**

1. **HumanEval Port (`benchmarks/humaneval/`)** — added as a parallel suite, not a replacement
   - 164 problems as `.ail` skeletons + test harnesses, sitting alongside existing `benchmarks/`
   - Translation log documenting every non-trivial mapping (e.g., Python list comprehension → AILANG `map`/`filter`)
   - Wired into existing runner via `--suite humaneval` flag (or runs as part of `--all`)
   - Comparable Python baseline run with the same model/temp on the original HumanEval problems
   - One additional headline on the dashboard: `AILANG: X/164 (Y%); Python: A/164 (B%); model: claude-sonnet-4-6; temp: 0.0` — sits next to existing in-house numbers, doesn't replace them

2. **Reproducibility Receipts (`internal/eval_harness/receipt.go`, `cmd/ailang/eval_reproduce.go`)**
   - Each eval run emits a receipt JSON containing: spec hash, prompt hashes, model+temperature+seed, ailang version, runner git SHA, total tokens, per-benchmark transcript URI
   - `ailang eval reproduce <manifest-url>` command: fetches a published manifest, replays it locally, diffs results
   - Transcript publisher: serialises (prompt, model output, judge verdict, error if any) per benchmark to JSONL, uploads to HuggingFace Datasets repo `ailang/eval-transcripts-v<version>`

3. **Pre-Registered Specs (`eval_results/v<version>/spec.yml`)**
   - Schema: list of benchmarks, models, temps, expected difficulty tiers, judging rubric, retry policy
   - Workflow rule: spec file must be committed *before* any model run produces results in that folder; CI gate verifies `git log --diff-filter=A spec.yml` precedes the first results JSON
   - Spec hash flows into every receipt produced under that version

4. **MCP Provenance Fields (`internal/mcp/benchmarks_*.go`)**
   - Every benchmark response gains: `methodology_url`, `prompt_hash`, `runner_version`, `reproducible: bool`, `transcript_uri`
   - Agents querying via MCP can weight results by provenance without follow-up calls

5. **Auto-Generated Limitations Page (`tools/gen_limitations.go`, `docs/docs/limitations/`)**
   - Reads consistent failures across the last N releases
   - Groups by failure category (parse error, type error, missing stdlib, semantic gap)
   - Generates Markdown with code examples drawn from actual failed transcripts
   - Each entry links to the originating transcript so readers can verify

### Implementation Plan

**Phase 1: HumanEval Port** (~40 hours, 1 week)
- [ ] Decide translation methodology + write 1-page methodology note
- [ ] Translate first 20 problems as a pilot, sanity-check approach
- [ ] Translate remaining 144 problems
- [ ] Build runner: `cmd/ailang/eval_humaneval.go`
- [ ] Run baseline: AILANG vs Python on claude-sonnet-4-6 + claude-opus-4-7 + gemini-3-flash + gpt-5
- [ ] Publish raw transcripts to HuggingFace Datasets
- [ ] Add headline to dashboard

**Phase 2: Reproducibility Receipts + MCP Provenance** (~30 hours, 1 week)
- [ ] Define receipt JSON schema (`internal/eval_harness/receipt.go`)
- [ ] Wire receipt emission into existing eval runner
- [ ] Implement `ailang eval reproduce <manifest>` command
- [ ] Add provenance fields to all `mcp__ailang-docs__benchmark*` tools
- [ ] Add CI gate: spec.yml must precede results in git history
- [ ] Test reproduce flow end-to-end against a published v0.15.x manifest

**Phase 3: Limitations Page Auto-Generator** (~30 hours, 1 week)
- [ ] Define "consistent failure" detector (`tools/gen_limitations.go`)
- [ ] Categorise failure modes (parse, type, runtime, missing stdlib, semantic gap)
- [ ] Generate `docs/docs/limitations/index.md` + per-category pages
- [ ] Wire into post-release skill so page regenerates each release
- [ ] Replace existing hand-maintained `docs/LIMITATIONS.md` with auto-generated content (keep it as a cover page that links to the new structure)

**Phase 4 (optional): External Witness Run** (~$2-5k, 1 week elapsed)
- [ ] Identify 1-2 academic or independent eval researchers
- [ ] Provide manifest + reproduce instructions
- [ ] They run, publish 2-page report; cross-link from dashboard

### Files to Modify/Create

**New files:**
- `benchmarks/humaneval/<problem-id>.ail` × 164 — Translated problems (~3000 LOC total)
- `benchmarks/humaneval/methodology.md` — Translation methodology + per-problem decisions log
- `benchmarks/humaneval/runner_test.go` — Local test that 164 skeletons type-check
- `cmd/ailang/eval_humaneval.go` — HumanEval-specific runner (~200 LOC)
- `cmd/ailang/eval_reproduce.go` — `ailang eval reproduce` command (~250 LOC)
- `internal/eval_harness/receipt.go` — Receipt schema + emission (~200 LOC)
- `internal/eval_harness/transcript_publisher.go` — HuggingFace upload (~150 LOC)
- `tools/gen_limitations.go` — Limitations page generator (~400 LOC)
- `eval_results/v0_15_x/spec.yml` — First pre-registered spec (template)
- `docs/docs/limitations/index.md` — Auto-generated cover page

**Modified files:**
- `internal/mcp/benchmarks_compare.go`, `benchmarks_for_model.go`, `benchmark_run.go` — Add provenance fields (~100 LOC across files)
- `internal/eval_harness/runner.go` — Emit receipts at run completion (~50 LOC)
- `.github/workflows/eval-suite.yml` — Spec-precedes-results CI gate (~30 LOC)
- `.claude/skills/post-release/` — Wire limitations regeneration into release workflow
- `docs/docs/benchmarks/` — Add HumanEval headline section

## Examples

### Example 1: Dashboard with both signals side-by-side

**Before** (current state — only in-house, no externally-comparable number):

> AILANG eval suite: 61.4% pass rate (216/352) across 22 benchmarks × 4 models × 4 harnesses.

**After** (in-house remains primary, HumanEval added alongside):

> **AILANG eval suite (primary signal): 61.4% (216/352)** across 22 benchmarks × 4 models × 4 harnesses, with MCP-attached vs unattached comparisons. [Dashboard]
>
> **HumanEval-164 (additional comparable anchor, claude-sonnet-4-6, temp 0.0):**
> AILANG: 89/164 (54.3%) · Python: 142/164 (86.6%) · Gap: 32.3pp
>
> Both runs reproducible: `ailang eval reproduce v0.15.0` · Transcripts: [HuggingFace] · Methodology: [link]

The in-house number stays prominent; HumanEval is the secondary anchor that gives a reader who's never heard of AILANG something to weight against.

### Example 2: MCP response with provenance

**Before:**
```json
{
  "model": "claude-sonnet-4-6",
  "benchmark": "log_file_analyzer",
  "result": "pass",
  "tokens": 1247
}
```

**After:**
```json
{
  "model": "claude-sonnet-4-6",
  "benchmark": "log_file_analyzer",
  "result": "pass",
  "tokens": 1247,
  "methodology_url": "https://github.com/sunholo/ailang/blob/v0.15.0/eval_results/v0_15_0/spec.yml",
  "prompt_hash": "sha256:8a4f...",
  "runner_version": "v0.15.0",
  "reproducible": true,
  "transcript_uri": "hf://datasets/ailang/eval-transcripts-v0.15.0/log_file_analyzer.jsonl#L42"
}
```

A downstream agent can now decide: "transcript exists + reproducible flag = high trust; weight 1.0" vs "no transcript = weight 0.5".

### Example 3: Auto-generated limitations entry

```markdown
## Y-Combinator and Recursive Lambdas

**Failure rate**: 4/4 models, 2/2 runs (consistent)
**Category**: Type system limitation
**First documented**: v0.1.0

### What fails

\`\`\`ailang
let Y = \f. (\x. f(x(x)))(\x. f(x(x))) in ...
\`\`\`

**Error**: `occurs check failed: type variable α occurs in (α → β)`

### Why

Hindley-Milner type inference prevents infinite types to ensure decidability.

### Workaround

Use named recursive functions; see [pattern guide].

### Source transcripts
- [claude-sonnet-4-6, 2026-04-27](hf://datasets/ailang/eval-transcripts-v0.15.0/y_combinator.jsonl)
- [gpt-5, 2026-04-27](hf://datasets/ailang/eval-transcripts-v0.15.0/y_combinator.jsonl)
```

## Success Criteria

- [ ] HumanEval-164 ported, all 164 problems type-check (no syntax errors)
- [ ] At least 3 frontier models scored on AILANG-HumanEval and Python-HumanEval with same prompts/temps
- [ ] Comparable headline number published on dashboard + README
- [ ] Raw transcripts published to HuggingFace Datasets, public, citable
- [ ] `ailang eval reproduce <manifest>` reproduces a published run within ±2pp
- [ ] All MCP `benchmarks_*` tools include the 5 provenance fields
- [ ] CI gate enforces spec.yml precedes results.json in git history
- [ ] `docs/docs/limitations/` auto-generates from consistent failures, replacing hand-curated page
- [ ] post-release skill regenerates limitations page automatically
- [ ] (Stretch) External researcher run + report linked from dashboard

## Testing Strategy

**Unit tests:**
- Receipt schema round-trips (serialise → deserialise → equal)
- Provenance fields populated on every MCP response
- Limitations generator handles 0/1/many transcripts per failure correctly

**Integration tests:**
- End-to-end: run eval → receipt published → reproduce command pulls + reruns → results match
- CI gate test: PR adding results without prior spec.yml fails the gate

**Manual testing:**
- Pick 3 random HumanEval problems, eyeball the AILANG translation against the Python reference
- Verify HuggingFace dataset is readable by `datasets.load_dataset("ailang/eval-transcripts-v0.15.0")`
- Read auto-generated limitations page and check examples render correctly

## Deferred Decisions

- **HumanEval translation per-problem details** — agent may choose idiomatic AILANG within the methodology rules, with each non-trivial choice logged in `methodology.md`
- **Receipt JSON exact field ordering** — agent may choose; only schema validity matters
- **HuggingFace dataset card content** — agent may draft, human reviews before first publication
- **Limitations page categorisation taxonomy** — agent may propose, human approves the top-level categories
- **External researcher selection** — human decides; deferred to Phase 4

## Non-Goals

- **Aider polyglot or LinuxArena integration** — assessed separately, ruled out as poor fit (see prior conversation).
- **Full MultiPL-E adoption** — only HumanEval-164, not MBPP or other tasks; scope discipline.
- **arXiv paper** — methodology page on the docs site is sufficient; paper is future work.
- **Live external leaderboard** — dashboard already serves this need; chasing a third-party leaderboard is amplification, not foundation.
- **Retroactive receipts for v0.14.x and earlier** — receipts begin at v0.15.0 onwards; older runs remain in their current form.
- **Replacing, converting, or restructuring the in-house eval suite** — HumanEval runs *alongside* the existing 22 benchmarks. None of the existing benchmarks, harnesses, or dashboard sections are touched. The in-house suite remains the primary signal.

## Timeline

**Week 1** (~40 hours):
- Phase 1: HumanEval port + first run + headline published

**Week 2** (~30 hours):
- Phase 2: Receipts + MCP provenance + reproduce command + CI gate

**Week 3** (~30 hours):
- Phase 3: Limitations page auto-generator + post-release wiring

**Week 4 (optional, async)**:
- Phase 4: External researcher engagement, ~$2-5k spend, 1 week elapsed

**Total: ~100 hours across 3 weeks (4 with external witness)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| HumanEval translation invites "you cherry-picked" criticism | High | Mechanical methodology + every non-trivial translation logged in `methodology.md`; transcripts public so any reader can audit |
| AILANG scores significantly lower than Python on HumanEval | Low-Med (reputational) | Risk drops because HumanEval is *additional*, not the headline. In-house numbers stay primary on dashboard. Ship the HumanEval gap with honest framing — "AILANG is ~3 years younger than Python tooling, here's the gap and what we're doing about it"; the credibility win is admitting the gap, not hiding it |
| HuggingFace Datasets policy change / takedown | Low | Mirror to GCS bucket as backup; manifest URLs use a redirect we control |
| Pre-registered spec adds friction to release process | Med | Automate spec.yml generation in `release-manager` skill so it's not a manual step |
| Auto-generated limitations page misses subtle failure modes | Med | Keep a "manual additions" section that doesn't get overwritten; generator only owns the auto-derived part |
| External researcher delay or pulls report | Low | Phase 4 is optional and async; success doesn't depend on it |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_14_0/m-eval-suite-prep.md](../../implemented/v0_14_0/m-eval-suite-prep.md) — Existing eval suite this builds on
- [design_docs/implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md](../../implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md) — Agent eval queue infrastructure

**Planned (check for overlap):**
- [design_docs/planned/v0_15_0/m-eval-cross-harness-comparison.md](m-eval-cross-harness-comparison.md) — Cross-harness comparison; complementary, not overlapping
- [design_docs/planned/v0_15_0/m-eval-results-folder-structure.md](m-eval-results-folder-structure.md) — Folder structure spec.yml will live alongside
- [design_docs/planned/v0_15_0/m-agent-mcp-website.md](m-agent-mcp-website.md) — MCP server gaining provenance fields

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [HumanEval (Chen et al., 2021)](https://arxiv.org/abs/2107.03374) — Original benchmark this port targets
- [MultiPL-E (Cassano et al., 2022)](https://nuprl.github.io/MultiPL-E/) — Reference for cross-language HumanEval translation methodology
- [HuggingFace Datasets](https://huggingface.co/datasets) — Proposed transcript hosting target
- [docs/LIMITATIONS.md](../../../docs/LIMITATIONS.md) — Existing hand-curated page being replaced
- Prior conversation 2026-04-28: assessment of LinuxArena, Aider polyglot, MultiPL-E, SWE-bench as external benchmark options

## Future Work

- **MBPP port** (additional ~150 problems) — once HumanEval port is stable, MBPP is mechanical to add
- **Live external leaderboard** — once 3+ releases worth of receipts exist, build a public-facing scoreboard
- **arXiv methodology paper** — if the limitations page + reproduce command get external citations, formalise into a paper
- **Cross-language MultiPL-E participation** — submit AILANG runner upstream once translation methodology proves stable
- **Differential eval** — automatic detection of regressions across releases using receipts (e.g., "this change broke 3 benchmarks that passed in v0.14.2")

---

**Document created**: 2026-04-28
**Last updated**: 2026-04-28

DESIGN_DOC_PATH: design_docs/planned/v0_15_0/m-eval-trust-signals.md
