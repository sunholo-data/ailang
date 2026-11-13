package types

import (
	"fmt"
	"sort"
	"strings"
)

// TypeRef names one nominal declaration: its defining module and its declared
// name (an `import M (T as U)` binds U to TypeRef{M, T}).
type TypeRef struct {
	Module string
	Name   string
}

// TC_AMBIGUOUS_TYPE_001 codes a bare type name that two or more modules reach
// through the auto-import channels and that nothing more specific (a local
// declaration, an explicit import, a built-in or the prelude) decides.
const TC_AMBIGUOUS_TYPE_001 = "TC_AMBIGUOUS_TYPE_001"

// TypeScope answers "what does this bare type name mean here" for one module
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3). This is THE precedence order; every type
// name the elaborator, the interface builder and the REPL/WASM registry loader
// read goes through Lookup. (Type variables never get here: the parser decides
// by scope which names are variables — a name an enclosing `[T]` list binds,
// or an undeclared lowercase name — #2211, #2240. A lowercase name that does
// get here bare was declared in the module or imported by name; one that
// resolves to nothing — an imported value — is read as a type variable, as
// before #2211, with warning TC_UNBOUND_TYPE_002.)
//
//  1. a type this module declares (BindLocal);
//  2. a type bound by an explicit selective import the USER wrote,
//     `import M (T)` or `import M (T as U)` (BindExplicit; two different ones
//     are MOD015 at the import, so there is at most one). The prelude the
//     loader injects into an entry module is NOT this step but step 4;
//  3. a built-in, module-less name: Int, String, Map, List, Array, Bytes, Unit
//     and the other names of IsModuleless;
//  4. the prelude: Option -> std/option, Result -> std/result;
//  5. the ONE type of that name the modules this module imports export, through
//     any import form (AddDirect); two or more distinct ones are
//     TC_AMBIGUOUS_TYPE_001;
//  6. only when no imported module exports the name: the ONE type of that name
//     declared by a module this module reaches only indirectly, through its
//     imports (AddClosure); two or more are TC_AMBIGUOUS_TYPE_001;
//  7. the builtin home table (Json -> std/json, ... DOMEvent -> $builtin);
//  8. otherwise this module: an undeclared uppercase name stays an opaque type
//     of the module that writes it (TC_UNBOUND_TYPE_001 for it is #1485).
//
// No step depends on import order, file order or module path: steps 1–2 hold
// one binding per name, and steps 5–6 each have exactly one candidate or fail
// naming all of them (sorted by module).
//
// The boundary of step 6 (design doc §9, "Indirect names"): a module of the
// import closure can never change what a name means when an imported module
// exports that name — adding `type T` to an indirect dependency cannot break,
// or re-point, a module that uses T from its own import. It CAN make a name
// that this module only reaches indirectly ambiguous (a second indirect module
// adds it), and a new import that exports the name re-points a name that was
// resolved indirectly (as on dev). Both are fixed by importing the type by
// name, which always wins (step 2).
type TypeScope struct {
	Module   string
	local    map[string]TypeRef
	explicit map[string]TypeRef
	direct   map[string][]TypeRef
	closure  map[string][]TypeRef
}

// NewTypeScope returns an empty scope for module.
func NewTypeScope(module string) *TypeScope {
	return &TypeScope{
		Module:   module,
		local:    make(map[string]TypeRef),
		explicit: make(map[string]TypeRef),
		direct:   make(map[string][]TypeRef),
		closure:  make(map[string][]TypeRef),
	}
}

// BindLocal records a type this module declares (step 1).
func (s *TypeScope) BindLocal(name string) {
	s.local[name] = TypeRef{Module: s.Module, Name: name}
}

// BindExplicit records a type an explicit selective import binds (step 2).
func (s *TypeScope) BindExplicit(name string, ref TypeRef) { s.explicit[name] = ref }

