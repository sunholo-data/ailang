# Gate 1 — reading CI health when the red is not about your code

On-demand reference for `mission-control`'s Gate 1. Split out of `SKILL.md` 2026-09-03 under the
progressive-disclosure convention (`.claude/rules/context-docs.md`): both rules below are war
stories with discriminating commands, needed only when a specific shape shows up, and neither is
worth a line in every session that loads the skill.

Read the first when a red arrives that you cannot attribute to the diff. Read the second whenever
you are about to record a Gate-1 health verdict at all — its trap is that nothing looks wrong.

---

## The red can be the CI provider itself, and then the deliverable is the diagnosis

**BUT A RED CAN BE THE CI PROVIDER ITSELF, AND THEN THE DELIVERABLE IS THE DIAGNOSIS — NOT A FIX,
AND EMPHATICALLY NOT A REVERT** (added 2026-08-06 V1 iteration 153; instance 1 was `mission-world`
iteration 58 the SAME DAY, which hit the identical signature in a different repo and carried it
into its next iteration as unfinished business). Everything above assumes the red is *about your
code* — a latent bug, a new advisory, a stale gate. It offers exactly two dispositions, fix or
revert, and both are WRONG when the provider is down. That matters because the rule "a RED dev
outranks the queue" plus "the fix IS the first deliverable" reads as a standing instruction to
*change something*, and the most available change during an outage is reverting the most recent
merge — i.e. destroying good work to appease an unrelated infrastructure event, while the real
green is unobtainable to confirm it either way. Rule 3d in its purest form: the red arrived right
after a merge, in the direction you would predict, and the co-occurrence is the whole illusion.
The discriminating signature is **that no repo command ever ran**: `steps=0` on the job, or a
failure whose last step is `Set up job`, i.e. before checkout. Read it with
`gh api repos/<o>/<r>/actions/runs/<id>/jobs --jq '.jobs[] | "\(.name) [\(.conclusion)]:
steps=\(.steps|length) last=\(.steps[-1].name // "-")"'` — `--log-failed` is useless here, because
there is no log to fail. **CORRECTION, measured 2026-08-06 iteration 154 on the FIRST re-use of
this rule: the signature above is a FAMILY, and `steps=0` is only its commonest member — do not
read it as an invariant.** Eleven of twelve failing jobs on that iteration's PR matched it exactly;
the twelfth, `Build macos-latest`, ran **17 steps, every one `success` or `skipped` — including
`Run tests`, `Build binary`, `Upload artifact` and `Complete job` — and the JOB still concluded
`failure`.** A job whose every step passed is not a code failure; the platform failed it outside
step execution. Read strictly, "no repo command ever ran" would have classed that one job as a
genuine regression **and pointed at exactly the revert this whole rule exists to prevent** — the
one non-matching job in a set of twelve is the one you would have blamed. So ask the question the
signature is a proxy for: **is the failure attributable to any STEP?** If no step failed, nothing
in the diff did, whatever `steps` counts. Then establish it with controls rather than vibes, because "CI is flaky
today" is exactly what someone says right before reverting a good commit: **(a)** the SAME jobs on
the PARENT commit — green there, minutes earlier, is the before-arm; **(b)** the provider's own
status API (`curl -s https://www.githubstatus.com/api/v2/summary.json`), checking that the
incident WINDOW covers your run's `createdAt`, not merely that an incident exists; **(c)** the
strongest control available and the one to reach for — **re-run and look for outcome divergence on
a byte-identical tree**; iteration 153's `docs` job went cancelled→success across re-runs with no
code change, which is only possible if the variable is the environment; **(d)** a sibling mission
or unrelated repo hitting the same signature in the same window. Disposition: **do not revert, do
not fix-forward, do not park the whole iteration.** Fire the re-runs (they queue and drain as
capacity returns — Build-and-Release recovered on its own this way), record the diagnosis with its
controls as the deliverable, and say plainly in the report that dev is red for a declared outage
and re-running is owed once the incident closes. Then **pick something that does not need the
landing gate**: an analysis, a triage, a decision-bearing investigation. And when the sprint work
IS done, Gate 3b still binds — **0 failures observed is not a green**, so the item does not become
LANDED; it becomes a resume point, named as such in the charter. The tell you are about to pay for
this: you are reading a red run's logs and finding nothing, and reaching for `git revert`.

