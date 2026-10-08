# Sprint plan — M-FLEET-GATE0-SELF-NOTICES: port World's Gate-0 self-notice read into the shared harness

- Status: **planned** (fleet-mission iteration 22, unattended planner; nothing implemented)
- Ticket: `gate0:driver-crash-notices-invisible` — sunholo-data/ailang#1160
- Base: `origin/dev` `e68a264fb`, worktree `.wt-fleet-iter22-gate0-self-notices`, branch `fleet/iter22-gate0-self-notices`
- Source design (judged + landed in World): `ailang-world` `origin/dev` (`1bf36bd` at planning time)
  `design_docs/implemented/w-gate0-blind-to-its-own-crash-notices.md` (D1–D7a, floors F0–F10, §11 landing record)
- Size: ~0.5 day, ~520 LOC of which ~430 is a port. Four milestones, serial.
- Write scope (pre-push guard): `scripts/mission_*`, `scripts/test_mission_*`, `tools/launchd/**`,
  `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, `make/test.mk`, `design_docs/**`,
  `changelogs/**`. **`scripts/testdata/` is out of scope** — fixtures go to
  `scripts/mission_gate0_self_notices_testdata/`.

## 1. Problem (verified against this tree)

- The driver posts its crash notice as the loop's own account:
  `tools/launchd/mission-control.sh:2513-2514` —
  `gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" --body "⚠️ Mission iteration **FAILED to complete** (rc=$RC — timeout or crash) …"`.
  Bytes of the prefix: `e2 9a a0 ef b8 8f` = U+26A0 U+FE0F, identical to World's literal (`grep -cF` of
  World's `SIG_TEXT` against this driver = 1).
- Gate 0 step 6 reads the bookkeeping issue only via `scripts/mission_directives.sh`
  (`.claude/skills/mission-control/resources/gate-0-preflight.md:108-131`), whose allowlist defaults to
  `MarkEdmondson1234` and refuses the authenticated account (`scripts/mission_directives.sh:46-60`).
  So a fire that died is invisible to the next fire's Gate 0, by construction.
- The rotation case is real here too: the driver resolves `MISSION_GH_ISSUE` once at start
  (`mission-control.sh:1122`), so a fire that rotates the issue at Gate 5 and then dies posts its
  notice to the **old** issue. Rotation writes `mission-${MISSION_NAME}-gh-issue-prev`
  (`.claude/skills/mission-control/resources/gate-5-retro.md:196-197`).
- Measured on `sunholo-data/ailang` (planning time, author-filtered, anchored):
  `#1380` (fleet `-prev`, CLOSED) 26 comments / REST 26, **3** notices
  (`2026-10-02T02:18:21Z rc=1`, `2026-10-02T21:56:53Z rc=1`, `2026-10-04T04:28:01Z rc=143`);
  `#852` (V1, CLOSED) 84 comments / REST 84, **10** notices; `#1072` 1; `#987`, `#1089`, `#745` 0.
  None of the 3 notices on #1380 was ever visible to a fleet Gate 0.

## 2. Premises in the ticket that are false or need adjusting (planner findings)

1. **World's second watermark has no shared equivalent.** World passes `mission-world-last-seen` (a
   mission-scoped, NON-rotating file) beside `mission-${ISSUE}-last-seen`. In `~/.ailang/state/`
   only `mission-world-last-seen` and `mission-docs-last-seen` exist; v1/fleet/motoko/stapledon have
   none. The shared rule therefore passes `mission-${ISSUE}-last-seen` and
   `mission-${PREV}-last-seen`. First fire after rotation: current file ABSENT → DEGRADED single read
   off the prev watermark (catches the rotate-then-die notice). Residual (declared, §7): a notice on
   the prev issue newer than the prev's final watermark is re-reported every fire of the `-prev`
   week; the rule makes the log credit idempotent by `url=`.
2. **World's script carries the signature TWICE** — bash `SIG_TEXT` and a `\u`-escaped python `SIG`
   (World `scripts/gate0_self_notices.sh` lines `SIG_TEXT=…` and `SIG = "⚠️…"`). Its F8
   `--driver-src` grep guards only the bash copy; the python copy (the one that classifies) can drift
   silently. The port keeps ONE literal (bash) and passes it to python via argv.
3. **`--control 107:4` is ailang-world data.** Missions post to three repos
   (`~/.config/ailang/mission-*.env`: `sunholo-data/ailang` for v1/fleet/motoko/docs,
   `sunholo-data/ailang-world`, `sunholo-data/stapledons-godot`). The shared rule needs a per-repo
   control → `--control auto` resolves from a table in the script; unknown repo = F6 (no fallback).
4. **The script cannot be cwd-relative.** World/stapledon controllers run in their own repos; World's
   tree has no `scripts/mission_directives.sh` (`git ls-tree origin/dev scripts/` in ailang-world).
   The driver exports `AILANG_MISSION_REGISTRY="$MC_DRIVER_ROOT/missions"` to children
   (`mission-control.sh:1968-1974`, called at `:1986` and `:2178`), so the rule derives
   `MC_ROOT="$(dirname "$AILANG_MISSION_REGISTRY")"` and calls the script and `--driver-src` by
   absolute path from the pinned driver tree.
5. **The FAILED notice is not the driver's only death-shaped notice.** `mission-control.sh:2474-2475`
   posts `⚠️ Mission slot verdict: **<verdict>** (rc=…` when a slot is reaped. Out of scope for
   this sprint (ticket names :2514 only; World scoped only the FAILED prefix); filed as follow-up F-2.
6. **Episode gating hides repeats:** identical-rc failures post once per episode
   (`mission-control.sh:2504-2510`). The instrument sees the first death of an episode, not the
   count. Declared residual; the driver log is the complete record.

## 3. Design decisions for the port (deltas from World only)

| # | World | Shared port | Why |
|---|---|---|---|
| P1 | `scripts/gate0_self_notices.sh` | `scripts/mission_gate0_self_notices.sh` | write scope + house naming (`scripts/mission_directives.sh`) |
| P2 | two SIG literals (bash + python `\u`) | one bash `SIG_TEXT`, passed to python as `sys.argv[4]` | finding 2 |
| P3 | `--control <issue>:<count>` mandatory | still mandatory; adds `--control auto` = table keyed by `--repo`; unknown repo → `✗ control required: no control for repo <r> (pass --control <issue>:<count>)` rc 2 | finding 3 |
| P4 | verdict suffix `gate0_self_notices.sh, …` | `mission_gate0_self_notices.sh, …` | name |
| P5 | suite scratch `$(pwd)/.g0_scratch.XXXXXX` | `mktemp -d "${TMPDIR:-/tmp}/g0_scratch.XXXXXX"` | never litter the shared checkout |
| P6 | fixtures `scripts/testdata/gate0_self_notices_*` | `scripts/mission_gate0_self_notices_testdata/{107,129}{,.meta}.json` copied byte-for-byte + NEW captured `1380{,.meta}.json` from `sunholo-data/ailang` | write scope; one fixture from THIS driver's repo |

Everything else is kept verbatim: exit codes **0** (no in-window notice, controls ok), **1** (≥1
in-window notice — A FIRE DIED), **2** (any floor F0–F10); floors F0–F10 with World's exact
messages; bash 3.2 only (no `declare -A`, `${x,,}`, `{n}`); `run_bounded` gh wrapper (default 5 s);
F5 REST-count truncation check; no comment body ever printed; anchored prefix + ASCII digits;
case-insensitive author match; D7a strict watermark validation with the BSD/GNU parser control.

Control table at planning time (executor MUST re-measure each value with the command in AC-M1-6
before committing; a mismatch is recorded, not smoothed):

```
sunholo-data/ailang        852:10   # V1 thread, CLOSED, 84 comments
sunholo-data/ailang-world  107:4    # World's published control (design D5)
sunholo-data/stapledons-godot  4:<measure>   # CLOSED; unfiltered count was 5 — author-filter it
```

## 4. Milestones

### M1 — Port the instrument (`scripts/mission_gate0_self_notices.sh`) + fixtures (~380 LOC)

Steps: `git -C ~/dev/sunholo-data/ailang-world show origin/dev:scripts/gate0_self_notices.sh >
scripts/mission_gate0_self_notices.sh`; apply P2–P4; `chmod +x`. Copy the four World fixtures with
`git show … >` (never retype). Capture `#1380`:
`gh issue view 1380 --repo sunholo-data/ailang --json comments > …/1380.json` and
`gh api repos/sunholo-data/ailang/issues/1380 | python3 -c 'import json,sys; print(json.dumps({"comments": json.load(sys.stdin)["comments"]}))' > …/1380.meta.json`
(meta reduced to the one field the parser reads, as World's 17-byte metas are). Record sha256 of all
six fixture files in the commit message.

Acceptance (each runnable from the worktree root):
- AC-M1-1 `test -x scripts/mission_gate0_self_notices.sh && /bin/bash -n scripts/mission_gate0_self_notices.sh`
- AC-M1-2 one literal: `grep -n 'FAILED to complete' scripts/mission_gate0_self_notices.sh | grep -vE '^[0-9]+: *#'` → exactly one line (the `SIG_TEXT=` assignment); `grep -c 'u26a0' scripts/mission_gate0_self_notices.sh` → 0
- AC-M1-3 bash-3.2 hygiene: `grep -nE 'declare -A|\$\{[a-zA-Z_]+,,\}|mapfile|readarray' scripts/mission_gate0_self_notices.sh` → no output
- AC-M1-4 fixtures verbatim: `for f in 107.json 107.meta.json 129.json 129.meta.json; do cmp <(git -C ~/dev/sunholo-data/ailang-world show origin/dev:scripts/testdata/gate0_self_notices_$f) scripts/mission_gate0_self_notices_testdata/$f || echo DRIFT $f; done` → no output
- AC-M1-5 1380 fixture is real and complete: `python3 -c 'import json;d=json.load(open("scripts/mission_gate0_self_notices_testdata/1380.json"));m=json.load(open("scripts/mission_gate0_self_notices_testdata/1380.meta.json"));print(len(d["comments"]),m["comments"])'` → `26 26` (or the re-measured equal pair)
- AC-M1-6 control values re-measured live (bounded, read-only):
  `for r in "sunholo-data/ailang 852" "sunholo-data/ailang-world 107" "sunholo-data/stapledons-godot 4"; do set -- $r; gh issue view $2 --repo $1 --json comments --jq '[.comments[]|select((.author.login|ascii_downcase)=="sunholo-voight-kampff" and (.body|startswith("⚠️ Mission iteration **FAILED to complete** (rc=")))]|length'; done` — each printed count equals the table entry
- AC-M1-7 `--help` exits 0 and names exit codes 0/1/2: `/bin/bash scripts/mission_gate0_self_notices.sh --help | grep -c '^  [012] '` → 3

### M2 — Suite `scripts/test_mission_gate0_self_notices.sh` + make wiring (~150 LOC delta over the port)

Port World's 24-arm suite (`git show … scripts/test_gate0_self_notices.sh`), retarget `SCRIPT_UT`
and fixture paths, apply P5. ADD four arms:

- `driver-literal` — runs the instrument with `--driver-src tools/launchd/mission-control.sh` (the
  REAL file, repo-relative) over a 1-comment stub and asserts `signature-source: tools/launchd/mission-control.sh ok`
  and rc 0. Then extracts the driver's literal independently:
  `grep -o '⚠️ Mission iteration \*\*FAILED to complete\*\* (rc=' tools/launchd/mission-control.sh | head -1`
  and asserts it equals the script's `SIG_TEXT` value read with `sed -n 's/^SIG_TEXT="\(.*\)"$/\1/p'`.
  This is the drift guard: changing either side reds it.
- `single-literal` — asserts the python block receives the signature from argv (feeds a stub whose
  only notice uses the literal and runs the instrument with `SIG_TEXT` mutated in a scratch COPY of
  the script; the copy must classify 0 → proves python reads the bash value, not its own).
- `control-auto` — three runs: `--repo sunholo-data/ailang --control auto` resolves `852:10`
  (stub serves `852` fixture built from a heredoc with 10 notices; asserts `control: issue=852 expect=10 got=10 ok`);
  `--repo test/unknown --control auto` → rc 2, stderr contains `✗ control required: no control for repo test/unknown`,
  stub argv log EMPTY (pre-network).
- `snapshot-1380` — real `sunholo-data/ailang` fixture, `--since 2026-10-01T00:00:00Z --control 1380:3`:
  rc 1; exactly three `crash: issue=1380` lines with the at/rc pairs in §1; `issue: 1380 comments=26 self=<measured> crash=3 other=<measured-3>`.

Wire after `scripts/test_mission_pi_run.sh` in `make/test.mk:76`:
`@$(LAUNCHD_SUITE) scripts/test_mission_gate0_self_notices.sh`.

Acceptance:
- AC-M2-1 `/bin/bash scripts/test_mission_gate0_self_notices.sh` exits 0 and its last line is `N passed, 0 failed` with N ≥ 88 + the new arms' assertions (record N)
- AC-M2-2 `grep -n 'test_mission_gate0_self_notices.sh' make/test.mk` → exactly one line, inside the `test-launchd-drivers` recipe
- AC-M2-3 `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` exits 0 (run UNPIPED — a piped make reports the pipe's rc)
- AC-M2-4 no scratch left behind: `git status --porcelain | grep -c g0_scratch` → 0 after AC-M2-1
- AC-M2-5 no network: with a `gh` shim that always fails placed FIRST on PATH, the suite still reports 0 failed (every arm uses `--gh-bin` stubs):
  `T=$(mktemp -d); printf '#!/bin/sh\nexit 99\n' > $T/gh; chmod +x $T/gh; PATH="$T:$PATH" /bin/bash scripts/test_mission_gate0_self_notices.sh | tail -1; rm -rf $T` → `N passed, 0 failed`

### M3 — Gate-0 rule (both skill copies) + changelog (~25 LOC)

Insert as step **6a** in `.claude/skills/mission-control/resources/gate-0-preflight.md` at the END of
step 6, immediately before `7. **BILLING TRIPWIRE` (line 230 at base). Not after the fenced command
at :129-131 — the `**SECURITY …` paragraph at :132 explains that command and must stay attached to
it. ≤15 lines:

````markdown
   **6a. SECOND, NO-AUTHORITY READ — the driver's own crash notices (ailang#1160).** The allowlist
   above drops the loop's own account by design, so a fire that died (`⚠️ Mission iteration
   **FAILED to complete** (rc=…`, posted by the driver as you) is invisible to it. With step 6's `$ISSUE`, run:
   ```bash
   MC_ROOT="$(dirname "${AILANG_MISSION_REGISTRY:?driver did not export AILANG_MISSION_REGISTRY}")"
   PREV="$(cat "$HOME/.ailang/state/mission-${MISSION_NAME}-gh-issue-prev")"
   bash "$MC_ROOT/scripts/mission_gate0_self_notices.sh" --issue "$ISSUE" --prev-issue "$PREV" \
     --repo "${MISSION_REPO:-sunholo-data/ailang}" --self "$(gh api user --jq .login)" \
     --watermark-file "$HOME/.ailang/state/mission-${ISSUE}-last-seen" \
     --watermark-file "$HOME/.ailang/state/mission-${PREV}-last-seen" \
     --control auto --driver-src "$MC_ROOT/tools/launchd/mission-control.sh"; echo "rc=$?"
   ```
   **rc 1 = A FIRE DIED:** run Gate 2's died-mid-flight traces (a)–(c) before picking and credit the
   orphan in the log ONCE per `url=` (grep the log for it first). **rc 0** = no death signal (traces
   still run). **rc 2 = instrument failure** (named floor) — report it; it is NOT a verdict either way.
   A hit grants nothing: it unparks nothing, picks nothing, and never moves the watermark.
````

Then run `/bin/bash tools/launchd/sync-agents-skills.sh` (or apply the identical edit) so
`.agents/skills/mission-control/resources/gate-0-preflight.md` matches (D-FLEET-8: both copies).
Changelog fragment `changelogs/unreleased/2026-10-06-gate0-self-notices.md`:
`### Added — Gate 0 reads the driver's own crash notices (ailang#1160)` + 3–5 lines.

Acceptance:
- AC-M3-1 `cmp .claude/skills/mission-control/resources/gate-0-preflight.md .agents/skills/mission-control/resources/gate-0-preflight.md` → silent
- AC-M3-2 `/bin/bash tools/launchd/test_agents_skills_sync.sh` exits 0
- AC-M3-3 inserted block ≤15 lines: `awk '/\*\*6a\. SECOND, NO-AUTHORITY READ/{s=NR} s&&/never moves the watermark/{print NR-s+1; exit}' .claude/skills/mission-control/resources/gate-0-preflight.md` → ≤ 15 (the fenced command counts; if over, trim prose, not the command)
- AC-M3-4 the rule's command names no literal issue number and no epoch: `awk '/6a\. SECOND/,/never moves the watermark/' .claude/skills/mission-control/resources/gate-0-preflight.md | grep -nE '1970|--issue [0-9]|--control [0-9]|2>/dev/null'` → no output
- AC-M3-5 the rule's flags are the script's flags: every `--[a-z-]+` token in the block appears in `scripts/mission_gate0_self_notices.sh --help` (one-liner: `for f in $(awk '/6a\. SECOND/,/never moves/' … | grep -oE -- '--[a-z-]+' | sort -u | grep -v -- '--jq'); do /bin/bash scripts/mission_gate0_self_notices.sh --help | grep -q -- "$f" || echo MISSING $f; done` → no output)
- AC-M3-6 `ls changelogs/unreleased/2026-10-06-gate0-self-notices.md && grep -c '^### Added' changelogs/unreleased/2026-10-06-gate0-self-notices.md` → ≥1; `make check-changelog` exits 0

### M4 — Live reading on the fleet thread (recorded, not predicted) (0 LOC)

Run the M3 command with fleet values (`MISSION_NAME=fleet`, `ISSUE=$(cat ~/.ailang/state/mission-fleet-gh-issue)`,
`AILANG_MISSION_REGISTRY=$PWD/missions`, `MISSION_REPO=sunholo-data/ailang`) from the worktree.
- AC-M4-1 rc ∈ {0,1}; output contains `control: issue=852 expect=10 got=10 ok`, `signature-source: <path> ok`, and one `issue:` line each for the current issue and `1380`. Paste the verdict line and rc into the PR body verbatim. Expected-at-planning (NOT an acceptance value): the 3 notices on #1380 classify; whether they are in-window depends on `mission-1380-last-seen`.
- AC-M4-2 `--control auto` with `--repo sunholo-data/ailang-world --issue 107` (World's closed control) reads `control: issue=107 expect=4 got=4 ok` — proves the table entry for a second repo on live data.

## 5. Mutation drills (executor MUST run each; record red set)

Mutate a **scratch copy** (`cp scripts/mission_gate0_self_notices.sh "$T/ut.sh"`, point the suite's
`SCRIPT_UT` at it via a one-line env override added for drills, e.g. `SCRIPT_UT="${G0_SCRIPT_UT:-scripts/mission_gate0_self_notices.sh}"`).
Never `git checkout <file>` and never `sed > tmp && mv` over the real script (World §11: that
dropped the exec bit). After all drills: `shasum -a 256` of the script equals the pre-drill digest
AND `test -x` passes.

| # | Mutation | Must red (at least) |
|---|---|---|
| D1 | drop the author check (`login.lower() == selfs and` removed) | `self-filter` |
| D2 | drop the anchor: `body.startswith(SIG)` → `SIG in body` AND slice at `body.find(SIG)` | `anchored-signature` |
| D3 | case-sensitive author (`.lower()` removed on login) | `self-filter` |
| D4 | change the prefix in the DRIVER: scratch copy of `tools/launchd/mission-control.sh` with `**FAILED to complete**` → `**FAILED**`, run `driver-literal` against it (`G0_DRIVER_SRC` override) | `driver-literal` |
| D5 | change `SIG_TEXT` in the script (drop the `**`) | `driver-literal`, `anchored-signature`, `snapshot-1380` |
| D6 | python ignores argv and hard-codes its own SIG (World's shape) then change `SIG_TEXT` | `single-literal` |
| D7 | `--control auto` falls back to `0` count for unknown repo | `control-auto` |
| D8 | `--prev-issue` ignored | `prev-issue` |
| D9 | F5 truncation check removed | `truncated` |
| D10 | watermark ceiling (`now+SKEW`) inverted | `watermark-strict` |
| D11 | drop the make wiring line | AC-M2-2 (grep) |

A drill that reds nothing is a finding to fix (add an assertion), not a note.

## 6. Out of scope / follow-ups

- **F-1 (driver):** reword the notice's reassurance ("The queue is untouched; the next interval
  will retry") into a Gate-2 trigger — a driver text change needing `--dry-run`s and the
  mission-loop-change skill; keep the PREFIX byte-identical (the `driver-literal` arm enforces it).
- **F-2 (driver/instrument):** classify `⚠️ Mission slot verdict: **…** (rc=` (`mission-control.sh:2475`)
  as a second death signature.
- **F-3:** a mission-scoped non-rotating watermark for v1/fleet/motoko/stapledon (World pattern) would
  remove the `-prev`-week re-report residual.
- **F-4:** World can retire its local copy in favour of the shared script once this lands (World's call).
- `mission-${ISSUE}-last-seen` is keyed by issue number across three repos (e.g. `mission-4-last-seen`
  could be stapledon or world); pre-existing, not touched here.

## 7. Risks and declared residuals

- `-prev` week re-report (finding 1) — mitigated by url-idempotent credit in the rule.
- Episode-gated repeats are invisible (finding 6).
- `--self "$(gh api user --jq .login)"` with gh down yields empty → F0 by name (loud, no fallback).
- A missing `-prev` file makes `cat` print its error and hands the script an EMPTY `--prev-issue`.
  World's F0 only digit-checks a NON-empty `--prev-issue` (`if [ -n "$PREV_ISSUE" ]; then case …`),
  so an explicitly empty value is silently a single-issue read. Port fix: track whether
  `--prev-issue` was PASSED; passed-but-empty → F0. Add that assertion to the `usage` arm
  (planner finding; World AC did not cover it).
- Control counts on closed issues are fixed only while nobody posts the signature there; a mismatch
  is the control working.
