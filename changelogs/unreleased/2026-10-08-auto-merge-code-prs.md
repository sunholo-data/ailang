### Added — per-agent code PR auto-merge (2026-10-08)

- Per-agent code PR auto-merge (`Refs #1599`): trusted opt-in, declared scope,
  base check-name verification, second-user approval and durable PR audit.
  Approval failure attempts to disable native auto-merge. Markdown remains the
  default; production activation requires confirmed repository protections.
