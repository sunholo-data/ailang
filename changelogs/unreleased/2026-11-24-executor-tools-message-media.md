### Fixed — executor-mcp starts again on std/ai `Message.media`

v0.61.0 renamed `std/ai` `Message.images` to `Message.media` (#2458). The #2458 migration missed
`executor_tools/agent.ail`, so `executor-mcp` and `executor-mcp-oauth` exited on startup and no
longer rolled (dev build `7fc64cac`). `tools/motoko/r8_headroom_band.ail` was updated the same way.
`make verify-mcp-tools`, which CI runs, now also type-checks `executor_tools/`: both directories ship
in the executor-mcp image, and the gate never checked the second one.
