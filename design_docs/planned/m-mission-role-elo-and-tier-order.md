# M-MISSION-ROLE-ELO — tier order should be evidence, not judgement

**Status**: PLANNED · **Created**: 2026-09-22 · **Priority**: P2 · **Est**: 2–3d
**Owner**: attended (harness work — mission-control is attended-only per the 2026-09-21 fleet rule)

## Problem

The mission fleet routes five roles (controller, designer, planner, executor, evaluator)
through ordered fallback chains. **Every rung in every chain was placed by judgement**, and the
chains drifted by accretion: the 2026-09-22 audit found `claude-fable-5-1` sitting LAST in an
ordered preference list at 2.5x the price of the model at the HEAD, and `codex:gpt-6-sol`
occupying two consecutive rungs so that a failed probe was guaranteed to be repeated. Both were
individually reasonable edits. Neither survived being read as a chain.

Mark's stated principle (2026-09-22, attended):

> best model as default, then degrade to slightly cheaper ones if not available in quota - and
> try to rotate around the models we have across anthropic, openai, gemini and
> openrouter/ollama - using ELO to help us pick the tiers.

Three of those four properties are checkable today and one is not. **We have no rating that
measures what these roles do**, so "best" and "slightly cheaper" are currently one measurable
axis (price) and one unmeasured one (capability at the role).

## Why the ELO we have does not answer it

`ailang eval-elo <results_dir>` fits a per-language ELO over **eval benchmark results** —
AILANG-vs-Python code synthesis, with benchmark difficulty fitted separately from model
strength so it survives baseline shifts. It is a good instrument and it measures the wrong
thing for this purpose: the mission roles are orchestration (controller), design authoring
(designer), sprint decomposition (planner), code change (executor) and judging (evaluator).
Only the executor resembles the benchmark task.

The one datapoint that directly contradicts importing it: **kimi-k3 is the strongest
open-weight model on external benchmarks** (88.3 Terminal-Bench 2.1, 81.2 FrontierSWE) and its
single real designer run produced **0 files in 1802s across 73 tool calls**
([[project_designer_rotation_kimi_lane_failed]]). A benchmark rating would have ranked it first
for a role it cannot perform.

## Proposal

Fit a **per-role rating from banked mission outcomes**, using the same anchored-ELO machinery
`eval-elo` already implements, on a corpus of mission runs rather than benchmark runs.

### M1 — make the corpus structured (prerequisite, and the real work)

Per-role outcomes exist today but are **not fittable**:

- `~/.ailang/state/mission-<name>-slot-verdicts.log` is structured and usable, but records only
  the CONTROLLER: 67 rows across 4 missions, 7 distinct controller arms
  (`opus-5` 35, `gpt-5.6-sol` 10, `glm-5.3:cloud` 6, `sonnet-5` 6, `gpt-6-astra` 4,
  `gpt-5.6-luna` 4, `z-ai/glm-5.3` 2).
- The other four roles appear only as **prose in the Gate-4 records** — `executor
  \`pi:ollama/deepseek-v4-flash:0731-cloud\`` and similar — parseable by regex but not a
  contract, and carrying the model without a verdict.

M1 emits one structured row per ROLE per iteration: `(mission, iteration, role, model, lane,
outcome, tokens, wall_clock, cost)`. Outcome is already decided by the loop at Gate 4/5 —
this records what it decided rather than inferring it.

### M2 — fit and expose the rating

`ailang mission role-elo [--role R] [--json]`, anchored the same way as the rolling ELO so a
new model can be placed ALONE without re-running comparators
([[project_rolling_elo_anchored_placement]]).

### M3 — use it to order the chains

A tier order becomes a *claim the instrument can check*: rung N+1 must be cheaper than rung N
(already checkable) **and** within a stated rating band of it (new). A chain that violates
either is a red test, not a thing someone notices six weeks later during an audit.

### M4 — the Google rung (separable, and blocked on its own prerequisites)

