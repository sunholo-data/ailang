### Changed — the cloud motoko image carries the ailang_only lane

`docker/Dockerfile.agent-motoko` now pins `551c947b` on
`sunholo/main-dst-20261001`. That is Arni's upstream `main` at `fe108db7`, which
includes the `extensions.strict` enforcement from upstream #205, plus our
commits:
- the `ailang_only` lane profile (`max_steps` 300) and its
  `motoko_ext_ailang_policy` extension;
- the cloud and `ollama_microrag` profiles, `motoko_ext_ailang_tools`, and
  rig-lease forwarding.

Two carried commits upstream made obsolete were dropped. The pin moves to a new
dated branch; the old `sunholo/main-dst` is not rewritten, so the previous pin
stays fetchable. Checked on the rebased tree: motoko's `check_core` passes
except the macOS-only `verify_native_path_guard`; the lane profile boots all
three extensions under a policy; and a live
`eval-suite --tool-policy ailang_only` run completes through the lane.
