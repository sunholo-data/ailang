# M-EVAL-AILANG-ADVANTAGE: A Long-Running Program to Measure, and Then Earn, Where AILANG Beats Python for AI Authors

**Status**: Planned (program doc). Requested by Mark on 2026-10-28 as a long-term project. Phase 0 is running. Quorum blocked twice, every objection fixed; awaiting human review of D1–D5.
**Target**: Phases 0–2 v0.55.0; then once per release, open-ended
**Priority**: P1
**Estimated**: Phase 1 is 1.5 days, Phase 2 is 3 days (excluding model runs); later phases are scoped by their own docs
**Dependencies**: [M-EVAL-HIDDEN-TESTS](m-eval-hidden-tests.md) (shipped to dev on 2026-10-28), the runtime docs in [Related Documents](#related-documents)
**Routing lane**: the eval harness and the teaching prompt. Language fixes this program finds are routed
to the AILANG-fix lane as their own docs (PROGRAM.md §4). The program decides *what to fix next*;
it does not fix the language itself.
**Quorum trigger**: #1 (design-freeze items D1–D5) and #3 (it defines the headline KPI the dashboard
publishes per release). See [Quorum](#quorum).

## Why this exists

PROGRAM.md §7 states the thesis: *can we build a language that AI models prefer and use better than
existing ones?* The v1.0 claim built on it is "the verified AI-orchestration language", measured as
cost per verified success ([m-cost-per-success-kpi](../../implemented/v1_0_0/m-cost-per-success-kpi.md)).

Neither the regular suite nor the competitive-programming (CP) spike can confirm or refute that thesis today:

- **The regular suite is saturated for cloud models** (V1). AILANG agent success in the published v0.54.18 data is
  98.7% for Claude and 95.6% for codex. A suite every strong model passes cannot rank changes.
- **On a non-saturated suite AILANG loses, and the loss is not about the language being hard to write** (V2, V3).
  Matched agent runs (Haiku 4.5, 26 pairs over 17 problems) passed the visible example equally often: 23/26 in
  each language. AILANG's final solutions had no compile errors. The gap appears on the hidden tests: 67% of
  them pass in AILANG against 84% in Python, and only 3 AILANG runs pass every small hidden test against 16 for
  Python. AILANG was never ahead in any pair.
- **AILANG's distinctive tools go unused** (V4). AILANG has inline tests, QuickCheck-style properties
  (`property … { forall(…) => … }`), `requires`/`ensures` contracts and Z3 verification. Haiku 4.5 used
  **none of them in 54 of 54** CP agent sessions. A Python agent can write `assert`s or a brute-force
  cross-check by hand, but the Python grader provides only the standard library (V9), which has no
  property-testing module or input generators (V15). AILANG's real asymmetry is therefore *integrated*
  tooling: generated inputs, seeded counterexample replay, and contracts checked by Z3 (V7). That
  advantage exists on paper and is worth 0 in practice.
- **AILANG agent runs cost about 3× as much** (V5): median $0.42 against $0.14, 23 turns against 8, on the same
  43 problems. Part of that is the 99KB teaching prompt riding on every turn (V6).

This doc turns those observations into a **program**: a fixed scoreboard, a set of falsifiable hypotheses
about where AILANG *should* win, and a once-per-release cadence that sends each loss to exactly one fix lane.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change; scoreboard rows are computed from banked rows, not re-run |
| A2: Replayability | +1 | Every scoreboard number names its cohort (binary SHA, models, benchmarks, trials), and the instrument that produced it can re-derive it |
| A3: Effect Legibility | 0 | No language change |
| A4: Explicit Authority | 0 | No new capabilities; agents keep the same tools |
| A5: Bounded Verification | +1 | H1 makes bounded verification (properties, Z3) part of the agent's loop instead of an unused feature, and measures whether it pays |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Optimises for AI authors on measured outcomes (hidden-test fraction, $/verified success), not on human ergonomics |
| A8: Minimal Syntax | 0 | No syntax. Kill criteria keep the program from adding CLI surface that does not pay |
| A9: Cost Visibility | +1 | Cost per verified success, per language, becomes a scoreboard row every release |
| A10: Composability | +1 | Reuses existing instruments (`eval-paired`, `eval-elo`, `chains stats --cost-per-verified-success`, hidden tests, `--conditions`) instead of building parallel ones |
| A11: Structured Failure | +1 | Each loss is classified (runtime, prompt/discovery, verification, harness) and routed to exactly one lane |
| A12: System Boundary | 0 | No change |

**Net Score: +6**. **Decision: move forward.** No −1 on A1, A3, A4 or A7.

## Hypotheses (each one falsifiable, each with a kill criterion)

| # | Hypothesis | How it is measured | Kill criterion | Status |
|---|---|---|---|---|
| **H1** | **Verification in the loop.** If agents actually use AILANG's tests, properties, contracts and Z3, the hidden-test gap closes, and can open in AILANG's favour. The predicted edge is integrated generators, counterexample replay and Z3, against a Python agent's hand-written asserts and brute-force checks; Phase 2 step 4 measures exactly that difference | A/B on the CP pack: `--conditions baseline,tool_aware` (exists, V8), then the Phase 2 variants; `ailang eval-paired` for discordant pairs | Per the [decision rule](#decision-rule-for-ab-arms): two variants in a row that end **null** or **inconclusive at the full cohort** → stop pushing verification-in-the-loop for algorithmic tasks, and record that | **Unmeasured.** `tool_aware` has no recorded result (V8) |
| **H2** | **Change reliability.** Explicit effects and types mean fewer regressions when an agent changes its own program over several steps | **Owned by the maintenance-eval program** ([taskpack](../m-maintenance-eval-taskpack.md), parent [real-world study](../m-real-world-language-maintenance-study.md)) (V10). This program only consumes its results as a scoreboard row | Their kill criteria | Their pilot is designed, not run |
| **H3** | **Cost.** A smaller, just-in-time teaching prompt cuts AILANG's cost per verified success without cutting success | Section-by-section prompt ablation (PROGRAM.md §5c already names it as open), measured as $/verified success | Per the [decision rule](#decision-rule-for-ab-arms): a trim whose arm ends **worse** is reverted; one that ends **inconclusive** is kept only if S3 falls and the discordant count is under 10 with no more than 2 lost pairs | Unmeasured |
| **H4** | **Runtime parity.** Speed and missing collections cause enough of the gap that, once the VM covers model-written code and cheap collections exist, AILANG's hidden fraction rises with no model or prompt change | Re-run the frozen CP cohort on each runtime landing; count failures by `hidden_first_fail` kind (timeout vs wrong output) | Timeouts already under 10% of AILANG hidden failures → runtime work leaves the CP critical path | Partly measured: 10 of the 26 AILANG spike runs that produced code first failed on a timeout, counting inputs of every size (V2) |
| **H5** | **Model-strength scaling.** The AILANG-vs-Python gap shrinks with model strength, because strong models learn the language from the prompt better | Phase 0's ladder: Haiku 4.5, Sonnet 5.5, Opus 5.5, gpt6-1-sol on the 6-benchmark CP pack, 3 trials | Not a kill; a fork. If the gap does *not* shrink, the limit is the runtime and library, not learnability (routes to H4); if it shrinks, learnability is the lever (routes to H1/H3 and the fine-tuning corpus) | **Answered descriptively** (Phase 0, 2026-10-28): it shrinks; see Phase 0 result |

## Goals

1. **One scoreboard per release** that answers "is AILANG a better language for AI authors than Python, on
   what, and by how much", with every number reproducible from banked rows.
2. **A discriminating benchmark set that stays discriminating**: the hidden-test CP pack grows and rotates
   so that no strong model saturates it, while each benchmark carries an AILANG reference solution
   proving it is solvable.
3. **Find the first task class where AILANG measurably beats Python** (H1 or H2), or show with data
   that none exists yet and name what blocks it.
4. **Every loss routed to exactly one lane**: runtime/library (AILANG fix), prompt/discovery
   (prompt-manager), verification-in-the-loop (this program), harness relic (eval harness).

**Non-goals:** comparing peer AI languages (that was M-THREE-CAMPS); changing graders already shipped;
any motoko core change; building a multi-step maintenance harness (H2 is owned elsewhere); per-benchmark
hints (benchmark-manager rule: if models need hints, the teaching prompt is incomplete).

## The scoreboard (Phase 1)

Computed per release from banked rows, by **language × model tier** (weak = Haiku-class, strong = Opus- or
Sol-class). No new run is needed beyond the release eval.

| Row | Definition | Source instrument |
|---|---|---|
| S1 Hidden-test fraction | Σ`hidden_passed` / Σ`hidden_total` over the CP pack | banked rows (M-EVAL-HIDDEN-TESTS) |
| S2 Full-pass rate | runs with `stdout_ok` (visible plus every hidden test) / runs | banked rows |
| S3 Cost per verified success | the existing KPI, by language | `ailang chains stats --cost-per-verified-success` (V11) |
| S4 Turns and tokens per success | median `agent_turns`, `total_tokens` over successful runs | banked rows |
| S5 Verification-tool use | share of AILANG agent sessions that ran `ailang test`, `--verify`/`ailang verify`, or wrote a contract or property | Per harness, from the transcript that harness keeps: claude session JSONL, codex rollout JSONL, the opencode session DB, motoko's session JSONL (V17). The banked `agent_transcript` holds tool names only (V12). A harness with no transcript instrument reports **n/a (no instrument)**, never 0 and never omitted |
| S6 Failure mix | AILANG hidden failures by kind (`wrong_output` / `timeout` / `runtime_error`) and by lane after triage | `hidden_first_fail` plus the triage table |
| S7 Difficulty-adjusted strength | per-language rating that survives benchmark rotation | `ailang eval-elo <results_dir> --json` (V21) |
| S8 Change reliability | regressions per change (consumed when the maintenance pilot reports) | maintenance-eval program |

S5 is the only row with no banked field. **D2** decides whether it becomes one.

## Workstreams

### Phase 0 — Baseline ladder (running, 2026-10-28)

144 agent runs: 4 models × 6 CP benchmarks × 2 languages × 3 trials, pinned to binary `57b0b8f20`
(so the fix PRs in flight cannot contaminate it). Answers H5 and gives every later phase a frozen
comparison point. Output: the first scoreboard, S1–S7.

**Result (2026-10-28).** Frozen cohort: `eval_results/cp_advantage_phase0_57b0b8f20/` (local; 144 rows).
The 6 rows killed by the sibling-`pkill` bug were re-run on `57b0b8f20` + #1990 only, with the same
prompt hash `305406c50bc7`. S2, full pass (visible plus every hidden test):

| Model | Python | AILANG | Median $ Python / AILANG | Median turns Python / AILANG |
|---|---|---|---|---|
| Haiku 4.5 | 13/18 | **3/18** | 0.15 / 0.46 | 8 / 24.5 |
| Sonnet 5.5 | 14/18 | **17/18** | 0.10 / 0.15 | 3 / 4 |
| Opus 5.5 | 18/18 | 15/18 | 0.19 / 0.67 | 3 / 6 |
| gpt6-1-sol | 18/18 | 16/18 | 0.20 / 0.92 | 1 / 1 |

- **H5: the gap shrinks with model strength.** The weak model loses heavily in AILANG; the strong
  models are near parity, and Sonnet passes more in AILANG. Descriptive only: Phase 0 is not a paired A/B.
- **Strong-model AILANG failures are mostly budget, not wrong code.** Of Opus's and Sol's 5:
  3 hit the 10-minute agent cap, 1 hit Opus's per-run cost cap, and 1 is a codex harness error. Sonnet's
  1 failure is a real wrong answer (`cp_60164`, 1/8 hidden). Python's strong-model failures are the same
  kind: Sonnet hit the 10-minute cap on all 3 `cp_60149` trials. The pack's `timeout: 600` overrides
  `--agent-timeout`; consider raising it.
- **What remains is S3/S4.** AILANG costs 1.5× (Sonnet) to 4.6× (Sol) as much, and agents take more turns,
  which is what pushes them into the caps. So for strong models the levers are H3 (prompt cost) and H4
  (VM speed); for weak models, learnability (prompt, examples, the Phase 5 corpus).
- **S7** (`ailang eval-elo`, per-language scales): the hardest benchmarks for AILANG are `cp_60164`
  (AILANG pass 50% vs Python 100%) and `cp_60159` (73% vs 92%). These are exactly the two that
  [m-persistent-collections](m-persistent-collections.md) Phase 1 (`accumulate`, `std/set`, merged
  2026-10-28) targets, so they are the first re-run after it.

### Phase 1 — Scoreboard and cadence (1.5 days)

- A scoreboard script (or an `eval-report` section; **D1**) that renders S1–S7 from a results directory,
  with each cohort's binary SHA and model list in the header.
- A **post-release step**: the post-release skill runs the CP pack in agent mode for one weak and one
  strong model, 3 trials, and publishes the scoreboard next to the dashboard. The four stretch CP
  benchmarks already run in the post-release tier scope (V18); this adds only the weak/strong pairing,
  the two `experimental` CP benchmarks, and the report.
- A **frozen cohort**: the Phase 0 runs are kept as the reference, so each release is compared with
  `eval-paired` against the same benchmarks rather than against a moving average.

### Phase 2 — Verification in the loop (H1, 3 days plus runs)

Measured in order; each step runs only if the previous one did not already settle H1.

1. **Measure what exists.** `--conditions baseline,tool_aware` on the CP pack, weak and strong model,
   3 trials. `tool_aware` adds contract-writing guidance and an `ai-check` step to the agent prompt (V8).
2. **Property-first variant.** The agent prompt asks for `property` tests on the core function *before*
   `main`, and for running `ailang test` before submitting. This is a prompt condition, not a language
   change. It targets the measured failure: edge cases the agent never exercised.
3. **Differential testing, if 2 shows that agents write properties but they don't catch the bugs.** The
   agent writes a slow, obviously-correct reference function and a property asserting the fast one agrees
   with it on generated inputs. This is the brute-force checker a good Python agent writes, expressed in
   AILANG's built-in tooling. Check first whether generators for the inputs these problems need (lists,
   bounded ints, grids) exist. Missing generators become an AILANG-fix doc, not a hack here.
