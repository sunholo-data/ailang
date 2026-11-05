
### Fixed — parking window survives release churn

- `tools/eval-park-set.sh` windows each benchmark over the last N banks that actually RAN it, not
  the last N banks overall. On 2026-11-04 two quick releases (v0.57.0, v0.57.1) that had not yet
  reached `cp_60166` pushed its old rows out of the global window (3 < min-rows), un-parking it
  with no new evidence and queueing ~9 h of full trials it has timed out on every time.
