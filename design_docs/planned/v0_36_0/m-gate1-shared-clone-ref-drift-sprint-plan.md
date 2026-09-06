# Sprint Plan — M-GATE1-SHARED-CLONE-REF-DRIFT

**Design doc**: [m-gate1-shared-clone-ref-drift.md](m-gate1-shared-clone-ref-drift.md) (quorum-cleared 2026-09-06, narrow-refinement carve-out, both R2 verbatim fixes applied)
**Target release**: v0.36.0 · **Priority**: P1 (process-correctness)
**Sprint ID**: `m-gate1-shared-clone-ref-drift` · **Planned in**: detached worktree @ `7870f40a01b99a459f0f8a8a72a4d36a9c1b2962` (V1 iteration 338)
**Milestones**: 3 · **Commit boundary**: ONE commit per milestone (3 commits total), built by the **controller** from executor snapshots — the executor's runner **cannot commit**.

---

## 1. Velocity calculation

Derived from `git log --oneline --since="2 weeks ago"` in this worktree and the design doc's own estimates:

| Measure | Value | Source |
|---|---|---|
| Commits, last 15 days (2026-08-23 → 2026-09-06) | **677** (≈ 45/day; peak 102 on 08-26) | `git log --oneline --since="2 weeks ago"` |
| Raw additions, all files | **+167,018** (≈ 11,135/day) | `git log --numstat --since=...` |
| Additions scoped to this sprint's trees (`tools/`, `.claude/`, `design_docs/`) | **+65,414 / −8,598** (≈ 4,361 added/day) | same, path-scoped |
| Authors | 531 fleet bot, 130 sunholo-voight-kampff, 10 dependabot, 6 Mark | `git shortlog` equivalent |

The design doc's own estimate (quorum-reviewed): **3 days — M1 1d, M2 1d, M3 1d**.

**Sprint load**: 440 LOC total → **target 147 LOC/day**. That is ≈ 3.4% of the demonstrated scoped-tree raw throughput (4,361 added/day) and 3 commits against a ~45 commits/day cadence. Volume is not the risk. The binding constraints are (a) the **ratchet arithmetic** in M2/M3 (every added SKILL.md line must be paid by moving prose to `resources/ref-drift.md`; the file must END at ≤ 2781), and (b) copying the helper **verbatim** from the design doc rather than improvising it. Schedule buffer lives inside each 1-day milestone, not in added days.

## 2. Milestones & LOC estimates

| # | Milestone | LOC (est.) | Commit (controller-built) |
|---|---|---|---|
| M1 | `mission-base.sh` helper + `test_mission_base.sh` non-vacuity test + `make/test.mk` wiring | **260** (helper ~55 verbatim from doc; test ~200, grounded on `test_mission_heartbeat.sh`=194 / `test_mission_memgate.sh`=206 / `test_mission_stall.sh`=221; test.mk +1) | commit 1/3: `feat(mission): add mission-base.sh base recorder + non-vacuity drift test (M-GATE1-SHARED-CLONE-REF-DRIFT M1)` |
| M2 | Gate 1 + Gate 3 SKILL.md wiring; move war-story/rationale bulk to NEW `resources/ref-drift.md`; net SKILL.md delta ≤ 0 | **130** (SKILL.md gross churn ~45: +~22 wiring/link lines, −~40 moved prose; new `resources/ref-drift.md` ~85 incl. moved prose + drift protocol + test recipe) | commit 2/3: `feat(mission): wire Gate 1/3 base record+drift; move war stories to resources/ref-drift.md (M2)` |
| M3 | Gate 3b poll-target + Gate 4 record/Routing-evidence wiring; optional gate2 stamp decision; final non-vacuity re-proof | **50** (SKILL.md +~20; ref-drift.md observation addendum ~10; optional gate2 ~4; notes) | commit 3/3: `feat(mission): wire Gate 3b/4 base record+drift + Routing-evidence base= stamp (M3)` |
| | **TOTAL** | **440** | 3 commits |

`estimated_loc` per feature in the sprint JSON matches this table; `estimated_total_loc = 440 = 260+130+50`; `estimated_days = 3`.

## 3. Day-by-day task breakdown

### Day 1 — M1: measurement & record helper + non-vacuity test (~260 LOC)

