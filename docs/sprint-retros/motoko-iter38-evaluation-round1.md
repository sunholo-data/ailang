# Motoko mission iteration 38 — independent sprint evaluation, round 1

**Evaluator:** independent, non-author. **Worktree:** `.wt-motoko-iter38-eval` at `bd4ee3d87`.
**Scope:** two picks on `sprint/motoko-iter38-changelog-arms` — `5a7203768` (changelog, row 16) and
`0656ac174`/`11572ac59`/`bd4ee3d87` (arm-count-floor gate, row 6s).

## Score: 68 / 100 — **FAIL** (pass line is 70)

### Rubric used (100 pts, stated up front so the score is auditable)

| # | Area | Points | Awarded |
|---|---|---|---|
| 1 | Gate correctness — pristine run + full mutation matrix, byte-identity proven each time | 25 | 25 |
| 2 | Host-scoping claim (56 off-Darwin) — reproduced independently, fairness assessed | 15 | 13 |
| 3 | Residual audit — partial search for a 5th reader of `$0` | 10 | 9 |
| 4 | Changelog entry (row 16) factual accuracy | 15 | 13 |
| 5 | Controller conduct — narrow-refinement carve-out compliance + measurement verification | 25 | 8 |
| 6 | Process hygiene — clean restoration, byte-identical tree, reproducible evidence | 10 | 10 |
| | **Total** | **100** | **68** |

The single fact that moves this below the pass line is **#5**: the narrow-refinement carve-out was
invoked to skip a third quorum round, but its own governing condition — apply the reviewer's fix
**verbatim**, never a controller-invented substitute — was not met for two of the three round-2
reviewers. Everything else scored is solid-to-excellent; the code shipped is correct and the
mutation evidence is real. This is a **process-legitimacy** failure, not a code-quality one, but the
mission's whole quality model rests on the quorum gate meaning what it says.

---

## A. Pristine run — VERIFIED, NON-BLOCKING (confirms the claim)

```
$ bash tools/eval/test_motoko_connection_probe.sh
...
ok 60 - arm-count still matches the set this suite covers (60)
PASS: 60 probe self-test arms ran
$ echo $?
0
```
Matches the design/sprint-plan acceptance criterion exactly.

## B. Mutation matrix — VERIFIED, NON-BLOCKING (this is the strongest part of the sprint)

Backup taken and hashed **before** any mutation:
```
$ shasum -a 256 tools/eval/test_motoko_connection_probe.sh
52ab531ef9659f750764bc0cfa2cd85a2c3c26c641f8b9f63a717a05e7296df7
```
This matches the SHA the executor's own commit message (`bd4ee3d87`) cites as the pristine base —
independently confirms the commit's self-reported baseline was real.

| Row | Edit | Command | rc | Observable | Restored SHA match |
|---|---|---|---|---|---|
| R2 — addition (LOOKS) | insert `pass_arm "extra"` immediately before the gate block | `bash tools/eval/test_motoko_connection_probe.sh` | 1 | `not ok - arm-count drift: suite ran 61 arms, this suite is written for 60.` | yes (`52ab53...`) |
| R3 — self-arm deletion (round-1 killer objection) | delete `pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"` | same | 1 | `not ok - arm-count drift: suite ran 59 arms, this suite is written for 60.` | yes |
| R4 — zero anti-vacuity | insert `arms=0` immediately before `if (( arms == 0 ))` | same | 1 | `not ok - zero test arms ran`, **zero** `arm-count drift` lines (`grep -c "arm-count drift"` → `0`) | yes |
| X1 — my own mutant: `!=` → `>=`, then re-apply R2's addition | change the comparison to a floor, then insert `pass_arm "extra"` | same | **0** | `PASS: 61 probe self-test arms ran` — the addition becomes **invisible** | yes |

