## Gate 1 — OBSERVE (cheap, read-only)

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-1`.

**Sync to origin FIRST — the local checkout LIES when a prior run merged via GitHub** (added
2026-07-12 iteration 12; second instance of the same gap — iteration 9's watch-list already flagged
"add a resume-detection step to Gate 2", and iteration 12 booted on a stale local dev that was 2
commits behind origin/dev with the picked item ALREADY merged+recorded, yet the local mission
log/queue/sprint-JSON read as "mid-flight iteration 11" and drove a full redundant re-evaluation
before the Gate-3b fetch caught it). Before reading ANY local mission state:

```bash
git fetch origin
git rev-parse dev origin/dev                  # differ? origin is ground truth. NO --short: see below
git log --oneline dev..origin/dev             # commits your working tree is missing
```

Record the exact shared-ref reading immediately after that sync block. The dedicated base file is
separate from the heartbeat; see the shared-clone rationale and disagreement protocol in
[`resources/ref-drift.md`](ref-drift.md).

```bash
base=$(bash tools/launchd/mission-base.sh record gate1)   # records SHA + read-time; echoes them
echo "Gate 1 base: $base"                                # full SHA<TAB>ISO8601-UTC
```

**`git rev-parse --short` accepts exactly ONE revision** (fixed 2026-07-27 iteration 108). This
snippet carried `--short dev origin/dev` for months; it fails `fatal: Needed a single revision`
(rc=128) **100% of the time, in every repo** — `--short HEAD HEAD` fails identically, so it is the
flag, not the refs. Iterations 55/56/57/58 all hit it, and all four concluded it was a *transient
ref-lock race that self-heals on retry*; iteration 56 explicitly tested `remote.origin.fetch`,
refuted the snippet-bug hypothesis, and recorded "a race has no code fix". **The retry was a
different command** — a single-arg `git rev-parse origin/dev` — so each retry silently swapped in a
valid form and the fix got attributed to elapsed time. Measured at iteration 108: two-arg with
`--short` rc=128 **5/5**; without `--short` rc=0 **5/5**; single-arg with `--short` rc=0 **5/5**.
If you want short SHAs from both, use `git rev-parse --short dev; git rev-parse --short origin/dev`
as separate calls, or drop `--short` as above.

Two general lessons, both cheap and both repeatedly earned:

- **A failed check is not a passed check.** This one fataled to *stderr* and printed nothing to
  stdout, so it looked exactly like the adjacent `git log dev..origin/dev` printing nothing because
  the branches are in sync — a broken sync check wearing the all-clear's clothes. Gate 1's whole job
  is deciding whether local state is trustworthy, so read the **exit code**, not the silence.
- **"It self-healed on retry" is a claim about your retry, not about time.** Before recording
  anything as transient, confirm the retry ran the *identical* command; if you changed the
  instrument, you measured the instrument. Same family as "a parked test is a claim, not evidence"
  and "exit codes through pipes lie" — the instrument's own validity must be established before its
  reading counts as evidence. Four iterations paid for this one, each cheaply enough to wave through.

If local dev is behind origin/dev, read the mission doc + log + queue tags FROM ORIGIN
(`git show origin/dev:design_docs/v1-mission.md`, `…:v1-mission-log.md`) — a GitHub squash/merge
advances origin/dev without touching the local ref, so the working-tree copies are stale. Do NOT
pull/reset the shared main tree (Critical Principle 0 — it may hold a sibling's uncommitted work);
treat origin as truth, and if you need the code, branch a worktree from `origin/dev`.

**REPAIRING the divergence, not just routing around it** (added 2026-08-03 iteration 132, after
THREE iterations each escalated the reconcile as a human ask instead of performing it — iter-128's
stale *skill*, iter-129's stale *charter*, iter-131's stale *driver*, which defeated an explicit
human instruction for two days). Everything above tells you how to SURVIVE a stale checkout; nothing
told you how to END one, so the divergence grew monotonically — 8 behind, then 10, then 11 — and each
iteration paid the tax again. A reconcile is **provably non-destructive** when four obligations all
hold; measure them, do not assume them, and if ANY fails, park for human:

1. **Every local ahead-commit is a duplicate of an upstream one** — compare `git patch-id --stable`,
   not commit titles. Iterations 129–131 all asserted "the ahead-commit is a duplicate" from its
   *subject line*; iteration 132 measured it (`fc808504e7…` on both sides). Same claim, different
   epistemic status, and it is the fact the whole operation rests on.
2. **No incoming commit touches any locally-modified file** —
   `comm -12 <(git diff --name-only dev origin/dev|sort) <(dirty files|sort)` must be EMPTY, and
   rule 3a applies: pair it with a control (the intersection against files you KNOW origin changed)
   so an empty answer proves the instrument ran.
3. **Back up every dirty file** outside the repo first, and re-verify byte-identity afterwards.
4. **Use `git checkout -B dev origin/dev`, which is protective** — it updates clean files, carries
   local modifications across, and ERRORS rather than clobbering. Its refusal is a feature: it is
   what distinguishes this from `reset --hard`, which Principle 0 forbids precisely because it
   cannot refuse. None of Principle 0's four named operations (branch checkout to a *different*
   branch, pull, reset, stash) is involved.

**The refusal you should expect, and its fix.** `checkout -B` compares the working tree against the
**stale local HEAD**, not against the target — so a file whose content ALREADY EQUALS origin (e.g. a
previous iteration's containment restore) still reads as a clobber risk and blocks the switch. Stage
origin's blob for exactly those paths — `git checkout origin/dev -- <paths>` — then retry. Verify by
sha256 that **no byte on disk changed**; it only updates the index. Do not "fix" this by reverting
them to the stale version first: that briefly re-arms the very bug you are clearing, and on a rig
where launchd fires on a timer, briefly is enough.

Standing authorisation is a HUMAN decision, not a controller one — Mark authorised the 2026-08-03
reconcile explicitly. Until he grants a standing one, ASK (a one-word DECISIONS row) and meanwhile
route around it as above. **A one-time reconcile is not the durable fix**: every launchd entry point
still executes from the shared checkout's *working tree*, so this recurs the moment the tree falls
behind. Note also which gates get CHEAPER once local == origin — Gate 4 may then write the
charter/log **in place** rather than via a worktree, since its stale-base hazard is gone.

Read: the mission doc (queue, guardrails, routing policy — they may have changed), the last 1–2
log entries (especially **Next** and **Ruled out** — do not re-chase), any parked
`needs-human-review` items that got human answers in the inbox.

**Check dev CI first — PER WORKFLOW, never a raw run list** (sharpened 2026-07-10 iteration 3:
a raw `--limit 6` list was flooded by Dependabot-Updates entries and read as green while dev CI
had been red for 3h; Build-and-Release and Docs-Deploy were equally invisible — TWO recorded
frictions, one gap):

```bash
# workflow names come from the Repo Profile (V1 defaults shown); --branch is the mission's dev
for wf in "CI" "Build and Release" "Deploy Documentation to GitHub Pages"; do
  gh run list --workflow "$wf" --branch dev --limit 1 \
    --json conclusion,headSha --jq '.[0] | "'"$wf"': \(.conclusion) @ \(.headSha[0:9])"'
