# Effect Row Mismatch

**Error kind**: `row_mismatch` (effect variant)

Raised by the type checker when two effect rows cannot be reconciled — one row has labels the other lacks.

## Message format

### Extra effects (introduced in actual row)

```
has extra effects: {AI (introduced at src/agent.ail:10:5), FS}
```

When the inferred effect row carries provenance (populated via `Row.Provenance`), each extra label is annotated with the source span where it was introduced. This is typically the call site of an effectful builtin.

### Missing effects (required by expected row)

```
missing required effects: {FS (slot at src/iface.ail:3:20)}
```

When the expected effect row carries provenance, each missing label is annotated with its declaration site (the "slot" in a type annotation or interface).

### Without provenance (legacy format)

```
has extra effects: {AI, FS}
missing required effects: {FS}
```

Both formats can appear in the same error when some labels carry provenance and others do not.

## Example

```ailang
-- Declared: func fetch(url: string) -> Result[string, Error] ! {Env}
-- Body calls ask(), which has effect ! {AI}
-- Result: "has extra effects: {AI (introduced at fetch.ail:7:12)}"
```

## Fix

1. Add the extra effect to the function's declared signature: `! {Env, AI}`
2. Remove the call that introduces the unwanted effect
3. Wrap the effectful call in a handler that eliminates the effect

## Provenance availability

Provenance is populated only when the `Row` was constructed with a source span (e.g. by the pipeline validator or explicitly in test code). Effect rows created by the type checker's constraint solver may not carry provenance for all labels.
