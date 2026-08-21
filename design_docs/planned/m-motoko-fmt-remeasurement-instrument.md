# M-MOTOKO-FMT-REMEASUREMENT-INSTRUMENT: the instrument that decides whether `motoko_ext_fmt` survives the new tree

**Status**: **UN-PARKED 2026-08-19 (iteration 13) — `D-MOTOKO-FMT-1` RESOLVED by Mark (attended,
2026-08-19): the trace is a PRECONDITION of D1, and iteration 13 ran it. O4 is CLOSED by measurement
(§12), and D1 is RE-SHAPED by what the trace found — the preflight cannot be made provider-conditional
where it currently sits, because `HealthCheck` never sees the lane's model.** Design only; nothing here
runs an eval. Two quorum rounds spent (R1 BLOCKED, revision, R2 BLOCKED — both rounds with BOTH
external reviewers present, `absent_reviewers` empty in both artifacts). Three of the four
objections raised across the two rounds are ANSWERED and carried into the text; the fourth is a
real, bounded, one-question park. **The instrument's DIRECTION was never disputed by any reviewer
in either round** — not the censored paired win-rate, not the benchmark-set reasoning, not the
power arithmetic, not the pre-registered decision rule. See §11.
**Created**: 2026-08-17 (motoko mission iteration 8, queue item 6, charter clause 3: *"Every carried improvement is measured, and RE-measured when the tree moves"*)
**Owner**: motoko mission
**Estimated**: ≤3 days of sprint work (driver fix + analyzer), plus 2 already-reserved Wednesday A/B rig slots (~4.9 h wall-clock each) once the migration's preconditions are met
**Dependencies**: `m-motoko-dst-refactor-migration.md` V6 (fmt port) and V7 (profile restoration / queue item 7) must land first; this doc supplies the measurement those items are gated on

**The decision this doc enables:** after the migration to Arni's phase-core tree, does
`motoko_ext_fmt` stay in the extension catalog (KEEP) or get dropped (RETIRE)? The migration doc
already pre-commits to *"kept if it holds, dropped if not"*
(`design_docs/planned/m-motoko-dst-refactor-migration.md:67`); this doc defines *what "holds"
means*, on what data, at what cost, with the decision rule written before any data exists. DST
cannot make this call: extension-hook coverage is ≈1 substantively-simulated hook of ~40
(mission charter, DST-scope section) — *"it will not tell us whether fmt saves tokens"* — so this
is a rig instrument, priced as one.

---

## 1. Why "re-prove the −74%" is not the target

The −74% tokens-to-pass headline (AC5,
`design_docs/implemented/v0_32_0/m-fmt-dialect-alignment.md`, run 2026-07-31) is real and
correctly banked, but it is not a number an instrument can be built to re-prove. Five reasons,
all re-derived this session (Verification Log rows V1–V5, V19):

1. **n=1 per pair.** Its own author: *"Claim strength: direction, not proof … p≈0.11 one-sided;
   the total is dominated by one benchmark; same-model run variance is real."* A re-run landing
   anywhere from −40% to −85% is consistent with the original data.
2. **74.7% of the token saving is one benchmark.** Re-computed from the AC5 table:
   all six pairs give −74.2%; dropping `log_file_analyzer` gives −47.1%; that one benchmark
   contributes 3,125,933 of the 4,182,882 saved tokens. The headline is one pair wearing a
   suite's clothes.
