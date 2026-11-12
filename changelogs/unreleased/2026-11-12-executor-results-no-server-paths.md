### Fixed
- **Hosted executor results no longer name server paths or digests.** `ailang_run` and
  `ailang_eval` returned the supervisor's policy echo as-is on stderr. That echo includes the policy
  file's path on the server (`/srv/executor-policy.toml`), its digest and the module-graph digest.
  OpenAI's app review asks that results carry no internal identifiers. The echo now drops those
  three keys (`publicPolicyLine` in `executor_tools/executor.ail`) and keeps the caps, timeout,
  limits and decision.
- **The docs tools' missing-file errors name the file relative to the snapshot**
  (`missing:unscoped/roadmap.json`), not the server's `/srv/snapshot/...` path
  (`mcp_tools/lib.ail`). This changes the error text only. The missing files themselves (roadmap,
  design docs, examples, compact prompt, benchmark runs and the compare index) are a separate
  snapshot-builder gap, recorded in the executor directory submission kit.
  `executor_oauth/test_oauth.sh` now has 50 checks, including both of these.
