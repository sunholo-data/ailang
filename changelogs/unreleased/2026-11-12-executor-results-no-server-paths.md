### Fixed
- **Hosted executor results no longer name server paths or digests.** `ailang_run` and
  `ailang_eval` returned the supervisor's policy echo as-is on stderr. That echo includes the policy
  file's path on the server (`/srv/executor-policy.toml`), its digest and the module-graph digest.
  OpenAI's app review asks that results carry no internal identifiers. The echo now drops those
  three keys (`publicPolicyLine` in `executor_tools/executor.ail`) and keeps the caps, timeout,
  limits and decision. A refused program (for example one that declares `FS`) got the same
  path in its stdout decision JSON. On a failed run, a JSON object that has both `decision` and
  `policy` now has those keys removed (`publicRefusal`). Output from a program that ran and exited
  0 is never touched.
- **The docs tools' missing-file errors name the file relative to the snapshot**
  (`missing:unscoped/roadmap.json`), not the server's `/srv/snapshot/...` path
  (`mcp_tools/lib.ail`). This changes the error text only. The missing files themselves (roadmap,
  design docs, examples, compact prompt, benchmark runs and the compare index) are a separate
  snapshot-builder gap, recorded in the executor directory submission kit.
  `executor_oauth/test_oauth.sh` now has 51 checks, including both of these.
