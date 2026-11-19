# M-FLEET-GATE0-LEDGER-PROVENANCE

Ticket: `skill:gate0-ledger-provenance-S` (stapledon, 2026-09-28). Planner: fleet iteration 35, 2026-10-13. Verified baseline: `b283b2d44`.

Planning only; controller owns approval, execution and commits. Two sequential milestones, about 400 changed LOC and one working day including buffer (M1 ~90 minutes, M2 ~60 minutes). No language or runtime changes, no `.ail` examples required. The fixture suite is the executable example.

## Measurements and scope

Read `CLAUDE.md`, the sprint-planner skill, rule 6(b), `mission_answer.sh`, the directive/self-notice precedents, make targets and CI. Both Gate 0 copies are byte-identical (`cmp`, rc 0). Only the two planning artifacts are written in this stage; implementation paths below are within the supplied fleet authority.

Re-verified with email-free git formats at HEAD:

| Row | Old `git log -1 -S` result | Earliest literal RESOLVED change | Latest literal RESOLVED change |
|---|---|---|---|
| D-FLEET-13 | `32d19153a` (2026-10-03) | `996c411a4` (2026-10-08) | `996c411a4` |
| D-FLEET-14 | `845bbbbdf` (2026-10-05) | `996c411a4` | `996c411a4` |
| D-FLEET-15 | `752ee765a` (2026-10-08) | `996c411a4` | `996c411a4` |
| D-FLEET-1 | `b222b2eea` (2026-09-26) | `e6990da9a` (2026-10-02) | `e6990da9a` |

**Measured correction to the ticket's D-FLEET-1 example:** `git show b222b2eea:design_docs/fleet-mission.md` has status `RULED 2026-09-26 ...`, not `RESOLVED`. The diff at `e6990da9a` normalizes RULED to RESOLVED. Thus this row does not demonstrate the later-reword defect; a synthetic fixture must demonstrate it. Do not relabel the historical ruling or invent an earlier literal RESOLVED flip. Its control is `b222b2eea`, literal flip is `e6990da9a`, verdict is `undetermined:squash-merge`.

`996c411a4` and `e6990da9a` have author display name `Voight-Kampff (bot)` and committer `GitHub`; `996c411a4` has one parent. Git alone cannot identify who ruled in these squash merges. `mission_answer.sh` already documents that identity defaults are a convention, not an authentication control. Report ambiguity, never reopen on this evidence. The direct-commit case in the reporter's repo remains repairable by identifying the real flip.

Systemic search found the faulty pickaxe command in exactly the two Gate 0 mirrors. `mission_decisions.sh` validates state, not commit provenance; leave the writer and validator unchanged. Registry reuse per milestone: **none**; these are repository-local Bash/git instrumentation and integration, not package-like AILANG capabilities. No registry search is applicable. Reuse existing script conventions and the suite environment wrapper. The velocity helper exited 141 before completing its summary; no reliable LOC/day measurement was obtained, so estimates are task-based rather than extrapolated from compiler work.

## M1: Instrument and mutation-focused fixtures (~340 LOC)

Files to create:

- `scripts/mission_ledger_provenance.sh` (~130 LOC).
- `scripts/test_mission_ledger_provenance.sh` (~210 LOC).

Implement a read-only instrument:

`scripts/mission_ledger_provenance.sh --id D-nn --file <charter> [--self <login-or-name-pattern>]`

One stdout line per invocation: `D-nn verdict=<value> flip=<full-sha|none> creation=<full-sha|none>`. Successful classifications (`attended`, `fleet`, `undetermined:squash-merge`, `never-resolved`) return 0; instrument failures return 2 with `verdict=instrument-failure` and a terse diagnostic on stderr. Preserve known SHAs on failure, use `none` only when unavailable. Never print names, email addresses or raw git errors containing identity data. No git writes, network access, ledger edits or automatic reopening.

Algorithm and safeguards:

