### Fixed — mission registry walk-up no longer hangs on Windows

`ailang mission` searched ancestors for `missions/` with a `d != "/"` guard. On Windows
`filepath.Dir(`C:\`)` is `C:\`, so the loop never ended (the #1580 `test-windows` job hung
to the 10-minute timeout). The walk now stops when `filepath.Dir` stops changing the path, on
every OS; the root itself is still not searched, as before on Unix.
