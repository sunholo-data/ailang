# Fleet iteration 138 — Guard mission base reads by repository

Ticket `skill:mission-base-relative-path-records-driver-repo`, filed by World iteration 431
after the incident in iteration 424. One slot, no blockers. Plan both milestones for one executor
session (~120–160 changed lines including tests and generated copies). Registry reuse is `none`
for both milestones because they repair local Bash and skill instructions.

The World report observed an AILANG SHA recorded into World's state after the controller moved
into the driver checkout to find the helper. Running the driver copy from the World repo
recorded the correct SHA. The helper currently resolves refs against the calling repo.
The pinned driver exports an absolute `MISSION_DRIVER_ROOT` and the mission's `MISSION_REPO`
slug. World and Stapledon have their own origins. Docs, Fleet, Motoko, Triage, and V1 use
`sunholo-data/ailang`, including from its pin worktree.

## M1 — Guard repository identity and prove the refusal

Files: `tools/launchd/mission-base.sh` and `tools/launchd/test_mission_base.sh`.

- Guard `record`, `snap`, and `drift` after argument validation and before ref resolution,
  state access, or writes. Validate even with `MISSION_BASE_REF` set. Keep `last` usable as a
  state-only read from any CWD. Unset or empty `MISSION_REPO` keeps the attended/scratch behavior.
- Read the CWD repository's configured fetch origin with `git config --local --get-all
  remote.origin.url`. Require one nonempty URL. Accept `https://github.com/owner/name`,
  `git@github.com:owner/name`, and `ssh://git@github.com/owner/name`, each with optional
  `.git` and trailing slash. Strip trailing slash before stripping the suffix, require
  exactly two nonempty slug components, and compare case-insensitively using
  `LC_ALL=C tr '[:upper:]' '[:lower:]'`. Reject an unsupported host or malformed or ambiguous
  origin. The pin helper uses a related origin check, but its identity contract accepts local
  paths and preserves repository path case. Keep this GitHub slug check private to mission-base.
- Refuse with exit **3**, distinct from existing exit 1 (drift/ref failure) and exit 2
  (usage/no record). Write a stderr diagnostic with the actual and expected slugs, for example
  `mission-base: CWD origin is sunholo-data/ailang, expected sunholo-data/ailang-world (MISSION_REPO)`.
  For an unavailable slug, identify the reason (outside Git, missing origin, or unsupported
  or ambiguous origin) and print the expected slug. Omit the raw URL. On refusal produce no
  SHA output and leave the base file absent or byte-identical.
- Use Bash 3.2.57-compatible syntax and portable flags. Remove the inherited unused `oldn`
  local from `drift` while editing the helper so ShellCheck can pass.

Keep all 13 existing arms. Clear inherited `MISSION_REPO` and `MISSION_BASE_REF` in the test
script and set each only for the calls that need it. Add fake origins to guarded Git fixtures
without fetching. Keep the no-origin arm's working remote-tracking refs and valid
`origin/HEAD`. Use a fresh test state directory for each arm. For refusal checks, observe
`absent` or the file's SHA-256 and line count before and after and assert equality. For the
present-file case, obtain the file from a successful helper record and observe those bytes.
Assert status and diagnostic, and capture stdout separately from stderr.

| New test arm | Mutation of the helper it must kill |
| --- | --- |
| World expected, AILANG fixture origin, valid ref override, `record` returns 3 with both slugs and no SHA. Exercise both absent and already recorded state. | Bypass the guard for `record` when a ref override is set, or move the guard after the append. The absent case detects creation and the present case detects changes to an existing record. |
| Matching HTTPS origin with and without `.git`, plus a trailing-slash variant. The record succeeds and echoes/stores the fixture SHA in one row. | Drop HTTPS handling, make `.git` mandatory, omit suffix stripping, or omit trailing-slash stripping. Run one mutant per omission. |
| Matching SCP-style and `ssh://` origins, each with and without `.git` and with a trailing-slash variant. Use mixed-case slugs in at least one fixture and assert the same successful record as for HTTPS. | Drop either SSH parser branch, mishandle its optional suffix or trailing slash, or compare slugs case-sensitively. |
| `MISSION_REPO` explicitly unset, then empty, in no-origin Git fixtures with valid refs. Both records succeed. | Require a repo identity when the variable is unset, or activate the guard by presence alone. |
| Expected slug set, valid refs but no origin URL. Refuse with 3 and the missing-origin reason. | Treat a missing URL as a successful match. |
| Expected slug set, CWD outside Git. Use Git's existing `GIT_CEILING_DIRECTORIES` in that call to stop discovery of the parent worktree. Refuse with 3 and the outside-Git reason. | Skip verification when outside Git. |
| Same slug on a non-GitHub host. Refuse with 3 and an unsupported-origin reason. | Compare the path while accepting an arbitrary host. |
| Wrong origin for `snap` and `drift`, with a deliberately unresolvable ref. Both return 3 and the repo diagnostic. | Omit the guard from either command or place it after the first Git ref read. |
| `last` returns a SHA recorded by the helper from a CWD without an origin, with `MISSION_REPO` set. | Apply the repository guard to `last`. |

