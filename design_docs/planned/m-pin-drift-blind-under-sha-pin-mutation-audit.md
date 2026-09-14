# Mutation audit — M-PIN-DRIFT-BLIND-UNDER-SHA-PIN (sprint `v1_iter354_pin_age`)

**Tree**: M1 `e1dcb1796` (pin-root.sh / mission-control.sh / test_pin_root.sh) + M2 working tree
(test_driver_notify.sh, this record, changelog). **Suites at the audited tree**:
`test_pin_root.sh` **117 passed, 0 failed** (plan contract 116; one assertion was split into two numbered
siblings — controller sign-off under Lane Rule 8), `test_driver_notify.sh` **81 passed, 0 failed**
(plan contract 81), `/bin/bash -n` rc=0 on all four touched shell files.

**Who drilled**: the CONTROLLER (V1 iteration 354), outside the executor sandbox, because both pi
executor runs (ollama-cloud deepseek for M1, OpenRouter deepseek for M2) hit the lane's 30-minute wall
clock before reaching the drill step; the executor's own gate readings were labelled
`UNINFORMATIVE UNDER SANDBOX` and were re-measured here. Protocol per Lane Rule 6: one mutant at a time
against the REAL file, `/bin/bash -n` (BUILDS), sha256 differs (LANDED), the whole suite run, every
`FAIL:` line recorded, `cp`-restore, sha256 identical to PRE, green re-run. No fixture count or
assertion was tuned to force a red; the two row defects found are recorded below and were repaired
as ROW fixes with the mutant re-drilled afterwards.

## Verdict: 22 of 22 named mutants KILLED after two row repairs; 3 extra mutants drilled

| Mutant | Result | Note |
|---|---|---|
| MUT-A..E, G, S, U (pin-root.sh) | KILLED by the named arm | see M1 table |
| MUT-F (delete the T5 warning) | KILLED, rc=1, by the AC-D **fixture-rot guard** (`t5warn=0 matched 0 lines`), NOT by the named assertion | the AC-D fixture derives the old helper from the file under test, so ANY edit of the T5 text starves the fixture before the arm runs; MUT-F′ (`!= "1"` → `= "1"`) and MUT-F″ (delete `unset PIN_AGE_SUPPORTED`) behave identically. The named assertion `pre-handoff compatibility warning fires` is the positive control (warning observed in a real hop), not the killer. Row's predicted FAIL line is wrong for every T5 mutation; recorded, not tuned. |
| MUT-H..Q, T, V (mission-control.sh) | KILLED by the named arm | see M2 table |
| MUT-R (delete the non-pinned status guard) | **SURVIVED** the first drill (suite rc=0) → ROW REPAIRED → KILLED (rc=1, `age-m1` and `age-m3` red) | with previous state 170 and age 30, the doubling dedupe suppressed the notice and left the state file untouched, so a deleted guard was invisible. Repair: both arms now run with an ABSENT previous state and assert `AGE_STATE:absent`, making the status guard the only suppressor. Re-drilled after the repair: `==== 79 passed, 2 failed ====`, restored byte-identical, green re-run 81/0. |


---

# M1 mutation drill — controller-run (V1 iter-354), tree = plan commit 74b284f22 + M1 working tree

PRE sha256 pin-root.sh: 82293ba646900439c958c758c04df8d611397a6fd5dc978eb61158ec958966db

