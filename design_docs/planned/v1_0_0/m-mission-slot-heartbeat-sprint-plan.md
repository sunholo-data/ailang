# Sprint Plan — M-MISSION-SLOT-HEARTBEAT

**Design doc**: [`design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md`](m-mission-slot-heartbeat.md)
(564 lines; V1–V28, Round 2 + Round 3 carve-out, residuals R1–R8)
**Mandate**: ratified ruling **D-52** (Mark Edmondson, attended 2026-09-01) — instrument the mission
driver so a slot that dies mid-flight *names the gate it died in*, and so the reaped-slot rate is
derivable without a hand audit.
**Planned**: 2026-09-01, V1 mission iteration 315, sprint-planner role.
**Duration**: **3 days**, 3 milestones. **Risk**: medium (surgical edits into a 67,759-byte live
driver that fires on a 5400 s launchd timer).
**Surface**: shell only. Zero Go, zero language/parser/typechecker/codegen surface.

---

## 0. What this plan is, and the discipline it was built under

Every number, path and line reference below was **re-derived first-party in the planning session**,
in `/Users/voightkampff/.ailang-driver-pin/v1` at `878e0a5a0`, never transcribed from the design
doc's prose. Negative results are paired with a known-positive control **in the same call and the
same scope**. Where the doc turned out to be wrong, §6 says so with the command that shows it.

The doc's own Round-3 closing note is the instruction this plan follows:

> "The line anchors here are evidence of where the seams are, not an instruction to patch by line
> number — the sprint should key off the `# --- SLOT VERDICT START/END ---` markers and the named
> functions."

Accordingly **no acceptance criterion in this plan asserts a line number as a constant.** The two
ordering criteria (A2.4, A2.5) compare line numbers *computed at check time* against anchors located
by their own text — which is exactly the property Rounds 1 and 2 both got wrong, now mechanized.

---

## 1. Baselined gate list — every gate, with its observed result on untouched `dev`

An AC nobody ran at base is not an AC. Each row below was executed in this session at
`878e0a5a0`. **RED at base** = the criterion can fail and currently does, so passing it is
information. **GREEN at base** = a regression gate; it must stay green.

### 1.1 The standing regression gate (all milestones)

| Gate | Command | Base result |
|---|---|---|
| **G0** | `PATH="/usr/sbin:$PATH" make test-launchd-drivers` | **rc=0, 0 `not ok` lines** — GREEN. Last lines: `ok 41 - refusal-branch count still matches the set this suite covers (24)` / `PASS: 41 probe self-test arms ran` / `launchd drivers: tests + bash 3.2 syntax OK` |
| **G0-neg** | `make test-launchd-drivers` (ambient PATH, no prefix) | **rc=2**, 1 `not ok`: `not ok - run_lane fixture arm requires real lsof on Darwin CI target` |

**The `PATH="/usr/sbin:$PATH"` prefix is mandatory and is part of the criterion.** `lsof` lives at
`/usr/sbin/lsof` (`-rwxr-xr-x root wheel 307600`) and `command -v lsof` on this shell prints nothing.
A criterion that names `make test-launchd-drivers` without the prefix is **red at base** and
therefore broken — it would report a pre-existing environment fact as a sprint failure. GitHub's
`macos-latest` runner has `/usr/sbin` on PATH, which is why CI is green today and the rig shell is
not; do not "fix" the test to accommodate the ambient PATH.

Two further `make ci` gates that this sprint's file list can plausibly break, both baselined:

| Gate | Command | Base result |
|---|---|---|
| **G1** | `make check-changelog` | **rc=0** — `✓ CHANGELOG.md is index-only and links changelogs/v0.32-current.md`. See §6 refutation R-1: writing to root `CHANGELOG.md` turns this **rc=2**. |
| **G2** | `make check-skills` | **rc=0** — `✓ all 39 skills have frontmatter with a matching name and a description` (the SKILL.md edit must not disturb frontmatter) |
| **G3** | `make check-tmpfile-hygiene` | **rc=0** — `✓ tmpfile hygiene holds across 12 make files` (the new `make/test.mk` line must carry **no fixed `/tmp` path**) |

`go build ./...` is **not** in any gate list — it is rc=1 on pristine `dev` (`cmd/wasm`, `gen/main`
have no native `main`), and this sprint is shell-only regardless.

### 1.2 Milestone acceptance criteria, baselined

Every one of these was run at base. All are **RED**, and each names a discriminating condition, not
a file's existence.