X1 is the load-bearing check I added myself (one of the suggested shapes, but I picked it because
it directly tests the design's central claim rather than taking that claim on faith): with a floor
instead of exact equality, R2's addition mutant — the *only* mutant in the whole matrix that
distinguishes "fires" from "looks" — silently passes. This independently confirms the design's
stated reason for choosing exact equality over `>=` is real, not rhetorical.

Every restore was verified byte-identical to `52ab531ef9659f750764bc0cfa2cd85a2c3c26c641f8b9f63a717a05e7296df7`
before the next mutant was applied. Final `git status --porcelain` in the worktree is empty except
for this report file (verified again at the end of this session).

I did not additionally re-run R1 (delete an arbitrary pre-existing arm) since it is not part of
what I was asked to independently verify and the design/sprint-plan's own R1 evidence is not the
row carrying the interesting claim (R2 is). I note this as **not independently re-verified by me**
(design/commit message claim it reds at 59-vs-60, same shape as R3, plausible but UNMEASURED here).

## C. Host-scoping claim (56 off-Darwin) — VERIFIED, NON-BLOCKING, with one caveat

Reproduced with a PATH-shadowed `uname`:
```
$ cat /tmp/fake_uname_bin/uname   # if [[ "$1" == "-s" ]]; then echo Linux; else /usr/bin/uname "$@"; fi
$ PATH=/tmp/fake_uname_bin:$PATH bash tools/eval/test_motoko_connection_probe.sh
...
ok 56 - arm-count still matches the set this suite covers (56)
PASS: 56 probe self-test arms ran   # rc=0
$ grep -c UNINFORMATIVE <log>
4
```
This matches the design's claimed 56 exactly (55 pre-gate + the gate's own arm = 56).

**Is the simulation fair?** For the specific thing being measured — arm count — yes, and I checked
why: every place that skips a Darwin-only arm is gated purely on `$host_os` (`skip_run_lane_fixture`
at line 31, derived from `host_os != Darwin`; the `REAL_LSOF` containment block at line ~999 gated
directly on `[[ "$host_os" == Darwin ]]`), and `$host_os` is set from a single `$(uname -s)` call
that the PATH shadow controls. I also checked the *other* `uname -s` call site (line 612, the
loopback-socket arm) and confirmed it is shadowed identically. So the host-scoping decision itself
is fairly tested by this simulation. The design's own doc already discloses the broader limit
("a real Linux box would also differ in shell and tools") — that framing is honest and I agree with
it: this simulation says nothing about `lsof`/`dig` availability, GNU vs BSD tool semantics, or
bash-version differences on a real Linux host, only about the host-scoping branch itself. I also
confirmed the suite genuinely runs under bash 3.2 on this rig (`bash --version` → `3.2.57`), matching
the CI target, so my runs are representative of the actual gated bash version.

**Caveat (why not full marks):** the design's Verification Log row for this claim says
`grep -c UNINFORMATIVE → 4 (exactly the 4 skipped arms)`, presented as corroborating evidence. I
checked the actual log lines:
```
34:UNINFORMATIVE UNDER SANDBOX: live synthetic socket arm requires darwin nc+lsof; ...
39:UNINFORMATIVE: run_lane fixture arm requires real lsof for cwd survivor checks
46:UNINFORMATIVE: run_lane SIGKILL-escalation arm requires real lsof for cwd survivor checks
47:UNINFORMATIVE: REAL_LSOF containment arms are Darwin-only, as is the gate they pin
```
Only lines 39/46/47 are the host-scoping skips (3 messages covering exactly 4 arms, since line 47
covers 2 arms in one echo). Line 34 is a **different, unrelated** gate (the loopback-socket arm,
gated on sandbox networking, not on `$host_os` — it fires on the *pristine Darwin* run too, just
with a different message: `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer`).
So `grep -c UNINFORMATIVE == 4` is a **coincidence**, not a 1:1 correspondence with "the 4 skipped
arms" as the doc's row claims. The final number (56) is still correct — I verified it by direct
arithmetic and by the actual `PASS: 56` terminal line, not by trusting this grep — but the specific
evidentiary sentence in the Verification Log is sloppy. **NON-BLOCKING** (the conclusion is right,
one piece of supporting evidence for it is not what it claims to be).

