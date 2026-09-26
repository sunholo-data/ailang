# Sprint Plan: m-autopush-gate-followups

## Summary

Close all five measured, non-blocking follow-ups from the committed-Go auto-push gate without changing its fast-forward-only contract. The sprint first isolates the test harness from the fleet evidence log, then aligns the hook and CI formatting gates, makes Git pathname transport lossless, makes early refusals observable, and gives both hook scripts a durable scoped ShellCheck gate.

**Duration:** 1 iteration (about 1 focused day)
**Estimated change:** 260 LOC (implementation, tests, and CI wiring)
**Dependencies:** `d2ef77e09`; no design document is owed for these verified defects
**Risk level:** Medium — shell/Git pathname handling and a shared integrity gate require mutation-specific tests

## Measured Baseline and Planning Corrections

- Pinned planning base: `2b5750ad9865f636768df5a277d8d1f74d166218` (the iteration-326 record). `origin/dev` has since advanced, so executor must rebase/refresh through the controller rather than switch this worktree.
- `make fmt-check` -> rc 0 at base, despite both an empty `.go` file and a non-empty unparseable `.go` file producing no `gofmt -l` stdout while returning rc 2. The repair must check the formatter exit code as well as its filename output.
- `bash -n scripts/hooks/push_dev_on_stop.sh` -> rc 0 at base.
- `shellcheck scripts/hooks/push_dev_on_stop.sh` -> rc 0 at base.
- `shellcheck scripts/hooks/test_push_dev_on_stop.sh` -> rc 1 at base. The prompt's count of three SC2164 warnings is **refuted** on the exact pinned commit: there are eight, at lines 10, 29, 30, 62, 87, 91, 97, and 106. M1 fixes all eight; a three-site repair would leave M5 red.
- Controller baseline: `HOME=$(mktemp -d) bash scripts/hooks/test_push_dev_on_stop.sh "$PWD/scripts/hooks/push_dev_on_stop.sh"` -> rc 0, `18 passed, 0 failed`. An unisolated run in the planner sandbox is rc 1, `17 passed, 1 failed`, solely because the sandbox forbids the synthetic merge-in-flight log write to the real home. This is environment evidence for M1, not a product regression.
- The `launchd drivers (bash 3.2)` workflow only runs `make test-launchd-drivers`, whose coverage and syntax loops name `tools/launchd/**`, two `tools/eval/**` files, and mission decision scripts. It does **not** cover `scripts/hooks/**`.
- Repo-wide ShellCheck is rc 1 at base: 399 tracked `.sh` files yield 629 diagnostic lines across 188 files. Excluding the two autopush scripts, 187 other files still report findings. M5 therefore scopes the gate to an explicit two-file list instead of expanding this sprint into unrelated cleanup.

## Milestone Order

M1 is deliberately first: every later harness run must stop writing synthetic rows to the shared `~/.ailang/state/autopush.log`. M2 changes both sides of the formatting parity contract in one independently committable milestone. M3 and M4 then harden input transport and observability. M5 wires the already-green named file list into CI.

## M1 — Isolate Harness HOME and Make Harness Control Flow ShellCheck-Clean

**Finding:** (e), plus the baseline repair required by (d)

### What

- Immediately after creating the harness workspace, create a private home under it and export `HOME` before the first hook invocation.
- Keep the private home inside the existing cleanup boundary.
- Add an arm that proves hook rows appear under the private home and that a caller-supplied sentinel log outside it is byte- and line-count-identical before/after the suite.
- Add `|| exit 1` to all eight unchecked harness `cd` sites reported by ShellCheck. This is test-harness fail-fast behavior, not a production semantic change.

### Why

Synthetic push/refusal rows are currently indistinguishable from fleet evidence. Fixing this first prevents the rest of the sprint from adding more pollution. Fixing all eight SC2164 sites establishes the green baseline M5 will later enforce.

### Files touched

- `scripts/hooks/test_push_dev_on_stop.sh`

### Test rows and killed mutations

