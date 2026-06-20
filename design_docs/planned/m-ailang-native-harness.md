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

The "Claude Code grep beat Cursor semantic-search" result is real but its conclusion is *specific to
language-agnostic harnesses*. The principle underneath it is **give the model the best ground truth
with the least noise** — for which grep (exact text) beats embeddings (fuzzy similarity). But grep
and embeddings are **both approximations of what a compiler already knows exactly.** When you own the
language, the right tool is neither: it is **exact semantic queries**, which are *simpler than
embeddings* (no vector store, no threshold, no model) **and** exact — the rare "more powerful AND
simpler", which is precisely when an assumption should be revisited.

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