## D. Residual audit — PARTIAL, NON-BLOCKING (no 5th reader found in the time available)

The design admits: "four direct readers of `$0` found; exhaustiveness remains unverified," and
that gpt6-astra's full-file audit was never run. I spent part of my budget on a partial version:

```
$ grep -n '\$0' tools/eval/test_motoko_connection_probe.sh          # every literal $0 use
$ grep -n 'BASH_SOURCE' tools/eval/test_motoko_connection_probe.sh  # none
$ grep -nE '(^|[^.a-zA-Z])(source|\. )' tools/eval/test_motoko_connection_probe.sh  # no sourced files
$ grep -rn 'test_motoko_connection_probe' --include='*.sh' --include='*.mk' --include='*.yml' .
    make/test.mk:72   (executes it)
    make/test.mk:75   (bash -n, syntax check only)
    scripts/test_check_referenced_paths.sh:46  (executes it)
```
Findings:
- 13 occurrences of `$0` are recursive self-invocations (`/bin/bash "$0"`, e.g. lines 958, 1007,
  1011, 1035-1122) used to test the suite's own error paths under env vars like
  `PROBE_SELFTEST_DERIVATION_ONLY=1`. I traced these and confirmed each one hits an early-exit guard
  (lines 181, 353) **before** reaching the new gate at line ~1211 — they never execute the inserted
  code, so they cannot interact with it. This is a different category from the design's "readers of
  $0" concern (content-census `grep ... "$0"`), and correctly out of scope for that specific claim.
- The only content-census readers of `$0` are the 4 the design already found (lines 1186-1189).
- No other file in the repo greps or counts patterns inside this test file; the only two other
  references execute it or syntax-check it.

**I did not find a 5th reader.** This is not a full exhaustive audit (I did not check for shell
aliases, or every indirect invocation path) — stated plainly as **UNMEASURED beyond what's above**,
consistent with what the design itself says is still open. Not blocking: no break found.

## E. Changelog entry (row 16) — MOSTLY VERIFIED, ONE NON-BLOCKING OVERSTATEMENT

All ten cited commits exist and match their stated subjects exactly (`8ae79b52e` … `7a0bbd5da` for
the feature, `45bbcf625` for the PR #1055 merge) — checked with `git log -1 --format='%H %ci %s' <c>`
for each.

**"`ailang mission list|doctor|install|apply` exists"** — VERIFIED:
```
$ grep -n '"list"\|"doctor"\|"install"\|"apply"' cmd/ailang/mission_cmd.go
31: case "list":
33: case "doctor":
35: case "install":
37: case "apply":
```

**"four registry entries"** — VERIFIED:
```
$ ls missions/
docs.toml  motoko.toml  v1.toml  world.toml
```

**Placement in `## [Unreleased]`** — checked whether this belongs there. `v0.35.1` is tagged at
`2026-09-05 13:51:03 +0200`; the `M-MISSION-LOOP-WORKBENCH` commits run `22:15` to `23:51` the same
day, i.e. strictly **after** the v0.35.1 tag (`git merge-base --is-ancestor v0.35.1 a9de67fe6` →
true). So this work is genuinely unreleased and belongs in `## [Unreleased]`, not folded into
v0.35.1 or a new section — the entry's placement judgment is correct.

**One overstatement, NON-BLOCKING:** the "Fixed" section says "all six originally-red checks were
green at its final commit." I pulled the actual PR:
```
$ gh pr view 1055 --json statusCheckRollup
```
At the final commit, `test`, `test-windows`, `Build windows-latest`, `launchd drivers (bash 3.2)`,
`build`, `lint` etc. are all `SUCCESS` — **but** `SonarCloud Code Analysis` is `FAILURE`. Branch
protection's required contexts are only `test, lint, build, docs-gate`
(`gh api repos/.../branches/dev/protection`), so Sonar isn't gating and the merge itself is not
contradicted. But "all six...were green" is not literally true if Sonar is included, and the PR
body's own defect table only names **4 distinct CI check names** (`test-windows`,
`Build windows-latest`, `launchd drivers (bash 3.2)`, `test`) — "six" appears to come from summing
table rows (M2:2 + M3:1 + M5:2 + M7:1) without noticing M2 and M5 name the *same two* checks twice.
The underlying substance (every actually-broken check is now green) is correct; the number "six" is
an imprecise characterization of it. **NON-BLOCKING.**

