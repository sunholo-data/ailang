# Smoke-gate precedents

Moved out of SKILL.md §5 (2026-09-29). Dated measurements that shaped the smoke-gate rules;
the rules themselves stay in SKILL.md.

**2026-05-04 finding (precedent):** Tested 6 SOTA OS models (Gemma 4 26B, Qwen3
30B-A3B, Qwen3 235B-A22B, DeepSeek V4 Flash, Kimi K2.6, Qwen3 Coder Flash)
against this smoke set. Proprietary baselines passed 3/3; **zero OS models
passed all 3**. Most common failure: WRONG_LANG (model produced Python). Even
frontier-class OS models fall back on training-corpus patterns when given
AILANG's 23k-token teaching prompt — they've seen plenty of Python but very
little AILANG. Two near-misses (`or-gemma-4-26b`, `or-qwen3-coder-flash`)
retained on the watchlist; rest cut.

**Implication for stdlib/prompt work:** the smoke test doubles as a
language-improvement metric. Re-run it after stdlib changes or prompt
revisions; if the near-miss watchlist starts passing the third benchmark, the
language has become more "trainable-feel."

**Caveat — agent mode is a separate gate:** the smoke set above runs in
**standard** (single-shot API generation) mode. Models that fail standard mode
may still perform usefully in **agent** mode (`--agent` flag, opencode/pi
harnesses) where they get multi-turn iteration. If a candidate fails standard
smoke, run `ailang eval-suite --agent --models <candidate> ...` separately
before fully cutting it. Agent mode results don't override the standard-mode
gate but can justify adding the model under a different harness entry (e.g.
`opencode-<candidate>`, `pi-<candidate>`).

**2026-05-04 agent-mode smoke finding (precedent):** Tested 9 OS-via-OR
candidates through opencode harness. Cross-mode behaviour:

| Model | Standard | Agent | Δ |
|-------|---------:|------:|--:|
| **GLM 5** (z.ai) | not tested | **3/3** ✅ | — first OS model to pass |
| Gemma 4 26B | 2/3 | 2/3 | 0 (same near-miss) |
| DeepSeek V4 Flash | 0/3 | 2/3 | **+2** (agent unlock) |
| GLM 4.7 Flash | not tested | 2/3 | — near-miss |
| Kimi K2.6 | 1/3 | 1/3 | 0 |
| Qwen3 30B-A3B | 1/3 | 1/3 | 0 |
| Qwen3 Coder Flash | 2/3 | 1/3 | **-1** (agent regressed) |
| DeepSeek V4 Pro | not tested | 1/3 | Pro under-performed Flash |
| Qwen3 235B-A22B | 0/3 | 0/3 | 0 |

Key takeaways for the model-manager workflow:

1. **Agent mode is not a universal fix.** Most models that fail standard
   smoke also fail agent smoke. Multi-turn helps when the model can read
   compile errors and adjust; it hurts when the model interprets tool-call
   setup as the answer (Qwen3 Coder Flash regression).

2. **Pro tier ≠ better.** DeepSeek V4 Pro (1/3) under-performed V4 Flash
   (2/3) on AILANG smoke. The Pro reasoning/long-output overhead can hurt
   simple-task accuracy. Test both tiers when available.

3. **csv_to_json_converter is a `core`-tier DISCRIMINATOR, not a smoke gate.**
   Of the 27 benchmark runs (9 models × 3), csv_to_json was the single most-failed
   test — only GLM 5 passed it among OS candidates. ⚠️ **CORRECTION (2026-06-02):**
   this is exactly why it must NOT gate inclusion — it's failed by the *majority of
   frontier models* (gpt5 base, gemini-3-pro, gemini-3-flash, sonnet-4-5, gpt5-mini
   all FAIL; only opus-4-6/4-7, sonnet-4-6, gemini-3-1-pro, gpt5-2-codex/gpt5-4
   pass). It lives in `tier: core`, not `tier: smoke`. Use it as a high-signal
   **ranking/discriminator** metric in `--tier core` runs and as a language-
   improvement tracker — never as an OS-model include/exclude gate. The gate is
   `--tier smoke`.

4. **GLM 5 is genuinely cost-competitive frontier OS.** $0.60/$2.08 per 1M
   tokens, ~5–7× cheaper than Claude Sonnet 4.6 on input. Worth standing
   inclusion in eval rotation alongside frontier proprietary models.

5. **Vendor-prefix wiring is forward-compat infrastructure.** When adding
   models from a new vendor (e.g. `z-ai/`, `moonshotai/`, `microsoft/`,
   `minimax/`), add the prefix to
   `internal/ai/config.go::openrouterVendorPrefixes` so future ad-hoc
   `ailang run --ai vendor/model` invocations work without needing a
   models.yml entry.

6. **Per-benchmark timeouts can be tighter than agent-mode needs.** The
   `csv_to_json_converter.yml` spec has `timeout: 90s` baked in (set to
   match Claude Sonnet 4.6's ~43s typical solve time). OS models in agent
   mode routinely need 90–180s of iteration on csv_to_json — they CAN
   solve it but get killed by the timeout. Two follow-up models that
   demonstrated this on 2026-05-04:
     - **Kimi K2.6** (Moonshot): fizzbuzz✅ 119s, adt_option✅ 47s,
       csv_to_json❌ (timeout — initial run also had api_errors)
     - **MiniMax M2.7**: fizzbuzz✅ 46s, adt_option✅ 42s,
       csv_to_json❌ (timeout, not capability)
   Both are effectively 2/3 near-misses pending a benchmark timeout bump.
   When investigating a model that fails only csv_to_json with
   `error_category=api_error` and stderr saying "exceeded hard timeout
   (1m30s)", the failure is the benchmark spec, not the model.

7. **api_error vs syntax-error vs WRONG_LANG matters.** When tabulating
   smoke results, always check `error_category`:
   - `api_error` — infrastructure issue (rate limit, timeout, network).
     Re-run before counting against the model.
   - `compile_error` (no err_code) — syntax-error: model produced AILANG
     that doesn't parse. Genuine model gap.
   - `WRONG_LANG` — model produced Python/JS/etc. instead of AILANG.
     Genuine prompt-following gap.
   - `runtime_error` — compiled but crashed. Logic bug in generation.
