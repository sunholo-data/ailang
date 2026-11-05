# Iteration 86 — reality check, gates, done-gate, Gate 3b

## Reality check at base `c2fbb9b58` (controller, first-party, synthetic data only)

- Code read: `transcript-secret-scan.sh` selects candidates with `find "$root" -type f -newer "$marker"` and then runs `grep -F -c -f /dev/fd/3 -- "$file"` over the WHOLE file. Nothing restricts matching to bytes written after the marker.
- Reproduction (fake key, fake value, temp dir): a file whose secret-bearing line predates the marker, then appended with a harmless line after it → `LEAK key=FAKE_KEY …`, `verdict=LEAK files=1`, rc 1.
- Control: the same file left untouched after the marker → `verdict=clean files=0`, rc 0.
- Policy screen: the fix changes detector accuracy only. No routing, lane order, quota/ration threshold or billing guard is touched, so it is mechanical (not HD-2a).

## Executor gate list baselined at base, outside any sandbox (false-green rule 4a)

`bash -n` ×3 rc 0; `shellcheck --shell=bash` on scanner + suite rc 0; suite under a synthetic HOME/TMPDIR/AILANG_STATE_DIR rc 0 (30 arms); `scripts/check_changelog.sh` rc 0; `git diff --check` rc 0. python3 present (3.14.5).

## Controller re-run at head `96882f837`, outside any sandbox

| Gate | rc |
|---|---|
| `/bin/bash -n` on scanner, test suite and driver | 0, 0, 0 |
| `shellcheck --shell=bash` on scanner and suite | 0 |
| `/bin/bash tools/launchd/test_transcript_secret_scan.sh` | 0 (52 arms passed) |
| `/bin/bash tools/launchd/lib/run-suites.sh 1 tools/launchd/test_transcript_secret_scan.sh` | 0 |
| `/bin/bash scripts/check_changelog.sh` | 0 |
| `git diff --check` | 0 |
| `env -i HOME PATH USER TERM=dumb make test-launchd-drivers` | 0 (`run-suites: all 53 suites passed`) |

- Observation guard, not a seeded write: the real `secrets.env` sha prefix was the same before and after, the `~/.ailang/state` entry count was 489 → 489, and the count of `transcript-scan` state files was 1 → 1 (this live fire's own marker).

## Ticket reproduction re-run at head (synthetic)

- With a baseline: old hit + harmless append → `verdict=clean files=1 leaks=0 baseline=1`, rc 0.
- Without a baseline (control: old behaviour preserved as the fallback): `LEAK`, `baseline=none`, rc 1.
- New secret appended after the snapshot → `LEAK`, `baseline=1`, rc 1.
- The fake value occurs 0 times in the baseline file.

## Cost on the rig

- Real metadata snapshot over the four default roots: rc 0, **1.3 s**, **12,055** entries, 1.76 MB, mode 600 (paths and sizes only; the file was deleted afterwards).
- Synthetic worst case, 60 candidates against 12,060 baseline entries (candidates last): **15 s**, against the 120 s scan cap. The judge measured 300 candidates at 66 s (~0.2 s per candidate absent from the baseline), so ~500+ fresh files in one fire would hit the cap. That fails loudly (INSTRUMENT FAILURE), never as clean. Iteration 85 measured ~48 fresh files per 6 h window and 185 per 24 h.

## Done-gate dry-runs, `AILANG_DRIVER_PINNED=96882f837`, executor worktree, `env -i`

These ran under the idle, armed `triage` profile (no live pidfile; world and stapledon were also idle). No pidfile before or after either run, and the `transcript-scan` state-file count was 1 → 1.

- **Healthy**: rc 0, `lanes=ok`, `skill-sync=current`, new line `DRY RUN transcript scan: … take a bounded 30-second metadata snapshot before spawning, then scan afterward with a 120-second limit`, then `DRY RUN ok`.
- **Degraded** (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i86`): rc 0, `codex model 'bogus-model-i86' unusable: probe failed (rc=1)`, `lanes=DEGRADED(1)` handed to `claude:claude-haiku-5-5`, then the same dry-run scan line and `DRY RUN ok`.

## Gate 3b

- PR #2127, head `96882f83721944242047bbc7766a215fed35331a`: 27/27 check-runs completed, 0 not green (SHA-pinned poll, verdict file named for the head).
- Gate-1 base steady at `c2fbb9b58` (drift rc 0); no commits on `origin/dev` since. `MERGEABLE`/`CLEAN`.
- Merged with `--merge --match-head-commit` → `9ea8adcf4f6de9954599c59bbfc6ad540ededff2`.
- Merge commit `9ea8adcf4`: CI `completed/success`, Build and Release `completed/success` (per-workflow read on the merge SHA), 18 check-runs, 0 non-green.
- Ticket resolved with `--sha 9ea8adcf4…`; reply sent to world; open tickets 26 → 25.
