# Sprint plan: controller classifies HTTP 402 as capacity (M-CONTROLLER-CAPACITY-ADMISSION)

**Design doc (frozen, Revision 3):** [m-controller-capacity-admission.md](m-controller-capacity-admission.md)
**Ruling:** D-FLEET-11 = **A** (Mark, attended, 2026-10-02). Narrow scope only.
**Mission:** fleet iteration 15 · P0 · ticket `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash`
**Worktree:** `/Users/voightkampff/.ailang-driver-pin/fleet-i15` (branch `fleet/i15-controller-402`).
Code base audited: origin/dev `76a5aef65`. The plan commit sits on `a44ae150c` (docs and evidence only; the
driver is byte-identical to `76a5aef65`). Work only in this worktree. Never touch the main checkout or
the pin worktree `/Users/voightkampff/.ailang-driver-pin/fleet`.
**Estimate:** about 4–6h, about 260 LOC (7 driver, about 50 chain test, about 180 new suite, 1 make line, about 15 changelog).
**Planner:** claude-opus-5-5 (planner role, unattended).

## Scope fence (frozen by the ruling — do not widen)

IN:
1. Add `|^402:` to the default `RUNTIME_QUOTA_SIG` and update the comment (four → five emitters).
2. A pause-aware final-block branch `elif [ "${MC_PAUSED:-0}" -eq 1 ]; then … log …` placed after the
   landed-record `if` and before the generic `else`.
3. Hermetic controller regression tests.

OUT, and they stay UNCHANGED: thresholds, lane order / `CONTROLLER_FALLBACK`, `_mc_load_ration` (the
billing reader), `RUNTIME_QUOTA_REWALKS`, process rc (`exit "$RC"`), the rcfail episode policy,
`lib/lane-probe.sh` logic, `.pi/extensions/**`, `scripts/mission_pi_run.sh`, `internal/coordinator`
(ClassifyFailure), schemas and the language core. Use the doc's "Proposed diff" exactly. It still
applies at HEAD: `git apply --check` gives rc 0, re-run at plan time.

### How "with `_mc_notify`" is satisfied (read before M1)

The routing brief describes the pause-aware branch as "with `_mc_notify`". The frozen design (quorum
round-1 objection 2 and its resolution) does **not** add a second `_mc_notify` call in the final block.
The pause is already announced by the existing `_mc_notify "Mission …: PAUSED — no capacity" … "pause"`
inside the PAUSE branch (`mission-control.sh:2389–2391`). A second call would announce the pause twice.
The new `elif` only logs, and it stops the generic "FAILED (rc=N) — timeout or crash" send and the
rcfail write. The contract the tests enforce is **exactly one `_mc_notify` pause notice per paused fire**.
**The executor must NOT add a `_mc_notify` call to the `elif`.** If an evaluator reads the brief
literally, this paragraph and the doc's objection-2 resolution are the answer.

## Line references (re-verified by grep at plan time; driver identical to `76a5aef65`)

