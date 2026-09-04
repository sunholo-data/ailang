# The codex/pi executor lane — the five false-greens, and the gate list you write yourself

On-demand reference for `mission-control`'s Gate 3 cross-provider executor recipe. Split out of
`SKILL.md` 2026-09-04 (V1 iteration 327) under the progressive-disclosure convention
(`.claude/rules/context-docs.md`): every rule below is a war story with discriminating commands,
needed only when you are actually driving a sandboxed executor lane, and none is worth a line in
every session that loads the skill. Nothing was reworded in the move; (5) is new.

**Read this in full before writing a directive for any `codex:` or `pi:` role, and again before
recording a Gate-4 verdict on that role's output.** (1)-(2) are about a run that never happened,
(3) about a verdict you cannot trust, (4) about the gate list you wrote yourself, and (5) about a
destructive side effect the sandbox hid from the executor and hands to you.

---

**THREE FALSE-GREENS this recipe used to carry** (all proposed by `world-coordinator` from
mission-world, which shares this skill but cannot edit it; each corroborated first-party before
being written in, rather than taken on trust — the sibling-claim ghost discipline).
**(1) stdin was never redirected**: `codex exec` reads stdin IN ADDITION to the positional
prompt, so under a backgrounded launch with an open (never-EOF) stdin it prints
`Reading additional input from stdin...` and blocks until the 30-min cap — a hang that *looks*
like normal long work (World: 39-byte log, zero diff, 6 minutes). That line appears in
iter-111's own `codex_out.log`; the run survived only because stdin happened to EOF.
**(2) delivery was never asserted**, as above. Both are the vacuous-pass class this mission has
closed twice elsewhere (silent z3 skip, silent `t.Skip`): *an exit code reporting success for
work never requested*. Iter-112 hit the near-miss live — its first `Write` of the directive file
FAILED (pre-existing file) and, unnoticed, would have produced exactly defect (2). *(Cheap
habit that closes it: give the directive file a per-iteration name, e.g.
`/tmp/codex_directive_iter<N>.txt` — `Write` refuses to overwrite a file this session has not
Read, so a fixed name collides with the previous iteration's leftover.)*
**(3) A GATE VERDICT FROM INSIDE THE SANDBOX IS NOT EVIDENCE.** `workspace-write` DENIES
loopback socket binds, so any suite touching `httptest`/servers fails with
`bind: operation not permitted` — an infrastructure denial that is **indistinguishable from a
real regression** in the exit code. V1 hit this three times (iters 110, 111, 113: `make test`
exit 2 each time, **rc=0 with zero FAIL** on the controller's re-run outside the sandbox), and
World hit the same wall from the other side — its sandbox panic on a loopback bind *masked* a
genuine `io.Pipe` startup deadlock. So it cuts BOTH ways: the sandbox invents failures **and
hides real ones**. Rule: the executor MUST label any such result
`UNINFORMATIVE UNDER SANDBOX` rather than reporting it as pass or fail (say so in the
directive), and the **controller MUST re-run the gates outside the sandbox** before recording
any Gate-4 verdict — mandatory whenever the diff touches `host/`, `daemon/`, `cmd/*`, or
anything serving a socket. Never bank an executor-reported gate result for those paths.
**(4) THE GATE LIST *YOU* WRITE INTO THE DIRECTIVE IS AN ACCEPTANCE LIST TOO, AND RULE 3e(a)
DOES NOT REACH IT — SO THE ONE GATE LIST NOBODY BASELINES IS THE CONTROLLER'S OWN** (added
2026-08-21 V1 iteration 245; instance 1 is iteration 147's `actionlint` plan gate, instance 2 is
this iteration's). Rule 3e(a) says to run each acceptance command on a pristine base before
routing, and every word of it is scoped to a **sprint plan's** acceptance list — written by a
planner, read at pick time. A direct-fix iteration has no plan and no planner: the controller
writes the gates straight into the directive, at which point no rule in this file has ever asked
whether they pass on untouched `dev`. That is this loop's own *guard the helper, miss the call
site* shape aimed at its own hands, and it is why 3e(a) can be documented, cited, and still
bought a third time. Measured here: my directive made `go build ./...` the mutant-BUILDS
assertion; it is **rc=1 on pristine dev** (`cmd/wasm` and `gen/main` have no native `main` —
the identical finding iteration 145 recorded), against `./cmd/ailang` rc=0 and
`./internal/builtins/...` rc=0. The executor stopped mid-sprint rather than assert a mutant
built, which cost a second run and was **the correct call**.
**Rules. (a)** Before sending any directive, run its gate list on the base and delete or repair
anything already red — the same discipline 3e(a) applies to a plan, applied to the list you just
typed. **(b)** Prefer the narrowest gate that can actually fail for your diff
(`go build ./internal/<pkg>/...`) over the widest that looks thorough (`./...`); a whole-repo
build is *more* likely to be red at base, not less. **(c)** Say in the directive that a gate the
executor finds red at base is a finding to REPORT, not an obstacle to work around — and treat
the report as the loop working (rule 3h(d)), never as non-compliance. **(d)**
Mission-independent: under `ailang-code` the same trap is an `ailang check` over a module set
that does not resolve on untouched `dev`. The tell: you are about to hand an executor a list of
commands and you have not run one of them yourself.

**(5) FALSE-GREEN (3) SAYS A GATE VERDICT FROM INSIDE THE SANDBOX IS NOT EVIDENCE. THE MIRROR
IS WORSE AND IS NOWHERE IN THIS FILE: A DESTRUCTIVE *WRITE* OUTSIDE THE WORKTREE IS DENIED
INSIDE THE SANDBOX, SO THE EXECUTOR'S RUN OF A STEP THAT WILL LATER DESTROY SHARED STATE
REPORTS SUCCESS — AND THE CONTROLLER'S MANDATORY OUT-OF-SANDBOX RE-RUN IS WHERE THE DAMAGE
ACTUALLY HAPPENS** (added 2026-09-04 V1 iteration 327; instance 1 is iteration 326, whose
sandboxed harness runs wrote four synthetic `[local] …` rows into the REAL shared
`~/.ailang/state/autopush.log` and were FLAGGED, instance 2 is this iteration, where the same
boundary hid a defect instead of merely leaking through it). False-green (3) is written about
*reading* a verdict — it cuts both ways, it says, because the sandbox invents failures and
hides real ones. Every word of it is about a **result**. Nothing points it at a **side
effect**, and the two have opposite signatures: a denied gate is loud (`bind: operation not
permitted`), while a denied write inside a test's setup is silent, because the test then
passes for the wrong reason. Note who is most exposed, and that it is by this file's own
instruction: false-green (3) *requires* the controller to re-run the gates outside the
sandbox. So the executor cannot see the defect and the controller is obliged to trigger it.
Measured here, at a cost that is not recoverable. A milestone's acceptance criterion was
worded as a **seeded write** — put a known sentinel line into the caller's real
`$HOME/.ailang/state/autopush.log`, then assert it is unchanged — to prove a test harness
never pollutes the shared fleet log. The executor implemented it literally
(`printf … > "$CALLER_LOG"`), its sandboxed run reported the arm PASSING, the judge would
have seen the same, and the first re-run outside the sandbox truncated that log from **92
lines to 1**. No second copy under `$HOME` (`find` → 0 hits) and no usable local snapshot;
the evidence is gone. A test written to prove the harness never touches the caller's log was
the thing that destroyed it — the vacuous-pass class this loop keeps closing, arriving
through a *guard*.
**Rules. (a)** Before the controller's out-of-sandbox re-run, read the diff for any command
that WRITES outside the worktree — `>`/`>>`/`tee`/`rm`/`mv`/`truncate` against `$HOME`,
`~/.ailang/state/`, `/usr/local`, a shared log or a database — and treat each as
**UNVERIFIED, POTENTIALLY DESTRUCTIVE** rather than as tested. The executor's green says
nothing about it. **(b)** Rehearse such a step under a synthetic root first
(`HOME=$(mktemp -d) …`) and diff the real artifact's sha before and after; that one habit
converts this from unrecoverable to a non-event. **(c) Specify the guard as an OBSERVATION,
never as a seeded write.** "Sha and line-count the artifact on both sides, with `absent` a
legitimate reading" kills exactly the same mutation as a sentinel and cannot destroy
anything — verified here: dropping the harness's `HOME` export still reddens the arm (3 lines
→ 17, sha differs) plus four others. **(d)** This binds the DESIGNER and PLANNER as much as
the executor: the defect was authored in an acceptance criterion, not in the code that
implemented it faithfully. A criterion of the form "assert X is unchanged" must say how X is
established, and if the answer is "by writing to it", the criterion is the bug.
Mission-independent — every lane that sandboxes writes (codex `workspace-write`, pi's
worktree fence) has this shape, and `ailang-code`'s version is a step that mutates the shared
lockfile or registry. The generalisation: **a sandbox makes a destructive step
indistinguishable from a harmless one, so "it passed under the sandbox" is not evidence of
safety — only of confinement.** The tell: an acceptance criterion mentions a path outside the
worktree, and the only run of it so far was a sandboxed one.

**(5) FALSE-GREEN (3) SAYS A GATE VERDICT FROM INSIDE THE SANDBOX IS NOT EVIDENCE. THE MIRROR IS
WORSE AND WAS NOWHERE IN THIS RULEBOOK: A DESTRUCTIVE *WRITE* OUTSIDE THE WORKTREE IS DENIED INSIDE
THE SANDBOX, SO THE EXECUTOR'S RUN OF A STEP THAT WILL LATER DESTROY SHARED STATE REPORTS SUCCESS —
AND THE CONTROLLER'S MANDATORY OUT-OF-SANDBOX RE-RUN IS WHERE THE DAMAGE ACTUALLY HAPPENS** (added
2026-09-04 V1 iteration 327; instance 1 is iteration 326, whose sandboxed harness runs wrote four
synthetic `[local] …` rows into the REAL shared `~/.ailang/state/autopush.log` and were FLAGGED,
instance 2 is this iteration, where the same boundary HID a defect rather than merely leaking
through it). False-green (3) is written about *reading* a verdict — it cuts both ways, it says,
because the sandbox invents failures and hides real ones. Every word of it is about a **result**.
Nothing pointed it at a **side effect**, and the two have opposite signatures: a denied gate is
loud (`bind: operation not permitted`), while a denied write inside a test's setup is **silent**,
because the test then passes for the wrong reason. Note who is most exposed, and that it is by this
rulebook's own instruction: (3) *requires* the controller to re-run the gates outside the sandbox.
So the executor cannot see the defect, and the controller is obliged to trigger it.

