### Fixed — `ailang mission rotate-log docs --stream status` refused the docs STATUS archive (2026-10-17)

A tombstone section, `## (Charter sections — moved back 2026-10-01)`, sat between the entries of
`design_docs/docs-mission-status-archive.md`, so rotation correctly refused to archive it as part of
an entry. It now sits in a new preamble above the first stamp, with its text unchanged, and the
docs charter tells the loop to file new stamps below that preamble rather than at the file's top.
The refusal is unchanged. A new ratchet test, `TestCanonicalRecordStreamsStrayHeadingsRatchet`,
counts interleaved structural headings in every live record stream (12: motoko's status
structure and one irregular docs-log record) and fails if the count changes, so the next one
fails CI in the commit that adds it.