Google is one of the four named vendors and appears in **zero** mission rungs. The recorded
reason — gemini cannot author, because the managed-agents lane runs in a server-side sandbox
and file edits never reach the worktree — is correct for designer and executor and **does not
apply to the evaluator**, which emits a verdict rather than files. `gemini-3-1-pro` is already
trusted as a design-quorum reviewer.

The evaluator is also the role that most needs it: three of its four rungs are Anthropic and
the fourth is OpenRouter, so a simultaneous anthropic+openrouter dry-out leaves it with
nothing while planner and executor still hold two lanes each.

It was NOT added during the 2026-09-22 repair because it is not a one-line change:

1. **There is no google bucket in the quota ledger** (`ailang mission quota` prints
   anthropic/codex/ollama/openrouter and nothing else).
2. **`_mc_rung_bucket` has no gemini case**, so a `gemini:*` rung falls to the `*)` default and
   is rationed against **anthropic** — blocked in precisely the state it exists to cover.
3. Gemini is metered, so it needs a spend ration like OpenRouter's, not a percentage gauge.

## Verification Log

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | `eval-elo` fits over an eval RESULTS DIR, not mission outcomes | `ailang eval-elo` usage | `Usage: ailang eval-elo <results_dir>`; "Fit per-language (AILANG vs Python) ELO over an eval results directory" |
| V2 | No per-role/agent rating exists to read | `ailang eval-elo agents --json` | `failed to load results: directory not found: agents` |
| V3 | **NEGATIVE**: no google/gemini bucket in the ledger | `ailang mission quota \| grep -i 'gemini\|google'` | empty; buckets are codex, ollama, anthropic, openrouter |
| V4 | **NEGATIVE**: `_mc_rung_bucket` has no gemini case | `grep -A10 '^_mc_rung_bucket()' \| grep -ic gemini` | `0` — falls to `*)` → anthropic |
| V5 | **NEGATIVE**: gemini appears in no mission chain | grep of all role pins/fallbacks | `0` |
| V6 | Controller corpus size and arms | `wc -l` + `uniq -c` over `mission-*-slot-verdicts.log` | 67 rows, 4 missions, 7 controller arms |
| V7 | Non-controller roles are prose-only | grep of Gate-4 records in `world-mission-log.md` | matches like ``executor `pi:ollama/deepseek-v4-flash:0731-cloud` `` — no verdict field |
| V8 | kimi-k3 designer failure is real, not folklore | `[[project_designer_rotation_kimi_lane_failed]]` | 1802s / 73 tool calls / 0 files |
| V9 | Anthropic's limit is account-wide (why extra same-bucket rungs add probes, not availability) | driver comment recording the 2026-08-16 drought | opus-5 / opus-4-8 / fable-5 all quota-limited together, 45 refusals each |

## Risks

- **Small-N.** 67 controller rows across 7 arms is thin, and the other roles start near zero
  once M1 lands. The rating must report confidence and REFUSE to order rungs it cannot
  separate, rather than emitting a precise-looking number. A tier order justified by an
  unseparated rating is worse than one justified by judgement, because it looks measured.
- **Confounding by task difficulty.** Models are not assigned to iterations at random — the
  expensive ones run when the queue row is hard. This is exactly what `eval-elo`'s separate
  difficulty fit exists for, so reuse it rather than fitting raw win rates.
- **Harness defects read as model defects.** [[feedback_motoko_never_model_wall]]: every
  "model wall" so far has been a harness bug. Rows whose outcome is a harness fault must be
  excluded before fitting, and the exclusion must be visible in the output.
- **Rotation is not a degrade chain.** The designer rotation deliberately alternates vendors at
  equal standing; M3's monotonic-cost rule must not be applied to it.

## Out of scope

- Changing the Fable diet (a spend-policy ruling, separate).
- M-MODEL-REGISTRY-SINGLE-SOURCE M8 — the registry/driver split stays as it is; this doc adds
  no new duplication and would benefit from M8, but does not wait for it.
