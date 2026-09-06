# M-PI-RUNNER-WORKTREE-ASSERTION-VACUOUS-ON-REVISION — replace the post-run dirty-worktree count with a bounded per-invocation content delta

**Status**: needs-human-review — iteration337, D-58; not approved for execution
**Target**: v0.35.2
**Priority**: P1 — it is a measured false-green in the runner whose only load-bearing assertion is this one
**Estimated**: ~1.25 days (≈0.5 d bounded snapshot helper + watchdog in `scripts/mission_pi_run.sh`, ≈0.5 d new hermetic `test_mission_pi_snapshot.sh` + revising TEST 1/TEST 5 + artifact/physical-path normalization, ≈0.25 d verification and judgement)
**Dependencies**: none. Parent context: `design_docs/planned/v0_35_0/m-pi-harness-upgrade.md` (no blocking dependencies).
**Planner-Lane**: codex-ok (single bash script + two bash suites; no AILANG surface)

---

## Problem Statement

`scripts/mission_pi_run.sh` is the pi executor lane's guard. Its own header says the worktree
assertion is the only **load-bearing** one: `stopReason` is measurable evadable in both directions,
so the runner relies on "the worktree changed" to decide whether a finished run did any work. That
assertion measures the tree's `git status` **count**, not the delta this run produced — so a
revision pass over an *already-dirty or already-untracked* file can report `ok` while writing
nothing. Quorum R1 (3/3 reject) and the in-session controller independently confirmed the root
cause; the direction (before/after content snapshots) is deemed appropriate. This revision answers
the concrete spec gaps: **bounded waits, conflict surface, symlink no-follow, distinct-path count,
physical-path normalization, pipefail isolation, and honest failure diagnostics.**

**Current State (verified first-party; see Verification Log):**
- `scripts/mission_pi_run.sh:232` — `DIFF_LINES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')`.
- `scripts/mission_pi_run.sh:241-243` — `if [ "${DIFF_LINES:-0}" -gt 0 ]; then VERDICT_NAME="ok"; RC=0 else VERDICT_NAME="empty_worktree"; RC=10`.
- `scripts/mission_pi_run.sh:60/146` — `set -u` and `set -m` only; **no** `set -e`, **no** `set -o pipefail`, **no** `trap`.
- Verified runner integration: `START=$(now)` at line 167 (after launch); process-group kill pattern at lines 172-179; `wait "$RUNNER_PID"`; verdict JSON built with `worktree_changed_files` at line 257. Repo has **24,380 tracked paths**.

A dirty file **edited again** → status still `M`, count identical → the delta is invisible. Live
repro (writeless stub `pi`, synthetic repos) matches independently:

| arm | current verdict | current rc | should be |
|-----|----------------|------------|-----------|
| clean (nothing written) | `empty_worktree` | 10 | 10 (correct) |
| pre-dirty tracked, no write | `ok` | 0 | **10 / empty_worktree** (defect) |
| pre-existing untracked, no write | `ok` | 0 | **10 / empty_worktree** (defect) |

**Impact:** the false-green is aimed at the runner that exists to detect exactly this class
("reports success for work never requested"). The existing suite **codifies** the defect (TEST 1
line 42-45 and TEST 5 line 79-82 pre-dirty and assert `ok` on a writeless stub).

---

## Goals

**Primary Goal:** make `empty_worktree` (rc 10) mean byte-for-byte identity of the eligible
filesystem content before/after the run, so a no-op run fails **even when the tree was already
dirty or untracked**, and an edit that leaves status/name identical (edited-`M`) still counts.

**Success metrics:** every arm in the tables below returns the "should be" column; one edited file
reports `worktree_changed_files == 1` (distinct paths, not symmetric records); a snapshot that
cannot be established reports `snapshot_error` (rc 15), **never** `ok` or `empty_worktree`; every
snapshot completes under a finite configured deadline and its subprocesses are reaped on expiry.

## Non-Goals (bounded scope)