done
```

**THEN READ THE COMMIT'S FULL CHECK SET, BECAUSE THE LOOP ABOVE ENUMERATES WORKFLOWS BY *NAME* AND
IS THEREFORE BLIND BY CONSTRUCTION TO EVERY CHECK THAT IS NOT AN ACTIONS WORKFLOW YOU LISTED**
(added 2026-08-07 iteration 158; instance 1 was iteration 157, which found the gap while doing a
Gate-3b cross-check and filed it as an observation rather than fixing it). The `for wf in …` loop is
a *hand-maintained allowlist*, so its all-green means "the three things I remembered are green" and
reads identically to "this commit is healthy". It is rule 3g — the hand-picked-subset gap — aimed at
Gate 1's health check rather than at your pre-push sweep, and it is worse here, because Gate 1's
verdict decides whether a RED outranks the queue. Measured on V1: `SonarCloud Code Analysis` had
been `failure` for **six consecutive analysed commits** while this loop reported `CI`,
`Build and Release` and `Docs-Deploy` all `success` on the *same SHA* — four iterations walked past
it. Sonar is a GitHub App, not a workflow, so no workflow name could ever have surfaced it. Add one
SHA-addressed read, which is complete by construction:

```bash
sha=$(git rev-parse origin/dev)               # no --short (Gate 1's rev-parse lesson); see note below
gh api "repos/${MISSION_REPO:-sunholo-data/ailang}/commits/$sha/check-runs" \
  --jq '"checks=\(.total_count)", (.check_runs[] | select(.conclusion != "success" and .conclusion != "skipped" and .conclusion != "neutral") | "  NOT-GREEN \(.name): \(.conclusion // "pending")")'
