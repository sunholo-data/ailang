### Fixed — sprint milestone LOC placeholders (2026-10-08)

- Sprint creators now use JSON `null` for unfilled LOC estimates. Validators accept numeric zero for net-zero milestones and reject null or missing estimates; session displays show `unset`. Both skill trees share the same contract. Closes #563.
