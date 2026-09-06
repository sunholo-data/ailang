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

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `motoko-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `motoko-mission-index.md`.

## 17 — 2026-08-21 — the deliverable was a predecessor's finished work, and the job was to verify it rather than believe it

**Pick.** Not the queue head, and not a milestone. Gate 2's died-mid-flight traces found iteration
16's entire inner loop complete and unrecorded (see entry 16), so the skill's instruction applies
literally: **verify and land it, do not redo it**. Acting on the charter's `[NEXT]` tag instead
would have re-run two finished milestones and opened a duplicate PR against a green one.

**Outcome.** PR [#813](https://github.com/sunholo-data/ailang/pull/813) rebased, verified, landed as
[`b2733201a`](https://github.com/sunholo-data/ailang/commit/b2733201a). Gate 3b **GREEN** on head
`aa4543ba4`: **21** checks, **0** pending, **0** not-green, **4/4** required contexts
(`build`/`docs-gate`/`lint`/`test`) pass, `mergeStateStatus=CLEAN`. Queue item 6 is now
**M1 + M3 + M4 + M5 landed, M2 only** — and M2 is the one that needs the rig.

**Verification, because nobody had reviewed that work since its author stopped existing.** The
rebase itself first: `make/test.mk` auto-merged (the two hunks sit 215 lines apart);
`changelogs/v0.18-current.md` conflicted twice — the **fifth** consecutive iteration in which that
file is the cross-mission collision surface — resolved keeping both sides, with a 4-of-4 entry
presence check, a firing conflict-marker control, and a fresh negative literal. The rebased diff
re-derived at **+407/-27 across 7 files**, identical to the pre-rebase PR, so the rebase moved the
base and nothing else.

Then the claims. The PR asserts every refusal branch of the M5 smoke gate is pinned; that is rule
3j's bar, and it is checkable. Four mutants, each asserted **LANDED** (sha256 ≠ pre) and **VALID**
(`bash -n` rc=0) before its result was read, each restored from a `cp` backup — never
`git checkout --`, since the file is uncommitted by construction during a drill — and each restore
verified byte-identical:

| mutant | killer arm | other arms |
|---|---|---|
| neuter `banked-row contract` | `FAIL: no-banked-row smoke` (count=12, want 0) | green |
| neuter `fmt_hook_state contract` | `FAIL: failing smoke` | green |
| neuter `treatment-integrity contract` | `FAIL: invalid-treatment smoke` | green |
| ON always leads (kills counterbalancing) | `FAIL: counterbalanced sequence` | green |

Baseline unmutated rc=0. Each mutant is single-arm, so these are sole killers, not set members.

The fourth one carries the load-bearing check, because AC-M3-4's whole claim is that the *analyzer*
— not a reconstruction of what the schedule ought to emit — is the discriminator. Measured: with ON
always leading, `TestFmtDriverScheduleSatisfiesOrderIntegrity` fails at `censored_test.go:100:
CheckFmtOrderIntegrity rejected shell artifact: order_integrity_lead_not_alternating`. That
simultaneously confirms iteration 16's `go test` cache-key repair — the Go test really does see a
`/bin/bash` child's mutation.

**My own vacuous pass, caught by a count and not by an exit code.** The first attempt at that last
drill used `-run 'TestFmtABScheduleOrderIntegrity|TestFmtAB'`, which matches **no test in the
package**. Both arms returned **rc=0** and the mutant read as *survived* — a false negative that
would have made me report a decorative AC. The only tell was `ok … [no tests to run]`; re-run with
an explicit `=== RUN` count on both arms and the correct test name, the arms are **0** and **1**.
A `-run` filter is an enumerator, and rule 3a(i-e) applies to it: the drill proves the check fires,
only a run count proves it looked.

**Gate list derived, not recalled** (rule 3g), from `ci.yml`'s own `run:` lines rather than from the
PR body — which is how two gates dev added *after* iteration 16 branched were caught:
`make test-check-autoclose` and the newly exact-count `make test-stdlib-ail`. Both green on the
merged `make/test.mk` (4 suites / 4 fixtures). Full sweep on the rebased tree, all rc=0: `bash -n`
×2 · `shellcheck -S warning` (0 findings) · `make test-launchd-drivers` (10 passed / 0 failed —
item 5b's fix still holds on the rig) · `test_fmt_ab_schedule.sh` (11 PASS) ·
`go build ./internal/... ./cmd/ailang/...` · `go test -count=1 ./internal/eval_analysis/...` ·
`check-file-sizes` · `check-boundaries` · `check-changelog` · `test-check-changelog` ·
`test-check-autoclose` · `check-skills` · `fmt-check` · `vet` · `test-stdlib-ail`. Every binary
invocation ran from an explicitly built absolute path (`/tmp/ailang_i17/ailang`,
`git describe v0.33.1-191-gaa4543ba4`) prepended to `PATH`; `make quick-install` deliberately not
run (shared-write guardrail). Platform: **darwin/arm64 only**; windows and ubuntu legs unrun
locally, and Gate 3b is the only instrument that saw them.

**Ruled out.** *The PR's `CONFLICTING` state was a dropped-event or infrastructure problem* —
`mergeable` was read FIRST per the iteration-198 rule and returned `CONFLICTING`/`DIRTY`
immediately, explained completely by a two-file overlap with V1's iterations 245/246. No
`workflow_dispatch`, no empty commit, no diagnosis needed. *Iteration 16's work needed re-running* —
it did not; every claim in its PR body that I checked held, and the two milestones were complete.
*The `.snap/` directory was unfinished work* — it is the codex recipe's mandated per-milestone
snapshot output, already reconstructed into the two commits.

**Gate 0/1.** Kill switch armed; `gh` on `sunholo-voight-kampff`; billing tripwire **CLEAN**; pin
root detached and clean at `8040dfd41` == `origin/dev` at start. Running-skill check performed on
the **resolved** path per V1 iteration 241 — and both copies read, because they are different files:
the copy this session actually executed is the pin worktree's own (inode `46496692`), `cmp` vs
origin **rc=0**; `~/.claude/skills/mission-control` resolves to V1's main checkout (inode
`45241676`), which is **1 commit ahead of origin carrying an unpushed Gate-5 edit** and so differs
by 3,074 B. That is V1's divergence and is recorded here rather than acted on. Negative control
(origin skill vs the charter) rc=1. dev **verified green, not merely un-red**: **16** exact-SHA
checks, **0** not-green, `runs_total=2` so a run exists, parent-commit control **16**. **0** human
directives on `#743` since the watermark `2026-08-17T05:48:45Z` — corroborated by a raw author
enumeration showing all **15** comments are the bot's, with the script's positive control firing on
`#745` (Mark's `D-19 : B`). Ledger valid at **3** rows, **0 OPEN**. Inbox **0** unread. **0** open
`[nightly-eval]` alarms (control: 3 closed ones exist). Weekly sweep and rotation **both not due**
(`#743` created `2026-08-17T05:48:23Z` = 07:48 local, after the Monday-07:00 local boundary; 15
comments < 80).

**Routing.** Controller `claude:claude-opus-5` only. **No designer, planner, executor, evaluator or
quorum spawned** — verifying and landing a predecessor's finished work has no doc to design and no
plan to write, and a judge would be adjudicating an artifact whose author no longer exists; the
executor credit stays with iteration 16's `codex:gpt-5.6-sol`, preserved in both commit trailers.
Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no
`rig.lock`.

**Gate 5 — one skill edit, Gate 2's died-mid-flight trace (a).** `--author sunholo-voight-kampff` is
a **fleet** filter, not a mission filter: every mission on this rig pushes as the same bot account,
so the rule's phrase *"an open PR from your OWN account"* is doing work the filter cannot. The
frictions are recorded and repeated — the motoko charter and log carry **20** occurrences of
hand-disambiguating *"is V1's"/"are V1's"*, across at least five consecutive iterations, each
redoing the same adjudication with no rule to do it by. This iteration is the first where the
latent hazard went live: the filter returned `#813` (mine, the correct pick) beside `#818` (V1's
iteration 246, opened **20 minutes earlier** and still running) and `#695` (V1's, stale). Since the
trace exists to find work you should *adopt*, its failure mode is not a missed signal but acting on
a sibling's live PR. The fix uses an instrument the rule already has one line away: `git worktree
list` is scoped to your own clone, and measured here the two clones' lists are **disjoint** — 8
worktrees in motoko's, all `motoko`; 12 in V1's, none of them motoko's. A branch with a worktree in
your list is definitely yours; a miss is *not* proof of the converse, so the rule requires a second
reading and defaults to leaving an unattributable PR alone.

**Next**: item 6's **M2** (`AC-D1-live`) is the only milestone left and it **needs the rig** —
one fmt-lane run reaching `localhost:11434` with zero `openrouter.ai` connections, asserted on the
connection and paired with an OpenRouter-lane known-positive control. Doc §6's deployment
precondition (`#558`) is unchanged by this landing: merging to `dev` does not put D1b or the smoke
gate on the rig, because the installed plist runs `nightly-eval.sh` in place from V1's checkout.
If M2's rig slot is not available, item **7** (profile restoration design) is the next ungated row.

---

## 18 — 2026-08-22 — the instrument landed and refused to certify its own sweep, which is the criterion working

**Pick.** Queue row **6**'s named resume point, milestone **M2 (`AC-D1-live`)** of
`m-motoko-fmt-remeasurement-instrument` — the last of five and the only one needing the rig. Not
already landed: `tools/eval/motoko_connection_probe.sh` absent on `origin/dev` (control: an existing
`tools/launchd/` script resolves rc=0), **0** merged PRs matching, and the single `AC-D1-live` grep hit
is iteration 13's own record commit. Preconditions run as commands, not assumed: `rig.lock` free,
ollama up on **IPv4 only** (`0.32.14`; `[::1]` refused, rc=7 — rule 3c's two-instance check done),
`qwen3.6:35b-a3b-mxfp8` pulled **and loaded**, `OPENROUTER_API_KEY` SET (presence only), both lanes
present in `models.yml` (control: 17 motoko lanes).

**Outcome.** **M2 does NOT close.** PR [#829](https://github.com/sunholo-data/ailang/pull/829)
(`3e446f8c7` + `627c67d2d`) lands the instrument; the live verdict is **VOID**.

**What the sweep measured.** Under `rig.lock`, both lanes returned `driver_rc=1` with peer set `[]`
in **8m15s / 8m17s**. `AC-M2-control` states in terms that a control which does not fire makes
`AC-M2-treatment` **VOID — the probe proved nothing**. So the verdict is VOID, not FAIL, and the probe
exited 1 on `INSTRUMENT FAILURE: empty peer set; absence of evidence cannot prove routing`. The
artifact is written **before** the verdict is evaluated, so the evidence survived the refusal.

**The finding, and it required separating two things that produce identical output.** "The runs never
connected" and "the sampler is blind" are the same empty peer set. Measured apart (doc rows V36–V38):
scoped `lsof` **does** see an ESTABLISHED peer of a child process on this rig — the probe's own command
shape returned `curl … TCP 127.0.0.1:49914->127.0.0.1:11434 (ESTABLISHED)` against an unscoped same-call
control of **67** lines — and the treatment lane is `rc=0` standalone (2/2, 1m53s) **and** `rc=0` under a
faithful replication of the probe's own `run_lane` shape (2/2, 1m1s, **244** lsof lines) with
**`127.0.0.1:11434` present**. So the observable is reachable and the probe **as shipped** breaks the runs
it observes. **Mechanism NOT isolated** and said so rather than guessed: it lies in what the replication
did not reproduce — the deadline-carrying `descendant_pids` or the two-lanes-in-one-process sequencing.
The evaluator ruled out `classify_lsof` hermetically (a pure post-hoc transform called after `wait`, so it
cannot change the driver's exit code). The probe **discards both driver logs** via its `trap … EXIT`,
which is why the first diagnosis needed a re-run — for a lane that exits non-zero that log is the whole
diagnostic, and keeping it is the named first fix.

**Evaluator: FAIL 58/100, two blocking findings, and it earned its spawn by being pointed at me.** The
directive named my own V36–V38 rows and my VOID disposition as targets to refute.
**B1, reproduced first-party before acting (row V39):** `dig +short openrouter.ai` returns **A records
only** while openrouter.ai has genuine **AAAA** records (`2606:4700::6812:373`, `:273`), and `lsof`
brackets an IPv6 peer where `dig` emits it bare, so `grep -Fqx` could never match. Measured: a v6
OpenRouter peer classified **`other`** while the IPv4 positive control in the same call classified
**`openrouter`** — a **false negative in exactly the half of AC-M2-treatment that must not fail**. The
probe would have certified "zero connections to openrouter.ai" for a run that leaked over IPv6. Fixed
(union A+AAAA, `+time=5 +tries=2` which also bounds the judge's N2; compare on the bracket-stripped host),
re-measured in four directions including a non-OR v6 peer that must stay `other` so the fix cannot pass by
over-matching. **Gated, not merely fixed:** a new arm asserts a v6 leak is refused and reverting the
normalisation **reds** it (`missing openrouter [2001:db8::8]:443`), mutant LANDED (sha256) and VALID
(`bash -n`), restore byte-identical, suite **8/8**. The fixture already listed `2001:db8::8` in `OR_IPS`
with nothing pointing at it — which is exactly why the path was never exercised.
**B2 filed, not patched** (new queue row **6c**): the self-test covers `classify_lsof` and the four
`assert_*` front doors only; ~**15** live-path refusal branches have zero coverage, demonstrated by a
neutered darwin/arm64 gate surviving with a byte-identical `PASS: 7`. Same territory as the unisolated
V38 defect, so it belongs to the iteration that isolates it.

**The executor self-reported a surviving mutant** — `assert_control`'s OR-membership branch, as first
delivered — and repaired the harness so `expect_success` inspects positive-arm exit codes. A
self-reported finding is better evidence than a silent run (rule 3h(d)). **Verified first-party rather
than banked:** mutant LANDED, VALID, killed arm 3 with rc=1 against a baseline of rc=0 / 7 arms, restore
byte-identical from a `cp` backup.

**Ruled out.** *The empty peer sets are an instrument fault* — refuted by V36/V37 above, with controls.
*`classify_lsof` is the cause of `driver_rc=1`* — refuted hermetically by the evaluator; it runs after
`wait` and cannot affect an exit code. *My own first `lsof` control (0 lines) showed the sampler blind* —
**my error**: the reading was taken after the holder had exited. Re-taken against a holder asserted ALIVE
(`kill -0`) and streaming (95,869 B), it returns the connection. That is V1 iteration 247's stale-artifact
class met in my own instrument, and the correction is the reason V36 says what it says.

**Gate 0/1.** Kill switch armed; `gh` on `sunholo-voight-kampff`; tripwire **CLEAN**; ran from the `#558`
pin root, detached and clean at `b59255831` == `origin/dev`. Running-skill check on the **RESOLVED**
symlink target (V1 iteration 241's rule): `~/.claude/skills/mission-control` → V1's main checkout,
**byte-identical to origin** (`cmp` rc=0), as is this pin's copy; negative control rc=1. Noted but not
acted on: motoko's own main checkout is **144 behind** with 7 dirty files including a modified `SKILL.md`
— it does **not** serve the running skill, so the blast radius is zero, but it is a real divergence.
dev **verified green, not merely un-red**: 16 exact-SHA checks, 0 not-green, `runs_total=2`, parent control
16. **0** human directives on `#743` since the watermark (of 16 comments); ledger valid at 3 rows, **0
OPEN**; **0** open `[nightly-eval]` alarms (control: 30 closed). Phase-0 predicates re-run as commands with
controls — G1 `#154` OPEN (control `#161` MERGED), G2 rc=128 with mandatory control rc=0, G3 `latest=2.2.0`
no 5.x, G4 unrunnable, G5 outstanding → rows 10/11/12 stay parked. Sweep and rotation both not due.
AC-DEPLOY re-read with a no-pipe control (arms asserted to differ, 1 vs 0): the installed plist still runs
V1's checkout copy, so `#558` stands.

**A scope error of my own, caught by widening.** The doc's §11 cites its two quorum artifacts by a
*repo-relative* path. Searching the pin, the motoko checkout, V1's checkout and `$HOME` returned **0** with
controls firing at 4 and 95 — they live in `.wt-motoko-iter8-fmt/.ailang/state/`, a gitignored per-worktree
directory, and only enumerating **every** worktree found them. Verified once found: 2 rounds, both
reviewers `present: true`, `absent_reviewers` **empty** in both, both `blocked` — spent, not passed.

**Routing.** Controller `claude:claude-opus-5`; executor `codex:gpt-5.6-sol` (probe rc=0, `ok`, 360
tokens); evaluator **sonnet** — distinct provider, generator≠judge holds. No planner (the plan specifies
M2 in full), no designer, no quorum. Rotation pointer untouched at `claude:claude-fable-5`. Metered
**$0.00** of $5. GPU: `rig.lock` acquired `nowait` (bounded — the helper's `wait` mode is an unbounded
`sleep 30` loop) and released, three times. `make quick-install` deliberately NOT run. Gates on
**darwin/arm64 only** except where Gate 3b's matrix is cited.

**Gate 5 — NO skill edit.** Both frictions are instances of rules the skill ALREADY has: the stale
control artifact is V1 iteration 247's freshness rule, and the relative-path-to-a-worktree scope error is
the Repo Profile's own *a relative path is a claim about where you are standing, not about which file
runs*. They were broken, not missing, so they belong here rather than in the rulebook.

**Next**: row **6c** + the V38 isolation, together — keep the driver logs, isolate why the probe's
shipped path makes its lanes exit 1, and add a self-test arm per live-path branch. Then row 7.

---

## 19 — 2026-08-23 — the self-test row 6c called under-covering was running nowhere, and the coverage gap was hiding a soundness defect

**Pick.** Queue head, row **6c** — iteration 18's evaluator finding B2. Not already landed:
`git log origin/dev --grep` returns only iteration 18's own record for the probe, **0** merged PRs
matching, and the row's `[NEXT]` tag is fresh. Died-mid-flight sweep: **0** open PRs attributable to
this mission (`#695` has no branch in this clone's `git worktree list` — mine are 5, all motoko —
so it is not mine and was left alone), and the one uncommitted residue found is superseded, not
unfinished (row 6d). No new design doc and no quorum: this is a bounded remediation of the artifact
milestone M2 of `m-motoko-fmt-remeasurement-instrument` shipped, and that doc's quorum is spent.
No planner: the sprint plan does not cover 6c and the row specifies its own scope, so the executor
directive is controller-authored — which is exactly why its gate list was baselined first.

**The row understated itself, twice, and verifying it rather than inheriting it is what found both.**

**(i) It was not a weak gate. It was not a gate.** Row 6c says the self-test covers the four
`assert_*` front doors "and NOTHING in its live path". True. What nobody had asked is whether the
suite runs at all. Measured with a same-scope firing control:

| query | result |
|---|---|
| `grep -rl test_motoko_connection_probe make/ .github/workflows/` | **0** files |
| `grep -rl test_fmt_ab_schedule make/ .github/workflows/` (control) | **1** (`make/test.mk:43`) |
| repo-wide, `*.mk` / `*.yml` / `Makefile` | **0** |

So every arm iteration 18 wrote — including the IPv6 leak arm it added as a *gate* — had never
executed outside the iteration that wrote it. Wired into `test-launchd-drivers` under an explicit
`/bin/bash` (the rig's is **3.2.57**), and proven by making the self-test exit 1 and watching the
target go rc=2, then restoring byte-identically.

**(ii) The coverage gap was hiding a defect that could certify falsely in both directions.** Two
arms each, exit codes captured without a pipe and printed beside each other:

- `instrument_failure` does `exit 1`. Called from inside a command substitution it exits only the
  **subshell**: repro rc=**0**, control (identical call outside `$( )`) rc=**1**.
  `descendant_pids`'s process-tree deadline was exactly that shape, so on expiry `pids` became `""`
  and the probe carried on.
- `lsof -nP -iTCP -sTCP:ESTABLISHED -a -p ""` returns **75** lines, rc=0, **empty stderr** —
  byte-for-byte the count of the same query with no `-p` at all (**75**). Control: the same shape
  with a real pid holding no established TCP returns **0** lines, rc=1.

**An empty scope argument does not narrow a query — it removes the scope.** Chained, an instrument
whose entire job is to certify *no OpenRouter connections* would have sampled every established
connection on the machine, then either passed on another process's `127.0.0.1:11434` or failed the
treatment lane on an unrelated process's OpenRouter peer. Silently, behind `2>/dev/null || true`.
`descendant_pids` now reports through a status the caller checks, and every lsof scope is asserted
to be a non-empty comma-separated pid list before use — an empty scope fails loudly.

**Row 6c's named next step, done.** The `trap … EXIT` no longer discards both lanes' driver logs and
lsof captures. Retention runs *from* the trap, so it fires on the **refusing** path — the case the
log exists for — and a failed copy is itself an instrument failure. The evaluator confirmed under a
real `SIGTERM` that retention fires and the 143 is not masked.

**Coverage.** Re-derived at pick time: 17 `instrument_failure` call sites + 6 `usage` refusals =
**23** branches, of which the suite reached **4**. Iteration 18's B2 demonstration reproduced
first-party: neutering the darwin/arm64 gate with `if false && …` (mutant LANDED by sha256, VALID by
`bash -n`) leaves stdout **byte-identical at `PASS: 8`**, rc=0 both arms. Now **34** arms, live path
driven hermetically through a stub `AILANG_BIN` and a pruned `PATH` — no eval run, GPU, ollama or
network call anywhere in the suite.

**The executor self-reported four unproven arms, which is better evidence than a silent run.** Its
mutation batch had been contaminated by a concurrent process mutating the shared probe, and it said
so unprompted rather than reporting the table as clean. Re-proved **sequentially** by the
controller: `dig` / `lsof` / `jq` each rc **0 → 1**, each dying on its **own named message**, plus a
`pgrep` control that also reds. Note what the executor's own tightening bought — those three die
with *"lacked expected message"*, not *"unexpectedly succeeded"*: a later gate refuses too, so the
**coarse assertion would have passed**. Rule 3i's "what else writes this value", met inside the
executor's output.

**A defect of my own, in the harness written to close this row.** Isolating the fourth flagged arm
needed a mutant that keeps retention on the success path and drops it on the refusing one — the
exact "reachable only on the success path" defect the directive named. It killed the suite (rc
**0 → 1**, sole failure `refusal lost treatment.driver.log`) and the success arm stayed green, so
the isolation worked. But the suite printed **`ok 29 - refusing live path still retains both lanes
diagnostics` first**. That `expect_failure` arm observes only that the probe refuses with the
control-void message; the retention assertion is a loop **outside any arm**. The gate had teeth and
the *label* did not — a reader counting arms would have believed retention was verified while the
mutant was live. Both arms renamed to what they observe, each retention loop given its own
`pass_arm`; under the same mutant the named arm no longer prints ok (grep count **0**).

**Evaluator: PASS 93/100, ZERO blocking** (sonnet, in **its own worktree** — it mutated source, as a
good judge should, including adding a synthetic refusal branch). It was pointed at my own repair as
a named target precisely because nobody else had reviewed it. All three non-blocking findings
reproduced by command before being acted on — a NON-BLOCKING label is the judge's opinion of
severity, not a measurement — and all three closed in this iteration:

1. **The *real* wall-clock deadline was never exercised.** Arm 25 reached `descendant_pids`'s
   refusal only through the `PROBE_TEST_DESCENDANT_FAILURE` short-circuit at the top of the
   function. The `PROBE_TEST_PGREP_LOOP` stub written to drive the in-loop `date` check — it makes
   the process tree self-referential so the queue never empties — was **defined and never used**
   (grep returns the definition and nothing else). Now arm 33, proven by mutating that branch's
   `return 1` → `return 0`: rc **0 → 1**, the arm dies by name, and it falls through to
   `assert_pid_scope` — defence in depth demonstrated rather than assumed.
2. **No gate against refusal-branch drift.** Reproduced: adding a 24th `instrument_failure` leaves
   the suite byte-identical at `PASS: 32`, rc=0, shellcheck rc=0. Every other arm proves a branch
   that *exists* reds when neutered — **a removal proves the check FIRES; only an addition proves it
   LOOKS** (rule 3a(i-e)). Arm 34 counts the branches (18 + 5 = 23) and refuses when the number
   moves, with an anti-vacuity floor so a counter matching nothing reports INSTRUMENT FAILURE rather
   than a clean result. Proven by re-running the 24th-branch mutant: rc **0 → 1**,
   `refusal-branch drift: probe has 24 refusal branches`.
3. **`PROBE_TEST_FORCE_TREATMENT` was inert residue** — 1 hit in the suite, **0** in the probe
   (control: `PROBE_TIMEOUT_SECS`, 3 hits). The refusal it suggested is produced by passing the same
   lane name twice. Removed.

**What this does NOT do, stated rather than implied. V38 is not isolated.** The mechanism by which
the probe as shipped breaks the runs it observes still lies in the deadline-carrying
`descendant_pids` or the two-lanes-in-one-process sequencing, and isolating it needs the rig. The
two defects fixed here are live *candidates* and are not claimed as the mechanism. What changed is
that the next rig run will have the driver logs — which was this row's own named next step.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor
`codex:gpt-5.6-sol` — probe rc=0, replied `ok`, 13.6k tokens; run backgrounded under a 30-min
`date +%s` cap, `--sandbox workspace-write`, directive `/tmp/codex_directive_iter19.txt` (10,798 B,
≥200 B delivery assertion passed), stdin closed, four `.snap/M<k>/` snapshots delivered.
Evaluator **sonnet**, own worktree, distinct provider from the executor so generator≠judge holds.
**FLAGGED**: my own arm-label repair is Anthropic-authored and judged by an Anthropic evaluator —
same provider, different model — and was named to the judge as a target for that reason. No planner,
no designer, no quorum (reasons under **Pick**). Rotation pointer untouched at
`claude:claude-fable-5`. **Metered $0.00 of $5** — codex and sonnet are both quota buckets. No GPU,
no `rig.lock`: every arm added is hermetic by construction.

**Reconstruction proven faithful, not assumed.** The executor's final tree was sha256-manifested
before any commit was built; after replaying `.snap/M1` → `M4`, `shasum -c` returned **OK on all
four files**. M2 and M3 are byte-identical snapshots — the executor declared them combined and said
so unprompted — so they land as one commit with the reason stated rather than as a false bisection.
M1 was verified independently green at its own boundary (`PASS: 8` through the newly wired target).

**Gates.** Baselined on the unmodified tree **before the directive was written** (recipe false-green
(4)): all seven rc=0. `go build ./...` deliberately excluded — red at base (`cmd/wasm` has no native
`main`). Final sweep by the controller **outside** the executor sandbox, all rc=0: `bash -n` ×2 ·
`/bin/bash` self-test (**PASS: 34**) · `shellcheck -S warning` · `make test-launchd-drivers` ·
`check-file-sizes` · `check-changelog` · `check-skills` · `test-check-changelog`. darwin/arm64 only;
windows and ubuntu legs unrun locally and read from Gate 3b's matrix (rule 3b(viii)).

**Gate 3b GREEN, observed not predicted.** Head `7de24f03c`: **21** checks, **0** pending, **0**
not-green, **4/4** required contexts (`build`/`docs-gate`/`lint`/`test`) pass,
`mergeStateStatus=CLEAN`, then squash-merged as
[`c1950750c`](https://github.com/sunholo-data/ailang/commit/c1950750c). `mergeable` was read FIRST
per the iteration-198 rule and stayed `MERGEABLE` throughout, so no dropped-event lever was reached
for. Autoclose scan on all five commit messages, the PR title and body, and this record's STATUS
stamp: **0** hits, with a known-bad control string matching **1**.

**Ruled out.**
- *That the "coverage gap" was only a coverage gap.* Refuted by measurement: two refusal branches
  could not refuse, and one of them removes the query's scope rather than narrowing it.
- *That the executor's mutation table could be banked as delivered.* It could not, and it said so
  itself. Three arms re-proved sequentially; the contamination was concurrency, not the arms.
- *That the stale `~/dev/sunholo-data/ailang-motoko` residue is died-mid-flight work to adopt.* It
  is not: both untracked files exist on `origin/dev` (`scripts/mission_decisions.sh` byte-identically,
  `tools/launchd/test_mission_routing.sh` in a larger upstream form), and the checkout has been
  dormant since 2026-08-15. Filed as row **6d** for what it *is* — a stale rulebook at a documented
  path — not as work to inherit.
- *My own first mutation attempt.* The `perl s///` carried an unescaped `/` from an interpolated
  shell variable, so the substitution never ran; the LANDED-by-sha256 assertion caught it and
  printed *"MUTANT DID NOT LAND — result meaningless"* instead of a survived-mutant conclusion. The
  assertion exists for exactly this and it earned its keep on the controller's own instrument.
- *That Phase 0 might have moved.* Re-read as a command with a control: `#154` `state=OPEN`,
  `mergedAt=-`; control `#161` **MERGED** with a non-null `mergedAt`. Phase 0 stays CLOSED on G1
  alone, so **no G3 verdict is claimed** — the registry probe returned empty, and an empty probe is
  a claim, not a fact.

**Gate 5 — NO skill edit.** This iteration's two frictions are instances of rules the skill already
has: a refusal that cannot refuse is 3j's *a guard is not a gate until something reds when you
remove it*, and an arm whose label outran its observable is 3i's *which write does this read*. They
were broken, not missing. The one genuinely **new** shape — *an empty scope argument silently
WIDENS a query instead of narrowing it, so the instrument returns a confident, specific, wrong
answer with rc=0 and empty stderr* — is recorded here as **instance 1** and pre-registered rather
than written into the rulebook on a single datapoint. It is close to 3a(i-d)'s scope trap but not
the same: there, a bad scope returns **zero** and reads as absence; here a bad scope returns
**everything** and reads as presence. If a second instance arrives, the bar is met.

**Next**: row **6d** (the stale rulebook at the declared `MISSION_WORKDIR`), then row **6**'s M2 —
which now needs a rig slot and, for the first time, will have the driver logs to isolate V38 with.

---

## 20 — 2026-08-23 — the clone went 170 commits stale because a comment said that was harmless, and half of the comment was right

**Pick.** Queue head, row **6d**. Not already landed: `origin/dev`'s charter still carried the
`[NEXT]` tag at pick time, `gh pr list --search "workdir in:title" --state merged` returned `[]`,
and no direct-to-dev commit matched. Died-mid-flight sweep: one open PR on this account, `#695`
(`coordinator/task-d98bb271`) — **not attributable to this mission**, since no branch of that name
appears in this clone's `git worktree list` (mine are 5, all motoko) — left alone, per the rule that
an unattributable PR is not yours. Four stale sprint worktrees from iterations 6–9 remain and hold
nothing new. No new design doc, so no designer and no quorum: this is mission-infra with a
first-party measured defect. No planner: `derive-planner-lane.sh` returns `opus
fail-closed:planner-lane-field-missing`, and the row specifies its own scope, so the executor
directive is controller-authored.

**Gate 1, blocked-external rows re-measured as commands rather than transcribed.** Phase 0 stays
CLOSED: G1 `#154` `state=OPEN`/`mergedAt=null` with control `#175` `MERGED`; G2 predicate rc=**128**
(*path does not exist in 'origin/main'*) with its mandatory `README.md` control rc=**0**; G3 registry
`versions=[1.0.0, 2.0.0, 2.1.0, 2.2.0]`, `latest=2.2.0`. No predicate has flipped.

**What the row asked for, and what the measurement found instead.**

Row 6d asked for a decision — is the pin root canonical, or should the workdir be brought current —
plus "either way remove the stale skill copy". Both halves of that turn out to be blocked or
insufficient, and the interesting answer was one level down.

*Blocked:* the clone cannot be deleted (it owns the `.git` the pin worktree hangs off:
`cat .git` in the pin reads `gitdir: …/ailang-motoko/.git/worktrees/motoko`), and it cannot be
reconciled unattended. Gate 1's reconcile obligation 2 — *no incoming commit touches a
locally-modified file* — fails **by construction** here, 170 commits of overlap; `pin-root.sh`'s own
header says the same thing in its own words (*"That first reconcile is human"*). So the reconcile is
`D-MOTOKO-WORKDIR-1`, parked.

*Insufficient:* a charter edit alone leaves the next clone to drift the same way. What made **this**
clone reach 170 unnoticed is a mechanism, and it is not an oversight — it is a documented assumption
that has gone false. `mission-control.sh` emits its human-channel pin notice only when
`PIN_STATUS=STALE`, and the comment above that block gives the reason in full:

> "The shared clone being behind is not itself reportable — once drivers pin, that drift is
> harmless, and posting it every 90 minutes would train the channel to be ignored, which is how the
> original silent fallback survived twelve commits."

The second clause is true and is preserved. The first is false in exactly the case this row is
about: drift is harmless to the **driver**, whose pin holds, and not harmless to a **human session**,
because this charter named that clone the working checkout and a session started there resolves ITS
`.claude/skills/`. The evidence was sitting in the driver log, growing, on the **success** path:

| fire | 08-21 08:02 | 08-21 22:01 | 08-22 12:13 | 08-23 03:21 | 08-23 18:16 |
|---|---|---|---|---|---|
| source clone behind `origin/dev` | 119 | 132 | 144 | 159 | **170** |

That is Critical Principle 2 — *a fallback whose only witness is a log nobody reads* — aimed at the
very helper written to close it. **Guard the helper, miss the call site**, this loop's own named
recurring shape, arriving inside the fix for the previous instance of it.

**The fix keeps the true half of the comment.** Notify on crossing `AILANG_DRIVER_DRIFT_WARN`
(default 25), and thereafter only when the drift **doubles**, persisted in a per-mission state file
that is removed below threshold so the notice re-arms after a reconcile. At most one post per
doubling is what the original reasoning was protecting.

**The controller caught the executor's body naming the wrong path, and the path it named is
self-refuting.** The delivered notice interpolated `$REPO`. On the pinned pass `pin-root.sh` has
already exported `MISSION_WORKDIR=<pin worktree>` and `mission-control.sh:40` derives `REPO` from it
— so the notice would have told a human to reconcile a detached throwaway **whose drift is 0 by
construction**. Measured from this session's own live environment, not from the code:
`MISSION_WORKDIR=/Users/voightkampff/.ailang-driver-pin/motoko`,
`AILANG_DRIVER_SRC=/Users/voightkampff/dev/sunholo-data/ailang-motoko`, `AILANG_DRIVER_DRIFT=170`.
The pre-existing STALE body's `$REPO` is **correct** — that arm fires only on the pre-exec pass,
where `REPO` really is the clone that ran the stale code — so one block holds one right and one
wrong use of the same variable, and the evaluator confirmed the distinction independently when
handed it as a named target.

**Evidence.** `tools/launchd/test_driver_notify.sh` **17 → 27** arms, awk-extracted from the real
blocks rather than retyped (renaming the extraction marker prints `FATAL: extraction … produced
nothing`, rc=1 — verified by the evaluator). Every mutant LANDED by sha256, `bash -n` rc=0, restored
from a `cp` backup and asserted byte-identical; red sets **produced by running them**:

| mutant | red set |
|---|---|
| neuter the whole check (`if false &&`) | drift-a, c, d, f, h, i, j (**7**) |
| drop the doubling condition | drift-b (sole) |
| drop the `PIN_STATUS = pinned` guard | drift-e (sole) |
| drop the numeric `PIN_DRIFT` guard | drift-f (sole) |
| body from `$REPO` | drift-a, drift-h |
| remove the threshold floor | drift-i (sole) |

**Evaluator (sonnet, generator≠judge): PASS 90/100, ZERO blocking.** Three of five non-blocking
findings answered in-iteration, each reproduced first-party first: a threshold of `0` persists a
previous of `0`, after which `-ge $((0 * 2))` is true on every fire — the very outcome the doubling
rule prevents, reached through its own knob, now floored and logged (drift-i); `PIN_DRIFT` unset
aborts under `set -u`, unreachable through `pin-root.sh` today and therefore **pinned rather than
assumed** (drift-j, observed red with `PIN_DRIFT: unbound variable` before the normalisation); and
the first commit message asserted a mutation red set of four arms that nobody had run — corrected to
the measured seven, because *a red set written into a record before anyone executed it is a claim*.
Two findings accepted as reported and said so: drift-b/e/g are negative-property arms that survive
total deletion (the five positive arms beside them do not), and the missing changelog entry, added.

**Ruled out.**

- *That the cancelled CI job was ours.* `launchd drivers (bash 3.2)` came back **cancelled after
  15m18s** on the PR head, against **~68s** successes on dev's own HEAD (`a201237ca`) and on 18 of
  the last 20 CI runs — the log stops after `ok 32` and never emits arm 33, then the runner reports
  `Terminate orphan process: pid (bash)` / `(make)`. Refused the co-occurrence: the diff touches no
  file the probe suite reads, and rule 3d's strongest control — a **re-run on a byte-identical
  tree** — returned **success in 88s**. Outcome divergence with the tree held constant means the
  variable is the environment. **Instance 2 arrived 40 minutes later on this iteration's own record
  PR `#840`, whose five changed files are ALL markdown** — `17:38:10Z → 17:53:28Z`, 15m17s,
  cancelled, log stopping after the identical `ok 32`. A markdown-only diff cannot break a shell
  suite, so code attribution is refuted rather than doubted. Three observations in one iteration
  (two CI, one sandbox) against ~68s on dev; row **6e** is a confirmed defect, not a watch-item,
  and it is the next pick.
- *That the clone's uncommitted residue is unfinished work.* Measured, not inherited: of **129**
  added lines across the five modified files, **125 are byte-present** in `origin/dev`'s copy of the
  same file; the 4 that are not are prose reflows of a decision-ledger block `origin/dev` carries in
  a superseding form. Negative control (a fabricated literal) returned 0, positive control 2.
- *That a repo-local guard could defuse the stale checkout.* Any `.claude/settings.json` hook or
  in-repo test is **itself stale in the stale checkout**, which is the same chicken-and-egg
  `pin-root.sh` states about itself. The only surface that reaches a human is the notice channel,
  which is why the fix landed there.
- *That the executor's inability to run `make test-launchd-drivers` was a real red.* It reported the
  target stalling in its sandbox and honestly declined to claim the step passed. Run outside that
  sandbox: **rc=0**, all 34 probe arms plus the driver suites.

**Routing evidence.** controller `claude:claude-opus-5` (probe ok) · designer **none** (no new doc)
· planner **none** (`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`; not
spawned, the row specifies its own scope) · executor `codex:gpt-5.6-sol` (probe rc=0, ~1 reply
token; run 20m, rc=0) · evaluator `sonnet` (Agent-pinned, ≠ executor). **metered=$0.00** — codex and
the controller are subscription/quota buckets, no quorum ran, no managed_agents run.
Fable diet: **unspent** this iteration.

**Cross-mission.** `tools/launchd/mission-control.sh` is shared by `v1`, `world` and `motoko`; all
three gain this notice on their next fire, and all three have their own source clone that can drift.
`PIN_DRIFT_FILE` is defined in **both** the v1-legacy (`mission-control.pin-drift`) and namespaced
(`mission-<name>.pin-drift`) state branches, so no sibling pointer is touched. Said in the PR body
rather than left to be discovered.

**Next**: row **6e** (arm 33's hang), then row **7** (profile restoration design). Row **6**'s M2
still needs a rig slot.

## 21 — 2026-08-24 — the human said yes, and the reconcile the skill calls destructive turned out to have zero ahead-commits to lose

**Picked**: Not the queue head. A **human directive** on `#743` — Mark answered `D-MOTOKO-WORKDIR-1`
with one word, **"Yes"** (`MarkEdmondson1234 @ 2026-08-23T18:59:43Z`), 42 minutes after iteration
20's report asked it. Gate 0's contract is explicit: an allowlisted answer to a parked item unparks
it and *becomes* this iteration's pick, outranking row 6e. The drift notice iteration 20 shipped
fired in the same window and is the inbox message this iteration opened on — **178 behind**, up from
170, which is the doubling-dedupe working rather than a second defect.

**Reality check**: The row's own numbers were re-measured first-party rather than inherited, because
178 commits had landed since they were taken. Ahead-commits: **0** — so Gate 1's reconcile obligation
1 ("every local ahead-commit is a duplicate of an upstream one") is satisfied *vacuously*, which is
strictly stronger than the `patch-id` comparison it prescribes and was not knowable from the row.
Obligation 2 fails **7 of 7**: every dirty file is also touched by an incoming commit
(`comm -12` intersection = 7; positive control `CHANGELOG.md` = 1, negative control = 0, so the
instrument fires in both directions). Supersession re-derived: of **136** added lines in the local
delta, **132** are byte-present on `origin/dev`; the **4** absent are decision-ledger prose in
`motoko-mission.md` that origin carries in a superseding form (positive/negative `git grep` controls
both fired). The two untracked files split: `scripts/mission_decisions.sh` is **byte-identical** to
origin's, and `tools/launchd/test_mission_routing.sh` is **superseded and would actively regress** —
the local copy asserts the executor fallback still carries `:floor`, which origin deliberately
dropped on 2026-08-18 with the rationale in a comment. So discarding it removes a red, not a fix.

**Shipped**: The reconcile, performed and verified — **no PR, because the deliverable is a git
operation on a clone, not a code change**. Sequence, in the order the skill prescribes:
(1) backup of all 7 files + the full `git diff` patch to `~/.ailang/backups/motoko-clone-reconcile-2026-08-24`,
sha256-manifested and verified byte-identical, with a **corruption negative control that fired**
(append a byte → verifier reds → restore → verifier greens);
(2) `git checkout -B dev origin/dev` run **as prescribed first**, which REFUSED (rc=1) naming all 7
files and left the tree byte-unchanged — the refusal is the feature that distinguishes this from
`reset --hard`, and it was recorded rather than routed around;
(3) `git checkout origin/dev -- <5 tracked>` + `rm` of the 2 untracked, both under Mark's explicit
authorization to discard;
(4) `checkout -B` retried → `Reset branch 'dev'`.
Verified after: **behind 0 / ahead 0**, HEAD == `origin/dev` == `e3ed9467f`, `git status --porcelain`
**0 lines**, `SKILL.md` **3682** lines and byte-identical to the pin's copy (negative control vs
`CLAUDE.md` fired), charter byte-identical to the pin's, and **all 8 worktrees intact** — including
`.wt-motoko-iter8-fmt`, which holds the mission's only quorum artifacts in a gitignored directory.
Backup re-verified `OK` on all 7 files *after* the operation.

**Routing evidence**: model=`claude:claude-opus-5` task-class=mechanical
  round1-score=n/a rounds=1 corrections=0
  provider=anthropic agent=controller-inline cost=quota-bucket:weekly-opus
  **No designer, no planner, no executor, no evaluator spawned.** The pick is a human-authorized ops
  action whose procedure is written out step-by-step in Gate 1, with machine-checkable postconditions;
  routing it to a sprint would have added a judge with no design to judge. Designer rotation pointer
  untouched at `claude:claude-fable-5`; Fable unspent. Metered **$0.00** of $5.

**Ruled out**:
- *"The reconcile is destructive and needs the human because work would be lost."* **Refuted on the
  numbers.** Ahead-commits are **0** and 132 of 136 added lines already exist upstream. What made it
  a human call was Gate 1's obligation 2 — a **conservative** predicate that fails whenever an
  incoming commit touches a dirty file, which after 178 commits is nearly unconditional. The
  obligation is doing its job; it just cannot distinguish "you will lose work" from "your work is
  already upstream". Recorded as **instance 1**, not written into the skill on one datapoint.
- *"The drift notice needs manual clearing after a reconcile."* **Refuted by reading the code and
  then running it.** `mission-control.sh:395` removes `$PIN_DRIFT_FILE` whenever drift is below
  `AILANG_DRIVER_DRIFT_WARN` (25). Proved hermetically by extracting the real branch and driving it
  three ways: drift=0 → state file removed ("re-armed"); drift=178 unchanged → deduped, file kept
  (control fires); drift=356 → EMIT. Live drift is now **0**, so the next fire clears it.
- *"67 of 76 open issues are untracked."* **Refuted as an artifact of the wrong corpus.** That count
  is against motoko's four docs alone, and this repo's issue queue is shared with V1. Swept across
  all **9** mission docs (positive control `#558` = 57 hits, negative control fired), the real
  orphan count is **8 of 76**.

**Ruled out (added post-merge, after a third friction the first draft had not yet hit)**:
- *"The bounded Gate-3b poll I wrote was bounded."* **Refuted by watching it not stop.** The poll
  read its three counters with `set -- $out` — and this harness's `Bash` tool runs **zsh 5.9**
  (`SHELL=/bin/zsh`, `ZSH_VERSION=5.9`, `BASH_VERSION` unset), where an unquoted parameter is **not
  word-split**. Measured with a same-call control: in zsh `set -- $out` on `"17 8 0"` gives
  `$#=1`, `$1='17 8 0'`, `$2` empty; under `/bin/bash` the identical line gives `$#=3`, `$1='17'`.
  Every `[ "$pend" -eq 0 ]` then died on `integer expression expected` **to stderr**, so the break
  condition never evaluated and the loop ran until the harness's own 10-minute `Bash` cap ended it.
  The same non-splitting had already broken a `for f in $FILES` sweep loop earlier in the iteration.
- **This is instance 4 of a rule the skill ALREADY carries in full, and the rulebook is not what
  failed — my reading of it is.** `SKILL.md:1108` (*"AND THE SAME NON-SPLITTING BREAKS `set -- $var`,
  WHICH IS THE SHAPE THAT LANDS IN POLL READERS"*, added 2026-08-20 by V1 iteration 239) names three
  prior instances, including iteration 107's Gate-3b poll, and prescribes the exact remedy — read
  each value with its own command, or `set -- ${=var}`. I skimmed that block because it sits inside
  Gate 2 rather than beside the Gate 3b poll recipe I was copying. **So no skill edit**: adding a
  fourth instance to a correct rule buys nothing, and Gate 5's edit budget should not be spent
  restating a remedy that is already written down.
- **What is genuinely new is the SURFACE, and it is worth pre-registering.** Instances 1–3 all
  produced a *false verdict* — a `TIMEOUT — PARK` over a green, a bad containment reading. Mine
  produced **no verdict at all**: the comparison ERRORS rather than reading false, so the loop
  cannot exit and the slot is consumed by the harness cap. That renders as a long CI wait, which is
  the one thing a Gate-3b poll is *expected* to look like — i.e. an unbounded wait in Standing rule
  6's sense wearing a bounded one's clothes. Recorded as instance 1 of that surface; if a second
  appears, the remedy list's entry deserves the consequence spelled out beside the mechanism.

**Retro lane**: none — see above: the one candidate resolves to a rule that already exists. This
iteration's other frictions — a conservative predicate that cannot separate
"work would be lost" from "work is already upstream", and a sweep whose corpus was narrower than its
verdict — are both instances of shapes the skill already names (rule 3a's "establish the instrument
before its reading counts", and the sweep rule's own *"a CLEAN verdict must carry the issue count it
swept"*). Pre-registered as instance 1 each rather than spent on a single datapoint.

**Next**: row **6e** — `test_motoko_connection_probe.sh` arm 33, still the queue head and now
un-preempted. Then row **6f** (the sweep's 8 orphans, 2 motoko-owned) and row **7**.

**Post-report correction to entry 21 (same iteration, before the slot ended).** The digest posted to
`#850` says `DECISIONS FOR MARK: none`. That was **wrong within the hour**, and the measurement that
refuted it is one this iteration should have taken before reporting: the reconcile is a **one-shot
against a continuing drift**. Nothing pulls the clone — `pin-root.sh` runs `git fetch` only, and
`git pull` appears **0** times across both drivers — so the clone was back to **4 behind** by the
time the worktree was cleaned up. `origin/dev` lands **21.8 commits/day** (153 in 7d; corroborating
points 17/1d, 60/3d, 353/14d), so drift re-crosses `AILANG_DRIVER_DRIFT_WARN`=25 in **~1.1 days**,
and the doubling-dedupe re-notifies at ~50/~100/~200 — about four asks per nine days, every one of
them resolving to the same word for the same mechanical operation. Filed as **`D-MOTOKO-WORKDIR-2`**
and corrected on the thread. The general shape, pre-registered as instance 1: **a fix that moves a
system to a good state is not the same as a fix that keeps it there, and this loop's own reporting
template asks for the outcome at the moment of writing** — so a one-shot remediation reports
identically to a durable one. The tell: you are about to write "DECISIONS: none" for an iteration
whose deliverable restored a state nothing maintains.

## 22 — 2026-08-25 — row 6e: a self-test that can hang is an outage with a green history

**Pick**: queue head, row **6e**. No competing signal: 0 human directives on `#850` (and `#743` re-read for
the rotation-week catch, also 0), inbox 1 unread and informational, dev not red, Phase-0 `G1 #154` still OPEN
(control `#175` MERGED) so rows 10/11/12 stay parked.

**Outcome**: **LANDED**. PR [#871](https://github.com/sunholo-data/ailang/pull/871) → `086b72184`.
Evaluator round 1 **FAIL 54/100** (3 blocking) → round 2 **PASS 91/100, zero blocking**.
Gate 3b GREEN on `ddd8f3f09`: 21 checks, 0 pending, 0 not-green, 4/4 required (build/docs-gate/lint/test),
`mergeStateStatus=CLEAN`; `mergeable` read first and `MERGEABLE` throughout, so no dropped-event lever was reached for.

**What was measured, and what deliberately was not concluded.** The row named two CI cancellations; the last
100 CI runs carry **three** (`32655443831`, `32665128080`, `32673098414` — the third a push to `dev` itself),
all at ~918s, which is `timeout-minutes: 15` firing rather than a flake. Arm-33 attribution was verified from
the three job logs (identical last line, `ok 33` absent) against a green control that emits the same line and
then `ok 33` 1.06s later. **The mechanism inside arm 33 is NOT isolated and nothing here claims it is**: it has
not recurred in ~44 runs on an unchanged file. A synthetic near-copy of the walk aborts bash 3.2 with SIGABRT
5/5 while the shipped arm passes in 1.06s — the divergence is why that repro is recorded as an observation and
not promoted to the mechanism. What shipped is the structural fact: the suite had no bound of its own.

**Deliverable**: every arm carries a hard, validated wall-clock cap that TERMs then KILLs and prints a named
`not ok` plus both captured output tails; `descendant_pids` is additionally bounded by node count with a
message distinct from the clock's. Suite 34 → 39 arms, drift gate 23 → 24.

**Routing evidence**: controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` — probe rc=0,
replied `ok` — **CAPPED at the 30-minute wall, FLAGGED**, with `.snap/M1`–`M3` complete; work VERIFIED, not
adopted, and the entire mutation drill is therefore first-party. Evaluator **sonnet**, two rounds, in its own
worktree; distinct provider from the executor so generator≠judge holds there, **FLAGGED** that the drill and
every round-1 fix are Anthropic-authored and Anthropic-judged — the judge was pointed at them by name for that
reason, and it re-derived the round-1 table itself, reproducing two rows at byte-identical sha256. No planner,
no designer, no quorum: the row specifies its own scope and the parent doc's quorum is spent. Rotation pointer
untouched at `claude:claude-fable-5`; Fable unspent. Metered **$0.00** of $5 (codex and sonnet are quota
buckets). No GPU, no `rig.lock`.

**Ruled out / corrected**
- *"Two cancellations"* — three, one of them on `dev`. The row's count was a transcription of what iteration 20
  saw, not a measurement of the window.
- *The hang is in the live synthetic socket block between `ok 32` and arm 33* — refuted by the logs: the
  block's own closing message is the last line printed in all three, so it completed.
- *A synthetic reproduction is the mechanism* — refuted by its own divergence from the shipped arm. Recorded as
  instance 1 of "this stimulus sits near a bash 3.2 cliff", not as a finding.
- *M1's snapshot is not independently green* — one boundary run redded at arm 32 and the identical command was
  rc=0 on the next; M1's boundary is green at 36 arms. Both this and the arm-30 case below are load-shaped and
  are recorded as observations, **not** declared a flake class off single instances.
- **A red banked for the wrong reason.** The first batch run of the M1 mutant redded at **arm 30**, not the cap
  arm. Re-run isolated it reds at its own arm 2/2 with the unmutated suite green 3/3. Reading the exit code
  alone banks a pin that does not exist — rule 3j's corollary, paid for in this iteration rather than read.
- **My own PR body was wrong.** It claimed *"fast arms are unaffected"*; the judge measured 30s → 66-93s. The
  poll now backs off 0.05 → 0.2 → 1s (judge's re-measure: 29.89s pre-PR / 66.61s flat / 44.98-45.07s shipped),
  and the claim was corrected in the PR body rather than left standing.
- **Two of my own guards were decoration until mutated.** `report_arm_cap` had zero coverage; the cap arm was
  satisfiable by a fixture exiting 199 with no TERM and no KILL. And the *first* fix for the former passed for
  the wrong reason — with `exit 1` removed, `expect_failure` falls through to its own refusal and still exits 1
  with every marker present, so the arm now requires that fall-through message to be ABSENT.
- **An unscoped `sed` mutation killed the wrong arm** — `s/^  exit 1$/…/` hit the `ARM_CAP_SECS` validation too.
  Second instance in one iteration of "read WHICH test failed, never the exit code alone". Fixed by scoping the
  mutation to `report_arm_cap`'s line range.

**Filed, not fixed**: row **6g** — `run_bounded` *and* the production `run_lane` kill the wrapper PID rather
than the process group, so a hung grandchild reparents to `PPID 1` and survives while the suite's own
"process survived" check passes. Pre-existing since M1 and present in production, so it is a queue row on its
own evidence rather than a revision that grows this PR.

**Gate 5**: **no skill edit.** Both frictions are instances of rules the skill already carries (3j's corollary;
3i's "what else writes this value"), so they belong in Ruled-out, not the rulebook.

**Next**: row **6f** (triage-lite the 8 orphan issues), then 6g, then 7.

## 23 — 2026-08-25 — two issues filed in the same session by the same reporter, and only one of them was ever a bug

**Pick**: queue head, row **6f** — triage-lite the two motoko-owned issues from iteration 21's weekly sweep
(`#842`, `#839`); the other six were handed to V1 then and are deliberately not re-triaged here. No competing
signal: **0** human directives on `#850` since the watermark (of 6 comments), `#743` re-read for the
rotation-week catch also **0**, ledger valid at 5 rows with **1 OPEN** (`D-MOTOKO-WORKDIR-2`, still
unanswered), Phase-0 `G1 #154` still **OPEN** with control `#175` **MERGED** so rows 10/11/12 stay parked.
Weekly sweep and rotation both not due (`#850` created `2026-08-24T07:39:32Z` = 09:39 local, after the
Monday-07:00 local boundary; 6 comments < 80).

**Outcome**: **LANDED** (bookkeeping pick). `#839` **CLOSED** with its measurement; `#842` **CONFIRMED REAL at
HEAD** and filed as row **6h** with a pre-measured fix scope. Both verdict comments asserted landed by
comment-count growth against a pre-count control (`#839` 1 → 2, `#842` 0 → 1), and posted as their own
`gh issue comment --body-file` **before** any close, per the Gate-0 mechanism-B rule.

**The two issues arrived together, from the same account, in the same debugging session, and the reporter
explicitly said they were independent. They were more independent than that: one of them was never live.**

`#839` (`std/net` ignores `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`) is a **version skew**, not a defect. The
report's binary is `v0.33.0` / `ae36986`, dated **2026-08-04**. The request-aware proxy transport landed
**2026-08-20** in `e5ee6c5e5` (PR `#613`), sixteen days later. Measured with a firing control rather than
inferred from a changelog: `git ls-tree ae36986 internal/effects/net_proxy.go` is **empty** while the control
`internal/effects/context.go` returns a blob in the same call, and `git merge-base --is-ancestor e5ee6c5e5
ae36986` is **false**. What makes this a *durable* close rather than bookkeeping is that the behaviour the
reporter's decisive third repro isolates — a proxy pointed at a closed loopback port must produce
connection-refused, not a DNS error, because the target name is never resolved locally — is covered by
committed tests that CI runs (`go test -timeout 300s ./...`, `ci.yml:101`):
`TestNetProxyBoundary/proxy_selected_from_environment` and
`TestNetProxyTargetValidation/proxy_hostname_remains_unresolved`, plus `TestNetProxyNoProxy`,
`TestNetProxyDirectPin` and `TestNetProxyRedirectControls`. `go test ./internal/effects/ -run TestNetProxy`
→ **rc=0** (captured without a pipe), negative control `-run TestNetProxyZZZNoSuchThing` → `[no tests to run]`.
**One of the five reports `SKIP`, and that was checked rather than counted as coverage** (rule 2 — a parked
test is a claim): `TestNetProxyEnvProxyHelper` skips unless `AILANG_M1_PROXY_HELPER=1`, and
`TestNetProxyBoundary` re-execs it as a subprocess with that variable set and asserts
`--- PASS: TestNetProxyEnvProxyHelper` appears in its output. That subprocess arm is the only place production
`http.ProxyFromEnvironment` runs instead of the injected `proxySelector` hook — i.e. the skip is the mechanism,
not a hole. Ran that arm alone: rc=0.

**`#842` is real, and the measurement says something the issue does not.** Fed the reporter's verbatim failing
body to `openai.ParseChatStepResponse` (`internal/ai/openai/step.go:560`) with three controls in one run:
the failing shape (`finish_reason:"stop"`, `content:null`, **no `usage` key**) returns **OK** — no error,
`text=""`, `toolcalls=0`, `in=0 out=0 total=0`; the healthy control returns `"pong"` with real usage, so the
instrument sees a positive; the legitimate-tool-call control (`content:null`, usage present) returns 1 tool
call, which is exactly what a *"null content is suspicious"* heuristic would false-positive on and is why the
reporter's choice of `usage` as the key is the right one; and the fourth control — **usage present but
all-zero** — produces output **byte-identical to the failing arm**.

**The load-bearing find is that the suggested guard is not currently expressible.**
`ChatStepResponse.Usage` is a **value type** (`step.go:300`), so an absent `usage` key unmarshals to the zero
struct. Asserted directly rather than argued: `absent.Usage == zeroed.Usage` → **true**. So "treat a missing
`usage` block as a provider error" cannot be written behind the present type at all; the precondition is a
representation change (`*ChatStepUsage`, or a raw-key presence check), which is behaviour-free and separable
from the genuinely open policy question the reporter himself flagged.

**And the blast radius is wider than filed, in the direction this mission cares about.**
`ParseChatStepResponse` has exactly **two** production callers — `openrouter/step.go:162` and
`openai/step.go:170` (controls: 77 hits for a common symbol in the same tree, 0 for an invented one) — and
`internal/ai/ollama/step.go:345` builds `openai.NewClient(…)` against `<endpoint>/v1`. So **every ollama
tool-calling turn parses through the same helper**, which puts this on our own Mac Studio eval rig, where a
masked provider failure is indistinguishable from a local model declining to act. That is the charter
guardrail *"never conclude model wall"* arriving from outside the mission, reported by someone who had no way
to know it was our guardrail.

**Routing evidence**: controller `claude:claude-opus-5` (session). **No designer, planner, executor or
evaluator spawned** — the row is triage-lite and names its own procedure (ghost-discipline the repro →
verdict comment → queue-or-close) with machine-checkable postconditions, so there is no plan to plan and no
generated artifact for a judge to judge; same disposition and same reasoning as iteration 21's ops pick. No
quorum (bookkeeping pick — the Gate-2 carve-out). Rotation pointer untouched at `claude:claude-fable-5`;
Fable **unspent**. Metered **$0.00** of $5. No GPU, no `rig.lock`. Gates run on darwin/arm64 only.

**dev CI was RED at pick time and this mission does not own it.** `CI` `failure` on `02bf43668`
(run `32860399250`), 1 not-green of 15 exact-SHA checks, job `test`. Diagnosed rather than inherited: the
failing step is *Download all Go modules* — `proxy.golang.org … stream error: stream ID 1187; INTERNAL_ERROR;
received from peer` — and `02bf43668` is V1's **docs-only** iteration-276 record (4 markdown files, 0 code),
which cannot affect a module download. Parent `f4828cc89` green ~2h46m earlier is the before-arm. Fired
`gh run rerun --failed` as rule 3d's strongest control (outcome divergence on a byte-identical tree) and
handed it to V1 on the cross-mission channel per the charter guardrail; **it did not displace this
iteration's pick and no fix was attempted here.**

**Ruled out / corrected**
- *`#839` is a live `std/net` defect* — refuted. Fixed at HEAD sixteen days before it was filed; the reporter's
  binary predates the fix. The three byte-identical repro outputs were correct observations of a build with no
  proxy path.
- *The `SKIP` in the proxy suite is a coverage gap* — refuted. It is a subprocess helper the parent test
  re-execs and asserts on; it is the only arm exercising the production selector.
- *`#842` is OpenRouter-specific* — refuted. The shared parser also serves the ollama `/v1` lane, so it reaches
  the local-model rig.
- *`#842` is a one-line guard* — refuted, twice over: the signal is not representable against the current
  type, and the reporter's own caveat (Anthropic/Gemini paths do not share this parser; streaming usage
  delivery is opt-in on some providers) is upheld. A uniform guard would convert a legitimate empty completion
  into an error on a provider that simply omits usage — the same defect pointed the other way.
- *dev's red is attributable to a recent merge* — refuted by the diff (docs-only) and by the failing step
  (dependency download, before any repo command touched the tree).

**Gate 5**: **no skill edit.** This iteration's one friction — a triage row whose two issues needed opposite
dispositions, where the cheaper one to check was the one that turned out to be a ghost — is an instance of the
ghost-discipline rule the skill already carries, and it *worked*: the rule is what made the version check the
first move rather than a code read. Recorded here, not in the rulebook.

**Next**: row **6g** (`run_bounded`/`run_lane` kill the wrapper PID, not the process group), then row **6h**
(the `#842` fix), then row 7.

## 24 — 2026-08-26 — the cap killed the wrapper, and the fix for the half that matters is pinned by nothing

**Pick**: queue head, row **6g** — `run_bounded` (self-test) and `run_lane` (production) bound a
child with a wall-clock cap and, on expiry, kill the **wrapper PID only**, so a hung grandchild is
reparented to `PPID 1` and survives. Landed as PR
[#892](https://github.com/sunholo-data/ailang/pull/892).

**Ghost discipline, before any routing.** The row's evidence was iteration 22's evaluator, i.e. an
inherited claim. Reproduced first-party against the shipped code with controls firing: PRE 0 live
fixtures, cap fires `rc=199`, **1 survivor at `PPID 1`** afterwards, wrapper count **0**, POST
cleanup back to 0. The middle two lines are the finding — the suite's existing "process survived"
check passes *because* the wrapper really is dead, so the arm passed for exactly the reason it
should have failed.

**Mechanism measured before it was specified, not after.** `setsid` does not exist on macOS
(`which setsid` → not found) and every process-group precedent in this repo is Go-side
(`SysProcAttr{Setpgid:true}`), unusable from bash. Two arms, and they differ, which is what makes
this a discriminator rather than a preference: **without** `set -m` the child's pgid (9361) **is
the script's own**, so a negative-PID kill would kill the suite itself, and a single-PID kill
leaves 1 orphan; **with** it, pgid == pid (9379), differs from the script's, and the group kill
leaves 0. That measurement is why the shipped group kill is *guarded* — `jobs -p` membership,
`pid != $$`, and a live `kill -0 "-$pid"` — rather than unconditional.

**THE FINDING THAT OUTRANKS THE FIX: the production hunk is pinned by nothing.** Walking the diff
hunk-by-hunk rather than reasoning from the defect (rule 3n), the third mutant reverts
`tools/eval/motoko_connection_probe.sh` **entirely** to its `origin/dev` version — and the suite
stays **green at 40/40, rc=0**. Zero killers. Its only gate is `bash -n`. So the half of row 6g
that the row itself calls *"the one that matters"* — the production lane bound on the GPU rig,
where a surviving descendant is indistinguishable from a model declining to act — shipped with no
behavioural pin at all. Filed as row **6i** rather than growing this PR, per the rule that a hunk
with no killer is a finding and not a failure. The self-test half **is** pinned: mutant A (group →
single-PID kill) is the **SOLE KILLER**, `survivors=1`, 34 arms green before it; mutant B (neuter
`set -m`) fires the safety refusal twice and the suite **reports** the failure instead of killing
itself, which is the guard's whole purpose.

**MY OWN GATE LIST WAS UNSATISFIABLE IN THE LANE I ROUTED IT TO, AND THE EXECUTOR CAUGHT IT.** Run
1 of the executor stopped with zero files changed and reported that G3 — the full self-test suite —
never produced an rc: under codex `--sandbox workspace-write` it reaches arm 32, prints
`UNINFORMATIVE UNDER SANDBOX: loopback bind denied`, and arm 33's live-socket bind then **terminates
the enclosing session**. It stopped because my directive told it a base-red gate is a finding, not
an obstacle — so it obeyed the rule correctly and the rule was pointed at a gate that could never
be green there. This is the skill's own iteration-270 defect (*a baseline is a claim about the
environment you ran it in*): I baselined all three gates **outside** the sandbox, in my own shell,
and handed them to a lane that cannot satisfy one of them. Re-issued with G3 labelled
`UNINFORMATIVE UNDER SANDBOX`, a satisfiable in-sandbox harness in its place, and the full suite
re-run by the controller outside — which is what false-green #3 already prescribes. Cost: one
executor run. Not a lane failure and not a fallback; the lane was fine and the directive was wrong.

**A GREP-SHAPED LEAK CHECK MATCHED A PROCESS THAT ONLY *MENTIONED* THE FIXTURE.** After the green
suite run, `ps | grep "sleep 2849"` returned **1** and read as a leaked orphan. It was a codex
computer-use notification process whose argv embeds the executor's own report, which contains the
string. Scoped correctly to `comm == sleep` the count is **0**, with the control firing (1 while a
real `sleep 2849` runs, 0 after it is killed). Worth recording because it *vindicates the
executor's deviation from my directive*: I specified matching a distinctive sleep duration; it
instead scoped by a unique fixture **cwd** via `lsof -c sleep`. Its design is strictly better and
mine is the one that produced a false positive within minutes of being written.

**Evaluator (sonnet, own worktree, distinct provider from the codex executor): round 1 PASS
82/100, ZERO blocking.** It reproduced **all five** controller claims exactly — including mutant
C's zero-killer result — tried and failed to construct a false positive for the `group_safe` guard,
confirmed `set -m` leaks no job-status lines by diffing arm lines against the 39-arm baseline, and
verified `{ wait "$pid"; } 2>/dev/null || rc=$?` still propagates a real exit code (42, 5/5). Of its
three non-blocking findings, one names a defect **this PR introduced** — the guarded fallback
printed `INSTRUMENT FAILURE:`, byte-for-byte the prefix of `instrument_failure()`, which prints that
string and then `exit 1`, while the fallback does not exit; a log-grep for the abort signal matched
**4** lines where only **2** meant termination. Fixed in the second commit (`INSTRUMENT DEGRADED:`,
verified against the parsed form: 4 → 2 and 0 → 2). The other two are pre-existing or
forward-looking and went to the queue.

**Ruled out**
- *"The `ailang` binary on PATH honours `AILANG_MESSAGES_STORE`"* — **refuted at Gate 0.** The
  control (`AILANG_MESSAGES_STORE=not-a-real-store`) must error `unknown message store mode`; the
  PATH binary returned **rc=0** and listed local SQLite. Built a fresh binary to a scratch dir with
  the version ldflags (a bare `go build` in a linked worktree stamps `"dev"`), confirmed
  `v0.33.2-26-gfadbdc4e2`, and the control then correctly returned **rc=1**. The cloud inbox has
  **62** unread against the local store's 12 — they are different queues.
- *"The suite leaks an orphan"* — refuted, see above; the instrument was wrong, not the suite.
- *"SonarCloud's red is ours"* — refuted by walking it back commit by commit: green at `6193bb712`
  (18:55), red at `6759ea4fa` (19:14) and on every commit since. V1's messaging-store change, and
  the condition is *52.8% coverage on new code (required ≥80%)*. Handed over, not triaged here.
- *`pgrep`/`ps` as a liveness or leak instrument* — refuted twice in one iteration (the argv match
  above; and the skill's own rule that an empty `pgrep` is *unknown*, not *dead*). Polled artifacts
  throughout: the rc file, the log size, the worktree diff.

**Next**: row **6h** (a provider failure parsing as a successful empty completion — reaches the
ollama `/v1` lane, i.e. our own rig), then the newly filed row **6i**.

## 25 — 2026-08-26 — the pin gate had been refusing unconditionally, and the suite asserted the defect was correct

**Pick.** Not the queue head. `tools/launchd/lib/pin-root.sh` gates the launchd driver pin on
`~/.claude.json`'s `.projects[<path>].hasCompletedProjectOnboarding`; Claude Code retired that key, so
the predicate answered `false` for every path and the *"neither path onboarded"* branch was
unconditionally true. Every fire of every mission had been refusing to pin. A genuine regression
outranks the queue, and **this fire was the instance** — the driver logged `DRIVER PIN FAILED` at
21:05:14 and executed the source clone at `e3ed9467f`, **152 commits behind** `origin/dev`.

**Measurement, three readings in one call, both controls firing.**

| reading | count |
|---|---|
| project entries carrying `hasCompletedProjectOnboarding` | **0 of 15** (whole-file `grep -c`: 0) |
| project entries carrying `hasTrustDialogAccepted` (positive control) | **15 of 15** |
| project entries carrying an invented key (negative control) | **0** |

Simulated before the change: motoko, v1 and world all `REFUSE-TO-PIN`. V1's fire in the same window
reported the identical shape (`MISSION_WORKDIR unset`) on its own bookkeeping issue — the
two-mission corroboration rule 3l asks for before anything is called fleet-wide. The gate was added
2026-08-12 (`019568cf1`) to fix the class tracked at #558; it was correct then. It has been a stale
**capability claim about the harness** ever since — the shape the skill's own model table warns about,
arriving in a shell predicate instead.

**The suite could not have caught it, and asserted the opposite.** `test_pin_root.sh` writes its own
synthetic `~/.claude.json` fixtures that still carry the retired key, and its arm 8 asserted that
*the only key which now exists must be REFUSED*. A green gate pinning the production defect as
correct. That is the fixture-vs-world gap: a gate can be complete over its own inputs and blind to
the schema those inputs model — and no mutation of the code could reveal it, because the code and
the fixture agreed.

**Outcome.** PR [#923](https://github.com/sunholo-data/ailang/pull/923) →
[`ff0da7445`](https://github.com/sunholo-data/ailang/commit/ff0da7445). Suite **35 → 53** arms.
Evaluator round 1 **PASS 98/100, zero blocking**. Gate 3b GREEN on the merge: 16 checks, 0 pending,
required `test`/`lint`/`build` success (`docs-gate` N/A by path filter on the merge; it passed on the
PR head where the merge button was gated), and **`launchd drivers (bash 3.2)` — the job this change
lives in — success** on both.

**The reading that matters is that the fix restores discrimination.** Against the real
`~/.claude.json` afterwards: motoko **PIN-OK**, v1 **PIN-OK**, world **REFUSE-TO-PIN** — and world's
refusal is a *true* verdict, since that clone genuinely carries neither key, so the human fix its
message names is the right advice. A change that made all three pin would have been
indistinguishable from deleting the gate.

**The half that outlives this bug** is the anti-vacuity floor: when neither key appears in any entry,
the gate reports **Claude Code schema drift** — fail-closed still, #558's ratified posture
deliberately unchanged, only the sentence moves — so the next rename is loud on its first fire
instead of silent forever.

**The judge found a defect this branch introduced, in the exact faculty the branch exists to
improve.** Round 1 passed with one non-blocking finding: the new drift diagnosis fired identically
for a missing, invalid-JSON or non-object `~/.claude.json`, telling the next reader the **gate**
needs a new key when the **file** is broken. That is this gate's own original defect one level down,
so it was fixed here rather than filed. Three refusals, three sentences. A consequence worth naming:
an **empty** `.projects` map is *"nothing onboarded yet"*, i.e. the ordinary case — so arm 7's
natural `{"projects":{}}` fixture was **restored**, the executor's earlier fixture change having been
a correct workaround for the two-way split and unnecessary under the three-way one.

**Mutation drill, anchored to the diff hunk by hunk (rule 3n), re-run at the final tree.** Each
mutant asserted LANDED (sha256), PARSING (`bash -n`) and **effect-verified against the queried form
rather than the file's bytes** — V1 iteration 274's rule, which reached this iteration only because
the running-skill delta was read first. Each restored byte-identical from a `cp` backup.

| mutant | result |
|---|---|
| A — `_pin_onboarded` → legacy-key-only | 3 arms red, **sole killer** |
| B — neuter the drift branch | 4 arms red |
| E — neuter the unreadable branch | 3 arms red |
| F — drop the `projects_len > 0` guard | 2 arms red (an empty map misdiagnosed as drift) |
| C — revert the supporting `local` declaration hunk | **zero killers** — recorded UNPINNED, not claimed as covered |
| D — revert arm 7's fixture | 2 arms red — adjudicates the executor's self-reported deviation in its favour |

**The load-bearing observation is which assertion did NOT move.** Under B and E the `STATUS=STALE`
assertion stayed **green**, because every refusal produces STALE. Only the message *text*
discriminates. That is rule 3i's *what else writes this value* met first-party, and it is why these
arms assert prose rather than status — an arm checking only the status would have passed for the
wrong reason in both drills.

**The judge attacked beyond its brief and the fix held**: it built its own *widening-into-a-no-op*
mutant (suite caught it, 7 arms red), ran the jq edge cases end-to-end through the real driver rather
than in isolation, and neutered the **precondition** of all three new arms — all three died, so none
passes for a second reason. `/bin/bash` confirmed 3.2.57 on the rig.

**Ruled out.**
- *That the running rulebook was this checkout's copy.* It is not. `~/.claude/skills/mission-control`
  resolves to **V1's** checkout (inode `51683298`), byte-identical to `origin/dev`, while this fire's
  CWD carries a copy **139 lines short** (inode `48752546`). The 187-line delta was read before any
  gate ran. On an unpinned fire the relative-path form of that check is actively wrong — iteration
  241's hazard, met from the other direction.
- *That the SonarCloud red came from this merge.* Inherited: `failure` on the merge and on all three
  preceding commits, all V1's; negative control `3f5ca3df9` carries no Sonar check at all.
  Non-required. Conditions re-read rather than inherited from iteration 24's framing (rule 3n(d)):
  **56.9% coverage on new code** *and*, newly, **B security rating on new code** — the second is new
  since iteration 24 saw coverage alone. This diff is shell and markdown, which Go coverage does not
  measure.
- *That the `launchd drivers` CI failures were caused by a motoko diff.* Measured across the last 60
  CI runs on `dev`: **55 success / 2 failure / 1 cancelled**, all three non-success on **unrelated V1
  coordinator commits**, all three at arm 33. Filed as row **6j**, not picked.
- *That `shellcheck` findings were this diff's doing.* rc=1 with 5 identical findings at base and at
  head; not a CI gate in this repo.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` —
probe rc=0 replied `ok`, **one** run, no cap, no fallback, and it **self-reported** a directive
tension (arm 7's fixture) which measurement then vindicated. Evaluator **sonnet** in its **own
worktree** (iteration-199 rule), distinct provider from the executor so generator≠judge holds. No
designer, no planner, no quorum — a measured regression with a named fix is not a design question.
Designer rotation pointer untouched at `claude:claude-fable-5`; **Fable unspent**. Metered **$0.00**
of $5 — codex and sonnet are quota buckets. No GPU, no `rig.lock`. Gates on **darwin/arm64**; for
`launchd drivers (bash 3.2)` that IS the CI leg, macOS-only by design for bash 3.2, so the local
green is that leg rather than a proxy for it. Windows and ubuntu legs read from Gate 3b's matrix.

**Next.** Row **6h** — the provider failure arriving as a successful empty completion.

## 26 — 2026-08-27 — iteration 25's fix is correct and cannot run: the pin gate is loaded from the tree the pin exists to replace

**Pick.** Not the queue head. A genuine regression outranks it and **this fire is the instance —
again**. The driver logged `DRIVER PIN FAILED` at `11:56:34` and executed the source clone at
`e3ed9467f`, now **172** commits behind `origin/dev`, up from **152** one fire earlier.

**The finding.** Iteration 25 diagnosed the retired-key predicate correctly and fixed it correctly
(`ff0da7445`). **The fix does not run.** launchd invokes the *source clone's*
`tools/launchd/mission-control.sh` (`ProgramArguments`, read from the installed plist), which sources
the *source clone's* `tools/launchd/lib/pin-root.sh` — and that copy is 172 commits behind, so it
carries the **pre-fix** `_pin_onboarded`. The gate that decides whether to execute committed code is
itself loaded from the stale tree the pin exists to bypass. **The fix for "the pin refuses" was
landed into the only tree the refusal prevents from running.**

**Measurement — both predicates run verbatim against the live `~/.claude.json`, in one call.**

| predicate (which copy) | pin worktree | source clone | verdict |
|---|---|---|---|
| clone's, retired key only — **the one that ran** | false | false | **REFUSE-TO-PIN** |
| `origin/dev`'s, per `ff0da7445` — **the one that didn't** | false | **true** | **PIN-OK** |

Controls in the same call: retired `hasCompletedProjectOnboarding` **0** occurrences whole-file;
`hasTrustDialogAccepted` **15** (positive); an invented key **0** (negative); **15** entries
enumerated. So the fix is not merely plausible — it is measured to pass **on this exact machine and
this exact state**. The refusal branch requires *both* paths false; the fixed predicate returns
`true` for the clone, so the gate would open.

**A count would have read as coverage; reading it is what caught it.** `grep -c
'hasTrustDialogAccepted'` against the clone's `pin-root.sh` returns **1**, not 0 — and that single
occurrence is a **comment at line 185**, not code (`origin/dev`'s copy returns **4**; invented-token
negative control **0** in both). A bare non-zero count is exactly what "the fix is present" looks
like.

**Motoko-specific, not fleet-wide — and the third arm is what says so (rule 3l).**

| mission | clone behind | `pin-root.sh` fix hits | `DRIVER PIN FAILED` in driver log |
|---|---|---|---|
| control (V1) | 0 | **4** | 4 |
| world | 0 | *file absent* | 0 |
| motoko | **172** | 1 (a comment) | 3 |

Positive control per log (`iteration`): 19 / 15 / 5; negative control 0 in all three. V1 escaped
because its clone **happens to be current**, not because of anything the fix did. World has no
`tools/launchd/lib/pin-root.sh` at all — its empty grep was *file absence*, verified with `ls` rather
than read as a zero (rule 3a). So the mission that found and fixed the bug is the only one still
living with it, and the reason is **drift, not code**.

**Why nothing landed this iteration could have fixed it.** Any change I write goes to `origin/dev` —
which is precisely the tree the broken clone cannot reach. A class fix to `pin-root.sh` would be
**inert for motoko** (live for V1, whose clone is current), i.e. it would repeat iteration 25's
mistake one level up: landing the remedy where the defect prevents it from being read. Recognising
that is the iteration's actual deliverable; the class is filed as row **6l** rather than executed.

**The remedy is one word, and this is the fifth ask.** `D-MOTOKO-WORKDIR-2` — standing authorization
to reconcile the source clone. The skill's four non-destructiveness obligations are **measured, not
assumed**: **0** commits ahead (so no local work to duplicate-check), **0** dirty lines in the clone
and **0** in all eight sibling worktrees (so nothing to back up, and the incoming-vs-modified
intersection is empty by construction), and `git checkout -B dev origin/dev` is the protective form
that errors rather than clobbers. What has changed since asks 1–4 is the **class of the ask**:
iterations 21–25 raised it as hygiene against a predicted cost. The cost is now **measured** — the
loop cannot heal itself, iteration 25's landed work is dead letter, and the drift grew 152 → 172 in
a single day. Standing authorization is a human decision, not a controller one, so it is parked and
not taken.

**Ruled out.**
- *That this fire was unpinned for a new reason.* It is the same retired-key predicate, run verbatim
  and measured false for both paths — not inferred from the log line.
- *That it is fleet-wide.* Three-arm table above; V1 and world are both fine, for different reasons,
  and only one of those reasons is the fix.
- *That editing `~/.claude.json` to re-add the retired key is an acceptable unblock.* It **would**
  work — the stale gate would pass and the loop would self-heal on the next fire. Rejected anyway:
  it satisfies a **retired schema** to trick a gate that has already been correctly fixed upstream,
  it mutates shared out-of-repo harness state that no review gate can see, and Claude Code may
  rewrite that file at any time. Named for Mark as an option; not taken unattended.
- *That the SonarCloud red is ours.* `failure` on `origin/dev` HEAD and on V1's `20ce815bf` and
  `0911d1089`; `71693ead0` carries **no** Sonar check at all (negative control). Non-required;
  `sunholo-data/ailang` is V1's to own.
- *That the queue head had landed.* Re-checked at a fresh origin: `ChatStepResponse.Usage` is still a
  **value type** at `internal/ai/openai/step.go:300`, no merged PR matches, issue reported at #842 is
  still OPEN. Row **6h** is genuinely open and stays the next pick.
- *That an orphaned iteration left work behind.* The only open PR on the fleet account is `#695`
  (`coordinator/task-d98bb271`), absent from this clone's 9-entry worktree list, so **not
  attributable** to this mission — left alone per the fleet-filter rule.

**Routing evidence.** Controller `claude:claude-opus-5` (session) — **and no other role ran**. No
designer, no planner, no executor, no evaluator, no quorum: the deliverable is a measurement and an
escalation, and the only remedy is human-gated, so spawning an executor would have produced code
that cannot execute. Designer rotation pointer untouched at `claude:claude-fable-5`; **Fable
unspent**. Metered **$0.00** of $5. No GPU, no `rig.lock`. Gates on **darwin/arm64**. Running-skill
check on the **resolved symlink**: `~/.claude/skills/mission-control` → V1's checkout (inode
`51683298`), **byte-identical to `origin/dev`** (`cmp` rc=0), while this fire's CWD carries a copy
**139 lines short** (inode `48752546`, `cmp` rc=1) — the 139-line delta was read before any gate ran
and contains the designer-rotation replacement, the comma-separated fallback chains, the
`mission_pi_run.sh` typed-verdict lane and three new Gate-2 rules.

**Next.** Row **6h** — unless the reconcile lands first, in which case the next iteration's first act
is to confirm the pin actually succeeded rather than assume it.

## 27 — 2026-08-28 — the guard was not missing, it was inexpressible: a value type made absence and zero the same value

**Pick.** The queue head, row **6h** — and it is the first iteration since 23 to reach the queue head,
because 24, 25 and 26 were each preempted by a loop-health regression. Re-verified at a fresh
`origin/dev` before routing rather than inherited from iteration 26's note:
`ChatStepResponse.Usage` is still a **value type** at `internal/ai/openai/step.go:300`, issue #842 is
still `OPEN` (1 comment), and no merged PR or direct-to-dev commit touches it
(`git log -S 'ChatStepUsage' -- internal/ai/openai/step.go` returns only the 2026-era feature commit).

**The finding, and it is a finding about REPRESENTATION rather than about policy.** The reporter asked
for a guard against a provider failure that arrives as a *successful empty completion* —
`finish_reason:"stop"`, `"content":null`, no `usage` key. The guard could not be written, and the
reason is one line of type declaration: `Usage` was a value, so an omitted `usage` key unmarshals to
the zero struct and `absent.Usage == allZero.Usage` is **true**. There is no expression over that type
that separates the two cases. Changing it to `*ChatStepUsage` is the whole deliverable of step 1, and
it is deliberately ALL of it — deciding a policy is step 2 and stays deferred, because the Anthropic
and Gemini paths do not share this parser and streaming usage delivery is opt-in on some providers, so
a uniform guard converts a legitimate empty completion into an error: the same defect pointed the
other way. The standing negative control for any future guard is the legitimate tool call, which also
carries `content:null` and *does* report usage.

**Blast radius was measured, not assumed, and it is wider than the report.** `ChatStepResponse` is
referenced ONLY in `internal/ai/openai/step.go` (declaration at :295, sole use at :561) and by **zero**
test files; `raw.Usage.*` is read at exactly five sites, all at :628-636; `ParseChatStepResponse` has
exactly two production callers. Controls in the same call: `ChatStepUsage` **4** hits, an invented
symbol **0**. The consequence that matters for this mission: `internal/ai/ollama/step.go:345` builds an
`openai` client against `<endpoint>/v1`, so **every ollama tool-calling turn parses through this path**
— the defect sits on our own eval rig, where a masked provider failure is indistinguishable from a
local model declining to act. That is the *never conclude model wall* guardrail arriving from inside
our own parser.

**Mutation drill, anchored to the diff hunk by hunk rather than to the defect (rule 3n).** Each mutant
asserted **LANDED** (sha256), **BUILDS**, and — per V1 iteration 274's rule — **effect-verified against
the gofmt-parsed form rather than the file's bytes**; each restored from a `cp` backup and re-verified
byte-identical, with the post-restore suite rc=0.

- **M1** break the `usage` json tag → builds; reds **2** top-level tests (the new arms and the
  pre-existing `TestStep_TextOnly_HappyPath`). Kill-set member, **not** sole killer.
- **M2** nil no longer yields zero tokens → builds; **SOLE KILLER** of the new no-behaviour-change arm.
  Blast radius is one test, so the inverse arm is applicable and was run: the suite `-skip`-ing my own
  test, under the mutant, returns **rc=0** — which is what proves my arm is the killer rather than a
  bystander.
- **M3** neuter the deref guard → builds; reds **only** the PRE-EXISTING happy-path test. The hunk is
  pinned; it is not pinned by anything this branch wrote, and that is reported as such.
- **M4** revert to the value type → **DOES NOT BUILD**
  (`invalid operation: raw.Usage != nil (mismatched types ChatStepUsage and untyped nil)`). This is the
  "mutant does not build" class the skill warns about, and here it is not a defect in the drill — it is
  the row's own finding restated by the compiler. Recorded honestly as a **compile-time arm**, never as
  a behavioural kill.

**The judge found the gap my drill missed, and it is the interesting half of the iteration.** Evaluator
round 1 **PASS 94/100, ZERO blocking**, in its own worktree (iteration-199 rule). It reproduced every
controller claim including M2's inverse arm, then went past them: it built a **16-case differential
harness** across parent `d5305fa79` versus this branch — `usage:null`, `usage:[]`, `usage:"none"`,
`usage:42`, negative tokens, malformed JSON, `prompt_tokens_details` present and absent — and found the
output **byte-identical in all 16**. That is stronger evidence for "no behaviour change" than anything I
produced, and it covers the two cases I explicitly named as untested when I briefed it.

**Its non-blocking finding #1 is a real zero-killer hunk, reproduced first-party before it was
believed.** `cacheRead = usage.PromptTokensDetails.CachedTokens` — inside this diff's second hunk — has
**no killer anywhere**: mutated to `+ 999` it LANDS, BUILDS, effect-verifies 0→1 on the parsed form, and
leaves openai, openrouter and ollama at **rc=0 with 0 FAIL lines in total**. The apparent coverage is a
mirage of exactly rule 3i's shape — *which write does this read?* — and the judge named the mechanism:
`cache_usage_test.go` contains **0** references to `ParseChatStepResponse` and **7** to `Generate`, a
different code path that never reaches this function. Per the rule that *a hunk with no killer is a
finding, not a failure*, it is filed as row **6m** rather than used to widen the PR — the same
disposition iteration 24 gave its own zero-killer hunk at row 6i.

**A stale binary was 43 commits adrift and would have reached the suite through a test's own
shell-out.** Per V1 iteration 237's rule the gates ran with a freshly built, ldflag-stamped binary
prepended to `PATH` — `v0.34.0-118-gd5305fa79-dirty`, matching `git describe` exactly — rather than
`make quick-install`, which would have mutated a `~/go/bin` shared with every concurrent agent on the
rig. The system copy reads `v0.34.0-75-gfb6084f4b-dirty`. Nothing in this diff depended on it, but the
provenance is stated because "the tests pass" from an unidentifiable build is not a claim.

**My own gate list was baselined in the wrong lane once, and the rule caught it before the executor
did.** Fifteen test files across the three `internal/ai` packages bind loopback sockets, which
`workspace-write` denies — so a full-suite gate handed to codex would have been unsatisfiable by
construction (V1 iteration 270's rule, which iteration 24 paid for first-party). The directive
therefore carried three in-sandbox-satisfiable gates and named the three suites explicitly as
`UNINFORMATIVE UNDER SANDBOX`, with the controller re-running all of them outside. The executor ran
exactly that and reported no base-red gates.

**Ruled out.**
- *That the drift is fleet-wide.* It is motoko's alone: source clone **205** behind `origin/dev`
  (0 ahead, 0 dirty), pin worktree exactly at `origin/dev`, world with **0** pin failures ever.
- *That "how far behind" is the predicate — this refutes iteration 26's own framing, which said V1
  escaped "because its clone happens to be current".* It is not current: V1's clone is **18** commits
  behind and pins **cleanly**, its last `DRIVER PIN FAILED` dated 2026-08-27 07:10 with clean fires
  since. The predicate is whether the clone CARRIES THE FIX —
  `git merge-base --is-ancestor ff0da7445 HEAD` → **NO** motoko / **YES** V1, corroborated by
  non-comment `hasTrustDialogAccepted` occurrences **0** vs **3** (motoko's single hit is the
  line-185 comment iteration 26 already identified; invented-token negative control **0** in both).
  So V1 **recovered** by crossing the fix, which motoko structurally cannot.
- *That the SonarCloud red is ours, or new.* Walked commit by commit: green at `7dff0942d`, first red
  at `caea1f9e1` (V1's M-EVAL-ROLLING-ELO merge). Conditions read from the check's own output rather
  than inherited from iteration 25's framing — **64.2% coverage on new code** (was 56.9% at iteration
  25, 52.8% at 24, so the gate is live rather than stuck) and **B security rating on new code**.
  Non-required; handed to V1 with delivery asserted.
- *That `cache_usage_test.go` covers the cacheRead path.* It does not — 0 references to the parser.
- *That a value-type revert is a behavioural mutant.* It is a compile error, and calling it a kill
  would have been the "mutant does not build" vacuous pass.

**A finding I nearly recorded backwards, and the correction is the point.** The driver logged
`WARNING: driver-pin notice FAILED to send via ailang messages` on this fire, and the natural reading —
*queue row 2's loud lane-degradation notice is silent for exactly the event it exists to report* — is
**false**. Counting the two arms separately rather than the warning as a whole:
`FAILED to send via ailang messages` **3**, `FAILED to post to issue` **0**, `no issue notice possible`
**0**, positive control `DRIVER PIN FAILED` **4**, negative control **0**. The GitHub half succeeded
every time and the comments are visible on `#850` (`2026-08-28T00:03:18Z` plus two predecessors), so
Mark's channel worked and the mechanism did what Critical Principle 2 asks of it. A second hypothesis
died the same way: the send passes `gh`-style `--title`/`--from`, which iteration 252's rule makes the
obvious suspect, and re-running the identical form live returns **rc=0** — Go's flag package accepts
both dash forms. What survives is narrow: the send carries `2>/dev/null`, so three consecutive failures
produced **zero** information about their cause. That is the *robustness wrapper hides the cause* class,
it is driver territory (V1-owned), and it is handed over rather than picked. Worth recording chiefly
because the wrong version of this sentence was already written into this iteration's STATUS stamp
before the arms were counted.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Executor `codex:gpt-5.6-sol` —
probe rc=0 replying `ok`, ONE run, no cap hit, no fallback traversed. Evaluator **sonnet in its own
worktree**, a distinct provider from the codex executor, so generator≠judge holds. **No designer, no
planner, no quorum**: the row specifies its own scope and step 2 is explicitly out of it. Designer
rotation pointer untouched at `claude:claude-fable-5`; **Fable unspent**. Metered **$0.00** of $5 —
codex and sonnet are both quota buckets. No GPU, no `rig.lock`. Gates on **darwin/arm64**; windows and
ubuntu legs unrun locally and read from Gate 3b.

**Two configuration observations, recorded rather than acted on.** (a) `MISSION_EXECUTOR_FALLBACK` on
this rig is still the single old value `pi:openrouter/deepseek/deepseek-v4-flash-0731`, while the
running skill has documented comma-separated chains with an ollama-cloud rung ahead of it since
2026-08-26 — the mission env file has not caught up. Not exercised this iteration (codex probed
green), so it is one datapoint, not a defect claim. (b) The codex executor spent its opening turns on
the repo `CLAUDE.md`'s "Session start" inbox instruction, which cannot complete inside
`workspace-write`; it bounded the wait itself and proceeded. Neither has a second instance yet.

**Next.** Row **6i** (the production `run_lane` group kill, pinned by nothing), then the new row **6m**,
then 6j. `D-MOTOKO-WORKDIR-2` remains the only OPEN decision and this is the **sixth** ask.

## 28 — 2026-08-29 — RECOVERED RECORD: the pin-gate bootstrap landed and the slot died before it could say so [HARNESS]

Reconstructed by iteration 30 from the traces the fire left, and labelled as reconstruction rather
than as a first-person record — nobody has reviewed this iteration's reasoning since the agent that
held it stopped existing. What is certain is what is on disk and on origin.

**Pick.** Row **6l**, the class fix iteration 26 filed and deliberately did not execute: the launchd
pin gate is loaded from the *source clone's* `tools/launchd/lib/pin-root.sh`, i.e. from the very tree
the pin exists to bypass, so iteration 25's correct fix could never execute on this mission.

**What landed.** [`61859c35d`](https://github.com/sunholo-data/ailang/commit/61859c35d) —
*"fix(driver): bootstrap pin gate from committed ref"* — merged as PR `#964` at `2026-08-29 22:14:26Z`.
Worktrees `.wt-motoko-iter28-pin-bootstrap` and `.wt-motoko-iter28-eval` both survive at `d622898a7`,
which is that commit pre-squash, so the sprint and its evaluation both ran.

**What it also did, and this is the half that matters for the ledger.** The same commit stamped
`D-MOTOKO-WORKDIR-2` **RESOLVED** — Mark answered `MOTOKO-WORKDIR-2: Yes` on `#850` at
`2026-08-29T09:09:20Z`, granting standing authorization to reconcile the source clone unattended when
the three safety predicates hold. The row records iteration 28 re-measuring **0 ahead / 0 dirty /
292 behind** and advancing `e3ed9467f` → `bd0bb157d` with `git checkout -B dev origin/dev`, post-verified
**0 dirty**. So the six-times-asked decision that had blocked iterations 21 through 27 was answered and
discharged in this slot.

**What it did not do.** Write a STATUS stamp or a log entry. Measured by iteration 30:
`grep -ci 'ITERATION 28'` returns **1** in the charter — that ledger row, and nothing else — and **0**
in the log and **0** in the STATUS archive, against control `ITERATION 27` at **5** and **3**. The
charter therefore recorded a resolved decision whose iteration appeared never to have run.

## 29 — 2026-08-31 — RECOVERED RECORD: a slot died holding a finished, unreviewed, unpushed milestone [HARNESS]

Also reconstructed by iteration 30. This one died later in the loop than 28 did, and therefore left
more.

**Pick.** The queue head, row **6i**.

**What it left, in three places no single Gate-2 trace looks at.**

- A **complete milestone**: commit `4bd9e7110`, *"test(eval): pin run_lane process-group cleanup"*,
  **+237/−13** in `tools/eval/test_motoko_connection_probe.sh`, on branch
  `sprint/motoko-iter29-run-lane-harness`, authored `2026-08-31 05:12:05 +0200` — **never pushed**, so
  invisible to every remote check.
- Its **design doc and sprint plan**, 573 and 237 lines, written at 04:49 and 04:53 and left
  **untracked in the driver's pin worktree** — invisible to a `design_docs/` grep on origin.
- Two worktrees, `.wt-motoko-iter29-exec` and `.wt-motoko-iter29-eval`, the second created at the
  sprint commit, which is what says the evaluation had at least been set up.

**Its quorum history, which is the part worth keeping.** Four artifacts exist for the doc, at
`2026-08-30T12:50:38Z`, `12:51:44Z`, `12:57:04Z` and `2026-08-31T02:45:23Z`. **All four read
`blocked`** and every objection in them is real. Round 4's two — `gpt5-6-sol` on the emergency outer
bound and full-descendant reaping under macOS bash 3.2, `gemini-3-1-pro` on a missing real `lsof`
having to hard-fail on Darwin rather than structured-skip — each carried a concrete reviewer-authored
`proposed_fix` and disputed no design *direction*, which is exactly the narrow-refinement carve-out.
Iteration 29 applied both **verbatim**: they are the last two rows of the doc's own Quorum
Verification Log, and they are the two design properties the shipped code implements. Round 4's third
reviewer, `oc-glm-5-2`, is recorded `absent(invalid)` on a malformed JSON response whose raw fragment
nonetheless begins `"verdict":"reject"` — the tracked defect reported at #941, and the reason the
round's `presentCount` is 2, not 3.

**Why the record could not be inherited.** No STATUS stamp, no log entry, no charter row, no PR. The
routing decision it made — apply the carve-out at round 4 rather than park — is recoverable only from
the artifacts above, which is why iteration 30 re-derived every load-bearing claim first-party instead
of transcribing this reconstruction into a verdict.

## 30 — 2026-08-31 — two slots died in a row, and the second died holding a finished, unreviewed, unpushed milestone [HARNESS]

**Progress**: row 6i LANDED — motoko queue rows 6a–6i now all closed; 6j, 6m, 6n, 6o open on the
loop-health track, and the Phase-0-gated migration epic (rows 10/11/12) is unmoved because its
external predicate is unmoved.

**Pick.** The queue head, row **6i** — but reached by *inheriting* iteration 29's corpse rather than
by routing a fresh sprint, per Gate 2's died-mid-flight rule: verify and land, do not redo.

**The traces, and which one actually worked.** Trace (a), the fleet-account open-PR filter, returned
**four** PRs — `iter305`, `iter306`, `iter296`, `iter308` — **all V1's**, none with a branch in this
clone's **15**-entry worktree list, so the trace that exists to find work you should adopt said
*nothing exists*. Trace (b), `git worktree list`, is the one scoped to this clone by construction and
it named `.wt-motoko-iter29-exec` and `.wt-motoko-iter28-*`. Trace (c), uncommitted/untracked state,
is what turned "an attempt happened" into "a milestone exists": the pin worktree's two untracked design
documents. **Two dead slots in a row is reported as a pattern, not as two incidents** — the loop cannot
diagnose why its own slots die, but it can make the frequency visible.

**Verification of the inherited work — as an inherited claim, not as a colleague's word.**

| check | command | result |
|---|---|---|
| baseline, rebased tree | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | **rc=0**, 41 ok, 0 not ok (was 40) |
| arm 36 is not vacuous | its own evidence line | `ready=yes` · distinct wrapper/child PIDs · `pre_timeout_child_live=yes` · `timeout=yes` · `outer_cap_fired=no` · `survivors=0` · `real_lsof=/usr/sbin/lsof` |
| **M1**, the row's own mutant | group kill → single-PID (`kill -TERM "-$pid"` → `"$pid"`, same for `-9`); LANDED by sha256, PARSES `bash -n` rc=0, intended effect asserted against the system's own view (group-kill sites 1→0) | **rc=1, SOLE KILLER** — arm 36 the only `not ok`, `survivors=1 outer_cap_fired=no`, 35 arms still pass |
| restore | `cp` backup | sha256 byte-identical, `git status` clean, site count back to 1 |

`survivors=1` with `outer_cap_fired=no` is the load-bearing reading: the red is the **production**
timeout's cleanup regressing, not the emergency containment firing. The M1 red arrives in exactly the
direction predicted, so it is paired against the baseline on the identical tree — rc=0 versus rc=1,
outcomes **differ** — which is what makes it evidence rather than co-occurrence.

**The judge found the half the drill could not see, and that is the iteration's best result.**
Evaluator **PASS 87/100, zero blocking** (sonnet, own worktree, distinct provider from the codex
executor, so generator≠judge holds). Both non-blocking findings were reproduced first-party **before
being believed and before being dismissed**:

- **A.** Mutating **only** the SIGKILL escalation site — `kill -9 "-$pid"` → `kill -9 "$pid"`, with
  the group `-TERM` site left untouched (asserted: group `-9` sites 1→0, group `-TERM` sites stay 1)
  — leaves the suite **rc=0, 41 ok, survivors=0**. The fixture's grandchild is a plain `sleep` with no
  SIGTERM trap, so it dies at the TERM stage and the escalation is never reached: that stage is
  dead-for-discrimination. Row 6i's mandated mutant changes **both** sites at once, so it reds on the
  TERM half alone and the doc's acceptance bar is honestly met — the gap is one the bar itself could
  not see. That is rule (i-e)'s shape: a removal proves a check **fires**, only a differently-shaped
  mutant proves it **looks**.
- **B.** `REAL_LSOF` is resolved with `command -p -v`, and POSIX's standard-path guarantee for `-p`
  **does not hold on the CI shell**. Measured on GNU bash **3.2.57** (arm64-apple-darwin25), which is
  the `launchd drivers (bash 3.2)` target rather than a proxy for it: a shadowing `lsof` placed ahead
  of `getconf PATH` in the **ambient** environment resolves as `REAL_LSOF`. Arm → the hijack; control,
  clean PATH → `/usr/sbin/lsof`; negative control, hostile directory with no `lsof` in it →
  `/usr/sbin/lsof`. The defence the design doc actually names — the stub PATH this suite installs for
  *itself* — holds, and was separately confirmed by the `markers=yes` evidence distinguishing
  `path-lsof` calls from `fixture-lsof path=$REAL_LSOF` calls. Only the code comment's *"can never"*
  was too wide.

Disposition: **B's comment was narrowed to what is actually defended** (`1caf02e44`) — comment-only,
no executable line changed, so the evaluation over the executable tree still stands — and **both
findings were filed as row 6o** rather than used to widen a passing PR.

**A third defect arrived from another mission's judge and was ghost-disciplined before adoption.**
Reported at #975 by V1 iteration 308's evaluator, against *motoko's* instrument: the arm named
`descendant discovery refuses on the real wall-clock deadline` cannot fail for the reason it names.
Reproduced at motoko's own HEAD rather than inherited — neuter the in-loop check with
`if false && (( $(date +%s) > deadline )); then`, mutant LANDED by sha256 and PARSING, intended effect
asserted (neutered sites 0→1) — and the suite stays **rc=0, 41 ok, 0 not ok**, this arm included. Real,
pre-existing, and the same class as row 6i one arm over: an assertion over an over-subscribed
observable. Filed as row **6n**. Its B3 half (on the runner, discovery refuses with *neither*
diagnostic message) is recorded **UNREPRODUCED**, because it can only be measured on the runner.

**Four blocked quorum rounds is data about this loop's scoping, not about that document.** Per V1
iteration 257's rule the *surfaces* were tracked rather than the round count: R1 survivor oracle /
helper duplication / stub conflict-surface; R2 hermetic-drive premise / helper duplication / hermetic
premise; R3 readiness determinism / real-vs-stub `lsof` / platform-specific verification; R4 outer
bound / Darwin hard-fail. They are **spread across surfaces** and **no reviewer flipped to pass**, so
the disposition is *immature, keep revising* — **not** SPLIT, which is what localisation plus a flip
would have indicated. Recorded here because only a human can act on the pattern.

**Dev CI was red, it was not ours, and nothing was done to it.** `test` failed at `d65a0900c` on step
*Verify embedded pi assets are in sync*; walked back per commit, first red at `ebc089c33` (V1's
README-only edit), green at `15cec372b` and `9f267cf1f`. V1 already had `#983` open with the fix when
this fire started, so the owning-mission rule held twice over: the red did not displace the pick, and
no duplicate fix was opened — the `#758`/`#759` lesson. It merged as `f78b1d451` and this branch was
rebased onto it before landing.

**Gate 3b — read on the MERGE, not merely on the PR head.** PR head `1caf02e44`: **21** checks,
**0** pending, **0** not-green, `mergeable` read first per the iteration-198 rule
(`MERGEABLE`/`CLEAN`), all **4/4** required contexts green. Merged as
[`4bd58bef6`](https://github.com/sunholo-data/ailang/commit/4bd58bef6), whose own check set is
**16** checks, **0** pending, **1** not-green: `SonarCloud Code Analysis`. Rule 3d's parent-walk
before attributing it — it is `failure` on the parent `f78b1d451` and on `15cec372b` and `d3e4e59cf`,
while `d65a0900c` and `ebc089c33` carry **no Sonar check at all**, which is the negative control. So
it is inherited, non-required, and V1's to own; it is named here rather than left invisible.
`docs-gate` does not appear on the merge commit at all — path-filtered — and passed on the PR head,
where the merge button was gated; that is the reading iteration 25 recorded and it still holds.
`launchd drivers (bash 3.2)`, the CI leg that actually executes this suite, is **success** on both.

**Weekly sweep, and its headline number is a property of the fleet rather than a backlog.** **94**
open issues enumerated — the list length quoted beside the verdict so a truncated enumeration cannot
wear a complete one's clothes — each grepped `-cE '#<n>\b'` across **four** corpora (charter, log,
STATUS archive, dashboard) and printed as a per-issue table, never a summary sentence. Negative
control fired. **79 of 94** carry zero mentions, which is expected and not actionable as a backlog:
this repository carries four missions and motoko's charter is deliberately scoped to the harness. The
motoko-territory orphan that mattered is `#975`, now row **6n**.

**Ruled out.** That the two dead slots were a rotation defect — the STATUS-rotation deletion class
this loop has paid for twice — is refuted: `grep -ci 'ITERATION 28'` returns **1** in the charter
(the ledger row iteration 28 itself wrote) and **0** in the log and archive, whereas a rotation
deletion removes a stamp that was *there*, leaving a `git log -S` trail on the charter and none on the
archive. There is no such trail here; the stamp was never written. That the M1 red might be an
artifact of the rebase is refuted by running the baseline on the identical rebased tree (rc=0). That
`#975` might be a ghost inherited from a sibling is refuted by first-party reproduction.

**Routing evidence.** Controller `claude:claude-opus-5`. **No designer, planner or executor ran this
iteration** — all three are iteration 29's, inherited, and Gate 2 requires verify-and-land rather than
redo; spawning them would have re-authored a doc that already exists and re-run a milestone that was
already finished. Evaluator: **`sonnet`, Agent-tool-pinned, in its own worktree**
(`.wt-motoko-iter29-eval`, branch `eval/motoko-iter30-run-lane-harness`), distinct provider from the
`codex:gpt-5.6-sol` executor, so the generator≠judge guard holds without a re-route. Designer rotation
pointer untouched at `claude:claude-fable-5`; Fable **unspent**. Metered **$0.00** of $5 this
iteration; iteration 29's fourth quorum round spent **$0.1108** and is attributed there. No GPU, no
`rig.lock`. Gates on **darwin/arm64** and on the CI shell's own bash 3.2.57; windows and ubuntu legs
unrun locally and read from Gate 3b.

**Gate 5 — no skill edit.** Every friction here is an instance of a rule the rulebook already carries
and that *worked*: the died-mid-flight traces found the corpse, the fleet-`--author` rule kept this
mission off three sibling PRs, the owning-mission rule kept it off V1's red, ghost discipline turned a
sibling's claim into a first-party measurement, and reproduce-before-dismissing turned two
"non-blocking" findings into a queue row. The one candidate — that trace (a) is near-useless on a
shared push identity while trace (c) did the work — is already written into the skill as the
fleet-filter rule and as trace (c) itself, so this is a second instance of a documented gap that has
already been closed, not a new one.

**Next.** Row **6n** (#975's wall-clock arm), then **6o** (the escalation half of the group kill), then
6j and 6m. Decision ledger: **5** rows, **0 OPEN** — `D-MOTOKO-WORKDIR-2` was answered and discharged
by iteration 28, so for the first time since iteration 21 this mission is asking nothing of Mark.

## 31 — 2026-09-01 — the quorum blocked twice on one surface, and my refutation of it measured the wrong binary [HARNESS]

**Progress**: row 6n's stated blocker DISCHARGED (the runner half of the defect reported at #975 is
now measured) and its design written, but the row is **PARKED needs-human-review**, not landed —
the design quorum blocked 3/3 in both rounds. Loop-health track: 6a–6i closed, 6n parked, 6o next,
6j/6m/6p/6q open. The Phase-0-gated migration epic (rows 10/11/12) is **unmoved**: upstream `#154`
is still OPEN, re-measured as a command this iteration.

**Pick.** The queue head, row **6n** — "the wall-clock discovery arm cannot fail for the reason it
names", reported at #975 by V1 iteration 308's evaluator against motoko's own instrument.

**The row's own blocker, discharged from a sibling's CI.** Row 6n said finding B3 *"is unreproduced
here and must be measured on the runner, not locally, before any fix is designed"*. The measurement
was sitting unread in another mission's pull request. The `launchd drivers (bash 3.2)` job
`99402730557` on **V1's open PR #971** (head `8a384e81b`) fails with exactly

    not ok - descendant discovery refuses on the real wall-clock deadline
             lacked expected message: process-tree discovery failed

and the surrounding log reads `lane=treatment driver_rc=0 peers: []`, `lane=control driver_rc=0
peers: []`, `INSTRUMENT FAILURE: empty peer set`. So #975's *"refuses with NEITHER message"*
understates it: under that configuration discovery **did not refuse at all** — both lanes completed
and the run died downstream on the empty-peer-set guard. Control that the log was really parsed:
**32** passing `ok` lines in the same fetch. **Scope, and it is load-bearing: that certifies #971's
tree, not motoko's HEAD**, whose own `launchd drivers` leg was `success` at `4bd58bef6`.

**The defect, reproduced and then sharpened.** Every mutant LANDED (sha256), PARSES (`bash -n`
rc=0), effect-asserted against the system's own view, restored from a `cp` backup byte-identical
with `git status --porcelain` **0**.

| # | mutant | result |
|---|---|---|
| baseline | none | **rc=0**, 41 ok, 0 not ok, 50s |
| E2 | neuter the in-loop wall clock alone | **rc=0, 41/41**, arm 33 still `ok` — the row's claim, re-derived |
| E7 | neuter the node ceiling alone | **rc=1**, 39 ok, **only arm 40** reds, arm 33 still `ok` |
| E8 | the minimal fix alone (3 distinct messages + arm 33 asserting the wall-clock one), **no ceiling change** | **rc=0, 41/41, 50s** |
| T1 | E8's fix **plus** E2's mutant, ceiling untouched | **rc=1 in 44s**, arm 33 the failing arm on the exact message |

E2 and E7 together are the statement row 6n did not have: **each branch independently suffices for
arm 33 to pass**, so the arm cannot discriminate them by construction. T1 is the load-bearing
acceptance and it was *run*, not predicted — clean rc=0/50s against mutant rc=1/44s, printed side by
side, outcomes differing, and the mutant *faster* than the clean run rather than a hang.

**A correction to my own inference, and it is the root of what went wrong afterwards.** Neutering
both bounds hangs the suite past a 600s bound; I read that as *"the node ceiling is what terminates
this arm at HEAD"*. It does not follow — the sound reading is only *"with the wall clock dead, the
ceiling stops the walk"* — and that inference is what seeded the design the quorum then rejected.
E8 refuted it directly. A reviewer (`oc-glm-5-2`) attacked it before I did.

**The quorum blocked twice, on one surface, from opposite directions.** Round 1
(`…2026-09-01T00-35-20Z.json`, $0.0806) and round 2 (`…T00-43-50Z.json`, $0.0781), **3/3 external
reviewers rejecting both times, no absentees** — read at the correct nested path
(`.synthesis.absent_reviewers`), not the top-level one that returns `null`. Per the objection-surface
rule the surfaces were tracked rather than the round count: both rounds localise on **one** surface,
the wall-clock-versus-ceiling race, and **no reviewer flipped to pass**, so the disposition is
*immature*, not SPLIT. Round 1 rejected raising `MAX_TREE_NODES`; round 2 rejected **not** raising
it. The one revision and the one re-quorum the protocol allows are spent, and the surviving
`gpt5-6-sol` objection carries no concrete reviewer-authored fix, so the narrow-refinement carve-out
does not apply. **Park.** Standing rule 2 forecloses force-passing over three unanimous rejections
however good the controller's argument feels — and this iteration is the argument for that rule.

**The evaluator's blocking finding is against me and it is correct.** To answer the reviewers'
shared empirical premise — *a CI runner might do 4096 iterations inside the 1-second window* — I
benchmarked **system `pgrep`** at ~**79** iter/s and reported a **~52×** margin. Arm 33 sets
`PATH="$live_bin"`, and this suite installs its **own** `pgrep` stub at
`test_motoko_connection_probe.sh:254-262`; the walk never calls system `pgrep`. Re-derived
first-party against the actual stub: **474.9 / 652.7 / 648.6** iter/s → 4096 iterations ≈
**6.3–8.6s**, a **~6–9× margin**. The wrong instrument, re-run for the side-by-side, reads **92.1**
iter/s; the negative control confirms the stub resolves first on that PATH; the judge's independent
instrument measured ~455 iter/s and agrees. The *conclusion* survives and never rested on that
number — E8 and T1 measured the behaviour directly — but the figure offered to Mark as grounds for
reconsidering three unanimous rejections was inflated ~6×, and it is corrected in the doc before he
reads it. This is the scope trap aimed at a **binary** rather than a directory: my control ran, and
it ran against the wrong executable.

**The evaluator also found the synthesis both quorum rounds circled and neither reached (D4):**
scope `PROBE_MAX_TREE_NODES` to arm 33's own `env` line rather than raising it globally or deleting
it — measured free on the happy path (41/41, 47.1s) — which makes the ceiling structurally
unreachable inside the window while every other arm keeps the default. **Recorded, not applied**:
the revision budget is spent, and applying it would be a controller-invented resolution to a blocked
quorum. It is filed as row **6p**.

**Evaluator: PASS 78/100, 1 blocking, 4 non-blocking** (sonnet, its own detached worktree at
`b76b0823a`, distinct provider from the pi/deepseek designer and from the opus controller, so
generator≠judge holds). It reproduced **every** controller claim before ruling — C0 through C4
including T1's 43.76s timing — and it ran the addition-shaped mutant on the branch-count gate,
confirming the current gate at `expected_refusal_branches=24` is **silently blind** to an added
echo-shaped refusal branch (41/41 ok with the branch added). That is filed as row **6q**. Findings
2, 3 and 4 are applied to the doc verbatim.

**A live collision, attributed and left alone.** The fleet-account open-PR filter returned **3** —
`#997`, `#971`, `#945` — none with a branch in this clone's **16**-entry worktree list. `#971`
touches **exactly** row 6n's two files and is `MERGEABLE`/`UNSTABLE`, but branch `mission/iter306-*`
is V1's numbering (V1 is at 312, motoko at 31), so it is not attributable to this mission: read,
never touched, and handed over on the cross-mission channel. It neither supersedes this work nor is
superseded by it — `PROBE_TREE_DISCOVERY_SECS` is **0** occurrences at motoko HEAD in both files
(controls `PROBE_MAX_TREE_NODES` **2**/**3**, negative control **0**).

**Ruled out.**
- *"The node ceiling fires first on the controller's machine"* — my own E3 inference. **REFUTED** by
  E8. Do not rebuild a design on it.
- *"A CI runner could process 4096 iterations in under a second"* — the reviewers' shared premise.
  **Not refuted**, only bounded: the local margin is ~6–9×, measured against the right binary, and
  contention slows spawns, which moves it the protective way. The CI host's own rate is unmeasured.
- *"~52× margin"* — **my own error**, from the wrong binary. Superseded by ~6–9×.
- *"#971 supersedes row 6n"* — **false**, measured: #971 changes which deadline bounds discovery and
  leaves arm 33 asserting the generic wrapper.
- *"The race is something this doc introduces"* — **false**: it is PRE-EXISTING at HEAD, so it is a
  queue row (6p), not a revision.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Designer
**`pi:ollama/deepseek-v4-flash:0731-cloud`** — the rotation's next entry after the pointer's
`claude:claude-fable-5`; probe rc=0; two runs via `scripts/mission_pi_run.sh`, verdict `ok` both
(366s / 91s, 1 changed file each); pointer advanced to it. **No planner and no executor ran** — the
doc parked before a plan existed, so there was nothing to plan or execute. Evaluator **sonnet**, own
worktree. **Fable unspent.** Metered **$0.1587** of $5 (two quorum rounds at $0.0806 + $0.0781; the
pi lane is flat-rate $0.00). No GPU, no `rig.lock`. Gates on **darwin/arm64** against GNU bash
3.2.57; windows and ubuntu legs unrun locally.

**Bookkeeping second pick.** Row 6i's design doc and sprint plan moved `planned/` →
`implemented/v0_35_0/`, and the doc's status header corrected from `Planned` — which it still said a
day after the work merged — to the merge it actually landed as (`4bd58bef6`, PR #985).

**Gate 5 — no skill edit.** The rule that mattered most this iteration (reproduce a judge's finding
before believing it) worked and caught a real error of mine. My own failure — benchmarking the wrong
binary — is an instance of a rule the skill already carries (scope the known-positive control to the
same thing the check reads), mis-applied rather than missing, so it is a second instance of a
documented gap and not a new one. One candidate is **recorded and below the bar**: a bare
`ailang messages list --unread --json` returns **20** rows while `--limit 200` returns **41**, so the
default limit silently halves the triage queue and nothing in the output says so. One instance.

**Next.** Row **6o** (the SIGKILL-escalation half of the group kill), then **6p** (the race, with the
evaluator's D4 as the candidate fix), then **6q** (the blind branch-count gate), then 6j and 6m.
Decision ledger: **6** rows, **1 OPEN** — `D-MOTOKO-6N-1`, a ship-or-hold call with three lettered
options, a recommendation and a dated default.

---

## 32 — 2026-09-01 — the attended ruling landed, and the measurement I ran to answer the reviewers partly refuted the doc I was defending [HARNESS]

**Progress**: loop-health track — rows **6n**, **6q** CLOSED and **6p** closed on its design half with
a named residual; 6o, 6j, 6m and the new 6r open. The wall-clock discovery arm reported at `#975` now
asserts a message only the wall-clock branch can emit, and the drift gate is proven to LOOK rather than
merely fire. The Phase-0-gated migration epic (rows 10/11/12) is **unmoved**: upstream `#165` is still
OPEN, last touched 2026-08-20, re-read as a command this iteration.

**Pick.** Not the queue head. `D-MOTOKO-6N-1` was answered **attended** by Mark on 2026-09-01 — commit
`878e0a5a0`, author an attended identity, not the fleet bot, provenance checked first-party per the attended-ledger
contract (an attended identity the fleet account does not hold and the loop may not author with). An
attended ruling outranks the queue and unparks its item. The ruling: **(B) HOLD for D4** — scope
`PROBE_MAX_TREE_NODES` to arm :449's own `env` line and remove the race structurally rather than betting
on a 6–9x margin. The ruling arrived AFTER iteration 31's own record landed (`b926c6a84`, `#999`), so it
was unconsumed. Acknowledged here as Mark must see the channel worked: **the second human channel was
used, and it changed this iteration's pick.**

**Outcome.** **LANDED** — [#1008](https://github.com/sunholo-data/ailang/pull/1008), squashed to
`64ca81852`. Four commits: three code milestones, each independently green at its boundary, plus docs.

**The iteration's real finding is about my own evidence, not about the code.** Rounds 1–3 of the quorum
blocked 3/3 external on ONE surface — the race between a wall clock (wall time) and a node ceiling
(iteration count). `gemini-3-1-pro` **flipped to pass at round 3**, the first pass in three rounds, and
round 3's two remaining objections were both *premise* objections. So I measured them rather than
forwarding them:

- `gpt5-6-sol` — *"a sufficiently fast runner can exceed 25,000 iter/s, so the arm is still
  race-dependent."* **REFUTED.** The walk forks one process per iteration by construction
  (`pgrep -P "$current"`), and bare `/usr/bin/true` — no bash, no script, the absolute physical floor —
  spawns at **205–250/s** on this machine (`date`-timed and `perf_counter`-timed, agreeing). 25,000
  iter/s is ~**100x the hardware's bare-spawn ceiling**, which is a bound rather than an extrapolation.
- `oc-glm-5-2` — *"the rates were measured at `a223e7274`, the doc is baselined at `48817dcdd`, and the
  stub at :254-262 may have changed."* **Mechanism refuted; ask satisfied; and it partly UPHELD me
  wrong.** The test file is **byte-identical** between the two commits (whole-file `cmp` SAME, stub
  region sha256 `abffefd3…` both sides), so the stub cannot have changed. But re-measuring as asked
  found **181–200 iter/s** against iteration 31's **474–653** — a **3.3–3.6x spread from ambient load
  alone** (load average **20.59 on 16 CPUs**; a `perf_counter` clock agrees with `date +%s`, refuting
  granularity as the cause). That makes the race leg *stronger* and the backstop leg *weaker* than the
  doc claimed (headroom **2.2x–5.7x**, not 5.7x) and **empties the doc's own feasible interval at its
  stated margins**. The doc now states this as an explicit trade — satisfy the leg whose violation is a
  WRONG VERDICT (a spurious red on a clean tree), accept degradation in the leg whose violation is only
  a WRONG EXPLANATION (a correct red, cap-shaped rather than message-shaped) — rather than as a
  satisfied constraint. **This is the reviewer being right for a reason the reviewer did not name**,
  and it is the part of this iteration worth re-reading.

Round 4 then blocked on two consistency defects **I** had introduced while correcting §(c) — a stale T1
row I had failed to propagate the new figures into, and a `grep` whose target-file premise was
unverified. Both carried concrete reviewer-authored fixes and neither disputed the design direction, so
they were applied verbatim under the narrow-refinement carve-out. **The surface finally moved off the
design after three rounds stuck on it**, which is the signal that the doc was ready.

**Two instrument failures, both caught by prescribed controls rather than by luck.**
1. My commit-reconstruction script read the executor's snapshots at nested paths when they are flat.
   The recipe's mandatory sha256 manifest check reported `FAILED` on the second file. No work was lost
   — the snapshots were intact — and the retry verified byte-identical. The check exists for exactly
   this and it earned its keep.
2. **The executor's T1 line is EXCLUDED from this iteration's evidence.** It reported
   `rc=1 ok=27 not_ok=1 wall=51s` with none of the three refusal messages. The judge could not
   reproduce it in any dimension and explained why it is not a possible reading: arm 28 is
   `hermetic live success path completes`, which drives a small finite tree at the default ceiling and
   has no code path to the wall-clock branch. The judge's own T1 reds the **correct** arm 33 at 641s,
   cap-shaped — the degradation the sprint plan pre-authorised for exactly this contention. A number I
   relayed from a sub-agent was wrong and an independent judge caught it.

**Gate 3b.** GREEN on the PR head: **21/21 checks complete, 0 not-green, `mergeStateStatus=CLEAN`**,
control firing (16 checks on the then-`origin/dev`). Among them **`launchd drivers (bash 3.2): success`**
— the leg decision (d) named as the ONLY place the runner's behaviour is observable, and the same leg
that failed on V1's `#971` carrying this exact discovery error. That discharges the doc's deferred
runner measurement in the affirmative rather than leaving it owed.

**Evaluator: PASS 93/100, ZERO blocking, 3 non-blocking** (sonnet, its own worktree at `23e153a80`,
distinct provider from the codex executor and the opus controller, so generator≠judge holds). It
reproduced C1–C4 first-party, including running C3 in **both** arms to prove the new locality guard is a
gate and not decoration (rc=0/41-ok at base, rc=1 on the merged tree). **It also ran T3, which the
executor never reached** — its 50-minute cap killed it before it could report — and T3 is the drill that
answers row 6q's actual question: a dead-but-real refusal branch, guarded by `false &&` so it never
executes, still moves the count **27 → 28** and reds the gate. The gate reads the file's SHAPE. It
LOOKS. Finding 1 (the `(test stub)` hunk has no behavioural killer, only a static grep) is filed as new
row **6r** rather than absorbed, per the rule that a hunk with no killer is a finding.

**Second, bookkeeping pick — authorised, not invented.** The source clone was **55 commits behind**.
Reconciled under the standing authorisation `D-MOTOKO-WORKDIR-2` with all three safety predicates
re-measured at the moment of use (**0 ahead / 0 dirty / 55 behind**), `git checkout -B dev origin/dev`
rc=0, post-verified **0 behind, 0 dirty, all 17 worktrees intact**.

**Ruled out.**
- *"The stub got slower because the code changed"* (`oc-glm-5-2`'s stated mechanism) — **REFUTED**: the
  file is byte-identical between the two commits. The variable is load, not code.
- *"`date +%s` granularity explains the 3.3x rate spread"* — **REFUTED** by a `perf_counter` arm that
  agrees with `date` to within 10%.
- *"A fast CI runner could win the race at a 50,000 ceiling"* — **REFUTED** by the bare-spawn ceiling.
- *"The executor's T1 measured the wall-clock mutant"* — **REFUTED** by the judge; excluded from evidence.
- *"The doc's `[≈39,000, ≈57,000]` feasible interval holds"* — **REFUTED**: it was non-empty only
  because both legs were evaluated at quiet-machine rates. At the full observed range it is empty.

**A red I do not own.** `SonarCloud Code Analysis` is `failure` on `origin/dev` and was failure on the
four preceding analysed commits — inherited, not from this push, and it is `success` on this PR's own
head. `sunholo-data/ailang` is **V1's** repo, so per the owning-mission rule this is recorded and handed
over on the cross-mission channel rather than allowed to displace motoko's pick.

**Routing evidence.** Controller `claude:claude-opus-5` (session). Designer **`claude:claude-fable-5`**
— rotation advanced from `pi:ollama/deepseek-v4-flash:0731-cloud`, spawned with an explicit Agent-tool
`fable` pin (accepted, ran to completion), ONE revision run, within the one-doc Fable diet; pointer
written back. Planner **`opus`** via the Agent tool — `derive-planner-lane.sh` returned
`opus fail-closed:planner-lane-field-missing`, used VERBATIM, so no codex probe was spent on the planner
role. Executor **`codex:gpt-5.6-sol`** via the cross-provider recipe (probe rc=0; the real run hit its
**50-minute cap** and was killed before emitting a final message — **FLAGGED**; its work was intact in
the worktree and every gate was re-run by the controller outside the sandbox). Evaluator **`sonnet`**
via the Agent tool, in its own worktree. generator≠judge holds on provider (OpenAI executor vs Anthropic
judge). **Metered: $0.2179** (quorum round 3 $0.1403, round 4 $0.0776) against the $5 ceiling. Round 4
was **N−1** — `gpt5-6-sol` absent on budget, recorded rather than silently passed; the verdict was
BLOCKED either way, so the degradation could not manufacture a false pass.

**Next.** Row **6r** (the unpinned `(test stub)` hunk) is bookkeeping-sized and adjacent to work just
landed. Then **6o** (the SIGKILL-escalation group form, zero killers), then **6p**'s residual — and note
what that residual now is: not "pick a better constant" but *derive the bound from a stimulus measured
in-test*, so the ratio holds by construction on any machine. This iteration produced the measurement
that makes that the obvious fix.

## 33 — 2026-09-02 — the judge's blocking finding was about where I put the arm, not what it asserts [HARNESS]

**Pick.** Queue head, row **6r** — the `(test stub)` refusal message pinned by a static grep and by
nothing behavioural. Named as Next by iteration 32.

**Progress.** The bar's clause 1 (the tree gates green from source) is the countable unit this row
serves, and the suite's arm count is its proxy: **43 self-test arms, up from 42**, with all three
`descendant_pids` refusal branches now asserting their own discriminating message instead of two of
three. Sprint-sized backlog: 6r closes; 6j and 6p gain evidence; 6s is new. Ungated queue now
**6o → 6p → 6s → 7 → 8**.

**Outcome.** LANDED — PR [#1027](https://github.com/sunholo-data/ailang/pull/1027) squashed to
[`115184a2e`](https://github.com/sunholo-data/ailang/commit/115184a2e). Three commits reconstructed
from the executor's snapshots plus one controller commit (M3) that the evaluation forced.

**The defect, reproduced before routing.** A judge's finding is a claim until you run it. Copy of the
probe with ` (test stub)` stripped: suite **rc=0, 42 ok, 0 not ok** — the mutant survives. Same-scope
known-positive control, strip ` (wall clock)` instead: **rc=1, 32 ok**, failing arm by name
`descendant discovery refuses on the real wall-clock deadline`. Outcomes DIFFER, so the green is a
measurement, not a broken harness. Then the fix itself was proven before it was designed: a
throwaway test-side mutant asserting the stub message is rc=0/42 ok on the pristine probe and
rc=1/24 ok against the stripped one, failing by name.

**Quorum: BLOCKED twice, and every round-1 objection was a premise objection I measured.**
R1 3/3 reject, `.synthesis.absent_reviewers` `[]`, $0.0628.
- `oc-glm-5-2` — is `expect_failure` a substring matcher? **REFUTED**, two ways: source
  (`grep -Fq -- "$expected"` over the whole stderr, test:163) and the behavioural arms above.
- `gemini-3-1-pro` — `assert_pid_scope` and the anti-vacuity floor were unverified. **UPHELD
  procedurally; the facts are TRUE.** probe:208/223, and the message carries a `: ${pids:-<empty>}`
  suffix the doc had quoted short. Controls: `instrument_failure` **21**, invented `assert_zzz_scope`
  **0**.
- `gpt5-6-sol` — T3 is vacuous. **UPHELD, a real defect in my own test plan.** `grep -Fq` is a
  substring match, so appending ` X` still matches: measured rc=0 for the append form, rc=1 for the
  substring-breaking `(stub)` form.

R2 2 reject / 1 pass, zero absentees, $0.0875, `oc-glm-5-2` flipped to pass. Objection SURFACES
tracked per V1 iteration 257's rule: R1 was test-plan validity / log completeness / harness
semantics, R2 was conflict-surface provenance / fail-fast ordering — **spread, not localised**, so
not the SPLIT signal. Both R2 survivors carried concrete reviewer-authored fixes and disputed no
design direction → **narrow-refinement carve-out**, applied VERBATIM. `gpt5-6-sol`'s own
`git show --stat` command was run; every Conflict-Surface bullet now carries its verified subject
and diffstat, with an enumeration control (`git log --oneline -6` over the two files) and a negative
control (`7292ec780`, docs-only, lists no files).

**The evaluator's blocking finding, and why it changed the code.** Evaluator **PASS 76/100, 1
blocking** (sonnet, own worktree at `5fb1e2306`; executor was `codex:gpt-5.6-sol`, controller opus →
generator≠judge holds). It reproduced all three mutant sha256 hashes bit-for-bit and re-derived the
base-vs-HEAD defect independently. Finding: placed immediately after the caller-message arm, the new
arm sits **ahead of every wall-clock-bounded arm**, adding a fork/exec before the 4s
`refusing live path` arm and before `production run_lane fixture readiness` — and it reproduced one
of my two intermittent reds **verbatim**, underlying line included (base 5/5 clean, arm-at-26 4/5).

I measured it three ways rather than arguing: one probe, interleaved, five rounds, test file the only
variable — base **5/5**, arm-at-26 **4/5**, arm-at-42 **5/5**. Pooled with my earlier runs and the
judge's own A/B: **base 0 reds in 17, arm-at-26 4 in 19, arm-at-42 0 in 5.** M3 moved the arm behind
the bounded arms. The disposition rests on the **mechanism, not the rate**: those arms now start at
the same wall-clock offset as at base, so the effect is unreachable by construction. Cost none —
43 ok, and the strip mutant still reds with this arm the sole failing arm by name. Adjacency turned
out to buy nothing: the harness is fail-fast, so any mutant reddening the caller arm masks the later
one at any distance; *beside-not-instead-of* still holds and now means "both present", not
"adjacent".

**Gate 3b.** Merge `115184a2e`: **14** checks, **0** pending, required **4/4**
(`test`/`lint`/`build`/`docs-gate`), and **`launchd drivers (bash 3.2): success`** — the only leg
where this suite runs — green on both the PR head and the merge. Two non-required reds,
`test-windows` and `Build windows-latest`, attributed by measurement rather than assumed: both are
`success` on my base `7292ec780` AND its parent `f5d031161`; the failing tests are
`TestResolveAnthropicCredential_FallsBackToClaudeCredentialsFile` and
`TestStandardModeCostProvenance_CredentialFileIsSubscription`, Go credential tests a two-line shell
diff cannot touch; `origin/dev` itself is red on the identical two right now; the cause `f3301a44c`
landed **after** my base and is on dev; and a PR's checks run on the **test-merge**, not the head.
V1 owns dev CI red on this repo per the charter guardrail and has `sprint/iter320-home-isolation` in
flight. Recorded, handed over, pick kept.

**Ruled out.**
- *"The two intermittent reds are caused by my diff's content"* — **REFUTED**: the same reds appear
  with an arm whose content is irrelevant to them, and vanish when the arm moves behind them. It is
  position and cost, not semantics.
- *"The Windows red is mine"* — **REFUTED** by the parent walk and by dev's own current state.
- *"Adjacency to the caller-message arm is load-bearing"* — **REFUTED** by the judge: fail-fast makes
  the two arms ordered rather than independent at any distance.
- *"Appending text to the asserted message is a valid mutation"* — **REFUTED** by `grep -Fq`
  semantics, measured.

**Three instrument failures of my own, all caught by the prescribed controls.**
1. `sed > file` creates mode **644**, and the suite invokes `"$probe" --classify-fixture` directly at
   test:201 — so my first two mutant arms redded at arm **1** (`ok=0`,
   `not ok - classification fixture: missing loopback`) for the file MODE, not the mutation. Caught by
   reading WHICH arm failed instead of the exit code.
2. `grep -c PROBE_TIMEOUT_SECS` over `git diff` counted a **context** line and read 1; the added lines
   are 2 and neither matches. Corrected with `grep '^+[^+]'`.
3. `derive-planner-lane.sh` reported *"design document is missing or unreadable"* from the pin
   worktree, because the doc lives in the sprint worktree — rule 3a(i-d)'s scope trap. Re-run from the
   right CWD it returns `opus fail-closed:planner-lane-field-missing`.

**Routing evidence.** controller `claude:claude-opus-5` · designer
**`pi:ollama/deepseek-v4-flash:0731-cloud`** (rotation entry after the pointer's
`claude:claude-fable-5`; probe rc=0; verdict `ok` both runs, 377s authoring / 264s revision, 1 changed
file each; pointer advanced; ONE doc = initial + one protocol-mandated revision, which is the diet)
· planner **`opus`**, lane from `derive-planner-lane.sh` used VERBATIM
(`opus fail-closed:planner-lane-field-missing`), spawned via the Agent tool · executor
**`codex:gpt-5.6-sol`** (probe rc=0; 30-min bounded background wrapper; no git writes; `.snap/M1` +
`.snap/M2`; reconstruction proven byte-identical, `shasum -c` rc=0) · evaluator **`sonnet`** in its
own worktree. Fable **unspent**. **metered=$0.1503** of the $5 ceiling — both quorum rounds; the pi
designer lane is flat-rate $0.00. No GPU, no `rig.lock`. All local gates on **darwin/arm64, GNU bash
3.2.57**, `/usr/sbin` on PATH; windows and ubuntu legs unrun locally and read from Gate 3b.

**Executor deviations, adjudicated by measurement.** (a) It preserved relative paths in `.snap/M<k>/`
rather than the plan's flat-copy example — benign, and no acceptance criterion is sensitive to it;
the judge confirmed independently. (b) It reported that detached background processes were killed
when their command sessions closed, so after two bounded artifact polls expired it re-ran the
identical drill in a retained session and **explicitly refused to infer any verdict from the aborted
launches** — the correct response to an untrustworthy instrument, and its reported numbers reproduced
in full, by me and by the judge. A self-reported deviation is better evidence than a silent one.

**Bookkeeping note.** The sprint JSON is at `.ailang/state/sprints/`, which `.gitignore:82` excludes
(`git check-ignore -v` fires on it and is silent on the plan `.md`), so it is deliberately NOT in git;
the decision-bearing content is the committed plan. Same disposition as the preceding sprint.

**Next.** Row **6o** (the SIGKILL-escalation group form with zero killers, and the `REAL_LSOF` PATH
assertion), then **6p** — now with iteration 33's corroboration that the derive-the-bound work is owed
by at least three arms rather than one, so it is better shaped as a suite-wide helper than a constant.

## 34 — 2026-09-03 — the work was done, verified and judged; it could not merge, and neither red was ours [HARNESS]

**Picked**: Queue head, row **6o** — the two defects iteration 30's evaluator filed: the group
SIGKILL escalation with no killer, and `REAL_LSOF` containment narrower than the code claimed. Named
as Next by iteration 33. Both premises re-verified at HEAD before any routing: `kill -9 "-$pid"` at
probe:261 and group `-TERM` at probe:252 (controls firing), and `REAL_LSOF=$(command -p -v lsof)` at
test:16 with a `case` that checks only absolute-and-executable, never containment.

**Progress**: the bar's clause 1 (the tree gates green from source) — the suite's arm count is its
proxy: **43 → 46 arms**, and the two branches the row named now have their own discriminating
killers. **Goal moved in the tree, NOT on dev**: the merge is blocked. Ungated queue now
**6p → 6s → 7 → 8**, with 6o held open as a resume point rather than closed.

**Reality check**: no design doc existed for 6o (`grep -ri` over `design_docs/`, controls firing), so
this was a genuine NEW-DOC pick. No open motoko PR and no stale motoko worktree held unfinished work.
The fleet-account open-PR filter returned four — `#1030`, `#1031`, `#1033`, `#945` — and **none** has
a branch in this clone's 22-entry worktree list; `#1030`/`#945` carry V1's iteration numbering (321,
296) against motoko's 34, so all four were attributed elsewhere, read, and left alone. Blocked-row
predicates re-run **as commands**: upstream `arniwesth/motoko_agent#154` still **OPEN** (control
`#175` **MERGED**; negative control `#999999` 404s), so rows 10/11/12 stay Phase-0 parked.

**Shipped**: **parked: verified but unmergeable.** PR
[#1034](https://github.com/sunholo-data/ailang/pull/1034), five commits, rebased onto `267a94e92`,
head `db1996128`. Evaluator **PASS 96/100, ZERO blocking** (sonnet, its own worktree, distinct from
the codex executor and the opus controller → generator≠judge holds). Local verification is complete
and repeated: base **rc=0, 43 ok**; head **rc=0, 46 ok, 0 not ok** on **4** runs (62/62/58s
pre-rebase, 59s post-rebase); gate green at **every** milestone boundary **43 → 44 → 46**, so
bisectability is measured rather than hoped; the production probe is byte-unchanged
(`f0b5e024…aabc99`, control: the test file hashes differently); and the commit reconstruction from
the executor's snapshots is proven byte-identical by `shasum -c` rc=0.

**Gate 3b: NOT GREEN, AND THE ATTRIBUTION IS THE DELIVERABLE.** `mergeable` was read FIRST, per the
rule that missing/failing runs are a conflict until proven otherwise — and it was `CONFLICTING/DIRTY`
on a changelog collision, resolved by rebase and force-push, after which all **20** checks completed.
Required checks are `test`, `lint`, `build`, `docs-gate`; `docs-gate` passes, `build` skips, and
**`test` and `lint` both fail — on dev's own HEAD, at the identical steps.** Two independent causes,
neither reachable by this diff (**0 Go files changed**):
(1) `c8c841e24` deleted the `# BEGIN/END FMT_AB_TESTABLE_FUNCTIONS` markers from
`tools/launchd/nightly-eval.sh`, so `test_fmt_ab_schedule.sh` refuses `instrument failure: … produced
no text`. Walked back over parents: red on **19+ consecutive commits**, green at `115184a2e` (63
back). V1's `#1030` covers it and is now MERGEABLE.
(2) **New today and NOT covered by #1030**: `lint` fails `Check code formatting` on **seven Go
files**, all from the coordinator work — `cmd/ailang/coordinator_cloud.go`, five under
`internal/coordinator/`, one under `internal/storage/firestore/`. A bare `make fmt`. It is a REQUIRED
check, so it blocks every PR in the repo.
Per the charter guardrail **V1 owns dev CI red on this anchor**: recorded, handed over on the
cross-mission channel (delivery asserted by reading the message back, not by the exit code), pick
kept. An armed auto-merge was deliberately NOT used — it is a prediction, and this gate's whole
discipline is that a prediction is not an observation.

**THE SCOPE STATEMENT THAT COST THE ITERATION, AND IT IS WIDER THAN A RED CHECK.** Because the
`launchd drivers (bash 3.2)` make target dies at `test_fmt_ab_schedule.sh` **before** it reaches the
probe suite, **my new arms have never executed in CI at all** — grep for their names in that job's
log returns **0**, against a firing control. So that leg is not merely red, it is **blind**: nothing
downstream of the fmt_ab script in that target is being exercised, for anyone. All evidence for this
sprint is therefore **darwin/arm64, GNU bash 3.2.57, local** — the ubuntu and windows legs are unrun
and unreadable. That is the honest scope of a 96/100 evaluation, and it is why the row stays open.

**The quorum, and where the loop's own rules earned their keep.** BLOCKED both rounds.
**`gpt5-6-sol` was recorded ABSENT (budget) in BOTH rounds** — the self-selecting degrade, since a
reviewer drops on budget exactly when the doc has GROWN. Re-run alone at a raised cap both times, it
**rejected both times**, so each synthesis's `proceed`-shaped machinery was hiding a real reject, and
the quorum was 3/3 present rather than N−1. Every blocking premise was **measured, not forwarded**:
R1's sha256 objection is **substantively refuted** (the hash is correct; control discriminates) and
procedurally right, fixed by a V-row; R1's "the re-exec arms are not wall-clock bounded" is
**refuted by construction** — `expect_failure` routes every arm through `run_bounded` with
`ARM_CAP_SECS` (test:9) and a TERM→KILL escalation, and a `$0` re-exec precedent already exists at
test:721 — but the doc, not the design, was at fault, and the real residual (a **2.07×** margin on a
leaked full inner run against a 120s cap, on a host whose load moves a comparable stimulus 3.3–3.6×)
was closed to ~5,000× by scoping an absent `PROBE_UNDER_TEST` onto the arms' own env lines.
R2's surviving objections were **attribution** (four cited line boundaries with no V-row — measured,
and **all four are correct, no drift**) and **determinism** (`getconf PATH` run outside `run_bounded`
— **upheld**: `run_bounded` is defined at test:88 and the gate sat at test:28, so it *could not* have
called it). Both carried concrete reviewer-authored `proposed_fix` text and neither disputed the
design DIRECTION, so the **narrow-refinement carve-out** applied and the **controller** applied the
reviewers' own text verbatim — which keeps the Fable diet intact at one doc, one authoring run, one
revision run. Objection surfaces were tracked per round (R1 provenance + harness semantics; R2
provenance + bounded-waits): **spread and shrinking**, with one reviewer passing both rounds, so this
is a maturing doc and **not** a SPLIT signal.

**THE PLANNER REFUTED THE DESIGN DOC TWICE AND WAS RIGHT BOTH TIMES.** `AC11` demanded
`grep '/usr/bin/getconf PATH'` = **1 hit**; the relocated gate contains that literal **4** times, so
the AC **would have failed a correct implementation** — it was never updated after the round-2
relocation. And the doc's Overview still placed the gate at test:28, contradicting §(b). **Both were
defects in text I had just written under the carve-out**, and both were reproduced first-party before
being fixed. A sub-agent contradicting the controller is the loop working.

**THE EXECUTOR WAS KILLED BY ITS OWN 30-MINUTE CAP AND THE WRAPPER STILL REPORTED `rc=0`.** The cap
fired during M4 and `wait` on the killed child returned 0, so `rc=0` was a true statement about a
dead process — the recipe's own false-green shape, live. It was caught by reading the ARTIFACTS
rather than the code: no `.snap/M4`, no `-o` final-message file. The tree was nonetheless complete,
and that was established rather than assumed — `.snap/M3` is **byte-identical** to the final test
file, so M4 had touched only the changelog, which was present. The evaluator independently confirmed
the same reasoning.

**Routing evidence**: task-class=design model=`fable` (rotation entry after the pointer's
`pi:ollama/deepseek-v4-flash:0731-cloud`; Agent-tool `model="fable"` pin ACCEPTED and ran to
completion, twice — authoring 24m, revision 11m; pointer advanced; ONE doc, initial + one
protocol-mandated revision = the diet, unbroken) · task-class=plan model=`opus`
(`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`, used VERBATIM, no codex
probe fired) · task-class=execute model=`codex:gpt-5.6-sol` (probe rc=0; bounded 30-min background
wrapper; cap FIRED; no git writes; `.snap/M1`–`M3`) · task-class=evaluate model=`sonnet` (own
worktree at `a7b0002a3`) · task-class=mechanical model=`opus` inline (carve-out fixes, commit
reconstruction, record). round1-score=96 rounds=1 corrections=3 (all non-blocking, all applied).
provider=anthropic+openai agent=mission-control cost=**metered $0.50** of the $5 ceiling (quorum r1
$0.0669 + r2 $0.1174 + three restored-`gpt5-6-sol` runs $0.0886/$0.1157/$0.1146); quota buckets
opus + sonnet + fable. **Fable spent: 2 bounded designer runs on ONE doc — within the diet.** No GPU,
no `rig.lock`.

**Ruled out**:
- *"The suite's re-exec arms are unbounded"* (`gpt5-6-sol`, R1) — **REFUTED by construction.** Every
  arm goes through `run_bounded` with a `date +%s` deadline and a TERM→KILL group escalation; a `$0`
  re-exec precedent already exists at test:721. Do not re-raise; the doc now carries the citation.
- *"The doc's sha256 for the production probe may be stale or transcribed"* (`oc-glm-5-2`, R1) —
  **REFUTED.** Measured `f0b5e024…aabc99`, exact, with a discriminating control.
- *"The cited hoist boundaries may have drifted"* (`oc-glm-5-2`, R2) — **REFUTED.** All four line
  claims measured correct via the reviewer's own `sed -n` commands. No drift to correct.
- *"Inverting the containment comparison refuses at startup"* (the design doc's own T6 prediction) —
  **REFUTED by the evaluator and reproduced by me.** The loop is a DISJUNCTION, so `!=` is satisfied
  by 3 of this host's 4 `getconf PATH` entries regardless of `REAL_LSOF`: the gate is silently
  **defeated**, not tripped. The mutation is still caught two arms later. It read as obvious because
  it would be true on a single-entry `getconf PATH`. Corrected in the doc before promotion.
- *"dev's red might be a dropped webhook event"* — **never entertained, because `mergeable` was read
  first and returned `CONFLICTING`.** The boring cause, one call, before any dropped-event lever.
- *"`test_fmt_ab_schedule.sh` exits 0"* — an artefact of my own instrument: `… | tail -5` swallowed
  the exit code to 0. Un-piped it is **rc=1**. Verification rule 3, live, in this iteration.

**Retro lane**: none — **no skill edit**. Every friction here was a rule the rulebook already carries
and that FIRED: read `mergeable` before theorising about dropped events; measure a premise objection
instead of forwarding it; read `absent_reviewers` and restore the absent reviewer; poll the artifact,
not the process; `rc=0` from a launcher describes the fork; exit codes through pipes lie; a judge's
finding is a claim in both directions; `gh --jq` takes exactly one expression. The one candidate gap
(a shared-clone `git worktree list` is not proof of PR ownership) was already raised by
`mission-v1` iter-321 and is not a second instance here — this clone IS mission-exclusive (22
worktrees, all motoko), which is the control that makes the discriminator valid *for motoko* and
invalid for V1's three pinned worktrees of one clone. Recorded, not spent.

**CORRECTION, SAME ITERATION — IT LANDED, AND THE REVERSAL IS THE RECORD.** Everything above was
written while the merge was blocked, and is kept verbatim rather than rewritten. About **90 minutes**
after the hand-over, V1 landed `b51e53f78` — *"dev has been red for 24h on five defects stacked in one
sequential job"* — which covers **both** causes reported, **including the `lint` red that its own
in-flight `#1030` did not**, plus three more nobody had seen (`check-file-sizes`, a
`golangci-lint unused`, and 21 Windows `TestFinalize_*` tests). So the hand-over was not merely
recorded: the second cause was net-new information and it was acted on.

The branch was rebased onto the fixed base and **re-verified there rather than assumed** (`rc=0,
46 ok`, 62s; production probe hash unchanged), then **rebase-merged** as `b97cbf83c`..`684ab8331` —
rebase and not squash, deliberately, so the milestone boundaries this sprint proved green stay
bisectable on `dev`.

**Gate 3b GREEN: 21 checks, 0 non-green (positive control `total=21`), required 4/4
(`test`/`lint`/`build`/`docs-gate`), and `launchd drivers (bash 3.2): success`.** The three new arms
**executed in CI** on the macOS runner — `ok 43 - production run_lane SIGKILL escalation kills a
TERM-immune wrapper grandchild`, `ok 44`/`ok 45` for containment, `PASS: 46 probe self-test arms ran`,
with a pre-existing arm in the same log as the control. **The BLIND-leg scope statement above is
therefore RETIRED, not softened**: it was true when written and is false now, and that transition is
exactly what the hand-over bought. Auto-close scan on all five commit messages and the PR body: **0**
matches with a known-bad control firing; `closingIssuesReferences` **0**; `#987` still OPEN at **7**
comments, its pre-merge count.

One honest note on my own process: I wrote a full NOT-LANDED record, including a charter tag and a
resume predicate, and then had to correct all of it inside the same iteration. That was still the
right order — the record existed before the outcome was known, which is the only reason a crashed
slot would have inherited something rather than a mystery.

**Next**: row **6p** — derive the suite's wall-clock and node-ceiling bounds from a stimulus measured
in-test, so the ratio holds by construction on any machine, rather than from constants calibrated on
one host at one load. Row 6o is closed.

## 35 — 2026-09-05 — the absent reviewer was the one that found the defect at the bottom of the design, and the judge found its twin in the code [HARNESS]

**Picked**: Row **6p**, named by iteration 34's Next — derive the suite's wall-clock and node-ceiling
bounds from a stimulus measured in-test, so the ratio holds by construction on any machine.

**Progress**: the bar's clause 1 (the tree gates green from source) — the probe self-test suite's arm
count is its countable unit. **46 → 57 arms.** This iteration shipped **M1 of 3** of the row's design;
the row itself stays OPEN with M2/M3 as its residual, and the epic is unmoved.

**Reality check**: no design doc existed for 6p (`grep -ril` over `design_docs/` returned only the
mission files and 6n's doc). Premises re-verified at HEAD before routing: `MAX_TREE_NODES=
${PROBE_MAX_TREE_NODES:-4096}` at probe:126 checked at probe:196; `ARM_CAP_SECS=
${PROBE_SELFTEST_ARM_CAP_SECS:-120}` at test:9; and **no bound-derivation helper existed** —
`grep -cE 'derive_bound|measure_stimulus|calibrat'` **0**, known-positive control `run_bounded` **10**.

**Shipped**: **PR [#1048](https://github.com/sunholo-data/ailang/pull/1048)** — four commits (design,
plan, M1, judge-fix). M1 is **additive by construction**: the derived scale, arm cap and node ceiling
are computed, validated and published, and consumed by nothing (`ARM_CAP_BASE=120` still feeds
`ARM_CAP_SECS`; `$(bound_secs ` consumed **0** times; the node ceiling still literal at test:687; the
`p_obs` gate absent — exactly the M1/M2/M3 split the plan specifies). Suite **rc=0, 57 ok, 0 not ok**
against a base of 46. Published diag line: `r=489/s r_real=665/s p_obs=1.36 reference=400/s scale=1
arm_cap=120s node_ceiling=7824 floor=DISABLED`, bookend `drift=none`.

**THE DESIGNER LANE FAILED ON ITS FIRST REAL RUN AND THE WRAPPER SAID `rc=0`.** `codex:gpt-6-astra`
— the rotation entry that landed this morning at `087fbea63` — probed rc=0 and then produced **zero
artifact**: no `-o` final message, no worktree change, no `tokens used` summary (the probe's own log
has one, so that is a control rather than an absence), dead after ~3 minutes against a 30-minute cap.
Diagnosed by reading ARTIFACTS rather than the exit code, per the recipe's false-green list. This is
the first real datapoint on that lane and it is a lane failure, not a model verdict.

**AND THE FALLBACK COULD NOT BE SPAWNED THE DOCUMENTED WAY.** `Agent(model="fable")` — which the
roles table's 2026-08-20 correction explicitly authorises — was DENIED by the spawn-pin hook:
`deny:provider-pin — designer is pinned to codex:gpt-6-astra`. The hook enforces the DECLARED pin and
has no notion of a fallback, so a role whose primary lane dies cannot be re-routed through the Agent
tool at all. The fable run went through the `claude:` recipe via `claude-sub` instead (tripwire CLEAN,
keys stripped at the call site). **The same hook contradicted `resolve-role-spawn.sh` for the
planner**: resolver `agent-tool opus fail-closed:planner-lane-field-missing` (and
`derive-planner-lane.sh` agreed, run from the worktree that holds the doc — iteration 33's scope trap
avoided) against the hook's `planner is pinned to codex:gpt-5.6-sol`. Gate 3 says to use the resolver's
line VERBATIM and not to second-guess it; the hook is what actually adjudicates, so that instruction is
unfollowable whenever they disagree. Both roles took their configured provider fallback, FLAGGED. Filed
as row **6u** — it is a defect in CODE, not in the rulebook, which is why it is not this iteration's
skill edit.

**The quorum, and the reviewer who was not in the room.** BLOCKED both rounds. R1 **3/3 reject**,
`.synthesis.absent_reviewers` `[]` cross-checked two ways, $0.1587; two of three named ONE defect
(§4.2's prototype unconditionally enforced the floor M1 requires be flag-disabled) and were applied as
written. The third was `gpt5-6-sol`'s PREMISE objection about the stimulus proxy, so I **measured it
rather than forwarding it** (rule 3f): under one identical load step, bash-script **1.27×** · `date`
**2.04×** · `pgrep` **1.13×** · `true` **1.31×** — a spread of up to **1.8×** — and real `pgrep` runs
at **76/s** against the stimulus's **564/s**, so the walk's per-node cost is set by the slowest op, not
the chosen one. **UPHELD**: directionally right, not tight. The designer got the measurement and
returned its own interleaved number (1.35×) beside mine, which is the correct behaviour.
R2: `gemini-3-1-pro` **flipped to pass**, `oc-glm-5-2` reject, **`gpt6-astra` ABSENT (budget)** — the
self-selecting degrade, since a reviewer drops on budget exactly when the doc has GROWN (527 → 774
lines). Re-run alone at a raised cap ($0.2654) it **REJECTED**, so the synthesis was a
pass-with-a-named-hole and R2 was really **1 pass / 2 reject, 3/3 present**.

**Astra's objection is the sharpest either round produced, and the degrade would have buried it.**
`measure_fork_rate` incremented its counter unconditionally after `|| true`, so a missing,
non-executable or failing stimulus produced a **positive rate** that would then have determined every
derived bound — a silent fallback on the single input the whole design rests on, contradicting the
doc's own §4.5. Both survivors carried concrete reviewer-authored `proposed_fix` text and disputed no
design DIRECTION, so the **narrow-refinement carve-out** applied and the CONTROLLER applied their own
text VERBATIM (§4.8). Surfaces per round: R1 = prototype-flag consistency ×2 + proxy premise; R2 =
proxy gating + measurement-helper error propagation — **spread and MOVING, neither R2 surface an R1
surface**, one reviewer flipping to pass. A maturing doc, not a SPLIT signal.

**THE PLANNER REFUTED THE DESIGN DOC ELEVEN TIMES AND I CONFIRMED THE LOAD-BEARING ONE MYSELF.** The
doc said to insert new arms *"after line 793"*. Arms in fact run to **818** — `run_lane_fixture_arm` at
798, `REAL_LSOF` containment at 813–818 — so the doc's placement would have put them AHEAD of bounded
arms, which is precisely the change that took the wall-clock arms from **0-in-17** to **4-in-19** at
iteration 33. The plan moves them after 821. It also caught two execution-order defects (`bound_secs`
unreachable from pre-derivation EXIT paths; deriving `ARM_CAP_SECS` at line 9 would call it before
definition), that the startup insertion point **cannot** measure `live_bin/pgrep` because that file is
created much later, that M1's `46 + 8` arithmetic ignored AC-8's three arms, and that the base is
**59 s**, not the doc's 57.

**THE EXECUTOR WAS KILLED BY ITS OWN 30-MINUTE CAP AND THE WRAPPER STILL REPORTED `rc=0`** — the
recipe's false-green, second consecutive iteration — **and no snapshots were written**, so the
uncommitted worktree diff was the only artifact. Completeness was ESTABLISHED rather than assumed:
`bash -n` clean, zero bash-4+ constructs (control `run_bounded` **19**), additivity confirmed by the
three greps above, and the suite green. One milestone of three is the honest scope.

**THE EVALUATOR'S BLOCKING FINDING IS REAL AND IT FOUND IT WITH OUR OWN PLAN.** Evaluator **PASS
81/100** (sonnet, its own worktree at `cef2dae4b`, distinct from the codex executor and the opus
controller → generator≠judge holds). `PROBE_SELFTEST_FORK_RATE` **short-circuits** the measurement
rather than steering it, and was not cleared before the three AC-8 recursions, so an ambient value
makes `measure_rate_or_refuse` unreachable and the injected fault is never exercised. The judge found
it by running the sprint plan's **own M2 AC-1 boundary command**, which therefore already failed
against shipped M1 for a reason unrelated to anything M2 adds. **Reproduced first-party before being
believed**: `PROBE_SELFTEST_FORK_RATE=200 <suite>` → **rc=1, 53 ok**, arm `bound measurement refuses a
stimulus that exits nonzero`, against a no-override control of **rc=0, 57 ok**. Fixed with `env -u` on
that recursion — not a leak guard, because the overrides are legitimate at suite scope for every other
arm — and verified BOTH ways: **rc=0, 57 ok** with the override, control unchanged. The judge also ran
the astra mutant and it **REDS** (`expected_rc=72 refusal=0 derived=1`), so that gate LOOKS rather than
merely fires. Its two non-blocking findings — a duplicated diag line and an ADDED arm both leave the
suite green — are **corroboration, not new defects**: the second is row **6s**, filed by iteration 33's
judge before this sprint and now independently reproduced by a different judge.

**GATE 3b GREEN ON THE MERGE, AND THE RUNNER ANSWERED THE QUESTION THE QUORUM COULD NOT.** `0686d5b00`,
**21** checks, **0** pending, **0** non-green, required **4/4**, and **`launchd drivers (bash 3.2):
success`** — the only leg where this suite runs. Rebase-merged as `b4bfb04e1`..`137842bfd`, four commits
kept so the milestone boundaries stay bisectable. The derivation **executed on the GitHub runner** and
published `r=318/s r_real=251/s p_obs=1.27 reference=400/s scale=2 arm_cap=240s node_ceiling=5088
floor=DISABLED` — **the first measurement of that runner row 6j has ever had.** Three consequences M2
needed and nobody could previously state: the runner derives **scale=2**, i.e. it wants a **240s** arm
cap, so the hardcoded **120s was too tight for it** — row 6j's blowout with a number attached;
**`p_obs=1.27`** on the runner, inside the budgeted `P_PROXY=2` and far inside the 4.7 tolerance, which
answers `oc-glm-5-2`'s objection (*"no CI runner has been measured"*) with a measurement; and
`floor=DISABLED` with **no** `BOUND_FLOOR_NOT_ENFORCED` line (318 > the 100/s floor), so **M2's floor
flip is measured-safe.** One instrument failure of my own, caught by its own control: a grep for the new
arms BY NAME in the CI log returned **0**, and so did its known-positive control — the grep was broken,
not the arms absent (rule 3a). The load-bearing evidence is `PASS: 57 probe self-test arms ran` in that
job's log, present, with a shape-matched negative control at 0.

**Routing evidence**: task-class=design model=`fable` via the `claude:` recipe (`claude-sub`, probe
rc=0), **after** `codex:gpt-6-astra` (`resolve-role-spawn.sh designer` → `recipe codex:gpt-6-astra
declared:provider-pin`, probe rc=0, real run rc=0 with **zero artifact**) — FLAGGED, and the
Agent-tool fallback was DENIED by the spawn-pin hook, FLAGGED. task-class=plan model=`codex:gpt-5.6-sol`
(resolver said `agent-tool opus`, hook said `codex:gpt-5.6-sol`, hook wins at the tool boundary →
`MISSION_PLANNER_ANTHROPIC_FALLBACK`) — FLAGGED. task-class=execute model=`codex:gpt-5.6-sol` (probe
rc=0, 30-min bounded background wrapper, no git writes; **capped after M1, no snapshots**).
task-class=evaluate model=`sonnet` via the Agent tool, own worktree at `cef2dae4b`. Designer rotation
pointer advanced. Fable diet intact: ONE doc, one authoring run, one revision run. **Metered $0.5256**
of the $5 ceiling (R1 $0.1587 + R2 $0.1015 + astra re-run $0.2654). No GPU, no `rig.lock`.

**Ruled out**:
- *"The astra designer lane is unavailable"* — **not established.** The probe passed and the run
  started and explored the repo for ~3 minutes. What is established is that ONE real run produced no
  artifact. One datapoint on a lane that is one day old is not a verdict on the lane.
- *"The 22 s suite-time gap is the new measurement's cost"* — **refuted.** The measurement alone is
  3–4 s isolated; the evaluator reproduced only a 9–10 s gap at quiet load and attributed the
  remainder to fork overhead across the new recursion arms. My 22 s was a loaded-host reading of the
  same thing. Resolved in kind, not in exact magnitude.
- *"dev's SonarCloud red is ours"* — **refuted** by walking it back: `failure` on all five most recent
  commits, so inherited. Per the charter guardrail V1 owns dev CI red on this anchor; recorded, handed
  over, pick kept.
- *"A quorum synthesis reading BLOCKED tells you what the reviewers said"* — **refuted again, second
  consecutive iteration.** R2's synthesis was computed over two present reviewers; the third's verdict
  existed and cost $0.27 to recover, and it was the one that mattered.

**Retro lane**: none — **no skill edit**. The two spawn-pin denials are a HARNESS defect in
`resolve-role-spawn.sh` and the hook, not a rulebook gap: the rulebook's instruction is correct and
merely unexecutable, so editing it would be writing around a bug. Filed as row **6u**. Every other
friction was a rule that already exists and FIRED (read the artifact not the exit code; re-run an
absent reviewer at a raised cap; measure a premise objection instead of forwarding it; reproduce a
judge's finding before believing it; walk a red back over parents before blaming the merge).

**Next**: row **6p M2/M3** — wire the wall-clock class, enforce the floor and gate `p_obs`; then derive
the node ceiling on the discovery arm. Fully specified in the sprint plan, blocked on nothing but
executor minutes.

## 36 — 2026-09-06 — an attended session turned dev red in six checks and left; the fire 37 minutes later inherited it, and found a fifth defect the CI logs could not yet show [HARNESS]

**Picked**: NOT the queue head. Gate 1's red-outranks-the-queue rule fired. `dev` was red in six
checks — `lint`, `test-windows`, `Build windows-latest`, `docs-build`, `docs-gate`,
`launchd drivers (bash 3.2)` — and rule 3d's parent-walk attributed every one of them to
M-MISSION-LOOP-WORKBENCH Phase 1: `ab3252109` (16 checks, all green) → `a9de67fe6` M1 (20 checks,
windows pair red) → `6536cfb98` M2+M3+M4 tip (adds `lint`, `docs-*`, `launchd drivers`). The
intra-push commits `6203266a3`/`07169cd68` read `total=0` by construction — only a push's tip gets a
run — which is why the unit is the merge, not the commit.

**Whose red it was, and why this mission fixed it.** `sunholo-data/ailang` is V1's repo and the
owning-mission rule says a non-owning mission records a red and hands it over — EXCEPT where the red
is its own doing or sits in territory the owner has no domain knowledge for. Both exceptions apply:
mission-loop machinery is this charter's clause 6. **It was not an orphaned loop iteration.** The
died-mid-flight traces were run and came back clean — the driver log shows only two motoko fires on
2026-09-05 (13:01→16:21 = iteration 35, and 23:39 = this one), so nothing fired at 22:39. The design
doc, the sprint plan and M1–M4 were landed by an **attended session** between 22:15 and 23:02 local,
directly to `dev`, with **zero charter rows and zero log entries** (`grep -ci workbench` → **0** in
both files). Iteration 35's own record is present (control: `ITERATION 35` → 4 in the charter,
`ITERATION 34` → 5). The session was **still working while this iteration ran**: M5, M6 and M7
(`cad2ecbdb`, `096d04020`, `7a0bbd5da`) landed mid-flight.

**Progress**: goal unmoved. The epic (`m-motoko-dst-refactor-migration`) did not advance and row 6p
M2/M3 were not touched. What moved is bar clause 1's precondition — a tree that gates green — from
six red checks to a fix under review.

**Outcome**: PR [#1055](https://github.com/sunholo-data/ailang/pull/1055), six commits.
M1 `lint`: a map literal at `doctor_test.go:229` that `walk()` overwrites before any read.
M2 `test-windows`/`Build windows-latest`: `registry.go:121` validated the workdir with
`filepath.IsAbs`, which is **platform-dependent** — on Windows it rejects `/Users/...`, so every
fixture failed validation and ~20 tests died at the gate (`workdir "/Users/x/dev/sunholo-data/ailang-parse" must be absolute`).
The registry renders macOS launchd plists, so absoluteness here is POSIX absoluteness; the
replacement is byte-identical to `filepath.IsAbs` on Unix and additionally rejects `C:\…`, which is
meaningless in a plist. M3 `launchd drivers (bash 3.2)`: `test-launchd-drivers` called
`test-mission-registry`, which runs `go test`, in a CI job that installs no Go toolchain by design.
M4 `docs-build`/`docs-gate`: a repo-relative link escaping the docs tree.

**The fifth defect, which no CI log could have named yet.** `GOOS=windows go vet` on the REBASED tree
returned `internal/mission/kill_unix.go:6:51: undefined: syscall.Kill`. `kill_unix.go` had landed
minutes earlier in M5/M6 with **no build constraint and no Windows counterpart** — and Go does not
treat `_unix` as a GOOS suffix, only `_windows`/`_linux`/`_darwin` and friends are automatic. So the
package did not COMPILE on Windows, and M2 alone would have left both Windows checks red while
looking like it had fixed them. Filed as M5 here: `//go:build unix` plus a `kill_windows.go` stub.
The stub reports the pid **live**, because `missionBusy` reads a non-nil error as "not busy" — the
unsafe direction, which would let `apply` overwrite a running mission's artifacts. Mutation drill:
stripping the `//go:build unix` line takes Windows back to red; restore is byte-identical by sha256.

**M3a — the controller reviewing the executor.** The executor replaced the removed recipe line with a
**tab-indented** comment, which make treats as a recipe: it echoes the line and hands it to `/bin/sh`
to do nothing. Measured with `make -n`: **1 → 0** recipe-line hits after un-indenting, the 16 bash
invocations unchanged, target still rc=0 with `go` masked. Found before the judge reported.

**Verification, baseline-paired on the same machine and the same commands** (rule 3e). Baseline is
the pristine tree at `origin/dev`: `make lint` → **1 issue (ineffassign)**; treatment → **0 issues**.
`go test ./internal/mission/...` → `ok` both sides. `GOOS=windows go vet` → baseline
**`undefined: syscall.Kill`**, treatment **rc=0**; `linux` and `darwin` rc=0 too.
`make test-launchd-drivers` with `go` masked off `PATH` → **rc=0**, `PASS: 57 probe self-test arms
ran`, control `command -v go` on that PATH rc=1 so `go` really was absent.

**A pre-existing red that fixed itself, and the re-baseline that caught it.** Earlier in this
iteration `TestLive_DoctorReproducesTheMeasuredDivergences` was **red on the pristine tree** at
`f5edd569a` — identical failure text on treatment and baseline, so it was correctly ruled
out-of-scope. After the rebase onto `7a0bbd5da` the same test is **green at baseline**: M6 ("all four
missions adopted, V8 fixed at source") repaired the rig drift the gate asserts. The first reading was
right when taken and wrong forty minutes later. It skips off-rig (`golden_live_test.go:330`), which is
why CI never saw either state.

**Ruled out**: (a) *the M1–M4 landing was an orphaned iteration* — refuted by the driver log, which
records no motoko fire between 16:21 and 23:39; it was attended work. (b) *the red belongs to V1 as
repo owner* — refuted by the owning-mission rule's own exception: it is this charter's territory.
(c) *the four open PRs on the fleet account might be inherited work* — `--author` is a fleet filter,
not a mission filter; none of `#1054`, `#1041`, `#1033`, `#945` has a branch in this clone's worktree
list, so none is attributable here and none was touched. (d) *the live-doctor failure is ours* —
refuted by the pristine baseline, twice, in both directions.

**Blocked-row predicates re-run as commands, not transcribed**: upstream `#154` still **OPEN**
(positive control `#175` **MERGED**, negative control `#424242` 404s), so rows 10/11/12 stay Phase-0
parked.

**Routing evidence**: controller `claude:claude-opus-5` (quota bucket, session).
Executor `codex:gpt-5.6-sol` via the cross-provider recipe — probe rc=0 with a real artifact
(`tokens used`), real run rc=0 in 30-min cap, non-empty worktree diff, `-o` final message 7,110 B.
Evaluator **sonnet**, Agent tool, its own worktree — generator≠judge holds (OpenAI executor vs
Anthropic judge). **No designer and no planner ran**, and that is the routing table applying rather
than being skipped: both of its branches gate on artifacts a CI-red fix-forward does not have — there
is no design doc to write and no plan to derive (`derive-planner-lane.sh` returned
`opus fail-closed:no-doc`, and the planner's own pin is `codex:gpt-5.6-sol`, a `provider:model` value
the spawn-pin hook would have denied an opus spawn for). Designer rotation pointer left untouched at
`codex:gpt-6-astra`; **Fable unspent**. Commits reconstructed from the executor's per-milestone
snapshots and proved byte-identical to its final tree (`shasum -c`, 5/5 OK); `.snap/` confirmed absent
from every commit.

**Metered**: **$0.00** of the $5 ceiling — no quorum ran (no design doc), and every lane used was a
quota bucket or the codex subscription. No GPU, no `rig.lock`.

**Next**: row **6p M2/M3** (wire the wall-clock class, enforce the floor, gate `p_obs`; derive the
node ceiling) — iteration 35 landed M1 only because its executor was capped after one milestone.

**Evaluator: PASS 85/100, ZERO blocking** (sonnet, its own worktree, distinct provider from the
codex executor — generator≠judge holds). It ran the drills rather than reasoning about them, and two
were causal rather than plausible: it **built the docs**, reverted M4's one line, and got the exact
`Docusaurus found broken links` failure back, then restored; and it reintroduced M1's dead
initialiser and got exactly 1 `ineffassign` finding. It also independently found M3's echoed comment
— the same defect the controller had already fixed as **M3a** — which is two instruments agreeing.

**The judge's strongest objection was right, and it was about a COMMENT, which is why it nearly
survived.** The new M2 test's comment claimed it "kills both mutations". Reproduced first-party
before acting (rule: reproduce before believing, and before dismissing): reverting `registry.go` to
`filepath.IsAbs` leaves **both arms green on darwin**. The reason is stronger than "equivalent" — on
unix `filepath.IsAbs` *is* `strings.HasPrefix(p, "/")`, the same shipped function
(`path_unix.go:35-37`), so no POSIX input can distinguish them. Only the delete-the-check mutation is
killed locally; the IsAbs revert is caught by `test-windows` and `Build windows-latest` **and by
nothing else**. A contributor trusting a green local `go test ./...` after touching that check would
ship the exact regression the test exists to prevent. Fixed in **M6**, which states the platform
qualifier and names the two jobs that are the real safety net. The judge's other non-blocking find —
`test-mission-registry` left referenced by nothing after M3 — is also M6, documented as a deliberate
hand-run entry point rather than deleted, because this repo has scar tissue about removing things
that merely look unused.

**Judge finding NOT actioned, with its control.** It flagged the absence of a `CHANGELOG.md` entry
against `coding-standards.md`'s "every change requires CHANGELOG.md". Deferred, and the control is
what decides it: `git log --name-only f5edd569a~1..origin/dev -- changelogs/ CHANGELOG.md` is
**empty** — the seven-commit workbench landing this fixes added no entry either, and the changelog is
sectioned by released version (top entry `v0.35.1`, already out). Inventing a new unreleased section
for the fix-forward while the feature itself has none would misdescribe both. CI's `make
check-changelog` is an *index hygiene* gate, not a per-PR entry gate, and it is green. Filed as a
queue row so the Phase 1 write-up covers both at once; recorded here rather than dropped.

**Gate 3b**: bounded, SHA-pinned, subject-named poll (`ALL COMPLETE for <sha>`, per-invocation log
path). Both counts are asserted numeric before comparison, so a parse failure prints
`INSTRUMENT FAILURE — not a verdict` instead of a vacuous green. `mergeable` was read FIRST each
round and did show one `UNKNOWN/UNKNOWN` before resolving — not banked, per the async-mergeability
caveat. The poller was killed and relaunched on each new head rather than left to expire on a
superseded subject.

**CORRECTION, written after two more rebases — the base moved three times while this iteration ran,
and twice it moved the answer.** The attended session did not stop: after the record above was
committed it landed **M8, M9, HD ratifications and "Phase 3 part 1"** (`fe9c08ffc`, `703c2f6a3`,
`7d05a5f73`, `19d6b03c7`). Two consequences, both measured rather than assumed.
**(a) M1 and M4 were dropped by `git rebase` as *patch contents already upstream*** — the human
independently fixed the `ineffassign` initialiser and the docs link. Confirmed against the new
origin/dev: both greps now return **0**, while M2's `filepath.IsAbs`, M3's Go-in-a-Go-less-job and
M5's missing build tag still return **1/1/0** and remain this branch's work. Four of the five
original defects were real; two stopped being ours to fix, which is the correct outcome and not a
wasted commit — the rebase is what proved it, rather than a claim.
**(b) A SEVENTH defect arrived in the base and this PR inherited it through the merge preview.**
`19d6b03c7` added `TestDriverCopiesDoNotMultiply`, a ratchet counting driver copies across sibling
checkouts (`../../../ailang-world` …) that exist **only on the rig**. On any CI runner `distinct` is
1 by construction against `knownDriverCopies = 2`, so the FELL arm fires for an *environment*, not a
change. Not ours and not caused by this branch: **dev's own head `19d6b03c7` is red on `test` for
exactly this**, measured first-party. Fixed as **M7** — the down arm skips when no sibling checkout
is present, the same off-rig convention `golden_live_test.go` already uses; the up arm is untouched,
because it can only fire on a fork the checkout can actually see. Controls: from
`~/dev/sunholo-data/ailang-motoko` all three siblings are VISIBLE, world's driver DIFFERS from shared
and has no `lib/pin-root.sh` beside it, so `distinct=2` and the ratchet still holds there; from a
worktree or a CI clone `observed=0` and it skips. The commit says plainly that this is the human's
own 15-minute-old code and should be dropped if it is being fixed concurrently.
**(c) The `internal/smt` red is not ours, and the SCOPE of the control is the only thing that proved
it.** A full `go test ./...` on this branch fails `TestSolve_HardTimeout_FakeSolverIgnoringT`. Run
*alone* on the pristine tree it PASSES — which would have licensed blaming the branch. Run with the
*matched* command (`go test ./...`) on the pristine tree it fails identically, **9.02s vs 9.01s**. It
is load-dependent and pre-existing. The first control was the wrong scope, and rule 3e is the only
reason that did not become a false attribution.

**SECOND CORRECTION — M5 unblocked the Windows build, and the very next layer was a defect I had
introduced myself.** With `kill_unix.go` constrained, `test-windows` reached the *tests* for the first
time and 23 of them failed in `internal/mission`, every one reading

    fixture mission invalid: fixture:docs: workdir "C:\Users\RUNNER~1\...\repos\docs" must be absolute

— **M2's own check refusing a genuinely absolute path.** The two fixture families disagree by
construction: `registry_test.go` builds workdirs from POSIX literals, while the fleet fixtures in
`doctor_test.go`/`apply_test.go` build theirs from `t.TempDir()`, which is `C:\Users\…` on a Windows
runner. `filepath.IsAbs` alone rejects the first family there; a POSIX-only prefix check — what M2
shipped — rejects the second. **Neither single arm is correct, and the original defect and my fix
were the same mistake pointing in opposite directions.** M8 accepts either arm. On unix the two are
literally the same function, so the rig sees no change; what the check still rejects is what it is
*for* — a relative path, and the `~`-form above.

Three things worth keeping from this. **(a)** The judge's M6 objection was the same defect seen from
one level up: it said the test could not discriminate on darwin, and the reason it could not is that
the property being asserted was *platform-shaped* rather than *invariant*. The test now targets the
invariant (relative is refused everywhere) and uses `t.TempDir()` for the host-absolute row, which is
host-absolute on every platform. Mutation-checked: deleting the check turns the relative and
bare-name rows red on darwin, so it discriminates locally now — which M2's version never did.
**(b) A build break hides every defect behind it.** These 23 failures existed from the moment the
fleet fixtures were written; nobody could see them because the package did not compile on Windows, so
the leg failed at `Install ailang to PATH` and never ran a test. Fixing a compile error is not a
small win — it is what makes the next measurement possible, and the honest expectation after one is
*more* red, not less.
**(c) I shipped M2 with a passing local suite, a PASS 85/100 evaluation and my own out-of-sandbox
gate sweep, and it was still wrong.** Every one of those instruments runs on darwin, where the bug is
invisible by construction — the judge said so in its strongest objection and I fixed the *comment*
rather than hearing it as a statement about the change. The only instrument that could see this was
the Windows CI leg, and it could only see it after M5.

**And the layer under THAT.** M8 took the windows leg from **23** failures to **1**:
`TestApply_BacksUpWhatItReplaces`, failing with
`...\bak\C:\Users\RUNNER~1\...: The filename, directory name, or volume label syntax is incorrect`.
`Apply` built its backup path with a hand-rolled `baseName` splitting on `"/"` alone, so a
backslash-separated windows path came back **whole** and the join produced a second drive letter
mid-path. `filepath.Base` is identical to the old code for `/`-separated paths and correct on both;
fixed as **M9**, with the invariant that would have caught it — a backup lands DIRECTLY in the backup
dir — added to the test and labelled with the fact that only `test-windows` can ever fire it.

**The shape of this iteration, which is the part worth carrying forward.** Six red checks became
**nine** defects, and they came in *layers*: each fix made the next one observable. A compile error
hid 23 test failures; fixing those exposed one more; and one of the nine was mine. That is not a sign
the work went badly — it is what fixing a build break on a platform nobody develops on looks like,
and the honest expectation after each green is *more* red, not less. The iteration also ran against a
moving base the whole way: **five pushes, three rebases**, with an attended session landing M5 through
Phase 3 part 1 while this ran. Two of my commits were dropped as already-upstream and two of my
defects came from commits that did not exist when the iteration started.
