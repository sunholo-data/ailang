# M-MISSION-ELO-ROUTING: Fit ELO ratings over the mission loop's own execution history, and graduate model routing from availability-probing to rating-based, pre-registered lane assignment

**Tracking**: to be filed against `mission-v1-gh-issue` (#852) at pick time — this doc is written for the V1 mission loop (`design_docs/v1-mission.md`) to take on as a queue row.
**Status**: PROPOSED — draft for the design quorum. Nothing below is approved to route.
**Target**: next sprint window (post-bar positioning — see Priority).
**Priority**: P2. The charter's bar-first ordering (D-28) outranks this doc, and nothing here blocks a bar clause. Its EXPERIMENT-class legs (fits, shadow reports, probe strata) ride the FORUM-RULE direct lane; only the engine code (M1–M3) is feature-shaped and rides the full pipeline.
**Estimated**: 4–6 days across six independently landable milestones; M1–M3 are ~2 days of it.
**Dependencies**: M-EVAL-ROLLING-ELO (landed: `eval_harness.FitFromTrialsAnchored`, `anchor_v1.json`, the `trial_history`/`model_ratings`/`benchmark_ratings` tables), M-COST-PER-SUCCESS-KPI (landed: `chains stats --cost-per-verified-success`, banked baseline $0.2121 after D-44), D-31 (lane capability is structural — the router may only assign structurally-capable lanes), D-40 (the judging apparatus; generator≠judge is non-negotiable).

---

## Problem statement

The V1 mission loop routes every role (controller, designer, planner, executor, evaluator) by
two mechanisms: ordered preference probing (`MISSION_MODEL_PREFS`, availability-first) and
hand-maintained default env pins (`mission-control.sh:481,525`). What it does NOT route by is
**difficulty**: a one-line gate fix and a five-milestone core change get the same lane, because
the loop has never had a number for "how hard is this item".

At the same time the loop is *sitting on* the data that would produce that number:

1. **The loop's own records** — but in a different shape than first assumed (V14, quorum round 1):
   observatory chains are the EVAL corpus (186 `eval_suite` + 2 `github_issue`; `provider` populated
   0/12,985), NOT per-sprint mission executions. The mission substrate is the **325 sprint records**
   + the **mission log's routing evidence rows** (272 recorded, one per iteration: role → model →
   cost) + **git metadata** + **session JSONLs** (stage-level model identity). Observatory contributes
   only the `trial_history`/ratings persistence targets.
2. **325 sprint records** in `.ailang/state/sprints/`, each with features, milestones,
   estimated LOC/duration, evaluator scores and routing evidence rows in the log.
3. **A converged, deterministic, zero-sum ELO fit** that already ranks models against
   benchmarks (`FitFromTrialsAnchored`, `anchor_v1`, 47 benchmarks) — the same mathematical
   object needed for (model, mission-task) trials.
4. **A cost channel**: the cost-per-verified-success KPI (M4b, D-26/D-44) already measures the
   per-verified-success denominator the routing decision is supposed to minimise.
5. **An empty, correctly-shaped table waiting for them**: `trial_history` has exactly the
   needed schema (`benchmark_id, model_id, mode, outcome, ratings before/after, recorded_at`)
   and currently holds **0 rows**.

The consequence of having no difficulty scale is visible in three standing behaviours. First,
the executor fallback chain still ships a DeepSeek default (`mission-control.sh:525`) whose
four-iteration silent-failure history (charter `D-WORLD-20` lineage) had to be diagnosed
manually, by hand-built size polls, because nothing was scoring lane quality per class of item.
Second, routing evidence rows are **recorded every iteration but never scored** — the
"evidence-updated, not vibes" routing table is updated by prose, not by a fit. Third, when a
routing change is demanded, the evidence bar (≥3 datapoints) is met anecdotally rather than by a
rating with error bars.

**The design in one sentence**: extract (model, mission-task, outcome) trials deterministically
from the loop's own records, fit a ratings snapshot with the *existing* anchored ELO engine,
serve lane advice through a new deterministic `ailang route` subcommand that is strictly
ADVISORY until a pre-registered exploration protocol earns an attended ratification, and bank
every probe with the same conditions discipline as `bench-conditions/2`.

## Verification Log

