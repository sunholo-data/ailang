### Changed — BREAKING: a type may not be named like a built-in (PAR_TYPE_DECL_BUILTIN) (2026-11-13)

`type float = float` used to pass and `type string = int` was silently ignored at every use
(the name always means the built-in). Declaring a type named `int`, `float`, `string`, `bool`,
`unit`, `char` or `bytes` is now the parse error `PAR_TYPE_DECL_BUILTIN`, the declaration
counterpart of `PAR_TYPE_PARAM_BUILTIN`. Measured first: 0 such declarations in the repo,
the banked AI-written solutions, ailang-packages, the motoko fork, the local downstream
checkouts and the package registry.

### Fixed — parse errors reach `check --json` with their own code (2026-11-13)

A module that does not parse reported every `PAR_*` error to `check --json` as
`"code": "ERROR"` with no position: the module loader flattened the parser's errors into one
string. It now returns `loader.ParseErrors`, which keeps the same text and unwraps to each
parser error, so `check --json` — for a single file, a directory (`check --json <dir>`) and
a package (`check --package --json`) — reports the FIRST parser error's code, line and
column. When that first error carries no code it is reported as it is (`"ERROR"`, no
position), never with a later error's code, so code, position and message describe one
error. `check --format agent` follows: a parse error now prints as `file:line:col CODE: …`
where it printed `file ERROR: …`.

### Changed — type-name suggestions are stricter and know other languages' names (2026-11-13)

The did-you-mean of `TC_UNBOUND_TYPE_003` (and now `TC_UNBOUND_TYPE_001`) maps names a
newcomer writes to the built-in they mean (`str`/`text` → `string`, `number`/`double` →
`float`, `integer`/`long` → `int`, `boolean` → `bool`), suggests the type spelled the same up
to case (`option[int]` → `Option`, `map[string, int]` → `Map`), and otherwise only a type
within edit distance 1 — 2 when both names have six letters or more (`strings` → `string`).
A shared prefix is not a reason: `interval`, `integral`, `charge`, `bytesize`, `boolish`,
`stream` get no suggestion, and `type X = point` no longer suggests `int`. With nothing close,
there is no suggestion. `PAR_TYPE_DECL_BUILTIN`'s fix names an example that fits the built-in
(`type Name = string`, `type Meters = float`, …).

### Fixed — #2288's record-row guards test what they claim again (2026-11-13)

`TestRecordRows_EncodeFrameRequiresEveryField` and `TestRecordRows_BumpVersionRejectsStringVer`
used std/sem, whose parameters became the closed declared `sem_frame` with #2211, so they no
longer exercised an open-row signature across the interface. They now use a non-std module
exporting open-row functions, and fail when the open~closed row binding is turned off or
defaulting uses the shared free-variable walk, respectively.
