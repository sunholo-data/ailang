### Fixed — Scope guard accepts rebased mission record pushes

The scope guard refused rebased mission record pushes that inherited sibling mission files from dev. It now verifies dev against the push target and checks both the net diff and each branch-authored commit on rewritten pushes. Branch-authored harness changes remain refused for product missions, even if a later commit reverts them. This fixes `push:scope-guard:rebased-record-inherited-sibling-paths`.
