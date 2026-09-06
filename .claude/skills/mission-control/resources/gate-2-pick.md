## Gate 2 — PICK + REALITY-CHECK

**⚠ BEFORE PICKING, GREP THE ITERATION INDEX. This is how the loop avoids redoing work.**

```bash
grep -i '<keyword>' design_docs/${MISSION_NAME}-mission-index.md
```

`design_docs/<name>-mission-index.md` is ONE LINE PER ITERATION covering the entire
history — live log and archive together — and it is small enough (~13k tokens for 331
iterations) to read whole. The live log holds only the newest 20 entries; the rest are in
`<name>-mission-log-archive.md`, retrievable but never loaded.

Why this rule exists: the v1 log reached 2.86 MB / ~715k tokens / 334 entries with no
rotation anywhere in this protocol, so "has this been tried?" had no cheap answer — the
only complete record was a file too big to read. Rotation without an index would have made
that worse, not better: it would have moved the history out of reach entirely. Grep the
index first; open the archive only when the index says something relevant happened.

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-2`.

Take the top `[NEXT]` queue item. **Before any work, verify the doc's claimed status against repo
reality**: `git log --grep`, does the code/test already exist, does `make test` already cover it.

**QUORUM-AT-PICK (Mark 2026-07-16 — "old docs may not be up to new standards"):** the creation-time
quorum hook only covers NEW/REVISED docs, so most of the backlog is pre-quorum (iteration 32's
auto_caps doc, Oct 2025, reached the planner with zero multi-provider eyes). At pick time, if the
picked doc has NO quorum artifact (`ls .ailang/state/mission-quorum/<doc-id>-*.json`), run the text
quorum BEFORE routing: `ailang design-quorum <doc.md> --controller-verdict <your own pass|reject>`
(cents, budget-capped, N−1 degrade). Any-reject → the objections go to the designer role for a
revision pass first (Gate 3's design-doc-creator lane), then re-quorum ONCE; still-rejected →
`needs-human-review`, park, next item. Skip only for: bookkeeping-only picks, ghost-closes, and
mission-infra docs the quorum already reviewed. This is a pick-time gate, not a re-litigation —
one round, bounded.
**NARROW-REFINEMENT CARVE-OUT (added iter-95, 2nd instance — iter-93 `m-pure-prng` `split` was the
1st):** twice now the one-revision-one-requorum→park gate parked a doc whose design DIRECTION both
reviewers accepted, blocking SOLELY on narrow, obviously-resolvable defects the reviewers themselves
fully specified (iter-93: defer a non-core `split` helper; iter-95 `m-budget-scoping-bug`: a
deterministic frame-selection tie-break + a Conflict-Surface inventory bullet — both quoted verbatim
in the reject's `proposed_fix`). Parking these for a human wastes an iteration on a non-judgment
call. So, AFTER the one re-quorum, IF **every** remaining blocking objection (a) carries a concrete
reviewer-authored `proposed_fix` AND (b) does NOT dispute the design DIRECTION (only
completeness / determinism / attribution / a scope-cut of a non-core helper), the controller MAY
make a **bounded 2nd revision that applies the reviewers' VERBATIM fixes** (their own text — never a
controller-invented resolution, never overriding an objection) and route straight to sprint-planner,
recording the applied fixes in the doc's Quorum verification log and the Gate-4 routing-evidence row.
This SATISFIES the objections; it is NOT force-passing (Standing rule 2 still forbids proceeding over
a contested design DIRECTION — that still parks). If ANY remaining objection disputes the direction,
or lacks a concrete fix, or would need controller judgment to resolve → park `needs-human-review` as
before. **Ratification of first use (iter-95):** because this is a controller-authored gate change,
the FIRST doc to use the carve-out is surfaced to Mark for a one-time OK before its sprint runs (a
`--from mission` report line + a parked `(0)` bookkeeping row); once ratified, later iterations
apply it without re-asking. Record which path was taken in the log's Ruled-out/routing rows.
**A QUORUM THAT PRINTS `proceed` WITH A NON-EMPTY `absent_reviewers` IS NOT A PASS — IT IS A PASS
WITH A NAMED HOLE, AND THE EYE THAT CLOSED IS SYSTEMATICALLY THE ONE YOU MOST NEEDED** (added
2026-08-11 iteration 175; proposed by `mission-world` iter-70, which shares this skill but cannot
edit it, and corroborated first-party in V1's own artifacts before adoption — sibling-claim ghost
discipline). The degrade itself is documented and correct: an unreachable/over-budget reviewer is
recorded by name with its reason and the quorum drops to N−1, never a silent pass. That is true and
it is not enough, because the synthesis still emits `verdict: proceed`, and a controller reading a
green verdict has no prompt to notice the gate just halved. Same family as Gate 3b's "an aggregate
over an incomplete check set is vacuously green", aimed at the quorum instead of at `check-runs`.
**The trigger is self-selecting, which is what makes this worth a rule rather than a caution.** A
reviewer drops out on `budget` when the DOC GREW — i.e. immediately after a substantial revision,
which is exactly when its opinion is most load-bearing, and the revision was usually driven by *its*
objection. World's instance: round 1 both reviewers reject → one revision → round 2 `gemini-3-1-pro`
PASS, `gpt5-6-sol` absent on budget → synthesis **proceed**; re-running that one reviewer alone for
**$0.08** returned **REJECT**, and the objection was real (the doc claimed an *enforced* authority
boundary its milestones only supplied as an optional helper). V1's instance is the same shape and
sharper on cost: `m-named-test-body-check-semantics-2026-08-07T04-40-40Z` printed `proceed` with
`gpt5-6-sol` absent for *"estimated cost $0.1048 (doc ~14818 input tok) exceeds cap $0.1000"* —
refused over **$0.0048** — on a doc `blocked` in the immediately preceding round at `04:26:48Z`
where `gpt5-6-sol` was the substantive rejecter.
**Rule.** Before acting on any synthesis whose verdict is `proceed`, read `absent_reviewers`. If it
is non-empty: re-run each absent reviewer alone with a raised cap
(`ailang design-review --reviewer <m> --max-cost-usd <raised>`) and fold its verdict in — a metered
re-run is cents against the $5 iteration ceiling and is the cheapest gate in this loop. If a
reviewer genuinely cannot be restored, then everywhere the verdict is quoted downstream it reads
"PROCEED at N−1, `<model>` absent (`<reason>`)" — never a bare "quorum passed".
**⚠ BUT THE PATH THIS RULE NAMES IS WRONG, AND THE WRONG PATH FAILS IN THE EXACT DIRECTION THE
RULE EXISTS TO PREVENT — `jq '.absent_reviewers'` RETURNS `null`, WHICH READS AS "NOBODY WAS
ABSENT"** (fixed 2026-08-31 V1 iteration 311; instance 1 is iteration 309, instance 2 is iteration
310, both of which recorded the friction as a queue row and could not spend their one Gate-5 skill
edit on it, and this iteration measured the writer). The rule above says *"read `absent_reviewers`"*
and the neighbouring rule says the verdicts are the reviewers'. **Both are one level off**, and a
controller who types the documented path gets a confident `null` — the vacuous pass this whole rule
was written to close, reached by following the rule.
Measured across **22 of 22** artifacts in `.ailang/state/mission-quorum/` (complete enumeration, not
a sample; control — `has("synthesis")` = **22**): top-level `has("absent_reviewers")` = **0**, and
`[.reviewers[] | select(has("verdict"))] | length` > 0 in **0**. The keys DO exist, one level down
— `Synthesis.AbsentReviewers` at `internal/mission/quorum/quorum.go:51` and the reviewer verdict
under `.result`, the reviewer object's keys being `cost_usd, landed, model, present, result,
tokens_in, tokens_out`. **This CORRECTS instances 1 and 2**, which both concluded *"there is no
`absent_reviewers` key at all"*: there is, it is populated correctly, and it is `.synthesis`-nested
— so the writer was never at fault and no code fix is owed.
**The correct reads, all three verified first-party on live artifacts:**
```bash
jq -r '.synthesis.absent_reviewers'                        # absence, authoritative
jq -r '[.reviewers[] | select(.present==false) | .model]'  # the same fact, cross-checked
jq -r '[.reviewers[].result.verdict]'                      # per-reviewer verdicts
```
Cross-check confirms they agree: the one artifact on disk with a real absentee
(`m-cohort-manifest-build-provenance-2026-08-23T04-06-…`) names `gpt5-6-sol` in BOTH.
Mission-independent — every mission on this rig reads the same artifact schema — and the
generalisation is this file's own recurring shape aimed at itself: **a rule that prescribes a
QUERY is only as good as the query's path, and a wrong path returns `null` rather than an error.**
Pair any documented `jq` read with a known-present sibling key in the same call (rule 3a aimed at a
JSON path), so a `null` proves the key is absent rather than that you spelled it wrong.
**And check `presentCount` has not been satisfied by YOU.** V1's artifacts carry three syntheses
reading `proceed` with **zero of two** model reviewers present
(`m-check-strict-fallbacks-2026-07-17T07-58-22Z`, `m-gemini-evaluator-diff-bridge-2026-07-16T23-03-39Z`,
`m-gemini-exec-project-plumbing-2026-07-16T17-44-15Z`) — and in all three the absent reviewers had
in fact said **reject**, recoverable from the captured raw text on disk. The zero-signal guard at
`internal/mission/quorum/quorum.go:168` did not fire because the controller's own
`--controller-verdict` increments the same `presentCount` it tests, and Gate 2 *mandates* that flag
(measured: **86 of 87** artifacts carry one; control — the single artifact without one reads
`blocked`). So the loop can pass a doc on its own self-assessment while the artifact says "quorum:
proceed". Filed as `#651` for the code fix; until it lands, the count that matters is **present
EXTERNAL reviewers**, and zero of them is a park, not a pass. The tell: you are about to route on a
quorum and you have not read `absent_reviewers` — or you have read it, seen a name, and let the
green verdict speak louder than the hole.
**AND WHEN A DOC BLOCKS ROUND AFTER ROUND, TRACK *WHICH SURFACE* EACH ROUND'S OBJECTIONS LAND ON —
BECAUSE EVERY RULE ABOVE TELLS YOU HOW TO ANSWER A BLOCKED QUORUM AND NONE TELLS YOU WHEN TO STOP
ANSWERING IT** (added 2026-08-23 V1 iteration 257; instance 1 is iteration 256, whose rounds 2 and 3
each blocked on a defect introduced by the previous round's fix, instance 2 is this iteration's
rounds 4 and 5 on the same doc, which did it twice more). The quorum machinery above is complete on
the question *"is this round's verdict trustworthy?"* — `absent_reviewers`, the external-present
count, the narrow-refinement carve-out, force-passing. All of it is scoped to ONE round, and each
round's disposition is *revise and re-quorum*. So a doc that blocks repeatedly has a rule for every
individual round and no rule for the sequence, and the loop's own discipline — every objection is
real, so answer it — is what keeps it revising. Note the trap is made of good behaviour: refusing to
force-pass is correct, and it is also what makes the loop unable to stop.
**The discriminating signal is not the round COUNT, it is where the objections LAND.** Objections
spread across surfaces mean the doc is immature — keep revising. Objections that **localise onto one
surface while another reviewer starts passing** mean the doc's *scope* is wrong: it bundles surfaces
with different correctness bars, and the hardest one is holding the others hostage. That is a
**decomposition** signal, and decomposition is a lane this skill already names ("the iteration's
deliverable is DECOMPOSITION into sprint-sized design docs") — filed under multi-week strategic
items, where a controller working a quorum will never look.
Measured on V1's `m-cohort-manifest-build-provenance`, five rounds, every objection real: R1 three
rejects across three surfaces; R2 and R3 on the controller's own cache predicate; **R4 on the freeze
gate and an acceptance assertion — neither of which R4 had touched**; R5 `gemini-3-1-pro` **PASS**
(first pass in five rounds) with `gpt5-6-sol` rejecting on the module-cache consumer alone. The doc
bundles three consumers — release evidence, compiler-cache identity, banking bucket — under one
shared cause, and *"what identifies compiler bytes"* is a strictly harder question than *"what
identifies release evidence"*. Note also what the round-5 objection turned out to be: a
**pre-existing** property of HEAD (`ModuleCacheKey` hashes a hand-bumped format constant, the commit,
the source hash and dep digests — `runtime.Version()` **0** in `internal/pipeline` against **4**
repo-wide, control firing), i.e. the doc was being blocked for not fixing something it never
introduced. **A pre-existing defect surfaced by a reviewer is a QUEUE ROW, not a revision** — file it
on its own first-party evidence and say so, rather than growing the doc until it can absorb it.
**Rules. (a)** From round 3 on, record per round which surface/consumer each objection names — one
line, in the doc's quorum log. **(b)** If the objections localise while any reviewer flips to pass,
the disposition is **SPLIT**, not revise: carry the surviving objection verbatim into the new doc's
opening problem statement, leave the reviewer-clean remainder in the parent, and re-quorum the
reduced doc once. **(c)** Before answering any objection, ask whether the defect is one the doc
INTRODUCES or one it merely fails to fix; measure it at HEAD (rule 3f) and route the second kind to
the queue. **(d)** A split is a controller routing call — it is **not** `needs-human-review`, and
filing it as one manufactures a decision the human does not have (standing rule 8). **(e)** Say the
round count in the report: a doc past round 4 is data about this loop's scoping, not about that doc,
and only the human can act on the pattern. Mission-independent — every mission on this rig runs the
same quorum. The tell: you are about to write "apply the fix and re-quorum" for the third time, and
each round's objections have been about a different part of the same document.

A design doc's status header is a claim, not a fact (M-EVAL-BENCH-UI shipped fully while its doc
said Planned for a month). **Also confirm the item is not ALREADY LANDED on origin** — check the
`origin/dev` queue tag (`git show origin/dev:design_docs/v1-mission.md | grep`) and any merged PR
(`gh pr list --search "<item> in:title" --state merged`) BEFORE starting a "resume" — iteration 12
ran a full redundant re-evaluation of an item that had already merged, because it trusted the stale
local queue/sprint-JSON (Gate 1's origin-sync now front-runs this, but re-check per item too). If
already done → the iteration's deliverable is the bookkeeping (move doc to implemented/ —
WITH its `*-sprint-plan.md` companions, Mark 2026-07-29: plans travel with their doc —
update queue, log it) and you pick the NEXT item too.
**The already-landed check must run against a FRESH origin, at pick time** (sharpened 2026-07-14
iteration 28; second instance of the landed-but-invisible class after iteration 12): re-run
`git fetch origin` immediately before the item-level check and grep `git log origin/dev --grep`
— NOT the local ref, which goes minutes-stale whenever a concurrent interactive session is
committing. And a PR search alone is NOT sufficient: direct-to-dev commits have no PR (iteration
28's Phase A landed mid-session as a direct commit `3bee6b6df`, invisible to both the stale
local log and the PR search; only the planner's own fetch caught it — the sprint was then
re-scoped in flight rather than pre-pick). When a sibling session is active (dirty shared tree,
fresh commits appearing), also send a controlplane CLAIM message naming the item before routing.

