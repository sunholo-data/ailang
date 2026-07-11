package builtins

import (
	"fmt"
	"regexp"
	"sync"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Regex builtins — a thin wrapper over Go's regexp package (the RE2 engine).
//
// RE2 guarantees matching runs in time linear in len(input) × len(pattern) with
// NO backtracking — the classic catastrophic pattern (a+)+$ against a long
// non-matching input returns in microseconds, not exponential time. The price is
// that RE2 (and therefore this module) does NOT support backreferences or
// lookaround; compile() surfaces those as Err, never a panic.
//
// The public AILANG surface is std/regex.ail. The wrappers there hold an opaque
// Regex(string) newtype and unwrap the pattern string before calling these
// builtins, so every builtin below takes the pattern as a plain string.
//
// INDEX CONVENTION (important): Go's regexp returns BYTE offsets, but AILANG
// string indexing (_str_slice / _str_len) is RUNE-indexed / UTF-8 aware. So the
// integer spans we expose in RegexMatch (start/end and each group's start/end)
// are converted to RUNE indices to stay consistent with std/string. The matched
// text is sliced using the raw byte offsets (correct regardless of encoding).

func init() {
	registerRegexBuiltins()
}

// compiledEntry memoizes a single regexp.Compile result (either the compiled
// program or the compile error) keyed by the pattern string.
type compiledEntry struct {
	re  *regexp.Regexp
	err error
}

var (
	regexCacheMu sync.Mutex
	regexCache   = make(map[string]compiledEntry)
)

// getCompiled returns the memoized *regexp.Regexp for a pattern, compiling and
// caching on first use. Pure memoization keyed by the pattern string — invisible
// to AILANG semantics and deterministic (same pattern → same result forever).
func getCompiled(pattern string) (*regexp.Regexp, error) {
	regexCacheMu.Lock()
	defer regexCacheMu.Unlock()
	if e, ok := regexCache[pattern]; ok {
		return e.re, e.err
	}
	re, err := regexp.Compile(pattern)
	regexCache[pattern] = compiledEntry{re: re, err: err}
	return re, err
}

func registerRegexBuiltins() {
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_compile", NumArgs: 1, IsPure: true,
		Type: regexCompileType, Impl: regexCompileImpl,
		Metadata: &BuiltinMetadata{
			Description: "Validate + memoize a regex pattern; Ok(pattern) or Err(message)",
			Returns:     "Result[string, string] — Ok(pattern) if valid, Err(message) for invalid/unsupported (backref/lookaround) patterns",
			Since:       "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_is_match", NumArgs: 2, IsPure: true,
		Type: regexIsMatchType, Impl: regexIsMatchImpl,
		Metadata: &BuiltinMetadata{
			Description: "Report whether the pattern matches anywhere in the subject",
			Returns:     "bool", Since: "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_find_first", NumArgs: 2, IsPure: true,
		Type: regexFindFirstType, Impl: regexFindFirstImpl,
		Metadata: &BuiltinMetadata{
			Description: "First match with capture groups, or None",
			Returns:     "Option[RegexMatch]", Since: "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_find_all", NumArgs: 2, IsPure: true,
		Type: regexFindAllType, Impl: regexFindAllImpl,
		Metadata: &BuiltinMetadata{
			Description: "All non-overlapping matches with capture groups",
			Returns:     "[RegexMatch]", Since: "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_replace_all", NumArgs: 3, IsPure: true,
		Type: regexReplaceAllType, Impl: regexReplaceAllImpl,
		Metadata: &BuiltinMetadata{
			Description: "Replace all matches; repl may use $1 / ${name} (Go syntax)",
			Returns:     "string", Since: "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
	mustRegisterRegex(BuiltinSpec{
		Module: "std/regex", Name: "_regex_split", NumArgs: 2, IsPure: true,
		Type: regexSplitType, Impl: regexSplitImpl,
		Metadata: &BuiltinMetadata{
			Description: "Split the subject around every match of the pattern",
			Returns:     "[string]", Since: "v0.30.0", Stability: StabilityExperimental, Category: "regex",
		},
	})
}

func mustRegisterRegex(spec BuiltinSpec) {
	if err := RegisterEffectBuiltin(spec); err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", spec.Name, err))
	}
}

// Type signatures

func regexCompileType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.App("Result", T.String(), T.String())).Build()
}

func regexIsMatchType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func regexFindFirstType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.App("Option", regexMatchRecordType(T))).Build()
}

func regexFindAllType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.List(regexMatchRecordType(T))).Build()
}

func regexReplaceAllType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String(), T.String()).Returns(T.String()).Build()
}

func regexSplitType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.List(T.String())).Build()
}

// regexMatchRecordType is the RegexMatch shape:
//
//	{ start: int, end: int, groups: [{start: int, end: int, text: string}], text: string }
func regexMatchRecordType(T *types.Builder) types.Type {
	group := T.Record(
		types.Field("start", T.Int()),
		types.Field("end", T.Int()),
		types.Field("text", T.String()),
	)
	return T.Record(
		types.Field("start", T.Int()),
		types.Field("end", T.Int()),
		types.Field("groups", T.List(group)),
		types.Field("text", T.String()),
	)
}

