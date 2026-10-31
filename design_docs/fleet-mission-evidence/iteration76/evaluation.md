# Iteration 76 evaluation (independent judge, sprint-evaluator role)

Pin `bf4f14473`. All checks first-party: prod Cloud Logging (`ailang-multivac`), `git` at HEAD. Nothing was written except this file.

## Verdicts

| Record | Verdict | Score |
|---|---|---|
| 1. Resolve `executor-image:no-make-no-cc-cgo-tests`, file D-FLEET-41 | PASS-WITH-EDITS | 72 |
| 2. Fold `eval:motoko-connection-probe-orphaned` into D-FLEET-18 | PASS-WITH-EDITS | 85 |

Record 1 passes only if edits E1-E4 are applied. As drafted it would FAIL (about 55): the "no missing toolchain" claim is false as worded, and the evaluator residual is understated by 4x.

## Record 1

### Confirmed

- **Lifecycle (claim 1).** task-69be551b started 19:09:18Z on `ailang-agent-executor-codex-go` (`gpt-6.1-sol`). It published `blocked` 19:12:19Z with `files=0`. The block text is "Environment provisioning of make and a C compiler for CGO/SQLite tests". Matches.
- **#1753 (claim 2).** `git show 939beb6fd` is dated Fri Oct 9 19:20 +0200, "Add authorized cloud task inputs and fix codex-go build prerequisites (#1753)". `docker/Dockerfile.agent-codex-go:6-13` installs make, gcc and libc6-dev.
- **Image digest (claim 3).** `gcloud run jobs describe ailang-agent-executor-codex-go` gives `agent-codex-go@sha256:cdc075e2dd67…`. The controller did not prove the digest contains #1753, but behaviour does, as follows.
- **Behavioural proof the toolchain is live.** The control window has codex-go itself saying "no C compiler" (10-09 10:44Z) and restoring "a temporary Zig compiler". After the fix, codex-go sprint-executor runs execute `make test-core`, `make lint` and `make test-changed` on 10-16 (task-9c11d207, 16:39-20:32Z) and 10-27 (task-53b8873a). No missing-make or cc line appears on codex-go after 10-09T18:00Z.
- **Counts.** 20 codex-go `sprint-executor` starts and 18 distinct tasks (two repeated: task-d61964fa, task-60e90d30). pi-go ran 26 starts since 10-01, all `sprint-evaluator`. 2 evaluator starts on `-go` and 6 on `-go-claude-credit`. Re-derived by `gcloud logging read` and a `sed|uniq -c`. All match.
- **Skill half (claim 5).** `.claude/skills/sprint-executor/resources/unattended_runs.md:58-64` has the "Never install toolchains" rule. Confirmed at HEAD.
- **Evaluator skill (claim 8).** `grep -rniE 'cgo|C compiler' .claude/skills/sprint-evaluator/` returns nothing.
- **Scope (claim 9, option B).** The guard (`tools/launchd/githooks/pre-push:17-18`) allows `.claude/skills/sprint-*` and `.agents/skills/sprint-*`. Option B's skill edit is in fleet scope. `docker/**` is not in the Authority allowlist.

### Findings

**F1. BLOCKING-IF-LEFT. "Since 10-10 ... none logged a missing toolchain" is false for the evaluator class.**

The draft's filter matches only `make: not found`, `make: command not found`, `C compiler`, `gcc": executable file not found` and `requires cgo`. Evaluators on `agent-go` say it in prose instead. Command:

```
gcloud logging read '... job_name="ailang-agent-executor-go-claude-credit" AND 2026-10-12 AND (textPayload:"not installed" OR textPayload:"isn")' --format='value(timestamp,textPayload)'
```

Output excerpt:

```
19:42:54Z  make` isn't on
19:45:37Z  , since `make` isn't
21:15:12Z  `make` isn
21:24:17Z  since `make` is not installed
21:29:53Z  far: `make` isn't
```

That is three distinct evaluator tasks on 10-12: task-62beea26 (19:38 start), task-f96e9162 (21:09) and task-f257f24d (21:28). task-f257f24d even ran `which make go ailang z3` at 21:29:54Z.

Together with task-68e3cfb1 (10-27, `box has no C compiler`), **at least 4 of the 8** post-fix evaluator runs hit the gap, not "1 of 8". I did not find hits in the other four: the 10-15 and 10-16 runs and task-ea2dfaac. I found nothing there but did not read them in full.

The draft's own control-window sentence ("pi-go ran only sprint-evaluator") shows the evaluator was the main victim all along.

Replacement for the D-FLEET-41 sentence "Since 10-10, 1 of 8 hit it (task-68e3cfb1, 10-27 ...)":

> Since 10-10, at least 4 of 8 evaluator runs logged the gap: three on 10-12 on `-go-claude-credit` (task-62beea26, task-f96e9162, task-f257f24d: "`make` isn't installed", "`make` is not installed") and task-68e3cfb1 (10-27, `box has no C compiler`; it ran `CGO_ENABLED=0` build and vet and scored 84). The first three do not match a `make: not found` filter; they were found by a prose search, so the count is a floor.

**F2. BLOCKING-IF-LEFT. Claim 4, "no missing toolchain on codex-go", misses a Go-on-PATH failure.**

task-401dce49 (codex-go sprint-executor, 10-29 14:51Z) logged:

- "`make test-changed`: cannot run tests (`go: not found`). `make lint`: unavailable golangci-lint. `make fmt-check`: unavailable gofmt."
- "Go lifecycle matrix ... UNMEASURED (no Go)."
- `validation_complete:false` in the handoff.

Its `make` calls ran (`make check-file-sizes` and `make check-changelog` passed), so make is present. The cause is probably a PATH reset by `bash -lc`. The 10-27 task ran `PATH=/usr/local/go/bin:$PATH make test-core`. An earlier run (10-16, task-cross-run) records "missing Go PATH". This is the same family ("executor cannot verify") but a different tool and a different cause.

It is not in the ticket's literal ask (make and a C compiler), so it does not block the resolve. It does falsify the sentence "all 20 sprint-executor starts ... logged no missing toolchain". Required edit to queue row 57 and the resolution text. Replace "with no missing-toolchain line" with:

> with no missing make or C compiler (a separate Go-not-on-PATH failure appears once on codex-go, task-401dce49 10-29, and is not this ticket)

Also add it to D-FLEET-41 as a one-line adjacent observation. Do not widen the row to cover it.

**F3. BLOCKING-IF-LEFT. Claim 7 dismisses the pi hits too quickly.**

`ailang-agent-executor-pi` is the non-Go image. It ran 34+ design-doc-creator, 9 triage and about 15 package and site tasks since 10-10. Its make-missing and go-missing lines recur: `make: command not found` on 10-13 and 10-16, `go: command not found` on 10-13 through 10-30 (at least 14 hits).

Those roles are not building Go, so "not the ticket's class" is defensible. But the class count ("is any other role or image still hitting missing make/cc") should say so explicitly. Add to D-FLEET-41:

> Non-Go planners on `ailang-agent-executor-pi` also log `make`/`go: command not found` (10-13..10-30). These are expected for that image and out of scope for this row.

**F4. NON-BLOCKING. Should the ticket be resolved at all?**

Yes. The ticket's subject is the sprint-executor image, and it asks for "make plus a C toolchain ... or the plan/skill must name a CGO_ENABLED=0 test subset". Both halves are verifiably built for the executor: #1753 for the image, #1905 for the skill rule. The evaluator residual is a different role and a different image, so it is correct to file it as a decision row and not hold the ticket open. Keeping it open parked on D-FLEET-41 would conflate two roles. Keep the resolve, with F2's wording fixed.

**F5. NON-BLOCKING. D-FLEET-41 options.**

- Option A names `Dockerfile.agent-pi-go` too, but pi-go has had 0 evaluator starts since 10-10 (26 starts in 10-01..10-09). Editing it is dead weight. Recommend: "A: add them to `Dockerfile.agent-go` (and `Dockerfile.agent-pi-go` only if the evaluator routes back there)." The same applies to the `go-claude-credit` job, which is on the same `agent-go` image.
- Option B's skill edit is in scope (verified above). It should copy the executor wording at `unattended_runs.md:58-64` into **both** `.claude/skills/sprint-evaluator/` and the `.agents` mirror. Say so.
- Default B is sensible: no spend, in scope, matches the 10-27 evaluator's behaviour (it reported the gap honestly and scored 84).
- Add the evaluator-side consequence the draft omitted. Three 10-12 runs spent turns discovering that `make` was absent and fell back to raw `go test` / scripts. This is wasted turns, not a failed gate, so the cost of B is modest.

**F6. NON-BLOCKING. Counts.**

"Tickets 33 → 32" and "Decisions 39 → 40, OPEN 24 → 25" depend on the ledger rows through D-FLEET-40, which exist. The draft numbering of D-FLEET-41 is consistent with the ledger's last row, D-FLEET-40. I did not recount the open tickets (that needs a `ticket` command, which I did not run).

**F7. NON-BLOCKING. Wording.**

"≥50 such lines on 10-08..10-09" in queue row 57: the draft's evidence says the 50 is the `--limit` over 2026-10-01..10-09, on codex-go and pi-go, not 10-08..10-09 on codex-go. Replace with "≥50 such lines on 10-01..10-09 (query `--limit`, so a floor), on codex-go and pi-go".

### Required edits for Record 1 (apply all)

- **E1.** The F1 replacement sentence in D-FLEET-41.
- **E2.** The F2 replacement in queue row 57 and the ticket resolution text ("no missing make or C compiler").
- **E3.** The F3 sentence in D-FLEET-41 and the F7 fix in row 57.
- **E4.** Option A names `Dockerfile.agent-go` first; option B names both skill copies (F5).

## Record 2

### Confirmed

- **Census (claim 1).** `git grep -n motoko_connection_probe -- ':!design_docs' ':!changelogs' ':!*.md' ':!.ailang/state'` returns:
  - `make/test.mk:128`, `:140`, `:141`
  - `scripts/test_check_referenced_paths.sh:45`, `:46`
  - the pair's own files
  
  The draft's line numbers are exact. The banked sprint JSON hits are excluded by my pathspec, but the draft is correct that they exist.
- **Fixture (claim 2).** `scripts/test_check_referenced_paths.sh:28-46` is `make_floor_fixture`, which writes `check18: ; @bash tools/eval/motoko_connection_probe.sh` and `check19` (the test script) into the fixture Makefile. `run_gate` passes `--repo-root "$REPO_ROOT"`. `check_referenced_paths.sh:90` does `[ ! -e "$REPO_ROOT/$path" ]` and then prints `missing referenced path`. So deleting the pair makes the floor fixture itself fail, which reds every arm that expects rc 0 (A3, A4 and others using the fixture). I read this and did not execute it (no scratch writes allowed), so mark it "by reading". The scope claim holds: `scripts/test_check_referenced_paths.sh` is not `scripts/test_mission_*`, so it is outside the fleet allowlist and outside D-FLEET-18 option A.
- **Last change (claim 3).** `git log` gives `6c03639f5 2026-09-06 test(motoko): complete suite-bound derivation (#1066)` for the pair. The probe alone was last changed `20cce785e` on 09-02. "Either file" is correct.
- **Kill switch (claim 4).** `~/.ailang/state/mission-motoko.disabled` exists (446 B, Sep 8). Note also `mission-motoko.pin-drift` dated Oct 27, which says nothing about re-enabling.
- **Remote branch (claim 5).** `git ls-remote origin refs/heads/sprint/motoko-iter39-armcount-r3` returns `f6750002d5762f4b8d3097f082a2bb700e60ea75`. Matches.
- **D-FLEET-18 consistency.** The row's option A is exactly "add `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`". The append is accurate to it.

### Findings

**F8. NON-BLOCKING. Fold vs separate row: folding is right.** Same two files, same owner question, same recommendation. A separate row would split a single allowlist decision. The ticket stays open, correctly, because the decision is Mark's. The draft's chosen disposition ("ticket stays open") is consistent with the Ticket-driven rule.

**F9. NON-BLOCKING. The "fleet could not retire it" claim is slightly over-strong.** It is true that deleting the pair needs the two fixture lines changed, and that file is outside option A. But a fleet-only retirement is possible if option A is widened by a third path, `scripts/test_check_referenced_paths.sh`. Add one clause to the D-FLEET-18 append: "Retiring it would need a third path added to A: `scripts/test_check_referenced_paths.sh`." This gives Mark a complete choice.

**F10. NON-BLOCKING. "Real at HEAD (no consumer outside its own suite)".** `make/test.mk:128` is the `test-launchd-drivers` CI job, which is a consumer of the test, not of the probe's output. The draft says so ("its own suite, `make/test.mk`, a self-test fixture"). I found no other consumer in the non-doc census. I did not inspect the doc and changelog hits excluded from the census, which are references only.

### Required edit for Record 2

- **E5.** Add the F9 clause to the D-FLEET-18 append. Not blocking on its own. Record 2 passes with or without it, but the clause makes the append complete.

## Summary

- Record 1: 72, PASS-WITH-EDITS. The core ghost claim is right (#1753 reached codex-go, proven behaviourally by `make test-core` and `make lint` runs on 10-16 and 10-27). The numbers about the sprint-executor are right. The failure is in the CLASS count: the evaluator hit the gap in at least 4 of 8 runs (not 1), a Go-on-PATH failure appeared on codex-go on 10-29, and the filter's wording could not have found either.
- Record 2: 85, PASS-WITH-EDITS (one optional clause).
- Both resolve/fold dispositions are correct as dispositions. The edits are about the evidence statements, not the actions.
