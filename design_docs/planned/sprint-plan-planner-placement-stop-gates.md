# Sprint plan: planner placement and human-decision STOP gates

**Sprint:** M-PLANNER-PLACEMENT-STOP-GATES · **Status:** planned · **Date:** 2026-10-29
**Ticket:** `sprint-planner:template-make-test-changelog-filesize` (attended filing 2026-10-09T06:22Z; blocking=none).
**Design authority:** controller-agreed direction in fleet iteration 68; this document is the design/plan artifact.
**Estimate:** 3 sequential milestones, ~3 hours, ~180 source/test/doc LOC plus generated mirrors; low risk.
Capacity is a task estimate, not measured velocity; the supplied first-party measurements govern scope.

## Problem and measured evidence

The ticket records repeated attended repairs to plan PRs #1694, #1696, #1700–#1703 and earlier #1674–#1676, #1678, #1692–#1693.
Its six asks covered full-suite executor crashes (8 GiB, RAM-backed `/tmp`, SIGBUS), invalid changelog placement, file-size growth, triage-directory plans, unresolved human gates, and missing jq.
The controller measured `origin/dev` at `4e6ea7b25`; accept that audit without re-deriving history:

| Ask | Measured disposition / source |
| --- | --- |
| 1–3: testing, changelog, size | Already landed in attended #1770, `dbe4b26a3` (2026-10-12): [planner checklist](../../.claude/skills/sprint-planner/SKILL.md), mirrored in `.agents`. Defaults are `make test-changed` (focused packages + `make test-core`), fragments in `changelogs/unreleased/`, companion files near 800 lines. Preserve these rules. |
| 6: jq image | Already landed in `b2d21a070` (2026-10-09): [agent-base Dockerfile](../../docker/Dockerfile.agent-base), apt install includes jq. No image change. |
| 4: plan placement | Still open. `17254535d` added `run-policy-worker-cache-dir-sprint-plan.md` inside triage (later moved to `v0_52_6/`); `a65e6c12b` added `m-process-result-semantics-sprint-plan.md` there. |
| 5: human gates | Still open. No planner rule distinguishes OPEN from ruled decisions or requires terminal STOP behavior. |

