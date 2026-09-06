---
name: mission-control
description: Run ONE outer-loop iteration of a long-running mission (default: the V1 mission) — observe mission state, pick the top backlog item, route it through design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator with the mission's model routing policy, record a log entry, and run the retro. Use when user says "run mission control", "mission iteration", "work the v1 backlog", or when fired nightly by the dev.ailang.mission-control launchd job.
---

# Mission Control — one outer-loop iteration

Run ONE iteration of the mission defined in [`design_docs/v1-mission.md`](../../../design_docs/v1-mission.md)
(or the mission doc passed as argument). The gates run in order and are not skippable; earlier
gates are cheap and prevent expensive mistakes. This is the outer loop around the four honed
inner-loop skills — it does not duplicate them.

## Repo Profile (M-MISSION-PORTABILITY M2)

**One skill runs EVERY mission — never fork it per mission** (a fork undoes the Gate-5 self-
improvement loop, since retro fixes must benefit all missions). What differs per mission is a small
**profile**, read from two places:

> **⚠ A SKILL EDIT IS LIVE FOR EVERY MISSION THE INSTANT YOU SAVE IT — there is no sync step, and
> no "the sibling is insulated" grace period** (added 2026-07-31 iteration 125; instances: iter-123
> filed `#544` after finding `.claude/skills/` vs `.agents/skills/` duplicated with **31 of 38
> diverged**, and iter-125's planner refuted the controller's own cross-mission blast-radius claim).
> Measured on the rig, and the measurement corrected the first draft of this very note:
> `~/.claude/skills/mission-control` is a **SYMLINK to `<repo>/.claude/skills/mission-control`**
> (`readlink` confirms; both paths report the **same inode**). So the "global copy" and the "repo
> copy" are **ONE FILE**, not two — `ls -la` on the *file* hides this, because the symlink is on the
> *directory*. Editing the repo copy is instantly live for every mission on the rig, and
> `ailang-world` has **no repo-local `.claude/skills/` directory at all**, so it resolves through
> that same symlink. A **third**, genuinely separate, git-tracked and already-drifted copy sits at
> `<repo>/.agents/skills/mission-control/SKILL.md` (44,067 B vs 72,254 B) — that one is a real
> divergence and is what `#544` tracks. Two consequences: **(a)** a Gate-5 skill edit needs no
> copy-sync, but it DOES take effect for the sibling mission's next fire with no review gate of its
> own, so write it to be true for every mission, not just this one; **(b)** never infer
> cross-mission blast radius from the DRIVER alone — the drivers really are two byte-identical
> files (`diff -q` silent), and iter-125 cited that true fact for a broader "blast radius = zero
> until synced" claim that is FALSE, because the skill is shared by symlink. That is a Gate-2
> rule-3b **scope** error: a green check quoted for a sentence wider than it supports. Check the
> skill path separately from the driver — `readlink` before concluding anything about copies — or
> say "driver-only" and mean it. **(c) "The instant you save it" holds only for an edit SAVED IN
> THE MAIN CHECKOUT — a Gate-5 edit COMMITTED FROM A WORKTREE reaches origin and never reaches the
> running skill** (added 2026-08-01 iteration 128; instance 2 of the diverged-checkout class after
> iter-127 surfaced the divergence, and the first time the harm was measured). The symlink resolves
> to the MAIN CHECKOUT's *working tree*, so what the loop executes is that checkout's file at its
> own HEAD — not origin's. Iteration 128 measured its own rulebook: of the last 8 commits touching
> this file the newest TWO (`858b067d4`, `c7fc3b954`) were `NOT-in-local-HEAD`, while every older
> one was — the older ones had been saved in place, the new ones committed from worktrees. So the
> loop was executing a copy **missing iter-127's own Gate-4 STATUS-rotation mass-deletion guard**,
> a rule added because that step had already destroyed the charter's 1,571-line queue once. The
> drift is one-way and unbounded: every sprint lands from a worktree, so every Gate-5 edit made
> that way widens the gap — silently, while this very note promises the opposite. At Gate 1, after
> the origin fetch, **diff the RUNNING skill against origin**:
> `git show origin/dev:.claude/skills/mission-control/SKILL.md | cmp -s - "$(readlink -f ~/.claude/skills/mission-control)/SKILL.md"`
> **— and the second path MUST be the RESOLVED symlink target, never the relative
> `.claude/skills/...`, because the loop does not run from the checkout the symlink points at**
> (added 2026-08-21 V1 iteration 241; instance 1 is iteration 240, which caught it by hand and
> recorded the correction in its STATUS stamp without fixing the prescription, instance 2 is this
> iteration, which ran the command as written and got a green from the wrong file). The relative
> path resolves against the CWD — and the V1 driver executes from a **pinned worktree**
> (`~/.ailang-driver-pin/v1`), whose own `.claude/skills/mission-control/SKILL.md` is a DIFFERENT
> FILE from the one `~/.claude/skills/mission-control` resolves to. Measured: inodes `45326796`
> (pin worktree) vs `45241676` (main checkout). So the check silently compares origin against a
> copy nothing executes, and — this is what makes it worse than useless — it reads GREEN in
> exactly the situation the whole rule exists to detect, because the pin worktree is checked out
> at `origin/dev` **by construction** while the main checkout is the one that drifts. A stale
> running skill is therefore *guaranteed* to pass the prescribed form. Rule 3a's trap aimed at
> this gate's own instrument: pair it with `readlink` and assert the two paths are the same file
> before trusting either answer. Mission-independent — any mission whose driver runs from a
> worktree, a pinned checkout, or any CWD other than the symlink's target has the same hole, and
> the generalisation is this file's own recurring shape: **a relative path is a claim about where
> you are standing, not about which file runs.**
> — if it DIFFERS, read the delta (`git diff origin/dev -- <skill>`) BEFORE proceeding and say so
> in the report, because the rules you are about to follow are not the rules the mission agreed on.
> Prefer saving Gate-5 edits in the MAIN checkout when that tree is clean enough to commit from;
> when Principle 0 forbids that, land via the worktree AND escalate the reconcile as a human
> decision — the loop cannot fix its own rulebook by writing only to a tree nobody executes.

