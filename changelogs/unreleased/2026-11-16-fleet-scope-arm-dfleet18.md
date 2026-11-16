### Changed — Fleet scope guard admits the two Motoko probe files (D-FLEET-18, ruled A 2026-11-16)

The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
exactly `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`,
per Mark's attended ruling D-FLEET-18 = A (2026-11-16). They are support paths, not harness, so product
missions (v1, motoko, docs...) keep their access; every other `tools/eval/` path stays refused for
the fleet. Guard arm, Authority line and 13 regression arms only; under the iteration-16 rule the
#13 probe flake fixes follow in a separate change once this lands on the pinned `origin/dev` guard.
