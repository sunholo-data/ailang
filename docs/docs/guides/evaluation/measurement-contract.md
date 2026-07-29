# The Measurement Contract

Every number in a trend line has to earn its place. This page describes the
checks that stand between a run and a banked datapoint, and — more usefully —
the four incidents that made each one necessary.

The pipeline used to be unable to distinguish **"the subject was measured and
did badly"** from **"we failed to measure the subject."** Both produce a low
number, and both were banked as findings.

## 1. Pre-flight canary — is the subject alive?

`HealthCheck` proves a CLI is installed. It does not prove the subject *works*.

motoko's `HealthCheck` passed **72 times** between 2026-07-22 and 07-28 while
its AILANG core was completely dead: `motoko --version` is handled at the
TypeScript argv level and exits before any AILANG module loads. Six nights of
"benchmark failures" were banked, indistinguishable downstream from a model
that simply could not solve the problems.

`CanaryCheck` runs one trivial end-to-end task per model and asserts the subject
got as far as dispatching a tool call. A model that fails is **skipped before
the run matrix is built**, so it cannot bank a single row.

```bash
ailang eval-suite --agent --models motoko-local-qwen3-6-35b-a3b-mxfp8 ...
# → canary: FAILED after 4.1s — effect checking failed in src/core/tool_runtime
#   SKIPPING model (canary_failed). 0 rows banked.

ailang eval-suite --agent --no-canary ...   # override
```

It is an **optional interface**: executors opt in one at a time, and one that
does not implement it is treated as passing.

## 2. Validity — is this row a measurement?

Every banked result can carry a `validity` field:

```json
{"validity": {"valid": false, "reason": "zero_pass_all",
              "detail": "..."}}
```

Reasons: `canary_failed`, `zero_files`, `zero_pass_all`, `config_mismatch`,
`harness_error`, `infra_outage`.

Analysis **excludes invalid rows by default**. That direction is deliberate: if
excluding garbage required opting in, the next analysis written in a hurry would
silently include it again. Use `LoadResultsIncludingInvalid` to reach quarantined
data — it is annotated, never deleted, because the row is evidence of the bug
that produced it.

**A missing `validity` field means valid.** Every row banked before v0.31.0 lacks
it, and treating absent as invalid would erase the entire history.

## 3. Paired analysis — can this comparison resolve anything?

Comparing two aggregate pass *rates* discards the pairing and leaves the
dominant between-benchmark variance in the error term. At n=84 and p≈0.73 the
unpaired difference has a ~6.8pp standard error, so **only effects above ~13pp
could ever reach significance**. Three weekly microRAG deltas (−3.1, −4.8, +13.1)
all sat inside that band.

Pairing on the benchmark cancels that variance — concordant pairs carry no
information about the treatment, only discordant ones do. Same runs, no extra
GPU time:

```bash
ailang eval-paired <on-arm-dir> <off-arm-dir>
```

Replaying the real 2026-07-27 A/B: the aggregate delta is unchanged at +13.1,
and the pairing yields b=18, c=7 → χ²=4.0, **p=0.0455**.

Two guardrails:

- **No p-value below 10 discordant pairs.** The counts are always shown so you
  can see the evidence base; the statistic is withheld until there is one.
- **Unpaired rows are surfaced**, never silently dropped — a rising unpaired
  count is itself the signal that one arm is failing to run.

## 4. Resolved config — did it run what it claimed?

All ten cloud motoko entries set no `motoko_profile`, so they fell through to
`dogfood` — no `ailang_docs`, no `microrag`, and a verify gate (`make
check_core`) that cannot work in a benchmark workspace — for weeks, while their
descriptions advertised "DP7 verifier + microRAG context".

motoko now broadcasts the profile it **actually loaded** at step 0, and the
harness asserts it against the `models.yml` claim. A contradiction quarantines
the row as `config_mismatch`. Reading the value from the subject's own broadcast,
rather than from the config we passed in, is what makes this a check and not a
tautology.

## 5. Headroom — is the subject capable of showing the effect?

An experiment can only detect an effect where the control arm has room to move.
The fmt A/B was run against haiku, where both arms sat at ~96% (30/30 vs 29/30,
42/45 vs 43/45). Those numbers cannot distinguish "fmt does nothing" from "fmt
helps but there was nothing left to fix".

When a control arm is at or above 90%, `eval-paired` warns. It never blocks — a
regression guard on a saturated arm is a legitimate thing to want — but a null
result there means *the benchmark set was already passed*, not *the treatment
does nothing*.

Pick the subject where the failure mode actually occurs.

## Known gaps

- Only the **profile** is asserted. Extensions and the verification command are
  not yet in the step-0 broadcast.
- `CanaryCheck` is implemented for motoko only; other executors default to
  passing.
- Historical `*_ab.jsonl` rows predate `pairs`, so they cannot be re-analysed
  paired unless their per-benchmark result files still exist.
