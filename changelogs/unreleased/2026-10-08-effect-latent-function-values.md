### Breaking — soundness (v0.53.0): latent function-value effects

- See #1326 and #573. Pure callers now reject undeclared effects when handing a
  named callback to an unannotated/effect-polymorphic HOF or invoking an effectful
  record field. Add the performed row (for example `! {IO}` or `! {FS}`) to the
  caller. Hook construction and storage-only concrete callback contracts remain pure.
- External consumers to audit: motoko_agent `ExtCtx.ports` hooks, docparse,
  ailang-parse. This acceptance-changing correction ships in a minor release with
  no opt-out. Concrete function annotations retain labels, budgets and modes;
  their width remains open (closed upper bounds are deferred).
- This lands before #616, which will extend the same per-application publication
  with resolved call rows. Row-tail union/difference semantics are unchanged.
- Known limitation: an effectful callback nested inside a list, tuple or ADT argument is not yet charged; tracked in #1718.
