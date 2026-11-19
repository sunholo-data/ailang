# migrate-stdlib-renames.sed — rename the deprecated std spellings in .ail files.
#
#   sed -E -i.bak -f tools/migrate-stdlib-renames.sed path/to/file.ail   (BSD and GNU sed)
#
# M-V1-STDLIB-SURFACE-POLISH §11. Safe for the renames whose old name is a unique
# token (R1-R11, R13); each rule runs twice so back-to-back uses such as
# head_or(head_or(xs, d), d) are both rewritten. A builtin such as _rand_int is
# untouched (the leading `_` is a word character).
#
# NOT handled — edit by hand, then let `ailang check` confirm no TC_DEPRECATED_001
# is left:
#   glob(root, suffix)        -> walkWithSuffix(root, suffix)   ("glob" is a common word)
#   bytes slice(b, start, n)  -> sliceLen(b, start, n)          ("slice" is a common word)
#   list member(x, xs)        -> contains(xs, x)                (the arguments SWAP)
#   array unsafeGet(arr, i)   -> get(arr, i)                    (may collide with json.get:
#                                                                import `get as arrayGet`)
# TestMigrateSedCoversRenames (internal/pipeline) keeps this list equal to the
# @deprecated annotations in std/.
s/(^|[^A-Za-z0-9_])head_or([^A-Za-z0-9_]|$)/\1headOr\2/g
s/(^|[^A-Za-z0-9_])head_or([^A-Za-z0-9_]|$)/\1headOr\2/g
s/(^|[^A-Za-z0-9_])last_or([^A-Za-z0-9_]|$)/\1lastOr\2/g
s/(^|[^A-Za-z0-9_])last_or([^A-Za-z0-9_]|$)/\1lastOr\2/g
s/(^|[^A-Za-z0-9_])nth_or([^A-Za-z0-9_]|$)/\1nthOr\2/g
s/(^|[^A-Za-z0-9_])nth_or([^A-Za-z0-9_]|$)/\1nthOr\2/g
s/(^|[^A-Za-z0-9_])charAt_or([^A-Za-z0-9_]|$)/\1charAtOr\2/g
s/(^|[^A-Za-z0-9_])charAt_or([^A-Za-z0-9_]|$)/\1charAtOr\2/g
s/(^|[^A-Za-z0-9_])abs_Int([^A-Za-z0-9_]|$)/\1absInt\2/g
s/(^|[^A-Za-z0-9_])abs_Int([^A-Za-z0-9_]|$)/\1absInt\2/g
s/(^|[^A-Za-z0-9_])abs_Float([^A-Za-z0-9_]|$)/\1absFloat\2/g
s/(^|[^A-Za-z0-9_])abs_Float([^A-Za-z0-9_]|$)/\1absFloat\2/g
s/(^|[^A-Za-z0-9_])rand_int([^A-Za-z0-9_]|$)/\1randInt\2/g
s/(^|[^A-Za-z0-9_])rand_int([^A-Za-z0-9_]|$)/\1randInt\2/g
s/(^|[^A-Za-z0-9_])rand_float([^A-Za-z0-9_]|$)/\1randFloat\2/g
s/(^|[^A-Za-z0-9_])rand_float([^A-Za-z0-9_]|$)/\1randFloat\2/g
s/(^|[^A-Za-z0-9_])rand_bool([^A-Za-z0-9_]|$)/\1randBool\2/g
s/(^|[^A-Za-z0-9_])rand_bool([^A-Za-z0-9_]|$)/\1randBool\2/g
s/(^|[^A-Za-z0-9_])rand_seed([^A-Za-z0-9_]|$)/\1randSeed\2/g
s/(^|[^A-Za-z0-9_])rand_seed([^A-Za-z0-9_]|$)/\1randSeed\2/g
s/(^|[^A-Za-z0-9_])readFileBytes([^A-Za-z0-9_]|$)/\1readFileBase64\2/g
s/(^|[^A-Za-z0-9_])readFileBytes([^A-Za-z0-9_]|$)/\1readFileBase64\2/g
s/(^|[^A-Za-z0-9_])printErr([^A-Za-z0-9_]|$)/\1eprint\2/g
s/(^|[^A-Za-z0-9_])printErr([^A-Za-z0-9_]|$)/\1eprint\2/g