| Test name | Observable | Mutation it kills |
|---|---|---|
| `N harness HOME: private log populated` | `$HOME/.ailang/state/autopush.log` exists and contains expected synthetic repo rows | Remove or move `export HOME="$W/home"` until after hook calls -> private log is absent/empty |
| `N harness HOME: caller log unchanged` | Sentinel log outside private HOME is byte-identical and has the same line count after all arms | Revert the HOME-isolation production hunk -> hook appends to the sentinel/caller log and the arm fails |
| Scoped ShellCheck baseline | Zero findings across both hook files after M1 | Revert any one of the eight `cd ... || exit 1` hunks -> SC2164 makes the command rc 1 |

### Acceptance commands and baselines

- `TEST_HOME=$(mktemp -d); HOME="$TEST_HOME" /bin/bash scripts/hooks/test_push_dev_on_stop.sh "$PWD/scripts/hooks/push_dev_on_stop.sh"; rc=$?; rm -rf "$TEST_HOME"; exit $rc` -> **base rc 0** (`18 passed, 0 failed`, controller); after M1 rc 0 with the new N arms.
- `shellcheck scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` -> **base rc 1** (hook clean; harness has eight SC2164 warnings); after M1 rc 0.
- `/bin/bash -n scripts/hooks/test_push_dev_on_stop.sh` -> **base rc 0**; must remain rc 0.

### Rollback

Revert only the M1 harness commit. No production hook behavior changes, but do not run later harness milestones until HOME isolation is restored.

## M2 — Make Hook and CI Reject Unparseable Go Together

**Finding:** (a)

### What

- Make `fmt-check` capture and inspect `gofmt -l .`'s exit status as well as its output. Preserve useful stderr and distinguish formatter failure from ordinary formatting drift.
- Make the hook explicitly reject a zero-byte committed Go blob before comparing formatter output. Preserve its current refusal of non-empty parse failures.
- Add a harness arm for a committed empty `.go` blob.
- Add a durable `scripts/test_fmt_check.sh` self-test with formatted, empty, and non-empty-unparseable fixtures, expose it as `make test-fmt-check`, and include that target in `make ci`. The fixtures must clean themselves with a trap.

### Why

The hook must not become stronger than the CI gate it mirrors. This milestone changes and tests both gates atomically: empty and malformed committed Go are rejected by both, while valid formatted Go stays admitted.

### Files touched

- `scripts/hooks/push_dev_on_stop.sh`
- `scripts/hooks/test_push_dev_on_stop.sh`
- `make/code-health.mk`
- `scripts/test_fmt_check.sh` (new)
- `make/test.mk`
- `make/ci.mk`

### Test rows and killed mutations

| Test name | Observable | Mutation it kills |
|---|---|---|
| `O empty committed Go: refused` | Bare origin SHA stays fixed and output contains formatting guidance | Remove the hook's zero-byte check -> stdin `gofmt` returns 0, `cmp` succeeds, and origin moves |
| `fmt-check formatted fixture: accepts` | Self-test subcase sees rc 0 | Replace the gate with unconditional failure or mishandle empty command output -> valid fixture fails |
| `fmt-check empty Go: refuses formatter error` | Self-test subcase sees nonzero rc and formatter-failure diagnostic | Revert to `[ -n "$(gofmt -l .)" ]` -> empty stdout is treated as clean |
| `fmt-check malformed Go: refuses formatter error` | Non-empty `package main\nfunc(` fixture sees nonzero rc | Ignore `gofmt` exit status while checking stdout -> rc 2 with empty stdout passes |

### Acceptance commands and baselines

- `make fmt-check` -> **base rc 0** on the clean tree; remains rc 0.
- `make test-fmt-check` -> **base: target absent**; after M2 rc 0 with all three named subcases. Its empty and malformed negative controls discriminate the production recipe.
- `HOME=$(mktemp -d) /bin/bash scripts/hooks/test_push_dev_on_stop.sh "$PWD/scripts/hooks/push_dev_on_stop.sh"` -> **base rc 0, 18/18** (controller); after M2 rc 0 including `O empty committed Go: refused`.
- `/bin/bash -n scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh scripts/test_fmt_check.sh` -> **base rc 0 for the two existing files**; after M2 rc 0 for all three.

