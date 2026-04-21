// Package types provides cycle detection for type definitions.
//
// This module detects cyclic type references at the AST level,
// working with type declarations before full type resolution.
// It handles both simple recursive types (Person → Person) and
// generic recursive ADTs (List[a] → List[a]).
package types

import (
	"github.com/sunholo-data/ailang/internal/ast"
)

// CycleKind classifies whether a cycle is expected or suspicious.
type CycleKind string

const (
	// CycleExpected marks cycles that are standard recursive ADT patterns.
	// These include List, Tree, and other common recursive type names,
	// as well as anything in the stdlib.
	CycleExpected CycleKind = "expected"

	// CycleSuspicious marks cycles that may cause traversal issues
	// if not handled with cycle-safe code.
	CycleSuspicious CycleKind = "suspicious"
)

// CycleInfo holds information about a detected cyclic type reference.
type CycleInfo struct {
	Kind     CycleKind // Expected or suspicious
	TypeName string    // The type that has a cycle
	Path     []string  // The path through fields that creates the cycle
	Depth    int       // Number of nodes in the cycle
	Note     string    // Optional explanation
}

// DetectCycles analyzes AST type declarations and returns any cyclic references.
// It handles both direct self-references and references through type applications.
func DetectCycles(decls []ast.Node, filename string) []CycleInfo {
	var cycles []CycleInfo

	for _, decl := range decls {
		if td, ok := decl.(*ast.TypeDecl); ok {
			detectTypeDeclCycles(td, filename, &cycles)
		}
	}

	return cycles
}

// detectTypeDeclCycles checks a single type declaration for cycles.
func detectTypeDeclCycles(td *ast.TypeDecl, filename string, cycles *[]CycleInfo) {
	typeName := td.Name

	// Collect all type references in the definition
	refs := collectTypeReferences(td.Definition, typeName)

	// Check for self-references (direct cycles)
	for _, ref := range refs {
		if ref.baseName == typeName {
			kind := classifyCycleKind(typeName, filename)
			note := ""
			if kind == CycleExpected {
				note = "Standard recursive ADT pattern"
			}

			*cycles = append(*cycles, CycleInfo{
				Kind:     kind,
				TypeName: typeName,
				Path:     buildCyclePath(typeName, ref),
				Depth:    len(ref.fieldPath) + 1,
				Note:     note,
			})
			break // Only report once per type
		}
	}
}

// typeReference holds information about a type reference found in a type definition.
type typeReference struct {
	baseName  string   // The base type name (without type parameters)
	fullName  string   // The full type expression as string
	fieldPath []string // The path of fields to reach this reference
}

// collectTypeReferences extracts all type name references from a type definition.
// The definingType parameter is the name of the type being defined, used to detect
// cycles through normalized types (e.g., List[a] normalized to [a]).
func collectTypeReferences(typeDef ast.TypeDef, definingType string) []typeReference {
	var refs []typeReference
	collectFromTypeDef(typeDef, nil, definingType, &refs)
	return refs
}

// collectFromTypeDef recursively collects type references from a type definition.
func collectFromTypeDef(typeDef ast.TypeDef, fieldPath []string, definingType string, refs *[]typeReference) {
	switch td := typeDef.(type) {
	case *ast.AlgebraicType:
		for _, ctor := range td.Constructors {
			for _, field := range ctor.Fields {
				path := append(fieldPath, ctor.Name+"("+field.Name+")")
				collectFromType(field.Type, path, definingType, refs)
			}
		}
	case *ast.RecordType:
		for _, field := range td.Fields {
			path := append(fieldPath, field.Name)
			collectFromType(field.Type, path, definingType, refs)
		}
	case *ast.TypeAlias:
		collectFromType(td.Target, fieldPath, definingType, refs)
	}
}

