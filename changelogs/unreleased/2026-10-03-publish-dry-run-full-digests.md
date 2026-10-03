### Fixed — `publish --dry-run` printed digests truncated to 68 bits (2026-10-03)

The tarball, content and interface digests were cut to `sha256:` plus 17 hex characters, so
operators had no way to check a full digest before an immutable publish. They are now printed in
full, together with the v2 interface hash the validator recomputes. (#636)