All commands run first-party in the source clone `/Users/mark/dev/sunholo/ailang` at
`0b35abd5d` (design session, 2026-08-28). A picking iteration re-baselines every gate at
pick time per Gate-0 discipline; rows marked INHERITED are carried claims the implementer
re-derives before relying on them.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | The trial substrate exists and is non-trivial | `python3` readonly over `~/.ailang/state/observatory.db` | `execution_chains` = **186** rows; `chain_stages` = **12,916** rows with `chain_id, stage_number, agent_id, provider, session_id, status, approval_status`; `agent_assignments` = 2; `model_ratings`, `benchmark_ratings`, `eval_baselines` tables present — AMENDED by V14: chain composition is eval_suite=186 / github_issue=2, so the per-sprint-item implication is superseded |
| V2 | The write target for mission trials already exists and is empty — i.e. the engine extends a landed schema instead of inventing one | `python3` readonly: `PRAGMA table_info(trial_history)` + count | Schema `trial_id, benchmark_id, model_id, mode, outcome, prompt_version, compiler_version, benchmark_rating_before, model_rating_before, benchmark_rating_after, model_rating_after, recorded_at`; **0 rows**. The persisted `mode` column is exactly the lever D1 uses (`mode='mission'`) |
| V3 | The fitting engine is landed, deterministic, and anchored | `grep -n "FitFromTrialsAnchored\|DefaultInitialRating\|DefaultCoverageThreshold" internal/eval_harness/ratings.go`; `cat internal/eval_harness/anchor_v1.json \| python3 -c '...'` | `FitFromTrialsAnchored(trials, fixedBench, fixedModels)` with deterministic trial ordering and decaying K (≈32→4); flat seed 1500 with an explicit no-tier-biased-seeding guard; coverage threshold 0.9; anchor file = `anchor_v1`, **47** benchmark difficulties |
| V4 | The routing DB already has in-tree consumers and a pinned path | `grep -n "benchmarks-by-confidence" cmd/ailang/eval_suite.go`; `grep -n "func defaultObservatoryDB" -A3 cmd/ailang/eval_confidence.go` | `eval-suite --benchmarks-by-confidence auto` reads `~/.ailang/state/observatory.db` (`defaultObservatoryDB()`); `eval-saturation` likewise. The ratings DB is already a consumed subsystem, not a proposal |
| V5 | Routing today is availability-first, difficulty-blind | `sed -n '481p;525p' tools/launchd/mission-control.sh` | `:481` `export MISSION_EXECUTOR_MODEL="${MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol}"`; `:525` default fallback chain still names **two DeepSeek entries** — the class the chat-ratified chain suspended by hand (attended, 2026-08-19) is present in the shipped default |
| V6 | The per-iteration executor override point exists and is consumed | `sed -n '791,799p' tools/launchd/mission-control.sh` | `$STATE_DIR/mission-${MISSION_NAME}-executor-model-once`: read once per iteration, exported into `MISSION_EXECUTOR_MODEL`, deleted on consumption — the *exact* mechanism M4's active mode would use after ratification, and the advisory-file mechanism can shadow it zero-risk |
| V7 | Sprint records are the plan-side substrate but lack *executed* model attribution — the DB, not the JSONs, is the lane-outcome substrate | sweep all records: `correlation_id` presence, `planner_lane` presence, `features` presence | **325** records; `correlation_id` on **48** (recent), `planner_lane` on **4**, `features` on **312**. Newer records additionally carry `planner_lane`, `mutation_battery`, `pristine_dev_baselines`, `hard_constraints` (sprint_M-LIST-ACCESSOR-API.json) — i.e. declared arms and gate baselines are already per-sprint, but **no record names the executed model**; lane identity is recoverable from `chain_stages.provider`, git metadata and the log's routing evidence rows |
| V8 | The cost channel is a landed KPI | `grep -n "cost-per-verified-success" cmd/ailang/chains_stats.go`; charter D-26/D-44 | `chains stats --cost-per-verified-success` (M-COST-PER-SUCCESS-KPI); frozen-cohort baseline banked $0.2121 (corrected per D-44; original $0.7778 annotated, never restated) — this is the denominator the router's savings claims must be denominated in |
| V9 | The charter's routing table and the driver's default disagree on the executor | charter `v1-mission.md:197` table vs V5 | Charter row: "Sprint execution — Opus — the default, per Mark 2026-07-10"; driver default + every recent log row: `codex:gpt-5.6-sol`. A live reconciliation this doc inherits rather than papers over (Open question Q1) |
| V10 | Lane assignment already carries structural constraints the router must respect | charter D-31; D-40 | The designer rotation splits authoring-capable vs review-capable lanes; `codex:gpt-5.6-sol` is a standing quorum reviewer (author-is-own-judge); gemini/managed_agents is read-only under `CapRemoteSandbox`; pi-lane failures return rc=0 with empty worktrees. The router consumes these as **preconditions**, never as ratings |
| V11 | No existing doc claims this surface | `ls design_docs/planned/ \| grep -ci "routing\|route\|elo"`; `grep -rl "mission.*elo\|elo.*routing" design_docs/` | `0` in planned/; no implemented doc claims mission-loop routing from ratings. M-EVAL-ROLLING-ELO owns the *benchmark* scale and M-COST-PER-SUCCESS-KPI owns cost — both are cited as substrate, not duplicated
| V12 | Contract/verification depth and deontic machinery are real, extractable AILANG features (descriptor fields in D5; scorer follow-up) — not imported assumptions | `grep -n "AILANG_Z3_PATH" internal/smt/solver.go`; `ls internal/parser/parser_contracts.go`; `docs/docs/guides/contracts.mdx`; `head design_docs/planned/v0_29_0/m-contracts-as-code-vertical.md` | Z3 shelled out by `internal/smt/solver.go:76-104` (search order: `AILANG_Z3_PATH` → PATH → common locations); `requires`/`ensures` parsed by `parser_contracts.go`; contracts guide documents runtime + static Z3 verification and IFC labels; M-CONTRACTS-AS-CODE (planned, P1) is the verified-deontic-engine flagship vertical; a `deontic` package exists on the registry (M-DEONTIC-PKG; referenced by v0_29_0 closeouts; absent from this checkout). Re-verified in-session 2026-08-31 at `63545c536` |
| V13 | The boundary gate exists, is green, and its OUTPUT shape bounds what `graph_locality` may claim (round-1 objection, gemini-3-1-pro) | `grep -in "boundar" Makefile make/*.mk`; `sed -n '161,162p' make/code-health.mk`; `make check-boundaries; echo rc=$?`; `go list -deps ./internal/types \| head -3` | CORRECTED in-session: `make check-boundaries` EXISTS — `make/code-health.mk:161`, included from the root Makefile, delegating to `bash scripts/check_boundaries.sh`; run → rc=0, `OK: no architecture boundary violations.` The round-1 record's "no make target exists / AGENTS.md stale" was a verification ERROR (root-Makefile-only grep; annotated, never restated — D-44 discipline) — AGENTS.md and ARCHITECTURE.md are correct as written. Gemini's surviving, valid point: the gate emits a pass/fail violation report, NOT a parseable layer map — so `graph_locality` (D5) sources layer classes from the script's CORE/DASHBOARD/BRIDGE package lists and dependency depth from `go list -deps`. Re-verified in-session 2026-08-31 at `63545c536` |
| V14 | The mission-trial substrate in observatory.db is NOT per-sprint-item mission executions — PoC join (round-1 objection, oc-glm-5-2) | python3 readonly: `SELECT source_type, COUNT(*) FROM execution_chains GROUP BY source_type`; provider/agent_id/approval_status population over `chain_stages`; `grep -c "Routing evidence" design_docs/v1-mission-log.md` | chains=188: **eval_suite=186, github_issue=2** (both failed, single `design-doc-creator` stage) — the observatory chains are the EVAL corpus; `provider` populated **0/12,985**, `approval_status` NULL on all, `agent_id` populated 12,985/12,985, eval-stage model identity lives in `eval_assessment.model` JSON; `chain_stages`→`execution_chains` joins 179/188. Mission (role→model→outcome) trials therefore extract from sprint records + the mission log's **272 routing evidence rows** + git metadata + session JSONLs (M1/D2/D8 amended); D1's namespace separation is now enforced by composition, not just discipline. Re-verified in-session 2026-08-31 at `63545c536` |
| V15 | D3's engine citation resolves cleanly (round-1 secondary catch, oc-glm-5-2) | `sed -n '147,151p' internal/eval_harness/ratings.go` | `FitFromTrials` is documented as "a behavior-preserving delegation to FitFromTrialsAnchored with no fixed entities" — the unanchored mission fit (D5's flat seed) inherits the V3-verified determinism (fixed (bench, model) ordering, decaying ≈32→4); `anchor_v1.json` binds only the benchmark scale. Re-verified in-session 2026-08-31 at `63545c536` |
| V16 | The type checker computes and exposes per-signature effect rows (round-2 objection, gemini-3-1-pro) | `grep -n "effectRow" internal/types/builder.go`; `sed -n '239,251p' internal/types/builder.go`; `grep -n "type EffectRowDiff" internal/types/effect_subsumption.go` | `builder.go:239-251` constructs `effectRow := &Row{…}` with `effectRow.Labels[eff] = &TCon{Name: eff}` per effect and attaches it via `EffectRow:` — the effect/capability annotations `effects_ingress` (D5) counts are computed per signature; `effect_subsumption.go` carries the row-diff machinery. Claim verified TRUE. Re-verified in-session 2026-08-31 at `63545c536` |
| V17 | Role→model identity IS recoverable at per-iteration granularity from the routing evidence rows (round-2 objection, oc-glm-5-2) | sample rows: `sed -n '20226p;20271p;20311p;20343p;6280p' design_docs/v1-mission-log.md`; `grep -c "Routing evidence" design_docs/v1-mission-log.md`; `ls -d ~/dev/mk-*/.motoko/logfile` (empty on this laptop) | **272** routing evidence rows, each naming every spawned role with its model, provider, agent, cost and worktree state — e.g. `controller=claude-opus-5 · designer=fable via Agent-tool model pin · planner=opus · executor=codex:gpt-5.6-sol · evaluator=sonnet` (line 20271). These are the PRIMARY identity source, recorded every iteration by the loop itself; outcome grades join via the iteration's landed/parked/refuted record, sprint-record↔log-row join by sprint/doc id (48 records carry `correlation_id`; the rest join by naming). Session JSONLs are OPTIONAL stage-level enrichment and are machine-local — M1 treats them as such. AC0 added. Re-verified in-session 2026-08-31 at `63545c536` |

