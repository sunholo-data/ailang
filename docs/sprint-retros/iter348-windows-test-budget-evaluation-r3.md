# Sprint Evaluation — M-COORDINATOR-WINDOWS-PACKAGE-TIMEOUT-HEADROOM — ROUND 3 (FINAL)

**Verdict: PASS — 86/100**

This is round 3 of 3. A round-3 FAIL parks the item for human review; a PASS lands it. Based on
independent re-verification of every prior finding plus fresh investigation, **this should land.**
No BLOCKING finding survives on the current tree (`7c9596193`). Residual issues are all
NON-BLOCKING documentation/coverage gaps, enumerated below for whoever picks up the follow-up.

---

## 0. Diff re-derivation (round 2 → round 3)

`git diff --stat 1f95c084c..7c9596193` on the tree I was given, independently:

```
 .gitattributes                                                                    |  3 +
 .../m-coordinator-windows-package-timeout-headroom-sprint-plan.md                 |  9 ++-
 .../m-coordinator-windows-package-timeout-headroom.md                             |  2 +-
 sprint_m-coordinator-windows-package-timeout-headroom.json                        | 11 ++-
 tools/ci/headroom/main.go                                                         |  5 ++
 tools/ci/headroom/main_test.go                                                    | 92 ++++++++++++++
 6 files changed, 113 insertions(+), 9 deletions(-)
```

**This exactly matches the stated diff.** Nothing omitted, nothing mis-stated. The whole-sprint
history (`39dfb6a84` base → `a3cbbca0c` M1 → `95e45d380` M2 → `73d4cee75` M3 → `1f95c084c` R2 fix →
`7c9596193` R3 fix) was used as the basis for adjudication below, not just the round-3 delta.

---

## 1. Adjudication — Round 1's nine findings

| # | Finding (Round 1) | Round 2 verdict | Round 3 verdict | Command |
|---|---|---|---|---|
| 1 | Parser anchored `^ok\t`, matched ZERO real records (BLOCKING) | CLOSED | **CLOSED** | `grep -n 'regexp\|MustCompile\|\^ok' tools/ci/headroom/main.go` → no hits; parser now splits on TAB. `go test -run TestParseRealGoTestOutput ./tools/ci/headroom/` → ok, 128 records (127 ok + 1 FAIL) from the real fixture. |
| 2 | Consequence: sprint's own PR red on both CI legs (BLOCKING) | PARTIAL/RECURRED (CRLF) | **CLOSED** | PR #1102 `test-windows`=SUCCESS, `Build windows-latest`=SUCCESS (both green). The two still-red checks (`test`, `lint`) are proven **not attributable to this sprint** — see §3. |
| 3 | `(cached)` branch was dead code (BLOCKING) | CLOSED | **CLOSED** | `parse()` matches `fields[2] == "(cached)"` after the new `TrimSuffix(line, "\r")`; cached fixture → 105 zero-second records, confirmed by `TestParseRealGoTestOutput` and `TestParseRealGoTestOutputCRLF`. |
| 4 | Three surviving mutations: `budget<=0`, sort tie-break, arg-count panic (NON-BLOCKING) | CLOSED | **CLOSED** | `TestBudgetValidation` (`"0","-5","abc"` → rc=2), `TestTopNTieBreak` (60 tied packages, ascending-name tiebreak verified), `TestUsageError` (`len(args)<3` → rc=2, no panic) — all present, all pass. |
| 5 | False "exactly two hits" grep claim in three documents (NON-BLOCKING) | PARTIAL | **CLOSED** | Design doc + sprint plan now assert `grep -c 'go test -timeout 416s' ci.yml` → 2 (verified: returns `2`). Sprint JSON's own M1 acceptance command claims "four hits" for the bare `grep -n 'timeout 416s'` pattern — verified true (`grep -c 'timeout 416s' ci.yml` → 4). No document left asserting an incorrect 2-hit count for the ambiguous pattern. |
| 6 | Missing CHANGELOG entry (NON-BLOCKING) | CLOSED | **CLOSED for the R2 fix; NEW gap for R3** — see NF-D below | `changelogs/v0.32-current.md:25` documents the `^ok\t` fix (landed at `1f95c084c`). No entry documents the round-3 CRLF fix itself — `git diff --stat 1f95c084c..7c9596193` does not touch `changelogs/`. |
| 7 | Untested CRLF path (NON-BLOCKING → escalated to BLOCKING at round 2, tracked as NEW-1) | OPEN, escalated | **CLOSED** | See §2, exhaustive. |
| 8 | Exit-code contract described two conditions, there are three (NON-BLOCKING) | CLOSED | **CLOSED** | `main.go`'s header comment now documents exit 0/1/2 with exit 1's three sub-conditions (cannot-read-log, anti-vacuity, package-count floor) explicitly and correctly. |
| 9 | Sprint JSON stuck at "planned" (NON-BLOCKING) | PARTIAL | **CLOSED for `status`; NEW staleness on M3** — see NF-E below | `sprint_...json`: `"status": "completed"`. M1/M2 `passes: true`; M3 `passes: false` with a note that Windows verification was PENDING — but test-windows is now green (see §3), so this note is stale, not false. |

