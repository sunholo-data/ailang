### Fixed: streaming callback effects propagate to callers (refs #1958)

- `std/ai.stepWithStream` and `stepWithStreamRecorded` accept pure callbacks under
  `AI` alone and propagate effectful callback rows to their callers. An IO
  renderer requires `AI, IO`; its IO is no longer silently discharged.
- Both builtin types share the callback row with the enclosing call. The
  `ai_streaming_effect_rows` example demonstrates pure and IO renderers with
  deterministic `--ai-stub` execution.

### Breaking — soundness: open rows cannot lose their labels (refs #1958)

- An annotated IO callback no longer fits a closed pure slot such as
  `std/list.map`, `filter`, `foldl`, or `any`. The check names labels the closed
  row does not permit, for both unification orientations and record rows.
- Use `mapE`/`filterE`/`foldlE` and a row-polymorphic callback parameter
  (`! {e}` on the callback and caller) for effectful traversal.
- Implicit widening of a pure returned closure to a concretely wider callback
  row is rejected until width subsumption is implemented. Use an explicitly
  annotated lambda matching the callback slot.

- Release target: v0.56.0 (next minor); this breaking guard is not a v0.55.x patch.
- The pinned motoko consumer narrows its `play_chunks` callback slot to `IO`,
  matching Ports while preserving the helper's outer `IO, Trace` effects.
