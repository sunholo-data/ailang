### Fixed — `serve-api --caps SharedMem` (and `SharedIndex`) now works

- `serve-api` granted the capability but never initialised the store, so every
  `std/sharedmem` call failed with "SharedMem effect not enabled (use --caps SharedMem)", even
  with the flag set. Only `ailang run` called `runner.SetupSharedMemHandler` /
  `SetupSharedIndexHandler`. `serve-api` now calls the same setup (`initServeAPIStores`), and
  `TestInitServeAPIStores` pins both directions: present when granted, absent when not.
- Found by the `sunholo/mcp_oauth` end-to-end flow, whose example service keeps its OAuth store in
  SharedMem.
