### Fixed: cloud completions for an unknown task were redelivered by Pub/Sub for 24h

The completion handler meant to ack a completion whose task this coordinator does not have,
but it only checked for a nil task. Both stores report a missing task as an error instead
(Firestore `task not found: <id>`, SQLite `sql.ErrNoRows`). That error reached the push
endpoint, which answered 500, so Pub/Sub kept redelivering the orphan for its full retention.
The dev coordinator's logs were full of `task not found: sec2probe-*`. `GetTask` now wraps a
new `coordinator.ErrTaskNotFound` in both stores (SQLite also keeps `sql.ErrNoRows` in the
chain), and the handler acks on it. A real store failure is still returned, so Pub/Sub still
retries it.
