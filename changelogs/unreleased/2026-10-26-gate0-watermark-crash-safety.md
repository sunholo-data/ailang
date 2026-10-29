### Fixed — Gate-0 directive watermark moves only after a directive is processed (#981) (2026-10-26)

Mission-control fires advanced the bookkeeping-issue watermark before routing a
human directive, then died, so the next fire never saw the directive. Two of
Mark's answers on #852 were lost this way. One sat below an advanced watermark.
The other's `createdAt` equalled the watermark, and the read was exclusive.

`scripts/mission_directives.sh` now reads with an inclusive boundary and prints
each directive's comment `id=`. Given `--watermark-file`, it labels ids already
processed and warns when a comment sits unrecorded at the watermark, which is
the #981 signature. A new `--mark <id>` re-fetches the issue and records the id.
It then holds the watermark at the oldest directive not yet processed, so an
out-of-order mark or a directive posted mid-iteration cannot be skipped. The
file is written by temp file and rename. Reading never writes it, and line 1
stays a bare timestamp for `mission_gate0_self_notices.sh`.

`--mark` holds a `mkdir` lock across its read-modify-write and breaks a lock
whose holder has died. The old `--since` read is unchanged, byte for byte
(exclusive, no ids), and now prints a deprecation hint. The world fork still
calls it with a hand-written watermark. A `BOUNDARY` warning names each
re-presented comment by time, id and first line. The skill then gives one rule:
found in the ledger, mark it without routing; not found, route it, then mark.

The Gate-0 skill text now says process-then-mark. A directive is marked only
after its ledger row, pick and report acknowledgement have landed. The old
"before routing" sentence is gone. `scripts/test_mission_directives.sh` runs in
`make test-launchd-drivers`. It kills a simulated fire at every step and
asserts each directive is routed exactly once. A control arm shows the old
order still loses one. See the
[sprint plan](../../design_docs/planned/sprint-plan-gate0-watermark-crash-safety.md).
