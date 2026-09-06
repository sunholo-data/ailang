# Independent evaluation — iter337 docs-only park of m-pi-runner-worktree-assertion-vacuous-on-revision

**Reviewed SHA:** `a46e4c015cecb26ad06a3b44d5b6f8895a2d111e`
**Disposition under review:** `needs-human-review D-58` (docs-only park, no implementation)
**Evaluator scope:** this report judges **whether the banked evidence and the parking decision honestly represent the unresolved design**. It is **not** approval to implement.
**Method:** bounded canonical-inbox list (45s deadline, no ack per evaluator guidance), read CLAUDE.md and `sprint-evaluator/SKILL.md`, read the design doc, R1/R2 quorum records, evidence JSON, consumer census, and the controller addendum. Independent first-party inspection of `scripts/mission_pi_run.sh`, `scripts/test_mission_pi_run.sh`, and a repo-wide grep for `mission_pi_run|empty_worktree|worktree_changed_files|snapshot_error` outside the design/sprint-retros paths. No full Go suite, no implementation, no git write, no inbox ack, no approval of blocked design.

---

## 1. What was actually banked (first-party verification)

The HEAD commit `a46e4c015` is "docs: bank rejected pi runner design and iteration337 gate evidence". It touches **only** the design doc and four retro artifacts — no source files in `scripts/`, `internal/`, or `cmd/` were modified. Working tree is clean. There is no `sprint_*.json` for D-58 and no implementation artifacts anywhere in `.ailang/state/sprints/`. The park is genuine: the doc remains in `design_docs/planned/v0_35_2/`, status `needs-human-review`, and no production change has been made.

## 2. Does the evidence honestly represent the unresolved design?

### 2.1 The root-cause claim is correctly verified

The doc's central claim — that `scripts/mission_pi_run.sh` decides the `finished` verdict from `git status --porcelain | wc -l`, so a revision pass over an already-dirty or already-untracked file yields `ok` while writing nothing — is **first-party verifiable and correct**:

- `scripts/mission_pi_run.sh:232` — `DIFF_LINES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')`.
- `scripts/mission_pi_run.sh:242-243` — `if [ "${DIFF_LINES:-0}" -gt 0 ]; then VERDICT_NAME="ok"; RC=0; else VERDICT_NAME="empty_worktree"; RC=10`.
- `scripts/mission_pi_run.sh:60` is `set -u`; `scripts/mission_pi_run.sh:146` is `set -m`. There is **no** `set -e`, **no** `set -o pipefail`, and **no** `trap` in the file (negative grep verified).
- The defect is **codified** by the existing suite: `scripts/test_mission_pi_run.sh:42-45` and `:79-82` use `mkrepo … dirty` plus a writeless stub and assert `ok`/rc 0. That is exactly the false-green the runner exists to detect.

So the design's `Problem Statement`, `Current State`, and V1–V3 verification rows are honest.

### 2.2 Quorum R1 and R2 records are consistent and complete

`iter337-pi-runner-quorum-r1.json` and `iter337-pi-runner-quorum-r2.json` both record all three external reviewers present (`gpt6-astra`, `gemini-3-1-pro`, `oc-glm-5-2`) with cost, token counts, full `strongest_objection` text, `catch`, and `proposed_fix`. Both round verdicts are `reject` from each reviewer and the in-session controller; `synthesis.verdict` is `blocked` in both rounds. `iter337-pi-runner-evidence.json` corroborates this with matching reviewer IDs, verdicts, and the same costs (R1 total ≈ $0.111, R2 total ≈ $0.130). The records are internally consistent and the verdict is unambiguously 3/3 reject in both rounds.

### 2.3 The addendum honestly corrects earlier false claims

The controller addendum (lines 358–384 of the design doc) overrides several earlier claims with first-party evidence:

