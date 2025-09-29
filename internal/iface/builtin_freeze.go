// Package iface provides the frozen $builtin interface
package iface

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BuiltinInterface represents the $builtin module interface
type BuiltinInterface struct {
	Module  string                   `json:"module"`
	Exports map[string]BuiltinExport `json:"exports"`
	Digest  string                   `json:"digest"`
}

// BuiltinExport represents a single builtin export
type BuiltinExport struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Arity    int    `json:"arity"`
	Category string `json:"category"` // "arithmetic", "comparison", "string", "io", etc.
}

// FrozenBuiltinInterface returns the deterministic $builtin interface
// This is the canonical source of truth for all builtins
func FrozenBuiltinInterface() *BuiltinInterface {
	exports := map[string]BuiltinExport{
		// Arithmetic operators (Int)
		"add_Int": {Name: "add_Int", Type: "Int -> Int -> Int", Arity: 2, Category: "arithmetic"},
		"sub_Int": {Name: "sub_Int", Type: "Int -> Int -> Int", Arity: 2, Category: "arithmetic"},
		"mul_Int": {Name: "mul_Int", Type: "Int -> Int -> Int", Arity: 2, Category: "arithmetic"},
		"div_Int": {Name: "div_Int", Type: "Int -> Int -> Int", Arity: 2, Category: "arithmetic"},
		"mod_Int": {Name: "mod_Int", Type: "Int -> Int -> Int", Arity: 2, Category: "arithmetic"},
		"neg_Int": {Name: "neg_Int", Type: "Int -> Int", Arity: 1, Category: "arithmetic"},

		// Arithmetic operators (Float)
		"add_Float": {Name: "add_Float", Type: "Float -> Float -> Float", Arity: 2, Category: "arithmetic"},
		"sub_Float": {Name: "sub_Float", Type: "Float -> Float -> Float", Arity: 2, Category: "arithmetic"},
		"mul_Float": {Name: "mul_Float", Type: "Float -> Float -> Float", Arity: 2, Category: "arithmetic"},
		"div_Float": {Name: "div_Float", Type: "Float -> Float -> Float", Arity: 2, Category: "arithmetic"},
		"neg_Float": {Name: "neg_Float", Type: "Float -> Float", Arity: 1, Category: "arithmetic"},

		// Comparison operators (Int)
		"eq_Int":  {Name: "eq_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},
		"neq_Int": {Name: "neq_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},
		"lt_Int":  {Name: "lt_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},
		"lte_Int": {Name: "lte_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},
		"gt_Int":  {Name: "gt_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},
		"gte_Int": {Name: "gte_Int", Type: "Int -> Int -> Bool", Arity: 2, Category: "comparison"},

		// Comparison operators (Float)
		"eq_Float":  {Name: "eq_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},
		"neq_Float": {Name: "neq_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},
		"lt_Float":  {Name: "lt_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},
		"lte_Float": {Name: "lte_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},
		"gt_Float":  {Name: "gt_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},
		"gte_Float": {Name: "gte_Float", Type: "Float -> Float -> Bool", Arity: 2, Category: "comparison"},

		// Comparison operators (String)
		"eq_String":  {Name: "eq_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},
		"neq_String": {Name: "neq_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},
		"lt_String":  {Name: "lt_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},
		"lte_String": {Name: "lte_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},
		"gt_String":  {Name: "gt_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},
		"gte_String": {Name: "gte_String", Type: "String -> String -> Bool", Arity: 2, Category: "comparison"},

		// Comparison operators (Bool)
		"eq_Bool":  {Name: "eq_Bool", Type: "Bool -> Bool -> Bool", Arity: 2, Category: "comparison"},
		"neq_Bool": {Name: "neq_Bool", Type: "Bool -> Bool -> Bool", Arity: 2, Category: "comparison"},

		// String operations
		"concat_String": {Name: "concat_String", Type: "String -> String -> String", Arity: 2, Category: "string"},

		// Logical operations (Note: && and || lower to if-then-else, not builtins)
		"not_Bool": {Name: "not_Bool", Type: "Bool -> Bool", Arity: 1, Category: "logical"},

		// Show functions (for debugging)
		"show_Int":    {Name: "show_Int", Type: "Int -> String", Arity: 1, Category: "show"},
		"show_Float":  {Name: "show_Float", Type: "Float -> String", Arity: 1, Category: "show"},
		"show_String": {Name: "show_String", Type: "String -> String", Arity: 1, Category: "show"},
		"show_Bool":   {Name: "show_Bool", Type: "Bool -> String", Arity: 1, Category: "show"},

		// IO operations
		"print": {Name: "print", Type: "String -> ()", Arity: 1, Category: "io"},
	}

	iface := &BuiltinInterface{
		Module:  "$builtin",
		Exports: exports,
	}

	// Compute deterministic digest
	iface.Digest = computeBuiltinDigest(iface)

	return iface
}

// computeBuiltinDigest computes a deterministic SHA256 digest of the interface
func computeBuiltinDigest(iface *BuiltinInterface) string {
	// Sort exports by name for deterministic ordering
	var names []string
	for name := range iface.Exports {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build canonical JSON representation
	var parts []string
	for _, name := range names {
		export := iface.Exports[name]
		part := fmt.Sprintf(`"%s":{"type":"%s","arity":%d,"category":"%s"}`,
			name, export.Type, export.Arity, export.Category)
		parts = append(parts, part)
	}

	canonical := fmt.Sprintf(`{"module":"$builtin","exports":{%s}}`, strings.Join(parts, ","))

	// Compute SHA256
	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash)
}

// ValidateBuiltin checks if a builtin reference is valid
func ValidateBuiltin(name string) error {
	frozen := FrozenBuiltinInterface()
	if _, ok := frozen.Exports[name]; !ok {
		return fmt.Errorf("LNK_BUILTIN404: Unknown builtin '%s'", name)
	}
	return nil
}

// GetBuiltinType returns the type signature of a builtin
func GetBuiltinType(name string) (string, error) {
	frozen := FrozenBuiltinInterface()
	if export, ok := frozen.Exports[name]; ok {
		return export.Type, nil
	}
	return "", fmt.Errorf("LNK_BUILTIN404: Unknown builtin '%s'", name)
}

// GetBuiltinArity returns the arity of a builtin
func GetBuiltinArity(name string) (int, error) {
	frozen := FrozenBuiltinInterface()
	if export, ok := frozen.Exports[name]; ok {
		return export.Arity, nil
	}
	return 0, fmt.Errorf("LNK_BUILTIN404: Unknown builtin '%s'", name)
}

// DumpBuiltinInterface exports the builtin interface as JSON
func DumpBuiltinInterface() ([]byte, error) {
	frozen := FrozenBuiltinInterface()
	return json.MarshalIndent(frozen, "", "  ")
}