- **Driver env** (exported by `tools/launchd/mission-control.sh`): `MISSION_NAME` (default `v1`),
  `MISSION_REPO` (default `sunholo-data/ailang`), `MISSION_DOC` (default `design_docs/v1-mission.md`);
  the bookkeeping-issue number lives in `~/.ailang/state/mission-${MISSION_NAME}-gh-issue`
  (V1 falls back to `329`).
  **⚠ NAMESPACE THAT PATH — the bare `~/.ailang/state/mission-gh-issue` this profile used to
  prescribe is ONE FILE SHARED BY EVERY MISSION ON THE RIG, and Gate 5's rotation step WRITES to
  it, so a sibling's rotation week silently overwrites another mission's live inbound human
  channel with an issue number that does not exist in that mission's repo** (fixed 2026-08-21 V1
  iteration 246; proposed by `mission-world` iter-106 and corroborated first-party in V1's own
  running copy before adoption — sibling-claim ghost discipline). This is the THIRD instance of a
  class this file has already fixed twice and told itself to sweep: the roles table's
  designer-rotation note (2026-08-13, iter-188) closes with *"any `~/.ailang/state/` key this
  skill names as a literal is shared by all missions — audit the whole path list before adding
  another"*, and Gate 4's dashboard note (2026-08-17, iter-216) closes with *"one namespacing fix
  landed, the neighbouring literal did not."* Neither audit was run; this is what it would have
  found. **It is worse than the dashboard collision**, which is a read-then-overwrite a careful
  controller notices: this is a **write onto a path the writer never reads**, so the damage lands
  entirely on the mission that did nothing wrong, and its only symptom is a Gate-0 read pointing
  at nothing. Measured on the rig: `mission-gh-issue` = `745` and `-prev` = `635`, both V1's
  (control — `745` does not resolve in `sunholo-data/ailang-world`), while
  `mission-world-gh-issue`, `mission-world-gh-issue-prev`, `mission-motoko-gh-issue` and
  `mission-motoko-gh-issue-prev` all exist as files **hand-created by careful sibling
  controllers** that this skill's literal never read — the same signature the rotation key showed.
  The bare literal appeared **4** times in the running copy (Repo Profile, Gate 5's report step,
  and both halves of Gate 5 rotation step 4); positive controls, the two keys already namespaced,
  read 2 and 1, and a fresh invented literal read 0.
  **Migration, one line and safe in both directions:** on first read, if the namespaced file is
  absent but the bare one exists **AND its number resolves in `$MISSION_REPO`**, seed from it;
  otherwise treat the bare file as another mission's and never write it again. The
  resolves-in-my-own-repo predicate is the load-bearing half — it is what makes the migration
  safe for whichever mission currently owns the bare path. **And run the audit rather than fixing
  one key**: whenever this skill names a `~/.ailang/state/` path or a `design_docs/` filename a
  mission writes to, ask what the sibling writes there first. The tell: you are about to write to
  a path whose name contains no mission identifier.
- **The mission doc's charter header** — a `## Repo Profile` block (single source of truth,
  versioned with the mission): repo slug, bookkeeping-issue state key, the CI workflow names Gate 3b
  polls, and the **verify profile** name.
- **⚠ `MISSION_WORKDIR` MEANS TWO DIFFERENT DIRECTORIES BEFORE AND AFTER THE PIN RE-EXEC, AND EVERY
  GATE THAT READS IT RUNS *AFTER* — SO THE ENV FILE'S LITERAL IS NEVER WHAT EXECUTES** (added
  2026-08-23 motoko iteration 20; instance 1 is V1 iteration 241's `readlink` rule immediately
  above, which is this same "which copy runs?" question aimed at the SKILL path, instance 2 is this
  iteration's own `$REPO` defect one layer down). `tools/launchd/lib/pin-root.sh` re-execs each
  driver out of a worktree pinned to `origin/dev` and, deliberately, **exports
  `MISSION_WORKDIR=<pin worktree>` before the `exec`** so the charter and skill move with the
  script. The mission env file's `MISSION_WORKDIR=<source clone>` is therefore a *pre-pin default*,
  read once and overwritten — while the charter's Repo Profile, the plists and every human's mental
  model still say "working checkout". Measured from a live pinned session:
  `MISSION_WORKDIR=~/.ailang-driver-pin/motoko` beside
  `AILANG_DRIVER_SRC=~/dev/sunholo-data/ailang-motoko`, the latter **170 commits behind** with a
  `SKILL.md` 1,063 lines short. **Two consequences.** (a) Any notice, log line or record you build
  from `$MISSION_WORKDIR` (or from a `$0`-relative `REPO` derived from it) on the post-exec pass
  names the **throwaway worktree**, whose drift is `0` by construction — so a staleness report built
  that way is not merely wrong, it is *self-refuting*, and it reads as a clean bill of health. Use
  `AILANG_DRIVER_SRC` for the clone and `MISSION_WORKDIR` for what ran. (b) A checkout the charter
  names but nothing executes drifts **without bound and without symptom**: the pin succeeds every
  fire, so the failure path that reports staleness never fires, and the growth sits on the success
  path in a log nobody reads (measured: 119 → 132 → 144 → 159 → 170 over five fires before anyone
  looked). Mission-independent — every mission on this rig is pinned this way — and the
  generalisation is the one iteration 241 already earned, aimed at a variable instead of a symlink:
  **a path is a claim about where you were standing when it was set, not about which tree is
  running.** The tell: you are about to name a checkout in a record or a notice, and the variable
  you are interpolating was set by something other than the code that is executing now.

