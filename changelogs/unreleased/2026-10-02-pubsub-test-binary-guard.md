### Fixed — tests no longer publish to prod Pub/Sub

`internal/pubsub.NewClient` now refuses to connect from inside a test binary
(`testing.Testing()`), returning `ErrRealProjectFromTest`. Every Pub/Sub client in
the codebase is built through it.

Five coordinator tests (full chain, handoff sender, landed-card sweep, stage
crossing) loaded the developer's `~/.ailang/config.yaml`, which has Pub/Sub
enabled for `ailang-multivac`. They published handoff notifications for messages
that existed only in their temporary stores. Every subscriber nacked them until
the dead-letter policy gave up. On 09-30 and 10-01 that caused 209 HTTP 500s on
the prod coordinator and about 3,000 nacks on `messages-rig`, under fixture ids
such as `task-aaaa1111` and `task-designer`.
