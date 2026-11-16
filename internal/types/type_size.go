package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// The type-size backstop (#2308, #2329).
//
// A type is held as a DAG: substitution shares the nodes it reuses, so
// `let t1 = (t0, t0)` repeated n times is n nodes in memory. But it denotes a
// TREE of 2^n nodes, and most of the checker's traversals (rendering, JSON,
// unification, free-variable collection, field-signature matching) walk the
// tree. A ~1 KB file could therefore pin a CPU for minutes or, where a
// traversal allocates per node, exhaust the machine's memory — through
// `check`, and so through the LSP and the hosted check tool.
//
// The issues' shapes are fixed at their cause: a record literal no longer
// carries its fields twice, and the traversals that expand aliases at every
// level (the IFC parameter-bound walk, the per-application callback scan) or
// run over every node (TypeName propagation) visit a shared node once. This
// file is the backstop for what remains, with one code, TC_TYPE_SIZE_001:
//
//   - a type whose tree exceeds maxTypeTreeNodes, raised where it first
//     appears — a let binding (inferLet), a declaration's final
//     node types (inferWithConstraints), or a module interface once its
//     aliases are closed (pipeline checkInterfaceTypeSizes) — before any later
//     traversal walks it;
//   - a declaration whose unifications take more than maxUnifySteps.
//
// The size is of the type as held: an alias application counts as written
// (`P0[int]` is two nodes however large its expansion), because the traversals
// that expand aliases share repeated work and a signature naming a large alias
// costs nothing until the expansion is materialised; once it is (a call binds
// a variable to the expanded argument, an interface closes the alias), the
// materialised type is what is measured. Written types are linear in the
// source, so they need no guard of their own. Counting is memoised by node, so
// measuring costs the size of the DAG, not of the tree.

// TC_TYPE_SIZE_001 codes a type too large to check.
const TC_TYPE_SIZE_001 = "TC_TYPE_SIZE_001"

// TypeSizeCodes lists the inline codes of this file; `check --json` reports
// each as itself.
func TypeSizeCodes() []string { return []string{TC_TYPE_SIZE_001} }

// TypeTooLargeError is the kind of TC_TYPE_SIZE_001.
const TypeTooLargeError TypeErrorKind = "type_too_large"

// maxTypeTreeNodes bounds the tree size of any one type. Measured 2026-11-16
// at every guard point: the largest type in the repository's .ail files is
// 107 nodes, and in the 14,779 banked passing solutions 100, so the bound is
// over 600x the largest real type (the deepest is 10 levels). One traversal
// walks a tree this size in milliseconds.
const maxTypeTreeNodes int64 = 1 << 16

// maxUnifySteps bounds the unification steps (Unify calls, each node pair
// one step) of one declaration. A large type is caught by size where it is
// held; this catches a walk over an expansion that never is held, such as two
// separately declared alias DAGs with variables in them unified with each
// other (ground ones share their pairs, unification_alias_pairs.go). Measured
// 2026-11-16: the most steps any declaration takes is 42,564 in the
// repository's .ail files and 365,050 in the 14,779 banked passing solutions,
// so the bound is over 180x the largest real declaration.
const maxUnifySteps = 1 << 26

// unifyWork counts the unification steps of one declaration.
type unifyWork struct {
	steps int
	limit int     // maxUnifySteps; smaller only in tests
	pos   ast.Pos // the declaration, for the diagnostic
}

func newUnifyWork(pos ast.Pos) *unifyWork { return &unifyWork{limit: maxUnifySteps, pos: pos} }

func (w *unifyWork) step() {
	if w == nil {
		return
	}
	if w.steps++; w.steps > w.limit {
		panic(typeTooLarge{what: "this declaration", pos: positionString(w.pos),
			detail: fmt.Sprintf("unifying its types took more than %d steps", w.limit)})
	}
}

// typeTooLarge unwinds inference when a type exceeds the bound; the
// declaration boundary (recoverLatentDataDepth) turns it into the coded error.
type typeTooLarge struct {
	what   string // what the type belongs to, e.g. "let binding t"
	pos    string // source position, when known
	detail string // the bound exceeded; "" = the type-node bound
	size   int64  // the measured tree size, for the type-node bound
}

func (e typeTooLarge) Error() string {
	where := ""
	if e.pos != "" {
		where = " at " + e.pos
	}
	if e.detail != "" {
		return fmt.Sprintf("%s: type too large to check (%s%s): %s. Simplify or split the types involved",
			TC_TYPE_SIZE_001, e.what, where, e.detail)
	}
	return fmt.Sprintf("%s: type too large to check (%s%s): its type has %s nodes when written out in full, over the limit of %s. "+
		"A type that holds the same part twice at every step doubles each time (`let t1 = (t0, t0)`, an alias `type P0[a] = P1[(a, a)]`); "+
		"wrap the repeated part in a data type (`type Pair = Pair(T, T)`), which is never expanded, or restructure the value",
		TC_TYPE_SIZE_001, e.what, where, groupDigits(e.size), groupDigits(maxTypeTreeNodes))
}