- **No** change to pi launch, stdin/stdin-close, process-group kill, the `message_update` awk
  filter, the ndjson size model, stall/`reasoning_stall`/`stream_dead`/`wall_timeout` detection, or
  the `agent_end`/`tool_execution_end` counting. Those are pre-existing launch/stall mechanics.
  Existing stall verdicts are **preserved unchanged**; `snapshot_error` only replaces the `finished`
  branch when a snapshot cannot be established.
- **No** global `set -o pipefail` (would change `PI_RC` semantics of the existing `pi | awk`
  pipeline). Error propagation is made explicit per-helper (see Fail-closed).
- **No** git index mutation, staging, or native-git content hashing (`git hash-object`, filter
  hooks, `git add`); it writes to the object DB (side effect) and is SHA-1 — see Conflict Surface.
- **No** hostile-concurrency detection. Correctness assumes a **quiescent tree** during each
  snapshot point (before launch / after `wait`); a concurrent writer racing the snapshot is
  unsupported and out of scope (worst case: `snapshot_error` or a stale-but-consistent digest, never
  a wrong `ok`). Symlink **no-follow** is asserted exactly and tested.
- **No** gitignored files as work; **no** repair of the pre-existing T3 stall-timing flake in
  `test_mission_pi_run.sh` (inherited, measured planner 8/9 vs controller 9/9 under `POLL1`); not
  folded into, and not required to pass by, this change.
- **No** general filesystem engine; scope is the eligible worktree paths + a private temp dir.

---

## Proposed Approach: a bounded per-invocation content delta

Replace the `git status | wc -l` count with a **before/after content snapshot** around the pi run.
Snapshot points are **independent of `START`** (line 167 stays post-launch for wall-timeout):
before-snapshot is taken after output truncation and **before** the pi job launches; after-snapshot
is taken after `wait "$RUNNER_PID"`. Both run under a finite watchdog (below).

### Verdict/exit-code surface (one additive code)

- **Unchanged:** `0 ok`, `10 empty_worktree`, `11 reasoning_stall`, `12 stream_dead`,
  `13 wall_timeout`, `14 launch_failed`.
- **New:** `15 snapshot_error` — the before or after eligible-content snapshot could not be
  established (enumeration/hash/name-reject/type-reject/deadline/non-repo). Fail-closed: a snapshot
  failure **never** resolves to `ok` or `empty_worktree`. `empty_worktree` is emitted only when
  both snapshots succeeded **and** their digests are equal.
- `worktree_changed_files` becomes the **distinct changed-path count** (deleted+created+edited
  paths), not the old status-line count; informational, never the deciding signal.
- `snapshot_error` carries an **honest** reason in the verdict JSON (e.g. `"snapshot_error":
  "SNAP_ERR: deadline 60s exceeded"`), and never claims the tree is unchanged.

### The bounded snapshot (finite deadline + reap)

Define `content_snapshot <workdir> <manifest-path>`, called only inside a watchdog:

```
snapshot_guarded <SNAPSHOT_DEADLINE> content_snapshot "$WORKDIR" "$BEFORE" || SNAP_ERR=1
... pi runs ...
snapshot_guarded <SNAPSHOT_DEADLINE> content_snapshot "$WORKDIR" "$AFTER"  || SNAP_ERR=1
```

`snapshot_guarded` reuses the runner's **existing process-group kill pattern**
(`scripts/mission_pi_run.sh:172-179`): launch the snapshot in its own group (job control is already
on via `set -m` line 146), poll liveness ~0.2 s up to the deadline, and on expiry
`kill -TERM -- -PGID; sleep 1; kill -KILL -- -PGID; wait` (reap), returning rc 125 → `snapshot_error`.
`SNAPSHOT_DEADLINE` default **60 s**, env `MISSION_PI_SNAPSHOT_DEADLINE`. This single deadline
**covers enumeration + hashing + sort + compare**; it is the enforceable read/bound bound, and the
batch size (`xargs -n 300`) bounds per-iteration spawn count. There is no unbounded wrapper spawn:
one helper process group per snapshot, no recursive fork.

