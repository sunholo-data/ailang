# Sprint: planner lane fence parsing — fleet iteration 65

2026-10-28 · Base `58a41bad4` · One session (~80 minutes): M1 ~60m, then M2 ~20m.
Resolves duplicate tickets `planner-lane:files-section-parses-fenced-bullets` and `planner-lane:fenced-code-in-files-section`; refs GitHub #581 items 1–2. Item 3 (diff drift detector) is OUT OF SCOPE.
The ticket's “fails safe” description is incomplete: a fenced `# comment` terminates extraction early, hiding a later real language-core bullet and granting the codex/pi lane. Fenced example bullets also falsely deny infra docs; fenced Files headings corrupt the section count.
Controller's first-party reproductions are accepted. Inspection confirms this parser exists only in `tools/launchd/derive-planner-lane.sh`; both awk programs need the fix. Clean-env baseline reproduced: rc 0, **108 passed, 0 failed**.
Authority: only `tools/launchd/**`, `design_docs/**`, `changelogs/**`. No allowlist, routing-policy, reason-token, declaration, or unrelated refactors; do not edit `make/test.mk` (suite already wired).
Executor writes **NO git commits**, performs no git writes, and snapshots nothing: no checkpoint snapshots or sprint JSON. Controller commits M1 and M2 separately by path.
All test artifacts are repository fixtures. Scratch mutation copies/logs stay in a disposable worktree-local directory; no writes outside the worktree, no `$HOME` writes, and no external temporary files except `mktemp` under `$TMPDIR` if strictly needed. Remove only scratch files created by this run.
Registry reuse: M1 `none` (existing Bash/awk parser correctness, no package capability); M2 `none` (documentation only). No registry lookup or dependency is applicable.

## Exact executor file list and estimates

M1 (~120–150 changed lines, including fixtures and assertions):

- `tools/launchd/derive-planner-lane.sh`
- `tools/launchd/test_mission_routing.sh`
- `tools/launchd/testdata/planner-lane/q-fenced-example.md`
- `tools/launchd/testdata/planner-lane/r-fenced-comment-before-core.md`
- `tools/launchd/testdata/planner-lane/s-fenced-files-heading.md`
- `tools/launchd/testdata/planner-lane/t-tilde-fence.md`
- `tools/launchd/testdata/planner-lane/u-long-backtick-fence.md`
- `tools/launchd/testdata/planner-lane/v-unterminated-after-bullet.md`
- `tools/launchd/testdata/planner-lane/w-unterminated-before-bullet.md`

M2 (~15–25 changed lines), depends on M1:

- `design_docs/implemented/v1_0_0/m-planner-codex-lane-sprint-plan.md`
- `changelogs/unreleased/2026-10-28-planner-lane-fences.md`

## M1 — Fence-aware count AND path extraction

Add the same small fence-state guard at the start of each awk record action, BEFORE heading detection, section termination, or bullet extraction; track fences over the whole document, including before Files. Keep the two existing awk passes and their non-fence logic.
Recognize openers with 0–3 literal leading spaces and a maximal run of 3+ backticks OR tildes; remember character and run length. An opener may have an info string (CommonMark backtick info cannot contain backticks). Consume the opener with `next`.
While fenced, consume EVERY line with `next`; close only on 0–3 leading spaces, the SAME fence character repeated at least the opening length, followed only by spaces/tabs. Shorter or mixed-character fences do not close it. Closing lines are consumed too; `#`/`## Files`/`---` and path bullets inside fences have no other effect.
Use portable `match`, `substr`, `length`, character loops, and `[[:space:]]`; no gawk extensions, `\s`, new interval-expression reliance, or Bash 4 features. Mandatory compatibility: `/bin/bash` 3.2 and macOS BSD awk 20200816. Existing heading regexes remain unchanged.
Update the Step-4 comment immediately above `section_count` (~107–109) to state that both scans ignore fenced content and unterminated fences consume the remainder. Do not add new reason tokens.
Unterminated fence AFTER a valid infra bullet still yields `codex:gpt-6.1-sol declared:codex-ok`; BEFORE all bullets in a real Files section yields `opus fail-closed:unparsable-path-entry`. If it hides the only Files heading, or the outside-fence section count is not one, retain `opus fail-closed:no-files-section`. A real disallowed bullet outside fences yields `opus fail-closed:path-not-in-codex-allowlist`.

