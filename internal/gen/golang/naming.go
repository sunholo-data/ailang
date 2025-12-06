// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"strings"
	"unicode"
)

// ToPascalCase converts snake_case or camelCase to PascalCase.
// Used for exported Go identifiers.
//
// Examples:
//
//	"hello_world" -> "HelloWorld"
//	"helloWorld"  -> "HelloWorld"
//	"hello"       -> "Hello"
//	"_private"    -> "Private"
//	"_"           -> "_" (special case for wildcard)
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s))

	capitalizeNext := true
	for _, r := range s {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	// Handle edge case where input was all underscores (e.g., "_")
	if result.Len() == 0 {
		return "_"
	}

	return result.String()
}

// ToCamelCase converts snake_case or PascalCase to camelCase.
// Used for package-private Go identifiers.
//
// Examples:
//
//	"hello_world" -> "helloWorld"
//	"HelloWorld"  -> "helloWorld"
//	"hello"       -> "hello"
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ToGoTypeName converts an AILANG type name to a valid Go type name.
// AILANG types are already PascalCase, so this is mostly a passthrough
// with validation. Also strips $ prefix from type variables.
//
// Examples:
//
//	"Tree"       -> "Tree"
//	"FrameInput" -> "FrameInput"
//	"$a"         -> "A" (type variable)
func ToGoTypeName(s string) string {
	// Strip $ prefix from type variables
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}
	if s == "" {
		return ""
	}
	// Sanitize invalid characters
	s = SanitizeGoIdentifier(s)
	// Ensure first character is uppercase
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ToGoFieldName converts an AILANG record field name to a Go struct field name.
// AILANG uses snake_case, Go uses PascalCase for exported fields.
//
// Examples:
//
//	"frame_input" -> "FrameInput"
//	"x"           -> "X"
//	"position"    -> "Position"
func ToGoFieldName(s string) string {
	return ToPascalCase(s)
}

// ToGoFuncName converts an AILANG function name to a Go function name.
// Exported functions use PascalCase, private functions use camelCase.
// Also strips $ prefix from type variables.
//
// Examples:
//
//	"step", true        -> "Step"
//	"step", false       -> "step"
//	"init_world", true  -> "InitWorld"
//	"init_world", false -> "initWorld"
func ToGoFuncName(s string, exported bool) string {
	// Strip $ prefix from type variables
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}
	// Sanitize invalid characters
	s = SanitizeGoIdentifier(s)
	if exported {
		return ToPascalCase(s)
	}
	return ToCamelCase(s)
}

// ToGoVarName converts an AILANG variable name to a Go variable name.
// Variables are always package-private (camelCase).
// Also sanitizes invalid characters like $ (from type variables).
// Go reserved keywords are escaped with underscore suffix.
//
// Examples:
//
//	"world"       -> "world"
//	"frame_input" -> "frameInput"
//	"$a"          -> "a"   (type variable)
//	"$foo_bar"    -> "fooBar"
//	"default"     -> "default_" (escaped Go keyword)
func ToGoVarName(s string) string {
	// Strip $ prefix from type variables
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}
	// Sanitize any remaining invalid characters
	s = SanitizeGoIdentifier(s)
	result := ToCamelCase(s)
	// Escape Go reserved keywords
	return EscapeKeyword(result)
}

// ToKindConstName generates a kind constant name for sum type discriminators.
// Avoids double-prefixing if the variant name already starts with the type name.
//
// Examples:
//
//	"Tree", "Leaf" -> "TreeKindLeaf"
//	"Tree", "Node" -> "TreeKindNode"
//	"Selection", "SelectionTile" -> "SelectionKindTile" (not "SelectionKindSelectionTile")
func ToKindConstName(typeName, variantName string) string {
	pascalType := ToPascalCase(typeName)
	pascalVariant := ToPascalCase(variantName)

	// Strip type prefix from variant if it already starts with type name
	variantSuffix := pascalVariant
	if strings.HasPrefix(pascalVariant, pascalType) {
		variantSuffix = pascalVariant[len(pascalType):]
		// Handle edge case where variant name equals type name exactly
		if variantSuffix == "" {
			variantSuffix = pascalVariant
		}
	}
	return pascalType + "Kind" + variantSuffix
}

// ToKindTypeName generates a kind type name for sum type discriminators.
//
// Examples:
//
//	"Tree" -> "TreeKind"
func ToKindTypeName(typeName string) string {
	return ToPascalCase(typeName) + "Kind"
}

// ToVariantStructName generates a variant struct name for sum types.
// Avoids double-prefixing if the variant name already starts with the type name.
//
// Examples:
//
//	"Tree", "Leaf" -> "TreeLeaf"
//	"Tree", "Node" -> "TreeNode"
//	"Selection", "SelectionTile" -> "SelectionTile" (not "SelectionSelectionTile")
//	"Selection", "None" -> "SelectionNone"
func ToVariantStructName(typeName, variantName string) string {
	pascalType := ToPascalCase(typeName)
	pascalVariant := ToPascalCase(variantName)

	// Avoid double-prefix if variant already starts with type name
	if strings.HasPrefix(pascalVariant, pascalType) {
		return pascalVariant
	}
	return pascalType + pascalVariant
}

// SanitizeGoIdentifier ensures a string is a valid Go identifier.
// Replaces invalid characters with underscores and ensures it doesn't
// start with a digit.
func SanitizeGoIdentifier(s string) string {
	if s == "" {
		return "_"
	}

	var result strings.Builder
	result.Grow(len(s))

	for i, r := range s {
		if i == 0 && unicode.IsDigit(r) {
			result.WriteRune('_')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	return result.String()
}

// IsGoKeyword returns true if the string is a Go reserved keyword.
func IsGoKeyword(s string) bool {
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true,
		"continue": true, "default": true, "defer": true, "else": true,
		"fallthrough": true, "for": true, "func": true, "go": true,
		"goto": true, "if": true, "import": true, "interface": true,
		"map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true,
		"var": true,
	}
	return keywords[s]
}

// EscapeKeyword adds an underscore suffix if the string is a Go keyword.
func EscapeKeyword(s string) string {
	if IsGoKeyword(s) {
		return s + "_"
	}
	return s
}
