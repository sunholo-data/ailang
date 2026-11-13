package types

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// M-V1-RIGID-SIGNATURES (#2273): a type variable written in a declaration's
// signature is rigid while that declaration's body is checked. The body must
// work for EVERY instantiation of the variable, so the declared signature is a
// promise the body keeps, and the exported scheme is the written one (plus the
// class constraints the body needs: AILANG has no syntax for writing a
// constraint, so they are inferred and attached, as before).
//
// The rule is enforced after generalization, at the binding's let/letrec
// boundary (checkRigidSignature): every signature variable must still be a type
// variable, no two of them may have become one, each must be quantified by the
// binding's scheme, and none may carry an integer literal's Num constraint (an
// AILANG literal has no value at an arbitrary type: the runtime has no fromInt,
// see literal_restriction.go). Inference itself is unchanged, so a program that
// passes produces exactly the types, CoreTI and code it produced before.
//
// Which variables are rigid is decided by the elaborator (it knows scopes): a
// top-level `func`'s explicit `[a]` parameters and the implicit lowercase
// variables of its parameter and return annotations. Each gets a name unique to
// the declaration ("a@f"), so it cannot meet the free `a`/`b`/`k`/`v` that the
// module environment carries from other schemes, and so a nested annotation
// that names it (a `let x: a`, or a function literal's `(y: a)`) denotes the
// same rigid variable. Annotation variables that are not in scope (a `let` or
// literal annotation's own variables) stay flexible: they are equalities, not
// declared schemes.

// TC_RIGID_001 codes a signature type variable the body does not keep general.
const TC_RIGID_001 = "TC_RIGID_001"

// RigidVar is one type variable a declaration's written signature introduces.
type RigidVar struct {
	Name     string  // unique name used during inference ("a@toInt")
	Display  string  // as written ("a")
	Explicit bool    // listed in the declaration's [..]
	Pos      ast.Pos // first occurrence in the signature
	// Suggest is a type in scope the name was probably meant to be (it differs
	// only by case, or by one or two edits), "" when there is none. Set only for
	// implicit variables: an explicit [a] was meant to be a variable.
	Suggest string
	// ParamName marks an implicit variable spelled like one of the
	// declaration's parameters (`-> (stack, int)`): a value name used as a type.
	ParamName bool
	// Row marks a record row variable (`{x: int | r}`). It is scoped like a
	// type variable; whether it is enforced is rigidRows.
	Row bool
}

// RigidSig is the rigid variables of one declaration, keyed by its lambda.
type RigidSig struct {
	Func string
	Pos  ast.Pos
	Vars []RigidVar
}

// RigidTypeVarName is the unique inference name of the signature variable
// display of the declaration fn; n > 1 disambiguates a repeated declaration
// name within one module. The `@{...}` suffix is what WrittenTypeVarNames
// strips, so no diagnostic shows it.
func RigidTypeVarName(display, fn string, n int) string {
	if n > 1 {
		return fmt.Sprintf("%s@{%s#%d}", display, fn, n)
	}
	return display + "@{" + fn + "}"
}

var rigidSuffixRE = regexp.MustCompile(`@\{[^{}\s]*\}`)

// WrittenTypeVarNames rewrites every rigid inference name in s (`a@{f}`) to
// the name the author wrote (`a`).
func WrittenTypeVarNames(s string) string {
	if !strings.Contains(s, "@{") {
		return s
	}
	return rigidSuffixRE.ReplaceAllString(s, "")
}

// writtenNamesError is an inference error whose text names the variables as
// written. It unwraps to the original, so errors.As still finds its kind.
type writtenNamesError struct {
	err error
	msg string
}

func (e *writtenNamesError) Error() string { return e.msg }
func (e *writtenNamesError) Unwrap() error { return e.err }

// withWrittenNames makes err's text, and the message and fix of a structured
// report inside it, use written type-variable names.
func withWrittenNames(err error) error {
	if err == nil || !strings.Contains(err.Error(), "@{") {
		return err
	}
	var re *ailerrors.ReportError
	if errors.As(err, &re) && re.Rep != nil {
		re.Rep.Message = WrittenTypeVarNames(re.Rep.Message)
		if re.Rep.Fix != nil {
			re.Rep.Fix.Suggestion = WrittenTypeVarNames(re.Rep.Fix.Suggestion)
		}
	}
	return &writtenNamesError{err: err, msg: WrittenTypeVarNames(err.Error())}
}

