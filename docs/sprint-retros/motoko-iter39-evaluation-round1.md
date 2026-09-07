# Evaluator report — motoko iteration 39 (row 6s, m-motoko-suite-arm-count-floor, round 3)

**Worktree**: `/Users/voightkampff/.ailang-driver-pin/.wt-motoko-iter39-eval`, detached at `f6750002d`.
Confirmed clean throughout (`git status --porcelain` empty before and after; no file in the tree was
ever edited — all reproduction work ran against copies in `/tmp/iter39_verify/scratch`). HEAD and doc
sha256 unchanged: `3900de022d2cb14db5e635f8e60f4df2046517ed317419db3567347199d95021`.

## Score: 90/100 (PASS, threshold 70)

## Summary verdict

The audit is real, not decorative. Every load-bearing count in `/tmp/audit_iter39_handoff.md` and
the doc's Conflict Surface / Verification Log was re-run first-party in my own worktree, from base
`878939117`, and matched exactly — including two rows I ran that the doc did not (a live self-arm
deletion mutant and a live addition mutant against the actual sprint file). Precondition P1 is real
and correctly bounded. glm's remedy 2 is applied verbatim (byte-for-byte against its round-2
`proposed_fix` text). Nothing but the design doc landed. The controller's PARK decision on P2 is the
right routing call — but for a narrower reason than the one it gave, which is item 6 below.

## Item-by-item

### 1. Did the audit happen, and is the doc faithful to it? — CONFIRMED, no blocking gaps

Reproduced independently, all from base `878939117` (`git show 878939117:... > base.sh`), not from
the doc's numbers:

| Claim | My command | My result | Doc/audit claim |
|---|---|---|---|
| `$0`/`BASH_SOURCE` hits | `grep -nE '\$0\|BASH_SOURCE' base.sh` | 18 hits, line 4 is a path alias, 14 are `/bin/bash "$0"` re-execs, 4 are the census | 18 / 14 / 4 — match |
| `exit 0` sites | `grep -nE '^[[:space:]]*exit 0' base.sh` | exactly 183, 204, 354 | match |
| Line 204 is inside a heredoc | `sed -n '196,205p' base.sh` | confirmed: `cat > "$canonical_pgrep" <<'EOF' ... exit 0 ... EOF` — a fixture body being written to a file, not suite control flow | match, and I verified the heredoc boundary myself rather than taking the audit's word for it |
| Anti-recursion refusals | `grep -n 'refusing to recurse' base.sh` | 988, 992, 996 | match |
| Sourced files | `grep -nE '^[[:space:]]*(source\|\.)[[:space:]]+' base.sh` (rc=1) + control `grep -c source` = 0 | 0 hits, control also 0 (the doc's control, "73 for 'probe'", is a *different* control string but the rule-3a discipline is satisfied either way) | match |
| External readers | `grep -rl 'test_motoko_connection_probe' .` minus self | 25 | match; negative control `zzq_no_such_suite_iter39` returns **one** hit now — the design doc itself, which quotes the string as documentation. This is expected (the string didn't exist in the repo when the audit ran) and not a defect. |
| Glob gates | `check-file-sizes` scope, `AUTOPUSH_SHELL_SCRIPTS`, `scripts/check_*.sh` count | Go-only/`internal`+`cmd`, two named files, 13 scripts | match |
| CI job scope | `grep -n 'test-launchd-drivers\|runs-on:' .github/workflows/ci.yml` | line 602 sits between `runs-on: macos-latest` (589) and the next `runs-on:` (606) — only job that reaches the suite | match |
| Proposed gate matches 0 of 7 glob patterns | saved the gate block to a file, ran all 7 `grep -Ec` patterns | all 0; control `expected_arms` = 4 | match |
| Container runtimes | `for c in docker podman colima lima nerdctl orb git; do command -v ...; done; docker info` | all six absent, `git` present, `docker info` rc=127 | match |
| `pass_arm` counts | `grep -cw pass_arm` on base/HEAD, `grep -n 'pass_arm ' \| grep -vc ':#'` | base 19 (18 calls + 1 def), HEAD 21 bare / **19 executable** | doc's criterion-5 number (19) reproduced exactly |

No row reported an empty/negative result without a same-call positive control except the one I
flagged above (self-referential negative control), which is a measurement-timing artifact, not a
gap in the audit's rigor.

### 2. Is P1 real and correctly bounded? — YES, verified independently, no blocking issues

I did not just read the audit's grep outputs — I ran a **live child-process test** the audit itself
does not include: I ran the pristine base suite (see below) and confirmed the single `UNINFORMATIVE`
line and zero recursion, then separately confirmed via the anti-recursion refusal lines (988/992/996)
that a leaked sub-mode variable is a loud `exit 1`, not a silent fall-through. The three terminators
(12-14 exit 1, 181-183 exit 0, 353-354 exit 0) are the only three `exit 0`/early `exit 1` sites in
the file outside the heredoc, and I traced each surrounding block myself. P1 is real, correctly
scoped to "no child sub-mode reaches the tail today," and correctly phrased as a *precondition* a
future sub-mode could break — not as a permanent guarantee.

### 3. Is glm's remedy applied verbatim? — YES

R2 artifact (`.../m-motoko-suite-arm-count-floor-2026-09-07T06-56-41Z.json`), glm's `proposed_fix`
text: *"...or mark the non-Darwin path as UNVERIFIED and gate it behind an explicit `if [[ "$host_os"
== Darwin ]]; then ... else echo 'UNVERIFIED host: arm-count gate skipped' >&2; exit 1; fi` to avoid
a silent false-red on an unmeasured platform."* The doc's shipped block is character-for-character
that `else` clause. Verbatim, confirmed by direct text comparison, not by trusting the doc's own
characterization of it.

Container-runtime unavailability is independently confirmed (item 1 table). No Linux host reachable
from this rig — real, not asserted.

**One precision the doc should be more explicit about**: M1b (swapping the *actual shipped* `else
expected_arms=56 ...` in `tools/eval/test_motoko_connection_probe.sh` for glm's verbatim refusal) has
**not been applied to the code** — only to the doc's prose/code-block. I confirmed this by reading
`tools/eval/test_motoko_connection_probe.sh` at HEAD: the `else` branch still reads
`expected_arms=56 # 60 - 4 Darwin-only arms ...`, i.e. the round-2 (ruled-out) shell, because M1b is
explicitly listed as a separate, not-yet-landed, "independently committable" milestone. The doc *is*
honest about this (Milestones section, and the Verification Log row for the refusal explicitly says
"scratch copy of the sprint file with the round-3 gate") — I flag it only so nobody mistakes "the
design doc shows glm's remedy" for "the repo now runs glm's remedy." **Non-blocking** — the doc does
not claim otherwise and C5's claim ("did NOT land the code") is correct either way.