**INHERITED, not re-verified here**: ELO `UpdateTrial` is zero-sum with step k and the fit
sorts trials lexically by (Bench, Model-ish) ordering before iterating, so same-trials ⇒
same-ratings — load-bearing for D3's determinism claim; `ratings_test.go` is the authority.

---

## Scope

**IN**: a read-only trial extractor over `observatory.db` + sprint JSONs + git metadata; a
`mission`-mode ELO fit persisted with snapshot IDs; a deterministic `ailang route` advisory
subcommand (JSON, fail-closed); driver integration that is **log-only** until attended
ratification; the pre-registered exploration protocol (strata, caps, escalation ladder,
sequential stopping, auto-suspend/re-qualification); the evidence report that goes to Mark.

**OUT**:
- **The ratified routing chain** (`MISSION_MODEL_PREFS`, planner lane derivation, executor
  chain, evaluator pin). This doc does not change one byte of ratified lane behaviour. The
  flip ADVISORY→ACTIVE is a charter amendment Mark attends, exactly like the DeepSeek
  suspension (attended, 2026-08-19) and the designer-rotation split (D-31).
- **`ailang-world`'s driver** — hand-synced fork, fleet-owned (`D-WORLD-DRIVER-1`); it ports
  the outcome later, not here.
- **Cross-corpus scale anchoring** (making mission-ELO numbers directly comparable to
  AnchorPanelV1 benchmark ELO). Correct thing eventually, wrong thing first (D5).