// rigidState is the CoreTypeChecker's #2273 state.
type rigidState struct {
	// rigidSigs: Lambda NodeID -> the rigid variables of its declaration's
	// written signature, from the elaborator.
	rigidSigs map[uint64]*RigidSig
	// extraRigid: the signature InferWithRigid checks the current expression
	// against (rigid_sig_hook.go); nil otherwise.
	extraRigid *RigidSig
}

// SetRigidSignatures installs the elaborator's rigid signature variables
// (Lambda NodeID -> signature). Plumbed wherever SetParamTypeAnnotations is.
func (tc *CoreTypeChecker) SetRigidSignatures(sigs map[uint64]*RigidSig) {
	tc.rigidSigs = sigs
}

// RigidSignature returns the rigid signature of the lambda lamID, or nil.
// Exported so a pass that infers more code against a declaration's parameter
// types (contracts, tests) can enforce the same rule with CheckRigidVars.
func (tc *CoreTypeChecker) RigidSignature(lamID uint64) *RigidSig {
	return tc.rigidSigs[lamID]
}

// RigidSigError is TC_RIGID_001. It unwraps to a structured report, so
// `check --json` carries its code, location and one-line fix.
type RigidSigError struct {
	kind       string // "concrete", "equal", "literal", "escape"
	Pos        ast.Pos
	Message    string
	Suggestion string
	rep        *ailerrors.ReportError
}

func (e *RigidSigError) Error() string {
	s := fmt.Sprintf("%s: %s: %s", e.Pos, TC_RIGID_001, e.Message)
	if e.Suggestion != "" {
		s += "\n  Suggestion: " + e.Suggestion
	}
	return s
}

// Unwrap exposes the structured report to ailerrors.AsReport.
func (e *RigidSigError) Unwrap() error { return e.rep }

// Kind is how the promise was broken: "concrete" (the body needs a specific
// type), "equal" (two variables made one), "literal" (an integer literal at
// the variable) or "escape" (the variable is an outer value's type).
func (e *RigidSigError) Kind() string { return e.kind }

func newRigidSigError(pos ast.Pos, msg, suggestion string, data map[string]any) *RigidSigError {
	rep := &ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    TC_RIGID_001,
		Phase:   "typecheck",
		Message: msg,
		Span:    &ast.Span{Start: pos, End: pos},
		Data:    data,
	}
	if suggestion != "" {
		rep.Fix = &ailerrors.Fix{Suggestion: suggestion, Confidence: 0.8}
	}
	return &RigidSigError{Pos: pos, Message: msg, Suggestion: suggestion, rep: &ailerrors.ReportError{Rep: rep}}
}

// rigidViolation is one broken promise, before it is worded.
type rigidViolation struct {
	v     RigidVar
	kind  string // "concrete", "equal", "literal", "escape"
	got   Type   // concrete: what the body made v
	other RigidVar
	at    []string // the constraint path that decided it, when known
	// typoOK: v.v.Suggest agrees with what the body needs (typoConsistent).
	typoOK bool
	// group: the recursive group whose monomorphic call decided it, if one did.
	group []string
}

// CheckRigidVars reports the first signature variable of sig that resolve
// (the final substitution, applied in order) does not leave a distinct type
// variable quantified by quantified, or that carries an integer literal's
// constraint. It returns the violation's message as an error, or nil.
func CheckRigidVars(sig *RigidSig, quantified map[string]bool, constraints []ClassConstraint, resolve func(Type) Type) error {
	if v := findRigidViolation(sig, quantified, constraints, resolve); v != nil {
		return v.toError(sig)
	}
	return nil
}