// Implementations

func regexCompileImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pattern, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_regex_compile: arg 0 - %w", err)
	}
	if _, cerr := getCompiled(pattern); cerr != nil {
		// Invalid or unsupported (backref/lookaround) pattern → structured Err, never a panic.
		return wrapErr(cerr.Error()), nil
	}
	return wrapOk(&eval.StringValue{Value: pattern}), nil
}

func regexIsMatchImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	re, s, err := regexArgs(args, "_regex_is_match")
	if err != nil {
		return nil, err
	}
	return &eval.BoolValue{Value: re.MatchString(s)}, nil
}

func regexFindFirstImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	re, s, err := regexArgs(args, "_regex_find_first")
	if err != nil {
		return nil, err
	}
	loc := re.FindStringSubmatchIndex(s)
	if loc == nil {
		return &eval.TaggedValue{ModulePath: "std/option", TypeName: "Option", CtorName: "None"}, nil
	}
	return &eval.TaggedValue{
		ModulePath: "std/option", TypeName: "Option", CtorName: "Some",
		Fields: []eval.Value{buildMatchRecord(s, loc)},
	}, nil
}

func regexFindAllImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	re, s, err := regexArgs(args, "_regex_find_all")
	if err != nil {
		return nil, err
	}
	all := re.FindAllStringSubmatchIndex(s, -1)
	elems := make([]eval.Value, 0, len(all))
	for _, loc := range all {
		elems = append(elems, buildMatchRecord(s, loc))
	}
	return &eval.ListValue{Elements: elems}, nil
}

func regexReplaceAllImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	re, s, err := regexArgs(args, "_regex_replace_all")
	if err != nil {
		return nil, err
	}
	repl, rerr := SafeAsString(args[2])
	if rerr != nil {
		return nil, fmt.Errorf("_regex_replace_all: arg 2 - %w", rerr)
	}
	return &eval.StringValue{Value: re.ReplaceAllString(s, repl)}, nil
}

func regexSplitImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	re, s, err := regexArgs(args, "_regex_split")
	if err != nil {
		return nil, err
	}
	parts := re.Split(s, -1)
	elems := make([]eval.Value, len(parts))
	for i, p := range parts {
		elems[i] = &eval.StringValue{Value: p}
	}
	return &eval.ListValue{Elements: elems}, nil
}

// regexArgs extracts (pattern, subject) from the first two args and returns the
// compiled regexp for the pattern. The std/regex wrappers only ever pass a
// pattern that already compiled cleanly (compile() validated it), so a compile
// error here means the builtin was called out of band — we fail loudly rather
// than silently returning a no-match (CLAUDE.md CP2: no silent fallbacks).
func regexArgs(args []eval.Value, name string) (*regexp.Regexp, string, error) {
	pattern, err := SafeAsString(args[0])
	if err != nil {
		return nil, "", fmt.Errorf("%s: arg 0 - %w", name, err)
	}
	s, err := SafeAsString(args[1])
	if err != nil {
		return nil, "", fmt.Errorf("%s: arg 1 - %w", name, err)
	}
	re, cerr := getCompiled(pattern)
	if cerr != nil {
		return nil, "", fmt.Errorf("%s: invalid pattern %q (compile with regex.compile first): %v", name, pattern, cerr)
	}
	return re, s, nil
}

// buildMatchRecord turns a FindStringSubmatchIndex result (byte offsets:
// [wholeStart, wholeEnd, g1Start, g1End, ...]) into the RegexMatch record.
// groups[0] is the whole match; groups[i] is capture group i. Non-participating
// optional groups report start = end = -1 and text = "". Integer spans are
// converted to RUNE indices (consistent with std/string); text is sliced from
// the raw byte offsets.
func buildMatchRecord(s string, loc []int) *eval.RecordValue {
	groupElems := make([]eval.Value, 0, len(loc)/2)
	for i := 0; i+1 < len(loc); i += 2 {
		bs, be := loc[i], loc[i+1]
		var text string
		if bs >= 0 && be >= 0 {
			text = s[bs:be]
		}
		groupElems = append(groupElems, &eval.RecordValue{Fields: map[string]eval.Value{
			"start": &eval.IntValue{Value: byteToRune(s, bs)},
			"end":   &eval.IntValue{Value: byteToRune(s, be)},
			"text":  &eval.StringValue{Value: text},
		}})
	}
	return &eval.RecordValue{Fields: map[string]eval.Value{
		"start":  &eval.IntValue{Value: byteToRune(s, loc[0])},
		"end":    &eval.IntValue{Value: byteToRune(s, loc[1])},
		"groups": &eval.ListValue{Elements: groupElems},
		"text":   &eval.StringValue{Value: s[loc[0]:loc[1]]},
	}}
}

// byteToRune converts a byte offset into s to a rune index. A byte offset of -1
// (non-participating optional group) is passed through as -1.
func byteToRune(s string, byteOff int) int {
	if byteOff < 0 {
		return -1
	}
	if byteOff > len(s) {
		byteOff = len(s)
	}
	return utf8.RuneCountInString(s[:byteOff])
}
