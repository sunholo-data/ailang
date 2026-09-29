### Added

- Route local pi and opencode eval lanes through explicit leased `ollama-rig`
  providers while keeping everyday and cloud Ollama routes lease-free.
- Register motoko subprocesses in the rig-lock child ledger between process
  start and wait.
- Reconcile complete-minute ollama traffic against the rig-gateway ledger,
  with durable cursoring, overlap suppression, observable input errors, and
  hourly-throttled control-plane bypass alerts.
