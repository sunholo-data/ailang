# Motoko Mission — iteration log (append-only)

One entry per mission-control iteration, newest LAST (append). Fixed template — keep every
section, write "none" rather than omitting. Same template as
[v1-mission-log.md](v1-mission-log.md); do not diverge it, so cross-mission comparisons parse.

```markdown
## N — YYYY-MM-DD — <headline>
**Picked**: <backlog item + why it was top>
**Reality check**: <what git/code verification of the doc's status found>
**Shipped**: <commits/branches/PRs, evaluator result + score, or "parked: reason">
**Routing evidence**: model=<m> task-class=<design|plan|execute|evaluate|mechanical>
  round1-score=<n> rounds=<n> corrections=<n>
  provider=<p> agent=<a> cost=<$<n>|quota-bucket:weekly-fable|quota-bucket:weekly-opus|unknown>
**Ruled out**: <hypotheses/approaches refuted this iteration — the anti-re-chase ledger>
**Retro lane**: <skill-fix: file+change | process-fix: change | backlog: new doc | none>
**Next**: <what iteration N+1 should pick up>
```

**Mission-specific note on the "Ruled out" ledger.** This mission's history is dense with
hypotheses that felt right and were wrong — "switch to qwen3.6", "it's a model wall", "the docx
loop is one bug". Every motoko conclusion so far has bottomed out in a harness bug. Write the
refutation down with its evidence; the ledger is the point.

**An idle iteration is a valid entry.** While the epic's Phase-0 gate is closed (see the charter's
Guardrails), exhausting the unblocked queue is a correct outcome. Record it as a real entry with
`**Shipped**: parked: Phase-0 gate closed, unblocked queue empty` rather than pulling gated work
forward to look productive.

---

## 0 — 2026-08-12 — charter RATIFIED; quorum blocked twice, all four objections true; the V21 driver defect fixed

**Picked**: Iteration 0 — ratify the rewritten charter. Not a sprint by construction (bootstrap
guide step 6); the deliverable is an agreed bar + queue, run attended with Mark.

**Reality check**: the 2026-06-24 charter was 7 weeks stale — 3 of its 4 stated goals solved or
superseded upstream (canonical prompt delivery solved; context-length superseded by Arni's affine
calibration; empty-response retry superseded by `motoko-ext-empty-stop-guard`). Rewritten rather than
edited; the original is preserved byte-identical in `motoko-mission-status-archive.md`. Bootstrap
verified end-to-end: separate checkout, namespaced state, distinct pidfile (dry-run), kill switch
armed BEFORE `launchctl bootstrap`, `RunAtLoad` fire confirmed to exit at Gate 0 with rc=0 and zero
tokens.

**Shipped**: charter + log + archive + bookkeeping issue #663 (`bc2fc3c2d`, `98ffaf5cf`); clause 6
(executor-fleet graduation) added on Mark's direction; Premise Verification Log V1-V21 added under
quorum pressure. **Bar + queue RATIFIED by Mark**, who also routed the V21 driver fix here — landed in
`tools/launchd/mission-control.sh` (lane-degradation ledger + one emit after every early exit).

**Routing evidence**: model=gpt5-6-sol,gemini-3-1-pro task-class=design-review
  round1-score=blocked rounds=2 corrections=4
  provider=openrouter agent=ailang-design-quorum cost=**$0.115** (round 1 $0.0535, round 2 $0.0613)
  controller=claude-opus-5 (in-session, attended) cost=quota-bucket:weekly-opus

**Ruled out**:
- *"A charter is exempt from premise verification because it is direction, not design."* **Refuted**
  by `gpt5-6-sol` round 1. A charter's claims gate isolation, routing and queue order — they are
  more safety-bearing than a design doc's, not less. V1-V21 now exist.
- *"Inheriting V1's `go-compiler` verification is sound because it is the same repo."* **Refuted** by
  `gemini-3-1-pro` round 2, correctly: `bin/ailang` is a **per-working-tree** artifact and this is a
  separate clone. Running it first-party (V19) then surfaced **V20**, which no reviewer could have
  predicted — `make quick-install` writes the SHARED `~/go/bin/ailang` that V1 and the eval rig
  resolve through. A separate checkout isolates the working tree, **not the installed toolchain**.
  New guardrail written.
- *"Requiring a loud fallback signal of the future `motoko:` lane is sufficient."* **Refuted** by
  `gpt5-6-sol` round 2. The mission runs TODAY on `codex→pi→opus` with no such guarantee, in a
  charter that cites World losing five iterations to precisely that. Measured as V21 and **confirmed**:
  demotion is `log`ged at driver lines 360/392 and posted to the issue at none of its 4 `gh issue
  comment` sites (control: the driver does call `gh`, 8 hits — the instrument works).
- *Not refuted, and worth recording as holding:* the separate-checkout decision. V1 was mid-iteration
  (pid 71129, 70 min elapsed) during this very bootstrap, so a shared tree would have contended
  immediately rather than eventually.

**Retro lane**: **driver-fix, LANDED.** Mark ratified the bar + queue and routed V21 here, so it was
taken this iteration rather than deferred. A degradation ledger accumulates at both probe loops and
emits ONCE, after every early exit and before the iteration starts, over the driver's existing two
channels. Not fail-closed on the post (that would make GitHub availability a hard dependency of every
fire) — but a failed post is loud, which the old path never was. Coverage without spending an
iteration: 5 stubbed-channel tests over the real emit block (fires when degraded; silent when
healthy; `gh` invoked; `gh` failure warns and does not abort; unset issue warns) + a forced codex
probe failure proving accumulation. The seam between the halves is now permanently testable —
`MISSION_DRY_RUN=1` reports `lanes=ok` / `lanes=DEGRADED(n)…`, both arms exercised. No Gate-5 skill
edit spent (this was driver, not skill).

**Next**: iteration 1 picks queue item 3 (disposition the 52 fork commits) — the top item neither
gated on #154 nor awaiting a routing decision. **Carried debt**: World is still owed the V21 fix and
cannot take it by copy (233 lines of drift, differently-shaped fallback site); handed over via the
cross-mission channel.

---

## 1 — 2026-08-12 — 51 fork commits dispositioned; the evaluator broke 2 of 16 SUPERSEDED, and both were real
**Picked**: Queue item 3 (**[NEXT]**, ungated) — disposition all fork commits. The gated items
(9/10/11) stay parked: `arniwesth/motoko_agent#154` is still **OPEN** against `main`, re-measured
this iteration, so the Phase-0 gate is genuinely closed and pulling that work forward was not an
option. This is the mission's **first iteration that actually ran** — iteration 0 was bootstrap,
and the 09:11 fire died on the trust-dialog defect (charter V22).
**Reality check**: Local `dev` was **4 behind** origin with a clean tree and 0 ahead, so mission
state was read from origin; reconciled by `git checkout -B dev origin/dev` after measuring all four
obligations with firing controls (ahead=0 / control behind=4; dirty∩incoming=0 / control 8=8).
Git wrote a **new inode** for the running driver (35083371→35155023), so the in-flight fire kept
its old fd — verified, not assumed. Item 3 was genuinely undone: only the 4-leg PR-#97 table
existed. No open motoko PRs, no stale worktrees, no uncommitted residue. `V8` re-verified
first-party: **52 ours-only / 805 theirs-only** vs `origin/main_dst@303d869`.
**Shipped**: [`752254d3f`](https://github.com/sunholo-data/ailang/commit/752254d3f) —
`design_docs/planned/m-motoko-fork-disposition.md` (114 lines) + 3 verification rows and 2 defect
fixes in the migration doc. **14 SUPERSEDED / 16 PORT / 14 DROP / 7 UNRESOLVED = 51.** Evaluator
round 1 **65/100 FAIL**; two rows downgraded on findings the controller reproduced first-party;
landed as REVISED-after-FAIL, not as a pass. Gate 3b: 13 of 14 checks green on the pushed SHA,
`test` still running at the poll's 8-min bound — **recorded as not-yet-green, per the rule that a
timed-out wait is not a green.**
**Routing evidence**: controller=`claude-opus-5` (quota-bucket:weekly-opus) task-class=triage/verify;
executor=`codex:gpt-5.6-sol` **as pinned** (probe rc=0, no degradation) task-class=execute
round1-score=65 rounds=1 corrections=2 provider=openai agent=codex-exec cost=quota-bucket(chatgpt-oauth);
evaluator=`sonnet` task-class=evaluate provider=anthropic — **generator≠judge held** (openai executor
vs anthropic judge); designer=**not spawned** (doc existed; rotation pointer untouched).
Quorum-at-pick: 2 reviewers, **both present**, `absent_reviewers` empty, **metered $0.064**.
**Iteration metered total: $0.064** of the $5 ceiling.
**Ruled out**:
- **"Does the file still exist upstream?" as a supersession test** — REFUTED before it was used, and
  it would have been ~85% wrong. 80 of 94 touched paths survive upstream, but `agent_loop_v2.ail`
  is **4,005 B** there vs **95,868 B** here: the paths survive as facades while the substance moved.
  Controls fired both ways (`.motoko/config/cloud/config.json` genuinely gone; `README.md` present).
- **"52 fork commits" as the number of dispositionable rows** — REFUTED. `ed61097` is a merge whose
  second-parent diff is **empty** (control: its first-parent diff is 2 files, so the instrument does
  read it). 51 is the ceiling; the doc's success criterion asked for a row that cannot exist.
- **`[stability] level = "stable"` as a machine-verifiable Phase-0 condition** — REFUTED, and this
  is the sharpest negative result of the iteration. It is the obvious remedy for the quorum's
  unbounded-wait objection, and it is **vacuous**: upstream ABI 5.0 declares it, and so does the
  2.2.0 we are pinned to and call unstable. Identical string, opposite truth. Any Phase-0 gate built
  on it passes immediately and falsely.
- **`ai-check` is already upstream** — REFUTED by scoping. 13 whole-tree hits (9 `.agent/` prose, an
  install script, 3 TUI scratchpad) collapse to **0** under `src/core` vs ours=1, with `compaction`
  10/26 as the same-path control and 68 upstream files proving the scope exists.
- **gemini's "the four PORT claims are unverified"** — procedurally UPHELD, substantively REFUTED
  4/4: `MOTOKO_MAX_STEPS` 0 upstream, `autoread` 0, the three `ws_*` helpers 0 (control: 126 `src`
  files contain `func`), `ai-check` 0 in `src/core`. The doc's verdicts were right; its evidence was
  absent. Measuring it (rule 3f) cost minutes and **shrank** the work to adding verification rows.
- **The executor's R16 SUPERSEDED** — REFUTED by the judge, reproduced by the controller. Upstream
  `should_retry_stream_error` exists *by name*; its body is `retryable && retry_enabled && budget > 1`
  and `session.ail:2379` builds `code: "Internal"` with `retryable: false`, in the file the row cited.
- **The executor's R34 SUPERSEDED** — REFUTED. `phase_vocab.ail` is a types module, not an emitted
  event; `runtime_config_resolved`/`config_resolved`/`policy_resolved`/`resolved_config` are all
  **0** upstream in `src/core` against a firing 29-file control.
**Retro lane**: **process-fix** — charter cadence corrected 12h/`43200s` (matched to the plist and
to Mark's 2026-08-12 note; the text said 6h), plus the namespaced
[motoko-mission-dashboard.md](motoko-mission-dashboard.md), because `mission-dashboard.md` is V1's
and Gate 4's "overwrite it" instruction would have destroyed V1's control context. **No skill edit**
— the one candidate (a first-party zsh `"$rev:path"` bite, see below) is at instance 1 for this
mission and the rule it would sharpen already exists.
**Next**: iteration 2 should run the **designer pass on the migration doc** for the two live quorum
objections, carrying this iteration's measurements (obj 2 is answered 4/4; obj 1 needs a real
bounded Phase-0 condition now that `[stability]` is refuted), then re-quorum ONCE. Queue item 4 or 5
otherwise. **R8 (output headroom) is now a named UNRESOLVED residual**, which strengthens the case
for item 5's instrument over item 4's upstream filing.

**Watch-item (instance 1 of 1 — not yet a skill edit, bar is two).** The controller hit the zsh
history-modifier trap the skill already documents at rule 3a(i-c), *inside this iteration*:
`git show "$BASE:src/core/recovery.ail"` returns **rc=128** and prints nothing, while
`"${BASE}:..."` returns rc=0 and 4,073 bytes. The sharpening the existing rule lacks is that the
bite is **path-selective** — the same unbraced form worked minutes earlier on
`"$BASE:packages/motoko-ext-abi/ailang.toml"`, because `:s` is a substitution modifier and `:p` is
not. So a spot-check that happens to use a different path **certifies the unbraced form as safe**.
Caught here only because the empty output was treated as a claim rather than a reading. If a second
instance lands, that is the skill edit.

---

## 2 — 2026-08-12 — Phase 0 is bounded now; the predicate as first drafted could never have flipped TRUE

**Picked**: Queue item 4 (head, `[NEXT]`, ungated) — designer pass on
[m-motoko-dst-refactor-migration.md](planned/m-motoko-dst-refactor-migration.md) for the two live
R1 quorum objections, then the one re-quorum the item allows.

**Reality check**: Gate 0/1 clean and measured, not assumed. Kill switch armed; `gh` on
`sunholo-voight-kampff`; billing tripwire CLEAN (both vars unset). The **RUNNING** skill is
byte-identical to `origin/dev` (`cmp -s` silent) — worth stating because this checkout has its own
tracked `.claude/skills/`, while `~/.claude/skills/mission-control` symlinks into **V1's** clone
(different inodes, identical bytes). Local `dev` == `origin/dev`, so no reconcile was owed and
Gate 4 could write in place. CI on HEAD: 18 checks, 17 green, `test` in-flight — no RED to outrank
the queue. Died-mid-flight sweep: no motoko worktrees, clean main tree, the one open loop-authored
PR (`#613`) is V1's. `#663` created 05:59Z **after** the Monday-07:00 boundary and holds 8
comments, so no rotation. **Item-level freshness**: only ONE quorum artifact existed for this doc
(the R1 block), and no commit touched it — the item was genuinely unstarted, not silently landed.
**Blocker re-verified rather than inherited** (the solved-upstream rule): `#154` is still `OPEN`,
control firing (`#150`/`#151`/`#152` all `MERGED` 2026-08-11 through the same instrument), so the
Phase-0 gate is genuinely closed and the item's premise holds.

**Shipped**: `1d0e2e511` — the revised migration doc. **Not a pass**: R2 BLOCKED, and item 4 is
**PARKED `needs-human-review`** on a decision that belongs to Mark (D1 below). R1's two objections
ARE answered and were not re-raised: Phase 0 is a bounded fail-closed gate (four conjunctive
predicates, each with its evaluating command and observed value; a 28-fire ≈14d timebox at the
charter's 12h interval; a structured BLOCKED expiry escalating to Mark; a declared human residual),
and the four "Port — carry forward" claims carry rows V21–V24, each with a **same-scope**
known-positive control. R2's two NEW objections were both internal-consistency defects in the R1
revision; `gemini`'s two were measured and FIXED in-loop, `gpt5-6-sol`'s was measured and PARKED.
No evaluator ran — the deliverable is a design doc, not an implementation.

**Routing evidence**: designer model=`claude:claude-fable-5` task-class=design
  round1-score=n/a rounds=2 corrections=3 (controller-authored, all measured)
  provider=anthropic agent=`claude-sub -p` (probe rc=0, no lane degradation) cost=quota-bucket:weekly-fable
  controller model=`claude-opus-5` task-class=orchestration cost=quota-bucket:opus
  quorum R2 provider=openai+google models=`gpt5-6-sol`,`gemini-3-1-pro` **both present,
  `absent_reviewers` empty** cost=**$0.096** · **metered total this iteration = $0.096 of $5**
  designer rotation: `mission-motoko-designer-rotation` was ABSENT → started at claude per the
  rotation rule, written back after the run. Note the shared un-namespaced
  `mission-designer-rotation` holds V1's state; using it would have collided (M1 namespacing).

**Ruled out**:
- **`[stability] level = "stable"` as the Phase-0 condition** — REFUTED first-party, reproducing
  iteration 1 rather than inheriting it: ABI 5.0's manifest and the registry's `2.2.0` (the line we
  pin *and call unstable*) both read `stable`. A gate on it passes immediately and falsely.
- **"the obvious remedy is vacuous, so there is no machine-verifiable condition"** — REFUTED, and
  this is the iteration's most useful correction to its own predecessor. `gpt5-6-sol`'s
  `proposed_fix` had **two** machine-checkable clauses and iteration 1 only tested one. The other —
  *does the registry expose 5.x at a pinned digest* — is **non-vacuous**: the registry lists
  `1.0.0,2.0.0,2.1.0,2.2.0` and no `5.x`, against a firing 4-entry control, so it reads FALSE today
  and flips TRUE only on Arni's republish. It also supplies the digest half (`content_hash` per
  version). Lesson: a `proposed_fix` is a list, and refuting its first clause is not refuting it.
- **The designer's G2 predicate** — REFUTED by running the table verbatim instead of reading it
  (V25). Bare `origin/main` is ambiguous across this mission's three repos; from this checkout it is
  `rc=128 invalid object name` because the anchor's default branch is `dev`. The wrong-repo error and
  the genuine path-absent answer **both return 128**, so an `exits 0` test cannot separate them —
  **G2 would have read FALSE forever, including after `#154` merges.** A gate that can never open,
  wearing a correct gate's clothes. Fixed with `-C` plus a `README.md` control.
- **`compaction-ai`'s `on_pre_step` "genuinely needs all ten effects"** — REFUTED, and **worse than
  the reviewer filed it** (V26). The row is `! {AI, IO, Trace}` — three. `gemini` flagged it as
  contradicting this doc's own V4; measuring it showed the sentence cited *as its authority* a
  passage upstream has **retracted**: *"WI-D8 NARROWED THIS ROW FROM TEN EFFECTS TO THREE, AND
  NOTHING HAD EVER MEASURED IT … it was taken as given. It was over-declared by SEVEN."* Control:
  `on_tool_handle` in the same record genuinely reads ten, so the instrument can see a ten-effect
  row where one exists. Second instance this mission of *a judge's finding being under-stated*.
- **"the Design Freeze checkboxes are harmless bookkeeping"** — REFUTED by `gemini`: gating the
  sprint-executor's start on G1–G4 is a **deadlock**, because Phase 0 is what the executor runs to
  evaluate G1–G4. It would have silently restored the unbounded manual wait R1 rejected.
- **Applying the narrow-refinement carve-out to `gpt5-6-sol`'s R2 objection** — REJECTED as a
  route. Its `proposed_fix` offers two branches, one of which explicitly requires "the named
  decision owner's acceptance", and branch A's G6 would make queue item 11 (itself Phase-0 gated)
  a Phase-0 predicate — a dependency loop. Choosing is judgment, and the carve-out forbids a
  controller-invented resolution. Parked instead.

**Retro lane**: none — no skill edit this iteration. The candidate gap (a predicate table in a
multi-repo mission needing its repo named, and "the command errored" not being "the predicate is
false") has **one** recorded friction, below the ≥2 bar; it is recorded in the doc as V25 and
carried as a watch-item. If a second instance lands, the fix is a Gate-2 clause aimed at
*evaluating* a doc's commands rather than reading them.

**Next**: item 4 stays parked on **D1**. Iteration 3 takes item **5** (output-headroom upstream
issue) — ungated, and the disposition's `R8` UNRESOLVED row now gives it a concrete instrument to
cite rather than a recollection. If Mark answers D1 first, item 4 unparks and outranks it: apply
his branch, then the doc is quorum-clean without another paid round.

---

## 3 — 2026-08-13 — the queue head waited on a human who never replied, and the case it was waiting to file was wrong in both directions

**Picked**: queue item 5 (head), **Output-headroom upstream issue** — clause 3, "file the case
against `main_dst` if Arni's #97 reply invites it". Nothing outranked it: dev CI green (16 checks on
`7ea12af93`, 0 not-green), zero human directives since the watermark, no regression in the inbox.
The three unread cross-mission messages (World iter-80 ×2, V1 iter-191) are sibling reports and one
skill proposal addressed to V1 — per Gate 0 they never auto-outrank, and none is a motoko demand.

**Reality check**: the item's **precondition is FALSE, and it is an unbounded wait**. Measured
first-party: **zero** `arniwesth` events on PR #97 across all four surfaces — `gh pr view --json
comments` (1 comment, ours, 2026-08-11T19:39:21Z), `pulls/97/comments` (0), `pulls/97/reviews` (0),
`issues/97/timeline` filtered to his login (0). **Control fires**: `search/issues?q=repo:…+commenter:arniwesth`
returns **34**, so the instruments can see him where he is. Waiting on a third party with no expiry
is precisely the defect `gpt5-6-sol` blocked Phase 0 for at iteration 1 — and it was sitting in the
queue head *one iteration after* we fixed the identical shape next door. The Phase-0 fix generalised
to the gate and not to the queue.

Also checked and clean: driver pin **pinned** (`7ea12af93`; source clone 38 behind, which is what the
pin exists to survive); the RUNNING skill byte-identical to `origin/dev` on **both** resolutions (the
repo-local `.claude/skills/` that wins here, and V1's symlinked copy); no died-mid-flight traces
(both motoko checkouts clean, `git worktree list` shows only the pin and the clone, and the one open
loop-authored PR `#613` is V1's); no rotation due (`#663` created 2026-08-12 07:59 local, after the
Monday 07:00 boundary; 10 comments < 80).