Create these exact fixtures from the repository root (single quotes preserve backticks; each argument is one line, with a final newline):
``````bash
p=tools/launchd/testdata/planner-lane
fixture() { name=$1; shift; printf '%s\n' '**Planner-Lane**: codex-ok' '' "$@" > "$p/$name"; }
fixture q-fenced-example.md '## Files' '- `tools/launchd/x.sh` — change' '   ```bash' '- `internal/parser/parser.go`' '   ````'
fixture r-fenced-comment-before-core.md '## Files' '- `tools/launchd/x.sh` — change' '```bash' '# run this' 'make test' '---' '```' '- `internal/parser/parser.go` — real change'
fixture s-fenced-files-heading.md '```markdown' '## Files' '```' '## Files' '- `tools/launchd/x.sh` — change'
fixture t-tilde-fence.md '## Files' '- `tools/launchd/x.sh` — change' '~~~bash' '```' '- `internal/parser/parser.go`' '~~~'
fixture u-long-backtick-fence.md '## Files' '- `tools/launchd/x.sh` — change' '````bash' '```' '- `internal/parser/parser.go`' '````'
fixture v-unterminated-after-bullet.md '## Files' '- `tools/launchd/x.sh` — change' '```bash' '- `internal/parser/parser.go`'
fixture w-unterminated-before-bullet.md '## Files' '```bash' '- `tools/launchd/x.sh` — fenced, not a real bullet'
unset -f fixture
``````

Add seven `out=...; want ...` arms beside the existing derive arms (~55–160). Each arm MUST explicitly unset inherited allowlist, pin Sol 6.1 and mark Anthropic available; use this invocation pattern (substitute fixture and expected result):
```bash
out=$(env -u MISSION_PLANNER_ALLOWLIST MISSION_PLANNER_MODEL=codex:gpt-6.1-sol MISSION_ANTHROPIC_AVAILABLE=1 "$DERIVE" "$ROOT/tools/launchd/testdata/planner-lane/q-fenced-example.md" 2>/dev/null)
want "q: fenced example ignored" "$out" "codex:gpt-6.1-sol declared:codex-ok"
```

| Arm name | Expected stdout (exact) |
|---|---|
| q: fenced example ignored | `codex:gpt-6.1-sol declared:codex-ok` |
| r: fenced comment cannot hide core path | `opus fail-closed:path-not-in-codex-allowlist` |
| s: fenced Files heading not counted | `codex:gpt-6.1-sol declared:codex-ok` |
| t: tilde fence ignores backtick closer | `codex:gpt-6.1-sol declared:codex-ok` |
| u: shorter backtick fence cannot close | `codex:gpt-6.1-sol declared:codex-ok` |
| v: unterminated fence preserves earlier bullet | `codex:gpt-6.1-sol declared:codex-ok` |
| w: unterminated fence hides all bullets | `opus fail-closed:unparsable-path-entry` |

Mutation drills, after a green run: use `scratch=$(mktemp -d "$PWD/.tmp-planner-fences.XXXXXX")`; `cp tools/launchd/derive-planner-lane.sh "$scratch/derive.original"`. This is a temporary restoration copy, not a milestone snapshot. For each drill separately, prove the mutant differs (`cmp -s` must return 1), run the clean-env routing gate below and report rc, failed arm names, and observed outputs. Require rc nonzero and at least one named arm red:

1. Delete the fence-state guard from path-extractor awk only: **q** and/or **r** must turn red.
2. Restore, then delete the guard from count awk only: **s** must turn red.
3. Restore, then weaken closure in both scans to accept any 3+ fence run regardless of character/remembered length: **t** and **u** must turn red.