**M1 — helper**

| AC | Command | Base result |
|---|---|---|
| **A1.1** | `/bin/bash tools/launchd/test_mission_heartbeat.sh` | **rc=127** (file absent) — RED |
| **A1.2** | `grep -q 'test_mission_heartbeat.sh' make/test.mk` | **rc=1** — RED |
| **A1.3** | test output contains `mutations: 2/2 killed` | RED (no output at base) |
| **A1.4** | test output contains **all three** of `PASS: sigkill mid-gate-1 leaves last label gate-1`, `PASS: v1 and world stamps land in distinct files`, `PASS: MISSION_NAME unset writes no file` | RED |
| **A1.5** | `[ -s tools/launchd/mission-heartbeat.sh ] && ! grep -qE 'declare -A\|mapfile\|readarray\|\$\{[A-Za-z_]+,,\}' tools/launchd/mission-heartbeat.sh` | **rc=1** — RED. (Written as `[ -s F ] && ! grep …` deliberately: a bare `! grep … F` on a *missing* file returns rc=0 and would pass vacuously.) |
| **A1.6** | test output contains `PASS: driver and helper resolve the same artifact path` | RED — planner-added, see §7 seam S-1 |

**M2 — driver**

| AC | Command | Base result |
|---|---|---|
| **A2.1** | test output contains `mutations: 7/7 killed` | RED |
| **A2.2** | test output contains `PASS: sigkill mid-gate-1 => REAPED at=gate-1` | RED |
| **A2.3** | `awk '/^# --- SLOT VERDICT START ---/,/^# --- SLOT VERDICT END ---/' tools/launchd/mission-control.sh \| grep -c .` ≥ 10 | **0** — RED. *Same-call positive control*: the identical awk over the file's existing markers, `/^# --- DRIVER PIN DECISION START ---/,/^# --- DRIVER PIN DECISION END ---/`, prints **62** at base, so the extraction instrument demonstrably sees a real block. |
| **A2.4** | END-marker line **<** guard-release line: `E=$(grep -n '^# --- SLOT VERDICT END ---' … \| cut -d: -f1); R=$(grep -n 'rm -f "$PIDFILE"   # this instance owns' … \| cut -d: -f1); [ -n "$E" ] && [ -n "$R" ] && [ "$E" -lt "$R" ]` | **RED** — `E` is empty at base; `R`=**1037**. This is Round-3 objection 1 mechanized. |
| **A2.5** | START-marker line **>** the retry-loop call line: `S=$(grep -n '^# --- SLOT VERDICT START ---' …); C=$(grep -n '^  _mc_run_once; RC=\$?' …); [ -n "$S" ] && [ "$S" -gt "$C" ]` | **RED** — `S` empty at base; `C`=**1023** |
| **A2.6** | `awk '/^_mc_run_once\(\) \{/,/^\}/' tools/launchd/mission-control.sh \| grep -c 'mission-heartbeat'` ≥ 1 | **0** — RED. *Same-call positive control*: that awk range prints **56** total lines at base, so the range is non-empty and the zero is a real zero, not an extraction failure. |
| **A2.7** | test output contains `PASS: 250 appends leave exactly 200 lines` | RED |
| **A2.8** | test output contains `PASS: superseded attempt leaves verdict=RETRIED row` and `PASS: attempt 2 pre-gate-0 => DIED-PRE-GATE-0 attempt=2` | RED |

**M3 — skill + release note**

| AC | Command | Base result |
|---|---|---|
| **A3.1** | `grep -c 'mission-heartbeat.sh stamp' .claude/skills/mission-control/SKILL.md` ≥ 9 | **0** — RED. *Same-call positive control*: `grep -c 'tools/launchd' SKILL.md` = **7**, so the file is readable and the grep works. |
| **A3.2** | test output contains `PASS: every gate section carries its own stamp instruction (8/8)` — an arm that locates each `^## Gate` heading, takes the span to the next heading, and asserts the matching `stamp <label>` falls **inside** that span | RED |
| **A3.3** | `grep -c 'slot-verdict\|mission-heartbeat' changelogs/v0.32-current.md` ≥ 1 | **0** — RED |
| **A3.4** | `make check-changelog` still rc=0 | GREEN at base, must stay green (see R-1) |

**Post-merge, controller-owned (NOT executor ACs — see §5)**