// AddDirect records a type a module this module imports exports (step 5).
func (s *TypeScope) AddDirect(ref TypeRef) {
	s.direct[ref.Name] = addCandidate(s.direct[ref.Name], ref)
}

// AddClosure records a type a module this module reaches only indirectly
// declares (step 6).
func (s *TypeScope) AddClosure(ref TypeRef) {
	s.closure[ref.Name] = addCandidate(s.closure[ref.Name], ref)
}

func addCandidate(list []TypeRef, ref TypeRef) []TypeRef {
	for _, r := range list {
		if r == ref {
			return list
		}
	}
	list = append(list, ref)
	sort.Slice(list, func(i, j int) bool { return list[i].Module < list[j].Module })
	return list
}

// AmbiguousTypeError is TC_AMBIGUOUS_TYPE_001: a bare name with more than one
// candidate at the first step that has any.
type AmbiguousTypeError struct {
	Name       string
	Candidates []TypeRef // sorted by module
	Transitive bool      // true: reached only through the import closure
}

func (e *AmbiguousTypeError) Error() string {
	mods := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		mods[i] = c.Module
	}
	how := "the modules this one imports export"
	if e.Transitive {
		// Step 6: every candidate is an INDIRECT dependency (none is imported
		// here, and no imported module exports the name).
		how = "modules this one does not import, but reaches indirectly through its imports, declare"
		for i := range mods {
			mods[i] += " (indirect)"
		}
	}
	alias := e.Name
	if len(e.Candidates) > 1 {
		second := lastSeg(e.Candidates[1].Module)
		alias = e.Name + " as " + strings.ToUpper(second[:1]) + second[1:] + e.Name
	}
	return fmt.Sprintf("%s: type '%s' is ambiguous here: %s more than one type named %s (%s), and none is declared here or imported by name.\n"+
		"  Suggestion: import the one you mean by name, e.g. import %s (%s), or import %s (%s).",
		TC_AMBIGUOUS_TYPE_001, e.Name, how, e.Name, strings.Join(mods, ", "), e.Candidates[0].Module, e.Name, e.Candidates[1].Module, alias)
}