- **Replacing the pricing work** (`verify-model-pricing`, M4b). The router *consumes* the
  pricing table; it does not re-price anything.
- **A/Bs of prompt text, system roles, skill phrasing** — `eval_prompt_ab.sh` /
  `ab_system_role.sh` territory; the router ships ratings, not prompt drafts.

---

## Design decisions

### D1 — Mission trials live in a separate ELO namespace, never blended with the benchmark scale

`trial_history.mode` already namespaces fits (`standard`/`agent`). Mission trials persist as
`mode='mission'` with a **role-qualified** model id: `mission:planner:opus`,
`mission:executor:codex-gpt-5-6-sol`, so one underlying model can hold different ratings per
role — the loop's own evidence says roles are differently difficult (the same model plans well
and executes differently). Nothing in this doc writes to any `standard`/`agent` row, and no
reader of those rows reads `mission` rows (AC-enforced, MU-2).

Rationale: the benchmark fit measures *canonical-solution capability*; the mission fit measures
*role execution* (did the item land, in how many rounds, at what gate cost). Same math —
`Trial{Model, Bench, Pass}` where `Bench` is a task item id and `Pass` is the outcome grade —
different population. Blending them would put a benchmark difficulty and a queue item on one
scale from two different games; the site's own discipline (per-language fits kept separate
because their scales differ, `eval_elo.go:70-83`) is the precedent to copy.

### D2 — Trial unit, outcome grading, and the exclusion classes are fixed first, fit second

One trial = one (role-lane, task item, attempt) with outcome:

| Outcome | Grade | Notes |
|---|---|---|
| Landed clean: gates green first attempt, no evaluator-blocking findings | 1.0 | the win |
| Landed with rework: ≥1 revision round, park-recovered, or evaluator blocking fixed | partial | graded 0.5 by round count (r1 rework = 0.5, ≥2 = 0.25) — kept SIMPLE and monotone; the fit accepts binary or graded |
| Not landed / reverted / parked by evaluator | 0.0 | the loss |
| `api_error`, provider auth, sandbox bind denial, disk-ceiling kill, probe hang | **EXCLUDED** | not a rating trial at all; counted separately as environment. The charter already owns this classification discipline (`error_category` = *cause unknown*, not *model failed*; the `UNINFORMATIVE UNDER SANDBOX` labelling) |
| Judge-model identity | **recorded, never a reward** | generator≠judge holds structurally (D-40); evaluator-model quality must not leak into `model_id`'s rating — the judge's score qualifies the *task's* bankability, not the judge's own rating |

Every trial row carries: sprint-record + routing-evidence-row + session ids (provenance; observatory chain ids when present — V14), the model's wire version if recorded,
compiler/prompt versions (`prompt_version`, `compiler_version` columns exist for exactly this),
and a `fit` snapshot id. Rows without extractable provenance are **dropped, not guessed** —
the extraction confidence note is banked with the dataset (V7).

### D3 — The fit is the landed engine, deterministic, with snapshot versioning

The fit IS `eval_harness.FitFromTrials` over mission-mode trials — no new math (V15: `FitFromTrials` is a behavior-preserving delegation to the V3-verified `FitFromTrialsAnchored(trials, nil, nil)`, so the unanchored mission scale inherits its determinism; `anchor_v1.json` binds only the benchmark scale). Determinism
(trial order → rating order fixed sort → converged iterate) is inherited and re-asserted by
test. Every fit writes a snapshot: `{fit_id, mode, n_trials, per-lane ratings + coverage,
anchor_version, created_utc}`, content-addressed (`fit_id` = sha256 of canonical trials), with
`model_ratings`/`benchmark_ratings` rows stamped `fit_id`. **Route time never fits** — it
reads the newest fit snapshot; recomputation is a separate, named, replayable act (re-run the
command, get a new `fit_id`, diff it). This is the same discipline as AnchorVersion/anchor_v2,
applied to the mission corpus.

Coverage gating ports unchanged: a lane below `DefaultCoverageThreshold` (0.9) of eligible
items in its stratum is **PROVISIONAL** — the router names it, never acts on it.

### D4 — `ailang route` is a deterministic advisory function with fail-closed output

```
ailang route --task <descriptor> [--role executor|planner|designer] [--db <ratings.db>] [--json]
```

