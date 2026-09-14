# M-PIN-DRIFT-BLIND-UNDER-SHA-PIN: PIN_AGE — a stale deployment pin reports its distance from origin/dev

**Status**: Planned
**Version**: v0.38.6
**Priority**: P2 (instrument gap; the fleet is currently on origin/dev so no live harm today)
**Estimated**: 0.5 day
**Author**: gpt-6-astra (codex), V1 mission iteration 354
**Queue row**: `m-pin-drift-blind-under-sha-pin` (charter design_docs/v1-mission.md; iter-349/353)
**Planner-Lane**: codex-ok

Scope: one milestone; shell instrumentation and its tests, plus the requested changelog entry.
This designer writes only this document. Implementation and Git writes belong to the controller's
later workflow. Quantities in examples and acceptance fixtures below are proposed inputs unless
explicitly marked measured. V identifiers point to commands and observed output in Verification Log.
The priority's fleet statement is supported by the configuration observation V9, not a claim that
this instrument has already shipped or that every running process's environment was inspected.

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Define the reading by an explicit Git reachability range |
| A2: Replayability | +1 | Ref, target SHA, and captured full baseline SHA make the measurement replayable |
| A3: Effect Legibility | 0 | No language effects change |
| A4: Explicit Authority | 0 | Reporting grants no permission to change pins |
| A5: Bounded Verification | +1 | Synthetic repositories and mutation-linked assertions |
| A6: Safe Concurrency | 0 | Preserve the driver's state ownership |
| A7: Machines First | +1 | Separate machine-readable age from clone drift |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No billing/accounting changes |
| A10: Composability | 0 | Additive helper contract |
| A11: Structured Failure | +1 | Unknown is explicit and never converted to a healthy zero |
| A12: System Boundary | 0 | Shell driver boundary unchanged |

Design assessment: net +5 (sum of this table); proceed to planning after review.

### Hard Violation Check

- [x] No new implicit authority over deployment pins or user repositories.
- [x] No silent zero on measurement failure or an unsupported helper.
- [x] Tests isolate HOME, pin worktrees, notifier channels, and state under temporary roots.
- [x] No language semantics, effects, or capability boundary changes.

## Problem Statement

The helper defines `PIN_DRIFT` as source-clone distance from the chosen ref and computes
`git -C "$src" rev-list --count "HEAD..$ref"`; its value crosses exec as
`AILANG_DRIVER_DRIFT` and is restored on the pinned pass [V2, `cat`].
For a pin reachable from source HEAD, that set is empty even when the pin is old.
Merely containing a commit object is insufficient: the zero argument requires reachability.

Re-measurement on the rig source clone gives the distinguishing pair:
`HEAD..48c4a6e49` = **0** [V7, `git rev-list`], while
`48c4a6e49..origin/dev` = **266** [V7, `git rev-list`].
The source clone itself is **17** commits behind `origin/dev` [V7, `git rev-list`].
These measure different hazards: an interactive session in the source clone can be stale;
a frozen driver pin can execute old driver/skill code even when clone drift reports zero.

The current consumer has threshold/floor, dedupe-until-doubling, and a source-clone-specific
notice [V3, `sed`]. The launchd tree has **0** `PIN_AGE` matching lines against **32**
`PIN_DRIFT` matching lines [V8, paired `rg`/`wc -l`].
There is no age instrument under that name in that tree [V8].
The charter records the historical freeze and D-61's attended resolution [V10, `rg`];
its historical counts are documentary evidence, not fresh historical telemetry measurements.
The sampled current log confirms the zero-reading wording but does not establish the entire
historical date interval in the briefing [V11].

A critical bootstrap boundary: before computing drift, the helper loads its own definition
from the target commit, invokes that definition, and ultimately execs the target's driver
[V2, V3]. A SHA predating this feature therefore supplies neither the new computation nor,
if its driver is also old, the new driver decision block. The design must expose that limit.

## Goals

- Add `PIN_AGE`: commits reachable from fetched `origin/dev` but not the pinned target.
- Keep `PIN_DRIFT` semantics and its notice intact.
- Log age on every fire reaching pin decision; alert only for a known, threshold-crossing age.
- Give age a separate dedupe file and notice type, independent of drift.
- Prove branch control, stale SHA detection, exec transport, and unknown compatibility.
- Preserve Bash 3.2 portability and the existing regression assertions.

## High-Impact Decisions

| Decision | Rationale / evidence | Owner |
|----------|----------------------|-------|
| Always compute age for every ref | `origin/dev..origin/dev` measured zero [V7]; same formula handles SHA, tag, or another remote branch without classification. A different remote branch can meaningfully lack default-branch commits | Designer |
| Keep drift and age separate | Current range and interactive-clone wording are specific to a different hazard [V2/V3] | Designer |
| Unknown is log-only; retain state | Matches drift's unknown policy [V3/V5]; unavailable measurement is neither recovery nor escalation | Designer |
| Add a helper capability marker and pre-handoff warning | Target helper replaces caller function before measurement [V2]; consumer normalization alone cannot make an old pinned driver log | Designer |
| One implementation milestone | Changes form one producer/transport/consumer contract; split milestones would leave an incomplete instrument | Designer |

### Design Freeze

