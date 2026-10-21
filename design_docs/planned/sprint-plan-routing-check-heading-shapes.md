# Sprint Plan: Routing evidence heading shapes

## Summary

Ticket: `routing-models-check:bold-colon-heading` (mission world, 2026-10-21); fleet iteration 54; sprint `M-FLEET-ROUTING-CHECK-HEADINGS`.
Goal: recognise every established routing-evidence heading convention and scan exactly its block, restoring Gate 4 findings for world, Stapledon, V1 and docs while preserving fleet/motoko/triage behaviour.
Planning only: this plan and its sprint JSON are the only planner deliverables; controller owns commits and execution dispatch. The ticket supplies the fixed design; no separate design doc.
Duration: 0.25 working day; one 60–90 minute executor run, followed by controller integration. Estimate: 100 LOC (10 checker + 87 tests + 3 changelog); low risk, no external dependency or open design question.

## Current Status Analysis

Base: detached `origin/dev d25d9b8d0`. Iteration 53 / PR #1922 introduced the checker (prior plan D7); local read-only inspection confirms its literal `^\*\*Routing evidence\*\*` match, first-block-only handling and generated-fixture `--only` test harness.
Controller measurements C1–C7 are established outside-sandbox evidence; do not repeat them as planner or require access to other repos:
- C1 world: 20 `**Routing evidence:**` headings; entries 295/294 return rc 2.
- C2 Stapledon: 41 `- **Routing evidence:**` headings; indented sub-bullets, then sibling `- **Ruled out:**`; currently rc 2.
- C3 V1: 20 `**Routing evidence.**` headings with inline/continuation prose, plus one `**Routing evidence**:`.
- C4 docs: nine standard headings, one `**Routing evidence (from the orphaned fire's PR body, re-verified rather than transcribed)**:` with a table.
- C5 fleet/motoko/triage: standard `**Routing evidence**:`; fleet's column-0 `- **Controller**:` / `- **Executor**:` belong inside the block; fleet entry 53 rc 0.
- C6 base shell suite: 20 passed / 0 failed, rc 0; already wired into `make test-launchd-drivers` at `make/test.mk:83`.
- C7 both Gate 4 skill copies prescribe the checker without a heading shape: fix the checker, leave skills untouched.
Velocity basis: preceding sprint JSON records 610 actual LOC in one day across two broader milestones; this bounded 100-LOC fixture/parser correction is estimated from its tasks, not a throughput guarantee.

## Registry Reuse Audit

`M1_ROUTING_HEADING_SHAPES`: `none`; internal Bash/awk harness correction, no package-like AILANG capability or `.ail` code. Registry search is inapplicable; reuse the existing checker, fixture helpers and make wiring. No network or registry calls.

## Proposed Milestone

### M1_ROUTING_HEADING_SHAPES — recognise real headings and preserve block scope

Dependencies: none. Estimate: 100 LOC including tests and fragment; one executor run.
Files: update `scripts/mission_routing_models_check.sh` and `scripts/test_mission_routing_models_check.sh`; create `changelogs/unreleased/2026-10-21-routing-check-heading-shapes.md`.
Examples: generated fixtures in the existing shell harness serve as working examples; no `.ail` example or teaching-prompt gate applies.

Day 1 tasks (ordered within the run):
1. Add the five arms below with generated fixtures; extend BOTH the usage line and default arm list, retaining `--only <arm>` and all existing arms. Record relevant pre-fix failures.
2. D1: replace the header regex with `^(- )?\*\*Routing evidence[^*]*\*\*`, guarded by `!seenBlock`. Record whether the matched header starts with column-0 `- `; initialise/reset that state per selected entry. Continue scanning the header's own tokens, including inline token at EOF.
3. D2: end both forms at the next column-0 `**` field or entry boundary. Only for a bulleted header, also end at the next column-0 `- **` sibling field. Indented sub-bullets remain inside; unbulleted blocks retain column-0 bold-label bullets. Preserve first-block-only behaviour.
4. D3: preserve entry-heading regex, allowlist, canonicalisation, argument parsing and rc contract: 0 no findings / 1 findings with original file:line and token / 2 instrument failure. Use Bash 3.2 and BSD awk only; no gawk extensions, `\s` or interval-expression dependency.
5. Prove the mutation kills below using disposable copies under a temporary directory, mirroring the existing relative checker/test layout. Apply one mutation at a time, run its `--only` arm, retain rc/assertion evidence, then restore the disposable copy. No git writes or real-log edits.
6. D5: create the 2–4-line fragment starting `### Fixed — ` per `changelogs/unreleased/README.md`; describe restored heading recognition and correct sibling-field boundaries. Run focused checks and report results.

