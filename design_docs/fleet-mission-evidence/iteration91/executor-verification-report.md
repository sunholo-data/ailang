# Fleet 91 — driver:idle-fire-writes-full-block-report

Implemented the generic no-pick fingerprint, product-mission pre-probe short-circuit,
24-hour expiry, fail-open consumption of changed/unreadable records, and bounded IDLE
slot history. The canonical retro instructions prescribe the only writer, `--record`,
only after human-gated no-pick fires with no lane/clock park. No boot-offset arms added.

## Files

- `tools/launchd/lib/idle-fingerprint.sh` — five sorted inputs, validated reads, atomic writer.
- `tools/launchd/mission-control.sh` — extracted block immediately after triage, before probes/drain.
- `tools/launchd/test_mission_idle_fingerprint.sh` — Bash 3.2 real local Git + stub APIs, mutation mode.
- `make/test.mk` — suite registered beside triage.
- `.claude/skills/mission-control/resources/gate-5-retro.md` — 18-line idle section and ordering clarification.
- `changelogs/unreleased/2026-11-06-mission-idle-fingerprint.md` — Added fragment naming the ticket.
- This report.

**Pending mirror:** `.agents/skills/mission-control/resources/gate-5-retro.md` is read-only
under this sandbox. Copying the canonical file there returned `Operation not permitted`.
The exact replacement is supplied at `.snap/M1/.agents/skills/mission-control/resources/gate-5-retro.md`.
The controller must apply that replacement (or run `tools/launchd/sync-agents-skills.sh`)
before committing. The manifest includes this pending path explicitly.

## Gates

All new tests export temporary HOME/state/TMPDIR before calling the helper/block; their
mktemp directories and local bare origin/clone are inside this worktree. APIs are stubbed.
The shell is `/bin/bash` 3.2.57(1)-release, arm64-apple-darwin25. Git writes in the new
suite are confined to disposable fixture repositories; no worktree Git writes were made.

| Command | Result |
|---|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | PASS, including dynamic reader/writer seam and both slot-state directories |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | PASS, seven killed, zero survived; pristine control passed |
| `/bin/bash tools/launchd/test_mission_triage_idle.sh` | PASS (also passed in the broad run) |
| `make test-launchd-drivers` | UNINFORMATIVE UNDER SANDBOX overall; see baseline and mirror findings below |
| `shellcheck -s bash tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` | PASS, zero findings |
| `bash -n tools/launchd/mission-control.sh` | PASS |
| `shellcheck -s bash tools/launchd/mission-control.sh \| grep -c '^In '` | PASS count comparison: 30 before, 30 after; the driver itself is not lint-clean |
| `make check-context-docs check-changelog check-tmpfile-hygiene check-home-isolation check-skills` | PASS (all five targets) |
| `git diff --check` | PASS |
| `/bin/bash tools/launchd/sync-agents-skills.sh --check` | FAIL: pending read-only mirror, not a baseline failure |

The broad suite's genuine mirror failures are `test_agents_skills_sync.sh` and
`test_quorum_artifacts.sh` (`generated_docs_and_mirror`). They require the supplied
mirror replacement. They are separate from the sandbox-dependent arms below.
The final broad run exited 2, reporting eight of 54 suites nonzero: the same six
baseline suites plus those two mirror checks. The new suite passed in 10 seconds.

## Untouched-base findings

The baseline `make test-launchd-drivers` exited 2, reporting six of 53 suites nonzero:

- `test_pin_root.sh`: its non-repo fixture was discovered inside this enclosing Git worktree;
  fetch attempted the enclosing read-only gitdir and was refused. Reported reason differed.
- `test_skill_sync.sh`: non-git fixture expected `skip:not-worktree`, observed `skip:not-dev`
  because its temporary directory is nested inside this worktree.
- `test_mission_pi_run_commits.sh`: non-repo arm I expected `10/empty_worktree`, observed
  `0/ok commits=0`, again with the fixture nested inside this worktree.
- `test_mission_kill_tree.sh`: UNINFORMATIVE UNDER SANDBOX, `ps` inspection denied, rc 2.
- `test_mission_stall.sh`: UNINFORMATIVE UNDER SANDBOX, live descendant CPU observation
  failed under the same process-inspection restriction (49 other arms passed).
