### Fixed

- `ailang run --policy` now marks worker stderr lines beginning with `policy:` or
  `policy-result:` as `worker: `, preventing programs from forging supervisor
  admission and limit verdicts. Supervisor lines start on fresh LF-delimited
  lines even after unterminated worker output. Markers do not consume the output cap.
- Exit 3 is reserved for actual supervisor limits. A worker exit 3 now becomes
  exit 1 with a `worker_reserved_exit` envelope naming the original code;
  supervisor `timeout` and `output_limit` envelopes and exit 3 are unchanged.
- The pi host selects the final LF-delimited contract line, preserves marked
  worker output, and rejects malformed final JSON without falling back. Refs #1548.
