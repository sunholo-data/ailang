# Fleet iteration 76 — draft records, as handed to the judge

Pin / base: `bf4f14473f704f57fd3c6c5454fc8eaf115c505e` (origin/dev at Gate 1, 2026-10-31T15:07Z).
Evidence files in this directory were written by the controller from prod Cloud Logging
(`ailang-multivac`) on 2026-10-31.

## Record 1 — `executor-image:no-make-no-cc-cgo-tests` (attended, 10-08)

Ticket text (abridged): sprint-executor task-69be551b (M-TASK-INPUTS, dispatched 2026-10-08 19:08
on prod) blocked: "make and a C compiler are missing … SQLite tests require CGO." Ask: "the
sprint-executor image must ship make plus a C toolchain for CGO (sqlite) tests, or the plan/skill
must name a CGO_ENABLED=0 test subset."

### Proposed disposition

RESOLVE the ticket as a ghost of #1753 (`939beb6fd`), and file D-FLEET-41 for a residual on the
sprint-evaluator's images.

### Claims (each with its source)

1. task-69be551b ran on Cloud Run job `ailang-agent-executor-codex-go` (`model=gpt-6.1-sol`), started
   19:09:18Z, published `blocked` 19:12:19Z with `files=0`. Source:
   `prod-logs-task-69be551b-lifecycle.txt`.
2. #1753 (`939beb6fd`, 2026-10-09 19:20 +0200, "Add authorized cloud task inputs and fix codex-go
   build prerequisites") is the M-TASK-INPUTS sprint the blocked task was executing, and it added
   `make`, `gcc`, `libc6-dev` to `docker/Dockerfile.agent-codex-go` (lines 6-13 at HEAD).
3. The live `ailang-agent-executor-codex-go` job image is
   `agent-codex-go@sha256:cdc075e2…`, whose latest execution was created 2026-10-29 (`gcloud run
   jobs describe`). (Controller did not prove the digest contains #1753; behavioural evidence is
   claim 4.)
4. Since 2026-10-10, every sprint-executor cloud start ran on codex-go: 20 starts, 18 distinct
   tasks (two tasks started twice — Cloud Run retry). Source:
   `prod-logs-sprint-roles-starts-since-1010.txt`. The missing-toolchain filter (see the header of
   `prod-logs-missing-toolchain-before-after.txt`) returns **0** codex-go lines from
   2026-10-09T18:00Z to now, against **50** lines (the query's `--limit`, so a floor) in
   2026-10-01..2026-10-09T18:00Z, on codex-go and pi-go.
5. The skill half of the ask is already built: #1905 `3f9a0c092` added the "Never install
   toolchains … record the exact skipped test and missing tool … leave that verification to CI"
   rule (`.claude/skills/sprint-executor/resources/unattended_runs.md:58-64`). (Iteration 75 found
   this; re-read at HEAD this iteration.)
6. **Residual — the sprint-evaluator.** Its cloud starts: 26 on `ailang-agent-executor-pi-go` in
   10-01..10-09, 0 there since; 8 since 10-10 on `ailang-agent-executor-go` (2) and
   `ailang-agent-executor-go-claude-credit` (6), both on image `agent-go@sha256:da132413…`. Neither
   `docker/Dockerfile.agent-go` nor `docker/Dockerfile.agent-pi-go` (nor their bases
   `Dockerfile.agent` / `Dockerfile.agent-pi` / `Dockerfile.agent-base:48-56`) installs make or gcc.
   Post-fix, 1 of the 8 evaluators logged the gap: task-68e3cfb1 (2026-10-27, evaluating
   task-9c11d207's sprint) printed `box has no C compiler`, ran `CGO_ENABLED=0` build and vet, and
   returned `EVALUATION_SCORE: 84/100`. The pre-fix pi-go lines in the control window are
   sprint-evaluator runs (pi-go ran only sprint-evaluator in that window).
7. The post-fix pi lines (`ailang-agent-executor-pi`, 10-13 and 10-16) are design-doc-creator tasks
   (task-4047d865, task-e8ae18c6) on a non-Go image; not the ticket's class.
8. The sprint-evaluator skill has no missing-toolchain guidance (`grep -rni 'cgo\|C compiler'
   .claude/skills/sprint-evaluator/` → 0); only the executor skill has it.
9. `docker/**` is outside the fleet's scope guard (charter § Authority).

### Draft ticket resolution text

"Ghost: fixed by #1753 (939beb6fd, 10-09), which added make, gcc and libc6-dev to the codex-go image
the sprint-executor runs on; the skill half is #1905 (3f9a0c092). Since 10-10, 20 sprint-executor
starts (18 tasks), all codex-go, logged no missing toolchain. The sprint-evaluator's images (agent-go,
earlier agent-pi-go) still lack make and gcc: D-FLEET-41 (fleet iteration 76)."
`--sha 939beb6fd…` (on origin/dev).

### Draft queue row 57

57. [LANDED · iteration 76, ghost] `executor-image:no-make-no-cc-cgo-tests` (attended, 10-08):
already fixed by #1753 → `939beb6fd` (10-09), which added make, gcc and libc6-dev to the codex-go
image the ticketed task ran on. Since 10-10 all 20 sprint-executor starts ran there with no
missing-toolchain line, against ≥50 such lines on 10-08..10-09. The skill half is #1905
`3f9a0c092`. Ticket resolved. The sprint-evaluator's images (agent-go now, agent-pi-go before
10-10) still lack both: D-FLEET-41.