Inputs: the newest `fit_id` for the role, the task descriptor (see D6), the lane roster with
its **structural capability constraints** (D-31/D-40 facts read from a checked-in lanes file,
not inferred), and the pricing table (from `verify-model-pricing`'s consistent data).

Output (JSON): `{task, fit_id, rating, candidates:[{lane, rating, ci, price, eligible, why}],
advice, why}`. Rules:

1. **Fail-closed.** No fit, no coverage, no rating for a candidate, or a task outside the
   descriptor vocabulary → `advice: "no-advice"` and a reason. It never guesses.
2. **Interval routing, not point routing.** A cheaper lane is *advisable* only when
   `rating_ci_low(cheap) > rating_ci_high(task_rating) + margin`. Near the boundary the advice
   is `default` (the ratified chain). Point-estimate ties route to default. This makes the
   engine honest about exactly the cases where ELO is weakest.
3. **Structural constraints are preconditions, not rankings.** A structurally-incapable lane
   is filtered before rating (author-is-own-judge, read-only lanes, review-lane constraints).
4. **Pure and deterministic**: same db + descriptor ⇒ byte-identical output (AC1, MU-1).

### D5 — Task difficulty: descriptors first, kNN prior, online refinement, unanchored scale

Each queue item, at design time, gets a descriptor: `{kind: gate-fix|driver-script|ail-core|
parser|eval-harness|docs, est_loc_band, milestones, mutation_arms_declared, files_touched,
effects_ingress, contract_depth, graph_locality, module_churn}` — the process-shaped fields the
sprint records already carry (V7) or the planner declares, plus four **code-shaped fields**
extracted from the touched sources (V12):

- **`effects_ingress`** — distinct effect/capability annotations in the signatures of touched
  files; a `!{IO, FS, AI}`-ingressing edit is a different risk class than a pure-logic leaf edit.
  Mechanically extractable — the type checker already computes effect rows (`internal/types/builder.go:239-251`, V16).
- **`contract_depth`** — `requires`/`ensures` density and SMT complexity of touched signatures;
  applies **when the task writes AILANG code** (a gate fix on a contract-heavy module is not the
  same difficulty class as a pure-logic leaf module), `0`/not-applicable for Go-side or doc
  targets. Extractable from the touched `.ail` sources' contract clauses (`parser_contracts.go`
  surface, Z3 behind `internal/smt` — V12).
- **`graph_locality`** — dependency-graph depth of the touched files (leaf module vs deep
  multi-layer change): layer classes from `scripts/check_boundaries.sh`'s CORE/DASHBOARD/BRIDGE
  package lists, dependency depth from `go list -deps` (Go targets) or the import walk the
  compiler already performs (`.ail` targets) — the gate emits a pass/fail report, not a parseable
  layer map (V13).
- **`module_churn`** — per-module git churn plus the loop's own historical rework-round rate for
  that module; the extractor already walks git metadata for provenance (M1).

All four are **descriptor fields only**: they feed kNN similarity and the capability side of the
eligibility preconditions; difficulty *numbers* stay owned by the fit (see Considered
alternatives). Difficulty rating:
(a) exact-match an item that has landed before → reuse its rating;
(b) otherwise kNN over descriptor similarity among fitted items → prior + wider CI;
(c) after the item runs, `UpdateTrial` folds the outcome in (online, K-ladder high→low).

The mission scale is **unanchored** (flat seed 1500) in v1 and carries a NON-ABSOLUTE warning:
its numbers are ordinal *within* the mission corpus. Anchoring mission difficulty onto
`AnchorPanelV1` needs bridge games (a benchmark task executed as a mission task) and is
queued as a follow-up item once M1–M3 provide the corpus. Doing it now would manufacture
comparability that the data cannot support — the exact sin the coverage gate exists to stop.

**Follow-up (EXPERIMENT class, direct lane): descriptor scoring in AILANG itself.** Once M1
fixes the descriptor vocabulary, the code-shaped fields are candidates for extraction and
scoring by an AILANG script rather than Go — their substrate is the language's own
type/effect/contract declarations, and the deontic machinery already exists: M-CONTRACTS-AS-CODE's
verified deontic engine (planned, P1) supplies the obligation/permission/prohibition core, and
the `deontic` package is live on the registry (M-DEONTIC-PKG, V12). The fit is structural, not
incidental: **a lane's structural eligibility (D4 rule 3) and D6's strata/NO-GO zones are deontic
predicates** — `permitted(lane, task)`, `forbidden(lane, zone)`, `obliged(lane, gate)` — so the
same scored script can emit descriptor fields *and* eligibility verdicts, with deterministic,
traced execution that makes the scoring evidence itself conditions-bankable. Guardrails: the
scorer supplies **descriptors and preconditions, never ratings** — difficulty stays owned by the
fit and activation by attended ratification, one authority per number; and M1's trial extractor
stays Go in v1 because its substrate (SQLite observatory, sprint JSONs, git metadata) is
host-shaped.

### D6 — The exploration protocol is pre-registered, capped, and self-halting

**Strata** (ratify at pick): `S1 gate-only/script fixes with mechanical gates`; `S2 doc/
comment/renames with deterministic verification`; `S3 sprint-execution on previously-landed
milestone resurfaces (regression-shaped)` — mechanically identifiable, cheap to verify, and
cheap to lose. Structural NO-GO (never probed, ratification-or-nothing): anything touching
core `.ail` semantics, the quorum path, or a ratified gate.

