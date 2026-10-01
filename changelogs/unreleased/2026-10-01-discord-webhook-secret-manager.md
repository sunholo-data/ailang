### Added — the notify daemon reads its Discord webhook from Secret Manager

The macOS login Keychain is the wrong store for a fleet daemon, and the rig proved it. Its webhook
item had been present and valid since 2026-05-28, the job was confirmed running in the Aqua domain
(`launchctl print gui/501/com.sunholo.ailang.daemon` → `domain = gui/501`), and every read still
returned `errSecInteractionNotAllowed` — because `/dev/console` was owned by a **different** user
(`daneel`) while the daemon ran as `voightkampff`. That locks the login Keychain to background
processes. Channel registration is fail-closed, so the only trace was one log line, and the same
breakage recurs on every console switch.

Resolution order is now env → **Secret Manager** → Keychain:

| Order | Source | Use it for |
|---|---|---|
| 1 | `AILANG_DISCORD_WEBHOOK_URL` | an explicit per-host override |
| 2 | Secret Manager `<prefix>-discord-webhook-url` | the fleet default — one value every machine resolves identically |
| 3 | macOS login Keychain | a host with no cloud access |

Secret Manager sits ahead of the Keychain deliberately: a machine that prefers the Keychain is one
console switch away from losing the channel.

- The read is bounded at 5s, because registration is on the daemon's startup path.
- **Registration now names the source it used.** "Discord is on" and "Discord is on, from the store
  you think it is" are different facts, and the second is what was missing while the rig silently
  ran with no channel at all.
- A Secret Manager read that *fails* is logged naming the secret and the project, then falls through
  to the Keychain. The fallback is fine — the channel is fail-closed either way — but it must never
  be silent, or the next outage reads as "not configured" again.
- No project configured is "not configured", not a failure: Secret Manager is not queried and
  nothing is logged as an error.
- No IAM was added: both daemons authenticate with the ADC of an account that is `roles/owner` on
  these projects. The terraform comment says to grant `secretmanager.secretAccessor` if that ever
  becomes a service account.

Terraform creates `${prefix}-discord-webhook-url` per environment (value unmanaged, like every other
secret); `scripts/setup-secrets.sh` populates it from `AILANG_DISCORD_WEBHOOK_URL`.
(`internal/notify/secret_gcp.go`, `register.go`, `cmd/ailang/daemon.go`)
