# #1771 narrow encoded-artifact evidence audit

Reviewed clean committed evidence at `40a5ca9eac5b4175a366fc441c451f4dfb0f12aa` after source-owner confirmation. **No material data discrepancy found within this scope.** This is a data/ceiling/cache-evidence audit, not a standalone sprint verdict. Root retains formal M1–M2 evaluation and exact-head CI. No builds, test runs, source/audit writes or GitHub mutations were performed.

| Actual encoded-file measurement | Motoko session | Post-alias social/protocol |
|---|---:|---:|
| Measured modules |143|97|
| On-disk module artifact files |715|485|
| Authenticated payload SHA256 checks |572|388|
| Total encoded module-artifact bytes |52,171,074|48,230,073|
| Maximum blob bytes |3,100,965|3,606,290|
| Maximum module bytes |3,400,869|4,664,163|
| Extracted compact IFC JSON bytes |473,459|417,677|
| Exports |1,365|1,223|
| Recorded cold / warm seconds |43.982 / 1.610|18.678 / 1.332|
| Cold cache hits / misses |0 / 143|0 / 97|
| Warm cache hits / misses |143 / 0|97 / 0|

Every per-file size and per-module sum matches both raw measurement and committed report. Every actual payload hash matches its `artifacts.json`; each stamp is epoch v8 and matches the cache index key. Every iface is schema `ailang.iface/v2`. Maximum blobs/modules remain below the unchanged **67,108,864-byte (64 MiB) /134,217,728-byte (128 MiB)** limits; no ceiling adjustment or artifact exclusion was inferred.

DAG preservation is visible in the actual payloads: a small read-only Gob wire-header decoder independently checked Format 1 and node count for every measured `coretypeinfo.gob`. Motoko sums 19,296 encoded DAG nodes. The supplied actual Gob-decoder output, restricted to the same 143 modules, sums 186,493 CoreTI IDs/roots, with each module's root count matching its CoreTI count; independently read wire node counts match that per-module output. The v7 shared-DAG codec file is byte-identical to actual dev (Git blob `8c3bc788180608fb926f83f95b88fa361a950c2e`) while cache identity advances to v8.

All 143 reported source hashes match real source files and the pinned consumer/stdlib Git blobs. Summed source footprint is 58,174 lines per compiled-module graph; the actual pinned session entry is 5,824 lines, SHA256 `547ba104253b67bd6458f40c88b683e861ec94e11cdde3edbf46bb047d891fe2`. This confirms the large gate concerns the original delivered consumer 083ef437, not only the smaller historical navigation representative.

Cold and warm 143/97 manifest snapshots are identical in hash and nanosecond mtime; current corresponding files still match. All actual blob hashes authenticate against those unchanged manifests, and no payload blob is newer than its module stamp. Raw logs report the matching 143→143 and 97→97 cache transitions and successful compilation. Recorded wall times are consistent with logged phase totals; no timing experiment was rerun.

Important qualifications:

- Stamps record per-module **artifacts.json** manifests, not separate historical mtime snapshots of every individual payload blob. Payload byte identity is authenticated by hashes from those unchanged manifests.
- IFC 473,459 is compact JSON size of extracted `ifc` objects. The 52,171,074 total is actual on-disk module-artifact bytes, including formatted interfaces. Neither number is a measured incremental v1-versus-v2 overhead; the shared cache index is separate (2,505,223 bytes for the current Motoko index).
- Later `gmake` added 49 modules to the Motoko cache (now 192). This audit correctly restricts totals and stamps to the original 143 session modules; it does not sum the enlarged directory as if it were the original measurement.
- The 143-module Motoko proof uses clean compiler 8f6b11d0f (binary SHA256 `ae819bd2acb221d38f7c094b65426099647f64f5801ddbb04a8dbd13040e7295`) before alias integration. The separate 97-module social/protocol proof uses a9971a364 (production source; only the evidence JSON was dirty at build) (binary SHA256 `0d95f20eda766a23572abaeda56ef1cc1db314ae838e4533e07cfb9b9e1d88d7`). Both saved binaries match those actual hashes. These are distinct recorded cohorts, not a claim that the full143-module experiment was repeated on final 40a.

The state correctly keeps M1/M2 measured complete, M3/M4 null/unstarted and the overall sprint in_progress. This evidence supports the bounded M2 persistence/size gate; it does not establish imported-call or REPL enforcement, release validation, or closure of #1134/#1132. Historical navigation evidence remains separately identified.

Machine results: `1771-independent-artifact-audit.json`; reproducible read-only checker: `1771-independent-artifact-audit.py`. The final supplementary binary/source/codec checks are recorded in the JSON. Raw owner measurements and committed evidence were not modified.
