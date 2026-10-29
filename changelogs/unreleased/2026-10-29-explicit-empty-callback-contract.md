### Breaking — soundness: explicit empty callback contracts are closed (refs #1958)

- A callback function type written `! {}` requires a pure value. An effectful
  value is rejected even when its enclosing caller already declares the effect.
- An omitted callback row remains implicit and open. Nonempty concrete rows and
  declared row-polymorphic contracts retain their existing behavior.
- This change targets v0.56.0. Use an omitted row for intended implicit callback
  effects, or declare the actual callback effect contract explicitly.
- Imported aliases, interfaces and warm compilation retain the contract. The
  `explicit_empty_callback_contracts` example stores a pure probe and IO renderer
  through a pure helper without invoking the renderer during construction.

- [Implemented design](../../design_docs/implemented/v0_56_0/m-effect-explicit-empty-callback-contract.md) and companion evidence are archived after independent acceptance; v0.56.0 remains unreleased.
