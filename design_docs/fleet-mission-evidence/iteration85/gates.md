# Iteration 85 — reality check, gates, done-gate, Gate 3b

## Reality check at base `93ac5a915` (controller, first-party)

The scan printed key NAMES and file COUNTS only. No value was printed, written or passed in argv.

- No existing scanner: `git grep` over `tools/`, `scripts/` and `.github` found no secret or transcript scan. `secrets.env` is sourced at `mission-control.sh:119`. Line 176 strips `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `OPENAI_API_KEY`, and nothing else.
- A whole-tree fixed-string scan covered every `secrets.env` value of 20+ chars (7 keys) over `~/.codex-missions/sessions` (650 files, 353M), `~/.pi/agent/sessions` (223 files, 76M) and `~/.claude/projects` (6,990 files, 6.4G). Results:
  - `OPENROUTER_API_KEY`: 4 files
  - `GOOGLE_API_KEY`: 4
  - `OLLAMA_API_KEY`: 4
  - `ANTHROPIC_API_KEY`: 1
  - `CLAUDE_CODE_OAUTH_TOKEN`: 0 (rotated by Mark 2026-11-04 09:58)
  - `OPENAI_API_KEY`: 0
  - `GEMINI_AGENTS_API_KEY`: 0
  - So the class is live beyond the ticket's one key. The scan did not attribute files to loop or interactive sessions. It took about 15 minutes, which is why the detector is scoped to files newer than a per-fire marker.
- Real fire size: files modified in the last 6 h were 5 (`~/.codex-missions/sessions`), 0 (`~/.codex/sessions`), 0 (pi) and 43 (`~/.claude/projects`). In the last 24 h they were 26, 0, 0 and 185.
- Inline-comment parse gap raised by the judge: 0 of 7 real assignments carry an inline `#` comment (count only).

## Controller re-run at head, outside any sandbox (build worktree, before commit `74daa5741`)

| Gate | rc |
|---|---|
| `/bin/bash -n` on scanner, test suite and driver | 0, 0, 0 |
| `shellcheck --shell=bash` on both new scripts | 0 |
| `/bin/bash tools/launchd/test_transcript_secret_scan.sh` | 0 (30 arms passed) |
| `/bin/bash tools/launchd/lib/run-suites.sh 1 tools/launchd/test_transcript_secret_scan.sh` | 0 |
| `/bin/bash scripts/check_changelog.sh` | 0 |
| `git diff --check` | 0 |
| `env -i HOME PATH USER TERM=dumb make test-launchd-drivers` | 0 (`run-suites: all 53 suites passed`) |

- Observation guard, not a seeded write: the real `secrets.env` sha prefix and the `~/.ailang/state` entry count were the same before and after the gate run.

## Controller edit (after the executor, before the judge)

The executor's leak notice posted the key/path hit lines to BOTH channels. One of them is `gh issue comment` on the PUBLIC bookkeeping issue. The controller changed the GitHub body to a count only. Key names and paths now go to the private controlplane inbox and the driver log only. The `Driver_leak` arm was rewritten to require each key/path exactly once (private body), no key or path in the GitHub body, and the hit count in it.

- Mutation drill on a scratch copy of the driver: restore the hit lines in the GitHub body, and `Driver_leak` FAILs ("private message body contains every key/path"). The judge independently killed the same mutant.

## Done-gate dry-runs, `AILANG_DRIVER_PINNED=74daa5741`, build worktree, `env -i`

These ran under the idle, armed `triage` profile. No pidfile before or after either run, and no `transcript-scan` marker in `~/.ailang/state` afterwards.

- **Healthy**: rc 0, `lanes=ok`, `skill-sync=current`, new line `DRY RUN transcript scan: a real legacy controller fire would create a per-fire marker …`, then `DRY RUN ok`.
- **Degraded** (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i85`): rc 0, `codex model 'bogus-model-i85' unusable: probe failed (rc=1)`, `lanes=DEGRADED(1)` handed to `claude:claude-haiku-5-5`, then the same dry-run scan line and `DRY RUN ok`.

## Gate 3b

- PR #2122, head `74daa5741642312040575f3995c481515b752b3f`: 27/27 check-runs completed, 0 not green (SHA-pinned poll, verdict file named for the head).
- Gate-1 base steady at `93ac5a915` (drift rc 0); no commits on `origin/dev` since. `MERGEABLE`/`CLEAN`.
- Merged with `--merge --match-head-commit` → `26b35b08115ee9582b896025900b81dffb691cd2`.
- Merge commit: see the log entry (CI and Build and Release read per workflow on the merge SHA).
