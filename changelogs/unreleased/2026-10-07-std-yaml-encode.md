### Added — std/yaml.encode

- Added pure `encode(Json) -> Result[string, string]` for deterministic block YAML: quoted keys and strings, two-space indentation, ordered object pairs and one trailing newline. Duplicate keys are preserved on emission and rejected by decode; nested non-finite numbers return `Err`.
- Finite, unique-key Json values round-trip modulo recursive object key order because decode currently sorts keys. Added boundary tests, a decode/edit/encode config example, and reference documentation. YAML-sensitive controls and line separators use Unicode escapes to preserve string values; oversized mapping keys use explicit block-key syntax to satisfy YAML’s simple-key limit.