**Measured representative full-tree cost** (real repo, 24,381 eligible paths, ~3.7 MB, this
machine): enumeration 0.04 s, batch `xargs shasum` 1.95 s, sort+digest 0.00 s, full `content_snapshot`
**≈5.5 s**. Default 60 s ≈ **10× headroom**, and the watchdog still hard-bounds a pathological tree.

### Manifest format and content identity

One manifest per snapshot; **binary-content-safe and name-byte-safe**. Line = `<relpath>\t<TAG>`,
`LC_ALL=C sort` on the whole line. `<relpath>` is relative to the worktree root and must contain no
NUL/tab/CR/LF (reject → `snapshot_error`, fail closed). Tag:

| path kind | TAG | content source |
|-----------|-----|----------------|
| regular file, not executable | `F::sha256` | SHA-256 of file bytes |
| regular file, executable | `F:+:sha256` | SHA-256 of file bytes (exec bit captured) |
| symlink | `L:sha256` | SHA-256 of **link TEXT** (`readlink`), never the target |
| absent on this side | `A` | deletion / not-yet-created |
| other object type | — | **reject** → `snapshot_error` (explicit, no silent fallback) |

Eligible paths = `git ls-files -z` + `git ls-files --others --exclude-standard -z`. Per-path
*binding* is done in a NUL-safe bash loop (builtins only, no per-file subprocess except link/absent
edges); content *hashing of regular files is batched* into a single `xargs -0 -n 300 shasum -a 256`
pass (order-preserved and pasted back onto the path list). No GNU `sort -z`, no `read -d ''`, no
`declare -A` — the prototype runs on Darwin bash 3.2 (Bash 3.2 the rig default).

### Delta, verdict, and the distinct-path count

- **equality:** `sha256(before-manifest) == sha256(after-manifest)` ⇒ `empty_worktree` rc 10.
- **changed:** else `ok` rc 0. `worktree_changed_files` = distinct changed **path** count:
  `comm -23` (paths only in before = deleted) + `comm -13` (only in after = created) + `join -t "\t" -1 1`
  (paths in both whose TAG differs = edited). This is a distinct-path count, **not** the symmetric
  record-set difference (which would double-count an edited path). Measured: editing one already-`M`
  file reports **1**.

### Symlink no-follow (exact, tested)

Hash the **link text**, never the target. Consequences (all measured in the prototype):
- retarget `a → b` where `a`/`b` are byte-identical targets: `shasum` of the target is the **same**
  (retarget missed by target-hash) but the link-text hash **differs** → detected.
- external target-content edit with the link text unchanged: link-text digest is **stable** (no
  false positive on the link), and we never read the target, so a link pointing **outside** the
  worktree (or to a blocking target such as a fifo) is never dereferenced and cannot stall or leak
  reads. No-follow is a guarantee; the tree's real, tracked content is still captured normally.

### Runner-owned artifact exclusion — exact physical path, not broad directory

Resolve `OUT`, `SNAP` (`${OUT}.snapshot.ndjson`), `ERR` (`${OUT}.stderr`), `VERDICT` to **physical
absolute** realpaths (`cd … && pwd -P`, symlinks resolved). For each, if it physically lies under the
worktree physical root, exclude that root-relative path (match on the manifest path field up to the
tab). If outside, exclusion is a no-op. This gives **physical-alias normalization**: two lexical
spellings of the same file (symlink, `.` / relative-from-subdir) exclude correctly; root-relative
paths are unambiguous. A broad-directory exclusion is expressly avoided (it would swallow
legitimate neighbor files). Runner truncates outputs exactly as today **before** the before-snapshot;
because artifacts are exact-excluded on both sides, truncation cannot influence the delta. Add a
startup diagnostic that logs the four resolved artifact realpaths so each exclusion decision is
visible (no silent surprise). A caller-`cwd` relative `--out` resolves against the caller's cwd
(existing behaviour); if it lands inside the worktree it is excluded, if outside it is never
enumerated — either way safe.

