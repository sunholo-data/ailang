---
name: mission-control
description: Run ONE outer-loop iteration of a long-running mission (default: the V1 mission) — observe mission state, pick the top backlog item, route it through design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator with the mission's model routing policy, record a log entry, and run the retro. Use when user says "run mission control", "mission iteration", "work the v1 backlog", or when fired nightly by the dev.ailang.mission-control launchd job.
---

# Mission Control — one outer-loop iteration

Run ONE iteration of the mission defined in [`design_docs/v1-mission.md`](../../../design_docs/v1-mission.md)
(or the mission doc passed as argument). The gates run in order and are not skippable; earlier
gates are cheap and prevent expensive mistakes. This is the outer loop around the four honed
inner-loop skills — it does not duplicate them.

## Current State

- **Kill switch**: !'test -f ~/.ailang/state/mission-control.disabled && echo "DISABLED — STOP" || echo "armed"'
- **Branch / tree**: !'git branch --show-current && git status --porcelain | head -5'
- **gh account**: !'gh auth status 2>&1 | grep -E "Active account|Logged in" | head -2'
- **Queue head**: !'grep -A2 "^## Queue" design_docs/v1-mission.md | tail -2'
- **Last log entry**: !'grep "^## " design_docs/v1-mission-log.md | tail -1'
- **Unread inbox**: !'ailang messages list --unread 2>/dev/null | head -8 || echo "none"'
- **Parked evaluations**: !'ls .ailang/state/evaluations/ 2>/dev/null | tail -3 || echo "none"'

> Use the injected data above first; re-run only if empty or stale.

## Gate 0 — PREFLIGHT (deterministic; abort = exit silently with a controlplane message)

1. Kill switch set → STOP (no message needed; this is the intended off state).
2. `gh auth status` must show `sunholo-voight-kampff` before any push. Wrong account → fix with
   `gh auth switch --user sunholo-voight-kampff` or park all push steps.
3. Dirty working tree in the main checkout → do NOT stash/checkout (Critical Principle 0).
   Doc-only edits (mission doc, log) may proceed; sprint work goes to a coordinator worktree anyway.
4. Unread inbox messages: triage per agent-inbox skill. A genuine regression or human directive
   OUTRANKS the queue — it becomes this iteration's pick.

## Gate 1 — OBSERVE (cheap, read-only)

**Sync to origin FIRST — the local checkout LIES when a prior run merged via GitHub** (added
2026-07-12 iteration 12; second instance of the same gap — iteration 9's watch-list already flagged
"add a resume-detection step to Gate 2", and iteration 12 booted on a stale local dev that was 2
commits behind origin/dev with the picked item ALREADY merged+recorded, yet the local mission
log/queue/sprint-JSON read as "mid-flight iteration 11" and drove a full redundant re-evaluation
before the Gate-3b fetch caught it). Before reading ANY local mission state:

```bash
git fetch origin
git rev-parse --short dev origin/dev          # differ? origin is ground truth
git log --oneline dev..origin/dev             # commits your working tree is missing
```

If local dev is behind origin/dev, read the mission doc + log + queue tags FROM ORIGIN
(`git show origin/dev:design_docs/v1-mission.md`, `…:v1-mission-log.md`) — a GitHub squash/merge
advances origin/dev without touching the local ref, so the working-tree copies are stale. Do NOT
pull/reset the shared main tree (Critical Principle 0 — it may hold a sibling's uncommitted work);
treat origin as truth, and if you need the code, branch a worktree from `origin/dev`.

Read: the mission doc (queue, guardrails, routing policy — they may have changed), the last 1–2
log entries (especially **Next** and **Ruled out** — do not re-chase), any parked
`needs-human-review` items that got human answers in the inbox.

**Check dev CI first — PER WORKFLOW, never a raw run list** (sharpened 2026-07-10 iteration 3:
a raw `--limit 6` list was flooded by Dependabot-Updates entries and read as green while dev CI
had been red for 3h; Build-and-Release and Docs-Deploy were equally invisible — TWO recorded
frictions, one gap):

```bash
for wf in "CI" "Build and Release" "Deploy Documentation to GitHub Pages"; do
  gh run list --workflow "$wf" --branch dev --limit 1 \
    --json conclusion,headSha --jq '.[0] | "'"$wf"': \(.conclusion) @ \(.headSha[0:9])"'
done
```

Any non-success → a RED dev outranks the queue (added 2026-07-10 per Mark; that day's red was a
pre-existing gofmt miss + a newly published stdlib vuln — neither from a sprint, both invisible
to local gates). Diagnose via `gh run view <id> --log-failed` — and check whether the SAME
failure exists on the parent commits before blaming any merge (iteration 3's three reds all
pre-dated the sprint; one first appeared on a docs-only commit). The fix (or a reasoned
allowlist/revert) IS this iteration's first deliverable. Time-based reds (new vuln advisories,
runner-image changes un-hiding latent bugs, dependabot peer-dep breaks) hit whoever observes
next — that's the mission's job now.