Wherever a gate below shows a literal `sunholo-data/ailang`, `design_docs/v1-mission.md`, or `329`,
that literal is the **V1 default** — use `$MISSION_REPO` / `$MISSION_DOC` /
`${MISSION_GH_ISSUE:-<the mission's default>}` so the same gate serves any mission. (War-story prose
below keeps its literal SHAs/issue numbers — only OPERATIVE commands parameterize.)

### Verify profiles — the mission doc names exactly ONE

Gates 1–3b run its commands instead of `make` literals:

| Profile | Rebuild-before-check | Full test suite | Binary staleness | Used by |
|---|---|---|---|---|
| `go-compiler` | `make quick-install && make build` (BOTH binaries) | `make test` | `~/go/bin/ailang` (PATH) + `bin/ailang` go stale independently — confirm `--version` == `git describe` before trusting output | **V1** (this repo compiles the toolchain) |
| `ailang-code` | `ailang install` (binary ships prebuilt — nothing to compile) | `ailang check` (types) · `ailang test` (tests) · `ailang ai-check` (unified check+verify) | binary is a released artifact, pinned in the mission's lockfile — no `-dirty` staleness class | **Ailang World** (an AILANG-code repo) |
| `docs-site` | nothing to rebuild — the site is the artifact | `make docs-build` (Docusaurus production build; the deploy workflow runs the same thing, so a green local build is the cheap pre-image of Gate 3b) · `make verify-examples` · `ailang check <file>` for any `.ail` touched | no dual-binary class — but the binary running `verify-examples` **can** be stale, so confirm `ailang --version` before quoting its output | **Docs** (the website upkeep loop) |

Under `ailang-code`, verification IS the binary's own gates: `ailang check` (types), `ailang test`
(tests), and `ailang ai-check` — the UNIFIED check+verify (types + Z3 in one JSON; do **not**
reinvent a split gate). Gate 2's Go-only steps (`make quick-install`, `bin/ailang` staleness,
`t.Skip` un-skip) apply to `go-compiler` **only**; under `ailang-code` the shipped binary is the gate.

Under `docs-site` there is no compile step and no dual-binary staleness class, so the `go-compiler`
rules about `~/go/bin/ailang` and `bin/ailang` drifting apart do not apply. Two things replace them.
**(a)** The gate is the *site build*, and it is path-filtered in CI (`docs/**`, `prompts/**`,
`llms.txt`, `CHANGELOG.md`) while `CI` itself has no push paths filter — so a docs-only commit still
runs full CI, and Gate 3b must wait for it rather than reading a path-filtered skip as "not
applicable". **(b)** The staleness class moves rather than vanishing: `make verify-examples` is only
as current as the `ailang` binary running it, which is exactly the trap `go-compiler` warns about,
wearing different clothes.


Everything else in this skill is already repo-agnostic and ports UNCHANGED: the directive-author
allowlist (`MarkEdmondson1234`), quorum-at-pick, the billing tripwire, the pidfile/overlap guard,
the rotation designer, and the weekly issue rotation. Namespaced state keys (M1) keep two missions
on one rig from colliding.

## Gate 0 — PREFLIGHT (deterministic; abort = exit silently with a controlplane message)

Deterministic preflight. Kill switch, git/gh identity, billing tripwire, dev CI, directives. Abort = exit silently.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-0-preflight.md` **NOW**, before doing anything in
Gate 0. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 1 — OBSERVE (cheap, read-only)

Cheap read-only observation. Charter head, last log entry, inbox, parked evaluations. Spends nothing.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-1-observe.md` **NOW**, before doing anything in
Gate 1. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 2 — PICK + REALITY-CHECK

Pick the top item and REALITY-CHECK it first-party. The judgement gate: most bad iterations start with an unchecked premise here.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-2-pick.md` **NOW**, before doing anything in
Gate 2. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 3 — ROUTE + EXECUTE (the inner loop, with the routing policy)

Route and execute: designer -> planner -> executor -> evaluator, with the routing policy and the cross-provider spawn recipes.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-3-route.md` **NOW**, before doing anything in
Gate 3. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 3b — CI GREEN (an item is not LANDED until remote CI passes on its merge)

An item is not LANDED until remote CI passes on its merge. Poll pinned to the SHA you pushed.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-3b-ci-green.md` **NOW**, before doing anything in
Gate 3b. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 4 — RECORD (append-only; the log is the mission's memory)

Append the log entry, refresh the dashboard, record the routing evidence row.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-4-record.md` **NOW**, before doing anything in
Gate 4. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Gate 5 — RETRO + REPORT

Retro, report on both channels, update the watermark.

**⚠ THE FULL RULES FOR THIS GATE ARE NOT IN THIS FILE.** Read
`.claude/skills/mission-control/resources/gate-5-retro.md` **NOW**, before doing anything in
Gate 5. It is the authoritative text; this stub is an index entry, not a summary you may
act on. Skipping the read is how a gate's rules silently stop applying.

