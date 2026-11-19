# Evaluation of PR #2419, commit ae343722d (iteration 122, evaluator)

**Score: 92 / 100. Verdict: PASS (threshold 70).**
Worktree `/Users/voightkampff/.ailang-driver-pin/fleet-i122-judge`. Verified clean (`git diff --quiet` rc 0) after every mutation.

## Fidelity
- The diff against 41f55ffdc touches 19 files: 2 grep scripts, 14 mirrors, 1 changelog, 2 tools/launchd scripts.
- Exactly 8 grep sites changed, 4 per copy. In each copy the `-` lines are `grep -E "^\d+\."` and the `+` lines are `grep -E "^[0-9]+\."`.
  - `.claude`: lines 128, 137, 154, 163.
  - `.agents`: lines 125, 134, 153, 162.
  - No other line in either file changed.
- SKILLS gained exactly the 16 names D-FLEET-38 lists, as two literal appended lines, with no wildcard.
  - My mutation M9 (a `*` in SKILLS) was expanded by the shell and turned both gates red.
- design-doc-creator is not in SKILLS (it appears only in a comment). My mutation M7 adding it turned two assertions red.
- 14 mirrors were regenerated. collaboration-hub and trace-debugger were already byte-identical at the parent, so 16 are on the list and 14 files changed. This matches the iteration-120 note in the ledger.

## Data loss: none
- At the parent, `diff -rq .claude/skills/<s> .agents/skills/<s>` over the 16 directories showed no `Only in` line and no differ other than `SKILL.md`.
  - Control: the same command on model-manager shows a differ, so the instrument can see one.
- Word-level diff of the removed text across all 14 changed mirrors: every removed token is Codex-only wording.
  - `Codex`, `.Codex/skills/...` paths, `` `Codex ``, `Codex-haiku-4-5`, `"Codex"`, `Codex/GPT/Gemini`, `Codex-facing`.
  - Every added token is the matching `Claude`, `Claude Code`, `.claude/...`, `claude-haiku-4-5` or `"claude-code"` wording.
- Cross-check: I applied a blind `Codex`->`Claude` and `.Codex`->`.claude` map to the parent `.agents` file.
  - 9 files matched the new file exactly.
  - 6 files differed in only 2 to 6 lines each: benchmark-runner, coordinator-helper, eval-analyzer, local-ollama-eval, prompt-manager, use-ailang.
  - Those lines are case or wording variants of the rename (`Claude -p` became `claude -p`, `Claude` became `Claude Code`, `"Claude"` became `"claude-code"`). None is `.agents`-only prose.
- After the change, `diff -rq` shows all 16 `.agents` directories identical to `.claude`.

## Gates
| Gate | rc | Result |
|---|---|---|
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | 0 | 7 passed, 0 failed |
| `/bin/bash tools/launchd/sync-agents-skills.sh --check` | 0 | "agents skills in sync", 23 skills listed |
| `make check-changelog` | 0 | 4 fragments valid; CHANGELOG.md is index-only |
| `git diff --check 41f55ffdc ae343722d` | 0 | no whitespace errors |
| `make test-launchd-drivers` (foreground, 5m31s) | 0 | "all 58 suites passed"; "bash 3.2 syntax OK" |

## Scope guard
Run with stdin `refs/heads/x <sha> refs/heads/x 000...0` and args `origin git@github.com:sunholo-data/ailang.git`.

| MISSION_NAME | rc | Output |
|---|---|---|
| fleet | 0 | no output; push allowed |
| v1 (control) | 1 | refuses `tools/launchd/sync-agents-skills.sh` and `tools/launchd/test_agents_skills_sync.sh` as "loop-harness path" |

- v1 refuses nothing else. The `.agents/skills/<16>/*`, design-doc-creator and changelog paths all pass for v1.
- The v1 refusal is the existing harness rule (`tools/launchd/*` in `_scope_is_harness`). It is not new.
- The pinned guard already carries the 16 `.agents` allowlist entries and the design-doc-creator entries from #2413.

## Mutations
Every mutation was applied from a /tmp copy and restored by copying it back. The sha256 matched after each restore, and the test was green again.

| # | Mutation | Test result |
|---|---|---|
| M1 | `\d` put back at `.claude` copy line 128 | RED (design-doc-creator searches use [0-9]: `\d` found, sites:3). Green after restore. |
| M2 | `\d` put back at `.agents` copy line 125 | RED (same assertion, line 125). Green after restore. |
| M3 | 17th name `docs-sync` added to SKILLS | RED (sync list is exactly the named skills; and mirrors equal .claude). Green after restore. |
| M4 | `trace-debugger` dropped from SKILLS | RED (list: only expected trace-debugger). Green after restore. |
| M5 | hand-edit appended to `.agents/skills/use-ailang/SKILL.md` | RED (STALE mirror). Green after restore. |
| M6 | `+` removed from one `[0-9]+` site, so the site count is 3 | RED (sites:3). Green after restore. |
| M7 | `design-doc-creator` added to SKILLS | RED (2 assertions). Green after restore. |
| M8 | `trace-debugger` dropped from EXPECTED in the test | RED (only in script: trace-debugger). Green after restore. |
| M9 | `*` added to SKILLS | RED (2 assertions). Green after restore. |
| M10 | changelog heading changed from `###` to `##` | RED (`make check-changelog` rc 2, "invalid changelog fragment(s)"). Green after restore. |
| M11 | changelog fragment deleted | NOT caught: `check-changelog` rc 0 and the sync test rc 0. |

## Portability
Fixture lines: `1. alpha`, `12. beta`, `d. lit`, `dd. lit2`, `  3. indented`, `x1. no`. Instrument: BSD grep 2.6.0-FreeBSD.

| Pattern | Lines matched |
|---|---|
| `^\d+\.` | `1. alpha`, `12. beta` |
| `^[0-9]+\.` | `1. alpha`, `12. beta` (byte-identical to `^\d+\.` by `cmp`) |
| `^d+\.` (negative control) | `d. lit`, `dd. lit2` (differs from `\d`, so the comparison can tell the difference) |

GNU grep was not available, so the GNU half stays unmeasured, as the ruling states.

## Findings
| # | Severity | Finding |
|---|---|---|
| 1 | NON-BLOCKING | The changelog fragment is not pinned by any test (M11). `make check-changelog` does not require a fragment. This is the repo's existing behaviour, not a regression of this PR. |
| 2 | NON-BLOCKING | The test's site check is `grep -cF` of at least 4 per copy, so a fifth added `\d` site would be caught by the `\d` check but a lost fifth `[0-9]` site would not. That is acceptable for the stated 4 sites. |
| 3 | NON-BLOCKING | The changelog says ten paired skills still differ and lists microrag, but D-FLEET-38 names headless-runner in its set of 11. The ruling and the changelog disagree on that name. `.claude/skills/headless-runner` does not exist at this commit, so the changelog is plausibly the accurate one. It is outside the 16 and harmless. |
| 4 | NON-BLOCKING | Behavioural change that is by design (ruling A): codex and pi now read the `.claude` wording, for example `claude -p`. |

No BLOCKING findings.
