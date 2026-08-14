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
