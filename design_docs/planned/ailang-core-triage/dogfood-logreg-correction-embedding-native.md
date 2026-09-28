# Correction to dogfood logreg report — std/embedding already has dot/scale/add_vectors; the narrow ask (native ops) is already shipped

- **Date**: 2026-09-27
- **Class**: already-covered
- **Recommend**: duplicate-of `design_docs/planned/m-numerics-vec-array-ingest.md`
- **Searched**: `embedding`, `logistic`, `dot product`, `cosine_encoded`, `8034dce8` over `design_docs/`; `design_docs/planned/v0_45_0/`; `design_docs/planned/ailang-core-triage/` (no file for the original report)
- **Source**: email-parse correction to `inbox_1790498369833_8034dce8` — message `inbox_1790530530325_58bd9fdb` (2026-09-27)

The correction retracts item 3 of the dogfood report (it wrongly said dot/axpy/elementwise ops were missing — `std/embedding` has exported them since v0.5.11) and narrows the ask to "make std/embedding's vector ops Go builtins, the way `_embedding_encode`/`_embedding_decode` already are", citing that `cosine_encoded` decodes to lists and calls the AILANG `cosine`, so the packed form buys nothing.

**The narrow ask is already implemented.** M-NUMERICS-QUICK (same change as `m-numerics-vec-array-ingest.md`, "What already shipped" section) landed exactly this in the working tree: `std/embedding.dot` is now `_vec_dot`, `scale` → `_vec_scale`, `add_vectors` → `_vec_add`, plus a new strict `axpy` → `_vec_axpy`, all native Go builtins in `internal/builtins/vector.go`; `std/list.range` also exists via `_list_range` (item 1 of the original report). Verified in-repo 2026-09-27: `std/embedding.ail:32,108,115,120` and `internal/builtins/vector.go:184-186`. The doc's Verification Log (V8, V9) measured 1,000 × 768-dim `dot` at 4.74 s → 0.04 s and an SGD step at ~0.1 ms.

The correction author checked v0.44.1, where `dot` was still the AILANG `dot_helper` recursion; the quick fixes landed after that, targeting v0.45.0 — consistent with "Mark tells me 0.45 addresses these asks". The doc also already covers the packed-bytes idea better than the correction asks for: it notes the `_embedding_encode`/`_embedding_decode` codecs exist but are not exported (V4) and Phase 0 exports them (`encodeF32LE`/`decodeF32LE`), and Phase 3 adds native binary ingest — so a native-over-packed-bytes dot is unnecessary once decode+native dot is ~0.1 ms. Remaining original items (O(n) `array.set`, JSON-only boundary) are Phases 0–3 of the same doc.

**Action for the sender**: none needed on the report; the corrected ask shipped. Worth a reply pointing at the doc's "What already shipped" table, since their 0.44.1 check predates it.

**Note**: the feedback-gate audit (`inbox_1790530532176_4578e549`) shows this correction was DRY-RUN / `not_authorized_for_dispatch`, so it was handled by manual triage here instead of the gate filing it.