## Standing rules

1. **One backlog item per iteration** (a bookkeeping-only pick allows taking a second).
2. **Never force through a guardrail** — park and report; the queue always has a next item.
3. **Commit per milestone** on `dev` (or the worktree branch); no pushes on the wrong gh account;
   NEVER release — stop at ready-to-release and report.
4. **The inner-loop skills are the contract** — improve them via Gate 5, don't bypass them
   mid-iteration because one is annoying. If a skill blocks you, that IS the retro finding.
5. **Data before conclusions** (PROGRAM.md invariant): no fix without a measured/reproduced
   failure; record refuted hypotheses in the log's Ruled out field.
6. **Every wait is bounded** (added 2026-07-12 after iteration 13 hung 4h in an unbounded
   `until COND; do sleep 30; done` — no worktree, no commit, claude idle at 0% CPU with a live
   `sleep` grandchild, until the 6h driver watchdog reclaimed the slot). ANY poll/wait you issue
   — CI (Gate 3b), a coordinator task, a background agent, an eval, a `make` step — MUST carry a
   hard ceiling: a `date +%s` deadline OR a max-iteration counter. On expiry, FAIL LOUDLY and
   park/report — never keep sleeping. Forbidden: a bare `gh run watch`, `while true`, or
   `until COND; do sleep …; done` with no cutoff. A headless iteration has no human to notice, so
   one unbounded wait burns the entire 6h slot. Default cap ≤30 min; treat expiry as a parkable
   failure, not an error to retry in place.
