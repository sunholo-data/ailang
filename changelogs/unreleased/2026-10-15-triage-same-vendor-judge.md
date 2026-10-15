### Changed — triage mission: verifier and judge may share a vendor (D-TRIAGE-10)

- T1 now requires a judge that is a different *model* from the verifier, not a different vendor. The fleet-default haiku executor rung is re-enabled in `mission-triage.env` (reverting the override from 39bbbcbd2); the resolver's family check still reroutes a same-model judge.