Measured, at a cost that is not recoverable. A milestone's acceptance criterion was worded as a
**seeded write** — put a known sentinel line into the caller's real
`$HOME/.ailang/state/autopush.log`, then assert it is unchanged — to prove a test harness never
pollutes the shared fleet log. The executor implemented it literally
(`printf … > "$CALLER_LOG"`), its sandboxed run reported the arm PASSING, an evaluator would have
seen the same, and the first re-run outside the sandbox truncated that log from **92 lines to 1**.
No second copy under `$HOME` (`find` → 0 hits), no usable local snapshot: the evidence is gone. A
test written to prove the harness never touches the caller's log was the thing that destroyed it —
the vacuous-pass class this loop keeps closing, arriving through a *guard*.

**Rules. (a)** Before the controller's out-of-sandbox re-run, read the diff for any command that
WRITES outside the worktree — `>`/`>>`/`tee`/`rm`/`mv`/`truncate` against `$HOME`,
`~/.ailang/state/`, `/usr/local`, a shared log or a database — and treat each as **UNVERIFIED,
POTENTIALLY DESTRUCTIVE** rather than as tested. The executor's green says nothing about it.
**(b)** Rehearse such a step under a synthetic root first (`HOME=$(mktemp -d) …`) and diff the real
artifact's sha before and after; that one habit turns this from unrecoverable into a non-event.
**(c) Specify the guard as an OBSERVATION, never as a seeded write.** "Sha and line-count the
artifact on both sides, with `absent` a legitimate reading" kills exactly the same mutation as a
sentinel and cannot destroy anything — verified here: dropping the harness's `HOME` export still
reddens the arm (3 lines → 17, sha differs) plus four others. **(d)** This binds the DESIGNER and
PLANNER as much as the executor: the defect was authored in an acceptance criterion, not in the
code that implemented it faithfully. A criterion of the form "assert X is unchanged" must say how X
is established, and if the answer is "by writing to it", the criterion is the bug.

Mission-independent — every lane that sandboxes writes (codex `workspace-write`, pi's worktree
fence) has this shape, and `ailang-code`'s version is a step that mutates the shared lockfile or
registry. The generalisation: **a sandbox makes a destructive step indistinguishable from a
harmless one, so "it passed under the sandbox" is not evidence of safety — only of confinement.**
The tell: an acceptance criterion mentions a path outside the worktree, and the only run of it so
far was a sandboxed one.