func lastSeg(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	var b strings.Builder
	for _, r := range p {
		if r == '_' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 || !('a' <= b.String()[0] && b.String()[0] <= 'z' || 'A' <= b.String()[0] && b.String()[0] <= 'Z') {
		return "Other"
	}
	return b.String()
}

// Lookup returns the declaration a bare type name denotes, or
// TC_AMBIGUOUS_TYPE_001 when the first step with candidates has several.
func (s *TypeScope) Lookup(name string) (TypeRef, error) {
	if s != nil {
		if r, ok := s.local[name]; ok {
			return r, nil
		}
		if r, ok := s.explicit[name]; ok {
			return r, nil
		}
	}
	if IsModuleless(name) {
		return TypeRef{Name: name}, nil
	}
	if home, ok := preludeTypes[name]; ok {
		return TypeRef{Module: home, Name: name}, nil
	}
	if s != nil {
		for i, tier := range []map[string][]TypeRef{s.direct, s.closure} {
			indirect := i == 1
			switch cands := tier[name]; len(cands) {
			case 0:
				continue
			case 1:
				if indirect {
					ObserveAmbiguity(AmbiguityEvent{Kind: "type-closure-only", Module: s.Module, Name: name, Candidates: cands})
				}
				return cands[0], nil
			default:
				observeAmbiguity(s.Module, name, cands, indirect)
				// No candidate is picked: the name denotes none of them.
				ambiguous := TypeRef{Module: AmbiguousModule, Name: name}
				if nominalAmbiguityIsError {
					return ambiguous, &AmbiguousTypeError{Name: name, Candidates: cands, Transitive: indirect}
				}
				return ambiguous, nil
			}
		}
	}
	if home, ok := builtinHomes[name]; ok {
		return TypeRef{Module: home, Name: name}, nil
	}
	if s == nil {
		return TypeRef{Module: BuiltinModule, Name: name}, nil
	}
	return TypeRef{Module: s.Module, Name: name}, nil
}

// Declares reports whether a bare type name denotes a declaration (any of
// steps 1–7) rather than step 8's opaque type of this module. It has no side
// effects: no ambiguity is observed or reported.
func (s *TypeScope) Declares(name string) bool {
	if s != nil {
		if _, ok := s.local[name]; ok {
			return true
		}
		if _, ok := s.explicit[name]; ok {
			return true
		}
		if len(s.direct[name]) > 0 || len(s.closure[name]) > 0 {
			return true
		}
	}
	_, prelude := preludeTypes[name]
	_, home := builtinHomes[name]
	return IsModuleless(name) || prelude || home
}

// ReadsAsTypeVar is THE decision for a bare name the parser left as an
// ast.SimpleType (#2211): true when it is nevertheless an implicitly
// quantified type variable. That happens for one case only — a lowercase name
// the file imports by name (`import std/list (map)`) that is a value, not a
// type in this scope. The parser cannot tell (it has no interfaces), so it
// builds a SimpleType and records the use in ast.File.LowerTypeNames with
// Kind LowerImported; the elaborator (elaborate.simpleType) and the interface
// builder both ask this method.
//
// Contract for later passes that collect a signature's type variables (e.g.
// rigid signature variables, #2273): the variables of an annotation are its
// ast.TypeVar nodes PLUS each ast.SimpleType for which ReadsAsTypeVar is true
// in the module's installed scope. ast.File.LowerTypeNames lists the
// candidates (Kind LowerImported) with their positions and declarations.
func (s *TypeScope) ReadsAsTypeVar(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' && !moduleless[name] && !s.Declares(name)
}

// Names lists every bare type name this scope resolves (steps 1–7), sorted,
// for did-you-mean suggestions.
func (s *TypeScope) Names() []string {
	set := map[string]bool{}
	if s != nil {
		for _, m := range []map[string]TypeRef{s.local, s.explicit} {
			for n := range m {
				set[n] = true
			}
		}
		for _, m := range []map[string][]TypeRef{s.direct, s.closure} {
			for n := range m {
				set[n] = true
			}
		}
	}
	for n := range moduleless {
		set[n] = true
	}
	for n := range preludeTypes {
		set[n] = true
	}
	for n := range builtinHomes {
		set[n] = true
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// TC_UNBOUND_TYPE_003 codes a lowercase name in the body of a type
// declaration that is neither one of the declaration's own parameters nor a
// type in scope (#2211 follow-up). A function signature quantifies such a name
// implicitly; a type declaration cannot (no caller supplies it), so the alias
// or ADT would silently stand for "any type".
const TC_UNBOUND_TYPE_003 = "TC_UNBOUND_TYPE_003"

// TC_TYPE_ALIAS_CYCLE_001 codes a type alias that refers to itself, directly
// or through other aliases, with no ADT or record declaration on the cycle: an
// alias stands for its body, so the cycle has no finite meaning.
const TC_TYPE_ALIAS_CYCLE_001 = "TC_TYPE_ALIAS_CYCLE_001"

// TC_UNBOUND_TYPE_001 codes a name in type position that is neither a type
// variable nor a type in scope. Today it is reported only for a lowercase name
// applied to type arguments (#2211), which has no type-variable reading; an
// uppercase undeclared name is still step 8's opaque type (#1485).
const TC_UNBOUND_TYPE_001 = "TC_UNBOUND_TYPE_001"

// UnboundLowerTypeError is TC_UNBOUND_TYPE_001 for a lowercase name applied
// to type arguments (`m[int]`) that names no type constructor in scope.
//
// suggest, when not empty, is the type the name most likely meant
// (`option` → Option).
func UnboundLowerTypeError(name, suggest string) error {
	if suggest != "" {
		return fmt.Errorf("%s: type '%s' is not in scope: no type constructor '%s' is declared here, imported by name, or exported by an import.\n"+
			"  Suggestion: did you mean the type `%s`?",
			TC_UNBOUND_TYPE_001, name, name, suggest)
	}
	return fmt.Errorf("%s: type '%s' is not in scope: no type constructor '%s' is declared here, imported by name, or exported by an import.\n"+
		"  Suggestion: declare it, import it by name (import <module> (%s)), or write the type you mean (a type variable cannot be applied to arguments).",
		TC_UNBOUND_TYPE_001, name, name, name)
}

// TC_UNBOUND_TYPE_002 codes the warning for a bare lowercase name read as a
// type variable that might have been meant as a type: an import exports a
// type of that name, or the name is imported by name as a value (#2211).
const TC_UNBOUND_TYPE_002 = "TC_UNBOUND_TYPE_002"

// AmbiguousModule is the module of a TCon minted for an ambiguous name (only
// reachable after TC_AMBIGUOUS_TYPE_001 was reported): it equals no real type.
const AmbiguousModule = "$ambiguous"

// Resolve is Lookup for callers that run after the elaborator has already
// reported any ambiguity (the interface builder): it never fails.
func (s *TypeScope) Resolve(name string) TypeRef {
	r, _ := s.Lookup(name)
	return r
}

// Con mints the TCon a bare type name denotes in this scope.
func (s *TypeScope) Con(name string) *TCon {
	r := s.Resolve(name)
	return &TCon{Name: r.Name, Module: r.Module}
}

// preludeTypes are the types the implicit prelude provides (step 4).
var preludeTypes = map[string]string{"Option": "std/option", "Result": "std/result"}

// PreludeTypeNames lists the prelude's type names, sorted.
func PreludeTypeNames() []string { return []string{"Option", "Result"} }

// ModulelessNames lists the module-less names (step 3), sorted.
func ModulelessNames() []string {
	out := make([]string, 0, len(moduleless))
	for n := range moduleless {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// nominalAmbiguityIsError makes an ambiguous name (and an ambiguous
// constructor) an error. Measurement builds turn it off to count, through the
// observer, how many real programs would be affected; production never does.
var nominalAmbiguityIsError = true

// AmbiguityEvent is one resolution that had more than one candidate (or, for
// "type-closure-only", one that only an indirect dependency decided).
type AmbiguityEvent struct {
	Kind       string // "type-direct", "type-closure", "type-closure-only" (one indirect candidate, used), "ctor-pattern", "ctor-pattern-resolved", "ctor-pattern-unresolved", "ctor-expr", "explicit-dup"
	Module     string // the module being compiled
	Name       string
	Candidates []TypeRef
}

var ambiguityObserver func(AmbiguityEvent)

// SetAmbiguityInstrument installs an observer of every multi-candidate
// resolution and chooses whether such a resolution is an error. It returns a
// function restoring the previous setting. Tests and measurement builds only.
func SetAmbiguityInstrument(observer func(AmbiguityEvent), asError bool) (restore func()) {
	prevObs, prevErr := ambiguityObserver, nominalAmbiguityIsError
	ambiguityObserver, nominalAmbiguityIsError = observer, asError
	return func() { ambiguityObserver, nominalAmbiguityIsError = prevObs, prevErr }
}

// AmbiguityIsError reports whether multi-candidate resolutions are errors.
func AmbiguityIsError() bool { return nominalAmbiguityIsError }

// ObserveAmbiguity reports a multi-candidate resolution decided outside this
// package (constructors, duplicate explicit imports) to the instrument.
func ObserveAmbiguity(ev AmbiguityEvent) {
	if ambiguityObserver != nil {
		ambiguityObserver(ev)
	}
}

func observeAmbiguity(module, name string, c []TypeRef, transitive bool) {
	kind := "type-direct"
	if transitive {
		kind = "type-closure"
	}
	ObserveAmbiguity(AmbiguityEvent{Kind: kind, Module: module, Name: name, Candidates: c})
}
