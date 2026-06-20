# M-AILANG-NATIVE-HARNESS: Exact Semantics over Approximation

**Status**: North-star / design vision (reframes m-ailang-semantic-context.md)
**Target**: v0.26.x → v1.x (staged)
**Mission**: the strategic answer to "make motoko the best AILANG coding harness"
**Origin**: 2026-06-20 design conversation — "compression is intelligence; what is a coding
harness when you control the language?"

## Thesis

A coding harness is a function `(goal, codebase) → (edits, verified)`. **General-purpose**
harnesses (Claude Code, Cursor) must run every step over **text**, because they cannot assume a
language. AILANG controls the language, so motoko can run every step over **semantics** — querying
the compiler's exact knowledge instead of approximating it.

**The axis (refined, 2026-06-20 second-agent synthesis):** not grep-vs-embeddings but
**text retrieval → semantic retrieval → proof-carrying retrieval.** The third tier is the genuinely
unique one: AILANG has contracts (Z3), effect/authority ceilings, and deterministic traces, so
retrieval can carry **obligations** (what must hold), not just types. Most languages cannot aspire to
this tier at all. **Two-stage, not either/or:** embeddings remain useful as the *coarse pointer* for
fuzzy intent ("where's the PDF chunking?"); once candidate regions are found, switch to **exact
structure** (which functions produce `Block[]`, which effects, which contracts protect it). Fuzzy
locate → exact verify.

The "Claude Code grep beat Cursor semantic-search" result is real but its conclusion is *specific to
language-agnostic harnesses*. The principle underneath it is **give the model the best ground truth
with the least noise** — for which grep (exact text) beats embeddings (fuzzy similarity). But grep
and embeddings are **both approximations of what a compiler already knows exactly.** When you own the
language, the right tool is neither: it is **exact semantic queries**, which are *simpler than
embeddings* (no vector store, no threshold, no model) **and** exact — the rare "more powerful AND
simpler", which is precisely when an assumption should be revisited.

## Generative principle: machine-convenience over human-convenience

The deepest framing of the whole mission, and the engine for *generating* the next experiments:
**our tooling optimizes for human convenience on axes the model doesn't care about, and ignores
machine-convenience on axes where the model is strong.** Text editors, formatting, syntax sugar,
file/navigation metaphors — all are human affordances. The question that drives the harness forward is:
*which of the model's latent strengths does human-shaped tooling under-serve?*

**The constraint that keeps this honest (or it fails).** The model's latent knowledge is
**human-code-shaped** — trained on billions of lines of *human* text code, not alien machine formats.
So the failure mode is inventing a "machine-native" representation the model has never seen
(out-of-distribution → *worse*, the grep lesson again). Wins are NOT alien formats. They are: **exploit
the structured knowledge the model already has, while stripping the human-convenience affordances that
are pure noise to it.** AST-as-text-spans threads this — text the model knows, scoped semantically.

**The inversion (sharpest insight):** the features that are *inconvenient for humans* are often
*convenient for machines*, and we've optimized the wrong direction. Humans hate explicit
effects/contracts/type-annotations (boilerplate) — but the model already knows what effects code should
have, annotating is trivial for it, and they make its downstream job easier (proof-carrying). Humans
need formatting/whitespace/sugar to read — the model tokenizes past it as noise; canonical forms are
machine-convenient. Humans need file/navigation metaphors (limited working memory) — a chunking
artifact. **AILANG's whole bet — explicit, deterministic, structured, no sugar — is the wager that
machine-convenience beats human-convenience for AI-written code.** Text→AST is one slice of it.

**Under-exploited latent capabilities (candidate affordances → future experiments):**
1. **Type-directed synthesis / typed holes** — "fill the hole of type `Result[Row, ParseError]`."
   Models excel at type→term; humans find hole-driven dev awkward. Offer typed holes; checker verifies.
2. **Distributional generation + verification** — the model *is* a distribution (calibrated
   alternatives/logprobs); we force single-shot text. Let it emit N candidates; use the
   **type checker / contracts as the selector** (model proposes, AILANG disposes).
3. **Self-verification before emit** — the model pre-checks an edit against the types it already knows,
   as a cheap filter before the expensive run.

**The method — harness as the discovery instrument.** Latent knowledge is invisible; you cannot know
a priori which capability is real. The only way to find it is to **offer the affordance and measure
whether the model uses it better than the human-shaped default.** So every machine-convenient affordance
(AST edit, typed hole, candidate-generation) is a *probe*, and the A/B *is* the discovery mechanism.
motoko is therefore not just the best-AILANG-harness candidate — it is the **measurement apparatus for
discovering which machine-convenient affordances unlock latent model knowledge.** Semantic-edit is
probe #1; typed-holes and candidate-generation are the queued probes.

