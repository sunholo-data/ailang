package types

import (
	"fmt"
	"sort"
	"strings"
)

// TypeErrorKind represents the kind of type error
type TypeErrorKind string

const (
	KindMismatchError       TypeErrorKind = "kind_mismatch"
	TypeMismatchError       TypeErrorKind = "type_mismatch"
	RowMismatchError        TypeErrorKind = "row_mismatch"
	OccursCheckError        TypeErrorKind = "occurs_check"
	UnboundVariableError    TypeErrorKind = "unbound_variable"
	ArityMismatchError      TypeErrorKind = "arity_mismatch"
	MissingFieldError       TypeErrorKind = "missing_field"
	ExtraFieldError         TypeErrorKind = "extra_field"
	MissingEffectError      TypeErrorKind = "missing_effect"
	ExtraEffectError        TypeErrorKind = "extra_effect"
	UnsolvedConstraintError TypeErrorKind = "unsolved_constraint"

	// MatchForeignConstructorError (M-MATCH-ADT-XCHECK, v0.18.10) fires
	// when a match arm uses a constructor from a different ADT than the
	// scrutinee's type. The previous mechanism — letting unification
	// produce a generic "type constructor mismatch" — caught the bug
	// but the error message didn't tell users which arm's constructor
	// was wrong or what they should have used instead. This kind
	// surfaces the offending constructor name AND lists the constructors
	// of both ADTs so the fix is obvious from the error message alone.
	MatchForeignConstructorError TypeErrorKind = "match_foreign_constructor"

	// RecordAccessOnTaggedUnionError (M-TYPECHECK-NO-AUTO-UNWRAP-RESULT,
	// v0.20.0) fires when `expr.field` is applied to a value typed as a
	// multi-constructor ADT (Result, Option, user-defined multi-variant).
	// Pre-v0.20.0 this silently auto-unwrapped the variant payload at
	// runtime — the Ok path of `result.message.content` worked while the
	// Err path crashed with `cannot access field of non-record value:
	// *eval.StringValue`. This error code rejects the access at compile
	// time with a prescriptive `match { Ok(x) => ..., Err(e) => ... }`
	// template, forcing the consumer to author both arms.
	RecordAccessOnTaggedUnionError TypeErrorKind = "record_access_on_tagged_union"

	// SinkRefinementError (M-SECRET-EFFECT / M-TAINT-TYPES, v0.26.0) fires when
	// a value whose information-flow label is forbidden by a parameter's
	// {not ℓ} refinement reaches that sink — e.g. a <secret> value passed to a
	// string{not secret} logging or network parameter without an intervening
	// ! {Declassify} step. Enforced by CheckModuleIFC (ifc_check.go).
	SinkRefinementError TypeErrorKind = "sink_refinement"

	// DeclassifyRequiredError (M-SECRET-EFFECT / M-TAINT-TYPES, v0.26.0) fires
	// when a function declares an explicit return label that hides taint actually
	// present in its body (label laundering) without declaring ! {Declassify}.
	// Enforced by CheckModuleIFC (ifc_check.go).
	DeclassifyRequiredError TypeErrorKind = "declassify_required"
)

// TypeCheckError represents a detailed type checking error
type TypeCheckError struct {
	Kind       TypeErrorKind
	Path       []string // Field/expression path
	Position   string   // Source position
	Expected   Type
	Actual     Type
	Message    string
	Suggestion string
}

func (e *TypeCheckError) Error() string {
	var parts []string

	if e.Position != "" {
		parts = append(parts, e.Position)
	}

	if len(e.Path) > 0 {
		parts = append(parts, fmt.Sprintf("at %s", strings.Join(e.Path, ".")))
	}

	parts = append(parts, e.Message)

	if e.Expected != nil && e.Actual != nil {
		parts = append(parts, fmt.Sprintf("\n  Expected: %s\n  Actual:   %s", e.Expected, e.Actual))
	}

	if e.Suggestion != "" {
		parts = append(parts, fmt.Sprintf("\n  Suggestion: %s", e.Suggestion))
	}

	return strings.Join(parts, ": ")
}

// NewKindMismatchError creates a kind mismatch error
func NewKindMismatchError(expected, actual Kind, path []string) *TypeCheckError {
	return &TypeCheckError{
		Kind:    KindMismatchError,
		Path:    path,
		Message: fmt.Sprintf("kind mismatch: expected %s, got %s", expected, actual),
	}
}

