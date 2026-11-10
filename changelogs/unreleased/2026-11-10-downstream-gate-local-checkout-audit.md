### Fixed — `make downstream-check` no longer fails because someone else edited a sibling checkout (2026-11-10)

The gate's write audit treated any path that changed in the operator's own sibling checkouts during
a run as an instrument failure (exit 3, "could not measure"). Those checkouts are not an input: every
arm reads a snapshot, or in `REF=worktree` mode a copy taken before the run. On a machine where
several sessions edit the same trees, the first release-tier run after the corpus fixes measured 0
newly failing and 0 not measured and still exited 3, because another session had saved files in one
checkout.

Changes there are now listed in the report ("Local checkouts that changed during the run") and in
`report.json` as `local_checkout_changes`, and never gate. The audit of the run's real inputs —
snapshots, provisioned trees and the package root copy — is unchanged and still exits 3.
