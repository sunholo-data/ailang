### Fixed — recovery preserves typed task inputs (2026-10-10)

The coordinator's recovery sweep now copies typed inputs into recovered messages. Previously, recovery could run before normal Pub/Sub delivery and create a task without its declared files; the later notification was then deduplicated. Regression coverage reproduces that ordering, checks all input fields survive in the persisted task, and retains exact repository grant enforcement. Fixes [#1757](https://github.com/sunholo-data/ailang/issues/1757).