| What | Where |
|---|---|
| `set -uo pipefail` | `tools/launchd/mission-control.sh:38` |
| `_mc_notify()` | :269 |
| `NOTIFY_TIMEOUT` | :870 |
| `. lib/lane-probe.sh` | :873 |
| `MC_DEMOTED=""` / `MC_PAUSED=0` | :893 / :894 |
| `_mc_canon_id` / `_mc_demote` / `_mc_is_demoted` | :898 / :905 / :907 |
| `_mc_set_controller` / `select_model` | :913 / :926 |
| fallback demote / ration checks | :1006 / :1010 |
| `HARD_TIMEOUT` / `STALL_GRACE` / `STALL_INTERVAL` | :1043 / :1050 / :1052 |
| `RUNTIME_QUOTA_SIG` / `RUNTIME_QUOTA_REWALKS` | :1077 / :1081 |
| `TRANSIENT_RETRIES` / `TRANSIENT_BACKOFF` / `TRANSIENT_SIG` | :1083 / :1084 / :1085 |
| `MC_PI_SESSION_ID` | :2224 |
| `_mc_run_once()`; text-mode `pi` launch | :2238; :2293 |
| retry loop `attempt=1` … `done` | :2364 … :2413 |
| quota check (attempt-sliced) | :2373 |
| PAUSE: `MC_PAUSED=1`, `_mc_notify` | :2387, :2389–2391 |
| SLOT VERDICT START/END; PAUSED-NO-CAPACITY | :2415/:2449; :2438 |
| SLOT NOTIFY START/END | :2453/:2468 |
| final rc block `if [ "$RC" -ne 0 ]` … `exit "$RC"` | :2470 … :2506 |
| rcfail: landed `rm` / generic marker / rc0 `rm` | :2476 / :2489 / :2503 |
| `_mc_probe_pi` ration recheck | `tools/launchd/lib/lane-probe.sh:128`, :130 |
| `_mc_load_ration` / `_mc_rung_bucket` / `_mc_is_over_ration` | `lib/lane-probe.sh:156` / :248 / :260 |
| `test-launchd-drivers` / chain-suite line | `make/test.mk:70` / :84 |
| chain-suite extraction (only 2 functions) | `tools/launchd/test_controller_chain.sh:22–23` |

After M1 the driver lines below :1077 move by +1, and lines below the final-block insertion move by
+4 more. Tests must locate code with awk patterns or markers, never with line numbers.

## Hermeticity rules (all milestones)

- `/bin/bash` 3.2.57 only: no `declare -A`, no `${v,,}`, no `mapfile`, no `|&`. Run every suite as
  `/bin/bash tools/launchd/lib/suite-env.sh <suite>`, the way `make` does.
- Zero inference. A fake `pi`, `claude`, `codex`, `ailang` and `gh` go first on `PATH` in a temp lab
  dir. Each fake appends its argv to a lab file. The fake `pi` prints a scripted per-attempt
  transcript and exits with a scripted rc.
- Clear ambient state, as `test_driver_notify.sh:66–72` does:
  `unset AILANG_STORAGE_MESSAGING AILANG_MESSAGES_PROJECT MISSION_RUNTIME_QUOTA_SIG MISSION_GH_ISSUE`
  and point `STATE_DIR`/`_mc_slot_state` at the lab.
- Extract code from the driver, never retype it. Use the awk patterns from `test_mission_kill_tree.sh:45`
  and `test_driver_notify.sh:74`: `_mc_run_once`, the loop `^attempt=1$`…`^done$` (the first `done`
  after it), the SLOT VERDICT and SLOT NOTIFY marker blocks, and the final block
  `^if \[ "\$RC" -ne 0 \]; then`…`^exit "\$RC"`. Guard every extraction: an empty extract is
  `FAIL extraction` with exit 1. Take the driver path from `MC_CAPACITY_DRIVER` (default
  `$HERE/mission-control.sh`) so the mutation drill can point at a scratch copy.
- Bounded: `MISSION_TIMEOUT=20 MISSION_STALL_GRACE=20 MISSION_TRANSIENT_BACKOFF=0
  MISSION_NOTIFY_TIMEOUT=5`. The whole suite must finish in under 60s. Any internal wait is a
  `date +%s` deadline loop, as in `test_mission_kill_tree.sh:61–80`.
- Use `/usr/bin/grep` in acceptance commands. The interactive `grep` is ugrep and does not behave like BSD grep.
- Mutation drills run on a **scratch copy** (`sed … mission-control.sh > $TMP/mut.sh`). Never use
  `git checkout -- <file>` or `git stash`.

---

## M1 — Driver change (exact diff) + pause-aware final block (about 1h)

**Files:** `tools/launchd/mission-control.sh` only (+7/−2).

Steps:
1. `git apply /path/to/extracted-doc.diff`. Extract the fenced block with
   `awk '/^```diff$/{f=1;next} /^```$/{if(f){f=0;exit}} f' design_docs/planned/m-controller-capacity-admission.md`.
   Or edit by hand to the same bytes.
