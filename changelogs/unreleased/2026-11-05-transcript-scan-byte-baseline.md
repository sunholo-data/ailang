### Fixed — Transcript secret scan byte baselines

Mission controller fires now take a bounded, metadata-only transcript size snapshot before spawning and scan only appended bytes in existing files afterward. Historical secret-bearing content followed by a harmless append no longer generates a fresh alarm. New files are scanned in full; truncated files and unavailable or malformed baselines use an explicit whole-file fallback. Scan summaries report the loaded baseline count or `none`/`failed`.

Private notices describe the fire's write window when the byte baseline is usable, including concurrent sessions. Whole-file fallback notices state that write time is unknown, and public notices contain only the hit count. Snapshot and scan artifacts are cleaned up without changing the controller's recorded result. Historical exposures remain parked under D-FLEET-42; this change does not rescan that evidence.

Snapshots use Python 3 filesystem metadata without opening transcript contents or secrets, publish atomically with mode 0600, and fall back visibly if collection fails or exceeds 30 seconds. Same-size or larger rewrites, values crossing the saved byte boundary, and concurrent writes absorbed during snapshot collection remain limits of the byte-offset approach.