### Fail-closed and pipefail isolation (SUT has no pipefail today)

Current options are `set -u` (line 60) + `set -m` (line 146) with **no** `set -e`/`pipefail`/`trap`.
Fail-closed is achieved **without adding global `set -o pipefail`**:
- each `content_snapshot` return is captured directly and short-circuits
  (`… || return 1`), so a failing `git ls-files`/batch hash/name-reject/type-reject propagates;
- `set -o pipefail` is enabled **only inside the `content_snapshot` helper subshell** (scoped), so
  the top-level `pi | awk` pipeline and `PI_RC` semantics are untouched;
- non-repo workdir ⇒ `git ls-files` fails ⇒ helper rc nonzero ⇒ `snapshot_guard` returns it ⇒
  `SNAP_ERR=1` ⇒ rc 15.

### Snapshot precedence and honest diagnostics

Dispatch: in the `finished` branch, `if SNAP_ERR ⇒ snapshot_error rc 15`, else digest decision.
All **non-finished** outcomes (`reasoning_stall`/`stream_dead`/`wall_timeout`/`launch_failed`) are
returned **unchanged**, so snapshot errors never mask a stall verdict (a snapshot is only taken on a
finished run anyway). `snapshot_error` JSON includes a reason field; it never claims
"unchanged".

### Code changes (proposed; the old prototype does not establish any of this)

- `scripts/mission_pi_run.sh`: add `content_snapshot`, `snapshot_guarded`, `snapshot_delta` helpers;
  take before/after snapshots around the run under the deadline; replace the `finished`-branch
  decision (`DIFF_LINES` at 232/241-243/257) with digest + distinct-count; map snapshot failure to
  rc 15; emit `snapshot_error` reason; one `trap 'rm -rf "$SNAPDIR"' EXIT` for `$TMPDIR`-scoped temp
  cleanup (the script currently has no trap; this is a single net-add, owner of its own temp dir,
  independent of the pi group-kill which uses `kill`, not traps).
- `scripts/test_mission_pi_snapshot.sh` (**new**): hermetic, direct-helper suite (no pi, no timing)
  covering every arm below.
- `scripts/test_mission_pi_run.sh`: revise TEST 1/TEST 5 to start clean and actually write a file
  (proving real work ⇒ `ok`); add dirty/no-op and untracked/no-op integration arms.
- Doc-comment only, on the two source-of-truth code enumerations (runner header
  `mission_pi_run.sh:35-40`; `mission-control/SKILL.md:1410`): append `15 snapshot_error`. No
  consumer code change (see Conflict Surface).

---

## Conflict Surface