2. Do not edit anything else in the driver.

Acceptance (run from the worktree root):

| # | Command | Expected |
|---|---|---|
| 1.1 | `git diff --numstat tools/launchd/mission-control.sh` | `7	2	tools/launchd/mission-control.sh` |
| 1.2 | `git diff --name-only` (M1 only) | exactly `tools/launchd/mission-control.sh` |
| 1.3 | `/bin/bash -n tools/launchd/mission-control.sh; echo $?` | `0` |
| 1.4 | `/usr/bin/grep -cF '|^429:|^402:}"' tools/launchd/mission-control.sh` | `1` |
| 1.5 | `/usr/bin/grep -nF 'elif [ "${MC_PAUSED:-0}" -eq 1 ]' tools/launchd/mission-control.sh` | one hit, after the landed-record `if` and before the generic `else` |
| 1.6 | `awk '/elif \[ "\$\{MC_PAUSED:-0\}" -eq 1 \]/,/^  else$/' tools/launchd/mission-control.sh \| /usr/bin/grep -v '^ *#' \| /usr/bin/grep -c '_mc_notify\|messages send\|rcfail'` (comments excluded: the inserted comment names `_mc_notify` and rcfail, so an unfiltered count is 2) | `0`: no second notice, and the rcfail marker is untouched |
| 1.7 | `/usr/bin/grep -cF '_mc_notify "Mission ${MISSION_NAME}: PAUSED' tools/launchd/mission-control.sh` | `1`: still the single pause announcer |
| 1.8 | Signature probe, positive. `S=$(sed -n 's/^RUNTIME_QUOTA_SIG="\${MISSION_RUNTIME_QUOTA_SIG:-\(.*\)}"$/\1/p' tools/launchd/mission-control.sh)`, then `printf '402: {"message":"x"}\n' \| /usr/bin/grep -qE "$S"; echo $?` | `0` |
| 1.9 | **NEGATIVE** probes with the same `$S`: `the provider returned 402: credits`, `  402: indented`, `HTTP 402 credits quota`, `insufficient credits`, `4020: x` | each prints `1`. Prose never classifies |
| 1.10 | **Unchanged** probes with `$S`: `429: {"x"}`, `You hit your usage limit`, `reached your session usage limit`, `Claude usage limit reached` | each prints `0`. The existing four emitters still match |
| 1.11 | `/usr/bin/grep -cF 'RUNTIME_QUOTA_REWALKS="${MISSION_RUNTIME_QUOTA_REWALKS:-4}"' tools/launchd/mission-control.sh; /usr/bin/grep -c '^exit "\$RC"$' tools/launchd/mission-control.sh` | `1` and `1`. Bound 4 and rc passthrough unchanged |

## M2 — Hermetic regression suites + make wiring (about 2.5–3h)

**Files:**
- `tools/launchd/test_controller_chain.sh` (about +50). Extract `_mc_canon_id`, `_mc_demote` and
  `_mc_is_demoted` from the driver. Extract `_mc_rung_bucket` and `_mc_is_over_ration` from
  `lib/lane-probe.sh`. Stub `_mc_load_ration` so it only sets `MC_OVER_RATION` from a test variable,
  because the billing reader stays out of scope and untested here. Add the new cases and an
  anti-vacuity stderr assertion.
- `tools/launchd/test_controller_capacity.sh` (new, about 180). Runs the real `_mc_run_once` + retry
  loop + slot verdict + slot notify + final block against a fake `pi`.
- `make/test.mk` (+1). Add `@$(LAUNCHD_SUITE) tools/launchd/test_controller_capacity.sh` directly after
  :84 (`test_controller_chain.sh`).

Fixture layout (delegated detail, fixed contract). A per-case controller chain made of `pi:` rungs
only (for example `CONTROLLER_FALLBACK=pi:openrouter/a,pi:openrouter/b,pi:openrouter/c`), with every
Anthropic and codex probe stubbed to fail. The fake `pi` reads `$LAB/script` and picks a
`<model> <attempt-index>` row that says what to print and which rc to return. The fake `ailang` logs
`--title` values to `$LAB/sends`. `MC_PAUSED`, `MC_DEMOTED` and the rcfail marker
(`$STATE_DIR/mission-<name>-rcfail.episode`) are checked after the final block runs. The final block
calls `exit`, so run the extracted sequence in a subshell and capture its rc.

