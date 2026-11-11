package types

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// M-V1-NOMINAL-TYPE-IDENTITY (M2): a nominal type is its defining module plus
// its name. TCon.Module carries the module; TCon.Equals, SafeEquals, equalHead
// and the unifier compare both, through NominalSame. There is NO wildcard: an
// empty Module matches only another empty Module, and only module-less names
// (primitives and the builtin structural heads) are meant to have one.

// BuiltinModule is the home of nominal names that builtin signatures use but no
// .ail module declares (DOMEvent). It is a real, distinct module identity: a
// user type named DOMEvent is not the builtin one.
const BuiltinModule = "$builtin"

// TC_NOMINAL_001 codes a mismatch between two different nominal types that
// print the same, i.e. same name, different defining modules.
const TC_NOMINAL_001 = "TC_NOMINAL_001"

// NominalSame reports whether two type constructors denote the same type:
// same name AND same defining module.
func NominalSame(a, b *TCon) bool {
	return a.Name == b.Name && a.Module == b.Module
}

// NominalKey is the map key of a nominal name: "module::name", or the bare
// name for a module-less TCon. Use it for every table keyed by type identity
// (alias env, newtype set, ADT arity).
func NominalKey(module, name string) string {
	if module == "" {
		return name
	}
	return module + "::" + name
}

// KeyOf returns NominalKey for a TCon.
func (t *TCon) KeyOf() string { return NominalKey(t.Module, t.Name) }

// SplitNominalKey splits a NominalKey back into (module, name).
func SplitNominalKey(key string) (module, name string) {
	if i := strings.LastIndex(key, "::"); i >= 0 {
		return key[:i], key[i+2:]
	}
	return "", key
}

// moduleless lists the type constructor names that legitimately carry no
// defining module: primitives (and their capitalised spellings, which some
// internal paths mint), unit, and the builtin structural heads that the parser
// and unifier map onto TList/TArray/TMap.
var moduleless = map[string]bool{
	"int": true, "float": true, "string": true, "bool": true, "unit": true, "()": true,
	"bytes": true, "char": true,
	"Int": true, "Float": true, "String": true, "Bool": true, "Unit": true, "Bytes": true,
	"list": true, "List": true, "Array": true, "Map": true,
}

// IsModuleless reports whether name is a primitive or builtin structural head,
// i.e. a TCon that is correct with Module == "".
func IsModuleless(name string) bool { return moduleless[name] }

// builtinHomes maps every nominal name a builtin signature (types.Builder.Con /
// App) uses to the module that declares it (D4). A name no .ail module
// declares gets BuiltinModule. TestBuiltinHomes_Complete walks every builtin
// signature and fails on a nominal TCon outside this table, and checks that each
// std home really declares the type.
var builtinHomes = map[string]string{
	"Option":            "std/option",
	"Result":            "std/result",
	"Json":              "std/json",
	"XmlNode":           "std/xml",
	"NodeKind":          "std/xml",
	"StreamConn":        "std/stream",
	"StreamErrorKind":   "std/stream",
	"StreamSource":      "std/stream",
	"StreamEvent":       "std/stream",
	"StreamStatus":      "std/stream",
	"NetError":          "std/net",
	"ProcessHandle":     "std/process",
	"ProcessError":      "std/process",
	"WorkerCancelError": "std/process",
	"EnvError":          "std/env",
	"FoldStep":          "std/iter",
	"DOMPatch":          "std/dom",
	"TerminalSession":   "std/terminal",
	"TerminalInfo":      "std/terminal",
	"TerminalEvent":     "std/terminal",
	"TerminalError":     "std/terminal",
	"StreamChunk":       "std/ai",
	"Verdict":           "std/stream/bridge",
	"BridgeFrame":       "std/stream/bridge",
	"BridgeEnd":         "std/stream/bridge",
	"DOMEvent":          BuiltinModule,
}