## Gate 2 — PICK + REALITY-CHECK

Take the top `[NEXT]` queue item. **Before any work, verify the doc's claimed status against repo
reality**: `git log --grep`, does the code/test already exist, does `make test` already cover it.
A design doc's status header is a claim, not a fact (M-EVAL-BENCH-UI shipped fully while its doc
said Planned for a month). **Also confirm the item is not ALREADY LANDED on origin** — check the
`origin/dev` queue tag (`git show origin/dev:design_docs/v1-mission.md | grep`) and any merged PR
(`gh pr list --search "<item> in:title" --state merged`) BEFORE starting a "resume" — iteration 12
ran a full redundant re-evaluation of an item that had already merged, because it trusted the stale
local queue/sprint-JSON (Gate 1's origin-sync now front-runs this, but re-check per item too). If
already done → the iteration's deliverable is the bookkeeping (move doc to implemented/, update
queue, log it) and you pick the NEXT item too.

**A queue row sourced from a survey/strategy review inherits that survey's verification debt —
live-repro the claimed bug BEFORE any routing** (added 2026-07-13 iteration 25; second instance
of the ghost class): a 10-minute `ailang check`/run probe at HEAD beats a design-doc sprint on a
phantom. Iteration 18's two "VERIFY-then-route" items were both ghosts (that tag saved them);
iteration 25's R4a/R4b were tagged as 2–3d NEW-DOC sprints yet were ALSO ghosts — R4a's design
doc had been archived Not-Applicable two months earlier, R4b was fixed in v0.7.0, and the
sourcing review's own Verification Log admitted "footgun list … not re-verified individually"
(4 of 7 survey-sourced rows so far were ghosts or mislabeled — a third, m-lambda-open-record-
pattern, was tagged NEW-DOC while a full design doc existed). Ghost → close with a CI-enforced
regression guard (example or test), never bare bookkeeping — that's what makes the close durable.

**Verification protocol** (added iteration 1 after three same-class frictions):
1. **Rebuild before any live check**: `make quick-install && make build` — BOTH binaries.
   `~/go/bin/ailang` (PATH) and `bin/ailang` (preferred by test helpers when present) go stale
   independently; a stale one silently falsifies results (1a: stale installed binary showed
   pre-fix behavior; 1b-eval: Jun-26 `bin/ailang` v0.26.0 broke `make test` with a phantom
   `_io_flush` error). Confirm `--version` matches `git describe` before trusting output.
2. **A parked test is a claim, not evidence**: `t.Skip`-ed / disabled tests say "nobody
   re-checked", not "still broken". Un-skip and RUN before treating the bug as open — the
   M-TYPEENV-SUB "open P0" was already fixed; only un-skipping revealed it.
3. **Exit codes through pipes lie**: `cmd | tail; echo $?` reports tail's status. Use direct
   invocation or PIPESTATUS.
