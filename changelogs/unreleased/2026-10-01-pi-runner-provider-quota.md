### Fixed — mission_pi_run.sh read a provider quota refusal as empty_worktree or ok (2026-10-01)

- pi exits 0 when the provider refuses a request, so the runner judged only the worktree. World iter 196's executor (4 × `429 … weekly usage limit`) banked `empty_worktree` rc 10; World iter 187's planner (4 × `429 … session usage limit`) banked `ok` rc 0 with 10 changed files.
- New verdict `provider_quota`, rc 19: pi finished and its LAST assistant `message_end` is `stopReason:"error"` with a capacity `errorMessage` (HTTP 429/402, usage limit, quota, rate limit, credits). It outranks `ok`/`empty_worktree`, since the work is truncated. A quota error followed by a later successful assistant message does not fire.
- Every verdict, including the preflight failure JSON, now carries `provider_errors` (count) and `provider_error` (last error text, ≤300 chars). A `stopReason":"error"` line that cannot be parsed is counted as `(unparsed provider error)` with a stderr WARNING.
- `tools/launchd/test_mission_pi_run_provider_quota.sh` (wired into `make test-launchd-drivers`) is backed by fixtures captured from real pi output, with provenance in its header.
