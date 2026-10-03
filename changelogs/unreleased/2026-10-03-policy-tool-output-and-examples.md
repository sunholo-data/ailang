### Fixed — `policy-tool` cut every CLI op at 64 KiB, so `builtins_list --json` never parsed (2026-10-03)

`policy-tool` capped each op's stdout and stderr at a hard-coded 64 KiB and appended `…[truncated]`;
`builtins list --json` is ~85 KB, so the JSON was always cut. The cap is now the policy's
`max_output_bytes` (stdout+stderr combined; the 8 MiB restricted default when the policy sets none).
A `--json` op over the cap is now a named refusal with no stdout, instead of a cut document. Text
output is still cut, and the marker names the limit and the original size. `ailang builtins list`
gains `--module <std/x>` and `--query <text>` filters, which `builtins_list` admits along with
`--by-module`, `--by-effect` and `--verbose`. Also, `examples_list` now forwards `--tags`, the flag
the CLI defines; it used to admit `--tag`, which the child rejected. (#1551)

### Fixed — the examples corpus is built into the binary; confined tools ignore cwd `examples/` (2026-10-03)

The release shipped only the binary, so `examples search` and `policy-tool examples_search` failed
on a clean machine or under a per-task `HOME`. `examples/embed.go` now embeds `manifest.json` and
`runnable/**` (the same pattern as `std/embed.go`: no copy to sync, nothing to drift; +468 KiB of
binary). The embedded corpus is the last fallback after `AILANG_EXAMPLES`, the paths next to the
binary, `~/.ailang/examples` and the cwd-relative paths. When `AILANG_AGENT_POLICY` is set (the
ailang_only lane, which `policy-tool` now pins on every CLI child), the cwd-relative `examples`,
`../examples` and `../../examples` are skipped, so a confined tool cannot pick up an examples dir an
agent planted. `examples show --run` refuses on the embedded corpus with a clear message, because
example module paths are repo-relative. (#1552)

### Fixed — `examples search` panicked when the corpus had no `manifest.json` (2026-10-03)

A corpus holding `runnable/*.ail` but no `manifest.json` caused a nil dereference (exit 2). Search,
list and tags now fail with `examples corpus at <dir> has no manifest.json`. `show` still displays
the file, without metadata. (#1553)
