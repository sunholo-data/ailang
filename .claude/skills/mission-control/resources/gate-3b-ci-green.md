## Gate 3b — CI GREEN (an item is not LANDED until remote CI passes on its merge)

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-3b`.

After any push to dev, wait for CI **with a hard deadline** (Standing rule 6). A headless run has
no human to notice a hang, and a bare `gh run watch … --exit-status` blocks FOREVER if the run
never leaves `queued` (no runner). Iteration 13 (2026-07-12) wedged 4h in exactly this class of
unbounded poll — an `until COND; do sleep 30; done` whose condition never came true — before the
6h driver watchdog reclaimed the slot. Use a BOUNDED poll that fails loudly on expiry (portable;
there is no GNU `timeout` on the rig):

Record the full-SHA poll target and its read time from the same shared-ref read. The comparison and
missing-evidence protocol are explained in [`resources/ref-drift.md`](ref-drift.md).

```bash
# PIN THE POLL TARGET TO THE SHA YOU PUSHED — never `--limit 1` (see the war story below).
target_is=$(bash tools/launchd/mission-base.sh record gate3b) || exit 2
target=${target_is%%$'\t'*}                   # FULL sha from the SAME recorded read; no `--short`
if bash tools/launchd/mission-base.sh drift gate1; then
  : # Gate 1 and the fresh Gate-3b reading agree
else
  drift_rc=$?
  case "$drift_rc" in
    1) echo "DRIFT: base moved Gate1->Gate3b; poll remains pinned to recorded $target" ;;
    2) echo "no base recorded — abort, Gate 1 did not stamp" >&2; exit 2 ;;
    *) echo "Gate 3b: base comparison failed (rc=$drift_rc) — abort" >&2; exit "$drift_rc" ;;
  esac
fi
rid=$(gh run list --branch dev --workflow CI --limit 10 --json databaseId,headSha \
      | jq -r --arg t "$target" '[.[] | select(.headSha == $t)][0].databaseId // empty')
[ -n "$rid" ] || echo "Gate 3b: no CI run for $target yet — re-list a few times, still bounded"
deadline=$(( $(date +%s) + 1800 ))            # 30-min cap; CI is ~15-20m — never open-ended
while :; do
  st=$(gh run view "$rid" --json status,conclusion --jq '.status + " " + (.conclusion // "")')
  case "$st" in "completed "*) echo "CI: $st"; break ;; esac
  [ "$(date +%s)" -ge "$deadline" ] && { echo "Gate 3b TIMEOUT after 30m (status=$st) — PARK, do not hang"; break; }
  sleep 30
done
```

**The poll target must be SHA-PINNED, because `--limit 1` silently watches the WRONG RUN** (added
2026-07-29 iteration 117; the 4th recorded instance of the stale-instrument class, and the 2nd
landing squarely on Gate 3b's own verdict). The old snippet selected the newest run on the branch
and then polled *that* id, while its own adjacent comment claimed it was "for HEAD" — a lie the
reader had to catch. On a branch with any recent prior run, or when a sibling merges while you
poll, `--limit 1` returns a run that PREDATES your push. Both directions have now been observed
live: the sibling Ailang World mission's iteration 33 got `completed failure` for the *previous*
commit's run on a push it had never seen (a **red verdict for a green landing**), and V1's own
iteration 115 was outrun by a sibling merge and reported **TIMEOUT on a landing that was green**.
Iteration 116 worked around it ad hoc by reading `gh api repos/.../commits/<sha>/check-runs`
instead — correct, but a workaround the snippet did not teach. Since a Gate-3b verdict decides
LANDED vs parked, an unpinned selector can park a landed item or green a red one, which is the
exact failure mode rule (c) below exists to catch. The pin above closes it at the selection step
rather than relying on the reader to notice; `// empty` plus the re-list keeps it bounded when the
run has not been created yet. **When you want maximum certainty, skip run-id selection entirely
and read the merge commit directly** — `gh api repos/<owner>/<repo>/commits/<sha>/check-runs`
is SHA-addressed by construction and cannot drift.

