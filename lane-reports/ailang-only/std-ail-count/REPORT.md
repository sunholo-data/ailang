# std/ .ail file counts

Counts computed with `count.ail` (uses `std/fs` `listDir`, `walk`, `isDir`; run via `ailang_run`).

| directory     | .ail files |
|---------------|-----------:|
| std/          |         48 |
| std/ai/       |          1 |
| std/stream/   |          1 |
| **total**     |     **50** |

Note: `std/` holds 48 `.ail` files at its top level; the only immediate
subdirectories are `ai/` and `stream/`, each containing a single `.ail` file
(recursive count equals direct count for both since they have no further
subdirectories).