- `PIN_AGE` means commit count, not wall-clock time, linear ancestry depth, or total divergence.
- After target resolution and gate refresh, resolve `origin/dev^{commit}` once to immutable full
  `origin_dev_sha`; compute `git -C "$src" rev-list --count "$target..$origin_dev_sha"`.
- Capture the already fetched remote-tracking `origin/dev`; add no production fetch, network call,
  or branch classification. Concurrent source-clone fetches can move that ref [V18].
- Failed/empty baseline resolution returns `?` loudly; failed/empty/non-numeric counts also become
  `?`. Never retry against the mutable name or turn failure into zero.
- Export `AILANG_DRIVER_AGE` and `AILANG_DRIVER_AGE_BASE_SHA`; pinned-pass defaults are
  `${AILANG_DRIVER_AGE:-?}` and `${AILANG_DRIVER_AGE_BASE_SHA:-?}`.
- Age notice threshold defaults to 25; zero, negative, empty, or malformed overrides use 25.
  Like drift, positive overrides smaller than 25 remain valid: this is a non-positive floor,
  not a minimum of 25 for all inputs [existing behavior: V3].
- Age dedupe state must never read, write, or remove `PIN_DRIFT_FILE`.
- Unknown age logs loudly but sends no notice, creates no state, and does not remove existing state.
- Stale/disabled pin status logs age as skipped with status; no age notice competes with pin failure.
- No silent promise of retrofit: a wholly pre-change source helper AND target driver cannot run
  new logging. At least the outer helper or the final driver must contain this change.
- No production test hook and no environment-controlled command execution in `pin-root.sh` or
  `mission-control.sh`; test-only interception happens via a lab-owned PATH shim (AC-P) and the
  disposable pre-age helper fixture (AC-D), never a conditional branch in production code.

## Solution Design

### Overview

Add the new reading adjacent to drift in the helper. Carry it through exec and include it in
`PIN_NOTE`. Add an independent age decision adjacent to the drift decision, followed by an
independently extractable notice body near the existing drift notice.
Preserve the existing decision START/END anchors and the existing drift notice anchor [V3/V5].

### Architecture

**Producer and transport (H1).** Extend the helper's header contract with `PIN_AGE` and
`AILANG_DRIVER_AGE` / `AILANG_DRIVER_AGE_BASE_SHA`, and distinguish source drift from target age.
Initialize `PIN_AGE="?"` and `PIN_AGE_BASE_SHA="?"`; add local `age` and `origin_dev_sha`.
After gate refresh, the intended measurement and transport are:

```bash
age="?"
origin_dev_sha=$(git -C "$src" rev-parse --verify --quiet "origin/dev^{commit}") || origin_dev_sha="?"
case "$origin_dev_sha" in
  ''|'?')
    origin_dev_sha="?"
    printf '%s\n' 'driver pin age: unknown (?); origin/dev baseline resolution failed' >&2 ;;
  *)
    age=$(git -C "$src" rev-list --count "$target..$origin_dev_sha" 2>/dev/null) || age="?" ;;
esac
case "$age" in ''|*[!0-9]*) age="?" ;; esac
PIN_AGE="$age"
PIN_AGE_BASE_SHA="$origin_dev_sha"
AILANG_DRIVER_AGE="$age"
AILANG_DRIVER_AGE_BASE_SHA="$origin_dev_sha"
export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA
```

No production test hook or environment-controlled command execution is added: the production
helper carries no test-conditional branch between baseline capture and rev-list (r3, sol/glm).
AC-P observes the race from OUTSIDE the helper with a test-local `git` shim (Testing Strategy).
On the pinned pass set `PIN_AGE="${AILANG_DRIVER_AGE:-?}"` and
`PIN_AGE_BASE_SHA="${AILANG_DRIVER_AGE_BASE_SHA:-?}"`; append
`; pinned target ${PIN_AGE} behind origin/dev (baseline ${PIN_AGE_BASE_SHA})` to `PIN_NOTE`.
Keep the existing source-clone clause and its meaning. Log the captured baseline with age and
resolved target identity, including unknown readings, so the measurement can be diagnosed later.
On a new, unpinned invocation clear inherited `AILANG_DRIVER_AGE` and
`AILANG_DRIVER_AGE_BASE_SHA` before gate refresh; ambient readings must not masquerade as fresh.
Do not clear them on the already-pinned branch, which must consume the exec transport.

**Old-helper compatibility (H1).** Declare `PIN_AGE_SUPPORTED=1` at helper source time.
Immediately before sourcing the refreshed helper, unset that marker and reset `PIN_AGE="?"`.
After successful sourcing, before calling its replacement `pin_root_to_committed_ref`, inspect
`${PIN_AGE_SUPPORTED:-}`. If it is not `1`, print to stderr:

```text
driver pin age: unknown (?); helper at <ref> @ <target> lacks PIN_AGE; notice suppressed
```

Use `printf`, not a caller-specific `log` dependency. The driver's captured stderr is the
witness even if the old target driver never executes an age decision. Include the resolved
ref and full target in this warning. Do not manufacture a numeric reading in the outer helper
or bypass the gate refresh. The old helper remains authoritative for the actual pin operation.
The marker is a source-time capability handshake, not a new environment transport or pin policy.
The new driver's independent `${PIN_AGE:-?}` normalization handles an old helper when the
consumer is new; both compatibility surfaces need tests.
An entirely old initial helper plus entirely old target remains outside retrofit reach.

