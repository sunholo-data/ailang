# Ollama Cloud models (flat-rate route)

Moved out of SKILL.md (2026-09-29) — on-demand detail.


Ollama Cloud hosts open-weight models at a flat subscription rate, reachable as a
naming convention on an existing row shape — no new provider code. Canonical design:
[design_docs/planned/v0_34_0/m-ollama-cloud-provider.md](../../../../design_docs/implemented/v0_34_0/m-ollama-cloud-provider.md).

1. **Find the model** — `curl -s https://ollama.com/v1/models | jq -r '.data[].id'`
   (200 unauthenticated; ~19 models). The catalogue id is the TAG (e.g.
   `glm-5.3`, `deepseek-v4-pro:0813`); you request the cloud route by appending
   `:cloud` to the tag.
2. **Test access** — `scripts/test_model_access.sh ollama-cloud <tag>` (no
   `:cloud` — it appends it). Inference auth is the **device key** via
   `ollama signin` (POST localhost:11434/api/me); the daemon proxies
   `:cloud` models to ollama.com and loads nothing on the GPU (V21).
   `OLLAMA_API_KEY` is ONLY for the `GET https://ollama.com/api/usage` gauge —
   which the local daemon does NOT proxy (V24).
3. **Quota semantics (measured, V26/V9)** — `/api/usage` is a numerator with NO
   denominator: no limit/remaining/reset_at is published, so a pre-flight
   "refuse to start if quota low" gate is unbuildable. Metering: model weight ×
   tokens at usage levels 1–4 (`gpt-oss:20b` = 1, `deepseek-v4-pro` = 4);
   session limit resets 5h, weekly 7d; concurrency Free/Pro/Max = 1/3/10 (Max
   paused for new subs); over-concurrency requests QUEUE, not reject.
   `activity.cost` reads `0.00000` forever.
4. **models.yml row conventions (M-Ollama-Cloud-Provider, D1/D2/D6 — RATIFIED):**
   - api_name: `"<tag>:cloud"`, provider: `"ollama"` (same path as local rows),
     env_var: `""`, agent_cli: `motoko`, agent_model_name: `ollama/<tag>:cloud`
   - row key convention: `motoko-cloud-*` (unenforced by test but human-audited;
     the bank stores the ROW KEY, never api_name, so the key IS the route marker)
   - pricing: **IMPUTED from the OpenRouter twin's list price** (D1 — do NOT
     write 0/0; it maps to the false `free-local` provenance). Banks as
     `list-price-equivalent` = "the run went through a subscription lane and was
     never billed". Re-impute whenever the twin's rate drifts.
   - budgets: `max_tokens_per_bench: 3000000`, `hard_timeout_secs: 3600`
   - `default_thinking: "unknown"` until probed (same §2a rules apply — probe
     reasoning with max_tokens ≥ 2000; V25 showed reasoning models burn a
     small budget entirely on thinking)
5. **Scope (D2)** — these are executor/day-to-day rows, NOT banked-eval-rotation
   models unless Mark ratifies: an opaque resetting quota can starve a nightly
   rotation mid-run and bank a cohort with a hole in it.
6. **Concurrency** — cloud rows are EXEMPT from the single-GPU serial clamp (they
   touch no VRAM, D4), but ANY motoko row still serializes on its fixed backend
   port until motoko takes a per-run port.
7. **Quota check as part of the workflow** — snapshot `/api/usage` before and
   after any manual test (the script does this), and expect the numbers to move
   from unrelated traffic: anything else running on the flat route burns the
   same quota (e.g. a coordinator agent session on `glm-5.3-flash:cloud`).
8. **Credit-rate comparison** — before adding a cloud row, and whenever a cost
   question needs answering in Ollama terms:
   `scripts/measure_ollama_credit_rate.sh <tags…>` measures each model's
   units-per-M on the session numerator (V36 method) and prints the ratio
   (measured 2026-08-31: glm-5.3 costs ~3x glm-5.3-flash per token; both match
   their published page levels Medium/High). Rules: one-shot shape unless you
   say otherwise (V46 — agentic meters ~2x cheaper); ≥140k tokens per arm
   because the gauge rounds to 3 decimals; snapshots inside ONE script run;
   cross-check the ratio against the models' published page levels. This
   complements, never replaces, D1: banked dollars stay the OpenRouter twin's
   list price, the credits here are the flat-plan's own internal currency.


## Access-test specifics

The `ollama-cloud` case additionally checks sign-in (`ollama signin` → POST
/api/me), catalogue membership, a reasoning-safe inference probe through the
local daemon, and a before/after quota snapshot from ollama.com/api/usage.


## Credit-rate measurement

Empirical Ollama Cloud credit-rate comparison (V36/V46 method) — the only way to
answer "how many credits does model X cost vs model Y" without a published rate:

```bash
# Compare credit burn per token, e.g. the GLM-5.3 family
scripts/measure_ollama_credit_rate.sh glm-5.3-flash glm-5.3
```

**Output:** per-arm sessions-numerator delta over real tokens burned → units/M,
plus the cross-model ratio:
```
model                    tokens   credits  units/M
glm-5.3-flash            142486    +0.004   0.0281
glm-5.3                 140543    +0.011   0.0783

credit ratio: glm-5.3 costs 2.79x the credits per token of glm-5.3-flash
```

**Measured 2026-08-31 (three runs, one-shot shape):** glm-5.3-flash ≈ 0.03
units/M (page level: Medium), glm-5.3 ≈ 0.08–0.14 units/M (High) — ratio ≈ 3x,
consistent with the published ~3-4x-per-level ladder (V36: gpt-oss:20b 0.0069,
deepseek-v4-flash 0.029, kimi-k3 0.124). Precision is capped by the 3-decimal
numerator: prefer ≥140k tokens per arm, and re-run if an arm shows ≤2 ticks.