## Compression is intelligence → the iface is lossless compression

The model's context window is a scarce channel; the harness's job is the **minimal sufficient
representation** at each step. For arbitrary code that representation is *unknowable* (hence
truncation / RAG / embeddings — all lossy approximations). For AILANG it is **computable**: the
**typed interface** is a lossless semantic compression of a module (a signature *is* what a function
means, without its body, computed for free). The harness should think in **ifaces and types**, not
files (raw) or embeddings (lossy). "Entropy collapse of semantic meaning" is what a type checker
already does — deterministically, not learned.

## The functions, rethought

| Function | General harness (text / approx) | AILANG-native (exact) | AILANG primitive (exists?) |
|---|---|---|---|
| **Search / discovery** | grep, embeddings | call graph + **type-directed search** ("functions of type `(User)->Email`") | LSP (go-to-def, refs) ✅; type-search ⬜ |
| **Context assembly** | read whole files, RAG | **iface projection** — signatures + effect rows, not bodies; type-directed relevance | `internal/iface` ✅ |
| **Edit application** | text patches (line numbers, full rewrites) | **AST / semantic edits** — "change F's return type", "add FS to G" | AST ✅; semantic-edit op ⬜ |
| **Verify / feedback** | run, parse stderr | typed + effect check, exact; actionable distilled errors | `ailang check`, R1/R1b ✅ |
| **Debug** | print, re-run | **deterministic trace** → the exact evaluation step that diverged | `AILANG_TRACE` ✅; trace-diff ⬜ |
| **Dedup / memory** | embeddings / SimHash | **structural identity** (alpha-equivalence) — exact, no vectors | AST compare ⬜; Brain (fuzzy) ✅ |

## Proof-carrying additions (second-agent synthesis — sharper than the table above)

These three escalate the table's "exact" column from *typed* to *proof-relevant*, using AILANG
structure no general harness has:

- **Obligation discovery (beats "run all tests").** On an edit, the harness reports the exact re-proof
  set: *"You changed `F` → 3 downstream typechecks, 1 effect ceiling, 2 contracts, 4 golden traces, 1
  API schema hash."* This is the verify-side of editing — the minimal set of obligations a change
  disturbed, derived from the type/effect/contract/trace graphs. Stronger and cheaper than a blanket
  test rerun.
- **Trace-equivalence as the test primitive.** Cache execution **witnesses** (input hash, effect
  *envelope*, trace hash, output hash, contract result); evaluate a patch by trace equivalence rather
  than textual rerun. *Precise caveat:* AILANG is deterministic **given fixed effect results** —
  AI/IO/Net outputs are the non-deterministic boundary, so equivalence compares the pure skeleton +
  effect *signature/envelope*, not the raw effect outputs. Not naive `trace_hash ==`.
- **AILANG-native indexes, grep as fallback.** symbol / type / effect / contract / trace / capability
  / semantic-hash / normal-form. Grep stays (Claude Code lesson) but as the *fallback*, not the core
  memory. **Earn-its-place gate:** each index must beat grep+model on precision AND simplicity before
  the next is built — type/effect indexes are exact+cheap (build); normal-form/semantic-hash are
  exotic+uncertain (defer until a measured win). Don't build the cathedral.

**Product surface (the primitive):** `harness.query { goal, scope } → SemanticSlice { symbols, types,
effects, contracts, traces, risks, candidate_edit_points }`, returning primary edit points + relevant
invariants + required checks. *Adoption caveat:* a rich query API only helps if the model wields it —
local models may do worse with it than with 3 simple tools. Offer alongside grep, measure adoption,
don't force.

**Scope honesty (the two timelines):** obligation/contract/trace queries shine on **contract-rich,
long-lived AILANG codebases** — that is the *AILANG-as-AI-coding-platform* north-star. The current
motoko-on-benchmarks residual (small, ~contract-free synthesis tasks where motoko is already at pi
parity) is a *different, nearer* problem. Build the cheap exact-semantic wins (iface context, semantic
edits) for the near-term; the proof-carrying stack is the strategic moat, staged and measured.

## Tools: same surface, divergent substrate

Design rule (from the grep lesson + adoption risk): the agent-loop tool **surface** stays identical
across languages so the model's behavior transfers; the **result** becomes semantic for `.ail`. The
harness routes by file type — `.ail` → semantic backend, else → text backend. **Enrich existing
tools' results first; add new tools (type-search, obligation-query) only when they measurably beat
the enriched ones.** R1/R1b already proved the template: the `run`/`check` tool's *results* are now
AILANG-semantic (distilled typed errors) with no surface change.