**Decision (H2).** Add `PIN_AGE_FILE` beside both existing drift-path assignments [V4]:

```bash
# V1 branch
PIN_AGE_FILE="$STATE_DIR/mission-control.pin-age"
# other missions
PIN_AGE_FILE="$STATE_DIR/mission-${MISSION_NAME}.pin-age"
```

For AC-O, wrap the existing mission path-selection `if`/`else`/`fi` in new
`# --- DRIVER PIN STATE PATHS START ---` / `# --- DRIVER PIN STATE PATHS END ---`
comments. Extract that actual assignment block into the synthetic harness; provide STATE_DIR
and MISSION_NAME, then observe both file variables. Do not retype the path-selection logic.

Inside the existing DRIVER PIN DECISION START/END region, append a separate block bounded by
`# --- DRIVER PIN AGE DECISION START ---` / `# --- DRIVER PIN AGE DECISION END ---`.
Initialize `_pin_age_degraded=""` unconditionally and normalize `PIN_AGE="${PIN_AGE:-?}"`.
For pinned status, use age-specific variables with the drift decision's shape [V3]:

| Reading / state | Action |
|-----------------|--------|
| Non-numeric or missing | Log `driver pin age: unknown ($PIN_AGE); notice suppressed`; leave state untouched |
| Numeric, below warning threshold | Remove only age file; log count, threshold, and `notice re-armed` |
| At/above threshold, previous absent/malformed | Arm age notice, persist count to age file |
| At/above threshold and at least twice previous | Arm age notice and persist new count |
| At/above threshold without doubling | Log count and `deduped until doubling from <previous>` |
| Status other than pinned | Log `driver pin age: skipped (status=<status>)`; leave age state untouched |

Use `_pin_age_warn`, `_pin_age_previous`, and `_pin_age_emit`; never borrow drift variables.
Log invalid `AILANG_DRIVER_AGE_WARN` with `using 25` before continuing.
Keep the additive age logic outside drift's pinned/STALE branches so every decision-reaching
fire has an age line, including a failed or disabled pin. No guarantee covers exits before
pin decision. Unknown must be handled before touching files or evaluating numeric comparisons.

**Notice (H2).** Use an anchored top-level block:
`if [ -n "$_pin_age_degraded" ]; then` through its own top-level `fi`.
Call `_mc_notify` with notice type `pin-age`; retain existing channel and spool machinery.
Suggested title: `Mission ${MISSION_NAME}: driver pin is stale (${_pin_age_degraded} behind)`.
Body must include:

```text
The driver is executing code N commits behind origin/dev.
Pinned ref: REF. Target SHA: SHA.
Baseline origin/dev SHA: <sha>
Every landed driver/skill fix newer than this pin is NOT in effect.
This notice repeats only when the measured pin age doubles.
```

Use `${AILANG_DRIVER_REF:-origin/dev}` [V19] and `${AILANG_DRIVER_PINNED:-?}` for identity.
Use `${AILANG_DRIVER_AGE_BASE_SHA:-?}` for the captured baseline; never resolve it again in the notice.
The transported SHA is the helper's resolved short SHA [V2]; label it as target SHA, never
substitute source HEAD or REPO HEAD. Missing identity is rendered `?`, not invented.
Describe no automatic pin reconciliation. Do not copy the source-clone interactive-session hazard.

### Implementation Plan

**M1 — producer, consumer, regression proof, changelog.**

- [ ] H1: helper contract, age computation, transport, note, and old-helper diagnostic.
- [ ] H2: mission age paths, age decision, and independently extracted age notice.
- [ ] H3: extend the synthetic pin lab and notifier harness with the acceptance arms below.
- [ ] H4: add an entry under the existing `## [Unreleased]` in
  `changelogs/v0.32-current.md`; preserve its current entry [V6].
- [ ] Run the suites and named mutation drills; record actual red assertions and exit codes.
- [ ] Run `make test-launchd-drivers` with isolated test environment; controller owns commits.

### Files to Modify/Create

| File | Planned change |
|------|----------------|
| `tools/launchd/lib/pin-root.sh` | H1 producer and compatibility contract |
| `tools/launchd/mission-control.sh` | H2 independent age decision/state/notice |
| `tools/launchd/test_pin_root.sh` | H3 observed age and transport arms |
| `tools/launchd/test_driver_notify.sh` | H3 extracted blocks, age arms, state independence |
| `changelogs/v0.32-current.md` | H4 prose entry; explicit non-shell exception requested by brief |

No new production file. No launcher, environment, charter, or skill edits.

## Examples

Proposed lab history: `A -> B -> C -> D`, with `origin/dev=D` and source HEAD=A.
The fake driver and new helper are committed in A, before any advance.
For `ref=origin/dev`, target D yields `AGE=0`; clone drift is `DRIFT=3` in this example.
For `ref=A`, target A yields `AGE=3` AND `DRIFT=0` on the same invocation.
This pairing detects the instrument gap; either number alone is insufficient.

