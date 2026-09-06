# M-GATE1-SHARED-CLONE-REF-DRIFT: A shared `.git` makes `origin/dev` a moving floor, so every gate that reads it must record what it read, when

**Status**: PLANNED — quorum-cleared by narrow-refinement carve-out (rev 2; both R2 objections verbatim-fixed); M1 implemented; recovery amendment below requires the existing sprint plan's M2/M3 file map to be refreshed before execution resumes.

**Quorum verification**: R1 (2026-09-06T08:49Z): BLOCKED — gemini (premise gaps), glm (double-snap race); revision applied 2026-09-06. R2 (2026-09-06T08:54Z): BLOCKED — gemini + glm independently on one surface, the `drift()` no-record path (`last` exits 0 with an empty string when the file exists but has no matching label, so the `\|\| return 2` guard never fires → false DRIFT with an empty old SHA). NARROW-REFINEMENT CARVE-OUT applied (both objections concrete, reviewer-authored, non-directional): gemini's verbatim fix (explicit `-n "$old"` check in `drift`) + glm's verbatim fixes (`last()` awk exits 1 on no-match; `git update-ref refs/remotes/origin/dev HEAD` as the primary mutation in the test recipe). No re-quorum run (carve-out routes straight to sprint-planner); the applied fixes are recorded here per the carve-out's contract.

**Recovery amendment (2026-09-06, post-quorum base drift):** the iteration branch is based on `034cdd02f`, while the shared `origin/dev` advanced through `c1212b3c` → `b5a5c45b` → `bdc5fbf7`. Commit `c1212b3c` mechanically split the operational 2,781-line skill into a 560-line index plus authoritative per-gate resources. This changes the M2/M3 **edit locations**, not the reviewed behavior, helper contract, conflict direction, or acceptance semantics. The authoritative recovery map is now:

| Reviewed concern | Current authoritative target on `origin/dev` |
|---|---|
| Gate 1 record | `.claude/skills/mission-control/resources/gate-1-observe.md` |
| Gate 3 pre-worktree re-read/drift | `.claude/skills/mission-control/resources/gate-3-route.md` |
| Gate 3b poll-target record | `.claude/skills/mission-control/resources/gate-3b-ci-green.md` |
| Gate 4 base record + routing evidence | `.claude/skills/mission-control/resources/gate-4-record.md` |
| Progressive-disclosure rationale | `.claude/skills/mission-control/resources/ref-drift.md` (new, linked from the relevant gate resources) |

The root `.claude/skills/mission-control/SKILL.md` is no longer an implementation target for M2/M3. Its 2,781-line historical ratchet remains a no-growth upper bound, but the operative gate is now link integrity plus the existing per-file context checks. This is a bounded recovery refinement: the upstream move only narrows each edit to the file that now owns the already-reviewed gate text. It neither answers nor overrides a reviewer objection; both R2 verbatim fixes remain unchanged and are already implemented in M1. The R1/R2 receipts therefore remain valid for the design direction and reviewer findings; no duplicate quorum or receipt rewrite is warranted. Verification rows 14–16 record the new first-party facts.