| AC | Command | Base result |
|---|---|---|
| **A4.1** | `grep -c 'slot-verdict' /tmp/ailang-mission-control.log` ≥ 1 | **0** — RED. *Same-call positive control*: `grep -c 'iteration complete (rc=0)'` on the same 13,198,267-byte file = **68**, so the log exists, is readable, and the zero is a real zero. |
| **A4.2** | the Q5 `awk` one-liner over `~/.ailang/state/mission-v1-slot-verdicts.log` prints ≥ 1 `verdict=` line | RED — `ls ~/.ailang/state \| grep -cE 'mission-.*-heartbeat$\|slot-verdicts'` = **0** with `test -d ~/.ailang/state` asserting the directory exists (72 entries) and `grep -cE 'heartbeat'` = **1** (`worker_heartbeats.json`, the coordinator's unrelated file) as the positive control that the pattern can match something |

---

## 2. Milestones

Three milestones, each **independently committable and bisectable**, each with its own gate list.
A checkout at the tip of M1 must be green on M1's gates without any part of M2 present; likewise M2
without M3.

### M1 — `mission-heartbeat.sh` + its suite, wired into the guarded target (Day 1, ~200 LOC)

**Scope**: the helper alone. No `mission-control.sh` edit in this milestone.

**Why the helper first, and why it is a real bisect point.** The headline physical claim of the whole
design — *a stamp written by `printf >>` at the moment a gate is crossed survives a SIGKILL of the
writer* — is provable with **zero driver changes**. M1 proves it. If M1's SIGKILL arm does not go
green, the mechanism is wrong and no amount of driver work rescues it.

**Files**

| File | Change | ~LOC |
|---|---|---|
| `tools/launchd/mission-heartbeat.sh` | NEW. `stamp <label> [note]`; `MISSION_NAME` resolution with visible no-op when unset; `AILANG_STATE_DIR` override; closed-set label validation via `case` (unknown label → exit 2); `${MISSION_ATTEMPT:-1}`; single `printf … >>` per stamp | ~90 |
| `tools/launchd/test_mission_heartbeat.sh` | NEW. Helper arms + SIGKILL survival + 2 mutation arms + path-agreement arm | ~110 |
| `make/test.mk` | +1 line inside the `test-launchd-drivers` recipe | 1 |

**Implementation notes the executor must honour**

- **bash 3.2 only.** `/bin/bash --version` here is `GNU bash, version 3.2.57(1)-release
  (arm64-apple-darwin25)` — measured, not inherited. No `declare -A`, no `${v,,}`, no `mapfile`,
  no `readarray`. A1.5 refuses these mechanically; the `launchd-drivers` CI job
  (`.github/workflows/ci.yml:535`, `runs-on: macos-latest`) asserts `[ "${BASH_VERSINFO[0]}" -eq 3 ]`
  before running `make test-launchd-drivers`, so the constraint is enforced remotely too.
- The `test-launchd-drivers` recipe already ends with
  `for f in tools/launchd/*.sh tools/launchd/lib/*.sh; do /bin/bash -n "$f" || exit 1; done`, so
  **both new files inherit `bash -n` syntax checking for free** the moment they land in that
  directory — no extra wiring needed for that part.
- The new recipe line must carry **no fixed `/tmp` path** (G3, `make check-tmpfile-hygiene`); use
  `${TMPDIR:-/tmp}`/`mktemp -d` **inside the test script**, which the hygiene gate does not scan.
- Follow `tools/launchd/test_driver_notify.sh`'s convention verbatim (its own header states it):
  *"The blocks are awk-extracted from the file, never retyped: a retyped copy tests the copy"*, an
  empty extraction is a **FATAL**, and each arm gets a **fresh `mktemp -d` STATE_DIR** so one arm's
  episode marker cannot silently dedupe another's.

**Mutation arms in M1**

| Mutation | Edit applied to the extracted copy | Named arm that must go red | Expected blast radius |
|---|---|---|---|
| `MUT-BUFFERED` | helper accumulates stamps in a shell variable and writes them in an `EXIT` trap instead of appending per stamp | `sigkill mid-gate-1 leaves last label gate-1` | **SOLE** expected. SIGKILL runs no traps, so the artifact is empty at kill time. |
| `MUT-SHARED-PATH` | drop `${MISSION_NAME}` from the artifact path | `v1 and world stamps land in distinct files` | **WIDE (2)** — also reds `MISSION_NAME unset writes no file`. Criterion is membership, see below. |

**Universal mutation criterion.** For every mutation the assertion is *"the named arm appears in the
red set of the mutated run"* — **never** `rc=0` on a `-skip` run and never "exactly one arm dies".
A mutant that reds several arms is fine and expected; a mutant that reds **zero** arms is a broken
test and the suite must report it as a FAIL (`mutation MUT-X SURVIVED`), not a warning.

**M1 gates**: G0, G3, A1.1, A1.2, A1.3 (`mutations: 2/2 killed`), A1.4, A1.5, A1.6.

---

### M2 — driver: per-attempt `fired`, the marked verdict block, bounded history (Day 2, ~210 LOC)

**Scope**: `mission-control.sh` + the five driver-side mutation arms + the SIGKILL→verdict
end-to-end arm.

**Files**

| File | Change | ~LOC |
|---|---|---|
| `tools/launchd/mission-control.sh` | Inside `_mc_run_once()`, immediately before the provider launch: export `MISSION_ATTEMPT` and write the `fired` stamp with `>` (truncate, **per attempt**). In the retry path, before `continue`: append one `verdict=RETRIED at=<last> rc=$RC attempt=N` history row. On the common exit path, **after the retry loop's `done` and before the guard release**: the `# --- SLOT VERDICT START ---` / `# --- SLOT VERDICT END ---` phase-1 block (classify, `slot-verdict:` log line, bounded history append). Phase-2 episode-gated notices **after** the guard release, routed through the existing `_mc_bounded()` | ~70 |
| `tools/launchd/test_mission_heartbeat.sh` | +5 mutation arms, reap→verdict arm, bound arm, retry/attempt arm | ~140 |

**The three placement facts, re-derived**

1. `rm -f "$PIDFILE"   # this instance owns the run; yield paths above never reach here` is at
   **line 1037**, on the **common** path. The rc split `if [ "$RC" -ne 0 ]; then` is at **line 1039**.
   So a block cannot be both *inside an rc branch* and *before the guard release* — Round-3
   objection 1, confirmed here by direct read. **A2.4 mechanizes it.**
2. `_mc_run_once; RC=$?` is at **line 1023**, inside the retry `while : ; do` loop whose `done` is at
   **line 1035**. The verdict block must start after 1035. **A2.5 mechanizes it** by comparing
   against the located call line, not against a hard-coded 1035.
3. `_mc_run_once()` is defined at **line 950** and called from exactly **one** site
   (`grep -n '_mc_run_once'` → 3 hits: a comment at 948, the definition, the call). Putting the
   `fired` write inside it is therefore per-attempt by construction, matching the pidfile refresh at
   line 974 (its own comment: *"per-attempt: retries refresh it"*) and the watchdogs at line 949.
   **A2.6 mechanizes it** by extracting the function body and requiring the reference inside it.

**Hazards specific to this file**

- **`set -uo pipefail` is on** (line 38). Any variable the new block reads must be bound on every
  path, and the extracted-block tests must bind them too or `set -u` aborts the arm — the existing
  `test_driver_notify.sh` already carries a comment about exactly this trap
  (`STATE_DIR="$5"   # episode gating (_lane_ep/_pin_ep) reads this; unbound => set -u abort`).
- **Do not introduce a line beginning with `}` at column 0 inside `_mc_run_once`** — A2.6's awk range
  `/^_mc_run_once\(\) \{/,/^\}/` would terminate early and silently shrink the tested region.
- **Phase 1 must not touch the network** while the overlap guard is held: local `tail`/`awk` reads and
  `O_APPEND` writes only. No `ailang messages send`, no `gh`, no `git`. Phase 2 (after line 1037)
  goes through `_mc_bounded()`, which is defined at **line 194** and already has **8** call sites
  (negative control `_mc_never_915` → 0), with `deadline=$(( $(date +%s) + secs ))`,
  `kill` → `sleep 2` → `kill -9`, return `124` on timeout, and `( exec "$@" )` so the kill reaches the
  real process rather than a subshell.
- **`MISSION_DRY_RUN=1` exits at line 785**, well before the retry loop, so the dry-run path writes no
  stamp and reaches no verdict block. No dry-run behaviour change is expected — and the executor must
  **not** run the real driver to check this: it fires live model probes and its overlap guard
  interacts with the running V1 loop. Prove it statically (dry-run `exit 0` precedes the START marker
  line) instead.

**Mutation arms in M2**

| Mutation | Edit applied to the extracted copy | Named arm that must go red | Expected blast radius |
|---|---|---|---|
| `MUT-DEFAULT-COMPLETE` | verdict `case` falls through to `COMPLETED` | `empty-stamps => DIED-PRE-GATE-0` | **WIDE (≥3)** — also reds the missing-artifact and gate-N arms. Membership criterion. |
| `MUT-MISSING-IS-PASS` | absent artifact classified `COMPLETED` | `deleted artifact => HEARTBEAT-MISSING` | **SOLE** expected |
| `MUT-UNSTAMPED-IS-PASS` | drop the `complete`-stamp requirement; accept any last stamp on rc=0 | `rc=0 last=gate-5 => REAPED at=gate-5` | **WIDE (2)** — also reds `sigkill mid-gate-1 => REAPED at=gate-1`. Membership criterion. |
| `MUT-UNBOUNDED` | remove the `tail -n 200` trim | `250 appends leave exactly 200 lines` | **SOLE** expected |
| `MUT-STALE-ATTEMPT` | move the `fired` truncation out of `_mc_run_once` (back to per-fire) | `attempt 2 pre-gate-0 => DIED-PRE-GATE-0 attempt=2` | **SOLE** expected |

**M2 gates**: G0, A2.1 (`mutations: 7/7 killed`), A2.2, A2.3, A2.4, A2.5, A2.6, A2.7, A2.8 — plus
all M1 ACs, which must still pass.

---

### M3 — SKILL.md stamp contract + release note (Day 3, ~60 LOC)

**Scope**: the controller-facing half, plus the one thing that keeps it from rotting.

**Files**

| File | Change | ~LOC |
|---|---|---|
| `.claude/skills/mission-control/SKILL.md` | One `bash tools/launchd/mission-heartbeat.sh stamp gate-N` instruction as the first action of each gate section; `stamp complete` at the end of Gate 5; `stamp abort <reason>` on the Gate-0 abort path; a short Standing-rule-7 note. **No literal state path and no mission name in the skill text** — the helper derives both from env | ~20 |
| `tools/launchd/test_mission_heartbeat.sh` | +1 arm: the SKILL.md span contract (A3.2) | ~30 |
| `changelogs/v0.32-current.md` | release note — **not** root `CHANGELOG.md`, see R-1 | ~10 |

**The eight gate sections, located first-party** (`grep -n '^## Gate' .claude/skills/mission-control/SKILL.md`,
file is 3,987 lines):

| Heading line | Section | Stamp label |
|---|---|---|
| 200 | `## Gate 0 — PREFLIGHT (deterministic; abort = exit silently with a controlplane message)` | `gate-0`, plus `abort <reason>` on the abort path |
| 417 | `## Gate 1 — OBSERVE (cheap, read-only)` | `gate-1` |
| 652 | `## Gate 2 — PICK + REALITY-CHECK` | `gate-2` |
| 2328 | `## Gate 3 — ROUTE + EXECUTE (the inner loop, with the routing policy)` | `gate-3` |
| 3011 | `## Gate 3b — CI GREEN (an item is not LANDED until remote CI passes on its merge)` | `gate-3b` |
| 3348 | `## Gate 4 — RECORD (append-only; the log is the mission's memory)` | `gate-4` |
| 3610 | `## Gate 5 — RETRO + REPORT` | `gate-5`, plus `complete` at its end |

The Gate-0 heading text already contains *"abort = exit silently with a controlplane message"*, which
is the designed path the doc's residual **R3** depends on — so the `abort` stamp has a real place to
go, verified rather than assumed.

**Why A3.2 is a span test, not a count.** `grep -c 'stamp'` cannot tell "eight instructions, one per
gate" from "eight instructions all in Gate 0". The arm locates each `^## Gate` heading, takes the
span to the next heading, and asserts the matching `stamp <label>` line falls **inside** that span.
That makes SKILL.md drift — a gate section reordered, renamed, or split — fail the suite instead of
silently degrading attribution (residual **R1**).

**M3 gates**: G0, G1 (`make check-changelog` still rc=0), G2 (`make check-skills` still rc=0),
A3.1, A3.2, A3.3 — plus all M1 and M2 ACs.

---

## 3. Mutation ledger (all seven, one table)

| # | Mutation | Milestone | Attacked refusal branch | Named arm | Blast radius |
|---|---|---|---|---|---|
| 1 | `MUT-BUFFERED` | M1 | "a stamp survives SIGKILL" (Q2) | `sigkill mid-gate-1 leaves last label gate-1` | SOLE |
| 2 | `MUT-SHARED-PATH` | M1 | namespacing + unset-name no-op (Q3) | `v1 and world stamps land in distinct files` | WIDE (2) |
| 3 | `MUT-DEFAULT-COMPLETE` | M2 | "no default-pass branch" (Q6-3) | `empty-stamps => DIED-PRE-GATE-0` | WIDE (≥3) |
| 4 | `MUT-MISSING-IS-PASS` | M2 | "missing artifact is loud" (Q6-2) | `deleted artifact => HEARTBEAT-MISSING` | SOLE |
| 5 | `MUT-UNSTAMPED-IS-PASS` | M2 | "COMPLETED needs positive stamp evidence" (Q6-1) | `rc=0 last=gate-5 => REAPED at=gate-5` | WIDE (2) |
| 6 | `MUT-UNBOUNDED` | M2 | the 200-line history bound (Q5) | `250 appends leave exactly 200 lines` | SOLE |
| 7 | `MUT-STALE-ATTEMPT` | M2 | per-attempt reset (Q1-1, Round-2 Finding 1) | `attempt 2 pre-gate-0 => DIED-PRE-GATE-0 attempt=2` | SOLE |

The suite prints `mutations: K/K killed` where K is the number of arms **present in that
milestone** — 2 at M1, 7 at M2 and M3. This is what makes A1.3 and A2.1 different criteria rather
than one criterion that is unreachable until the end.

---

## 4. Executor operating contract

**The executor runs in a `workspace-write` sandbox and CANNOT commit.** A linked worktree's `.git`
is a *file* pointing at a directory outside the sandbox, so every git write fails. Therefore:

1. **NO git write operations by the executor.** No `add`, `commit`, `stash`, `checkout`, `branch`,
   `worktree`, `push`, `merge`, `rebase`, `reset`, `clean`. Read-only git (`log`, `show`, `diff`,
   `status`, `rev-parse`) is fine.
2. **The controller builds the commits**, one per milestone, from the executor's snapshots.
3. **After each milestone M\<k\>, the executor snapshots every created-or-modified file** into
   `.snap/M<k>/<repo-relative-path>` — **cumulative** (every file the sprint has touched so far, not
   just the ones this milestone changed) and **full post-milestone content** (not a diff):

   ```bash
   mkdir -p .snap/M1/tools/launchd .snap/M1/make
   cp tools/launchd/mission-heartbeat.sh       .snap/M1/tools/launchd/
   cp tools/launchd/test_mission_heartbeat.sh  .snap/M1/tools/launchd/
   cp make/test.mk                             .snap/M1/make/
   ( cd .snap/M1 && find . -type f | sort | xargs shasum -a 256 ) > .snap/M1/SHA256SUMS
   ```

   M2 repeats this into `.snap/M2/` including the M1 files *as they stand after M2*, and M3 into
   `.snap/M3/`. The controller reconstructs one commit per milestone and verifies **byte-identity by
   sha256** against `SHA256SUMS` before committing.
4. **`.snap/` is scratch and is never committed.** It sits at the workspace root, outside
   `tools/launchd/`, so the `test-launchd-drivers` recipe's `tools/launchd/*.sh` glob cannot reach it
   and no snapshot copy is ever syntax-checked or executed as a test.
5. **Do not run `make quick-install` or install anything to `~/go/bin`** — that binary is shared with
   concurrent agents on this rig.
6. **Do not run `tools/launchd/mission-control.sh`**, in any mode including `MISSION_DRY_RUN=1`: it
   fires live model probes and its overlap guard interacts with the running V1 loop. Every driver
   claim in this plan is provable statically or through extracted-block tests.
7. **Every gate command that names `make test-launchd-drivers` carries `PATH="/usr/sbin:$PATH"`.**
   Without it the target is rc=2 at base for reasons unrelated to this sprint.

---

## 5. What was cut, and why

**The design doc's "Day 3 — M3: land + live-fire observation" is cut from the executor's sprint.**
It is not implementation work and the executor cannot do it: it requires a merge to `origin/dev`, a
pin advance, and then *waiting for the next scheduled V1 fire* on a 5400 s launchd timer. Its two
criteria (the doc's AC-6 and AC-7, this plan's **A4.1** and **A4.2**) are **controller-owned,
post-merge** observations and are listed in §1.2 under that heading, baselined, so they are not lost.

In its place M3 is the SKILL.md + release-note milestone, which *is* implementable and which carries
the span-contract test that keeps the skill half from rotting. Net effect: three implementable
milestones inside 3 days, with the live-fire report — D-52's explicit *"report the MECHANISM when you
have it, not just the rate"* — handed to the controller as a follow-up observation rather than
pretended to be sprint work.

Velocity supports this. `tools/launchd/` took **53 commits / 3,571 insertions in the last 30 days**
(`git log --since="30 days ago" -- tools/launchd/`), i.e. ~119 insertions/day sustained in this exact
directory by this exact loop. ~470 LOC of shell across 3 days is **below** demonstrated pace, which
is the right side to be on for surgical edits into a live 67,759-byte driver.

---

## 6. Where this plan refutes the design doc

### R-1 — `CHANGELOG.md — entry under Unreleased` would turn a `make ci` gate RED. **CONFIRMED.**

The doc's "Files to Modify" lists `CHANGELOG.md — entry under Unreleased`. Both halves are wrong.

There is no `Unreleased` section: `grep -n -i 'unreleased' CHANGELOG.md` → **0 hits**. Root
`CHANGELOG.md` is an *index* — its body is a table of archive links, and
`make/code-health.mk` describes the gate as *"Check root CHANGELOG.md stays an index, not a
changelog (CI gate)"*. `check-changelog` is in the `make ci` list (`make/ci.mk:11`).

Measured, with a before/after control and a restore:

```
make check-changelog                        -> rc=0   "✓ CHANGELOG.md is index-only …"
printf '\n## [Unreleased]\n\n### Added\n- test entry\n' >> CHANGELOG.md
make check-changelog                        -> rc=2   "The only heading an index may carry is '## Changelog Archives'."
                                                      "release-manager builds release notes from the active
                                                       changelogs/ file. Anything left here is silently dropped"
cp /tmp/CL.bak CHANGELOG.md; git status --porcelain CHANGELOG.md   -> 0 dirty paths (restored)
```

**Correction carried into this plan**: M3 writes the release note to **`changelogs/v0.32-current.md`**
(the active file the index links), and **G1**/`A3.4` keep `make check-changelog` at rc=0 as a gate.
Following the doc literally would have shipped a red CI gate *and* silently dropped the note from the
next release.

### R-2 — V22's `PIDFILE` citation is right about the line and wrong about V1. **CORRECTION.**

V22 cites `PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"` at line 81. Line 81 does say that — but
it is inside the **`else` (non-v1)** branch. For **V1**, the mission this ships for, line **64** sets
the LEGACY bare `PIDFILE="$STATE_DIR/mission-control.pid"`:

```
64:  PIDFILE="$STATE_DIR/mission-control.pid"        # inside  if [ "$MISSION_NAME" = "v1" ]
81:  PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid" # inside  else
```

**This does not change the design.** The guard is still effectively per-mission (only V1 uses the
bare name; every other mission uses its own), so the Conflict Surface's ordering argument holds. But
the doc's phrase *"while this slot holds the per-mission PIDFILE"* is not literally true for V1, and
an executor who greps for `mission-v1.pid` will find nothing and may conclude the guard is broken.
Recorded so that does not cost an hour.

### R-3 — `MISSION_CONTROL_DRY_RUN` does not exist. **CORRECTION.**

The doc's Milestone-1 line says *"manual dry-run (`MISSION_CONTROL_DRY_RUN` path unaffected —
verify)"*. `grep -n 'DRY_RUN' tools/launchd/mission-control.sh` returns **only**
`MISSION_DRY_RUN` (lines 776, 777, 779). The correct name is `MISSION_DRY_RUN=1`, its branch exits at
line **785**, and §4.6 forbids running it anyway.