### Cases in `test_controller_capacity.sh` (each prints `PASS <name>` / `FAIL <name>`)

| Case | Scenario | Must observe |
|---|---|---|
| `402-rewalk-then-complete` | rung a: `402: {"message":"…credits…"}` with rc 1. Rung b: rc 0 and a heartbeat stamp `complete` | `MC_DEMOTED` = ` pi:openrouter/a`; one log line `controller re-walk 1/4: pi:openrouter/a → pi:openrouter/b`; verdict `COMPLETED`; 0 sends titled `FAILED`/`PAUSED`; rc 0 |
| `402-all-rungs-pause` | every rung emits `402:` with rc 1 | demotes a, b and c, one per rung, with no same-rung retry (fake-pi argv shows each model exactly once); verdict `PAUSED-NO-CAPACITY`; **exactly 1** send whose title contains `PAUSED — no capacity`; **0** sends titled `Mission iteration FAILED`; rcfail marker is **byte-identical** before and after (run absent→absent and `1`→`1`); process rc = the original nonzero (1) |
| `402-rewalk-bound` | `MISSION_RUNTIME_QUOTA_REWALKS=1`, every rung 402 | exactly 1 re-walk, then pause; same notice counts as above |
| `prior-402-no-poison` (NEGATIVE) | rung a: 402, rc 1. Rung b: plain `boom` with no signature, rc 1 | `MC_DEMOTED` contains only a, **not** b; verdict `CRASHED at=…`; **exactly 1** `Mission iteration FAILED (rc=1)` send; 0 pause sends; marker = `1` |
| `prose-402-not-capacity` (NEGATIVE) | rung a prints `the provider returned 402: credits exhausted` and `HTTP 402 insufficient credits` (not at line start), rc 1 | not demoted; no re-walk line; verdict `CRASHED`; exactly 1 generic FAILED send; 0 pause sends |
| `generic-crash-once` (NEGATIVE) | rc 1, no signature, marker absent; then the same fire again with marker = `1` | first run: 1 generic send, marker written `1`. Second run: 0 sends and a `notice suppressed` log line. Episode gate intact |
| `generic-crash-mcpaused-unset` | final block alone, run under `set -uo pipefail` with `MC_PAUSED` **unset** and RC=1 | exactly 1 generic send; **0 bytes** on stderr |
| `429-unchanged` | rung a: `429: {…}`, rc 1; rung b completes | identical to `402-rewalk-then-complete` (demote a, re-walk, COMPLETED) |
| `usage-limit-unchanged` | rung a: `You've hit your usage limit`, rc 1, all rungs the same | pause with 1 pause notice and 0 FAILED sends (pause-aware branch now covers the existing emitters too) |
| `transient-unchanged` | rung a: `API Error: Overloaded`, rc 1, then rc 0 on the 2nd attempt | same rung retried (argv shows a twice); no demotion; COMPLETED |
| `rc0-old-402-normal` | rung a prints `402: {…}` but exits **0** and stamps `complete` | no demotion; COMPLETED; 0 sends; marker removed |
| `landed-record-precedence` | every rung 402 (pause), but the mission log gains a `## ` record during the run | the landed-record notice (`killed post-record`) is sent; no generic FAILED; marker removed. `if` wins over `elif` |
| `watchdog-kill-stays-killed` | fake pi exits 143 after printing `402: {…}` | loop breaks before the quota check; no demotion; verdict `KILLED at=…` |

### Cases added to `test_controller_chain.sh`