**BUT `check-runs` CANNOT DRIFT AND CAN STILL COME BACK SHORT — AND EVERY AGGREGATE OVER AN
INCOMPLETE CHECK SET IS VACUOUSLY GREEN** (added 2026-08-06 iteration 154; 2nd instance after
iteration 153). The pin above fixes the instrument pointing at the WRONG run. It does nothing about
the instrument pointing at the RIGHT run and returning too few rows — and the sentence immediately
above actively recommends `check-runs` for "maximum certainty", which is precisely how this one gets
bought. A poll that computes `pending == 0 && failures == 0` is measuring the set it was HANDED;
when that set is short, both conditions are trivially satisfied and the verdict reads GREEN in the
same voice as a real green. A Gate-3b verdict decides LANDED vs parked, so this is the load-bearing
kind of wrong. Two instances, both inside the same outage window, both landing on the loop's own
verdict: **(a)** iteration 153 recorded "0 failures observed", corrected it to 1, and it was 2
within minutes — an aggregate over an **in-flight** run, stale the moment it is written;
**(b)** iteration 154 fired `rerun-failed-jobs` and its poll immediately reported
`pending=0 failures=0 SETTLED` over a `check-runs` list of **ONE** (`automerge/skipped`) where
**18** had existed minutes earlier. **Re-running a workflow EMPTIES the `check-runs` collection**,
while `actions/runs?head_sha=` correctly showed all four workflows `queued` — so the very re-run
this gate tells you to fire is what breaks the instrument it tells you to trust.
Rules: **(i)** assert COMPLETENESS before any verdict counts — enumerate the workflows EXPECTED for
this diff (the path-filter rule below already makes you do this) and require every one to be
PRESENT, else print `INSTRUMENT INCOMPLETE — no verdict` and keep polling; a count of what you
found is not a count of what exists; **(ii)** during or after a re-run prefer
`actions/runs?head_sha=<sha>`, which keeps reporting a run as `queued` rather than letting it
vanish; **(iii)** never infer "settled" from the absence of pending rows alone — pair it with
`present == expected`, which is rule 3a's known-positive control aimed at a POLL rather than a
search; **(iv)** a zero-failure count over an unfinished run is not a fact about the run, it is a
fact about the clock. The tell: your poll's verdict line and its own row count disagree in size, or
you are about to write "0 failures" for a run you have never seen COMPLETE.

**AND THE COMPARISON THAT DECIDES "SETTLED" IS AN INSTRUMENT TOO — IN `sh`, TWO *EMPTY* VALUES ARE
EQUAL, SO A POLL WHOSE COUNTS BOTH FAILED TO PARSE REPORTS COMPLETION IN EXACTLY THE VOICE OF A REAL
ONE** (added 2026-08-20 V1 iteration 233; instance 1 is iteration 107's `set -- $res` poll that
printed `TIMEOUT` on a run its own last line showed `completed success`, instance 2 is iteration
154's vacuously-green aggregate, and this is instance 3 — a new mechanism, which is why it earns a
note rather than a louder restatement). Rules (i)–(iv) above all police the check *set*: is it
complete, is it the right run, has it finished. **None of them polices the arithmetic you run over
it.** The rule immediately above even sends you to `present == expected` — a two-value comparison —
without saying what happens when neither value exists.
Measured first-party this iteration: a Gate-3b poll piped the runs payload to a local `jq`, which
died with `parse error: Invalid string: control characters ... must be escaped` on a commit message
containing a control character. Both counts were therefore the empty string, `[ "$pres" = "$done" ]`
evaluated **true**, and the loop printed **`ALL COMPLETE`** — while the very next command, a direct
per-workflow read, showed **three runs still `in_progress`**. Note what makes this worse than a
missing row: rule (i)'s remedy, comparing two numbers, is *itself* the vulnerable shape, and the
failure is total rather than partial — no rows at all, so no completeness check on the SET can see
it. It is the vacuous-pass class aimed at the gate that decides **LANDED vs parked**, and the
disposition it produces is the dangerous one: a *premature green*, not a false red.
**Rules. (a)** Before any count is compared, assert it is a NUMBER — `case "$v" in ''|*[!0-9]*)` —
and treat a non-numeric reading as **`INSTRUMENT FAILURE — not a verdict`**, never as a value.
Print that phrase; it is what distinguishes "I could not measure" from "nothing is pending", which
is the whole defect. **(b)** Prefer `gh --jq` (evaluated inside `gh`, on the raw payload) over
`gh api | jq`: the pipe is where a payload with control characters, a truncated body or a rate-limit
HTML page turns a measurement into an empty string. **(c)** Never let `-eq`/`=` on two derived
values be the only thing standing between you and a LANDED tag — pair it with the direct
per-workflow read Gate 3b already prescribes, and believe the read. This iteration was saved by
exactly that pairing and by nothing else. **(d)** The floor pays for itself immediately: once added,
the same poll correctly reported `INSTRUMENT FAILURE` straight through a real transient network drop
(`dial tcp ...: network is unreachable`) instead of calling it a completion — a second, unplanned,
live confirmation in the same iteration.
Mission-independent, and the general form is this skill's own recurring shape one level down: **a
remedy is an instrument too, so when this file prescribes a comparison, the comparison's own failure
modes become the rulebook's problem.** The tell: your poll decides it is done by comparing two
values you extracted from a payload, and you have never asked what those variables hold when the
extraction fails.