For each mutation, `cp -p` the implemented helper to a fresh executable file under the
disposable `TMPDIR`. Change only that copy, then run
`MISSION_BASE_HELPER="$TMPDIR/mission-base-mutant.sh" /bin/bash tools/launchd/test_mission_base.sh`
directly, since the launchd suite wrapper scrubs the override. Require suite exit 1 and
the named arm's `not ok` line. First run the same override with an untouched copy and
require all arms green. Inspect the targeted failure for each mutant.

## M2 — Use the driver path at every gate

Files under `.claude/skills/mission-control/resources/` and their generated counterparts
under `.agents/skills/mission-control/resources/`:

- `gate-1-observe.md:24`
- `gate-3-route.md:660,662,667`
- `gate-3b-ci-green.md:31,34`
- `gate-4-record.md:265` and prose at 276 (ticket cited 250 and 261)
- `ref-drift.md:24,55,56,61` (24 is prose)

Replace all ten executable invocations, including `last`, with
`bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-base.sh" …`, matching the heartbeat
helper path. Make Gate 1 stop on record failure before it echoes the base.
The Gate 3 and 3b drift conditionals already abort on exit 3. Update the two prose
references to describe the helper or its result. Put one sentence in `ref-drift.md`:
“Run from the mission's own repository, since the helper refuses a mismatched origin
when `MISSION_REPO` is set.” Keep `SKILL.md` outside the edit so its context budget holds.

Regenerate the mirror with `/bin/bash tools/launchd/sync-agents-skills.sh`. The base
mirror gate is green and the generator processes the complete sync list, so confirm the
resulting mirror diff contains only the five resource counterparts. Add
`changelogs/unreleased/2026-11-25-mission-base-repo-guard.md` with a
`### Fixed — Mission base records reject the wrong repository (2026-11-25)`
heading and a short description linked to this plan, per the fragment README.

## Acceptance gates and measured base

Base `3f5143370`, clean before tests, using macOS Bash 3.2.57. Run from the worktree
with `TMPDIR=$(mktemp -d "$PWD/.i138-tests.XXXXXX")` and export it. Keep
`AILANG_STATE_DIR` under that directory and remove the disposable directory after the run.

| Command | Base result | Required result |
| --- | --- | --- |
| `/bin/bash tools/launchd/test_mission_base.sh` | rc 0, 13 arms | All 13 and the new arms pass. |
| `MISSION_BASE_HELPER="$TMPDIR/mission-base.sh" MISSION_REPO=unrelated/caller /bin/bash tools/launchd/test_mission_base.sh` with `cp -p` of the helper | rc 0, 13 arms | All arms pass with an unchanged copy and an inherited repo. The mutation runs use the same override with the expected targeted failures above. |
| `shellcheck -s bash tools/launchd/mission-base.sh tools/launchd/test_mission_base.sh` | rc 1, inherited SC2034 on `oldn` at line 44 | rc 0. |
| `rg -n 'bash tools/launchd/mission-base\.sh' .claude/skills/mission-control/` | rc 0, 10 relative calls | rc 1 (no matches). Review all occurrences of `mission-base.sh` to confirm the ten absolute calls. |
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | rc 0, 7 passed | rc 0, including exact mirror equality. |
| `/bin/bash scripts/check_context_docs.sh` | rc 0, 13 rules and 41 skills | rc 0, including the `SKILL.md` ratchet. |
| `make check-changelog` | rc 0, 28 valid fragments | rc 0 with 29 valid fragments. |

These targeted base gates used local files and scratch Git repos. The controller owns
the separate out-of-sandbox baseline for `make test-launchd-drivers`. Any socket-dependent
results for that suite in this sandbox are **UNINFORMATIVE UNDER SANDBOX**.
