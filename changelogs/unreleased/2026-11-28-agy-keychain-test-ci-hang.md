### Fixed — `TestEnsureLockedKeychain_Darwin` no longer hangs macOS CI

- On a GitHub macOS runner, which has no login session, `security show-keychain-info` on the
  locked test keychain waited for a password. That hung both macOS Build jobs for the full
  15-minute test timeout. The probe is now killed after 20s and the test skips. On a machine
  with a login session, such as the rig, it still asserts exit 36 (locked). Test-only change.
