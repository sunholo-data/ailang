### Fixed — materialised agent/run policies failed to load on Windows

- `${WORKSPACE}` was substituted raw into `fs_sandbox = "${WORKSPACE}"`. A Windows path such as `C:\Users\…`
  then reads as an invalid TOML `\U` escape, so every run policy and coordinator agent policy failed to load
  on Windows (`TestMaterializeRunPolicy`, `test-windows` and `Build windows-latest` red since #1451). The
  path is now escaped for the TOML basic string at both sites (`MaterializeRunPolicy`,
  `MaterializeAgentPolicy`). The new test reproduces the bug on any OS.
