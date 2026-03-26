# M-STDLIB-UUID4: Add uuid4() to std/rand

## Status: Implemented
## Version: v0.9.4
## Priority: Nice-to-have (P3)
## Effort: Small (1-2 hours)

## Problem

DocParse agent requested uuid4() for API key generation. Currently using 32x rand_int hex char approach which is verbose and error-prone.

## Design

### New Builtin: `_uuid4`

- **Module**: `std/rand` (extends existing module)
- **Signature**: `func uuid4() -> string ! {Rand}`
- **Returns**: RFC 4122 v4 UUID string (e.g., "550e8400-e29b-41d4-a716-446655440000")
- **Effect**: Rand (uses crypto/rand internally)
- **Pure**: No

### Implementation

1. Add `_uuid4` builtin in `internal/builtins/rand.go`
2. Use Go's `crypto/rand` for secure random bytes
3. Format as standard UUID v4 (set version and variant bits)
4. Export from `std/rand.ail` as `uuid4()`

### AILANG Interface

```ailang
-- In std/rand.ail
export func uuid4() -> string ! {Rand} = _uuid4()
```

### Usage

```ailang
import std/rand

let key = rand.uuid4()  -- "550e8400-e29b-41d4-a716-446655440000"
```

## Testing

- Unit test in `internal/builtins/rand_test.go`
- Verify format matches UUID v4 regex
- Verify uniqueness across multiple calls
- Example file: `examples/uuid.ail`

## Alternatives Considered

- Separate `std/uuid` module: Overkill for a single function
- uuid5/uuid3 (name-based): Not requested, can add later

## Origin

Agent message d2f2668a from docparse (2026-03-19)
