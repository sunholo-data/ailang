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
		message = fmt.Sprintf("missing required effects: {%s}", strings.Join(missing, ", "))
		suggestions = append(suggestions, fmt.Sprintf("Consider adding capability %s", strings.Join(missing, ", ")))
	}

	if len(extra) > 0 {
		if len(missing) > 0 {
			message += fmt.Sprintf("; has extra effects: {%s}", strings.Join(extra, ", "))
		} else {
			message = fmt.Sprintf("has extra effects: {%s}", strings.Join(extra, ", "))
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
