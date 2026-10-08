### Fixed — codex controller loses the driver's role env and the HD-4 scope guard

A mission loop that fell back to a `codex exec` controller lost every `MISSION_*` / `AILANG_DRIVER_*` variable,
because the rig's `shell_environment_policy.inherit = "core"` strips the driver's exports from the controller's
shells. `resolve-role-spawn.sh` then failed closed for every role and the slot was lost (ticket
`agent-tool:mission-role-pins-unavailable`). The driver now forwards the role env per variable with
`-c shell_environment_policy.set.NAME=…` (`tools/launchd/lib/codex-env-args.sh`): a name allowlist, a secret-name
denylist, and the rig's own `inherit`/`.set` policy untouched.

The same stripping had switched the HD-4 mission scope guard off for codex controllers (its `GIT_CONFIG_*`
`core.hooksPath` entry was lost); that one entry is now forwarded again. New suite
`tools/launchd/test_codex_controller_env.sh`, wired into `make test-launchd-drivers`.
