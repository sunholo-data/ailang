### Fixed — package cascade notifications dispatch again (#1730, P0)

No cascade had dispatched since 2026-09-17. `ailang publish` wrote the upgrade-available row to
the publisher's local SQLite store, then published a cascade notification naming that local id.
The cloud coordinator requires the row (hydration) and dropped every notification: "names a
message that does not exist in the store". When a cascade notification's row is missing, the
coordinator now rebuilds it from the envelope the notification carries, using the same builder
`ailang publish` uses (`messaging.UpgradeAvailableMessage`). It stores the row under the
notification's id (`PutMessageIfAbsent`, so a redelivery cannot write it twice) and dispatches.
Readers that look a task's message up by id, such as the autonomy router, now find it too.
Notifications from other topics that name a missing message are still dropped.