// NewMatchForeignConstructorError creates a structured error for match
// arms that reference a constructor from a different ADT than the
// scrutinee's type. ctor is the offending constructor name (e.g. "Err"),
// ctorADT is the ADT it belongs to (e.g. "Result"), and scrutADT is the
// scrutinee's ADT (e.g. "Option"). ctorList/scrutList enumerate the
// valid constructors of each ADT so the user can fix the mistake from
// the error message alone.
//
// Example output:
//
//	match arm constructor 'Err' belongs to ADT 'Result', not 'Option'
//	(the scrutinee's type).
//	  Option's constructors are: None, Some
//	  Result's constructors are: Err, Ok
//	  Suggestion: did you mean 'None' or 'Some'?
//
// M-MATCH-ADT-XCHECK (v0.18.10). See also:
// design_docs/implemented/v0_18_10/m-match-adt-xcheck.md.
func NewMatchForeignConstructorError(ctor, ctorADT, scrutADT string, ctorList, scrutList []string, path []string) *TypeCheckError {
	sort.Strings(ctorList)
	sort.Strings(scrutList)
	suggestion := ""
	if len(scrutList) > 0 {
		if len(scrutList) == 1 {
			suggestion = fmt.Sprintf("did you mean '%s'?", scrutList[0])
		} else {
			suggestion = fmt.Sprintf("did you mean one of: %s?", strings.Join(scrutList, ", "))
		}
	}
	msg := fmt.Sprintf(
		"match arm constructor '%s' belongs to ADT '%s', not '%s' (the scrutinee's type).\n  %s's constructors are: %s\n  %s's constructors are: %s",
		ctor, ctorADT, scrutADT,
		scrutADT, strings.Join(scrutList, ", "),
		ctorADT, strings.Join(ctorList, ", "),
	)
	return &TypeCheckError{
		Kind:       MatchForeignConstructorError,
		Path:       path,
		Message:    msg,
		Suggestion: suggestion,
	}
}

// NewTypeMismatchError creates a type mismatch error
func NewTypeMismatchError(expected, actual Type, path []string) *TypeCheckError {
	return &TypeCheckError{
		Kind:     TypeMismatchError,
		Path:     path,
		Expected: expected,
		Actual:   actual,
		Message:  "type mismatch",
	}
}

// NewRowMismatchError creates a detailed row mismatch error
func NewRowMismatchError(expected, actual *Row, path []string) *TypeCheckError {
	if expected.Kind.Equals(EffectRow) {
		return newEffectRowError(expected, actual, path)
	}
	return newRecordRowError(expected, actual, path)
}

// newEffectRowError creates an error for effect row mismatches
func newEffectRowError(expected, actual *Row, path []string) *TypeCheckError {
	expectedEffects := make([]string, 0, len(expected.Labels))
	for k := range expected.Labels {
		expectedEffects = append(expectedEffects, k)
	}
	sort.Strings(expectedEffects)

	actualEffects := make([]string, 0, len(actual.Labels))
	for k := range actual.Labels {
		actualEffects = append(actualEffects, k)
	}
	sort.Strings(actualEffects)

	// Find missing and extra effects
	missing := []string{}
	for _, e := range expectedEffects {
		found := false
		for _, a := range actualEffects {
			if e == a {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, e)
		}
	}

	extra := []string{}
	for _, a := range actualEffects {
		found := false
		for _, e := range expectedEffects {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, a)
		}
	}

	message := "effect row mismatch"
	suggestions := []string{}

	if len(missing) > 0 {
		parts := make([]string, len(missing))
		for i, label := range missing {
			if expected.Provenance != nil {
				if span, ok := expected.Provenance[label]; ok && span.Start.File != "" {
					parts[i] = fmt.Sprintf("%s (slot at %s)", label, span.Start)
					continue
				}
			}
			parts[i] = label
		}
		message = fmt.Sprintf("missing required effects: {%s}", strings.Join(parts, ", "))
		suggestions = append(suggestions, fmt.Sprintf("Consider adding capability %s", strings.Join(missing, ", ")))
	}

	if len(extra) > 0 {
		parts := make([]string, len(extra))
		for i, label := range extra {
			if actual.Provenance != nil {
				if span, ok := actual.Provenance[label]; ok && span.Start.File != "" {
					parts[i] = fmt.Sprintf("%s (introduced at %s)", label, span.Start)
					continue
				}
			}
			parts[i] = label
		}
		if len(missing) > 0 {
			message += fmt.Sprintf("; has extra effects: {%s}", strings.Join(parts, ", "))
		} else {
			message = fmt.Sprintf("has extra effects: {%s}", strings.Join(parts, ", "))
		}
		suggestions = append(suggestions, fmt.Sprintf("Consider handling effect %s", strings.Join(extra, ", ")))
	}

	return &TypeCheckError{
		Kind:       RowMismatchError,
		Path:       path,
		Message:    message,
		Suggestion: strings.Join(suggestions, " or "),
	}
}

