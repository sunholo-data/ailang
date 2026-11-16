### Fixed — design-quorum: the Anthropic seat runs on an unreadable quota (2026-11-16)

D-FLEET-28 A (Mark, attended). The quorum's Anthropic seat (`claude-sonnet-5@claude-p`)
was refused whenever subscription usage could not be read, because it gated on
`AnthropicQuotaObservation.Blocked()`, which is true for `unknown` as well as `over`. With
the usage endpoint answering 429 and the `claude -p /usage` fallback dead, the seat was
absent from every quorum. It now refuses only on a measured refusal (`over`, `expired`) and
runs on `unknown`, matching the mission driver's rule in `tools/launchd/lib/lane-probe.sh`
("an unreadable quota is not an exhausted one"). `Blocked()` is unchanged, so admission and
`mission quota --over` behave as before. Test:
`cmd/ailang/design_quorum_anthropic_seat_test.go`.