| Concern | Code evidence | Resolution |
|---------|---------------|------------|
| Global `pipefail` could alter `PI_RC`/pipeline semantics | `set -u` L60, `set -m` L146; `pi --mode json \| awk …` L148-158; `PI_RC` at L177 | Do **not** add global `pipefail`; scope it inside `content_snapshot` only; snapshot errors propagate via explicit rc checks |
| Watchdog vs existing stall/wall loop | stall/wall poll loop L166-197; group-kill L172-179 | New `snapshot_guarded` reuses the **same** kill/reap pattern; it is a separate, shorter deadline (default 60 s) that never touches the pi loop |
| Trap ownership for `SNAPDIR` cleanup | script has **no** trap today (grep: 0 hits); test suite has its own `trap 'rm -rf "$TMP"' EXIT` (separate process) | One new `trap … EXIT` in the runner, scoped to `${TMPDIR:-/tmp}`, does not conflict with the harness's own trap |
| rc-15 consumers | only operational consumer is `mission-control/SKILL.md:1403-1445`: documents `0/10/11/12/13/14`, treats **any non-zero as LANE FAILURE** (fallback+flag); `stream_dead`(12) is the one retry exception | rc 15 (non-zero) is already captured by that branch — **no consumer code change**; only the two enumerating comments gain `15`. `nightly_classify.py`/`eval_harness/metrics.go` `reasoning_stall` is the **AILANG runtime category**, unrelated to the pi verdict |
| Native Git diff machinery instead of a bespoke hash | proposal `git diff --name-only`/`status --porcelain` name/status comparison | A content edit that leaves status/name identical (editing an already-`M` file) is **invisible** to any name/status comparison — measured counterexample (porcelain stays `M f.txt` while bytes change); index hashing (`git hash-object`) writes the object DB and is SHA-1 (disallowed); `git diff` does not cleanly enumerate untracked-content deltas. A manifest of eligible bytes is the minimal correct instrument |
| Process census across 24,380 paths | 24,380 tracked paths measured; a per-file `shasum` spawn would be ~48 k spawns | Hash regular files in **one batched** `xargs -0 -n 300 shasum` pass (~1.95 s measured); per-path binding uses bash builtins only; full snapshot **≈5.5 s** measured, under a 60 s hard deadline |
| Make/CI wiring | existing suite has **zero** Make/CI calls (negative grep + positive-control same call); no `timeout(1)`, GNU sort `-z`, or `read -d ''` reliance | **No new Make/CI wiring**; new suite is a plain `bash scripts/test_mission_pi_snapshot.sh`; Darwin-bash-3.2 portable (hex/path-keyed `LC_ALL=C` sort, `tr`-based NUL reject) |

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Content delta is a pure function of eligible bytes before/after; identical eligible content ⇒ identical verdict; no `git status` count/ordering ambiguity |
| A2: Replayability | +1 | Manifests are bankable; digest reproducible from the same tree state |
| A3: Effect Legibility | 0 | No AILANG effect surface touched |
| A4: Explicit Authority | 0 | Reads the worktree + a private `$TMPDIR` snapshot dir; no new ambient authority; symlinks never dereferenced |
| A5: Bounded Verification | 0 | No type-checking surface touched |
| A6: Safe Concurrency | 0 | No concurrency change; snapshots run sequentially before/after the run under a finite deadline |
| A7: Machines First | +1 | Replaces a human-readable count with a machine-verifiable content digest; `empty_worktree` becomes a falsifiable claim |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No token/cost surface touched |
| A10: Composability | 0 | Additive rc 15 + `worktree_changed_files` meaning; existing fields/codes unchanged; no consumer code change |
| A11: Structured Failure | +1 | New explicit `snapshot_error` rc 15, fail-closed, honest reason field, never resolves to `ok`/`empty` |
| A12: System Boundary | +1 | Constrains reads to eligible worktree paths + private temp; symlink no-follow never reads outside; hard deadline/reap |

---

## Testing Strategy

**New hermetic `scripts/test_mission_pi_snapshot.sh`** (deterministic, no pi, no stall timing; runs
a synthetic stub `pi`-free direct-helper suite). **Existing** `test_mission_pi_run.sh` TEST 1 and
TEST 5 revised (start clean + actually write); dirty/no-op + untracked/no-op integration arms added.
The pre-existing T3 stall-timing flake is **isolated** (not folded into and not required by this
change). No Make/CI calls are added; suites run as `bash scripts/test_…sh`.

---

## Acceptance Criteria

1. Finished, nothing written, clean repo ⇒ `empty_worktree` rc 10 (preserved).
2. Finished, nothing written, **pre-dirty (tracked)** ⇒ `empty_worktree` rc 10 (defect fixed).
3. Finished, nothing written, **pre-existing untracked** ⇒ `empty_worktree` rc 10 (defect fixed).
4. Edit an **already-dirty tracked file to new content** (status still `M`) ⇒ `ok` rc 0 —
   content identity, not status/name; `worktree_changed_files` == **1** (distinct, not 2).
