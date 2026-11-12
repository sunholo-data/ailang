package importhint

// StdlibRename is one std export renamed or deprecated before 1.0
// (M-V1-STDLIB-SURFACE-POLISH §4, rows R1-R14, D1, D2).
//
// The old names live on as @deprecated forwarders in std/ until they are removed
// before the v1.0.0 soak; this table outlives them, so a program written against
// 0.x still gets the replacement named after removal (the "removed-name table"
// of §6.3). The @deprecated annotations in std/ and this table must agree:
// TestStdlibRenamesMatchDeprecatedAnnotations (internal/pipeline) fails when
// either side moves without the other.
type StdlibRename struct {
	Module string // e.g. "std/list"
	Old    string // the 0.x spelling, e.g. "head_or"
	New    string // its replacement, e.g. "headOr"
	// Swapped is true when the replacement takes the arguments in a different
	// order (list.member(x, xs) -> contains(xs, x)); every other row is called
	// with exactly the same arguments.
	Swapped bool
	// Note is the @deprecated note, shown with the replacement.
	Note string
}

// StdlibRenames lists every deprecated std export and its replacement.
var StdlibRenames = []StdlibRename{
	{"std/list", "head_or", "headOr", false, ""},
	{"std/list", "last_or", "lastOr", false, ""},
	{"std/list", "nth_or", "nthOr", false, ""},
	{"std/string", "charAt_or", "charAtOr", false, ""},
	{"std/math", "abs_Int", "absInt", false, ""},
	{"std/math", "abs_Float", "absFloat", false, ""},
	{"std/rand", "rand_int", "randInt", false, ""},
	{"std/rand", "rand_float", "randFloat", false, ""},
	{"std/rand", "rand_bool", "randBool", false, ""},
	{"std/rand", "rand_seed", "randSeed", false, ""},
	{"std/fs", "readFileBytes", "readFileBase64", false, ""},
	{"std/fs", "glob", "walkWithSuffix", false, ""},
	{"std/io", "printErr", "eprint", false, ""},
	{"std/bytes", "slice", "sliceLen", false, "the third argument is a length, not an end index"},
	{"std/list", "member", "contains", true, "arguments swap: member(x, xs) becomes contains(xs, x)"},
	{"std/array", "unsafeGet", "get", false, "identical behaviour; if get collides with an imported json.get, import it as `get as arrayGet`"},
}

// Renamed returns the rename entry for an old std spelling, if there is one.
func Renamed(module, old string) (StdlibRename, bool) {
	for _, r := range StdlibRenames {
		if r.Module == module && r.Old == old {
			return r, true
		}
	}
	return StdlibRename{}, false
}

// RenameHint is the diagnostic suffix for an old std spelling that a module
// no longer exports (or "" when symbol is not a renamed std name).
func RenameHint(symbol, modID string) string {
	r, ok := Renamed(modID, symbol)
	if !ok {
		return ""
	}
	hint := " — '" + symbol + "' was renamed: use '" + r.New + "' from '" + modID + "'"
	if r.Note != "" {
		hint += " (" + r.Note + ")"
	}
	return hint
}
