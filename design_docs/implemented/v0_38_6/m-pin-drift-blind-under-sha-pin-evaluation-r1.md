# Sprint evaluation — v1_iter354_pin_age (m-pin-drift-blind-under-sha-pin)

Reviewed commit: b2bb2f2f8 (HEAD, detached, worktree `.wt-v1-iter354-eval`). Base: fd44be9ee.

## Verdict

**PASS 96/100** — blocking findings: 0

## Gates observed

| Command | rc | Totals (observed) |
|---|---|---|
| `/bin/bash tools/launchd/test_pin_root.sh` (first run) | 0 | `==== 117 passed, 0 failed ====` [.eval/pin_root_head.log] |
| `/bin/bash tools/launchd/test_driver_notify.sh` (first run) | 0 | `==== 81 passed, 0 failed ====` [.eval/notify_full.log] |
| `/bin/bash -n` on all 4 touched files | 0 each | pin-root.sh, mission-control.sh, test_pin_root.sh, test_driver_notify.sh — all rc=0 |
| `shellcheck -S warning tools/launchd/lib/pin-root.sh` | 1 (warnings only) | 3 warnings (SC2034×2, SC1090×1) — **all 3 present verbatim at base `fd44be9ee`** too (same lines, same rule ids); zero new warnings |
| `shellcheck -S warning tools/launchd/mission-control.sh` | 1 (warnings only) | SC2034×2 + SC2163×6 + SC1090×2 — identical count/kind at base and HEAD (`grep -c` diff = 0 lines changed); zero new warnings |
| Final re-confirmation run, `test_pin_root.sh` | 0 | `==== 117 passed, 0 failed ====` [.eval/final_pin.log] |
| Final re-confirmation run, `test_driver_notify.sh` | 0 | `==== 81 passed, 0 failed ====` [.eval/final_notify.log] |
| `make test-launchd-drivers` | UNMEASURED — not run (controller running it per the task brief; time budget spent on the mutation re-drill and AC verification instead; the two constituent suites were run and confirmed green twice each) |

Plan-contract check: pin suite is 117 (plan said "116"; the mutation-audit itself records this as a
controller sign-off under Lane Rule 8 — one assertion split into two numbered siblings,
`failed baseline resolution keeps AGE and baseline ?` / `(2)`). Notify suite is 81, matching plan
contract exactly.

## Lint / bash-3.2 portability (added lines only, `grep '^+'` on the diff)