// newRecordRowError creates an error for record row mismatches
func newRecordRowError(expected, actual *Row, path []string) *TypeCheckError {
	// Find missing and extra fields
	missing := []string{}
	for k := range expected.Labels {
		if _, ok := actual.Labels[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)

	extra := []string{}
	typeMismatches := []string{}
	for k, actualType := range actual.Labels {
		if expectedType, ok := expected.Labels[k]; ok {
			// Field exists, check type
			if !expectedType.Equals(actualType) {
				fieldPath := append(path, k)
				typeMismatches = append(typeMismatches,
					fmt.Sprintf("%s: expected %s, found %s",
						strings.Join(fieldPath, "."), expectedType, actualType))
			}
		} else {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)

	message := "record row mismatch"
	suggestions := []string{}

	if len(missing) > 0 {
		message = fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", "))
		suggestions = append(suggestions, fmt.Sprintf("Add fields: %s", strings.Join(missing, ", ")))
	}

	if len(extra) > 0 {
		if len(missing) > 0 {
			message += fmt.Sprintf("; has extra fields: %s", strings.Join(extra, ", "))
		} else {
			message = fmt.Sprintf("has extra fields: %s", strings.Join(extra, ", "))
		}
		if expected.Tail == nil {
			suggestions = append(suggestions, "This record type doesn't allow extra fields")
		}
	}

	if len(typeMismatches) > 0 {
		if len(missing) > 0 || len(extra) > 0 {
			message += "; "
		}
		message += fmt.Sprintf("field type mismatches: %s", strings.Join(typeMismatches, ", "))
	}

	return &TypeCheckError{
		Kind:       RowMismatchError,
		Path:       path,
		Message:    message,
		Suggestion: strings.Join(suggestions, "; "),
	}
}

// NewOccursCheckError creates an occurs check error
func NewOccursCheckError(varName string, inType Type) *TypeCheckError {
	return &TypeCheckError{
		Kind:       OccursCheckError,
		Message:    fmt.Sprintf("infinite type: %s occurs in %s", varName, inType),
		Suggestion: "This would create an infinite type. Check for recursive definitions without a base case.",
	}
}

// NewUnboundVariableError creates an unbound variable error
func NewUnboundVariableError(name string, path []string) *TypeCheckError {
	return &TypeCheckError{
		Kind:       UnboundVariableError,
		Path:       path,
		Message:    fmt.Sprintf("unbound variable: %s", name),
		Suggestion: fmt.Sprintf("Variable '%s' is not defined. Did you mean to define it with 'let' first?", name),
	}
}

// NewArityMismatchError creates an arity mismatch error
func NewArityMismatchError(expected, actual int, path []string) *TypeCheckError {
	return &TypeCheckError{
		Kind:    ArityMismatchError,
		Path:    path,
		Message: fmt.Sprintf("function expects %d argument(s), but %d provided", expected, actual),
	}
}

// TC_ARITY_001 codes an irreconcilable function-arity mismatch (M-ARITY-STYLE).
// See arityMismatchMsg for the rendered, directional, style-aware text.
const TC_ARITY_001 = "TC_ARITY_001"

// arityMismatchMsg builds the coded, directional, style-aware arity diagnostic
// emitted at unification_types.go's post-curry-flatten else branch (M-ARITY-STYLE).
//
// The code + Suggestion are embedded INLINE in the returned string (not a
// *TypeCheckError.Suggestion field): the error is wrapped by a plain `%w` at
// inference_helpers.go and nothing recovers *TypeCheckError via errors.As, so a
// struct Suggestion would never render. Mirrors the TC_REC_00X convention.
//
// AILANG has strict arity and NO partial application, so the hint targets the
// two fleet-tier ML habits directly: under-supply (partial application) and
// over-supply (extra arguments).
func arityMismatchMsg(expected, actual int) string {
	var hint string
	switch {
	case actual < expected:
		hint = fmt.Sprintf("AILANG has no partial application — call with all %d arguments, or wrap in a lambda `\\a b. f(a, b)`.", expected)
	case actual > expected:
		extra := actual - expected
		hint = fmt.Sprintf("Remove the extra %d argument(s); this function takes %d.", extra, expected)
	default:
		// Unreachable in the emission-site else branch (that branch runs only
		// when arities differ), but stay coded and neutral rather than emit a
		// nonsense hint if ever reached.
		hint = "Check the function's declared arity against the call site."
	}
	return fmt.Sprintf("%s: function expects %d argument(s), but %d provided\n  Suggestion: %s",
		TC_ARITY_001, expected, actual, hint)
}