Proposed notice sequence with threshold 25: age 30 emits and stores 30; age 30 or 59
is deduped; age 60 emits and stores 60; age 3 clears age state; age 30 emits again.
An intervening `?` logs unknown and preserves whichever age state existed.
These counts are test inputs, not measurements of the live rig.

## Success Criteria

Every acceptance row identifies a concrete mutation; all arms require successful subprocess
completion and positive evidence that the intended production block ran. Counts below are fixtures.

| ID / suite arm | Required observation | Mutation killed |
|----------------|----------------------|-----------------|
| AC-A / pin-age-branch | Existing case 1 still `DRIFT=1`, plus exact `AGE=0`, pinned status, note age clause | MUT-A: use `HEAD..origin/dev` for age, giving source drift instead |
| AC-B / pin-age-sha | Pin lab first commit A containing new helper; origin/dev exactly 3 ahead; same output has exact `AGE=3`, `DRIFT=0`, `STATUS=pinned`, note age 3 | MUT-B: delete age computation; MUT-C: reverse to `origin/dev..$target` |
| AC-C / pin-age-carry | Case 3 style: `AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_AGE=9` yields exact `AGE=9`; drift carry stays 7 | MUT-D: delete pinned-pass age readback |
| AC-C2 / pin-age-export | Real re-exec with origin/dev exactly 9 ahead of A, env AGE initially unset: fake driver shows exported `AILANG_DRIVER_AGE=9` AND `AGE=9` | MUT-E: remove AGE from export list (assignment remains) |
| AC-D / pin-age-old-helper | Actual refresh to synthetic pre-age helper; output has compatibility `driver pin age: unknown` and never `AGE=0`; updated-consumer extracted block also logs unknown | MUT-F: remove pre-handoff capability warning; MUT-G: default missing age to zero |
| AC-E / age-a | Age 25, threshold 25, absent age state: both stub channels receive pin-age notice with ref, SHA, count, and executing-old-code wording; age file 25 | MUT-H: use `-gt` instead of `-ge`; MUT-I: copy drift body or use wrong identity |
| AC-F / age-b | Age 170, previous 170: age state unchanged, positive dedupe log, no sends | MUT-J: always emit above threshold |
| AC-G / age-c | Age 340, previous 170: both channels, age state 340; companion 339 arm dedupes | MUT-K: use `-gt` at doubling; MUT-L: lower doubling boundary |
| AC-H / age-d | Age 3, previous 170: re-arm log, age state absent, no sends; following 25 emits | MUT-M: omit age-file removal |
| AC-I / age-f | `?` and malformed age, previous 170: unknown log, no sends, state unchanged | MUT-N: treat unknown as zero or remove state on unknown |
| AC-J / age-i | Age 3 with warning 0, -1, malformed, or unset: `using 25` for invalid explicit values; below-threshold log, no sends; positive override 2 emits | MUT-O: remove threshold validation/default or impose minimum 25 on every override |
| AC-K / age-j | Unset PIN_AGE under `set -u`: `DECISION_RC:0`, unknown log, no sends, state unchanged | MUT-P: bare unset-variable reference or default zero |
| AC-L / age-independent | Shared synthetic STATE_DIR, distinct age/drift files; sequence described below preserves separate counters | MUT-Q: use PIN_DRIFT_FILE anywhere in age read/write/remove path |
| AC-M / age-status | STALE/disabled: age skipped log, no pin-age sends, state unchanged; original failure notice still works | MUT-R: remove status guard |
| AC-N / age-measure-fails | Selectively fail the age rev-list in lab; `AGE=?`, no fabricated 0, pinned status; successful drift query as control | MUT-S: coerce failed/empty count to 0 |
| AC-P / pin-age-baseline-moves | Test-local `git` shim first in the synthetic harness PATH delegates every command to the real Git binary, but on the matching `rev-parse --verify --quiet 'origin/dev^{commit}'` call captures and returns the original SHA B while advancing/fetching the synthetic ref before control returns; real exec reports `AGE=N`, exported baseline SHA B, and note baseline B, although the moved ref has age N+1 — i.e. production `rev-list` received the captured immutable SHA rather than `origin/dev` | MUT-U: substitute mutable `origin/dev` for `$origin_dev_sha` in rev-list → AC-P red |
| AC-Q / pin-age-nondefault-ref | Lab `origin/feature` is N commits behind origin/dev, N ≥ threshold; fake driver exports `AILANG_DRIVER_REF=origin/feature`, `AGE=N`; extracted notice prints `Pinned ref: origin/feature`, never `Pinned ref: origin/dev`, with captured baseline SHA | MUT-V: hardcode `origin/dev` in notice ref line → AC-Q red |
| AC-O / age-paths | Extract actual mission path assignments; v1 and motoko resolve to their required pin-age names under synthetic STATE_DIR, distinct from drift | MUT-T: typo/shared age path in either production mission branch |

### Named mutations, anchored to the diff

H1 owns MUT-A through MUT-G, MUT-S and MUT-U; H2 owns MUT-H through MUT-R, MUT-T and MUT-V.
H3 supplies the killers; H4 is prose and needs review, not an artificial executable mutation.
Do not claim the injected carry arm kills a missing export: an externally exported AGE masks
that defect. AC-C2 must begin with AGE unset and observe a real exec. AC-B also detects most
transport failures, but AC-C2 isolates export from computation with a separately known count.
Apply each mutation independently to a disposable test copy during the executor phase,
record the failing assertion and rc, restore the fixed content, and re-run green.
Do not adjust fixture counts to make a surviving mutant red; report the design/test defect.

