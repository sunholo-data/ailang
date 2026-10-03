### Fixed — mission heartbeat stamps resolve the helper via an absolute MISSION_DRIVER_ROOT (2026-10-03)

- The nine `mission-heartbeat.sh stamp` commands in the mission-control gate files used a repo-relative
  path, so World and Stapledon fires (whose cwd is not the driver checkout) failed every gate stamp with
  rc=127. Each stamp is now `case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp … ;; *) … false ;; esac`.
  Both `.claude` and the generated `.agents` skill trees carry the fix (`.agents` via `sync-agents-skills.sh`).
- **Contract change (intentional):** an attended session that stamps without exporting an absolute
  `MISSION_DRIVER_ROOT` now fails loudly instead of silently resolving nothing. Each gate file states the setup.
- `gate-3-route.md` names rc 19 (`provider_quota`) in the `mission_pi_run.sh` return-code list; like every
  non-18 non-zero code it is a lane failure that falls back along the declared chain.

### Changed — push guard admits the `.agents` skill mirror

- `_scope_is_harness` in `tools/launchd/githooks/pre-push` and the Authority "May change" list in
  `design_docs/fleet-mission.md` now include `.agents/skills/mission-*` and `.agents/skills/sprint-*`.
  Side effect: product missions (v1, docs, motoko) can no longer push those mirror paths, matching `.claude`.