## 2. Round 2's CRLF finding (NEW-1) — closed at the level of BEHAVIOUR

Round 2's BLOCKING finding: `.gitattributes` had no `eol=lf` pin for the headroom testdata, so a
Windows checkout rewrote the fixtures and dropped all 105 cached records. Attacked from every
angle asked for:

- **`.gitattributes` pin actually covers the paths:**
  ```
  $ git check-attr text eol -- tools/ci/headroom/testdata/real_go_test_output.txt tools/ci/headroom/testdata/real_go_test_output_cached.txt
  tools/ci/headroom/testdata/real_go_test_output.txt: text: set
  tools/ci/headroom/testdata/real_go_test_output.txt: eol: lf
  tools/ci/headroom/testdata/real_go_test_output_cached.txt: text: set
  tools/ci/headroom/testdata/real_go_test_output_cached.txt: eol: lf
  ```
- **Byte-identical LF vs CRLF report** (built binary, ran both): confirmed via `diff` — zero output,
  exit 0 both times.
- **Lone `\r` on the last field of the last line, no trailing `\n`:** constructed a 60-package log
  ending `...1.0s\r` (no `\n` at all) — parsed all 60 records, rc=0.
- **Mixed line endings** (alternating `\r\n`/`\n` across 60 records): all 60 parsed, rc=0.
- **No trailing newline at all:** single-line and multi-line variants both parse correctly (the
  package-count floor, not a parse bug, is what fails a too-short synthetic log — expected).
- **Mutation kills the new test:** reverted the `strings.TrimSuffix(line, "\r")` fix →
  `TestParseRealGoTestOutputCRLF/cached` fails (`got 23 records, want 128`). Restored, green again.
- **Real Windows CI, not just local synthesis:** pulled the actual `test-windows` job log for PR
  #1102 (job `101848640109`) — `tools/ci/headroom` itself reports `ok ... 0.044s` on the real
  Windows runner, the `HEADROOM: top 5 slowest packages` report line is present, and there is no
  `::error:: headroom` line anywhere in the log. This is the first round where the CRLF fix has
  been proven on an actual Windows checkout, not just synthetically.

**Verdict: CLOSED, thoroughly.**

## 3. CI attribution — independently re-derived, YOUR CLAIM HOLDS

I did not accept the attribution; I reproduced it against the actual merge tree GitHub built.

- `lint` fails on `internal/coordinator/msg_id_suffix_test.go:44:6 SA4000` (confirmed via
  `gh run view ... --log`, exact match). `git cat-file -e 7c9596193:internal/coordinator/msg_id_suffix_test.go`
  → fails (absent on this branch). `git cat-file -e origin/dev:...` → present. One correction to
  your evidence: the file's originating commit is **`a12ed330d`** ("fix(coordinator): task ids from
  deterministic message ids collided by construction", landed 2026-09-07 18:54 UTC), not
  `22150ef72` — `22150ef72` (Release v0.35.2) is a later commit that also touches the file (a
  release-squash artifact) but is not the introduction point. This doesn't change your conclusion,
  only its precision.
- `test` fails at `make verify-pi-assets`: `DRIFT: cmd/ailang/pi_assets is stale vs .pi/extensions`.
  I did not just retest at pristine `origin/dev` HEAD (which no longer reproduces it — see below) —
  I found the **actual merge commit GitHub built for this run**
  (`gh api repos/.../pulls/1102 --jq .merge_commit_sha` → `e92e71bb0`, merging `7c9596193` into
  `86c71fb9b6`, the dev tip *at trigger time*, not current dev tip) and checked that exact tree out
  in a scratch worktree:
  ```
  $ make verify-pi-assets   # at e92e71bb0, the actual PR merge ref
  Files .pi/extensions/prepush-gate.ts and cmd/ailang/pi_assets/prepush-gate.ts differ
  DRIFT: cmd/ailang/pi_assets is stale vs .pi/extensions — run 'make pi-assets'
  ```
  Reproduced exactly. Current `origin/dev` tip (`ca5069df0`) no longer reproduces this because a
  later dev commit, `4b7817f53` ("fix(pi): apply the prepush-gate fix to the SOURCE, not the
  generated copy"), fixed the drift *after* this PR's CI run started — a real, transient dev-side
  defect this sprint's diff had nothing to do with (it touches none of `.pi/extensions`,
  `cmd/ailang/pi_assets`, or `internal/coordinator`). The `go test -timeout 416s` step itself, and
  the headroom binary, both ran and passed cleanly before this unrelated later step failed — visible
  in the job log (`ok ... internal/errors`, `ok ... tools/gen-error-codes`, then the
  binary-gated-integration assertion, THEN `Verify embedded pi assets are in sync` fails).