| Case | Must observe |
|---|---|
| `all-blocked-no-controller` | `MC_OVER_RATION="codex ollama anthropic openrouter"`, so every bucket is blocked: `select_model` returns 1, `CONTROLLER_ID` empty, **0 probe invocations** (the probe stubs count calls) |
| `openrouter-blocked-reaches-next` | chain `pi:openrouter/z-ai/glm-5.3,pi:ollama/glm-5.3:cloud`, `MC_OVER_RATION=openrouter`, ollama probe OK: selects `pi:ollama/glm-5.3:cloud`; the openrouter probe is invoked 0 times |
| `demoted-rung-skipped` | `MC_DEMOTED=" pi:openrouter/z-ai/glm-5.3"`: that rung is skipped and the next eligible rung is chosen |
| anti-vacuity | `/bin/bash tools/launchd/test_controller_chain.sh 2>"$e"; grep -c 'command not found' "$e"` gives `0`. At HEAD it is `63` (21 × 3 helpers, re-measured at plan time), so this case is red before the change |

### M2 acceptance

| # | Command | Expected |
|---|---|---|
| 2.1 | `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_controller_capacity.sh; echo rc=$?` | every case `PASS`, `rc=0`, wall time < 60s |
| 2.2 | `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_controller_chain.sh 2>/tmp/cc.err; echo rc=$?; grep -c 'command not found' /tmp/cc.err` | `rc=0`; `0` |
| 2.3 | `grep -n 'test_controller_capacity.sh' make/test.mk` | one hit, the line after `test_controller_chain.sh` |
| 2.4 | `grep -nE 'declare -A\|\$\{[a-zA-Z_]+,,\}\|mapfile\|readarray' tools/launchd/test_controller_capacity.sh tools/launchd/test_controller_chain.sh` | no output (bash 3.2) |
| 2.5 | Zero inference. Run the suite with `PATH` limited to `$LAB/bin:/usr/bin:/bin` and assert the fakes' argv logs are the only `pi`/`ailang`/`gh` invocations, as `command -v pi` inside the suite resolves to `$LAB/bin/pi` | asserted inside the suite as a `PASS hermetic-path` case |
| 2.6 | `pgrep -f 'test_controller_capacity'` 30s after the suite exits | no output. Watchdog `sleep`s are bounded by `MISSION_TIMEOUT=20` |

### M2 mutation drill (each mutant must turn the named case RED; scratch copies only)

Template: `sed '<edit>' tools/launchd/mission-control.sh > "$TMPDIR/mut.sh"` then
`MC_CAPACITY_DRIVER="$TMPDIR/mut.sh" /bin/bash tools/launchd/test_controller_capacity.sh; echo rc=$?`.

| Mutant | Edit | Must FAIL (rc ≠ 0) |
|---|---|---|
| **MUT-402 (required)** revert the arm | `s/|\^402:}"/}"/` on the `RUNTIME_QUOTA_SIG=` line | `402-rewalk-then-complete`, `402-all-rungs-pause`, `402-rewalk-bound` |
| MUT-ELIF drop pause branch | delete the 4 inserted `elif … log` lines | `402-all-rungs-pause` (generic FAILED send appears, marker written) |
| MUT-UNSET bare variable | `s/"\${MC_PAUSED:-0}"/"$MC_PAUSED"/` on the elif line | `generic-crash-mcpaused-unset` (aborts under `set -u`, 0 sends) |
| MUT-SLICE un-slice the log | `/RUNTIME_QUOTA_SIG"; then$/s/tail -n +\$((logpos + 1)) "\$LOG"/cat "$LOG"/` (quota line only; transient line untouched) | `prior-402-no-poison` |
| MUT-LOOSE loosen the anchor | `s/|\^402:}"/|402}"/` | `prose-402-not-capacity` |
| MUT-CHAIN (chain suite) | point the chain suite's extraction at a copy with the :1010 ration `continue` removed | `openrouter-blocked-reaches-next` / `all-blocked-no-controller` |

Record each mutant's rc and the failing case names in the M3 evidence. Every mutant that stays green
is a gap to close before M3, not a note.

## M3 — Done gate: suite, dry-runs, changelog, evidence (about 1h)

