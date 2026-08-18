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