### Everything else in the doc that this plan re-derived, held

V3 (bash 3.2.57), V4 (the `launchd-drivers` CI job at `ci.yml:535` on `macos-latest`; the
`test-launchd-drivers` target at `make/test.mk:40` — so the "tools/launchd has zero CI coverage"
memory really is stale), V5 (`export MISSION_NAME` line 52), V6 (`STATE_DIR` line 59 and the V1
legacy block), V9 (`log "iteration complete (rc=0)"` at line **1071**), V11 (all seven gate headings,
line-for-line), V20 (`wc -c` = **67759**), V21 (`_mc_run_once` defined 950, called 1023;
`TRANSIENT_RETRIES` line 342), V22 (guard release **1037**, guard read **763–768**), V26
(`grep -c 'AFTER recording itself'` = **0**, positive control `iteration exited rc=` = **10**,
negative control `zzq_plan_915` = **0**), V27 (`_mc_bounded()` line **194**, 8 uses, negative control
`_mc_never_915` = 0). `MISSION_ATTEMPT` and `mission-heartbeat`/`slot-verdict`/`SLOT VERDICT` are all
**unallocated** across `tools/`, `make/`, `.claude/` (0 hits, with positive controls: 13 files match
`MISSION_NAME`, 3 lines match the existing `DRIVER PIN DECISION` marker).