1. Require both arguments, a safe exact ID (D- followed by letters/digits/hyphens/underscores), a readable tracked file, and exactly one matching current row. Inspect committed HEAD, refusing uncommitted changes to the target file. Quote paths, use `--` path separation, and validate the optional nonempty case-insensitive ERE self pattern. Reject missing/duplicate rows, malformed arguments/patterns, shallow or unavailable history and git failures with rc 2; none is `never-resolved`.
2. Use anchored, whitespace-tolerant `-G` row patterns, including the status cell's closing pipe. IDs are exact, so D-1 cannot match D-13. Enumerate full history oldest first (`--reverse --topo-order`, **without `-1`**) with rename detection disabled so whole-file additions are visible. Find creation as the first verified addition of that exact row (parent row absent), independent of status. The old `-S` result is a regression control, not the production flip selector.
3. For RESOLVED candidates, inspect added diff lines and the commit/parent row states. Select the earliest actual addition of a RESOLVED row whose parent is absent or not RESOLVED; root additions count. A removed RESOLVED line alone is never a flip; a RESOLVED-to-RESOLVED reword is never a flip. Created-already-RESOLVED has `flip=creation`. No verified resolution anywhere in complete history gives `never-resolved`, `flip=none`, and the actual creation SHA. Historical noncanonical statuses such as RULED are not silently rewritten or equated to a human identity.
4. Classify the selected flip, never creation or the latest reword. **First** detect a GitHub-like committer name (case-insensitive exact `GitHub`, `GitHub[bot]`, or `web-flow`): conservatively return `undetermined:squash-merge` irrespective of author. This is an ambiguity heuristic, not a claim that git proves the merge strategy. Unsupported ambiguous history (e.g. a multi-parent resolution whose introduction cannot be established) is instrument failure, not fleet. Historical path renames/ID reuse that prevent a unique creation control also fail rather than inventing provenance.
5. For a supported non-squash flip, compare author display name only, case-insensitively, against the self ERE. Default whole-name aliases cover both `sunholo-voight-kampff` and `Voight-Kampff (bot)`; `--self` supplies a mission-specific whole-name ERE and requires no GitHub lookup. Match => `fleet`; other author => `attended` under the existing convention. Do not read an email to map a login to a name. Document that callers with different git display names must supply their name pattern. The squash branch must precede this comparison.

Both scripts must run under `/bin/bash` 3.2.57 and BSD tools: no associative arrays, lowercase expansion, GNU-only flags, or `grep -P`. Fixture repos use `mktemp -d`, reserved `.invalid` identity addresses, explicit local identities/hooks isolation, and cleanup traps; they must never touch the main repo's git state. Save expected commit SHAs immediately after each fixture commit, independently of the instrument. Assert verdict, rc, flip SHA, creation SHA, one-line stdout and no address leakage. Allow a test-only instrument-path override for mutation probes, following the self-notice test precedent.

| Test arm / fixture | Expected result | Kills which mutation |
|---|---|---|
| A1: fleet creates OPEN; human directly commits RESOLVED | attended; flip differs from creation. Also assert old `-S` selects fleet creation | Restore the old occurrence-count pickaxe / classify creation instead of flip |
| A2: human creates OPEN; fleet directly flips it | fleet at fleet flip, not human creation | Always attended / inspect creation author |
| A3: fleet author, GitHub committer, one-parent flip | undetermined:squash-merge with both SHAs | Ignore committer / classify author before squash ambiguity |
| A4: human creates a row already RESOLVED, including a root-commit variant | attended; flip equals creation | Require an OPEN parent / omit root additions |
| A5: human resolves; fleet later rewords the RESOLVED row | attended at ORIGINAL flip | Latest `-G` or `-1` combined with `--reverse` / treat reword as a flip |
| A6: fleet creates OPEN and edits prose; never resolves | never-resolved; flip=none, creation retained | Treat any row edit as resolution / fabricate a flip |
| A7: row absent in a tracked charter | rc 2; instrument-failure | Missing row becomes never-resolved |
| A8: missing file | rc 2; instrument-failure | Swallow file/git errors as empty history |
| A9: human resolves, then a fleet commit removes the RESOLVED line and re-adds the row OPEN | original human flip and creation, not removal SHA | Accept a removed-only `-G` hit as the flip (especially latest-hit selection) |
| A10: D-1 stays OPEN while D-13 resolves; variable cell whitespace | D-1 never-resolved; D-13 resolved | Unanchored ID prefix / fixed-space-only regex |
| A11: custom self name ERE in mixed case; direct fleet flip; empty/invalid ERE variants | fleet for valid pattern; rc 2 for invalid configuration | Ignore override / case-sensitive identity / silently empty self |
| A12: duplicate row, missing option value, invalid ID, uncommitted target and shallow fixture | rc 2 for each | Success on malformed input or incomplete/uncommitted provenance |
| A13: creator and resolver have distinct reserved identity addresses | neither stdout nor stderr contains either address (also enforce across all arms) | Reintroduce `%ae` / print raw identity diagnostics |

