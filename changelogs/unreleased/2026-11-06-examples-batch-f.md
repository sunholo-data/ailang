### Added — examples for std/sem, std/sharedmem, std/sharedindex, std/dom

- Six more CI-verified examples in `examples/runnable/stdlib/`, covering:
  - `sharedmem` cas / get_or / put_if_absent;
  - `sem` frames (with_embedding, has_embedding, get_simhash, is_newer, validate_namespace,
    create_or_update_frame);
  - `sharedindex` embedding upsert and search;
  - the `dom` batch API's no-handler fallback.

### Fixed — verify-examples granted neither SharedMem nor DOM

- `scripts/verify_examples.go` now grants `SharedMem` and `DOM` to examples that need them.
  Without them, those examples failed with "effect requires capability".