### Draft decision row D-FLEET-41

| D-FLEET-41 | OPEN | **`executor-image:no-make-no-cc-cgo-tests` is fixed for the sprint-executor, but the cloud
sprint-evaluator has always run on an image with no make and no C compiler. Who adds them?**
Measured at iteration 76 on `bf4f14473` from prod Cloud Logging. #1753 (`939beb6fd`) gave codex-go
make, gcc and libc6-dev; since 10-10 all 20 sprint-executor starts ran there and none logged a
missing toolchain. The evaluator ran 26 times on `agent-pi-go` (10-01..10-09) and 8 times on
`agent-go` since (`ailang-agent-executor-go`, `-go-claude-credit`). Neither Dockerfile installs make
or gcc. Since 10-10, 1 of 8 hit it (task-68e3cfb1, 10-27, `box has no C compiler`): it ran
`CGO_ENABLED=0` build and vet and scored 84 without the CGO tests. The evaluator skill has no
missing-toolchain rule; the executor's is `unattended_runs.md:58-64`. `docker/**` is outside fleet
scope. **A (recommended):** an attended session adds the same three packages to
`Dockerfile.agent-go` and `Dockerfile.agent-pi-go`, as #1753 did for codex-go. **B:** no image change;
evaluators skip CGO tests and leave them to CI, and the fleet adds the executor's missing-toolchain
rule to the sprint-evaluator skill (in scope). Default if unanswered: B. | OPEN (filed 2026-10-31,
iteration 76; `fleet-mission-evidence/iteration76/`) |

## Record 2 — `eval:motoko-connection-probe-orphaned` (motoko, 10-01), bookkeeping second pick

Ticket text (abridged): `tools/eval/motoko_connection_probe.sh` was the instrument for the fork-era
fmt remeasurement, whose premise was superseded. Nothing uses it except its own test suite,
`make/test.mk` (test-launchd-drivers, the "launchd drivers (bash 3.2)" CI job) and
`scripts/test_check_referenced_paths.sh`. Unmerged work sits on `sprint/motoko-iter39-armcount-r3`.
Decide: retire the probe and its CI target, or keep it with an owner.

### Proposed disposition

Fold into D-FLEET-18 (already OPEN, recommendation A = add exactly these two files to the fleet's
allowlist). Ticket stays open. Queue row 58.

### Claims

1. `git grep -n motoko_connection_probe -- ':!design_docs' ':!changelogs' ':!*.md'` at HEAD: outside
   the pair itself, the only hits are `make/test.mk:128` (`LAUNCHD_SERIAL_SUITES`), `:140-141`
   (`bash -n` lines), `scripts/test_check_referenced_paths.sh:45-46` (fixture lines), and banked
   sprint JSONs under `.ailang/state/sprints/`.
2. The fixture in `scripts/test_check_referenced_paths.sh` (`make_floor_fixture`, used by A2–A8)
   names both paths as existing referenced paths, so deleting the pair would red that self-test
   until its two lines change; that file is outside D-FLEET-18 option A's two paths and outside the
   fleet's scope.
3. Last change to either file: `6c03639f5` (2026-09-06).
4. Motoko's kill switch `~/.ailang/state/mission-motoko.disabled` exists (disabled).
5. `refs/heads/sprint/motoko-iter39-armcount-r3` is still on origin at `f6750002d`.

### Draft D-FLEET-18 append

**Re-measured at iteration 76 on `bf4f14473`, folding in `eval:motoko-connection-probe-orphaned`
(motoko, 10-01):** that ticket asks to retire or own the same two files. Outside the pair, only
`make/test.mk:128,140-141` and two fixture lines in `scripts/test_check_referenced_paths.sh:45-46`
name them (plus banked sprint JSONs). Last change `6c03639f5` (09-06). Motoko is still disabled;
`sprint/motoko-iter39-armcount-r3` is still on origin. Under A the fleet could own the pair. It
could not retire it: deleting the files reds the fixture until those two lines change, and that
file is outside A.

### Draft queue row 58

58. [D-FLEET-18 class] `eval:motoko-connection-probe-orphaned` (motoko, 10-01): real at HEAD (no
consumer outside its own suite, `make/test.mk` and a self-test fixture). Retire-or-own is D-FLEET-18's
question; folded there. Ticket stays open.

## Counts after these records

Tickets: 33 open → 32 (one resolved). Decisions: 39 rows → 40, OPEN 24 → 25.