## F. Controller conduct — BLOCKING

This is the finding the task specifically asked me to hunt for, and I found it by pulling the raw
quorum JSON artifacts (not present in this worktree; located and read read-only from the sibling
checkout `/Users/voightkampff/.ailang-driver-pin/motoko/.ailang/state/mission-quorum/`, without
modifying anything there):

- `m-motoko-suite-arm-count-floor-2026-09-07T06-48-47Z.json` (round 1)
- `m-motoko-suite-arm-count-floor-2026-09-07T06-56-41Z.json` (round 2)

### F1 — the narrow-refinement carve-out's own condition was not met (BLOCKING)

The carve-out (`.claude/skills/mission-control/resources/gate-2-pick.md:47-53`) permits a bounded
2nd revision **without** a third quorum round only if, for **every** remaining blocking objection,
the controller "applies the reviewers' VERBATIM fixes (their own text — never a controller-invented
resolution, never overriding an objection)." I compared round 2's `proposed_fix` text (raw JSON)
against what the design doc actually did:

**gpt6-astra, round 2** — `proposed_fix` (verbatim from the JSON) has two parts: (1) replace the
closure sentence with an exact quoted replacement, and (2) *"Before approval, add a
verification-log row documenting a full-file review at base 878939117... Mark the prior quorum
objection unresolved until this audit is recorded."* Part (1) was applied verbatim — the doc's
Conflict Surface now reads exactly astra's proposed sentence. **Part (2) was not done.** The doc's
own "Residual" section admits: *"gpt6-astra's further ask — a full-file audit... — was NOT run."*
By astra's own stated condition ("mark unresolved until this audit is recorded"), the objection was
**not** actually resolved — only reworded to be honest about being unresolved. Proceeding to
sprint-planner anyway is not what the carve-out authorizes; per the carve-out's own text this case
— fix not fully/verbatim applied — should have parked `needs-human-review`.

**oc-glm-5-2, round 2** — `proposed_fix` (verbatim): *"either run the suite once on a Linux host to
measure the actual count and verify it equals 56, or mark the non-Darwin path as UNVERIFIED and
gate it behind an explicit `if [[ "$host_os" == Darwin ]]; then ... else echo 'UNVERIFIED host:
arm-count gate skipped' >&2; exit 1; fi`."* Neither literal option was implemented. What shipped
is a **third thing**: a PATH-shadowed simulation, honestly labeled "SIMULATED... not evidence from a
real Linux host" in the doc text — but the shipped code's `else` branch unconditionally sets
`expected_arms=56`; it does **not** implement glm's "UNVERIFIED, skip" fallback. This is a
controller/designer-invented middle path, not either of glm's specified remedies, and glm was never
asked whether the substitute was acceptable (no round 3).

Both deviations are in the same direction: take the reviewer's harder, more expensive remedy and
substitute an honestly-labeled-but-weaker one, then treat the objection as satisfied. I want to be
fair to the authors here: the substitutions are not hidden — the doc is unusually candid about both
shortfalls (it says "NOT run" and "SIMULATED" in its own text) — which is good practice and better
than silently claiming closure. But candor about a shortfall is not the same as meeting the carve-out's
bar, and the carve-out's bar is what licenses skipping the third quorum round. Under the rule as
written, this doc's correct disposition after round 2 was `needs-human-review`, not "controller
applies a bounded 2nd revision and routes straight to sprint-planner." **I am willing to say
explicitly: this should have parked.**