| Pattern | Hits in added lines | Positive control |
|---|---|---|
| `declare -A` | 0 | n/a (obviously absent from shell-only diff) |
| `${var,,}` | 0 | n/a |
| `mapfile` / `readarray` | 0 | n/a |
| GNU `timeout ` | 0 | n/a |
| `eval ` (sanity check the grep mechanism works) | **2** [`run_both`'s `eval "local _d=…"` / `_a=…`] | confirms grep isn't silently matching nothing |

No portability violations. The `eval` calls in `run_both` are pre-existing style (positional-arg
indirection in bash 3.2, no arrays) and are test-only, not production.

## Acceptance table (AC-A .. AC-Q, 17/17 = 100%)

All PASS lines below are **OBSERVED** in `.eval/pin_root_head.log` / `.eval/notify_full.log` (not
inferred from source).

| AC | Suite arm | Observed |
|---|---|---|
| AC-A | case 1 | `age is exactly 0 at the origin/dev tip`, `baseline SHA is origin/dev's full commit`, `note carries the pinned-age clause` — all PASS |
| AC-B | §11 | `sha pin has exact line AGE=3`, `sha pin still has exact line DRIFT=0`, `sha pin note names age 3 and the baseline SHA` — PASS |
| AC-C | case 3 + D-2 witness | `carries age across the exec`, `carries the age baseline across the exec`, `missing age carries as ?, never zero` — PASS |
| AC-C2 | §12 | `real exec exports AILANG_DRIVER_AGE=9…`, `pinned pass reads back exact line AGE=9`, `exported baseline equals origin/dev's full commit` — PASS |
| AC-D | §13 | all 5 named PASS lines observed, incl. the consumer witness `extracted new-consumer age decision logs unknown for unset PIN_AGE` |
| AC-E | age-a | 4/4 PASS |
| AC-F | age-b | 2/2 PASS |
| AC-G | age-c/age-c2 | 2/2 PASS |
| AC-H | age-d/age-d-followup | 2/2 PASS |
| AC-I | age-f/age-f2 | 2/2 PASS |
| AC-J | age-i1..5 | 5/5 PASS |
| AC-K | age-j | 1/1 PASS |
| AC-L | age-independent-1..4 | 4/4 PASS |
| AC-M | age-m1..3 | 3/3 PASS |
| AC-N | §14 | 6/6 PASS (two-shim scheme) |
| AC-O | age-paths | 4/4 PASS |
| AC-P | §15 | 5/5 PASS |
| AC-Q | §16 (lab) + age-q (notice) | 4 + 2 = 6/6 PASS |

Ratio: **17/17 AC rows fully observed (100%)**, well above the 50% hard-fail floor.

Base-suite name preservation (rubric-prescribed check, adapted — see note): `test_pin_root.sh`
doesn't call `ok "literal name"` directly (it wraps via `check`/`checkeq`/`checkno`), so the
literal `grep -o 'ok "[^"]*"'` from the rubric only yields the 3 function-definition echoes
(`ok "$1"` ×3 — a degenerate result, not real names). I substituted the equivalent
`grep -oE '(check|checkeq|checkno|ok) "[^"]*"'` (excluding the `"$1"` template artifacts) to get
69 real base-suite assertion names, all of which are present in HEAD's observed output — 6 initially
reported "missing" were verified to be variable-interpolated names (`$flag`, `$shape`, `$origin`)
that appear fully substituted in the real PASS lines (e.g. `flag 0 overrides OPPOSITE inference`).
For `test_driver_notify.sh` the rubric's literal command worked as given: 51 raw names minus 2
template artifacts (`$1`, `$name`) = 49 real names, **all 49 present** in HEAD's observed output.
Drift arms (`drift-a` .. `drift-j`) and all pre-M2 arms are byte-identically named and still PASS.

## Mutation re-drill table

PRE hashes (both matched the audit record's own PRE hashes exactly, confirming the audited tree =
this worktree's HEAD):
`pin-root.sh` = `82293ba6…58966db`, `mission-control.sh` = `e5798a3e…0e7ed`.

7 named mutants re-drilled (≥2 from each named group, plus both audit-flagged "unusual" rows
MUT-F and MUT-R), protocol = apply → `bash -n` (build) → sha256 (landed) → run suite once →
record FAIL lines → `cp`-restore → sha256 match:

| MUT | File | Edit | Build rc | Landed | Suite rc | FAIL lines observed | Restore sha match | Result | vs. audit |
|---|---|---|---|---|---|---|---|---|---|
| MUT-A | pin-root.sh | `$target..$origin_dev_sha` → `HEAD..$origin_dev_sha` | 0 | yes | 1 | `age is exactly 0…`, `note carries the pinned-age clause`, `failed age rev-list yields AGE=?` (3, `114 passed, 3 failed`) | yes | **KILLED** | exact match |
| MUT-S | pin-root.sh | `age="?"` coercion → `age="0"` on failure | 0 | yes | 1 | `failed age rev-list yields AGE=?`, `…fabricates no zero`, `failed baseline resolution keeps AGE and baseline ?` (3, `114 passed, 3 failed`) | yes | **KILLED** | exact match |
| MUT-U | pin-root.sh | `$target..$origin_dev_sha` → `$target..origin/dev` (mutable ref) | 0 | yes | 1 | `reported age is B's count, not the moved ref's` (1, `116 passed, 1 failed`) | yes | **KILLED** | exact match |
| MUT-F | pin-root.sh | delete T5 pre-handoff warning `if` | 0 | yes | 1 (hard `exit 1`) | `fixture error: old-helper substitution 't5warn=0' matched 0 lines` — no PASS/FAIL totals line at all (script aborts) | yes | **KILLED** (by fixture-rot guard, NOT the named assertion) | exact match — confirms the audit's own honesty note that MUT-F's predicted kill line is wrong and the real killer is AC-D's regex-derivation guard |
| MUT-R | mission-control.sh | delete non-pinned-status guard (`if [ "$PIN_STATUS" != "pinned" ]` → `if false`) | 0 | yes | 1 | `age-m1: STALE skips age with no pin-age send, state untouched`, `age-m3: disabled skips age silently with state untouched` (2, `79 passed, 2 failed`) | yes | **KILLED** | exact match with the audit's *post-repair* re-drill (the row this audit itself flagged as a first-drill SURVIVOR, then repaired) |
| MUT-H | mission-control.sh | threshold `-lt` → `-le` | 0 | yes | 1 | 5 FAIL lines incl. all 4 age-a assertions + `age-d-followup…` (`76 passed, 5 failed`) | yes | **KILLED** | exact match (same total; audit's table cell text was truncated but total matches) |
| MUT-Q | mission-control.sh | age-file write → `$PIN_DRIFT_FILE` (state-isolation violation) | 0 | yes | 1 | 8 FAIL lines incl. all 4 `age-independent-*`, `age-a: age state stores 25`, `age-c: doubling…`, `age-d-followup…`, `age-q: baseline SHA…` (`73 passed, 8 failed`) | yes | **KILLED** | exact match |

All 7 re-drilled mutants: **KILLED**, and every rc/total/FAIL-line I observed matches the audit
record's own recorded numbers exactly — the audit is honest on this sample (7/22, spanning both
groups and both flagged-unusual rows).

## My own two diff-anchored mutants (not named in the design's tables)

1. **T4 "clear inherited AGE on unpinned invocation"** (`pin-root.sh`, the
   `unset AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA` line right after the already-pinned
   short-circuit). Reasoning: T6 unconditionally reassigns `AILANG_DRIVER_AGE="$age"` on every
   reachable fresh-exec path before export, so this unset looked behaviourally redundant and
   untested (no arm sets ambient `AILANG_DRIVER_AGE` before a **fresh**, non-already-pinned,
   **new-helper** invocation). Deleted it, `bash -n` rc=0, sha landed, ran `test_pin_root.sh`
   once: **KILLED** — not by a behavioural assertion, but by the *same* AC-D fixture-rot guard
   that kills MUT-F (`fixture error: old-helper substitution 't4clear=0' matched 0 lines`), because
   the AC-D fixture's Python derivation script regex-matches this exact comment+unset block to
   build the "old helper" fixture. Non-blocking, informational: the guard's incidental reach
   across nearly every T1-T7 textual hunk is a *side effect* of AC-D's implementation, not a
   deliberate mutation-coverage design for T4 — if the AC-D fixture script is ever refactored to
   stop deriving text-for-text from the live helper, T4 would lose its only kill mechanism.

2. **Notice body prose** (`mission-control.sh`, the two sentences
   `Every landed driver/skill fix newer than this pin is NOT in effect.` and
   `This notice repeats only when the measured pin age doubles.` inside `_pin_age_body`).
   Confirmed by `grep` that neither sentence's substring is asserted anywhere in
   `test_driver_notify.sh` (0 hits, with a positive control proving the grep mechanism works via
   `AILANG_DRIVER_REF` at 6 hits elsewhere). Inverted both sentences to their **opposite meaning**
   (`All landed driver/skill fixes still apply as normal.` / `This notice never repeats.`),
   `bash -n` rc=0, sha landed, ran `test_driver_notify.sh` once: **SURVIVED** —
   `==== 81 passed, 0 failed ====`, rc=0, zero FAIL lines. **This is a genuine, non-vacuous
   survivor.**

Both mutants restored via `cp`, sha256 confirmed identical to the recorded PRE hash in each case.

## Findings

1. **NON-BLOCKING — notice body's substantive warning sentences are unchecked (confirmed
   survivor).** File: `tools/launchd/mission-control.sh`, lines 1792 and 1794. The two sentences
   that tell a human operator *why the notice matters* (fixes not in effect; repeat cadence) can
   be silently deleted, garbled, or inverted and `test_driver_notify.sh` stays green
   (`81 passed, 0 failed`, confirmed by direct mutation above). Severity is low — the notice still
   fires with the correct count/ref/SHA (those fields ARE asserted), so no decision-relevant data
   is lost, only the explanatory prose. Fix: add one `case`/`check` asserting a substring of
   either sentence in the `age-a` arm (mirrors how `age-a` already asserts the "executing-old-code
   wording" substring for the first sentence's sibling in `_pin_age_body`'s first line — that
   line IS checked; these two trailing sentences are not).

2. **NON-BLOCKING — informational.** MUT-F and (per my own drill) the T4 unset are both "killed"
   via the AC-D fixture-rot guard rather than via any assertion that exercises the *actual runtime
   behaviour* the mutation would change. This matches the audit's own honesty note for MUT-F
   (recorded, not tuned) and I confirm it generalizes to at least one more hunk (T4). Not a defect
   in the shipped code — the guard is genuinely loud and non-vacuous (`exit 1`, suite aborts) — but
   it means several of the design's named "producer-side" mutation kills are actually
   text-derivation-fragility kills, not behavioural kills. No action required; noting for the
   record since the rubric asks for honest severity assessment of survivors and of unusual kills.

3. **NON-BLOCKING — audit record self-contradiction, resolved as benign.** The mutation-audit
   file's M2 table footer reads `Post-drill green re-run: rc=0 ==== 81 passed, 0 failed ====;
   sha256 == PRE: False` (design_docs/planned/…mutation-audit.md:87), immediately followed by a
   separate MUT-R re-drill entry recording a *matching* restore. Read in context this is not
   dishonesty: the `False` is the batch-level hash check taken **before** the controller's
   subsequent MUT-R row-repair-and-redrill (which the file itself documents next, with its own
   `restore sha match True`) — i.e. the audit is transparently showing an intermediate,
   not-yet-fully-restored state rather than hiding it. I did not find any suite run in my own
   redrill where a restore actually failed to match (all 7 of my drills + both diff-anchored
   mutants matched PRE byte-for-byte). Flagging only because the juxtaposition reads oddly on
   first pass and a future reader might mistake it for an unresolved integrity gap.

No blocking findings.

## Design-fidelity notes

- **Always-compute**: T6 runs unconditionally after drift computation, no ref-type branching —
  confirmed by reading the code (`tools/launchd/lib/pin-root.sh:240-256`) and by AC-A (age=0 at
  tip) / AC-Q (non-default ref still computes).
- **Separate state file**: `PIN_AGE_FILE` distinct from `PIN_DRIFT_FILE` for both mission branches,
  wrapped in new `# --- DRIVER PIN STATE PATHS START/END ---` anchors — confirmed via AC-O
  (4/4 PASS) and via my own MUT-Q drill (cross-writing to `PIN_DRIFT_FILE` is caught, 8 FAIL
  lines).
- **Unknown is log-only, never zero**: verified structurally (no `:-0` default for any AGE
  variable anywhere in the diff — `grep` positive/negative checked) and behaviourally via AC-N,
  AC-K, AC-I, AC-D, and my MUT-S drill.
- **No production test hook / no env-gated branch**: the r2 quorum-rejected hook name
  `AILANG_TEST_PIN_AGE_AFTER_BASELINE_HOOK` is verified absent repo-wide (0 hits, positive control
  via `AILANG_DRIVER_REF` confirms the grep isn't silently failing). AC-P's race is proven from
  OUTSIDE the helper via a PATH `git` shim, exactly as the design freeze requires.
  `_pin_age_body`/`_pin_age_degraded` gating has no `AILANG_TEST_*` or similar conditional.
- **No PIN_DRIFT semantics change**: drift's own byte-preserved assertions (`DRIFT=1` case 1,
  carry-7 case 3, `drift-a`..`drift-j`) all still PASS by name — verified above (base-name
  preservation section).
- **Drift notice wording untouched**: `_pin_drift_body`/`_pin_drift_degraded` block is
  textually unchanged in the diff (only a new sibling `if` block is appended after its `fi`) —
  confirmed by reading the diff hunk at `tools/launchd/mission-control.sh:1783-1784` (context
  lines only, no `+`/`-` inside the drift block itself).
- **PIN_AGE_SUPPORTED handshake across the gate refresh**: reasoned through the refresh hop
  (`tools/launchd/lib/pin-root.sh:222-236`) — the marker is unset and `PIN_AGE` reset to `"?"`
  immediately before `. "$refreshed_helper"`; an OLD target helper's blob (built by AC-D's checked
  deletions) never re-declares `PIN_AGE_SUPPORTED=1`, so after sourcing the check
  `${PIN_AGE_SUPPORTED:-}" != "1"` is true and the outer helper — which IS the new one — prints the
  loud diagnostic before recursing. Verified behaviourally: AC-D's 5 PASS lines, incl. the
  consumer-side witness.