| Tool (same name) | General language (text) | AILANG (semantic backend) |
|---|---|---|
| read | raw text | + iface view (sigs+effects); type-at-point |
| search | grep | grep fallback + exact typed queries (by-type, find-refs, effect/contract) |
| edit | text patch / full rewrite | + semantic AST edit — no rewrite |
| write | write bytes | + immediate typecheck / iface-delta feedback |
| run | stdout/stderr | `ailang run` → distilled typed errors (R1/R1b ✅) + deterministic trace |
| test | pass/fail rerun | trace-equivalence witnesses; obligation discovery |

Where this lives — **it's all AILANG (no fork/FFI boundary).** motoko_agent is itself an AILANG
package (`ailang.toml`; `compaction.ail` is `module src/core/compaction` that `import std/ai`,
`std/string`, … directly). So the semantic-harness capabilities are **AILANG stdlib modules** (a new
`std/iface`-summary / `std/semantic` alongside the existing `std/ai.runTools`, `std/sem`), and
motoko's `.ail` packages **import them like any AILANG program** — a stdlib capability + a package
dependency, not a cross-language bridge. "Language-level capability any agent inherits" is therefore
literal: it's stdlib. And motoko being AILANG means this work **dogfoods the language** — the flagship
AILANG application validating AILANG for real agentic code. (Compiler internals — `internal/iface`,
`internal/ast`, `AILANG_TRACE` — back the stdlib surface that `.ail` calls.)

## Semantic compaction (the collapse stage — distinct from assembly)

Context handling has two stages and the vision improves BOTH; the routes above cover **assembly**
(iface projection = what to *include*). This route covers **compaction** (what to *collapse when over
budget*), which currently lives in motoko's `compaction.ail` as lossy text-elision (`elide_content`
truncates a tool_result to 80 chars). Replace that with **collapse-to-meaning**: a read → its iface,
a run → trace-hash + output + effect envelope, an edit → its obligation set. "Compression is
intelligence" applied to the compaction package — controlled collapse of program meaning, not prose.

Caveats: (1) **moot for qwen today** — motoko's compaction never fires (`context_limit_for(qwen)=0`,
fire-rate 0, see analysis-log 2026-06-20), so this is gated on first re-enabling compaction (the
`context_limit_for`-for-ollama fix, measurement-gated — and re-confirm it helps before tuning).
(2) The collapse summaries come from the compiler, so the **capability is AILANG-side** (`std/ai` /
`internal/`), consumed by `compaction.ail` (fork). Same language-level-capability pattern.

## How we test these (the mission's measurement framework fits)

The eval harness *is* the test bed; the A/B mechanism already exists as the **motoko profile system**
(the 2026-06-20 DP7 A/B — `ollama` vs `ollama_dp7` — is exactly this shape). A new tool = a new
profile (e.g. `ollama_semantic_edit`), A/B'd vs base on the same benchmarks.

- **Metric = efficiency, not pass-rate** (motoko is already at pi parity): turns-to-success,
  tokens-per-run, and the **rewrite-rate / re-read counts** (mined from transcripts). All captured.
- **Adoption check:** confirm the model actually *called* the new tool (transcript) — a good tool the
  model ignores shows no A/B gain.

**Testability per tool:**
- **Semantic edit — testable NOW** on the current single-file suite (the 59-rewrite thrash is on
  single files). Highest-leverage *and* most measurable on what we have → first experiment:
  `ollama_semantic_edit` profile, A/B by rewrite-rate + turns + tokens + adoption.
- **Navigation tools (read-relevant / grep-structured / impact)** — need a **multi-file /
  codebase-navigation benchmark tier**; our ~30-line single-file tasks are too small to show
  context-selection value (the whole file *is* the context). Adding that tier is itself a mission task.
- **check/verify gate** — already under test (DP7 A/B).

## Tool surface (prioritized — enrich existing, add new sparingly)

Enrich the 4 the model already calls; add `check`+`impact` only as they earn it; defer the exotic.

| Tool | Enrichment | Priority |
|---|---|---|
| read | semantic slice (iface / fn-at-symbol / type-at-point) | needs multi-file bench |
| edit | AST / semantic edit | **first experiment (testable now)** |
| grep | structured query (by-type/effect, find-refs) | needs multi-file bench |
| run | distilled typed errors + trace | ✅ done (R1/R1b) |
| check | typed+effect+contract gate (def-of-done) | under test (DP7) |
| impact | obligation discovery (what an edit re-proves) | needs multi-file bench |
| *(defer)* | effects/authority query, trace-debug, outline, canonicalize | later |

