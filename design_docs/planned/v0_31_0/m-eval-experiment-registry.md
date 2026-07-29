# M-EVAL-EXPERIMENT-REGISTRY: Preregistration You Can Execute

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1
**Estimated**: 3 days
**Created**: 2026-07-29
**Dependencies**: [M-EVAL-MEASUREMENT-CONTRACT](../../implemented/v0_31_0/m-eval-measurement-contract.md) (shipped) — supplies validity, paired/McNemar, headroom, canary

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | An experiment becomes a frozen artifact: same declaration ⇒ same arms, benchmarks, N, seed. Today those live in shell and drift silently |
| A2: Replayability | +1 | The declaration is banked with the result, so a comparison can be re-derived rather than trusted |
| A3: Effect Legibility | 0 | No language-level effects |
| A4: Explicit Authority | 0 | No new ambient access |
| A5: Bounded Verification | +1 | Treatment integrity and first-run gates are bounded, local, pre-commitment checks |
| A6: Safe Concurrency | 0 | Runner is serial, as the rig is |
| A7: Machines First | +1 | Turns a prose preregistration into a machine-checkable declaration — the whole point |
| A8: Minimal Syntax | 0 | Config, not language |
| A9: Cost Visibility | +1 | An experiment declares its run cost (arms × benchmarks × N) before consuming a night of rig time |
| A10: Composability | +1 | One runner over any experiment; the analysis half already composes over any two result dirs |
| A11: Structured Failure | +1 | "Void by preregistration" becomes a typed outcome (`treatment_unproven`) rather than a silently-reported null |
| A12: System Boundary | +1 | Makes the experiment→harness boundary explicit and asserted |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes a source of drift
- [x] A3 (Effects): none
- [x] A4 (Authority): none
- [x] A7 (Machines First): the central goal

---

## Problem Statement

M-EVAL-MEASUREMENT-CONTRACT made *banked data* trustworthy. It did nothing for *how an experiment is defined and run*, and that half is still bespoke.

**The standard already exists — as prose.** [`m-eval-fmt-weakmodel-ab-prereg.md`](../../implemented/v0_31_0/m-eval-fmt-weakmodel-ab-prereg.md) is a genuinely rigorous preregistration: frozen hypothesis, frozen matched benchmark set, frozen model/N/settings, frozen treatment definition, frozen metrics, frozen confidence method and refutation threshold, reporting commitments, and dated amendments. It is exactly the right shape.

Nothing executes it. The experiment it describes is *separately* hand-wired into `tools/launchd/nightly-eval.sh`, where two experiments now live as hardcoded shell blocks (`RUN_AB` for microRAG, `AILANG_AB_FMT` for fmt), each duplicating its own validity gate, banking, git commit, and messaging. Adding a third means copying ~60 lines — and the last time that shell was hand-written it produced the `${PASS_ON:-0}` bug that banked a 0/84 harness artefact as a real measurement.

**The gap is not analysis.** Validity, paired/McNemar, headroom and `eval-paired` are already generic: they work on any two result directories, for any executor. What is missing is a declarative *definition* and a single runner.

### Two failures this would have caught, both found on 2026-07-29

**1. The arm label is wrong, so the fmt A/B is void by its own preregistration.**
The motoko-extension ON arm (`motoko-local-qwen3-6-fmt`, profile `ollama_fmt`) banks `fmt_hook_state: "off"` and zero `fmt_hook_events`. `fmt_hook_state` is resolved from the **Claude** `-fmt-hook` flag (`internal/eval_harness/fmt_hook_mode.go`), which a motoko-profile arm never sets. So both arms would be labelled `off`, and there is no evidence the treatment applied at all.

The M6 doc's own verification gate says a `fmt_on` run must bank `fmt_hook_events > 0`, and M5's void clause says an unproven treatment voids the result. That gate is prose in a document; nothing enforces it. A run scheduled tonight would produce a number that looks like a finding and is not one.

**2. Nothing gates an experiment on ever having run.** Every bug in the measurement-contract sprint came from code that had never been executed: a panic in the canary, the canary probing the wrong subject, the canary never calling `HealthCheck` (so it could not find the session log), and a dedup key that silently dropped half of every multi-trial run. Unit tests missed all of them; the first real execution found each within minutes.