4. **Fairness control (D3).** If a variant helps AILANG, run the matched Python variant (asking for
   `assert`-based self-tests or a brute-force cross-check) so the comparison measures the *language's*
   tooling, not the extra instruction.

A variant that wins on both tiers and survives the fairness control is proposed as the default agent
prompt (**D4**). That changes baseline comparability, so it needs a recorded boundary.

### Phase 3 — Cost (H3, scoped by its own doc)

Prompt ablation by section, measured as S3 on a frozen cohort. PROGRAM.md §5c already lists "measure
what each section earns before trimming"; this program supplies the metric and the cohort, and the
prompt-manager lane owns the edits. Includes just-in-time teaching: the MCP stdlib tools
(`stdlib_search`, `stdlib_module`) already exist, so the question is whether a shorter system prompt plus
lookup beats the 99KB prefix (`stdlib_search` is defined in `mcp_tools/language.ail`, V19).

### Phase 4 — Runtime parity (H4, consumed)

No work here: the [VM coverage](m-vm-coverage-letrec-prelude-bridge.md),
[persistent collections](m-persistent-collections.md) and
[overflow/BigInt](m-int-overflow-trap-and-bigint.md) docs own the fixes. On each landing the frozen
cohort re-runs and S6's timeout share is the gauge. When `exec_timeout` headroom grows (V16: the `cp_60159`
reference's slowest hidden test takes 8.99s on the evaluator and 0.26s under `--strict-bytecode`), the
pack's inputs can grow past 64KB. Whether the speed deltas the spike saw on `cp_60212` and `cp_60198`
reappear there is a measurement this phase takes, not a premise.