### Rollback

Revert the whole M2 commit, including both formatting gates and the new self-test wiring. Never retain only the hook half or only the CI half.

## M3 — Use NUL-Delimited Git Pathnames for Committed Go Blobs

**Finding:** (b)

### What

- Replace newline/C-quoted `git diff --name-only` transport with NUL-delimited output (`-z`) consumed without Bash-4-only features.
- Use a temp file plus a Bash-3.2-compatible `while IFS= read -r -d ''` loop (with `read` fed by redirection, not a pipeline subshell) so exact bytes reach `git show "dev:$go_file"`.
- Preserve spaces, non-ASCII bytes, quotes, and backslashes; keep the existing fail-safe refusal if enumeration or blob extraction fails.
- Add a well-formatted committed non-ASCII filename arm using a byte-stable `printf` construction rather than relying on source-file normalization.

### Why

Git's human-readable default quotes some pathnames. Treating that presentation as a literal path strands valid commits. `-z` is Git's machine interface and removes decoding ambiguity.

### Files touched

- `scripts/hooks/push_dev_on_stop.sh`
- `scripts/hooks/test_push_dev_on_stop.sh`

### Test rows and killed mutations

| Test name | Observable | Mutation it kills |
|---|---|---|
| `P non-ASCII Go pathname: pushed` | Ahead-only well-formatted commit moves bare origin | Revert `git diff -z`/NUL consumer to newline C-quoted transport -> `git show` misses the blob and origin stays fixed |
| Existing `I formatted commit: pushed` | Ordinary pathname still moves origin | Break the new NUL loop so it skips or corrupts ordinary entries -> origin does not move |
| Existing `H unformatted commit: refused with guidance` | Origin stays fixed for unformatted exact-path blob | Drop formatting checks while changing enumeration -> origin moves |

### Acceptance commands and baselines

- `git -c core.quotepath=true diff --name-only HEAD^..HEAD` in the P fixture -> **base behavior: quoted/escaped pathname**, proving the control fires.
- `HOME=$(mktemp -d) /bin/bash scripts/hooks/test_push_dev_on_stop.sh "$PWD/scripts/hooks/push_dev_on_stop.sh"` -> **base rc 0, 18/18** (controller); after M3 rc 0 including P.
- `/bin/bash -n scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` -> **base rc 0**; remains rc 0 under macOS Bash 3.2 syntax.

### Rollback

Revert M3 only. The prior implementation fails safe by refusing affected filenames, so rollback may strand a commit but cannot push unchecked content.

## M4 — Log the Two Earliest Guard Exits

**Finding:** (c)

### What

- On failure to `cd "$ROOT"`, append a repo-attributed `SKIP`/failure line naming the root and reason before exiting 0.
- On failure of the initial `git rev-parse --git-dir`, append a repo-attributed line stating the resolved root is not a Git worktree.
- Keep stdout quiet and preserve the hook-wide always-exit-0 contract.
- Add isolated-HOME harness arms for an absent root and an existing non-Git directory, asserting distinct log reasons.

### Why

These are the only precondition failures that currently leave no evidence. Distinct log tokens make the shared log diagnostic without turning normal non-dev/opt-out paths noisy.

### Files touched

- `scripts/hooks/push_dev_on_stop.sh`
- `scripts/hooks/test_push_dev_on_stop.sh`

### Test rows and killed mutations

| Test name | Observable | Mutation it kills |
|---|---|---|
| `Q missing root: logged, stdout quiet` | Private log gains exactly one repo-attributed `ROOT`/`cd` failure row; stdout remains empty; rc 0 | Restore `cd ... || exit 0` -> no matching row |
| `R non-Git root: logged, stdout quiet` | Private log gains exactly one distinct `NOT_GIT`/`rev-parse` row; stdout remains empty; rc 0 | Restore silent `git rev-parse ... || exit 0` -> no matching row |
| Q/R distinction assertion | The two rows carry different stable reason tokens | Collapse both guards to one ambiguous message -> expected token pair fails |

