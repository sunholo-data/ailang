### Fixed — package quality badge describes verifier skips accurately (2026-10-23)

PUB016 no longer describes every skipped contract as an unsupported type or
builtin. The verifier also skips functions without postconditions. The badge
continues to direct authors to per-function reasons; counts, proof status and
publish gates are unchanged. Reported by the `sunholo/audio` roadmap audit.
