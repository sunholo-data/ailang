### Added — `policy-tool` `read` takes an optional line window (`offset`/`limit`, + `total_lines`) (2026-10-17)

`ailang policy-tool`'s `read` op (pi's `ailang_read`, AILANG World's `ailang-read`) accepts optional
integer `offset` (1-based first line) and `limit` (line count, capped at 2000). A ranged read answers
with `total_lines` and the window actually served (`limit` = lines returned). Lines keep their own
endings, so consecutive windows reassemble the file byte for byte. An `offset` past the end returns
empty content, not an error. Only the window is held in memory, and it is cut at a whole line under
`max_fs_transfer_bytes`, so a file too large to read whole can still be paged. A single line over the
cap is refused by number. Without either field, `read` is unchanged. Sandbox confinement is identical
to a whole read, and directories refuse a window. The pi adapter exposes the two fields.

Why: on 2026-10-16 a World docs episode (`wdocs`) made 73 `ailang-run` calls. Most of them were a
throwaway `zz_peek.ail` line-reader the agent wrote to page large files (#1872).
