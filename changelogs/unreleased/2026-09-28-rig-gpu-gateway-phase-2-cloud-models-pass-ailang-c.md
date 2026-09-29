### Added — rig GPU gateway Phase 2: cloud models pass, ailang clients carry the lease, rig cut over (2026-09-28)

Phase 2 of M-RIG-GPU-ADMISSION-GATEWAY. Before the ports could move, three gaps had to close, each of
which would have refused live work:

- **Ollama Cloud models are always admitted.** `kimi-k3:cloud` and `deepseek-v4-flash:0731-cloud` go
  through the local ollama daemon to ollama.com and never touch the GPU, yet the gateway would have
  refused them whenever an eval held the lease. The mission fleet runs on them. The gateway now
  reads the body's `model`. `rig_admission.ail` gains `isCloudModel` (a `:cloud` tag or a `-cloud`
  tag suffix) and a `cloud` ledger label. The ledger records `model`.
- **ailang's own clients send the lease.** motoko reaches local models through ailang's
  OpenAI-compatible client (`OPENAI_BASE_URL=…:11434/v1`), and `--ai ollama:` through the ollama
  client. Neither sent a token, so motoko-local lanes would have been refused under their own
  eval's lock. `riglock.WithLease` adds `X-Rig-Lease` from `AILANG_RIG_LEASE`, reading it on
  every request and sending it only to loopback hosts. Both clients use it.
- **pi sends the lease without the `ollama-rig` provider the design proposed.** A `!` command
  value, `printf %s "${AILANG_RIG_LEASE:-none}"`, on the existing `ollama` provider sends `none` when
  unset, where a `${…}` template would stop pi. So model ids are unchanged, and hand or mission
  pi use still works. opencode uses `"apiKey": "{env:AILANG_RIG_LEASE}"`, and sends no key when
  unset. Both were measured against a capture server (pi 0.85.1, opencode 1.15.7).

- **A yield keeps the lease.** `riglock.Checkpoint` used to delete the lock and re-acquire it, which
  minted a new token and wrote this process's pid as the holder. When a shell holds the lock
  (nightly-eval, rotation filler), every later eval-suite it started inherited the old token and got
  423. Daneel asks for yields four times an hour. The lock also read as held by a dead pid, and so
  stealable, once the child exited. Checkpoint now restores the holder line and token it had before
  the yield (`TestCheckpoint_YieldKeepsTheLeaseAndHolder`, mutation-checked).
- `tools/ollama-tap` now listens on :11436 by default. :11435 is ollama's private port.

The rig moves as follows. `dev.ollama.serve.plist` now binds `127.0.0.1:11435`. rig-watchdog
probes ollama there directly, and probes the gateway on :11434 (any HTTP status counts as alive).
Daneel still holds the lock without a token and is admitted as `legacy-holder`. Its minting is a
cross-repo follow-up, because eparse runs under Daneel's lock and must receive the token first.
