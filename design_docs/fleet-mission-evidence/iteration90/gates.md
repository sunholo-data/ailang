# Iteration 90 — controller gates (outside the sandbox)

Worktree `.wt-fleet-iter90-secrets-m3`, branch `fleet/i90-dfleet42-m3`. Every run was bracketed by sha256 + line count of the real `~/.config/ailang/secrets.env`, printing SAME/CHANGED only: SAME every time.

## Base (7f1e13d5f, #2143 head), before the directive was sent
`make test-launchd-drivers` 54/54 rc 0 · `test_secrets_scope.sh` 58/58 · `sync-agents-skills.sh --check` rc 0 · `make check-changelog` rc 0 · `git diff --check` rc 0.

## Build commit b216285db (executor output, committed by the controller)
- Snapshot `.snap/M3b/` byte-identical (cmp) to the tree for all 5 files.
- Diff scan for writes outside the worktree: the new suite writes only under its own `mktemp -d` (trap-removed) and Python `TemporaryDirectory`; synthetic HOME per case. No real-path writes.
- `bash -n` ×3 rc 0 · `test_secrets_scope.sh` 59/59 · `--mutations` rc 0 (5/5 dry-run mutants killed) · `make test-launchd-drivers` 54/54 · shellcheck new file rc 0 · driver shellcheck findings 33 vs base 33, 0 new (gcc format, line numbers stripped, `comm -23` empty) · sync, changelog, file-sizes, no-personal-email, diff-check rc 0.

## Fix commit 913c6f9e1 (judge findings 1-3)
- `test_secrets_scope_dry_run.sh` 8/8 arms · `--mutations` 8/8 killed, 0 survivors · `test_secrets_scope.sh` 59/59 · `make test-launchd-drivers` 54/54 · shellcheck new file rc 0, driver 33 · changelog, no-personal-email, file-sizes, diff-check rc 0.

## Live dry-runs (profile `triage`, idle and armed: no kill switch, no pidfile; `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=<head>`)
| Set | stdin | Tree | claude H/D | codex H/D | pi H/D |
|---|---|---|---|---|---|
| 1 | inherited (session socket) | b216285db | 0 / **2** | 0 / 0 | **124 / 124** |
| 2 | `< /dev/null` (launchd condition) | b216285db | 0 / 0 | 0 / 0 | 0 / 0 |
| 3 | inherited (session socket) | 913c6f9e1 tree | 0 / **2** | 0 / 0 | 0 / 0 |

Every run that reached the preview printed `secrets-scope: lane=<p>-controller kept=4[…] stripped=3[ANTHROPIC_API_KEY,GOOGLE_API_KEY,GEMINI_AGENTS_API_KEY]` (names only). Every passing run printed `marker=ok controller-probe=ok`. Degraded runs printed `lanes=DEGRADED(1)` with the injected codex executor handed to `claude:claude-haiku-5-5`. Codex printed `codex-marker-forward=ok driver-root-forward=ok`. `~/.ailang/state` listing diff: only ambient daemon files (heartbeats, hooks, brain, observatory). No mission-triage file was created. Dry-runs tee into `/tmp/ailang-mission-triage.log`, as every dry-run does.

## Stdin investigation (stdin-controls/)
- ctl1 (direct, explicit `< /dev/null`): scoped and unscoped pi probe rc 0, 2 s.
- ctl2 (through `_mc_bounded`, from the worktree): unscoped rc 0 in 2 s; scoped **rc 124**, 0 bytes.
- ctl3 (2×2, cwd × launcher, scoped): `_mc_bounded` rc 124 in both cwds; direct rc 0 in both. cwd is irrelevant.
- ctl4: scoped `_mc_bounded … < /dev/null` rc 0 (A1); direct background without the redirect hangs (A2, rc 143 at the cap); in the scoped shell a background child's fd 0 is a socket (A3).
- Minimal repro on `/bin/bash` 3.2.57: before any redirected loop, a background job's fd 0 is `/dev/null` (`cr--`). After a function runs `done < <(…)`, a heredoc or a here-string, it is the inherited socket (`srw-`).
- Reading: `_mc_bounded` relied on bash's implicit `/dev/null` for async stdin. bash 3.2 drops that default after any redirected loop, so a probe started from a socket-stdin session waits on stdin. Under launchd stdin is `/dev/null` (set 2 green). Fix: the preview's probe `case` reads `< /dev/null`. `_mc_bounded`'s other callers are unchanged (pre-existing; not in this draft's scope).
- Set 3's claude-degraded rc 2 is a different cause. The opus probe takes about 10–13 s (healthy runs: 10 s and 13 s), and the degraded recipe's `MISSION_PROBE_TIMEOUT=10` now also caps the controller preview probe. That log has no stdin warning. Set 1's claude-degraded log had both the "no stdin data received in 3s" warning and an `ok` reply cut off by the 10 s cap.