3. **That pair's ON arm cannot be reproduced today.** In the nightly rotation lane
   (opencode-qwen3 models, arm `rag_on` — the only arm in
   `~/.ailang/state/nightly-eval-history.jsonl`), `log_file_analyzer` is 3/30 trials lifetime
   and 0/10 over the last five nights, with an open sustained-failure issue (#649).
   Tokens-to-pass is undefined when nothing passes.
4. **One of the six baseline pairs is formally void.** Settling V-E from the banked rows
   themselves (not the doc): 5 of 6 ON rows in `eval_results/ab2_fmt_on/` carry exactly one
   `status=formatted` fmt-hook event — fmt provably fired — but the `emit_exact_bytes_varied`
   ON row banked **zero events and was quarantined** by the harness
   (`validity: treatment_unproven`), and its −65% pair is nevertheless summed into the
   published −74% total. All 6 OFF rows banked zero events (control clean). So the baseline's
   treatment integrity was mechanically enforced per row (the gate landed 2026-07-29, two days
   before the run) — but the *analysis* summed a row its own machinery had voided, and the AC5
   writeup never mentions the sink at all (0 occurrences of `fmt_hook_events` in the doc).
5. **The run was order-confounded by construction: every ON row completed before the first OFF
   row started.** Sorting the twelve banked AC5 rows by their in-file `timestamp` (V19) gives a
   perfect arm block — six ON rows 16:42–17:00, then six OFF rows 17:01–17:36 (+02:00; V4's
   14:42–15:36 are the same instants in UTC). Model warming, rig load, or any temporal drift
   over that hour is fully aliased with the treatment, so the sign-test null
   P(ON wins | non-tied) = 0.5 was never verified for the run that produced the headline. Worse
   for "pairing by trial index": the within-arm benchmark ORDER differs between the two arms,
   so index pairing did not even pair the same position in time. This is a measured property of
   the baseline, not a hypothetical risk — and a fifth independent reason it cannot simply be
   "re-proven": the −74% was produced under an execution schedule this instrument (§5.3) voids.

**Restated target:** not "reproduce −74%", but *"on the new tree, does the fmt extension deliver
a benefit large enough to justify carrying it — measured by an estimand that is defined even
when arms fail, on a benchmark set with headroom, with treatment integrity enforced at both
banking and analysis?"*

A fifth finding changes the cost picture: **the instrument's driver is currently broken.** The
Wednesday fmt A/B lane fired on 2026-08-05 and 2026-08-12 and both times died in seconds at the
motoko canary pre-flight (`OPENROUTER_API_KEY not set — motoko routes ALL models via
OpenRouter`, `internal/executor/motoko/healthcheck.go:65`, an unconditional env check), banking
nothing. The lane has produced zero data since AC5. Any re-measurement design that does not fix
this first is designing a run that will not happen.

---

## 2. The target metric

### Candidates, and what each concludes under the null (no effect)

| candidate | defined when an arm fails? | null expressible? | fatal flaw |
|---|---|---|---|
| tokens-to-pass, passing pairs only | **no** — pair silently dropped | no clean null: conditioning on pass (a post-treatment variable) can manufacture phantom savings when pass rates differ by chance | selection bias; the shrinking self-selected set the mission keeps closing |
| tokens-per-attempt, any outcome | yes | yes — expected paired log-ratio 0 | rewards fast failure: an arm that dies at step 1 "wins" the pair |
| **censored pair verdict (chosen)** | **yes** | **yes — P(ON wins \| non-tied) = 0.5** | ties (both-fail, both-pass-within-margin) shrink effective n; handled by an n_eff floor |
| pass-rate primary, tokens secondary | yes | yes — McNemar null | measured saturation: on the ELO-selected set the control arm hit 24/24 (100%), the headroom warning fired, 0 discordant pairs — structurally uninformative for this subject |

### Chosen primary: paired censored win-rate

Treat tokens-to-pass as right-censored at the run's token cap
(`MAX_TOKENS_PER_BENCH=4,000,000`, `tools/launchd/nightly-eval.sh:150`): a non-pass is an
observation "> cap". Within each (benchmark, trial) pair:

1. one arm passes, the other does not → the passing arm **wins** (a pass at ≤ cap beats a
   censored > cap, at any token count);
2. both pass → fewer total tokens (input cache-inclusive + output, as banked) **wins**, with a
   practical-equivalence margin: |log ratio| ≤ 0.10 (±10%) is a **tie**;
3. both fail → **tie** (two censored observations are incomparable).

Primary test: one-sided exact sign test on non-tied pairs, α = 0.05, H₁ = ON wins more often.
The null is exactly P(win) = 0.5 and is defined regardless of pass rates — if nothing passes,
every pair is a tie and the instrument reports "no evidence" loudly (n_eff floor, §7) instead
of comparing a vacuous subset.

Secondaries (reported, never deciding alone): (a) mean paired log token ratio on both-pass
pairs with a 95% CI — the magnitude estimate the win-rate deliberately doesn't carry; (b)
pass-rate delta via the existing `ailang eval-paired` (`cmd/ailang/eval_paired.go`) with its
McNemar + headroom warnings, as a guardrail that fmt does not *reduce* passes.

Pairing note: pairing by (benchmark, trial-index) controls benchmark difficulty — the dominant
variance source — but trial indices themselves are arbitrary labels over independent samples.
That is the same convention `eval-paired` already uses; it is stated here so nobody reads
per-trial pairing as a seed-level guarantee.

Validity precondition on that null: P(ON wins | non-tied) = 0.5 also requires that neither arm
systematically runs earlier than the other. Under the whole-arm blocking the baseline actually
executed (§1.5, V19), time-of-run is aliased with treatment and the null is unverifiable.
Counterbalanced arm execution is therefore a REQUIREMENT of this instrument, not an assumption:
§5.3 specifies the schedule, how the executed order is recorded in the banked rows, and the
gate that VOIDs any slot banked in a different order.

## 3. The benchmark set

**Criteria** (all three required): (1) headroom for the *current* subject on the *new* tree —
E[pass] in 0.20–0.70 by ELO expected score, nearest 0.50 first, exactly the runtime confidence
selection the Wednesday lane already implements (`select_ab_benchmarks`,
`tools/launchd/nightly-eval.sh:225`) after two documented set-staleness burns; (2) not under an
open flake/sustained-failure investigation; (3) selected at run time against fresh ratings,
never frozen in this doc — a hardcoded list goes stale the moment the subject's rating moves.

**The original six, against those criteria** (pass rates re-derived this session; scope: the
nightly rotation lane, opencode-qwen3 models, 23 nights 2026-07-24..2026-08-17 — the only lane
with nightly history; the motoko lane has banked nothing since AC5 because the driver is
broken):

| benchmark | lifetime | last 5 nights | verdict |
|---|---|---|---|
| log_file_analyzer | 3/30 (10.0%) | 0/10 | **OUT** — below band; open issue #649 |
| emit_exact_bytes_varied | 10/30 (33.3%) | 2/10 | eligible if it rates into band |
| numeric_modulo | 36/46 (78.3%) | 10/10 | likely saturated; selection decides |
| immutable_data_structures | 37/46 (80.4%) | 8/10 | likely saturated; selection decides |
| decision_block_capture | 18/30 (60.0%) | 6/10 | eligible |
| binary_tree_sum | 26/46 (56.5%) | 7/10 | eligible |

**`log_file_analyzer` is out**, and the consequence is stated plainly: **numeric continuity
with −74% is abandoned, deliberately.** You cannot drop the benchmark that carries 74.7% of the
saving and still claim to be re-measuring the same number — and keeping an unpassable,
under-investigation benchmark to preserve a headline is a design defect, not an inherited
constraint (its rows would be all-tie and pure cost). The instrument answers the decision
question ("does fmt earn its place on the new tree"), not the continuity question ("is −74%
reproducible") — the latter was answered in §1: it was never a suite-level fact.

## 4. Power and sample size, priced

**Noise floor, measured not assumed.** The only paired replicate token data for this subject on
this rig is the 2026-07-30 pre-dialect-fix write-mode A/B
(`eval_results/fmt_ab/tokens_20260730`: 8 benchmarks × 3 trials × 2 arms, 24 matched pairs —
runs 15:36–19:31, before the fix `ca3d04cd8` landed 2026-07-31 11:31, so its *direction* [+42%
mean, fmt harmful] is pre-fix and unusable, but its *variance* is the honest noise estimate):
**sd of paired log token ratio = 1.04**. That is enormous — single pairs swing from −83% to
+1000% (log-ratios −1.80 to +2.41) on the same benchmark. One caveat cuts in the instrument's
favor: that run was itself executed as a perfect arm block — all 24 ON rows banked before the
first OFF row (V24), the same defect §1.5 measures in the AC5 baseline — so sd = 1.04 includes
whatever order/drift component the §5.3 counterbalanced schedule removes. As a noise estimate
for this instrument it is, if anything, conservative; the sample-size arithmetic below stands.

**Effect worth acting on.** The extension's claimed value is a halving-or-better; against sd
1.04 the power arithmetic (exact binomial for the sign test; normal approximation for the
paired t on log-ratio; computed this session, V12) is:

| n non-tied pairs | sign-test power, P(win)=0.70 | P(win)=0.75 | t-power, ratio 0.50 | ratio 0.60 | ratio 0.75 |
|---|---|---|---|---|---|
| 24 | 0.56 | 0.77 | 0.90 | 0.67 | 0.27 |
| 32 | 0.64 | 0.85 | 0.96 | 0.79 | 0.35 |
| 40 | 0.81 | 0.95 | 1.00 | 0.93 | 0.48 |

**We power for a large effect only, and say so.** Detecting a −25% saving at 80% would need
~100+ pairs ≈ ≥16 h of rig — more than this mission can afford, and beside the point: on this
noise floor, an effect the instrument cannot see at 40 pairs is too small to justify porting
and maintaining the extension. The cheapest instrument that supports a survive/retire decision
is therefore: **60 planned pairs (expecting ≥40 non-tied), giving ~81% power for a clear
benefit (P(win)=0.70) and ~93% for a token halving.** Wide error bars on magnitude are
accepted; the decision rule (§7) never needs the magnitude to be precise.

**Configuration:** two Wednesday slots of the existing lane config — 6 ELO-selected benchmarks
× 5 trials × 2 arms = 60 rows/slot — pooled across slots (the lane's design already pools
across weeks). 120 rows → 60 planned pairs.

## 5. Treatment integrity

The banking-side machinery exists and is verified working (V6–V8):

- **Arm resolution from reality, not the flag**: `ResolveFmtArm`
  (`internal/eval_harness/fmt_treatment.go`) derives the arm from the subject's step-0
  `resolved_extensions` broadcast (exact name match on `fmt#N`).
- **Positive control per row**: the extension appends one JSONL event per invocation to
  `<workspace>/.claude/fmt_hook_events.jsonl`; `ReadFmtHookSink` reads it unconditionally
  post-run. `AssertFmtTreatmentIntegrity` (wired at `cmd/ailang/eval_benchmark_agent.go:239`)
  quarantines an ON row with zero events *or* zero `status=formatted` events (firing is not
  delivering), and quarantines an OFF row with any event (contamination). This gate caught a
  real void row in the AC5 run itself.

What this design **adds** — because the AC5 defect was at analysis level, not banking level
(a quarantined row was summed into the headline):

1. **The analyzer refuses quarantined input.** A pair whose ON row is quarantined is dropped
   *and counted*; if > 20% of ON rows in a run are quarantined, or any OFF row shows
   contamination, the entire run is **VOID — no numbers are reported at all**, and the void
   reason is the output. Silence is not an option; a void run pages the mission via
   `ailang messages send`.
2. **New-tree contract checks, before the first slot** (acceptance criteria for the V6 port,
   verified by one smoke bank of a single benchmark): (a) the ported extension still writes
   `<workspace>/.claude/fmt_hook_events.jsonl` with the `{status,file}` schema; (b) the
   new-tree step-0 broadcast still names the extension `fmt` in `resolved_extensions`. Failure
   direction is loud either way — a renamed extension makes `ResolveFmtArm` label the ON arm
   "off", and its events then trip the contamination check — but discovering that inside a
   4.9 h slot wastes the slot; the smoke bank costs ~5 minutes.
3. **Counterbalanced arm execution, required and machine-verified (order integrity, §5.3).**
   Both prior fmt A/B runs on this rig executed as perfect arm blocks — every ON row before the
   first OFF row (AC5: V19; the 48-row noise-floor run: V24) — and the current Wednesday block
   can only produce that order: a `for arm in on off; do … done` loop wraps one whole-suite
   `eval-suite` call per arm (`tools/launchd/nightly-eval.sh:492-507` in this worktree, V22;
   V19 is the rig-side confirmation that this is what actually executes). Time and treatment
   are aliased by construction. The instrument requires instead:
   - **Schedule.** The driver iterates over the ELO-selected benchmarks in selection order; for
     benchmark index i (0-based) it runs both arms back-to-back — ON then OFF when i is even,
     OFF then ON when i is odd — each arm as one `eval-suite --benchmarks <b> --trials 5`
     invocation. Deterministic alternation is chosen over per-benchmark randomization: it is
     exactly balanced (lead-arm counts differ by ≤ 1), it needs no seed to be recorded, and it
     is verifiable from the banked rows alone. Residual stated honestly: within one benchmark
     the leading arm's 5-trial block still precedes the trailing arm's by minutes; alternating
     the lead makes any short-scale drift push the two arms symmetrically across the set
     instead of one way.
   - **Record.** No new banking field is needed. Every banked row already carries an ISO
     `timestamp` (V19/V24 reconstructed both prior runs' orders from exactly this field), and
     bank filenames embed the bank time (`<id>[_trialN]_<lang>_<model>_<timestamp>.json`,
     `internal/eval_harness/metrics.go`, V23), so repeated invocations into the same arm output
     dir do not collide. The executed order IS the banked timestamps, sorted.
   - **Gate (order integrity).** The D2 analyzer sorts a slot's rows by `timestamp` and
     requires: (a) rows form contiguous same-(benchmark, arm) blocks; (b) each benchmark's two
     blocks are adjacent; (c) the lead arm alternates across the benchmark sequence
     (|#ON-lead − #OFF-lead| ≤ 1). Any failure → the slot is **VOID, no numbers reported**
     (§7), exactly parallel to the treatment gate: a design that asks for counterbalancing but
     cannot check it after the fact has the same defect as a treatment gate nobody reads.
   - **Feasibility.** This schedule is NOT expressible through `tools/launchd/nightly-eval.sh`
     as it stands (V22) — restructuring the fmt block is a driver change, priced as **D1b** in
     §6 and carried in the same sprint as D1.

## 6. Cost model

Anchors measured from banked runs this session (V13), same subject, same rig:

- 48-row A/B (8 bench × 3 trials × 2 arms): 15:36:11 → 19:31:39 = **235.5 min ≈ 4.91 min/row**.
- 12-row A/B (6 bench × 1 trial × 2 arms): 53.5 min summed row durations (ON 17.8 + OFF 35.8),
  consistent with the elapsed 14:42 → 15:36. Failing rows dominate (one OFF row alone: 25.5 min).

| item | arithmetic | cost |
|---|---|---|
| D1: canary fix (`healthcheck.go:65` unconditional `OPENROUTER_API_KEY` check blocks local lanes) | small Go change + test; also un-breaks the *current* tree's Wednesday lane, so it pays for itself regardless of this instrument | ~0.5 day |
| D1b: counterbalanced Wednesday block (§5.3) | restructure the fmt block's whole-arm `for arm in on off` loop (`nightly-eval.sh:492-507`, V22) into the per-benchmark interleave with alternating lead arm; bash only, same sprint as D1 | ~0.25 day |
| D2: censored-pair analyzer | extend `eval-paired` (or sibling command) with the §2 verdict + §5 void rules (treatment AND order integrity) + §7 rule; Go, no new deps | 0.5–1 day |
| smoke bank (new-tree contract check, §5.2) | 1 benchmark × 1 trial ON | ~5 min rig |
| measurement slots | 2 × 60 rows × 4.91 min ≈ 2 × 294 min | **~9.8 h rig wall-clock** |
| GPU-hours | local qwen3.6 held loaded for the duration; both arms drive the same model, no extra VRAM | ≈ 9.8 GPU-h |
| metered dollars | both arms are local ollama lanes | **$0** |
| contingency (§7 third slot) | +294 min | +4.9 h rig, only if triggered |

The rig constraint is honored by construction: the slots are the **already-reserved Wednesday
A/B window** (`RUN_AB_FMT`, day gate `%u == 3`), inside the nightly job that already holds
`rig.lock` — currently producing nothing because of D1. Net-new rig cost over the status quo is
therefore ≈ 0; the cost is *reclaiming* a broken reserved slot, not claiming a new one.
Preconditions owned elsewhere: V6 fmt port + V7 profile restoration (migration/queue item 7),
and ELO ratings for the new-tree subject (the selection returns nothing without ratings and the
lane correctly skips — seed from the new tree's first rotation nights, as the skip message
already instructs).

**Deployment precondition (D1 + D1b), before the first measured slot:** merging the fixes to
`origin/dev` does NOT put them on the rig. The installed plist executes `nightly-eval.sh` in
place from V1's checkout (`~/dev/sunholo-data/ailang`, V20) — the standing stale-checkout
defect, open issue #558 (V21). The verification step is therefore shaped like this: read the
script at the exact path named in the installed plist's `ProgramArguments` (not at any path in
a working tree) and confirm the D1 canary fix and D1b interleave loop are present there. Until
that read passes, no Wednesday slot counts as a measurement attempt.

**Sequencing** (≤3–4 day sprint discipline): sprint = D1 + D1b + D2 + smoke-bank wiring. The
slots themselves fire on the lane's own schedule once V6/V7 land — this doc's sprint does not
wait on the migration, and the migration does not wait on the sprint.

## 7. The decision rule — pre-registered, before any data

Let n_eff = non-tied pairs after pooling both slots; W = ON wins among them.

- **VOID** (instrument failure, no verdict, fix and re-run): any OFF-arm contamination; > 20%
  of ON rows quarantined; the banked execution order fails the §5.3 order-integrity gate (the
  slot did not run the required counterbalanced schedule); ELO selection returned an empty set;
  or the run died pre-bank (D1 class). A void slot does not count against the slot budget.
- If n_eff < 24 after two slots → **one** third slot (pre-authorized here, +4.9 h). Still
  n_eff < 24 → **RETIRE**: an extension whose benefit cannot surface 24 decidable pairs in
  three reserved slots on a headroom-selected set has not carried clause 3's evidence burden.
  (It may be re-proposed later with new evidence; retirement is of the *port*, not the idea.)
- **KEEP** iff all three: (1) exact one-sided sign test rejects at α = 0.05 (e.g. W ≥ 26 at
  n_eff = 40, W ≥ 17 at n_eff = 24); (2) both-pass median token ratio ≤ 0.90 (direction
  sanity); (3) McNemar guardrail does not show a significant pass-rate *loss* for ON.
- **RETIRE** iff: the sign test rejects in the *opposite* direction at α = 0.05 (fmt harms —
  the pre-fix regime), **or** KEEP fails with n_eff ≥ 40.
- **INCONCLUSIVE** (KEEP fails, 24 ≤ n_eff < 40): one additional slot (the same pre-authorized
  third slot), then the same thresholds decide; if n_eff is still < 40, **RETIRE**. The
  asymmetry is intentional and stated: on a null, the extension is dropped. Clause 3 puts the
  burden of proof on the carried improvement, and `m-motoko-dst-refactor-migration.md:81`
  already records "propose upstream once re-measured" — an unmeasurable or null effect has
  nothing to propose.

Nothing in this section may be edited after the first non-void slot banks. (The order-integrity
VOID condition was added in the pre-data quorum revision pass; V10 establishes no slot has
banked since AC5, so the freeze had not engaged.)

## 8. Conflict surface

- `internal/executor/motoko/healthcheck.go:65` — D1 fix (eval-harness executor layer; Go).
- `cmd/ailang/eval_paired.go` / new analyzer command — D2 (eval-harness analysis layer).
- `tools/launchd/nightly-eval.sh` Wednesday block — modified by D1b (§5.3), no longer reused
  as-is. Deployment reality, measured (V20): the installed plist
  (`~/Library/LaunchAgents/dev.ailang.nightly-eval.plist`) names
  `/Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/nightly-eval.sh` in its
  `ProgramArguments`, so the script is executed **in place** — there is nothing to `cp` for a
  script edit, and an earlier draft of this bullet was wrong to say otherwise. The distinction
  that makes the trap durable: `.plist` files in `~/Library/LaunchAgents/` ARE installed copies
  and DO need re-installing when changed; the `.sh` they point at runs in place — the old
  bullet applied the plist rule to the script. The real constraint is different: that path is
  **V1's checkout** (`~/dev/sunholo-data/ailang`), not this mission's, so a fix merged to
  `origin/dev` reaches the rig only when that clone next pulls the file — the standing defect
  tracked as open issue #558 (V21). Hence §6's deployment precondition: verify the fix by
  reading the file at the plist-named path.
- `internal/eval_harness/models.yml:1880` (`motoko-local-qwen3-6-fmt`, profile `ollama_fmt`) —
  entries exist today; V7 (queue item 7) owns their new-tree equivalents. This doc consumes,
  does not define, them.
- `m-motoko-dst-refactor-migration.md` V6 — the fmt port (currently pinned `motoko_ext_fmt@0.4.2`
  in `mk-ast/ailang.toml`) is a precondition; §5.2's contract checks become V6 acceptance rows.
- The nightly rotation and `rig.lock` — untouched; the instrument lives entirely inside the
  existing Wednesday window.
- **Parser / types / codegen: not touched.** No AILANG core change anywhere in this design
  (mission default bias holds: everything here is harness lane or extension lane).

## 9. Verification Log

Every row: command run this session → observed result.

| # | claim | command | observed |
|---|---|---|---|
| V1 | −74% arithmetic & domination | `python3` over the six AC5 (ON, OFF) pairs | all six: ON=1,451,485 OFF=5,634,367 → −74.2%; drop log_file_analyzer → −47.1%; lfa saving 3,125,933 / 4,182,882 = 74.7% |
| V2 | rotation-lane pass rates, scope | `python3` over `~/.ailang/state/nightly-eval-history.jsonl` | 516 rows, 23 nights 2026-07-24..08-17; log_file_analyzer 3/30 lifetime, 0/2×5 last five nights; config_file_parser (control, known-present) 4/46 |
| V3 | history file covers ONLY the rotation lane, no fmt arms | same script: count arms/models | arms `{'rag_on': 516}`; models `{opencode-qwen3-5-…: 372, opencode-qwen3-6-…: 144}`; fmt rows: 0 (positive control: rag_on rows present) |
| V4 | AC5 rows live in `ab2_fmt_*`; tokens match the doc's table | `find`/`ls` + `python3` over `eval_results/ab2_fmt_{on,off}/agent/*.json` | 6+6 rows dated 2026-07-31T14:42–15:36; per-row tokens equal the AC5 table (e.g. lfa ON 262,670 / OFF 3,388,603) |
| V5 | V-E settled: 5/6 ON rows proven, 1 quarantined-but-summed, control clean | same script, fields `fmt_hook_events`/`validity` | 5 ON rows: 1 formatted event each; `emit_exact_bytes_varied` ON: 0 events, `validity={valid:False, reason:treatment_unproven}`; all 6 OFF rows: 0 events |
| V6 | integrity gate exists, predates AC5 | `git log --follow -- internal/eval_harness/fmt_treatment.go` | `fb84f50fc`/`ee36be967`/`b29d9b1a0`, all 2026-07-29 |
| V7 | gate is wired into banking | `grep -rn AssertFmtTreatmentIntegrity` (non-test) | one caller: `cmd/ailang/eval_benchmark_agent.go:239` |
| V8 | sink contract | `sed` `internal/eval_harness/fmt_hook_mode.go:160-240` | `FmtHookSinkPath` = `<ws>/.claude/fmt_hook_events.jsonl`; missing file → nil, not error; events parsed with `status` field |
| V9 | AC5 doc never mentions the sink | `grep -c fmt_hook_events` vs control `grep -c AC5` on the doc | 0 vs 3 |
| V10 | Wednesday lane broken since AC5 | `grep -n fmt /tmp/ailang-nightly-eval.log` | 2026-08-05 & 08-12 both arms: `canary pre-flight: OPENROUTER_API_KEY not set…`, then `directory not found: /tmp/nightly_eval_202608??_fmt_on`; `/tmp/nightly_eval_*` listing shows `_rag_*` dirs only (positive control) |
| V11 | canary check is unconditional | `sed` `internal/executor/motoko/healthcheck.go:50-70` | `if os.Getenv("OPENROUTER_API_KEY") == "" { return fmt.Errorf(…) }` with no lane/model condition |
| V12 | noise floor + power table | `python3` over `eval_results/fmt_ab/tokens_20260730/fmt_{on,off}/agent/*.json`; exact binomial + normal-approx script | 24 pairs, sd(log ratio)=1.041, mean +0.350, ON cheaper 10/24; power numbers as tabled in §4; run is pre-fix: files 15:36–19:31 Jul 30 vs fix `ca3d04cd8` dated 2026-07-31 11:31:56 (`git log -1`) |
| V13 | cost anchors | `head`/`tail` of that run's `run.log`; `python3` sum of `ab2` durations | 48 rows 15:36:11→19:31:39 = 235.5 min = 4.91 min/row; ab2 ON 17.8 + OFF 35.8 = 53.5 min / 12 rows |
| V14 | lane config | `grep`/`sed` `tools/launchd/nightly-eval.sh` | `RUN_AB_FMT` day gate `== 3`; `FMT_TRIALS` default 5; `select_ab_benchmarks` at line 225, default N=6; `MAX_TOKENS_PER_BENCH=4000000` (line 150); ON/OFF models as §8 |
| V15 | current wiring exists | `grep fmt internal/eval_harness/models.yml`; `grep fmt ~/dev/mk-ast/ailang.toml` | `motoko-local-qwen3-6-fmt` at models.yml:1880 (profile `ollama_fmt`); `"sunholo/motoko_ext_fmt" = "0.4.2"` pinned in mk-ast |
| V16 | saturation on ELO-selected set is real, headroom warning works | `tail run.log` of tokens_20260730 | control 24/24 (100.0%), warning text emitted, 0 discordant pairs, McNemar "not reportable" |
| V17 | DST cannot answer this; migration pre-commits to re-measure | `sed` mission doc DST-scope; `grep fmt m-motoko-dst-refactor-migration.md` | "≈1 of 40 covered hooks is substantively simulated"; migration line 67: "re-measured on the new tree — kept if it holds, dropped if not" |
| V18 | `eval-paired` exists | `grep -rln eval-paired cmd/ailang/` (non-test) | `cmd/ailang/eval_paired.go` |
| V19 | AC5 run executed as a perfect arm block; within-arm benchmark order differs between arms | `python3`: load all 12 `eval_results/ab2_fmt_{on,off}/agent/*.json`, sort by in-file `timestamp`, print (arm, benchmark, ts) | positions 1–6 all `on` (16:42:37→17:00:10 +02:00, order: binary_tree_sum, decision_block_capture, numeric_modulo, emit_exact_bytes_varied, immutable_data_structures, log_file_analyzer), positions 7–12 all `off` (17:01:28→17:36:49, order: numeric_modulo, decision_block_capture, immutable_data_structures, binary_tree_sum, emit_exact_bytes_varied, log_file_analyzer); zero interleaving; +02:00 local — V4's 14:42–15:36 are the same instants in UTC |
| V20 | installed plist executes `nightly-eval.sh` in place, from V1's checkout | `/usr/libexec/PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist` | `Array { /bin/bash /Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/nightly-eval.sh }` — a repo path, not a `~/Library` copy |
| V21 | stale-checkout deployment defect already tracked, open | `gh issue view 558 --json number,title,state` | `558 [OPEN] launchd drivers execute from the stale main checkout — #556's qwen3.5 retirement never reached the rig` |
| V22 | current Wednesday fmt block runs whole-arm blocks; §5.3 interleave not expressible as-is | `grep -n 'RUN_AB_FMT\|fmt_on\|fmt_off\|FMT_TRIALS\|select_ab_benchmarks' tools/launchd/nightly-eval.sh`; read of lines 479–523 (this worktree's copy — rig-side behavior confirmed independently by V19) | lines 492–507: `for arm in on off; do … "$BIN" eval-suite --agent --models "$m" --benchmarks "$FMT_BENCH_LIST" … --trials "$FMT_TRIALS" … done` — one whole-suite call per arm, no per-benchmark loop |
| V23 | banked filenames embed bank time (no collision across repeated invocations); rows carry sortable `timestamp` | `sed -n 285,320p internal/eval_harness/metrics.go`; `ls eval_results/ab2_fmt_on/agent/`; row field shown in V19's output | comment + code: `Generate filename: <id>[_trialN]_<lang>_<model>_<timestamp>.json`; listing shows e.g. `binary_tree_sum_ailang_motoko-local-qwen3-6-fmt_1785508957.json`; every row parsed in V19/V24 had a non-null ISO `timestamp` |
| V24 | the §4 noise-floor run (48 rows) was also arm-blocked, so sd=1.04 includes any order/drift component | `python3`: load all 48 `eval_results/fmt_ab/tokens_20260730/fmt_{on,off}/agent/*.json`, sort by `timestamp`, print arm sequence | `NNNNNNNNNNNNNNNNNNNNNNNNFFFFFFFFFFFFFFFFFFFFFFFF` (24 on then 24 off), first 2026-07-30T15:54:39+02:00, last 19:31:39 |

## 10. Open questions for the human

1. §7 retires on a conclusive **null**, not only on harm (burden of proof on the extension) —
   confirm? *(yes / no)*
2. A third Wednesday slot (+4.9 h rig) is pre-authorized when n_eff < 24 or the result is
   inconclusive — confirm? *(yes / no)*
3. Numeric continuity with the −74% headline is deliberately abandoned (`log_file_analyzer`
   excluded while #649 is open) — accept? *(yes / no)*

---

## 11. Quorum verification log (iteration 8) — what is settled, and the one thing that is not

Two rounds, both with BOTH external reviewers present (`gpt5-6-sol`, `gemini-3-1-pro`);
`absent_reviewers` is empty in both artifacts, so neither `proceed`/`blocked` verdict is an N−1
degrade. Metered total **$0.1424** ($0.0619 R1 + $0.0805 R2).

Artifacts:
- R1 `.ailang/state/mission-quorum/m-motoko-fmt-remeasurement-instrument-2026-08-17T05-26-49Z.json`
- R2 `.ailang/state/mission-quorum/m-motoko-fmt-remeasurement-instrument-2026-08-17T05-38-56Z.json`

Every objection below was classified *premise* vs *design* and every premise was **measured by the
controller** before routing, per the mission-control rule that a reviewer's objection is a claim too.

| # | round | reviewer | objection | controller measurement | disposition |
|---|---|---|---|---|---|
| O1 | R1 | `gpt5-6-sol` | the sign test is invalid without randomized/counterbalanced arm execution; "*if* the lane runs all ON before all OFF", order effects fake a directional win rate | **UPHELD, and the "if" is fact.** All 12 banked AC5 rows sorted by their in-file `timestamp`: ON 16:42:37→17:00:10, then OFF 17:01:28→17:36:49. A perfect block, zero interleaving; within-arm benchmark order also differs between arms, so "pair by trial index" pairs different positions | **ANSWERED** — §5.3 now *requires* per-benchmark counterbalanced execution, records the order, and VOIDs a slot banked otherwise (V19) |
| O2 | R1 | `gemini-3-1-pro` | §8's claim that `nightly-eval.sh` must be `cp`-installed to `~/Library/LaunchAgents/` is an unverified premise; launchd loads `.plist`, not `.sh` | **UPHELD.** `PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist` → `{/bin/bash, /Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/nightly-eval.sh}`. The script executes **in place, from V1's checkout** | **ANSWERED** — false instruction removed; replaced with the real constraint (a merged fix reaches the rig only when V1's clone pulls — open issue `#558`) and a precondition to verify at the plist's own path (V20/V21) |
| O3 | R2 | `gemini-3-1-pro` | the load-bearing "`#649` open sustained-failure" claim justifying the `log_file_analyzer` exclusion is itself unverified in the Verification Log | **UPHELD procedurally, and the claim is TRUE.** `gh issue view 649` → `#649 OPEN — [nightly-eval] Nightly sustained failure: log_file_analyzer`, created `2026-08-11T02:57:37Z`. Known-positive control in the same sweep: `gh issue view 721` → `MERGED`, so the instrument distinguishes states | **ANSWERED by measurement** — the row is owed in the Verification Log; the exclusion itself stands |
| O4 | R2 | `gpt5-6-sol` | D1 rests on an unverified routing premise: the doc bypasses the unconditional `OPENROUTER_API_KEY` preflight because both fmt arms are *allegedly* local Ollama lanes, but V15 verified a models.yml **profile name**, not the **resolved runtime provider**; removing the preflight could delete a required fail-fast or admit a silent provider fallback | **PARTIALLY measured — this is the park.** Established: the preflight is unconditional (`internal/executor/motoko/healthcheck.go:64`, no lane/model condition, reached via `MotokoExecutor.HealthCheck` → `runHealthCheck`); both arms declare `provider: "ollama"`, `env_var: ""` ("No API key — local inference"), `agent_model_name: "ollama/qwen3.6:35b-a3b-mxfp8"` (models.yml:1854, :1880); and the preflight empirically **did** block both Wednesday slots (V10). **NOT established: whether removing it would let motoko silently resolve to OpenRouter at runtime.** That trace needs the `mk-ast` fork's own resolution path and/or a live motoko run holding `rig.lock` — neither is a text fix, and choosing between "make the trace a D1 precondition" and "redesign D1" is controller judgment, not a verbatim reviewer fix | **CLOSED 2026-08-19 by measurement — see §12.** Answer: **both halves are true, of different lanes**, so the remedy is a CONDITION, never a deletion; and the condition is not expressible at the current call site |

**Why this parks rather than taking the narrow-refinement carve-out.** The carve-out permits a
bounded second revision only when *every* remaining objection carries a concrete reviewer-authored
fix AND needs no controller judgment to resolve. O3 clears that bar comfortably. **O4 does not** —
its remedy is an investigation, not an edit. Forcing it through would be exactly the vacuous pass
this mission keeps closing: shipping a prerequisite whose safety nobody established, in a doc whose
whole subject is that the previous measurement's integrity was never checked.

**The human decision is one word** (see also §10's three questions):

> **D-MOTOKO-FMT-1** — Is tracing motoko's runtime provider resolution a **precondition of D1**
> (the sprint traces it, then changes the preflight), or does D1 need a **redesign** that leaves the
> preflight alone and reaches the local lanes another way? *(precondition / redesign)*

Nothing else in this document is blocked on that answer.

---

## 12. The provider-resolution trace (iteration 13, 2026-08-19) — O4 answered, and D1 re-shaped

`D-MOTOKO-FMT-1` was resolved **precondition** (Mark, attended 2026-08-19): *"the sprint TRACES
motoko's resolved runtime provider first, then changes the preflight. Do not redesign around the
unknown."* The ruling names the instrument — *"the `mk-ast` fork's own resolution path **and/or** a
live motoko run holding `rig.lock`"* — and requires it to **discriminate**: does removing the
unconditional `OPENROUTER_API_KEY` refusal at `internal/executor/motoko/healthcheck.go:64` delete a
real fail-fast, or admit a silent OpenRouter fallback for entries declaring `provider: "ollama"`,
`env_var: ""`?

This section reports the **fork-resolution-path** arm, run first-party. No GPU was taken and no
`rig.lock` was held; §12.4 states exactly what the static arm cannot settle and moves it into the
sprint's acceptance criteria rather than claiming it.

### 12.1 The answer: BOTH halves are true, of DIFFERENT lanes

**(a) For the two fmt arms, no OpenRouter routing is reachable, so the preflight is a FALSE
fail-fast.** The resolved model is `ollama/…` at every tier that can fire, and the AILANG runtime
resolves that to the local provider *before* the generic `vendor/model → OpenRouter` rule:

| tier | source | value for the fmt lanes | `GuessProvider` (measured) |
|---|---|---|---|
| 1 | `process.env.MODEL` — always set by the Go executor (`internal/executor/motoko/motoko.go:343`) from `agent_model_name` (`internal/eval_harness/models.go:493-494`) | `ollama/qwen3.6:35b-a3b-mxfp8` | `ollama`, env var `""` |
| 2 | `profileAgent.model` — `.motoko/config/ollama_fmt/config.json` and `.motoko/config/ollama/config.json` | `ollama/qwen3.5:35b-a3b-mxfp8` | `ollama`, env var `""` |
| 3 | hardcoded default, `src/tui/src/index.ts:771` | `anthropic/claude-sonnet-4-6` | **`openrouter`**, env var `OPENROUTER_API_KEY` |

Precedence is `process.env.MODEL ?? profileAgent.model ?? "anthropic/claude-sonnet-4-6"`
(`mk-ast/src/tui/src/index.ts:768-771`, read first-party — note the `models.yml:1854` comment citing
`index.ts:580` for this is **stale**, that line is now event-printing code). Both profiles also pin
`agent.openai_base_url = "http://localhost:11434/v1"`, so the endpoint is local independently of the
model string. Motoko's own preflight asks for **nothing**: `required_secret_for_model`
(`mk-ast/src/core/supervisor.ail:21-26`) matches only the `anthropic/`, `openrouter/`, `openai/` and
`google/` prefixes and returns `""` otherwise, so `validate_secrets` returns `[]` for `ollama/…`.

**(b) For the OpenRouter-routed motoko lanes the preflight IS a real fail-fast — and the only one.**
Motoko's `validate_secrets` result is passed to `emit_warnings`, which prints
`{"type":"warning",…}` and **proceeds** to `run_with_config` (`supervisor.ail:11-19`, `:42-51`). So
motoko warns and runs; it never refuses. Deleting the Go preflight outright would let an
OpenRouter-lane eval start and burn its wall-clock before failing at the provider. **Hence a
condition, not a deletion** — which is what O4 was really asking.

### 12.2 The blocker the trace found, which changes D1's shape and cost

**A provider-conditional preflight is not expressible where the check currently sits.**
`HealthCheck(ctx context.Context) error` takes no task — it is the shared `executor.Executor`
interface method (`internal/executor/executor.go:31`) — so the only model it can read is `e.model`.
And `e.model` is **never** set from `models.yml`: `cfg.MotokoModel` has exactly one assignment in
non-test Go, the hardcoded `"openrouter/anthropic/claude-haiku-4-5"` at
`internal/executor/factory.go:71`. The lane's real model arrives later, per task, as `task.Model`
(`cmd/ailang/eval_benchmark_agent.go:174,195,253,389`), consumed by `getModel(task)`
(`motoko.go:610-620`) at Execute time — after the health check has already refused.

So an `if` added at `healthcheck.go:64` would evaluate `openrouter/anthropic/claude-haiku-4-5` for
**every** lane, conclude "OpenRouter", and refuse the ollama arms exactly as today. The sprint has
three options, none of them a one-line change, and D1's estimate should absorb whichever it takes:

1. **Set `cfg.MotokoModel` from `models.yml` at construction.** Smallest true fix, no interface
   change, and it also corrects `e.model` being wrong for every motoko lane today.
2. **Move the key check into `Execute`,** where `task.Model` exists. No interface change; loses
   fail-fast-before-workspace-setup.
3. **Widen the `HealthCheck` signature.** Cross-executor interface change (6+ implementors) —
   most expensive, and the trace gives no reason to prefer it.

**Express the condition on the RESOLVED PROVIDER, not on the string `OPENROUTER_API_KEY`**:
`ai.EnvVarForProvider(ai.GuessProvider(model))` returns the required variable *or `""`* for every
provider (`internal/ai/config.go:104-119`), so one check covers ollama, OpenAI, Anthropic and Google
instead of hardcoding one vendor. Note `internal/executor/motoko` does not import `internal/ai`
today — confirm the boundary allows it before assuming option (1) is free.

### 12.3 Ordering note, free from the same read

The preflight at `healthcheck.go:64` returns **before** the `motoko --version` query that discovers
`e.motokoRepo` (`healthcheck.go:70-77`), and `MOTOKO_REPO` is what stops motoko's profile silently
degrading to `extensions.order=[]` — the defect `motoko.go:344-364` records as *39 of 39 eval
sessions with `loaded_extensions=[]`*. Moot while the check refuses outright; load-bearing the moment
it becomes conditional, because a degraded profile drops the `fmt` extension that IS the treatment.
Whichever option D1 takes, the resolved-provider check must not run ahead of repo discovery.

### 12.4 What this arm does NOT settle — moved to acceptance, not claimed

The static arm proves no OpenRouter routing is **reachable** on the wired path. It does not prove no
OpenRouter connection is **made** at runtime, and the live arm is circular today: the fmt lane cannot
run while the preflight refuses it. So the live confirmation becomes a **D1 acceptance criterion**,
not a precondition:

> **AC-D1-live.** With the fix in place, one fmt-lane run reaches `localhost:11434` and makes **zero**
> connections to `openrouter.ai`. Assert on the connection, not on the absence of an error — an
> absence is satisfied equally by "no OpenRouter call" and "the run never started" (the observable
> must be unique to the mechanism). Pair it with a known-positive control: an OpenRouter-lane run in
> the same sweep must show the connection, or the instrument proves nothing.

### 12.5 Verification rows

| # | claim | command | observed |
|---|---|---|---|
| V25 | `GuessProvider` routes every tier as tabled in §12.1 | throwaway `go test ./internal/ai/ -run TestZZTraceIter13ProviderResolution -v` over the five literal model strings; file removed after, `git status --porcelain` empty | `ollama/qwen3.6:35b-a3b-mxfp8`→`ollama`/`""`; `ollama/qwen3.5:35b-a3b-mxfp8`→`ollama`/`""`; `anthropic/claude-sonnet-4-6`→`openrouter`/`OPENROUTER_API_KEY`; `openrouter/auto`→`openrouter`/`OPENROUTER_API_KEY`; `openrouter/anthropic/claude-haiku-4-5`→`openrouter`/`OPENROUTER_API_KEY`. **Both arms present, so the ollama reading discriminates rather than being vacuous** |
| V26 | ollama prefix is matched BEFORE the generic vendor/model→OpenRouter rule, by design | `sed -n '45,73p' internal/ai/config.go` | `if strings.HasPrefix(lower,"ollama:")||HasPrefix(lower,"ollama/") { return ProviderOllama }` at `:55-57`, above the `strings.Contains(lower,"/")` OpenRouter block at `:67-73`, with a comment stating exactly this ordering requirement |
| V27 | motoko's own secret check ignores `ollama/`, and is a WARNING not a refusal | read `mk-ast/src/core/supervisor.ail:11-51` | `required_secret_for_model` matches only `anthropic/`, `openrouter/`, `openai/`, `google/` → `""` for `ollama/…`; result flows to `emit_warnings` (prints `{"type":"warning"}`) then `main` continues to `run_with_config` |
| V28 | model precedence in the fork; the `index.ts:580` citation in `models.yml` is stale | `sed -n '760,780p' mk-ast/src/tui/src/index.ts`; `sed -n '1,60p' src/tui/src/config.ts` | `const model = process.env.MODEL ?? profileAgent.model ?? "anthropic/claude-sonnet-4-6"` at `:768-771`; `config.ts:23` maps `agent.model`→`MODEL`. Line 580 is `native_tool_results` event printing — the models.yml comment is a stale transcription |
| V29 | both ollama profiles pin an `ollama/` model AND a local base URL | `cat .motoko/config/ollama_fmt/config.json .motoko/config/ollama/config.json` | both: `agent.model = "ollama/qwen3.5:35b-a3b-mxfp8"`, `agent.openai_base_url = "http://localhost:11434/v1"`; `ollama_fmt` adds `fmt` to `extensions.order` (the treatment) |
| V30 | `MODEL` is always non-empty on the eval path, so tier 3 is unreachable there | `sed -n '340,350p' internal/executor/motoko/motoko.go`; `sed -n '490,500p' internal/eval_harness/models.go` | `env = append(env, "MODEL="+e.getModel(task))` unconditionally; `modelName = *model.AgentModelName` when non-empty, else `model.APIName` |
| V31 | `cfg.MotokoModel` is never set from `models.yml` — the health check cannot see the lane's model | `grep -rn 'MotokoModel' --include='*.go' . \| grep -v _test.go` (control: same grep for `MotokoProfile`) | **3** hits — a field decl, the hardcoded `factory.go:71` default, and `motoko.go:145` reading it. Control `MotokoProfile` → **11** hits, so the instrument finds wiring where it exists. `HealthCheck(ctx)` takes no task (`internal/executor/executor.go:31`); `task.Model` is set at `cmd/ailang/eval_benchmark_agent.go:174,195,253,389` |
| V32 | no key-absence-driven provider fallback exists anywhere in non-test Go | `grep -rn 'OPENROUTER_API_KEY' --include='*.go' . \| grep -v _test.go` (control: same for `OPENAI_API_KEY`) | **4** functional sites: `cmd/ailang/ai_handlers.go:276`, `cmd/ailang/exec.go:475` (both explicit `openrouter` subcommands), `internal/ai/config.go:115,143` (keyed by an ALREADY-resolved provider), and the motoko preflight at `healthcheck.go:64`. None re-routes on absence. Control: **23** `OPENAI_API_KEY` sites, so the grep sees this class of hit |
| V33 | D1b's production driver and D2's analyzer agree on the counterbalanced schedule, **and the analyzer is the discriminator** | `/bin/bash tools/launchd/test_fmt_ab_schedule.sh`; `go test -count=1 ./internal/eval_analysis/ -run TestFmtDriverScheduleSatisfiesOrderIntegrity` | the shell artifact contains 12 single-benchmark invocations in `b0:on,b0:off … b5:off,b5:on` order; the Go test consumes that artifact in **emit-only** mode and `CheckFmtOrderIntegrity` returns `""`. **Discrimination proven, not assumed:** under the ON-always-leads mutant the Go test dies at `CheckFmtOrderIntegrity rejected shell artifact: order_integrity_lead_not_alternating`. The first form of this row ran the shell harness in full-assert mode, where the same mutant killed the process at the fixture step (`shell schedule fixture failed`) **before the analyzer was ever called** — a decorative assertion for exactly the defect AC-M3-4 cites it to catch (rule 3i: which write does this read?) |
| V35 | `go test` caching is UNSOUND for a test whose observable is produced by a subprocess reading files outside the package — and the stale verdict reads as a passing mutation drill | mutate `nightly-eval.sh` to ON-always-leads, then `go test ./internal/eval_analysis/ -run TestFmtDriverScheduleSatisfiesOrderIntegrity` with and without `-count=1` | without `-count=1`: **rc=0** (cache hit, pre-mutation verdict re-printed); with `-count=1`: **rc=1**, `order_integrity_lead_not_alternating`. Go keys the test cache on files the TEST PROCESS opens and cannot see a `/bin/bash` child's opens. Fixed structurally rather than documented: the test now `os.ReadFile`s both driver scripts, which puts them in the cache key — re-measured after the fix, the same mutant reds **without** `-count=1` and the restored tree greens again |