**Independent verdict: your attribution is correct. Neither red is caused by this sprint's diff.**
Worktrees used for this reproduction (`/tmp/dev-control-check`, `/tmp/mergeref-check`) were removed
after use; nothing was left in `.ailang-driver-pin/`.

`test-windows` and `Build windows-latest`: both **SUCCESS** (not pending) — confirmed via
`gh pr view 1102 --json statusCheckRollup`.

## 4. Mutation non-vacuity, per milestone

- **M1** (derived budget + gate): reverted `.github/workflows/ci.yml`'s Windows-leg
  `-timeout 416s` back to `300s` → `TestGoTestTimeoutIsDerived` fails with the exact two expected
  errors (stale-value + missing-value). Restored, green. **Non-vacuous.**
- **M2** (parser core): reverting the round-3 CRLF fix alone reds `TestParseRealGoTestOutputCRLF`
  (23 vs 128 records). Reverting `status := strings.TrimSpace(fields[0])` to the untrimmed form
  (simulating a status-column-padding regression, the ROOT CLASS of round 1's bug) reds both
  `TestParseRealGoTestOutput` and the new CRLF test (1 and 0 records respectively, vs 128 wanted).
  **Non-vacuous.**
- **M3** (CI wiring): the milestone's own named mutations (drop exit-code preservation, drop
  `|| rc=$?`, remove the build step alone) are correctly argued to fail LOUDLY in real CI (missing
  binary → `hr_rc=127` → job reds). **But I found a compound mutation the plan didn't consider —
  see NF-A below — where this "loud, not silent" property does NOT hold.**

## 5. New findings (this round's independent investigation)

None are BLOCKING. All are real, reproduced, and none affect the sprint's *current*, verified-green
behavior — they are coverage/documentation gaps for future regressions.

| ID | Finding | Class | Repro |
|---|---|---|---|
| NF-A | Removing the **build step AND the invocation line together** (e.g. via a bad merge) on the Linux leg makes the job **silently green** — no headroom report, no test failure, no CI redness. This differs from the milestone's own named "remove build step alone" mutation, which the plan correctly shows fails loudly (`hr_rc=127`); the compound deletion defeats that because `hr_rc` is never set, staying at its `0` default. | NON-BLOCKING (coverage gap, current wiring correct & CI-verified) | `sed` both lines out of `ci.yml`'s Linux leg → `go vet`/`go test` on `tools/ci/...`, `internal/cihygiene/` still pass (rc=0). Restored, `git diff --stat` clean. |
| NF-B | The `cannot-read-log` exit-1 path (one of the three documented exit-1 sub-conditions) has **zero test coverage**. | NON-BLOCKING | Mutated `return 1`→`return 0` on the `os.ReadFile` error branch → `go test ./tools/ci/headroom/` still `ok`. Restored. |
| NF-C | `budget="Inf"` or `budget="NaN"` bypass the `budget <= 0` validation guard (`strconv.ParseFloat` accepts both; `Inf > 0` is true, `NaN <= 0` is false) and produce a degenerate report (`budget Infs`/`budget NaNs`, 0% everywhere, rc=0) instead of a usage error. Real-world risk is ~zero — `ci.yml` always passes the literal `"416"` — but the stated contract ("a non-positive or non-numeric budget is a usage error") is not fully enforced. | NON-BLOCKING, low severity | `/tmp/hrbuild/headroom testdata Inf` / `NaN` → rc=0 both times, no error. |
| NF-D | No CHANGELOG entry for round 3's CRLF fix specifically (the R2 fix is documented at `changelogs/v0.32-current.md:25`; nothing was added for `7c9596193`). | NON-BLOCKING, documentation | `git diff --stat 1f95c084c..7c9596193 -- changelogs/` → empty. |
| NF-E | Sprint JSON's M3 milestone still says `"passes": false"` / "NOT VERIFIED on Windows... PENDING this round's CI", but this round's actual PR CI shows `test-windows` = SUCCESS with the real report visible (§2). The artifact under-reports its own verified state — not a false claim, but stale relative to evidence now available. | NON-BLOCKING | See §2's Windows-log evidence vs. the JSON's M3 `status` field. |
| NF-F | The `final_gates` "approved file list" entry, updated this round to add the newer files, still **omits `.gitattributes`** — the exact file this round's own diff adds. This is a recurrence of round 2's NEW-6 (stale approved-file-list gate), not a full close. | NON-BLOCKING, but a repeat pattern across two rounds | `git diff --name-only 39dfb6a84..7c9596193` → 11 files incl. `.gitattributes`; the JSON's listed set → 10 files, `.gitattributes` absent. Diffed programmatically, confirmed. |