## Testing Strategy

Extend the existing real-origin lab [V12] without changing its first advance: case 1's
`DRIFT=1` must remain valid. Record A immediately after the base commit, which must contain
both the updated helper and fake driver. Add exact AGE and exported AGE output to that driver.
Unset `AILANG_DRIVER_AGE` and the capability marker alongside existing transport cleanup.
After existing cases finish, restore synthetic onboarding/ref defaults and advance origin
until A..origin/dev is exactly 3 (assert the count before invoking the SHA arm).
Later advance to exactly 9 for AC-C2. Keep source HEAD at A; fetched objects do not move HEAD.
AC-P (r3, reviewer-specified): place a test-local `git` shim FIRST in the synthetic harness PATH
for that arm only. The shim delegates all commands to the real Git binary (resolved once at shim
creation, never via PATH recursion), but on the matching `rev-parse --verify --quiet
'origin/dev^{commit}'` call it captures and returns the original SHA B while advancing the synthetic
origin by one commit and fetching into the source clone before control returns. Assert the shim
fired exactly once (a lab counter file), that `origin/dev` now differs from captured B, and that
production `rev-list` — unchanged — received the captured immutable SHA: AGE and the exported
baseline equal B's independently recorded count/SHA, with the moved ref's N+1 as positive control.
Remove the shim from PATH after the arm. No production file carries a test hook; do not retype
producer logic in the shim — it intercepts one Git invocation and delegates everything else.
AC-Q creates `origin/feature` at the lab's helper-bearing A, N ≥ 25 behind origin/dev; use the real
exec output as inputs to the extracted decision/notice with fresh age state and stub channels.
Assert both exported ref identity and the emitted ref line; an absent notice must fail the arm.
Also extend AC-N to fail baseline resolution selectively: `AGE=?`, baseline `?`, loud diagnostic,
with successful drift query as positive control. Include baseline transport in AC-C/C2 readback
and real-exec assertions, and clear inherited baseline alongside AGE in fixture setup.
All fixture Git writes are for the executor's disposable lab, not authorized in this designer run.

For AC-D create a separate synthetic target branch whose helper omits all H1 age additions,
while retaining the existing refresh/re-exec machinery. Generate this fixture by explicit,
checked transformations of the seeded helper; assert expected substitutions actually matched,
`/bin/bash -n` succeeds, age marker/computation is absent, and drift computation remains present.
Also remove the age output from that target fake driver to model a wholly old consumer;
assert the pre-handoff warning remains observable even without a target-side age print.
Do not depend on an external historical SHA being retained in a shallow CI checkout.
The outer helper must be the new helper, and the target blob must actually be loaded by
`git show`; assert pinned status, so a fixture failure cannot pass as the unknown outcome.
With an inherited bogus AGE, repeat and assert it is not carried as a valid target reading.
Run the extracted production age decision with age unset as a separate compatibility witness.

Extend the notification harness to extract the new notice with
`awk '/^if \[ -n "\$_pin_age_degraded" \]; then/,/^fi$/'` and fail if empty.
The whole pin-decision extraction must continue including both instruments; additionally
extract the new age START/END block for the compatibility-only arm. Never retype production logic.
Add age file, age/ref/SHA inputs, and notice sourcing to the drift harness with neutral AGE=0
for its existing numeric drift arms. Keep unknown-variable arms genuinely unset.
Initialize and capture each block's rc immediately, before echo/log/trace formatting.

AC-L must run real successive decisions in one temporary state root: drift=170/age=30
first emits both; drift=170/age=60 emits age only and leaves drift state 170;
drift=340/age=60 emits drift only and leaves age state 60;
drift=340/age=3 removes age state only. Assert both file contents and per-type stub sends
at each step, not merely total send count. This exercises read, write, and remove independence.

Use `mktemp -d` to allocate the outer test HOME and TMPDIR and per-arm STATE_DIR.
Keep notifier PATH stubs and unified trace; never invoke production channels [V5].
Guard live locations by observation only: before/after SHA-256 and line count for any existing
live pin-age/drift files, with `absent` a valid reading. Never seed, truncate, create, or delete
a sentinel at a real HOME or path outside the synthetic lab. Observe repository status too.
Acceptance commands: `/bin/bash tools/launchd/test_pin_root.sh`,
`/bin/bash tools/launchd/test_driver_notify.sh`, then `make test-launchd-drivers`.
Run syntax checks per touched shell file with `/bin/bash -n`; no GNU timeout, associative arrays,
`${v,,}`, `mapfile`, or `readarray`. Existing CI invokes the suites under macOS Bash 3.2 [V6].

## Ruled-out

- Inheriting the charter's SHA-only age/branch-age-absent sketch: always-compute is simpler
  and gives the branch control an exact zero [V7/V10].