```

Read it with rule 3a's discipline: `total_count` is the known-positive control, so `checks=0` means
the endpoint did not answer, **not** that the commit is clean. **Note the endpoints differ on SHA
truncation and do NOT assume otherwise** — this rule's first draft carried a "a truncated SHA
silently returns 0" comment copied across from Gate 3b, and the negative control run in the same
breath refuted it: `commits/<sha>/check-runs` resolves an abbreviated SHA fine (**19** both ways),
while `actions/runs?head_sha=` genuinely needs all 40. Two endpoints, two behaviours, and the
warning was true of the other one. Then triage whatever it surfaces
exactly like any other red — with rule 3d's negative control, since the commonest answer is
"inherited, not from this push": walk the same check back over the parent commits before attributing
it to anything recent. A red non-required check is not automatically the pick (`UNSTABLE` is not
`BLOCKED`), but it must be *seen and named* rather than invisible; a standing red nobody has looked
at for six commits is exactly how a required one eventually gets missed. Mission-independent: the
allowlist is per-repo, the blind spot is not.

**AND WHEN THAT ZERO IS *TRUE* — THE INSTRUMENT FIRES AND THE COMMIT GENUINELY HAS NO RUNS — dev IS
NOT GREEN AND NOT RED, IT IS **UNVERIFIED**, AND NOTHING ABOVE GIVES YOU A DISPOSITION FOR THAT**
(added 2026-08-14 V1 iteration 196; instance 1 was iterations 154–155, whose fix landed in the wrong
gate). Every rule above asks whether the runs you FOUND are green. None asks whether a run **exists**.
The paragraph immediately above even trains you to read `checks=0` as *the endpoint did not answer* —
correct, and it stops there, so a controller who re-probes, sees the control fire, and gets zero again
has followed the rule to its end and still has no next step. The failure mode is that an unverified
HEAD renders **identically to a clean one**: no red to triage, no name to record, nothing to say.
Measured here: `#701` merged at `22:18:30Z`; `commits/<sha>/check-runs` → **0** with the control
`fc357a045` → **16** in the same call, `actions/runs?head_sha=<full 40>` → `total=0`, **0** runs
repo-wide after 22:00Z against **20** in the hour before, `actions/permissions` → `enabled`, and the
provider's status API → *All Systems Operational*. GitHub had recorded **no PushEvent for the merge**
while carrying one for both merges 30 and 54 minutes earlier. Concealed behind that silence was a
genuine `govulncheck` red — **7 reachable stdlib advisories** — which the three named workflows
reported `success` straight past, because their greens were on the *parent* commits.
**The lever already exists in this skill and is filed where Gate 1 will never look**: Gate 3b's
2026-08-06 outage rule teaches that `workflow_dispatch` is an *API call, not a webhook delivery*, so
it creates a run when the event was dropped. That was written about a PR during a declared incident;
nothing pointed it at dev's own HEAD, and iterations 154–155 paid for the same missing disposition
one gate over (`#608`: *"ZERO workflow runs created at all, so Gate 3b never even had an instrument"*).
Guard the helper, miss the call site — this loop's own named recurring shape, applied to its rulebook.
**Rule.** At Gate 1, after reading the check set, assert a run EXISTS for `origin/dev`'s HEAD. If the
count is a true zero, do **not** record a health verdict: fire `gh workflow run <wf> --ref dev` (check
the workflow declares `workflow_dispatch` first), give it ~15s and confirm `runs/<id>/jobs` is
non-zero — `total_count` alone only proves a *record* exists — then triage whatever it surfaces as an
ordinary Gate-1 red. **Do not stop at "an event was dropped"**, which is a fact about GitHub; the
deliverable is the *verdict on the commit*, which only a run can give you. Then scope the anomaly
before reporting it: **count runs per PR MERGE COMMIT, never per commit**, because only a push's
**tip** gets a run and every intra-push commit reads as zero by construction — the wrong unit turned
one dropped event into an apparent 9-of-15 pattern here, and the correct one showed **7 of the last 8
merges fine**. Mission-independent, and note the standing exposure it reveals: a PR's green is taken
on the PR head, while the squash-merge produces a *different commit* — so "the PR was green" never
implies "dev's HEAD was verified", and on the one occasion those diverge it is invisible. The tell:
your CI health check printed nothing alarming and you have not confirmed that anything ran.

