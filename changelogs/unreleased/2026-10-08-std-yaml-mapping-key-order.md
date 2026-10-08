### Fixed — std/yaml preserves mapping key order
- `yamlToJson` and `decode` now preserve document order in all mappings, including nested mappings, aliases and merges. Merge entries appear at the merge position; explicit keys override merged keys, and earlier merge sources win conflicts.
- JSON bytes change when source mapping order differs from the previous sorted order. Public signatures, purity, scalar bytes, first-document behavior and typed errors remain unchanged.