### Phase 5 — Data flywheel (consumed)

Hidden-test-verified AILANG solutions are the first eval output trustworthy enough to train on. They feed
[m-eval-finetuning-data-pipeline](../v1_1_0/m-eval-finetuning-data-pipeline.md) as a labelled source. That
is the long-run answer to AILANG having no pretraining corpus.

### Ongoing — Pack growth and rotation

- **Sources:** more AetherCode problems (CC-BY-4.0; the spike used 29 from the `v1_2024` split; whether
  other splits fit is unchecked), selected by the rule used on 2026-10-28: a delta that holds over 3 or more trials, a Python
  reference passing every hidden test, and an AILANG reference for the `stretch` tier (otherwise
  `experimental`).
- **Rotation** follows `benchmarks/CURATION.md`: when a strong model saturates a CP benchmark in *both*
  languages it is demoted. A benchmark where AILANG wins is kept as value evidence.
- **Attribution:** every benchmark's description says which gap it tracks, or says "unattributed".
  Two of six are unattributed today (`cp_60149`, `cp_60197`), and attributing them is triage work.

## Decision rule for A/B arms

`ailang eval-paired <on> <off>` (V20) pairs runs by benchmark, language and trial. It reports
`delta_pp`, the discordant counts (`only_on_passed` = b, `only_off_passed` = c), a headroom warning,
and McNemar's test over b + c (exact binomial up to 25 discordant pairs, chi-square above). **It withholds
the p-value below 10 discordant pairs** (`MinDiscordantForPValue`). "Pass" is `RunMetrics.Passed()`:
compile, run and `stdout_ok`, and `stdout_ok` already requires every hidden test (V20).