| MUT | edit | build rc | landed | suite rc | predicted FAIL present | all FAIL lines | restore sha match | post-restore total |
|---|---|---|---|---|---|---|---|---|
| MUT-A | `rev-list --count "$target..$origin_dev_sha"` → `rev-list --count "HEAD..$origin_dev_sha"` | 0 | True | 1 | YES | 3: FAIL: age is exactly 0 at the origin/dev tip; FAIL: note carries the pinned-age clause; FAIL: failed age rev-list yields AGE=? | True | ==== 114 passed, 3 failed ==== |
| MUT-B | `age="?"` → `age="?"; origin_dev_sha="?"` | 0 | True | 1 | YES | 8: FAIL: age is exactly 0 at the origin/dev tip; FAIL: baseline SHA is origin/dev's full commit; FAIL: note carries the pinned-age clause; FAIL: sha pin has exact line AGE=3 | True | ? |
| MUT-C | `rev-list --count "$target..$origin_dev_sha"` → `rev-list --count "$origin_dev_sha..$targ` | 0 | True | 1 | YES | 6: FAIL: sha pin has exact line AGE=3; FAIL: sha pin note names age 3 and the baseline SHA; FAIL: real exec exports AILANG_DRIVER_AGE=9 to the driver; FAIL: pinned pass reads back exact line AGE=9 | True | ==== 111 passed, 6 failed ==== |
| MUT-D | `PIN_AGE="${AILANG_DRIVER_AGE:-?}"` → `(deleted)` | 0 | True | 1 | YES | 6: FAIL: age is exactly 0 at the origin/dev tip; FAIL: note carries the pinned-age clause; FAIL: carries age across the exec; FAIL: sha pin has exact line AGE=3 | True | ? |
| MUT-E | `export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_` → `export AILANG_DRIVER_PINNED AILANG_DRIVE` | 0 | True | 1 | YES | 8: FAIL: age is exactly 0 at the origin/dev tip; FAIL: baseline SHA is origin/dev's full commit; FAIL: note carries the pinned-age clause; FAIL: sha pin has exact line AGE=3 | True | ? |
| MUT-F | `if [ "${PIN_AGE_SUPPORTED:-}" != "1" ]; then printf '%s\n' "` → `(deleted)` | 0 | True | 1 | NO — SURVIVOR? | 0:  | True | ? |
| MUT-G | `PIN_AGE="${AILANG_DRIVER_AGE:-?}"` → `PIN_AGE="${AILANG_DRIVER_AGE:-0}"` | 0 | True | 1 | YES | 1: FAIL: missing age carries as ?, never zero | True | ? |
| MUT-S | `case "$age" in ''|*[!0-9]*) age="?" ;; esac` → `case "$age" in ''|*[!0-9]*) age="0" ;; e` | 0 | True | 1 | YES | 3: FAIL: failed age rev-list yields AGE=?; FAIL: failed age rev-list fabricates no zero; FAIL: failed baseline resolution keeps AGE and baseline ? | True | ==== 114 passed, 3 failed ==== |
| MUT-U | `rev-list --count "$target..$origin_dev_sha"` → `rev-list --count "$target..origin/dev"` | 0 | True | 1 | YES | 1: FAIL: reported age is B's count, not the moved ref's | True | ==== 116 passed, 1 failed ==== |

Post-drill green re-run: rc=0 ==== 117 passed, 0 failed ====; sha256 == PRE: True

## Controller note on MUT-F (measured 2026-09-14, V1 iter-354)
MUT-F (delete the T5 warning line) is KILLED with suite rc=1 but by the AC-D fixture-rot guard
(`fixture error: old-helper substitution 't5warn=0' matched 0 lines`), NOT by the plan's predicted
`FAIL: pre-handoff compatibility warning fires` — the AC-D fixture derives the old helper from the
helper under test, so deleting the line starves the fixture before the arm runs. The named assertion
is non-vacuous against the LOGIC mutant MUT-F' (`!= "1"` → `= "1"`, warning never fires while the
line exists): see the MUT-F' row below. Both mutants are loud; the row's predicted kill line is wrong
for the deletion form and right for the logic form. Recorded, not tuned.

MUT-F' (`!= "1"` → `= "1"`) and MUT-F'' (delete `unset PIN_AGE_SUPPORTED`) — both build rc=0, landed, suite rc=1,
killed by the fixture-rot guard (`t5warn=0` / `t5handshake=0` matched 0 lines), 0 `FAIL:` lines, restored
byte-identical. CONCLUSION: every mutation of the T5 text is detected loudly by the fixture guard before the
AC-D arm runs; the named assertion `pre-handoff compatibility warning fires` acts as the POSITIVE control
(warning observed in a real hop) rather than as the killer. Controller sign-off (Lane Rule 8): pin total
117 (plan 116) — the executor split `failed baseline resolution keeps AGE and baseline ?` into two
numbered assertions; all 35 plan-named lines present, 81 baseline names retained.

---

# M2 mutation drill — controller-run (V1 iter-354), tree = M1 commit e1dcb1796 + M2 working tree

PRE sha256 mission-control.sh: e5798a3e90753717fc000c2b9f9d8ae32da63afd963d817dba700f6e91d0e7ed

