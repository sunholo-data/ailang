# Iteration 66 evaluation (judge worktree .wt-fleet-iter66-judge @ 60f60367c)

## A. Fix: gate5:controlplane-send-refused-local-store -> PASS, 84/100

Verified: store difference reproduced read-only (`AILANG_STORAGE_MESSAGING=local ailang messages inboxes` shows controlplane
"NOTHING ... accepted and never dispatched"; gcp registry shows it served). `sync-agents-skills.sh --check` rc 0. Suite file
test_agents_skills_sync.sh 5/5 pass under /bin/bash (3.2). Changed paths all fleet-authorised (.claude/.agents mission-control,
tools/launchd, design_docs, changelogs). All four prescribed lines (gate-5 send, $(cat) form, 3(c) read, gate-3 approvals send) are
scoped, .agents mirror identical. gate-3-triage.md already used $MSG with a defined prefix.

Completeness: no remaining unscoped send/read/ack in .claude/skills/mission-control. Residual unscoped `ailang messages` uses outside
the guard's remit: SKILL.md:622 `list --unread` (read-only, local), gate-3-route.md:246 import-github (prose), sprint-evaluator/executor
local-inbox sends (intentional, not controlplane), driver mission-control.sh sends (driver env). No blocking gap.

Mutation drills (backup in mktemp, restored, cmp identical every time; test script itself cmp-unchanged):
- D5 strip prefix from gate-3 approvals send -> scope arm FAIL (gate-3-route.md:829). Right reason.
- D6 strip prefix from 3(c) read -> scope arm FAIL (gate-5-retro.md:75). Right reason.
- D7 (novel) wrong project (ailang-dev) on $(cat) line -> scope arm FAIL (gate-5-retro.md:73). Right reason.
- D8 (novel) scope pair moved AFTER the command as a trailing "# env ...=gcp ... ailang messages" comment, real prefix replaced -> scope arm PASSES (only the mirror arm failed). GUARD FOOLED.
- D9 (novel) `ailang messages send` at end of line, args on next line -> regex needs trailing space so the line is unseen; only the positive control caught it because it was the sole site. A NEW such site elsewhere is invisible.
- D10 (novel) $MSG definition blanked in gate-3-triage.md -> FAIL, correct reason ("$MSG not defined in").
- D11 (novel) appended unscoped `ailang messages ack abc` in a fenced block -> FAIL, correct.
- D12 (novel) prose backtick mention `ailang messages send foo` -> FAIL (false red on prose; gate is strict, acceptable but noisy).
Note: in each drill the mirror-equality arm also went red because only .claude was mutated; scope arm was verified independently via its own message.

BLOCKING: none.

NON-BLOCKING:
1. Guard is a single-line regex: scope pair anywhere before `ailang messages` on the line satisfies it (D8: trailing comment / any text). Prefix adjacency is not enforced; the 3(c) arm even strips the prefix before matching.
2. Line-continuation / newline-split `ailang messages send` and double-space variants are not matched (D9). The positive control covers only the two existing sites.
3. Guard scans only .claude/skills/mission-control; mission-brief and mission-loop-change are mirrored/fleet-owned but unscanned (currently they prescribe no controlplane send).
4. The `unscoped`/`msg_bad` reporting prints `; $MSG not defined in:` with empty fields, cosmetic.
5. gate-5 prose sentence says "as the driver's own sends do", true in effect (driver env), not by explicit prefix; fine.

## B. Parked measurement pi:unowned-dirty-absolute-vs-relative-paths -> verdict PARTIAL (core claims CONFIRMED; two caveats)

(i) CONFIRMED. unowned-dirty.ts:42,78 store `${ctx.cwd}/${p}` absolute; unownedDirty compares against parsePorcelain paths (repo-root-relative). Never equal, so own edits always flagged. Existing test feeds relative own paths so wiring is untested.
(ii) CONFIRMED. Diff applies cleanly (-p1) to a mktemp copy; 4/4 pass. Mutant 1 (drop slice(prefix.length)) -> 3 pass / 1 fail (new test). Mutant 2 (startsWith -> true) -> 3 pass / 1 fail. Worktree untouched. Caveat: the test named "wiring" exercises only pure functions; the new `git rev-parse` call and its use in the handler are untested.
(iii) CONFIRMED. args.go:38 `--mode json`; mission_pi_run.sh:302 `pi --mode json`; print-mode.js rebindSession calls bindExtensions({mode,...}) with no uiContext, agent-session.js passes undefined -> runner.setUIContext(undefined) -> noOpUIContext, notify `() => {}` (runner.js:88-92). Extension is installed globally in ~/.pi/agent/extensions so it does load in evals; the only effect is the notify, which is dropped. Only residual eval-visible delta is one extra `git rev-parse` subprocess on sweeping git commands (and a nonzero-exit early return).
(iv) CONFIRMED. cmd/ailang/pi_assets/unowned-dirty.ts is byte-identical (diff SAME); Makefile `pi-assets` target copies .pi/extensions/*.ts there. Fleet scope excludes cmd/ailang outside mission*. profile_assets copies only 3 other extensions, not this one.

Strongest objection: the fix compares git's `--show-toplevel` (physical realpath) against `ctx.cwd`-derived paths and model-supplied absolute paths. On macOS, a symlinked cwd (verified: `pwd` = /var/folders/.../link, `git rev-parse --show-toplevel` = /private/var/.../r) makes `startsWith(prefix)` false and relativeTo silently drops every own file, reproducing the original defect with no failing test or log. The claude-mods port's contract has the same weakness, and the diff's test uses synthetic /repo paths so it cannot see it. Also the diff makes the extension fail-open on non-repo (fine) but adds an untested subprocess to a hot path. Neither changes the "no eval score impact" conclusion, which holds, but the claimed benefit for interactive sessions is unproven on this rig.

EVALUATION_VERDICT: PASS 84

## Round 2 (judge worktree @ ffde0469a)

Clean tree: test_agents_skills_sync.sh 5/5 under /bin/bash 3.2.57, sync --check rc 0, no new false reds.
Drills (backup+cp restore, cmp verified, git status clean after each):
- D8 (trailing-comment pair, real prefix removed): FAIL "unscoped: gate-5-retro.md:73". Fixed.
- D9 (sole approvals send split at EOL): FAIL, but via the positive control, not the unscoped path.
- D9b (EXTRA end-of-line `ailang messages send` appended elsewhere): FAIL via the unscoped path. Fixed.
- D14 (pair then `true; ailang messages send`): FAIL unscoped. D15 (scoped `list;` then unscoped send same line): FAIL unscoped (per-occurrence works).
- D16 (unscoped `ailang messages read` EOL new site): FAIL. D17 ($MSG use without definition): FAIL with clean message.
- Residual (non-blocking): double-space or tab between `messages` and `send` is still not matched (as an extra site it passes silently); failure only shows if it is the sole positive-control site. Scan still limited to mission-control (accepted).
Failure message now prints no empty fields.

Round-2 score: 91/100.
EVALUATION_VERDICT: PASS 91