// sizeCap saturates the size count: a doubling of 62 levels already reaches it.
const sizeCap int64 = 1 << 62

// groupDigits writes n with thousands separators (65,536).
func groupDigits(n int64) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func positionString(pos ast.Pos) string {
	if pos.Line > 0 {
		return pos.String()
	}
	return ""
}

// guardTypeSize panics typeTooLarge when t's tree exceeds the bound.
func guardTypeSize(t Type, what string, pos ast.Pos) {
	newTypeSizer().guard(t, what, pos)
}

// letBindingWhat names a let binding for the diagnostic; the elaborator's own
// temporaries ($tmpN, for a nested application) are not names the user wrote.
func letBindingWhat(name string) string {
	if strings.HasPrefix(name, "$") {
		return "an intermediate expression"
	}
	return "let binding " + name
}

// guardDeclarationSizes bounds the final type of every node the current
// declaration typed, after its substitution is applied: what lowering,
// interface export and the finalisation passes walk next. One sizer serves
// them all, so a type shared by many nodes is measured once.
func (tc *CoreTypeChecker) guardDeclarationSizes(decl core.CoreExpr, nodes []uint64) {
	s := newTypeSizer()
	for _, id := range nodes {
		s.guard(tc.CoreTI[id], "this declaration", decl.Span())
	}
}

// TypeSizeCheck bounds types outside inference, where an expansion is first
// materialised: a module interface closes its nullary aliases into their
// bodies, and the interface is then hashed and serialised as a tree. One check
// shares its memo across calls.
type TypeSizeCheck struct{ s *typeSizer }

// NewTypeSizeCheck starts a check with an empty memo.
func NewTypeSizeCheck() *TypeSizeCheck { return &TypeSizeCheck{s: newTypeSizer()} }

// Check returns TC_TYPE_SIZE_001 naming what when t's tree exceeds the bound.
func (c *TypeSizeCheck) Check(t Type, what string) error {
	if t == nil || c.s.size(t) <= c.s.limit {
		return nil
	}
	return &TypeCheckError{Kind: TypeTooLargeError, Message: typeTooLarge{what: what, size: c.s.size(t)}.Error()}
}

// typeSizer measures tree sizes exactly (saturating at sizeCap, far above any
// limit), so the diagnostic can name the size it measured. Each node is
// counted once and its subtree size memoised, so the cost is the size of the
// DAG, however large the tree.
//
// Cycle-safe: a node met again on the current path counts 1 and is not
// re-entered (cyclic graphs exist transiently, see safeSubstitute).
type typeSizer struct {
	limit  int64
	memo   map[Type]int64
	active map[Type]bool
}

func newTypeSizer() *typeSizer {
	return &typeSizer{limit: maxTypeTreeNodes, memo: map[Type]int64{}, active: map[Type]bool{}}
}

func (s *typeSizer) guard(t Type, what string, pos ast.Pos) {
	if t != nil && s.size(t) > s.limit {
		panic(typeTooLarge{what: what, pos: positionString(pos), size: s.size(t)})
	}
}

func (s *typeSizer) size(t Type) int64 {
	if t == nil {
		return 0
	}
	if n, ok := s.memo[t]; ok {
		return n
	}
	if s.active[t] {
		return 1
	}
	s.active[t] = true
	n := s.measure(t)
	delete(s.active, t)
	if n > sizeCap {
		n = sizeCap
	}
	s.memo[t] = n
	return n
}

func (s *typeSizer) measure(t Type) int64 {
	n := int64(1)
	add := func(ts ...Type) {
		for _, c := range ts {
			if m := s.size(c); m > sizeCap-n {
				n = sizeCap
			} else {
				n += m
			}
		}
	}
	switch x := t.(type) {
	case *TList:
		add(x.Element)
	case *TArray:
		add(x.Element)
	case *TMap:
		add(x.Key, x.Value)
	case *TTuple:
		add(x.Elements...)
	case *TApp:
		add(x.Constructor)
		add(x.Args...)
	case *TFunc2:
		add(x.Params...)
		add(x.Return)
		if x.EffectRow != nil {
			add(x.EffectRow)
		}
	case *TRecord:
		for _, f := range x.Fields {
			add(f)
		}
		add(x.Row)
	case *TRecordOpen:
		for _, f := range x.Fields {
			add(f)
		}
		add(x.Row)
	case *TRecord2:
		if x.Row != nil {
			add(x.Row)
		}
	case *Row:
		for _, l := range x.Labels {
			add(l)
		}
		if x.Tail != nil {
			add(x.Tail)
		}
	case *TLabelled:
		add(x.Inner)
	}
	return n
}