**Target**: v0.36.0
**Priority**: P1 (process-correctness; prevents a silent verdict-expiry class that has struck V1's own loop twice, iterations 331 and 332)
**Estimated**: 3 days (M1: 1d, M2: 1d, M3: 1d)
**Dependencies**: none hard. Adds a dedicated per-mission `mission-${MISSION_NAME}-base` state file for the durable record (NOT the heartbeat), reuses the driver's pin banner and the S-guard test suite; touches none of the S1–S5 guard literals.

## Problem Statement

### The mechanism: one clone, many writers, one shared `.git`

Every mission on this rig — V1, docs, world, motoko — runs out of **worktrees of the same source clone**. A git worktree does not get its own ref store: all worktrees and the source clone mutate the **same shared `.git`** (verified first-party: this very worktree reports `git rev-parse --git-dir` → `<shared>/.git/worktrees/-wt-v1-iter338`). Consequently `refs/remotes/origin/dev` is a **single ref shared by every mission and every attended session on the machine**. When a *sibling* runs `git fetch origin` (or an attended session pushes), `origin/dev` advances **for everyone**, with no action on V1's part and no message.

The mission-control skill's Gate 1 opens by reading that shared ref:

```bash
git fetch origin
git rev-parse dev origin/dev     # differ? origin is ground truth
```

That is a **point-in-time reading of a ref OTHER PROCESSES mutate**, and every later gate that trusts "the base" (Gate 1 sync, Gate 2 already-landed, Gate 3 worktree creation, Gate 3b poll-target pinning, Gate 4 record-base re-confirmation) reuses that reading, or a stale local copy of it, **without being told that the ref can move under it** — *nothing in the skill says so*. When a later gate disagrees with what Gate 1 saw, the natural reading is "I fetched, so origin moved; is this my error?" when the true cause is "a sibling's fetch moved the shared ref between my reads."

### Instance 1 (iteration 331) — the silent verdict, caught only by a side effect

dev went red **mid-iteration** from a concurrent attended session's landing. It was found **only because** the inherited red stranded the iteration's own Gate-4 record (the red was a REQUIRED context, so the record could not merge). Had it not stranded the record, the iteration would have proceeded on a base that silently contained a failing commit it had never reviewed. The log entry (v1-mission-log.md §331) records the surprise and the lucky find — "The remedy is the one 3g already names … I had not applied" — but the *base-expiry* itself is not called out as the class of bug it was.

### Instance 2 (iteration 332) — the point-in-time read demonstrably expired under me

Gate 1 read `origin/dev` at one SHA. **Minutes later** a `git worktree add … origin/dev` came up at a SHA **four commits ahead** (the attended session's fetch had advanced my ref) — with no fetch of my own in between. My worktree was created from a base Gate 1 had never seen. The log entry (§332) captures it honestly: *"`origin/dev` moved four commits mid-iteration (`fe9c08ffc`…`19d6b03c7`, the attended session's workbench Phase 2/3). Caught only because the worktree created from `origin/dev` came up at a SHA the Gate-1 fetch had never seen"* — and then the key admission: **"the clone is shared, so a sibling's fetch advances my refs under me."**

### Why it is silent

- The advance is **asymmetric**: a sibling's fetch moves *my* ref with no fetch on my side, so the local `FETCH_HEAD`/reflog story reads as "nothing new" while `origin/dev` has moved.
- Every gate that matters (3 worktree, 3b poll, 4 record) is *designed around* a stable base; when the base is not stable, the disagreement looks like an **operator error** (or a retry-induced ephemeral race — the skill's own "it self-healed on retry" war story) rather than like **drift from a known reading**.
- Nothing records *what* Gate 1 read, nor *when*. A later different reading therefore has no prior to compare against, so it cannot be classified as "drift" — it is just "wrong."

### What this doc produces

A **measurement-and-record discipline** (helper + minimal prose) so that (a) every gate that acts on a base re-reads the shared ref immediately before the act, (b) every quoted base is recorded **with its read time** in a durable, per-mission, machine-readable channel, and (c) a later disagreement reads as **DRIFT** (re-read and re-run the affected gate), never as an unexplained error. The two instances were benign only because the surprise was *visible*; this makes the surprise *expected and routed*.

## Goals

1. **Re-read before action.** Every gate that commits to a base (1 sync, 2 already-landed, 3 worktree creation, 3b poll-target pin, 4 record-base re-confirmation) re-reads `origin/dev` **immediately before** the act that depends on it, on the same line of reasoning Gate 3b already uses for SHA-pinning the poll target.
2. **Record SHA + read time, durably and per-mission.** A quoted base is `SHA@<ISO8601-UTC>`, written to a **dedicated per-mission append-only state file (`mission-${MISSION_NAME}-base`, via a `base-<label>` stamp — NOT the heartbeat, which the driver's slot-verdict reader owns)** **and** to the iteration's log **Routing evidence** row at Gate 4.
3. **Disagreements classify as drift, not error.** A later reading that differs from the recorded one is reported as `DRIFT <old> -> <new> (n commits)`, attributed to "shared-clone sibling/session," and the controller **re-reads → re-runs the affected gate** — aborting only when the drift invalidates a worktree/provenance (not on a benign advance).
4. **Measured, not prose-only.** A bash-3.2 `mission-base.sh` helper plus a CI test (`test_mission_base.sh`, wired into `make test-launchd-drivers`/the CI `launchd-drivers` job) that simulates a ref advance with **no real sibling**, so the guard is proven non-vacuous on every push.
5. **Context-clean and S-guard-safe.** The 560-line root SKILL index does not grow; every new resource link resolves; S1–S5 guard literals remain present in their current authoritative files; all additions are bash-3.2/portable and mission-independent.

## High-Impact Decisions

| # | Decision | Answer | Rationale |
|---|---|---|---|
| HD-1 | Where does the defense live? | **Option C (Hybrid): a testable `mission-base.sh` measurement helper + minimal authoritative gate-resource prose that names it, with rationale in `resources/ref-drift.md`.** | A tool nobody invokes is as useless as prose nobody measures. The helper gives a CI-provable instrument (non-vacuity test); the gate resources tell the controller to invoke it. Pure prose (A) can't be tested; pure tooling (B) is never called. See Rejected Options and the recovery amendment. |
| HD-2 | Which artifact carries "SHA + read time"? | **A dedicated per-mission state file `mission-${MISSION_NAME}-base`** (machine record, `base-<label>` stamp carrying the SHA in the 5th/note column) **+ the Gate-4 log Routing-evidence row** (human record, `base=<sha>@<iso>`) **+ the worktree's own HEAD** (provenance, already the base by construction). | A separate file keeps `base-*` rows OFF the heartbeat, whose **last row drives the driver's slot-verdict classifier** — measured: a trailing `base-*` row at a quiet rc=0 death flips REAPED→CRASHED (Verif. 10). Namespaced per-mission like every other state key, append-only, ISO-stamped. Routing-evidence is the loop's memory; the worktree HEAD *is* the base it was created from. |
| HD-3 | On a recorded disagreement, what does the controller do? | **Re-read once; if still different, classify DRIFT; re-run the affected gate against the fresh base. Abort only if the drift invalidates the action (e.g. the worktree/provenance must match a reviewed commit).** | A sibling advance is expected and benign; only a *consequence* can be harmful. Re-read-only risks re-validating against the same drifted ref; the affected gate must be re-run so its derived decisions (worktree creation, poll-target pin, record base) are against the fresh value. |
| HD-4 | Does Gate 3b re-read for the poll target? | **Yes, and the target is pinned to the fresh SHA (not `--limit 1`), exactly as the existing war-story mandates** — but the *poll target* and the *recorded base* are now the same re-read value, made explicit. | Gate 3b already has the SHA-pin war story; this extends the *base* to the same discipline and records it with a time. |
| HD-5 | `.agents/skills/mission-control/SKILL.md` — is it the same file? | **No.** It is a distinct 551-line condensed stub, **not** a copy of the 2781-line `.claude/skills` operational skill. No sync/copy mechanism exists (the only skill-install script, `tools/skill-updates/install-skill.sh`, handles only `ailang-feedback`, not mission-control). | So this change touches the operational `.claude` copy only; the `.agents` stub is hand-maintained separately. Verified by diff + line count (see Verif. 6). |

## Design Freeze

**None.** Every decision above is a process/correctness choice within the existing skill's own precedent (Gate 3b already SHA-pins; Gate 4 already re-confirms the base; the driver already logs a pin banner). No language semantics, no cross-mission policy, no model routing, no human-authority change. The durable-record artifact is now **fixed: a dedicated `mission-${MISSION_NAME}-base` state file** — the heartbeat is off-limits for base rows because its last row drives the driver's slot verdict (Conflict Surface 6).

## Deferred Decisions

- Whether `mission-base.sh snap` should `git fetch` by default or read the already-fetched shared ref. Default: **read-only `git rev-parse`** (matching the queue row "re-read `git rev-parse origin/dev`"—reading the *current* shared ref is exactly the drift-detection we need; a fetch would over-shoot the point). A `--fetch` flag is reserved for where a genuine fresh-origin guarantee is needed (Gate 1 already fetches; Gate 4 already fetches).
- Which field carries the SHA in the dedicated `mission-${MISSION_NAME}-base` file. Adopted: the existing 5-column format (`epoch<TAB>iso<TAB>label<TAB>attempt<TAB>note`) with `label=base-<n>` and `note=<sha>`; a dedicated 6th column is a future formatting change if a scraper ever needs it. Because the base file is standalone (same format, separate path), `mission-heartbeat.sh` is not touched.
- Whether Gate 2 needs a `base-` stamp at all (see Gate-by-gate below; it already re-reads fresh at pick time). Deferred to M3 as an observation.

## Solution Design

### Overview

Introduce one tiny, bash-3.2, portable measurement helper (`tools/launchd/mission-base.sh`) with three verbs — `snap`, `record`, `drift` — plus the **minimum** prose in each authoritative gate resource that tells that gate when to call it, with the war-story/rationale in a new `resources/ref-drift.md`. The helper writes the durable record to a **dedicated per-mission state file (`mission-${MISSION_NAME}-base`, separate from the heartbeat)** and introduces no new timestamp scheme. The recovery amendment supersedes the pre-`c1212b3c` monolith file map below wherever they conflict.

### Chosen option: C (Hybrid). Rejected options with reasons

- **A — Pure SKILL.md prose.** Add the re-read instruction + "record SHA with time" discipline in prose only. *Rejected:* (1) it pays the full ratchet tax (each added line must be offset by a line moved to `resources/`) *without* adding an instrument that CI can prove; (2) it recreates the exact failure it fixes — a controller that skips an unmeasured instruction reproduces "prose nobody measures." The queue row's own caveat names this.
- **B — Pure tooling.** A `mission-base.sh` that gates "call" but with no prose telling them to. *Rejected:* the skill is what the controller reads; a helper with no prose call-site is *exactly* "a tool nobody invokes." The one-line SKILL.md reference B permits is the same prose A wanted, just under-accounted.
- **C — Hybrid (CHOSEN).** Helper for the *measurement* + minimal prose that *names it* in each authoritative gate resource, with the explanatory bulk in `resources/ref-drift.md`. This satisfies "consider whether a tool nobody invokes is better than prose nobody measures" by making the tool *invoked by* the prose and *measured by* CI.

### The helper — `tools/launchd/mission-base.sh` (new, bash-3.2, portable)

```bash
#!/bin/bash
# mission-base.sh — record the shared origin/dev reading; classify later disagreement as drift.
# bash 3.2.57-safe: no associative arrays, no ${v,,}, no GNU timeout. Portable to ubuntu/windows/bash5.
set -u
STATE_DIR="${AILANG_STATE_DIR:-$HOME/.ailang/state}"
# Dedicated per-mission base state file. NOT the heartbeat: the driver's slot-verdict
# reader takes its verdict from the heartbeat's LAST row, so a trailing base-* row would
# flip REAPED->CRASHED and degrade the crash-site at= label. (mission-control.sh:1480-1502; Verif. 10)
BASE="$STATE_DIR/mission-${MISSION_NAME:-v1}-base"
REF="${MISSION_BASE_REF:-origin/dev}"

snap() {  # FULL sha<TAB>ISO8601-UTC from the CURRENT shared ref (no fetch: we read the shared .git)
  local sha iso
  sha=$(git rev-parse "$REF" 2>/dev/null) || { echo "mission-base: cannot resolve $REF" >&2; return 1; }
  iso=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  printf '%s\t%s\n' "$sha" "$iso"
}

record() {  # snap EXACTLY ONCE, append base-<label> row to the per-mission base file; echo sha<TAB>iso
  local label="$1" rec sha iso epoch attempt
  rec=$(snap) || return $?
  sha=${rec%%$'\t'*}; iso=${rec#*$'\t'}  # single-read invariant: the SHA recorded is the SHA of exactly one rev-parse
  epoch=$(date +%s); attempt="${MISSION_ATTEMPT:-1}"
  printf '%s\t%s\tbase-%s\t%s\t%s\n' "$epoch" "$iso" "$label" "$attempt" "$sha" >> "$BASE" \
    || { echo "mission-base: cannot write $BASE" >&2; return 1; }
  printf '%s\t%s\n' "$sha" "$iso"
}

last() {  # the full SHA last recorded under base-<label>; exits 1 when no matching row exists
  # (glm R2 verbatim fix) — so the missing-record guard in drift() fires for BOTH the
  # missing-file case and the exists-but-no-matching-label case, never a silent empty string
  awk -F '\t' -v l="base-$1" '$3==l {sha=$5} END {if (sha) {print sha; exit 0} else exit 1}' "$BASE"
}

drift() { # compare the last base-<label> row against a fresh snap; exit 1 on disagreement
  local label="$1" old new oldn n
  # (gemini R2 verbatim fix) — check for an empty string explicitly rather than relying on
  # the exit code: an empty `old` must be NO-RECORD (exit 2), never a false DRIFT with an
  # empty old SHA
  old=$(last "$label"); [ -n "$old" ] || { echo "mission-base: no base-$label record yet" >&2; return 2; }
  new=$(git rev-parse "$REF" 2>/dev/null) || return 1
  if [ "$old" = "$new" ]; then
    echo "base $label steady at $new"; return 0
  fi
  n=$(git rev-list --count "$old..$new" 2>/dev/null || echo '?')
  echo "DRIFT base $label $old -> $new ($n commits) — shared clone moved under this read"; return 1
}
```

Implementation note (exact bash-3.2 mechanics): `snap` is **the only sanctioned way a gate quotes a base** — one `git rev-parse` (no `--short`) + one `date -u`, printing `<full-sha>\t<iso>`. `record <label>` snaps **exactly once** into `$rec` and parses it — the single-read invariant, *the SHA recorded is the SHA of exactly one rev-parse*: no double-snap, so the recorded SHA can never diverge from the snap it resolves (fixes the glm double-snap race). It then appends a `base-<label>` row to the dedicated `mission-${MISSION_NAME}-base` file (same 5-column tab format, same state dir — a standalone companion; the heartbeat keeps only gate labels). `last` greps the last recorded SHA for a label (a negative control: empty proves no record was ever written). `drift <label>` re-snaps and compares; steady → exit 0; moved → exit 1 with the `DRIFT base <label> old -> new (n commits)` line; no record → exit 2. **The base file is the record; the heartbeat is untouched.** `${var%%$'\t'*}` and `${var#*$'\t'}` are bash-3.2-safe (no associative arrays); this is executable, and `test_mission_base.sh` runs `snap`/`record`/`last`/`drift` against a scratch clone and a temp `$AILANG_STATE_DIR`.

`mission-base.sh` is added to `make test-launchd-drivers` (`@/bin/bash tools/launchd/test_mission_base.sh`), which the CI `launchd-drivers (bash 3.2)` job already runs on macOS (see Verif. 9).

### Gate-by-gate edits (exact, with proposed wording/order) — and the enumeration the design question demands

**Gate 1 — OBSERVE (sync). INCLUDE: record the base.**
After the existing sync (`git fetch origin; git rev-parse dev origin/dev`), add, verbatim:

```bash
base=$(bash tools/launchd/mission-base.sh record gate1)   # records SHA + read-time; echoes them
echo "Gate 1 base: $base"          # e.g. "c9d5f7a1b2e3 2026-09-06T10:38:42Z"
```

Justification: this is the nominal base the iteration is supposed to reason about; it must be *recorded with a time* so every later gate (and every later reader of the record) has the prior to compare against. **Inclusion is the whole point of the doc.**

**Gate 2 — PICK + REALITY-CHECK (already-landed). RE-READ (already happens); stamp optional.**
The skill already sharpened this to "re-run `git fetch origin` immediately before the item-level check" (see §780–794). So Gate 2 already re-reads a fresh base at pick time. **Exclusion/observation:** no new `base-` stamp is *required* here, because the already-landed check re-reads origin at the moment it decides. But the *conclusion* "not already landed" is itself a base-relative claim; the doc directs Gate 2 to echo `git rev-parse origin/dev` into the pick note when it quotes a base, reusing `mission-base.sh snap`. Decision: **include a snap-for-the-record**, defer a full `base-` stamp to M3 after the first live iteration proves value. Justify inclusion (light): the already-landed verdict is the one that silently produced a redundant re-evaluation in instance-class iteration 12; recording the base it decided on makes the next gate's re-read comparable.

**Gate 3 — ROUTE + EXECUTE (worktree creation). RE-READ: this is the loaded instance-2 hole.**
Immediately before **any** `git worktree add ... origin/dev`, re-read and compare:

```bash
base=$(bash tools/launchd/mission-base.sh snap)          # fresh read, right now
newsha=${base%%$'\t'*}
oldsha=$(bash tools/launchd/mission-base.sh last gate1)
[ "$newsha" = "$oldsha" ] || bash tools/launchd/mission-base.sh drift gate1   # prints DRIFT, tells controller to re-run
```

and **create the worktree from the FRESH `$newsha`**, recording `base=$base` in the worktree's provenance / routing evidence. Justification: this is precisely where instance 2 struck — a worktree born from a base Gate 1 had never seen. Re-reading immediately before `worktree add` and creating from the fresh SHA means the worktree's HEAD *is* the recorded base; a disagreement is caught and re-routed *before* the worktree is materialized. The worktree's own HEAD is the provenance (HD-2).

**Gate 3b — CI GREEN (poll-target pinning). INCLUDE: re-read the base and pin the poll target to the same fresh SHA.**
The existing snippet already pins `target=$(git rev-parse origin/dev)` (full SHA, no `--short`). Extend it to (a) route the target through `mission-base.sh snap`, (b) record it as `base-3b`, (c) on drift from Gate 1, print `DRIFT base gate1->3b` and re-note the base. Proposed wording:

```bash
target_is="$(bash tools/launchd/mission-base.sh record gate3b)"; target=${target_is%%$'\t'*}
bash tools/launchd/mission-base.sh drift gate1 2>/dev/null || echo "DRIFT: base moved Gate1->Gate3b; re-verified poll target above"
rid=$(gh run list --branch dev --workflow CI --limit 10 --json databaseId,headSha \
      | jq -r --arg t "$target" '[.[] | select(.headSha == $t)][0].databaseId // empty')
```

Justification: the poll target is a *base* (the SHA that decides LANDED vs parked). Re-reading it fresh and *recording it with a time* closes the loop instance 117's `--limit 1` miss already fought — now the recorded value is queryable. **Inclusion mandatory: a poll target that silently watches the wrong run is the vacuous-green class at the most load-bearing verdict.**

**Gate 4 — RECORD (base re-confirmation). REUSE the existing re-confirmation; stamp + write to Routing evidence.**
Gate 4 already re-confirms the base before the first write (`git fetch origin; git rev-parse dev origin/dev`; §2196–2200). **Reuse, don't duplicate**: keep that fetch+rewrite, route the value through `mission-base.sh record gate4`, and on the **Routing evidence** row add `base=<sha>@<iso>`. Justification: Gate 4 is the record; the base it commits against *is* the memory. Writing it into Routing evidence means the human/mission log carries the same value the base-file machine-record carries — the two artifacts agree (HD-2).
**Exclusion (justified):** no new re-read *before* Gate 4 itself; its existing re-confirmation already performs the read at the moment of record, which is the correct point.

### What "record the base SHA with its read time" means, concretely

- **Artifact 1 — dedicated base state file (machine record):** `$AILANG_STATE_DIR/mission-${MISSION_NAME}-base` gains rows `epoch<TAB>iso<TAB>base-gate1<TAB>attempt<TAB><full-sha>`, `…-gate3b`, `…-gate4`, etc. Append-only, per-mission by construction, ISO-stamped, and standalone — **the heartbeat gains no rows**, so the driver's slot-verdict reader (which classifies from the heartbeat's last row) keeps seeing only gate labels.
- **Artifact 2 — log Routing-evidence row (human record):** at Gate 4, `**Routing evidence**: … base=c9d5f7a1b2e3@2026-09-06T10:38:42Z`. This is the line a future reader (or a later iteration picking up the log) compares against.
- **Artifact 3 — worktree provenance:** the worktree is created from the fresh `$newsha`, so `HEAD` of that worktree *is* the base; `mission-base.sh record gate3` plus the echo in the pick note names it.

### How a later disagreement is recognized as drift

The controller compares a **fresh `mission-base.sh snap`** (the current shared `origin/dev`) against the **last recorded `base-<label>` SHA** for the same stage in the dedicated base file. On match → steady. On mismatch → the helper prints `DRIFT base <label> $old -> $new ($n commits)` and exits 1. The controller then:
1. **Re-read once** (`snap` again) to rule out a transient mis-read (a `rev-parse` is atomic, so this is cheap insurance, not a superstition).
2. If still different → **re-run the affected gate** with the fresh base as its input: Gate 3 re-creates the worktree from `$new`; Gate 3b re-pins the poll target to `$new`; Gate 4 writes the record against `$new` and logs the `base=…@…`.
3. **Abort** only if the drift invalidates the *action's integrity*: e.g. Gate 3's worktree must be created from a SHA that a *reviewed* commit produced, and the drift moved it past a state the iteration already verified — in that case, park, report DRIFT, and let the queue's next iteration pick up (a benign advance never aborts; a consequence-bearing one does).

### Relations to driver-level scripts (reuse, not duplication)

- **`tools/launchd/mission-control.sh`** — already logs a pin banner citing the pinned SHA (`driver pin: running committed origin/dev @ <sha>`, via `PIN_NOTE`/`PIN_DRIFT`; see `pin_root_to_committed_ref`, Verif. 5). The new `mission-base.sh` **reuses the same `$REPO`-resolution and `$MISSION_NAME` defaulting spirit** but is a *different* concern: the driver pins the *code it runs*; `mission-base.sh` records the *base it reasons about*. They are complementary; the doc points the controller to both and does **not** merge them (a merge would couple a measurement helper to the pin/re-exec machinery).
- **`tools/launchd/mission-heartbeat.sh`** — NOT used for the base record. Its label whitelist (`case "$label" in fired|gate-0|…|complete|abort)`, unknown → exit 2; lines 11–14) applies to the LABEL field and would reject `base-<label>` strings, and its last row drives the driver's slot verdict. The design therefore writes `base-<label>` rows to the **separate `mission-${MISSION_NAME}-base` file**, leaving `mission-heartbeat.sh` and its whitelist completely untouched.

## Implementation Plan

### M1 — Measurement & record helper + non-vacuity test (~1 day)
- Create `tools/launchd/mission-base.sh` (`snap`, `record`, `last`, `drift`), bash-3.2, no assoc arrays, no `${v,,}`, portable; reads the current shared `$REF`, records to a dedicated `mission-${MISSION_NAME}-base` file under `$AILANG_STATE_DIR` (off the heartbeat).
- Create `tools/launchd/test_mission_base.sh`.
- Wire `@/bin/bash tools/launchd/test_mission_base.sh` into `make test-launchd-drivers`; add `/bin/bash -n tools/launchd/mission-base.sh` to the syntax loop.
- **Boundary tests (must pass):**
  - `snap` prints `<40-hex-sha><TAB><ISO8601-UTC>` for `origin/dev` and returns 0.
  - `record gate1` appends exactly one `base-gate1` row to a temp `MISSION_NAME=test` **`mission-test-base`** file; `last gate1` returns that SHA; the temp **`mission-test-heartbeat` stays empty**, proving the record stays off the heartbeat.
  - `drift` **non-vacuity**: in a scratch clone, `record gate1` at commit A; advance the shared ref to commit B (simulating a sibling fetch — see §Test story); `drift gate1` exits 1 and prints `DRIFT …A->B`. Positive/negative controls included (drift returns 0 when unchanged; the `last` grep fires on a known-matching record, proving the instrument runs). **No-record semantics (R2 fixes' own acceptance): `drift` exits 2 with `no base-… record yet` in BOTH cases — state file absent, and state file present with no matching `base-` row — never a false DRIFT.**
  - `/bin/bash -n` clean under bash 3.2.
- Commit per milestone (M1).

### M2 — Gate 1 + Gate 3 wiring + ratchet-clean prose (~1 day)
- `resources/gate-1-observe.md`: Gate 1 `base=…record gate1`; `resources/gate-3-route.md`: pre-`worktree add` re-read + drift + create-from-fresh; echo base into pick/evidence notes.
- **Create the war-story/rationale resource** (the two instances + "why silent" + the drift-on-mismatch protocol) at `resources/ref-drift.md`; link it from the gate resources that invoke the helper. Do not edit the root SKILL index.
- **Boundary:**
  - Root SKILL remains **560 lines** at the rebased start (and in all cases ≤ 2781); `make check-context-docs` exits 0 and all new links resolve.
  - S1–S5 guard literals still match (grep each; see Verif. 2).
  - `make test-launchd-drivers` green (the guard suite + M1 test).
- Commit.

### M3 — Gate 3b pin + Gate 4 evidence + live-iteration observation (~1 day)
- `resources/gate-3b-ci-green.md`: route the poll target through `mission-base.sh record gate3b` + drift note; `resources/gate-4-record.md`: reuse the existing re-confirmation, add `base=<sha>@<iso>` to the Routing-evidence row, and stamp `base-gate4`.
- Optionally stamp `base-gate2` (observation arm) after one live iteration proves its value.
- **Boundary:**
  - Root SKILL unchanged from the rebased start and ≤ 2781, `make check-context-docs` rc=0, new resource links resolve, S-guards green.
  - `make test-launchd-drivers` green.
  - **Non-vacuity mutation (repeat of M1's):** kill the guard by a simulated ref advance between Gate 1 and the worktree step (below) and show `drift` catches it — the exact mutation the reviewer asked the executor to be able to run.
- Commit.

### The non-vacuity mutation (how a later reviewer/executor kills and re-proves the guard)

In a scratch clone with no real sibling anywhere:

```bash
git clone --bare <this-repo> /tmp/base-fix-test.git 2>/dev/null   # or git init + a remote
# 1. Gate 1 record: at commit A
MISSION_NAME=test bash tools/launchd/mission-base.sh record gate1
# 2. Simulate the SIBLING advance (no fetch of ours): move the shared ref to a new commit B
# (glm R2 verbatim fix) — use `git update-ref` as the PRIMARY mutation, not `git commit
# --allow-empty`, since `snap` resolves refs/remotes/origin/dev, not the working-tree branch HEAD
git update-ref refs/remotes/origin/dev HEAD
# (in a scratch clone whose HEAD is commit B; a bare `git commit --allow-empty -m sibling-advance`
#  on a checked-out branch would NOT move the ref `snap` reads, so the mutation would be vacuous)
# 3. Now the Gate-3 re-read:
MISSION_NAME=test bash tools/launchd/mission-base.sh drift gate1
# MUST exit 1 and print DRIFT. The control (no step 2 → drift exits 0) proves the instrument fires.
```

This avoids depending on a real sibling by constructing the advance **in the test's own scratch clone**, the same way `test_mission_heartbeat.sh` fabricates its state. The test asserts the `drift` exit code and the `DRIFT` substring, so a future executor who deletes the re-read has a red the instant the guard is gone.

## Files to Modify/Create

- **Create** `tools/launchd/mission-base.sh` — measurement + record + drift helper (M1).
- **Create** `tools/launchd/test_mission_base.sh` — non-vacuity test (M1).
- **Create** `.claude/skills/mission-control/resources/ref-drift.md` — war-stories + drift protocol + test recipe (M2; pays the ratchet).
- **Modify** `make/test.mk` — add `@/bin/bash tools/launchd/test_mission_base.sh` and `/bin/bash -n` for the helper (M1).
- **Modify** `.claude/skills/mission-control/resources/gate-1-observe.md`, `gate-3-route.md`, `gate-3b-ci-green.md`, and `gate-4-record.md` — the four authoritative post-`c1212b3c` gate locations (M2, M3).
- **Do not modify** `.claude/skills/mission-control/SKILL.md` — it is now the 560-line index, and the gate bodies live in resources.
- **Not touched** `.agents/skills/mission-control/SKILL.md` (distinct 551-line stub; HD-5 — no sync mechanism, out of scope).

## Conflict Surface

This doc touches several shared mechanisms; each is enumerated with its reuse/override stance so a reviewer can check it deliberately.

1. **The four authoritative gate resources — shared by 4 missions (v1, docs, world, motoko).** The additions use only `$MISSION_NAME`/`$MISSION_BASE_REF`/`$AILANG_STATE_DIR`; no V1-specific literal is introduced. The base record is already per-mission by construction (`mission-${MISSION_NAME}-base`). Resource edits are live for every mission the moment they save — the change must and does hold for all 4 (it is *more* correct for them, since they share the same `.git`). Override: none — additive + parameterised.
2. **`tools/launchd/test_mission_routing.sh` guard suite (S1–S5).** Not modified. The new prose must not move or reword the S1–S5 literals (S1 `resolve-role-spawn.sh`+`MISSION-ROLE:`, S2 `enum in this build lists`, S3 the designer-rotation string, S5 `ASTRA IS ALSO A QUORUM REVIEWER`). The new helper/Gate-1/3/3b/4 edits are in entirely different regions (Gate 1 sync, Gate 3 worktree, Gate 3b poll, Gate 4 record) and touch none of them. Verification gate: re-grep all four literals after M2/M3.
3. **`scripts/check_context_docs.sh` / progressive-disclosure links.** The design **never bumps the historical 2,781-line baseline** and no longer adds root-SKILL lines. The root is 560 lines on current `origin/dev`; gate prose belongs in the existing authoritative resources, and every new `ref-drift.md` link must resolve. Verification: `make check-context-docs` rc=0.
4. **`tools/launchd/mission-control.sh` (driver) + `tools/launchd/lib/pin-root.sh`.** Reuse (not duplicate): `mission-base.sh` reuses `$REPO`/`$MISSION_NAME`/`$AILANG_STATE_DIR` conventions and the driver's pin-banner SHA as a cross-check; it does **not** merge with the pin/re-exec machinery (different concern). Not modified to make this land.
5. **`tools/launchd/mission-heartbeat.sh` + `test_mission_heartbeat.sh`.** **Not consumed by this doc** — the base rows go to the separate `mission-${MISSION_NAME}-base` file, so the heartbeat's gate-label set and its whitelist (lines 11–14) are entirely untouched. `test_mission_heartbeat.sh` (§8 "every gate section stamps", the sigkill `tail -1` last-label arm, the verdict arms) is unaffected because the heartbeat never sees a `base-<label>` row.
6. **`tools/launchd/mission-control.sh` slot-verdict reader.** The driver classifies every slot from the heartbeat's **last row**: `_mc_slot_line=$(tail -1 …heartbeat)` (1486), `_mc_slot_last=$(…awk -F '\t' '{print $3}')` (1491), `case "$RC:$_mc_slot_last"` (1492), where `0:gate-*→REAPED` (1496) and `*:*→CRASHED` (1498), plus `wc -l` stamp count (1485). A trailing `base-*` row at a quiet rc=0 death flips REAPED→CRASHED, degrades the crash-site `at=` label, and inflates `stamps=` — this attribution is the loop's died-mid-flight trace (used this morning to locate iteration 337's crash). **The separate base file is the fix: the heartbeat gains no rows, so this reader is never misclassified.** The retry-resume reader (1461, `tail -1` last label) shares the sensitivity and is likewise protected. Override: none — the base file is a newly-created, unread state key.
7. **CI `launchd-drivers (bash 3.2)` job (`.github/workflows/ci.yml`)** — runs `make test-launchd-drivers`, now including `test_mission_base.sh`, on macOS bash 3.2. Ubuntu/Windows legs run bash 5 / PowerShell; the helper is bash-3.2-safe and portable, so it degrades gracefully everywhere (Verif. 3, 9).

## Success Criteria

Measurable, one per goal:

1. **Re-read before action:** a reviewer can point at the code in each of Gate 1/2/3/3b/4 that re-reads `origin/dev` (or a recorded `base-` value) immediately before the base-dependent act. (Satisfied by the snippets above.)
2. **Record SHA+time:** the dedicated `mission-${MISSION_NAME}-base` file contains `base-gate1`/`base-gate3b`/`base-gate4` rows with `<full-sha>` in the `note` field, the heartbeat contains **no** `base-*` rows, and the iteration's Gate-4 Routing-evidence row carries `base=<sha>@<iso>`.
3. **Drift not error:** `mission-base.sh drift` exits 0 on steady, 1 with a `DRIFT base <label> old -> new (n commits)` message on disagreement, and a live/fabricated advance never reads as an operator error.
4. **Measured, not prose-only:** `tools/launchd/test_mission_base.sh` runs in `make test-launchd-drivers`, and the **non-vacuity mutation** (scratch-clone ref advance between Gate 1 and the worktree step) turns the guard red — an executor can run it and show the fix detects it, with a positive control (`last`/`drift` steady) proving the instrument fires.
5. **Context/S-guard clean:** root SKILL remains unchanged from the rebased start and ≤ 2781; `make check-context-docs` rc=0; every new resource link resolves; all four S-guard literals still grep-match in their current authoritative files; `make test-launchd-drivers` green.

## Verification Log

Every codebase claim below was measured in this worktree (commit `034cdd02f`, branch `docs/v1-iter338-gate1-ref-drift`). For each empty/negative result a known-positive control is given in the same row.

1. **SKILL.md line count vs the ratchet baseline entry.**
   `wc -l .claude/skills/mission-control/SKILL.md` → **2781**. `grep -n "mission-control/SKILL.md" scripts/context_docs_baseline.txt` → **line 19: `.claude/skills/mission-control/SKILL.md     2781`**. Both match; the ratchet pins at EXACTLY 2781, so any growth fails.
2. **Four guard literals present.** `grep -n` each (from `tools/launchd/test_mission_routing.sh` S1–S5):
   - `grep -n "resolve-role-spawn.sh"` → line 1100; `grep -n "MISSION-ROLE:"` → line 1107.
   - `grep -n "enum in this build lists"` → line 1118.
   - `grep -nF "pi:ollama/deepseek-v4-flash:0731-cloud"` → 2 hits, incl. the S3 rotation string; `grep -cF "now \`claude:claude-fable-5-1\`"` → 1; `grep -cF "codex:gpt-6-astra"` → 1; `grep -cF "repeat"` → 8 (line-wrapped; the full S3 literal `now \`claude:claude-fable-5-1\` → \`codex:gpt-6-astra\` → \`pi:ollama/deepseek-v4-flash:0731-cloud\` → repeat` matches).
   - `grep -n "ASTRA IS ALSO A QUORUM REVIEWER"` → line 1059 (inside the S3/S5 region). All four literals confirmed inline.
3. **Gate-1 opening snippet exists as quoted.** `grep -n "git fetch origin"` → lines 446, 794, 2199. `grep -n "rev-parse dev origin/dev"` → line 447 (`# differ? origin is ground truth. NO --short: see below`) and line 2200. The Gate-1 opening at 446–447 matches the problem statement literally.
4. **What `mission-heartbeat.sh` writes and where.** `grep -n` in `tools/launchd/mission-heartbeat.sh` → `printf '%s\t%s\t%s\t%s\t%s\n' "$epoch" "$iso" "$label" "$attempt" "$note" >> "$heartbeat"` and `heartbeat="$state_dir/mission-${MISSION_NAME}-heartbeat"` with `epoch=$(date +%s)`, `iso=$(date -u '+%Y-%m-%dT%H:%M:%SZ')`, `state_dir="${AILANG_STATE_DIR:-$HOME/.ailang/state}"`. The 5-column tab format is exact — `mission-base.sh` uses it for the separate `mission-${MISSION_NAME}-base` file.
5. **Driver already prints/records the pinned SHA.** `grep -n` in `tools/launchd/mission-control.sh` → `log "driver pin: $PIN_NOTE"` (with `PIN_NOTE="running committed ${AILANG_DRIVER_REF:-origin/dev} @ ${AILANG_DRIVER_PINNED}"` set by `tools/launchd/lib/pin-root.sh` `pin_root_to_committed_ref`), and the STALE branch logs `…rev-parse --short HEAD…` + a behind-count via `rev-list --count HEAD..origin/dev`. The driver pins the *code it runs*; the new helper records the *base it reasons about* — reused/not duplicated (HD-3-related, Solution Design "driver-level scripts").
6. **`.agents/skills/mission-control/SKILL.md` relationship.** `wc -l .agents/skills/mission-control/SKILL.md` → **551** (a distinct condensed stub) vs `.claude` **2781** (the operational skill). `diff` of the first 30 lines shows the `.agents` copy diverges at line 3 (frontmatter) and carries its own prose; both files are tracked by git. `cat tools/skill-updates/install-skill.sh` → installs **only `ailang-feedback`** (`DEST_DIR="$HOME/.claude/skills/ailang-feedback"`), NOT mission-control. **No sync/copy mechanism exists for the mission-control sibling.** Positive control: `grep -c "Gate 1\|Gate 2\|Gate 4\|Gate 3b\|worktree"` → 33 in `.agents` stub vs 141 in `.claude`, confirming they are different documents (the stub is a summary, not a copy).
7. **Gate-4 base-reconfirmation snippet current wording.** SKILL.md lines 2196–2200 (read directly): "Before the first Gate-4 write, re-confirm the base: `git fetch origin / git rev-parse dev origin/dev # differ at all? working tree is NOT the base / git diff --stat origin/dev -- "$MISSION_DOC" design_docs/*-mission-log.md`" and the "cheap tell … grep for the PREVIOUS iteration's stamp." My design **reuses** this re-confirmation and adds the record + Routing-evidence `base=` field; it does not remove the fetch/rev-parse.
8. **`make check-context-docs` current exit status.** Ran in this worktree → prints `✓ context docs: 12 rules, 40 skills, CLAUDE.md — scoped, linked, within budget`, **rc=0**. The gate is green today; the design must keep it green (`make check-context-docs` is wired into CI at `.github/workflows/ci.yml:223`).
9. **How `test_mission_routing.sh` arms are covered/invoked.** `make/test.mk` `test-launchd-drivers` target runs `@/bin/bash tools/launchd/test_mission_routing.sh` (and its suite siblings). `.github/workflows/ci.yml` job `launchd-drivers` (line 582) runs on `macos-latest`, asserts `/bin/bash` is 3.x, then runs `make test-launchd-drivers` (`/bin/bash` explicitly = rig bash 3.2). So the suite is CI-covered on the rig's actual shell; adding `test_mission_base.sh` to the same target gives it the same coverage.
10. **Driver's slot-verdict reader — why the base record must NOT be the heartbeat (controller's measured, load-bearing premise).** `grep -n '_mc_slot_last\|REAPED\|CRASHED\|_mc_slot_stamps' tools/launchd/mission-control.sh` → line 1485 `_mc_slot_stamps=$(wc -l < "$_mc_slot_hb" …)` (stamp count); line 1486 `_mc_slot_line=$(tail -1 "$_mc_slot_hb" …)` (**last row** is the verdict input); line 1491 `_mc_slot_last=$(printf '%s\n' "$_mc_slot_line" | awk -F '\t' '{print $3}')`; lines 1493–1498 `case "$RC:$_mc_slot_last"` with `0:complete→COMPLETED`, `0:abort→ABORTED`, `0:fired|0:→DIED-PRE-GATE-0`, `0:gate-*→REAPED at=…`, `*:*→CRASHED at=…`. A trailing `base-*` row at a quiet rc=0 death hits `*:*` → **CRASHED** (not REAPED), the crash-site `at=` label degrades to `base-<n>`, and `wc -l` (1485) inflates `stamps=`. This attribution is the loop's died-mid-flight trace (used this morning to locate iteration 337's crash). **The separate base file keeps the heartbeat label set `fired|gate-*|complete|abort`-only, so this reader is never misclassified.** (Re-grepped this worktree; matches the controller's prior reading.)
11. **(Premise a) Gate 3b already pins its poll target via the full SHA.** `grep -n 'target=$(git rev-parse origin/dev)' .claude/skills/mission-control/SKILL.md` → **line 1742: `target=$(git rev-parse origin/dev)            # FULL sha; no `--short` (Gate 1's rev-parse lesson)`**. The design therefore only ADDS the `record gate3b` + drift note around an existing SHA-pin; it does not introduce SHA-pinning. (Controller's reading: line 1742; re-run and confirmed in this worktree.)
12. **(Premise b) Gate 2 already re-fetches at pick time.** `grep -n 'immediately before the item-level check' .claude/skills/mission-control/SKILL.md` → **line 794: `` `git fetch origin` immediately before the item-level check and grep `git log origin/dev --grep` ``**. Gate 2's re-read is therefore pre-existing; the doc only adds a snap-for-the-record there, not a new re-fetch. (Controller's reading: line 794; re-run and confirmed in this worktree.)
13. **Heartbeat consumers audit (`grep -rn 'heartbeat' tools/launchd/*.sh`) — one row per consumer, label-sensitivity classified.**
    - `mission-control.sh:1480–1502` — **SLOT-VERDICT** reader (LAST row → label → `case` verdict; `wc -l` stamps). **LABEL-SENSITIVE** (expects `fired|gate-*|complete|abort`; a `base-*` last row flips REAPED→CRASHED). Protected by the separate base file
    - `mission-control.sh:1461–1467` — **RETRY-RESUME** reader (`tail -1` last label to detect a resumed slot; `wc -l < heartbeat` into retry history). **LABEL-SENSITIVE** + stamp-count-sensitive. Protected by the separate base file.
    - `mission-control.sh:230` — **LIVENESS** arm `hb=$(wc -c < …heartbeat)` byte-size growth. **LABEL-INSENSITIVE** (any appended row grows the file). Untouched.
    - `mission-control.sh:1380` — **PRODUCER** (writes the `fired` first row). Not a consumer.
    - `mission-heartbeat.sh:11–28` — **PRODUCER** helper; label whitelist `fired|gate-0|…|complete|abort`, unknown → exit 2. **LABEL-SENSITIVE**: would reject `base-*`, which is why base rows use a separate file.
    - `test_mission_heartbeat.sh` — **TEST FIXTURE**: asserts last-label on sigkill (§34), verdict arms (COMPLETED/ABORTED), "every gate section stamps" (§8). **LABEL-SENSITIVE**; unaffected because base rows are off-heartbeat.
    - `test_mission_stall.sh:87–216` — **STALL TEST**: a gate stamp counts as progress (`grow … gate-3`, §127). **LABEL-INSENSITIVE** to extra rows; unaffected.
    Net: no consumer wants `base-*` rows on the heartbeat; the separate-file design serves every one.

14. **Recovery ref sequence and ancestry.** `git reflog show --date=iso refs/remotes/origin/dev -12` records `c1212b3c` at 12:02, `b5a5c45b` at 12:17, and `bdc5fbf7` at 12:26 local time. `git merge-base --is-ancestor 034cdd02f c1212b3c`, then `c1212b3c b5a5c45b`, then `b5a5c45b bdc5fbf7` each exit **0**. The iteration branch's design base is therefore a strict ancestor of all three observed readings; this is monotonic shared-ref drift, not a rewrite.
15. **The upstream drift changed the implementation file map.** `git show origin/dev:.claude/skills/mission-control/SKILL.md | wc -l` → **560**, while this pre-rebase branch's committed root skill is 2,781 lines (2,770 after the uncommitted M2 attempt). `git cat-file -e origin/dev:.claude/skills/mission-control/resources/{gate-1-observe,gate-3-route,gate-3b-ci-green,gate-4-record}.md` exits **0** for all four; same-scope negative control: those four files do not exist at branch `HEAD`, while positive control `.claude/skills/mission-control/SKILL.md` exists at both refs. `git show origin/dev:.claude/skills/mission-control/SKILL.md` identifies each resource as the authoritative full rules for its gate. Therefore the old monolith-targeted M2 diff cannot be carried forward as-is.
16. **The reviewed artifacts themselves did not drift.** `git diff --name-only 034cdd02f..bdc5fbf7 | rg 'm-gate1-shared-clone-ref-drift|iter338-gate1-ref-drift-quorum'` is empty; same-scope positive control finds `.claude/skills/mission-control/SKILL.md` in that diff. SHA-256 comparison of the design and both committed receipts against commit `7870f40a0` matched byte-for-byte before this recovery amendment; `git status --short` showed no owned-path changes. The receipt JSONs remain immutable records of R1/R2 and are not rewritten by this amendment.

R1 revision note: rows 10–13 were added/measured during this revision pass with the controller's prior readings cited where re-run here (lines 1742, 794, 1480–1502).

*(Additional first-party facts above, all measured: shared `.git` shown by `git rev-parse --git-dir` → `<shared>/.git/worktrees/-wt-v1-iter338`; `/bin/bash --version` → GNU bash 3.2.57; `git ls-files` confirms both `.claude` and `.agents` mission-control SKILL.md are tracked; no existing `m-…ref-drift` doc in `design_docs/planned/v0_36_0/`.)*

---
**R1 revision summary (2026-09-06, controller-measured):**
1. Durable record moved from the heartbeat to a dedicated per-mission `mission-${MISSION_NAME}-base` file — the driver's slot-verdict reader classifies from the heartbeat's LAST row, so a trailing `base-*` row would flip REAPED→CRASHED (Verif. 10; Conflict Surface 6; HD-2; Goals 2; Success 2).
2. `record()` now snaps EXACTLY once (single-read invariant: the SHA recorded is the SHA of exactly one rev-parse) — fixes the glm double-snap race (Solution Design; helper snippet).
3. Corrected the false "whitelist already accepts arbitrary strings" claim: the whitelist applies to the LABEL field; written rows now bypass it via the separate file (Relations; Conflict Surface 5).
4. Added Verification Log rows 10–13 (driver slot-verdict 1480–1502, premises a=1742/b=794, full heartbeat-consumer audit) and a new Conflict Surface row 6.
5. Doc stays sprint-sized (3 milestones, one helper + one test) and ratchet-clean — heartbeat, its whitelist, and the S-guard suite untouched.

DESIGN_DOC_PATH: design_docs/planned/v0_36_0/m-gate1-shared-clone-ref-drift.md
