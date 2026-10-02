### Added — `std/fs.readFileRaw` and `std/fs.writeFileBytesResult`

- `readFileRaw(path) -> Result[bytes, string] ! {FS}` returns a file's bytes unchanged.
  `readFileBytes` is named like a bytes API, but it returns base64 text in a `Result[string, string]`
  (`"hi"` on disk comes back as `Ok("aGk=")`). Its std/fs doc line now says so. Reported by
  stapledons_godot.
- `writeFileBytesResult(path, data) -> Result[(), string] ! {FS}` is the Result form of `writeFileBytes`.
  `writeFileBytes` panics when a parent directory is missing, the path is protected, or the sandbox
  refuses it. The new function returns `Err` instead, like `writeFileResult`, `renameFileResult` and
  `removeFileResult`.
- Both go through the same FS capability check, `AILANG_FS_SANDBOX` root, read size cap, transfer
  limits and `fs_deny_write` protection as the other FS ops, and both have rows in the FS containment
  matrix. Under `--bytecode` they reach the evaluator through the bridge, as every FS builtin does.
  Under `--strict-bytecode` they are refused, also like every other FS builtin: effectful builtins are
  not yet native in the VM.
- New example: `examples/runnable/fs_bytes_roundtrip.ail`. The std/fs interface golden has been
  re-frozen.