**Down-tier probes (the money mechanism)**: in a stratum, with per-day probability cap
(default: ≤1 probe-day/iteration-day), the executor lane advice names the cheaper eligible
lane. Consumption = the **existing exec-once override file** (V6) — the exact mechanism the
driver already honours, so the probe rides an existing, audited consumption point rather than
adding one. Escalation on failure is the *existing* fallback chain. **Paired trials fall out
for free**: same task, two lanes, one outcome each — the within-task pairing is what kills the
covariate confound, and it costs a wasted partial run when it fails and a *permanent down-tier
move* for the whole stratum when it succeeds. That asymmetry is the whole bet, formalised.

**Up-tier floor probes**: thin, pre-registered, 1-in-20 stratum days: a known-easy item runs on
the expensive lane. Purpose: measure whether the current top-lane necessity is real, not to
save money. Information-poor by construction — sampled thin forever.

**Sequential stopping / auto-suspend**: after **3 consecutive rating-eligible zero-outcome
trials** on one lane (the DeepSeek precedent's measured threshold: 4 zero-byte failures, the
probe blind to all of them), the router marks the lane `SUSPENDED` in the snapshot, stops
advising it, and the morning report says so. Re-qualification = an attended, ratification-note
run that **changes bytes** on the lane (verbatim the D-WORLD-20 re-qualification bar).
Environment-class failures are excluded from the counter (D2) so a bad API day cannot suspend
a healthy lane.

**Every probe banks a conditions row** (`bench-conditions/2` shape: rig load, provider,
version, timestamp, budget metered) — the MJ.B/MJ.C lesson: an A/B without recorded conditions
is not a measurement. Probe results land in the ratings DB with their `fit_id`, never
hand-edited.

### D7 — The driver integration is additive and env-gated