5. Create a new untracked file ⇒ `ok` rc 0.
6. Write **only** a gitignored file ⇒ `empty_worktree` rc 10 (scoped).
7. Non-repo worktree ⇒ `snapshot_error` rc 15, never `ok`.
8. `--out`/`--verdict` inside the worktree: exact-path excluded; **output-only no-op ⇒
   `empty_worktree`**; a **neighbor** edit is not swallowed ⇒ `ok`; a symlinked/differently-spelled
   artifact path resolving to the same file is excluded (physical-alias normalization).
9. Content-preserving rename ⇒ `ok` (path-set change; distinct count = deleted+created).
10. **Symlink retarget** `a → b` with byte-identical targets ⇒ `ok` (link-text sees it).
11. Symlink pointing at a **blocking/outside** target (fifo) ⇒ no dereference, no out-of-tree read,
    no hang; snapshot completes under the deadline.
12. `chmod +x` and regular-file↔symlink **type change** ⇒ `ok` (mode/type captured).
13. Filenames containing newline/tab/CR ⇒ `snapshot_error` (fail-closed, honest diagnostic), never
    mis-serialized.
14. Snapshot **deadline expiry** ⇒ `snapshot_error` rc 15; subprocesses reaped; verdict never
    `ok`/`empty`; `reasoning_stall`/`stream_dead`/`wall_timeout`/`launch_failed` preserved unchanged.
15. Snapshot failure emits an honest `snapshot_error` reason; never claims "unchanged".
16. Existing verdict fields and `0/10/11/12/13/14` unchanged; only `15` added; no global `set -o
    pipefail`; `PI_RC`/`pi | awk` semantics unchanged.

---

## Risks

- **Deadline false-empty.** A snapshot that times out fails to `snapshot_error` (not `empty`); the
  60 s default ≈ 10× the measured 5.5 s, so expiry indicates a pathological/hostile tree, correct
  to fail closed.
- **Name-byte rejection.** A newline/tab filename makes a snapshot fail closed. Such names are
  vanishingly rare in a code worktree; failing loudly is the safe, honest choice (never mis-serialized).
- **External symlink-target edits.** A link to a tracked target whose content changes is detected
  via the target's own record (correct); a link to an *external* target is intentionally not
  followed (no out-of-tree read), so an external edit is invisible — that is the desired boundary.
- **Bash 3.2 portability.** Solved: path-keyed `LC_ALL=C` sort, `tr`-based NUL reject, batched
  `xargs -0 shasum`, `read` on `tr '\0' '\n'` (no `read -d ''`), no GNU `sort -z`, no `declare -A`.
  Prototype ran on this machine (bash 3.2).

---

## Verification Log

Every current-codebase claim is first-party. Proposed/intended behaviour is validated in a
**/tmp/claude prototype only**; nothing was written into the worktree and no implementation exists.
Negatives carry a **positive control in the same call**.

