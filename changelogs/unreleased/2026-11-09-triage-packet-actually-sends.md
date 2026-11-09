### Fixed — triage mission: the daily decisions packet is sent, and records stop stacking

- The packet was never sent after 2026-11-04. The §7 urgent rule ("only for something Mark has not
  been mailed before") suppressed it every time, because every row in a packet is inherited. The
  packet is now the day's `[urgent]` by design. Whether it is due is a mechanical test (time window
  plus `~/.ailang/state/mission-triage-last-packet` older than 12h), and the Gate-5 report must
  say `packet: sent <id>` or which condition failed.
- Decisions no longer appear in the `[normal]` digest, so there is one place to answer.
- §7's example no longer uses the banned "A: review X" form.
- Record PRs: one open at a time, opened ready rather than draft, a failed run unrelated to the
  record's files is rerun once, and the unmerged record is named in the digest and the packet.
  8 stacked drafts (#2114 … #2170) had kept D50–D79 off `dev`.
