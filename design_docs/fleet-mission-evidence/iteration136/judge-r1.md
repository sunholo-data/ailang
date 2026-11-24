# Fleet iteration 136 — independent judge, round 1

Judge: `sonnet` via the Agent tool (`MISSION-ROLE: evaluator`, foreground, own detached worktree `.fleet-i136-judge` at `2db1b4604`; 67,263 tokens, 9 tool uses, 87 s). Brief carried the ticket text and the code location, not the controller's verdict.

**Verdict: PASS 84/100. 0 BLOCKING.**

| Category | Score |
|---|---|
| Tests | 20/20 (suite 19/19 under `env -i`, bash 3.2.57) |
| Lint | 10/10 (shellcheck clean) |
| Acceptance criteria | 24/30 |
| Code quality | 11/15 |
| Documentation | 13/15 |
| Design fidelity | 6/10 |

## NON-BLOCKING findings
1. **Silent clean on bz2 and xz content**: `a.tar.bz2`, `a.tar.xz`, `e.bz2`, `e.xz` holding a synthetic credential value all returned `verdict=clean hits=0 control=ok`, while a non-tar gzip fails closed. Fix: magic-number check, fail closed.
2. **A value in a top-level file name or a directory child's name** scans clean (tar/zip member names are checked).
3. Out of scope but silent: a value split across lines, base64, bare zlib. The rule does not say only literal values are matched.
4. Encrypted zip, nested archives, symlinks, FIFOs, non-tar gzip, missing path: all fail closed (correct).
5. The `SKILL.md` intro re-wrap (to keep the file line-neutral) is diff noise.

## Checks
- Values never reach argv/stdout/stderr; scratch removed on every exit path (none left behind; `interrupted` arm passes).
- Positive control is real: matcher disabled → rc 2, `control=failed`.
- Skill rule prevents both incident habits (scan in its own call, read before a separate publish; values, not key shapes; named files).
- 300 MB file with a trailing value caught (streaming works). Python 3.9.6 suffices.

## Judge's own mutants (via `--helper <copy>`)
| Mutant | Killed? |
|---|---|
| m1 `MIN_BYTES` 20→200 | yes |
| m2 drop chunk-overlap carry | yes |
| m3 control check no-op | no (equivalent mutant: control only fires when the matcher is broken) |
| m4 ignore credential-named env vars | yes |
| m5 ignore failed flag in verdict | yes |
| m6 skip scratch removal | yes |
| m7 matcher always empty | yes (rc 2) |
| m8 m7 + control disabled | yes (59 FAILs) |

## Bypass attempts (synthetic HOME, `env -i`)
LEAK (correct): plain file, env value, directory, `.tar`, `.tgz`, `.zip`, tgz/tar without suffix, 300 MB file.
instrument-failure (correct): non-tar gzip, dir with symlink, FIFO, encrypted zip, no inputs.
**clean (silent miss)**: `.tar.bz2`, `.tar.xz`, `.bz2`, `.xz` (finding 1); value in file name / dir child name (finding 2).
clean (out of scope): split value, base64, zlib (finding 3).

EVALUATION_VERDICT: PASS score=84