Round 1's oc-glm-5-2 objection (exhaustiveness of the conflict-surface enumeration) has the same
shape in miniature: glm's round-1 `proposed_fix` asked for a specific broad census command
(`grep -n 'expected_\|grep -[cnE]\|grep -Ec\| != '`); the round-1 revision instead ran a narrower,
different command (`grep -n 'grep [-A-Za-z]* .*"\$0"'`). That substitution was not re-caught by glm
in round 2 (glm moved on to a different objection), but it *was* caught by a **different** reviewer
(gpt6-astra) in round 2, raising essentially the same exhaustiveness gap. That is a second, weaker
data point for the same pattern: substituted fixes in this doc's history have concretely gone on to
be judged insufficient by the quorum itself.

### F2 — controller-supplied measurements — all VERIFIED accurate (mitigating, not blocking)

I independently re-ran every number the controller fed to the designer/executor:

| Claim | My reproduction | Result |
|---|---|---|
| Non-Darwin count = 56 | PATH-shadowed `uname -s` → `Linux` | `PASS: 56 probe self-test arms ran`, rc=0, `grep -c UNINFORMATIVE` → 4 — **matches** |
| Determinism (design cites 4/4 pre-gate; I re-ran post-gate) | 1 pristine + 2 more consecutive runs | `PASS: 60`, rc=0, 3-for-3 identical — **consistent, no drift observed** |
| Global `grep -nw arms` census | `grep -nw arms tools/eval/test_motoko_connection_probe.sh` | Single assertion hit at (now-shifted) line 1208 plus counter init/increment/print and the new gate's own lines — **matches the documented shape** |
| CI-wiring correction (`make/test.mk:72`, `ci.yml` macos-latest job) | `sed -n` on both files | `make/test.mk:72` runs the suite inside `test-launchd-drivers`; `ci.yml`'s `launchd-drivers` job runs `runs-on: macos-latest` and calls `make test-launchd-drivers` — **matches** |

None of the controller's own first-party measurements were wrong. The failure in F1 is not "the
controller fed the designer a false number" (the standing-rule 4 trap this task warned about) — it
is "the controller declared reviewer objections satisfied by a substitute the reviewers did not
propose and were never asked to accept." That is a distinct, and in this case more serious, failure
mode: it's not a wrong fact, it's a self-certified process shortcut.

---

## What I could NOT verify (stated plainly)

- **R1** (delete a pre-existing, non-gate arm) — not independently re-run by me; relied on the
  commit's own claim. Same failure shape as R3 (which I did verify), so plausible, but UNMEASURED
  here.
- **Full exhaustiveness of the `$0`-reader audit** (section D) — I ran a partial, targeted search
  and found nothing broken, but I did not do gpt6-astra's originally-requested full audit of
  aliases/sourced files/invoked helpers across the whole 1212-line file line by line. Genuinely
  open, same as the design admits.
- **Whether oc-glm-5-2 or gpt6-astra would have accepted the substituted fixes** if actually asked
  in a round 3 — by definition unknowable without running that round. I can only say the carve-out's
  own text requires *their* fix, verbatim, not a plausible-looking alternative.
- **Whether SonarCloud's failure on PR #1055's final commit reflects a real, still-open issue** — I
  did not open the SonarCloud report; I only confirmed the check's conclusion via the GitHub API and
  that it is not a required merge context.

## Restoration proof (repeated at end of session)

```
$ git status --porcelain
$ shasum -a 256 tools/eval/test_motoko_connection_probe.sh
52ab531ef9659f750764bc0cfa2cd85a2c3c26c641f8b9f63a717a05e7296df7
```
Tree is clean; the probe file hash matches the pristine base used throughout this evaluation.