- **Two mission-branch state paths**: both `PIN_AGE_FILE` assignments (v1 legacy path,
  `mission-${MISSION_NAME}` else-branch) present and distinct from `PIN_DRIFT_FILE` — AC-O.

## Score breakdown

| Rubric row | Points available | Awarded | Basis |
|---|---|---|---|
| Tests pass (HARD FAIL gate) | 20 | 20 | Both suites rc=0 with correct totals, run twice each; all 4 touched files `bash -n` clean; shellcheck warnings identical at base and HEAD (zero new) |
| Lint clean (bash 3.2 portability) | 10 | 10 | Zero forbidden constructs in added lines, positive control confirms grep validity |
| Acceptance criteria (HARD FAIL if <50%) | 30 | 29 | 17/17 AC rows observed (100%); −1 for the notice-body-prose gap surfaced by my own mutant (a real, if narrow, acceptance gap on the notice's prose fields, which the design freeze explicitly named as required content) |
| Code quality | 15 | 14 | All named invariants (immutable baseline, exec transport, readback, handshake, `set -u` safety, identity fields, state paths) verified correct by direct reading + behavioural drill; −1 for the T4/MUT-F class of kills being fixture-fragility rather than behavioural (informational, not a code defect, but it means test *design* over-relies on one guard for several distinct hunks) |
| Documentation | 15 | 14 | Changelog entry present, byte-preserves the prior entry, accurately describes the mechanism; mutation-audit record has 22/22 rows with the required fields and is honest on every sampled row (7/22 re-verified byte-for-byte); −1 for the odd (though ultimately benign) `sha256 == PRE: False` line in the audit's own M2 footer that a future reader could misread as an integrity failure |
| Design fidelity | 10 | 10 | All Design Freeze bullets verified: always-compute, separate state, unknown log-only, no test hook/env-gated branch, PIN_DRIFT semantics/notice untouched |
| Non-vacuity (2 own diff-anchored mutants) | (gate, folded into AC/quality above) | — | 1 KILLED (via fixture-rot side effect), 1 SURVIVED (notice-body prose) — both reported honestly above |
| **Total** | **100** | **96** | |

PASS ≥70 with no hard fail: tests pass (20/20), AC ratio 100% (≥50% floor). **PASS 96/100.**
