# Fixed in v0.54.9: crew_core cold compile + coretypeinfo cache size; please retry registry publish (stapledons_godot)

- **Date**: 2026-10-13
- **Class**: already-covered (fix notification + verification request; no new defect or feature)
- **Recommend**: duplicate-of `design_docs/planned/v0_55_0/m-crew-core-cold-compile-followup.md`
- **Searched**: `coretypeinfo`, `cold compile`, `crew_core`, `registry publish`, `retry publish`, `#1811`, `#1787`, `#1788` across `design_docs/` and `changelogs/`; hits listed in Why.
- **Estimate**: n/a (no code change recommended)

The message is a fix notification from ailang-core (v0.54.9) answering three earlier reports, and every thread already has a ruling doc:

1. **Residual cold-compile time (#1787, substitution skips ground alias records)** — designed in `design_docs/planned/v0_55_0/m-crew-core-cold-compile-followup.md` Phase 2; that doc's CPU profile (V4) found 61.7% of cold time in `ApplySubstitution`, 30.5% behind `applyApplicationSubstitution`, i.e. exactly the path #1787 attacks. The reporter's v0.54.9 numbers (2.8 s / 121 MB vs the doc's 7.0 s / 124 MB on v0.54.8) suggest the doc's ≤ 2.5 s Success Metric is nearly met upstream — the v0.55.0 sprint should re-verify against v0.54.9 and re-scope (likely shrinking Phase 2 to the guard + retirement work) instead of implementing blind.
2. **Registry publish retry of crew_core/crew_lab@0.4.1** — operational, no design doc warranted. The v0.55.0 doc already verifies the validator budgets fit (V10: 384 MiB / 200 s cap in `cmd/registry-validator/bounded.go`; V11: crew_core `check --package .` 29.1 s / 134 MB on v0.54.8, and v0.54.9's 6 s / <200 MB only adds headroom). Action for the game repo: retry publish on ≥ v0.54.9 and, per the message, send the full response body if it still fails.
3. **CACHE_WRITE_FAILED / ARTIFACT_TOO_LARGE on ism and protocol_test** — covered by `ailang-core-triage/coretypeinfo-gob-inflation-breaks-session-cache.md` (the class fixed by v0.54.7 #1788's DAG encoding; motoko session.ail 188 MB → 275 KB). Action: confirm the warnings are gone on those two modules; the core team has not re-run them itself.
4. **Workaround retirement** — the v0.55.0 doc's Phase 3 item 2 retires the memory-limited CI warm-up (two cold sims ~250 MB total against the 16 GB runner); the register for the game's other workarounds is `design_docs/planned/v0_54_7/stapledon-v0-54-6-workarounds.md`. Note the message's caveat: keep the ism chunked helpers — the >256 register limit (#1737) is a separate, still-open issue.

No new triage action beyond pointing the v0.55.0 sprint at the v0.54.9 re-verification; nothing here changes semantics, surfaces, or gates.