Any non-success → a RED dev outranks the queue (added 2026-07-10 per Mark; that day's red was a
pre-existing gofmt miss + a newly published stdlib vuln — neither from a sprint, both invisible
to local gates). Diagnose via `gh run view <id> --log-failed` — and check whether the SAME
failure exists on the parent commits before blaming any merge (iteration 3's three reds all
pre-dated the sprint; one first appeared on a docs-only commit). The fix (or a reasoned
allowlist/revert) IS this iteration's first deliverable. Time-based reds (new vuln advisories,
runner-image changes un-hiding latent bugs, dependabot peer-dep breaks) hit whoever observes
next — that's the mission's job now.

**Scoped to the OWNING mission (2026-08-17).** "Whoever observes next" assumed one observer per
repo. That is false wherever two missions share a `MISSION_REPO`: V1 and motoko both carry
`sunholo-data/ailang` from separate clones, the overlap guard is per-mission by construction
(`PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"`, each loop guarding only itself), and no
cross-mission mutex exists. Both loops therefore preempt onto the *same* red and do the *same*
work — measured as [#758](https://github.com/sunholo-data/ailang/pull/758)/[#759](https://github.com/sunholo-data/ailang/pull/759),
identical six-file fixes opened four minutes apart. Gate 2's open-PR check does not save you: it
is point-in-time at pick time, aimed at a *past* iteration's abandoned work, so a peer that opens
its PR later in the same window is invisible (V1 checked at ~18:58Z; #758 appeared 19:05Z).
So: **a red outranks the queue only for the mission that OWNS the repo** — for
`sunholo-data/ailang` that is V1. A non-owning mission records the red, hands it to the owner on
the cross-mission channel, and keeps its own pick, EXCEPT where the red is its own doing or sits
in territory the owner has no domain knowledge for. Check your charter's guardrails for which
side of that line you are on before letting any red displace your pick.

**A RED IS NOT ALWAYS ABOUT YOUR CODE, AND A GREEN DURING AN INCIDENT IS NOT ABOUT YOUR CODE
EITHER — and separately, a red that fails EARLY in a long ordered job SUSPENDS every gate behind
it, so the check set reports one failure where there may be six.** Both are failure modes of this
gate's own verdict, both have discriminating commands and known-wrong first instincts (reverting a
good commit; declaring dev green after fixing the one red you found), and both are on-demand rather
than always-on. Read [`resources/ci-health.md`](resources/ci-health.md) BEFORE recording any Gate-1
health verdict on a red you cannot immediately attribute, or on a job whose steps you have not counted.