- `test_codex_controller_env.sh`: UNINFORMATIVE UNDER SANDBOX, real Codex roundtrip and
  resolver arms cannot establish the required environment (14 other assertions passed).

No baseline failure was worked around. Tests requiring real network, sockets, gh, ailang,
Codex credentials, keychain or process inspection have no live verdict here. The controller
must rerun outside the sandbox. The new suite's `gh` and `ailang` are private stubs, and its
Git origin is a local bare repository; its PASS does not claim a live-service result.

Isolation deviation: the first baseline static checks used their scripts' default temporary
directory. Subsequent checks and the broad-suite runs used worktree-local temporary HOME,
state and TMPDIR. The new test itself always creates all state under its own local mktemp.

## Slot-verdict reader audit

Searched the entire repository (including hidden files), then `internal/`, `cmd/`,
`scripts/`, `tools/` and the dashboard for slot-verdict filenames, variable aliases and
field references. No reader changes were needed.

| Reader/reference | Finding for IDLE |
|---|---|
| Driver attempt `_mc_history` assignment | Path assignment only; no history interpretation |
| Driver retry `_mc_retry_history` | Appends RETRIED and tails 200 rows; retry decisions use current output/heartbeat, not verdict history; IDLE cannot reset attempts |
| Driver final `_mc_slot_history` | Appends and tails 200 rows; classifies current in-memory rc/heartbeat; no counting of completed iterations |
| Driver recovery/notifications | Read current in-memory verdict and heartbeat; do not reread history |
| `cmd/ailang/mission_ticket.go`, `internal/mission/ticket.go` | User-supplied slot-verdict text stored as an opaque string; no filename reader or verdict parser |
| `test_mission_heartbeat.sh` | Assertions over its own isolated fixture logs; IDLE production rows cannot contaminate them |
| New idle suite | Explicitly asserts IDLE layout, retained record and correct slot path |
| Dashboard, remaining internal/cmd/scripts/tools code | No additional slot-verdict history readers found |
| Gate-2/mission-loop-change skills, v1 mission/plist, design/evidence docs | Human instructions, historical text or example aggregate-by-verdict queries; IDLE remains its own explicit category |

The IDLE row uses `verdict=IDLE rc=0 attempt=1/3 elapsed_s=0 stamps=0 controller=none`
(with configured attempt/retry fields), in the existing seven-field timestamped layout.
It uses `${AILANG_STATE_DIR:-$STATE_DIR}` and preserves the 200-row retention limit.

## Mutants (scratch copies only)

Every mutant was actually run against the suite. Controls passed; all mutant runs exited
nonzero at the named behavioral assertion. No tracked file was mutated in place.

| Mutant | Killing arm | Outcome |
|---|---|---|
| Remove both Git exclude pathspecs | Real record-only design-doc commit must leave hash unchanged | KILLED |
| Driver reads `mission-wrong-idle-fingerprint` | Unchanged record must exit 0, call helper once and append IDLE | KILLED |
| Keep record after changed hash | Changed arm must fall through and remove record | KILLED |
| Failed read exits 0 (fail closed) | Error arm must fall through, consume record and log loudly | KILLED |
| Raise expiry threshold, effectively dropping daily expiry | Record aged 86401 seconds must expire without helper call | KILLED |
| Replace `--limit` guard with `true` | 1000 unread messages must fail with no fingerprint | KILLED |
| Include pending (`null`) check conclusions | Pending/green check must leave hash unchanged | KILLED |

Additional evidence: separate real commits prove each exclusion works on this Git; a ledger-only
record-path commit still wakes, while uncommitted charter edits do not. All five red conclusions
wake; success/skipped/neutral and a nil inbox are accepted without waking. Partial ledger markers,
malformed/empty JSON, fetch/API/mail errors and missing/non-numeric issue pointers print no hash.

## Remaining uncertainty

Live-service behavior and any downstream consumer outside this repository were not exercised.
The skill mirror is the only required implementation file that could not be installed here;
its complete replacement is in the snapshot. No deployment, commit or external message was sent.