// NewUnsolvedConstraintError creates an unsolved type class constraint error
func NewUnsolvedConstraintError(className string, typ Type, path []string) *TypeCheckError {
	suggestion := ""
	switch className {
	case "Num":
		suggestion = fmt.Sprintf("Type %s must support numeric operations (+, -, *, /). Ensure it's a numeric type (int, float).", typ)
	case "Ord":
		suggestion = fmt.Sprintf("Type %s must support ordering operations (<, >, <=, >=). Ensure it's an orderable type.", typ)
	case "Eq":
		suggestion = fmt.Sprintf("Type %s must support equality operations (==, !=). Most types support equality by default.", typ)
	case "Show":
		suggestion = fmt.Sprintf("Type %s must be convertible to string. Consider implementing a Show instance.", typ)
	default:
		suggestion = fmt.Sprintf("Type %s needs an instance of type class %s.", typ, className)
	}

	return &TypeCheckError{
		Kind:       UnsolvedConstraintError,
		Path:       path,
		Message:    fmt.Sprintf("unsolved type class constraint: %s[%s]", className, typ),
		Suggestion: suggestion,
	}
}

// ErrorList represents multiple type errors
type ErrorList []*TypeCheckError

func (e ErrorList) Error() string {
	if len(e) == 0 {
		return "no errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d type errors:", len(e)))
	for i, err := range e {
		parts = append(parts, fmt.Sprintf("\n[%d] %s", i+1, err.Error()))
	}
	return strings.Join(parts, "\n")
}

// Record-specific error codes (M-R5 Day 3.3)
const (
	TC_REC_001 = "TC_REC_001" // Missing field
	TC_REC_002 = "TC_REC_002" // Duplicate field in literal
	TC_REC_003 = "TC_REC_003" // Row occurs check
	TC_REC_004 = "TC_REC_004" // Field type mismatch
)

// NewMissingFieldError creates a TC_REC_001 error
func NewMissingFieldError(field string, recordType Type, position string) *TypeCheckError {
	// Get available fields for suggestion
	availableFields := getRecordFields(recordType)
	sort.Strings(availableFields)

	msg := fmt.Sprintf("Field '%s' not found in record", field)
	if len(availableFields) > 0 {
		msg += fmt.Sprintf(". Available fields: %s", strings.Join(availableFields, ", "))
	}

	return &TypeCheckError{
		Kind:       MissingFieldError,
		Position:   position,
		Message:    fmt.Sprintf("%s: %s", TC_REC_001, msg),
		Actual:     recordType,
		Suggestion: fmt.Sprintf("Check for typos. Valid fields: %s", strings.Join(availableFields, ", ")),
	}
}

// NewDuplicateFieldError creates a TC_REC_002 error
func NewDuplicateFieldError(field string, pos1, pos2 string) *TypeCheckError {
	return &TypeCheckError{
		Kind:       TypeErrorKind("duplicate_field"),
		Position:   pos2,
		Message:    fmt.Sprintf("%s: Duplicate field '%s' in record literal (first defined at %s)", TC_REC_002, field, pos1),
		Suggestion: "Remove the duplicate field definition",
	}
}

// NewRowOccursError creates a TC_REC_003 error
func NewRowOccursError(rowVar string, inType Type, position string) *TypeCheckError {
	return &TypeCheckError{
		Kind:       OccursCheckError,
		Position:   position,
		Message:    fmt.Sprintf("%s: Row variable '%s' occurs in %s (infinite type)", TC_REC_003, rowVar, inType.String()),
		Actual:     inType,
		Suggestion: "This would create an infinite type. Check your type annotations.",
	}
}

// NewFieldTypeMismatchError creates a TC_REC_004 error
func NewFieldTypeMismatchError(field string, expected, actual Type, position string) *TypeCheckError {
	return &TypeCheckError{
		Kind:     TypeMismatchError,
		Position: position,
		Message:  fmt.Sprintf("%s: Field '%s' type mismatch", TC_REC_004, field),
		Expected: expected,
		Actual:   actual,
		Suggestion: fmt.Sprintf("Field '%s' expects %s, but got %s",
			field, expected.String(), actual.String()),
	}
}

// Helper: Get all field names from a record type
func getRecordFields(t Type) []string {
	switch r := t.(type) {
	case *TRecord:
		fields := make([]string, 0, len(r.Fields))
		for name := range r.Fields {
			fields = append(fields, name)
		}
		return fields
	case *TRecordOpen:
		fields := make([]string, 0, len(r.Fields))
		for name := range r.Fields {
			fields = append(fields, name)
		}
		return fields
	case *TRecord2:
		if r.Row != nil {
			fields := make([]string, 0, len(r.Row.Labels))
			for name := range r.Row.Labels {
				fields = append(fields, name)
			}
			return fields
		}
		return []string{}
	default:
		return []string{}
	}
}
