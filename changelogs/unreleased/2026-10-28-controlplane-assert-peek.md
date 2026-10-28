### Fixed — Gate-5 delivery assertions preserve unread controlplane digests (2026-10-28)

Gate-5 rule 3(c) now verifies the delivered body with `ailang messages read --peek --json <id>`,
keeping the digest unread for Daneel's hourly triage. All flags must precede the id: Go's flag
parser silently ignores a trailing `--peek`, which would mark the message read. A repository-file
guard locks the safe command and flag order, alongside the generated skill mirror's sync check.
See the [sprint plan](../../design_docs/planned/sprint-plan-controlplane-assert-peek.md).
