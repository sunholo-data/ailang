### Breaking — soundness: explicit empty callback contracts are closed (refs #1958)

- A callback function type written `! {}` requires a pure value. An effectful
  value is rejected even when its enclosing caller already declares the effect.
- An omitted callback row remains implicit and open. Nonempty concrete rows and
  declared row-polymorphic contracts retain their existing behavior.
- This change targets v0.56.0. Use an omitted row for intended implicit callback
  effects, or declare the actual callback effect contract explicitly.
