### Added — fleet-wide rulings ledger and `ailang mission doctor` ruling notes (D-FLEET-25 A)

`design_docs/fleet-rulings.md` holds rulings that bind every mission. The five in-repo charters and Gate 0 of the
mission-control skill link to it. `ailang mission doctor` now prints `note` findings for registry, charter or
mission-skill text that contains a literal retired by an active ruling (`ruling-contradiction`), plus
`ruling-link-missing`, `rulings-malformed`, `rulings-ledger-absent`, `ruling-file-unreadable` and `rulings-skipped`.
Notes never change the exit code. The doctor now prints `N mission(s): no drift` whenever nothing drifted, after any notes.
Design: `design_docs/planned/m-fleet-rulings-ledger.md`.
