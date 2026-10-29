### Added

- `ailang publish --key-file PATH` (also `ailang pkg publish`) reads an explicit
  protected macOS/Linux publisher key once before package subprocesses. It
  requires a regular effective-user-owned file with mode0400/0600, rejects final
  symlinks and unsafe/control-byte/oversized contents, and never exports the file
  key into subprocess environments. Windows refuses the file lane explicitly.
- Publishing refuses conflicting file/environment sources or missing credentials
  locally. The legacy environment lane remains available with an inheritance
  warning. Help and dry-run do not read credentials; exact selected-key echoes in
  validator errors are redacted. Operator confinement remains authoritative.