**Shipped**: the item's **evidence half**, which is what any filing must cite — and it says the case
as written would have filed something incorrect. Re-measured against `main_dst@6c06b08` (advanced
from the `303d869` the doc's rows were taken at) and landed as migration-doc rows **V27–V29** plus a
rewritten compaction section, a sharpened disposition **R8**, and a bounded queue item 5.

1. **The live ladder targets 70%, not the 95% the doc asserted.** `register.ail:24` binds
   `compact_for_pre_step`; every branch gates on `candidate_fits` → `< result_target_pct()` =
   `elide_tier_pct()` = **70**. `emergency_pct() = 95` occurs at exactly 3 sites, all inside the
   off-path `compact_step_with_limit`. On a 262144 window 70% leaves **≥78,644** — more than the
   65,536 output cap *and* more than the 75k reserve `96542f8` adds. Upstream already beats our
   patch **when the ladder reaches its target**.
2. **The mitigation the doc credited has zero production callers.** Every importer of
   `compact_step_with_limit` is a smoke or integration test. Control: `compact_for_pre_step` is
   imported by the live `register.ail:17`, so the instrument distinguishes live from test.
3. **The live refusal exists anyway, and is loud** — the phase core's `seal_compacted_payload`
   (`phase_vocab.ail:145`) at `exhaustion_pct() = 95` (`compaction.ail:30`), called at
   `session.ail:2561` with the **raw** `context_limit`, terminating `CompactionExhausted` /
   `code: "ContextExhausted"` / `retryable: false`. "Hard-stop, not silent corruption" is
   **CONFIRMED at a different function than we named**.

Net: the residual is neither "no reserve" nor "no refusal" but **the band between the ladder's 70%
aspiration and the seal's 95% permission**, whose predicate counts messages only. That shrinks the
upstream ask from "adopt an output reserve in a compaction extension" to **one argument at
`session.ail:2561`** — and the precedent to cite is one line up at `:2534`, where upstream already
hands the extension `context_limit - pinned_tokens`, subtracting a reserve before delegating.

**Routing evidence**: model=`claude-opus-5` task-class=**mechanical/analysis** (controller-only —
Gate 2 reality-check plus first-party measurement; **no sub-agent spawned**, because no design,
plan, execution or judgement role was needed and spawning one would have billed a role with nothing
to do) round1-score=n/a rounds=1 corrections=1 (self, see Ruled out) provider=anthropic
agent=mission-control cost=**metered $0.00** of $5 · quota-bucket:opus. Designer rotation
UNTOUCHED at `claude:claude-fable-5` (no doc was created — the migration doc was *corrected* from
measurement, which rule 3f assigns to the controller, not to the designer). No lane degradation, no
fallback traversed. Quorum: **not run** — no new or revised *design*; the migration doc's one
re-quorum is spent and these edits are Verification-Log corrections, not design changes.

**Ruled out**:
- **"The live compaction path has no refusal at all"** — my own first reading, from the
  unconditional `else Compacted(floor, …)` branch at `compaction_structural.ail:191`. **FALSE.**
  Caught only by chasing the *second* `compaction_exhausted` site rather than trusting upstream's
  own ADR-002, which calls the off-path one "the only `compaction_exhausted` `Err`" and is itself
  **stale** at `6c06b08`. Had I banked the first reading, the correction to the doc would have been
  as wrong as the thing it corrected, in the opposite direction.
- **"V9 established that upstream has no output-headroom protection."** V9 is true *as measured* —
  a token grep for `headroom|reserve|max_tokens|output_budget|effective_window` genuinely returns
  nothing. The behavioural clause the compaction table built on it ("tiers run off raw
  `ctx.context_limit`") is not something any token grep can support, and is wrong for the live path.
  **Generalisable**: a negative-existence grep for the vocabulary of *our* fix establishes only that
  the other tree does not spell it our way — never that it lacks the property. Upstream's protection
  is spelled `result_target_pct`, a word no such grep contains. Two quorum rounds read past it.
- **"R8 needs a rig A/B."** No — both `compact_for_pre_step` and `seal_compacted_payload` are
  `pure`, so the one open question (is the 70→95 band reachable?) is a direct unit-level call.
  R8 re-sharpened accordingly.
- **"`main_dst` is unchanged since iteration 0 measured it."** It moved `303d869 → 6c06b08`. The
  delta is docs-only (`.agent/prs/2026-08-11-main-dst-integration.md`, +184), so no measurement was
  invalidated — but the check is why V27–V29 name their own base rather than inheriting one.

**Retro lane**: **process-fix** — queue item 5's precondition replaced with a bounded rule
(2026-08-27 expiry ⇒ file standalone + carry locally), and the migration doc's matching Deferred
Decision struck and bounded the same way. No skill edit: the two frictions this iteration surfaced
(the unbounded-wait-in-a-queue-row, and V9's vocabulary grep) are each at **instance 1** for this
mission, below Gate 5's ≥2 bar. Both are logged here as watch-items. The V9 shape is close kin to
the skill's existing rule 3a(i-d) (scope) but is genuinely distinct — 3a(i-d) is about pointing an
instrument at a path that does not exist; this is about pointing a *correct* instrument at the
wrong *vocabulary*. If it recurs, that is the skill edit.

**Next**: item 5's remaining unit-level assertion (R8's re-sharpened form) — cheap, and it decides
PORT vs SUPERSEDED for the last open leg of PR #97. If it is deferred, item 6 (`fmt`
re-measurement instrument) is the next unblocked design item; items 9–14 remain Phase-0 gated and
Phase 0 still measures CLOSED (`#154` OPEN, control: `#152` MERGED).

---

## 4 — 2026-08-14 — R8 settled → PORT, and the ladder turns out to have no lever at all; plus the recovery job that made this iteration exist was live but uncommitted

**Picked**: Two items, per Standing rule 1's bookkeeping clause. **(a)** A Gate-2 died-mid-flight
trace found before the queue was consulted: `git status` in the mission workdir showed
`tools/launchd/dev.ailang.mission-recovery-motoko.plist` — untracked, complete, written 08:44 —
which changes the pick by the skill's own rule (verify and land, do not redo). **(b)** The queue
head, item 5, whose only remaining unit was the disposition's **R8**, sharpened by iteration 3 to
one `pure`-function assertion.

**Reality check**: Local HEAD == `origin/dev` (`bad8f3647`), no reconcile owed. Running skill
byte-identical to `origin/dev` — and checked against the **resolved symlink target** in V1's
checkout (`readlink` → `~/dev/sunholo-data/ailang/.claude/skills/...`), not the repo-local copy,
which is a different file that happens to match. Gate 1: **20** checks on HEAD, **0** not-green
(`checks=20` is the known-positive control, so the endpoint answered). Zero human directives since
the `#663` watermark (control: World's `#53` → **1**, instrument fires). Weekly external-issue
sweep NOT due — `#663` was created 2026-08-12, after the most recent Monday-07:00 local boundary
(2026-08-10), and holds 12 comments, so neither rotation trigger fires.
Open loop-authored PRs `#695`/`#613` are both V1's, not this mission's.
The plist's own header claims were verified rather than inherited: `mission-recovery.sh` really is
`MISSION_NAME`-parameterized (lines 32, 42–47) and the installed v1 plist really does pin
`MISSION_NAME=v1`, so motoko's `.blocked` marker genuinely had nothing watching it.

**Shipped**:
- `ceb2bb055` — the recovery launchd job, committed. Landed only after proving it is behaviourally
  identical to the copy actually running (`plutil -convert json` byte-equal, v1 plist as a firing
  negative control), so the commit changes comments and nothing else.
- R8 → **PORT** in `m-motoko-fork-disposition.md`; ledger now **14 SUPERSEDED / 17 PORT / 14 DROP /
  6 UNRESOLVED = 51**. Counts propagated to the migration doc (4 sites, each substitution asserted
  unique before writing) and the dashboard; the iteration 1–3 STATUS stamps and log entries are left
  as written, since they record what was true then.
- `tools/motoko/r8_headroom_band.ail` + `tools/motoko/README.md` — the instrument, committed to the
  anchor repo so the measurement is reproducible rather than a number in a log entry. It cannot go
  upstream: we are guests there.
- New queue item **5a** — the empty-output probe hang, diagnose-only.
- No evaluator ran: both deliverables are controller measurement, not generated artifacts, so no
  role needed a pin and generator≠judge was not engaged.

**Routing evidence**: model=claude-opus-5 (controller session) task-class=mechanical+measurement
  round1-score=n/a rounds=1 corrections=0
  provider=anthropic agent=controller cost=quota-bucket:weekly-opus · **metered $0.00 of $5**
  No designer/planner/executor/evaluator/quorum lane fired — designer rotation pointer untouched at
  `claude:claude-fable-5`. `make quick-install` deliberately NOT run: an `ailang eval-suite` (3
  models incl. `motoko-local-qwen3`) has been live on the shared `~/go/bin/ailang` since 07:58, and
  the charter's shared-write guardrail makes swapping it mid-eval a language-regression-shaped
  hazard. The probe therefore ran on the installed `v0.33.1-23-g644cf178a-dirty`, and that narrowing
  travels with the result (rule 3b(ii)). No GPU touched (`pure` functions only) — no `rig.lock`.

**Ruled out**:
- **"The ladder grinds down under a long transcript and lands in the band."** Refuted. The ladder
  has no lever at all on this shape: `elide_walk` only rewrites `role=="tool"` messages, so a large
  **user** message is invisible to all four tiers, which removed **2,061 of 208,980** tokens (~1%)
  between them. The floor branch fires on an essentially unchanged payload. This matters beyond the
  row: it is why an extension-side reserve cannot fix the case and the seal is the only place that
  can see it.
- **"The seal's 95% hard stop catches the over-target case."** Refuted for the band — arm A returns
  **`Ok`** at 79% and SENDS with 54,905 tokens of headroom against a 65,536 output cap. The seal is
  real (arm C, 158% → `Err(SealExhausted)`), it simply permits everything below 95%.
- **"The plist header's 'three of six refusals' is the current rate."** Refuted by re-derivation:
  **6 refusals / 4 starts = 10 fires, 60%**. The header was true when written at 08:44 and three
  more refusals followed it within the hour.
- **"The probe stall is a motoko-specific misconfiguration."** Refuted by control: v1 refused with
  the identical empty-output signature at 05:20/05:36/08:49 on 08-14, overlapping motoko's
  08:47/09:07/09:27, from a separate checkout with separate config.
- **"`ailang lock` regeneration is optional for a local `main_dst` worktree."** Refuted: the
  committed lock carries **absolute** `/workspaces/motoko_agent` paths (19/19), so every package
  import fails with "package directory not found" — which reads as a broken checkout, not a stale
  lock. Ten minutes, now documented so it is paid once.

**Retro lane**: none — no skill edit this iteration. The two frictions found (an `ailang fmt`-shaped
PostToolUse hook rewriting `.ail` files on write, against the charter guardrail; and the driver's
"quota-limited, timed out, or errored" summary flattening a *hang* into the same sentence as a quota
refusal) are **one recorded instance each**, below the skill-fix bar of ≥2 frictions on the same
gap. Both are named here so instance 2 has something to point at, and the second is now queue item
5a's starting evidence.

**Cross-mission collision, found at push time and worth more than the inconvenience**: my push to
`dev` was rejected because **V1's iteration 198 had landed the same orphaned plist ~15 minutes after
I did** (`f5a7ce8be`, "motoko had no recovery watcher"). Two missions independently found the same
uncommitted artifact and both acted on it — exactly the contention the charter's Repo Profile
predicts ("there is no cross-mission lock … the driver's overlap guard is a per-mission pidfile"),
now observed rather than theorised, and on *work* rather than on a git ref. Resolved by keeping
their commit and rebasing mine on top as a **correction**: the two files are **operatively
identical** (`plutil -convert json` byte-equal, and the resolved file is byte-equal to the copy
actually running), so the only delta is comments — and theirs carries the "three of six refusals"
figure that three later refusals had already falsified. Nothing was lost either way; the notable
part is that neither loop could have known. Generalises past this instance: a died-mid-flight trace
is visible to *every* mission on the rig, so the Gate-2 verify-and-land rule has a race in it that
only shows up when two loops are awake at once.

**Next**: item **5a** (diagnose the empty-output probe hang — it is costing 60% of this mission's
fires, and recovery-by-brute-retry now hides that from the human channel), then item 6 (the fmt
re-measurement instrument). Item 5 needs no more work from us: it is waiting only on the
2026-08-27 bound.

---

## 5 — 2026-08-14 — the fleet's "no usable model" refusal was never about models, quota, or contention: our own SessionStart hook was holding its stdout open

**Picked**: Queue head **5a** — "the driver's model probe hangs with EMPTY output, and it is
costing this mission most of its fires", scoped by iteration 4 to *diagnose only*, with an
explicit prohibition on lengthening the 120s probe timeout before the mechanism is known.

**Reality check**: Local HEAD == `origin/dev` (`6f3e6bce7`), no reconcile owed. Running skill
byte-identical to `origin/dev` — `cmp` against the **resolved symlink target** in V1's checkout
(`readlink` → `~/dev/sunholo-data/ailang/.claude/skills/...`), paired with a deliberately-different
file as a firing control, since a silent `cmp` proves nothing unless it can also report a
difference. Gate 1: **16** checks on HEAD, **0** not-green, `total_count=16` as the known-positive
control. Zero human directives on `#663` since the watermark. Weekly external-issue sweep NOT due
(`#663` created 2026-08-12, after the Monday-07:00 *local* boundary; 13 comments < 80). No
died-mid-flight traces: both motoko checkouts clean, no stale motoko worktrees, and all three open
loop-authored PRs (`#719`, `#695`, `#613`) are V1's. No quorum owed — the pick has no design doc;
it is a diagnostic with a one-line remedy.

**What was found**. `scripts/hooks/session_start.sh` ended with
`ailang docs embed-warmup --quiet --timeout 3m &`, commented *"(non-blocking)"*. That is true of
the **script** and false of every **consumer that captures its stdout**: a backgrounded child
inherits the stdout descriptor, so a `$(...)`-style capture cannot observe EOF until the **child**
exits, however promptly the script itself `exit 0`s. Claude Code captures hook stdout — that is
how the SessionStart banner reaches a session — so the hook was held open for as long as the
warmup ran, bounded only by the warmup's own `--timeout 3m` (**180s**) against the driver's
**120s** `PROBE_TIMEOUT`. Hence `probe timed out after 120s — captured output: ''`: empty because
`claude -p` emits nothing until it completes, and identical across all three models because **it
was never a model verdict — it is one hook stall observed six times** (3 models × 2 attempts).

**Measurements** (darwin/arm64; windows and ubuntu legs unrun locally, rule 3b(viii)):

| arm | elapsed |
|---|---|
| hook stdout **captured** | **8,433 ms** |
| hook stdout **redirected to a file** | 8 ms |
| identical script, **background child removed** (negative control) | 237 ms |
| one real `session_start.sh` capture, contended | **96,377 ms** |
| warm `embed-warmup` alone | ~1.2 s |

| mission | refusals / fires | hooks |
|---|---|---|
| v1 | **47 / 186** | 3 SessionStart hooks |
| motoko | **6 / 11** | 3 SessionStart hooks |
| world | **0 / 89** | **no `.claude/settings.json` at all** |

World's zero is a measurement, not a broken grep: its log carries **90** `probe ok` lines over the
same 24-day window (both logs start 2026-07-21). And **`quota-limited` has never fired once** in
either driver log, so the driver's own `quota-limited, timed out, or errored` summary has only
ever meant *timed out* — every reading of these refusals as quota pressure, this charter's
included, was wrong.

**Amplification, measured rather than inferred**: `_mc_bounded` kills the `claude` process on
expiry, but the warmup is a **grandchild** and survives — verified reparented to `ppid=1`. So each
timed-out probe leaves a GPU tenant behind and the next probe adds another, up to six per fire.

**Ruled out**: **GPU contention alone is not sufficient**, refuting this controller's own leading
hypothesis. The filler held `rig.lock` from **07:58:30 to 12:39:58** on 08-14; motoko's
*successful* 09:45 fire sits inside that window alongside the three refusals preceding it. The
correlation fit three data points and died on the fourth — rule 3d, where the evidence arrived in
exactly the predicted direction and only the negative control separated it.
Also ruled out: auth/trust (V22's class) — the never-failing mission, world, has
`hasTrustDialogAccepted=False`, which inverts the trust story rather than supporting it.

**Corrected**: iteration 4's charter claim that the stall is *"not motoko-specific"* and therefore
environmental. It is common to v1 and motoko because both are `sunholo-data/ailang` checkouts
carrying the three SessionStart hooks; world is immune for a structural reason, not a lucky one.
"Two missions failing together" was read as evidence of an environment when it was evidence of a
shared **repo**.

**Deviation from the item's scope, declared**: widened from diagnose-only to include the fix. The
prohibition was specifically against lengthening the 120s timeout before the mechanism was known;
the mechanism is now known and the remedy is a stdout redirect, which is the opposite change.

**The sharpest finding is about the guard, not the fix.** `tools/launchd/test_hook_stdout.sh`'s
first draft **passed against the deliberately mutated hook** — it reported `0ms` with the defect
present. `session_start.sh` early-returns on the no-unread-messages path (`:287`) **before** the
warmup line, so a stub answering `[]` never reaches the code under test: an observable that is not
downstream of the mechanism, which is rule 3i exactly. It was caught only because the mutation was
*run* rather than reasoned about. Repaired with a marker control asserting the warmup line is
REACHED, plus an anti-vacuity floor on the hook enumeration and a control proving the stubbed
warmup is genuinely slow. The arm now goes **1s → 11s RED** on the mutant; mutant asserted LANDED
by sha256 and parsing under bash 3.2, restored **byte-identical from a `cp` backup** — never
`git checkout --`, which in a worktree would have deleted the uncommitted work.

**Routing evidence**: controller `claude-opus-5` (session, `probe ok`). **No sub-agent spawned** —
the deliverable was controller measurement plus a one-line fix with a guard, so no role needed a
pin; designer rotation pointer untouched. metered=**$0.00** of $5. No GPU work, so no `rig.lock`
taken. `make quick-install` deliberately NOT run (shared-write guardrail, V20).

**Landed**: PR **#721** — `scripts/hooks/session_start.sh` (the redirect + a comment explaining why
it is load-bearing), `tools/launchd/test_hook_stdout.sh` (new guard), `make/test.mk` (wired into
`make test-launchd-drivers`, already a CI gate at `ci.yml:472`), `changelogs/v0.18-current.md`.

**Next**: item **5b** — split out of 5a and genuinely still open: `make test-launchd-drivers` is
green in CI and **1 passed / 28 failed** on the rig (`test_pin_root.sh`), re-measured this
iteration from a pristine `origin/dev` worktree, so the one gate covering the driver scripts is
blind where those scripts run. Then item 6 (the fmt re-measurement instrument). Item 5 needs
nothing from us until its 2026-08-27 bound.

---

## 7 — 2026-08-16 — recovered iteration 6: item 5b had landed, but the mission record never did

**Picked**: Gate 2's died-mid-flight trace outranked the queue. The named motoko checkout was 21
commits behind origin with mission-related uncommitted residue, and a clean
`sprint/motoko-iter6-pin-root` worktree remained after PR #728 had merged. This is credited as
iteration 6's work and recorded by iteration 7; it is not re-executed.

**Reality check**: `origin/dev` at `a2f2dc193` is authoritative; the scheduled pin checkout is
clean and byte-identical to it, while the named main checkout is stale and dirty and was left
untouched. The running mission-control skill matches origin. Gate 1 found 16 exact-SHA checks and
zero not-green. GitHub account is `sunholo-voight-kampff`; billing tripwire CLEAN; no allowlisted
directive since the #663 watermark; one sibling report was informational and acknowledged. Weekly
issue rotation/sweep was not due: #663 was created after the latest Monday 07:00 local boundary
and has 16 comments.

**Shipped / recovered**: PR #728 (`4f300bfa1`) had already merged item 5b. Its fix clears ambient
scheduled-driver pin state before the hermetic pin-root lab, making the rig execute the same
synthetic paths as CI. First-party re-run: `tools/launchd/test_pin_root.sh` = 35 passed / 0 failed;
the PR records the negative mutation without isolation = 3 passed / 32 failed. All 21 PR checks
were terminal success/skipped, including `launchd drivers (bash 3.2)`. The routing/decision files
in the residue were already landed by `de0e41099`; byte hashes matched origin for the skill,
driver, planner router, make target, and decision script. This record also restores the Motoko
decision ledger (2 RESOLVED, 0 OPEN), which had existed only in the stale checkout.

**Routing evidence**: controller=`codex:gpt-5.6-sol`; task-class=recovery/bookkeeping;
provider=codex; no designer/planner/executor/evaluator/quorum lane fired; generator≠judge not
engaged; no GPU and no `rig.lock`; metered=$0.00 of $5.

**Ruled out**:
- "Iteration 6 work is still unlanded" — refuted by merged PR #728 and merge SHA `4f300bfa1`.
- "The dirty main checkout is the correct record base" — refuted: it is 21 commits behind origin;
  editing it would erase newer mission history. Record written from an isolated origin-based worktree.
- "CI green already proves the rig test" — not used. The exact rig-facing test was re-run locally
  and passed 35/35; the mutation evidence proves the isolation is load-bearing.

**Retro lane**: none. The existing died-mid-flight rule found both traces and prescribed the right
action. No second friction warrants a skill edit; no routing-policy change was made.

**Next**: queue item 6 — design the real `motoko_ext_fmt` re-measurement instrument. Item 5 remains
bounded until 2026-08-27; there are no OPEN human decisions.

## 8 — 2026-08-17 — the −74% is one benchmark and one void pair; on the honest pairs it is −5.7%

**Picked**: queue head item 6 (fmt re-measurement instrument). No regression, no human directive
and no died-mid-flight trace outranked it. The weekly external-issue sweep was due and was run, but
a sweep never outranks a pick — its output is the new queue row 6b.

**Reality check**: `origin/dev` `127c1443e`; the scheduled pin checkout is clean, detached at that
exact SHA, and the running mission-control skill is byte-identical to origin (`cmp` rc=0). Gate 1
read 16 exact-SHA checks with 0 not-green, and the count is non-zero, so dev is *verified* rather
than merely un-red. `gh` on `sunholo-voight-kampff`; billing tripwire CLEAN (re-checked immediately
before the nested `claude`). Zero allowlisted directives since the `#663` watermark
(`2026-08-16T00:23:15Z`). Died-mid-flight sweep: both motoko worktrees clean; the two open
loop-authored PRs (`#695`, `#613`) are V1's; the stale main checkout's 7 modified paths were
re-verified as 5 byte-equal-to-origin plus 2 explained by its 21-commit lag (`test_mission_routing.sh`
exists on origin — it is untracked *there* only because that clone lacks the commit), so no orphaned
work. Item-level freshness: `grep -ril` found prior fmt art (`m-eval-fmt-weakmodel-ab-M6-motoko-ext`,
`m-fmt-dialect-alignment`) but no doc for the *instrument*, and no merged PR for it.

**The finding — the pick's own premise did not survive Gate 2.** The −74% (AC5,
`m-fmt-dialect-alignment.md`, 2026-07-31) is real and correctly attributed to the fmt extension.
It is also not a target an instrument can reproduce, for five reasons measured first-party:
(a) its author rates it *"direction, not proof"* — n=1/pair, sign test p≈0.11; (b) **74.7% of the
saving is ONE benchmark** — six pairs give −74.2%, dropping `log_file_analyzer` gives −47.1%, and
that pair alone is 3,125,933 of 4,182,882 saved tokens; (c) that benchmark is **3/30 lifetime and
0/10 over the last five nights** in the `rag_on` opencode rotation lane (open `#649`), so
tokens-to-pass is undefined there; (d) `ab2_fmt_on/emit_exact_bytes_varied` banked **zero** fmt-hook
events with `validity={valid:False, reason:treatment_unproven}` and was summed into the headline
regardless, while the other 5 ON rows carry exactly one `status=formatted` event and all 6 OFF rows
are clean; (e) the run was **order-confounded** — sorting all 12 rows by in-file `timestamp` gives a
perfect block, ON 16:42:37→17:00:10 then OFF 17:01:28→17:36:49, and the within-arm benchmark order
differs between arms, so index pairing never paired the same position.
**Net: on the four pairs with proven treatment and a currently-passable benchmark, −5.7% (ON cheaper
3/4), not −74.2%.**

**Shipped**: `design_docs/planned/m-motoko-fmt-remeasurement-instrument.md` (421 lines,
`0e1edd80c`) — paired **censored win-rate** (non-pass right-censored at the 4M cap; null
P(ON wins | non-tied)=0.5, defined even when nothing passes), a **counterbalanced** per-benchmark
schedule with an order-integrity gate that VOIDs a slot banked otherwise, ELO-banded selection with
`log_file_analyzer` out and continuity with −74% deliberately abandoned, ~9.8 rig-hours priced off a
measured 4.91 min/row anchor, and a **pre-registered** KEEP/RETIRE/inconclusive rule.
**PARKED `needs-human-review`** on D-MOTOKO-FMT-1 alone.

**Confirmed live defect (not the pick, found on the way)**: the Wednesday fmt A/B lane has banked
nothing since AC5. Both the 2026-08-05 and 08-12 fires died at
`internal/executor/motoko/healthcheck.go:64` — an **unconditional** `OPENROUTER_API_KEY` refusal
with no lane/model condition, reached via `MotokoExecutor.HealthCheck`→`runHealthCheck` — whose own
error text (*"motoko routes ALL models via OpenRouter"*) is false for both fmt arms, which declare
`provider: "ollama"`, `env_var: ""` (*"No API key — local inference"*) and
`agent_model_name: "ollama/qwen3.6:35b-a3b-mxfp8"` (models.yml:1854, :1880).

**Quorum**: two rounds, **both external reviewers present both times**, `absent_reviewers` empty in
both artifacts — so neither verdict is an N−1 degrade and no re-run at a raised cap was owed. R1
BLOCKED → one revision → R2 BLOCKED. Metered **$0.1424** ($0.0619 + $0.0805). All four objections
were classified premise-vs-design and **every premise was measured rather than forwarded** (rule 3f):
- **O1** (`gpt5-6-sol`, arm ordering) — **UPHELD, and its "if" is fact**: the reviewer supposed
  hypothetically what the banked rows show outright (finding (e)). Answered in §5.3.
- **O2** (`gemini-3-1-pro`, the `cp`-to-LaunchAgents claim) — **UPHELD**:
  `PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist` →
  `{/bin/bash, /Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/nightly-eval.sh}`, i.e. the
  script runs **in place from V1's checkout**. Instruction removed; replaced with the real
  constraint (a merged fix reaches the rig only when V1's clone pulls — open `#558`).
- **O3** (`gemini-3-1-pro`, `#649` unverified) — procedurally right, claim measured **TRUE**
  (`gh issue view 649` → OPEN, created 2026-08-11; control `#721` → MERGED).
- **O4** (`gpt5-6-sol`, D1's routing premise) — **the park**. Partly measured (see the ledger row);
  the missing half needs the `mk-ast` resolution path and/or a live motoko run under `rig.lock`.

**Why parked and not carved out**: the narrow-refinement carve-out requires *every* remaining
objection to carry a concrete reviewer-authored fix needing no controller judgment. O3 clears that
bar; **O4 does not** — its remedy is an investigation, and choosing between "make the trace a D1
precondition" and "redesign D1" is judgment. Forcing it would ship a prerequisite whose safety
nobody established, inside a doc whose entire subject is that the previous measurement's integrity
went unchecked. **No reviewer disputed the instrument's DIRECTION in either round.**

**Weekly external-issue sweep** (due: first fire past the 2026-08-17 07:00 **local** Monday
boundary; `#663` created 08-12 07:59 CEST, before it): **15 orphans of 75 enumerated open issues**,
printed as a per-issue table over 8 charter/log/archive/dashboard files. Controls fired
(`#663`→motoko charter 4; `#617`→v1 charter 2; `#663`→`mission-dashboard.md` 1) and the list length
was asserted against `gh issue list … | wc -l` = 75. Batched into queue row **6b**; most orphans are
AILANG-lane and belong to V1.

**Routing evidence**: controller=`claude-opus-5` (session); designer=`claude:claude-fable-5` via
`claude-sub`, probe rc=0, **FLAGGED as a rotation fallback** — the pointer's next entry is
`codex:gpt-5.6-sol` and codex is genuinely quota-exhausted (my own re-probe: rc=1,
*"You've hit your usage limit … try again at Aug 20th"*, corroborating the driver's degradation
notice), gemini unwired pending G4, so the rotation wrapped to claude; planner/executor/evaluator
**never spawned** (a parked design doc has no plan and nothing to judge, so generator≠judge was not
engaged); quorum=`gpt5-6-sol` + `gemini-3-1-pro`, both present both rounds; no GPU, no `rig.lock`;
`make quick-install` deliberately NOT run (shared-write guardrail); gates ran on **darwin/arm64
only** (rule 3b(viii)); metered=**$0.1424** of $5.

**Ruled out**:
- *"The −74% is a ghost / was never measured"* — refuted. It is banked, arithmetically correct
  (recomputed −74.2%), and correctly attributed to the fmt extension; the defect is its
  *strength and composition*, not its existence.
- *"The M6 doc's 'firing not yet observed' means fmt never fired"* — refuted by the rows: 5 of 6
  AC5 ON rows carry a `status=formatted` event. The M6 header simply predates AC5.
- *"`gemini`'s O2 was pedantry about a doc footnote"* — refuted; it was a false instruction, and
  the truth underneath it (V1's checkout is what the plist executes) is load-bearing for D1.
- *"My V-D pass rates are a property of the benchmarks"* — **corrected by the designer, and it was
  my error**: `nightly-eval-history.jsonl` holds `arm=rag_on` **only** (516 rows, 2 opencode
  models), so those rates are scoped to the rotation lane and not to any motoko or fmt arm. Rule
  3b(ix) — the scope now travels with the number everywhere it is quoted. A sub-agent narrowing a
  controller "fact" is the loop working.
- *"15 orphans" from the sweep's first run* — the count survived, **the table did not**: see Retro.

**Retro lane — skill fix (one, per the one-edit rule)**: Gate 0's weekly-sweep rule says to *print*
per-issue counts so a zero is an auditable row. The natural zsh implementation of that instruction
is wrong: **zsh arrays are 1-indexed**, so `${counts[0]}` is empty, every column silently shifts
right by one, and the **last file in the list is dropped from every row**. My first sweep table
printed `mission-dashboard.md`'s counts under the wrong header and never printed the 8th file at
all; the orphan *verdict* was unaffected only because the accumulator summed the loop variable
rather than the display array. This is the second instance of the zsh-array class the skill already
records (iteration 140's `FILES=$(…)` word-splitting produced a vacuous mutation test), and the
first to hit a *reporting* instrument, where the failure is invisible precisely because the output
looks like a table. Edit made to the Gate-0 sweep rule.

**Next**: answer **D-MOTOKO-FMT-1** (one word: precondition / redesign) and item 6 becomes a normal
sprint. Otherwise queue row **6b** (triage-lite the 15 orphaned issues) is the head. Item 5 remains
bounded until 2026-08-27.

## 9 — 2026-08-17 — dev was red two different ways, the changelog gate saw 1 of 5 offenders, and V1 was four minutes behind me on the same fix

**Picked**: NOT the queue head. `origin/dev` HEAD `0002c9b0b` was RED, which under the rulebook in
force at pick time outranked the queue. (By the end of the iteration that rule had changed under me
— see Ruled out.)

**Reality check**: ran from the `#558` driver pin root, detached and clean at `0002c9b0b` ==
`origin/dev`; running skill byte-identical to origin (`cmp` rc=0). Prior motoko worktrees (iter 6/7/8)
all clean; open loop-authored PRs `#744`/`#695`/`#613` are all V1's. Decision ledger valid, 3 rows,
1 OPEN (`D-MOTOKO-FMT-1`) and no directive answered it. Weekly issue sweep not due (iteration 8 ran
it this morning; `#743` created 05:48Z, after the Monday-07:00 local boundary).

**The red was TWO reds, and the split is the whole disposition.** `test` failed at `steps=41`,
failing step `Check changelog index hygiene` — a repo command that ran and failed. `lint` and
`Build macos-latest` (×2) failed at `steps=1, last=Set up job`, i.e. before checkout, inside a
declared GitHub incident (`13:40:03Z`, impact **critical**, `investigating`) whose window covers the
run's `15:49:32Z`. Those three were diagnosed, never reverted or fixed-forward. The negative control
came later and is decisive: the **same three jobs were green** on my PR head at 21:30Z on a
byte-different tree — they failed for the platform, not for any diff.

**The finding — the gate could only ever have caught this by luck.** `4c3ef27c8` appended release
notes to root `CHANGELOG.md`, which is an index; release-manager builds notes from
`changelogs/v0.18-current.md`, so anything left there is silently dropped. But
`scripts/check_changelog.sh` matched Keep-a-Changelog *keyword* headers
(`### Added|Fixed|Changed|Removed|Deprecated|Security`) and *bracketed* `## [Unreleased]`, and this
repo writes neither. Measured at `0002c9b0b`: **168 lines of release notes in 5 blocks, 1 flagged** —
the only one whose header began with the word "Added". Confirmed against last-green `22c74d7c3`:
**3** `###` blocks present, **zero** gate hits. So the gate had been printing a green checkmark over
misfiled release notes for as long as they had been there. Rule 3j: the defect is the **enumerator**,
not the branches. The one-line fix (rename the header to dodge the regex) was rejected — it satisfies
the enumerator while leaving exactly the content release-manager drops.

**Shipped as PR #758** (2 commits, now closed — see below): 5 blocks moved (167 of 168 lines verbatim,
asserted line-by-line with a firing negative control), detector widened to any `###` sub-section plus
any version/Unreleased `##` bracketed or not, an anti-vacuity floor (missing/empty `changelogs/` now
fails loudly — every other check in the script is an absence test that an empty destination satisfies
just as well as a clean index), and `scripts/test_check_changelog.sh` / `make check-changelog-selftest`
wired into CI because **the gate had no test at all** (control: `test_pin_root` is in `make/test.mk`,
so the instrument finds tests where they exist). `make check-changelog` rc=1 → rc=0, baselined red
first on a pristine `origin/dev` worktree. Self-test 11/11, non-vacuous by drill: against the pre-fix
detector **5 arms fail** and the four shapes live in the repo return `rc=0` **with the green
checkmark**. Mutant restored from a `cp` backup, sha256-verified.

**Gate 3b GREEN on the PR head `ef3b6a32c`**: 21 checks, pending=0, 0 not-green, 4/4 required
contexts, both changelog steps green including the new self-test. Three bounded poll windows; every
reading of `notgreen=0` over a non-zero `pending` was recorded as vacuously green, not as a verdict.

**Then the iteration's real lesson landed under me, and I stood down.** V1's iteration 217 had
independently preempted onto the same red and opened **#759 four minutes after #758** — same six
files, both adding `scripts/test_check_changelog.sh`, both moving the same 169 lines. Mid-poll V1
landed `c2022c7fa`, scoping "a red outranks the queue" to the mission that **owns** the repo; for
`sunholo-data/ailang` that is V1. #758 was CLOSED with the verdict (comment-then-close, comment
growth asserted 4→5). This is also right on merit, not just ownership: V1's detector
(`^#{2,6}[[:space:]]` minus the archive heading) is **strictly more general** than mine.

**Handed to #759 — two measurements it predates, both of which change its diff**: (1) the index has
**accreted two more stranded sections since the red** (`9504393d0`, `dfb19a551`), so `origin/dev` now
holds **7** blocks not 5, and #759 as written would move 5 of 7 and **strand a `### SECURITY` entry**
(the prelude `println` capability bypass); (2) the outage-vs-real split with its negative control, so
a lingering `lint`/`Build macos-latest` red is not mis-attributed to the changelog fix.

**Routing evidence**: controller=`claude:claude-opus-5` (session). Designer, planner, executor,
evaluator and quorum **never spawned** — deliberately: the work was a doc move plus a ~10-line shell
gate, which the skill routes as deterministic mechanical work; codex is quota-exhausted until Aug 20
and `pi` is metered. metered=**$0.00** of $5. No GPU, no `rig.lock`. `make quick-install` deliberately
NOT run (shared-write guardrail). Gates on **darwin/arm64 only** (rule 3b(viii)).

**Ruled out**:
- *"The failing `lint`/`Build` jobs are a regression from the docs commits"* — REFUTED by the
  `steps=1 / Set up job` signature, the incident window, and the same jobs going green on my PR head.
  Reverting the docs merges would have destroyed good work to appease an unrelated outage.
- *"Renaming the offending header fixes the red"* — true and rejected. It is the enumerator-satisfying
  non-fix; 4 of 5 blocks would have stayed invisible.
- *"V1's PR has no self-test"* — my own `grep -c` for arm markers returned **0** against a test file
  that demonstrably exists. Rule 3a on my own instrument: I read their file instead, and it carries 9
  arms plus its own instrument-failure floor. Had I banked that zero I would have argued to merge the
  weaker PR.
- *"Gate 2's open-PR check protects against duplicate work"* — it does not. It is point-in-time at
  pick time and aimed at a *past* iteration's abandoned work, so a peer that opens its PR later in the
  same window is invisible by construction (V1 checked ~18:58Z; #758 appeared 19:05Z).

**Retro lane — no skill edit.** The one friction worth a rule (two missions preempting one red with no
cross-mission mutex) was written into the shared skill and both charters by V1's `c2022c7fa` **while
this iteration was in flight**. Duplicating it would spend the one-edit budget for zero information.
Recorded here as the second first-party instance behind that rule rather than as a new one.

**Next**: queue row **6b** (triage-lite the 15 charter-orphaned issues) is the head and is untouched —
a hand-off is not a pick. **D-MOTOKO-FMT-1** remains the one OPEN decision (one word: precondition /
redesign), which unblocks item 6. Item 5 remains bounded until 2026-08-27.

## 10 — 2026-08-18 — auto-merge is not a landing mechanism: iteration 9's record sat BLOCKED on a red its base had already been fixed for

**Picked**: not the queue head, and not by preemption either. Gate 2's died-mid-flight check fired:
**PR [#760](https://github.com/sunholo-data/ailang/pull/760)** — iteration 9's entire mission record
— was OPEN, `MERGEABLE`/**`BLOCKED`**, with a stale worktree `.wt-motoko-iter9-record` corroborating
it. The charter's newest stamp read **iteration 8**, so a reader could not tell whether iteration 9
had run, crashed, or never fired. Per the rule, the deliverable is to **verify and land** it, not to
redo it. Standing rule 1 then allowed a second item, since recovery is bookkeeping: queue head **6b**.

**Reality check**: ran from the `#558` driver pin root, detached and clean at `c0dde65eb` ==
`origin/dev`; running skill byte-identical to origin (`cmp` rc=0). dev **verified green, not merely
un-red**: **16** exact-SHA checks, **0** not-green, `test: success` present in the set and a non-zero
run count. `gh` on `sunholo-voight-kampff`; billing tripwire **CLEAN**; kill switch armed. **0**
human directives on `#743` since the watermark `2026-08-17T05:48:45Z` (of 4 comments; the rest are
public feedback). Decision ledger valid, 3 rows, **1 OPEN** (`D-MOTOKO-FMT-1`, unanswered). Weekly
external-issue sweep **not due** — `#743` was created `2026-08-17T05:48:23Z`, i.e. **after** the
Monday-07:00 **local** boundary (= `05:00Z`), and holds 4 comments < 80, so no rotation either.
All four prior motoko worktrees clean; open loop-authored PRs `#695`/`#613` are V1's.

**THE FINDING — auto-merge waits on a check set that can never change, so a base-inherited red
freezes a PR forever, and the iteration that enabled it has already ended.** Iteration 9 closed by
enabling auto-merge on #760 (`enabledAt 2026-08-17T19:36:39Z`, SQUASH) rather than force-merging over
a red, and wrote in the PR body: *"It goes green once #759 lands."* Measured this iteration, and every
number is first-party:

- `#759` **merged at `2026-08-17T20:02:27Z`** — 26 minutes later — and its merge commit `cf56772bf`
  reads `test=success`, as does every commit after it up to HEAD. So the premise came true.
- #760's `updatedAt` is still **`2026-08-17T19:36:34Z`**, comments **0**, state `MERGEABLE`/`BLOCKED`,
  **12 h 13 m** after creation. Its `test` check is still `failure` on head `bf66f7655`, `run_attempt=1`.
- Mechanism: GitHub auto-merge merges when *the PR's own* required checks pass. It does **not** update
  the branch and does **not** re-run checks when the base advances, so a red inherited from the base is
  pinned to the PR's head SHA and outlives the fix. Nothing on either side re-evaluates it.

Consequence, which is the part worth the rule: this is a **silent** loss. There is no timeout, no red
to triage, no failed command — Gate 3b was never left in a state that could fail. The iteration ended
"successfully" and the mission's memory simply skipped a number. It is the vacuous-pass class the loop
keeps closing, aimed this time at the *landing* step: success reported for work that never shipped.
Two of motoko's last five iterations have now lost their record this way (6→7, 9→10) by two different
mechanisms.

**SECOND FINDING — iteration 9 attributed #760's red to the wrong step, and it was the step it had
just spent the whole iteration fixing.** The PR body says *"the `test` job's `Check changelog index
hygiene` step fails here for the base's reason"*, and baselines `make check-changelog` rc=2 at base and
with the diff applied. Measured on the actual job (`actions/jobs/95482959213`): `Check changelog index
hygiene` is step **17** and its conclusion is **`skipped`** — it never ran. The failing step is **14.
`Run stdlib .ail test suites`**. And on the base `714f1cecc` (`actions/jobs/95481571091`) the failing
step is **also 14**, identically. So the *verdict* — inherited, not caused by this docs-only diff — is
**correct**, and the *mechanism cited for it* is wrong: a real measurement was taken, of a different
gate. Rule 3d in its purest form; the red arrived in exactly the direction the iteration had been
looking all day, so nothing prompted reading the step list. Note the reading is only visible in
`actions/jobs/<id>`, not in `check-runs`, which reports the job and not its steps.

**Disposition — verified, then landed.** Rebased `docs/motoko-iter9-record` onto `origin/dev` (clean:
`git diff --stat 714f1cecc origin/dev` over the five mission files is **empty**, so no conflict was
possible), which produces a fresh check set on a green base. Iteration 9's own load-bearing claims
were re-derived rather than adopted (rule 3b(v)): `#758` CLOSED `2026-08-17T19:27:25Z`, `#759` MERGED
`20:02:27Z`, base `714f1cecc` `test=failure`, `cf56772bf` `test=success`. **One correction applied to
the inherited diff**: it wrote `design_docs/mission-dashboard.md`, the bare unnamespaced path that
V1's iteration 216 rule now forbids — that hunk was dropped (`git checkout origin/dev --` on that
path; `git diff origin/dev` on it is empty) and the snapshot went to
`design_docs/motoko-mission-dashboard.md` instead. The bare file still holds motoko's own stale
iteration-7 snapshot; per the rule it is **left alone**, not "fixed", and is flagged here as one line
of cleanup for a future iteration.

**Queue row 6b — closed as a measurement, and motoko owns none of it.** Full verdict in the charter
row. Headline: of the 15, **3 have closed** since the sweep (`#727`, `#708` on 08-18; `#696` on
08-17); **11 of the remaining 12 carry `ailang-message` + `from:<consumer>` labels** and are already
enumerated in V1's charter at `v1-mission.md:2104-2109`; the 12th, `#687`, is V1's **declared next
pick** in both its iteration 220 and 221 reports. There was therefore nothing to hand over — the
recipient charter already lists all twelve — and no verdict comment was posted, because a verdict
from the non-owning loop is noise on someone else's lane. Table taken over 8 files with firing
controls and an asserted array length.

**Routing evidence**: controller `claude:claude-opus-5` (session). Designer, planner, executor,
evaluator and quorum **never spawned** — the deliverable is a recovery-and-record plus a measured
queue disposition, so there is no doc to design, no plan to write and nothing to judge. Designer
rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no `rig.lock`;
`make quick-install` deliberately NOT run (shared-write guardrail). Gates ran on **darwin/arm64
only** (rule 3b(viii)).

**Ruled out**:
- *"#760 is blocked by the changelog gate its own iteration fixed"* — REFUTED above; that step is
  `skipped` on both the PR head and the base, and the real failure is `Run stdlib .ail test suites` on
  both.
- *"The sweep row poisons its own detector, so this is a Gate-0 defect"* — investigated and REJECTED.
  It is true that all 15 issues now grep to ≥1 because row 6b names them, and true that a re-run of the
  weekly sweep can never re-detect them. But that is the sweep *working*: they are tracked, by that
  row. The residual hazard is narrower — a row that CLOSES without action makes them invisible again —
  which is why 6b's close carries an ownership measurement rather than a bare tag. Not a skill edit.
- *"Auto-merge is unsafe and should be dropped"* — no. V1's log carries **76** auto-merge mentions and
  it lands PRs routinely; the failure is specific to a red inherited from the base, where the PR's own
  check set is frozen. The remedy is a rebase, not abandoning the mechanism.
- *"#695/#613 are the same shape"* — checked, and they are not: neither has auto-merge enabled
  (`#695` is `CONFLICTING/DIRTY`, `#613` is a deliberate DO-NOT-MERGE). So #760 is instance **1** of
  the auto-merge shape specifically, which is why the skill edit below is filed under the
  died-mid-flight class it extends rather than as a new rule of its own.

**Retro lane — ONE skill edit**, to Gate 3b: auto-merge is not a landing mechanism when the red is
base-inherited, because the base moving does not re-run the PR's checks. Two recorded frictions in the
class it extends (Gate 2's died-mid-flight rule, instances at iterations 121/148-149/160-161, plus
motoko 6→7), and this iteration supplies the new mechanism with a clean negative control.

**Next**: **D-MOTOKO-FMT-1** is the one OPEN decision (one word: *precondition* / *redesign*) and
unblocks item **6**. With 6b closed, the untagged queue head is item **7** (profile restoration
design, clause 4 — 5 profiles, 14 of 18 model entries). Item 5 remains bounded until 2026-08-27.

---

## 11 — 2026-08-18 — the queue's bounded wait had already expired: Arni answered five days ago, and the row that tracked him was re-asserting a measurement taken before his reply

**Picked**: queue item **5** (output-headroom upstream case), not the untagged head item 7 — because
Gate 2's blocker re-verification found item 5's declared blocker **dead**. The row reads
*"Remaining: the bounded wait only — still zero `arniwesth` events on #97; 2026-08-27 stands"*.
Measured this iteration: `arniwesth` commented on `#97` at **2026-08-13T18:45:54Z**, and the control
fires (`commenter:arniwesth` in that repo → **34** issues/PRs, so the instrument can see him).

**Reality check**: ran from the `#558` driver pin root, detached and clean at `1350d308f` ==
`origin/dev`; running skill **byte-identical to origin** (`cmp` rc=0, with a known-different control
at rc=1). dev **verified green, not merely un-red**: **16** exact-SHA checks, **0** not-green,
`test: completed/success` present in the set, `runs_total=2` non-zero. `gh` on
`sunholo-voight-kampff`; billing tripwire **CLEAN**; kill switch armed. **0** human directives on
`#743` since watermark `2026-08-17T05:48:45Z` (of 6 comments). Decision ledger valid, 3 rows, **1
OPEN** (`D-MOTOKO-FMT-1`, unanswered). Weekly sweep and rotation **both not due** — `#743` created
`2026-08-17T05:48:23Z`, after the Monday-07:00 **local** boundary (= `05:00Z`), 6 comments < 80.
Inbox 5 unread: 3 `eval-suite` telemetry, 2 sibling iteration reports (`mission-world` 92,
`mission-v1` 222) — neither carries a request or a bug against motoko, so neither outranks.
No died-mid-flight traces: all four prior motoko worktrees `dirty=0`; open loop-authored PRs
`#695`/`#613` are V1's; the stale `ailang-motoko` checkout's 7 modified paths re-verified as
**superseded, not orphaned** — 3 of 4 blobs exist verbatim in `de0e41099`, and the 4th is the
decision-ledger block already present in origin's charter.

**THE FINDING — a blocker row can re-assert a stale measurement as a fresh one, and nothing in the
row's own text distinguishes the two.** Iteration 3 measured "zero `arniwesth` events on #97" on
**2026-08-13** and that was true when taken. Iteration 4 (2026-08-14) then wrote *"still zero
`arniwesth` events on #97; 2026-08-27 stands"* — after his 08-13T18:45Z comment — and iterations
5–10 each carried the sentence forward. The word "still" is doing the work: it reads as a re-check
and is in fact a transcription (rule 3b(v)(b)). The cost is not abstract: the bounded rule iteration
3 wrote as a *remedy* for an unbounded wait became the thing that hid the wait ending, because a
timebox invites you to check the clock rather than the predicate. Five days of a nine-day window,
on a queue row sitting above two ungated items.

**What Arni actually said** (`#97`, `2026-08-13T18:45:54Z`), and it is better than the row assumed:
*"Agreed on all four points."* `#97` should close as superseded — calibration, the elision ladder and
system-prefix pinning are all covered more strongly by `main_dst`/`#154`, and nothing from that
branch should be revived. **But the output-headroom concern is still valid**, and *"Please go ahead
with the separate issue against `main_dst`."* He specified its scope: the effective input budget used
by **both** the pre-step compactor chain and the final payload seal, retaining the raw context limit
for telemetry, plus regression coverage for the 262,144 / 65,536 case **and** the unknown- and
small-limit fail-open behaviour. So the original precondition ("if Arni's reply invites it"), which
iteration 3 declared FALSE and replaced with the 08-27 timebox, is now **satisfied on its own terms**.

**Delivered — upstream issue [arniwesth/motoko_agent#165](https://github.com/arniwesth/motoko_agent/issues/165)**,
built from claims re-derived first-party at `main_dst@6c06b08` rather than inherited from the charter
(rule 3b(v)). The base is favourable and was checked, not assumed: `origin/main_dst` is **still exactly
`6c06b08`**, zero commits since our V27–V29 rows were taken, so the measurements are fresh by
construction.

- `src/core/session.ail:2561` — the seal is called with the **raw** `context_limit`.
- `src/core/session.ail:2556` — one line up, extensions get `context_limit - pinned_tokens`. That is
  the precedent, and it makes the seal the *only* consumer still handed the unreduced number.
- `src/core/phase_vocab.ail:145-155` + `src/core/compaction.ail:30` — the seal refuses only at
  `usage_percent_with_limit(payload_msgs, limit) >= exhaustion_pct()`, and `exhaustion_pct() -> int { 95 }`.
  The predicate is **input-only**.
- `packages/motoko-ext-compaction-structural/compaction_structural.ail` — the live hook
  `compact_for_pre_step` (`:170`) targets `result_target_pct() = elide_tier_pct() = 70` (`:30`, `:16`),
  but all four tiers route through `elide_walk` (`:97-113`), which rewrites a message **only** when
  `m.role == "tool"` (`:101`) and passes everything else through (`:112`). A large **user** message is
  invisible to every tier, and the hook then returns the unconditional
  `Compacted(floor, structural_note("floor", …))` (`:191`) — a payload it has already determined does
  not fit, with no error.
- `src/core/compaction.ail:25-28` — `usage_percent_with_limit` returns **0** when `limit == 0`, so an
  unknown model makes the seal compare `0 >= 95` and return `Ok` unconditionally. That is the
  fail-open arm Arni asked the follow-up to cover, and it is the reason a percentage-shaped reserve
  cannot inherit its own zero case.

**The R8 probe was RE-RUN, not quoted** (`tools/motoko/r8_headroom_band.ail` against a fresh
`~/dev/mk-r8-main-dst` worktree at `6c06b08`; `ailang lock` rewrote the 19 dev-container absolute
paths — `/workspaces` hits **0**, control `"path":` **19**). Every number reproduced:

| arm | history | ladder | seal | headroom |
|---|---|---|---|---|
| B control | small | `PassThrough` | `Ok` | 259,642 |
| **A** | one large **user** msg | `structural: tier=floor keep_last=1` at **79%** | **`Ok` — SENDS** | **54,905 < 65,536** |
| C control | ≈158% | `tier=floor keep_last=1` | `Err(SealExhausted)` at 157% | — |

Both controls fire, so arm A's `Ok` is a real permission and not a dead gate. Newly visible on the
re-run and carried into the issue: the ladder removed **2,061 of 208,980** segment tokens (~1%) in
arm A, and the extension saw `ext_limit = 261,824` (= 262,144 − 320 pinned) **while the seal saw the
raw 262,144** — the asymmetry stated as one line of output rather than as an argument.

**`#97` CLOSED with the verdict**, comment-then-close per the Gate-0 rule, comment count asserted
**2 → 3** (`gh issue close --comment` drops the body on an already-closed issue and exits 0; here the
issue was open, but the order costs nothing and the assertion is the point).

**RULED OUT**: *the charter's "zero production callers" claim for `try_emergency_compaction_with_limit`
is wrong* — it has exactly one caller, `compact_step_with_limit` (`:137`), so the bare claim is false
as written, but the substance holds and upstream says so itself: `006_compactor_strategy/PLAN-compactor-strategy.md:75`
records `compact_step_with_limit` as **test-only** with a single caller, and its non-test callers are
smoke/DST scripts, never the session. The issue says "off-path", citing their line, rather than
repeating our overstatement. Also ruled out: *an output-headroom issue already exists upstream* —
enumerated all **20** issues in that repo, none is one (the adjacent `#31`/`#49` are cited as related,
titles verified). And: *`main_dst` moved under our measurements* — `git log 6c06b08..origin/main_dst`
is **0** commits.

**Routing evidence**: controller `claude:claude-opus-5` only. **No designer, planner, executor,
evaluator or quorum spawned** — the deliverable is an upstream issue assembled entirely from
first-party measurements the controller took itself; there is no doc to design, no plan to write and
nothing to judge. Quorum-at-pick N/A: item 5 has no design doc of its own, its case living in the
migration doc's V27–V29 rows and the disposition ledger's R8, both already quorum-reviewed. Designer
rotation pointer untouched at `claude:claude-fable-5`; codex quota-dry until Aug 20 and `pi:deepseek`
is 3-for-3 zero-byte across the fleet per `mission-world` iter 92. Metered **$0.00** of $5. No GPU, no
`rig.lock`. `make quick-install` deliberately NOT run (shared-write guardrail, V20) — the PATH binary
is `v0.33.1-103-g0002c9b0b-dirty`, stated because the R8 re-run used it. Gates ran on **darwin/arm64
only** (rule 3b(viii)).

**Next**: item **7** (profile restoration design) is the queue head, ungated. `#165` is Arni's to
triage — nothing local waits on it.

---

## 12 — 2026-08-18 — the issue we filed six hours ago was already anchored to a base that had moved 59 commits, and upstream had just cut the branch to work it

**Picked**: not the untagged queue head (item **7**). Gate 1's external-predicate re-read — the rule
iteration 11 added to this very skill — surfaced that queue row 5's freshly-written claim
*"`origin/main_dst` is **still exactly `6c06b08`**, **0** commits since our V27–V29 rows"* was
**FALSE**, measured `git rev-list --count 6c06b08..8110ffc` = **59**. The row was six hours old. The
deliverable is the re-anchor of the artifact that claim underwrites, upstream issue
[**#165**](https://github.com/arniwesth/motoko_agent/issues/165), before its reader diffs against a
base it no longer describes.

**Reality check**: ran from the `#558` driver pin root, detached and clean at `9467ef183`; running
skill **byte-identical to `origin/dev`** (`cmp` rc=0). Local pin root was **4** behind
`origin/dev` (`d22681e27`), so all mission state was read from origin blobs, and the record is
written in a worktree branched from `origin/dev` — never in the shared checkout. `origin/dev` last
**settled** commit `9467ef183` is green: **16** exact-SHA checks, **0** not-green; the current tip
`d22681e27` is mid-flight (**15** checks, all `pending`) and is therefore recorded as *unverified,
not green* — no verdict taken from an in-flight run. `5f9f4ba4f` returns `checks=0` because it is an
intra-push commit and only a push tip gets runs, which is the documented zero, not an anomaly. dev is
V1's to own regardless (2026-08-17 scoping rule), so no red here could have displaced this pick.

**Died-mid-flight sweep**: five motoko worktrees, `git status --porcelain` **0** in all five; the two
open loop-authored PRs (`#695` `CONFLICTING`, `#613` `[DO NOT MERGE — awaiting D-1]`) are V1's, as
the last three iterations also measured; the stale `~/dev/sunholo-data/ailang-motoko` source clone
carries the same **7** paths iteration 11 already adjudicated *superseded, not orphaned*, unchanged.

**THE FINDING — a LANDED row's deliverable can rot against a base nobody re-reads, and every
freshness rule in this skill is scoped to rows you are about to PICK.** Iteration 11 wrote the rule
that catches a *blocked* row re-asserting a stale measurement. This is the mirror: the row is
**LANDED**, so by construction no gate revisits it — and its deliverable is not a local file but an
artifact **published to an external party**, pinned to that party's moving branch. `main_dst` gained
**59** commits (**100** files) between our filing at `12:46:42Z` and this iteration, `src/core/session.ail`
among them (`+67/-13`), and the maintainer cut `arniwesth/mot-100-fix-output-headroom` in the same
window. Nothing in the loop would have looked; the queue row reads LANDED and the issue reads filed.

**The defect is INTACT — only the offsets moved, and that is a measurement, not a hope.** The ten-line
window `src/core/session.ail:2552-2561@6c06b08` is **byte-identical** to `:2606-2615@8110ffc`,
sha256 `c53792eb5b778cf32e72001006e485274b05f33d13f14cbe578c836e9e15f1dc`. **Negative control**: the
same window shifted by one line hashes `01d10306…`, so the match is a real match rather than a hash of
nothing. So all three `session.ail` citations in `#165` shift **+54**: `2552→2606` (the
`context_limit` binding), `2556→2610` (`ext_context_limit = context_limit - pinned_tokens`),
`2561→2615` (`seal_compacted_payload(…, context_limit, …)`). The asymmetry the issue is about —
extensions get the reduced limit, the seal gets the raw one, five lines apart — survives verbatim.

**The other cited files are byte-unchanged across the range, with a same-call control.**
`git diff --stat 6c06b08 8110ffc -- src/core/phase_vocab.ail src/core/compaction.ail
src/core/compaction_structural.ail src/core/context_usage.ail src/core/session.ail` returns
**`session.ail` only** — so the instrument fires (rule 3a(i-d): the control is in the same scope as
the check, not a different path) and the other four are genuinely untouched. `compaction.ail:25-28`,
`:30`; `compaction_structural.ail:97-113`, `:170`, `:191`; `phase_vocab.ail:145-155` therefore stand
as written.

**Delivered**: a correction comment on
[**#165**](https://github.com/arniwesth/motoko_agent/issues/165#issuecomment-5332596902) carrying the
old→new citation table, the block sha256 with its negative control, the same-call unchanged-files
control, and the note that `arniwesth/mot-100-fix-output-headroom` currently points at `ffd6256` —
an **ancestor** of `main_dst@8110ffc` with **zero** commits of its own — so branching from the tip
saves a rebase. Posted `--body-file` (never inline `--body`; markdown is made of backticks), and
**delivery asserted by comment count 0 → 1**, because a reporting command's exit code describes the
request, not the delivery.

**Phase-0 predicates re-run, not transcribed — all still FALSE, so rows 10/11/12 stay parked.**
**G1**: `#154` `state=OPEN`, `mergedAt=-` (control: `#161`/`#162` in the same repo return
`state=MERGED` with non-null `mergedAt`, so the instrument can see a merge) — but note `#154`
`updatedAt=2026-08-18T16:48:58Z`, i.e. it is actively moving, which is what put `main_dst` 59 ahead.
**G2**: predicate `git -C <upstream> cat-file -e origin/main:packages/motoko-ext-abi/ailang.toml`
rc=**128**, mandatory control `…:README.md` rc=**0** → FALSE for the right reason (V20/V25).
**G3**: registry `latest=2.2.0`, `versions=1.0.0,2.0.0,2.1.0,2.2.0` → no 5.x → FALSE.
**G4**: unrunnable while G3 is FALSE. **G5** (Arni's ABI-settled declaration) unchanged.
**D-MOTOKO-FMT-1** remains the sole OPEN ledger row; `scripts/mission_directives.sh` returned **0**
directives from `MarkEdmondson1234` on `#743` since the `2026-08-17T05:48:45Z` watermark, of 8
comments.

**RULED OUT**: *upstream already fixed this* — `session.ail:51` now imports `resolve_context_limit`
from `src/core/context_usage`, which looks exactly like the fix; `context_usage.ail` is **byte-unchanged**
across `6c06b08..8110ffc` and resolves the context limit rather than reserving output headroom, and
the seal at `:2615` still receives the raw value. This is the solved-upstream class the Gate-2
iteration-145 rule names, and it came back negative on measurement rather than on reading.
*The four merge commits in the range moved `session.ail`* — **refuted before posting**, and the draft
comment had already asserted it: `git log --oneline 6c06b08..8110ffc -- src/core/session.ail` returns
exactly **two** commits, `b1ad13b` and `a45f708`, both `#160` work, and they are the whole `+54`. That
is rule 3b(v)(b) caught on this controller's own prose — an inference from an adjacent list, written
as if measured. *The fix branch has work on it* — `arniwesth/mot-100-fix-output-headroom` is an
**ancestor** of `main_dst`, `git log main_dst..branch` empty, so it is a freshly cut branch and not a
patch to read.

**Routing evidence**: controller `claude:claude-opus-5` only. **No designer, planner, executor,
evaluator or quorum spawned** — a base-freshness re-measurement of a filed artifact has no doc to
design, no plan to write and nothing to judge; quorum-at-pick N/A (no design doc is in play).
Designer rotation pointer **untouched** at `claude:claude-fable-5`. Metered **$0.00** of the $5
ceiling. No GPU, no `rig.lock`. `make quick-install` deliberately NOT run (shared-write guardrail,
V20) — no local AILANG behaviour was under test; every measurement here is `git`/`gh`/`shasum` over
two checkouts. Gates ran on **darwin/arm64 only** (rule 3b(viii)).

**Gate 5 — no skill edit.** The gap is real and narrowly stated: *every freshness rule in this file
is scoped to a row you are about to pick, and none of them covers an artifact a LANDED row has
already published to a third party whose base moves.* That is **instance 1**, pre-registered here at
the skill's own ≥2-friction bar (the precedent is iteration 140 pre-registering rule 3d). Iteration
11's rule and this one are the same family but not the same gap — its trigger is *blocked*, mine is
*landed and published* — so folding this into it now would be a rule written on one datapoint.
Queue row 5 is corrected in the charter instead (process lane), which is where a stale measurement
belongs.

**Next**: item **7** (profile restoration design) is the queue head, ungated, and is the pick for
iteration 13 unless a predicate flips. `#165` is Arni's to triage; nothing local waits on it.

## 13 — 2026-08-19 — Mark's ruling un-parked item 6; the trace it demanded answers the reviewer both ways, then finds the fix it authorises cannot be written where everyone assumed

**Picked**: not the untagged queue head (item **7**). `D-MOTOKO-FMT-1` was RESOLVED **precondition**
by Mark in an attended session dated **2026-08-19**, recorded in `1a3ca2d5f` — and under Gate 0's
decision-recording contract an answer to a parked item unparks it and becomes this iteration's pick.
The ruling is specific: *"the sprint TRACES motoko's resolved runtime provider first, then changes
the preflight. Do not redesign around the unknown: the objection is precisely that nobody has
measured which provider actually serves the ollama-declared lanes, so measure it."*

**Reality check**: ran from the `#558` driver pin root, detached and clean at `c29c48e96`, which is
exactly `origin/dev`; running skill **byte-identical to origin** (`cmp` rc=0). `origin/dev` is
**verified green, not merely un-red** — **16** exact-SHA checks with **0** not-green, and
`actions/runs?head_sha=<full 40>` returns `runs_total=2`, so a run exists rather than the commit
being unverified. dev is V1's to own regardless (2026-08-17 scoping rule). Charter and log were
byte-identical to `origin/dev` before the first write (`git diff --stat origin/dev --` empty), the
previous-iteration tell fired (`grep -ci "ITERATION 12"` → **3**, control `ITERATION 11` → **4**),
and the record is written in a worktree branched from `origin/dev`, never in the shared checkout.
**Died-mid-flight sweep**: iteration 12 left no worktree and no PR; `#695`/`#613` are V1's, as the
last four iterations also measured. **0** directives from `MarkEdmondson1234` on `#743` since the
`2026-08-17T05:48:45Z` watermark (of 10 comments) — the ruling arrived through the attended-session
channel, not the issue, which is why the directive script's zero is correct and not a miss.

**THE TRACE — run via the fork-resolution-path arm the ruling names as sufficient (`and/or`), no GPU
and no `rig.lock`.** The question was required to discriminate: does removing the unconditional
`OPENROUTER_API_KEY` refusal at `internal/executor/motoko/healthcheck.go:64` delete a real fail-fast,
or admit a silent OpenRouter fallback for `provider: "ollama"`, `env_var: ""` entries?

**Answer: BOTH, of DIFFERENT lanes — so the remedy is a CONDITION, never a deletion.**

**(a) For the two fmt arms no OpenRouter routing is reachable, so the preflight is a FALSE
fail-fast.** Precedence in the fork is `process.env.MODEL ?? profileAgent.model ??
"anthropic/claude-sonnet-4-6"` (`mk-ast/src/tui/src/index.ts:768-771`). Tier 1 is set
unconditionally by the Go executor (`motoko.go:343`) from `agent_model_name`
(`models.go:493-494`) → `ollama/qwen3.6:35b-a3b-mxfp8`; tier 2 is the profile's own
`ollama/qwen3.5:35b-a3b-mxfp8`, and **both** `ollama` and `ollama_fmt` profiles additionally pin
`agent.openai_base_url = "http://localhost:11434/v1"`. Measured: `GuessProvider` returns `ollama`
with env var `""` for both, matched at `internal/ai/config.go:55-57` **before** the generic
`vendor/model → OpenRouter` rule at `:67-73`, under an in-code comment demanding exactly that
ordering. Motoko's own preflight asks for nothing — `required_secret_for_model`
(`mk-ast/src/core/supervisor.ail:21-26`) matches only `anthropic/`, `openrouter/`, `openai/` and
`google/`, returning `""` for `ollama/`.

**The measurement discriminates rather than passing vacuously.** The same call returned
`openrouter` / `OPENROUTER_API_KEY` for all three fallback defaults —
`anthropic/claude-sonnet-4-6` (`index.ts:771`), `openrouter/auto` (`config.ail:434`) and
`openrouter/anthropic/claude-haiku-4-5` (`factory.go:71`) — so both arms are present and an
`ollama`/`""` reading is a real answer, not an instrument that cannot say otherwise.

**(b) For the OpenRouter-routed motoko lanes the preflight IS a real fail-fast, and the only one.**
Motoko's `validate_secrets` result flows to `emit_warnings`, which prints `{"type":"warning",…}`
and `main` then proceeds to `run_with_config` (`supervisor.ail:11-19`, `:42-51`). Motoko warns and
runs; it never refuses. Deleting the Go preflight outright would let an OpenRouter-lane eval start
and burn wall-clock before failing at the provider.

**THE FINDING THAT RE-SHAPES D1, and no reading of `healthcheck.go` could have produced it: the
condition is not expressible where the check sits.** `HealthCheck(ctx context.Context) error` takes
**no task** — it is the shared `executor.Executor` interface method (`internal/executor/executor.go:31`)
— so the only model it can read is `e.model`. And `cfg.MotokoModel` is **never set from
`models.yml`**: the whole non-test Go tree has **3** `MotokoModel` hits — a field declaration, the
hardcoded `"openrouter/anthropic/claude-haiku-4-5"` at `internal/executor/factory.go:71`, and
`motoko.go:145` reading it — against a control of **11** `MotokoProfile` hits, so the grep finds
wiring where wiring exists. The lane's real model arrives per-task as `task.Model`
(`cmd/ailang/eval_benchmark_agent.go:174,195,253,389`) and is consumed by `getModel(task)`
(`motoko.go:610-620`) at Execute time — **after** the health check has already refused.

So an `if` added at `healthcheck.go:64` would evaluate the OpenRouter default for **every** lane,
conclude "OpenRouter", and refuse the ollama arms exactly as today. **D1 is a plumbing change, not
the ~0.5-day one-liner §6 prices it as.** Three options are costed in the doc's new §12.2: set
`cfg.MotokoModel` from `models.yml` at construction (smallest true fix, and it also corrects
`e.model` being wrong for every motoko lane today); move the check into `Execute`; or widen the
interface (6+ implementors, and the trace gives no reason to prefer it). The recommendation is to
express the condition on the **resolved provider** — `ai.EnvVarForProvider(ai.GuessProvider(model))`
returns the required variable *or `""`* for every provider (`internal/ai/config.go:104-119`) —
rather than hardcoding one vendor; `internal/executor/motoko` does not import `internal/ai` today,
so the boundary must be checked before assuming that is free.

**Ordering note, free from the same read.** The preflight returns **before** the `motoko --version`
query that discovers `e.motokoRepo` (`healthcheck.go:70-77`), and `MOTOKO_REPO` is what stops the
profile silently degrading to `extensions.order=[]` — the defect `motoko.go:344-364` records as
**39 of 39** eval sessions with `loaded_extensions=[]`. Moot while the check refuses outright;
load-bearing the moment it becomes conditional, because a degraded profile drops the `fmt`
extension that IS the treatment.

**No key-absence-driven fallback exists anywhere.** Repo-wide non-test Go reads
`OPENROUTER_API_KEY` at exactly **4** sites: `cmd/ailang/ai_handlers.go:276` and
`cmd/ailang/exec.go:475` (both explicit `openrouter` subcommands), `internal/ai/config.go:115,143`
(keyed by an ALREADY-resolved provider), and the motoko preflight. None re-routes on absence.
Control: **23** `OPENAI_API_KEY` sites, so the grep sees this class of hit.

**What the trace does NOT settle, moved to acceptance rather than claimed.** The static arm proves
no OpenRouter routing is *reachable*; it does not prove no connection is *made*. The live arm is
circular today — the fmt lane cannot run while the preflight refuses it — so it becomes a D1
acceptance criterion, `AC-D1-live`: one fmt-lane run reaches `localhost:11434` and makes **zero**
`openrouter.ai` connections, asserted **on the connection** rather than on the absence of an error
(an absence is satisfied equally by "no OpenRouter call" and "the run never started"), paired with
an OpenRouter-lane known-positive control in the same sweep.

**Delivered**: `design_docs/planned/m-motoko-fmt-remeasurement-instrument.md` un-parked (status
header rewritten), **O4 CLOSED** with its disposition rewritten from `PARKED — needs-human-review`,
a new **§12** carrying the trace, the three costed options, the ordering note and `AC-D1-live`, and
verification rows **V25–V32**. Charter queue row 6 re-tagged and the `D-MOTOKO-FMT-1` ledger row's
evidence column marked DISCHARGED. Ledger re-validated: **3** rows, **0 OPEN**.

**Phase-0 predicates re-run, not transcribed — all still FALSE, so rows 10/11/12 stay parked.**
**G1**: `#154` `state=OPEN`, `mergedAt=-`; control `#161`/`#162` in the same repo return `MERGED`
with non-null `mergedAt`, so the instrument can see a merge. **G2**: predicate
`git -C <upstream> cat-file -e origin/main:packages/motoko-ext-abi/ailang.toml` rc=**128** with the
mandatory `README.md` control rc=**0** → FALSE for the right reason. **G3**: registry
`latest=2.2.0`, `versions=1.0.0,2.0.0,2.1.0,2.2.0` → no 5.x. **G4** unrunnable while G3 is FALSE.
**G5** unchanged.

**Upstream is acting on our issue.** `arniwesth/mot-100-fix-output-headroom` moved
`ffd6256 → 2f61665` since iteration 12 and now carries **7** commits ahead of `main_dst`, including
`da999ac fix(compaction): reserve provider output headroom` and their PR **#166**. `#165` is OPEN
with **2** comments. Iteration 12's re-anchor of the line citations landed in time to be useful,
which is the first evidence that the re-anchor rule it pre-registered actually pays.

**RULED OUT**: *the `models.yml:1854` comment "`MODEL env > profile config model (index.ts:580)`" is
current* — the **behaviour** it asserts is TRUE, re-derived first-party at `index.ts:768-771`, but
its **citation is stale**: line 580 is now `native_tool_results` event printing. That is rule
3b(v)(b) caught in the repo's own comments rather than in a controller's prose, and it is why the
precedence was re-measured instead of inherited. *The silent-OpenRouter fallback the reviewer feared
is live today* — it is **real but unreachable on the wired path**: tier 3
`anthropic/claude-sonnet-4-6` does resolve to OpenRouter (measured), and it fires only if `MODEL` is
empty AND the profile fails to resolve, which the unconditional `MODEL=` at `motoko.go:343`
forecloses. Recorded as a residual with its trigger named, not as a live defect. *A live
`rig.lock` run was needed to answer O4* — the ruling itself says *"and/or"*, and the fork-resolution
arm settles the discriminating question while the live arm is circular until D1 lands; claiming it
had been run would have been the vacuous pass this mission keeps closing.

**Routing evidence**: controller `claude:claude-opus-5` only. **No designer, planner, executor,
evaluator or quorum spawned** — a controller-run trace that answers a standing reviewer objection has
no doc to design, no plan to write and nothing to judge, and the doc's own quorum is spent (2 rounds,
both external reviewers present, `absent_reviewers` empty in both artifacts, metered $0.1424 at
iteration 8). Designer rotation pointer **untouched** at `claude:claude-fable-5`. Metered **$0.00**
of the $5 ceiling. No GPU, no `rig.lock`. `make quick-install` deliberately NOT run (shared-write
guardrail, V20); the one build step taken was a throwaway `go test ./internal/ai/` whose test file
was deleted immediately and the tree re-asserted clean (`git status --porcelain` empty). Gates ran on
**darwin/arm64 only** (rule 3b(viii)).

**Gate 5 — no skill edit.** The iteration's friction is real and narrowly stated: *rule 3b(v)(b)
(a transcribed identifier is not a measurement) is written for controller and document prose, and
nothing points it at the CODEBASE'S OWN COMMENTS* — `models.yml:1854` carries a `file:line` citation
for a behaviour that is still true at a different line, so the comment reads as verified and its
pointer is stale. That is **instance 1**, pre-registered here at the skill's own ≥2-friction bar
(the precedent is iteration 140 pre-registering rule 3d, and iteration 12 pre-registering the
published-artifact freshness gap). The correction went into the doc's V28 row instead.

**Next**: item 6 is now a normal sprint — planner → executor on D1 (with §12.2's plumbing decision
made explicitly) + D1b + D2. Item 7 (profile restoration design) is the untagged head behind it.

---

## 14 — 2026-08-20 — D1 landed; the guard was pinned and its wiring was not, and the mutation that would have shown it survived the first delivery

**Picked**: queue item **6**, `m-motoko-fmt-remeasurement-instrument`, un-parked at iteration 13 by
Mark's `D-MOTOKO-FMT-1` ruling. Item 6 sits above the untagged head (item 7) because iteration 13's
trace discharged its precondition and left it *"now a normal sprint (planner → executor)"*.

**Reality check**: ran from the `#558` driver pin root, detached and clean at `44aa3cab4`, which is
exactly `origin/dev`; running skill **byte-identical to origin** (`cmp` rc=0). dev verified green,
not merely un-red — **16** exact-SHA checks, **0** not-green. Ledger valid at 3 rows, **0 OPEN**.
**0** human directives on `#743` since the watermark, of **12** comments. Inbox: 2 unread, both
sibling reports (`mission-world` 97, `mission-v1` 231) — neither carries a request or a bug against
motoko, so under Gate 0 they are read and acked, never obeyed. Weekly sweep and rotation both not
due (`#743` created `2026-08-17T05:48:23Z` = 07:48 local, *after* the Monday-07:00 local boundary;
12 comments < 80). Died-mid-flight check: `#792` is V1's iteration 232, `#695`/`#613` are V1's, as
the last five iterations also measured.

**External-predicate re-read** (the rule iteration 11 added, aimed at rows nobody picks): G1
`gh pr view 154` → `state=OPEN`, `mergedAt=-`, control `#161`/`#162` → `MERGED` with non-null
`mergedAt`; G2 predicate rc=**128** with the mandatory `README.md` control rc=**0** → FALSE for the
right reason; G3 registry `latest=2.2.0`, versions `1.0.0,2.0.0,2.1.0,2.2.0` → no 5.x; G4 unrunnable
while G3 is FALSE; G5 unchanged. Rows 10/11/12 stay parked. Upstream `main_dst` still `8110ffc`, so
row 5's re-anchored `#165` citations remain valid; upstream's own fix PR `#166` is OPEN.

**Delivered**: **M1 (D1) of a 5-milestone sprint** — PR
[#794](https://github.com/sunholo-data/ailang/pull/794) → `bc0b5a8d4`. M2–M5 named as the resume
point rather than silently compressed; M3 specifically was NOT shipped alone because its `AC-M3-4`
depends on M4's order-integrity checker, and shipping it would have left a milestone with an
unclosable acceptance criterion.

**The planner refuted the design doc.** §12.2 preferred option (1) — set `cfg.MotokoModel` from
models.yml at construction. Unsound, on two independent measurements I re-verified first-party:
`ExecutorFactory.GetExecutor` caches by executor NAME (`factory.go:96-122`), so ONE
`*MotokoExecutor` serves all **17** `agent_cli: "motoko"` lanes — **7** ollama, **10** requiring a
credential (control: **89** total `agent_cli` lines) — and one string cannot be right for both; and
`HealthCheck` runs under **`sync.Once`** (`motoko.go:134`, `healthcheck.go:40`), so a model-dependent
verdict there freezes at whichever model canaried first, turning a loud uniform refusal into a
silent order-dependent one. Chosen instead: the refusal moves to `ExecuteStreaming`, keyed on
`e.getModel(task)`, expressed on the resolved provider and strictly downstream of repo discovery.

**THE FINDING — a guard is not a gate until something reds when you remove it, and the thing nobody
removed was the wiring.** Neutering the call site (`if err := error(nil); err != nil {`) left the
**entire package green** as the executor delivered it: every arm called `requireProviderCredential`
directly, and the single `Execute`-level arm (`T-ORDER`) drives an `ollama/…` task the guard
**ADMITS**, so its observable could not move no matter how the wiring broke. That is this repo's own
named recurring shape — *guard the helper, miss the call site* — reproduced **inside the milestone
whose entire subject is a guard**. Per rule 3i(c) the ROW was repaired, not the code: `T-CALLSITE`
asserts a refusal that reaches `Execute`, on the resolved provider name and the missing variable,
plus a mock-not-invoked marker whose absence is admissible only because a control arm in the same
test proves the marker fires for an admitted lane. MUT-4 now dies with `T-CALLSITE` as **sole
killer** (`-skip` inverse → rc=0).

**Honest full-package mutation table** (every mutant asserted LANDED by sha256 and BUILDING):

| mutant | failing tests, full package |
|---|---|
| neuter the no-credential ADMIT branch | **12** |
| neuter the credential-missing REFUSAL | **3** |
| neuter the unresolvable-model guard | **1** (sole killer) |
| neuter the `ExecuteStreaming` call site | **1** (sole killer; SURVIVED as first delivered) |
| move the check into `HealthCheck` *before* the version query | 1 — `T-ORDER` at `execute_test.go:711` |
| move it into `HealthCheck` *after* the version query | 1 — `T-ORDER` at `execute_test.go:718`, a *different* assertion |

**Evaluator** (sonnet; distinct provider from the pi/DeepSeek executor, so generator≠judge holds):
**PASS 84/100, zero blocking**, three non-blocking findings, all reproduced before being acted on —
and the third came back **worse** than filed. (a) `T-ORDER` was **decorative**: its surviving
assertion was a bare nil-vs-error on `HealthCheck`, which fires identically whether the check
precedes the version query (violating §12.3) or follows it (not violating it), and the `MOTOKO_REPO`
observable the plan calls the pin was never reached under either mutant. Repaired so the ordering
claim rests on `e.motokoRepo` read independently of the returned error, then **proven to
discriminate** by running both placements — they now fail at different assertions. (b) The same
false claim survived one file over: `internal/executor/motoko/README.md` still read *"motoko routes
ALL models via OpenRouter"*, false for 7 of 17 lanes — CLAUDE.md rule 3 applied to the code and not
to the docs. (c) The mutation table in my own commit message was stale.

**MY OWN ERROR, and it is the iteration's Gate-5 material.** My first published table reported the
neuter-ADMIT mutant as killing **1** test. The evaluator measured ≥2. The full-package re-measure is
**12**. Cause: those per-mutant runs were scoped `-run 'TestRequireProviderCredential'`, and I
dropped the narrowing when I quoted the number — a `-run`-narrowed result cited for a whole-package
sentence. That is **rule 3b(ii), which this skill already has**; it was broken, not missing, so it
belongs here rather than in the rulebook.

**Gate 3b caught two Windows-only defects no local command could** (rule 3b(viii)); the base arm was
**green** on windows, so the red was ours, not inherited. `%q` **escapes backslashes**, so
`strings.Contains(msg, dirPath)` can never match a windows path the message names correctly — now
compares `strconv.Quote(dirPath)`. And a `#!/bin/bash` mock named `motoko` with no `.exe` is not
executable there, so the version query never fired — skipped on windows, matching the **10** existing
bash-mock skips (control), with the narrowing declared: T-B4's version-query half is darwin/posix-only.

**Landing**: the PR first read `CONFLICTING`/`DIRTY`. `mergeable` was checked **FIRST**, per the
iteration-198 rule, so the missing `pull_request` runs were explained in one call by the boring cause
instead of by a dropped-event hypothesis; the overlap was exactly **one** file
(`changelogs/v0.18-current.md`, from V1's iteration 232). Rebase + force-push produced **5** runs
within seconds. Final PR head `fb1ef41b5`: **21** checks, **0** not-green, **4/4** required contexts
pass, `mergeStateStatus=CLEAN` — observed, then squash-merged. No auto-merge was armed (iteration
10's rule).

**LANE DEFECT, found and fixed: the pi executor lane had never been runnable from this mission's
checkout.** The mandated sandbox extension failed to load —
`Cannot find module '@anthropic-ai/sandbox-runtime'` — because
`tools/pi-extensions/sandbox/node_modules` exists **only in V1's checkout**. The extension sources
are byte-identical across checkouts (`cmp` rc=0, with a rc≠0 control proving `cmp` discriminates), so
this was a missing `npm ci`, not drift. Installed in the pin root; `node_modules/` is gitignored by
that directory's own `.gitignore`, and the tree was verified clean afterwards. The skill's pi recipe
says `$REPO` is "the mission's checkout" and gives path resolution as the reason — it does not say
the dependency must be installed there, which is why this sat undiscovered until the first pi run
from this mission.

**Ruled out**
- *The first pi run's `agent_end=0` was a lane failure of the documented iteration-173 shape.* It was
  **mine**: I backgrounded the runner inside an already-backgrounded Bash call, so the harness reaped
  it. 425 turns, all `stopReason:"stop"`, **zero** `"length"`, empty stderr — an external kill, not
  an output-cap stall. Misattributing it would have burned the lane on my own harness misuse.
- *The 300 MB NDJSON kill was a runaway turn.* Growth had already fallen from ~1.4 MB/s to ~93 KB/s
  by 268 MB, and the executor had written `.snap/`, the changelog and cleaned its scratch dirs before
  the cap fired — it was killed during its final *report* turn, so the work was complete. The cap was
  my own guard against disk exhaustion, and disk was measured at **541 GB** free, so the guard was
  mis-sized for this run rather than the run being pathological.
- *`go build ./...` is a usable gate.* It is **RED at base** (`cmd/wasm` has no native `main`; it
  builds only under `GOOS=js`). I put it in the planner directive; the planner refuted it and
  substituted `go build ./internal/... ./cmd/ailang/...`, green at base. Rule 3e, caught by the role
  I handed the error to.

**Routing evidence**

| role | pinned | actually ran | notes |
|---|---|---|---|
| controller | `$MODEL` | `claude:claude-opus-5` | triage/pick/verify/record |
| designer | rotation | **not spawned** | quorum spent (2 rounds, both reviewers present, `absent_reviewers` empty); §12 is a measurement, not a new design. Pointer untouched at `claude:claude-fable-5` |
| planner | `opus` | **opus** | `derive-planner-lane.sh` → `opus fail-closed:env-pin`, used verbatim; no codex probe |
| executor | `pi:openrouter/deepseek/deepseek-v4-flash-0731` | same | probe rc=0; run killed at my 300 MB cap after the work was complete; **metered $0.2325** |
| evaluator | `sonnet` | **sonnet** | Anthropic ≠ the executor's OpenRouter/DeepSeek → generator≠judge holds |

**Metered**: **$0.2326** of the $5 ceiling (pi executor $0.2325 + probe $0.0001). Quota buckets:
opus (controller, planner), sonnet (evaluator). No GPU, no `rig.lock`.

**Platform**: every local gate ran on **darwin/arm64** only; the windows and ubuntu legs are known
only through Gate 3b, which is exactly how the two windows defects above were found.

**Next**: **M2–M5** of the same plan. M2 (`AC-D1-live`) is now unblocked in principle — its
circularity is broken, since the preflight no longer refuses the fmt lane — but it needs the rig, and
doc §6's **deployment precondition** still stands: merging to `origin/dev` does not put D1 on the
rig, because the installed plist runs `nightly-eval.sh` in place from V1's checkout (open issue
`#558`). Then items 7 and 8.

**Gate 5**: **no skill edit.** The iteration's own friction — quoting a `-run`-narrowed blast radius
for a whole-package sentence — is a rule the skill already carries (3b(ii)). A rule broken is not a
rule missing, and adding a louder restatement would be the documentation-bias failure the skill's own
iteration-198 note warns about.

## 15 — 2026-08-20 — M4 landed, and the integrity gate the milestone exists to build was unreachable from its own CLI — twice, in opposite directions

**Pick.** Queue item **6**'s named resume point, milestone **M4** (D2 censored-pair analyzer) —
not M2, M3 or M5. Reason, stated because it is an ordering claim: the plan declares M4
*parallel-safe with M1–M3*; it is the only remaining milestone with **no rig dependency** (M2's
`AC-D1-live` needs ollama + a metered OpenRouter control leg); and **M3's AC-M3-4 closes by calling
M4's order-integrity checker on synthesised rows**, so M4-first is the ordering that leaves no
milestone with an unclosable acceptance criterion. M2, M3 and M5 remain the resume point.

**Outcome.** PR [#806](https://github.com/sunholo-data/ailang/pull/806) → `d5bcfa0c8`. Gate 3b
GREEN on head `922190dd3`: **21** checks, **0** pending, **0** not-green, **4/4** required contexts
pass, `mergeStateStatus=CLEAN`, `test-windows` and `Build windows-latest` both `success`.
Evaluator **FAIL 80/100**, one BLOCKING finding — against the controller's own repair — plus six
non-blocking; blocking fixed and three non-blocking closed before landing.

### The finding, and it is the same shape as last iteration's

`ailang eval-censored-pairs` is a **sibling** of `eval-paired`, never an extension:
`cmd/ailang/eval_paired.go` is byte-identical to `origin/dev` (`cmp` rc=0, with a control
confirming a file we DID change differs), because its stdout JSON has two live callers
(`nightly-eval.sh:339` microRAG, `:515` fmt).

The command loaded its arms through `LoadArmForPairing`, which wraps `FilterValidResults` and drops
rows whose `Validity` is invalid **at load time**. The `>20% of ON rows quarantined → VOID` gate
counts exactly those rows — so through the CLI it could never fire. The analyzer's unit tests passed
throughout, because they construct `[]*BenchmarkResult` directly and bypass the loader. That is
*guard the helper, miss the call site*, reproduced inside the milestone whose subject is two
integrity gates, for the **second consecutive iteration**.

Measured, both arms in one call, control firing:

| loader | ON rows | quarantined visible | AC-M4-5 refusal reason |
|---|---:|---:|---|
| `LoadArmForPairing` (as delivered) | 5 | **0** | `order_integrity_unpaired_block` |
| `LoadResultsFromDirsIncludingInvalid` (fixed) | 6 | **1** | `order_integrity_nonadjacent_arms` |

The reason change is the sharper half. **AC-M4-5 is cited in the plan as "a real dataset whose
correct verdict is known in advance"** — V19 established those 12 rows are a perfect whole-arm
block. The delivered code returned VOID for an **odd row count**, an artifact of the silent drop,
not for the blocking. Right verdict class, wrong mechanism — and the AC as written could not tell
the difference, because it asks only for "VOID with an order-integrity reason".

### And the repair traded one defect for another — only the OFF arm paid

Filed by the evaluator as BLOCKING, and it is against me. `AnalyzeCensoredPairs` filtered only the
**ON** arm into `validOn`. That was invisible while the filtering loader dropped invalid rows from
**both** arms; switching to the raw loader — necessary for the quarantine gate — removed the
protection for OFF with no compensating filter. The treatment gate scans OFF rows only for
*contamination* (`FmtHookEvents`), so an OFF row invalid for any other reason (`harness_error`,
`config_mismatch`, `canary_failed`) reached `PairArms` and the win tally.

Reproduced first-party before acting, two arms with a firing control: all-valid arms give
`off_wins=1 both_pass=2`; marking that same decisive OFF row `Validity{Valid:false,
Reason:"harness_error"}` gives **`off_wins=1 both_pass=2`** — identical. A row that is not a
measurement was deciding a §7 verdict.

Fixed with `partitionMeasurements` applied to **both** arms before any statistic. The gates keep the
raw slices deliberately: the executed order is a fact about the run, and the quarantine *rate* is
defined over the banked set, so filtering before either would be the same mistake one level up.

**The evaluator earned its spawn by being pointed at me.** Its directive named the controller's
repair as a target and said it had had no independent review at all — and that is the finding it
returned. A judge that is told what to attack finds things a judge that is told to score does not.

### Mutation discipline, including its own failure

The first drill on the repair **did not build**: neutering `validOff` left the binding unused, so
the red was a compile error rather than a guard firing — the "a mutation red counts only when the
mutant BUILDS" class, caught rather than banked. Redone with the binding kept live:

| mutant | LANDED | BUILDS (scoped) | killer row alone | inverse (`-skip` that row) |
|---|---|---|---|---|
| `loadCensoredArm` → `LoadArmForPairing` | grep=1 | rc=0 | rc=**1** | rc=**0** |
| `PairArms(validOn, off)` (OFF unfiltered) | grep=1 | rc=0 | rc=**1** | rc=**0** |

Both are **sole killers**: with that one row `-skip`ped the packages are green under the mutant —
i.e. exactly as first delivered. The killing assertions are the win tally and the loader's row
count, never the verdict enum, because every refusal branch also produces `VOID` and the enum
cannot discriminate.

The executor's own 13-row table (both treatment voids, six order-refusal branches, the censoring
rule, the practical-equivalence margin, each of the three KEEP conjuncts) had zero survivors.

### Three non-blocking findings, closed

- **`order_integrity_repeated_benchmark` is not merely untested — it is unreachable.** Blocks are
  deduplicated by `(benchmark, arm)`, so a third block for one benchmark is always a duplicate key
  caught by `noncontiguous_block`, and a pair holding one of its blocks is caught by
  `nonadjacent_arms`. That is *why* its `if false && …` mutant survived the whole package. An
  unreachable branch is acceptable only when **declared**, so it is now declared in the code and
  pinned by `TestD2OrderRefusalRepeatedBenchmarkIsUnreachable`, which fails loudly if it ever
  becomes reachable.
- **The `>20%` boundary is pinned on both sides**: 1 of 5 (exactly 20%) does not void, 1 of 4 does.
  The doc says strictly greater; nothing tested it.
- **`TestCensoredVerdictMatrix` passed with `[]` as its fixture** — zero subtests, zero assertions,
  one green line. Now guarded, with a required-verdict-coverage check.

### Gate 3b caught a windows-only defect in a test the controller wrote

Rule 3b(viii), and the base arm was **green** on windows minutes earlier, so the red was ours. The
fixture filename embedded an RFC3339 timestamp; `:` is illegal in a Windows filename, so
`os.WriteFile` failed at `eval_censored_test.go:77` and the test died in its own setup. Swept the
milestone's other two test files for the same shape — none (control: the grep matches the 3
`filepath` lines in that file, so it can see this class). The timestamp belongs in the row body,
which is what the loader reads; the filename only has to be unique.

### Ruled out

- ***The executor's `go build ./...` mutation precondition was skipped carelessly.*** It is red on
  the **unmodified** tree (`cmd/wasm` has no native `main`), which the sprint plan §4 already
  records, so the scoped build was the correct precondition. The executor **self-reported** this
  with a checkable proposition rather than quietly using the scoped build — a self-reported
  deviation is better evidence than a silent one.
- ***AC-M4-3 "DIFFERS".*** A stderr Observatory banner carrying a timestamp leaked into a `2>&1`
  capture. Redone with stderr separated: byte-identical, 1185 B both sides.
- ***My own first AC-M4-3 was evidence.*** It was not: both the "before" and "after" binaries were
  built from the **modified** worktree, so they were the same binary and the comparison was
  vacuous (rule 3e(b), a control contaminated by a step of my own change). Redone against a binary
  built from a pristine `origin/dev` checkout, with a control asserting the two binaries differ.
- ***The PR's missing `pull_request` runs were a dropped event.*** `mergeable` read FIRST per the
  iteration-198 rule: `CONFLICTING`/`DIRTY`, from V1's iteration 237 merging while we worked. The
  overlap was **one** file, `changelogs/v0.18-current.md` — the **fourth** time that file has been
  the cross-mission collision surface (iterations 5, 6, 9, 15). Rebase keeping both entries,
  force-push, all runs appeared.

### Two frictions that did NOT become skill edits

Both are instances of rules the skill already carries, so they are recorded here rather than in the
rulebook (Gate 5's bar is a *missing* rule, not a broken one):

1. *A guard whose call site loads past it* — rule 3j already says to anchor the refusal-branch
   enumeration to the **diff**, not to the design doc's decision list, and the call site is in the
   diff. It was not run.
2. *A repair that fixes one arm of a symmetric pair* — rule 3i's "which write does this read, and
   what else writes it?" applied to the OFF arm would have caught it before the evaluator did.

The one thing genuinely worth watching, pre-registered at instance 1 rather than written on a single
datapoint: **an acceptance criterion phrased as a verdict CLASS ("VOID with an order-integrity
reason") cannot detect a verdict that is right for the wrong reason.** AC-M4-5 was the strongest
check in the plan and it passed under the defect. If a second instance appears, the rule is that an
AC over a known-in-advance dataset names the *specific* expected reason, not its class.

### Routing and cost

Controller `claude:claude-opus-5`. Executor **`codex:gpt-5.6-sol`** — the ratified primary, probe
rc=0 replying `ok`; iteration 14 had fallen to the pi lane, this one did not. Evaluator **sonnet**,
a distinct provider from the executor, so generator≠judge holds against the executor. **FLAGGED:**
the controller-authored repair is Anthropic-authored and judged by an Anthropic evaluator — same
provider, different model. **No planner** (the plan already specifies M4 in full, so Gate 3's "plan
exists → sprint-executor" applies), **no designer, no quorum** (the doc's quorum is spent: 2 rounds,
both external reviewers present, `absent_reviewers` **empty in both artifacts**, verified
first-party rather than transcribed — and note both verdicts are `blocked`, the doc proceeding
because `D-MOTOKO-FMT-1` was RESOLVED by Mark and O3/O4 were answered by measurement; *spent*, not
*passed*). Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5 — codex
and sonnet are both quota lanes. No GPU, no `rig.lock`; `make quick-install` deliberately not run
(shared-write guardrail), and every load-bearing binary invocation used an explicitly built absolute
path rather than PATH.

### Gate 0/1

Kill switch armed · `gh` on `sunholo-voight-kampff` · billing tripwire **CLEAN** · ran from the
`#558` driver pin root, detached and clean at `daf881eaf` == `origin/dev` · running skill
**byte-identical to origin** (`cmp` rc=0) at start · dev **verified green, not merely un-red**: 16
exact-SHA checks, 0 not-green, `runs_total=2` so a run exists · **0** human directives on `#743`
since the watermark (of 13 comments) · decision ledger valid at 3 rows, **0 OPEN** · inbox **0**
unread · weekly sweep and rotation both **not due** (`#743` created `2026-08-17T05:48:23Z` = 07:48
local, after the Monday-07:00 local boundary; 13 comments < 80) · no died-mid-flight traces (`#804`
is V1's iteration 237, `#695` is V1's, the four motoko worktrees are iterations 6–9 already
adjudicated).

**External predicates re-read as commands, with controls** — G1 `#154` `state=OPEN`/`mergedAt=-`
(control `#161`/`#162` MERGED with non-null `mergedAt`) · G2 rc=**128** with the mandatory
`README.md` control rc=**0** · G3 registry `latest=2.2.0`, versions `1.0.0,2.0.0,2.1.0,2.2.0`, no
5.x · G4 unrunnable while G3 is FALSE · G5 unchanged. Rows 10/11/12 stay parked.

**Row 5's published artifact re-read too**, per iteration 12's own pre-registered lesson: upstream
`arniwesth/motoko_agent#165` is OPEN, **labelled by `arniwesth` today**, and upstream's bot reports
it **implemented in PR #166** (`MERGEABLE`, base `main_dst`, not yet merged). The issue we filed is
being acted on.

**A skill edit landed on origin mid-iteration** — `3c4bbbda2` (#805, V1's iteration 237: *the stale
binary reaches you through TESTS, not only your own commands*). The copy this iteration executed was
byte-identical to origin at Gate 1 and is one commit behind it by the end; the delta was read before
recording. It applies here and is satisfied: every load-bearing binary invocation used an explicit
absolute path, never PATH, and no test red required attributing.

**Next**: item 6's **M3** (D1b counterbalanced Wednesday block) — now unblocked, since AC-M3-4
closes by calling M4's order-integrity checker; then **M5** (depends on M3) and **M2**
(`AC-D1-live`, which needs the rig). Then item 7.

---

## 16 — 2026-08-21 — M3 and M5 built; the slot died 58 minutes in, one command short of landing them

> **RECONSTRUCTED BY ITERATION 17 FROM ARTIFACTS — iteration 16 wrote no record of its own.** This
> entry exists so the log does not silently skip a number. It is not a first-person account and it
> claims nothing the artifacts do not show; everything below is a fact about a commit, a worktree, a
> driver log line or a GitHub object.

**What ran.** Fired `2026-08-21 08:02:25` CEST from the `#558` driver pin at `df68ccc8c`
(controller `claude:claude-opus-5`, roles as configured). It picked queue item **6**'s resume
point, took **M3** (D1b counterbalancing) and **M5** (smoke-bank wiring), routed the work to the
ratified `codex:gpt-5.6-sol` executor lane, spawned an evaluator, and opened PR
[#813](https://github.com/sunholo-data/ailang/pull/813) at `06:45:40Z`.

**How it died.** `2026-08-21 09:00:33` CEST, killed by the driver's stall watchdog:
`STALL: claude 94489 idle with a descendant alive ≥2400s across 3 samples (unbounded poll loop?) —
killing early`, then `iteration exited rc=143`. The driver posted the failure to `#743` at
`07:00:35Z`. Elapsed: **58 minutes**, of which the last ~40 were idle with a live child.

**What it left.** The three traces Gate 2 names, all present: an open PR from this account; **two**
worktrees, `.wt-motoko-iter16-d1b` (branch head) and `.wt-motoko-iter16-eval` (so the evaluator had
been spawned and had its own tree, per the iteration-199 rule); and `.snap/` — the codex executor's
per-milestone snapshots — as the only uncommitted residue. What it did **not** leave: any charter
row, log entry or STATUS stamp (`grep -ci "ITERATION 16"` = **0** in both files; control
`ITERATION 15` = **3**). The one durable record was the sprint JSON, which it had already updated to
`"M3 + M5 delivered in iteration 16"`.

**Credit.** The two commits are its work, and their trailers are preserved verbatim through
iteration 17's rebase: `Co-Authored-By: codex <gpt-5.6-sol>` and
`Co-Authored-By: Claude Opus 5`. The PR body's self-critique — two of three smoke refusal branches
surviving neutering as first delivered, the decorative `AC-M3-4` analyzer call, the unsound `go
test` cache key, the uncleared scratch dir — is iteration 16's own finding, not iteration 17's.

**The note worth keeping.** Standing rule 7 is written about the *silent* failure: a `claude -p` run
that reaps background tasks after 600 s and exits **rc=0**, so neither watchdog fires and the slot
ends looking successful. That is not what happened here. This rig's driver carries a stall watchdog
that detects the idle-with-live-descendant shape directly, kills early, exits **143**, and posts to
the bookkeeping issue — loud, attributable, and visible to the human within the hour. The defect was
still real (a controller went quiet while work was outstanding, which rule 7 forbids), but the cost
was **one landing step**, not a lost slot, and the loss was announced rather than concealed.

---

## 17 — 2026-08-21 — the deliverable was a predecessor's finished work, and the job was to verify it rather than believe it

**Pick.** Not the queue head, and not a milestone. Gate 2's died-mid-flight traces found iteration
16's entire inner loop complete and unrecorded (see entry 16), so the skill's instruction applies
literally: **verify and land it, do not redo it**. Acting on the charter's `[NEXT]` tag instead
would have re-run two finished milestones and opened a duplicate PR against a green one.

**Outcome.** PR [#813](https://github.com/sunholo-data/ailang/pull/813) rebased, verified, landed as
[`b2733201a`](https://github.com/sunholo-data/ailang/commit/b2733201a). Gate 3b **GREEN** on head
`aa4543ba4`: **21** checks, **0** pending, **0** not-green, **4/4** required contexts
(`build`/`docs-gate`/`lint`/`test`) pass, `mergeStateStatus=CLEAN`. Queue item 6 is now
**M1 + M3 + M4 + M5 landed, M2 only** — and M2 is the one that needs the rig.

**Verification, because nobody had reviewed that work since its author stopped existing.** The
rebase itself first: `make/test.mk` auto-merged (the two hunks sit 215 lines apart);
`changelogs/v0.18-current.md` conflicted twice — the **fifth** consecutive iteration in which that
file is the cross-mission collision surface — resolved keeping both sides, with a 4-of-4 entry
presence check, a firing conflict-marker control, and a fresh negative literal. The rebased diff
re-derived at **+407/-27 across 7 files**, identical to the pre-rebase PR, so the rebase moved the
base and nothing else.

Then the claims. The PR asserts every refusal branch of the M5 smoke gate is pinned; that is rule
3j's bar, and it is checkable. Four mutants, each asserted **LANDED** (sha256 ≠ pre) and **VALID**
(`bash -n` rc=0) before its result was read, each restored from a `cp` backup — never
`git checkout --`, since the file is uncommitted by construction during a drill — and each restore
verified byte-identical:

| mutant | killer arm | other arms |
|---|---|---|
| neuter `banked-row contract` | `FAIL: no-banked-row smoke` (count=12, want 0) | green |
| neuter `fmt_hook_state contract` | `FAIL: failing smoke` | green |
| neuter `treatment-integrity contract` | `FAIL: invalid-treatment smoke` | green |
| ON always leads (kills counterbalancing) | `FAIL: counterbalanced sequence` | green |

Baseline unmutated rc=0. Each mutant is single-arm, so these are sole killers, not set members.

The fourth one carries the load-bearing check, because AC-M3-4's whole claim is that the *analyzer*
— not a reconstruction of what the schedule ought to emit — is the discriminator. Measured: with ON
always leading, `TestFmtDriverScheduleSatisfiesOrderIntegrity` fails at `censored_test.go:100:
CheckFmtOrderIntegrity rejected shell artifact: order_integrity_lead_not_alternating`. That
simultaneously confirms iteration 16's `go test` cache-key repair — the Go test really does see a
`/bin/bash` child's mutation.

**My own vacuous pass, caught by a count and not by an exit code.** The first attempt at that last
drill used `-run 'TestFmtABScheduleOrderIntegrity|TestFmtAB'`, which matches **no test in the
package**. Both arms returned **rc=0** and the mutant read as *survived* — a false negative that
would have made me report a decorative AC. The only tell was `ok … [no tests to run]`; re-run with
an explicit `=== RUN` count on both arms and the correct test name, the arms are **0** and **1**.
A `-run` filter is an enumerator, and rule 3a(i-e) applies to it: the drill proves the check fires,
only a run count proves it looked.

**Gate list derived, not recalled** (rule 3g), from `ci.yml`'s own `run:` lines rather than from the
PR body — which is how two gates dev added *after* iteration 16 branched were caught:
`make test-check-autoclose` and the newly exact-count `make test-stdlib-ail`. Both green on the
merged `make/test.mk` (4 suites / 4 fixtures). Full sweep on the rebased tree, all rc=0: `bash -n`
×2 · `shellcheck -S warning` (0 findings) · `make test-launchd-drivers` (10 passed / 0 failed —
item 5b's fix still holds on the rig) · `test_fmt_ab_schedule.sh` (11 PASS) ·
`go build ./internal/... ./cmd/ailang/...` · `go test -count=1 ./internal/eval_analysis/...` ·
`check-file-sizes` · `check-boundaries` · `check-changelog` · `test-check-changelog` ·
`test-check-autoclose` · `check-skills` · `fmt-check` · `vet` · `test-stdlib-ail`. Every binary
invocation ran from an explicitly built absolute path (`/tmp/ailang_i17/ailang`,
`git describe v0.33.1-191-gaa4543ba4`) prepended to `PATH`; `make quick-install` deliberately not
run (shared-write guardrail). Platform: **darwin/arm64 only**; windows and ubuntu legs unrun
locally, and Gate 3b is the only instrument that saw them.

**Ruled out.** *The PR's `CONFLICTING` state was a dropped-event or infrastructure problem* —
`mergeable` was read FIRST per the iteration-198 rule and returned `CONFLICTING`/`DIRTY`
immediately, explained completely by a two-file overlap with V1's iterations 245/246. No
`workflow_dispatch`, no empty commit, no diagnosis needed. *Iteration 16's work needed re-running* —
it did not; every claim in its PR body that I checked held, and the two milestones were complete.
*The `.snap/` directory was unfinished work* — it is the codex recipe's mandated per-milestone
snapshot output, already reconstructed into the two commits.

**Gate 0/1.** Kill switch armed; `gh` on `sunholo-voight-kampff`; billing tripwire **CLEAN**; pin
root detached and clean at `8040dfd41` == `origin/dev` at start. Running-skill check performed on
the **resolved** path per V1 iteration 241 — and both copies read, because they are different files:
the copy this session actually executed is the pin worktree's own (inode `46496692`), `cmp` vs
origin **rc=0**; `~/.claude/skills/mission-control` resolves to V1's main checkout (inode
`45241676`), which is **1 commit ahead of origin carrying an unpushed Gate-5 edit** and so differs
by 3,074 B. That is V1's divergence and is recorded here rather than acted on. Negative control
(origin skill vs the charter) rc=1. dev **verified green, not merely un-red**: **16** exact-SHA
checks, **0** not-green, `runs_total=2` so a run exists, parent-commit control **16**. **0** human
directives on `#743` since the watermark `2026-08-17T05:48:45Z` — corroborated by a raw author
enumeration showing all **15** comments are the bot's, with the script's positive control firing on
`#745` (Mark's `D-19 : B`). Ledger valid at **3** rows, **0 OPEN**. Inbox **0** unread. **0** open
`[nightly-eval]` alarms (control: 3 closed ones exist). Weekly sweep and rotation **both not due**
(`#743` created `2026-08-17T05:48:23Z` = 07:48 local, after the Monday-07:00 local boundary; 15
comments < 80).

**Routing.** Controller `claude:claude-opus-5` only. **No designer, planner, executor, evaluator or
quorum spawned** — verifying and landing a predecessor's finished work has no doc to design and no
plan to write, and a judge would be adjudicating an artifact whose author no longer exists; the
executor credit stays with iteration 16's `codex:gpt-5.6-sol`, preserved in both commit trailers.
Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no
`rig.lock`.

**Gate 5 — one skill edit, Gate 2's died-mid-flight trace (a).** `--author sunholo-voight-kampff` is
a **fleet** filter, not a mission filter: every mission on this rig pushes as the same bot account,
so the rule's phrase *"an open PR from your OWN account"* is doing work the filter cannot. The
frictions are recorded and repeated — the motoko charter and log carry **20** occurrences of
hand-disambiguating *"is V1's"/"are V1's"*, across at least five consecutive iterations, each
redoing the same adjudication with no rule to do it by. This iteration is the first where the
latent hazard went live: the filter returned `#813` (mine, the correct pick) beside `#818` (V1's
iteration 246, opened **20 minutes earlier** and still running) and `#695` (V1's, stale). Since the
trace exists to find work you should *adopt*, its failure mode is not a missed signal but acting on
a sibling's live PR. The fix uses an instrument the rule already has one line away: `git worktree
list` is scoped to your own clone, and measured here the two clones' lists are **disjoint** — 8
worktrees in motoko's, all `motoko`; 12 in V1's, none of them motoko's. A branch with a worktree in
your list is definitely yours; a miss is *not* proof of the converse, so the rule requires a second
reading and defaults to leaving an unattributable PR alone.

**Next**: item 6's **M2** (`AC-D1-live`) is the only milestone left and it **needs the rig** —
one fmt-lane run reaching `localhost:11434` with zero `openrouter.ai` connections, asserted on the
connection and paired with an OpenRouter-lane known-positive control. Doc §6's deployment
precondition (`#558`) is unchanged by this landing: merging to `dev` does not put D1b or the smoke
gate on the rig, because the installed plist runs `nightly-eval.sh` in place from V1's checkout.
If M2's rig slot is not available, item **7** (profile restoration design) is the next ungated row.

---

## 18 — 2026-08-22 — the instrument landed and refused to certify its own sweep, which is the criterion working

**Pick.** Queue row **6**'s named resume point, milestone **M2 (`AC-D1-live`)** of
`m-motoko-fmt-remeasurement-instrument` — the last of five and the only one needing the rig. Not
already landed: `tools/eval/motoko_connection_probe.sh` absent on `origin/dev` (control: an existing
`tools/launchd/` script resolves rc=0), **0** merged PRs matching, and the single `AC-D1-live` grep hit
is iteration 13's own record commit. Preconditions run as commands, not assumed: `rig.lock` free,
ollama up on **IPv4 only** (`0.32.14`; `[::1]` refused, rc=7 — rule 3c's two-instance check done),
`qwen3.6:35b-a3b-mxfp8` pulled **and loaded**, `OPENROUTER_API_KEY` SET (presence only), both lanes
present in `models.yml` (control: 17 motoko lanes).

**Outcome.** **M2 does NOT close.** PR [#829](https://github.com/sunholo-data/ailang/pull/829)
(`3e446f8c7` + `627c67d2d`) lands the instrument; the live verdict is **VOID**.

**What the sweep measured.** Under `rig.lock`, both lanes returned `driver_rc=1` with peer set `[]`
in **8m15s / 8m17s**. `AC-M2-control` states in terms that a control which does not fire makes
`AC-M2-treatment` **VOID — the probe proved nothing**. So the verdict is VOID, not FAIL, and the probe
exited 1 on `INSTRUMENT FAILURE: empty peer set; absence of evidence cannot prove routing`. The
artifact is written **before** the verdict is evaluated, so the evidence survived the refusal.

**The finding, and it required separating two things that produce identical output.** "The runs never
connected" and "the sampler is blind" are the same empty peer set. Measured apart (doc rows V36–V38):
scoped `lsof` **does** see an ESTABLISHED peer of a child process on this rig — the probe's own command
shape returned `curl … TCP 127.0.0.1:49914->127.0.0.1:11434 (ESTABLISHED)` against an unscoped same-call
control of **67** lines — and the treatment lane is `rc=0` standalone (2/2, 1m53s) **and** `rc=0` under a
faithful replication of the probe's own `run_lane` shape (2/2, 1m1s, **244** lsof lines) with
**`127.0.0.1:11434` present**. So the observable is reachable and the probe **as shipped** breaks the runs
it observes. **Mechanism NOT isolated** and said so rather than guessed: it lies in what the replication
did not reproduce — the deadline-carrying `descendant_pids` or the two-lanes-in-one-process sequencing.
The evaluator ruled out `classify_lsof` hermetically (a pure post-hoc transform called after `wait`, so it
cannot change the driver's exit code). The probe **discards both driver logs** via its `trap … EXIT`,
which is why the first diagnosis needed a re-run — for a lane that exits non-zero that log is the whole
diagnostic, and keeping it is the named first fix.

**Evaluator: FAIL 58/100, two blocking findings, and it earned its spawn by being pointed at me.** The
directive named my own V36–V38 rows and my VOID disposition as targets to refute.
**B1, reproduced first-party before acting (row V39):** `dig +short openrouter.ai` returns **A records
only** while openrouter.ai has genuine **AAAA** records (`2606:4700::6812:373`, `:273`), and `lsof`
brackets an IPv6 peer where `dig` emits it bare, so `grep -Fqx` could never match. Measured: a v6
OpenRouter peer classified **`other`** while the IPv4 positive control in the same call classified
**`openrouter`** — a **false negative in exactly the half of AC-M2-treatment that must not fail**. The
probe would have certified "zero connections to openrouter.ai" for a run that leaked over IPv6. Fixed
(union A+AAAA, `+time=5 +tries=2` which also bounds the judge's N2; compare on the bracket-stripped host),
re-measured in four directions including a non-OR v6 peer that must stay `other` so the fix cannot pass by
over-matching. **Gated, not merely fixed:** a new arm asserts a v6 leak is refused and reverting the
normalisation **reds** it (`missing openrouter [2001:db8::8]:443`), mutant LANDED (sha256) and VALID
(`bash -n`), restore byte-identical, suite **8/8**. The fixture already listed `2001:db8::8` in `OR_IPS`
with nothing pointing at it — which is exactly why the path was never exercised.
**B2 filed, not patched** (new queue row **6c**): the self-test covers `classify_lsof` and the four
`assert_*` front doors only; ~**15** live-path refusal branches have zero coverage, demonstrated by a
neutered darwin/arm64 gate surviving with a byte-identical `PASS: 7`. Same territory as the unisolated
V38 defect, so it belongs to the iteration that isolates it.

**The executor self-reported a surviving mutant** — `assert_control`'s OR-membership branch, as first
delivered — and repaired the harness so `expect_success` inspects positive-arm exit codes. A
self-reported finding is better evidence than a silent run (rule 3h(d)). **Verified first-party rather
than banked:** mutant LANDED, VALID, killed arm 3 with rc=1 against a baseline of rc=0 / 7 arms, restore
byte-identical from a `cp` backup.

**Ruled out.** *The empty peer sets are an instrument fault* — refuted by V36/V37 above, with controls.
*`classify_lsof` is the cause of `driver_rc=1`* — refuted hermetically by the evaluator; it runs after
`wait` and cannot affect an exit code. *My own first `lsof` control (0 lines) showed the sampler blind* —
**my error**: the reading was taken after the holder had exited. Re-taken against a holder asserted ALIVE
(`kill -0`) and streaming (95,869 B), it returns the connection. That is V1 iteration 247's stale-artifact
class met in my own instrument, and the correction is the reason V36 says what it says.

**Gate 0/1.** Kill switch armed; `gh` on `sunholo-voight-kampff`; tripwire **CLEAN**; ran from the `#558`
pin root, detached and clean at `b59255831` == `origin/dev`. Running-skill check on the **RESOLVED**
symlink target (V1 iteration 241's rule): `~/.claude/skills/mission-control` → V1's main checkout,
**byte-identical to origin** (`cmp` rc=0), as is this pin's copy; negative control rc=1. Noted but not
acted on: motoko's own main checkout is **144 behind** with 7 dirty files including a modified `SKILL.md`
— it does **not** serve the running skill, so the blast radius is zero, but it is a real divergence.
dev **verified green, not merely un-red**: 16 exact-SHA checks, 0 not-green, `runs_total=2`, parent control
16. **0** human directives on `#743` since the watermark (of 16 comments); ledger valid at 3 rows, **0
OPEN**; **0** open `[nightly-eval]` alarms (control: 30 closed). Phase-0 predicates re-run as commands with
controls — G1 `#154` OPEN (control `#161` MERGED), G2 rc=128 with mandatory control rc=0, G3 `latest=2.2.0`
no 5.x, G4 unrunnable, G5 outstanding → rows 10/11/12 stay parked. Sweep and rotation both not due.
AC-DEPLOY re-read with a no-pipe control (arms asserted to differ, 1 vs 0): the installed plist still runs
V1's checkout copy, so `#558` stands.

**A scope error of my own, caught by widening.** The doc's §11 cites its two quorum artifacts by a
*repo-relative* path. Searching the pin, the motoko checkout, V1's checkout and `$HOME` returned **0** with
controls firing at 4 and 95 — they live in `.wt-motoko-iter8-fmt/.ailang/state/`, a gitignored per-worktree
directory, and only enumerating **every** worktree found them. Verified once found: 2 rounds, both
reviewers `present: true`, `absent_reviewers` **empty** in both, both `blocked` — spent, not passed.

**Routing.** Controller `claude:claude-opus-5`; executor `codex:gpt-5.6-sol` (probe rc=0, `ok`, 360
tokens); evaluator **sonnet** — distinct provider, generator≠judge holds. No planner (the plan specifies
M2 in full), no designer, no quorum. Rotation pointer untouched at `claude:claude-fable-5`. Metered
**$0.00** of $5. GPU: `rig.lock` acquired `nowait` (bounded — the helper's `wait` mode is an unbounded
`sleep 30` loop) and released, three times. `make quick-install` deliberately NOT run. Gates on
**darwin/arm64 only** except where Gate 3b's matrix is cited.

**Gate 5 — NO skill edit.** Both frictions are instances of rules the skill ALREADY has: the stale
control artifact is V1 iteration 247's freshness rule, and the relative-path-to-a-worktree scope error is
the Repo Profile's own *a relative path is a claim about where you are standing, not about which file
runs*. They were broken, not missing, so they belong here rather than in the rulebook.

**Next**: row **6c** + the V38 isolation, together — keep the driver logs, isolate why the probe's
shipped path makes its lanes exit 1, and add a self-test arm per live-path branch. Then row 7.

---

## 19 — 2026-08-23 — the self-test row 6c called under-covering was running nowhere, and the coverage gap was hiding a soundness defect

**Pick.** Queue head, row **6c** — iteration 18's evaluator finding B2. Not already landed:
`git log origin/dev --grep` returns only iteration 18's own record for the probe, **0** merged PRs
matching, and the row's `[NEXT]` tag is fresh. Died-mid-flight sweep: **0** open PRs attributable to
this mission (`#695` has no branch in this clone's `git worktree list` — mine are 5, all motoko —
so it is not mine and was left alone), and the one uncommitted residue found is superseded, not
unfinished (row 6d). No new design doc and no quorum: this is a bounded remediation of the artifact
milestone M2 of `m-motoko-fmt-remeasurement-instrument` shipped, and that doc's quorum is spent.
No planner: the sprint plan does not cover 6c and the row specifies its own scope, so the executor
directive is controller-authored — which is exactly why its gate list was baselined first.

**The row understated itself, twice, and verifying it rather than inheriting it is what found both.**

**(i) It was not a weak gate. It was not a gate.** Row 6c says the self-test covers the four
`assert_*` front doors "and NOTHING in its live path". True. What nobody had asked is whether the
suite runs at all. Measured with a same-scope firing control:

| query | result |
|---|---|
| `grep -rl test_motoko_connection_probe make/ .github/workflows/` | **0** files |
| `grep -rl test_fmt_ab_schedule make/ .github/workflows/` (control) | **1** (`make/test.mk:43`) |
| repo-wide, `*.mk` / `*.yml` / `Makefile` | **0** |

So every arm iteration 18 wrote — including the IPv6 leak arm it added as a *gate* — had never
executed outside the iteration that wrote it. Wired into `test-launchd-drivers` under an explicit
`/bin/bash` (the rig's is **3.2.57**), and proven by making the self-test exit 1 and watching the
target go rc=2, then restoring byte-identically.

**(ii) The coverage gap was hiding a defect that could certify falsely in both directions.** Two
arms each, exit codes captured without a pipe and printed beside each other:

- `instrument_failure` does `exit 1`. Called from inside a command substitution it exits only the
  **subshell**: repro rc=**0**, control (identical call outside `$( )`) rc=**1**.
  `descendant_pids`'s process-tree deadline was exactly that shape, so on expiry `pids` became `""`
  and the probe carried on.
- `lsof -nP -iTCP -sTCP:ESTABLISHED -a -p ""` returns **75** lines, rc=0, **empty stderr** —
  byte-for-byte the count of the same query with no `-p` at all (**75**). Control: the same shape
  with a real pid holding no established TCP returns **0** lines, rc=1.

**An empty scope argument does not narrow a query — it removes the scope.** Chained, an instrument
whose entire job is to certify *no OpenRouter connections* would have sampled every established
connection on the machine, then either passed on another process's `127.0.0.1:11434` or failed the
treatment lane on an unrelated process's OpenRouter peer. Silently, behind `2>/dev/null || true`.
`descendant_pids` now reports through a status the caller checks, and every lsof scope is asserted
to be a non-empty comma-separated pid list before use — an empty scope fails loudly.

**Row 6c's named next step, done.** The `trap … EXIT` no longer discards both lanes' driver logs and
lsof captures. Retention runs *from* the trap, so it fires on the **refusing** path — the case the
log exists for — and a failed copy is itself an instrument failure. The evaluator confirmed under a
real `SIGTERM` that retention fires and the 143 is not masked.

**Coverage.** Re-derived at pick time: 17 `instrument_failure` call sites + 6 `usage` refusals =
**23** branches, of which the suite reached **4**. Iteration 18's B2 demonstration reproduced
first-party: neutering the darwin/arm64 gate with `if false && …` (mutant LANDED by sha256, VALID by
`bash -n`) leaves stdout **byte-identical at `PASS: 8`**, rc=0 both arms. Now **34** arms, live path
driven hermetically through a stub `AILANG_BIN` and a pruned `PATH` — no eval run, GPU, ollama or
network call anywhere in the suite.

**The executor self-reported four unproven arms, which is better evidence than a silent run.** Its
mutation batch had been contaminated by a concurrent process mutating the shared probe, and it said
so unprompted rather than reporting the table as clean. Re-proved **sequentially** by the
controller: `dig` / `lsof` / `jq` each rc **0 → 1**, each dying on its **own named message**, plus a
`pgrep` control that also reds. Note what the executor's own tightening bought — those three die
with *"lacked expected message"*, not *"unexpectedly succeeded"*: a later gate refuses too, so the
**coarse assertion would have passed**. Rule 3i's "what else writes this value", met inside the
executor's output.

**A defect of my own, in the harness written to close this row.** Isolating the fourth flagged arm
needed a mutant that keeps retention on the success path and drops it on the refusing one — the
exact "reachable only on the success path" defect the directive named. It killed the suite (rc
**0 → 1**, sole failure `refusal lost treatment.driver.log`) and the success arm stayed green, so
the isolation worked. But the suite printed **`ok 29 - refusing live path still retains both lanes
diagnostics` first**. That `expect_failure` arm observes only that the probe refuses with the
control-void message; the retention assertion is a loop **outside any arm**. The gate had teeth and
the *label* did not — a reader counting arms would have believed retention was verified while the
mutant was live. Both arms renamed to what they observe, each retention loop given its own
`pass_arm`; under the same mutant the named arm no longer prints ok (grep count **0**).

**Evaluator: PASS 93/100, ZERO blocking** (sonnet, in **its own worktree** — it mutated source, as a
good judge should, including adding a synthetic refusal branch). It was pointed at my own repair as
a named target precisely because nobody else had reviewed it. All three non-blocking findings
reproduced by command before being acted on — a NON-BLOCKING label is the judge's opinion of
severity, not a measurement — and all three closed in this iteration:

1. **The *real* wall-clock deadline was never exercised.** Arm 25 reached `descendant_pids`'s
   refusal only through the `PROBE_TEST_DESCENDANT_FAILURE` short-circuit at the top of the
   function. The `PROBE_TEST_PGREP_LOOP` stub written to drive the in-loop `date` check — it makes
   the process tree self-referential so the queue never empties — was **defined and never used**
   (grep returns the definition and nothing else). Now arm 33, proven by mutating that branch's
   `return 1` → `return 0`: rc **0 → 1**, the arm dies by name, and it falls through to
   `assert_pid_scope` — defence in depth demonstrated rather than assumed.
2. **No gate against refusal-branch drift.** Reproduced: adding a 24th `instrument_failure` leaves
   the suite byte-identical at `PASS: 32`, rc=0, shellcheck rc=0. Every other arm proves a branch
   that *exists* reds when neutered — **a removal proves the check FIRES; only an addition proves it
   LOOKS** (rule 3a(i-e)). Arm 34 counts the branches (18 + 5 = 23) and refuses when the number
   moves, with an anti-vacuity floor so a counter matching nothing reports INSTRUMENT FAILURE rather
   than a clean result. Proven by re-running the 24th-branch mutant: rc **0 → 1**,
   `refusal-branch drift: probe has 24 refusal branches`.
3. **`PROBE_TEST_FORCE_TREATMENT` was inert residue** — 1 hit in the suite, **0** in the probe
   (control: `PROBE_TIMEOUT_SECS`, 3 hits). The refusal it suggested is produced by passing the same
   lane name twice. Removed.

**What this does NOT do, stated rather than implied. V38 is not isolated.** The mechanism by which
the probe as shipped breaks the runs it observes still lies in the deadline-carrying
`descendant_pids` or the two-lanes-in-one-process sequencing, and isolating it needs the rig. The
two defects fixed here are live *candidates* and are not claimed as the mechanism. What changed is
that the next rig run will have the driver logs — which was this row's own named next step.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor
`codex:gpt-5.6-sol` — probe rc=0, replied `ok`, 13.6k tokens; run backgrounded under a 30-min
`date +%s` cap, `--sandbox workspace-write`, directive `/tmp/codex_directive_iter19.txt` (10,798 B,
≥200 B delivery assertion passed), stdin closed, four `.snap/M<k>/` snapshots delivered.
Evaluator **sonnet**, own worktree, distinct provider from the executor so generator≠judge holds.
**FLAGGED**: my own arm-label repair is Anthropic-authored and judged by an Anthropic evaluator —
same provider, different model — and was named to the judge as a target for that reason. No planner,
no designer, no quorum (reasons under **Pick**). Rotation pointer untouched at
`claude:claude-fable-5`. **Metered $0.00 of $5** — codex and sonnet are both quota buckets. No GPU,
no `rig.lock`: every arm added is hermetic by construction.

**Reconstruction proven faithful, not assumed.** The executor's final tree was sha256-manifested
before any commit was built; after replaying `.snap/M1` → `M4`, `shasum -c` returned **OK on all
four files**. M2 and M3 are byte-identical snapshots — the executor declared them combined and said
so unprompted — so they land as one commit with the reason stated rather than as a false bisection.
M1 was verified independently green at its own boundary (`PASS: 8` through the newly wired target).

**Gates.** Baselined on the unmodified tree **before the directive was written** (recipe false-green
(4)): all seven rc=0. `go build ./...` deliberately excluded — red at base (`cmd/wasm` has no native
`main`). Final sweep by the controller **outside** the executor sandbox, all rc=0: `bash -n` ×2 ·
`/bin/bash` self-test (**PASS: 34**) · `shellcheck -S warning` · `make test-launchd-drivers` ·
`check-file-sizes` · `check-changelog` · `check-skills` · `test-check-changelog`. darwin/arm64 only;
windows and ubuntu legs unrun locally and read from Gate 3b's matrix (rule 3b(viii)).

**Gate 3b GREEN, observed not predicted.** Head `7de24f03c`: **21** checks, **0** pending, **0**
not-green, **4/4** required contexts (`build`/`docs-gate`/`lint`/`test`) pass,
`mergeStateStatus=CLEAN`, then squash-merged as
[`c1950750c`](https://github.com/sunholo-data/ailang/commit/c1950750c). `mergeable` was read FIRST
per the iteration-198 rule and stayed `MERGEABLE` throughout, so no dropped-event lever was reached
for. Autoclose scan on all five commit messages, the PR title and body, and this record's STATUS
stamp: **0** hits, with a known-bad control string matching **1**.

**Ruled out.**
- *That the "coverage gap" was only a coverage gap.* Refuted by measurement: two refusal branches
  could not refuse, and one of them removes the query's scope rather than narrowing it.
- *That the executor's mutation table could be banked as delivered.* It could not, and it said so
  itself. Three arms re-proved sequentially; the contamination was concurrency, not the arms.
- *That the stale `~/dev/sunholo-data/ailang-motoko` residue is died-mid-flight work to adopt.* It
  is not: both untracked files exist on `origin/dev` (`scripts/mission_decisions.sh` byte-identically,
  `tools/launchd/test_mission_routing.sh` in a larger upstream form), and the checkout has been
  dormant since 2026-08-15. Filed as row **6d** for what it *is* — a stale rulebook at a documented
  path — not as work to inherit.
- *My own first mutation attempt.* The `perl s///` carried an unescaped `/` from an interpolated
  shell variable, so the substitution never ran; the LANDED-by-sha256 assertion caught it and
  printed *"MUTANT DID NOT LAND — result meaningless"* instead of a survived-mutant conclusion. The
  assertion exists for exactly this and it earned its keep on the controller's own instrument.
- *That Phase 0 might have moved.* Re-read as a command with a control: `#154` `state=OPEN`,
  `mergedAt=-`; control `#161` **MERGED** with a non-null `mergedAt`. Phase 0 stays CLOSED on G1
  alone, so **no G3 verdict is claimed** — the registry probe returned empty, and an empty probe is
  a claim, not a fact.

**Gate 5 — NO skill edit.** This iteration's two frictions are instances of rules the skill already
has: a refusal that cannot refuse is 3j's *a guard is not a gate until something reds when you
remove it*, and an arm whose label outran its observable is 3i's *which write does this read*. They
were broken, not missing. The one genuinely **new** shape — *an empty scope argument silently
WIDENS a query instead of narrowing it, so the instrument returns a confident, specific, wrong
answer with rc=0 and empty stderr* — is recorded here as **instance 1** and pre-registered rather
than written into the rulebook on a single datapoint. It is close to 3a(i-d)'s scope trap but not
the same: there, a bad scope returns **zero** and reads as absence; here a bad scope returns
**everything** and reads as presence. If a second instance arrives, the bar is met.

**Next**: row **6d** (the stale rulebook at the declared `MISSION_WORKDIR`), then row **6**'s M2 —
which now needs a rig slot and, for the first time, will have the driver logs to isolate V38 with.

---

## 20 — 2026-08-23 — the clone went 170 commits stale because a comment said that was harmless, and half of the comment was right

**Pick.** Queue head, row **6d**. Not already landed: `origin/dev`'s charter still carried the
`[NEXT]` tag at pick time, `gh pr list --search "workdir in:title" --state merged` returned `[]`,
and no direct-to-dev commit matched. Died-mid-flight sweep: one open PR on this account, `#695`
(`coordinator/task-d98bb271`) — **not attributable to this mission**, since no branch of that name
appears in this clone's `git worktree list` (mine are 5, all motoko) — left alone, per the rule that
an unattributable PR is not yours. Four stale sprint worktrees from iterations 6–9 remain and hold
nothing new. No new design doc, so no designer and no quorum: this is mission-infra with a
first-party measured defect. No planner: `derive-planner-lane.sh` returns `opus
fail-closed:planner-lane-field-missing`, and the row specifies its own scope, so the executor
directive is controller-authored.

**Gate 1, blocked-external rows re-measured as commands rather than transcribed.** Phase 0 stays
CLOSED: G1 `#154` `state=OPEN`/`mergedAt=null` with control `#175` `MERGED`; G2 predicate rc=**128**
(*path does not exist in 'origin/main'*) with its mandatory `README.md` control rc=**0**; G3 registry
`versions=[1.0.0, 2.0.0, 2.1.0, 2.2.0]`, `latest=2.2.0`. No predicate has flipped.

**What the row asked for, and what the measurement found instead.**

Row 6d asked for a decision — is the pin root canonical, or should the workdir be brought current —
plus "either way remove the stale skill copy". Both halves of that turn out to be blocked or
insufficient, and the interesting answer was one level down.

*Blocked:* the clone cannot be deleted (it owns the `.git` the pin worktree hangs off:
`cat .git` in the pin reads `gitdir: …/ailang-motoko/.git/worktrees/motoko`), and it cannot be
reconciled unattended. Gate 1's reconcile obligation 2 — *no incoming commit touches a
locally-modified file* — fails **by construction** here, 170 commits of overlap; `pin-root.sh`'s own
header says the same thing in its own words (*"That first reconcile is human"*). So the reconcile is
`D-MOTOKO-WORKDIR-1`, parked.

*Insufficient:* a charter edit alone leaves the next clone to drift the same way. What made **this**
clone reach 170 unnoticed is a mechanism, and it is not an oversight — it is a documented assumption
that has gone false. `mission-control.sh` emits its human-channel pin notice only when
`PIN_STATUS=STALE`, and the comment above that block gives the reason in full:

> "The shared clone being behind is not itself reportable — once drivers pin, that drift is
> harmless, and posting it every 90 minutes would train the channel to be ignored, which is how the
> original silent fallback survived twelve commits."

The second clause is true and is preserved. The first is false in exactly the case this row is
about: drift is harmless to the **driver**, whose pin holds, and not harmless to a **human session**,
because this charter named that clone the working checkout and a session started there resolves ITS
`.claude/skills/`. The evidence was sitting in the driver log, growing, on the **success** path:

| fire | 08-21 08:02 | 08-21 22:01 | 08-22 12:13 | 08-23 03:21 | 08-23 18:16 |
|---|---|---|---|---|---|
| source clone behind `origin/dev` | 119 | 132 | 144 | 159 | **170** |

That is Critical Principle 2 — *a fallback whose only witness is a log nobody reads* — aimed at the
very helper written to close it. **Guard the helper, miss the call site**, this loop's own named
recurring shape, arriving inside the fix for the previous instance of it.

**The fix keeps the true half of the comment.** Notify on crossing `AILANG_DRIVER_DRIFT_WARN`
(default 25), and thereafter only when the drift **doubles**, persisted in a per-mission state file
that is removed below threshold so the notice re-arms after a reconcile. At most one post per
doubling is what the original reasoning was protecting.

**The controller caught the executor's body naming the wrong path, and the path it named is
self-refuting.** The delivered notice interpolated `$REPO`. On the pinned pass `pin-root.sh` has
already exported `MISSION_WORKDIR=<pin worktree>` and `mission-control.sh:40` derives `REPO` from it
— so the notice would have told a human to reconcile a detached throwaway **whose drift is 0 by
construction**. Measured from this session's own live environment, not from the code:
`MISSION_WORKDIR=/Users/voightkampff/.ailang-driver-pin/motoko`,
`AILANG_DRIVER_SRC=/Users/voightkampff/dev/sunholo-data/ailang-motoko`, `AILANG_DRIVER_DRIFT=170`.
The pre-existing STALE body's `$REPO` is **correct** — that arm fires only on the pre-exec pass,
where `REPO` really is the clone that ran the stale code — so one block holds one right and one
wrong use of the same variable, and the evaluator confirmed the distinction independently when
handed it as a named target.

**Evidence.** `tools/launchd/test_driver_notify.sh` **17 → 27** arms, awk-extracted from the real
blocks rather than retyped (renaming the extraction marker prints `FATAL: extraction … produced
nothing`, rc=1 — verified by the evaluator). Every mutant LANDED by sha256, `bash -n` rc=0, restored
from a `cp` backup and asserted byte-identical; red sets **produced by running them**:

| mutant | red set |
|---|---|
| neuter the whole check (`if false &&`) | drift-a, c, d, f, h, i, j (**7**) |
| drop the doubling condition | drift-b (sole) |
| drop the `PIN_STATUS = pinned` guard | drift-e (sole) |
| drop the numeric `PIN_DRIFT` guard | drift-f (sole) |
| body from `$REPO` | drift-a, drift-h |
| remove the threshold floor | drift-i (sole) |

**Evaluator (sonnet, generator≠judge): PASS 90/100, ZERO blocking.** Three of five non-blocking
findings answered in-iteration, each reproduced first-party first: a threshold of `0` persists a
previous of `0`, after which `-ge $((0 * 2))` is true on every fire — the very outcome the doubling
rule prevents, reached through its own knob, now floored and logged (drift-i); `PIN_DRIFT` unset
aborts under `set -u`, unreachable through `pin-root.sh` today and therefore **pinned rather than
assumed** (drift-j, observed red with `PIN_DRIFT: unbound variable` before the normalisation); and
the first commit message asserted a mutation red set of four arms that nobody had run — corrected to
the measured seven, because *a red set written into a record before anyone executed it is a claim*.
Two findings accepted as reported and said so: drift-b/e/g are negative-property arms that survive
total deletion (the five positive arms beside them do not), and the missing changelog entry, added.

**Ruled out.**

- *That the cancelled CI job was ours.* `launchd drivers (bash 3.2)` came back **cancelled after
  15m18s** on the PR head, against **~68s** successes on dev's own HEAD (`a201237ca`) and on 18 of
  the last 20 CI runs — the log stops after `ok 32` and never emits arm 33, then the runner reports
  `Terminate orphan process: pid (bash)` / `(make)`. Refused the co-occurrence: the diff touches no
  file the probe suite reads, and rule 3d's strongest control — a **re-run on a byte-identical
  tree** — returned **success in 88s**. Outcome divergence with the tree held constant means the
  variable is the environment. **Instance 2 arrived 40 minutes later on this iteration's own record
  PR `#840`, whose five changed files are ALL markdown** — `17:38:10Z → 17:53:28Z`, 15m17s,
  cancelled, log stopping after the identical `ok 32`. A markdown-only diff cannot break a shell
  suite, so code attribution is refuted rather than doubted. Three observations in one iteration
  (two CI, one sandbox) against ~68s on dev; row **6e** is a confirmed defect, not a watch-item,
  and it is the next pick.
- *That the clone's uncommitted residue is unfinished work.* Measured, not inherited: of **129**
  added lines across the five modified files, **125 are byte-present** in `origin/dev`'s copy of the
  same file; the 4 that are not are prose reflows of a decision-ledger block `origin/dev` carries in
  a superseding form. Negative control (a fabricated literal) returned 0, positive control 2.
- *That a repo-local guard could defuse the stale checkout.* Any `.claude/settings.json` hook or
  in-repo test is **itself stale in the stale checkout**, which is the same chicken-and-egg
  `pin-root.sh` states about itself. The only surface that reaches a human is the notice channel,
  which is why the fix landed there.
- *That the executor's inability to run `make test-launchd-drivers` was a real red.* It reported the
  target stalling in its sandbox and honestly declined to claim the step passed. Run outside that
  sandbox: **rc=0**, all 34 probe arms plus the driver suites.

**Routing evidence.** controller `claude:claude-opus-5` (probe ok) · designer **none** (no new doc)
· planner **none** (`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`; not
spawned, the row specifies its own scope) · executor `codex:gpt-5.6-sol` (probe rc=0, ~1 reply
token; run 20m, rc=0) · evaluator `sonnet` (Agent-pinned, ≠ executor). **metered=$0.00** — codex and
the controller are subscription/quota buckets, no quorum ran, no managed_agents run.
Fable diet: **unspent** this iteration.

**Cross-mission.** `tools/launchd/mission-control.sh` is shared by `v1`, `world` and `motoko`; all
three gain this notice on their next fire, and all three have their own source clone that can drift.
`PIN_DRIFT_FILE` is defined in **both** the v1-legacy (`mission-control.pin-drift`) and namespaced
(`mission-<name>.pin-drift`) state branches, so no sibling pointer is touched. Said in the PR body
rather than left to be discovered.

**Next**: row **6e** (arm 33's hang), then row **7** (profile restoration design). Row **6**'s M2
still needs a rig slot.

## 21 — 2026-08-24 — the human said yes, and the reconcile the skill calls destructive turned out to have zero ahead-commits to lose

**Picked**: Not the queue head. A **human directive** on `#743` — Mark answered `D-MOTOKO-WORKDIR-1`
with one word, **"Yes"** (`MarkEdmondson1234 @ 2026-08-23T18:59:43Z`), 42 minutes after iteration
20's report asked it. Gate 0's contract is explicit: an allowlisted answer to a parked item unparks
it and *becomes* this iteration's pick, outranking row 6e. The drift notice iteration 20 shipped
fired in the same window and is the inbox message this iteration opened on — **178 behind**, up from
170, which is the doubling-dedupe working rather than a second defect.

**Reality check**: The row's own numbers were re-measured first-party rather than inherited, because
178 commits had landed since they were taken. Ahead-commits: **0** — so Gate 1's reconcile obligation
1 ("every local ahead-commit is a duplicate of an upstream one") is satisfied *vacuously*, which is
strictly stronger than the `patch-id` comparison it prescribes and was not knowable from the row.
Obligation 2 fails **7 of 7**: every dirty file is also touched by an incoming commit
(`comm -12` intersection = 7; positive control `CHANGELOG.md` = 1, negative control = 0, so the
instrument fires in both directions). Supersession re-derived: of **136** added lines in the local
delta, **132** are byte-present on `origin/dev`; the **4** absent are decision-ledger prose in
`motoko-mission.md` that origin carries in a superseding form (positive/negative `git grep` controls
both fired). The two untracked files split: `scripts/mission_decisions.sh` is **byte-identical** to
origin's, and `tools/launchd/test_mission_routing.sh` is **superseded and would actively regress** —
the local copy asserts the executor fallback still carries `:floor`, which origin deliberately
dropped on 2026-08-18 with the rationale in a comment. So discarding it removes a red, not a fix.

**Shipped**: The reconcile, performed and verified — **no PR, because the deliverable is a git
operation on a clone, not a code change**. Sequence, in the order the skill prescribes:
(1) backup of all 7 files + the full `git diff` patch to `~/.ailang/backups/motoko-clone-reconcile-2026-08-24`,
sha256-manifested and verified byte-identical, with a **corruption negative control that fired**
(append a byte → verifier reds → restore → verifier greens);
(2) `git checkout -B dev origin/dev` run **as prescribed first**, which REFUSED (rc=1) naming all 7
files and left the tree byte-unchanged — the refusal is the feature that distinguishes this from
`reset --hard`, and it was recorded rather than routed around;
(3) `git checkout origin/dev -- <5 tracked>` + `rm` of the 2 untracked, both under Mark's explicit
authorization to discard;
(4) `checkout -B` retried → `Reset branch 'dev'`.
Verified after: **behind 0 / ahead 0**, HEAD == `origin/dev` == `e3ed9467f`, `git status --porcelain`
**0 lines**, `SKILL.md` **3682** lines and byte-identical to the pin's copy (negative control vs
`CLAUDE.md` fired), charter byte-identical to the pin's, and **all 8 worktrees intact** — including
`.wt-motoko-iter8-fmt`, which holds the mission's only quorum artifacts in a gitignored directory.
Backup re-verified `OK` on all 7 files *after* the operation.

**Routing evidence**: model=`claude:claude-opus-5` task-class=mechanical
  round1-score=n/a rounds=1 corrections=0
  provider=anthropic agent=controller-inline cost=quota-bucket:weekly-opus
  **No designer, no planner, no executor, no evaluator spawned.** The pick is a human-authorized ops
  action whose procedure is written out step-by-step in Gate 1, with machine-checkable postconditions;
  routing it to a sprint would have added a judge with no design to judge. Designer rotation pointer
  untouched at `claude:claude-fable-5`; Fable unspent. Metered **$0.00** of $5.

**Ruled out**:
- *"The reconcile is destructive and needs the human because work would be lost."* **Refuted on the
  numbers.** Ahead-commits are **0** and 132 of 136 added lines already exist upstream. What made it
  a human call was Gate 1's obligation 2 — a **conservative** predicate that fails whenever an
  incoming commit touches a dirty file, which after 178 commits is nearly unconditional. The
  obligation is doing its job; it just cannot distinguish "you will lose work" from "your work is
  already upstream". Recorded as **instance 1**, not written into the skill on one datapoint.
- *"The drift notice needs manual clearing after a reconcile."* **Refuted by reading the code and
  then running it.** `mission-control.sh:395` removes `$PIN_DRIFT_FILE` whenever drift is below
  `AILANG_DRIVER_DRIFT_WARN` (25). Proved hermetically by extracting the real branch and driving it
  three ways: drift=0 → state file removed ("re-armed"); drift=178 unchanged → deduped, file kept
  (control fires); drift=356 → EMIT. Live drift is now **0**, so the next fire clears it.
- *"67 of 76 open issues are untracked."* **Refuted as an artifact of the wrong corpus.** That count
  is against motoko's four docs alone, and this repo's issue queue is shared with V1. Swept across
  all **9** mission docs (positive control `#558` = 57 hits, negative control fired), the real
  orphan count is **8 of 76**.

**Ruled out (added post-merge, after a third friction the first draft had not yet hit)**:
- *"The bounded Gate-3b poll I wrote was bounded."* **Refuted by watching it not stop.** The poll
  read its three counters with `set -- $out` — and this harness's `Bash` tool runs **zsh 5.9**
  (`SHELL=/bin/zsh`, `ZSH_VERSION=5.9`, `BASH_VERSION` unset), where an unquoted parameter is **not
  word-split**. Measured with a same-call control: in zsh `set -- $out` on `"17 8 0"` gives
  `$#=1`, `$1='17 8 0'`, `$2` empty; under `/bin/bash` the identical line gives `$#=3`, `$1='17'`.
  Every `[ "$pend" -eq 0 ]` then died on `integer expression expected` **to stderr**, so the break
  condition never evaluated and the loop ran until the harness's own 10-minute `Bash` cap ended it.
  The same non-splitting had already broken a `for f in $FILES` sweep loop earlier in the iteration.
- **This is instance 4 of a rule the skill ALREADY carries in full, and the rulebook is not what
  failed — my reading of it is.** `SKILL.md:1108` (*"AND THE SAME NON-SPLITTING BREAKS `set -- $var`,
  WHICH IS THE SHAPE THAT LANDS IN POLL READERS"*, added 2026-08-20 by V1 iteration 239) names three
  prior instances, including iteration 107's Gate-3b poll, and prescribes the exact remedy — read
  each value with its own command, or `set -- ${=var}`. I skimmed that block because it sits inside
  Gate 2 rather than beside the Gate 3b poll recipe I was copying. **So no skill edit**: adding a
  fourth instance to a correct rule buys nothing, and Gate 5's edit budget should not be spent
  restating a remedy that is already written down.
- **What is genuinely new is the SURFACE, and it is worth pre-registering.** Instances 1–3 all
  produced a *false verdict* — a `TIMEOUT — PARK` over a green, a bad containment reading. Mine
  produced **no verdict at all**: the comparison ERRORS rather than reading false, so the loop
  cannot exit and the slot is consumed by the harness cap. That renders as a long CI wait, which is
  the one thing a Gate-3b poll is *expected* to look like — i.e. an unbounded wait in Standing rule
  6's sense wearing a bounded one's clothes. Recorded as instance 1 of that surface; if a second
  appears, the remedy list's entry deserves the consequence spelled out beside the mechanism.

**Retro lane**: none — see above: the one candidate resolves to a rule that already exists. This
iteration's other frictions — a conservative predicate that cannot separate
"work would be lost" from "work is already upstream", and a sweep whose corpus was narrower than its
verdict — are both instances of shapes the skill already names (rule 3a's "establish the instrument
before its reading counts", and the sweep rule's own *"a CLEAN verdict must carry the issue count it
swept"*). Pre-registered as instance 1 each rather than spent on a single datapoint.

**Next**: row **6e** — `test_motoko_connection_probe.sh` arm 33, still the queue head and now
un-preempted. Then row **6f** (the sweep's 8 orphans, 2 motoko-owned) and row **7**.

**Post-report correction to entry 21 (same iteration, before the slot ended).** The digest posted to
`#850` says `DECISIONS FOR MARK: none`. That was **wrong within the hour**, and the measurement that
refuted it is one this iteration should have taken before reporting: the reconcile is a **one-shot
against a continuing drift**. Nothing pulls the clone — `pin-root.sh` runs `git fetch` only, and
`git pull` appears **0** times across both drivers — so the clone was back to **4 behind** by the
time the worktree was cleaned up. `origin/dev` lands **21.8 commits/day** (153 in 7d; corroborating
points 17/1d, 60/3d, 353/14d), so drift re-crosses `AILANG_DRIVER_DRIFT_WARN`=25 in **~1.1 days**,
and the doubling-dedupe re-notifies at ~50/~100/~200 — about four asks per nine days, every one of
them resolving to the same word for the same mechanical operation. Filed as **`D-MOTOKO-WORKDIR-2`**
and corrected on the thread. The general shape, pre-registered as instance 1: **a fix that moves a
system to a good state is not the same as a fix that keeps it there, and this loop's own reporting
template asks for the outcome at the moment of writing** — so a one-shot remediation reports
identically to a durable one. The tell: you are about to write "DECISIONS: none" for an iteration
whose deliverable restored a state nothing maintains.

## 22 — 2026-08-25 — row 6e: a self-test that can hang is an outage with a green history

**Pick**: queue head, row **6e**. No competing signal: 0 human directives on `#850` (and `#743` re-read for
the rotation-week catch, also 0), inbox 1 unread and informational, dev not red, Phase-0 `G1 #154` still OPEN
(control `#175` MERGED) so rows 10/11/12 stay parked.

**Outcome**: **LANDED**. PR [#871](https://github.com/sunholo-data/ailang/pull/871) → `086b72184`.
Evaluator round 1 **FAIL 54/100** (3 blocking) → round 2 **PASS 91/100, zero blocking**.
Gate 3b GREEN on `ddd8f3f09`: 21 checks, 0 pending, 0 not-green, 4/4 required (build/docs-gate/lint/test),
`mergeStateStatus=CLEAN`; `mergeable` read first and `MERGEABLE` throughout, so no dropped-event lever was reached for.

**What was measured, and what deliberately was not concluded.** The row named two CI cancellations; the last
100 CI runs carry **three** (`32655443831`, `32665128080`, `32673098414` — the third a push to `dev` itself),
all at ~918s, which is `timeout-minutes: 15` firing rather than a flake. Arm-33 attribution was verified from
the three job logs (identical last line, `ok 33` absent) against a green control that emits the same line and
then `ok 33` 1.06s later. **The mechanism inside arm 33 is NOT isolated and nothing here claims it is**: it has
not recurred in ~44 runs on an unchanged file. A synthetic near-copy of the walk aborts bash 3.2 with SIGABRT
5/5 while the shipped arm passes in 1.06s — the divergence is why that repro is recorded as an observation and
not promoted to the mechanism. What shipped is the structural fact: the suite had no bound of its own.

**Deliverable**: every arm carries a hard, validated wall-clock cap that TERMs then KILLs and prints a named
`not ok` plus both captured output tails; `descendant_pids` is additionally bounded by node count with a
message distinct from the clock's. Suite 34 → 39 arms, drift gate 23 → 24.

**Routing evidence**: controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` — probe rc=0,
replied `ok` — **CAPPED at the 30-minute wall, FLAGGED**, with `.snap/M1`–`M3` complete; work VERIFIED, not
adopted, and the entire mutation drill is therefore first-party. Evaluator **sonnet**, two rounds, in its own
worktree; distinct provider from the executor so generator≠judge holds there, **FLAGGED** that the drill and
every round-1 fix are Anthropic-authored and Anthropic-judged — the judge was pointed at them by name for that
reason, and it re-derived the round-1 table itself, reproducing two rows at byte-identical sha256. No planner,
no designer, no quorum: the row specifies its own scope and the parent doc's quorum is spent. Rotation pointer
untouched at `claude:claude-fable-5`; Fable unspent. Metered **$0.00** of $5 (codex and sonnet are quota
buckets). No GPU, no `rig.lock`.

**Ruled out / corrected**
- *"Two cancellations"* — three, one of them on `dev`. The row's count was a transcription of what iteration 20
  saw, not a measurement of the window.
- *The hang is in the live synthetic socket block between `ok 32` and arm 33* — refuted by the logs: the
  block's own closing message is the last line printed in all three, so it completed.
- *A synthetic reproduction is the mechanism* — refuted by its own divergence from the shipped arm. Recorded as
  instance 1 of "this stimulus sits near a bash 3.2 cliff", not as a finding.
- *M1's snapshot is not independently green* — one boundary run redded at arm 32 and the identical command was
  rc=0 on the next; M1's boundary is green at 36 arms. Both this and the arm-30 case below are load-shaped and
  are recorded as observations, **not** declared a flake class off single instances.
- **A red banked for the wrong reason.** The first batch run of the M1 mutant redded at **arm 30**, not the cap
  arm. Re-run isolated it reds at its own arm 2/2 with the unmutated suite green 3/3. Reading the exit code
  alone banks a pin that does not exist — rule 3j's corollary, paid for in this iteration rather than read.
- **My own PR body was wrong.** It claimed *"fast arms are unaffected"*; the judge measured 30s → 66-93s. The
  poll now backs off 0.05 → 0.2 → 1s (judge's re-measure: 29.89s pre-PR / 66.61s flat / 44.98-45.07s shipped),
  and the claim was corrected in the PR body rather than left standing.
- **Two of my own guards were decoration until mutated.** `report_arm_cap` had zero coverage; the cap arm was
  satisfiable by a fixture exiting 199 with no TERM and no KILL. And the *first* fix for the former passed for
  the wrong reason — with `exit 1` removed, `expect_failure` falls through to its own refusal and still exits 1
  with every marker present, so the arm now requires that fall-through message to be ABSENT.
- **An unscoped `sed` mutation killed the wrong arm** — `s/^  exit 1$/…/` hit the `ARM_CAP_SECS` validation too.
  Second instance in one iteration of "read WHICH test failed, never the exit code alone". Fixed by scoping the
  mutation to `report_arm_cap`'s line range.

**Filed, not fixed**: row **6g** — `run_bounded` *and* the production `run_lane` kill the wrapper PID rather
than the process group, so a hung grandchild reparents to `PPID 1` and survives while the suite's own
"process survived" check passes. Pre-existing since M1 and present in production, so it is a queue row on its
own evidence rather than a revision that grows this PR.

**Gate 5**: **no skill edit.** Both frictions are instances of rules the skill already carries (3j's corollary;
3i's "what else writes this value"), so they belong in Ruled-out, not the rulebook.

**Next**: row **6f** (triage-lite the 8 orphan issues), then 6g, then 7.

## 23 — 2026-08-25 — two issues filed in the same session by the same reporter, and only one of them was ever a bug

**Pick**: queue head, row **6f** — triage-lite the two motoko-owned issues from iteration 21's weekly sweep
(`#842`, `#839`); the other six were handed to V1 then and are deliberately not re-triaged here. No competing
signal: **0** human directives on `#850` since the watermark (of 6 comments), `#743` re-read for the
rotation-week catch also **0**, ledger valid at 5 rows with **1 OPEN** (`D-MOTOKO-WORKDIR-2`, still
unanswered), Phase-0 `G1 #154` still **OPEN** with control `#175` **MERGED** so rows 10/11/12 stay parked.
Weekly sweep and rotation both not due (`#850` created `2026-08-24T07:39:32Z` = 09:39 local, after the
Monday-07:00 local boundary; 6 comments < 80).

**Outcome**: **LANDED** (bookkeeping pick). `#839` **CLOSED** with its measurement; `#842` **CONFIRMED REAL at
HEAD** and filed as row **6h** with a pre-measured fix scope. Both verdict comments asserted landed by
comment-count growth against a pre-count control (`#839` 1 → 2, `#842` 0 → 1), and posted as their own
`gh issue comment --body-file` **before** any close, per the Gate-0 mechanism-B rule.

**The two issues arrived together, from the same account, in the same debugging session, and the reporter
explicitly said they were independent. They were more independent than that: one of them was never live.**

`#839` (`std/net` ignores `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`) is a **version skew**, not a defect. The
report's binary is `v0.33.0` / `ae36986`, dated **2026-08-04**. The request-aware proxy transport landed
**2026-08-20** in `e5ee6c5e5` (PR `#613`), sixteen days later. Measured with a firing control rather than
inferred from a changelog: `git ls-tree ae36986 internal/effects/net_proxy.go` is **empty** while the control
`internal/effects/context.go` returns a blob in the same call, and `git merge-base --is-ancestor e5ee6c5e5
ae36986` is **false**. What makes this a *durable* close rather than bookkeeping is that the behaviour the
reporter's decisive third repro isolates — a proxy pointed at a closed loopback port must produce
connection-refused, not a DNS error, because the target name is never resolved locally — is covered by
committed tests that CI runs (`go test -timeout 300s ./...`, `ci.yml:101`):
`TestNetProxyBoundary/proxy_selected_from_environment` and
`TestNetProxyTargetValidation/proxy_hostname_remains_unresolved`, plus `TestNetProxyNoProxy`,
`TestNetProxyDirectPin` and `TestNetProxyRedirectControls`. `go test ./internal/effects/ -run TestNetProxy`
→ **rc=0** (captured without a pipe), negative control `-run TestNetProxyZZZNoSuchThing` → `[no tests to run]`.
**One of the five reports `SKIP`, and that was checked rather than counted as coverage** (rule 2 — a parked
test is a claim): `TestNetProxyEnvProxyHelper` skips unless `AILANG_M1_PROXY_HELPER=1`, and
`TestNetProxyBoundary` re-execs it as a subprocess with that variable set and asserts
`--- PASS: TestNetProxyEnvProxyHelper` appears in its output. That subprocess arm is the only place production
`http.ProxyFromEnvironment` runs instead of the injected `proxySelector` hook — i.e. the skip is the mechanism,
not a hole. Ran that arm alone: rc=0.

**`#842` is real, and the measurement says something the issue does not.** Fed the reporter's verbatim failing
body to `openai.ParseChatStepResponse` (`internal/ai/openai/step.go:560`) with three controls in one run:
the failing shape (`finish_reason:"stop"`, `content:null`, **no `usage` key**) returns **OK** — no error,
`text=""`, `toolcalls=0`, `in=0 out=0 total=0`; the healthy control returns `"pong"` with real usage, so the
instrument sees a positive; the legitimate-tool-call control (`content:null`, usage present) returns 1 tool
call, which is exactly what a *"null content is suspicious"* heuristic would false-positive on and is why the
reporter's choice of `usage` as the key is the right one; and the fourth control — **usage present but
all-zero** — produces output **byte-identical to the failing arm**.

**The load-bearing find is that the suggested guard is not currently expressible.**
`ChatStepResponse.Usage` is a **value type** (`step.go:300`), so an absent `usage` key unmarshals to the zero
struct. Asserted directly rather than argued: `absent.Usage == zeroed.Usage` → **true**. So "treat a missing
`usage` block as a provider error" cannot be written behind the present type at all; the precondition is a
representation change (`*ChatStepUsage`, or a raw-key presence check), which is behaviour-free and separable
from the genuinely open policy question the reporter himself flagged.

**And the blast radius is wider than filed, in the direction this mission cares about.**
`ParseChatStepResponse` has exactly **two** production callers — `openrouter/step.go:162` and
`openai/step.go:170` (controls: 77 hits for a common symbol in the same tree, 0 for an invented one) — and
`internal/ai/ollama/step.go:345` builds `openai.NewClient(…)` against `<endpoint>/v1`. So **every ollama
tool-calling turn parses through the same helper**, which puts this on our own Mac Studio eval rig, where a
masked provider failure is indistinguishable from a local model declining to act. That is the charter
guardrail *"never conclude model wall"* arriving from outside the mission, reported by someone who had no way
to know it was our guardrail.

**Routing evidence**: controller `claude:claude-opus-5` (session). **No designer, planner, executor or
evaluator spawned** — the row is triage-lite and names its own procedure (ghost-discipline the repro →
verdict comment → queue-or-close) with machine-checkable postconditions, so there is no plan to plan and no
generated artifact for a judge to judge; same disposition and same reasoning as iteration 21's ops pick. No
quorum (bookkeeping pick — the Gate-2 carve-out). Rotation pointer untouched at `claude:claude-fable-5`;
Fable **unspent**. Metered **$0.00** of $5. No GPU, no `rig.lock`. Gates run on darwin/arm64 only.

**dev CI was RED at pick time and this mission does not own it.** `CI` `failure` on `02bf43668`
(run `32860399250`), 1 not-green of 15 exact-SHA checks, job `test`. Diagnosed rather than inherited: the
failing step is *Download all Go modules* — `proxy.golang.org … stream error: stream ID 1187; INTERNAL_ERROR;
received from peer` — and `02bf43668` is V1's **docs-only** iteration-276 record (4 markdown files, 0 code),
which cannot affect a module download. Parent `f4828cc89` green ~2h46m earlier is the before-arm. Fired
`gh run rerun --failed` as rule 3d's strongest control (outcome divergence on a byte-identical tree) and
handed it to V1 on the cross-mission channel per the charter guardrail; **it did not displace this
iteration's pick and no fix was attempted here.**

**Ruled out / corrected**
- *`#839` is a live `std/net` defect* — refuted. Fixed at HEAD sixteen days before it was filed; the reporter's
  binary predates the fix. The three byte-identical repro outputs were correct observations of a build with no
  proxy path.
- *The `SKIP` in the proxy suite is a coverage gap* — refuted. It is a subprocess helper the parent test
  re-execs and asserts on; it is the only arm exercising the production selector.
- *`#842` is OpenRouter-specific* — refuted. The shared parser also serves the ollama `/v1` lane, so it reaches
  the local-model rig.
- *`#842` is a one-line guard* — refuted, twice over: the signal is not representable against the current
  type, and the reporter's own caveat (Anthropic/Gemini paths do not share this parser; streaming usage
  delivery is opt-in on some providers) is upheld. A uniform guard would convert a legitimate empty completion
  into an error on a provider that simply omits usage — the same defect pointed the other way.
- *dev's red is attributable to a recent merge* — refuted by the diff (docs-only) and by the failing step
  (dependency download, before any repo command touched the tree).

**Gate 5**: **no skill edit.** This iteration's one friction — a triage row whose two issues needed opposite
dispositions, where the cheaper one to check was the one that turned out to be a ghost — is an instance of the
ghost-discipline rule the skill already carries, and it *worked*: the rule is what made the version check the
first move rather than a code read. Recorded here, not in the rulebook.

**Next**: row **6g** (`run_bounded`/`run_lane` kill the wrapper PID, not the process group), then row **6h**
(the `#842` fix), then row 7.

## 24 — 2026-08-26 — the cap killed the wrapper, and the fix for the half that matters is pinned by nothing

**Pick**: queue head, row **6g** — `run_bounded` (self-test) and `run_lane` (production) bound a
child with a wall-clock cap and, on expiry, kill the **wrapper PID only**, so a hung grandchild is
reparented to `PPID 1` and survives. Landed as PR
[#892](https://github.com/sunholo-data/ailang/pull/892).

**Ghost discipline, before any routing.** The row's evidence was iteration 22's evaluator, i.e. an
inherited claim. Reproduced first-party against the shipped code with controls firing: PRE 0 live
fixtures, cap fires `rc=199`, **1 survivor at `PPID 1`** afterwards, wrapper count **0**, POST
cleanup back to 0. The middle two lines are the finding — the suite's existing "process survived"
check passes *because* the wrapper really is dead, so the arm passed for exactly the reason it
should have failed.

**Mechanism measured before it was specified, not after.** `setsid` does not exist on macOS
(`which setsid` → not found) and every process-group precedent in this repo is Go-side
(`SysProcAttr{Setpgid:true}`), unusable from bash. Two arms, and they differ, which is what makes
this a discriminator rather than a preference: **without** `set -m` the child's pgid (9361) **is
the script's own**, so a negative-PID kill would kill the suite itself, and a single-PID kill
leaves 1 orphan; **with** it, pgid == pid (9379), differs from the script's, and the group kill
leaves 0. That measurement is why the shipped group kill is *guarded* — `jobs -p` membership,
`pid != $$`, and a live `kill -0 "-$pid"` — rather than unconditional.

**THE FINDING THAT OUTRANKS THE FIX: the production hunk is pinned by nothing.** Walking the diff
hunk-by-hunk rather than reasoning from the defect (rule 3n), the third mutant reverts
`tools/eval/motoko_connection_probe.sh` **entirely** to its `origin/dev` version — and the suite
stays **green at 40/40, rc=0**. Zero killers. Its only gate is `bash -n`. So the half of row 6g
that the row itself calls *"the one that matters"* — the production lane bound on the GPU rig,
where a surviving descendant is indistinguishable from a model declining to act — shipped with no
behavioural pin at all. Filed as row **6i** rather than growing this PR, per the rule that a hunk
with no killer is a finding and not a failure. The self-test half **is** pinned: mutant A (group →
single-PID kill) is the **SOLE KILLER**, `survivors=1`, 34 arms green before it; mutant B (neuter
`set -m`) fires the safety refusal twice and the suite **reports** the failure instead of killing
itself, which is the guard's whole purpose.

**MY OWN GATE LIST WAS UNSATISFIABLE IN THE LANE I ROUTED IT TO, AND THE EXECUTOR CAUGHT IT.** Run
1 of the executor stopped with zero files changed and reported that G3 — the full self-test suite —
never produced an rc: under codex `--sandbox workspace-write` it reaches arm 32, prints
`UNINFORMATIVE UNDER SANDBOX: loopback bind denied`, and arm 33's live-socket bind then **terminates
the enclosing session**. It stopped because my directive told it a base-red gate is a finding, not
an obstacle — so it obeyed the rule correctly and the rule was pointed at a gate that could never
be green there. This is the skill's own iteration-270 defect (*a baseline is a claim about the
environment you ran it in*): I baselined all three gates **outside** the sandbox, in my own shell,
and handed them to a lane that cannot satisfy one of them. Re-issued with G3 labelled
`UNINFORMATIVE UNDER SANDBOX`, a satisfiable in-sandbox harness in its place, and the full suite
re-run by the controller outside — which is what false-green #3 already prescribes. Cost: one
executor run. Not a lane failure and not a fallback; the lane was fine and the directive was wrong.

**A GREP-SHAPED LEAK CHECK MATCHED A PROCESS THAT ONLY *MENTIONED* THE FIXTURE.** After the green
suite run, `ps | grep "sleep 2849"` returned **1** and read as a leaked orphan. It was a codex
computer-use notification process whose argv embeds the executor's own report, which contains the
string. Scoped correctly to `comm == sleep` the count is **0**, with the control firing (1 while a
real `sleep 2849` runs, 0 after it is killed). Worth recording because it *vindicates the
executor's deviation from my directive*: I specified matching a distinctive sleep duration; it
instead scoped by a unique fixture **cwd** via `lsof -c sleep`. Its design is strictly better and
mine is the one that produced a false positive within minutes of being written.

**Evaluator (sonnet, own worktree, distinct provider from the codex executor): round 1 PASS
82/100, ZERO blocking.** It reproduced **all five** controller claims exactly — including mutant
C's zero-killer result — tried and failed to construct a false positive for the `group_safe` guard,
confirmed `set -m` leaks no job-status lines by diffing arm lines against the 39-arm baseline, and
verified `{ wait "$pid"; } 2>/dev/null || rc=$?` still propagates a real exit code (42, 5/5). Of its
three non-blocking findings, one names a defect **this PR introduced** — the guarded fallback
printed `INSTRUMENT FAILURE:`, byte-for-byte the prefix of `instrument_failure()`, which prints that
string and then `exit 1`, while the fallback does not exit; a log-grep for the abort signal matched
**4** lines where only **2** meant termination. Fixed in the second commit (`INSTRUMENT DEGRADED:`,
verified against the parsed form: 4 → 2 and 0 → 2). The other two are pre-existing or
forward-looking and went to the queue.

**Ruled out**
- *"The `ailang` binary on PATH honours `AILANG_MESSAGES_STORE`"* — **refuted at Gate 0.** The
  control (`AILANG_MESSAGES_STORE=not-a-real-store`) must error `unknown message store mode`; the
  PATH binary returned **rc=0** and listed local SQLite. Built a fresh binary to a scratch dir with
  the version ldflags (a bare `go build` in a linked worktree stamps `"dev"`), confirmed
  `v0.33.2-26-gfadbdc4e2`, and the control then correctly returned **rc=1**. The cloud inbox has
  **62** unread against the local store's 12 — they are different queues.
- *"The suite leaks an orphan"* — refuted, see above; the instrument was wrong, not the suite.
- *"SonarCloud's red is ours"* — refuted by walking it back commit by commit: green at `6193bb712`
  (18:55), red at `6759ea4fa` (19:14) and on every commit since. V1's messaging-store change, and
  the condition is *52.8% coverage on new code (required ≥80%)*. Handed over, not triaged here.
- *`pgrep`/`ps` as a liveness or leak instrument* — refuted twice in one iteration (the argv match
  above; and the skill's own rule that an empty `pgrep` is *unknown*, not *dead*). Polled artifacts
  throughout: the rc file, the log size, the worktree diff.

**Next**: row **6h** (a provider failure parsing as a successful empty completion — reaches the
ollama `/v1` lane, i.e. our own rig), then the newly filed row **6i**.

## 25 — 2026-08-26 — the pin gate had been refusing unconditionally, and the suite asserted the defect was correct

**Pick.** Not the queue head. `tools/launchd/lib/pin-root.sh` gates the launchd driver pin on
`~/.claude.json`'s `.projects[<path>].hasCompletedProjectOnboarding`; Claude Code retired that key, so
the predicate answered `false` for every path and the *"neither path onboarded"* branch was
unconditionally true. Every fire of every mission had been refusing to pin. A genuine regression
outranks the queue, and **this fire was the instance** — the driver logged `DRIVER PIN FAILED` at
21:05:14 and executed the source clone at `e3ed9467f`, **152 commits behind** `origin/dev`.

**Measurement, three readings in one call, both controls firing.**

| reading | count |
|---|---|
| project entries carrying `hasCompletedProjectOnboarding` | **0 of 15** (whole-file `grep -c`: 0) |
| project entries carrying `hasTrustDialogAccepted` (positive control) | **15 of 15** |
| project entries carrying an invented key (negative control) | **0** |

Simulated before the change: motoko, v1 and world all `REFUSE-TO-PIN`. V1's fire in the same window
reported the identical shape (`MISSION_WORKDIR unset`) on its own bookkeeping issue — the
two-mission corroboration rule 3l asks for before anything is called fleet-wide. The gate was added
2026-08-12 (`019568cf1`) to fix the class tracked at #558; it was correct then. It has been a stale
**capability claim about the harness** ever since — the shape the skill's own model table warns about,
arriving in a shell predicate instead.

**The suite could not have caught it, and asserted the opposite.** `test_pin_root.sh` writes its own
synthetic `~/.claude.json` fixtures that still carry the retired key, and its arm 8 asserted that
*the only key which now exists must be REFUSED*. A green gate pinning the production defect as
correct. That is the fixture-vs-world gap: a gate can be complete over its own inputs and blind to
the schema those inputs model — and no mutation of the code could reveal it, because the code and
the fixture agreed.

**Outcome.** PR [#923](https://github.com/sunholo-data/ailang/pull/923) →
[`ff0da7445`](https://github.com/sunholo-data/ailang/commit/ff0da7445). Suite **35 → 53** arms.
Evaluator round 1 **PASS 98/100, zero blocking**. Gate 3b GREEN on the merge: 16 checks, 0 pending,
required `test`/`lint`/`build` success (`docs-gate` N/A by path filter on the merge; it passed on the
PR head where the merge button was gated), and **`launchd drivers (bash 3.2)` — the job this change
lives in — success** on both.

**The reading that matters is that the fix restores discrimination.** Against the real
`~/.claude.json` afterwards: motoko **PIN-OK**, v1 **PIN-OK**, world **REFUSE-TO-PIN** — and world's
refusal is a *true* verdict, since that clone genuinely carries neither key, so the human fix its
message names is the right advice. A change that made all three pin would have been
indistinguishable from deleting the gate.

**The half that outlives this bug** is the anti-vacuity floor: when neither key appears in any entry,
the gate reports **Claude Code schema drift** — fail-closed still, #558's ratified posture
deliberately unchanged, only the sentence moves — so the next rename is loud on its first fire
instead of silent forever.

**The judge found a defect this branch introduced, in the exact faculty the branch exists to
improve.** Round 1 passed with one non-blocking finding: the new drift diagnosis fired identically
for a missing, invalid-JSON or non-object `~/.claude.json`, telling the next reader the **gate**
needs a new key when the **file** is broken. That is this gate's own original defect one level down,
so it was fixed here rather than filed. Three refusals, three sentences. A consequence worth naming:
an **empty** `.projects` map is *"nothing onboarded yet"*, i.e. the ordinary case — so arm 7's
natural `{"projects":{}}` fixture was **restored**, the executor's earlier fixture change having been
a correct workaround for the two-way split and unnecessary under the three-way one.

**Mutation drill, anchored to the diff hunk by hunk (rule 3n), re-run at the final tree.** Each
mutant asserted LANDED (sha256), PARSING (`bash -n`) and **effect-verified against the queried form
rather than the file's bytes** — V1 iteration 274's rule, which reached this iteration only because
the running-skill delta was read first. Each restored byte-identical from a `cp` backup.

| mutant | result |
|---|---|
| A — `_pin_onboarded` → legacy-key-only | 3 arms red, **sole killer** |
| B — neuter the drift branch | 4 arms red |
| E — neuter the unreadable branch | 3 arms red |
| F — drop the `projects_len > 0` guard | 2 arms red (an empty map misdiagnosed as drift) |
| C — revert the supporting `local` declaration hunk | **zero killers** — recorded UNPINNED, not claimed as covered |
| D — revert arm 7's fixture | 2 arms red — adjudicates the executor's self-reported deviation in its favour |

**The load-bearing observation is which assertion did NOT move.** Under B and E the `STATUS=STALE`
assertion stayed **green**, because every refusal produces STALE. Only the message *text*
discriminates. That is rule 3i's *what else writes this value* met first-party, and it is why these
arms assert prose rather than status — an arm checking only the status would have passed for the
wrong reason in both drills.

**The judge attacked beyond its brief and the fix held**: it built its own *widening-into-a-no-op*
mutant (suite caught it, 7 arms red), ran the jq edge cases end-to-end through the real driver rather
than in isolation, and neutered the **precondition** of all three new arms — all three died, so none
passes for a second reason. `/bin/bash` confirmed 3.2.57 on the rig.

**Ruled out.**
- *That the running rulebook was this checkout's copy.* It is not. `~/.claude/skills/mission-control`
  resolves to **V1's** checkout (inode `51683298`), byte-identical to `origin/dev`, while this fire's
  CWD carries a copy **139 lines short** (inode `48752546`). The 187-line delta was read before any
  gate ran. On an unpinned fire the relative-path form of that check is actively wrong — iteration
  241's hazard, met from the other direction.
- *That the SonarCloud red came from this merge.* Inherited: `failure` on the merge and on all three
  preceding commits, all V1's; negative control `3f5ca3df9` carries no Sonar check at all.
  Non-required. Conditions re-read rather than inherited from iteration 24's framing (rule 3n(d)):
  **56.9% coverage on new code** *and*, newly, **B security rating on new code** — the second is new
  since iteration 24 saw coverage alone. This diff is shell and markdown, which Go coverage does not
  measure.
- *That the `launchd drivers` CI failures were caused by a motoko diff.* Measured across the last 60
  CI runs on `dev`: **55 success / 2 failure / 1 cancelled**, all three non-success on **unrelated V1
  coordinator commits**, all three at arm 33. Filed as row **6j**, not picked.
- *That `shellcheck` findings were this diff's doing.* rc=1 with 5 identical findings at base and at
  head; not a CI gate in this repo.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` —
probe rc=0 replied `ok`, **one** run, no cap, no fallback, and it **self-reported** a directive
tension (arm 7's fixture) which measurement then vindicated. Evaluator **sonnet** in its **own
worktree** (iteration-199 rule), distinct provider from the executor so generator≠judge holds. No
designer, no planner, no quorum — a measured regression with a named fix is not a design question.
Designer rotation pointer untouched at `claude:claude-fable-5`; **Fable unspent**. Metered **$0.00**
of $5 — codex and sonnet are quota buckets. No GPU, no `rig.lock`. Gates on **darwin/arm64**; for
`launchd drivers (bash 3.2)` that IS the CI leg, macOS-only by design for bash 3.2, so the local
green is that leg rather than a proxy for it. Windows and ubuntu legs read from Gate 3b's matrix.

**Next.** Row **6h** — the provider failure arriving as a successful empty completion.

## 26 — 2026-08-27 — iteration 25's fix is correct and cannot run: the pin gate is loaded from the tree the pin exists to replace

**Pick.** Not the queue head. A genuine regression outranks it and **this fire is the instance —
again**. The driver logged `DRIVER PIN FAILED` at `11:56:34` and executed the source clone at
`e3ed9467f`, now **172** commits behind `origin/dev`, up from **152** one fire earlier.

**The finding.** Iteration 25 diagnosed the retired-key predicate correctly and fixed it correctly
(`ff0da7445`). **The fix does not run.** launchd invokes the *source clone's*
`tools/launchd/mission-control.sh` (`ProgramArguments`, read from the installed plist), which sources
the *source clone's* `tools/launchd/lib/pin-root.sh` — and that copy is 172 commits behind, so it
carries the **pre-fix** `_pin_onboarded`. The gate that decides whether to execute committed code is
itself loaded from the stale tree the pin exists to bypass. **The fix for "the pin refuses" was
landed into the only tree the refusal prevents from running.**

**Measurement — both predicates run verbatim against the live `~/.claude.json`, in one call.**

| predicate (which copy) | pin worktree | source clone | verdict |
|---|---|---|---|
| clone's, retired key only — **the one that ran** | false | false | **REFUSE-TO-PIN** |
| `origin/dev`'s, per `ff0da7445` — **the one that didn't** | false | **true** | **PIN-OK** |

Controls in the same call: retired `hasCompletedProjectOnboarding` **0** occurrences whole-file;
`hasTrustDialogAccepted` **15** (positive); an invented key **0** (negative); **15** entries
enumerated. So the fix is not merely plausible — it is measured to pass **on this exact machine and
this exact state**. The refusal branch requires *both* paths false; the fixed predicate returns
`true` for the clone, so the gate would open.

**A count would have read as coverage; reading it is what caught it.** `grep -c
'hasTrustDialogAccepted'` against the clone's `pin-root.sh` returns **1**, not 0 — and that single
occurrence is a **comment at line 185**, not code (`origin/dev`'s copy returns **4**; invented-token
negative control **0** in both). A bare non-zero count is exactly what "the fix is present" looks
like.

**Motoko-specific, not fleet-wide — and the third arm is what says so (rule 3l).**

| mission | clone behind | `pin-root.sh` fix hits | `DRIVER PIN FAILED` in driver log |
|---|---|---|---|
| control (V1) | 0 | **4** | 4 |
| world | 0 | *file absent* | 0 |
| motoko | **172** | 1 (a comment) | 3 |

Positive control per log (`iteration`): 19 / 15 / 5; negative control 0 in all three. V1 escaped
because its clone **happens to be current**, not because of anything the fix did. World has no
`tools/launchd/lib/pin-root.sh` at all — its empty grep was *file absence*, verified with `ls` rather
than read as a zero (rule 3a). So the mission that found and fixed the bug is the only one still
living with it, and the reason is **drift, not code**.

**Why nothing landed this iteration could have fixed it.** Any change I write goes to `origin/dev` —
which is precisely the tree the broken clone cannot reach. A class fix to `pin-root.sh` would be
**inert for motoko** (live for V1, whose clone is current), i.e. it would repeat iteration 25's
mistake one level up: landing the remedy where the defect prevents it from being read. Recognising
that is the iteration's actual deliverable; the class is filed as row **6l** rather than executed.

**The remedy is one word, and this is the fifth ask.** `D-MOTOKO-WORKDIR-2` — standing authorization
to reconcile the source clone. The skill's four non-destructiveness obligations are **measured, not
assumed**: **0** commits ahead (so no local work to duplicate-check), **0** dirty lines in the clone
and **0** in all eight sibling worktrees (so nothing to back up, and the incoming-vs-modified
intersection is empty by construction), and `git checkout -B dev origin/dev` is the protective form
that errors rather than clobbers. What has changed since asks 1–4 is the **class of the ask**:
iterations 21–25 raised it as hygiene against a predicted cost. The cost is now **measured** — the
loop cannot heal itself, iteration 25's landed work is dead letter, and the drift grew 152 → 172 in
a single day. Standing authorization is a human decision, not a controller one, so it is parked and
not taken.

**Ruled out.**
- *That this fire was unpinned for a new reason.* It is the same retired-key predicate, run verbatim
  and measured false for both paths — not inferred from the log line.
- *That it is fleet-wide.* Three-arm table above; V1 and world are both fine, for different reasons,
  and only one of those reasons is the fix.
- *That editing `~/.claude.json` to re-add the retired key is an acceptable unblock.* It **would**
  work — the stale gate would pass and the loop would self-heal on the next fire. Rejected anyway:
  it satisfies a **retired schema** to trick a gate that has already been correctly fixed upstream,
  it mutates shared out-of-repo harness state that no review gate can see, and Claude Code may
  rewrite that file at any time. Named for Mark as an option; not taken unattended.
- *That the SonarCloud red is ours.* `failure` on `origin/dev` HEAD and on V1's `20ce815bf` and
  `0911d1089`; `71693ead0` carries **no** Sonar check at all (negative control). Non-required;
  `sunholo-data/ailang` is V1's to own.
- *That the queue head had landed.* Re-checked at a fresh origin: `ChatStepResponse.Usage` is still a
  **value type** at `internal/ai/openai/step.go:300`, no merged PR matches, issue reported at #842 is
  still OPEN. Row **6h** is genuinely open and stays the next pick.
- *That an orphaned iteration left work behind.* The only open PR on the fleet account is `#695`
  (`coordinator/task-d98bb271`), absent from this clone's 9-entry worktree list, so **not
  attributable** to this mission — left alone per the fleet-filter rule.

**Routing evidence.** Controller `claude:claude-opus-5` (session) — **and no other role ran**. No
designer, no planner, no executor, no evaluator, no quorum: the deliverable is a measurement and an
escalation, and the only remedy is human-gated, so spawning an executor would have produced code
that cannot execute. Designer rotation pointer untouched at `claude:claude-fable-5`; **Fable
unspent**. Metered **$0.00** of $5. No GPU, no `rig.lock`. Gates on **darwin/arm64**. Running-skill
check on the **resolved symlink**: `~/.claude/skills/mission-control` → V1's checkout (inode
`51683298`), **byte-identical to `origin/dev`** (`cmp` rc=0), while this fire's CWD carries a copy
**139 lines short** (inode `48752546`, `cmp` rc=1) — the 139-line delta was read before any gate ran
and contains the designer-rotation replacement, the comma-separated fallback chains, the
`mission_pi_run.sh` typed-verdict lane and three new Gate-2 rules.

**Next.** Row **6h** — unless the reconcile lands first, in which case the next iteration's first act
is to confirm the pin actually succeeded rather than assume it.

## 27 — 2026-08-28 — the guard was not missing, it was inexpressible: a value type made absence and zero the same value

**Pick.** The queue head, row **6h** — and it is the first iteration since 23 to reach the queue head,
because 24, 25 and 26 were each preempted by a loop-health regression. Re-verified at a fresh
`origin/dev` before routing rather than inherited from iteration 26's note:
`ChatStepResponse.Usage` is still a **value type** at `internal/ai/openai/step.go:300`, issue #842 is
still `OPEN` (1 comment), and no merged PR or direct-to-dev commit touches it
(`git log -S 'ChatStepUsage' -- internal/ai/openai/step.go` returns only the 2026-era feature commit).

**The finding, and it is a finding about REPRESENTATION rather than about policy.** The reporter asked
for a guard against a provider failure that arrives as a *successful empty completion* —
`finish_reason:"stop"`, `"content":null`, no `usage` key. The guard could not be written, and the
reason is one line of type declaration: `Usage` was a value, so an omitted `usage` key unmarshals to
the zero struct and `absent.Usage == allZero.Usage` is **true**. There is no expression over that type
that separates the two cases. Changing it to `*ChatStepUsage` is the whole deliverable of step 1, and
it is deliberately ALL of it — deciding a policy is step 2 and stays deferred, because the Anthropic
and Gemini paths do not share this parser and streaming usage delivery is opt-in on some providers, so
a uniform guard converts a legitimate empty completion into an error: the same defect pointed the
other way. The standing negative control for any future guard is the legitimate tool call, which also
carries `content:null` and *does* report usage.

**Blast radius was measured, not assumed, and it is wider than the report.** `ChatStepResponse` is
referenced ONLY in `internal/ai/openai/step.go` (declaration at :295, sole use at :561) and by **zero**
test files; `raw.Usage.*` is read at exactly five sites, all at :628-636; `ParseChatStepResponse` has
exactly two production callers. Controls in the same call: `ChatStepUsage` **4** hits, an invented
symbol **0**. The consequence that matters for this mission: `internal/ai/ollama/step.go:345` builds an
`openai` client against `<endpoint>/v1`, so **every ollama tool-calling turn parses through this path**
— the defect sits on our own eval rig, where a masked provider failure is indistinguishable from a
local model declining to act. That is the *never conclude model wall* guardrail arriving from inside
our own parser.

**Mutation drill, anchored to the diff hunk by hunk rather than to the defect (rule 3n).** Each mutant
asserted **LANDED** (sha256), **BUILDS**, and — per V1 iteration 274's rule — **effect-verified against
the gofmt-parsed form rather than the file's bytes**; each restored from a `cp` backup and re-verified
byte-identical, with the post-restore suite rc=0.

- **M1** break the `usage` json tag → builds; reds **2** top-level tests (the new arms and the
  pre-existing `TestStep_TextOnly_HappyPath`). Kill-set member, **not** sole killer.
- **M2** nil no longer yields zero tokens → builds; **SOLE KILLER** of the new no-behaviour-change arm.
  Blast radius is one test, so the inverse arm is applicable and was run: the suite `-skip`-ing my own
  test, under the mutant, returns **rc=0** — which is what proves my arm is the killer rather than a
  bystander.
- **M3** neuter the deref guard → builds; reds **only** the PRE-EXISTING happy-path test. The hunk is
  pinned; it is not pinned by anything this branch wrote, and that is reported as such.
- **M4** revert to the value type → **DOES NOT BUILD**
  (`invalid operation: raw.Usage != nil (mismatched types ChatStepUsage and untyped nil)`). This is the
  "mutant does not build" class the skill warns about, and here it is not a defect in the drill — it is
  the row's own finding restated by the compiler. Recorded honestly as a **compile-time arm**, never as
  a behavioural kill.

**The judge found the gap my drill missed, and it is the interesting half of the iteration.** Evaluator
round 1 **PASS 94/100, ZERO blocking**, in its own worktree (iteration-199 rule). It reproduced every
controller claim including M2's inverse arm, then went past them: it built a **16-case differential
harness** across parent `d5305fa79` versus this branch — `usage:null`, `usage:[]`, `usage:"none"`,
`usage:42`, negative tokens, malformed JSON, `prompt_tokens_details` present and absent — and found the
output **byte-identical in all 16**. That is stronger evidence for "no behaviour change" than anything I
produced, and it covers the two cases I explicitly named as untested when I briefed it.

**Its non-blocking finding #1 is a real zero-killer hunk, reproduced first-party before it was
believed.** `cacheRead = usage.PromptTokensDetails.CachedTokens` — inside this diff's second hunk — has
**no killer anywhere**: mutated to `+ 999` it LANDS, BUILDS, effect-verifies 0→1 on the parsed form, and
leaves openai, openrouter and ollama at **rc=0 with 0 FAIL lines in total**. The apparent coverage is a
mirage of exactly rule 3i's shape — *which write does this read?* — and the judge named the mechanism:
`cache_usage_test.go` contains **0** references to `ParseChatStepResponse` and **7** to `Generate`, a
different code path that never reaches this function. Per the rule that *a hunk with no killer is a
finding, not a failure*, it is filed as row **6m** rather than used to widen the PR — the same
disposition iteration 24 gave its own zero-killer hunk at row 6i.

**A stale binary was 43 commits adrift and would have reached the suite through a test's own
shell-out.** Per V1 iteration 237's rule the gates ran with a freshly built, ldflag-stamped binary
prepended to `PATH` — `v0.34.0-118-gd5305fa79-dirty`, matching `git describe` exactly — rather than
`make quick-install`, which would have mutated a `~/go/bin` shared with every concurrent agent on the
rig. The system copy reads `v0.34.0-75-gfb6084f4b-dirty`. Nothing in this diff depended on it, but the
provenance is stated because "the tests pass" from an unidentifiable build is not a claim.

**My own gate list was baselined in the wrong lane once, and the rule caught it before the executor
did.** Fifteen test files across the three `internal/ai` packages bind loopback sockets, which
`workspace-write` denies — so a full-suite gate handed to codex would have been unsatisfiable by
construction (V1 iteration 270's rule, which iteration 24 paid for first-party). The directive
therefore carried three in-sandbox-satisfiable gates and named the three suites explicitly as
`UNINFORMATIVE UNDER SANDBOX`, with the controller re-running all of them outside. The executor ran
exactly that and reported no base-red gates.

**Ruled out.**
- *That the drift is fleet-wide.* It is motoko's alone: source clone **205** behind `origin/dev`
  (0 ahead, 0 dirty), pin worktree exactly at `origin/dev`, world with **0** pin failures ever.
- *That "how far behind" is the predicate — this refutes iteration 26's own framing, which said V1
  escaped "because its clone happens to be current".* It is not current: V1's clone is **18** commits
  behind and pins **cleanly**, its last `DRIVER PIN FAILED` dated 2026-08-27 07:10 with clean fires
  since. The predicate is whether the clone CARRIES THE FIX —
  `git merge-base --is-ancestor ff0da7445 HEAD` → **NO** motoko / **YES** V1, corroborated by
  non-comment `hasTrustDialogAccepted` occurrences **0** vs **3** (motoko's single hit is the
  line-185 comment iteration 26 already identified; invented-token negative control **0** in both).
  So V1 **recovered** by crossing the fix, which motoko structurally cannot.
- *That the SonarCloud red is ours, or new.* Walked commit by commit: green at `7dff0942d`, first red
  at `caea1f9e1` (V1's M-EVAL-ROLLING-ELO merge). Conditions read from the check's own output rather
  than inherited from iteration 25's framing — **64.2% coverage on new code** (was 56.9% at iteration
  25, 52.8% at 24, so the gate is live rather than stuck) and **B security rating on new code**.
  Non-required; handed to V1 with delivery asserted.
- *That `cache_usage_test.go` covers the cacheRead path.* It does not — 0 references to the parser.
- *That a value-type revert is a behavioural mutant.* It is a compile error, and calling it a kill
  would have been the "mutant does not build" vacuous pass.

**A finding I nearly recorded backwards, and the correction is the point.** The driver logged
`WARNING: driver-pin notice FAILED to send via ailang messages` on this fire, and the natural reading —
*queue row 2's loud lane-degradation notice is silent for exactly the event it exists to report* — is
**false**. Counting the two arms separately rather than the warning as a whole:
`FAILED to send via ailang messages` **3**, `FAILED to post to issue` **0**, `no issue notice possible`
**0**, positive control `DRIVER PIN FAILED` **4**, negative control **0**. The GitHub half succeeded
every time and the comments are visible on `#850` (`2026-08-28T00:03:18Z` plus two predecessors), so
Mark's channel worked and the mechanism did what Critical Principle 2 asks of it. A second hypothesis
died the same way: the send passes `gh`-style `--title`/`--from`, which iteration 252's rule makes the
obvious suspect, and re-running the identical form live returns **rc=0** — Go's flag package accepts
both dash forms. What survives is narrow: the send carries `2>/dev/null`, so three consecutive failures
produced **zero** information about their cause. That is the *robustness wrapper hides the cause* class,
it is driver territory (V1-owned), and it is handed over rather than picked. Worth recording chiefly
because the wrong version of this sentence was already written into this iteration's STATUS stamp
before the arms were counted.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` —
probe rc=0 replying `ok`, ONE run, no cap hit, no fallback traversed. Evaluator **sonnet in its own
worktree**, a distinct provider from the codex executor, so generator≠judge holds. **No designer, no
planner, no quorum**: the row specifies its own scope and step 2 is explicitly out of it. Designer
rotation pointer untouched at `claude:claude-fable-5`; **Fable unspent**. Metered **$0.00** of $5 —
codex and sonnet are both quota buckets. No GPU, no `rig.lock`. Gates on **darwin/arm64**; windows and
ubuntu legs unrun locally and read from Gate 3b.

**Two configuration observations, recorded rather than acted on.** (a) `MISSION_EXECUTOR_FALLBACK` on
this rig is still the single old value `pi:openrouter/deepseek/deepseek-v4-flash-0731`, while the
running skill has documented comma-separated chains with an ollama-cloud rung ahead of it since
2026-08-26 — the mission env file has not caught up. Not exercised this iteration (codex probed
green), so it is one datapoint, not a defect claim. (b) The codex executor spent its opening turns on
the repo `CLAUDE.md`'s "Session start" inbox instruction, which cannot complete inside
`workspace-write`; it bounded the wait itself and proceeded. Neither has a second instance yet.

**Next.** Row **6i** (the production `run_lane` group kill, pinned by nothing), then the new row **6m**,
then 6j. `D-MOTOKO-WORKDIR-2` remains the only OPEN decision and this is the **sixth** ask.

## 28 — 2026-08-29 — RECOVERED RECORD: the pin-gate bootstrap landed and the slot died before it could say so [HARNESS]

Reconstructed by iteration 30 from the traces the fire left, and labelled as reconstruction rather
than as a first-person record — nobody has reviewed this iteration's reasoning since the agent that
held it stopped existing. What is certain is what is on disk and on origin.

**Pick.** Row **6l**, the class fix iteration 26 filed and deliberately did not execute: the launchd
pin gate is loaded from the *source clone's* `tools/launchd/lib/pin-root.sh`, i.e. from the very tree
the pin exists to bypass, so iteration 25's correct fix could never execute on this mission.

**What landed.** [`61859c35d`](https://github.com/sunholo-data/ailang/commit/61859c35d) —
*"fix(driver): bootstrap pin gate from committed ref"* — merged as PR `#964` at `2026-08-29 22:14:26Z`.
Worktrees `.wt-motoko-iter28-pin-bootstrap` and `.wt-motoko-iter28-eval` both survive at `d622898a7`,
which is that commit pre-squash, so the sprint and its evaluation both ran.

**What it also did, and this is the half that matters for the ledger.** The same commit stamped
`D-MOTOKO-WORKDIR-2` **RESOLVED** — Mark answered `MOTOKO-WORKDIR-2: Yes` on `#850` at
`2026-08-29T09:09:20Z`, granting standing authorization to reconcile the source clone unattended when
the three safety predicates hold. The row records iteration 28 re-measuring **0 ahead / 0 dirty /
292 behind** and advancing `e3ed9467f` → `bd0bb157d` with `git checkout -B dev origin/dev`, post-verified
**0 dirty**. So the six-times-asked decision that had blocked iterations 21 through 27 was answered and
discharged in this slot.

**What it did not do.** Write a STATUS stamp or a log entry. Measured by iteration 30:
`grep -ci 'ITERATION 28'` returns **1** in the charter — that ledger row, and nothing else — and **0**
in the log and **0** in the STATUS archive, against control `ITERATION 27` at **5** and **3**. The
charter therefore recorded a resolved decision whose iteration appeared never to have run.

## 29 — 2026-08-31 — RECOVERED RECORD: a slot died holding a finished, unreviewed, unpushed milestone [HARNESS]

Also reconstructed by iteration 30. This one died later in the loop than 28 did, and therefore left
more.

**Pick.** The queue head, row **6i**.

**What it left, in three places no single Gate-2 trace looks at.**

- A **complete milestone**: commit `4bd9e7110`, *"test(eval): pin run_lane process-group cleanup"*,
  **+237/−13** in `tools/eval/test_motoko_connection_probe.sh`, on branch
  `sprint/motoko-iter29-run-lane-harness`, authored `2026-08-31 05:12:05 +0200` — **never pushed**, so
  invisible to every remote check.
- Its **design doc and sprint plan**, 573 and 237 lines, written at 04:49 and 04:53 and left
  **untracked in the driver's pin worktree** — invisible to a `design_docs/` grep on origin.
- Two worktrees, `.wt-motoko-iter29-exec` and `.wt-motoko-iter29-eval`, the second created at the
  sprint commit, which is what says the evaluation had at least been set up.

**Its quorum history, which is the part worth keeping.** Four artifacts exist for the doc, at
`2026-08-30T12:50:38Z`, `12:51:44Z`, `12:57:04Z` and `2026-08-31T02:45:23Z`. **All four read
`blocked`** and every objection in them is real. Round 4's two — `gpt5-6-sol` on the emergency outer
bound and full-descendant reaping under macOS bash 3.2, `gemini-3-1-pro` on a missing real `lsof`
having to hard-fail on Darwin rather than structured-skip — each carried a concrete reviewer-authored
`proposed_fix` and disputed no design *direction*, which is exactly the narrow-refinement carve-out.
Iteration 29 applied both **verbatim**: they are the last two rows of the doc's own Quorum
Verification Log, and they are the two design properties the shipped code implements. Round 4's third
reviewer, `oc-glm-5-2`, is recorded `absent(invalid)` on a malformed JSON response whose raw fragment
nonetheless begins `"verdict":"reject"` — the tracked defect reported at #941, and the reason the
round's `presentCount` is 2, not 3.

**Why the record could not be inherited.** No STATUS stamp, no log entry, no charter row, no PR. The
routing decision it made — apply the carve-out at round 4 rather than park — is recoverable only from
the artifacts above, which is why iteration 30 re-derived every load-bearing claim first-party instead
of transcribing this reconstruction into a verdict.

## 30 — 2026-08-31 — two slots died in a row, and the second died holding a finished, unreviewed, unpushed milestone [HARNESS]

**Progress**: row 6i LANDED — motoko queue rows 6a–6i now all closed; 6j, 6m, 6n, 6o open on the
loop-health track, and the Phase-0-gated migration epic (rows 10/11/12) is unmoved because its
external predicate is unmoved.

**Pick.** The queue head, row **6i** — but reached by *inheriting* iteration 29's corpse rather than
by routing a fresh sprint, per Gate 2's died-mid-flight rule: verify and land, do not redo.

**The traces, and which one actually worked.** Trace (a), the fleet-account open-PR filter, returned
**four** PRs — `iter305`, `iter306`, `iter296`, `iter308` — **all V1's**, none with a branch in this
clone's **15**-entry worktree list, so the trace that exists to find work you should adopt said
*nothing exists*. Trace (b), `git worktree list`, is the one scoped to this clone by construction and
it named `.wt-motoko-iter29-exec` and `.wt-motoko-iter28-*`. Trace (c), uncommitted/untracked state,
is what turned "an attempt happened" into "a milestone exists": the pin worktree's two untracked design
documents. **Two dead slots in a row is reported as a pattern, not as two incidents** — the loop cannot
diagnose why its own slots die, but it can make the frequency visible.

**Verification of the inherited work — as an inherited claim, not as a colleague's word.**

| check | command | result |
|---|---|---|
| baseline, rebased tree | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | **rc=0**, 41 ok, 0 not ok (was 40) |
| arm 36 is not vacuous | its own evidence line | `ready=yes` · distinct wrapper/child PIDs · `pre_timeout_child_live=yes` · `timeout=yes` · `outer_cap_fired=no` · `survivors=0` · `real_lsof=/usr/sbin/lsof` |
| **M1**, the row's own mutant | group kill → single-PID (`kill -TERM "-$pid"` → `"$pid"`, same for `-9`); LANDED by sha256, PARSES `bash -n` rc=0, intended effect asserted against the system's own view (group-kill sites 1→0) | **rc=1, SOLE KILLER** — arm 36 the only `not ok`, `survivors=1 outer_cap_fired=no`, 35 arms still pass |
| restore | `cp` backup | sha256 byte-identical, `git status` clean, site count back to 1 |

`survivors=1` with `outer_cap_fired=no` is the load-bearing reading: the red is the **production**
timeout's cleanup regressing, not the emergency containment firing. The M1 red arrives in exactly the
direction predicted, so it is paired against the baseline on the identical tree — rc=0 versus rc=1,
outcomes **differ** — which is what makes it evidence rather than co-occurrence.

**The judge found the half the drill could not see, and that is the iteration's best result.**
Evaluator **PASS 87/100, zero blocking** (sonnet, own worktree, distinct provider from the codex
executor, so generator≠judge holds). Both non-blocking findings were reproduced first-party **before
being believed and before being dismissed**:

- **A.** Mutating **only** the SIGKILL escalation site — `kill -9 "-$pid"` → `kill -9 "$pid"`, with
  the group `-TERM` site left untouched (asserted: group `-9` sites 1→0, group `-TERM` sites stay 1)
  — leaves the suite **rc=0, 41 ok, survivors=0**. The fixture's grandchild is a plain `sleep` with no
  SIGTERM trap, so it dies at the TERM stage and the escalation is never reached: that stage is
  dead-for-discrimination. Row 6i's mandated mutant changes **both** sites at once, so it reds on the
  TERM half alone and the doc's acceptance bar is honestly met — the gap is one the bar itself could
  not see. That is rule (i-e)'s shape: a removal proves a check **fires**, only a differently-shaped
  mutant proves it **looks**.
- **B.** `REAL_LSOF` is resolved with `command -p -v`, and POSIX's standard-path guarantee for `-p`
  **does not hold on the CI shell**. Measured on GNU bash **3.2.57** (arm64-apple-darwin25), which is
  the `launchd drivers (bash 3.2)` target rather than a proxy for it: a shadowing `lsof` placed ahead
  of `getconf PATH` in the **ambient** environment resolves as `REAL_LSOF`. Arm → the hijack; control,
  clean PATH → `/usr/sbin/lsof`; negative control, hostile directory with no `lsof` in it →
  `/usr/sbin/lsof`. The defence the design doc actually names — the stub PATH this suite installs for
  *itself* — holds, and was separately confirmed by the `markers=yes` evidence distinguishing
  `path-lsof` calls from `fixture-lsof path=$REAL_LSOF` calls. Only the code comment's *"can never"*
  was too wide.

Disposition: **B's comment was narrowed to what is actually defended** (`1caf02e44`) — comment-only,
no executable line changed, so the evaluation over the executable tree still stands — and **both
findings were filed as row 6o** rather than used to widen a passing PR.

**A third defect arrived from another mission's judge and was ghost-disciplined before adoption.**
Reported at #975 by V1 iteration 308's evaluator, against *motoko's* instrument: the arm named
`descendant discovery refuses on the real wall-clock deadline` cannot fail for the reason it names.
Reproduced at motoko's own HEAD rather than inherited — neuter the in-loop check with
`if false && (( $(date +%s) > deadline )); then`, mutant LANDED by sha256 and PARSING, intended effect
asserted (neutered sites 0→1) — and the suite stays **rc=0, 41 ok, 0 not ok**, this arm included. Real,
pre-existing, and the same class as row 6i one arm over: an assertion over an over-subscribed
observable. Filed as row **6n**. Its B3 half (on the runner, discovery refuses with *neither*
diagnostic message) is recorded **UNREPRODUCED**, because it can only be measured on the runner.

**Four blocked quorum rounds is data about this loop's scoping, not about that document.** Per V1
iteration 257's rule the *surfaces* were tracked rather than the round count: R1 survivor oracle /
helper duplication / stub conflict-surface; R2 hermetic-drive premise / helper duplication / hermetic
premise; R3 readiness determinism / real-vs-stub `lsof` / platform-specific verification; R4 outer
bound / Darwin hard-fail. They are **spread across surfaces** and **no reviewer flipped to pass**, so
the disposition is *immature, keep revising* — **not** SPLIT, which is what localisation plus a flip
would have indicated. Recorded here because only a human can act on the pattern.

**Dev CI was red, it was not ours, and nothing was done to it.** `test` failed at `d65a0900c` on step
*Verify embedded pi assets are in sync*; walked back per commit, first red at `ebc089c33` (V1's
README-only edit), green at `15cec372b` and `9f267cf1f`. V1 already had `#983` open with the fix when
this fire started, so the owning-mission rule held twice over: the red did not displace the pick, and
no duplicate fix was opened — the `#758`/`#759` lesson. It merged as `f78b1d451` and this branch was
rebased onto it before landing.

**Gate 3b — read on the MERGE, not merely on the PR head.** PR head `1caf02e44`: **21** checks,
**0** pending, **0** not-green, `mergeable` read first per the iteration-198 rule
(`MERGEABLE`/`CLEAN`), all **4/4** required contexts green. Merged as
[`4bd58bef6`](https://github.com/sunholo-data/ailang/commit/4bd58bef6), whose own check set is
**16** checks, **0** pending, **1** not-green: `SonarCloud Code Analysis`. Rule 3d's parent-walk
before attributing it — it is `failure` on the parent `f78b1d451` and on `15cec372b` and `d3e4e59cf`,
while `d65a0900c` and `ebc089c33` carry **no Sonar check at all**, which is the negative control. So
it is inherited, non-required, and V1's to own; it is named here rather than left invisible.
`docs-gate` does not appear on the merge commit at all — path-filtered — and passed on the PR head,
where the merge button was gated; that is the reading iteration 25 recorded and it still holds.
`launchd drivers (bash 3.2)`, the CI leg that actually executes this suite, is **success** on both.

**Weekly sweep, and its headline number is a property of the fleet rather than a backlog.** **94**
open issues enumerated — the list length quoted beside the verdict so a truncated enumeration cannot
wear a complete one's clothes — each grepped `-cE '#<n>\b'` across **four** corpora (charter, log,
STATUS archive, dashboard) and printed as a per-issue table, never a summary sentence. Negative
control fired. **79 of 94** carry zero mentions, which is expected and not actionable as a backlog:
this repository carries four missions and motoko's charter is deliberately scoped to the harness. The
motoko-territory orphan that mattered is `#975`, now row **6n**.

**Ruled out.** That the two dead slots were a rotation defect — the STATUS-rotation deletion class
this loop has paid for twice — is refuted: `grep -ci 'ITERATION 28'` returns **1** in the charter
(the ledger row iteration 28 itself wrote) and **0** in the log and archive, whereas a rotation
deletion removes a stamp that was *there*, leaving a `git log -S` trail on the charter and none on the
archive. There is no such trail here; the stamp was never written. That the M1 red might be an
artifact of the rebase is refuted by running the baseline on the identical rebased tree (rc=0). That
`#975` might be a ghost inherited from a sibling is refuted by first-party reproduction.

**Routing evidence.** Controller `claude:claude-opus-5`. **No designer, planner or executor ran this
iteration** — all three are iteration 29's, inherited, and Gate 2 requires verify-and-land rather than
redo; spawning them would have re-authored a doc that already exists and re-run a milestone that was
already finished. Evaluator: **`sonnet`, Agent-tool-pinned, in its own worktree**
(`.wt-motoko-iter29-eval`, branch `eval/motoko-iter30-run-lane-harness`), distinct provider from the
`codex:gpt-5.6-sol` executor, so the generator≠judge guard holds without a re-route. Designer rotation
pointer untouched at `claude:claude-fable-5`; Fable **unspent**. Metered **$0.00** of $5 this
iteration; iteration 29's fourth quorum round spent **$0.1108** and is attributed there. No GPU, no
`rig.lock`. Gates on **darwin/arm64** and on the CI shell's own bash 3.2.57; windows and ubuntu legs
unrun locally and read from Gate 3b.

**Gate 5 — no skill edit.** Every friction here is an instance of a rule the rulebook already carries
and that *worked*: the died-mid-flight traces found the corpse, the fleet-`--author` rule kept this
mission off three sibling PRs, the owning-mission rule kept it off V1's red, ghost discipline turned a
sibling's claim into a first-party measurement, and reproduce-before-dismissing turned two
"non-blocking" findings into a queue row. The one candidate — that trace (a) is near-useless on a
shared push identity while trace (c) did the work — is already written into the skill as the
fleet-filter rule and as trace (c) itself, so this is a second instance of a documented gap that has
already been closed, not a new one.

**Next.** Row **6n** (#975's wall-clock arm), then **6o** (the escalation half of the group kill), then
6j and 6m. Decision ledger: **5** rows, **0 OPEN** — `D-MOTOKO-WORKDIR-2` was answered and discharged
by iteration 28, so for the first time since iteration 21 this mission is asking nothing of Mark.

## 31 — 2026-09-01 — the quorum blocked twice on one surface, and my refutation of it measured the wrong binary [HARNESS]

**Progress**: row 6n's stated blocker DISCHARGED (the runner half of the defect reported at #975 is
now measured) and its design written, but the row is **PARKED needs-human-review**, not landed —
the design quorum blocked 3/3 in both rounds. Loop-health track: 6a–6i closed, 6n parked, 6o next,
6j/6m/6p/6q open. The Phase-0-gated migration epic (rows 10/11/12) is **unmoved**: upstream `#154`
is still OPEN, re-measured as a command this iteration.

**Pick.** The queue head, row **6n** — "the wall-clock discovery arm cannot fail for the reason it
names", reported at #975 by V1 iteration 308's evaluator against motoko's own instrument.

**The row's own blocker, discharged from a sibling's CI.** Row 6n said finding B3 *"is unreproduced
here and must be measured on the runner, not locally, before any fix is designed"*. The measurement
was sitting unread in another mission's pull request. The `launchd drivers (bash 3.2)` job
`99402730557` on **V1's open PR #971** (head `8a384e81b`) fails with exactly

    not ok - descendant discovery refuses on the real wall-clock deadline
             lacked expected message: process-tree discovery failed

and the surrounding log reads `lane=treatment driver_rc=0 peers: []`, `lane=control driver_rc=0
peers: []`, `INSTRUMENT FAILURE: empty peer set`. So #975's *"refuses with NEITHER message"*
understates it: under that configuration discovery **did not refuse at all** — both lanes completed
and the run died downstream on the empty-peer-set guard. Control that the log was really parsed:
**32** passing `ok` lines in the same fetch. **Scope, and it is load-bearing: that certifies #971's
tree, not motoko's HEAD**, whose own `launchd drivers` leg was `success` at `4bd58bef6`.

**The defect, reproduced and then sharpened.** Every mutant LANDED (sha256), PARSES (`bash -n`
rc=0), effect-asserted against the system's own view, restored from a `cp` backup byte-identical
with `git status --porcelain` **0**.

| # | mutant | result |
|---|---|---|
| baseline | none | **rc=0**, 41 ok, 0 not ok, 50s |
| E2 | neuter the in-loop wall clock alone | **rc=0, 41/41**, arm 33 still `ok` — the row's claim, re-derived |
| E7 | neuter the node ceiling alone | **rc=1**, 39 ok, **only arm 40** reds, arm 33 still `ok` |
| E8 | the minimal fix alone (3 distinct messages + arm 33 asserting the wall-clock one), **no ceiling change** | **rc=0, 41/41, 50s** |
| T1 | E8's fix **plus** E2's mutant, ceiling untouched | **rc=1 in 44s**, arm 33 the failing arm on the exact message |

E2 and E7 together are the statement row 6n did not have: **each branch independently suffices for
arm 33 to pass**, so the arm cannot discriminate them by construction. T1 is the load-bearing
acceptance and it was *run*, not predicted — clean rc=0/50s against mutant rc=1/44s, printed side by
side, outcomes differing, and the mutant *faster* than the clean run rather than a hang.

**A correction to my own inference, and it is the root of what went wrong afterwards.** Neutering
both bounds hangs the suite past a 600s bound; I read that as *"the node ceiling is what terminates
this arm at HEAD"*. It does not follow — the sound reading is only *"with the wall clock dead, the
ceiling stops the walk"* — and that inference is what seeded the design the quorum then rejected.
E8 refuted it directly. A reviewer (`oc-glm-5-2`) attacked it before I did.

**The quorum blocked twice, on one surface, from opposite directions.** Round 1
(`…2026-09-01T00-35-20Z.json`, $0.0806) and round 2 (`…T00-43-50Z.json`, $0.0781), **3/3 external
reviewers rejecting both times, no absentees** — read at the correct nested path
(`.synthesis.absent_reviewers`), not the top-level one that returns `null`. Per the objection-surface
rule the surfaces were tracked rather than the round count: both rounds localise on **one** surface,
the wall-clock-versus-ceiling race, and **no reviewer flipped to pass**, so the disposition is
*immature*, not SPLIT. Round 1 rejected raising `MAX_TREE_NODES`; round 2 rejected **not** raising
it. The one revision and the one re-quorum the protocol allows are spent, and the surviving
`gpt5-6-sol` objection carries no concrete reviewer-authored fix, so the narrow-refinement carve-out
does not apply. **Park.** Standing rule 2 forecloses force-passing over three unanimous rejections
however good the controller's argument feels — and this iteration is the argument for that rule.

**The evaluator's blocking finding is against me and it is correct.** To answer the reviewers'
shared empirical premise — *a CI runner might do 4096 iterations inside the 1-second window* — I
benchmarked **system `pgrep`** at ~**79** iter/s and reported a **~52×** margin. Arm 33 sets
`PATH="$live_bin"`, and this suite installs its **own** `pgrep` stub at
`test_motoko_connection_probe.sh:254-262`; the walk never calls system `pgrep`. Re-derived
first-party against the actual stub: **474.9 / 652.7 / 648.6** iter/s → 4096 iterations ≈
**6.3–8.6s**, a **~6–9× margin**. The wrong instrument, re-run for the side-by-side, reads **92.1**
iter/s; the negative control confirms the stub resolves first on that PATH; the judge's independent
instrument measured ~455 iter/s and agrees. The *conclusion* survives and never rested on that
number — E8 and T1 measured the behaviour directly — but the figure offered to Mark as grounds for
reconsidering three unanimous rejections was inflated ~6×, and it is corrected in the doc before he
reads it. This is the scope trap aimed at a **binary** rather than a directory: my control ran, and
it ran against the wrong executable.

**The evaluator also found the synthesis both quorum rounds circled and neither reached (D4):**
scope `PROBE_MAX_TREE_NODES` to arm 33's own `env` line rather than raising it globally or deleting
it — measured free on the happy path (41/41, 47.1s) — which makes the ceiling structurally
unreachable inside the window while every other arm keeps the default. **Recorded, not applied**:
the revision budget is spent, and applying it would be a controller-invented resolution to a blocked
quorum. It is filed as row **6p**.

**Evaluator: PASS 78/100, 1 blocking, 4 non-blocking** (sonnet, its own detached worktree at
`b76b0823a`, distinct provider from the pi/deepseek designer and from the opus controller, so
generator≠judge holds). It reproduced **every** controller claim before ruling — C0 through C4
including T1's 43.76s timing — and it ran the addition-shaped mutant on the branch-count gate,
confirming the current gate at `expected_refusal_branches=24` is **silently blind** to an added
echo-shaped refusal branch (41/41 ok with the branch added). That is filed as row **6q**. Findings
2, 3 and 4 are applied to the doc verbatim.

**A live collision, attributed and left alone.** The fleet-account open-PR filter returned **3** —
`#997`, `#971`, `#945` — none with a branch in this clone's **16**-entry worktree list. `#971`
touches **exactly** row 6n's two files and is `MERGEABLE`/`UNSTABLE`, but branch `mission/iter306-*`
is V1's numbering (V1 is at 312, motoko at 31), so it is not attributable to this mission: read,
never touched, and handed over on the cross-mission channel. It neither supersedes this work nor is
superseded by it — `PROBE_TREE_DISCOVERY_SECS` is **0** occurrences at motoko HEAD in both files
(controls `PROBE_MAX_TREE_NODES` **2**/**3**, negative control **0**).

**Ruled out.**
- *"The node ceiling fires first on the controller's machine"* — my own E3 inference. **REFUTED** by
  E8. Do not rebuild a design on it.
- *"A CI runner could process 4096 iterations in under a second"* — the reviewers' shared premise.
  **Not refuted**, only bounded: the local margin is ~6–9×, measured against the right binary, and
  contention slows spawns, which moves it the protective way. The CI host's own rate is unmeasured.
- *"~52× margin"* — **my own error**, from the wrong binary. Superseded by ~6–9×.
- *"#971 supersedes row 6n"* — **false**, measured: #971 changes which deadline bounds discovery and
  leaves arm 33 asserting the generic wrapper.
- *"The race is something this doc introduces"* — **false**: it is PRE-EXISTING at HEAD, so it is a
  queue row (6p), not a revision.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Designer
**`pi:ollama/deepseek-v4-flash:0731-cloud`** — the rotation's next entry after the pointer's
`claude:claude-fable-5`; probe rc=0; two runs via `scripts/mission_pi_run.sh`, verdict `ok` both
(366s / 91s, 1 changed file each); pointer advanced to it. **No planner and no executor ran** — the
doc parked before a plan existed, so there was nothing to plan or execute. Evaluator **sonnet**, own
worktree. **Fable unspent.** Metered **$0.1587** of $5 (two quorum rounds at $0.0806 + $0.0781; the
pi lane is flat-rate $0.00). No GPU, no `rig.lock`. Gates on **darwin/arm64** against GNU bash
3.2.57; windows and ubuntu legs unrun locally.

**Bookkeeping second pick.** Row 6i's design doc and sprint plan moved `planned/` →
`implemented/v0_35_0/`, and the doc's status header corrected from `Planned` — which it still said a
day after the work merged — to the merge it actually landed as (`4bd58bef6`, PR #985).

**Gate 5 — no skill edit.** The rule that mattered most this iteration (reproduce a judge's finding
before believing it) worked and caught a real error of mine. My own failure — benchmarking the wrong
binary — is an instance of a rule the skill already carries (scope the known-positive control to the
same thing the check reads), mis-applied rather than missing, so it is a second instance of a
documented gap and not a new one. One candidate is **recorded and below the bar**: a bare
`ailang messages list --unread --json` returns **20** rows while `--limit 200` returns **41**, so the
default limit silently halves the triage queue and nothing in the output says so. One instance.

**Next.** Row **6o** (the SIGKILL-escalation half of the group kill), then **6p** (the race, with the
evaluator's D4 as the candidate fix), then **6q** (the blind branch-count gate), then 6j and 6m.
Decision ledger: **6** rows, **1 OPEN** — `D-MOTOKO-6N-1`, a ship-or-hold call with three lettered
options, a recommendation and a dated default.
