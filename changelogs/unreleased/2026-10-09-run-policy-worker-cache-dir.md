### Fixed

- Restricted `run --policy` honours `AILANG_CACHE_DIR`. Empty or unset overrides use a private per-run compile cache outside `fs_sandbox`, removed after worker termination. Unsafe temporary placement is refused and cleaned up; explicit in-sandbox overrides warn about poisoning risk and `fs_deny_write`. Operator caches and `trusted_host` behaviour are preserved. Refs #1547.
