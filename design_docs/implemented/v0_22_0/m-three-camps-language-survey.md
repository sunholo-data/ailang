# M-THREE-CAMPS-LANGUAGE-SURVEY: Gap Analysis Across 16 AI-Designed Languages

**Status**: Planned
**Target**: v0.22.0 (Phase 1: gap-driven benchmark expansion + comparison page; Phase 2: peer-language evals) → v0.23.0+ (Phase 3: audit memos + idea borrowing)
**Priority**: P1 — High (talk deadline 2026-05-25; talk-load-bearing)
**Estimated**: ~50–70 hours for talk-week (Phase 1 + Phase 2); ~30h for Phase 3 post-talk
**Dependencies**:
  - Existing eval harness (`internal/eval_harness/`, `benchmarks/`)
  - Public docs surface (`docs/docs/guides/`)
  - Precedent: [m-zero-language-learnings.md](../v0_21_0/m-zero-language-learnings.md) — same shape, single-language scope

**Commissioning context**: A May 2026 post by Negroni Venture Studios — ["Three Camps Alike in Dignity"](https://negroniventurestudios.com/2026/05/20/three-camps-alike-in-dignity/) — surveys AI-designed programming languages that emerged independently over ~6 months and groups them into three camps: **Syntactic** (fix tokenization), **Verification** (fix correctness contracts), **Orchestration** (fix the loop around the language). The post is being actively edited; current count is **17 named languages** (post says "~20"). AILANG is explicitly listed in the Verification camp.

**Talk-deadline note**: The author of this design doc is giving a talk on AI languages **the week of 2026-05-25**. The deliverable is a public comparison page + a gap-driven expansion of the eval suite that probes the camps' hypotheses empirically.

---

## The Core Reframe (Why This Doc Is Gap-Analysis-First)

The original draft of this doc focused on adding peer languages to AILANG's existing eval suite. **That's the wrong end of the stick.** Each peer language exists because its authors believe a specific failure mode of LLM codegen is critical — *and our existing benchmarks don't probe most of those failure modes*. Running peer languages on our current smoke tier would measure "can the LLM write FizzBuzz in language X" — which tells us nothing the survey doesn't already.

The sharper move:

1. **Map each language's distinguishing capability to a benchmark gap in AILANG's suite.**
2. **Add the gap benchmarks** — they probe testable hypotheses about *why* each camp exists.
3. **Run AILANG on them first** as a self-audit — which gaps does AILANG already cover, which surface real improvements we should make?
4. **Then add peer languages** to the relevant subset of benchmarks (verification-camp peers on contract-tier; FP peers on smoke; orchestration peers on AI-effect benchmarks).

This produces a stronger talk story: not "AILANG vs the field" but "here's the empirical map of what each camp's hypothesis actually claims, tested on a shared benchmark grid."

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](../../../docs/docs/references/axioms.mdx)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | New `canonical_convergence` benchmark explicitly tests A1 |
| A2: Replayability | +2 | New `audit_chain_replay` benchmark explicitly tests A2 |
| A3: Effect Legibility | +1 | New `ai_effect_*` benchmarks surface our biggest unique capability |
| A4: Explicit Authority | +1 | New `unauthorized_fs_refused` benchmark tests capability gating |
| A5: Bounded Verification | +2 | Contract-tier expansion (`shadowing_heavy_contract`) stresses Z3 path |
| A6: Safe Concurrency | +1 | New `parallel_map_reduce` benchmark tests Fork/Call/Done |
| A7: Machines First | +2 | Public comparison page; benchmarks designed to be agent-readable |
| A8: Minimal Syntax | 0 | No language-surface changes |
| A9: Cost Visibility | 0 | N/A |
| A10: Composability | +1 | New typed-pipeline benchmark stresses composition |
| A11: Structured Failure | +1 | Decision-block benchmark formalizes failure rationale capture |
| A12: System Boundary | 0 | N/A |

**Net Score: +12** → **Decision: Proceed**

### Hard Violation Check

No violations.

---

## The 16 Languages, Grouped by Camp

(Unchanged from prior revision — kept for reference. Detailed comparison table below in "Camp Matrix" section.)

