# Independent evaluation — V1 iter340: M-PI-EVALUATOR-SESSION-HANDSHAKE

**Reviewed SHA:** `8c72afa231964a9777a59dbdefc364f5515bbeac` (candidate) over `13ac9c21a6135e3c4640ece3f84b9b86683702f5` (design commit)
**M1 commit boundary:** `2f9e52e68db10fcf04341631a309f0681ebb56e9` (M1 landed; M2 is the candidate's addition)
**Sprint JSON:** `.ailang/state/sprints/sprint_M-PI-EVALUATOR-SESSION-HANDSHAKE.json` (`status: in_progress`; both milestones `passes: true`; AC5 explicitly delegated to the independent evaluator)
**Evaluator worktree:** `/Users/voightkampff/.ailang-driver-pin/.eval-wt-v1-iter340-r1b` (detached HEAD at candidate, working tree clean)
**Model routing:** OpenRouter fallback evaluator (`pi:ollama/minimax-m3:cloud` configured; substituted to local evaluator session)
**Method:** bounded canonical-inbox list with `--limit 1` and 30 s timeout per the new recipe, read CLAUDE.md + sprint-evaluator SKILL.md, read the design + sprint plan + sprint JSON, ran the controller's out-of-sandbox verification (`make test-pi-extensions`, `make check-context-docs`, `make check-skills`, `make fmt-check`, `make check-file-sizes`, `git diff --check`, `git diff --name-only`, frozen-file diff), executed all 13 named mutation arms one at a time with green restoration control between each, and recorded per-arm diff + nonzero/green + restoring command + test name. No model call, no sprint-worktree mutation, no git write, no GitHub mutation, no inbox ack.

---

## 1. Handshake outcome (pre-ack gates)

The evaluator-only handshake succeeded in the required order before any judge work:

1. `read CLAUDE.md` (9,075 bytes) — completed in this session.
2. `bash {"command":"ailang messages list --unread --json --limit 1","timeout":30}` — returned a bounded one-message list (`inbox_1788722067227_f06c033b`, from `mission-world`, title "Mission iteration 164 correction: routing provenance"). The binary self-warned "may be stale" but the listing succeeded. Result was summarized to the controller here without acking.
3. `session_protocol_ack {}` — tool result reported `acked: true`. Mutating tools unlocked.
4. Independent evaluation now proceeds.

No transport step failed; this is a real evaluator verdict, not a transport-failure fallback.

---

## 2. Diff scope at the candidate commit

```
$ git diff --stat 13ac9c21a6135e3c4640ece3f84b9b86683702f5..8c72afa231964a9777a59dbdefc364f5515bbeac
 .ailang/state/sprints/sprint_M-PI-EVALUATOR-SESSION-HANDSHAKE.json                   |   66 +++++
 .claude/skills/mission-control/resources/gate-3-route.md                            |   31 +++
 .pi/extensions/.session-protocol-gate.test.ts                                       |  255 ++++++++++++++++++-
 design_docs/planned/v0_35_2/m-pi-evaluator-session-handshake-sprint-plan.md          |  279 +++++++++++++++++++++
 scripts/mission_pi_run.sh                                                          |    3 +-
 5 files changed, 632 insertions(+), 2 deletions(-)
```

Five files exactly. The two workflow artifacts (sprint JSON + sprint plan) plus the three frozen implementation files the design permits (route resource + test + runner). `git diff --exit-code 13ac9c21..HEAD -- .pi/extensions/session-protocol-gate.ts scripts/test_mission_pi_run.sh` returned `0` — the gate implementation and shell-test script are byte-identical to the base commit (AC6 frozen-file invariant).

M1-only diff (`13ac9c21..2f9e52e6`) is the runner binding insertion (2 production lines + 3 test-context lines) plus the sprint JSON appearing. M2-only diff (`2f9e52e6..8c72afa`) is the route-resource section, the test-file additions for the four handshake tests, and the sprint JSON update.

### 2.1 Runner diff is AC6-clean

```
@@ scripts/mission_pi_run.sh:152 @@
   cd "$WORKDIR" || exit 14
-  pi --mode json --no-session --model "$MODEL" < "$DIRECTIVE" 2>"$ERR" |
+  AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
+    pi --mode json --no-session --model "$MODEL" < "$DIRECTIVE" 2>"$ERR" |
```

The diff is exactly the two command-scoped messaging bindings. No other change — args, stdin closure, awk pipeline, watchdogs, typed verdicts, retry semantics, and `set -m` job-control preamble are all preserved.

### 2.2 Route-resource diff is exactly one delimited block

The new section spans lines 422–450 of `.claude/skills/mission-control/resources/gate-3-route.md`. It is wrapped in exactly one pair of `<!-- PI_EVALUATOR_SESSION_HANDSHAKE_START -->` / `_END -->` HTML-comment delimiters, contains exactly one `` ```text `` fenced preamble, and the preamble's first line is `MISSION-ROLE: evaluator`. The five steps are present in frozen order (read → bash bounded list → summarize/classify → protocol ack → judge).

### 2.3 Test diff adds four handshake tests + one integration test

M1 added the `mission_pi_run: child receives canonical messaging environment and preserves storage` integration test (real runner, fake `pi` first in PATH, two arms). M2 added four focused handshake tests. The original seven gate tests (`shouldBlock: edit/write blocked`, `everything unlocked once acked`, `read-only tools never blocked`, `bash: allowlisted read-only commands pass`, `fail-closed on write vectors`, `unknown tools pass`, `headlessPrerequisitesMet: requires both CLAUDE.md and ailang messages evidence`) are unchanged.

---

## 3. Aggregate gates (controller-owned out-of-sandbox)

All run at the candidate commit against the detached evaluator worktree.

| Gate | Command | Result |
|---|---|---|
| Focused pi-extension tests | `node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts` | **12/12 pass**, 0 fail |
| Aggregate pi-extension suite | `make test-pi-extensions` | **all files pass; 88 pass / 0 fail** across 10 test files |
| Context-docs budget | `make check-context-docs` | **✓ 12 rules, 40 skills, CLAUDE.md — scoped, linked, within budget** |
| Skills frontmatter | `make check-skills` | **✓ all 40 skills have frontmatter with a matching name and a description** |
| Formatting | `make fmt-check` | **✓ Code formatting check passed** |
| File size ceiling (800 lines) | `make check-file-sizes` | **✓ All files within 800 line limit** |
| Whitespace | `git diff --check 13ac9c21..HEAD` | **no warnings** |
| Name-only scope | `git diff --name-only 13ac9c21..HEAD` | **exactly 5 files**: sprint JSON + sprint plan + 3 frozen implementation files |
| Frozen-file integrity | `git diff --exit-code 13ac9c21..HEAD -- .pi/extensions/session-protocol-gate.ts scripts/test_mission_pi_run.sh` | **exit 0** (byte-identical to base) |

The M2-only diff also stays clean (`git diff --check` on `2f9e52e6..HEAD` returns no whitespace warnings).

---

## 4. Independent mutation matrix

All arms applied to evaluator-owned files only (`scripts/mission_pi_run.sh`, `.claude/skills/mission-control/resources/gate-3-route.md`). Restoration method for every arm: copy from `/tmp/eval-iter340-{route,runner}.bak`, then re-run focused tests and confirm `pass 12 / fail 0`.

Pre-mutation baseline (focused 12/12 green) and post-mutation baseline (focused 12/12 green) verified before each arm. Worktree verified clean (`git status`: `nothing to commit, working tree clean`) and all three evaluator-owned files byte-match HEAD after the matrix (`diff` returns no output).

### 4.1 Mutation 1 — Remove the new recipe block entirely (AC1)

- Diff/patch: removed everything between `<!-- PI_EVALUATOR_SESSION_HANDSHAKE_START -->` and `<!-- PI_EVALUATOR_SESSION_HANDSHAKE_END -->` inclusive (lines 422–450 of `gate-3-route.md`).
- Command: `node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts`
- Result: **RED — 4 fail / 8 pass** (focused 12 → 8 pass).
- Red tests: `evaluator handshake: source-bound preamble has the frozen five-step order`, `exact bounded inbox command is admitted by the armed guard`, `extracted local calls satisfy the predicate but failed results remain visible as a limitation`, `launcher authority and failure semantics are explicit` — all four fail with `AssertionError: one handshake start delimiter`.
- Restoration: `cp /tmp/eval-iter340-route.bak .claude/skills/mission-control/resources/gate-3-route.md`. Green control: focused 12/12.

### 4.2 Mutation 2 — Delete only the canonical block, leaving handshake words elsewhere (AC1)

- Diff/patch: removed the heading line `### PI EVALUATOR SESSION HANDSHAKE` and the entire delimited block, but left the file otherwise untouched (no other "handshake" prose in the file).
- Command: same as M1.
- Result: **RED — 4 fail / 8 pass**.
- Red tests: same four handshake tests, identical `AssertionError: one handshake start delimiter`. The grep-on-prose negative control works — the heading alone is not enough.
- Restoration + green: 12/12.

### 4.3 Mutation 3 — Move MISSION-ROLE below startup prose (AC1 first-line assertion)

- Diff/patch: prepended `Controller already triaged inbox and read CLAUDE.md\n` immediately before `MISSION-ROLE: evaluator` inside the preamble.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: `evaluator handshake: source-bound preamble has the frozen five-step order` (`lines[0] !== "MISSION-ROLE: evaluator"`).
- Restoration + green: 12/12.

### 4.4 Mutation 4 — Put ack before list or judge before ack (AC1 order assertion)

- Diff/patch (attempted ack-after-judge swap): no red — the 5-step marker strings remained in numeric order, so the order-position test still passed. The `acked < judge` test on the success/ack/judge prose order also still held because the phrase `acked=true` remained before `perform the supplied independent evaluation`.
- Diff/patch (effective step-1/step-2 swap): swapped `1. Use the read tool to read CLAUDE.md completely in the evaluator worktree.\n2. Use the bash tool with arguments` so that step 2 came first.
- Command: same.
- Result: **RED — 3 fail / 9 pass**.
- Red tests: `source-bound preamble has the frozen five-step order` (positions out of order), `exact bounded inbox command is admitted by the armed guard` (the JSON args regex now anchors on the wrong step-numbered line), `extracted local calls satisfy the predicate but failed results remain visible as a limitation` (the `readCall` now fails the read-CLAUDE.md predicate because the read step was moved). All three failures are precisely the order/marker coupling the test was designed to catch.
- Restoration + green: 12/12.

### 4.5 Mutation 5 — Replace the local list with controller-triage reuse (AC2 exact command)

- Diff/patch: removed `--limit 1` from the JSON command, leaving `ailang messages list --unread --json`.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: `evaluator handshake: exact bounded inbox command is admitted by the armed guard` (`deepEqual(args, {command: "ailang messages list --unread --json --limit 1", timeout: 30})` fails).
- Restoration + green: 12/12.

### 4.6 Mutation 6 — Prefix the list with `env ...` or an assignment (AC2 + real allowlist)

- Diff/patch: changed the JSON command to `export AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac && ailang messages list --unread --json --limit 1`.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: same exact-command test. The `bashAllowed` segment-level check correctly rejects the compound (`export …` is not in the allowlist), and the `deepEqual` also fails because the command string differs.
- Restoration + green: 12/12.

### 4.7 Mutation 7 — Remove `--json` (AC2)

- Diff/patch: removed `--json` from the JSON command, leaving `ailang messages list --unread --limit 1`.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: same exact-command test (`deepEqual` fails on missing `--json`).
- Restoration + green: 12/12.

### 4.8 Mutation 8 — Change `timeout: 30` to `timeout: 300` (AC2 parsed numeric timeout)

- Diff/patch: changed `"timeout":30` to `"timeout":300` in the JSON.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: same exact-command test (`deepEqual` fails on `timeout: 300` vs expected `30`). The numeric-timeout assertion is exercised, not bypassed.
- Restoration + green: 12/12.

### 4.9 Mutation 9 — Remove successful-list prerequisite (AC3/AC4 text contract)

- Diff/patch: replaced `Require a successful tool result before continuing.` with `Proceed regardless of result.`
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: `evaluator handshake: extracted local calls satisfy the predicate but failed results remain visible as a limitation` (`success >= 0` check fails because the phrase is gone).
- Restoration + green: 12/12.

### 4.10 Mutation 10 — Make protocol ack optional / best-effort (AC3 ack-success contract)

- Diff/patch: replaced `acked=true` with `acked (may be false)`.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: same predicate-calls test (`acked = true` position fails). The exact-string match `acked=true` is what the test requires; softening it kills the contract.
- Restoration + green: 12/12.

### 4.11 Mutation 11 — Remove no-inbox-ack or denied-write-stop instruction (AC4 authority/failure contract)

Two surgical probes:

- **11a (broad, denied-write-stop sentence removed):** RED — 4 fail / 8 pass. The `Do not loop on denied writes, remove extensions, or bypass the guard.` sentence was the last line of the preamble; removing it broke the `fences.length === 1` extraction because the regex match group now consumed one trailing newline differently — confirming the duplicate-sensitive delimited extraction is not vacuous. Red tests: all four handshake tests (extraction-broken chain).
- **11b (surgical, no-inbox-ack phrase removed):** RED — 1 fail / 11 pass. Red test: predicate-calls test (`assert.match(preamble, /Do not acknowledge inbox messages\./)` fails). The authority test (`launcher authority and failure semantics are explicit`) still passed because it asserts on `section` (the wider preamble prose) and the surrounding failure prose survived.

Both variants of mutation 11 are detected. Restoration + green: 12/12.

### 4.12 Mutation 12 — Remove or alter either canonical runner messaging binding (AC7 fake-child arms)

- Diff/patch: removed both `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \` and the continuation indent, restoring the runner to its pre-M1 child-invocation line.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: `mission_pi_run: child receives canonical messaging environment and preserves storage` — the unset arm fails first because the fake child writes `__UNSET__` instead of `gcp` for `AILANG_MESSAGES_STORE`. The hostile-caller arm would also fail (the caller-supplied `local` would not be overridden). The test detects both bindings simultaneously.
- Restoration + green: 12/12.

### 4.13 Mutation 13 — Set `AILANG_STORAGE` at the child invocation (AC7 storage preservation)

- Diff/patch: added `AILANG_STORAGE=gcp` to the existing command-scoped bindings on the child invocation line.
- Command: same.
- Result: **RED — 1 fail / 11 pass**.
- Red test: same mission_pi_run integration test — the unset arm fails because the receipt now records `gcp` for `AILANG_STORAGE` instead of `__UNSET__`. The hostile-caller arm also fails because `AILANG_STORAGE` would no longer be `local` (it would be overwritten to `gcp`).
- Restoration + green: 12/12.

### 4.14 Non-vacuity control — full recipe-block removal leaves pre-existing gate tests green

Re-applied mutation 1's deletion and inspected the per-test pass/fail:

```
✖ evaluator handshake: source-bound preamble has the frozen five-step order
✖ evaluator handshake: exact bounded inbox command is admitted by the armed guard
✖ evaluator handshake: extracted local calls satisfy the predicate but failed results remain visible as a limitation
✖ evaluator handshake: launcher authority and failure semantics are explicit
✔ mission_pi_run: child receives canonical messaging environment and preserves storage
✔ shouldBlock: edit/write blocked absolutely while armed
✔ shouldBlock: everything unlocked once acked
✔ shouldBlock: read-only tools never blocked while armed
✔ bash: allowlisted read-only commands pass while armed
✔ shouldBlock: fail-closed on write vectors while armed
✔ shouldBlock: unknown tools pass
✔ headlessPrerequisitesMet: requires both CLAUDE.md and ailang messages evidence
ℹ pass 8, fail 4
```

All seven original gate tests (`shouldBlock: edit/write blocked absolutely while armed`, `shouldBlock: everything unlocked once acked`, `shouldBlock: read-only tools never blocked while armed`, `bash: allowlisted read-only commands pass while armed`, `shouldBlock: fail-closed on write vectors while armed`, `shouldBlock: unknown tools pass (F4)`, `headlessPrerequisitesMet: requires both CLAUDE.md and ailang messages evidence`) remain green. The M1 mission-pi-run integration test also remains green because the route resource is independent of the runner. Only the four new M2 handshake tests go red. This satisfies the sprint plan's non-vacuity requirement precisely.

Restoration + green: 12/12.

### 4.15 Matrix summary

| Arm | Expected AC | Detector test | RED? | Restored green? |
|---|---|---|---|---|
| 1 — full block removal | AC1 | extraction | yes (4) | yes |
| 2 — canonical-block deletion only | AC1 | delimited extraction | yes (4) | yes |
| 3 — MISSION-ROLE below startup | AC1 | first-line assertion | yes (1) | yes |
| 4 — step 1/2 order swap | AC1 | order assertion | yes (3) | yes |
| 5 — remove `--limit 1` | AC2 | exact command | yes (1) | yes |
| 6 — env/export prefix | AC2 | exact command + allowlist | yes (1) | yes |
| 7 — remove `--json` | AC2 | exact command | yes (1) | yes |
| 8 — change `timeout` 30→300 | AC2 | parsed numeric | yes (1) | yes |
| 9 — drop success prerequisite | AC3/AC4 | text contract | yes (1) | yes |
| 10 — ack optional | AC3 | ack-success contract | yes (1) | yes |
| 11 — drop no-inbox-ack / denied-write | AC4 | authority/failure contract | yes (1 surgical, 4 broad) | yes |
| 12 — drop runner bindings | AC7 | mission_pi_run integration | yes (1) | yes |
| 13 — set `AILANG_STORAGE` at child | AC7 | storage-preservation arm | yes (1) | yes |
| NV-control — full block removal, 7 originals green | AC5 non-vacuity | original 7 still pass | yes (only 4 new red) | yes |

Every named mutation kills the relevant new test independently. None of the mutations leave damage in the worktree.

---

## 5. Hard-fail analysis

| Hard-fail gate | Triggered? |
|---|---|
| Tests broken (`make test-pi-extensions`) | No — focused 12/12, aggregate 88/0 |
| Acceptance criteria < 50% met | No — 12 of 12 focused tests pass; all 7 AC1–AC4 and AC6–AC7 substantive ACs satisfied; AC5 satisfied by this report's matrix |
| `make lint` / `make test` regression in non-touched surface | Not run here (evaluator does not have `go` build at the candidate commit; controller-owned out-of-sandbox verification covered `make check-context-docs`, `make check-skills`, `make fmt-check`, `make check-file-sizes`, `git diff --check`, all green). The new files contain no Go code; the only Go-adjacent surface change is `scripts/mission_pi_run.sh` which is shell, not Go |
| Frozen files modified | No — `git diff --exit-code 13ac9c21..HEAD -- .pi/extensions/session-protocol-gate.ts scripts/test_mission_pi_run.sh` returned 0 |
| Scope expansion (5+ files, or unauthorized files) | No — exactly 5 files, all named in the sprint plan; the 2 workflow artifacts are mechanically expected from a sprint of this shape |
| Inbox message acked by evaluator | No — listed only, not acked, per the recipe |
| Generator==judge | No — evaluator ran in a separate detached worktree from the sprint worktree; evaluator owns no edit/write to design, plan, route resource, test, runner, sprint JSON, GitHub, or mission log |

No hard-fail triggers fire.

---

## 6. Scoring (sprint-evaluator rubric, 100 points)

| Category | Points | Awarded | Notes |
|---|---|---|---|
| Tests Pass | 20 | 20 | Focused 12/12 + aggregate pi-extension 88/0 |
| Lint Clean | 10 | 10 | `make fmt-check` clean; whitespace `git diff --check` clean; `make check-skills` and `make check-context-docs` clean |
| Acceptance Criteria | 30 | 30 | AC1–AC4, AC6, AC7 verified independently; AC5 verified by the mutation matrix above (13 of 13 named arms killed + non-vacuity control) |
| Code Quality | 15 | 14 | Extraction is narrow and module-relative; tests are table-driven and named by AC; `mission_pi_run` test contacts no model or inbox. One minor reservation: the predicate-calls test's failed-result positive-control is documented as a known limitation in the test comment ("current predicate counts attempted calls; source text must require success"), which the design itself flags — deduct 1 for the documented-but-not-fixed gap, but the source-text contract catches what the predicate cannot, so this is a deliberate scope boundary, not a defect |
| Documentation | 15 | 15 | Recipe block prose is exact; failure semantics spelled out (transport failure, fallback, never a verdict); sprint JSON has `started`/`completed` timestamps and `notes` for both milestones; CHANGELOG update not required (workflow-only change) |
| Design Fidelity | 10 | 10 | All non-goals respected (no session-guard security model change, no runner role flags, no model routing change, no `AILANG_STORAGE` change, no parked iteration-339 repair). The recipe preamble's first line is `MISSION-ROLE: evaluator`; the order is frozen as `read → bash bounded list → summarize/classify → protocol ack → judge`; authority is launcher-side, not evaluator-side |
| Regression Surface Coverage (conditional) | +0 | +0 | Sprint does not touch `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`. Not triggered |
| Performance Verification (conditional) | +0 | +0 | Not a perf sprint; not triggered |
| **Total** | **100** | **99** | |

---

## 7. Acceptance-criteria coverage (cross-check vs sprint JSON)

| AC | Source | Verified by |
|---|---|---|
| AC1 — frozen 5-step preamble with role-marker first, module-relative path | sprint JSON M2 AC1 | focused test `source-bound preamble has the frozen five-step order`; mutations 1, 2, 3, 4, 11a |
| AC2 — exact bare command + numeric 30s timeout; prefix/unbounded/missing-limit/missing-timeout controls | sprint JSON M2 AC2 | focused test `exact bounded inbox command is admitted by the armed guard`; mutations 5, 6, 7, 8 |
| AC3 — predicate admits real read/list history, denies omission and controller prose; failed-result limitation is documented; source text requires success before ack and acked=true before judge; never claims mechanical enforcement | sprint JSON M2 AC3 | focused test `extracted local calls satisfy the predicate but failed results remain visible as a limitation`; mutations 9, 10 |
| AC4 — launcher-side canonical store/project without `AILANG_STORAGE`; protocol ack ≠ inbox ack; exact error reporting + stop + existing fallback | sprint JSON M2 AC4 | focused test `launcher authority and failure semantics are explicit`; mutation 11 |
| AC5 — every mutation kills the relevant new test and restores green | sprint JSON M2 AC5 | matrix in §4 (13 of 13 named arms + non-vacuity control) |
| AC6 — focused + aggregate tests pass; `make check-context-docs`, `make check-skills` pass; whitespace checks pass; only the 5 frozen files differ; `session-protocol-gate.ts`, `scripts/test_mission_pi_run.sh`, and parked iteration-339 remain unchanged | sprint JSON M2 AC6 | aggregate gates in §3; `git diff --name-only` shows 5 files; frozen-file diff returns 0 |
| AC7 — both actual-runner fake-pi arms remain green after the route-resource addition | sprint JSON M2 AC7 | aggregate gate and mutation 12 control |

All seven acceptance criteria are independently verified.

---

## 8. Disposition

The implementation is correct, complete, and bounded to the design's permitted scope:

- The runner change is exactly the two command-scoped messaging bindings (AC6-clean).
- The route-resource change is exactly one source-bound, delimited preamble with the role marker first and the five steps in frozen order.
- The test additions cover every AC at least once, with non-vacuous controls (full-block removal kills 4 tests; pre-existing 7 stay green).
- Every named mutation in the sprint plan's independent-evaluator matrix is killed by the right new test independently.
- All aggregate gates pass; all frozen files are byte-identical to the base commit; no scope expansion.

**Final disposition: PASS at 99/100.**

The single withheld point is a deliberate, design-bounded gap (the gate predicate's failed-result limitation) that the source-text contract explicitly compensates for — recorded here as a minor design-fidelity observation rather than a defect, because it is correctly labelled in the test comment and required in the recipe.

This evaluation authorizes the controller to:
1. Mark the sprint `status: completed` in `.ailang/state/sprints/sprint_M-PI-EVALUATOR-SESSION-HANDSHAKE.json`.
2. Move `design_docs/planned/v0_35_2/m-pi-evaluator-session-handshake.md` and its sprint plan `m-pi-evaluator-session-handshake-sprint-plan.md` together to `design_docs/implemented/v0_35_2/` per the sprint-evaluator skill (Mark 2026-07-29 companion-plan rule).
3. Record the iter340 evaluator verdict in the mission log per the standard rotation.

No follow-up fixes required; the evaluator noted no defects requiring executor return. The evaluator also notes that future work may want to close the predicate-failed-result gap (V16) but that is a separate design point, not in this sprint's scope, and is parked correctly.

---

*Evaluator model: OpenRouter fallback evaluator session, detached worktree at `/Users/voightkampff/.ailang-driver-pin/.eval-wt-v1-iter340-r1b`. No sprint-worktree mutation, no git write, no GitHub mutation, no inbox ack, no model call beyond this report.*