**Who is affected:** every "does technique X help?" decision, and — per the stated goal — every future feature introduction, since this is intended to become the standard shape for introducing one.

---

## Goals

**Primary goal:** An experiment is a declaration, not a shell script — and it cannot bank a number until it has proven it actually applied its treatment.

**Success metrics:**

1. microRAG and fmt are both expressed as declarations; the two bespoke shell blocks in `nightly-eval.sh` are deleted, not supplemented.
2. Adding a third experiment requires **zero** new shell.
3. An experiment whose treatment cannot be shown to have applied banks `validity: invalid(treatment_unproven)` rather than a delta.
4. An experiment cannot be scheduled until it has completed one real end-to-end run (the first-run gate).
5. The declaration is banked with every result, so an old comparison can be re-derived.

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Cost to change later |
|---|---|---|---|---|---|
| D1 | Declaration format | (a) YAML in `experiments/` · (b) a block in `models.yml` · (c) Go structs | **(a)** — mirrors `benchmarks/*.yml`, reviewable in a PR, and a prereg is a document | Mark | Medium |
| D2 | Arm configuration model | (a) harness-flag only · (b) models.yml-entry only · (c) both, declared per arm | **(c)** — the two live experiments already use different mechanisms (`--microrag on/off` vs two model entries); a registry that supports only one cannot express what we already run | Agent | High |
| D3 | Treatment-integrity proof | (a) trust the config · (b) require a declared observable and assert it · (c) warn only | **(b)** — (a) is exactly how the fmt arm reached "void but reported"; the prereg docs already demand it in prose | Mark | Medium |
| D4 | First-run gate enforcement | (a) advisory warning · (b) refuse to schedule until a recorded successful run exists | **(b)** — every bug this sprint came from unexecuted code; an advisory gate would be ignored | Mark | Low |
| D5 | Scheduling | (a) registry owns cron/day · (b) declaration states cadence, `nightly-eval.sh` asks the registry what is due | **(b)** — keeps launchd as the only scheduler; the registry stays a library | Agent | Low |

### Design Freeze

- [ ] D1 confirmed (a) — YAML in `experiments/`
- [ ] D3 confirmed (b) — declared observable, asserted, voids on failure
- [ ] D4 confirmed (b) — hard gate

---

## Solution Design

### Overview

Make the existing prose preregistration executable. The YAML schema deliberately mirrors the seven frozen sections of `m-eval-fmt-weakmodel-ab-prereg.md`, so writing an experiment *is* preregistering it.

### Architecture

```
experiments/fmt-motoko-ext.yml          <- the preregistration, executable
        │
        ▼
internal/eval_experiment/               <- NEW
   registry.go   parse + validate declarations
   runner.go     run both arms via the existing eval-suite path
   integrity.go  assert the declared treatment observable  ── voids on failure
   schedule.go   "what is due today?"
        │
        ▼  reuses, unchanged:
   FilterCanaryHealthyModels · Validity · PairArms/McNemar · CheckHeadroom
        │
        ▼
   experiments/<name>_ab.jsonl  (result + the declaration that produced it)
```

### The declaration

```yaml
name: fmt-motoko-ext
hypothesis: >
  Forcing `ailang fmt --write` on every .ail write removes syntax drift, so a weak
  local model spirals into compile-stuck loops less often.

subject:
  model: motoko-local-qwen3-6-35b-a3b-mxfp8   # the OFF arm is the subject baseline
  min_headroom: 0.10                          # refuse a control arm above 90%

arms:
  on:
    model: motoko-local-qwen3-6-fmt           # models.yml entry (profile ollama_fmt)
  off:
    model: motoko-local-qwen3-6-35b-a3b-mxfp8

# What must be TRUE for the treatment to count as applied. Without this an
# experiment can report a confident null while the treatment never fired —
# which is the current state of the fmt A/B.
treatment_integrity:
  on:  { field: fmt_hook_events, min_count: 1 }
  off: { field: fmt_hook_events, max_count: 0 }

benchmarks: [contract_rle_roundtrip, config_file_parser, contract_roman_numeral]
trials: 5
langs: [ailang]
schedule: monday
```

`--microrag on/off` is expressed the same way, with `flags:` instead of `model:` per arm — which is why D2 must support both.