That floor drives the cohort size. One tier on the CP pack is 6 benchmarks × 3 trials = 18 pairs per
language, and only a fraction of those will be discordant. So an arm is read like this:

1. **Run** the arm on the CP pack, 3 trials, for the weak and the strong tier, and pool both tiers in one
   `eval-paired` call (36 pairs per language).
2. **Win** if McNemar is reportable, p < 0.05 two-sided, and b > c. **Worse** if reportable, p < 0.05
   and c > b. **Null** if reportable and p ≥ 0.05.
3. **Inconclusive** if b + c < 10. Double the trials once (72 pairs) and re-read. If it is still under
   10, the arm ends **inconclusive at the full cohort**. That counts as "no measurable effect" for kill
   criteria and can never justify D4.
4. **Hidden-test fraction (S1) is descriptive only.** It is reported next to every arm, because it moves
   before `Passed()` does, but no decision rests on it alone. A paired statistic over `hidden_passed`
   would be a new instrument; Phase 1 decides (D5) whether it is worth building.

The rule is fixed **before** an arm runs, so a result cannot pick its own threshold.

## Loss-routing table (used every release)

| Loss kind | Signal | Lane |
|---|---|---|
| Runtime or library (speed, missing structure, overflow) | `timeout` with a fast reference; Python solution uses a structure AILANG lacks | AILANG fix (own doc) |
| Type-checks-then-crashes | `runtime_error` with a type-level message | AILANG fix (soundness) |
| Prompt or discovery | an existing feature unused (e.g. `std/array` in 1 of 17 solutions); a dialect error | prompt-manager |
| Verification not used | wrong answer on edge cases; S5 low | this program (Phase 2) |
| Harness relic | `no_code` from a kill, prose fed to the compiler, a per-language timeout | eval harness |

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1: Where the scoreboard lives: a section in `ailang eval-report` vs a standalone script under `tools/` | CLI surface is under a simplification budget (CLI 89 → 17); a script adds no command | human | Phase 1 | low |
| D2: Bank verification-tool use (S5) as a row field (`agent_verify_calls`), or keep deriving it from transcripts | Schema change to banked rows (quorum trigger #3); transcript instruments differ per harness (V17): opencode has no in-repo reader and pi has no instrument, so a banked field is the only way every harness gets S5 | human | Phase 1 | medium |
| D3: Fairness rule for H1: does every AILANG prompt condition get a matched Python condition? | Without it a win measures the instruction, not the language | human | Phase 2 | low |
| D4: If a Phase 2 variant wins, make it the default agent prompt | Moves every agent baseline; needs a recorded comparability boundary like the three local-model boundaries | human | after Phase 2 | high |
| D5: Build a paired statistic over `hidden_passed` (e.g. a sign test on per-pair hidden fractions) | Would let decisions use S1, which moves earlier than pass/fail; a new instrument to maintain | human | Phase 1 | low |

### Design Freeze

- [ ] D1 (recommended: a section in `eval-report`, since it already renders banked results)
- [ ] D2 (recommended: derive from transcripts first; bank it only if S5 drives a decision twice)
- [ ] D3 (recommended: yes, always matched)
- [ ] D4 (decided after Phase 2 data, not now)
- [ ] D5 (recommended: not until an arm ends inconclusive while S1 moves by 10 or more points; then it has a use)

## Deferred Decisions (agent latitude)

- The exact wording of the Phase 2 prompt variants, provided each claim passes `ailang check`.
- Which weak and strong models represent each tier in a release, provided both lanes are the direct
  (subscription/OAuth) lanes, never OpenRouter for first-party vendors.
- Scoreboard formatting.

## Success Criteria

- [x] Phase 0: scoreboard S1–S7 for 4 models, with H5 answered descriptively (the AILANG−Python gap in S1 and S2 per model). Phase 0 is not a paired A/B, so it reports no p-value
- [ ] Phase 1: one command renders the scoreboard from a results directory; the post-release skill runs it; the Phase 0 cohort is frozen and documented
- [ ] Phase 2: `tool_aware` measured on two tiers; H1 either confirmed on at least one variant that survives the fairness control, or killed by its criterion
- [ ] Program: within three releases, either a task class where AILANG's S1 or S8 beats Python's on both tiers, or a written analysis of what blocks it

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Small cohorts read as trends | Every A/B goes through `eval-paired` and the [decision rule](#decision-rule-for-ab-arms): McNemar only at 10 or more discordant pairs, otherwise "inconclusive" and the cohort grows; frozen cohorts |
| Prompt conditions overfit the CP pack | Phase 2 variants are general ("write properties first"), not problem-specific; confirm on the regular stretch tier before D4 |
| Subscription and codex quota spend | Weak plus strong only on the release cadence; full ladders only when a hypothesis needs them; `--max-cost-usd` on metered lanes |
| A harness relic masquerades as a language result | The loss-routing table puts harness relics in their own lane; the `no_code` rows from batch kills (14 of 17 in the spike) are excluded from S1/S2 and reported separately |
| Optimising for CP drifts from the real thesis | H2 (maintenance) and H3 (cost) sit on the same scoreboard; CP is one row family, not the headline |

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | The regular suite is saturated for cloud agents | Published `docs/static/benchmarks/latest.json` at HEAD (v0.54.18, 2026-10-26): `languages.ailang.agent_success_rate_claude` 0.987 (78 runs), `agent_success_rate_codex` 0.956 (91 runs) |
| V2 | Matched-pair CP competence (26 pairs, 17 problems, Haiku 4.5 agent) | Re-graded spike rows `grade_ha1–3`: visible 23/26 in both languages; hidden-test fraction 0.84 Python vs 0.67 AILANG; every small hidden test 16 vs 3; AILANG > Python in 0 pairs, equal in 7, lower in 19. First failure of the 26 AILANG runs with code, counting inputs over 64KB too: 16 wrong output, 10 timeout |
| V3 | AILANG's missing-code rows are mostly harness kills | 17 of 43 AILANG agent rows had no code; 14 result files sit in the spike's `killed_143` set (SIGTERM at parallel ≥10) |
| V4 | Verification tools go unused | Parsed tool calls in the Claude CLI session JSONLs of every CP agent session in AILANG (2026-10-28): Haiku 4.5 54 sessions, 0 ran `ailang test`, 0 ran `--verify`/`ailang verify`, 0 wrote `requires {`/`ensures {`, 0 wrote a `property`/`forall(`. Early baseline sessions: Opus 5.5 1 of 5 ran `ailang test` and verify; Sonnet 5.5 0 of 4 (in progress, not a result) |
| V5 | AILANG agent runs cost about 3× | Banked rows, 43 per language, Haiku 4.5: median `cost_usd` 0.419 vs 0.142, median `agent_turns` 23 vs 8, median `total_tokens` 1.77M vs 258k |
| V6 | The teaching prompt is about 99KB | `ailang prompt \| wc -c` = 99,175 (2026-10-28) |
| V7 | AILANG has working property tests with counterexamples | `ailang test` on a scratch module: `property "…" { forall(x: int) => … }` runs 100 cases; a false property reports `property failed on input: [0]`; `--seed` replays. Syntax as in `docs/docs/guides/testing.md` |
| V8 | `tool_aware` exists and has no recorded measurement | `internal/eval_harness/spec.go` `ResolveCondition` case `tool_aware` (contract guidance + `ai-check` step, `agent_prompt.go:461–464`), added in `7d689d87e` (2026-02-14). `grep -rl tool_aware design_docs changelogs docs` is empty; positive control: the same search finds `agent_prompt` in design docs (e.g. `motoko-harness-analysis-log.md`) |
| V9 | The Python grader is stdlib-only | `internal/eval_harness/python.go:92`: `uv run --python <pinned> -- python …` with no dependencies |
| V10 | H2 is already owned | `design_docs/planned/m-maintenance-eval-taskpack.md` (requested 2026-10-23): 2 families × 2 languages × 2 repeats × 3 changes pilot, "a neutral or negative AILANG result is a valid outcome"; harness code in `internal/eval_harness/maintenance_*.go` |
| V11 | Cost per verified success is implemented | `cmd/ailang/chains_stats_cvs.go` (`ailang chains stats --cost-per-verified-success --baseline <id>`); design `implemented/v1_0_0/m-cost-per-success-kpi.md` |
| V12 | Banked transcripts hold tool names only | A banked AILANG row's `agent_transcript` reads `[TURN 2]\n[TOOL] Bash` with no command text; CLAUDE.md §1 names `ailang chains chat` and the session JSONL as the instrument for tool results |
| V13 | The teaching prompt already teaches contracts and inline tests | `ailang prompt`: a `requires { … }` / `ensures { … }` section, and `tests [(0, 0), (5, 25)]` with `Run: ailang test file.ail`. The agent prompt lists "Testing: `ailang test`" (`agent_prompt.go:398`). Taught, still unused (V4) |
| V14 | No existing doc owns this scope | Scaffold search (SimHash + neural) plus manual reads: `m-oracle-adequacy` (evidence bundles per result, target v0.23.0, not run) and `m-eval-trust-signals` (external credibility: HumanEval port, receipts) are adjacent and referenced, not duplicated; neither defines a per-release language-advantage scoreboard or the verification-in-the-loop experiment |
| V15 | The Python grader's environment has no property-testing module | `uv run --python 3.12 -- python -I`: `sys.stdlib_module_names` (301 names) contains no property-testing or generator module (only `random`/`_random`); `importlib.util.find_spec('hypothesis')` is `None` |
| V16 | VM headroom on the `cp_60159` reference | Binary `57b0b8f20`, `ailang run --entry main --quiet --relax-modules --caps IO` on each of the 17 hidden tests: evaluator 17/17 correct, slowest 8.99s; with `--strict-bytecode` 17/17 correct, slowest 0.26s |
| V17 | Transcript instruments per harness | Claude: `~/.claude/projects/*/*.jsonl` tool_use inputs carry command text (V4 was parsed from them). Codex: `~/.codex/sessions/2026/10/28/rollout-*.jsonl`; 7 of today's carry the CP AILANG prompt, and one contains 8 `ailang run`/`ailang check` command lines. motoko: `ailang chains import-motoko` (`cmd/ailang/chains_import.go`) imports its run logs. opencode: `~/.local/share/opencode/opencode.db` exists on this machine, but `git grep opencode.db` finds **no in-repo reader** (only docs), so S5 for opencode needs a reader or reports n/a. pi: no instrument found |
| V18 | Stretch CP benchmarks run in post-release evals | `.claude/skills/post-release/SKILL.md:77`: `Tier scope: --tier core,stretch,frontier` |
| V19 | The MCP stdlib lookup exists | `mcp_tools/language.ail:47–48`: `@route("POST", "/api/mcp/stdlib_search")`, `@mcp_name("stdlib_search")` |
| V20 | What `eval-paired` reports | `internal/eval_analysis/paired.go`: `PairedResult` fields `on_pass`/`off_pass`, `delta_pp`, `only_on_passed`, `only_off_passed`, `unpaired`, `mcnemar`, `headroom`; `MinDiscordantForPValue = 10`; `exactBinomialMaxN = 25`; `McNemarResult{reportable, method, statistic, p_value, note}`. Success is `RunMetrics.Passed()` = `CompileOk && RuntimeOk && StdoutOk` (`internal/eval_harness/metrics.go:330`), and `StdoutOk` is false when any hidden test fails (M-EVAL-HIDDEN-TESTS) |
| V21 | `eval-elo` exists | `ailang eval-elo` usage: `ailang eval-elo <results_dir> [--json]`, "Fit per-language (AILANG vs Python) ELO over an eval results directory" |
| V22 | The CP pack's structure | `ls benchmarks/cp_*.yml`: 6 files; `tier:` 4 `stretch`, 2 `experimental`; `experimental` is in `ValidTiers` (`internal/eval_harness/spec.go`) |

## Quorum

Triggers #1 and #3 fired.

**Round 1 (2026-10-28T13:06Z, $0.18): BLOCKED**, 3 rejects (oc-kimi-k3, gemini-4-argon, oc-glm-5-3; gpt6-1-sol
unreachable), controller pass. Every objection was correct; all were closed in-session:
- *kimi, glm*: the VM timing in Phase 4 had no log row → measured, V16 (8.99s vs 0.26s, 17/17 both).
- *gemini, glm*: V15 was PENDING → verified, V15.
- *glm*: H1's "no built-in equivalent" was misframed against Phase 2's own Python control → H1 and the
  Why section now name the real asymmetry (integrated generators, replay, Z3 vs hand-written checks).
- *gemini*: S5 had no answer for harnesses without transcripts → per-harness instruments (V17), with
  "n/a (no instrument)" stated, never 0; D2 now says a banked field is the only way to cover pi.
- *glm*: three codebase claims had no row → V18 (post-release tiers), V19 (MCP `stdlib_search`); the
  "every tier-based eval" wording was narrowed to the post-release scope.

**Round 2 (2026-10-28T13:10Z, $0.17): BLOCKED**, 3 rejects (oc-kimi-k3, gemini-4-argon, oc-glm-5-3; gpt6-1-sol
unreachable), controller pass. All three named the same correct gap: the kill criteria relied on an
`eval-paired` "noise band" that does not exist. Read from the code (V20), the tool reports McNemar over
discordant pairs and withholds the p-value below 10 of them, which a single-tier CP cohort will rarely
reach. Fixed with a pre-registered [decision rule](#decision-rule-for-ab-arms) (pooled tiers, a doubling
step, an explicit "inconclusive at the full cohort" outcome) and D5. Also fixed: `eval-elo` (V21), pack
structure (V22), and V17 now cites code for motoko and states that opencode has no in-repo reader.

The re-review guardrail is spent (one re-run). **Handed to the human** with these fixes unreviewed.

## Related Documents

- [M-EVAL-HIDDEN-TESTS](m-eval-hidden-tests.md): the oracle and the CP pack this program measures with
- [M-VM-COVERAGE](m-vm-coverage-letrec-prelude-bridge.md), [M-PERSISTENT-COLLECTIONS](m-persistent-collections.md), [M-INT-OVERFLOW](m-int-overflow-trap-and-bigint.md): H4's fixes
- [M-MAINTENANCE-EVAL-TASKPACK](../m-maintenance-eval-taskpack.md) and [real-world study](../m-real-world-language-maintenance-study.md): own H2
- [Cost-per-verified-success KPI](../../implemented/v1_0_0/m-cost-per-success-kpi.md): S3
- [M-ORACLE-ADEQUACY](../v1_1_0/m-oracle-adequacy.md), [M-EVAL-TRUST-SIGNALS](../v1_1_0/m-eval-trust-signals.md): adjacent
- [M-EVAL-FINETUNING-DATA-PIPELINE](../v1_1_0/m-eval-finetuning-data-pipeline.md): Phase 5
- [M-CONTRACT-GUIDED-EVAL](../../implemented/v0_8_1/m-contract-guided-eval.md): the conditions mechanism Phase 2 reuses
- [PROGRAM.md](../../PROGRAM.md) §5c, §7