func findRigidViolation(sig *RigidSig, quantified map[string]bool, constraints []ClassConstraint, resolve func(Type) Type) *rigidViolation {
	if sig == nil {
		return nil
	}
	literal := literalVarPaths(constraints)
	owner := make(map[string]int, len(sig.Vars))
	for i, v := range sig.Vars {
		if v.Row && !rigidRecordRows {
			continue
		}
		name, ok := rigidVarResolution(v, resolve)
		if !ok {
			return &rigidViolation{v: v, kind: "concrete", got: resolve(rigidVarType(v))}
		}
		if j, dup := owner[name]; dup {
			return &rigidViolation{v: sig.Vars[j], kind: "equal", other: v}
		}
		owner[name] = i
		if path, isLit := literal[name]; isLit {
			return &rigidViolation{v: v, kind: "literal", at: path}
		}
		if quantified != nil && !quantified[name] {
			return &rigidViolation{v: v, kind: "escape"}
		}
	}
	return nil
}

// rigidRecordRows: a signature's record row variable (`{name: string | r}`)
// is rigid too, so the body cannot use a field the signature does not list.
const rigidRecordRows = true

// rigidVarType is the variable v names, as the type checker writes it.
func rigidVarType(v RigidVar) Type {
	if v.Row {
		return &RowVar{Name: v.Name, Kind: RecordRow}
	}
	return &TVar2{Name: v.Name, Kind: Star}
}

// rigidVarResolution returns the variable v resolves to, and false when it
// resolved to anything else (a concrete type, or a row with fields).
func rigidVarResolution(v RigidVar, resolve func(Type) Type) (string, bool) {
	switch r := resolve(rigidVarType(v)).(type) {
	case *TVar2:
		return r.Name, true
	case *RowVar:
		return r.Name, true
	case *Row:
		if len(r.Labels) == 0 && r.Tail != nil {
			return r.Tail.Name, true
		}
	case *TRecordOpen:
		if len(r.Fields) == 0 && r.Row != nil {
			if rv, ok := r.Row.(*RowVar); ok {
				return rv.Name, true
			}
		}
	}
	return "", false
}

// literalVarPaths maps each free variable of an integer literal's Num
// constraint to that constraint's path.
func literalVarPaths(constraints []ClassConstraint) map[string][]string {
	out := map[string][]string{}
	for _, c := range constraints {
		if !c.IntLiteral {
			continue
		}
		vars := map[string]bool{}
		collectFreeVars(c.Type, vars)
		for v := range vars {
			if _, seen := out[v]; !seen {
				out[v] = c.Path
			}
		}
	}
	return out
}

// rigidScheme is the let/letrec generalization boundary's #2273 step: it
// checks scheme against value's rigid signature (when it has one) and returns
// it with the signature's variables under their written names.
func (tc *CoreTypeChecker) rigidScheme(ctx *InferenceContext, value core.CoreExpr, scheme *Scheme, constraints []ClassConstraint, subs ...Substitution) (*Scheme, error) {
	if err := tc.checkRigidSignature(ctx, value, scheme, constraints, subs...); err != nil {
		return nil, err
	}
	return displayRigidScheme(scheme, value, tc.rigidSigs), nil
}

// checkRigidSignature enforces the rigid signature of a let/letrec binding
// whose value is a declaration's lambda (see the file comment). subs is the
// binding's final substitution, in the order it was applied.
func (tc *CoreTypeChecker) checkRigidSignature(ctx *InferenceContext, value core.CoreExpr, scheme *Scheme, constraints []ClassConstraint, subs ...Substitution) error {
	lam, ok := value.(*core.Lambda)
	if !ok || scheme == nil {
		return nil
	}
	sig := tc.rigidSigs[lam.ID()]
	if sig == nil || len(sig.Vars) == 0 {
		return nil
	}
	quantified := make(map[string]bool, len(scheme.TypeVars)+len(scheme.RowVars))
	for _, v := range scheme.TypeVars {
		quantified[v] = true
	}
	for _, v := range scheme.RowVars {
		quantified[v] = true
	}
	resolve := func(t Type) Type {
		for _, s := range subs {
			t = ApplySubstitution(s, t)
		}
		return t
	}
	v := findRigidViolation(sig, quantified, constraints, resolve)
	if v == nil {
		return nil
	}
	if v.kind == "concrete" || v.kind == "equal" {
		v.at = ctx.rigidCulprit(sig, v, false)
		// The culprit was a recursive group's binding equality ("f ~ its
		// value"): the real cause is a call of a group member at another type.
		// Replaying with the group's bindings tied first finds that call.
		if len(v.at) == 1 && ctx.rigidGroup[v.at[0]] {
			if at := ctx.rigidCulprit(sig, v, true); at != nil {
				v.at = at
			}
			v.group = sortedKeys(ctx.rigidGroup)
		}
	}
	v.typoOK = tc.typoConsistent(v.v.Suggest, v)
	return v.toError(sig)
}