### Implementation Plan

**M1 — Schema + registry (~1 day).** Parse and validate `experiments/*.yml`. Reject a declaration that omits `treatment_integrity` (D3): an experiment that cannot say how it would know the treatment applied is not ready to run. Bank the declaration alongside results.

**M2 — Runner (~1 day).** Execute both arms through the existing `eval-suite` path, then hand the two directories to `PairArms`. No new analysis: the point is that the analysis half is already generic.

**M3 — Treatment integrity + void (~0.5 day).** After each arm, assert the declared observable. Failure ⇒ `validity: invalid(treatment_unproven)` and no delta is reported. This is M5's prose void-clause made executable.

**M4 — First-run gate + schedule (~0.5 day).** A declaration carries `first_run: {at, result_dir}`, written only by a real successful run. `schedule.go` refuses to return an experiment as "due" until that exists. `nightly-eval.sh` shrinks to: ask what is due, run it.

### Files to Modify/Create

| File | Change | Est. LOC |
|---|---|---|
| `internal/eval_experiment/registry.go` | **new** — schema, parse, validate | +200 |
| `internal/eval_experiment/runner.go` | **new** — run arms, delegate to PairArms | +180 |
| `internal/eval_experiment/integrity.go` | **new** — assert observable, void | +120 |
| `internal/eval_experiment/schedule.go` | **new** — due-today + first-run gate | +90 |
| `cmd/ailang/eval_experiment.go` | **new** — `ailang experiment run/list/validate` | +150 |
| `experiments/microrag.yml`, `experiments/fmt-motoko-ext.yml` | **new** — the two live experiments | +60 |
| `tools/launchd/nightly-eval.sh` | **delete** both bespoke blocks | −180 |
| `internal/eval_harness/fmt_hook_mode.go` | record the arm from the experiment, not only the Claude flag | +30 |

---

## Examples

### The fmt A/B, today vs under the registry

```
# today — runs, banks a delta, and is void without saying so
fmt A/B result: on=.../84 off=.../84   [fmt_hook_state "off" on BOTH arms; 0 events]

# under the registry
$ ailang experiment run fmt-motoko-ext
→ canary: both arms healthy
→ arm on:  treatment integrity FAILED — expected fmt_hook_events >= 1, got 0
✗ VOID by preregistration (treatment_unproven). No delta reported.
  The treatment cannot be shown to have applied; a null result here would be
  meaningless. Banked with validity.reason=treatment_unproven.
```

### Adding a third experiment

Write `experiments/my-thing.yml`. Run it once (the first-run gate). It is then eligible to be scheduled. No shell is edited.

---

## Success Criteria

- [ ] microRAG and fmt run from declarations; both bespoke shell blocks are **deleted**
- [ ] A declaration without `treatment_integrity` is rejected at validation
- [ ] An arm failing its integrity assertion banks `invalid(treatment_unproven)` and reports no delta
- [ ] An experiment that has never completed a real run cannot be returned as "due"
- [ ] The declaration is banked with each result and the comparison is re-derivable from it
- [ ] The fmt A/B's arm-labelling bug is fixed: the ON arm is recorded as ON
- [ ] `make test` green; `make check-boundaries` green

## Testing Strategy

- **Unit**: schema validation (esp. rejecting a missing `treatment_integrity`); integrity assertion on fixtures; first-run gate; due-today logic across weekdays.
- **Regression**: replay the banked 2026-07-27 microRAG arms through the registry runner's analysis path and reproduce `delta_pp = +13.1` — the same anchor M3 used.
- **End-to-end**: run `fmt-motoko-ext` for real once. Per the first-run gate, this is not optional — and per this sprint's evidence, it is where the bugs are.

## Deferred Decisions

Agent latitude: YAML field naming; whether `ailang experiment` is one command with subcommands or several; internal file layout; how `first_run` is persisted (in-file vs sidecar).

## Non-Goals

