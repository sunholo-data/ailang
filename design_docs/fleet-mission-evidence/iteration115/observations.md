# Iteration 115 — observations

## The attended rulings (input that changed)

- origin/dev `a82f0180a` (2026-11-16 07:26:42Z): "docs(fleet): record 26 attended rulings D-FLEET-16..43
  as recommended". It landed 22 minutes after iteration 114's record (07:04Z), on dev, while the
  95–114 record stack sits unmerged on stacked PRs, so iteration 114 never saw it.
- **Provenance check.** `scripts/mission_ledger_provenance.sh` reports `verdict=fleet` with flip
  `a82f0180a` for D-FLEET-32 and D-FLEET-43 (D-FLEET-17: `never-resolved`), because this rig's git
  identity is the fleet bot in every session. Gate 0 rule 6(b) reads `fleet` on a non-squash flip as
  self-resolution. The controller checked whether a human actually ruled rather than which identity
  committed (rule (c)'s own test, CLAUDE.md principle 4): an interactive Claude Code session
  (entrypoint `cli`, not `-p`) holds a user message at 2026-11-16T07:18:44Z saying the fleet rows can
  be answered as recommended, eight minutes before the commit, and every flipped row carries an
  "Attended ruling 2026-11-16" stamp from `mission_answer.sh`. Adopted as attended. No row re-opened.
  The instrument gap: the provenance script cannot tell an attended session from the loop on this
  rig, so its `fleet` verdict here is uninformative, not a finding of self-resolution.
- **Division of work.** The same attended session took D-17, D-19, D-20, D-22, D-23(3), D-28, D-29,
  D-35, D-39, D-40, D-41 and the human half of D-42 (PRs #2343–#2349 and config rollouts), and named
  "D-33, D-37, D-38 and the rest" as the fleet's. This fire claimed its items on `controlplane` before
  routing and touched none of the attended set.

## Record stack reconciliation

- The stack (iterations 95–114, top #2342) and dev both edited the ledger since the merge base
  `252680e1b`. A per-row three-way comparison: 28 rows changed only on dev (taken from dev); D-FLEET-43
  changed on both (dev's RESOLVED ruling kept, the stack's iteration 113/114 notes re-inserted);
  D-FLEET-44/45 exist only on the stack (kept, OPEN). Log conflict: the stack rotated entry 70 into
  the archive while dev rewrote one path in it; the path rewrite was re-applied in the archive.
  `mission_decisions.sh --check`: 44 rows valid, 2 OPEN. Merge commit `37b284157`.

## Pre-push scope guard refuses a rebased branch (not ticketed; the fleet does not self-source)

- `git push --force-with-lease` of the rebased `fleet/i79-dfleet26-capacity-transient` was refused:
  `tools/launchd/githooks/pre-push:80-86` diffs an EXISTING remote branch from its old remote SHA
  (`edb099371`), so after a rebase the checked range includes every dev commit since 11-03, including
  language-core paths. A NEW branch takes the `merge-base … origin/dev` path, which checks only the
  branch's own diff. Pushed as `fleet/i115-dfleet26-capacity-transient` (all 4 paths `allow` under the
  guard's own `_scope_verdict`; control refused), PR #2368, and #2065 closed as superseded with a
  comment (count 0 → 1). Any stacked or rebased fleet draft (#2082, #2108, #2110, #2141–#2150) will
  hit the same refusal when it is refreshed onto dev.

## Pick order versus the attended groom (controller miss)

- The charter's Queue section is Mark's triage order and outranks `slots_lost`. By it, rows 13/19
  (`ci:launchd-suite-flakes-…` / `ci:motoko-probe-…`, D-FLEET-18 A, guard-arm draft #2110) and row 33
  (D-FLEET-25 A) rank above row 34 (D-FLEET-26 A, this fire's build). The controller picked by
  impact (a finished iteration lost) and read the queue section only at Gate 4. Gate 2 rule (d): the
  loop may not reorder an attended groom. The landing stands on its own merits (ruled, judged PASS
  88), and **Next** returns to the groom order: row 13 first.

## Gate 0 self-notice instrument

- `mission_gate0_self_notices.sh` watermark form: rc 2 (both `mission-2341-last-seen` and
  `mission-2172-last-seen` absent). Nineteenth consecutive fire.

## Iteration 79's draft was wrong for the live path

- The planner found, and the controller re-checked first-party, that mission controllers run codex
  as `ailang mission codex-exec` (daemon) since `9d9190082` (2026-10-09), printing `Error: Codex daemon
  turn failed: <msg>` (`internal/executor/codex/daemon.go:344`; `lib/codex-auth-profile.sh:17-18`
  forces `AILANG_CODEX_RUNTIME=daemon`). Column-0 `ERROR: ` lines: 0 in every `/tmp/ailang-mission-*.log`
  except stapledon's 9 (Godot). Draft #2065's signature would never have fired; iteration 79's
  independent judge scored it PASS-WITH-EDITS 88, then PASS 94, against the raw `codex exec`
  rendering; neither round traced which command the driver actually runs.