4. **The shared main checkout is mutable mid-iteration** (added 2026-07-10 iteration 4, TWO
   frictions: a sibling agent opened a conflicted merge in the main tree mid-iteration, turning
   the Gate-2 rebuild `-dirty` — binaries built from a half-merged tree; and a persisted `cd`
   into a worktree made a later "main-tree" check read the WORKTREE's `.git` and report the
   merge cleared when it wasn't). Rules: (a) Bash cwd persists across calls — before trusting
   any main-tree git check, re-confirm `pwd` or use absolute paths; (b) re-run `git status` at
   the moment of use, not from memory — a clean tree at preflight proves nothing an hour later;
   (c) if `MERGE_HEAD` exists (a sibling's in-progress merge), do NOT commit in the main tree —
   your commit would complete THEIR merge; integrate via a worktree branch + PR with
   `gh pr merge --auto` instead (worked cleanly: PR #336); (d) a `-dirty` version suffix on a
   rebuilt binary means the tree changed under you — rebuild inside the isolated worktree.

## Gate 3 — ROUTE + EXECUTE (the inner loop, with the routing policy)

Apply the mission doc's **model routing policy** (read the charter's routing table — it is the
source of truth and changes for quota; as of 2026-07-11 the controller/design/evaluation roles
run on the controller's own model, Opus, with the independence caveat noted there; sprint-planner
+ sprint-executor = Opus; deterministic mechanical work = Sonnet).

- No design doc yet → invoke **design-doc-creator** (its hard gates apply: live `ailang check`
  verification, Conflict Surface for parser/types/codegen).
- Design doc but no plan → **sprint-planner** → sprint JSON + handoff.
- Plan exists → **sprint-executor** in an isolated worktree (coordinator-managed or
  `git worktree add` — NEVER the shared main tree; concurrent agents stomp uncommitted work).
- Execution complete → **sprint-evaluator**. Max 3 rounds; on round-3 fail →
  `needs-human-review`, park, message controlplane.

**GPU rule (two-tier)**: default iterations never touch `rig.lock` — it is a GPU mutex only.
If (and only if) a step drives ollama/local models: `source tools/launchd/rig-lock.sh &&
rig_lock_acquire wait` around THAT STEP, release immediately after. Ask explicitly at routing
time: "does this step touch the GPU?" — never let a test reach it by accident.

**Multi-week strategic items**: do not execute — the iteration's deliverable is DECOMPOSITION
into sprint-sized design docs (≤3–4 days each), queued individually.

## Gate 3b — CI GREEN (an item is not LANDED until remote CI passes on its merge)

After any push to dev, wait for CI **with a hard deadline** (Standing rule 6). A headless run has
no human to notice a hang, and a bare `gh run watch … --exit-status` blocks FOREVER if the run
never leaves `queued` (no runner). Iteration 13 (2026-07-12) wedged 4h in exactly this class of
unbounded poll — an `until COND; do sleep 30; done` whose condition never came true — before the
6h driver watchdog reclaimed the slot. Use a BOUNDED poll that fails loudly on expiry (portable;
there is no GNU `timeout` on the rig):

```bash
rid=$(gh run list --branch dev --workflow CI --limit 1 --json databaseId --jq '.[0].databaseId')
[ -n "$rid" ] || echo "Gate 3b: no CI run for HEAD yet — re-list a few times, still bounded"
deadline=$(( $(date +%s) + 1800 ))            # 30-min cap; CI is ~15-20m — never open-ended
while :; do
  st=$(gh run view "$rid" --json status,conclusion --jq '.status + " " + (.conclusion // "")')
  case "$st" in "completed "*) echo "CI: $st"; break ;; esac
  [ "$(date +%s)" -ge "$deadline" ] && { echo "Gate 3b TIMEOUT after 30m (status=$st) — PARK, do not hang"; break; }
  sleep 30
done
```

On timeout, do NOT keep waiting: park the item `needs-human-review` with the last status and
report (Gate 5), same as for a red run — a timed-out wait is NOT green. Local `make test`/`make
lint` do NOT cover the remote-only gates (fmt-check, govulncheck, check-file-sizes, docs build).
Red → fix-forward immediately if small; otherwise revert the merge and park the item with the CI
log excerpt. Only an OBSERVED green run upgrades the queue tag to [LANDED].

## Gate 4 — RECORD (append-only; the log is the mission's memory)

Append an entry to `design_docs/v1-mission-log.md` using its fixed template — every section,
"none" over omission. The **Routing evidence** row and **Ruled out** ledger are the two highest-
value fields: evidence drives routing-policy changes; ruled-out stops re-chasing. Update the
mission doc's queue tags ([LANDED], [PARKED], etc.) and STATUS stamp.

## Gate 5 — RETRO + REPORT

1. Scan this iteration's friction (evaluator feedback, executor corrections, your own dead ends)
   plus unread `docs/sprint-retros/` material. Route each item to exactly ONE lane:
   - **skill fix** — edit the offending SKILL.md. Max ONE skill edit per iteration; requires ≥2
     recorded frictions pointing at the same gap; state both in the commit message.
   - **process fix** — edit the mission doc (guardrails/ordering/routing policy per its rules).
   - **backlog** — new design doc via design-doc-creator, or re-prioritize the queue.
2. Routing-policy change? Only with ≥3 evidence rows; stamp it in the mission doc.
3. Morning report, TWO channels (both required):
   - `ailang messages send controlplane "<summary>" --title "Mission iteration N: <headline>"
     --from mission-control`
   - `gh issue comment 329 --repo sunholo-data/ailang --body "<markdown report>"` — the
     human-facing bookkeeping thread (Mark reads by email). Markdown, lead with the headline,
     link commits by SHA, name anything parked for a human. End the body with:
     `🤖 Generated with [Claude Code](https://claude.com/claude-code)`

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