---

## A failing job suspends every gate behind it, and their silence reads as green

**(added 2026-09-03 V1 iteration 323; instance 1 is iteration 154's Gate-3b rule that *"an
aggregate over an INCOMPLETE check set is vacuously green"*, which is correct and is scoped to
the check-run **SET** — it asks whether every expected check is PRESENT, and cannot see inside
one; instance 2 is this iteration, where every expected check WAS present and four further
defects were hidden anyway.)** Every Gate-1 rule teaches you to read the checks that RAN: is this
red inherited, is it the provider, does a run exist at all. None asks what a red **cost you the
ability to measure**. A CI job is a long ordered list of steps, and a failure at step *k*
`skip`s steps *k+1…n*; the check set then reports **one** failing check, in exactly the voice of
a repo with one defect.

**The bias is the dangerous part: the numbers all move in the reassuring direction.** A
controller who fixes the step-*k* red and watches the check count go green has, at that moment,
*more* unmeasured surface than when it started — because the gates behind it are now running for
the first time in however long the red stood. Fixing the visible red is therefore the step that
CREATES the risk of a premature "dev is green", and Gate 3b's discipline (observe, never predict)
is the only thing standing there.

Measured on V1, `327db37cd`..`b51e53f78`, ~24 hours: `check-file-sizes` is **step 15** of the
`test` job and the job was dying at **step 11**, so steps **12–60 — 45 gates —** read `skipped`.
Two files crossed the repo's 800-line limit inside that window (`backend_gcp.go` **788→811**,
`inbox.go` **773→850**; both were under the limit and step 15 read `success` at the last green
`7668ed9df`) and no instrument in this loop could see it. The same mechanism fired twice more in
different clothes, which is what makes it a shape rather than an incident: inside the **`lint`
job's** step list a `golangci-lint unused` red sat behind an `fmt-check` red, and across the
**build matrix** `fail-fast` left `Build windows-latest` reading `cancelled` on *every* recent
dev commit while it concealed a 21-test failure. A sixth, `SonarCloud`, is step 58 and read
`none` throughout. **One visible red, six real ones.**

**Rules.**

**(a)** When a job fails, read its STEP list — `gh api repos/<o>/<r>/actions/jobs/<id> --jq
'.steps[] | "\(.number) \(.conclusion) — \(.name)"'` — and COUNT the `skipped` steps behind the
failure. That count is the size of your blind spot, and it belongs in the report as a number, not
as a caveat. `check-runs` reports the job and never its steps, so no amount of care at the
check-set level reaches this.

**(b)** Never write "dev is green" on the strength of fixing the red you found; write "dev's
first *n* gates are green and gates *k+1…n* ran for the first time in *T*", and let the
post-merge run say the rest.

**(c)** Treat a `cancelled` matrix leg as **UNMEASURED**, never as "not our problem" —
`fail-fast` makes one leg's red erase every other leg's verdict, and the erased legs are exactly
the platforms your local gates cannot stand in for.

**(d)** Expect the count of defects to GROW as you fix them, and budget the iteration for that;
an iteration that finds a second defect behind the first is the instrument working, not scope
creep.

**(e)** The durable fix is a **job-shape** change (split independent gates out of the test job,
`continue-on-error` with a final aggregator, `fail-fast: false`) and it is a design question with
a real CI-minutes cost — so it is a QUEUE ROW, not something to improvise while landing a red
fix. V1 filed it as `m-ci-serial-gate-masking`.

Mission-independent — every mission on this rig runs long ordered jobs, and under `docs-site` the
same trap is a `make docs-build` failure suspending `verify-examples` behind it. The
generalisation is this file's own recurring shape aimed one level below where it has been
looking: **this loop has learned to ask whether the check set is complete, and has never asked
whether a check that RAN got to the end of its own list.** The tell: you fixed a red, the check
count went green, and you have not looked at how many steps were `skipped` in the run you were
reading before.