## Test Arms and Mutation Evidence

Each new arm must pass on the fix and fail under its named one-line checker mutation. Assert exact original file:line/token for every retired-token finding; clean cases must have empty findings.

| Arm | Generated real-shape fixture and expected result | One-line mutation that must turn the arm red |
| --- | --- | --- |
| `w` | World `**Routing evidence:**`: allowed `gpt-6.1-sol` inside → rc 0; retired `gpt-6-astra` inside → rc 1 naming its line. | Replace D1 header regex with old `^\*\*Routing evidence\*\*` → rc 2 instead. |
| `s` | Stapledon `## 44 — 2026-10-21 — ...`, `- **Routing evidence:**`, indented `  - Controller: ...` / `  - Designer: ...`: retired token inside → rc 1. Separate fixture with allowed routing and retired token ONLY after sibling `- **Ruled out:**` → rc 0. | Delete `(- )?` from D1 → header unrecognised. ALSO remove the bulleted-sibling termination clause, leaving only `^\*\*` → sibling-only case incorrectly rc 1. |
| `v` | V1 `**Routing evidence.**` with inline allowed prose and `gpt-6-astra` on a following unbulleted continuation line that does not start `**` → rc 1. | Replace D1 regex with old literal → rc 2 instead. |
| `p` | Docs exact parenthetical heading from C4 followed by a markdown table containing `gpt-6-astra` → rc 1, proving recognition and table scanning. | Replace D1 regex with old literal → rc 2 instead. |
| `f` | Fleet `**Routing evidence**:` then column-0 `- **Controller**:` allowed and `- **Executor**:` carrying `gpt-6-astra` → rc 1. | Make `^- \*\*` terminate unbulleted blocks too → executor token missed, rc 0. |

## Accept

- [ ] Preserve the base 20/20 checks and all existing arm selections, including inline token at EOF and entry-boundary coverage.
- [ ] Every new arm (`w s v p f`) is green on the fix and red under its named mutation; `s` kills both header and sibling-boundary mutations. Record baseline, mutant red and restored green with the failed assertion, not just process failure.
- [ ] `bash -n scripts/mission_routing_models_check.sh scripts/test_mission_routing_models_check.sh` is clean; `/bin/bash` syntax validation is also clean on the rig's Bash 3.2.
- [ ] `/bin/bash scripts/test_mission_routing_models_check.sh` returns rc 0 outside the sandbox; findings retain original line numbers and tokens.
- [ ] Fragment is 2–4 lines, opens `### Fixed — `; `make check-changelog` and `make check-file-sizes` pass. Existing make wiring stays unchanged.
- [ ] Only scoped files change; no full Go suite/build required for this shell-only fix. Success is behavioural coverage above, not a coverage percentage.

## Controller-only Integration and Completion

After execution, outside the sandbox: run HEAD's pre-fix script and the new script on EVERY entry of every reachable real log (`design_docs/*-mission-log.md` here, plus world `~/dev/sunholo-data/ailang-world/design_docs/world-mission-log.md` and Stapledon's log in `stapledons-godot`). Keep per-log/entry old/new rc and findings; explain every rc change, including newly recognised clean blocks or genuine retired-token findings. Preserve fleet control; unexplained changes fail the corpus gate. Out-of-tree logs are controller-owned and never modified.
Controller also runs the direct shell suite and `env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" make test-launchd-drivers` outside the sandbox, then owns commit, PR, CI and ticket resolution. Executor owns only focused socket-free checks and mutation evidence; planner runs artifact validation only.
Sandbox socket-touching results are **UNINFORMATIVE UNDER SANDBOX**; this sprint's focused shell checks touch no sockets. No network attempts in planning/execution.

## Scope and Risks

D6 scope is exactly the two scripts, the named fragment, this plan and `.ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json`. No skill edits, make wiring edits, allowlist edits, entry-heading changes, other repo access, `~/.ailang/state` or `~/.config` changes.
Main risk: the conventions conflict at column-0 bold-label bullets; `s` and `f` independently protect both boundary rules. Header broadening intentionally accepts all D1 suffixes; first-block-only and existing entry boundaries prevent widening the scan beyond the selected routing block.
