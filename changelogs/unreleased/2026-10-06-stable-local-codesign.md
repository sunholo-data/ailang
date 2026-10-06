### Changed — `make install` / `make quick-install` re-sign with a stable local identity

When a `ailang-local` code-signing identity exists in the keychain, the installed binary is
re-signed as `com.sunholo.ailang`. `go install` leaves an ad-hoc signature, so macOS privacy
(TCC) treated every rebuild as a new app and re-raised file/network-volume prompts for the
agents ailang spawns. Without the identity the step is a no-op.
