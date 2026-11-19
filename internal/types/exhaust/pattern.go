// Package exhaust decides whether a `match` covers every value of its
// scrutinee's type and which of its arms can never be reached, and names a
// concrete missing pattern (a witness such as `Blue`, `Some(Err(_))` or `[]`)
// when it does not.
//
// It implements Maranget's usefulness algorithm ("Warnings for pattern
// matching", JFP 2007) over an abstract pattern matrix: exhaustiveness is
// "the wildcard row is not useful after the unguarded arms", redundancy is
// "arm i is not useful after the unguarded arms before it". Guarded arms
// never count toward coverage; they are still checked for their own
// reachability.
//
// The package is pure: it knows nothing about the pipeline, the type checker
// or modules. An ADT's complete, declaration-ordered constructor list comes
// from a Universe the caller supplies, keyed by whatever type identity the
// caller resolved (design: m-v1-match-exhaustiveness.md §4, §8 — the nominal
// type identity plugs in there). Every inconsistency — an unknown type, a
// constructor its universe does not list, two families in one column — is an
// error, never a silent "exhaustive": the caller fails closed.
package exhaust

// Pattern is one pattern of a match arm, in the language's surface forms.
// The forms (enumerated from internal/parser/parser_pattern.go and Core):
// Wild (`_`, a variable), Ctor, Lit (bool, int, float, string, unit), List
// (`[]`, `[p, q]`, `[p, ...r]`, `h :: t`), Tuple and Record. There are no
// or-patterns and no as-patterns in the language.
type Pattern interface{ isPattern() }

// Wild is `_` or a variable: it matches every value.
//
// On a witness, Domain/Excluding say that the missing values are those of an
// infinite literal domain other than the listed literals ("any int other than
// 0, 1"); input patterns leave both empty.
type Wild struct {
	Domain    string   // "int", "float" or "string"; witness only
	Excluding []string // rendered literals already covered; witness only
}

// Ctor is a constructor pattern `C` or `C(p1, …, pn)`.
type Ctor struct {
	// TypeName is the caller's identity for the ADT the constructor belongs
	// to. It is opaque here and only handed to the Universe.
	TypeName string
	Name     string
	Args     []Pattern
}

// Lit is a literal pattern. Value is bool, int, int64, float64 or string;
// nil is the unit pattern `()`.
type Lit struct{ Value any }

// List is `[p1, …, pk]` (Rest nil) or `[p1, …, pk, ...r]` / `h :: t`
// (Rest = r).
type List struct {
	Elems []Pattern
	Rest  Pattern
}

// Tuple is `(p1, …, pn)`, n ≥ 2. A zero-element tuple is the unit pattern.
type Tuple struct{ Elems []Pattern }

// Record is `{f: p, g, ...}`. Fields not named match anything; whether the
// pattern was open (`...`) changes typing, not coverage.
type Record struct{ Fields map[string]Pattern }

func (Wild) isPattern()   {}
func (Ctor) isPattern()   {}
func (Lit) isPattern()    {}
func (List) isPattern()   {}
func (Tuple) isPattern()  {}
func (Record) isPattern() {}

// CtorSig is one constructor of an ADT: its name and how many arguments it
// takes.
type CtorSig struct {
	Name  string
	Arity int
}

// Universe supplies an ADT's complete constructor list. It must return EVERY
// constructor of the type that c belongs to, in declaration order — never
// just the constructors in scope, or a match naming the in-scope ones would
// be wrongly called exhaustive. ok=false means the type is unknown; the
// check then fails closed with ErrUnknownType.
type Universe interface {
	CtorsOf(c Ctor) (sigs []CtorSig, ok bool)
}

// StaticUniverse is a Universe backed by a map from type identity
// (Ctor.TypeName) to that type's constructors.
type StaticUniverse map[string][]CtorSig

// CtorsOf implements Universe.
func (u StaticUniverse) CtorsOf(c Ctor) ([]CtorSig, bool) {
	sigs, ok := u[c.TypeName]
	return sigs, ok
}

// Arm is one match arm: its pattern and whether it has an `if` guard.
type Arm struct {
	Pattern Pattern
	Guarded bool
}
