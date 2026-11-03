Completed the mechanical edits, left uncommitted.

Changed files and line numbers:

- [missions/world.toml](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/missions/world.toml:3): 3, 4, 5, 6.
- [.claude/skills/mission-loop-change/SKILL.md](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/.claude/skills/mission-loop-change/SKILL.md:3): 3, 27, 44, 45, 48, 49, 50, 51, 52, 58, 59, 61, 63, 65, 66, 67, 68, 69, 73, 140, 141, 228.
- [scripts/mission_directives.sh](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/scripts/mission_directives.sh:34): 34, 114, 115, 116.
- [internal/mission/driver_agreement_test.go](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/internal/mission/driver_agreement_test.go:177): 177, 178.
- [internal/mission/doctor_test.go](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/internal/mission/doctor_test.go:191): 191.
- [Changelog fragment](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter81-dfleet25/changelogs/unreleased/2026-11-03-world-shared-driver-comments.md:1): new file, lines 1, 2, 3, 4, 5, 6.

Gate results:

| Gate | Result |
|---|---|
| `git diff --check` | PASS |
| Bash syntax | PASS |
| `make check-changelog` | PASS |
| `make check-skills` | PASS |
| Stale-claim grep | **RED:** two pre-existing historical matches |
| Positive control | PASS: count `1` |
| `go test ./internal/mission/...` | **UNINFORMATIVE UNDER SANDBOX:** denied socket bind; also reports stale mirror |
| `make check-context-docs` | **UNINFORMATIVE UNDER SANDBOX:** temporary writes outside tree; observed rc 0 |

The snippet’s grep counts are **1** for `mission-control.sh` and **0** for `mission-heartbeat.sh`. Its Bash syntax passes; the skill is **231 lines**.

Left unchanged:

- `driver_agreement_test.go:114`: historical test diagnostic containing “world fork”; changing it would exceed the comment-only restriction.
- `doctor.go:140`: historical comment matched by the broad grep. Left as a finding, per the no-workaround rule.
- `.agents` mirror: expected **STALE** pending controller sync; untouched.

No TOML keys, test assertions, or script logic changed. No git writes, messages, background commands, or sync command run.