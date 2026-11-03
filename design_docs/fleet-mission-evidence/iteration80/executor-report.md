Partial diff prepared. Sandbox denied writes to `.agents`; both mirrors there remain unchanged.

Files changed:

- [.claude Gate 3b](.claude/skills/mission-control/resources/gate-3b-ci-green.md)
- [.claude SKILL.md](.claude/skills/mission-control/SKILL.md)
- [Changelog fragment](changelogs/unreleased/2026-11-03-gate-3b-ci-poll-cap.md)

Verification results:

| Step | rc | Observation |
|---|---|---|
| 1. Both `cmp` checks | 1 each | Mirrors differ — **UNINFORMATIVE UNDER SANDBOX** |
| 2. Old-cap grep | `.claude`: 1; `.agents`: 0 | `.claude` has no matches; `.agents` blocked — **UNINFORMATIVE UNDER SANDBOX** |
| 3. New-cap grep | `.claude`: 0; `.agents`: 1 | `.claude` matches lines 16, 19, 20, 47, 68; `.agents` blocked — **UNINFORMATIVE UNDER SANDBOX** |
| 4. `wc -l` | 0 | Both SKILL.md copies remain 628 lines |
| 5. Diff / whitespace | 0 / 0 | Eight replacements in `.claude`; mirror completeness **UNINFORMATIVE UNDER SANDBOX** |
| 6. Status | 0 | Two modified files and one new fragment; expected five-file scope **UNINFORMATIVE UNDER SANDBOX** |

Unexpected: zero-context diff combines adjacent replacements at lines 19–20 into one hunk, yielding seven hunks for eight replacements.

No git write operations, make targets, or background jobs ran.