- Replacing drift's range: would erase the source-clone hazard and alter existing assertions [V3/V12].
- Reporting old-helper absence as zero, or a threshold notice: neither is a measured age.
- Only adding consumer fallback: a target's old driver can lack that consumer too [V2/V3].
- Bypassing the gate refresh: changes authority over onboarding and which committed helper runs.
- Sharing dedupe state or automatically moving pins: confuses independent readings with policy.

## Deferred Decisions

Exact test helper names and log punctuation may be chosen by the executor; anchored extraction,
unknown prefixes, state isolation, identity, and mutation outcomes are frozen.

## Non-Goals

- No change to PIN_DRIFT semantics.
- No pin management; D-61 scopes the loop's pin authority as a bridge [V10].
- No change to the source-clone drift notice wording.
- No Go code, binary mission migration, skill edits, or production configuration writes.
- No wall-clock age, ahead-count, merge-base policy, or promise that age zero implies identical code.
- No retroactive logging from a completely unupgraded outer helper and pinned driver.

## Timeline

One milestone, estimated 0.5 day: helper/consumer contract, fixture arms, mutation proof,
then changelog and regression sweep. This is an estimate, not measured velocity.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Pinned helper predates the instrument | Pre-handoff capability warning plus new-consumer unknown fallback; explicitly document wholly old deployment limit |
| Remote branch missing or count fails | `?`, loud log-only, preserve dedupe state |
| Other-branch pin diverges | Define age as missing default commits, not an ancestry or safety judgment |
| Tests inherit real driver transport | Unset AGE with other pin variables; real-exec export arm begins clean |
| New age branch breaks existing nounset harness | Supply isolated paths/neutral inputs; separate unset arms retain coverage |
| Alert rewrites old pin-management policy | Notice describes stale execution only; no automatic mutation |
| Marker accidentally inherited across source | Unset it immediately before target helper sourcing; AC-D exercises the real hop |
| A test seam leaks into production | None exists: AC-P intercepts via a PATH shim outside the helper; the helper has no env-gated branch, so no variable can leak an executable into a live fire |

## Conflict Surface

| Existing surface / evidence | Touch and preservation requirement |
|-----------------------------|------------------------------------|
| Helper contract, pinned pass, gate refresh, drift measurement, exec export [V2/V19] | Add AGE and AGE_BASE_SHA to the existing PINNED/SRC/DRIFT/REF/MISSION_WORKDIR export list; read back both on pinned pass; preserve drift formula, ref identity and gate authority |
| Mission path branches [V4] | Add age names adjacent to drift; test production assignments for both missions |
| DRIVER PIN DECISION anchors [V3/V5] | Keep exact START/END lines; preserve drift text including `driver pin drift: unknown` |
| Drift notice top-level `if` / `fi` [V3/V5] | Byte-preserve existing wording and extraction; add sibling age block |
| Pin lab cases 1/3, fake-driver heredoc, env cleanup [V12] | Preserve DRIFT=1 and drift carry=7; seed new helper before capturing SHA |
| Notify drift-a through drift-j, unified PATH trace [V5] | Preserve all original assertions; neutral age inputs except explicit age arms |
| Baseline totals | Preserve all 81 pin assertions [controller briefing; V13 not rerun] and 50 notify assertions [V14]; added assertions increase totals |
| Make/CI [V6] | Existing launchd target remains the integration gate; no new CI job |
| Production authority chain (`pin_root_to_committed_ref`, `_mc_notify`) [V20] | Both functions exist and are the only two production APIs this design calls; no test-conditional branch is added to either path |

## Verification Log

Commands run in this worktree unless an absolute source clone path is shown. Output excerpts are
literal or explicitly summarized. Negative claims include a positive control in the same row.
No implementation or mutation result is claimed at design time.

