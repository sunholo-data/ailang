### Added — design: motoko as the harness inside the ailang_only executor boxes

`design_docs/planned/m-motoko-ailang-only-lane.md` plans how motoko replaces pi
as the harness for the `ailang_only` lane without changing its security boundary:
- **The lane's tools:** a new extension, `motoko-ext-ailang-policy`, provides the
  8 lane tools through the same `ailang policy-tool` and `ailang run --policy`
  endpoints pi uses. It denies motoko's native tools; verdicts merge Deny-first.
- **Fail-closed loading, a rollout precondition:** the lane does not roll out
  until upstream motoko enforces `extensions.strict` (arniwesth/motoko_agent#205,
  ours).
- **Executor checks:** the executor reads the profile through motoko's own
  resolver and probes that the build enforces `strict`.
- **The switch to motoko waits on a paired same-model A/B against pi.**

Spike A, running motoko's runtime under a restricted policy, was measured and
ruled out. Its Node env-server runs shell commands outside any AILANG policy, so
box-level hardening of the executor image is the harness-independent boundary.
Step 1, refusing restrictions motoko cannot enforce, shipped in #1429.