Acceptance: all arms pass with independent SHA assertions; each named mutation is demonstrated to fail its corresponding arm using disposable copies outside the production tree. Report actual kills, not merely the labels. No behavior is inferred from an all-green suite with the original implementation alone.

## M2: Gate mirrors, CI hook and release note (~60 LOC)

Depends on M1. Files to update/create:

- `.claude/skills/mission-control/resources/gate-0-preflight.md`.
- `.agents/skills/mission-control/resources/gate-0-preflight.md` (identical bytes).
- `make/test.mk`.
- `changelogs/unreleased/2026-10-13-gate0-ledger-provenance.md` (opens with `### Fixed`).

Replace only rule 6(b)'s broken command/author-only disposition with the instrument call. Keep the sentence **Use that output to COMPARE, never to RECORD** and the prohibition on recording addresses. Store/report only verdict and commit controls. `attended` uses the existing attended-channel contract; **only `fleet` from a non-squash flip** triggers SELF-RESOLUTION/reopen/flag. `undetermined:squash-merge` is reported with flip and creation SHAs and never triggers reopening or a confirmation request on that basis. `never-resolved` establishes no attended resolution; rc 2 stops the provenance-dependent action and reports the instrument failure. Neither ambiguity nor failure upgrades authority. Preserve the other rule clauses and no-reasking contract.

**CI measurement:** `make/test.mk:79-80` explicitly enumerates existing `scripts/test_mission_*.sh`; there is no wildcard discovery. An unhooked new file would run only by hand. Add `@$(LAUNCHD_SUITE) scripts/test_mission_ledger_provenance.sh` to `test-launchd-drivers`, plus `/bin/bash -n` for both new scripts. `LAUNCHD_SUITE` already invokes `tools/launchd/lib/suite-env.sh` and sanitizes inherited mission/git configuration. `.github/workflows/ci.yml:748-769` runs this target on macOS with an explicit Bash 3.x assertion; its shell path classifier includes `scripts/*mission*.sh` and `make/test.mk`. Therefore the implementation PR activates the job and the new suite runs in CI; no workflow edit is needed or authorized. `make ci` also includes the target. Current touched files are 257 lines per gate mirror and 427 lines for `make/test.mk`, safely below 800; no companion split required.

Acceptance: fixture suite is actually wired, mirror comparison passes, the baseline fleet rows below remain undetermined with the measured SHAs, and hygiene gates pass. Changelog explains both creation-vs-flip and squash-author ambiguity without claiming stronger authentication.

## Controller acceptance commands and evidence status

Run from the repo root **outside the planner sandbox after implementation**, with complete git history. These are focused executor checks; the full Go suite belongs to CI. Baseline checks are not evidence for code that does not yet exist.

