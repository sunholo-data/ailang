### Added — `ailang coordinator cancel` drops a task that has not started

`ailang coordinator cancel <task-id>... [--remote gcp] [--yes]` cancels tasks that are still `pending`. It shows each task, asks for confirmation unless you pass `--yes`, and refuses anything else, naming its status. For a task awaiting approval, it points you to `reject`, which also resolves the approval card. Before this, dropping a prod task needed a hand-written Firestore write.

### Fixed — a cancel can no longer be undone by an in-flight dispatch

- **`MarkTaskCancelled`** is now a compare-and-set in both stores: pending → cancelled only, otherwise `ErrTaskNotCancellable`.
- **`ResetTaskToPending`**, which a failed dispatch uses to put a task back, now resets only a `queued` or `running` task. It used to be an unconditional write, so a dispatch failing a second after a cancel put the task straight back to `pending`.