- **Tracked file size.** Doc originally said "≈3.7 MB"; the actual measured tree is **24,380 tracked files, 268,361,724 bytes (≈256 MiB)** — `evidence.controller_revision_checks.tracked_regular_bytes = 268361724`. The addendum also says the prototype completed one local run in 5.442 s (rc 0, 24,381 manifest records); the evidence JSON confirms the same. The addendum is careful to call that "one local timing, not a general performance bound" — honest scope, not a benchmark claim.
- **`git hash-object`.** Original doc asserted it "writes to the object DB"; the controller's synthetic-repo negative control (without `-w`) and `-w` positive control showed that only the `-w` form writes. The earlier blanket claim is corrected.
- **Symlink no-follow.** Original prototype used final-leaf-only `-L`. The controller's synthetic-repo experiment — committing `d/f`, replacing `d` with a symlink to an external directory, and observing that enumeration and shasum followed it — shows the no-follow guarantee is **not** actually established by V6/V9. The addendum says so plainly.
- **Command substitution strips trailing newlines from link text.** Original prototype claimed exact link-byte identity; the addendum correctly notes this needs an explicit acceptance arm before any implementation can claim it.
- **Consumer census.** The addendum claims 106 matching lines in 14 files from a repo-wide `rg` over py/go/sh/md, with only the runner and shell suite as executable consumers. I re-ran a narrower `grep -rln` with the same patterns outside the doc/sprint-retros paths and got six files: `scripts/mission_pi_run.sh`, `scripts/test_mission_pi_run.sh`, `.claude/skills/mission-control/SKILL.md`, `docs/internal/harness-upgrade-runbook.md`, `docs/docs/guides/debugging.md`, `changelogs/v0.32-current.md`. Of those, **only** `mission-control/SKILL.md:1403-1445` documents the rc taxonomy operationally; the other hits are descriptive prose. This is consistent with the addendum's stronger claim that there are no executable consumers outside the runner+suite, and **confirms GLM's R2 objection is now resolved** (their proposed V14 repo-wide census is essentially banked).
- **Backlog defects are honestly classed.** The pre-existing T3 stall-timing flake and the absence of Make/CI wiring for the shell suites are flagged as **backlog defects**, not as evidence that the fix passed CI. That distinction is exactly right and prevents a future reader from mis-attributing green status.

The disclosure is sufficient: the addendum explicitly says it overrides the earlier claims and lists each correction. A reader who only saw the body would be misled; a reader who reaches the addendum is correctly informed.

### 2.4 The retained-but-rejected draft is properly framed

Lines 348–356 of the doc make clear that "the preceding text is the rejected designer revision, retained as evidence, not an approved implementation contract." Status line says "needs-human-review — iteration337, D-58; not approved for execution." The doc remains in `design_docs/planned/`, not `design_docs/implemented/`. The framing is honest.

### 2.5 The narrow-refinement decline is defensible

The task description summarizes the controller's reason for declining a narrow refinement: unresolved snapshot-failure ordering plus final artifact aliases need design judgment, and reviewer feedback lacks verbatim fixes for those two. I verified this against the doc:

- **Snapshot-failure ordering.** The doc says (L82) "before-snapshot is taken after output truncation and **before** the pi job launches" and (L188) "Dispatch: in the `finished` branch, `if SNAP_ERR ⇒ snapshot_error rc 15`". A before-snapshot that fails **before launch** has not yet entered the `finished` branch; the dispatch site is therefore ambiguous between "before launch" and "post-`wait`". This is not something a verbatim reviewer-fix can resolve — it requires choosing a design point (kill the run, defer the verdict, refuse to start). R2's controller note flags exactly this.
- **Final-component symlink in artifact exclusion.** The doc (L162-164) says: "Resolve `OUT`, `SNAP`, `ERR`, `VERDICT` to **physical absolute** realpaths (`cd … && pwd -P`, symlinks resolved)." But `cd … && pwd -P` resolves the containing directory, not a final-component symlink in `OUT`. To handle a symlink at the leaf, the proposal needs `readlink -f` (or equivalent). The text does not say which it is doing, and no reviewer proposed a verbatim fix. This too is a design-judgment item.
- **Comparison not covered by the shown watchdog.** L112 says `SNAPSHOT_DEADLINE` "covers enumeration + hashing + sort + compare", but the watchdog calls in L103-105 only wrap `content_snapshot`. The compare/delta step is not wrapped, so the doc's claim is internally inconsistent. A reviewer fix would be "wrap the compare in the watchdog too" — that is the kind of small change the addendum's verbatim-fix category was supposed to admit, but the controller chose to park it as part of the broader unresolved ordering.