### Syntactic Camp — fix the tokens going in

X07, NERD, Magpie, Laze.

### Verification Camp — fix the contract on what comes out

Vera, Aver, Raskell, Prove, Pact, MoonBit, Zero, **AILANG**.

### Orchestration Camp — fix the loop around the language

Pel, Marsha, Plumbing, Quasar, Boruna.

---

## Camp Matrix (snapshot 2026-05-20)

### Syntactic Camp

| Language | Mechanism | URL | AILANG parity |
|----------|-----------|-----|---------------|
| X07 | JSON AST + RFC 6902 patches | x07lang.org | ❌ Opposite bet |
| NERD | English keywords for operators | nerd-lang.org | ❌ Conventional operators |
| Magpie | SSA form as user syntax | magpie-lang.com | ❌ ANF-flavored, not SSA |
| Laze | Indentation-only, compiles to C | github.com/kerv/laze | ❌ |

**AILANG position**: deliberate non-member. Bets that *contract on output* matters more than *tokens on input*.

### Verification Camp

| Language | Mechanism | URL | AILANG parity |
|----------|-----------|-----|---------------|
| Vera | Z3 verification, De Bruijn slot refs | veralang.dev | ✅ Z3 + contracts. Names not slots. |
| Aver | Lean 4 proofs, verify blocks, decision blocks | averlang.dev | ⚠️ Z3 not Lean; no decision blocks |
| Raskell | Haskell + new tooling | raskell.io | ❌ Different host |
| Prove | AI-resistant (license-forbidden) | prove.botwork.se | ❌ Excluded |
| Pact | Intent annotations, MCP server | github.com/KikotVit/pact-lang | ⚠️ MCP ✅ ([agent-mcp guide](../../../docs/docs/guides/agent-mcp.md)), no intent annotations |
| MoonBit | Semantics-aware token sampler | moonbitlang.com | ❌ Post-hoc verify, not constrained gen |
| Zero | One canonical form, structured diagnostics | github.com/vercel-labs/zerolang | ⚠️ See [m-zero-language-learnings](../v0_21_0/m-zero-language-learnings.md) |

**AILANG position**: full member via `requires`/`ensures` + Z3. Closest peers: Vera, Aver.

### Orchestration Camp

| Language | Mechanism | URL | AILANG parity |
|----------|-----------|-----|---------------|
| Pel | Agent coordination as language primitive | arxiv.org/abs/2505.13453 | ⚠️ External coordinator, not language primitive |
| Marsha | Agent coordination | github.com/alantech/marsha | ⚠️ Similar |
| Plumbing | Typed streaming pipelines, static well-formedness | johncarlosbaez.wordpress.com | ❌ No streaming primitive |
| Quasar | Python subset + auto-parallel + uncertainty quant | arxiv.org/abs/2506.12202 | ❌ No transpile target |
| Boruna | Capability-gated effects, hash-chained audit | github.com/escapeboy/boruna | ⚠️ Caps ✅, no audit chain |

**AILANG position**: strong member via `std/ai` as a first-class effect — but this is *under-surfaced* in current docs and *not benchmarked*.

---

## Benchmark Coverage Audit (2026-05-20 baseline)

Current suite: **56 benchmarks** across smoke/core/stretch/vision tiers. Strong existing coverage:

| Tag | Count | Notes |
|-----|-------|-------|
| `adt_pattern_match` | ~10 | Heavy coverage |
| `contracts` | 5 + 3 mixed | Z3 verification path well-tested |
| `effects_io` | 5 | IO/FS effects covered |
| `recursion` | ~8 | Standard CS |
| `string_algo` | ~6 | |
| `state_machine` | 3 | |
| `type_safety` | 4 | |
| `records` | 3 | |

---

## Gap Analysis: Where Each Language Would Shine That We Don't Test

The matrix below maps each peer language's defining capability to a benchmark we should add. Each gap is a **testable hypothesis** about why that camp exists.

### Gaps driven by Syntactic camp claims