After EACH mutation (including failures), restore with `cp "$scratch/derive.original" tools/launchd/derive-planner-lane.sh` and require `cmp "$scratch/derive.original" tools/launchd/derive-planner-lane.sh` rc 0. Never `git checkout`. Re-run the green gate after final restoration.

## M2 — Correct historical AC9 and record the fix

Evidence: `git show -s --format=%B cd76499c5` (full SHA `cd76499c5e0de5fd1d3423f4d88d9209247fbb2d`), squashed #580 commit's **M2 — role-generic codex probe loop + D6 rollback plumbing** message. Exact recorded form:
```text
  AC9  three-sided: file pinning a SENTINEL -> planner=SENTINEL-FROM-FILE; file
       deleted -> planner=opus; explicit command-line pin -> planner=opus.
```
Replace ONLY the AC9 bullet (~332–340) in the implemented sprint plan, retaining the existing commented V1 rollback preparation/cleanup prose, with historical pre-flip sentinel instructions: in the `acheck` namespace, file content `MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-SENTINEL-FROM-FILE}"`; clean environment/no planner pin gives `planner=SENTINEL-FROM-FILE`; delete file gives the THEN built-in `planner=opus`; recreate the sentinel file and explicitly pin `MISSION_PLANNER_MODEL=opus` to get `planner=opus` (env-pin wins). Each command uses `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR` plus `MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus` and `/bin/bash tools/launchd/mission-control.sh`, with the planner pin added only in side three. Explain that these are historical pre-flip expectations, not a current-default gate; execute NO AC9 commands or home writes in this sprint.
Add one line within that bullet: “2026-10-28: AC9 plan text corrected in fleet iteration 65 from the M2 commit message of cd76499c5.” Change nothing else in that file; verify the diff.
Fragment must open `### Fixed — Planner lane Files parsing respects code fences (2026-10-28)` per `changelogs/unreleased/README.md`. Explicitly name the fail-open shape (fenced shell `#` comment stops scanning before a real language-core path), the false denials from fenced examples/headings, both corrected scans, and #581 items 1–2.

## Acceptance gates and handoff evidence

Executor runs, from repository root, with successful exit codes reported:
```bash
/bin/bash -n tools/launchd/derive-planner-lane.sh
env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR /bin/bash tools/launchd/test_mission_routing.sh
make check-changelog
```
Routing must report **115 passed, 0 failed**, rc 0 (108 existing + 7 new); inherited `MISSION_PLANNER_ANTHROPIC_FALLBACK` contaminates the non-clean invocation. Use the identical clean-env form for every mutation run. `make help` confirms `check-changelog`; its script also checks fragment folding.
ShellCheck coverage inspected: `shellcheck-autopush`/`test-shellcheck-autopush` scope only four `scripts/hooks/` files, not `tools/launchd`. No applicable ShellCheck target; do not invent one or require the unrelated target.
CONTROLLER, outside the sandbox, runs `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR make test-launchd-drivers`; executor does not run the full driver suite. Report BSD awk/Bash versions used, targeted results, three killed mutants plus byte-identical restorations, and exact changed paths; leave M1 and M2 uncommitted for controller review/commits.

## Amendment after the round-1 judgement (controller, 2026-10-28)

The independent sonnet judge scored round 1 PASS 74 with one blocking finding: a CRLF closer never closed, so a CRLF doc with any fence swallowed its later bullets and could be granted codex (fail-open). The controller amended M1. The fence logic moved into one shared awk function (`fence_skip`) used by both scans; a trailing CR is stripped before fence matching; and a fence still open at end of file inside the Files section now emits `__UNPARSABLE_PATH_ENTRY__`, so fixture **v** expects `opus fail-closed:unparsable-path-entry` instead of codex, overriding this plan's line on unterminated fences. Six fixtures (x, y, z, za, zb, zc) give every fence rule its own killing arm. The CRLF fixtures are stored LF (`* text=auto` would normalise a committed CRLF file), so the x/y arms build CRLF copies at run time and assert the CR bytes exist. Routing suite 122/0; nine mutation drills all killed, each restored with `cp` + `cmp`.
