# Feedback-gate shadow — offline batch over `public-feedback` (45 submissions, transport `direct`)

The gate is NOT enabled in prod (no `feedback_gate` block in config.cloud.yaml, #900), so there is no Haiku arm to compare against and no human disposition label. This table shows what the gate **would do** on System One answers, and how often the model's category agrees with the **submitter's declared category** — the gate's own mismatch check. Nothing acted.

## Would-do distribution

| would_action | would_reason | n |
|---|---|---|
| dispatch | passed | 44 |
| reject | classifier_prompt_injection | 1 |

**Category agrees with the submitter's declared category:** 44/45. **Injection ≥ 0.5:** 1. **Model errors:** 0. **Mean model latency:** 660 ms. **List-price cost for the batch:** $0.0028.

## Per submission

| id | declared | Jev category (conf) | genuine | injection | value | would |
|---|---|---|---|---|---|---|
| fb_851c18a9a07b18b6 | bug | ✗ limitation (0.46) | 0.42 | 0.53 | low (1.37) | reject / prompt_injection |
| fb_7b59f0cb24f80a48 | limitation | ✓ limitation (0.98) | 0.57 | 0.04 | high (2.91) | dispatch / passed |
| fb_30e82f6bdc5fc8c3 | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_3812fa33fd18fea6 | bug | ✓ bug (1.00) | 0.98 | 0.02 | high (3.00) | dispatch / passed |
| fb_b023726953f2ee5a | bug | ✓ bug (0.81) | 0.96 | 0.07 | high (2.99) | dispatch / passed |
| fb_0003e02912bc5fa8 | docs | ✓ docs (0.72) | 0.97 | 0.03 | high (2.99) | dispatch / passed |
| fb_5c0baeb1ee5ae1eb | bug | ✓ bug (0.55) | 0.96 | 0.03 | high (3.00) | dispatch / passed |
| fb_913ee851c83c0d8c | limitation | ✓ limitation (0.98) | 0.97 | 0.02 | high (3.00) | dispatch / passed |
| fb_3f91769664f77b3d | bug | ✓ bug (1.00) | 0.97 | 0.03 | high (2.99) | dispatch / passed |
| fb_80db67106b51714b | bug | ✓ bug (1.00) | 0.97 | 0.03 | high (2.99) | dispatch / passed |
| fb_8d3c8f68b74c3573 | docs | ✓ docs (0.98) | 0.97 | 0.06 | high (2.98) | dispatch / passed |
| fb_0a80899564c0828f | limitation | ✓ limitation (0.86) | 0.97 | 0.03 | high (2.99) | dispatch / passed |
| fb_a71eab12139bee26 | feature | ✓ feature (0.95) | 0.96 | 0.04 | high (2.96) | dispatch / passed |
| fb_dfb699d91224be9c | feature | ✓ feature (1.00) | 0.96 | 0.02 | high (2.99) | dispatch / passed |
| fb_0a8ab94817a74c3e | bug | ✓ bug (1.00) | 0.97 | 0.01 | high (2.99) | dispatch / passed |
| fb_334b4d8dea088ada | bug | ✓ bug (0.87) | 0.96 | 0.02 | high (2.99) | dispatch / passed |
| fb_cc1cb8fb23e1ead1 | limitation | ✓ limitation (0.99) | 0.96 | 0.02 | high (2.99) | dispatch / passed |
| fb_2dbfd79dbe2d1d3c | bug | ✓ bug (1.00) | 0.96 | 0.02 | high (3.00) | dispatch / passed |
| fb_c1cf1a339764a683 | limitation | ✓ limitation (0.83) | 0.96 | 0.03 | high (2.96) | dispatch / passed |
| fb_f7ecc535fde19c8e | feature | ✓ feature (0.99) | 0.96 | 0.02 | high (2.98) | dispatch / passed |
| fb_1cce034a84df5cec | bug | ✓ bug (0.53) | 0.56 | 0.48 | low (0.90) | dispatch / passed |
| fb_15bb9f87bcb7859b | bug | ✓ bug (0.99) | 0.56 | 0.04 | high (2.99) | dispatch / passed |
| fb_59eff5532a7df542 | bug | ✓ bug (1.00) | 0.98 | 0.02 | high (2.97) | dispatch / passed |
| fb_8c5ff9be592ce633 | limitation | ✓ limitation (0.91) | 0.96 | 0.05 | high (2.99) | dispatch / passed |
| fb_aa130315430290be | bug | ✓ bug (1.00) | 0.98 | 0.02 | high (2.99) | dispatch / passed |
| fb_0f70d66af0fddb2c | limitation | ✓ limitation (0.98) | 0.96 | 0.03 | high (2.99) | dispatch / passed |
| fb_ebebcacb3f29f3a5 | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_2ad074d754cd2c25 | bug | ✓ bug (1.00) | 0.96 | 0.02 | high (2.96) | dispatch / passed |
| fb_e44ba922db1c42be | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_b39697480a4e8bbc | limitation | ✓ limitation (0.99) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_6c81854baf59b316 | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_d230853828108783 | bug | ✓ bug (1.00) | 0.96 | 0.02 | high (2.99) | dispatch / passed |
| fb_74f53de3ae65854c | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_9458c0556edad08f | bug | ✓ bug (1.00) | 0.97 | 0.02 | high (2.99) | dispatch / passed |
| fb_28c91526c3595794 | feature | ✓ feature (1.00) | 0.94 | 0.04 | high (2.98) | dispatch / passed |
| fb_c3427ec3365aae6c | docs | ✓ docs (1.00) | 0.97 | 0.03 | high (2.99) | dispatch / passed |
| fb_942b7f3dff3d8e52 | bug | ✓ bug (0.97) | 0.96 | 0.03 | high (2.99) | dispatch / passed |
| fb_343bd6ad28827a2e | feature | ✓ feature (1.00) | 0.94 | 0.04 | high (2.99) | dispatch / passed |
| fb_cef305f96ccc24ae | bug | ✓ bug (1.00) | 0.94 | 0.04 | high (2.99) | dispatch / passed |
| fb_d6e51f9c44f19b95 | docs | ✓ docs (0.92) | 0.71 | 0.16 | low (0.91) | dispatch / passed |
| inbox_1778106626965_3eed0571 | bug | ✓ bug (1.00) | 0.96 | 0.02 | high (2.98) | dispatch / passed |
| inbox_1777886035535_f3be3aff | docs | ✓ docs (0.64) | 0.54 | 0.25 | low (1.30) | dispatch / passed |
| inbox_1777368293924_824467e2 | docs | ✓ docs (0.92) | 0.79 | 0.03 | medium (1.59) | dispatch / passed |
| inbox_1777368277383_581d7f08 | docs | ✓ docs (0.92) | 0.78 | 0.03 | medium (1.67) | dispatch / passed |
| inbox_1777310260628_b3066df4 | docs | ✓ docs (0.45) | 0.71 | 0.03 | low (1.07) | dispatch / passed |

Titles for the disagreements (declared vs Jev):

- `fb_851c18a9a07b18b6` declared **bug**, Jev **limitation** (0.46): Need API key to check quota

## Reading it (hand-written, 2026-09-19)

- **44/45 dispatch, 1 reject, 0 file.** On the real public backlog the System One arm would have let through everything a human would recognise as feedback, and stopped exactly one thing: `fb_851c18a9a07b18b6`, whose body asks the reader to *"please provide the full API key"* — a credential-solicitation shape. It scored injection **0.53** (borderline, and the model said so), genuine 0.42, value low. A `gate` threshold anywhere near 0.5 flags it; the mirror uses the bool translation `p ≥ 0.5`, so it rejects. Worth a human look either way — which is what the current human-triage inbox does.
- **Category agreement 44/45 with the submitter's own label.** The one miss is that same message (`limitation` 0.46 vs declared `bug`). The gate's "category must match" branch would therefore have filed **nothing** else — the declared categories on this endpoint are reliable.
- **`value` separates our own probes from real feedback without being told.** The five `low` rows are `TEST - maintainer verification probe`, `M7.1 prod smoke test`, `Connectivity check for feedback publisher`, `[setup-test] motoko_agent integration smoke test`, and the API-key message. Nothing scored `none`.
- **Cost of the whole batch at list price: $0.0028; mean 660 ms per submission via the direct transport.**

**What it does not show:** accuracy against a Haiku arm or a human label — neither exists for this backlog, because the feedback gate is not enabled in prod. The live shadow (`AILANG_FEEDBACK_GATE_SHADOW`) will produce that comparison the day the gate itself is switched on (#900). Until then this batch says: on real traffic the System One arm's would-do distribution is sane, its one refusal is defensible, and its value score already knows a smoke test from a bug report.

Rows: `.ailang/state/decisions/feedback_shadow_2026-09-19.jsonl` (D6: full distributions, model, latency). Regenerate: `tools/decisions/feedback_shadow_batch.py`.