`mission-control.sh` gains: (a) a `MISSION_ROUTE_DB` opt-in env (empty default = engine off),
(b) when set, the same dry-run/log line (V5's lane printout) gains
`route-advice=<advice>(fit=<fit_id>, age=<days>)`, and (c) in ACTIVE mode post-ratification
only, the engine may write the exec-once override file it already consumes (V6). No ratification
of the chain is performed by code in this doc; no `.plist` change; no change to
`derive-planner-lane.sh` (whose fail-closed defaults remain the *floor*, not the *ceiling*:
when the router has no advice, the current chain runs unchanged).

### D8 — Generator≠judge is load-bearing in the data, not just the loop

`chain_stages.provider` is unpopulated in practice (0/12,985 — V14), so judge identity extracts
from the routing evidence rows and session JSONLs, never from that column. Rule: the
evaluator lane's own *identity* is metadata; evaluator *scores* bank task outcomes, never
judge ratings; and any task whose only outcome signal is its own author's evaluation is
`grade=untrusted` and contributes to **descriptive** stats only until an independent judge or
objective gate confirms it. This is D-40's lesson encoded as a trial-inclusion predicate.

### Considered alternatives (external review — Gemini 3.1 Pro; triaged in-session 2026-08-31)

The proposal converged independently on this doc's core — graded 1.0/0.5/0.0 outcomes,
threshold-plus-margin gating, cost-aware selection, capped exploration — convergence is evidence
the shape is right. Its deltas, triaged:

- **Adopted**: the four code-shaped descriptor fields (`effects_ingress`, `contract_depth`,
  `graph_locality`, `module_churn` — D5). The features carry the information; contract depth
  applies specifically to tasks that write AILANG code, where the `requires`/`ensures`/SMT
  surface is the difficulty signal (V12).
- **Adopted as follow-up, EXPERIMENT class**: descriptor scoring by an AILANG deontic script
  (end of D5) — descriptors and eligibility verdicts only.
- **Rejected — closed-form weighted score** (`Ctask = w1·EffectScore + w2·GraphLocality +
  w3·VerificationDepth + w4·HistoricalChurn`): hand-tuned weights are unmeasured constants —
  vibes with numbers, the exact failure mode this doc exists to eliminate. Difficulty is fitted
  from outcomes; features only make kNN similarity meaningful.
- **Rejected — (model, harness) pairwise rating key**: the rating key stays role-qualified model
  (D1) because cell sparsity is the binding limit (186 chains); per-trial
  `prompt_version`/`compiler_version` stamps already preserve the slice to compute post-hoc
  once the corpus supports it.
- **Rejected — blanket ε-greedy** (route 5–10% of low-complexity tasks to lower-ranked models):
  no structural NO-GO zones, no auto-suspend, no conditions banking, no re-qualification bar;
  unpaired probes carry the covariate confound D6's paired-trial design exists to kill. The
  DeepSeek silent-failure precedent (four zero-outcome trials noticed only by hand) is why
  exploration must be pre-registered and self-halting.
- **Rejected — full autonomy framing** ("self-tuning engine, eliminating manual complexity
  guesswork"): ADVISORY-until-ratification (D7) and judge/ratifier identity never entering the
  reward (D2, D8) are load-bearing governance, not friction to automate away.

---

## Milestones (independently landable, each with gates run outside the sandbox)

**M1 — Trial extractor** (`internal/routing/trials.go`, read-only):
mission trials extract from `.ailang/state/sprints/*.json` + the mission log's routing evidence
rows (the PRIMARY role→model source, populated every iteration — V17) + git provenance + session
JSONLs as optional stage-level enrichment (machine-local; absent on this laptop — V17) — observatory.db contributes
only the `trial_history`/ratings persistence targets and the eval corpus for the
namespace-separation test (V14) → typed `MissionTrial` rows, deterministic order (sort by sprint
record, log row, stage id), exclusion classes applied, provenance-confidence stamped. Also emits
each item's D5 descriptor, the four code-shaped fields included (effect rows via the type checker,
contract clauses via the parser surface, import-graph locality per V13, module churn from git).
Fixture: frozen synthetic sprint-record + log-row fixtures in
`internal/routing/testdata/` — no live DB dependency in tests, ever.

**M2 — Fit + persist** (`internal/routing/fit.go`, `cmd/ailang/route_fit.go`):
`ailang route fit --db ~/.ailang/state/observatory.db` → mission-mode fit, coverage-gated,
snapshotted (D3), printed table (lane leaderboard + task-difficulty table side by side — the
`eval-elo` presentation, same code paths where possible) + persisted. Determinism test:
two runs over the same fixture = identical `fit_id`.

**M3 — `ailang route` advisory** (`cmd/ailang/route.go`): descriptor → advice JSON, fail-closed
(D4), kNN prior for unfitted tasks (D5), interval margin. Changelog entry (planner resolves
the live current changelog; `make check-changelog` baselined rc=0 in the lane, iteration-297
lesson).

**M4 — Driver shadow wiring**: MISSION_ROUTE_DB opt-in; log-only lane advice on the dry-run and
iteration-start lines; **zero** lane behaviour change; `tools/launchd/` gate (`test-launchd-drivers`)
baselined and green; the `#558` lesson applies — the driver file reaches the rig at next fire,
so the default-off switch is verified by dry-run evidence, not assumed.

**M5 — Exploration protocol engine**: strata config file (checked in), probe-day scheduler with
caps, escalation ladder consumption via exec-once, sequential-stop counter, auto-suspend +
re-qualification state machine, conditions-row writer. All counters and state changes appear in
the morning report block so the loop cannot silently stop probing.

**M6 — Evidence and the ratification packet**: after N pre-registered shadow iterations,
generate the report: shadow-disagreement log vs actual lanes, per-stratum probe outcomes,
per-lane trials + CI table, cost delta denominated in cost-per-verified-success (V8), and the
one-word DECISIONS FOR MARK question (**A**: ratify advisory→ACTIVE for a named stratum with
the caps as written; **B**: extend shadow; **C**: kill). The loop cannot pick this one (Standing
rule 2), and this packet is designed to make the ruling one glance.

## Acceptance criteria

| AC | Claim (rc-observable) | Base | Post |
|----|----|----|----|
| AC0 | The extractor emits ≥1 trial row from **real (non-fixture) sources** with a verified role→model identity (routing-evidence-row join per V17); a fixture-only trial set does not satisfy M1 | M1 | |
| AC1 | `go test ./internal/routing/ -run TestRouteAssignDeterministic` — same snapshotted fit + descriptor, 2 runs, byte-identical JSON incl. `fit_id` | rc=0 (red→green: M3) | PASS, and a mutant that reorders the candidate sort flips it |
| AC2 | Descriptor with no fit/coverage → `{"advice":"no-advice"}` + reason, rc=0 | M3 | and MU-4 |
| AC3 | Exclusion: a chain marked `api_error`/sandbox-denial never enters trials (fixture asserts absent) | M1 | MU-3 |
| AC4 | Judge-authored evaluation rows land `grade=untrusted` and are absent from `model_ratings` deltas | M1 | |
| AC5 | Two identical fits over the same trial set produce the same `fit_id` (replay discipline) | M2 | |
| AC6 | Coverage-gate: a lane with <90% stratum coverage is flagged PROVISIONAL and excluded from active advice (shadow reports it) | M2 | |
| AC7 | Driver dry-run with `MISSION_ROUTE_DB` prints `route-advice=` with fit id + age; without it, byte-identical dry-run line as today | M4 | |
| AC8 | Auto-suspend: 3 consecutive eligible zero-outcomes → lane flagged SUSPENDED in snapshot + report; the api_error arm does NOT increment | M5 | MU-5 |
| AC9 | Every persisted trial row carries conditions (provider, version, timestamps, outcome class) — schema assertion | M5 | |
| AC10 | `make check-changelog` still rc=0 (M3 changelog entry in the right file) | rc=0 | rc=0 |

**Mutation arms (pre-registered)**: MU-1 shuffle trial order → determinism test RED;
MU-2 delete the `mode='mission'` filter → scale-blend assert RED (benchmark rows must never appear in mission snapshots); MU-3 reclassify one api_error as a loss → AC3 RED; MU-4 corrupt the task descriptor (unknown kind) → fail-closed RED; MU-5 make the suspend counter skip a zero-outcome → AC8 RED; MU-6 drop coverage-gate threshold to 0 → AC6 RED; MU-7 point advice at a structurally-incapable lane (rotated-reviewer constraint) → precondition filter RED.

**Gates**: `gofmt -l` (0 bytes) · `go vet ./...` rc=0 · `go build ./... && go test ./...` rc=0 ·
`make test-launchd-drivers` rc=0 (M4) · `make check-changelog` rc=0 (M3) · all re-run by the
controller outside the executor sandbox (mandatory — the loopback-owning rows especially).

## Risks and the honest limits

- **Sample size**: 186 chains / ~13k stages sounds like a lot and is not — per-lane ×
  per-stratum cells will be sparse for months. The fit is ORDINAL evidence with CIs, published
  as such; the coverage gate is what makes it safe. Do not quote a mission-ELO number without
  its provisional flag, ever (the site dims them for the same reason).
- **Judge bias**: evaluator scores are one model's grades (D-40's apparatus). AC4 keeps them
  out of reward; if judge coverage is poor, M6 reports the fraction — a routing claim resting on
  untrusted grades is explicitly not banked.
