### Fixed — a yielding rig-lock holder no longer loses the GPU to whoever is queued

A holder that yields the rig lock (a cooperative handoff, or lending to the
operator) used to take the lock back with the same 30-second `mkdir` retry as
every other waiter. Any job queued in Wait mode, or the filler firing at the
wrong moment, could take the gap once the borrower left, and the holder then
sat idle through that job's whole run. On 2026-11-25 a fine-tune eval lost 2h42m
this way to an attended test. A second hole made it worse: the handoff guard was
checked once before the retry loop, so a waiter already queued never consulted
it again and could walk into a gap meant for the named borrower.

- The lender now writes a reclaim claim (`rig.reclaim`, beside `rig.handoff`)
  before it releases. While the claim stands, every acquirer except the borrower
  a live handoff names is refused (NoWait) or queues (Wait). The lender takes
  the lock back at the 2-second yield interval. A dead, expired or garbled claim
  is ignored and removed.
- The handoff/reclaim guard is checked on every acquire attempt and again right
  after a successful `mkdir`, which hands back a gap that is not the caller's.
  This applies in both `internal/riglock` and `tools/launchd/rig-lock.sh`.
- Daneel's own lock client (`~/dev/daneel/tools/daneel_rig.ail`) does not read
  the claim yet.