7. **Every wait is ACTIVE — NEVER END YOUR TURN WHILE A BACKGROUND AGENT OR BACKGROUND `Bash` IS
   STILL RUNNING** (added 2026-08-08 V1 iteration 167; proposed by `mission-world` iter-65, which
   shares this skill but cannot edit it, and corroborated first-party in V1's own driver log before
   adoption — sibling-claim ghost discipline). This is rule 6's mirror, and the two collide in a way
   that is lethal precisely because rule 6 is correct: rule 6 tells you to bound your waits, and the
   most natural way to "wait" for a background agent is to stop making tool calls and let the
   harness hold the slot for you. **That is the fatal move.** `claude -p` terminates still-running
   background tasks **600 s after the assistant's last turn ends**, prints
   `Background tasks still running after 600s; terminating.` — and exits **rc=0**. The driver then
   logs `iteration complete (rc=0)` and **neither watchdog fires**, because a clean exit code is
   exactly what they are built to ignore. So the slot ends with a plausible transcript, zero
   commits, zero charter rows, zero log entries: the vacuous-pass class this loop keeps closing
   elsewhere — *success reported for work that never happened* — now aimed at the loop itself. It is
   invisible afterwards to every check except Gate 2's died-mid-flight traces.
   **Attribution, measured, both missions, zero misses and zero false positives:** `grep -c
   'Background tasks still running after 600s'` returns **2** in `/tmp/ailang-mission-control.log`
   (V1, lines 3193 / 3420 = the 2026-08-07 12:26 fire → iteration 159, and the 2026-08-08 09:09
   fire → iteration 167 attempt 1, which died holding six freshly-measured verification rows) and
   **2** in `/tmp/ailang-mission-world.log` — World's *only* two orphaned slots in 67 iterations.
   Both V1 hits sit immediately above a transcript reading "Gates 0–2 complete; sprint-planner
   running", i.e. the controller had just spawned a background role and stopped.
   The per-gate `mission-heartbeat.sh` stamps above are the durable attribution contract for this
   failure class; never skip or defer a gate's first-action stamp.
   **The rule:** while any background work you spawned is outstanding, keep the turn alive with
   *chained bounded waits* — a `Monitor`, a bounded `date +%s` poll, or repeated short status reads
   — so the harness never sees you stop. Rule 6 still binds each individual wait; what changes is
   that expiry must be handled **by you, in-turn** (park, kill, report), never by going quiet and
   hoping. Two corollaries: **(a)** prefer a FOREGROUND spawn (`run_in_background: false`) whenever
   the work fits inside the tool's own limit — a synchronous call cannot be reaped this way at all;
   **(b)** the driver-side safety net is
   `export CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0`, landed in V1's
   `tools/launchd/mission-control.sh` at iteration 167 — `0` means "wait indefinitely", which is
   **not** an unbounded wait in rule 6's sense: it hands the bound to the driver's `HARD_TIMEOUT`
   and stall watchdog, both of which fail LOUDLY, replacing a silent 10-minute rc=0. Any mission
   whose driver lacks that export is still exposed; check it rather than assume it. The tell: you
   are about to write "the planner is running; I will report back", or to end a turn whose last
   tool call started something you have not yet read the result of.
   **⚠ THE CEILING=0 SAFETY NET DOES NOT HOLD THE SLOT, AND INSTALLING IT DELETED THE ONLY WAY TO
   SEE THAT IT DIDN'T** (added 2026-08-11 V1 iteration 176; instance 3 of this gap, and the first
   under the post-fix regime). Everything above treats (b) as the belt behind (a)'s braces. It is
   not. Measured on V1: the 11:23:12 fire ran with `bg-wait-ceiling=0ms` — confirmed in its own
   driver banner — spawned a background executor, and its controller reasoned, in the transcript,
   *"bg-wait ceiling is `0` (indefinite) — the 600s reaper is disabled, so waiting on the Monitor is
   safe … I'll wait for the next event"*. The slot then logged `iteration complete (rc=0)` **nine
   minutes later** while the executor kept writing for a further twelve. Zero commits, zero charter
   rows, zero log entries — the same vacuous pass as before, arriving faster and quieter. So read
   (b) narrowly: it removes the harness's 600-second *reap*, it does **not** make ending your turn
   safe, and nothing in a `claude -p` run will re-invoke you when a `Monitor` fires. **A `Monitor` is
   an event stream, not a wait.** Chain bounded in-turn polls, or spawn with
   `run_in_background: false`, exactly as (a) says — (b) never relaxes (a).
   **And fix your instrument before you use it to look for this.** `grep -c 'Background tasks still
   running after 600s'` is the attribution tell this rule prescribes, and it is now wrong in **both**
   directions. It is **blind by construction** to every post-fix instance, because the ceiling
   suppresses the very line it greps for — so a mission with the fix installed reads clean while
   dying exactly as before. And it **over-counts**: on V1 it returns **3** against **2** real reap
   lines, the third being iteration 167's own log record *quoting the count*. This file already
   warns that a self-describing file poisons a known-ABSENT control; this is the same trap sprung on
   a known-PRESENT one, which is worse — an absent control that fires announces itself, while a
   present one that over-counts just looks like more evidence. Attribute a dead slot by the shape
   instead: an `iteration complete (rc=0)` whose elapsed time is far below the work it claims, sitting
   above a transcript that ends by announcing a wait. Then confirm with Gate 2's died-mid-flight
   traces (a stale worktree, an orphaned output file still growing after the exit) — those are
   mechanism-independent, which is the whole reason they outlive each fix.

   **⚠ AND `pgrep` IS NOT AN AUTHORITATIVE LIVENESS SIGNAL ON THIS RIG — IT FAILS IN BOTH
   DIRECTIONS, AND ITS FALSE *NEGATIVE* IS THE ONE THAT ENDS YOUR TURN** (added 2026-08-20 V1
   iteration 235; instance 1 is iteration 234's worktree rule, which prescribes `pgrep` and then
   read its output backwards, instance 2 is this iteration's own executor poll). Rule 7 tells you
   to chain bounded waits rather than go quiet. It does not say what to POLL, so the natural
   choice is "is the process still alive?" — and that question is answered by an instrument this
   loop has now mis-read three times. **False negative:** a `pgrep -f 'codex exec --model …'`
   liveness loop reported **"codex process gone"** while codex was demonstrably still working —
   its output file grew from **70 KB to 693 KB** afterwards, and the worktree diff was still
   empty at the moment the poll declared completion. Had the turn ended there, the run would have
   been abandoned mid-flight and the slot would have exited **rc=0**, i.e. exactly the vacuous
   pass rule 7 exists to prevent, reached *through* rule 7's own remedy. **False positive:**
   `pgrep -fl codex` returns **four** `ChatGPT.app` helper processes on this rig, and
   `pgrep -f 'claude-fable-5'` matches every long-lived interactive session — so a non-empty
   result is not evidence either. The pattern is doing the work, and a pattern that must match a
   process's *full argv* is fragile by construction: the argv may be truncated, the process may be
   wrapped by `sh`/`exec`, and a substring you chose can match something unrelated.
   **Rules. (a)** Poll the **artifact**, not the process — the output file's size, the final-message
   file's existence, the worktree diff, the task's own notification. An artifact is produced by the
   work; a process name is produced by the shell. **(b)** The harness's own completion notification
   is authoritative and `pgrep` is not: never conclude "done" from `pgrep` while the notification
   for that task has not arrived. **(c)** When you do use `pgrep`, pair it with a known-positive
   control in the same call (rule 3a aimed at a process table) — a pattern that matches nothing
   anywhere is broken, not informative — and treat an EMPTY result as *unknown*, never as *dead*.
   **(d)** This NARROWS the worktree rule's "run `pgrep` and **believe the output**": believe a PID
   it returns (that direction is safe — something matched), never believe its silence. The tell:
   you are about to act on "the process is gone" and the only thing that told you so printed
   nothing.

   **⚠ CLAUSE (b) IS TRUE OF THE *TASK* AND FALSE OF THE *WORK* — A COMPLETION NOTIFICATION REPORTS
   THE COMMAND YOU HANDED THE HARNESS, AND IF THAT COMMAND BACKGROUNDED ITS OWN CHILD, IT REPORTS
   `exit code 0` THE INSTANT THE LAUNCHER RETURNS** (added 2026-08-22 V1 iteration 249; three
   first-party instances in one iteration). Clause (b) says the notification "is authoritative", and
   for a foreground command it is. But this file's own recipes hand you the counter-example: the
   codex and pi recipes both wrap the real work in `( … ) &` plus a `date +%s` deadline loop, and
   the worktree rule tells you to create a 23k-file checkout with `run_in_background: true`. A
   launcher script whose last statement is `&` has *finished* — so `rc=0` is a true statement about
   a shell that did nothing but fork. Note it is the same shape as Gate 0's `gh issue close
   --comment`: **an exit code describes the request, not the delivery** — arriving here at the one
   signal clause (b) elevates above every other.
   Measured this iteration, three times, all benign only because the polls caught them: a
   `go build` launcher notified `completed (exit code 0)` while the 96 MB binary was still being
   written — a `test -f` moments later printed `INSTRUMENT FAILURE — no binary`, and the artifact
   was present shortly after with `rc=0` in its own file; a `git worktree add` notified complete
   while `git` was still at *"Updating files: 1% (345/23852)"*; and a five-gate baseline sweep
   notified complete before its first gate had run. Read strictly, clause (b) authorises acting on
   all three — and acting on the second is precisely iteration 234's half-built-worktree trap,
   whose `git status` reports 23,835 files as deleted.
   **Rules. (a)** A notification for a command that contains `&` means *launched*, not *done* —
   have the backgrounded body write its own terminal marker (`echo DONE > …`, `echo "rc=$?" > …`)
   and poll for THAT, per clause (a)'s artifact discipline aimed at the notification rather than at
   `pgrep`. **(b)** Prefer not backgrounding twice: if you pass `run_in_background: true`, let the
   command run in the foreground *inside* it, so the harness's notification and the work's
   completion are the same event. Double-backgrounding is what breaks (b), and it is easy to do by
   accident when copying a recipe that already backgrounds. **(c)** Keep the rc-file convention the
   codex recipe already uses — the code you want is the *work's*, and it must be captured without a
   pipe (step 3) inside the backgrounded body. **(d)** A `test -f` immediately after a notification
   is not a refutation either: treat an absent artifact as *not yet*, and let the bounded poll
   decide. Mission-independent, and it generalises past this harness: **wherever a launcher and its
   work are different processes, a status from the launcher is a claim about the fork.** The tell:
   you got a completion notification, and the command you launched ended in `&`.

   **⚠ AND THE ARTIFACT THAT REMEDY SENDS YOU TO IS AN INSTRUMENT TOO — ASSERT IT IS *FRESH*, NOT
   MERELY PRESENT, BECAUSE A PATH YOU WRITE TO TWICE HOLDS THE PREVIOUS RUN'S ANSWER AND READS AS A
   RESULT** (added 2026-08-22 V1 iteration 247; instance 1 is iteration 244's worktree-readiness
   poll, instance 2 is this iteration's probe harness). Clause (a) immediately above is correct and
   it points the whole loop at artifacts — output-file size, a final-message file, a worktree diff,
   a built binary — precisely because processes lie. That is a good trade and it hands you a new
   failure mode with no guard: **a process cannot be stale, and an artifact can.** `pgrep` at worst
   tells you nothing; a leftover file tells you something *specific and wrong*, in the voice of a
   measurement.
   Note where this loop's existing remedy sits, because the shape is this file's own named one.
   The codex recipe already says to give the directive file a per-iteration name — *"a fixed name
   collides with the previous iteration's leftover"* — which is exactly this defect, guarded for
   **one file in one recipe** while every other probe artifact you write to a fixed path is
   unguarded: *guard the helper, miss the call site*.
   Two shapes, and they fail in opposite directions. **(a) Present but not yet COMPLETE.**
   Iteration 244's readiness poll greened on `grep -q .` against a log `git` was still writing
   progress into, so work that had not started read as work that had finished. **(b) Present but
   from a PREVIOUS run.** Iteration 247 built a compiled-vs-interpreted harness around a hardcoded
   generated-package subdirectory; when codegen emitted a different directory name the `go build`
   silently failed, and the harness executed the binary **left over from the previous round**. It
   then reported three byte-identical runs and zero heap addresses — a green, for a determinism
   property, from a binary that no longer corresponded to the tree under test. Nothing in the
   output said "stale"; it was caught only because the *paired* interpreted-vs-compiled comparison
   came back rc=1 with content that did not match either arm, and the natural reading of that is
   "the fix regressed", not "my instrument is lying".
   **Rules. (a)** DELETE the artifact before the run that is supposed to produce it, and assert it
   EXISTS afterwards — `rm -f out; cmd; test -f out || echo "INSTRUMENT FAILURE — not a verdict"`.
   Absence after a real run is loud; a leftover is silent. **(b)** Prefer a per-invocation path
   (`/tmp/x_iter<N>_round<K>`) over a fixed one, generalising the codex recipe's rule from its one
   file to every artifact you read back. **(c)** Assert the artifact is NEWER than the input that
   produced it, not merely non-empty — `[ out -nt input ]` — which catches (b) even when the path
   is reused. **(d)** Never let a build's exit code go unread on the way to running its output:
   `go build ... ; rc=$?` then refuse to execute on non-zero, since "the binary is missing" and
   "the binary is old" are the same command line away. **(e)** When a paired comparison disagrees
   with a same-run single reading, suspect the INSTRUMENT before the code — that disagreement is
   the only tell either shape produces. Mission-independent, and it generalises past files to any
   reused sink: a scratch directory, a database row, a fixed branch name, an `--out` target. The
   tell: you are about to read a path that a previous invocation also wrote, and nothing between
   the two runs removed it.
   **⚠ AND REMEDY (a) IS VOID WHILE THE PREVIOUS WRITER IS STILL ALIVE — `rm -f` DELETES A FILE,
   NOT A FILE DESCRIPTOR, SO A POLLER YOU THOUGHT WAS FINISHED RE-CREATES THE PATH AND ITS VERDICT
   LANDS IN YOUR NEW LOG AS IF IT WERE ABOUT YOUR NEW SUBJECT** (added 2026-09-02 V1 iteration 320;
   instance 1 is iteration 247's rule immediately above, whose remedy (a) I followed to the letter,
   instance 2 is this iteration, where following it produced the failure). Clause (a) is written
   about a *previous* run — one that has ENDED, leaving a corpse. It says nothing about a
   *concurrent* one, and standing rule 7 is what manufactures those: it tells you to bound every
   wait with a `date +%s` deadline, so a 30-minute CI poller keeps running long after the thing it
   watched has been superseded. Delete its log and it simply appends again. Note the two failure
   surfaces differ in kind: the `.done` marker is written by whichever writer finishes FIRST, so it
   can be the stale one; and the log ends up INTERLEAVED, which is worse than stale, because the
   two verdicts are individually true and their juxtaposition is not.
   Measured here, at the gate that decides LANDED vs parked: a first poller was watching a
   pre-rebase head; I force-pushed, `rm -f`'d both artifacts and launched a second poller on the new
   head; the first then wrote **`ALL COMPLETE`** into the fresh log — a correct verdict about a
   commit that no longer existed — and created `.done`. A `tail` showed `ALL COMPLETE` at the top
   and `pending=3` at the bottom of the same file. Read as prescribed, that is a green for a run
   still in flight. It was caught only because the two lines contradicted each other, i.e. by rule
   (e), not by rule (a).
   **Rules. (a-bis)** Before reusing an artifact path, prove the previous writer is DEAD — not that
   the file is gone. In this harness a background task's completion notification is that proof;
   `pgrep` is not (standing rule 7's own amendment: believe a PID it returns, never its silence).
   **(b-bis)** Better, obey (b) rather than (a): give every poller a per-invocation path
   (`/tmp/ci_iter<N>_head<sha7>.log`), which makes the question moot and, unlike (a), cannot be
   defeated by a writer you forgot about. **(c-bis)** Put the SUBJECT in the verdict —
   `ALL COMPLETE for <sha>` rather than `ALL COMPLETE` — so a stale line identifies itself the
   moment it is read; that one change turns this whole class from silent into loud. **(d-bis)** When
   a superseding event happens (a force-push, a rebase, a re-run), KILL the poller watching the old
   subject rather than letting its deadline expire; a bounded wait is bounded in time, not in
   relevance. Mission-independent, and the generalisation is this file's own recurring shape aimed
   at the intersection of two of its own rules: **rule 7 tells you to spawn bounded background
   watchers, and the stale-artifact rule assumes writers stop when you stop caring — the two are
   only jointly safe if the artifact path names its subject.** The tell: you deleted a file to get
   a clean reading, and something that was writing to it before you deleted it has not notified you
   that it finished.
   **⚠ AND EVERY WORD OF RULE 7 IS ADDRESSED TO THE CONTROLLER, SO THE SUB-AGENTS THIS SKILL SPAWNS
   INHERIT NONE OF IT — A ROLE THAT ENDS ITS TURN ON A WAIT COSTS YOU A WHOLE RESUME CYCLE, AND ITS
   REPORT IS SIMPLY ABSENT** (added 2026-09-01 motoko iteration 32; instance 1 is iteration 176's
   controller, which read this very rule and then reasoned *"I'll wait for the next event"*, instance 2
   is this iteration's EVALUATOR, which ended its turn with *"I'll pause here and wait for the Monitor
   notification to arrive before compiling the report"*). Rule 7 and its four amendments are complete on
   the question *"what must I not do while background work runs?"* — and every one of them is written in
   the second person to the loop's own session. The designer, planner, executor and evaluator never read
   this file. For a cross-provider role that is obvious; the trap is the **Agent-tool** roles, which run
   on the same harness, hit the same `Monitor`-is-not-a-wait semantics, and are the ones you are most
   likely to assume already know. Guard the helper, miss the call site — this file's own named shape,
   aimed at the directives it tells you to write.
   Note the failure is quiet in the direction that matters: the sub-agent does not crash and does not
   report an error, it returns a *plausible* closing sentence announcing an intention. Iteration 32's
   evaluator returned exactly one line of output after 37 minutes and 93 tool calls; the score, the
   findings and the mutation drills existed nowhere. Recovering it cost a `SendMessage` resume, and the
   resumed run then produced the single most valuable result of the iteration (the T3 drill neither the
   executor nor the controller had run). Had the controller banked that one-line return as the
   evaluation, the iteration would have landed with no judge — standing rule 7's vacuous pass, arriving
   through a role rather than through the slot.
   **Rule.** Every spawn directive for a role that may background work carries the operative half of
   rule 7 in its own words: *poll the artifact inside a bounded `date +%s` loop; a `Monitor` is an event
   stream, not a wait; nothing will re-invoke you when it fires, so do not end your turn announcing one.*
   Two corollaries. **(a)** When a role returns a suspiciously short result — a single sentence, no
   deliverable, a stated intention — treat it as **not finished** rather than as a verdict, and resume it
   by name (`SendMessage` keeps its transcript, so nothing is re-run). **(b)** A resumed role must be
   told what to do about the thing it was waiting for: recover the result from disk, or declare it
   UNMEASURED — otherwise it waits again. Mission-independent: every mission on this rig spawns the same
   four roles through the same harness. The tell: a sub-agent's last sentence contains the words "wait",
   "pause" or "once it completes", and there is no deliverable attached to it.

8. **THERE ARE TWO KINDS OF PARK AND THIS SKILL ONLY NAMES ONE — A DOC WAITING ON A QUOTA BUCKET IS
   NOT `needs-human-review`, AND FILING IT AS ONE MANUFACTURES A DECISION THE HUMAN DOES NOT HAVE**
   (added 2026-08-19 V1 iteration 229; two consecutive first-party frictions, 228 and 229). Every
   park in this file resolves to one state: Gate 2's quorum flow parks `needs-human-review`, Gate 3's
   round-3 evaluator failure parks `needs-human-review`, Gate 3b's timeout parks
   `needs-human-review`. That is correct whenever the blocker is **judgment** — a contested design
   direction, a red nobody can attribute, an objection needing a call only the human can make. It is
   wrong, and expensively wrong, whenever the blocker is **capacity**: a model lane that is
   quota-exhausted, a rotation entry that is structurally incapable of the role, a provider outage.
   The two are indistinguishable once written down, and they have opposite resume conditions — a
   judgment park waits indefinitely and *must* appear in Gate 5's `DECISIONS FOR MARK`; a capacity
   park unblocks on a **clock**, needs no ask, and must NOT appear there at all.
   The failure mode is not merely cosmetic. Gate 0 unparks by looking for an allowlisted directive,
   so a capacity park filed as a judgment park can only be cleared by a human answering a question
   that was never real — and the human's queue is this loop's scarcest channel, which Gate 5 already
   protects with a hard digest cap. In the other direction the next iteration inherits a park with no
   machine-checkable resume predicate and must reconstruct the entire routing story from prose to
   learn that it could simply have re-run.
   Measured, both iterations, same rotation and same shape: `codex:gpt-5.6-sol` probed **rc=1**
   ("usage limit … try again at Aug 20th, 2026 5:34 AM"), the next rotation entry
   (gemini/managed_agents) is **read-only under `CapRemoteSandbox`** and cannot author a file at all,
   so the designer resolved to Fable as a fallback — where the Fable diet allows **one** bounded run
   per iteration. Iteration 228 spent two (create + revision) and FLAGGED the diet violation;
   iteration 229 hit the identical wall on a round-1 block and declined to repeat it, at which point
   there was no state in this skill that says *"a revision is owed, nothing is being asked of anyone,
   and the lane returns at 05:34."*
   **Rule.** Classify every park before writing it. **(a)** Judgment → `needs-human-review`, and it
   carries a one-word-answerable ask into the Gate-5 DECISIONS row, as today. **(b)** Capacity →
   **`PARKED-ON-LANE`**, and it MUST name three things: the role that could not run, the lane that
   refused **with the command and its rc**, and the time or condition the lane returns. **(c)** A
   `PARKED-ON-LANE` item never enters DECISIONS and never counts as a decision in the ledger — if you
   find yourself writing an ask whose answer is "wait", you have misclassified it. **(d)** Its resume
   is a **predicate, not a narrative**: the next iteration re-probes the named lane and proceeds on
   rc=0, exactly as the blocked-external-row rule requires the predicate be run as a command rather
   than transcribed. **(e)** When a lane park recurs for the same role in consecutive iterations,
   that is a routing-policy signal, not a fact about the item — surface it in the report, because the
   loop cannot widen its own rotation but the human can.
   Mission-independent by construction: every mission on this rig shares one rotation file and the
   same quota buckets, so all three hit this the same way. The tell: you are about to file
   `needs-human-review` and the sentence explaining why contains a **reset time**, a **quota**, or
   the phrase "no other route".

## Current State

> **⚠ POSITION IS DELIBERATE — THIS BLOCK IS LAST BECAUSE IT IS VOLATILE.**
> Every line below runs a command at load time, so its output differs on every fire.
> Prompt caching keys on a byte-identical PREFIX, so anything placed AFTER volatile
> content can never be served from cache. Keeping the ~12k-token stable body above this
> block is what lets it cache between fires; moving this section up would silently
> forfeit that for everything below it.

**⚠ EVERY LINE BELOW IS A V1-SHAPED DEFAULT — namespace it per the ACTUAL mission before trusting
it** (added 2026-08-28, docs-mission iteration 0; instance 5 of the "bare `~/.ailang/state/`
literal in this shared skill is fleet-shared, not per-mission" class this file has already fixed
four times over — designer-rotation, `mission-gh-issue`, `mission-dashboard.md` — and each time
called for auditing "the whole path list", which this is). `tools/launchd/mission-control.sh`
itself resolves the kill switch to `mission-control.disabled` **for V1 only** (bit-for-bit legacy
compat) and to `mission-${MISSION_NAME}.disabled` for every other mission — so the bare literal
below is CORRECT for V1 and WRONG for docs/world/motoko, and a controller who runs it as-written
for a non-V1 mission gets a confident "armed" reading from a file that mission never writes to.
Measured first-party: docs-mission's own charter STATUS names `~/.ailang/state/mission-docs.disabled`
as ITS kill switch; checking the bare path instead would have silently missed a real disable. The
"Queue head" and "Last log entry" lines have the identical defect one file over — `v1-mission.md`/
`v1-mission-log.md` are V1's own filenames, not this skill's.
- **Kill switch**: !'K="$HOME/.ailang/state/mission-control.disabled"; [ "${MISSION_NAME:-v1}" = "v1" ] || K="$HOME/.ailang/state/mission-${MISSION_NAME}.disabled"; test -f "$K" && echo "DISABLED — STOP ($K)" || echo "armed ($K)"'
- **Branch / tree**: !'git branch --show-current && git status --porcelain | head -5'
- **gh account**: !'gh auth status 2>&1 | grep -E "Active account|Logged in" | head -2'
- **Queue head**: !'grep -A2 "^## Queue" "${MISSION_DOC:-design_docs/v1-mission.md}" | tail -2'
- **Last log entry**: !'grep "^## " "design_docs/${MISSION_NAME:-v1}-mission-log.md" | tail -1'
- **Unread inbox**: !'ailang messages list --unread 2>/dev/null | head -8 || echo "none"'
- **Parked evaluations**: !'ls .ailang/state/evaluations/ 2>/dev/null | tail -3 || echo "none"'

> Use the injected data above first; re-run only if empty or stale.