1. **(Morning, BEFORE any edit) Baseline capture.** Run the launchd-drivers suite twice — once under the ambient runner env, once scrubbed — and save ALL `FAIL:`/`not ok`/`====` lines to `.snap/H0-test-baseline.txt`:
   ```bash
   mkdir -p .snap
   make test-launchd-drivers 2>&1 | grep -E "^FAIL|^not ok|^====" > .snap/H0-test-baseline.txt
   env -i HOME="$HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb \
     make test-launchd-drivers 2>&1 | tail -60 >> .snap/H0-test-baseline.txt
   ```
   Measured at HEAD `7870f40a0` by the planner: ambient runner env (exported `MISSION_*`/`CONTROLLER_*` vars) induces **6 pre-existing reds in `test_mission_routing.sh`** (R6/arm12 class, all `…anthropic-fallback:fail-closed:path-not-in-codex-allowlist`); the **scrubbed-env run is fully green** (54+27+77+17 + quiet scripts + probe, rc=0). `make` stops at the first failing script, so a polluted ambient run never reaches `test_motoko_connection_probe.sh`. The scrubbed run is the CI-parity gate (the CI `launchd-drivers` job at `.github/workflows/ci.yml:582` runs with a clean env on `macos-latest`).
2. **Create `tools/launchd/mission-base.sh`** by copying the code block **VERBATIM** from the design doc, Solution Design §"The helper — `tools/launchd/mission-base.sh`" (doc lines 87–140). Do NOT re-derive or "improve" it: that text already carries both R2 verbatim fixes — (a) `record()` snaps EXACTLY ONCE into `$rec` (single-read invariant, glm's double-snap-race fix); (b) `last()` uses `awk … END {if (sha) {print sha; exit 0} else exit 1}` (exits 1 on no-match); (c) `drift()` has the explicit `old=$(last "$label"); [ -n "$old" ] || { … return 2; }` no-record guard (gemini's fix). bash-3.2-safe by construction: no associative arrays, no `${v,,}`, no GNU timeout.
3. **Syntax check**: `/bin/bash -n tools/launchd/mission-base.sh` (rig bash is 3.2.57, verified `/bin/bash --version`).
4. **Create `tools/launchd/test_mission_base.sh`** modeled on `test_mission_heartbeat.sh`'s fabricate-state-in-tmpdir pattern, with these NAMED arms (each is an acceptance check — see §4):
   - `snap-format` — `snap` prints `<40-hex-sha><TAB><ISO8601-UTC>` and exits 0 for `origin/dev`.
   - `record-last-roundtrip` — `record gate1` appends exactly one `base-gate1` row to a temp `MISSION_NAME=test` `mission-test-base` file; `last gate1` returns that exact SHA.
   - `heartbeat-untouched` — the temp `mission-test-heartbeat` stays empty/absent (record stays OFF the heartbeat; protects the driver's slot-verdict reader).
   - `nonvacuity-drift-fires` — scratch clone, `record gate1` at A, `git update-ref refs/remotes/origin/dev HEAD` to move the ref to B, `drift gate1` exits **1** and prints `DRIFT` with `A -> B`.
   - `steady-control` — without the mutation, `drift gate1` exits **0** (`base gate1 steady at …`).
   - `no-record-absent-file` — no state file at all → `drift gate1` exits **2** with `no base-gate1 record yet`.
   - `no-record-missing-label` — state file exists but has no matching `base-` row → `drift gate1` exits **2** with the same message (never a false DRIFT with an empty old SHA — this is the R2 fixes' own acceptance).
   - positive grep control — `last` against a known-matching record returns it (proves the instrument runs, not vacuously green).
5. **Wire the suite**: add the line `@/bin/bash tools/launchd/test_mission_base.sh` to the `test-launchd-drivers` target in `make/test.mk` (place with the other `test_mission_*` lines, after `test_cron_kicker.sh`, BEFORE the "Keep this shell-only" comment block). **Deviation from the design doc (verified, harmless, note it in the commit-1 handoff):** the doc says to also add `/bin/bash -n tools/launchd/mission-base.sh` to the syntax loop — unnecessary: the target's existing loop `@for f in tools/launchd/*.sh tools/launchd/lib/*.sh; do /bin/bash -n "$$f" || exit 1; done` already glob-covers the new helper (measured at `make/test.mk:65`). Only the test-invocation line is needed.
6. **Run boundary gates** (§4 M1 column). All green → snapshot (§5) → hand off for commit 1/3.

### Day 2 — M2: Gate 1 + Gate 3 wiring + ratchet-clean prose (~130 LOC)

1. Re-open the design doc's Gate-by-gate §Gate 1 and §Gate 3 snippets; the edit sites in THIS worktree (verified): Gate-1 sync block at SKILL.md **446–447** (`git fetch origin` / `git rev-parse dev origin/dev`); Gate 3 section head at **§1043**; the worktree-from-`origin/dev` sites (creation rule line **479**, worktree-prose region §1518–1560, `--detach` probe note line 1249).
2. Apply the doc's verbatim Gate-1 block: `base=$(bash tools/launchd/mission-base.sh record gate1)` + `echo "Gate 1 base: $base"` immediately after the 446–447 sync.
3. Apply the doc's verbatim Gate-3 block immediately before any `git worktree add … origin/dev` in the Gate-3 region: fresh `snap`, compare against `last gate1`, `|| bash tools/launchd/mission-base.sh drift gate1`, and **create the worktree from the FRESH `$newsha`**, recording `base=$base` in the provenance/routing evidence.
4. **Pay the ratchet**: create `.claude/skills/mission-control/resources/ref-drift.md` (header convention per sibling `resources/role-spawn-routing.md`: state that the prose moved out of SKILL.md verbatim under the progressive-disclosure gate). MOVE the two-instance war stories + "why silent" + drift-on-mismatch protocol prose OUT of SKILL.md INTO it; SKILL.md keeps the operative rules inline and adds a 1-line link. **Invariant: lines added ≤ lines moved out; file ends ≤ 2781.**
5. Do NOT move/reword any S-guard literal (S1 `resolve-role-spawn.sh` / `MISSION-ROLE:` region ~1100–1107, S2 line 1118, S3/S5 region 1059) — the M2 edits are in entirely different regions (Gate 1 sync, Gate 3 worktree).
6. Run boundary gates (§4 M2 column) → snapshot §5 → hand off for commit 2/3.

### Day 3 — M3: Gate 3b pin + Gate 4 evidence + mutation re-proof (~50 LOC)

1. Apply the doc's Gate-3b wording around the EXISTING pin at SKILL.md **1742** (`target=$(git rev-parse origin/dev)  # FULL sha…`): route through `mission-base.sh record gate3b`, add the drift note (`drift gate1` advisory line) — the design ADDS record+drift around the pre-existing SHA-pin; it does not introduce SHA-pinning.
2. Apply the doc's Gate-4 edit at **2196–2200**: KEEP the existing `git fetch origin; git rev-parse dev origin/dev` re-confirmation (reuse, don't duplicate), route the value through `mission-base.sh record gate4`, and add `base=<sha>@<iso>` to the **Routing evidence** row wording. Exclusion per doc: no NEW re-read before Gate 4 — the existing re-confirmation IS the read at the moment of record.
3. **Gate 2 decision** (deferred decision, doc §Deferred): default = defer the full `base-gate2` stamp; only add the light snap-for-the-record echo if the executor judges the first live iteration proved its value. Record the decision in milestone notes.
4. **Final non-vacuity re-proof** — re-run the M1 mutation recipe end-to-end (§4 M3 column) as the milestone's LAST check.
5. Run boundary gates (§4 M3 column) → snapshot §5 → hand off for commit 3/3. The live-iteration observation (first real mission iteration emitting `base-gate1/3b/4` rows) is noted by the controller in the mission log — not a sprint file change.

## 4. Per-milestone acceptance criteria (NAMED tests/checks at each boundary)

Every boundary also requires: snapshot written (§5) and **no NEW reds** versus `.snap/H0-test-baseline.txt` (see Exemption rule below).

**M1 — helper + test**
1. `tools/launchd/mission-base.sh` exists and is byte-content equal to the design doc's Solution Design code block (modulo trailing newline) — contains `snap`, `record` (single-read invariant comment), `last` (awk `exit 1` no-match), `drift` (explicit `[ -n "$old" ]` guard).
2. `/bin/bash -n tools/launchd/mission-base.sh && /bin/bash -n tools/launchd/test_mission_base.sh` — clean under rig bash 3.2.57.
3. `grep -n 'test_mission_base.sh' make/test.mk` — hit inside the `test-launchd-drivers` target.
4. `/bin/bash tools/launchd/test_mission_base.sh` green; its named arms pass: `snap-format`, `record-last-roundtrip`, `heartbeat-untouched`, `nonvacuity-drift-fires`, `steady-control`, `no-record-absent-file`, `no-record-missing-label`.
5. **Non-vacuity mutation (design doc recipe, verbatim — the executor MUST be able to run this):**
   ```bash
   git clone --bare <this-repo> /tmp/base-fix-test.git 2>/dev/null   # or git init + a remote
   # 1. Gate 1 record: at commit A
   MISSION_NAME=test bash tools/launchd/mission-base.sh record gate1
   # 2. Simulate the SIBLING advance (no fetch of ours): move the shared ref to a new commit B
   # (glm R2 verbatim fix) — `git update-ref` is the PRIMARY mutation, not `git commit --allow-empty`,
   # since `snap` resolves refs/remotes/origin/dev, not the working-tree branch HEAD
   git update-ref refs/remotes/origin/dev HEAD
   # 3. Now the Gate-3 re-read:
   MISSION_NAME=test bash tools/launchd/mission-base.sh drift gate1
   # MUST exit 1 and print DRIFT. The control (no step 2 → drift exits 0) proves the instrument fires.
   ```
   Acceptance: mutated case exits **1** with a `DRIFT` line `…A -> B…`; control with no advance exits **0**; **both** no-record cases (state file absent; file present without a matching `base-` row) exit **2** with `no base-gate1 record yet` — never a false DRIFT with an empty old SHA.
6. CI-parity suite gate: `env -i HOME="$HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb make test-launchd-drivers` exits **0** (now including `test_mission_base.sh`).

**M2 — Gate 1/3 wiring + ratchet-clean prose**
1. Greppable anchors in SKILL.md: `grep -c 'mission-base.sh record gate1' SKILL.md` ≥ 1 (Gate 1 block); `grep -c 'mission-base.sh drift gate1' SKILL.md` ≥ 1 (Gate 3 block).
2. `test -f .claude/skills/mission-control/resources/ref-drift.md` and SKILL.md links to it (`grep -c 'resources/ref-drift.md' SKILL.md` ≥ 1) — the checker's dead-link arm (links advertised from SKILL.md must resolve) must pass.
3. **Ratchet**: `wc -l .claude/skills/mission-control/SKILL.md` → **≤ 2781** (baseline `scripts/context_docs_baseline.txt:19` pins EXACTLY 2781; growth fails; shrink is free). AND `make check-context-docs` exits **0**.
4. **S-guard literals still inline** (each `grep -c`/`-cF` on SKILL.md ≥ 1):
   - `resolve-role-spawn.sh`
   - `MISSION-ROLE:`
   - `enum in this build lists`
   - `now \`claude:claude-fable-5-1\` → \`codex:gpt-6-astra\` → \`pi:ollama/deepseek-v4-flash:0731-cloud\` → repeat` (grep -F; line-wrapped elsewhere is NOT the literal — verified exactly one full-line hit at HEAD)
   - `ASTRA IS ALSO A QUORUM REVIEWER`
5. CI-parity suite gate (scrubbed env, as M1 check 6) exits 0 — S1–S5 arms live in `test_mission_routing.sh`.
6. SKILL.md net delta ≤ 0 accounting shown in the milestone notes (added wiring lines vs moved prose lines).

**M3 — Gate 3b/4 wiring + re-proof**
1. Greppable anchors: `grep -c 'mission-base.sh record gate3b' SKILL.md` ≥ 1 and `grep -c 'mission-base.sh record gate4' SKILL.md` ≥ 1; the existing pin at 1742 and the existing fetch/rev-parse re-confirmation at 2196–2200 remain (grep still hits `target=$(git rev-parse origin/dev)` and the fetch at ~2199).
2. Gate-4 Routing-evidence wording carries `base=<sha>@<iso>`: `grep -c 'base=<sha>@' SKILL.md` ≥ 1 (or the equivalent literal the edit lands — note it).
3. Ratchet re-checked: `wc -l SKILL.md` ≤ 2781 AND `make check-context-docs` rc=0. S-guard literal greps (M2 check 4, all five) re-run.
4. CI-parity suite gate (scrubbed env) exits 0.
5. **Final non-vacuity re-proof**: re-run M1 check 5's full recipe (mutation → exit 1 + DRIFT; steady control → exit 0; both no-record cases → exit 2). This is the milestone's LAST check before the snapshot.
6. Gate-2 stamp decision recorded (adopted light snap-echo OR deferred-with-reason).

**Exemption rule (whole sprint):** ONLY reds ABSENT from `.snap/H0-test-baseline.txt` fail a boundary. Pre-existing, environmental, at HEAD `7870f40a0`: under a runner-polluted ambient env, `test_mission_routing.sh` shows 6 reds of the R6/arm12 class (induced by exported `MISSION_*`/`CONTROLLER_*` vars — scrubbed env is 77/77 green). The prompt-cited arm `run_lane fixture arm requires real lsof on Darwin CI target` (`tools/eval/test_motoko_connection_probe.sh:28`) fires ONLY on a Darwin host where `command -p -v lsof` fails; **this rig has `/usr/sbin/lsof` and the arm is GREEN here** — if it appears on the executor machine it is likewise environmental and pre-existing (CI's `macos-latest` runner ships lsof). Never fix pre-existing reds inside this sprint; report them.

## 5. Snapshot / commit protocol (executor CANNOT commit)

The executor's runner forbids git writes. After EACH milestone's gates pass:

1. Snapshot **cumulative, full post-milestone content** of every file the sprint has created or modified so far, at **RELATIVE paths under the worktree**:
   - `.snap/M1/tools/launchd/mission-base.sh`, `.snap/M1/tools/launchd/test_mission_base.sh`, `.snap/M1/make/test.mk`
   - `.snap/M2/` = all of M1's files **plus** `.snap/M2/.claude/skills/mission-control/SKILL.md`, `.snap/M2/.claude/skills/mission-control/resources/ref-drift.md`
   - `.snap/M3/` = M2's full set with final post-M3 content (SKILL.md, ref-drift.md; helper/test/test.mk carried forward).
2. Leave the worktree dirty with the same content (controller diffs snapshot vs worktree as a cross-check).
3. The **controller** builds exactly ONE commit per milestone from `.snap/M<n>/` (messages in §2), 3 commits total. The executor MUST NOT run `git add`/`git commit`/`git checkout`.
4. `.snap/` is NOT gitignored (verified: `git check-ignore .snap` → no match) — the controller decides whether snapshot dirs join the commits or are dropped; the sprint does not presume either.

## 6. Hard constraints (all measured in this worktree)

1. **Ratchet**: `.claude/skills/mission-control/SKILL.md` is at EXACTLY **2781** lines; `scripts/context_docs_baseline.txt:19` pins 2781 and `scripts/check_context_docs.sh` fails growth ("grew to N lines (baseline B)"). After EVERY milestone touching SKILL.md (M2, M3): `wc -l` ≤ 2781 AND `make check-context-docs` rc=0 (green at HEAD: "✓ context docs: 12 rules, 40 skills, CLAUDE.md — scoped, linked, within budget"). Added lines are paid by moving war-story prose into the NEW linked `resources/ref-drift.md` (sibling convention: `resources/role-spawn-routing.md`).
2. **S-guards**: the five literals listed in §4-M2-check-4 are all present at HEAD (counts verified 1/1/1/1/1) and armed by `tools/launchd/test_mission_routing.sh` S1–S5 (all 5 arms PASS at HEAD under the scrubbed gate). M2/M3 edits must not move or reword them; `make test-launchd-drivers` must stay green subject to the Exemption rule.
3. **bash 3.2.57** is the rig shell (`/bin/bash --version` → GNU bash 3.2.57(1) arm64): no associative arrays, no `${v,,}`, no GNU timeout; `/bin/bash -n` clean on every created/modified script. The CI `launchd-drivers` job (`.github/workflows/ci.yml:582`, runs `make test-launchd-drivers` at :602) asserts bash 3.x on macOS.
4. **Verbatim helper**: M1 implements EXACTLY the design doc's Solution Design code (both R2 verbatim fixes baked in: single-read invariant in `record`; `last` awk exit-1-on-no-match; explicit `-n` no-record guard in `drift`).
5. **Heartbeat/driver isolation**: base rows go to the NEW `$AILANG_STATE_DIR/mission-${MISSION_NAME}-base` file (same 5-column `epoch⇥iso⇥base-<label>⇥attempt⇥sha` format as the heartbeat, separate path). The heartbeat gains NO rows — its LAST row drives the driver's slot-verdict classifier (`mission-control.sh`: stamps 1488 / `tail -1` 1489 / label awk 1491 / `case "$RC:$_mc_slot_last"` 1492–1498, `0:gate-*→REAPED`, `*:*→CRASHED`; note: doc cites 1480–1502, actual at this HEAD is 1484–1505 — same mechanism, ±few lines drift). A trailing `base-*` row there would flip REAPED→CRASHED — the separate file exists precisely so this reader is never misclassified.
6. **No default fetch in `snap`** (deferred decision): read-only `git rev-parse` of the current shared ref; a `--fetch` flag is reserved, not built.

## 7. OUT-OF-SCOPE (explicit)

- **No changes to `.agents/skills/mission-control/SKILL.md`** — distinct 551-line hand-maintained stub (verified: 551 lines vs 2781; no sync/copy mechanism exists; `tools/skill-updates/install-skill.sh` installs only `ailang-feedback`).
- **No baseline bumps** — `scripts/context_docs_baseline.txt` (and `scripts/context_docs_links_baseline.txt`) untouched. The baseline's own header calls bumping the wrong answer; the design pays with `resources/ref-drift.md`.
- **No S-guard edits** — `tools/launchd/test_mission_routing.sh` untouched; S1–S5 literals not moved/reworded.
- **No `mission-heartbeat.sh` edits** — whitelist (`fired|gate-*|complete|abort`, lines 11–14, unknown → exit 2) untouched; base rows bypass it via the separate state file.
- **No `mission-control.sh` driver edits** — the base record goes to the new separate state file precisely so the driver (slot-verdict + retry-resume readers) is untouched.
- **No `tools/launchd/lib/pin-root.sh` / pin-reexec machinery changes** — complementary concern, reuse only.
- **No Gate-2 full `base-` stamp** unless M3's observation arm adopts it (default: deferred).
- No new timestamp scheme; reuse `date -u '+%Y-%m-%dT%H:%M:%SZ'` exactly as the heartbeat does.

## 8. Planner verification record (this worktree @ 7870f40a0)

Verified TRUE: SKILL.md=2781 / `.agents` stub=551 / baseline line 19=2781 · all five S-guard literals (each exactly 1 hit) · Gate-1 block 446–447 · Gate-2 re-fetch line 794 · Gate-3b pin line 1742 · Gate-4 re-confirmation 2196–2200 · heartbeat 5-col format + whitelist 11–14 · slot-verdict reader mechanism (`tail -1` → label → case) · `/bin/bash`=3.2.57 · `make check-context-docs` rc=0 · ci.yml:223 context-docs + ci.yml:582/602 launchd-drivers · `test-launchd-drivers` target contents (11 scripts + glob syntax loop) · existing test sizes 194/206/494/221 · `resources/` dir exists with 4 siblings · create-targets (`mission-base.sh`, `test_mission_base.sh`, `ref-drift.md`) absent · 677 commits/15d velocity inputs.

Deviations found (planned around, stated openly):
1. **`make/test.mk` syntax loop already glob-covers the new helper** — the doc's "add `/bin/bash -n … mission-base.sh` to the syntax loop" is satisfied by the existing `tools/launchd/*.sh` glob (`make/test.mk:65`); M1 adds only the test-invocation line.
2. **Slot-verdict reader line numbers drifted**: doc says 1480–1502; at this HEAD the mechanism sits at 1484–1505 (same code, same labels). Plan cites both.
3. **The prompt's claimed pre-existing red did NOT reproduce**: see §9.

## 9. Premise the planner could NOT verify

The mission brief asserted one pre-existing local red in `make test-launchd-drivers`: `run_lane fixture arm requires real lsof on Darwin CI target`. **Not reproduced at HEAD `7870f40a0` in this worktree.** That arm (`tools/eval/test_motoko_connection_probe.sh:28`) fires only when `uname == Darwin && command -p -v lsof` fails; this rig HAS `/usr/sbin/lsof`, and under a scrubbed env the ENTIRE suite is green (probe rc=0). What IS reproducible: under the mission runner's ambient exported env (`MISSION_*`/`CONTROLLER_*`), `test_mission_routing.sh` shows **6** reds (R6/arm12 class, env-induced; 77/77 green scrubbed) and `make` halts there before the probe ever runs. The plan therefore generalizes the exemption: Day-0 baseline capture + ONLY-new-reds-fail, with BOTH observed classes named. The executor must NOT treat either class as its own breakage, and must NOT "fix" them in-sprint.

---

*Plan authored by the sprint-planner role, V1 iteration 338, 2026-09-06. Design doc is the authority for all helper code and gate wording; this plan is the authority for sequencing, gates, snapshots, and commit boundaries.*