### Acceptance commands and baselines

- `HOME=$(mktemp -d) /bin/bash scripts/hooks/test_push_dev_on_stop.sh "$PWD/scripts/hooks/push_dev_on_stop.sh"` -> **base rc 0, 18/18**, but Q/R do not exist; after M4 rc 0 with both arms.
- `/bin/bash -n scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` -> **base rc 0**; remains rc 0.

### Rollback

Revert M4 only. Rollback restores silent early exits but does not alter push eligibility or working-tree safety.

## M5 — Wire a Scoped Auto-Push ShellCheck Gate into CI

**Finding:** (d)

### What

- Add a named make target (for example `shellcheck-autopush`) with an explicit file list:
  - `scripts/hooks/push_dev_on_stop.sh`
  - `scripts/hooks/test_push_dev_on_stop.sh`
- Fail loudly if `shellcheck` is absent; do not silently skip an integrity gate.
- Invoke the target from the Ubuntu `lint` workflow after installing/pinning an available ShellCheck package/version as needed. Do not attach it only to `launchd-drivers`: that job currently covers `tools/launchd/**`, and merely adding ShellCheck there would blur its Bash-3.2 purpose.
- Keep `/bin/bash -n` and the functional harness as separate acceptance legs; ShellCheck is not a runtime test.

### Why

The production hook is clean today, but no workflow preserves that property. A named two-file list is deliberate: a repo-wide gate is red across 187 other shell files and would turn this focused sprint into an unrelated cleanup campaign.

### Files touched

- `make/code-health.mk`
- `.github/workflows/ci.yml`

### Test rows and killed mutations

| Test name | Observable | Mutation it kills |
|---|---|---|
| `shellcheck-autopush known-bad production mutation` | In a disposable copy, append an unquoted expansion to `push_dev_on_stop.sh`; target returns nonzero and names that file | Remove production hook from the explicit list -> mutation passes |
| `shellcheck-autopush known-bad harness mutation` | In a disposable copy, remove one M1 `|| exit 1`; target returns nonzero SC2164 naming the harness | Remove harness from the explicit list -> mutation passes |
| Workflow wiring assertion | `ci.yml` invokes the named make target | Remove the workflow step -> repository grep/assertion fails even though a local target remains |

The executor should implement these as a small self-test target/script using disposable copies, not as manual one-shot edits to tracked files.

### Acceptance commands and baselines

- `shellcheck scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` -> **base rc 1** because of eight harness SC2164 warnings; **post-M1 baseline rc 0**; after M5 remains rc 0.
- `make shellcheck-autopush` -> **base: target absent**; after M5 rc 0.
- `make test-shellcheck-autopush` -> **base: target absent**; after M5 rc 0 and both named mutation controls go red internally.
- `rg -n 'shellcheck-autopush' .github/workflows/ci.yml make/ Makefile` -> **base rc 1 / zero hits**; after M5 rc 0 with both target definition and workflow invocation.

### Rollback

Revert M5 CI/Make wiring together. The scripts remain fixed and runnable, but their static-analysis property becomes unguarded again.

## Day Plan and Dependencies

1. Land M1 alone; rerun the full harness under its caller sentinel and confirm no shared-log change.
2. Land M2 atomically across hook and `fmt-check`; prove both empty and malformed mutations against durable tests.
3. Land M3; run the pathname arm with `core.quotepath=true` as a positive control.
4. Land M4; verify the two early exits are log-distinct while stdout stays quiet.
5. Land M5; prove both explicit-list mutations and workflow wiring, then run the final acceptance matrix.

Dependencies: M2–M4 depend on M1 so their harness runs are isolated. M5 depends on M1's eight-warning cleanup and should follow M2–M4 so it certifies the final script content.

