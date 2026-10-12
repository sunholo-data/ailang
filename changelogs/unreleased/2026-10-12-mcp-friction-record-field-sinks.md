### Fixed

- Enforce `{not label}` on plain record-field writes and function-field arguments across local aliases, nested records, updates, bindings, and returns. Reject unsupported annotations and transfers that erase callback domain restrictions instead of ignoring them. Source-label regressions retain their existing behavior.
- Invalidate earlier compile-cache entries with version v8 so cached acceptance cannot bypass the new IFC checks; v7 is reserved by concurrent PRs. Refs #1618.
