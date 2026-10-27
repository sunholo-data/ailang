### Cloud executors preserve native provider sessions

- Persist pi sessions in task-scoped cloud storage while keeping local and eval runs ephemeral.
- Bank native Claude, Codex and pi session paths alongside existing provider evidence; retain motoko's existing path contract.
- Copy native session bytes without truncation to `tasks/{taskID}/session.jsonl` for all four providers, including failed executor results. Retry missing, unreadable or empty files three times with two-second intervals and report searched paths on failure.
- Record `session_provider` and `session_jsonl_bytes` in task metrics. Deterministic dependency bumps skip session discovery.
- Live cloud canaries and Holosun nightly mirror verification remain pending deployment.