## Final Sprint Acceptance

- `make fmt-check` -> base rc 0; final rc 0.
- `make test-fmt-check` -> new durable self-test, final rc 0.
- `make shellcheck-autopush` and its mutation self-test -> new targets, final rc 0.
- `/bin/bash -n scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh scripts/test_fmt_check.sh` -> final rc 0.
- An isolated-HOME functional run reports every old and new arm passing and leaves a caller sentinel log byte-identical.
- `make ci` contains both durable self-test wiring and the scoped static-analysis gate; executor may use the appropriate focused targets during milestone commits and run the full target at sprint end if the controller's iteration budget permits.

## Conflict Surface

- `.claude/settings.json` installs/invokes `scripts/hooks/push_dev_on_stop.sh` as a Stop hook. M1–M5 must not change opt-out behavior, the always-exit-0 contract, or hook stdout expectations beyond the already-loud refusal paths.
- `.github/workflows/ci.yml` owns the proposed scoped ShellCheck invocation and already runs `make fmt-check` in the Ubuntu `lint` job. The macOS `launchd drivers (bash 3.2)` job does not currently cover `scripts/hooks/`; retain explicit `/bin/bash` 3.2 local acceptance rather than pretending that job already protects the hook.
- `make/code-health.mk`, `make/test.mk`, and `make/ci.mk` are shared CI surfaces. M2 must keep the hook no stronger than `fmt-check`; new self-tests must be wired without masking later serial gates.
- `ailang-docs`, `ailang-motoko`, and the separate `ailang-world` repository install the same hook and all write the same `~/.ailang/state/autopush.log`. Deployment/copying to those clones is outside this implementation sprint and remains controller/release workflow work. M4 log tokens must remain repo-attributed because these consumers share the file.
- The harness must never use the real shared log again. Any downstream clone test should also supply an isolated HOME.

## Deferred Findings

None. All five findings fit one focused iteration because they share two scripts and small Make/CI surfaces. Repo-wide ShellCheck cleanup is explicitly deferred: 187 other tracked shell files currently report findings, and repairing them is unrelated to the measured auto-push gate defects.

## Executor Guardrails

- Portable to macOS Bash 3.2: no associative arrays, no `${v,,}`, and no GNU `timeout`. Reuse the hook's Perl-alarm `bounded()` helper.
- Use Git's NUL machine format for pathname transport; do not write a C-quote decoder.
- Every milestone is a separate, independently green commit candidate. The controller performs commits and integration.
- Do not weaken fail-safe behavior: enumeration, blob extraction, formatter, or tooling failures refuse the push loudly and exit 0.
- Do not run the functional harness without an isolated HOME until M1 is present.

## Controller correction — 2026-09-04, V1 iteration 327 (post-execution)

M1's test row was specified as "caller sentinel": seed a known line into the caller's
`$HOME/.ailang/state/autopush.log`, then assert it is unchanged. The executor implemented it
literally, with `printf ... > "$CALLER_LOG"`.

Under the codex `workspace-write` sandbox that write is denied, so the arm looked correct. The
controller's first re-run **outside** the sandbox destroyed the real shared fleet log: **92 lines
of cross-clone push evidence replaced by one sentinel line**, unrecoverable (no copy on disk, no
usable local snapshot). This is finding (e) — the exact artifact this milestone exists to protect
— committed by the test written to protect it.

The arm is now **read-only**: it observes the caller log's sha + line count before and after, with
`absent` as a legitimate reading on both sides, and never writes. The mutation it exists to kill is
unaffected — dropping `export HOME="$W/home"` still turns it red (measured: 3 lines → 17, sha
differs), along with four other arms.

Generalisation, which outranks the incident: **a sandboxed executor cannot distinguish "my
destructive step was denied" from "my step was harmless"**, so any acceptance step that writes
outside the worktree is unverifiable on that lane by construction. A guard for a shared artifact
must be specified as an observation, never as a seeded write.