| Command | Required evidence / planning status |
|---|---|
| `/bin/bash scripts/test_mission_ledger_provenance.sh` | All A1-A13 and demonstrated mutation kills; **UNINFORMATIVE UNDER SANDBOX** at planning time: suite absent and fixture git writes excluded by planner instructions. Run outside after M1. |
| `/bin/bash -n scripts/mission_ledger_provenance.sh scripts/test_mission_ledger_provenance.sh` | Both parse; **UNINFORMATIVE UNDER SANDBOX** at planning time: implementation absent. |
| `make test-launchd-drivers` | Includes and passes new suite under sanitized Bash environment; **UNINFORMATIVE UNDER SANDBOX** for the new hook before M2; not run by this planner. Existing unrelated suite failures must be distinguished from new-arm failures, never counted as a pass. |
| `make -n test-launchd-drivers` | Output includes new fixture command and both syntax checks; future hook not present during planning. |
| `make test-changed` | Standard focused Go/core smoke after integration; not run in planning-only stage. If sandbox/network prevents it, record **UNINFORMATIVE UNDER SANDBOX**, rerun outside; do not substitute a green fixture suite. |
| `cmp .claude/skills/mission-control/resources/gate-0-preflight.md .agents/skills/mission-control/resources/gate-0-preflight.md` | rc 0; baseline passed inside sandbox, rerun after M2. |
| `make check-no-personal-email check-changelog check-file-sizes` | rc 0; baseline passed inside sandbox, rerun after M2. Address-free output additionally asserted by fixtures because the email gate scans tracked files and permits reserved addresses. |
| `/bin/bash .claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-FLEET-GATE0-LEDGER-PROVENANCE` | Passed inside sandbox: both populated milestones and registry decisions validate. |
| `jq -e . .ailang/state/sprints/sprint_M-FLEET-GATE0-LEDGER-PROVENANCE.json` | Passed inside sandbox; rerun after state updates. |

Real-history checks (future implementation commands; **UNINFORMATIVE UNDER SANDBOX** during this planning stage because instrument is absent):

```bash
for ledger_id in D-FLEET-13 D-FLEET-14 D-FLEET-15 D-FLEET-1; do
  /bin/bash scripts/mission_ledger_provenance.sh --id "$ledger_id" --file design_docs/fleet-mission.md || exit "$?"
done
```

Assert `undetermined:squash-merge` for all four, full flip SHA expanding `996c411a4` for 13/14/15 and `e6990da9a` for 1, and creation SHAs expanding the measurement table. These checks do not edit rows. Later history may add rewords; the original verified literal flips must remain selected. The synthetic A4/A5 arms, not D-FLEET-1's legacy RULED row, prove creation-as-RESOLVED and immunity to later rewords.

## Risks and limits

- Git author display names are conventional identity evidence, not authenticated rulings. The GitHub committer heuristic deliberately over-reports ambiguity; it must never reopen an attended squash ruling. It cannot prove whether a non-GitHub commit was produced by a squash; do not advertise it as authentication.
- Complete, inspectable history is required. Shallow, renamed, reused-ID or ambiguous merge history must surface failure instead of a guessed earliest flip. History dates do not override topology; inspect real added rows and parent state.
- ROOT additions, `-G` removed-line matches and later rewords are distinct traps; bank actual mutation failures. Fixtures must isolate git configuration/hooks/signing so the rig's identity cannot contaminate expected verdicts.
- This sprint changes source copies, not a live installed skill surface. The controller's normal merge/skill-sync workflow owns rollout; no driver, global skills, pending approvals or unrelated active sprint are modified here.
- The planner stops after writing and validating these two artifacts. No execution handoff message, git write, implementation, background work or ledger reopening is part of this stage.

## M1 mutation results (executor, in-sandbox)