**AND `head_sha=` NEEDS ALL 40 CHARACTERS — A TRUNCATED SHA RETURNS `total=0`, NOT AN ERROR; AND
WHEN THE OUTAGE ATE THE EVENT, `workflow_dispatch` IS THE LEVER, BECAUSE IT IS NOT A WEBHOOK**
(added 2026-08-06 iteration 155; both halves are second instances). The paragraph above sends you to
`actions/runs?head_sha=<sha>` as the trustworthy instrument. Two things it does not tell you, and
each one silently produces a *confident wrong answer* about whether an item landed.

**(a) Truncation is silent.** The endpoint matches the full 40-character SHA and returns an EMPTY
SET — `total=0`, HTTP 200, no error — for anything shorter. Iteration 155 passed a 9-character SHA
(the same `[0:9]` its own jq had used for display) and read `total=0` for a PR that `gh run list`
simultaneously showed **5** runs for. Measured both ways on the identical commit: full →
`total=5`, truncated → `total=0`. This is the same class Gate 1 already records for
`rev-parse --short` — an abbreviated SHA quietly voiding an instrument — now on the endpoint this
gate recommends for "maximum certainty", which is why it is worth its own line. **Never build the
value with `[0:9]`/`--short`; get it from `gh pr view --json headRefOid` or `git rev-parse` (no
`--short`), and validate the instrument on a known-positive before believing any zero** — a SHA
you know has runs must come back non-empty in the same breath (rule 3a, aimed at an API call).
**AND THE CONTROL'S OWN SHA MUST BE `rev-parse`d, NEVER HAND-EXPANDED — A FABRICATED 40-CHAR STRING
RETURNS THE SAME CONFIDENT `total=0` AS A REAL DROPPED EVENT, SO THE CONTROL AGREES WITH YOUR FINDING
FOR THE WRONG REASON** (added 2026-08-21 V1 iteration 243; instance 1 is iteration 176's known-PRESENT
control that over-counted, instance 2 is this iteration's, and the mechanism is new). The rule
immediately above is correct and it protects the *measurement*: don't truncate the SHA you are asking
about. It says nothing about the SHA you build the **control** from, and that is the one you are most
likely to invent, because a control feels like scaffolding rather than evidence — you already know the
answer it should give, so you type an identifier instead of deriving one. Rule 3b(v)(b) already forbids
exactly this ("anything a downstream role will treat as ground truth — especially a SHA — is re-derived
by command"), aimed at findings; nothing points it at controls. That is this file's own *guard the
helper, miss the call site* shape, one level down.
**What makes it worse than an ordinary broken instrument is the DIRECTION of the failure.** Rule 3a's
trap is a control that does not fire — loud, and it announces itself as an instrument failure. This one
fires *wrongly*, and it fires in agreement: a hand-expanded SHA matches no commit, the endpoint answers
`total=0` with HTTP 200 and no error, and you read that as *"even the known-good commit shows zero, so
this really is a platform-wide outage"* — upgrading a single dropped event into a fleet-wide diagnosis
on the strength of a string you made up. Measured here: iteration 242's merge, hand-expanded, returned
`total=0`; `git rev-parse`d it returns **3**, and the finding under test (dev's own HEAD at `total=0`)
survived — but only because the control was re-derived rather than believed.
**Rule. (a)** Derive every control identifier by command in the same block that uses it —
`ctl=$(git rev-parse <ref>)`, `gh pr view --json headRefOid` — and never paste, complete, or
reconstruct one from memory or from an adjacent document. **(b)** Prefer a control the API itself can
confirm exists: if the control's own lookup can 404 or return empty, assert a *second* property of it
(here, `commits/<sha>/check-runs` reads **20** on the same commit, which is what proved the SHA
resolvable at all). **(c)** When a control returns the SAME reading as the thing it is controlling for,
treat that as *suspect before it is corroborating* — a control exists to DIFFER, so agreement is the
one outcome that deserves a second look rather than fewer. **(d)** Mission-independent, and it
generalises past SHAs to every addressed control: an issue number, a run id, a package path, a model
name. The tell: you are about to trust a negative because "the control shows the same thing", and you
typed the control's identifier rather than deriving it.

**(b) A dispatch is a different code path from an event.** When the provider is throttling webhook
delivery — GitHub's 2026-08-06 incident processed **~15%**, so "many events such as pushes and pull
requests are not triggering workflow runs" — there is nothing to re-run and nothing to poll, because
the run was never created. Iteration 154 hit exactly that (`#608`: zero runs at all) and recorded it
as a dead end. It is not one: `gh workflow run <wf.yml> --ref <branch>` is a direct API call, not a
webhook delivery, and it **creates the run**. Measured at iteration 155 on that same PR: `total=0` →
`total=1` (`CI: queued`, `event=workflow_dispatch`). Caveats, both load-bearing: the workflow must
declare `workflow_dispatch` in its `on:` block (check first — it is not universal), and any job
needing PR context will not be equivalent — a docs/paths gate whose detector diffs against the PR
base cannot be satisfied this way, so dispatch may reach only *some* of the required contexts. Say
which ones it reached and which still need a real `pull_request` event; do not report a partial set
as a green (that is rule (i) above). Prefer this over close/reopen, which is itself webhook-borne
and can leave the PR closed if the reopen is dropped.

Both halves generalise past this outage: **an instrument that takes an identifier can be voided by
the identifier's FORMAT**, and **when an event-driven trigger is unavailable, look for the API-driven
one before concluding the work cannot proceed.**

**BUT `workflow_dispatch` IS ONLY HALF A LEVER, AND THE RULE ABOVE — WRITTEN ONE ITERATION EARLIER —
WILL TELL YOU A PR IS UNBLOCKED WHEN IT IS NOT** (added 2026-08-07 iteration 156; instance 1 is
iteration 155's own dispatch, instance 2 is this iteration measuring what it bought). The clause
above ends on a triumphant note — the run was CREATED, `total=0 → total=1` — and stops there. Two
things it never checked, each of which independently voids the verdict, and both cheap:
**(a) A CREATED RUN IS NOT A RUNNING RUN.** During the recovery phase a run can be accepted and then
sit forever: iteration 155's dispatch (`21:30Z`) and `#606`'s `pull_request` run (`17:02Z`) were both
still `queued` with **`jobs=0` seven hours later**, while dispatches fired at `23:38Z` reached
**`jobs=6` within 12 seconds**. So `total_count` on `actions/runs` only proves a *record* exists;
the discriminator is `runs/<id>/jobs`. Worse, a wedged run **cannot be cleaned up** — `gh run cancel`
answers *"Cannot cancel a workflow run that is completed"* while the runs API simultaneously reports
`queued`, an inconsistent server record with no client-side remedy. Check jobs, not totals, and give
a fresh run ~15s to acquire them before believing in it.
**(b) A DISPATCH'S CHECKS DO NOT SATISFY BRANCH PROTECTION.** This is the load-bearing half. All four
required contexts came back **success on the head SHA** via `commits/<sha>/check-runs` (18 rows) —
and `gh pr checks --required` still listed only the one context that had come from a real
`pull_request` event, with `mergeStateStatus=BLOCKED`. Branch protection is gated on the PR's
`pull_request` check suite, which is precisely the suite the outage wedged. So the dispatch buys a
green you **cannot spend**, and reading it as "3 of 4 contexts reached" is rule 3d in its purest
form: the mechanism visibly did *something* in the direction you hoped, and nobody checked the thing
it was for. (Note the mirror-image instrument failure in the same measurement: `gh pr view --json
statusCheckRollup` returned **1** and **0** rows where the SHA-addressed endpoint returned **18** and
**12** — so that aggregate was vacuously *red*. Gate 3b already warns that an aggregate over an
incomplete check set is vacuously green; it is equally vacuous in the other direction.)
**THE FIX, WHICH NEEDS NO LOCAL CHECKOUT AND TOUCHES NO WORKING TREE:** create a **tree-identical
empty commit through the git API** and move the ref —
`POST git/commits` with the branch's EXISTING `tree` sha and its current head as `parents[]`, then
`PATCH git/refs/heads/<branch>`. That fires a genuine `pull_request: synchronize`, so the new check
suite is PR-scoped and *does* count. Assert tree identity by comparing the new commit's `.tree.sha`
to the old one before moving the ref, and prefer this to close/reopen (itself webhook-borne).
Squash-merge absorbs the commit. Critical Principle 0 is not engaged — no checkout, no branch
switch, no stash. **Quote `'parents[]'`**: unquoted, zsh glob-expands it (`no matches found`), the
commit is never created, and the ref PATCH then fails 422 on an empty sha — rule 3a(i-b), on the
very shape this skill already warns about.
**AND THE SAME REASONING VOIDS A GREEN** (proposed by `mission-world` iter-59, which shares this
skill but cannot edit it; corroborated first-party in V1's own repo before adoption). The outage
clause in Gate 1 teaches that a RED during a declared incident is unattributable. The symmetric half
was never stated: during an open incident **outcome is not a function of the tree, so a GREEN is
unattributable too** — yet a green is exactly what gets read as "the incident is over". Measured
here: CI on `dev` was `success` at `17:32Z`, `failure` at `20:03Z` — *worse after the green* — and
`success` again at `21:57Z`, while runs created at `17:02Z` and `21:30Z` sat wedged straight through
all three. A green during an incident licenses a **code** inference (these jobs passed on this tree)
and **never** an **infrastructure** one (the incident is over). Close an incident on the provider's
status API, not on one of your own runs, and where a green is load-bearing require one taken AFTER
the incident is marked resolved.

**Poll SEVERAL workflows with THIS snippet, run once per workflow — do NOT hand-roll a
multi-workflow loop** (added 2026-07-27 iteration 107; TWO defects in one iteration, both from
hand-rolling a variant because Gate 3b only shipped the single-workflow form above while a real
push needs CI *and* Build-and-Release). Defect 1: an associative-array "already done" tracker
whose writes did not persist, so completed workflows re-printed every round and the final summary
read `INCOMPLETE` for two jobs that had in fact reported. Defect 2: a `set -- $res` positional-split
poll that printed `TIMEOUT ... — PARK` while its own last reading was `completed success <target-sha>`
— i.e. it declared a park on a run that was GREEN. A Gate-3b verdict is load-bearing (it decides
LANDED vs parked), so a poll bug can park a landed item or, worse, green a red one. Rules:
(a) reuse the snippet above verbatim, once per expected workflow, instead of inventing a combined
loop; (b) the poll's output is a HINT, never the verdict — before recording Gate 3b, confirm the
truth directly and cheaply with a per-workflow read, which is also what Gate 1 already prescribes:

```bash
for wf in "CI" "Build and Release"; do        # only workflows EXPECTED for this diff
  gh run list --workflow "$wf" --branch dev --limit 1 \
    --json status,conclusion,headSha --jq '.[0] | "'"$wf"': \(.status)/\(.conclusion // "-") @ \(.headSha[0:9])"'
done
git rev-parse --short origin/dev              # the SHA those runs MUST match
```

(c) never report green from a poll whose own last-seen line contradicts its verdict — re-read and
believe the direct query.

On timeout, do NOT keep waiting: park the item `needs-human-review` with the last status and
report (Gate 5), same as for a red run — a timed-out wait is NOT green. Local `make test`/`make
lint` do NOT cover the remote-only gates (fmt-check, govulncheck, check-file-sizes, docs build).
Red → fix-forward immediately if small; otherwise revert the merge and park the item with the CI
log excerpt. Only an OBSERVED green run upgrades the queue tag to [LANDED].

**AND AUTO-MERGE IS NOT A LANDING MECHANISM WHEN THE RED IS INHERITED FROM THE BASE — IT WAITS ON A
CHECK SET THAT CAN NEVER CHANGE, AND THE ITERATION THAT ARMED IT HAS ALREADY ENDED** (added
2026-08-18 motoko iteration 10; a new mechanism inside Gate 2's died-mid-flight class, which already
carries instances at iterations 121, 148/149 and 160/161, plus motoko 6→7). Every rule above assumes
you are WATCHING a run: it goes green, it goes red, or your bounded poll expires and you park. Auto-
merge is attractive precisely because it removes that obligation — "the required checks will pass and
GitHub will merge it" — and it is a correct, routine mechanism (V1's log carries **76** mentions of it
landing PRs). It has exactly one hole, and it is the case a mission-record PR most often lands in:
**a red the PR inherited from its base**. GitHub auto-merge merges when *the PR's own* required checks
pass; it does **not** update the branch and does **not** re-run checks when the base advances. So the
failing check stays pinned to the PR's head SHA and outlives its own fix, forever, with nothing on
either side re-evaluating it.
The reason this earns a rule rather than a caution is that the loss is **SILENT**. There is no
timeout, no red to triage, no failed command — Gate 3b was never left in a state that *could* fail —
so the iteration ends looking successful and the mission's memory simply skips a number. It is the
vacuous-pass class this loop keeps closing, aimed at the *landing* step: success reported for work
that never shipped. Afterwards it is invisible to every check except Gate 2's died-mid-flight traces.
Measured on motoko: `#760` (iteration 9's entire record) enabled auto-merge at `19:36:39Z` over a
`test` red inherited from base `714f1cecc`, with the PR body predicting *"it goes green once #759
lands"*. **The prediction came true and the PR still did not merge** — `#759` merged at `20:02:27Z`,
26 minutes later, and its merge commit `cf56772bf` reads `test=success`, as does every commit after
it; `#760`'s `updatedAt` was still `19:36:34Z` with `test=failure` on head `bf66f7655` at
`run_attempt=1` **12h13m** later, `MERGEABLE`/`BLOCKED`.
**Rules. (a)** Never end an iteration with its record behind auto-merge on a PR whose checks are not
GREEN NOW — an armed auto-merge is a *prediction*, and Gate 3b's whole discipline is that a
prediction is not an observation. **(b)** When the PR's red is base-inherited, the fix is a **rebase
and force-push**, not a wait: that is what produces a new head SHA and therefore a new check set.
Confirm it is safe first — `git diff --stat <base> origin/dev -- <the files you touch>` empty means no
conflict is possible. **(c)** If you genuinely cannot land it in-turn, park the item
`needs-human-review` and say so **in the charter and the log**, so the next iteration inherits a
record rather than a mystery — the failure here was not the unmerged PR, it was that the charter's
newest stamp read one iteration behind with nothing explaining why. **(d)** Diagnose which red you
have before choosing: read the failing STEP, not the check name — `gh api
repos/<o>/<r>/actions/jobs/<id> --jq '.steps[] | select(.conclusion=="failure")'`. `check-runs`
reports the job and never its steps, and the same iteration got this wrong in the other direction:
`#760`'s body blamed `Check changelog index hygiene` — the gate that iteration had spent all day
fixing — while that step was **`skipped`** on both the PR head and the base, and the real failure was
step 14 `Run stdlib .ail test suites` on both. Right verdict, wrong mechanism; rule 3d, since the red
arrived in exactly the direction the author had been looking. The tell: you are about to write "auto-
merge will take it from here", or to end a turn whose PR you have not seen go green.

**Poll only checks that CAN complete for this push** (added 2026-07-16 iteration 31; second
friction in the blind-poll class — iteration 30 burned a full 35-min cap watching a
conflict-skipped PR suite, iteration 31's first poll demanded a Docs-Deploy run that its
`paths:` filter guaranteed would never trigger for a non-docs diff). Before arming any Gate-3b
poll: (a) determine which workflows are EXPECTED for this push — check each workflow's `on.push.
paths` filter against the diff, or confirm a run for the target SHA appears within the first 2–3
listings; a path-filtered workflow with no run is **N/A, record it as such — not pending**;
(b) for PR polls, check `gh pr view --json mergeable` each round and bail on CONFLICTING —
Actions skips `pull_request` workflows it cannot build a test-merge for (they never complete).
A poll that waits on a check that cannot complete is an unbounded wait wearing a deadline.

**AND CHECK `mergeable` BEFORE YOU REACH FOR ANY DROPPED-EVENT LEVER — MISSING RUNS ON A PR ARE A
CONFLICT UNTIL PROVEN OTHERWISE, AND THIS FILE NOW ARGUES LOUDLY FOR THE RARE EXPLANATION WHILE THE
COMMON ONE IS HALF A LINE** (added 2026-08-14 V1 iteration 198; instance 1 is iteration 30's
35-minute cap above, and this is instance 2 — but the *mechanism* is new, which is why it earns its
own note rather than a louder restatement of clause (b)). Clause (b) is correct and it is one
sub-clause of one sentence. Since then this gate has grown three multi-paragraph blocks on dropped
webhook deliveries — the `checks=0` zero-run rule, the all-40-characters rule, and the
`workflow_dispatch`-is-only-half-a-lever rule — and `workflow_dispatch` now appears **6** times in
this file against **one** operative mention of `CONFLICTING`. A controller who has just read this
gate top-to-bottom has been handed a vivid, recent, heavily-evidenced story about GitHub silently
eating events, and a passing mention of the boring answer. **Prominence is not evidence**, but it
is what you reach for first.
Measured here, and the shape is worth more than the incident: a PR head showed `checks=1`
(`automerge/skipped`) with **1** of **5** expected workflow runs, the provider's status API read
*All Systems Operational*, and repo-wide run creation was demonstrably healthy — three `push` runs
for a sibling's commits appeared in the same window. Every one of those observations is *consistent
with* a dropped `pull_request` delivery, and the controller spent ~15 minutes assembling exactly
that diagnosis, right up to selecting the tree-identical-empty-commit lever. One command refuted it:
`gh pr view --json mergeable` → **`CONFLICTING`/`DIRTY`**, from a three-commit sibling merge to
`dev` that touched **one** file in common (a changelog). Rebase, force-push, and all **5** runs
appeared within 25 seconds. This is rule 3d in its purest form — the evidence arrived in exactly the
direction the loudest rule predicted, so nothing prompted a negative control — and note that the
partial delivery is what sells it: `pull_request_target` fired while `pull_request` did not, which
reads as *selective* event loss and is in fact just Actions declining to build a test-merge.
**Rule.** On a PR, before diagnosing anything as a dropped event and before firing `workflow_dispatch`
or an empty commit, read `mergeable`/`mergeStateStatus` — it is one call, it is authoritative, and
`CONFLICTING` explains missing `pull_request` runs completely. Order matters, not just inclusion:
put it FIRST, because the dropped-event levers are expensive and mutate the PR, while this is a
read. Two corollaries. **(a)** `MERGEABLE`/`UNKNOWN` is not a clearance — GitHub computes
mergeability asynchronously and answers `UNKNOWN` while it does; poll until it resolves rather than
banking the first non-`CONFLICTING` reading (this iteration saw `UNKNOWN UNKNOWN` before the truth).
**(b)** The rare explanation stays rare *even after you have personally seen it once*: iteration 196
met a genuine dropped push event, wrote it up at length two iterations before this one, and that
write-up is precisely what made the wrong hypothesis feel earned. **Mission-independent, and the
generalisable half is about this file rather than about `gh`: when a gate accumulates a long, vivid
war story about an uncommon cause, the common cause needs re-promoting, or the documentation itself
becomes the bias.** The tell: you are about to explain missing CI runs with an infrastructure
failure, and you have not yet run the one-line check for the boring one.