---

## 7. Open seams the executor must close

### S-1 — the driver and the helper resolve the artifact path by **different** rules

The helper (per Q3) resolves `${AILANG_STATE_DIR:-$HOME/.ailang/state}`. The driver hardcodes
`STATE_DIR="$HOME/.ailang/state"` at line 59. In production, with `AILANG_STATE_DIR` unset, the two
agree — but nothing in the design *makes* them agree, and the tests set the two variables
independently, so a divergence would be **invisible to the suite** while silently splitting the
`fired` stamp from the controller's gate stamps into two files. The verdict would then read
`DIED-PRE-GATE-0` on every healthy iteration: the Q6 floor inverted, exactly the failure mode
Round-2 Finding 4 was raised to prevent.

**Required (minimal, scoped to the two new files only):** the driver derives its heartbeat and
history paths as `"${AILANG_STATE_DIR:-$STATE_DIR}/mission-${MISSION_NAME}-…"` — the same
`AILANG_STATE_DIR`-wins rule as the helper. **Do not** make the driver's `STATE_DIR` itself honour
`AILANG_STATE_DIR`; that would move the V1 legacy paths (pidfile, kill switch, model override) and is
exactly the migration the doc's Non-Goals forbid.

**AC A1.6** pins it: one arm sets a single `AILANG_STATE_DIR=$tmp`, runs the driver's extracted
`fired` write and the real helper's `stamp gate-0` in that one environment, and asserts **one** file
holds both lines — `PASS: driver and helper resolve the same artifact path`. (A1.6 is written in M1
against the helper plus a stub, and re-run unchanged in M2 against the real extracted driver write.)

