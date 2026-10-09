### Added — typed cloud task inputs (Refs #1600)

Messages accept `--inputs-file` and HTTP `inputs`, preserved through SQLite,
Firestore, task records and dispatch. Trusted per-agent `inputs_allow` exact repo
grants deny by default; unauthorized inputs fail permanently before dispatch.
The cloud job parent fetches branch/tag data, verifies file/manifest SHA256 pins,
stages a bounded batch and delivers new files before the executor. Defaults are
excluded under `.incoming/<n>/`; explicit destinations support binary assets.
Traversal, symlinks, overwrites and harness instruction paths are refused.
Job logs and completion summaries include commit and digest provenance. Local
inputs fail explicitly. Deployment and the Daneel sender migration remain pending;
see `examples/task_inputs/README.md` for staged acceptance.