| # | Claim | Exact command | Observed output |
|---|-------|---------------|-----------------|
| V1 | Verdict derives from a `git status` line count; `set` options are `-u`/`-m` only | `grep -n "DIFF_LINES\|VERDICT_NAME=\"ok\"\|set -\|pipefail\|trap" scripts/mission_pi_run.sh` | `60:set -u`, `146:set -m`; `232:DIFF_LINES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null \| wc -l…)`; `242:if [ "${DIFF_LINES:-0}" -gt 0 ]…`; zero `pipefail`/`trap` hits |
| V2 | `START` is taken after pi launch (independent of before-snapshot point) | `grep -n "RUNNER_PID\|\bSTART=" scripts/mission_pi_run.sh` | `153:RUNNER_PID=$!`, `167:START=$(now)` (after launch) |
| V3 | Suite **codifies** the defect (TEST 1/TEST 5 pre-dirty + assert ok on writeless stub) | `grep -n "dirty\|check \"happy path\"\|check \"slow-but-working" scripts/test_mission_pi_run.sh` | TEST 1 L42-45 `mkrepo … dirty` + `check "happy path" 0 … ok`; TEST 5 L79-82 same |
| V4 | **Live repro**: pre-dirty/pre-existing-untracked no-op ⇒ `ok` rc 0; clean ⇒ `empty_worktree` | `bash /tmp/claude/verify_mppi.sh` (writeless stub `pi`) | `clean rc=10 empty_worktree`; `dirty rc=0 ok` (**defect**); `untracked rc=0 ok` (**defect**) |
| V5 | **No-op on pre-dirty/untracked ⇒ equality ⇒ `empty_worktree`; edit-of-`M`-file distinct=1** | `bash /tmp/claude/mppi_snap_proto.sh` (prototype manifest+delta; synthetic repos) | `B dirty-noop: UNCHANGED 0 [ M f.txt ]`; `C untrk-noop: UNCHANGED 0`; `D dirty-edit: CHANGED 1 (…edited=1) [M f.txt]`; `E create: CHANGED 1`; `F rename: CHANGED 2` |
| V6 | **Symlink no-follow**: target-hash misses retarget; link-text sees it; stable on external target edit | `bash /tmp/claude/symtest.sh` | both target-hashes equal `b7db13…` (retarget missed by target-hash); link-text `8db51…` vs `1f395…` (retarget seen); link-text `1f395…` unchanged after external mutation (no false positive) |
| V7 | **Distinct-path count**, not symmetric records | prototype `D dirty-edit` (above) | one `M`-file edit returns `CHANGED 1`, status still `M` |
| V8 | **Name-byte reject** (newline filename) fails closed; non-repo fails closed | prototype arms K/L | `K newline-name: rc=1 SNAP_ERR`; `L non-repo: rc=1` |
| V9 | **Watchdog reaps on deadline**; symlink-to-fifo is safe (never followed) | prototype arms M1/M2 | `M1 watchdog-reap: rc=125, orphan procs=0`; `M2 symlink-to-fifo: rc=0 (no block)` |
| V10 | **Artifact exact-path exclusion**: output-only no-op ⇒ `empty`; neighbor edit ⇒ `ok` | prototype artifact arms (physical-cwd, in-worktree `--out`) | `output-only-noop: UNCHANGED 0`; `+neighbor-edit: CHANGED 1 (edited=1)` |
| V11 | **Full-repo snapshot cost** (24,381 eligible paths) | `bash /tmp/claude/costmain.sh`; timed `git ls-files -z`, batched `xargs shasum`, sort | enumeration 0.04 s; batch hash 1.95 s; sort+digest 0.00 s; full `build_manifest` **5.5-6 s** stable |
| V12 | Suite has **zero Make/CI calls** with a positive control in the same invocation | `grep -n "make \|CI" scripts/test_mission_pi_run.sh` (negative) **+** `grep -rln "make " scripts/` (positive control) | test file: `NONE`; positive control returns many hits (so the negative is real); repo tracked count `git ls-files \| wc -l` → `24380` |
| V13 | Only operational consumer of the pi verdict treats non-zero as LANE FAILURE | `grep -rn "mission_pi_run\|empty_worktree\|worktree_changed_files" .claude/skills/mission-control/` | SKILL.md:1410 documents `0/10/11/12/13/14`; L1441 non-zero → fallback+flag. `nightly_classify.py`/`metrics.go` `reasoning_stall` is the AILANG runtime category, unrelated |

---

## Related Documents

- `design_docs/v1-mission.md` line 512 — the queue entry naming this defect and the fix shape.
- `design_docs/planned/v0_35_0/m-pi-harness-upgrade.md` — the surrounding pi-harness migration.
- `scripts/mission_pi_run.sh`, `scripts/test_mission_pi_run.sh` — affected files; proposed new
  `scripts/test_mission_pi_snapshot.sh`.
- `.claude/skills/mission-control/resources/codex-lane-false-greens.md` — (3) sandbox verdicts and
  (5) observation-not-seeding; every "assert unchanged" arm is established by **observation**, never
  by a seeded write.
