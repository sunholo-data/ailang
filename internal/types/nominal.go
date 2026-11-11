package types

import (
	"fmt"
	"sort"
	"strings"
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
// for primitives and structural heads, its home module otherwise. A name in
// neither class gets BuiltinModule so that it is still a distinct identity
// (never a silent match with a user type of the same name); the completeness
// test makes that case a test failure.
func BuiltinCon(name string) *TCon {
	if IsModuleless(name) {
		return &TCon{Name: name}
	}
	if home, ok := builtinHomes[name]; ok {
		return &TCon{Name: name, Module: home}
	}
	return &TCon{Name: name, Module: BuiltinModule}
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
	return fmt.Sprintf("%s: cannot unify %s (defined in %s) with %s (defined in %s): "+
		"they are different types that share a name.\n"+
		"  Suggestion: use the %s from one module on both sides (import it by name from the module whose %s you mean, e.g. import %s (%s)), or convert between them explicitly.",
		TC_NOMINAL_001, a.Name, displayModule(a.Module), b.Name, displayModule(b.Module),
		a.Name, a.Name, displayModule(a.Module), a.Name)
}

func displayModule(m string) string {
	if m == "" {
		return "<no module>"
	}
	return m
}

// TypeRef names one nominal declaration: its defining module and its declared
// name (an `import M (T as U)` binds U to TypeRef{M, T}).
type TypeRef struct {
	Module string
	Name   string
}

// TypeScope answers "what does this bare type name mean here" for one module
// (§4.3 of m-v1-nominal-type-identity.md). Names holds every binding already
// ordered by precedence (local, explicit import, prelude, direct-dependency
// auto-import, transitive); Resolve falls back to the builtin home table and
// finally to this module (an unbound name is an opaque local type, as before —
// TC_UNBOUND_TYPE_001 is M3).
type TypeScope struct {
	Module string
	Names  map[string]TypeRef
}

// NewTypeScope returns an empty scope for module.
func NewTypeScope(module string) *TypeScope {
	return &TypeScope{Module: module, Names: make(map[string]TypeRef)}
}

// Bind records name -> ref unless name is already bound (first binding wins,
// so callers bind in precedence order).
func (s *TypeScope) Bind(name string, ref TypeRef) {
	if _, ok := s.Names[name]; !ok {
		s.Names[name] = ref
	}
}

// Resolve returns the declaration a bare type name denotes in this scope.
func (s *TypeScope) Resolve(name string) TypeRef {
	if IsModuleless(name) {
		return TypeRef{Name: name}
	}
	if s != nil {
		if r, ok := s.Names[name]; ok {
			return r
		}
	}
	if home, ok := builtinHomes[name]; ok {
		return TypeRef{Module: home, Name: name}
	}
	if s == nil {
		return TypeRef{Module: BuiltinModule, Name: name}
	}
	return TypeRef{Module: s.Module, Name: name}
}

// Con mints the TCon a bare type name denotes in this scope.
func (s *TypeScope) Con(name string) *TCon {
	r := s.Resolve(name)
	return &TCon{Name: r.Name, Module: r.Module}
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