| Gap benchmark | Inspired by | Tests | Hypothesis |
|---------------|-------------|-------|------------|
| `ast_patch_roundtrip` | X07 | Generate a code transformation as a structural diff, not free text | X07's claim: structural editing reduces ambiguity vs text |
| `dense_operator_program` | NERD | Operator-heavy code (`<<`, `>>`, `&&`, `==`) | NERD's claim: tokenizer-ambiguous operators hurt LLM pass rate. **Direct refutation test for AILANG.** |
| `explicit_dataflow_ssa` | Magpie | Heavy let-chain with single-assignment | Magpie's claim: SSA-shaped code is easier to reason about |

### Gaps driven by Verification camp claims

| Gap benchmark | Inspired by | Tests | Hypothesis |
|---------------|-------------|-------|------------|
| `shadowing_heavy_contract` | Vera | Contracts in scope-shadowing-heavy code | Vera's claim: named refs break down under shadowing; AILANG's HM should not |
| `decision_block_capture` | Aver | Agent must emit implementation AND structured rationale | Aver's claim: rationale-in-code improves auditability |
| `intent_annotated_solver` | Pact | Same task with and without `@intent("...")` prompting | Pact's claim: intent annotations improve LLM pass rate. **Direct test.** |
| `canonical_convergence` | Zero | Run N=20 generations of same prompt; measure semantic-equivalent ratio | Zero's claim: one canonical form helps; AILANG's A1 determinism should already help here |

### Gaps driven by Orchestration camp claims

| Gap benchmark | Inspired by | Tests | Hypothesis |
|---------------|-------------|-------|------------|
| `multi_agent_handoff` | Pel / Marsha | Agent uses `std/ai` to delegate a subtask, composes the result | AILANG's `std/ai` makes this expressible; competitors need external wrapping |
| `typed_stream_pipeline` | Plumbing | Static well-formedness of a streaming transform | Plumbing's claim: typed pipelines catch wiring errors statically |
| `parallel_independent_subtasks` | Quasar | Code structure that exposes parallelism opportunities | Quasar's claim: explicit independence improves optimizer leverage |
| `audit_chain_replay` | Boruna | Execute → capture chain → replay; bit-identical output | A2 replayability under audit — direct AILANG strength |

### Gaps for AILANG's own untested differentiators

| Gap benchmark | Why it matters | Tests |
|---------------|----------------|-------|
| `ai_effect_summarize` | `std/ai` is our biggest unique capability, **currently unbenchmarked** | Function with `! {AI, IO}` calling `call()` |
| `ai_effect_json_schema` | Structured AI calls with schema | `callJson(prompt, schema)` end-to-end |
| `unauthorized_fs_refused` | A4 explicit authority | Code that *should fail* because FS capability not granted |
| `parallel_map_reduce` | A6 safe concurrency | Fork/Call/Done structured concurrency |

**Total gap benchmarks: ~14.** AILANG self-audit on these is the centerpiece talk material.

---

## Phase Plan

### Phase 1 — Public comparison page (Talk-critical, ~8h)

Live by Sun 2026-05-24 for the Mon talk.

- `docs/docs/guides/three-camps-comparison.md` — Camp Matrix + Gap Analysis sections from this doc, rewritten for public audience
- Sidebar entry under Guides
- Landing page mentions `std/ai` + coordinator in headline area

### Phase 2 — Gap benchmark authoring + AILANG self-audit (Talk-critical, ~25h)

The centerpiece. Add the 14 gap benchmarks; run AILANG on them; honest scoreboard.

- 14 × `benchmarks/<gap>.yml` written, with `expected_stdout`, prompt, tags
- Each benchmark runs cleanly under `ailang eval --lang ailang --benchmark <name>`
- Self-audit report: which gaps does AILANG pass, which fail, what does the failure tell us
- Output: `docs/docs/guides/three-camps-self-audit.md` with raw data

### Phase 3 — Peer language bring-up (Stretch for talk, ~20–30h)

Add peer languages where the comparison is **fair**, not just mechanically possible:

| Peer | Tier they belong on | Rationale |
|------|---------------------|-----------|
| **MoonBit** | smoke + new gap benchmarks | ML-family, ADTs, records — apples-to-apples |
| **Vera** | contract tier + `shadowing_heavy_contract` | Direct Z3 peer; closest comparator |
| **Aver** | contract tier + `decision_block_capture` | Lean variant; tests `decision_block_capture` natively |
| **Pact** | `intent_annotated_solver` | Intent annotations are its differentiator |