### 4. P2 — reproduced, and the conflict question (the highest-value item)

**Reproduced first-party**, not from the doc's numbers:
- Pristine base run (878939117, my own copy, own execution): `PASS: 59 probe self-test arms ran`,
  one `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer` line, zero `not ok`
  lines. 71s wall time.
- Deletion mutant (line 1184, the doc's own chosen instrument): `PASS: 58 probe self-test arms ran`,
  zero `not ok` lines — **the defect the whole design exists to fix, reproduced live, not asserted.**
- Sprint-tree pristine (HEAD, M1 applied): `PASS: 60`, gate's own arm prints `ok 60 - arm-count
  still matches the set this suite covers (60)`.
- Sprint-tree removal mutant (same line): gate fires — `not ok - arm-count drift: suite ran 59 arms,
  this suite is written for 60.` — matching the doc's Test-plan row exactly.
- Sprint-tree **self-arm-deletion mutant** (delete the gate's own `pass_arm` line) — **not present as
  an executed row anywhere in the doc's round-3 Verification Log**, only as an *expected* row in the
  Test Plan table and as a narrative claim in the Round-1 resolution ("a new Test-plan row proves the
  self-arm deletion now reds" — a Test-plan row is a spec, not a measurement). I ran it myself:
  `not ok - arm-count drift: suite ran 59 arms, this suite is written for 60.` — **confirmed true**,
  but the doc oversold this as already "proven" when it had not, in round 3, actually re-run it.
  Non-blocking (the property holds; M2 — the full mutation matrix — is explicitly deferred, and this
  gap is inside that deferral), but worth naming per the directive's "reproduce before you dismiss."
- Sprint-tree **addition mutant** (inserted an extra `pass_arm` before the gate) — same gap, same
  resolution: I ran it, gate fires (`not ok - arm-count drift: suite ran 61 arms, this suite is
  written for 60.`), confirming the exact-equality direction is sound, but again this specific row
  was not freshly executed in round 3's Verification Log, only specified as an expected observable.

**Is the astra/gemini conflict real or manufactured?** **Real, not manufactured.** Read directly from
primary sources:
- astra (`/tmp/astra_r3_iter39.json`, round-3 re-run): *"count only mandatory, environment-independent
  arms in the exact-equality gate... **Do not adjust the expected count using the observed optional
  outcome.**"*
- gemini (round-3 quorum artifact): *"set an explicit flag when the arm runs...
  `expected_arms=$(( 60 + ${loopback_sampled:-0} ))`."*

Gemini's fix **is** exactly the thing astra's fix explicitly forbids — adjusting `expected_arms` by
the observed optional-arm outcome. This is a genuine, verified, opposite-direction disagreement
about how to handle the one environment-conditional arm, not a controller-invented one.

**But the park decision does not actually depend on resolving that conflict**, and the stated
justification overstates the human ruling's scope — see item 6, the one finding I am marking
BLOCKING-adjacent (filed as non-blocking below, with reasoning).

### 5. Is anything landing on the controller's own verdict? — NO, clean

`git show f6750002d --stat`: one file changed, `design_docs/planned/m-motoko-suite-arm-count-floor.md`
(343 insertions, 99 deletions). `git merge-base --is-ancestor bd4ee3d87 origin/dev` → **NO**;
same for `f6750002d` → **NO**. Neither the M1 code nor this doc revision is on `dev`. Nothing is
landing on the controller's reject; the commit is exactly what C5 claims — doc plus records only.

### 6. Something the controller got wrong that nobody has named — NON-BLOCKING, but worth stating plainly

The controller's stated reason for parking is: *"choosing between [astra's and gemini's fixes] would
be the same controller-invented resolution the ruling forbids."* This **misreads the scope of
D-MOTOKO-CARVEOUT-1**. That ruling was specifically about the Gate-2 narrow-refinement carve-out —
applying a reviewer's *verbatim* fix without a re-quorum. It was not a blanket prohibition on the
controller ever synthesizing between reviewer positions; every prior round of this very doc did
exactly that (round 1's controller "position (a)" pick between host-scoping options; this round's own
choice of glm's remedy 2 over remedy 1). Read literally, over-applying this framing risks a future
iteration treating *any* controller judgment as forbidden, which is not what was ruled.

That said, **the park is still the objectively correct disposition**, on a narrower and sturdier
basis than the one given: per this mission's own Gate-2 protocol
(`.claude/skills/mission-control/resources/gate-2-pick.md`, "QUORUM-AT-PICK" /
narrow-refinement-carve-out section), the default rule is *one revision, one re-quorum, then park if
still blocked* — and the carve-out that would let the controller skip that is explicitly unavailable
here because it requires that no remaining objection "dispute the design DIRECTION." Both astra's and
gemini's P2 fixes change what the exact-equality gate counts and how — that is a direction-level
disagreement about the shape of the invariant, not a narrow completeness/attribution nit (contrast
glm's own round-3 objection, about an unverified `host_os` premise, which genuinely *would* qualify
for the carve-out in isolation, and I confirmed the underlying fact is true and the doc is missing
the verification-log row that would have closed it — see below). So: **park is correct even without
invoking the ruling at all**; the ruling's name is doing rhetorical work it doesn't need to do, and a
future reader citing this iteration as precedent for "the ruling forbids controller synthesis" would
be wrong. I am marking this **non-blocking** because it does not change what should happen to row 6s
this iteration — but it is a real reasoning defect worth correcting in the mission log so it isn't
inherited as a wrong precedent.

**Secondary, smaller finding, non-blocking**: glm's round-3 objection ("no verification-log row
confirms the content of line 20") is factually correct as filed — I checked, and the doc indeed
asserts `host_os=$(uname -s 2>/dev/null || printf '%s\n' unknown)` at line 20 in prose (design doc
line 80) with no corresponding Verification Log row. I independently confirmed the line-20 content
is exactly as claimed (`sed -n '20p'` on the sprint file). The objection is genuine and unresolved by
this revision, though moot for the park outcome since gemini's direction-level objection blocks
regardless.

## Reproduction inventory (for anyone re-checking this report)

All in `/tmp/iter39_verify/`: `base.sh` (878939117 copy), `scratch/test_mutant.sh` (removal mutant on
base), `scratch/test_sprint_pristine.sh`, `test_sprint_removal_mutant.sh`,
`test_sprint_selfarm_mutant.sh`, `test_sprint_addition_mutant.sh` (all against the actual HEAD sprint
file + its unmodified probe), with `.out` files for each run. Total suite wall time consumed: ~5 runs
× ~70s ≈ 6 minutes, each launched backgrounded and polled in a bounded `date +%s` loop, never awaited
by ending a turn.

## Score breakdown

- Audit fidelity (item 1): 30/30 — every row reproduced, one self-referential control artifact noted, not a defect.
- P1 (item 2): 20/20 — real, correctly bounded, independently verified including the heredoc boundary.
- glm verbatim (item 3): 15/17 — verbatim confirmed; -2 for the M1b not-yet-landed nuance being easy to misread from the doc's Design section alone (mitigated by the Milestones section, so a small deduction not a blocking one).
- P2 reproduction + conflict analysis (item 4): 18/20 — P2 fully reproduced live; -2 for two Test-plan rows (self-arm, addition) narratively claimed as "proven" in round 1 without a fresh round-3 execution row, which I had to supply myself.
- No landing on controller's own verdict (item 5): 10/10.
- Unnamed controller error (item 6): 7/13 — real finding (ruling-scope overreach in the stated rationale) that does not change the correct outcome; not blocking, but material enough that full marks aren't warranted for "nothing else wrong."

**Total: 90/100 — PASS.**

## BLOCKING findings

**None.** The audit is genuine, P1 and P2 are both real and correctly characterized, glm's remedy is
verbatim, and no code lands on the controller's own reject. The park is the correct call.

## Answer to item 4 (restated)

The astra/gemini round-3 conflict over P2 is **real**, verified against primary sources, not
manufactured. However, the controller's specific *justification* for parking (invoking
D-MOTOKO-CARVEOUT-1 as forbidding a choice between them) overreaches what that ruling actually
covers. The park itself is still correct — independently, via this mission's own Gate-2 default
(one-revision-one-requorum→park, with the carve-out unavailable because the objections dispute
design direction) — so no routing call was wrongly avoided here, but the stated reasoning should not
be repeated as precedent for a general ban on controller synthesis.