- **Non-stationarity**: model providers change under pins; `compiler_version`/`prompt_version`
  columns exist. Fits carry their window; a version bump re-fits under a new `fit_id`, and the
  anchor-bump discipline is follow-up work, not v1.
- **The charter-table/driver-default divergence (V9/Q1)** is a live inconsistency this doc
  surfaces deliberately: it is exactly the class of fact that should be settled by a rating with
  CI, not continued as folklore. The router's first shadow report will quantify it.
- **Cost framing**: savings claims are only bankable in the M4b denominator (V8). A "cheaper
  lane" claim with no cost-per-verified-success comparison is prose, not evidence — the mission's
  own standard.

## Field measurements, 2026-09-06 (attended session — evidence, not design)

Contributed from an attended session that spent a day on mission-loop cost. **Nothing here
changes a design decision in this doc**; it is data the investigation would otherwise have
to re-derive, plus one gap the measurements exposed.

### The role-qualified rating key is right, and here is its magnitude

D-«rating key» keeps the key role-qualified. Measured, that choice is not a refinement —
it is the dominant term. The SAME model in two roles, same day:

| model | role | cost |
|---|---|---|
| `codex:gpt-6-astra` | **controller** | 511,180 · 527,910 · 399,933 (v1) · 682,476 (world) — mean **394,254**, 2,121,499 total over four fires |
| `codex:gpt-6-astra` | designer | never material — bounded to one run per iteration by the Fable diet |

Astra became **60% of all codex spend and emptied a weekly bucket inside a day**, and it
was not a bad model: a controller drives the WHOLE iteration, so its cost scales with
iteration length, while designer/planner/executor are bounded spawns. A rating key that
was model-only would have concluded the model was expensive. The role-qualified key is
what lets it conclude the ROLE is.

### The context prefix is a routing-relevant cost term

The controller's skill file was 251 KB (~63k tokens), loaded every session. Reducing it to
48 KB and dropping the controller tier astra→sol together moved world's cost:

| world, same mission, same day | cost |
|---|---|
| 11:40 pre-change (astra controller, 63k prefix) | 454,310 |
| 12:44 post-change (sol controller, 12k prefix) | **256,656** (−44%) |

Two variables moved at once, so this does not attribute the split — it establishes that
per-run cost is steerable by roughly half without changing the work, which any router
optimising cost should know before it starts trading quality for tokens.

**A correction, so it is not inherited:** during that session I modelled the prefix cost as
`63k × ~50 turns ≈ 3.1M input tokens/iteration`. The measured totals do not support that —
docs' pre-change run was 178k *total*. Caching and real turn counts mean the prefix is not
paid in full every turn. The reduction is still worth having; the multiplier was an upper
bound presented as a mechanism, and it was not one.

### GAP: the cost channel has no notion of bucket scarcity

The cost channel (M4b, D-26/D-44) measures cost-per-verified-success in price terms. It has
no concept of **refill period**, and this doc's own grep confirms it: no mention of weekly
buckets, rolling windows or resets.

That is not an academic gap. On 2026-09-05→06:

- **codex** is a WEEKLY bucket. Exhausted, it returned *"try again at Sep 12th"* — six days.
- **Anthropic** is a ~5-hour rolling window. It emptied overnight and had refilled by 11:10.

So a codex token and an Anthropic token of equal price are **not** equally scarce: Anthropic
refills ~30 times before codex does. A router optimising price-per-success would have
picked astra-as-controller exactly as we did — it was the right *quality* call and the wrong
*bucket* call — and the fleet fell through to pi/ollama lanes for hours.

Suggested shape, for the quorum rather than for the loop to adopt unilaterally: the cost
channel needs a **scarcity multiplier** — remaining-bucket-fraction ÷ time-to-refill — so
"cheap" means cheap *relative to what that bucket has left before it refills*. Availability
probing already answers "is it up?"; this answers "can it afford to be used?".

## Quorum questions (parked for Mark — the loop cannot choose these)

- **Q1**: the executor default divergence — charter table (`opus`, 2026-07-10) vs driver default
  (`codex:gpt-5.6-sol`) — is settled by an attended ratification BEFORE M4 Active work exists, or
  explicitly ruled out of scope.
- **Q2**: ratify the strata definitions and caps (D6): probe-day rate ≤1/day, margin parameter,
  3-strike suspension, re-qualification bar.
- **Q3**: which lanes are route-eligible is a *capability list Mark attends to* (D-31 lineage) —
  the router reads it as config; the list's first draft ships with M4's evidence packet.
- **Q4**: budget ceiling for the shadow phase: suggested $5 metered cap (in line with standard
  iteration metering), exploration legs excluded from quorum budget.

**Suggested queue row**: `m-mission-elo-routing` — P2, post-bar (D-28), feature legs via the
full pipeline, experiment legs direct-lane per the FORUM RULE.