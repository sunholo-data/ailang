# Sprint Plan: M-PLANNER-CODEX-LANE

**Design doc**: [`m-planner-codex-lane.md`](m-planner-codex-lane.md) (rev 3, committed `e980c72d5`)
**Planned**: 2026-07-31, mission iteration 125 (sprint-planner role, opus)
**Execution**: **PARKED to the 2026-08-03 07:00 re-arm** (`dev.ailang.mission-rearm.plist`). This plan
is written to be picked up **cold** by a future iteration's executor — it assumes no context from
iteration 125 beyond this file and the design doc.

> **READ THIS FIRST — five verified corrections to the design doc.** Six of the doc's acceptance
> criteria and its entire rollback story are wrong as written. They are corrected inline below and
> the corrections are the binding version. Section [§0](#0-corrections-to-the-design-doc-binding)
> has the evidence.

## Summary

Route `MISSION_PLANNER_MODEL` through the hardened `PROVIDER=codex` spawn recipe so the mission
sprint-planner rides the ChatGPT-subscription quota bucket and opus stays controller-only
(Mark quota-offload #1, directive 2026-07-30).

**Duration:** 1 day (~8h), 4 milestones across 2 gated phases
**Total LOC:** ~310 (≈190 new shell + fixtures, ≈40 driver, ≈60 skill markdown, ≈15 landing)
**Dependencies:** M1b codex executor lane (implemented); the 2026-08-03 re-arm
**Risk Level:** **HIGH** — this edits `tools/launchd/mission-control.sh`, the launchd driver that
fires the mission loop. A syntax error here wedges the loop. See [§Point of no return](#point-of-no-return-and-the-real-revert).
**CI relevance: ZERO.** `tools/launchd/` has no workflow reference, no `shellcheck`, no `bash -n`
gate, no test file (verified — see §0.6). **A green CI tells you nothing about this change.** The
acceptance criteria below are the only real gate.

---

## 0. Corrections to the design doc (BINDING)

Each row was verified first-party during planning. Command included so the executor can re-run it.

### 0.1 — AC1, AC2, AC7(driver form) are UNRUNNABLE as written. **BLOCKING.**

The doc's commands are `MISSION_DRY_RUN=1 bash tools/launchd/mission-control.sh`. The dry-run
early-exit is at line 351, but the **overlap guard is at line 334**, ahead of it. The sprint
executes *inside* a live mission iteration, whose driver has written `$PIDFILE`
(`~/.ailang/state/mission-control.pid`) with a live PID. So the command yields and exits **rc=0
printing no roles line at all** — the exact vacuous-pass shape the doc's D5 preamble is written
to kill, reproduced by the doc's own ACs.

```
$ MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh
[2026-07-31 18:50:02] previous iteration still running (pid 99870) — yield (next interval retries)
rc=0                     # <-- no roles line. AC1 "passes" by not running.
```

**Correction (verified working, zero side effects, zero tokens):** every dry-run AC runs under a
throwaway mission namespace, which relocates `$PIDFILE`/`$KILL_SWITCH`/`$LOG` (driver lines 56-73)
away from the live V1 ones:

```
$ MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus \
    MISSION_PLANNER_MODEL=codex:gpt-5.6-sol /bin/bash tools/launchd/mission-control.sh
[...] DRY RUN ok: mission=acheck ... | roles: designer=claude:claude-fable-5 \
      planner=codex:gpt-5.6-sol executor=opus evaluator=sonnet
rc=0                     # live V1 pidfile verified untouched afterwards
```

Consequences the executor must carry:
- **AC2's log target changes.** `log()` is `tee -a "$LOG"` (line 87); under `MISSION_NAME=acheck`
  `$LOG` is `/tmp/ailang-mission-acheck.log`, **not** `/tmp/ailang-mission-control.log`. Grep the
  namespaced log (or stdout), never the live one.
- Every dry-run AC pins `MISSION_EXECUTOR_MODEL=opus` so the run costs **zero** codex probes.
- Do **not** "fix" this by moving or deleting the live pidfile. That disarms the loop's only
  concurrency guard.

### 0.2 — D6's rollback ("one env var") is FALSE for the V1 mission. **BLOCKING.**

The doc says: *"set `MISSION_PLANNER_MODEL=opus` in `~/.config/ailang/mission-v1.env` (driver
sources it, `mission-control.sh:4`)"*. Line 4 is a comment. The actual source is lines 47-48 and
it is **gated on `$MISSION_PROFILE` being non-empty**:

```bash
# tools/launchd/mission-control.sh:47-48
[ -n "${MISSION_PROFILE:-}" ] && [ -f "$HOME/.config/ailang/mission-${MISSION_PROFILE}.env" ] \
  && . "$HOME/.config/ailang/mission-${MISSION_PROFILE}.env"
```

`~/Library/LaunchAgents/dev.ailang.mission-control.plist` sets only `PATH` and `HOME` — **no
`MISSION_PROFILE`**. Only the World plist sets it (`MISSION_PROFILE=world`). And
`~/.config/ailang/mission-v1.env` does not exist. Verify:

```
$ plutil -p ~/Library/LaunchAgents/dev.ailang.mission-control.plist | grep -c MISSION_PROFILE   # 0
$ ls ~/.config/ailang/                    # mission-world.env, secrets.env — no mission-v1.env
```

So today the documented V1 rollback is a **no-op**: the file is never read. (Bitter irony: it
works for World, the mission the doc says is insulated.)

**Correction — M2 makes the rollback real** with two lines, inserted immediately after
`MISSION_NAME` is resolved (driver line ~49):

```bash
# D6 rollback pin: always sourced for the RESOLVED mission name, so the documented
# `MISSION_PLANNER_MODEL=opus` rollback works for V1 (whose plist sets no MISSION_PROFILE).
# Convention: entries use ${VAR:-value} so a command-line env pin still wins (the AC fixtures
# depend on that). Double-source for World is idempotent.
[ -f "$HOME/.config/ailang/mission-${MISSION_NAME}.env" ] \
  && . "$HOME/.config/ailang/mission-${MISSION_NAME}.env"
```

Plus **new AC9** (below) that *exercises* the rollback. A rollback nobody has run is a claim, not
a fact — and D6 is the entire safety argument for a control-plane change.

### 0.3 — AC4's `git status --porcelain` clause can never pass. **BLOCKING.**

AC4 demands the main checkout show "ONLY the two intended artifacts". The main checkout is
chronically dirty with rig-generated files:

```
$ git status --porcelain | head
 M .claude/fmt_hook_events.jsonl
 M docs/static/benchmarks/latest.json
 M docs/static/benchmarks/os/history.json
 ...        # 6 modified files at plan time, none related to this sprint
```

**Correction:** scope the assertion —
`git status --porcelain -- design_docs/planned .ailang/state/sprints` shows exactly the two
intended artifacts and nothing else. The worktree-leak half of AC4 (`git worktree list` has no
`planner-wt-*`) is fine as written; note 13 unrelated worktrees exist, so grep for the prefix,
never assert an empty list.

### 0.4 — The D2 allowlist still fail-closes this doc, even after the controller's ruling. **BLOCKING.**

The controller ruled (iteration 125): *the allowlist is right and stays; Files-to-Modify declares
**repo-relative source paths only**; the `~/.claude/...` and ailang-world copies are sync/deploy
steps and move to a landing checklist.* That ruling is adopted here (M1 step 4).

**It is necessary but not sufficient.** D2 step 4 says "parse **every** declared path in the …
section". Taken literally, the doc's own section yields four non-paths and a language path:

```
$ awk '/^### Files to Modify\/Create/{f=1;next} f&&/^---$/{exit} f' \
    design_docs/planned/v1_0_0/m-planner-codex-lane.md | grep -oE '`[^`]+`' | tr -d '`' | sort -u
**Planner-Lane**: codex-ok | opus-required
codex-ok
internal/...                 # <-- prose inside the fixture bullet
internal/parser/…            # <-- prose inside the fixture bullet
tools/launchd/...            (etc.)
```

`internal/parser/…` is *prose describing fixture (a)*, not a declared path — but a literal
"every backticked token" reader fail-closes on it regardless of the `~` paths. **The extractor
contract must be pinned, not left to the implementer.**

**Correction — the binding contract:** one declared path per top-level `- ` bullet = the **first**
backticked token of that bullet; continuation lines and later backticks in the same bullet are
prose and ignored. Verified to yield exactly the real paths:

```
$ awk '/^### Files to Modify\/Create/{f=1;next} f&&/^---$/{exit} f&&/^- /{
    if (match($0,/`[^`]+`/)) print substr($0,RSTART+1,RLENGTH-2) }' <doc>
tools/launchd/mission-control.sh
.claude/skills/mission-control/SKILL.md
~/.claude/skills/mission-control/SKILL.md                      # removed by the M1 doc correction
~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh   # removed by the M1 doc correction
.claude/skills/design-doc-creator/SKILL.md
tools/launchd/derive-planner-lane.sh
tools/launchd/testdata/planner-lane/
```

A bullet whose first backticked token is **not path-shaped** (contains no `/` and has no
`.md`/`.sh`/`.go`/`.yml` extension) → `opus fail-closed:unparsable-path-entry`. New fixture (j)
covers it.

### 0.5 — The section-heading pattern must be a regex, and the lane will engage rarely at first.

Eight distinct spellings exist across the 102 planned design docs, including a truncated one:

```
$ grep -rhoE '^#{2,4} +Files[^\n]*' design_docs/planned/ | sort | uniq -c | sort -rn
  27 ### Files to Modify/Create      4 ## Files to Modify/Create      2 ### Files to Modify
   2 ### Files to Create             2 ## Files to Modify             1 ### Files to touch
   1 ### Files to cha                1 ### Files to Create/Modify
$ grep -rlE '^#{2,4} +Files' design_docs/planned/ | wc -l     # 40 of 102
```

**Correction:** match `^#{2,4}[[:space:]]+Files\b` (case-insensitive), range-terminated by the
next `^#{1,4} ` heading or `^---$`. Even so, **62 of 102 existing planned docs have no Files
section at all** and will return `opus fail-closed:no-files-section`; and the `**Planner-Lane**`
field exists in exactly **one** doc today (this sprint's own). State this honestly in the landing
report: **the codex planner lane engages only on newly authored infra docs.** That is the
fail-closed design working as intended, not a bug — but it means the doc's success metric
("`planner=codex…` appears in the evidence row on infra iterations") matures over weeks, not on
day one. Add fixture (k) for an alternate heading spelling.

### 0.6 — Confirmations (controller premises I re-verified and can vouch for)

| Premise | Verdict | Command |
|---|---|---|
| Driver runs Bash 3.2.57; no 4.x anywhere on rig | **CONFIRMED** | `env bash --version` → 3.2.57; `ls /opt/homebrew/bin/bash /usr/local/bin/bash` → both ENOENT |
| `declare -A` and `${var,,}` both fail under it | **CONFIRMED** | `/bin/bash -c 'declare -A t; t[gpt-5.6-sol]=1'` → `invalid option` + `invalid arithmetic operator`; `/bin/bash -c 'role=P; echo "${role,,}"'` → `bad substitution` |
| `tools/launchd/` has zero CI coverage | **CONFIRMED** | `grep -rn "tools/launchd" .github/workflows/` → empty, with positive control `grep -rln "tools/" .github/workflows/` → 2 files; no `shellcheck`/`bash -n` in workflows or Makefile |
| `create_sprint_json.sh:88` interactive `read -p`; placeholder-milestone generator at :160-172 | **CONFIRMED** | read the file |
| World driver is byte-identical | **CONFIRMED** | `diff -q` silent |
| The doc's D3 Bash-3.2 sketch is correct | **CONFIRMED** — I extracted it verbatim into a stub harness: `bash -n` rc=0 under `/bin/bash`; dedupe, per-role fallback, and non-`codex:` pass-through all behave (see M2 task 1) | `/tmp/sketchtest/sk.sh` |

---

## 1. Current status analysis

### Velocity
- 148 commits in the last 7 days; recent comparable infra sprints:
  `m-nightly-sustained-failure-label` 415 LOC / 0.9d, `m-nightly-run-validity-gate` 635 LOC / 1.1d.
- This sprint: ~310 LOC, **~8h**. Comfortably inside recent velocity. The cost here is **not** LOC
  — it is verification discipline on an uncovered, loop-critical file.

### Execution shape (codex executor, 30-min bounded runs, cannot commit)
Confirmed by design-doc L6: a linked worktree's `git commit` is **denied** under the
`workspace-write` sandbox. So every milestone is authored in the sprint worktree by codex and
**committed by the controller** in the main checkout. Each milestone below is sized to fit one or
two 30-minute bounded runs plus controller finalization.

---

## 2. Proposed milestones

### Milestone 1 — Derivation script + fixtures + doc correction (NO runtime effect)
**Estimated:** ~190 LOC (~90 script, ~90 fixtures, ~10 doc/template) · **~3h** · **2 codex runs**
**Runtime effect: NONE.** Nothing invokes `derive-planner-lane.sh` until M3 edits the skill.
**Revert:** `rm tools/launchd/derive-planner-lane.sh` + revert two markdown files.

**Tasks**
1. Write `tools/launchd/derive-planner-lane.sh`, Bash 3.2 only, pure text, **no network, no codex
   invocation**. Contract in strict order:
   - step 0 (env-pin): `$MISSION_PLANNER_MODEL` does not start with `codex:` →
     `opus fail-closed:env-pin`, exit 0.
   - step 1: doc missing/unreadable → `opus fail-closed:no-doc`.
   - step 2: `grep -m1 -E '^\*\*Planner-Lane\*\*:'`; absent →
     `opus fail-closed:planner-lane-field-missing`; value not exactly `codex-ok`|`opus-required` →
     `opus fail-closed:planner-lane-field-invalid`.
   - step 3: `opus-required` → `opus declared:opus-required`.
   - step 4: allowlist cross-check using the §0.4 extractor and the §0.5 heading regex. Section
     absent / found more than once → `opus fail-closed:no-files-section`. Non-path-shaped first
     token → `opus fail-closed:unparsable-path-entry`. Any path that is absolute, starts `~`,
     contains `..`, or does not begin with one of
     `tools/launchd/`, `.claude/skills/mission-control/SKILL.md`,
     `.claude/skills/design-doc-creator/SKILL.md` →
     `opus fail-closed:path-not-in-codex-allowlist`.
   - step 5: all paths allowlisted → `codex declared:codex-ok`.
   Exactly one line on stdout, always exit 0 (the caller reads the line, not the rc). Every
   diagnostic to stderr.
2. Write 11 fixtures under `tools/launchd/testdata/planner-lane/` (each a ~10-line markdown doc):
   `a-unlisted-language-path.md`, `b-field-missing.md`, `c-clean-infra.md`,
   `f-future-internal-path.md`, `g-mixed-infra-language.md`, `h-malformed-files-section.md`,
   `i-duplicate-files-sections.md`, plus **new**: `j-prose-first-bullet.md` (§0.4),
   `k-alternate-heading.md` (`## Files to Modify`, §0.5), `l-field-invalid.md`,
   `m-opus-required.md`.
3. `.claude/skills/design-doc-creator/resources/design_doc_structure.md:12-14` — add the
   `**Planner-Lane**: codex-ok | opus-required` header line to the template block, plus one
   sentence next to `### Files to Modify/Create` (line 295) stating the one-path-per-bullet,
   path-first convention that §0.4 now depends on. (Note: the doc names
   `.claude/skills/design-doc-creator/SKILL.md`; the actual template body lives in
   `resources/design_doc_structure.md`. Edit the resource; SKILL.md:444 gets the one-line
   convention note. **Add both paths' prefix to the allowlist reasoning** — both are under
   `.claude/skills/design-doc-creator/`, so widen that allowlist entry from the single SKILL.md
   file to the `.claude/skills/design-doc-creator/` **prefix**; record this as a deliberate,
   minimal widening in the landing commit.)
4. **Controller ruling, encoded:** correct `m-planner-codex-lane.md`'s own
   `### Files to Modify/Create` — delete the two `~`/absolute sync-copy bullets and add a new
   `### Landing Checklist` section listing them as deploy steps. Also de-backtick the
   `internal/parser/…` / `internal/...` prose inside the fixture bullet (§0.4 defence in depth).
   Then prove the doc now self-qualifies (AC-D0 below).

**Acceptance**
- [ ] **AC7b** (doc, unchanged): `/bin/bash -c '/bin/bash --version | head -1; MISSION_PLANNER_MODEL=codex:gpt-5.6-sol /bin/bash tools/launchd/derive-planner-lane.sh tools/launchd/testdata/planner-lane/c-clean-infra.md'`
      → output shows `version 3.2` **and** `codex declared:codex-ok`, rc=0. Zero tokens.
- [ ] **AC8** (doc, extended): with `MISSION_EXECUTOR_MODEL=opus` and
      `MISSION_PLANNER_MODEL=codex:gpt-5.6-sol` pinned — (a) → `path-not-in-codex-allowlist`;
      (b) → `planner-lane-field-missing`; **(c) → `codex declared:codex-ok` (the known-positive
      control — an all-opus fixture sweep is a claim, not a fact)**; (f)(g)(h)(i) →
      `path-not-in-codex-allowlist`; **(j) → `unparsable-path-entry`; (k) → `codex
      declared:codex-ok` (alternate heading still parses); (l) → `planner-lane-field-invalid`;
      (m) → `opus declared:opus-required`**.
- [ ] **AC8d** (doc): `grep -cE 'codex (exec|sandbox)' tools/launchd/derive-planner-lane.sh` → **0**,
      with known-positive control `grep -cE 'codex (exec|sandbox)' tools/launchd/mission-control.sh`
      → **4** (measured at plan time) in the same command.
- [ ] **AC8-env** (new, from the rev-3 gemini fix): with `MISSION_PLANNER_MODEL=opus`, fixture (c)
      → `opus fail-closed:env-pin`. Proves the env pin beats a `codex-ok` doc.
- [ ] **AC-D0** (new, §0.4): `derive-planner-lane.sh` run against the **corrected**
      `m-planner-codex-lane.md` with the planner pinned → `codex declared:codex-ok`. The doc no
      longer fail-closes itself. *Kills:* "the ruling was applied to the prose but the parser still
      chokes."

**Risks** — extractor over-fitting to this one doc. Mitigation: fixtures (j)(k) plus a sanity
sweep printing the derivation for all 40 docs that have a Files heading (expected: nearly all
`planner-lane-field-missing`; **any `codex declared:codex-ok` on a doc touching `internal/` is a
hard stop**).

---

### Milestone 2 — Driver: role-generic probe loop + real rollback plumbing ⚠️ POINT OF NO RETURN
**Estimated:** ~42 LOC net · **~1.5h** · **1 codex run + controller verification**
**Runtime effect: IMMEDIATE at the next launchd fire** (`StartInterval` 5400s = 90 min).

**Tasks**
1. In the **sprint worktree**, replace `mission-control.sh:295-308` (the executor-only `case`
   block) with the doc's D3 role-generic loop **verbatim** — it is correct; I extracted it into a
   stub harness and it passes `bash -n` under `/bin/bash` 3.2 and behaves in all four cases
   (both-default; planner-bad; both-bad-same-model dedupe; non-`codex:` pass-through). Do not
   "improve" it.
2. Insert the §0.2 two-line rollback source after `MISSION_NAME` resolution (line ~49).
3. **Do NOT touch line 280.** The default flip is M4.
4. **Verification gate, in the worktree, in this order — all must pass before the file leaves the worktree:**
   a. `/bin/bash -n tools/launchd/mission-control.sh` → rc=0.
   b. `MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus /bin/bash tools/launchd/mission-control.sh`
      → roles line printed, rc=0 (zero probes).
   c. AC1 / AC2 / AC7 / AC9 below.
5. **Controller** copies the verified file into the main checkout, re-runs (a) and (b) **there**,
   and commits. The executor must never write `tools/launchd/mission-control.sh` in the main
   checkout.

**Acceptance** (all in the `acheck` namespace per §0.1)
- [ ] **AC1** (corrected): `MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus MISSION_PLANNER_MODEL=codex:gpt-5.6-sol /bin/bash tools/launchd/mission-control.sh`
      → roles line contains `planner=codex:gpt-5.6-sol`. (Pre-flip, so the value is env-supplied;
      M4 re-runs it without the pin.)
- [ ] **AC2** (corrected): `MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus MISSION_PLANNER_MODEL=codex:no-such-model /bin/bash tools/launchd/mission-control.sh`
      → **stdout / `/tmp/ailang-mission-acheck.log`** contains `codex planner lane -> falling back
      to opus` AND the roles line prints `planner=opus`. Costs exactly ONE real codex probe (the
      executor is pinned opus). *Kills:* the #486 silent false-green. **The probe must fail
      *because* it carries `--model`** — if it returns rc=0, #486 has regressed.
- [ ] **AC7** (corrected, pre-flip, zero probes):
      `/bin/bash -c '/bin/bash --version | head -1; MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus /bin/bash tools/launchd/mission-control.sh'`
      → shows `version 3.2` **and** the roles line, rc=0.
- [ ] **AC9** (NEW, §0.2 — the rollback is exercised, not asserted): create
      `~/.config/ailang/mission-acheck.env` containing
      `MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-opus}"`, then
      `MISSION_DRY_RUN=1 MISSION_NAME=acheck MISSION_EXECUTOR_MODEL=opus /bin/bash tools/launchd/mission-control.sh`
      → roles line shows `planner=opus`, sourced **from the file** (delete the file, re-run,
      confirm the value changes back — a one-sided test is a claim, not a fact). Then create the
      real `~/.config/ailang/mission-v1.env` with the same `${VAR:-...}` line **commented out**,
      ready to uncomment as the D6 rollback. Remove the `acheck` fixture file afterwards.
- [ ] `git diff` of the driver is **exactly** the probe-block replacement + the 2 rollback lines.
      No line-280 change. No reformatting.

**Risks**
| Risk | Mitigation |
|---|---|
| Syntax error wedges the loop at the next fire (90 min) | `bash -n` + namespaced dry-run in the **worktree first**, re-run in main after copy. `set -uo pipefail` (no `-e`) means a probe failure will not abort — verified. |
| Copy lands mid-iteration | Harmless: a fire during a live iteration yields at the overlap guard. First real exposure is the first fire after this iteration ends. |
| The 2-line profile source changes World semantics | World already sources `mission-world.env` via `MISSION_PROFILE`; the second source is idempotent (plain/`:-` assignments only). Prove with a `MISSION_PROFILE=world MISSION_DRY_RUN=1` dry-run against the World copy **before** syncing. |

---

### Milestone 3 — Skill recipe (mission-control) + copy sync
**Estimated:** ~60 lines markdown · **~1.5h** · **1 codex run**
**Runtime effect: the NEXT controller session** — including **the Ailang World controller** (§0.7).

**Tasks**
1. `.claude/skills/mission-control/SKILL.md:451` roles table — Sprint-planner default becomes
   `codex:gpt-5.6-sol`, cell states "effective lane = `derive-planner-lane.sh` output, verbatim;
   fail-closed to opus". Supersede the stale "down-tier A/B = M3" note.
2. Add step **1b** (derivation, MANDATORY, runs before any planner probe/spawn) per D4.
   `opus …` → spawn the opus Agent path directly, no codex probe for the planner role; the reason
   token goes verbatim in the evidence row. Script missing on disk → fail closed to opus, loudly.
3. Add the planner sub-bullet to the `PROVIDER=codex` section, **parameterizing** the executor
   recipe (four deltas only: `-C` the ephemeral detached worktree; per-iteration directive file
   `/tmp/codex_planner_directive_iter<N>.txt` with the identical ≥200 B / exit-64 assertion and
   `< /dev/null` on probe **and** run; keep `--add-dir "$GOCACHE" --add-dir "$GOMODCACHE"` with
   the "in-sandbox gate verdicts are NOT evidence" caveat carried verbatim; the six post-run
   controller steps). Reference the executor recipe's guards — do not fork them.
4. **Worktree creation must be `--detach` from local `HEAD`**, not `-b` from `origin/dev`
   (a just-committed-unpushed design doc would otherwise be invisible). Precondition: assert
   `git status --porcelain -- <design-doc>` is empty before spawning.
5. Sync to `~/.claude/skills/mission-control/SKILL.md` in the same commit window (both were
   70544 bytes / identical at plan time — `diff -q` after).
6. **Record a decision on the third copy:** `.agents/skills/mission-control/SKILL.md` is a
   **git-tracked third copy**, 44067 bytes, dated Jul 24, and **already drifted** from the
   `.claude` copy. Either sync it or write one line in the landing report declaring `.agents/`
   out of scope and stale-by-design. (`.agents/skills/sprint-planner/scripts/create_sprint_json.sh`
   is byte-identical to the `.claude` one — verified — so the planner's own scripts are unaffected
   either way.) Do **not** add `.agents/` to the D2 allowlist as a side effect.

**Acceptance**
- [ ] `diff -q` between the repo and `~/.claude` copies is silent after the edit.
- [ ] **AC8e** (doc): fixture (a) run through the skill's Gate-3 planner step → the spawned planner
      is the opus Agent path and the evidence row records the up-route token.
- [ ] The planner sub-bullet contains **no** duplicated guard text — every shared guard is a
      reference to the executor recipe (grep: the phrases `exit 64`, `< /dev/null`,
      `run_in_background` appear in the planner bullet only as references).
- [ ] `.agents/` decision recorded in the landing report either way.

**Risk — World blast radius is NOT zero (§0.7).** Mitigated by contract step 0 + missing-script
fail-closed, but must be stated in the commit, not discovered later.

---

### Milestone 4 — AC3a rehearsal, the default flip, and landing
**Estimated:** ~15 LOC · **~2h** (dominated by one real codex planner run) · gated on M1-M3 green

**Tasks**
1. **AC3a rehearsal** — run the full D4 planner recipe against the **corrected**
   `m-planner-codex-lane.md` (which AC-D0 proved now derives `codex declared:codex-ok`), sprint id
   `rehearsal-iter<N>`, in `/tmp/planner-wt-iter<N>` (`git worktree add --detach … HEAD`).
   Copy-back and commit are **skipped by design**; inspect in the worktree, then remove it.
   - If AC-D0 somehow fails, rehearse against fixture `c-clean-infra.md` instead — the doc's
     own escape hatch. Say which in the landing report.
2. **THE FLIP (final commit of the sprint):** `mission-control.sh:280` →
   `export MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-codex:gpt-5.6-sol}"` with a comment
   citing this doc and the D2 derivation. Then re-run AC1 **without** the planner env pin.
3. **World-sync decision (§0.7) — mandatory, one of two, recorded in the landing commit:**
   - **RECOMMENDED: sync the driver, insulate World's planner.** `cp` the driver to
     `~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh` (restores `diff -q`
     identity, which is itself the drift detector) **and** add
     `MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-opus}"` to `~/.config/ailang/mission-world.env`
     (which World **does** source — verified; it currently pins no role models). This keeps
     World's planner on opus despite the flipped default, and simultaneously demonstrates that the
     D6 rollback mechanism works. Verify with
     `MISSION_PROFILE=world MISSION_DRY_RUN=1 /bin/bash ~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh`
     → `planner=opus`. World is currently kill-switched (`~/.ailang/state/mission-world.disabled`)
     until the 08-03 re-arm, so this is verifiable with zero live risk.
   - **OR: explicitly do not sync** — then send a cross-mission message to `world-coordinator`
     stating that World's driver now lags V1's and lacks the planner probe, and that
     `derive-planner-lane.sh` is absent from the World checkout so any planner derivation there
     fails closed to opus, loudly. Record the message ID in the landing commit.
4. **Landing checklist** (the section M1 added to the design doc): `~/.claude` skill copy synced ·
   World driver synced-or-declined · `.agents/` decision · `mission-v1.env` rollback line present
   (commented) · `git worktree list | grep planner-wt-` empty.
5. Move the design doc to `design_docs/implemented/v1_0_0/` per the standard flow, and record the
   engagement-rate reality from §0.5 (62/102 docs fail closed; 1 doc carries the field today).

**Acceptance**
- [ ] **AC3a**: derivation returns `codex declared:codex-ok`; codex runs in the detached worktree;
      **in the worktree** `jq -e .` validates the JSON, `features | length >= 1`, **zero**
      occurrences of `MILESTONE_ID` or `auto-parse failed`, the plan file names the input doc, and
      the `-o` final message is a plan summary — **not** "What would you like me to work on?".
- [ ] **AC4** (corrected, §0.3): `git worktree list | grep -c planner-wt-` → 0, and
      `git status --porcelain -- design_docs/planned .ailang/state/sprints` shows only the intended
      artifacts.
- [ ] **AC5**: invoking the planner wrapper with the directive file **absent** exits **64** without
      spawning codex (grep the wrapper log for the FATAL line).
- [ ] **AC1 (post-flip)**: same command as M2's AC1 **minus** `MISSION_PLANNER_MODEL` →
      roles line still shows `planner=codex:gpt-5.6-sol`. (Post-flip this fires one real probe —
      the dry-run exit is after the probe block. Budget for it.)
- [ ] **AC6**: the iteration's routing-evidence row records the derivation output verbatim.

**Slip rule (doc, adopted):** if M4 slips, M1-M3 land and **the flip does not**. The lane stays
explicit-opt-in via `MISSION_PLANNER_MODEL=codex:gpt-5.6-sol`. A smaller provably-safe lane beats
a larger one gated on judgement.

---

## Point of no return, and the real revert

**The point of no return is M2, not M4.** M2 edits the file launchd executes every 90 minutes.

**The doc's "rollback = one env var" is overstated. The honest revert is three-tier:**

| Landed | Revert | Cost | Covered by D6's env var? |
|---|---|---|---|
| M1 (script, fixtures, template, doc) | `rm` the script + revert 2 markdown files | seconds | n/a — zero runtime effect |
| **M2 (driver probe loop + rollback plumbing)** | **`git checkout <sha> -- tools/launchd/mission-control.sh`** — a **code revert**. An env var cannot un-break a driver that fails to parse. | minutes, manual, and the loop is down until it happens | **NO** |
| M3 (skill) | revert both skill copies + re-sync `~/.claude` | minutes | **NO** (mitigated: derivation fails closed to opus if the script is gone) |
| M4 (line-280 flip) | `MISSION_PLANNER_MODEL=opus` — **and only after §0.2's plumbing exists** | seconds | **YES** (once M2 shipped it) |

So D6 is accurate **only for M4, and only because M2 first builds the delivery mechanism D6
already claimed existed.** Say so in the landing report.

**Wedge-proofing, in order:** (1) all driver editing happens in the sprint worktree;
(2) `bash -n` then namespaced dry-run **in the worktree**; (3) controller copies to main and
re-runs both **there**; (4) if the main-checkout dry-run does not print a roles line, `git
checkout` the file back immediately and stop the sprint. The main checkout copy is the last
irreversible act and it belongs to the controller, not to codex — which is enforced anyway, since
codex cannot commit from its sandbox.

**A wedge requires two failures, not one:** the driver falls back to opus on ANY non-zero probe rc,
and the opus Agent path is today's status quo. The genuinely new single-point risk is a **shell
parse error**, which is exactly what step (2) exists to catch.

---

## §0.7 World blast radius — the doc's "zero until synced" is wrong

The doc states: *"Ailang World mission: **zero until synced**."* False for the skill half.

```
$ ls ~/dev/sunholo-data/ailang-world/.claude/skills/mission-control/SKILL.md
ls: No such file or directory          # World has NO repo-local copy
```

World's controller therefore loads the **global** `~/.claude/skills/mission-control/SKILL.md` —
the exact file M3 mandates editing. The M3 edit reaches World at its next fire, with **no driver
sync at all**.

What saves it is accidental and worth naming: World's driver line 280 still defaults
`planner=opus`, so the rev-3 gemini step-0 env-pin check returns `opus fail-closed:env-pin` before
any path parsing; and `tools/launchd/derive-planner-lane.sh` does not exist in the World checkout,
so the skill's missing-script rule fails closed to opus, loudly. **World degrades safely but
noisily** — expect fail-closed tokens in World's evidence rows from the first fire after M3.

M4 task 3 makes this deliberate instead of accidental.

---

## Success metrics
- All corrected ACs green: AC1, AC2, AC3a, AC4, AC5, AC6, AC7, AC7b, AC8(a-m), AC8d, AC8e,
  AC8-env, AC9, AC-D0.
- Test coverage: **N/A** — no Go, no AILANG. Do **not** run `make test`.
- Docs: design doc corrected (§0.4 ruling) then moved to `design_docs/implemented/v1_0_0/`;
  CHANGELOG entry under mission infra; landing report carries the §0.5 engagement-rate reality
  and the §0.7 World statement.
- Example files: **N/A** (no language feature).

## Dependencies
- The **2026-08-03 07:00** re-arm (`dev.ailang.mission-rearm.plist`, one-shot, self-removing;
  it deletes `mission-control.disabled` and `mission-world.disabled`). At plan time the V1 loop is
  **live** (pid 99870) and World is kill-switched — do not assume a quiet rig.
- codex CLI authenticated via ChatGPT subscription (`~/.codex/auth.json` `auth_mode: chatgpt`).

## Open questions for the controller
1. **`.agents/skills/mission-control/SKILL.md`** — sync as a fourth copy, or declare out of scope?
   (M3 task 6 needs a ruling; either is defensible, silence is not.)
2. **World sync** — M4 task 3 recommends "sync driver + pin World's planner to opus in
   `mission-world.env`". Confirm or choose the decline-and-message branch.
3. **`mission-v1.env` creation** (§0.2) — M2 creates it with the rollback line commented out. If
   you would rather add `MISSION_PROFILE=v1` to the plist instead, say so; that needs a
   `launchctl bootout`/`bootstrap` of a live job and is the riskier option, which is why it is not
   the recommendation.