**Skip from sprint scope**:
- Zero — C-family systems language; smoke tests are unfair, no contract tier. Reference [zero-comparison.md](../../../docs/docs/guides/zero-comparison.md) instead.
- Syntactic camp (X07, NERD, Magpie, Laze) — talk story is the gap benchmarks (`ast_patch_roundtrip`, `dense_operator_program`), not running their toolchains
- Research-only (Pel, Plumbing, Quasar without confirmed impl) — defer to Phase 4

Per-peer cost: ~5–7h (langreg + prompt + runner + 5–8 benchmark opt-ins).

### Phase 4 — Audit memos + borrowable ideas (Post-talk, ~15h)

Short memos for each idea worth borrowing: decision blocks (Aver), intent annotations (Pact), constrained sampling (MoonBit research), audit chains (Boruna), typed streaming pipelines (Plumbing), Quasar parallel. Each declares build/eval/shelve. Output: `design_docs/planned/v0_22_0/m-three-camps-audit/`.

---

## Talk-Week Schedule

| Day | Date | Focus | Hours |
|-----|------|-------|-------|
| 1 | Wed 5/20 | Phase 1 comparison page draft + start gap benchmarks (4–5 of the 14) | ~8h |
| 2 | Thu 5/21 | Finish 14 gap benchmarks; start AILANG self-audit | ~8h |
| 3 | Fri 5/22 | AILANG self-audit complete; MoonBit bring-up | ~8h |
| 4 | Sat 5/23 | Vera + Aver bring-up (toolchain-gated) | ~8h |
| 5 | Sun 5/24 | Pact bring-up (if time); comparative chart; landing page polish; final QA | ~6h |
| — | Mon 5/25 | TALK | — |

**Floor (must ship for talk)**:
- Comparison page live
- 14 gap benchmarks merged
- AILANG self-audit data
- At least 1 peer language (MoonBit) running on the comparison subset

**Ceiling (nice-to-have)**:
- All 4 peer languages (MoonBit, Vera, Aver, Pact) running
- Public self-audit blog post

---

## Risk Mitigation

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Toolchain install fails for one+ peer | Medium-high | Phase 3 has scope-reduction rules; Phase 1+2 ship regardless |
| Gap benchmark proves harder to design than expected | Low | Each is a small file (~50 LOC YAML); precedent in existing 56 benchmarks |
| AILANG fails several gap benchmarks badly | Medium | **That's the talk insight**, not a failure. Honest scoreboard >> defensive scoreboard |
| `canonical_convergence` requires N=20 runs (cost) | Low | Use cheapest model for that one; mention cost in the writeup |
| `ai_effect_*` benchmarks need real LLM calls | Medium | Use stub provider for CI; real provider for talk-week run only |

---

## Open Questions

1. Should `decision_block_capture` and `audit_chain_replay` produce new AILANG features (decision blocks; chain replay primitive), or just test what's possible with current primitives? **Default: test current primitives, file follow-up issues if gaps surface.**
2. For `intent_annotated_solver`, do we add `@intent` as a language feature or as a prompt convention? **Default: prompt convention this sprint; language feature is a Phase 4 decision.**
3. Public talk material: blog post vs slide deck vs both? **Default: comparison page + self-audit page are the artifacts; slides are the user's own work.**

---

## Success Criteria

**Must-have for talk** (Mon 5/25):
- [ ] `docs/docs/guides/three-camps-comparison.md` published
- [ ] 14 gap benchmarks merged and runnable
- [ ] AILANG self-audit data captured in `docs/docs/guides/three-camps-self-audit.md`
- [ ] At least 1 peer language wired up (MoonBit floor)

**Should-have**:
- [ ] All 4 peer languages wired up
- [ ] Comparative chart embedded in self-audit page
- [ ] Landing page foregrounds orchestration

**Phase 4 (post-talk)**:
- [ ] Audit memos for borrow candidates with build/eval/shelve decisions
- [ ] Follow-up sprint designs for any "build" decisions
