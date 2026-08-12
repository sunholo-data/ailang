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
