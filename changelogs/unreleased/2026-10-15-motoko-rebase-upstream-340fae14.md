### Changed — motoko eval build rebased onto upstream main 340fae14

The rig's `motoko` shim and the cloud `agent-motoko` image now run `sunholo/main-dst-20261015` @ `d5aa3e31`:
our profiles and lane commits rebased onto arniwesth/motoko_agent `main` 340fae14 (2026-10-14), 346 commits
newer than the previous base. Three of our fixes are now upstream's (#220, #260, #262). `make check_core`
passes in full. Rollback and details: MOTOKO.md §11.