| MUT | edit | build rc | landed | suite rc | predicted FAIL present | all FAIL lines | restore sha match | post-restore total |
|---|---|---|---|---|---|---|---|---|
| MUT-H | `if [ "$PIN_AGE" -lt "$_pin_age_warn" ]; then` → `if [ "$PIN_AGE" -le "$_pin_age_warn" ]; ` | 0 | True | 1 | YES | 5: FAIL: age-a: first threshold age notice reaches both channel; FAIL: age-a: notice carries pinned ref, target SHA and basel; FAIL: age-a: notice carries the executing-old-code wording; FAIL: age-a: age state stores 25 | True | ==== 76 passed, 5 failed ==== |
| MUT-I | `Pinned ref: ${AILANG_DRIVER_REF:-origin/dev}. Target SH` → `Pinned ref: ${AILANG_DRIVER_REF:-origin/` | 0 | True | 1 | YES | 1: FAIL: age-a: notice carries pinned ref, target SHA and basel | True | ==== 80 passed, 1 failed ==== |
| MUT-J | `if [ "$PIN_AGE" -ge $((_pin_age_previous * 2)) ]; then` → `_pin_age_emit=1` | 0 | True | 1 | YES | 4: FAIL: age-b: equal age dedupes, state unchanged, no sends; FAIL: age-b: dedupe is logged positively with the previous c; FAIL: age-c2: one below doubling (339) dedupes; FAIL: age-independent-3: drift 340 emits drift only, age sta | True | ==== 77 passed, 4 failed ==== |
| MUT-K | `if [ "$PIN_AGE" -ge $((_pin_age_previous * 2)) ]; then` → `if [ "$PIN_AGE" -gt $((_pin_age_previous` | 0 | True | 1 | YES | 3: FAIL: age-c: doubling notifies both channels and stores 340; FAIL: age-independent-2: age 60 emits age only, drift state ; FAIL: age-independent-3: drift 340 emits drift only, age sta | True | ==== 78 passed, 3 failed ==== |
| MUT-L | `if [ "$PIN_AGE" -ge $((_pin_age_previous * 2)) ]; then` → `if [ "$PIN_AGE" -ge $((_pin_age_previous` | 0 | True | 1 | YES | 1: FAIL: age-c2: one below doubling (339) dedupes | True | ==== 80 passed, 1 failed ==== |
| MUT-M | `if [ "$PIN_AGE" -lt "$_pin_age_warn" ]; then` → `if [ "$PIN_AGE" -lt "$_pin_age_warn" ]; ` | 0 | True | 1 | YES | 2: FAIL: age-d: below threshold re-arms and removes age state; FAIL: age-independent-4: age 3 removes age state only, drift | True | ==== 79 passed, 2 failed ==== |
| MUT-N | `log "driver pin age: unknown ($PIN_AGE); notice suppres` → `log "driver pin age: unknown ($PIN_AGE);` | 0 | True | 1 | YES | 3: FAIL: drift-j: unset PIN_DRIFT is log-only, not a set -u abo; FAIL: age-f: unknown age is log-only and preserves state; FAIL: age-f2: malformed age is log-only and preserves state | True | ==== 78 passed, 3 failed ==== |
| MUT-O | `case "$_pin_age_warn" in` → `(deleted)` | 0 | True | 1 | YES | 3: FAIL: age-i1: zero warn is floored loudly to 25; FAIL: age-i2: negative warn is floored loudly to 25; FAIL: age-i3: malformed warn is floored loudly to 25 | True | ==== 78 passed, 3 failed ==== |
| MUT-P | `PIN_AGE="${PIN_AGE:-?}"` → `PIN_AGE="${PIN_AGE:-0}"` | 0 | True | 1 | YES | 2: FAIL: drift-j: unset PIN_DRIFT is log-only, not a set -u abo; FAIL: age-j: unset PIN_AGE under set -u is log-only, not an  | True | ==== 79 passed, 2 failed ==== |
| MUT-Q | `printf '%s\n' "$PIN_AGE" > "$PIN_AGE_FILE"` → `printf '%s\n' "$PIN_AGE" > "$PIN_DRIFT_F` | 0 | True | 1 | YES | 8: FAIL: age-a: age state stores 25; FAIL: age-c: doubling notifies both channels and stores 340; FAIL: age-d-followup: threshold age emits again after re-arm; FAIL: age-independent-1: drift 170 + age 30 emits both notic | True | ==== 73 passed, 8 failed ==== |
| MUT-R | `if [ "$PIN_STATUS" != "pinned" ]; then` → `if false; then` | 0 | True | 0 | NO — SURVIVOR? | 0:  | True | ==== 81 passed, 0 failed ==== |
| MUT-T | `PIN_AGE_FILE="$STATE_DIR/mission-${MISSION_NAME}.pin-ag` → `PIN_AGE_FILE="$STATE_DIR/mission-${MISSI` | 0 | True | 1 | YES | 2: FAIL: age-paths-mission: motoko resolves mission-motoko.pin-; FAIL: age-paths-mission: age path is distinct from the drift | True | ==== 79 passed, 2 failed ==== |
| MUT-V | `Pinned ref: ${AILANG_DRIVER_REF:-origin/dev}. Target SH` → `Pinned ref: origin/dev. Target SHA:` | 0 | True | 1 | YES | 1: FAIL: age-q: notice sends on both channels and names origin/ | True | ==== 80 passed, 1 failed ==== |

Post-drill green re-run: rc=0 ==== 81 passed, 0 failed ====; sha256 == PRE: False


---

MUT-R re-drill after the row repair (controller, 2026-09-14): build rc=0, landed True, suite rc=1, `FAIL: age-m1: STALE skips age with no pin-age send, state untouched`, `FAIL: age-m3: disabled skips age silently with state untouched`, `==== 79 passed, 2 failed ====`, restore sha match True, post-restore `==== 81 passed, 0 failed ====`.
