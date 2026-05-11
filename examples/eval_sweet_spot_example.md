# `ailang eval-sweet-spot` — example output

Real output from `ailang eval-sweet-spot eval_results/v0_18_5_core_3harness`,
captured 2026-05-11 against the v0.18.5 core-3-harness benchmark snapshot
(69 agent-mode runs across 22 benchmarks × 3 harnesses).

## Default text output

```
═══════════════════════════════════════════════════════════════════════════════════════════════
  Sweet-Spot Report (slow threshold = 60.0s, 69 total runs)
═══════════════════════════════════════════════════════════════════════════════════════════════

Model                                   Pass%   MedTTS   Tok/s   p90$/win  Fast  Slow  Bdgt   Cap
───────────────────────────────────────────────────────────────────────────────────────────────
claude-haiku-4-5 · claude               95.7%    26.2s     119    0.0793$    20     2     0     1
motoko-claude-haiku-4-5 · motoko        91.3%    22.6s       0    0.1067$    21     0     0     2
opencode-haiku · opencode               87.0%    22.6s     107    0.1892$    18     2     0     3

Cheapest / Fastest Pass per Benchmark
───────────────────────────────────────────────────────────────────────────────────────────────
Benchmark                    Cheapest                           $/win  Fastest                            TTS
api_call_json                motoko-claude-haiku-4-5/motoko   0.0413$ motoko-claude-haiku-4-5/motoko   17.4s
cli_args                     motoko-claude-haiku-4-5/motoko   0.0842$ motoko-claude-haiku-4-5/motoko   32.1s
contract_bst_validate        motoko-claude-haiku-4-5/motoko   0.0342$ motoko-claude-haiku-4-5/motoko   17.9s
csv_to_json_converter        claude-haiku-4-5/claude          0.0737$ claude-haiku-4-5/claude          39.1s
effect_composition           motoko-claude-haiku-4-5/motoko   0.0419$ opencode-haiku/opencode          16.5s
effect_tracking_io_fs        motoko-claude-haiku-4-5/motoko   0.0416$ motoko-claude-haiku-4-5/motoko   17.6s
error_handling               motoko-claude-haiku-4-5/motoko   0.0419$ opencode-haiku/opencode          11.8s
fold_reduce                  motoko-claude-haiku-4-5/motoko   0.0416$ motoko-claude-haiku-4-5/motoko   18.8s
state_machine_vending        claude-haiku-4-5/claude          0.0793$ opencode-haiku/opencode          21.9s
(...22 benchmarks total)
```

## What the report tells you

Reading the row table left-to-right:

- **All three harnesses run the same base model** (`claude-haiku-4-5`), but
  through different agent shells. Bucket counts split that performance:
- **motoko wins on cost** (cheapest model on 14 of 22 benchmarks) — its
  cost-optimization is real.
- **claude direct wins capability** (95.7% pass rate, only 1 capability_blocked)
  — slight edge when motoko's executor crashes.
- **opencode wins speed on small benchmarks** but pays for it with the highest
  `p90 $/win` ($0.189).
- **No `budget_blocked` rows** in this dataset — the motoko cost cap was set
  generously enough that no run hit it. (Future runs may show
  `step_exhausted` and `cost_killed` here once those signals propagate from
  cheaper / longer-running OpenRouter models.)

## CSV output (`--format=csv`)

```csv
model,harness,total_runs,pass_rate,median_tts_ms,median_tokens_per_sec,p90_cost_per_success,speed_efficiency,fast_pass,slow_pass,budget_blocked,capability_blocked,provider_blocked,cost_killed,step_exhausted,timeout,quota_exhausted,rate_limit,api_error
claude-haiku-4-5,claude,23,0.957,26159,119,0.0793,0.689,20,2,0,1,0,0,0,0,0,0,0
motoko-claude-haiku-4-5,motoko,23,0.913,22631,0,0.1067,0.681,21,0,0,2,0,0,0,0,0,0,0
opencode-haiku,opencode,23,0.870,22631,107,0.1892,0.649,18,2,0,3,0,0,0,0,0,0,0

## Champions (cheapest / fastest pass per benchmark)
benchmark_id,cheapest_model,cheapest_cost_usd,cheapest_tts_ms,fastest_model,fastest_tts_ms,fastest_cost_usd
api_call_json,motoko-claude-haiku-4-5/motoko,0.0413,17431,motoko-claude-haiku-4-5/motoko,17431,0.0413
(...)
```

## JSON output (`--format=json`)

```json
{
  "rows": [
    {
      "model": "claude-haiku-4-5",
      "harness": "claude",
      "total_runs": 23,
      "pass_rate": 0.957,
      "median_tts_ms": 26159,
      "median_tokens_per_sec": 119,
      "p90_cost_per_success": 0.0793,
      "speed_efficiency": 0.689,
      "cost_killed_count": 0,
      "step_exhausted_count": 0,
      "timeout_count": 0,
      "quota_count": 0,
      "rate_limit_count": 0,
      "api_error_count": 0,
      "buckets": {
        "fast_pass": 20,
        "slow_pass": 2,
        "budget_blocked": 0,
        "capability_blocked": 1,
        "provider_blocked": 0
      }
    }
  ],
  "champions": [...],
  "slow_threshold_ms": 60000,
  "total_runs": 69
}
```

## See also

- [Cost and speed budgets guide](../docs/docs/guides/evaluation/cost-and-speed-budgets.md) — full schema reference + failure-category taxonomy
- [M-EVAL-SWEET-SPOT design doc](../design_docs/planned/v0_19_0/m-eval-sweet-spot.md)