- Multi-arm (>2) experiments. Both live experiments are two-arm; McNemar is a paired two-arm test. Adding N-arm changes the statistics, not just the config.
- Changing benchmarks, tiers, or grading.
- Re-running or re-analysing historical experiments.
- Any language, compiler, or stdlib change. **No Conflict Surface section is required**: this touches `internal/eval_*`, `cmd/ailang`, `tools/`, and a new `experiments/` directory — none of the parser/typechecker/codegen/effects surfaces that mandate one.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 schema + registry + validation |
| 2 | M2 runner, both experiments expressed as declarations |
| 3 | M3 integrity/void + M4 first-run gate + delete the shell blocks + docs |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The schema cannot express a future experiment, so someone adds shell again | Medium | High | D2 requires both arm mechanisms up front, because the two live experiments already differ; add an `env:` escape hatch per arm |
| Treatment-integrity observables differ per experiment and the assertion grows special cases | Medium | Medium | Keep it to field + count bounds over banked fields; anything richer is a signal the experiment needs its own metric, not a richer schema |
| Deleting the shell blocks loses behaviour nobody documented | Medium | High | Port with the 2026-07-27 replay as the regression anchor before deleting; delete in the same PR so there is never a second path |
| The first-run gate blocks a genuinely urgent experiment | Low | Low | The gate is "has it run once", satisfiable in minutes with `--trials 1` on one benchmark |

## Related Documents

- [M-EVAL-MEASUREMENT-CONTRACT](../../implemented/v0_31_0/m-eval-measurement-contract.md) — supplies every analysis primitive this reuses. This doc adds the definition/execution half it deliberately left out.
- [m-eval-fmt-weakmodel-ab-prereg.md](../../implemented/v0_31_0/m-eval-fmt-weakmodel-ab-prereg.md) — **the prose standard this makes executable.** The YAML schema mirrors its frozen sections.
- [m-eval-fmt-weakmodel-ab-M5-hardset-prereg.md](m-eval-fmt-weakmodel-ab-M5-hardset-prereg.md) — source of the void clause M3 implements.
- [m-eval-fmt-weakmodel-ab-M6-motoko-ext.md](m-eval-fmt-weakmodel-ab-M6-motoko-ext.md) — its outstanding verification gate (`fmt_hook_events > 0`) is the concrete first user of `treatment_integrity`.
- [measurement-contract guide](../../../docs/docs/guides/evaluation/measurement-contract.md)

## Verification Log

Checked against the code on 2026-07-29.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | (negative) No experiment/registry concept exists | `grep -rniE "type Experiment\|experiments\.yml\|ExperimentRegistry"` over `internal/`, `cmd/` | CONFIRMED ABSENT |
| V2 | Preregistrations exist but only as prose | `find design_docs -name "*prereg*"` | CONFIRMED — 2 docs, both markdown, no machine-readable form |
| V3 | Two experiments are hardcoded in the nightly shell | read `tools/launchd/nightly-eval.sh` (`RUN_AB`, `AILANG_AB_FMT`) | CONFIRMED — 5 matching lines, two independent blocks |
| V4 | The analysis half is already generic | `eval-paired` run against two arbitrary result dirs | CONFIRMED — reproduced `delta_pp=+13.1`, McNemar p=0.0455 on the real 07-27 data |
| V5 | The fmt ON arm is mislabelled `off` | read a banked row from a live `motoko-local-qwen3-6-fmt` run | CONFIRMED — `fmt_hook_state: "off"`, zero `fmt_hook_events` |
| V6 | `fmt_hook_state` is resolved from the Claude `-fmt-hook` flag, not the motoko profile | read `internal/eval_harness/fmt_hook_mode.go` (`ResolvedState`); `ailang eval-suite --help` shows `-fmt-hook` | CONFIRMED — a motoko-profile arm never sets it |
| V7 | The harness already has a `fmt_hook_events` sink it can read | `grep fmt_hook_events` → `fmt_hook_mode.go` (`fmtHookSinkName`), `metrics.go`, `agent_runner.go` | CONFIRMED — the plumbing exists; nothing asserts on it |
| V8 | M6's outstanding gate is exactly `fmt_hook_events > 0` | read `m-eval-fmt-weakmodel-ab-M6-motoko-ext.md` §"Verification gates" | CONFIRMED |

## Future Work

- N-arm experiments (needs a different statistic than McNemar).
- A dashboard view of experiments: declaration, status, last verdict, validity.
- Extend `treatment_integrity` to the config-mismatch check M4 shipped, so "ran the wrong profile" and "treatment never fired" are one mechanism.
