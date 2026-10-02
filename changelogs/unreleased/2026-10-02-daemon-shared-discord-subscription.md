### Fixed — Discord posts each message once, however many notify daemons run

Every device's notify daemon pulls its own messages subscription, and Pub/Sub gives
each one a full copy, which is right for per-machine macOS banners. Since 10-01,
every daemon could also read the Discord webhook from Secret Manager, so each one
posted its own copy and every message reached Discord once per running daemon.

When Discord is registered, `ailang daemon run` now posts remote channels only from
the shared subscription `messages-discord`. A shared subscription hands each message
to a single puller. The per-device subscriptions (including extra message sources)
now reach local channels only, so a Discord failure no longer re-fires the macOS
banner. Task events are unchanged, since their subscription was already shared. The
startup line reports `remote_messages=`.

Needs the `discord` entry in ailang-multivac's `client_subscriptions`. A daemon on
an older binary still posts from its per-device subscription until it is upgraded.
