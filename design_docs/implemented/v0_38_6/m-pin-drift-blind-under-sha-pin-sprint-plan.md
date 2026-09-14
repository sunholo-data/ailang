# Sprint Plan: M-PIN-DRIFT-BLIND-UNDER-SHA-PIN — PIN_AGE, a stale deployment pin reports its distance from origin/dev

**Sprint ID:** `v1_iter354_pin_age`
**Design doc (approved):** [m-pin-drift-blind-under-sha-pin.md](m-pin-drift-blind-under-sha-pin.md) (r3 — quorum rounds 1/2 BLOCKED, all reviewer fixes applied verbatim under the narrow-refinement carve-out; controller-committed as `f4c3dfd11`)
**Target:** v0.38.6 · **Mission:** V1, iteration 354
**Duration:** 1 day (hard ceiling; design-doc estimate 0.5 day — planned as ~8 hours across two atomic-commit milestones)
**Dependencies:** the design-doc commit `f4c3dfd11` (this worktree's HEAD, one above base `9422ff628`); the executor must run on a bash 3.2.57-compatible shell; the controller owns every git write.
**Risk Level:** Low (shell-only, strictly additive; both production touch points sit beside exercised, extraction-tested blocks with byte-exact anchor preservation requirements)

## Summary

Teach the driver-pin instrument a second number. Today `PIN_DRIFT` answers "how far is the
*source clone* behind the ref" and is **0** for any pin reachable from source HEAD — even a pin
266 commits old (measured at V7 in the design doc). `PIN_AGE` answers the orthogonal question:
"how many commits on fetched `origin/dev` are NOT reachable from the pinned target" —
`git -C "$src" rev-list --count "$target..$origin_dev_sha"`, computed once against an immutable
full `origin_dev_sha` captured immediately after the gate refresh. The reading crosses the
re-exec as `AILANG_DRIVER_AGE` / `AILANG_DRIVER_AGE_BASE_SHA`, is read back on the pinned pass,
appears in `PIN_NOTE`, and drives an *independent* threshold/dedupe/notice chain in
mission-control.sh (own state file, own dedupe-until-doubling, own `_mc_notify` type `pin-age`,
own top-level notice block). Old pinned targets that predate the instrument get a loud
`PIN_AGE_SUPPORTED` handshake diagnostic on stderr — never a fabricated zero.

The sprint splits the design's single H1–H4 implementation block into **two atomic commits the
controller can make**: M1 = the complete instrument (H1 producer/transport + H2 consumer) plus
the pin-lab proof arms (AC-A..AC-D, AC-C2, AC-N, AC-P, AC-Q-lab) and the MUT-A..G/S/U drill;
M2 = the notifier-harness arms (AC-E..AC-M, AC-O, AC-Q-notice) + H4 changelog + the mutation
audit record. M1 is self-green against the *unchanged* notify suite (50/0), because every age
code path normalizes unknown to log-only before touching anything — see D-3 below.

## Current Status Analysis

### Velocity

- Design-doc estimate: 0.5 day, one milestone (~4 h). This plan splits it into **two atomic
  commits** (M1 = instrument + pin-lab proof; M2 = notify proof + prose + evidence) because the
  mission requires exactly two milestones and the controller commits per milestone. The design's
  "one implementation milestone" concern — a split leaving an *incomplete instrument* — is
  honoured: the whole producer/transport/consumer contract lands in the M1 commit.
- **Estimated total LOC: ~+555 / −5 across the sprint** —
  `tools/launchd/lib/pin-root.sh` ~+33/−1 · `tools/launchd/mission-control.sh` ~+61/0 ·
  `tools/launchd/test_pin_root.sh` ~+175/−4 (heredoc + in-place case-1/case-3 assertions + five
  new sections) · `tools/launchd/test_driver_notify.sh` ~+195/0 (two extractions, `run_age` /
  `run_both`, eleven arm sections) · `changelogs/v0.32-current.md` ~+10/0 ·
  `design_docs/planned/m-pin-drift-blind-under-sha-pin-mutation-audit.md` ~+90 (new).
- Expected assertion totals after the sprint (planner-specified, **binding** — see Lane Rule 8):
  pin suite **116 passed, 0 failed** (81 + 35 new), notify suite **81 passed, 0 failed**
  (50 + 31 new).

### Planner re-verification at this worktree (HEAD `f4c3dfd11`)

The doc's Verification Log was measured at base `9422ff628`; HEAD here is the design-doc commit
one above it. Every load-bearing row re-measured; the two suites were **re-run** (they build
disposable git repos under `mktemp`, which is permitted and expected). No drift that changes any
task. Commands were run read-only in this worktree.

| # | Check | Doc says | Measured here (command → output) | Verdict |
|---|-------|----------|----------------------------------|---------|
| P1 | Drift computation line | `drift=$(git -C "$src" rev-list --count "HEAD..$ref" ...)` in pin-root.sh (V2) | `sed -n '219p' tools/launchd/lib/pin-root.sh` → exactly that line at **:219**; `[ -n "$drift" ] \|\| drift="?"` at :220; `PIN_DRIFT="$drift"` at :221 | ✓ matches |
| P2 | Export list | `AILANG_DRIVER_PINNED … MISSION_WORKDIR` at 322-323 (V19) | `sed -n '319,325p'` → assignments :319-322 (`AILANG_DRIVER_REF="$ref"` at :322), `export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR` at **:323**, `exec /bin/bash "$wt/tools/launchd/$script" "$@"` at :325 | ✓ matches |
| P3 | Two PIN_DRIFT_FILE assignments | V1 + other-mission branches (V4) | `grep -n 'PIN_DRIFT_FILE=' tools/launchd/mission-control.sh` → **:100** `"$STATE_DIR/mission-control.pin-drift"` (v1 branch), **:111** `"$STATE_DIR/mission-${MISSION_NAME}.pin-drift"` (else) | ✓ matches |
| P4 | Decision anchors, exact text | `# --- DRIVER PIN DECISION START/END ---` (V3) | `grep -n` → START at **:965**, END at **:1026**; `grep -c` of each exact line = **1** (unique, awk-extractable) | ✓ matches |
| P5 | Drift notice block first/last lines | top-level `if [ -n "$_pin_drift_degraded" ]` … `fi` (V3/V5) | `sed -n '1720,1733p'` → first line **:1720** `if [ -n "$_pin_drift_degraded" ]; then`, last line **:1733** `fi`; `grep -c '^if \[ -n "\$_pin_drift_degraded" \]; then$'` = **1** | ✓ matches |
| P6 | `_mc_notify` signature | `title body label` (V20) | `sed -n '179,181p'` → `:179 _mc_notify() {`, `:180 local title="$1" body="$2" label="$3" …`; send at :200-201 `ailang messages send controlplane "$body" --title "$title" --from "$MSG_FROM"`; gh at :226-227; notice types in use: `driver-pin`, `pin-drift`, `lane-degradation` | ✓ matches |
| P7 | Pin-lab case 1 / case 3 assertions | case 1 `DRIFT=1`, case 3 carry 7 (V12) | `sed -n '107p'` → `check "drift measured as 1" "$OUT" "DRIFT=1"`; `sed -n '127,129p'` → `AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_DRIFT=7 …` asserting `DRIFT=7` | ✓ matches |
| P8 | Notify awk extraction lines | extraction, never retyping (V5) | `sed -n '69,77p' tools/launchd/test_driver_notify.sh` → `:69` _mc_notify, **:70** pin_decision (`START`..`END`), :71 pin_block, **:72** pin_drift_block (`/^if \[ -n "\$_pin_drift_degraded" \]; then/,/^fi$/`), :73 lane, :76-77 bounded/drain; non-empty FATAL guard :81-83 | ✓ matches |
| P9 | `run_drift` harness shape | status/drift/threshold/state harness (V5) | `sed -n '138,175p'` → exactly as doc describes: per-arm `$LAB/state.$$.$RUN_SEQ`, stub env, `DECISION_RC` echo at :163, `STATE:` echo :167-171; arms at :323-376 incl. drift-a..j | ✓ matches |
| P10 | Changelog `## [Unreleased]` head | Unreleased followed by one Fixed entry before v0.38.5 (V6) | `grep -n '^## \[Unreleased\]' changelogs/v0.32-current.md` → **:5**; `sed -n '5,13p'` → exactly one entry, `### Fixed — the stderr-capture helper closed the pipe read end before the copier drained it` (:7-12), then `## [v0.38.5]` at :14 | ✓ matches |
| P11 | Suite baselines | 81/0 pin, 50/0 notify (controller at 9422ff628) | `/bin/bash tools/launchd/test_pin_root.sh` → `==== 81 passed, 0 failed ====`, **rc=0** · `/bin/bash tools/launchd/test_driver_notify.sh` → `==== 50 passed, 0 failed ====`, **rc=0** — both re-run at `f4c3dfd11` | ✓ re-measured green |
| P12 | Shell | bash 3.2 (V6) | `/bin/bash --version` → `GNU bash, version 3.2.57(1)-release (arm64-apple-darwin25)` | ✓ matches |
| P13 | Age-instrument absence | 0 PIN_AGE lines vs 32 PIN_DRIFT (V8) | `rg -c 'PIN_AGE' tools/launchd/` → no matches (rc=1); `rg 'PIN_DRIFT' tools/launchd/ \| wc -l` → `32` | ✓ matches |
| P14 | Gate-refresh bootstrap (the doc's gotcha) | helper re-sources itself from the TARGET ref before measuring, ~lines 187-214 | measured block: comment :185-193, `if [ "${AILANG_DRIVER_PIN_GATE_REFRESHED:-}" != "$target" ]` at **:194**, `git show "$target:tools/launchd/lib/pin-root.sh"` at **:198**, `/bin/bash -n` gate :202, marker set+export :206-207, `. "$refreshed_helper"` at **:208**, recurse `pin_root_to_committed_ref "$@"` at **:214**, `return $?` :215, `fi` **:216** | ✓ semantics match; line numbers refreshed → D-1 |

**What the bootstrap gotcha means (P14), applied to this sprint:**

- On a fresh (unpinned) fire the code that *computes* age runs from the **target blob** loaded by
  `git show` at :198 and sourced at :208 — the outer working-tree helper hands off at :214. So
  every lab arm that pins a SHA reads the helper *from that SHA's tree*. The pin suite seeds
  `cp "$SRC_HELPER" tools/launchd/lib/pin-root.sh` at test_pin_root.sh**:46** and commits it as
  the base commit at **:70** — once H1 lands, the lab's first commit **A carries the new helper
  automatically**. The plan's SHA-ref arms (AC-B, AC-C2, AC-N, AC-P) therefore need no extra
  seeding; AC-D's whole point is to construct the one target whose blob lacks it (oldhelper
  branch, built by *checked deletions* from that same seeded helper).
- For AC-D: the outer run stays on the clone working tree, whose helper is the new one (the
  suite's only mutation of it is the perl onboarding-predicate edit at :79, which never touches
  age code), so the pre-handoff `PIN_AGE_SUPPORTED` warning fires from the *new outer* when the
  sourced old blob doesn't redeclare the marker. `AILANG_DRIVER_PIN_GATE_REFRESHED` is already in
  the suite's unset list (:28), so each arm gets a genuine refresh hop.
- Marker hygiene is load-bearing: the new helper declares `PIN_AGE_SUPPORTED=1` at source time,
  and must `unset` it (plus reset `PIN_AGE="?"`) **immediately before** `. "$refreshed_helper"`,
  or the outer's own marker would leak past the hop and silence the AC-D warning.

**Measured disagreements (doc text vs. repo truth; resolution stated, ruling in favour of the CODE):**

- **D-1 — gate-refresh line range.** The doc's briefing text says "~lines 187-214"; measured
  194-216 (P14). Trivially stale line numbers only, semantics exact. *Resolution:* this plan's
  task anchors use the measured numbers. No code disagreement.
- **D-2 — MUT-G has no producer-side killer as the AC table stands.** MUT-G is "default missing
  age to zero", owned by H1 (pin-root.sh). Its only mapped arm, AC-D, execs an *old* fixture
  driver whose pinned pass prints no AGE at all, and the AC-D consumer witness reads
  mission-control.sh (unmutated by an H1 drill) — so a pin-root.sh pinned-pass mutation
  `${AILANG_DRIVER_AGE:-?}` → `${AILANG_DRIVER_AGE:-0}` would go **green** everywhere: a known
  survivor. The doc's drill discipline says report survivors, never tune fixtures. *Resolution
  (planner-added test assertion, not a new mutation, not expanded scope):* case 3 gains one
  sibling run with `AILANG_DRIVER_PINNED=deadbee` and **no AGE/BASE_SHA in env**, asserting the
  readback stays `?` — PASS name `missing age carries as ?, never zero`. This is the
  producer-side mirror of AC-K's consumer-side witness for MUT-P and gives doc-named MUT-G its
  structural kill. It is **flagged here explicitly** per the mission's no-silent-invention rule;
  pin-suite total reflects it (116 = 81 + 35 including this one).
- **D-3 — the doc designs one milestone; the mission mandates two.** Doc Implementation Plan:
  "M1 — producer, consumer, regression proof, changelog"; doc High-Impact Decisions: "One
  implementation milestone". *Resolution:* two commits, but the *instrument* (H1+H2) is atomic in
  M1 — M2 contains only additional test arms, prose, and evidence. M1's tree stays green against
  the **unchanged** notify suite (verified reasoning: the extracted pin_decision block
  normalizes an unset `PIN_AGE` to `?` and logs `driver pin age: unknown (?)` before touching
  any file or numeric test, so the 50 existing assertions survive; the age notice block sits in
  a sibling top-level `if`/`fi` that the unchanged harness never sources). M1 gate therefore
  expects notify **50/0**, M2 expects **81/0**.

## Proposed Milestones

### M1: PIN_AGE instrument (H1 producer/transport + H2 consumer) + pin-lab proof arms + MUT drill (~4.5 h, ~+270 LOC)

**Goal:** Land the complete, self-consistent instrument in one commit: helper computation against
an immutable captured `origin/dev` SHA, exec transport, pinned-pass readback, `PIN_NOTE` clause,
header contract, `PIN_AGE_SUPPORTED` handshake with pre-handoff stderr diagnostic, inherited-value
clearing; mission-control state paths, independent age decision, sibling `pin-age` notice — plus
the seven pin-lab arm groups (AC-A, AC-B, AC-C, AC-C2, AC-D, AC-N, AC-P, AC-Q-lab-half) that
prove producer and transport, and the nine-mutant drill MUT-A..G, S, U.

**Estimated:** pin-root.sh ~+33/−1 · mission-control.sh ~+61/0 · test_pin_root.sh ~+175/−4.
**Dependencies:** none.

**Tasks (file:line anchors verified in P1-P14 above):**

- [ ] **T1 — header contract** (`tools/launchd/lib/pin-root.sh:40-43`, the `out` block): add
  `PIN_AGE` (commits on fetched origin/dev not reachable from the pinned target; `"?"` unknown)
  and `PIN_AGE_BASE_SHA` rows, naming `AILANG_DRIVER_AGE` / `AILANG_DRIVER_AGE_BASE_SHA` as the
  exec transport; keep the PIN_DRIFT row's source-clone meaning verbatim.
- [ ] **T2 — source-time init + marker** (after :54 `PIN_DRIFT="?"`): `PIN_AGE="?"`,
  `PIN_AGE_BASE_SHA="?"`, and `PIN_AGE_SUPPORTED=1` (capability handshake, declared at source
  time, never exported, never read from env).
- [ ] **T3 — pinned-pass readback + note** (:146-152): before the PIN_NOTE line (:150) add
  `PIN_AGE="${AILANG_DRIVER_AGE:-?}"` and `PIN_AGE_BASE_SHA="${AILANG_DRIVER_AGE_BASE_SHA:-?}"`;
  append `; pinned target ${PIN_AGE} behind origin/dev (baseline ${PIN_AGE_BASE_SHA})` to
  PIN_NOTE, preserving the existing `(source clone … was N behind)` clause. *(MUT-D, MUT-G site.)*
- [ ] **T4 — clear inherited readings on an unpinned invocation** (immediately after the
  pinned short-circuit's `fi` at :152, before the disabled branch): `unset AILANG_DRIVER_AGE
  AILANG_DRIVER_AGE_BASE_SHA` — ambient values must not masquerade as fresh; the already-pinned
  branch above must NOT clear them (it consumes the transport). Add `age origin_dev_sha` to the
  `local` list at :160.
- [ ] **T5 — refresh-hop handshake** (inside :194-216): immediately before `. "$refreshed_helper"`
  (:208) insert `unset PIN_AGE_SUPPORTED` and `PIN_AGE="?"`; after the source rc-check (:211-213)
  and before the recurse (:214):
  `if [ "${PIN_AGE_SUPPORTED:-}" != "1" ]; then printf '%s\n' "driver pin age: unknown (?); helper at $ref @ $target lacks PIN_AGE; notice suppressed" >&2; fi`.
  Exactly one warning, via `printf`, no `log` dependency. *(MUT-F site.)*
- [ ] **T6 — age computation** (immediately after `PIN_DRIFT="$drift"` at :221): the design's
  freeze-verbatim snippet —
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
  PIN_AGE="$age"; PIN_AGE_BASE_SHA="$origin_dev_sha"
  AILANG_DRIVER_AGE="$age"; AILANG_DRIVER_AGE_BASE_SHA="$origin_dev_sha"
  ```
  No production fetch is added; no test-conditional branch. *(MUT-A/B/C/S/U sites.)*
- [ ] **T7 — export list** (:323): extend to
  `export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA`. *(MUT-E site.)*
- [ ] **T8 — state paths + anchors** (`tools/launchd/mission-control.sh`): wrap the existing
  path-selection `if` (:84) … `fi` (:113) in NEW full-line comments
  `# --- DRIVER PIN STATE PATHS START ---` (before :84) and `# --- DRIVER PIN STATE PATHS END ---`
  (after :113); add `PIN_AGE_FILE="$STATE_DIR/mission-control.pin-age"` after :100 and
  `PIN_AGE_FILE="$STATE_DIR/mission-${MISSION_NAME}.pin-age"` after :111. *(MUT-T site.)*
- [ ] **T9 — age decision** (inside the existing START(:965)/END(:1026) region, appended before
  :1026, anchors preserved byte-exact): a new block bounded by
  `# --- DRIVER PIN AGE DECISION START ---` / `# --- DRIVER PIN AGE DECISION END ---`;
  unconditional `_pin_age_degraded=""`; `PIN_AGE="${PIN_AGE:-?}"`; then the design's decision
  table with age-variables only (`_pin_age_warn`, `_pin_age_previous`, `_pin_age_emit`; never
  drift's): unknown/malformed → log `driver pin age: unknown ($PIN_AGE); notice suppressed`, no
  file access; numeric below warn → `rm -f "$PIN_AGE_FILE"` + log count/threshold/`notice
  re-armed`; at/above with absent/malformed previous or ≥ 2× previous → `_pin_age_degraded="$PIN_AGE"`,
  persist `printf '%s\n' "$PIN_AGE" > "$PIN_AGE_FILE"`, log armed; else log `… deduped until
  doubling from <previous>`; non-pinned status → log `driver pin age: skipped (status=$PIN_STATUS)`.
  `_pin_age_warn="${AILANG_DRIVER_AGE_WARN:-25}"`; explicitly-set invalid values
  (`''|*[!0-9]*|0`) log `driver pin age: AILANG_DRIVER_AGE_WARN='…' is not a positive integer;
  using 25` and floor to 25 (positive overrides below 25 stay valid — non-positive floor, not a
  minimum). *(MUT-H/J/K/L/M/N/O/P/Q/R sites.)*
- [ ] **T10 — age notice** (new sibling top-level block immediately after the drift notice's
  `fi` at :1733, drift block byte-preserved): `if [ -n "$_pin_age_degraded" ]; then … fi` building
  `_pin_age_body` containing — verbatim sentences from the design — `The driver is executing code
  N commits behind origin/dev.`, `Pinned ref: REF. Target SHA: SHA.`, `Baseline origin/dev SHA:
  <sha>`, `Every landed driver/skill fix newer than this pin is NOT in effect.`, `This notice
  repeats only when the measured pin age doubles.`; identity from `${AILANG_DRIVER_REF:-origin/dev}`
  and `${AILANG_DRIVER_PINNED:-?}` (the transported short SHA is the *target* SHA — never source
  HEAD); baseline from `${AILANG_DRIVER_AGE_BASE_SHA:-?}`, never re-resolved here; send
  `_mc_notify "Mission ${MISSION_NAME}: driver pin is stale (${_pin_age_degraded} behind)"
  "$_pin_age_body" "pin-age"`. *(MUT-I/R/V sites; MUT-R's guard is T9's status case.)*
- [ ] **T11 — pin-lab arms** (`tools/launchd/test_pin_root.sh`):
  (a) fake-driver heredoc (:47-67): add `echo "AGE=$PIN_AGE"`, `echo "AGE_BASE_SHA=$PIN_AGE_BASE_SHA"`,
  `echo "ENV_AGE=${AILANG_DRIVER_AGE:-unset}"`, `echo "ENV_AGE_BASE_SHA=${AILANG_DRIVER_AGE_BASE_SHA:-unset}"`,
  `echo "ENV_REF=${AILANG_DRIVER_REF:-unset}"`;
  (b) unset list (:27-29): add `AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA` and unset
  `PIN_AGE_SUPPORTED` for hygiene;
  (c) record `A=$(git -C "$T/seed" rev-parse HEAD)` immediately after the base commit (:70);
  (d) **AC-A** in case 1 (assertions listed below); (e) **AC-C** in case 3 plus the planner-added
  missing-age witness run (D-2);
  (f) after the existing content (the foreign-workrepo block ends :326; insert before the totals
  at :328) add sections `== 11`..`== 16`: restore the onboarding fixture (`printf
  '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/clone" > "$HOME/.claude.json"`, the
  :314 pattern) and confirm the remote URL (:138 already restored it);
  `§11 AC-B pin-age-sha` — advance seed dev until `rev-list --count $A..origin/dev` is exactly 3
  (assert the count BEFORE invoking; two empty commits + push), run `AILANG_DRIVER_REF="$A"`;
  `§12 AC-C2 pin-age-export` — advance to exactly 9, `unset AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA`,
  run `AILANG_DRIVER_REF="$A"` and compare the exported baseline against the seed's
  `rev-parse dev^{commit}`;
  `§13 AC-D pin-age-old-helper` — build the oldhelper branch fixture (below), run
  `AILANG_DRIVER_REF=origin/oldhelper`, then re-run with inherited `AILANG_DRIVER_AGE=99
  AILANG_DRIVER_AGE_BASE_SHA=bogus`;
  `§14 AC-N age-measure-fails` — two PATH-shim sub-arms (below);
  `§15 AC-P pin-age-baseline-moves` — the moving-baseline shim (below);
  `§16 AC-Q pin-age-nondefault-ref (lab half)` — `git -C "$T/seed" push origin "$A":refs/heads/feature`,
  advance dev to exactly 26 ahead of A, run `AILANG_DRIVER_REF=origin/feature`.
  **Every AGE/DRIFT value assertion is line-anchored** (`printf '%s\n' "$OUT" | grep -q '^AGE=3$'`
  style) — never the suite's substring `check`, because `ENV_AGE=<n>` lines alias the substring
  `AGE=<n>` (false green for MUT-D; see Lane Rule 9).
  **AC-D fixture, per the design:** generate the old helper by *checked* transformations of the
  seeded helper (delete the `PIN_AGE_SUPPORTED=1` line, the whole T6 computation block, the T3
  readback line, the AGE/BASE_SHA init lines, the AGE names from the export line, and the note
  clause; generate the old fake-driver by deleting the five new echo lines). Assert each
  transformation matched ≥1 line (grep -c before/after), `/bin/bash -n` rc=0 on both, `grep -c
  'PIN_AGE'` = 0 and `grep -c 'AILANG_DRIVER_AGE'` = 0 in the helper, and the drift line `rev-list
  --count "HEAD..$ref"` still present; commit on branch `oldhelper` from A, push to the lab origin.
  Never rely on an external historical SHA. The AC-D consumer witness is one extra awk extraction
  in this suite: `awk '/^# --- DRIVER PIN AGE DECISION START ---/,/^# --- DRIVER PIN AGE DECISION
  END ---/' "$REPO_ROOT/tools/launchd/mission-control.sh"` (fail if empty), sourced in a
  subshell with `log(){ echo "LOG:$*"; }`, `PIN_STATUS=pinned`, a per-arm `STATE_DIR`, `PIN_AGE`
  **unset**, asserting rc=0 and `driver pin age: unknown`.
  **Shims, per the design (AC-N/AC-P):** a test-local `git` placed FIRST in PATH for that arm
  only, delegating every invocation to the real binary resolved ONCE at shim creation
  (`REAL_GIT=$(command -v git)` baked in literally — never PATH recursion). AC-N shim-1 fails
  (exit 1) only `rev-list --count` calls whose range does NOT begin `HEAD..` (the age shape;
  drift's `HEAD..$ref` is the positive control); AC-N shim-2 fails only `rev-parse --verify
  --quiet origin/dev^{commit}`; AC-P shim, on that same rev-parse call, captures the true answer
  B, then advances the seed by one commit, pushes, fetches into the source clone, returns B, and
  bumps a lab counter file. Shims are removed from PATH after each arm.
- [ ] **T12 — M1 gates + drill:** run the M1 gate battery (below), then the nine-mutant drill
  (MUT-A/B/C/D/E/F/G/S/U) per the mutation table and Lane Rule 6, evidence to `.snap/drill/M1/`,
  restore-verify each, final re-run green, snapshot `.snap/M1/` (Lane Rule 5).

**M1 acceptance tests (expected output lines; each names the mutation it kills):**

Gate G0 (preflight, before edits): `/bin/bash -n` on the three touched files → rc=0 each; both
suites re-run at their baselines (pin `==== 81 passed, 0 failed ====`, notify `==== 50 passed, 0
failed ====`); a red baseline means the sandbox is broken — stop and report.

Gate G1 (fixed-form green, after edits):

- `/bin/bash tools/launchd/test_pin_root.sh` → rc=0 AND `==== 116 passed, 0 failed ====`, with
  these exact new PASS lines present (35 = 81→116):
  - case 1 (AC-A; kills **MUT-A**): `PASS: age is exactly 0 at the origin/dev tip` · `PASS:
    baseline SHA is origin/dev's full commit` · `PASS: note carries the pinned-age clause`
  - case 3 (AC-C; kills **MUT-D**, and **MUT-G** via the D-2 witness): `PASS: carries age across
    the exec` · `PASS: carries the age baseline across the exec` · `PASS: missing age carries as
    ?, never zero`
  - §11 (AC-B; kills **MUT-B**, **MUT-C**): `PASS: lab control: origin/dev is exactly 3 ahead of
    A` · `PASS: sha pin reports pinned` · `PASS: sha pin has exact line AGE=3` · `PASS: sha pin
    still has exact line DRIFT=0` · `PASS: sha pin note names age 3 and the baseline SHA`
  - §12 (AC-C2; kills **MUT-E**; AGE env unset before the run, real re-exec observed): `PASS:
    lab control: origin/dev is exactly 9 ahead of A` · `PASS: real exec exports
    AILANG_DRIVER_AGE=9 to the driver` · `PASS: pinned pass reads back exact line AGE=9` ·
    `PASS: exported baseline equals origin/dev's full commit`
  - §13 (AC-D; kills **MUT-F**): `PASS: old-helper pin still reports pinned` · `PASS:
    pre-handoff compatibility warning fires` · `PASS: no fabricated AGE=0 under an old helper` ·
    `PASS: extracted new-consumer age decision logs unknown for unset PIN_AGE` · `PASS:
    inherited bogus AGE is not carried into the target`
  - §14 (AC-N; kills **MUT-S**): `PASS: failed age rev-list yields AGE=?` · `PASS: failed age
    rev-list fabricates no zero` · `PASS: pin still reports pinned when age fails` · `PASS:
    drift control still succeeds during age failure` · `PASS: failed baseline resolution keeps
    AGE and baseline ?` · `PASS: failed baseline resolution is loud on stderr`
  - §15 (AC-P; kills **MUT-U**): `PASS: baseline shim fired exactly once` · `PASS: moved
    origin/dev differs from captured baseline B` · `PASS: reported age is B's count, not the
    moved ref's` · `PASS: exported baseline SHA equals captured B` · `PASS: note baseline equals
    captured B`
  - §16 (AC-Q lab half; feeds the **MUT-V** killer inputs): `PASS: lab control: origin/feature
    is exactly 26 behind origin/dev` · `PASS: feature-ref pin reports pinned` · `PASS:
    feature-ref pin has exact line AGE=26` · `PASS: driver sees exported
    AILANG_DRIVER_REF=origin/feature`
- `/bin/bash tools/launchd/test_driver_notify.sh` → rc=0 AND `==== 50 passed, 0 failed ====`
  (unchanged suite against the new production tree — the D-3 invariant).
- `make test-launchd-drivers` → rc=0.
- `/bin/bash -n tools/launchd/lib/pin-root.sh tools/launchd/mission-control.sh
  tools/launchd/test_pin_root.sh` (one invocation per file) → rc=0 each. (`make` also runs the
  glob syntax sweep; run it per-file first for a named failure.)

**M1 mutation drill (MUT-A..G, S, U — apply to the REAL `tools/launchd/lib/pin-root.sh`; protocol = Lane Rule 6):**

| MUT | Exact edit (template; record the verbatim landed command) | Named arm(s) that must go red | Expected FAIL line |
|-----|-----------------------------------------------------------|-------------------------------|--------------------|
| MUT-A | rev-list range → `"HEAD..$origin_dev_sha"` (source drift formula) | AC-A | `FAIL: age is exactly 0 at the origin/dev tip` |
| MUT-B | delete the T6 computation block (assignments stay `?`) | AC-B (also AC-A) | `FAIL: sha pin has exact line AGE=3` |
| MUT-C | rev-list range → `"$origin_dev_sha..$target"` (reversed) | AC-B | `FAIL: sha pin has exact line AGE=3` |
| MUT-D | delete `PIN_AGE="${AILANG_DRIVER_AGE:-?}"` readback (T3) | AC-C | `FAIL: carries age across the exec` |
| MUT-E | remove `AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA` from the :323-export (assignments remain) | AC-C2 | `FAIL: real exec exports AILANG_DRIVER_AGE=9 to the driver` |
| MUT-F | delete the T5 `${PIN_AGE_SUPPORTED:-}` warning `if` | AC-D | `FAIL: pre-handoff compatibility warning fires` |
| MUT-G | T3 readback default `:-?` → `:-0` (missing age defaults to zero) | AC-C missing-age witness (D-2) | `FAIL: missing age carries as ?, never zero` |
| MUT-S | T6 normalize `''\|\|*[!0-9]*) age="0"` (failed/empty count coerced to 0) | AC-N | `FAIL: failed age rev-list yields AGE=?` |
| MUT-U | rev-list receives the mutable name: `"$target..origin/dev"` | AC-P | `FAIL: reported age is B's count, not the moved ref's` (and/or `FAIL: baseline shim fired exactly once` if the rev-parse is deleted by the mutation) |

**Risks:** (i) the AC-P shim's advance+push+fetch runs inside the driver subprocess — keep the
shim's baked paths absolute and its delegation total, or every git call in the fire breaks;
mitigated by the counter file and the N+1 positive control. (ii) AC-D's checked deletions rot as
the helper text evolves — mitigated by the per-substitution match-count assertions (fixture rot
reds immediately, never vacuously greens). (iii) The M1 tree must keep the unchanged notify suite
green — the D-3 invariant is a G1 gate line, so a breach fails loudly instead of slipping to M2.

### M2: Notifier arms (AC-E..M, AC-O, AC-Q-notice) + H4 changelog + mutation audit record (~3.5 h, ~+290 LOC)

**Goal:** Extend the notify harness to prove the consumer: both extractions, `run_age`/`run_both`
harnesses, the eleven arm sections below, one changelog entry, and the durable mutation-audit
record covering ALL 22 mutants (M1's nine transcribed from `.snap/drill/M1/` evidence, M2's
thirteen drilled here).

**Estimated:** test_driver_notify.sh ~+195/0 · changelogs/v0.32-current.md ~+10/0 ·
`design_docs/planned/m-pin-drift-blind-under-sha-pin-mutation-audit.md` ~+90 (new file).
**Dependencies:** M1.

**Tasks (anchors verified in P6-P9):**

- [ ] **T13 — extractions** (test_driver_notify.sh :69-83): add
  `awk '/^if \[ -n "\$_pin_age_degraded" \]; then/,/^fi$/' "$DRV" > "$LAB/pin_age_block.sh"` and
  `awk '/^# --- DRIVER PIN STATE PATHS START ---/,/^# --- DRIVER PIN STATE PATHS END ---/' "$DRV"
  > "$LAB/state_paths.sh"`; add both names to the non-empty FATAL guard (:81-83) and the size
  echo (:84). The :70 pin_decision extraction already spans the age decision (it lives inside
  the existing START/END) — do not change that line; add an age-decision-only extraction
  `awk '/^# --- DRIVER PIN AGE DECISION START ---/,/^# --- DRIVER PIN AGE DECISION END ---/'`
  ONLY if an arm needs it standalone (AC-D's consumer witness already covers it in M1 — reuse,
  don't duplicate).
- [ ] **T14 — `run_drift` neutral extension** (:138-175): inside the subshell add
  `PIN_AGE=0; PIN_AGE_FILE="$4/pin-age"` (neutral: below threshold, `rm -f` on a per-arm fresh
  file, one benign log line) and source `$LAB/pin_age_block.sh` as a new positional arg `$9`
  between the drift and pin blocks. Existing numeric arms untouched; genuinely-unset arms
  (drift-f '?', drift-j) keep their exact current inputs — PIN_AGE=0 neutral applies to all
  run_drift arms uniformly. Verify the D-3 invariant is preserved for every existing arm name.
- [ ] **T15 — `run_age` harness** (new function modeled on run_drift, same discipline):
  `$1=status $2=age $3=warn ("" = genuinely unset) $4=age-state-value (absent for no file)`;
  per-arm `mktemp -d` state; sets `PIN_STATUS/PIN_AGE` (omitted entirely when unset arms require
  it), `AILANG_DRIVER_AGE_WARN` (unset when `$3` = `unset`), `PIN_AGE_FILE="$state/pin-age"`,
  `PIN_DRIFT=0`, `PIN_DRIFT_FILE="$state/pin-drift"`, `STATE_DIR="$state"`, identity fixtures
  `AILANG_DRIVER_REF="${AGE_REF:-origin/dev}"`, `AILANG_DRIVER_PINNED="${AGE_SHA:-deadbee}"`,
  `AILANG_DRIVER_AGE_BASE_SHA="${AGE_BASE:-<40-hex fixture>}"`; sources notify, pin_decision,
  pin_age_block; echoes `DECISION_RC` and `STATE:`/from the age file.
- [ ] **T16 — `run_both` harness** for AC-L: one shared `mktemp -d` STATE_DIR across four
  sequential fires ((drift,age) = (170,30), (170,60), (340,60), (340,3)), each fire sourcing
  notify+decision+drift-block+age-block with its own fresh MC_TRACE_FILE; per fire capture both
  state files' contents and the per-type sends (distinguishable in the trace: titles
  `source clone drifted` vs `driver pin is stale` ride the `AILANG:messages send controlplane …
  --title …` record, per P6).
- [ ] **T17 — the eleven arm sections** (each `mktemp -d` STATE_DIR, suite PATH stubs and
  MC_TRACE_FILE only — Lane Rule 4), with these exact PASS lines:
  - `age paths` (AC-O; kills **MUT-T**) — source `$LAB/state_paths.sh` with `STATE_DIR=$(mktemp
    -d)` and `MISSION_NAME=v1`, then `motoko`: `PASS: age-paths-v1: v1 resolves
    mission-control.pin-age` · `PASS: age-paths-v1: age path is distinct from the drift path` ·
    `PASS: age-paths-mission: motoko resolves mission-motoko.pin-age` · `PASS:
    age-paths-mission: age path is distinct from the drift path`
  - `age-a` (AC-E; kills **MUT-H**, **MUT-I**) — age=25 warn=25 state absent: `PASS: age-a:
    first threshold age notice reaches both channels with count` · `PASS: age-a: notice carries
    pinned ref, target SHA and baseline SHA` · `PASS: age-a: notice carries the
    executing-old-code wording` · `PASS: age-a: age state stores 25`
  - `age-b` (AC-F; kills **MUT-J**) — 170, prev 170: `PASS: age-b: equal age dedupes, state
    unchanged, no sends` · `PASS: age-b: dedupe is logged positively with the previous count`
  - `age-c` (AC-G; kills **MUT-K**, **MUT-L**) — 340 prev 170, plus companion 339: `PASS: age-c:
    doubling notifies both channels and stores 340` · `PASS: age-c2: one below doubling (339)
    dedupes`
  - `age-d` (AC-H; kills **MUT-M**) — 3 prev 170, then 25/absent: `PASS: age-d: below threshold
    re-arms and removes age state` · `PASS: age-d-followup: threshold age emits again after
    re-arm`
  - `age-f` (AC-I; kills **MUT-N**) — `?` then malformed `abc`, prev 170: `PASS: age-f: unknown
    age is log-only and preserves state` · `PASS: age-f2: malformed age is log-only and
    preserves state`
  - `age-i` (AC-J; kills **MUT-O**) — age 3 with warn 0 / -1 / abc / genuinely-unset, then
    age 3 warn 2: `PASS: age-i1: zero warn is floored loudly to 25` · `PASS: age-i2: negative
    warn is floored loudly to 25` · `PASS: age-i3: malformed warn is floored loudly to 25` ·
    `PASS: age-i4: unset warn defaults quietly (no floor log)` · `PASS: age-i5: positive
    override below 25 stays valid (2 emits at age 3)`
  - `age-j` (AC-K; kills **MUT-P**) — PIN_AGE genuinely unset under `set -u` (drift-j's
    standalone heredoc shape, :357-376): `PASS: age-j: unset PIN_AGE under set -u is log-only,
    not an abort` (`DECISION_RC:0`, unknown log, no sends, state untouched)
  - `age-status` (AC-M; kills **MUT-R**) — STALE age=30, then disabled age=30: `PASS: age-m1:
    STALE skips age with no pin-age send, state untouched` · `PASS: age-m2: the original
    pin-failure notice still works` · `PASS: age-m3: disabled skips age silently with state
    untouched`
  - `age-independent` (AC-L; kills **MUT-Q**) — the T16 four-fire sequence: `PASS:
    age-independent-1: drift 170 + age 30 emits both notices` · `PASS: age-independent-2: age 60
    emits age only, drift state stays 170` · `PASS: age-independent-3: drift 340 emits drift
    only, age state stays 60` · `PASS: age-independent-4: age 3 removes age state only, drift
    state stays 340` (both file contents AND per-type sends asserted at each step)
  - `age-q notice half` (AC-Q; kills **MUT-V**) — run_age with `AGE_REF=origin/feature`, age=26,
    SHA/baseline fixtures equal to M1's observed lab values (40-hex fixtures; the *binding* to
    the real exec — ref name and count — is proven in M1's §16): `PASS: age-q: notice sends on
    both channels and names origin/feature, never a hardcoded origin/dev ref` (required
    substring `origin/feature`; forbidden `Pinned ref: origin/dev` AND `Pinned ref:
    \`origin/dev\``) · `PASS: age-q: baseline SHA is printed and age state stores 26`
- [ ] **T18 — H4 changelog** (`changelogs/v0.32-current.md`): under the existing `##
  [Unreleased]` (:5), add one `### Fixed —` entry (matching the shape of the existing entry at
  :7-12, which is **preserved byte-exact**); recommended title `### Fixed — a stale driver pin
  reported drift 0: it now reports its age behind origin/dev`, body naming PIN_AGE, the
  AILANG_DRIVER_AGE(_BASE_SHA) transport, threshold 25/dedupe-until-doubling, notice type
  `pin-age`, and the old-helper `PIN_AGE_SUPPORTED` diagnostic. Exact prose is the executor's —
  see Open Questions.
- [ ] **T19 — mutation audit record** (`design_docs/planned/m-pin-drift-blind-under-sha-pin-mutation-audit.md`,
  new): one row per MUT-A..MUT-V (22 rows) with: mutation id · target file · one-line edit · the
  verbatim apply command landed · `/bin/bash -n` rc (=0, mutant must BUILD) · pre/post sha256
  (must differ, mutant LANDED) · the drill command + named arm(s) · observed rc + exact FAIL
  line(s) · restore sha256 match (byte-identical to pre-mutation) · post-restore green re-run rc.
  M1's nine rows transcribe `.snap/drill/M1/` evidence; M2's thirteen rows (table below) are
  drilled in this milestone into `.snap/drill/M2/`. A surviving mutant is a finding about the
  TEST ROW: record it as `SURVIVOR — report to controller`, never tune a fixture count.
- [ ] **T20 — M2 gates + drill + snapshot.**

**M2 acceptance tests:**

- `/bin/bash tools/launchd/test_driver_notify.sh` → rc=0 AND `==== 81 passed, 0 failed ====`,
  with the 31 exact new PASS lines from T17 present, AND the pre-existing 50 intact (the
  extraction guard at :81-83 and the wiring check `(7)` at the suite tail still PASS).
- `/bin/bash tools/launchd/test_pin_root.sh` → rc=0 AND `==== 116 passed, 0 failed ====`
  (unchanged since M1).
- `make test-launchd-drivers` → rc=0.
- `/bin/bash -n tools/launchd/test_driver_notify.sh` → rc=0 (production files unchanged this
  milestone, but re-run `-n` on them anyway as a drill-restore canary).
- `grep -n '### Fixed' changelogs/v0.32-current.md | head -3` shows the new entry under `##
  [Unreleased]` AND the previous entry still present; `wc -l
  design_docs/planned/m-pin-drift-blind-under-sha-pin-mutation-audit.md` ≥ 60 and the file
  contains 22 MUT rows with rcs and restore hashes.

**M2 mutation drill (MUT-H..R, T, V — apply to the REAL `tools/launchd/mission-control.sh`; protocol = Lane Rule 6):**

| MUT | Exact edit (template) | Named arm(s) that must go red | Expected FAIL line |
|-----|------------------------|-------------------------------|--------------------|
| MUT-H | threshold `-ge` → `-gt` (`$PIN_AGE` vs `$_pin_age_warn`) | age-a (AC-E) | `FAIL: age-a: first threshold age notice reaches both channels with count` |
| MUT-I | age notice body copied from the drift body / wrong identity source | age-a (AC-E) | `FAIL: age-a: notice carries pinned ref, target SHA and baseline SHA` (or the wording PASS) |
| MUT-J | delete the doubling gate (always emit at/above threshold) | age-b (AC-F) | `FAIL: age-b: equal age dedupes, state unchanged, no sends` |
| MUT-K | doubling `-ge $((prev*2))` → `-gt` | age-c (AC-G) | `FAIL: age-c: doubling notifies both channels and stores 340` |
| MUT-L | lower the doubling boundary (e.g. `* 2` reduced) | age-c companion (AC-G) | `FAIL: age-c2: one below doubling (339) dedupes` |
| MUT-M | delete the below-threshold `rm -f "$PIN_AGE_FILE"` | age-d (AC-H) | `FAIL: age-d: below threshold re-arms and removes age state` |
| MUT-N | unknown branch coerces to 0 and/or removes state | age-f (AC-I) | `FAIL: age-f: unknown age is log-only and preserves state` |
| MUT-O | delete the warn validation/floor, or impose min-25 on all overrides | age-i (AC-J) | `FAIL: age-i1: zero warn is floored loudly to 25` (or `age-i5` for the min-25 variant) |
| MUT-P | bare `$PIN_AGE` reference (no `:-?`) or default zero in the consumer | age-j (AC-K) | `FAIL: age-j: unset PIN_AGE under set -u is log-only, not an abort` |
| MUT-Q | age path reads/writes/removes `PIN_DRIFT_FILE` | age-independent (AC-L) | `FAIL: age-independent-…` (the first step whose file/send assertion breaks) |
| MUT-R | delete the non-pinned status guard | age-status (AC-M) | `FAIL: age-m1: STALE skips age with no pin-age send, state untouched` |
| MUT-T | typo or shared age path in either mission branch (T8 lines) | age-paths (AC-O) | `FAIL: age-paths-…` (the branch mutated) |
| MUT-V | hardcode `origin/dev` in the notice's `Pinned ref:` line | age-q (AC-Q) | `FAIL: age-q: notice sends on both channels and names origin/feature, never a hardcoded origin/dev ref` |

## Lane Rules (binding on the executor)

1. **Sandboxed executor, no git authority on the worktree.** The executor (pi lane,
   deepseek-v4-flash) may edit files ONLY inside this worktree and must never run `git add`,
   `git commit`, `git stash`, `git checkout`, `git reset`, `git branch`, `git worktree`, or
   `git push` against the worktree — in particular never `git checkout -- <file>` to undo a
   mutant (the milestone edits are uncommitted). The suites' and shims' own git writes happen
   under `mktemp` labs (`$T`, `$LAB`) — those are expected and permitted. The controller commits
   one commit per milestone after reviewing `.snap/M<k>/`.
2. **Gate order, each with its expected result:** G0 preflight baselines → `/bin/bash -n` per
   touched file (rc=0) → `/bin/bash tools/launchd/test_pin_root.sh` → `/bin/bash
   tools/launchd/test_driver_notify.sh` → `make test-launchd-drivers` (rc=0). Expected totals:
   M1: pin **116/0**, notify **50/0**; M2: pin **116/0**, notify **81/0**. Baselines re-measured
   by this planner at `f4c3dfd11`: pin `81 passed, 0 failed` rc=0, notify `50 passed, 0 failed`
   rc=0 (matches the controller's 9422ff628 numbers).
3. **Sandbox honesty.** Any gate result the executor cannot trust inside its sandbox (e.g. a
   blocked syscall, a PATH it cannot reproduce) must be labelled `UNINFORMATIVE UNDER SANDBOX`
   next to the command and output — the controller re-runs every gate outside the sandbox before
   recording anything. Never paste an expected line as if it were observed.
4. **Stubbed channels only.** Every acceptance arm that tests a notice uses the suite's existing
   PATH stubs for `ailang`/`gh` and the unified `MC_TRACE_FILE` — never a real channel; the
   `AILANG_MESSAGES_STORE`/`AILANG_MESSAGES_PROJECT` unset at suite :67 stays. Every state file
   lives under a per-arm `mktemp -d` STATE_DIR. **NEVER write a sentinel into a real
   `$HOME/.ailang/state/*` path** — guards on live paths are OBSERVATIONS only: sha256 + line
   count before/after any existing live pin-age/pin-drift file (`absent` is a legitimate
   reading), plus a `git status --short`-style observation of the repo before/after the suites.
5. **Snapshots are the handoff.** After each milestone, before moving on: `mkdir -p
   .snap/M<k>/` mirroring repo-relative paths of every file created or modified in that
   milestone, plus `SHA256SUMS` (`shasum -a 256`). `.snap/M2/` is cumulative (includes M1's
   files). `.snap/drill/M1/`, `.snap/drill/M2/` hold mutation evidence (hashes, rcs, FAIL
   lines). Restore discipline for drills is copy-based: `cp file .snap/drill/<mut>.bak` BEFORE
   the mutant, `cp` back after, assert `shasum -a 256 file` equals the recorded pre-mutation
   hash, remove the .bak. **Never `git checkout --`.**
6. **Mutation drill protocol (verbatim discipline):** apply ONE mutant at a time to the real
   file → assert it BUILDS (`/bin/bash -n` rc=0; a non-building red is not a kill) → assert it
   LANDED (`shasum -a 256` differs from pre-mutation; both hashes recorded) → run the named
   arm(s) (notify suite: a slice reassembled from the suite's own lines — harness preamble
   `sed -n '1,<arm-start-1>p'` + the named arm block + the totals trailer, i.e. only the named
   arm executes; pin suite: the suite truncated with `sed -n '1,<named-arm-end>p'` + totals
   trailer, because the lab arms are stateful-sequential — setup and predecessor arms execute as
   prerequisites, the red ASSERTION is scoped to the named arm; record any cross-killed sibling
   arms too) → record suite rc + the exact `FAIL:` line → restore → assert sha256 identical to
   pre-mutation → re-run the named slice green. A mutant that survives (builds + landed + still
   green) is a finding about the TEST ROW, not the code — write `SURVIVOR` in the audit record
   and report to the controller; never tune fixture counts to force red.
7. **Bash 3.2.57 portability everywhere** (production and tests): no `declare -A`, no `${v,,}`,
   no `mapfile`/`readarray`, no GNU `timeout` (the suite's `_pin_bounded`/`_mc_bounded` pattern
   is the only bound), POSIX-ish `case` patterns as in the existing code.
8. **PASS names and totals are binding.** The 35+31 assertion names and the 116/81 totals in
   this plan are the graded contract. An executor that needs to rename or recount must stop and
   get controller sign-off recorded in the handoff note.
9. **Line-anchored value assertions in the pin lab.** The fake driver prints both `AGE=<n>` and
   `ENV_AGE=<n>`, so the suite's substring `check "…" "$OUT" "AGE=3"` would alias `ENV_AGE=3` and
   could vacuously green a broken readback (MUT-D). All AGE/AGE_BASE_SHA/ENV_* value assertions
   use `printf '%s\n' "$OUT" | grep -q '^NAME=value$'` (feed ok/bad from grep's rc). `checkno`
   against a value is safe as a substring (an alias only makes it fail-safe red).
10. **AC-P shim discipline (design-frozen):** shim delegates ALL commands to the real git
    resolved once at shim creation; intercepts only the matching `rev-parse --verify --quiet
    'origin/dev^{commit}'` call; captures B, advances the synthetic origin by one commit, fetches
    into the source clone, returns B; counter file proves exactly-one fire; shim leaves PATH
    after the arm. No production file carries a hook; the shim never retypes producer logic.

## Day-by-Day (Day 1, ~8 h)

- **M1 (≈4.5 h):** T1-T7 pin-root.sh (1 h) · T8-T10 mission-control.sh (1 h) · T11 pin-lab arms,
  fixtures and shims (1.5 h) · G0/G1 gates + nine-mutant drill + `.snap/M1/` (1 h).
- **M2 (≈3.5 h):** T13-T17 notify harness + eleven sections (1.5 h) · T18 changelog + T19 audit
  record with the thirteen-mutant drill (1 h) · M2 gates, sweep outputs pasted to
  `.snap/M2/acceptance-sweep.txt`, cumulative snapshot (1 h).
- **Controller (outside the 8 h):** review `.snap/`, re-run every gate outside the sandbox,
  commit M1 then M2, fold the audit record into the iteration evidence.

## Success Metrics

- All 17 acceptance rows (AC-A..AC-Q) green via their named PASS lines; suites at 116/0 and
  81/0; `make test-launchd-drivers` rc=0.
- All 22 doc-named mutants drilled with build/land/red/restore evidence; zero unexplained
  survivors (any `SURVIVOR` row is a reported, controller-visible finding).
- Design-freeze invariants held and observable: drift semantics untouched (case 1 `DRIFT=1`,
  case 3 carry 7, drift-a..j all still PASS); age state file never touches `PIN_DRIFT_FILE`
  (AC-L); unknown never becomes zero (AC-D/AC-I/AC-K/AC-N); anchors byte-preserved (extractions
  non-empty, wiring check PASS).
- Documentation: `changelogs/v0.32-current.md` (+1 entry, prior Unreleased entry intact),
  `design_docs/planned/m-pin-drift-blind-under-sha-pin-mutation-audit.md` (new, 22 rows).

## Dependencies

None in-repo beyond this worktree at `f4c3dfd11`. No Go toolchain needed (the suites are pure
bash/git on synthetic repos). The controller owns: commits, PR, and any post-merge observation.

## Open Questions

- **AC-N fault-injection mechanism** (doc leaves it unnamed): use the AC-P pattern — a
  test-local PATH `git` shim that faults one invocation shape and delegates everything else.
  Recommended (already reviewer-ratified for AC-P; no production hook otherwise exists).
- **AC-Q notice-half inputs:** the lab's SHAs are mktemp-ephemeral, so the notify arm cannot
  quote them; the binding between the halves is the *logical* identity (`origin/feature`, age
  26), proven from the real exec in M1 §16, with 40-hex fixtures for the SHA fields in M2.
  Recommended; the alternative (sharing the lab across suites) is heretical to suite isolation.
- **Log punctuation / test helper names** (`run_age`, `run_both`, section numbers, shim
  filenames): executor's choice per the design's Deferred Decisions; the frozen parts are the
  anchored extractions, unknown-prefix wording, state isolation, identity fields, PASS names,
  and mutation outcomes.
- **Changelog heading verb:** `### Fixed —` (matches both the live Unreleased entry and the
  file's convention; the `check_changelog.sh` gate is structural on the ROOT index, not on
  entry verbs). Prose is the executor's; the entry must name PIN_AGE, the transport vars, the
  25/doubling policy, notice type `pin-age`, and the old-helper diagnostic.
- Scope stays exactly here: two milestones, one day, `tools/launchd/` + changelog + audit record
  only. No launcher, env, charter, skill, or CI edits.

## Notes

- The unchanged notify suite MUST stay 50/0 against M1's tree (D-3). The mechanism: the
  pin_decision extraction always normalizes `PIN_AGE` to `?` first and the unknown branch is
  log-only with no file access, so no existing arm aborts under `set -u`; the age notice block
  is a sibling top-level block the unchanged harness never sources. This is a gate line, not a
  hope.
- MUT-G's producer-side killer (the `missing age carries as ?, never zero` assertion in case 3)
  is a planner addition beyond the doc's AC table, made transparently under D-2; without it
  MUT-G is a known survivor in the producer file.
- The `PIN_AGE_SUPPORTED` marker is source-time state, NOT an environment transport: it is
  never exported and never read from env; the refresh hop unsets it so only the sourced blob's
  own declaration counts.
- Baselines were re-measured by running both suites at `f4c3dfd11` in this worktree
  (81/0 and 50/0, rc=0 each); logs at `/tmp/iter354_pin_baseline.log`,
  `/tmp/iter354_notify_baseline.log` on the planning machine (ephemeral — the controller
  re-runs anyway).
- Planner: AILANG V1 iter-354 planner lane (pi agent), worktree
  `.wt-v1-iter354-pin-drift` at `f4c3dfd11`. Design doc: quorum-approved r3 (rounds 1/2 blocked,
  carve-out fixes applied verbatim).
