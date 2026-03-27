# AILANG Standard Library

The standard library provides core types and utilities for AILANG programs.

## Modules

| Module | Description |
|--------|-------------|
| `std/option` | Option type (`Some`, `None`) for nullable values |
| `std/result` | Result type (`Ok`, `Err`) for error handling |
| `std/list` | List operations (map, filter, fold, etc.) |
| `std/json` | JSON encoding/decoding with accessor functions |
| `std/io` | Console I/O operations |
| `std/fs` | File system operations |
| `std/clock` | Time operations |
| `std/math` | Mathematical functions |
| `std/string` | String manipulation |
| `std/array` | Array operations |
| `std/crypto` | Cryptographic hashing and RSA verification (v0.9.4+) |
| `std/jwt` | JWT parsing, RS256 verification, and claims helpers (v0.9.5+) |
| `std/sem` | Semantic caching types (v0.5.11+) |

## std/sem - Semantic Caching (v0.5.11)

Provides types for semantic frame caching with near-duplicate detection:

```ailang
import std/sem (sem_frame, make_frame_at, encode_frame, decode_frame, is_similar)

-- Create a semantic frame
let frame = make_frame_at("doc-123", "content text", opaque_bytes, timestamp)

-- Check similarity (Hamming distance threshold)
let similar = is_similar(frame1, frame2, 10)

-- JSON serialization
let json = encode_frame(frame)
let decoded = decode_frame(json)  -- returns Option[sem_frame]
```

**Types:**
- `sem_frame` - Cache entry with content/opaque separation
- `simhash64` - Type alias for 64-bit locality-sensitive hash
- `similarity_match` - Result of similarity search
- `UpdateResult` - CAS update result (`CASSuccess` | `CASConflict`)

**Functions:**
- `make_frame_at(id, content, opaque, ts)` - Pure frame constructor
- `make_frame(id, content, opaque)` - Frame constructor with Clock effect
- `encode_frame(f)` - Serialize to JSON (bytes as base64)
- `decode_frame(json)` - Parse from JSON
- `is_similar(f1, f2, threshold)` - Check Hamming distance
- `has_embedding(f)` - Check if frame has embedding
- `bump_version(f)` - Increment version for CAS

## Usage

```ailang
import std/option (Option, Some, None)
import std/list (map, filter)
import std/json (decode, get, asString)

-- Use imported functions directly
let doubled = map(\x. x * 2, [1, 2, 3])
```