These are real, bounded defects in the design. Declining to narrow-refine and instead parking for human ruling (D-58) is a **conservative and honest** call given that the doc's only externally-bounded failure mode (deadline) is currently inconsistent with the dispatch site and the artifact-alias contract is half-specified.

### 2.6 Banked evidence integrity

`iter337-pi-runner-evidence.json` carries the baseline (`make_test_rc=0`, `pi_suite.rc=0`, `assertions=9`), the live repro (`clean rc=10`, `dirty rc=0`, `untracked rc=0`), preflight (`gh_account=sunholo-voight-kampff`, `billing_tripwire=CLEAN`, `open_decisions=[D-55, D-56, D-57]`), designer run details (model, runner_rc=0, 538s, 39 tools, 1 file), and both quorum totals. The `controller_revision_checks` and `controller_prototype_control` blocks match the addendum's first-party measurements (24380 files, 268361724 bytes, prototype 5.442 s, rc 0, 24381 records). The disposition field is `needs-human-review D-58` — consistent with the doc's status line.

The one mildly weak point: the evidence's `baseline.pi_suite` says `assertions=9` while the doc's "Estimated" line and the addendum note T3 had a pre-existing timing flake at planner 8/9, controller 9/9 under `POLL1`. The `assertions=9` is the controller count under `POLL1`, not the planner count — the disclosure is in the addendum, not the evidence JSON. Minor and not load-bearing for a docs-only park.

## 3. Bounded checks performed (this evaluator)

- `git rev-parse HEAD` → `a46e4c015cecb26ad06a3b44d5b6f8895a2d111e` ✓ matches.
- `git status --porcelain` → empty ✓ working tree clean.
- `git show --stat HEAD` → 5 files changed, 818 insertions, all docs. ✓ no source modifications.
- `grep -n "set -\|pipefail\|^trap \|DIFF_LINES\|VERDICT_NAME" scripts/mission_pi_run.sh` → `set -u` L60, `set -m` L146, `DIFF_LINES` L232, `VERDICT_NAME="ok"` L242, `worktree_changed_files` L257. Zero `pipefail` and zero `trap` hits ✓ matches V1.
- `grep -n "mkrepo.*dirty\|check.*happy path\|empty_worktree" scripts/test_mission_pi_run.sh` → TEST 1 L42-45 and TEST 5 L79-82 use `mkrepo dirty` + writeless stub + assert `ok`/rc 0. ✓ defect codified, matches V3.
- `grep -rln "mission_pi_run\|empty_worktree\|worktree_changed_files\|snapshot_error" --include='*.py' --include='*.go' --include='*.sh' --include='*.md' .` outside the design/sprint-retros paths → six files (`scripts/mission_pi_run.sh`, `scripts/test_mission_pi_run.sh`, `.claude/skills/mission-control/SKILL.md`, `docs/internal/harness-upgrade-runbook.md`, `docs/docs/guides/debugging.md`, `changelogs/v0.32-current.md`). Only `mission-control/SKILL.md:1403-1445` documents the rc taxonomy operationally; others are descriptive prose. ✓ matches the addendum's consumer-census claim.
- `ls .ailang/state/sprints/` and `find . -name "sprint_m-pi-runner*"` → no sprint JSON, no implementation artifacts ✓ docs-only.
- Quorum records R1/R2 cross-checked against evidence JSON (costs, reviewer IDs, verdicts, totals). ✓ consistent.

## 4. Caveats and concrete blockers (scoped to docs-only park)

**The banked evidence and the parking decision honestly represent the unresolved design.** Two minor disclosure caveats, neither of which is load-bearing for a docs-only park:

1. **Axiom-label alignment.** The addendum says "the axiom labels above need alignment with the canonical axiom names before approval." I did not independently verify the axiom names against `design_docs/PROGRAM.md` (out of scope for a docs-only park). If a future revision lifts D-58, that alignment is a pre-condition; the disclosure is already there.
2. **Performance claim softening.** The body still has "full `content_snapshot` **≈5.5 s** measured" at L221 inside the Conflict Surface. The addendum correctly relabels this as "one local timing, not a general performance bound". A reader who skims only the body could still read 5.5 s as a measured bound. The disclosure is present but not in the same paragraph as the number. Not a defect, but a tighter wording ("one prototype run on this tree; not a general bound") would harden the disclosure.

**Concrete blocker that justifies the park, not a critique of the banked evidence:**
- Snapshot-failure dispatch point is ambiguous (before-launch vs. post-`finished`).
- `pwd -P` on the parent directory does not resolve a final-component symlink in `OUT`/`SNAP`/`ERR`/`VERDICT`.
- Compare/delta is claimed deadline-covered but the watchdog in the proposal only wraps `content_snapshot`.

These are exactly the items the task description flagged as needing design judgment, and the addendum correctly declines to narrow-refine them.

## 5. What this evaluation does **not** say

- It does **not** say the design is ready to implement.
- It does **not** authorize a third quorum round.
- It does **not** resolve D-58 in either direction.
- It does **not** assert that the prototype's behavior under hostile concurrent writes, an external-blocking-fifo symlink, or a tracked path whose ancestor is a symlink is correct — the addendum itself flags those as unresolved.
- It does **not** approve the `git hash-object` substitute or the `git diff --name-only` alternative that R1 reviewers raised; the addendum does not claim either is sufficient.

## 6. Score and disposition

**Score: 82/100**

Breakdown (adapted for docs-only park; the standard sprint rubric's test/lint/code-quality categories do not apply):

| Category | Points | Notes |
|---|---|---|
| Evidence honesty (root cause + repro + verification log) | 25/25 | First-party verifiable, R1+R2 + addendum consistent. |
| Park justification (R2 3/3 reject + controller reject + bounded narrow-refinement decline) | 25/25 | Defensible; the three blockers named in §4 are genuine and need design judgment, not verbatim fixes. |
| Addendum discipline (corrects earlier false claims, frames retained draft, names unresolved items) | 20/25 | Strong. Two minor disclosure caveats in §4. |
| Consumer / conflict-surface coverage | 10/10 | GLM R2 census is banked; only operational rc-taxonomy consumer is `mission-control/SKILL.md`; the runner+suite are the only executable consumers; corpus code-changes list is correct. |
| Implementation / CI coverage (N/A — docs-only) | 0/0 | Out of scope; this is not an implementation pass. |
| Performance / correctness under hostile inputs | 2/15 | One local prototype timing is banked, properly scoped; no general bound, no concurrent-writer test, no ancestor-symlink test. Documented as unresolved. |
| **Total** | **82** | |

**PASS (docs-only park).** The banked evidence and the `needs-human-review D-58` disposition honestly represent the unresolved design. The retained rejected draft is properly framed as evidence, not a contract. The addendum's first-party corrections are sufficient disclosure of the earlier false claims (size, hash-object, no-follow, command substitution, quiescence). Narrow-refinement was correctly declined for the three named blockers. **This evaluation is not approval to implement.** A future lift of D-58 must (a) pick a snapshot-failure dispatch point and write it down, (b) replace `pwd -P` with `readlink -f` (or equivalent) for final-component symlinks in the artifact paths, (c) extend the watchdog to wrap compare/delta or narrow the deadline claim, and (d) align the axiom labels with the canonical set before re-entering quorum.

EVALUATION_RESULT: pass
EVALUATION_SCORE: 82/100
EVALUATION_REPORT_PATH: docs/sprint-retros/iter337-pi-runner-independent-evaluation.md