The first six rows are the controller-supplied prior-run measurements; they were not rerun.
The remaining 14 drills used `LEDGER_SCRIPT_UT` with disposable instrument copies under
`mktemp -d` in `/tmp`, a 45-second limit per suite, and `/bin/bash` 3.2.57. All completed
in 5.46–8.10 seconds; every intended arm failed. No vacuous arm was found, and neither
committed script was changed. Red lists enumerate failing arms, not assertion counts.
For reproducibility, latest-hit overwrote the selected flip with the latest RESOLVED `-G`
hit; minus-one-selection limited row history to `-1`; reword-as-flip removed earliest/parent
guards; any-row-edit accepted added rows regardless of status. The removed-only mutant
accepted the latest RESOLVED `-G` hit when the current row was OPEN. The ID-prefix mutant
removed the ID-cell boundary, and fixed-space-regex required literal `| D-nn |` spacing.
The self mutants ignored the override, removed case folding, or accepted an empty pattern.
Missing-row and file-error mutants returned never-resolved; malformed-input failures returned
rc 0; the email mutant emitted `%ae` on stderr. None printed captured identity data in the report.

| Mutation | Target arm | Red arms | Target kill | Suite rc / evidence |
|---|---|---|---|---|
| old-pickaxe | A1 | A1,A2,A3,A5,A6,A9,A10,A11,A12,A13 | PASS | Prior run; not rerun |
| classify-creation | A1 | A1,A2,A5,A9,A11 | PASS | Prior run; not rerun |
| always-attended | A2 | A2,A11 | PASS | Prior run; not rerun |
| ignore-committer | A3 | A3 | PASS | Prior run; not rerun |
| author-before-squash | A3 | A3 | PASS | Prior run; not rerun |
| require-open-parent | A4 | A4 | PASS | Prior run; not rerun |
| latest-hit | A5 | A3,A5,A9 | PASS | 1 (measured this resume) |
| minus-one-selection | A5 | A1,A2,A3,A5,A6,A9,A10,A11,A12,A13 | PASS | 1 (measured this resume) |
| reword-as-flip | A5 | A3,A5 | PASS | 1 (measured this resume) |
| any-row-edit-as-resolution | A6 | A1,A2,A3,A5,A6,A9,A10,A11,A12,A13 | PASS | 1 (measured this resume) |
| missing-row-as-never-resolved | A7 | A7 | PASS | 1 (measured this resume) |
| swallowed-file-errors | A8 | A8,A12 | PASS | 1 (measured this resume) |
| removed-only-hit-accepted | A9 | A9 | PASS | 1 (measured this resume) |
| unanchored-id-prefix | A10 | A10 | PASS | 1 (measured this resume) |
| fixed-space-regex | A10 | A10 | PASS | 1 (measured this resume) |
| ignored-self | A11 | A11 | PASS | 1 (measured this resume) |
| case-sensitive-self | A11 | A11 | PASS | 1 (measured this resume) |
| empty-self-accepted | A11 | A11 | PASS | 1 (measured this resume) |
| success-on-malformed-input | A12 | A7,A8,A11,A12 | PASS | 1 (measured this resume) |
| reintroduced-author-email | A13 | A1,A2,A3,A4,A5,A9,A10,A11,A13 | PASS | 1 (measured this resume) |

## Controller revision after evaluation (iteration 35)

The sonnet judge (PASS 88, round 1) found N1: when a human resolves, then reopens, and then the fleet
re-resolves directly, the earliest-flip rule reports `attended` even though the current resolution is
fleet-authored. The controller reproduced it in a fixture. The instrument now selects the **latest**
non-RESOLVED→RESOLVED transition, which is the one that produced the current resolution. Rewords and
removed-only hits are still never transitions. New arm **A14** covers the reopen case. The A3 alias
loop now expects the latest flip. The mutant that restores earliest selection (a `[ "$flip" = none ]`
guard, applied on a disposable copy) turns **A14** and **A3** red. The suite passes 115/0. The real-history
verdicts for D-FLEET-1/13/14/15/16 are unchanged. Judge N3 is adopted as a one-clause limit in rule
6(b): git cannot see a squash-merged self-resolution. N2 (bold status cells), N4 (no attended-stamp
check) and N5 (three defensive guards no arm exercises) stay as follow-ups, not ticketed.
