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
// read goes through Lookup. (Lowercase names never get here: the parser makes
// them type variables. An uppercase name is always a type constructor, even
// inside a `[T]` type-parameter list — #2240.)
//
//  1. a type this module declares (BindLocal);
//  2. a type bound by an explicit selective import the USER wrote,
//     `import M (T)` or `import M (T as U)` (BindExplicit; two different ones
//     are MOD015 at the import, so there is at most one). The prelude the
//     loader injects into an entry module is NOT this step but step 4;
//  3. a built-in, module-less name: Int, String, Map, List, Array, Bytes, Unit
//     and the other names of IsModuleless;
//  4. the prelude: Option -> std/option, Result -> std/result;
//  5. the ONE declaration of that name among the types every imported module
//     exports (AddDirect) and every module of the import closure declares
//     (AddClosure) — two or more distinct ones are TC_AMBIGUOUS_TYPE_001, so
//     an unrelated import can never silently change what a name means;
//  6. the builtin home table (Json -> std/json, ... DOMEvent -> $builtin);
//  7. otherwise this module: an undeclared name stays an opaque type of the
//     module that writes it (TC_UNBOUND_TYPE_001 is a later milestone).
//
// No step depends on import order, file order or module path: steps 1–2 hold
// one binding per name, and step 5 has exactly one candidate or fails naming
// all of them (sorted by module).
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

// BindLocal records a type this module declares (step 2).
func (s *TypeScope) BindLocal(name string) {
	s.local[name] = TypeRef{Module: s.Module, Name: name}
}

// BindExplicit records a type an explicit selective import binds (step 3).
func (s *TypeScope) BindExplicit(name string, ref TypeRef) { s.explicit[name] = ref }

// AddDirect records a type a direct dependency exports (step 6).
func (s *TypeScope) AddDirect(ref TypeRef) {
	s.direct[ref.Name] = addCandidate(s.direct[ref.Name], ref)
}

// AddClosure records a type a module of the import closure declares (step 7).
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
		how = "modules this one reaches through its imports declare"
	}
	alias := e.Name
	if len(e.Candidates) > 1 {
		alias = e.Name + " as " + strings.ToUpper(lastSeg(mods[1])[:1]) + lastSeg(mods[1])[1:] + e.Name
	}
	return fmt.Sprintf("%s: type '%s' is ambiguous here: %s more than one type named %s (%s), and none is declared here or imported by name.\n"+
		"  Suggestion: import the one you mean by name, e.g. import %s (%s), or import %s (%s).",
		TC_AMBIGUOUS_TYPE_001, e.Name, how, e.Name, strings.Join(mods, ", "), mods[0], e.Name, mods[1], alias)
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
		var cands []TypeRef
		for _, r := range s.direct[name] {
			cands = addCandidate(cands, r)
		}
		transitive := len(cands) == 0
		for _, r := range s.closure[name] {
			cands = addCandidate(cands, r)
		}
		switch len(cands) {
		case 0:
		case 1:
			return cands[0], nil
		default:
			observeAmbiguity(s.Module, name, cands, transitive)
			// No candidate is picked: the name denotes none of them.
			ambiguous := TypeRef{Module: AmbiguousModule, Name: name}
			if nominalAmbiguityIsError {
				return ambiguous, &AmbiguousTypeError{Name: name, Candidates: cands, Transitive: transitive}
			}
			return ambiguous, nil
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

// preludeTypes are the types the implicit prelude provides (step 5).
var preludeTypes = map[string]string{"Option": "std/option", "Result": "std/result"}

// PreludeTypeNames lists the prelude's type names, sorted.
func PreludeTypeNames() []string { return []string{"Option", "Result"} }

// ModulelessNames lists the module-less names (step 4), sorted.
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

// AmbiguityEvent is one resolution that had more than one candidate.
type AmbiguityEvent struct {
	Kind       string // "type-direct", "type-closure", "ctor-pattern", "ctor-expr", "explicit-dup"
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