## 6. Gates run

| Gate | Result |
|---|---|
| `go vet ./tools/ci/... ./internal/cihygiene/` | rc=0 |
| `go test -count=1 ./tools/ci/... ./internal/cihygiene/` | rc=0, both `ok` |
| `gofmt -l tools/ci internal/cihygiene` | clean (no output) |
| `go test -count=1 -timeout 416s ./...` | rc=1 first run (`internal/pipeline` flake, confirmed non-reproducing in isolation twice and on a clean full re-run: rc=0). Not attributable to this sprint — `internal/pipeline` is untouched by the diff. |
| `make fmt-check` | rc=0 |
| `go build ./...` | rc=1, fails identically at `cmd/wasm` — cited, not re-litigated: rounds 1 and 2 both confirmed this is a pristine-`origin/dev` failure unrelated to this sprint. Independently reconfirmed here too. |
| `[ $(grep -c 'go test -timeout 416s' ci.yml) -eq 2 ]` | PASS |
| `[ $(grep -c 'go_test_headroom.log' ci.yml) -ge 6 ]` | PASS |
| `git diff --exit-code -- internal/coordinator/ tools/ci/motoko_smoke.sh internal/cihygiene/workflow_timeouts_test.go internal/cihygiene/gate_wiring_test.go` (hard-unchanged paths) | rc=0, clean |

## 7. What I could NOT verify, and why

- **The pwsh-specific `$ErrorActionPreference`/`$PSNativeCommandUseErrorActionPreference` mutation**
  (M3's fourth named mutation) — `pwsh` is not installed on this rig (confirmed: `command -v pwsh`
  fails), matching every prior round's constraint. I substituted the real `test-windows` job log as
  the closest available evidence (report present, no error line, real ok/FAIL lines) rather than a
  local mutation-kill.
- **Whether a genuine `go test -timeout` panic on a hung package always emits a trailing
  `FAIL <pkg> <duration>` summary line.** I constructed a synthetic log without one (panic text only,
  no summary) and confirmed headroom degrades safely — the hung package silently doesn't appear in
  the report, but the package floor and downstream `go_rc` exit-code check still catch the failure at
  the CI-job level. I could not force a *real* `go test -timeout` panic cheaply within this
  evaluation's time budget to confirm which shape Go actually emits, so I can't say whether this
  edge case is realistic. Flagging as an open question, not a finding.
- **SonarCloud / golangci-lint's full ruleset locally** — `lint`'s failure was diagnosed from the
  real CI log rather than a local golangci-lint run (not installed here); sufficient given the exact
  file/line/rule match against a file provably absent from this branch.

## 8. Score

| Category | Points | Score |
|---|---|---|
| Tests Pass | 20 | 19 — all green except a confirmed-unrelated, non-reproducing flake |
| Lint Clean | 10 | 10 — clean locally; PR's red `lint` proven unattributable |
| Acceptance Criteria | 30 | 26 — nearly all independently verified; NF-E/NF-F are real but minor artifact mismatches |
| Code Quality | 15 | 11 — clean CRLF fix, but NF-A/NF-B/NF-C are genuine uncovered edges |
| Documentation | 15 | 11 — CHANGELOG/design-doc/sprint-plan correctness mostly restored; NF-D gap remains |
| Design Fidelity | 10 | 9 — WARN-ONLY intent and exit-code contract faithfully implemented and now accurately documented |
| **Total** | **100** | **86 — PASS** |

## 9. Bottom line

**This should land.** Every BLOCKING finding from rounds 1 and 2 is closed at the behavioral level,
with the CRLF fix now proven on a real Windows CI run (not just synthetic tests), and the two red
PR checks are independently confirmed to be dev-branch noise unrelated to this sprint's diff — I
reproduced both at the exact commit GitHub built, not just at a moving `origin/dev` tip.

What remains is a short, entirely NON-BLOCKING punch list for a fast follow-up, not a blocker for
this sprint: add a CHANGELOG line for the round-3 CRLF fix (NF-D); fix the sprint JSON's stale M3
status and the approved-file-list gate to include `.gitattributes` (NF-E/NF-F — this is the second
round this exact gate has been left stale, worth a process note); and add three small tests
(cannot-read-log, Inf/NaN budget rejection, and either a Go-test or a documented rationale for why
the M3 compound-deletion mutation NF-A is accepted as residual risk).