| ID | Claim / exact command run | Observed output / interpretation |
|----|---------------------------|----------------------------------|
| V1 | `cat CLAUDE.md`; `git status --short; git rev-parse HEAD` | Operational instructions read. Status empty, positive control HEAD=`9422ff628f9086b40d7a2b69b9aacb3effd389d3`; clean base before document creation |
| V2 | `cat tools/launchd/lib/pin-root.sh` | Header PIN_DRIFT contract; `PIN_DRIFT="?"`; pinned-pass `${AILANG_DRIVER_DRIFT:-?}`; refresh `git ... show "$target:tools/launchd/lib/pin-root.sh"`, source and recurse before `drift=$(git ... rev-list --count "HEAD..$ref" ...)`; `AILANG_DRIVER_PINNED="$short"`; DRIFT assignment; full export set AILANG_DRIVER_PINNED, AILANG_DRIVER_SRC, AILANG_DRIVER_DRIFT, AILANG_DRIVER_REF, MISSION_WORKDIR; final exec of target driver |
| V3 | `sed -n '958,1030p;1718,1738p' tools/launchd/mission-control.sh`; `sed -n '950,965p' tools/launchd/mission-control.sh` | Sources helper then calls pin function; missing helper sets STALE/DRIFT ?. Decision anchors present; threshold default/fallback 25; unknown log-only; below-threshold rm; doubling comparison; own drift state; notice says interactive source-clone session, sends type `pin-drift` |
| V4 | `sed -n '96,114p' tools/launchd/mission-control.sh` | V1 `PIN_DRIFT_FILE="$STATE_DIR/mission-control.pin-drift"`; other missions `PIN_DRIFT_FILE="$STATE_DIR/mission-${MISSION_NAME}.pin-drift"` |
| V5 | `sed -n '1,185p;318,390p' tools/launchd/test_driver_notify.sh` | PATH ailang/gh stubs, unified MC_TRACE_FILE, production awk extraction incl. decision and drift notice; fresh temp state; run_drift inputs; drift-b equal 170, c 340, d 3/rearm, f ?, i threshold 0, j unset under set -u; no-send checks paired with positive decision/log/state checks |
| V6 | `rg -n 'test-launchd-drivers\|launchd drivers\|bash 3.2' make/test.mk .github/workflows`; `sed -n '59,88p' make/test.mk`; `sed -n '680,706p' .github/workflows/ci.yml`; `head -22 changelogs/v0.32-current.md`; `/bin/bash --version` | Make invokes both requested suites plus other launchd tests and syntax checks; CI job name `launchd drivers (bash 3.2)`, macos-latest, `make test-launchd-drivers`; changelog Unreleased followed by one Fixed entry before v0.38.5; shell `3.2.57(1)-release (arm64-apple-darwin25)` |
| V7 | `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-list --count HEAD..48c4a6e49`; `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-list --count 48c4a6e49..origin/dev`; `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-list --count HEAD..origin/dev`; `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-list --count origin/dev..origin/dev` | Respectively `0`, `266`, `17`, `0`. Positive nonempty ranges beside empty controls; supports always-compute decision without ref classification |
| V8 | `rg PIN_AGE tools/launchd/ \| wc -l; rg PIN_DRIFT tools/launchd/ \| wc -l` | `0` age lines; positive control `32` drift lines. Scoped to named symbols, not a claim about all conceivable age-related features |
| V9 | `rg -n 'AILANG_DRIVER_REF' /Users/voightkampff/.config/ailang/mission-{v1,docs,world,motoko}.env`; `rg -n 'AILANG_DRIVER_PIN=' /Users/voightkampff/.config/ailang/mission-{v1,docs,world,motoko}.env` | Only REF matches are default-origin/dev comments at v1:50, docs:162, world:127, motoko:41; positive control active `export AILANG_DRIVER_PIN=1` at v1:55, docs:167, world:132, motoko:46. File observation, not process-env inspection |
| V10 | `rg -n -A3 'm-pin-drift-blind-under-sha-pin\|^.*D-61.*RULED' design_docs/v1-mission.md` | D-61 RESOLVED, attended return-to-default authority explicitly a bridge; historical row records 43 then 100 ahead; queue row at :544-545 describes gap and SHA-only proposed sketch. This design intentionally replaces that sketch with always-compute |
| V11 | `rg -n -m 2 'driver pin drift: 0 below warning threshold 25\|driver pin drift: 17 below warning threshold 25' /tmp/ailang-mission-control.log`; `rg -n -m 1 'driver pin drift:' /tmp/ailang-mission-control.log` | Zero wording at :6 (2026-09-09), :54 (2026-09-10); positive general control :2 reports 15 on 2026-09-09. Briefing's full 2026-09-07..08 interval and exact 17 log not independently established by this limited sample |
| V12 | `sed -n '1,180p' tools/launchd/test_pin_root.sh`; `rg -n 'git \|HOME=\|mktemp\|^====\|passed,\|trap ' tools/launchd/test_driver_notify.sh tools/launchd/test_pin_root.sh` | Pin lab seeds copied helper and fake driver in base commit, makes stale clone then one advance; unsets PINNED/DRIFT/SRC/REF; synthetic HOME; case 1 asserts DRIFT=1; case 3 injects 7. Pin suite contains init/checkout/add/commit/push under temp lab. Notify suite uses mktemp state/traces; no Git write command matches, positive control pin suite has them |
| V13 | Static preflight V12 for requested `/bin/bash tools/launchd/test_pin_root.sh` | NOT EXECUTED: invokes Git writes, disallowed by this designer's explicit any-kind Git-write prohibition. `81 passed, 0 failed` is controller-reported baseline only; executor must remeasure, not inherit a designer green |
| V14 | `task_lab=$(mktemp -d /tmp/pin-age-notify.XXXXXX)`; `HOME="$task_lab/home" TMPDIR="$task_lab" /bin/bash tools/launchd/test_driver_notify.sh > "$task_lab/result.log" 2>&1`; `task_rc=$?`; `tail -4 "$task_lab/result.log"`; `printf 'rc=%s\n' "$task_rc"` | `PASS: direct send reaches child with store=gcp`; `PASS: wiring: exactly one preflight drain call`; `==== 50 passed, 0 failed ====`; `rc=0`. Actual designer execution, with synthetic HOME/TMPDIR and suite-owned notifier stubs |
| V15 | `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-parse --short HEAD origin/dev`; corrected with `git -C /Users/voightkampff/dev/sunholo-data/ailang rev-parse --short HEAD; git -C /Users/voightkampff/dev/sunholo-data/ailang rev-parse --short origin/dev` | First command failed: `fatal: Needed a single revision` (rc 128). Separate commands succeeded: `9845acf2c`, `9422ff628`. Failure does not invalidate independent rev-list readings V7 |
| V16 | `for f in tools/launchd/lib/pin-root.sh tools/launchd/mission-control.sh tools/launchd/test_pin_root.sh tools/launchd/test_driver_notify.sh; do /bin/bash -n "$f"; printf '%s rc=%s\n' "$f" "$?"; done` | Each file printed `rc=0`; syntax only, not suite execution |
| V17 | `cat design_docs/implemented/v0_38_6/m-debugcacheforms-flaky-on-macos-ci.md`; `sed -n '130,380p' design_docs/implemented/v0_38_6/m-debugcacheforms-flaky-on-macos-ci.md`; `sed -n '406,554p' design_docs/implemented/v0_38_6/m-debugcacheforms-flaky-on-macos-ci.md`; `rg -n '^##\|^###' design_docs/implemented/v0_38_6/m-debugcacheforms-flaky-on-macos-ci.md` | Reference read, key body windows re-read after long-output truncation; section list includes axioms, problem, decisions/freeze, solution, examples, success/mutations, tests, ruled-out, non-goals, risks, conflict, verification and quorum |