// rigidCulprit replays the context's equalities in order and returns the path
// of the first one after which v is no longer a variable of its own: the place
// in the body that broke the promise. Runs only on failure.
func (ctx *InferenceContext) rigidCulprit(sig *RigidSig, v *rigidViolation, groupFirst bool) []string {
	sub := make(Substitution)
	at := func(t Type) Type { return ApplySubstitution(sub, t) }
	broken := func() bool {
		name, ok := rigidVarResolution(v.v, at)
		if !ok {
			return true
		}
		if v.kind != "equal" {
			return false
		}
		o, ok := rigidVarResolution(v.other, at)
		return ok && o == name
	}
	order := ctx.constraints
	if groupFirst {
		order = make([]TypeConstraint, 0, len(ctx.constraints))
		var rest []TypeConstraint
		for _, c := range ctx.constraints {
			if eq, ok := c.(TypeEq); ok && len(eq.Path) == 1 && ctx.rigidGroup[eq.Path[0]] {
				order = append(order, c)
			} else {
				rest = append(rest, c)
			}
		}
		order = append(order, rest...)
	}
	for _, c := range order {
		if groupFirst {
			if eq, ok := c.(TypeEq); ok && len(eq.Path) == 1 && ctx.rigidGroup[eq.Path[0]] {
				var err error
				if sub, err = ctx.unifier.Unify(ApplySubstitution(sub, eq.Left), ApplySubstitution(sub, eq.Right), sub); err != nil {
					return nil
				}
				continue // tying the group is not the culprit
			}
		}
		var err error
		var path []string
		switch eq := c.(type) {
		case TypeEq:
			path = eq.Path
			sub, err = ctx.unifier.Unify(ApplySubstitution(sub, eq.Left), ApplySubstitution(sub, eq.Right), sub)
		case RowEq:
			path = eq.Path
			sub, err = ctx.unifier.rowUnifier.UnifyRows(eq.Left, eq.Right, sub)
		default:
			continue
		}
		if err != nil {
			return nil
		}
		if broken() {
			return path
		}
	}
	return nil
}

// displayRigidScheme renames a checked binding's quantified signature
// variables back to their written names (alpha-renaming: same scheme), so the
// environment, the REPL and diagnostics at use sites read as the source does.
func displayRigidScheme(scheme *Scheme, value core.CoreExpr, sigs map[uint64]*RigidSig) *Scheme {
	lam, ok := value.(*core.Lambda)
	if !ok || scheme == nil {
		return scheme
	}
	sig := sigs[lam.ID()]
	if sig == nil {
		return scheme
	}
	taken := map[string]bool{}
	collectFreeVars(scheme.Type, taken)
	for _, v := range scheme.TypeVars {
		taken[v] = true
	}
	quantified := make(map[string]bool, len(scheme.TypeVars))
	for _, v := range scheme.TypeVars {
		quantified[v] = true
	}
	rename := map[string]Type{}
	newName := map[string]string{}
	for _, v := range sig.Vars {
		if !quantified[v.Name] || taken[v.Display] {
			continue
		}
		taken[v.Display] = true
		rename[v.Name] = &TVar2{Name: v.Display, Kind: Star}
		newName[v.Name] = v.Display
	}
	if len(rename) == 0 {
		return scheme
	}
	out := &Scheme{RowVars: scheme.RowVars, Type: scheme.Type.Substitute(rename)}
	for _, v := range scheme.TypeVars {
		if n, ok := newName[v]; ok {
			v = n
		}
		out.TypeVars = append(out.TypeVars, v)
	}
	for _, c := range scheme.Constraints {
		out.Constraints = append(out.Constraints, Constraint{Class: c.Class, Type: c.Type.Substitute(rename)})
	}
	return out
}