[TestTriageRowsInTreeObeyTheRubric](../../internal/coordinator/triage_rubric_test.go) at line 108 checks every `*.md` row in `design_docs/planned/ailang-core-triage/`; misplaced plans redden CI (#1700).
The existing versioned path examples imply proper placement but did not prevent two violations.
Good STOP precedent: [subscription-lane plan, lines 6–19](v0_52_6/m-codex-subscription-lane-sprint-plan.md).
[Context baseline](../../scripts/context_docs_baseline.txt) line 24 allows 597 planner skill lines; the measured current count is 596. The baseline is outside fleet scope.

## Direction and boundaries

Only change `.claude/skills/sprint-planner/**`, its generated `.agents` mirror, `make/test.mk`, and the named changelog fragment.
Plans and associated files belong in `design_docs/planned/<version dir>/`, or the planned root as `sprint-plan-<slug>.md`; never in the triage directory, even for a design row located there.
An executor human-decision gate requires a still-OPEN decision and a citation to its design section or decision-ledger row ID. Copy already-ruled decisions into the plan as rulings, never gates.
Each permitted gate says: **STOP: commit current work, report `BLOCKED: <decision>`, and end the run**. No open-ended “pause for human review” or “confirm with the user”.
No new human decision is open for this sprint; execute M1–M3 sequentially once the plan is approved through the existing pipeline.
Registry reuse: `none` for M1–M3; instructions, host-shell validation and release notes are not package-like capabilities. Reuse existing scripts; no registry dependency/search is applicable.
No `.ail` or Go edits, new runtime behavior, full `make test`, CI/docker edits, rubric edits, or baseline bump.

## Milestones

### M1: Skill rules — M1_SKILL_RULES (~30 LOC; 35 minutes)

Files: `.claude/skills/sprint-planner/SKILL.md`, `resources/coordinator.md`, `resources/sprint_plan_template.md`, and generated `.agents/skills/sprint-planner/` equivalents.
Add ONE checklist bullet line containing both PLAN LOCATION and HUMAN-DECISION GATES rules; net growth at most one line (597 maximum).
Put expanded OPEN/ruling/citation/terminal-STOP wording and the placement rationale in the template resource, linked from that bullet. Add one line per rule in the linked coordinator resource.
Confirm all `SPRINT_PLAN_PATH` / `plan_path` examples remain outside triage. Preserve the 2026-10-12 focused-test/changelog/companion-file bullet verbatim.
Generate mirrors with `/bin/bash tools/launchd/sync-agents-skills.sh`; never hand-edit `.agents`. Inspect the diff for unrelated mirror drift; leave unrelated work parked.

Acceptance (all exit 0):

- `wc -l .claude/skills/sprint-planner/SKILL.md` → at most 597; growth from measured 596 at most one.
- `rg -n 'PLAN LOCATION|HUMAN-DECISION GATES' .claude/skills/sprint-planner/SKILL.md .claude/skills/sprint-planner/resources/coordinator.md` → both rules visible; one added checklist line in SKILL.md, two coordinator lines.
- `rg -n 'OPEN|BLOCKED:|STOP|ruled|decision-ledger' .claude/skills/sprint-planner/resources/sprint_plan_template.md` → explicit citations, ruled-decision handling and terminal behavior; review against the direction above.
- `rg -n 'SPRINT_PLAN_PATH|plan_path' .claude/skills/sprint-planner/SKILL.md .claude/skills/sprint-planner/resources/coordinator.md` → examples use permitted locations; no example places a plan in triage.
- `/bin/bash tools/launchd/sync-agents-skills.sh --check` → `agents skills in sync (...)`.
- `bash scripts/check_context_docs.sh` → `context docs: ... within budget`; `/bin/bash tools/launchd/test_agents_skills_sync.sh` → `0 failed`; `make check-skills` → success, no new invalid skill.

### M2: Create-JSON placement guard — M2_CREATE_JSON_GUARD (~140 LOC; 110 minutes)

Depends on M1. Files: `.claude/skills/sprint-planner/scripts/create_sprint_json.sh`, new sibling `test_create_sprint_json.sh`, `make/test.mk`, and regenerated planner mirrors.
Place the guard immediately after argument validation/assignment, before `mkdir`, any import, JSON/temp-file writes, or other side effects.
Reject a sprint-plan path inside the triage directory with exit **2** and exactly one diagnostic line naming **PLAN LOCATION** and `design_docs/planned/ailang-core-triage/`.
Check both the given path and its absolute/canonical resolution, anchored to the script's repository and the caller's CWD as appropriate. Resolve directory/file symlinks and `..`; use component boundaries, not substring matching.
Use Bash 3.2 syntax (no associative arrays or `${v,,}`); Python 3 path resolution is available already in this script. Do not require GNU-only `realpath`.
The optional third argument remains allowed to name a triage design row: the restriction applies to the sprint-plan argument, not the design argument.

The test copies the script into a throwaway repo-shaped fixture inside a `mktemp -d` tree; all fixture plans, CWDs, HOME, TMPDIR, logs and `.ailang/state` live there, with cleanup on EXIT.
Put a stub `ailang` first on PATH; it records argv to a fixture file and exits 0. Assert its resolution before invoking the copied script. Never call the real binary or seed real state.
Every refusal arm asserts rc=2, one-line diagnostic with rule/directory, empty stub log, and absence of the fixture output directory/JSON. Use a fresh fixture per arm.
The valid-path control runs the copied remainder against the stub only, without a design doc: rc=0, stub records `messages import-github`, valid JSON exists only in fixture `.ailang/state/sprints/`.
Also allow a valid plan plus a triage design argument; stub all message operations. This proves the design/plan distinction without touching production messaging.
Wire explicit `/bin/bash -n` lines for both scripts and a `/bin/bash` test run into `test-launchd-drivers`, alongside the sibling cloud-branch-sync syntax gates. Preserve the existing suite runner.
Re-run `/bin/bash tools/launchd/sync-agents-skills.sh` after source edits.

Acceptance (commands run from repository root):

```bash
for tree in .claude .agents; do
  for name in create_sprint_json test_create_sprint_json; do
    /bin/bash -n "$tree/skills/sprint-planner/scripts/$name.sh" || exit 1
    shellcheck -S warning "$tree/skills/sprint-planner/scripts/$name.sh" || exit 1
  done
done
/bin/bash .claude/skills/sprint-planner/scripts/test_create_sprint_json.sh
/bin/bash tools/launchd/sync-agents-skills.sh --check
/bin/bash tools/launchd/test_agents_skills_sync.sh
make test-launchd-drivers
```

Syntax and shellcheck: rc=0, no diagnostics. New test: rc=0, named PASS lines for every arm below and a zero-failure summary. Mirror gates: rc=0, sync confirmation / `0 failed`.
Driver target: rc=0 ending `launchd drivers: tests + bash 3.2 syntax OK`; includes the new test's PASS lines. Compare any existing failures with the controller-supplied base result, report them explicitly, and do not silently waive a new failure.
OBSERVATION: read-only before/after `find "$HOME/.ailang/state" -type f -exec shasum -a 256 {} \;` manifests (if the directory exists) show no test-caused changes; never create a sentinel in real HOME. Record absence if absent, and distinguish concurrent unrelated changes.
The stub log and fixture assertions prove no real messaging invocation; do not query or seed the production message store to test isolation.

#### Mutation drills (temporary copies only)

Make the test accept an optional script-under-test path, still copying that source into its fixture. For each named mutant, run `/bin/bash .claude/skills/sprint-planner/scripts/test_create_sprint_json.sh "$mutant"`; expect nonzero and the corresponding named FAIL. Keep mutants in the throwaway tree; never edit production/shared state.

| Arm | Input / assertion | Named mutation that must turn the arm red |
| --- | --- | --- |
| raw-refusal | `design_docs/planned/ailang-core-triage/x.md`, including nonexistent leaf | delete guard call |
| dotted-refusal | `./design_docs/planned/ailang-core-triage/x.md` | compare raw path only with an unprefixed exact prefix |
| foreign-cwd-refusal | absolute forbidden path, invoked from another fixture CWD | only recognize repo-relative input |
| canonical-refusal | safe-looking symlink alias and a normalized `..` path into triage | remove canonical check, compare raw spelling only |
| before-mkdir | every refused case has no fixture `.ailang/state` | move guard below `mkdir -p` |
| before-import | every refused case has an empty stub log | move guard below `import-github` |
| diagnostic-contract | refused rc=2 and one rule/directory line | change rc to 1 or omit directory |
| valid-control | versioned plan and root `sprint-plan-ok.md` produce fixture JSON | unconditionally reject every plan |
| boundary-control | sibling `ailang-core-triage-extra/` is accepted | use a substring/prefix without directory boundary |
| design-control | legal plan with triage design argument accepted | apply the guard to the third argument |

### M3: Record — M3_CHANGELOG (~10 LOC; 15 minutes)

Depends on M1–M2. Create `changelogs/unreleased/2026-10-29-planner-plan-location-stop-gates.md`.
Start with `### Fixed — ...`; explain placement refusal before side effects and OPEN-only terminal STOP gates. Link this plan; follow [fragment format](../../changelogs/unreleased/README.md).
Acceptance: `make check-changelog` → rc=0, fragment/fold checks succeed; `head -1 changelogs/unreleased/2026-10-29-planner-plan-location-stop-gates.md` → `### Fixed — ...`.
Final `bash scripts/check_context_docs.sh`, `make check-skills`, and mirror `--check` → rc=0. Repeat script/driver gates only if further edits affect them.

## Risks, reach and completion

Canonicalization must handle symlinks, other CWDs and nonexistent leaves without writing; the fixture cases prevent a spelling-only guard. Keep touched scripts/resources below 800 lines; SKILL.md has the stricter 597-line cap. No Go file grows.
Mirror sync also covers other fleet skills: inspect generated changes and do not absorb unrelated drift. Do not change `internal/**`, Dockerfiles, CI, or `scripts/context_docs_baseline.txt` to make gates pass.
Reach: coordinator planners read the skill from their executing image/checkout. The next `sunholo-data/ailang` coordinator sprint-planner task checkout containing this merge (or the next skill-bearing `agent` image built from `docker/Dockerfile.agent` containing it) is the rollout boundary; existing checkouts/images retain old instructions. Name the actual SHA/digest when observed; image rebuild/deployment is outside this sprint.
Completion evidence: source/mirror diff, gate results, named mutant failures and read-only isolation observations. Deliver M1–M3 without inventing a human gate; report genuine external blockers as terminal `BLOCKED: <reason>`.
