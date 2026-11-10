### Changed — Per-request API-key lane retired (2026-11-10)

The "external" lane from M-CLOUD-DUAL-AUTH (v0.9.2) is gone. An outside caller could put an
Anthropic key in `POST /api/messages` (`anthropic_api_key`). The coordinator KMS-encrypted it
into an in-memory cache, and the dispatcher sent the task to a `*-apikey` Cloud Run job. That
job decrypted the key and billed the run to it. None of the five `*-apikey` jobs ever ran in
prod. Mark ruled on 2026-11-10 to retire the lane.

- `POST /api/messages` with `anthropic_api_key` in the body, empty or not, now gets a
  `400` that names the retirement. Nothing is stored. Before, an expired cache entry dropped the
  key and the task ran on the coordinator's own credentials with no warning.
- Removed: the coordinator's key cache (`apikey_cache.go`), its KMS encrypter (`kms.go`), the
  executor's `ENC:` KMS decrypt (`internal/executor/claude/kms.go`), the `-apikey` job suffix,
  `DispatchParams.APIKey`, and the `AILANG_KMS_KEY` registry entry.
- `auth_mode: apikey` now works only together with `credit_account`, in the guarded Claude
  credit lane (its `-claude-credit` jobs and `AILANG_AUTH_MODE=apikey` are unchanged). Without
  `credit_account` the cloud dispatcher fails the task permanently with an error naming the
  retirement. It no longer launches a job that does not exist.
- Deploy order: this change must reach prod before ailang-multivac#16 deletes the five jobs,
  the `agent-external` service account and the coordinator's `AILANG_KMS_KEY`. A coordinator
  without this change would cache an unencrypted user key and dispatch it to a job that no
  longer exists. #16 also removes the jobs from `scripts/cloudbuild-lib.sh`, the only list the
  dev and release builds roll by name.