**Files:** `changelogs/unreleased/2026-10-03-controller-402-capacity.md` (new, `### Fixed — …` section).
No other files.

| # | Command | Expected |
|---|---|---|
| 3.1 | `make test-launchd-drivers; echo rc=$?`. Run it **unpiped**: a piped make reports the pipe's rc | `rc=0` under `/bin/bash` 3.2.57; output includes the new suite's `PASS` lines |
| 3.2 | `/bin/bash tools/launchd/test_mission_heartbeat.sh` | rc 0, `PASS: 25 heartbeat arms ran` (doc V10, unchanged) |
| 3.3 | Healthy dry-run: `SHA=$(git rev-parse HEAD); AILANG_DRIVER_PINNED=$SHA MISSION_PROFILE=fleet MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh`, bounded by a `date +%s` deadline ≤ 30 min (in practice < 2 min) | log line `DRY RUN ok: … lanes=ok …` (or `lanes=DEGRADED(...)` only if a real lane is down; record which) and rc 0 |
| 3.4 | Degraded dry-run: same, plus `MISSION_EXECUTOR_MODEL=codex:bogus MISSION_PROBE_TIMEOUT=10` | `DRY RUN ok: … lanes=DEGRADED(…)…` and rc 0 |
| 3.5 | `ls changelogs/unreleased/2026-10-03-controller-402-capacity.md && grep -c '^### Fixed' changelogs/unreleased/2026-10-03-controller-402-capacity.md` | file exists; `≥1`. Text states: `^402:` joins the runtime-capacity signature; a pause no longer also posts a crash notice or writes the rcfail marker; the ration half of the ticket was **not reproduced** at HEAD (no guard added) |
| 3.6 | `make check-changelog` | rc 0 |
| 3.7 | `git diff --name-only 76a5aef65 -- . ':!design_docs' ':!.ailang'` | exactly: `changelogs/unreleased/2026-10-03-controller-402-capacity.md`, `make/test.mk`, `tools/launchd/mission-control.sh`, `tools/launchd/test_controller_capacity.sh`, `tools/launchd/test_controller_chain.sh` |
| 3.8 | `git diff 76a5aef65 -- tools/launchd/lib .pi scripts/mission_pi_run.sh internal \| wc -l` | `0` |
| 3.9 | Mutation table from M2 recorded with the rc of each mutant | MUT-402 at least is RED; all six are RED |

Notes for M3:
- The dry-run (3.3/3.4) exits at :1907, before the retry loop. It proves that the edited driver
  parses, wires and routes under the pin. It cannot exercise the 402 path; only M2 does that. The
  dry-run's probes fire real ~1-reply-token requests (driver comment :1899). That is the only
  inference in the sprint. It is a done-gate rule from the doc, not a test.
- Mission-loop-change pre-flight (doc acceptance): no reload or plist change mid-iteration; the
  change reaches the rig only through the pinned origin/dev after merge. Do not push from the
  executor. The controller lands the change.
- Ticket disposition happens only after the change is on origin/dev, and it states that the ration
  half was not reproduced.

## Risks

- **Brief vs doc wording on `_mc_notify`.** Resolved above: one pause notice, sent from the existing
  call. An `_mc_notify` added to the `elif` is a defect, and `402-all-rungs-pause` (exactly 1 pause send)
  fails on it.
- **Extraction drift.** The retry loop is top-level code, not a function. If a later edit adds a
  `done` between `attempt=1` and the loop end, the awk range could cut short. Guard: assert that the
  extracted loop contains both `RUNTIME_QUOTA_SIG` and `TRANSIENT_SIG`, otherwise `FAIL extraction`.
- **`_mc_notify` same-day dedupe/spool.** Point its state at the lab dir per case. Otherwise case 2's
  dedupe hides case 3's notice and the counts go vacuous.
- **Orphan watchdog sleeps.** `_mc_run_once` forks `sleep $HARD_TIMEOUT`. Keep `MISSION_TIMEOUT` small
  (20) so orphans die inside the bound (acceptance 2.6).
