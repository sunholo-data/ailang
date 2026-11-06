### Added — Per-lane mission secret scope (proposed)

Draft gated on D-FLEET-42 option A and Mark's ruling; not merged. M1 ships the
pure bash/zsh library, marker-conditional bridge and tested host-line text. M2
adds child-only controller scopes, Codex marker/root forwarding with strict
validation, scoped Codex/Claude role wrappers, and pi-role startup clearing.
Synthetic actual-spawn arms and killing mutations cover the wiring.

M3 builds pinned-controller scoped dry-run probes/previews, Codex marker/root
forwarding checks and eight synthetic production-block test arms with killing
mutations. Failures return nonzero with a failed summary; absent pins skip the
preview. Parent environment and existing healthy/degraded summaries are preserved.
The probe runs with `/dev/null` stdin, because bash 3.2 stops supplying it to
background jobs after any redirected loop. All six live healthy/degraded dry-runs
(claude, codex, pi) passed under the `triage` profile. Reused provider probes may
write auth/cache/session/runtime state. Without Mark's host-line installation, Claude
and Codex zsh tool shells retain the measured startup exposure. Retained keys,
credential files and native Agent-tool subagents' inherited controller scope
remain exposed. See [the design record](design_docs/planned/m-fleet-secrets-scope-per-lane.md).
