### Fixed — mission-control's controlplane and approvals sends are scoped to the prod store (2026-10-28)

Gate 5 and Gate 3 now prefix their `ailang messages send controlplane` / `send approvals` commands, and the Gate-5 3(c) read, with `env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac`, so they no longer depend on a shell dotfile; the local store refuses `controlplane` (gate5:controlplane-send-refused-local-store). A repository-file guard fails if a prescribed send or read is unscoped.