### S-2 — `set -u` and the extracted-block tests

Phase 1 reads `RC`, `MISSION_ATTEMPT`, `TRANSIENT_RETRIES`, `MISSION_NAME`, `STATE_DIR`, `LOG`.
Under `set -uo pipefail` (line 38) any unbound one aborts the *driver*, not just the test. Every read
in the new block uses `${VAR:-default}` unless the variable is provably bound on every path to it,
and each extracted-block arm binds them explicitly — the trap `test_driver_notify.sh` already
documents in-line.

### S-3 — extraction emptiness is a FATAL, not a skip

Per the directory convention, if `awk` between the `SLOT VERDICT` markers yields zero lines the suite
**exits non-zero with `FATAL: extraction of slot_verdict produced nothing`**. A suite that silently
skips a block it could not find is a suite that reports green on a deleted feature — the precise
failure class this whole design exists to remove.

---

## 8. Recommended queue rows — NOT this sprint

The doc routes both of these away and this plan does **not** absorb them. Recommended as queue rows:

- **R7 — the late-kill record detector is dead on the pin.** The rc≠0 branch compares
  `pre_last_record` / `post_last_record` read from the pin's **working tree**, which is frozen for the
  whole run, while records land via PR to `origin/dev`. Re-measured here: `grep -c 'AFTER recording
  itself' /tmp/ailang-mission-control.log` = **0** against a positive control of **10** rc≠0 exits in
  the same file. Consequence: a genuine late-kill-after-landed-work is misreported as a lost
  iteration — the exact iter-145 failure the branch was built to prevent. Suggested fix: bounded
  `git fetch` + `git show origin/dev:<log>` (the V25-proven reader) **outside** any guard-held
  section, with fetch failure degrading to a named `record-unverified` outcome, never a false alarm.
- **R8 — two unbounded `ailang messages send` calls on the driver's exit path.** Re-derived
  first-party: `grep -n 'ailang messages send controlplane'` puts them at **1046** (late-kill notice)
  and **1063** (rc-fail notice), both unwrapped. `_mc_bounded()` is already in the same file at line
  **194** with 8 existing call sites, so the fix is a wrapper swap, not new machinery. This sprint
  routes its **own** phase-2 notice through `_mc_bounded()` and leaves those two alone.

Both are pre-existing defects surfaced during design; per the mission's rule that is a queue row, not
a scope expansion.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-mission-slot-heartbeat-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_m-mission-slot-heartbeat.json`