// collectFromType recursively collects type references from an AST type.
// definingType is the name of the type being defined (for detecting normalized cycles).
func collectFromType(t ast.Type, fieldPath []string, definingType string, refs *[]typeReference) {
	if t == nil {
		return
	}

	switch typ := t.(type) {
	case *ast.SimpleType:
		// Direct type reference (e.g., "Person", "int")
		*refs = append(*refs, typeReference{
			baseName:  typ.Name,
			fullName:  typ.Name,
			fieldPath: copyPath(fieldPath),
		})

	case *ast.TypeApp:
		// Generic type application (e.g., "List[a]", "Tree[int]")
		*refs = append(*refs, typeReference{
			baseName:  typ.Constructor,
			fullName:  typ.String(),
			fieldPath: copyPath(fieldPath),
		})
		// Also recurse into type arguments
		for _, arg := range typ.Args {
			collectFromType(arg, fieldPath, definingType, refs)
		}

	case *ast.ListType:
		// List type [T]
		// Special case: if we're defining "List" and see [T], the parser has
		// normalized "List[T]" to "[T]". This represents a cycle back to List.
		if definingType == "List" {
			*refs = append(*refs, typeReference{
				baseName:  "List",
				fullName:  typ.String(),
				fieldPath: copyPath(fieldPath),
			})
		}
		// Also recurse into element type
		path := append(fieldPath, "[]")
		collectFromType(typ.Element, path, definingType, refs)

	case *ast.ArrayType:
		// Array type Array[T] - recurse into element type
		path := append(fieldPath, "Array[]")
		collectFromType(typ.Element, path, definingType, refs)

	case *ast.TupleType:
		// Tuple type (T1, T2) - recurse into all elements
		for i, elem := range typ.Elements {
			path := append(fieldPath, "["+string(rune('0'+i))+"]")
			collectFromType(elem, path, definingType, refs)
		}

	case *ast.FuncType:
		// Function type - recurse into params and return
		for _, param := range typ.Params {
			collectFromType(param, fieldPath, definingType, refs)
		}
		collectFromType(typ.Return, fieldPath, definingType, refs)

	case *ast.RecordType:
		// Inline record type - recurse into fields
		for _, field := range typ.Fields {
			path := append(fieldPath, "."+field.Name)
			collectFromType(field.Type, path, definingType, refs)
		}

	case *ast.TypeVar:
		// Type variable (a, b, etc.) - not a concrete type reference
		// Skip these as they don't create cycles by themselves
	}
}

// copyPath creates a copy of a field path slice.
func copyPath(path []string) []string {
	if path == nil {
		return nil
	}
	result := make([]string, len(path))
	copy(result, path)
	return result
}

// buildCyclePath constructs a readable path string for the cycle.
func buildCyclePath(typeName string, ref typeReference) []string {
	path := []string{typeName}
	path = append(path, ref.fieldPath...)
	path = append(path, ref.fullName)
	return path
}

// classifyCycleKind determines if a cycle is expected or suspicious.
func classifyCycleKind(typeName string, filename string) CycleKind {
	// Common recursive ADT names are expected
	commonRecursive := map[string]bool{
		"List": true, "Tree": true, "Node": true,
		"Expr": true, "Stmt": true, "AST": true,
		"Stream": true, "Lazy": true, "Option": true,
		"Result": true, "Either": true,
	}

	if commonRecursive[typeName] {
		return CycleExpected
	}

	// Stdlib types are expected to be recursive
	if isStdlibFile(filename) {
		return CycleExpected
	}

	return CycleSuspicious
}

// isStdlibFile checks if a filename is part of the standard library.
func isStdlibFile(filename string) bool {
	// Check common stdlib path patterns
	return containsPath(filename, "std/") ||
		containsPath(filename, "stdlib/") ||
		containsPath(filename, "prelude")
}

// containsPath checks if a filename contains a path component.
func containsPath(filename, path string) bool {
	return len(filename) >= len(path) &&
		(filename == path ||
			(len(filename) > len(path) && filename[len(filename)-len(path)-1] == '/' &&
				filename[len(filename)-len(path):] == path) ||
			containsSubstring(filename, "/"+path) ||
			containsSubstring(filename, path))
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