- Quorum R1 record `/tmp/v1-iter337-quorum-r1.json` (3/3 reject; synthesized objections addressed
  here).
- `CLAUDE.md` §2 — no silent fallbacks (fail-closed motivation).

---

## Estimated Effort

**≈1.25 days**: 0.5 d bounded snapshot helper + watchdog + artifact/path normalization in
`mission_pi_run.sh`; 0.5 d new hermetic `test_mission_pi_snapshot.sh` + revising TEST 1/TEST 5 and
adding dirty/untracked-integration arms; 0.25 d verification + judgement. A single contributor, no
AILANG/Go work, no network/GPU; all repros are local stub `pi` in synthetic trees.

---

**Designer-owned file:** `design_docs/planned/v0_35_2/m-pi-runner-worktree-assertion-vacuous-on-revision.md` (this document). No implementation, no git writes, no shared-state writes, no quorum round were made.


## Controller gate record — iteration337 (authoritative qualifications)

The preceding text is the rejected designer revision, retained as evidence, not an approved
implementation contract. Quorum R1 and R2 each rejected with all three external reviewers present.
No third quorum, sprint plan, production implementation, or controller approval was performed.
The independent evaluator reviews this park and its evidence separately.

R2 concrete reviewer fixes are retained verbatim in
`docs/sprint-retros/iter337-pi-runner-quorum-r2.json`. Astra requires checking every path ancestor
before reading the leaf; Gemini requires explicit path-only comm inputs (already done in the
prototype, insufficiently explicit in prose); GLM requests a complete consumer census.
The narrow-refinement carve-out was considered, not exercised: the controller still rejects
unresolved snapshot-failure ordering and physical artifact-alias semantics, which lack a complete
reviewer-authored resolution. The before snapshot is both required before launch and described as
finished-only; comparison is claimed deadline-covered although the shown watchdog calls exclude
it. `pwd -P` on the containing directory does not resolve a final-component output symlink.
Resolving these interactions here would require controller design judgment, beyond verbatim fixes.

First-party corrections (these override inaccurate claims above):
- The tree has 24,380 tracked files totaling 268,361,724 bytes, not approximately 3.7 MB.
  A separate controller run of the revised prototype completed in 5.442 seconds with 24,381
  manifest records and rc0. That is one local timing, not a general performance bound.
- The initial revision performance command had an unexported WD and was killed after 177 seconds;
  a later command had a syntax error. Neither is admissible timing evidence. All children ended.
- `git hash-object` without `-w` does not write the object database; a synthetic repository
  negative control and `-w` positive control confirmed this. The earlier blanket claim is false.
- Final-leaf-only `-L` testing does not prevent traversing a symlink ancestor. In a synthetic
  repository tracking d/f, replacing d with a symlink to an external directory made the prototype
  read external f bytes. No-follow therefore remains unimplemented and unproven.
- Command substitution strips trailing newlines from link text. Exact link-byte identity needs
  an explicit acceptance arm before any implementation can claim it.
- No guarantee is made under hostile concurrent writes. The text's unconditional “never a wrong
  ok” qualification contradicts its quiescence assumption and must be removed on revision.
- Consumer census: 106 matching lines in 14 files using repo-wide hidden rg over py/go/sh/md;
  the only executable consumers found were the runner and its shell suite. A positive control
  matched the runner. The complete path/line census is banked beside this document's quorum.
- Existing shell-suite Make/CI wiring is absent; T3 has a pre-existing timing flake (planner8/9,
  controller9/9 with POLL1). These are backlog defects, not evidence that this fix passed CI.
- The axiom labels above need alignment with the canonical axiom names before approval.

**D-58:** authorize one fresh designer revision and quorum to settle the bounded-snapshot and
artifact-alias contract (A, recommended), or keep this approach parked and commission a Git-native
content-comparison design (B). Default: remain parked immediately until an explicit human ruling.
Neither option authorizes implementation before design, planner, executor and evaluator gates.
