### Docs — the Discord webhook's Keychain read fails when another user holds the console

`notification-channels.md` presented `security find-generic-password` from a user LaunchAgent as
something that simply works. Two things stop it, and both are silent because channel registration
is fail-closed and reports failure in a single log line:

- the agent must load into the **Aqua** session, which `ailang daemon install` now sets; and
- even in Aqua, the **login Keychain is locked whenever a different user holds the console**.

The second is the sharper trap on a shared machine. Measured on the rig 2026-09-30: the webhook item
had been present since 2026-05-28, the job was confirmed running in `domain = gui/501`, and the read
still exited 36 (`errSecInteractionNotAllowed`) printing nothing — because `/dev/console` was owned
by `daneel` while the daemon ran as `voightkampff`. `security show-keychain-info` on that keychain
returned `User interaction is not allowed` as well.

The guide now says not to use the login Keychain for this secret on a multi-user or
fast-user-switched host, and points at the env var (with `chmod 600`) or the System keychain, which
is unlocked at boot and already on the default search list.
