### Fixed — pricing consistency checks use managed-agent model pins (2026-10-22)

The offline pricing gate now compares a pinned Gemini managed agent with the
native row for its pinned model. An unpinned agent retains its own identity.
This fixes the release-blocking false disagreement between the unpinned
Antigravity agent and its Argon-pinned sibling, while continuing to catch rate
drift between the pinned agent and native Argon. No model rates changed.
