### Changed — mission scope guard admits the `.agents` mission/sprint skill mirror

- `tools/launchd/githooks/pre-push` `_scope_is_harness` now treats `.agents/skills/mission-*` and
  `.agents/skills/sprint-*` as harness paths, matching their `.claude` sources (D-FLEET-8: the fleet fixes
  both copies). The fleet may push them; product missions (v1, docs, motoko) are now refused on them,
  exactly as they already are on the `.claude` copies. The fleet charter's Authority allowlist names
  both arms. Landed ahead of the heartbeat driver-root fix (fleet iteration 16) because the pinned
  guard judges every push, so the mirror half of that fix could not push until this arm is on
  `origin/dev`.