## Highest-leverage unlock: semantic edits

Measured thrash (2026-06-20 h2h): **59 full-rewrites** — qwen rewriting whole files because text
patching (line numbers, exact-match edits) is error-prone for a local model. If the model emitted
**semantic edits against the AST** ("set the return type of `parseRow` to `Result[Row, string]`";
"replace the body of `encode` with …"), the harness applies them by transformation — no line-number
fumbling, no full rewrites. This is a tool only a language-controlling harness can offer, and it
attacks the dominant measured inefficiency directly. *Risk:* the model must learn to emit semantic
edits (prompt/tool-schema work); start with a hybrid (semantic edit tool offered alongside text edit)
and measure adoption + rewrite-rate.

## Honest update on embeddings (demote R7)

This reasoning cuts *against* embeddings for in-codebase work. Embeddings are the **Cursor
approximation**; for discovery, context, and dedup over AILANG code, exact semantic queries dominate
them on **both** precision and simplicity (the grep lesson, applied honestly, points *toward* exact
semantics, not toward adding a vector store). **Demote R7** to the one genuinely-fuzzy niche where no
exact structure exists to query: **cross-session memory** ("have I solved something *like* this
before"), where the Brain/SimHash earns its place. Everything in-codebase → exact semantic routes.

## Reordered route priority (supersedes m-ailang-semantic-context sequencing)

1. **Verify** — R1 + R1b ✅ (actionable typed diagnostics). Extend to more error classes.
2. **Context** — **iface projection** as the context-assembly primitive (R6, exact flavor): surface
   signatures + effect rows, not bodies. The lossless-compression core.
3. **Edit** — **semantic/AST edit tool** (the rewrite-thrash killer). Highest leverage.
4. **Search** — type-directed / LSP-backed discovery, replacing grep+embeddings for the agent.
5. **Debug** — deterministic trace-diff (the diverging step).
6. **Memory** — embeddings ONLY for cross-session fuzzy recall (demoted R7); structural identity for
   in-session dedup.

Near-term pi-match (tool-result truncation, echo-writes) still lands first as the cheap floor — but
the *strategic* differentiator is this exact-semantic stack, which a general harness structurally
cannot copy.

## Measurement frontier: project-scale eval (the falsification test)

Single-file benchmarks show motoko ≈ pi *precisely because* the AILANG-native advantage is irrelevant
at 30 lines (the whole file fits one read). The semantic tools only pay off on **multi-file projects**
— so a project-scale eval is both the test bed for them AND the first arena where motoko should pull
*ahead* of pi. **This is the falsification test for the whole north-star:** if motoko+semantic-tools
does not beat pi+text on real projects, the thesis is wrong (or the model can't exploit the tools) —
the most valuable negative result we could get, and a reason to build the measurement before the stack.

**Grading (what makes projects gradable — AILANG-specific):** `ailang check --package` (build/typecheck,
exact) + a provided **acceptance test suite** (behavior) + **contracts**/Z3 (proof, where the spec has
invariants) + **iface comparison** (required interface exposed). Plus existing efficiency metrics
(tokens/turns) and new ones (build-pass, test-pass, spec-conformance, adoption of semantic tools).

**Sources, by tractability:** (1) **modify-existing** on the demos repo / a curated multi-module AILANG
project — "add feature / fix bug", graded by existing-tests-still-pass + new-acceptance-test; exercises
read-relevant / grep-structured / impact; **start here.** (2) **docparse** (real AILANG app, cloud
pipeline — access/setup needed) for "extend a real system". (3) **build-from-scratch** — the "ultimate
A/B": same spec to pi and motoko, compare complete projects; purest but needs per-spec acceptance suites.

**Sequencing:** semantic-edit on the current single-file suite (now, cheap) → **build the project-eval
harness** (modify-existing on demos; a real eval-harness extension, its own mission task) → the
**pi-vs-motoko complete-project A/B** (the north-star's falsification test).

## Capstone: type-safe self-extension (the harness modifies its own source)

**Grounding (confirmed):** motoko is composed from ~13 **published, versioned, hash-verified** AILANG
extension packages (`ailang.toml [dependencies]`: `sunholo/motoko_ext_abi@2.2.0`,
`motoko_ext_compaction_ai`, `motoko_ext_microrag`, …), each conforming to the **typed extension ABI**
`motoko_ext_abi` (`ExtensionHooks`, `PreStepDecision`, `FinalizeDecision`, `ToolPolicyDecision`) and
declaring its effects, loaded via `import pkg/…/register (register_with_config)`. So motoko's behavior
is typed, versioned, hot-loadable plugins.

That makes **type-safe self-modification** possible: the harness writes a *new extension package* and
adopts it through five gates that already exist —
| Layer | Mechanism (exists) | Prevents |
|---|---|---|
| Structural | type-checks against `ExtensionHooks` (`ailang check`) | malformed / non-conforming edits |
| Authority | effect rows + enforced effect ceiling | privilege escalation (authority widening) |
| Provenance | hash-verified versioned package + `ailang.toml` | unauditable / irreversible changes (rollback = pin prior) |
| Behavioral | **eval-gate**: A/B the new extension on the suite | type-correct-but-*worse* edits |
| Verification | contracts prove invariants | silent invariant breakage |

Loop: **write typed extension → `ailang check` (conformance + effect ceiling) → eval-A/B vs baseline
→ adopt only on a measured win → publish as a versioned package → hot-load.** The type system bounds
the *blast radius*; eval-gating bounds the *quality*. Self-modifying code → self-improving, auditable,
reversible package. Recursion closes: motoko writes AILANG, *is* AILANG, so it can safely write motoko,
gated by its own eval harness. A self-proposed extension is just another profile to A/B — the mission's
measurement *is* the adoption gate.

**Discipline:** (1) **type-safe ≠ better** — the eval-gate is mandatory, not optional, or it drifts.
(2) Capstone, not next lever (v1.x+); don't let it distract from cheap near-term wins. (3) The effect
ceiling (what authority a self-written extension may request) is safety-critical config — guard it.

## Language/compiler changes this unlocks (in-scope this mission, via design docs)

Building the semantic-edit core (`internal/astedit`, span-anchored decl-replace) surfaced concrete
language changes worth making. Discipline: ship the *experiment-grade* version now, run the A/B, and
**earn each language investment with the result** (don't build the cathedral before the lever proves out).

1. **Edit-grade parser spans (modest, broadly useful — first to earn).** Today `FuncDecl.Pos`/`Span`
   carry line/col but **`Offset` is 0** (lexer doesn't track byte offsets) and `Span.Start` begins at
   `func`, **excluding a leading `export`/`pure` modifier** (and preceding-line annotations). `astedit`
   currently works around this (line:col→offset + splice from line-start) — fine for simple top-level
   decls, fragile on annotations/multi-modifier/unicode. **Fix:** lexer populates `Pos.Offset`; parser
   records the **full-declaration span** (modifiers + annotations → closing brace). Benefits not just
   semantic edits but **LSP, SID, formatting, refactoring** — a small compiler change with wide payoff.
2. **Faithful AST↔source round-trip formatter / `ailang fmt` (bigger — earn after #1).** AILANG has
   AST `String()` methods but no round-trip-faithful printer. A real formatter unlocks **richer
   sub-declaration edits** (edit an expression/signature, not just replace a whole decl) and is a
   standard language tool AILANG lacks (gofmt/rustfmt analog) with independent value. Hard part:
   preserving comments/whitespace. Gate the build on the semantic-edit A/B showing decl-level edits help.

These are mission backlog items (design-doc'd), not blockers: the experiment-grade `astedit` runs the
A/B today; #1 makes it production-grade; #2 widens the edit vocabulary.

## Open questions / risks

- **Model adoption:** semantic edits + typed search are new tool shapes the model must use well.
  Hybrid + measure; don't force.
- **Simplicity bar:** every semantic tool must beat grep on *both* precision AND simplicity, or it's
  the embeddings mistake again. Exact + deterministic + no extra infra is the bar.
- **Build cost:** type-directed search, semantic-edit ops, and trace-diff are real builds in
  `internal/` (this repo) consumed by motoko (fork). Stage them; each must show a measured win
  (turns-to-success / tokens / rewrite-rate) before the next.
- **Validation discipline:** same as the compaction lesson — confirm the lever from data before
  building (e.g., confirm the rewrite-thrash is text-patching difficulty, not model capability,
  before building semantic edits).

## References
- `m-ailang-semantic-context.md` (the context routes this reframes)
- `motoko-harness-analysis-log.md` (2026-06-20: residual, thrash = 59 rewrites, compaction refuted)
- AILANG primitives: `ailang lsp`, `internal/iface`, `internal/ast`, `AILANG_TRACE`, `std/sem`/Brain
