### Added — the motoko executor runs the ailang_only lane, gated

A task asking for exactly the `ailang_only` tools now runs on motoko only when
every property that makes the lane safe holds, read as motoko itself resolves
it:
- **The policy:** a program policy resolving as `restricted`.
- **The profile,** via motoko's own resolver (`print_config_json`), never a Go
  re-parse: `ailang_policy` first, only allowlisted companion extensions,
  `extensions.strict`, and `tools.hybrid` off.
- **Strict loading actually enforced:** a probe profile naming an uninstalled
  extension must exit 2 (arniwesth/motoko_agent#205).

Any failed check refuses the task before spawn. A run that passes gets the
policy forwarded as `AILANG_AGENT_POLICY`, records the lane tools and the policy
digest, and is checked afterwards. If `ailang_policy` was not loaded first, or a
non-lane tool executed, the result is a `policy_violation` failure.

New model entry `motoko-lane-or-glm-5-3-flash` (profile `ailang_only`): the
motoko arm of the pi-vs-motoko lane A/B. Design:
m-motoko-ailang-only-lane.
