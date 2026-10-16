### Fixed — triage never marks a draft PR VERIFIED

- Gate 3 verifiers now read `isDraft` and report a draft PR as `NEEDS-DECISION` instead of
  `VERIFIED`. Draft PR #1754 was listed "ready for Mark to approve" and its merge was refused.
