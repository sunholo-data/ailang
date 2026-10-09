## Fixed

- Policy-supervised runs authenticate stderr contract lines: worker `policy:` and
  `policy-result:` output is preserved behind `worker: `, including CR and Unicode
  line boundaries and tokens split across writes. Supervisor lines always begin
  on fresh LF-delimited lines after worker output drains.
- Exit 3 is reserved for supervisor limits. A worker's own exit 3 now returns 1
  with reason `worker_reserved_exit` and a message naming the original code.
  Genuine timeout/output-limit kills retain exit 3 and their existing envelopes.
- The pi host selects the last LF-delimited contract line and preserves marked
  worker diagnostics; admitted stdout cannot substitute a forged denial decision.

Refs #1548.