**AND CHECK FOR AN ITERATION THAT DIED MID-FLIGHT — ITS WORK IS INVISIBLE TO EVERY CHECK
ABOVE, IN EXACTLY THE WINDOW WHERE IT IS MOST NEARLY DONE** (added 2026-08-06 iteration 149;
2nd instance after iteration 121). Every check above looks for work that FINISHED: a merged
PR, a direct-to-dev commit, an origin queue tag. None of them sees an iteration that was
killed BETWEEN doing the work and landing it — and that is not a rare case, because the loop
runs headless on a timer under a 6h watchdog, so a slot can expire at any point. The traces
it leaves all sit outside the surfaces the checks search. Iteration 121's attempt died
leaving a 453-line design doc on an **unmerged branch**, invisible to a `design_docs/` grep
and to the origin/merged-PR checks. Iteration 149 found the sharper form: iteration 148 had
completed the ENTIRE inner loop for the queue head — executor, evaluator **PASS 88/100 r1
zero blocking**, a full out-of-sandbox acceptance sweep — opened PR **#600**, watched it go
green and MERGEABLE, then died before Gate 3b. It left **zero** charter rows, **zero** log
entries and **zero** STATUS stamps (`grep -c 'ITERATION 148'` = 0 in both files), so the
charter still read `[NEXT]` for a milestone that was finished, reviewed and ready. Acting on
that tag would have re-run a completed milestone and opened a duplicate PR against a green
one. Concretely, add two cheap searches to the ones above, at pick time: **(a)** open PRs
authored by this loop — `gh pr list --repo "$MISSION_REPO" --state open --author
sunholo-voight-kampff --json number,title,headRefName,mergeable` — matched against the item
you are about to pick; an open PR from your OWN account is either mid-flight work or
abandoned work, and both change the pick. **(b)** stale sprint worktrees — `git worktree
list` — whose branch names encode the item and the iteration (`.wt-iter148-ci-flake` on
`sprint/m-ci-flake-m5` was the corroborating trace here, and it is what upgraded "an open PR
exists" into "iteration 148 ran"). When you find one, the iteration's deliverable is to
VERIFY AND LAND it, not to redo it — but verify it exactly as any other inherited claim
(rule 3b(v)): re-derive its load-bearing counts and confirm its claimed edits actually landed,
because nobody has reviewed that work since the agent which wrote it stopped existing.
Iteration 149 did precisely that and the PR held up, but the check is what made merging it a
measurement rather than a hope. **(c) UNCOMMITTED WORKING-TREE STATE — in the stale worktree
AND in the MAIN CHECKOUT — because that is where a dying iteration's most recent work
necessarily sits, and (a) and (b) are both blind to it** (added 2026-08-07 iteration 161; two
frictions, both first-party in one iteration, and the rule's own trace list is what missed
them). (a) and (b) find *the existence of* an attempt: a PR record, a worktree directory.
Neither says anything about **content**, so both come back looking identical whether the
worktree is pristine or holds the milestone. Note the asymmetry that makes this the likelier
half: an iteration dies at its LAST step, and the last step before landing is always
"uncommitted edits exist". Iteration 161 found iteration 160 had left **two** residues, and
`git worktree list` showed only that a worktree existed: two untracked source files
implementing a whole milestone, and — the one no rule anywhere covers — a Gate-5 **skill edit
saved in the main checkout and never committed**, which the `~/.claude` symlink made LIVE for
every mission on the rig while it was absent from origin. That is the Repo Profile's one-way
divergence arriving by a mechanism note (c) there does not list: it covers an edit *committed
from a worktree*, not one *never committed at all*. Two commands, both cheap:
`git -C <each worktree> status --porcelain` and `git status --porcelain` in the main checkout,
read for files your predecessor would plausibly have written rather than for a clean/dirty
verdict. Gate 1's `cmp` against origin catches the skill case specifically — **so when it
fires, do not stop at "read the delta": ask WHY the running copy diverged**, because an
uncommitted edit means an author who never finished, and that author had other work too.
And verify what you find rather than adopting it: iteration 161's inherited milestone had
**2 of its 9 tests RED as delivered**, plus a third that passed locally and was vacuous on
another platform — an executor's unfinished output has been reviewed by nobody, not even by
the executor, which is exactly why the VERIFY-AND-LAND instruction above is worth more than
the traces that lead you to it. Finally, **record the orphaned iteration inside your own log
entry and credit it** — otherwise the log silently skips a number and no later reader can
tell whether that iteration ran, crashed, or never fired at all. When you find **two in a
row**, say so in the report as a pattern rather than as two incidents: the loop cannot
diagnose why its own slots are dying, but it can make the frequency visible to someone who
can.

**AND ON A SHARED REPO `--author` IS A *FLEET* FILTER, NOT A MISSION FILTER — SO TRACE (a)
RETURNS YOUR SIBLING'S PRs UNDER THE HEADING "YOUR OWN ACCOUNT", AND THE ONE IT WILL EVENTUALLY
HAND YOU IS THE SIBLING'S *LIVE, MID-FLIGHT* WORK** (added 2026-08-21 motoko iteration 17; the
frictions are 5+ consecutive motoko iterations each hand-disambiguating the same PRs, and this
iteration is the first where the filter returned a genuinely-mine PR *beside* a sibling's live
one). Trace (a) says *"an open PR from your OWN account is either mid-flight work or abandoned
work, and both change the pick"* — and every mission on this rig pushes as the SAME bot account,
so "your own account" is the fleet's account. Gate 1 already scopes a RED to the owning mission;
nothing scopes a **PR trace**, though it is the trace that tells you to go and *land something*.
Note the asymmetry that makes this worth a rule: the trace exists to find work you should ADOPT,
so its failure mode is not missing a signal, it is **acting on someone else's**. Rebasing,
force-pushing or merging a sibling's PR collides with a loop that is still running, and the two
missions cannot see each other — the overlap guard is a per-mission pidfile by construction.
Measured here: `--author sunholo-voight-kampff --state open` returned **three** PRs — `#813`
(mine, iteration 16's orphan, the correct pick), `#818` (V1's iteration 246, opened **20 minutes
earlier and live**) and `#695` (a stale coordinator PR, V1's). The motoko charter and log carry
**20** occurrences of *"is V1's"/"are V1's"*, i.e. this adjudication has been redone by hand every
iteration with no rule to do it by.
**The instrument is already in trace (b) and is clean by construction: `git worktree list` is
scoped to YOUR clone.** Measured, and the control is what makes it usable: the motoko clone lists
**8** worktrees, all `motoko`; V1's clone lists **12**, none of them motoko's — the two sets are
**disjoint**, because the missions are deliberately separate checkouts (the Repo Profile says so).
So a PR whose `headRefName` has a worktree in your list is *definitely yours*.
**Rules. (a)** Treat `--author` as necessary and never sufficient; attribute every hit before
acting. **(b)** Cross-reference `headRefName` against `git worktree list` in your own checkout —
a hit is proof of ownership. **(c)** A *miss* is NOT proof it is a sibling's (worktrees get
pruned, and a merged branch's may be gone), so fall back to a second reading — a `MISSION_NAME`
token in the branch, your own iteration numbering, or your charter — and where none resolves,
**leave it alone and say so in the report**: the safe default is that an unattributable PR is not
yours. **(d)** Never rebase, force-push, comment on or merge a PR you have not attributed to this
mission; if it looks like it needs attention, hand it over on the cross-mission channel exactly as
Gate 1 requires for a red you do not own. Mission-independent wherever two missions share a
`MISSION_REPO` and a push identity, which on this rig is V1 and motoko by design. The tell: you
are reading a PR list you filtered by author, and the word "own" in the rule is doing work the
filter cannot.

**AND THE ITEM'S DECLARED BLOCKERS ARE CLAIMS TOO — RE-VERIFY THE BLOCKER, NOT JUST THE ITEM,
AND CHECK WHETHER ITS *PURPOSE* WAS SOLVED UPSTREAM RATHER THAN WHETHER IT IS STILL OPEN** (added
2026-08-05 iteration 145; second instance of the solved-upstream class after the 2026-08-04
motoko_agent batch, where 1 of 3 "new" reports was already fixed and a 2nd was superseded). Every
rule above points the freshness check at *the thing you are picking*. Nothing pointed it at the
**collisions, blocking PRs and external dependencies that the doc or plan declares** — and those
are the claims most likely to rot, because they describe *someone else's* work, which moves without
telling you. They also feel pre-verified: a planner wrote them down, a quorum read them, and they
carry a PR number, which reads like a citation. Iteration 142 confirmed PR `#532` first-party as a
live collision blocking M2 — and it *was* `CONFLICTING`, which is what got checked. What was never
checked is whether `#532` was still **needed**: its entire purpose (one shared binary build instead
of fourteen, to escape a Windows timeout) had landed independently on `dev` as `#564`/`3c28cc322`
two days earlier. So iterations 142, 143 and 144 each carried "resolve `#532` before M2" in the
charter, and iteration 145 spent a real controller decision (plan §6.1: land-first vs rebase-after)
choosing between two options for a blocker that no longer existed. Worse, the same iteration then
posted a comment on `#532` asserting its fix was "still wanted" — inheriting the plan's description
of the PR instead of measuring HEAD, which is rule 3b(v)(b) exactly. The tell that a blocker is
dead is never its own state: `#532` sat `OPEN`/`CONFLICTING` for a week *because* it was superseded
— nobody rebases a PR whose reason is gone, so **staleness looks identical to importance**.
Concretely, at pick time, for each declared blocker/collision: **(a)** ask what problem it exists to
solve, then check whether that problem is still present at HEAD — `git log -S '<the symbol it
introduces>'`, or run the failing scenario — rather than reading its `state`/`mergeable`;
**(b)** treat `OPEN` + long-untouched as *evidence toward* superseded, not evidence of blocking;
**(c)** when it is dead, close it with the measurement and say so in the charter, so the next three
iterations do not re-plan around it; **(d)** never quote a PR's or issue's *purpose* from a doc that
merely cites it — re-derive it from the diff or the commit that superseded it. Cheap, and the whole
check is two commands.

**AND THE RULE ABOVE FIRES ONLY ON THE ITEM YOU PICK, WHICH IS NEVER THE BLOCKED ONE — SO A ROW
WAITING ON AN EXTERNAL PREDICATE IS RE-CHECKED ONLY BY THE ITERATION THAT WOULD ALREADY HAVE
UNBLOCKED IT** (added 2026-08-18 motoko iteration 11; instance 1 is motoko iteration 3, instance 2 is
this iteration, and the second happened *inside the remedy written for the first*). Everything above
is scoped "at pick time, for each declared blocker" — correct, and it leaves a circular hole: a row
you do not pick is never re-verified, and you do not pick it **because** it is blocked. Meanwhile the
predicate is what decides the row's ORDERING, so when it flips, the queue is silently wrong about
what comes next and no gate in this file looks. The tell is grammatical: a row that says **"still"**
— *still open*, *still zero events*, *still no reply* — where "still" reads as a re-check and is in
fact a transcription (rule 3b(v)(b)) of a measurement taken by an earlier iteration.
**A TIMEBOX MAKES THIS WORSE, NOT BETTER, WHICH IS THE PART WORTH THE RULE.** The natural fix for an
unbounded external wait is a deadline — and a deadline invites you to check the CLOCK instead of the
PREDICATE, so it converts "nobody knows when this unblocks" into "nothing to do until <date>", which
reads like coverage. Measured on motoko: iteration 3 found item 5 waiting unbounded on an upstream
maintainer's reply, correctly called that the same defect a reviewer had blocked Phase 0 for, and
replaced it with *"if no response by 2026-08-27, file anyway"*. He replied **2026-08-13T18:45:54Z**
— agreeing on all four points and **explicitly inviting** the issue the row existed to file.
Iteration 4 then wrote *"still zero `arniwesth` events on #97; 2026-08-27 stands"* on **08-14**,
after the comment, and iterations 5–10 each carried the sentence forward. Five days of a nine-day
window, on a row sitting above two ungated items, and the deadline was still fourteen days out when
it was found — i.e. the timebox would eventually have caught it, at maximum cost, and by then the
invitation would have been the *last* thing anyone learned rather than the first.
**Rule.** At Gate 1, when reading the queue, re-evaluate the PREDICATE of every row blocked on an
**external** party (an upstream reply, a third-party PR, a release, a person) — not the rows you are
picking, and not the deadline. It is one API read per row and the queue has a handful of them.
**(a)** Run the predicate as a command with rule 3a's control, exactly as if you were picking it —
the reply that unblocks you does not announce itself, and the flip is invisible in the row's text.
**(b)** Where the row's prose says "still", require the word to carry the date and command of THIS
iteration's measurement, or delete the word — an undated "still" is the whole defect.
**(c)** A timebox is a floor on when you act, never a ceiling on when you look; a row with a future
deadline gets the same predicate read as one without.
**(d)** When it has flipped, the row is this iteration's pick regardless of position, because the
ordering it was competing under was computed from a fact that is no longer true.
Mission-independent, and it generalises past queues: **a blocked thing is exactly the thing nobody
re-measures, so the freshness of a blocker is inversely proportional to how confidently it is
stated.** The tell: you are reading past a queue row because it is blocked, and you have not run the
one command that would tell you whether it still is.

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

**A finding you did not verify YOURSELF is a claim, however authoritative its author — never
launder it into an "established fact" for a downstream role** (added 2026-07-27 iteration 105;
second instance of the inherited-verification-debt class after iteration 25's survey rows). The
class is the same — a claim from another author, propagated without the controller's own live
re-check — but the surface is new and more dangerous: it is the controller's OWN reality-check
sub-agent, whose output feels like first-party evidence. Iteration 105's Gate-2 Explore agent
reported "verification is already wired for agent mode (`agent_runner.go:316`)" with a plausible
file:line; the controller handed it to the planner under the heading TREAT AS ESTABLISHED FACT.
It was FALSE — that call sits in `RunAgentBenchmark`, a function whose only repo reference is a
comment saying it must NOT be used, while the live path had zero verification. Had the planner
not re-checked on its own initiative, the sprint would have shipped a cohort-freeze mechanism for
a headline KPI that could never produce a number, and the next iteration would have spent real
metered dollars on a cohort run guaranteed to bank a zero denominator. Rules: (a) label every
handed-down fact with its provenance — `VERIFIED BY ME (<command/file:line>)` vs `UNVERIFIED,
inherited from <role> — re-check before relying`; (b) any claim that a capability EXISTS, that
code is REACHABLE, or that a bug is ALREADY FIXED is load-bearing enough to warrant the
controller's own 2-minute probe (`grep` for callers, not just for the call) before it is routed;
(c) the cheap probe for "is this code live?" is a CALLER search — a `RunAICheck(` hit proves a
call site exists, never that anything reaches it; (d) when a sub-agent contradicts a fact you
handed it, that is a SUCCESS of the loop — record it in Ruled out and fix the provenance habit,
never wave it through.

**This applies to the EVALUATOR/JUDGE too, and a judge's finding may be UNDER-stated as easily as
over-stated** (added 2026-07-28 iteration 111; the gap iteration 110 recorded as "watch-item,
instance 1", now at its second instance). The rule above is written for sub-agents feeding the
controller *facts*; a judge instead feeds you *findings*, which feel like conclusions rather than
claims and so slip past the provenance habit. Iteration 110's evaluator reported a mutation that
survived both suites (true — but reproduced first-party before being acted on). Iteration 111's
evaluator reported that the public guide taught a function name absent from the example file; that
was true AND worse than reported — the judge filed it as a maintenance nit, while the guide in fact
hands the user copy-pasteable commands against the very file lacking the name. Rules: (a) reproduce
a judge's finding before acting on it, exactly as for any other sub-agent claim; (b) reproduce it
before DISMISSING it too — a NON-BLOCKING label is the judge's opinion of severity, not a
measurement, and re-checking is how you discover the finding was bigger than filed; (c) when you
close a judge's finding by changing code, prove the new guard actually catches the defect (add the
assertion while the mutation is still applied), then revert and confirm byte-identity; (d) a PASS
with zero blocking findings is not a reason to skip this — both instances of this class arrived
inside passing evaluations.

**Verification protocol** (added iteration 1 after three same-class frictions; now 18
rules, each one an epistemics failure this loop committed and paid for). Steps 1–3 are the
`go-compiler` verify profile (V1); under `ailang-code` the shipped binary IS the gate — skip
the compile/staleness steps and run `ailang check`/`ailang test`/`ailang ai-check` instead
(see the Repo Profile above).

**Read [`resources/verification-protocol.md`](resources/verification-protocol.md) in full
before you verify anything at this gate.** The list below is a recall aid for rules you have
already read — it carries each rule's claim but none of its discriminating commands, and a
rule you can name but cannot run is one you will violate. The rules are cited by number
throughout the later gates and in the mission log; the numbering is stable.

- **1.** Rebuild before any live check
- **2.** A parked test is a claim, not evidence
- **3.** Exit codes through pipes lie
- **3a.** A SEARCH THAT FOUND NOTHING IS A CLAIM, NOT A FACT — and so is any probe that came back empty
- **3b.** A PASSING check is a claim too — match its SCOPE and its VERSION to the sentence you cite it for
- **3c.** "THE SERVICE" IS AN ASSUMPTION — a probe identifies the endpoint you REACHED, never the service you NAMED
- **3d.** A RESULT THAT CAME BACK RED IN THE DIRECTION YOU PREDICTED IS THE MOST SEDUCTIVE CLAIM OF ALL — IT NEEDS A NEGATIVE CONTROL EXACTLY AS MUCH AS AN EMPTY RESULT NEEDS A POSITIVE ONE
- **3e.** BASELINE EVERY ACCEPTANCE COMMAND ON A PRISTINE TREE — A GATE ALREADY RED AT BASE MEASURES THE REPO, NOT YOUR CHANGE, AND A CONTROL RUN AFTER AN EARLIER STEP HAS MUTATED SHARED STATE IS NOT A CONTROL
- **3f.** A REVIEWER'S OBJECTION IS A CLAIM TOO — WHEN A QUORUM BLOCKS ON AN "UNVERIFIED PREMISE", THE CONTROLLER'S JOB IS TO *MEASURE* IT, NOT TO FORWARD IT
- **3g.** YOUR LOCAL GATE SWEEP IS A HAND-PICKED SUBSET; THE CI JOB'S OWN COMMAND LIST IS KNOWABLE, SO DERIVE IT INSTEAD OF REMEMBERING IT
- **3h.** AN EXECUTOR'S DEVIATION FROM THE PLAN IS A CLAIM IN *BOTH* DIRECTIONS — ADJUDICATE IT BY MEASUREMENT, AND NEVER BY A "DEVIATIONS ARE SUSPECT" PRIOR, WHICH GETS MOST OF THEM BACKWARDS
- **3i.** A TEST-PLAN ROW'S "KILLS WHICH MUTATION" COLUMN IS A CLAIM, AND IT IS THE ONE CLAIM IN THE WHOLE SPRINT NOBODY EVER CHECKS — RUN THE NAMED MUTATION AGAINST THE ROW THAT NAMES IT, NOT AGAINST THE SUITE
- **3j.** WHEN A MILESTONE'S DELIVERABLE IS A REFUSAL, THE UNIT OF MUTATION IS THE *BRANCH*, NOT THE MILESTONE — AND A ONE-SHOT ACCEPTANCE COMMAND IS NOT A GUARD
- **3k.** IF THE PRODUCT HANDS A HUMAN SOMETHING TO RUN, A TEST MUST RUN EXACTLY THAT — A TEST THAT REBUILDS THE SAME COMMAND BY A SECOND ROUTE VERIFIES YOUR ARITHMETIC, NEVER YOUR ARTIFACT
- **3l.** "ENVIRONMENTAL" IS A CLAIM, AND THE FLEET IS ITS CONTROL GROUP — THREE MISSIONS RUN ON THIS RIG, SO ANY "IT'S THE MACHINE, NOT US" DIAGNOSIS HAS A READY-MADE THIRD ARM, AND SKIPPING IT COSTS MONTHS
- **3m.** A STRESS OR LOAD CONTROL ONLY CERTIFIES THE AXIS YOU VARIED — AND WHERE A BOUND AND ITS STIMULUS BOTH SCALE WITH THE MACHINE, THE BOUND MUST BE *DERIVED* FROM THE MEASURED STIMULUS
- **3n.** YOUR MUTATION SET IS DERIVED FROM WHAT THE MILESTONE *FIXES*, SO IT SYSTEMATICALLY MISSES WHAT THE MILESTONE *SHIPS* — ANCHOR THE ENUMERATION TO THE DIFF, WHICH IS COMPLETE BY CONSTRUCTION
- **4.** The shared main checkout is mutable mid-iteration
