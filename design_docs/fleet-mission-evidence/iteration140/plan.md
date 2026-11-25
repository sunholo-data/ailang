# Fleet iteration 140 — Target a record checkout with `rotate-log --workdir`

World iteration 433 filed `skill:rotate-log-index-only-targets-registry-workdir` with no blockers. The controller selected `--workdir <dir>` for both rotation and index regeneration. The current command uses the registry checkout for `sunholo-data/ailang` missions and the registry entry’s `workdir` for external missions. World’s registry entry points at the shared main checkout, and its record worktree has no `missions/` directory. Both milestones fit one executor session. Registry reuse is `none` for each milestone because the work uses the existing Go CLI and mission helpers.

## M1 — Add the CLI override and regression tests

Files

- `cmd/ailang/mission_rotate.go` — parse the flag, resolve and validate the override, and update flag-list diagnostics.
- `cmd/ailang/mission_cmd.go` — add `[--workdir <dir>]` and its meaning to `printMissionHelp`.
- `cmd/ailang/mission_rotate_test.go` — add the regression cases below using temporary directories.

Implementation

1. Parse `--workdir <dir>` alongside `--keep`, `--index-only`, and `--stream`. Reject a trailing `--workdir` before registry access with an error such as `--workdir needs a directory`. Reject a quoted empty value with the same diagnostic.
2. Keep registry loading and mission lookup. When the flag is present, use `filepath.Abs` to resolve the supplied directory against the process working directory and propagate any resolution error with the input path. Use that directory for both repo types. Otherwise, keep the current `m.Workdir` and shared-repo `m.Root()` branches, including the missing-registry-origin error.
3. Construct `<dir>/design_docs/<name>-mission-log.md` or `<dir>/design_docs/<name>-mission-status-archive.md` for `--stream status`. With an explicit override, check the selected source with `os.Stat` before calling either helper. Wrap a stat error with a `--workdir target` diagnostic containing the absolute source path. Fail without trying the canonical checkout or writing an index, even when archive history exists.
4. Pass the selected path to the existing `mission.RotateLog` or `mission.RegenerateIndex`. The helpers derive archive and index paths next to the source. Preserve their output format and the byte-identical default target, file effects, and output when the flag is omitted. Keep the existing tests unchanged. Add the flag to the missing-name usage, the `--keep`/`--index-only` conflict usage, and the unknown-flag list.

### Test table

Use `writeSyntheticRepo`, `use`, `synthWrite`, `checkRotated`, and `captureStdout` where applicable. Use real, distinguishable records in decoy directories. Compare directory entries and file bytes so an unexpected file creation fails even if the original source stays intact. Check exact printed path fields. Name new tests with `Rotate` so the focused gate runs them.

| Case | Required observation | Mutation it kills |
| --- | --- | --- |
| External mission with `--keep 5 --workdir <absolute target>` | Rotate ten entries in a target without `missions/`, retain the newest five and write the archive and index beside it. The registry workdir decoy keeps the same files and bytes. | Drop the explicit directory assignment or always use `m.Workdir` for external missions. |
| Shared-repo mission with `--keep 5 --workdir <absolute target>` | Rotate in the explicit target, distinct from both registry origin and registry workdir. Both decoys keep the same files and bytes. | Allow `m.Root()` to overwrite the explicit directory for shared missions. |
| `--index-only --workdir <target>` | With distinct live and archived iterations, rebuild only the target index with rows from both sources. Both sources and the decoy stay byte-identical. Stdout gives the target `INDEX` path. | Ignore the override for index-only or dispatch index-only to `RotateLog`. |
| `--stream status --workdir <target>` | Rotate a status archive with recognized STATUS headings and write `<name>-mission-status-archive-old.md` and `<name>-mission-status-index.md` in the target. Keep the ordinary log and the default status location byte-identical. | Route status to the ordinary log or resolve the default directory for status. |
| Missing override source | Cover log rotation and status index-only with valid default sources. Seed optional target archive history and, in the status case, a target ordinary log. Require the `--workdir target` error and exact missing absolute path, with all files unchanged. | Skip the explicit existence check, check the log before selecting status, or fall back to the default directory. |
| Missing flag value | With an invalid registry, call `rotate-log <name> --workdir` and require the missing-directory error before registry access. Cover an empty argument separately. | Drop the end-of-args guard, accept an empty directory, or defer the check until after registry load. |
| Relative `--workdir` | Set the working directory to a temporary parent and pass a relative path to its child target. Assert the exact absolute `live log`, `archive`, and `INDEX` paths in stdout and the artifacts under that child. | Drop `filepath.Abs` or resolve the argument against the registry origin. |
| Flag discoverability | Assert `[--workdir <dir>]` in mission help and the three flag-list diagnostics, using valid argument shapes for each error. | Omit the flag from any advertised syntax string. |

## M2 — Update the Gate 4 recipe and release notes

Files

- `.claude/skills/mission-control/resources/gate-4-record.md`
- `changelogs/unreleased/2026-11-25-rotate-log-workdir.md`

Keep the Gate 4 edit limited to the target-selection warning and the two out-of-checkout forms. Preserve the fleet iteration 28/32 evidence sentence and the measured iteration 32 observation. Explain that the historical shared-repo warning describes default routing, and point its source reference at `cmd/ailang/mission_rotate.go`. Keep the canonical-checkout example and the existing printed-path checks. For a record written outside its canonical checkout, run from the record checkout and prescribe these forms:

```bash
ailang mission rotate-log ${MISSION_NAME} --keep 20 --workdir "$PWD"
ailang mission rotate-log ${MISSION_NAME} --index-only --workdir "$PWD"
```

State that the override works for missions without a `missions/` directory in the record checkout, including World and Stapledon. Keep the `--stream status` tip. The controller regenerates the `.agents/skills/mission-control/` mirror with `tools/launchd/sync-agents-skills.sh` outside the sandbox. The executor leaves the mirror alone.

Add a `### Added` section to the fragment describing `--workdir <dir>` for rotation and index regeneration, with a link to this plan.

## Executor gates

Run from the worktree after the two milestones:

```bash
go build ./cmd/ailang
go test ./cmd/ailang -run Rotate -count=1
go test ./internal/mission -count=1
make check-context-docs
```

The executor makes no git writes. The context-docs gate reads tracked paths internally. Report the result of each gate and label any sandbox-denied result **UNINFORMATIVE UNDER SANDBOX**, including the command and denial. Remove any `ailang` binary newly produced by the build gate after validation.