// BuiltinHome returns the defining module of a nominal name used by builtin
// signatures, and whether the name is in the table.
func BuiltinHome(name string) (string, bool) {
	m, ok := builtinHomes[name]
	return m, ok
}

// BuiltinHomeNames returns the names in the builtin home table, sorted.
func BuiltinHomeNames() []string {
	out := make([]string, 0, len(builtinHomes))
	for n := range builtinHomes {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// BuiltinCon mints the TCon for a name used in a builtin signature: module-less
// for primitives and structural heads, its home module otherwise. A nominal
// name missing from the home table is minted WITHOUT a module — there is no
// fallback home — so builtins.RegisterEffectBuiltin rejects the signature at
// registration (MissingBuiltinHomes) and TestBuiltinHomes_Complete fails.
func BuiltinCon(name string) *TCon {
	if IsModuleless(name) {
		return &TCon{Name: name}
	}
	if home, ok := builtinHomes[name]; ok {
		return &TCon{Name: name, Module: home}
	}
	return &TCon{Name: name}
}

// MissingBuiltinHomes lists, sorted, the nominal type names in t that carry
// no defining module: in a builtin signature, a name the home table lacks. It
// walks every type form, including both record representations, rows (record
// and effect) and their budgets' carriers, function effect rows and labels.
func MissingBuiltinHomes(t Type) []string {
	set := map[string]bool{}
	var walk func(Type, int)
	walkRow := func(r *Row, depth int) {
		if r == nil {
			return
		}
		for _, l := range r.Labels {
			walk(l, depth+1)
		}
		if r.Tail != nil {
			walk(r.Tail, depth+1)
		}
	}
	walk = func(t Type, depth int) {
		if t == nil || depth > 64 {
			return
		}
		switch x := t.(type) {
		case *TCon:
			if UnqualifiedNominal(x) {
				set[x.Name] = true
			}
		case *TApp:
			walk(x.Constructor, depth+1)
			for _, a := range x.Args {
				walk(a, depth+1)
			}
		case *TFunc2:
			for _, p := range x.Params {
				walk(p, depth+1)
			}
			walk(x.Return, depth+1)
			walkRow(x.EffectRow, depth)
		case *TList:
			walk(x.Element, depth+1)
		case *TArray:
			walk(x.Element, depth+1)
		case *TMap:
			walk(x.Key, depth+1)
			walk(x.Value, depth+1)
		case *TTuple:
			for _, e := range x.Elements {
				walk(e, depth+1)
			}
		case *TRecord:
			for _, f := range x.Fields {
				walk(f, depth+1)
			}
			walk(x.Row, depth+1)
		case *TRecordOpen:
			for _, f := range x.Fields {
				walk(f, depth+1)
			}
			walk(x.Row, depth+1)
		case *TRecord2:
			walkRow(x.Row, depth)
		case *Row:
			walkRow(x, depth)
		case *TLabelled:
			walk(x.Inner, depth+1)
		case *TVar, *TVar2, *RowVar:
			// no constructor inside
		default:
			// A type form this walk does not know must not pass silently.
			set[fmt.Sprintf("<unwalked %T>", t)] = true
		}
	}
	walk(t, 0)
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// IsNominalHome reports whether the TCon is the nominal type `name` declared in
// `module` (e.g. IsNominal(t, "std/option", "Option")).
func IsNominal(t Type, module, name string) bool {
	c, ok := t.(*TCon)
	return ok && c.Name == name && c.Module == module
}

// UnqualifiedNominal reports whether t is a TCon that names a nominal type but
// carries no defining module: the signature of a minting site M2 missed.
func UnqualifiedNominal(t *TCon) bool {
	return t != nil && t.Module == "" && !IsModuleless(t.Name) && t.Name != "" &&
		!strings.HasPrefix(t.Name, "$")
}

// nominalMismatchMsg renders TC_NOMINAL_001 for two TCons with the same name
// and different defining modules. Unification does not know which side is the
// expected one, so the text names both without claiming a direction.
func nominalMismatchMsg(a, b *TCon) string {
	// The suggestion names a module that declares the type; a built-in has no
	// module to import from, so the other side is named instead.
	from := a
	if from.Module == "" || from.Module == BuiltinModule {
		from = b
	}
	hint := fmt.Sprintf("use the %s from one module on both sides (import it by name from the module whose %s you mean, e.g. import %s (%s)), or convert between them explicitly.",
		a.Name, a.Name, from.Module, a.Name)
	if from.Module == "" || from.Module == BuiltinModule {
		hint = fmt.Sprintf("use one %s on both sides, or convert between them explicitly.", a.Name)
	}
	return fmt.Sprintf("%s: cannot unify %s with %s: they are different types that share a name.\n  Suggestion: %s",
		TC_NOMINAL_001, describeTCon(a), describeTCon(b), hint)
}

// describeTCon names a type constructor for a diagnostic: "T (defined in q/a)",
// or "the built-in Map" for a module-less or $builtin one.
func describeTCon(t *TCon) string {
	if t.Module == "" && t.Name != "" && unicode.IsLower(rune(t.Name[0])) {
		return t.Name // int, string, ...: a primitive needs no qualifier
	}
	if t.Module == "" || t.Module == BuiltinModule {
		return "the built-in " + t.Name
	}
	return fmt.Sprintf("%s (defined in %s)", t.Name, t.Module)
}

func displayModule(m string) string {
	if m == "" {
		return "the built-in"
	}
	return m
}

// unqualifiedNominalPanic is the test-only hook of §4.1: when set, any nominal
// TCon without a defining module that reaches unification panics, naming it.
// A minting site M2 missed therefore fails a test instead of quietly forming
// an identity class of its own. Production builds keep it off: there a
// qualified/unqualified pair of one name is reported as an internal compiler
// error by checkNominalPair, and two unqualified TCons compare as equal only
// to each other ("" is not a wildcard).
var unqualifiedNominalPanic bool

// SetUnqualifiedNominalPanic turns the §4.1 test hook on or off and returns the
// previous setting. Tests only.
func SetUnqualifiedNominalPanic(on bool) bool {
	prev := unqualifiedNominalPanic
	unqualifiedNominalPanic = on
	return prev
}

// noteNominal runs the test hook on a type about to be unified.
func noteNominal(t Type) {
	if !unqualifiedNominalPanic {
		return
	}
	if app, ok := t.(*TApp); ok {
		t, _ = decomposeApp(app)
	}
	if c, ok := t.(*TCon); ok && UnqualifiedNominal(c) {
		panic(fmt.Sprintf("M-V1-NOMINAL-TYPE-IDENTITY: nominal type %q reached unification with no defining module (a minting site was not qualified)", c.Name))
	}
}

// checkNominalPair reports the internal compiler error for a same-named pair
// where exactly one side carries a module: that is never a user error, it is a
// minting site that forgot to stamp the type.
func checkNominalPair(a, b *TCon) error {
	if a.Name != b.Name || (a.Module == "") == (b.Module == "") || IsModuleless(a.Name) {
		return nil
	}
	return fmt.Errorf("internal compiler error (M-V1-NOMINAL-TYPE-IDENTITY): type %s was minted without its defining module "+
		"on one side of a comparison (other side: %s). Please report this with the program that triggered it", a.Name, displayModule(a.Module+b.Module))
}

// NominalIdentityCodes lists the diagnostic codes M-V1-NOMINAL-TYPE-IDENTITY
// introduced that are reported inline in an error message (MOD015 is a
// structured report). `check --json` reports each as itself.
func NominalIdentityCodes() []string {
	return []string{TC_NOMINAL_001, TC_AMBIGUOUS_TYPE_001, TC_AMBIGUOUS_CTOR_001}
}
