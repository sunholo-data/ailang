### Fixed

- Callback records behind local or imported transparent type aliases now retain
  their enclosing binder ownership while being forwarded. Pure forwarding
  no longer fails with an unresolved effect tail or missing callback-row
  invariant. Actual callers still declare the IO/FS effects of stored callbacks,
  including callbacks not invoked by the receiving function.