| V18 | `grep -rn "git.*fetch" tools/launchd/*.sh tools/launchd/lib/*.sh scripts/hooks/*.sh` | Controller-measured shared-source concurrent-fetch premise, confirmed locally: `tools/launchd/nightly-eval.sh:85: git -C "$REPO" fetch --quiet origin`, `tools/launchd/os-rotation-filler.sh:251: git fetch -q origin dev`, `scripts/hooks/push_dev_on_stop.sh:71: bounded 20 git fetch origin dev --quiet` (command excerpts). Positive control: helper's own fetch at `tools/launchd/lib/pin-root.sh:173`: `_pin_bounded "$fetch_s" git -C "$src" fetch --quiet origin; rc=$?`. The remote-tracking ref can move between fetch and rev-list. |
| V20 | `rg -q "pin_root_to_committed_ref" tools/launchd/lib/pin-root.sh && rg -q "_mc_notify" tools/launchd/mission-control.sh; echo rc=$?` | `rc=0`; matches confirm both function names exist — definitions at `tools/launchd/lib/pin-root.sh:145` (`pin_root_to_committed_ref() {`) and `tools/launchd/mission-control.sh:179` (`_mc_notify() {`, signature `title body label`). Negative control: `rg -c "_mc_notify_nonexistent_zz" tools/launchd/mission-control.sh` → no match. (Controller-measured, r3.) |
| V19 | `grep -n "AILANG_DRIVER_REF" tools/launchd/lib/pin-root.sh`; negative/positive control: `grep -c "AILANG_DRIVER_AGE" tools/launchd/lib/pin-root.sh` | Controller measurement verbatim: `34: # AILANG_DRIVER_REF ref to pin to (default origin/dev)`; `150: PIN_NOTE="running committed ${AILANG_DRIVER_REF:-origin/dev} @ …"`; `161: ref="${AILANG_DRIVER_REF:-origin/dev}"`; `322: AILANG_DRIVER_REF="$ref"`; `323: export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR`. Negative/positive control: AGE grep → `0` (the not-yet-existing variable), so the grep discriminates. Local rerun confirms these sites and additionally matches the REF refresh comment at :191. |

## Related Documents

- [V1 mission charter](../v1-mission.md): queue row and D-61 [V10].
- [Reference design](../implemented/v0_38_6/m-debugcacheforms-flaky-on-macos-ci.md): structure and mutation discipline [V17].
- [CLAUDE.md](../../CLAUDE.md): operational guardrails and no silent fallbacks [V1].

## References

Implementation surfaces and exact evidence commands are in V2–V20 above.
No external service or web premise is necessary for this design.

## Future Work

Binary mission control may eventually replace this shell instrumentation; that is outside this
milestone and does not extend D-61's bridge authority.

## Quorum verification log

Round 1 (2026-09-14, reviewers gpt5-6-sol/gemini-3-1-pro/oc-glm-5-2, astra recused as author): BLOCKED —
sol REJECT (mutable origin/dev in the measurement; premise verified REAL, fix applied verbatim → V18,
AC-P, MUT-U); glm REJECT (AILANG_DRIVER_REF export unverified; premise measured FALSE at pin-root.sh:322-323,
recorded as V19; requested arm added anyway → AC-Q, MUT-V); gemini PASS.
Round 2 (2026-09-14, same reviewers): BLOCKED 3/3, zero absentees — sol REJECT (the r2 test hook
`AILANG_TEST_PIN_AGE_AFTER_BASELINE_HOOK` is an unbounded env-controlled executable in production;
fix: delete it, implement AC-P with a test-local `git` shim); glm REJECT (same hook, same remedy —
lab-only interception, no test-conditional branch in production); gemini REJECT (V20 row for
`pin_root_to_committed_ref` / `_mc_notify` existence). Every fix concrete and reviewer-authored, none
disputing the direction → NARROW-REFINEMENT CARVE-OUT (r3): controller applied all three verbatim
(hook deleted; AC-P re-specified as the PATH shim; freeze/risks/conflict rows updated; V20 added).
Round 3: not run — carve-out, per the mission-control Gate-2 rule (ratified iter-95).

Last updated: 2026-09-14 — revision r3 (controller-applied verbatim reviewer fixes after quorum round 2, narrow-refinement carve-out)
