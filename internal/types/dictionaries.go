package types

import (
	"fmt"
	"math"
)

// DictionaryRegistry manages type class dictionaries for all instances.
// Keys are in the format: "namespace::ClassName::TypeNF::method"
// Example: "prelude::Num::Int::add"
type DictionaryRegistry struct {
	dictionaries map[string]DictionaryEntry
}

// DictionaryEntry represents a method implementation in a dictionary
type DictionaryEntry struct {
	ClassName string
	TypeName  string  // Normalized type name
	Method    string
	Impl      interface{} // The actual implementation
}

// NewDictionaryRegistry creates a new registry with built-in instances
func NewDictionaryRegistry() *DictionaryRegistry {
	r := &DictionaryRegistry{
		dictionaries: make(map[string]DictionaryEntry),
	}
	r.registerBuiltins()
	return r
}

// Register adds a dictionary entry to the registry
func (r *DictionaryRegistry) Register(namespace, className, typeName, method string, impl interface{}) {
	key := MakeDictionaryKey(namespace, className, &TCon{Name: typeName}, method)
	r.dictionaries[key] = DictionaryEntry{
		ClassName: className,
		TypeName:  NormalizeTypeName(&TCon{Name: typeName}),
		Method:    method,
		Impl:      impl,
	}
}

// RegisterInstance registers a complete type class instance
func (r *DictionaryRegistry) RegisterInstance(key string, dict interface{}) {
	// This is a simplified registration for REPL use
	// In practice, would decompose the dict into individual method entries
}

// Lookup retrieves a dictionary entry by key
func (r *DictionaryRegistry) Lookup(key string) (DictionaryEntry, bool) {
	entry, ok := r.dictionaries[key]
	return entry, ok
}

// LookupMethod retrieves a specific method implementation
func (r *DictionaryRegistry) LookupMethod(namespace, className string, typ Type, method string) (interface{}, bool) {
	key := MakeDictionaryKey(namespace, className, typ, method)
	entry, ok := r.dictionaries[key]
	if !ok {
		return nil, false
	}
	return entry.Impl, true
}

// registerBuiltins registers all built-in type class instances
func (r *DictionaryRegistry) registerBuiltins() {
	// Num instances for Int
	r.registerNumInt()
	
	// Num instances for Float (law-compliant)
	r.registerNumFloat()
	
	// Eq instances
	r.registerEqInt()
	r.registerEqFloat()
	r.registerEqBool()
	r.registerEqString()
	
	// Ord instances
	r.registerOrdInt()
	r.registerOrdFloat()
	r.registerOrdString()
}

// Num instance for Int
func (r *DictionaryRegistry) registerNumInt() {
	ns := "prelude"
	
	// add: Int -> Int -> Int
	r.Register(ns, "Num", "int", "add", func(x, y int) int {
		return x + y
	})
	
	// sub: Int -> Int -> Int
	r.Register(ns, "Num", "int", "sub", func(x, y int) int {
		return x - y
	})
	
	// mul: Int -> Int -> Int
	r.Register(ns, "Num", "int", "mul", func(x, y int) int {
		return x * y
	})
	
	// div: Int -> Int -> Int (integer division)
	r.Register(ns, "Num", "int", "div", func(x, y int) int {
		if y == 0 {
			panic("division by zero")
		}
		return x / y
	})
	
	// neg: Int -> Int (unary minus)
	r.Register(ns, "Num", "int", "neg", func(x int) int {
		return -x
	})
	
	// abs: Int -> Int
	r.Register(ns, "Num", "int", "abs", func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	})
	
	// fromInt: Int -> Int (identity for Int)
	r.Register(ns, "Num", "int", "fromInt", func(x int) int {
		return x
	})
}

// Num instance for Float (law-compliant)
func (r *DictionaryRegistry) registerNumFloat() {
	ns := "prelude"
	
	// add: Float -> Float -> Float
	r.Register(ns, "Num", "float", "add", func(x, y float64) float64 {
		return x + y
	})
	
	// sub: Float -> Float -> Float
	r.Register(ns, "Num", "float", "sub", func(x, y float64) float64 {
		return x - y
	})
	
	// mul: Float -> Float -> Float
	r.Register(ns, "Num", "float", "mul", func(x, y float64) float64 {
		return x * y
	})
	
	// div: Float -> Float -> Float
	r.Register(ns, "Num", "float", "div", func(x, y float64) float64 {
		return x / y // IEEE 754 handles ±Inf and NaN
	})
	
	// neg: Float -> Float (unary minus)
	r.Register(ns, "Num", "float", "neg", func(x float64) float64 {
		return -x
	})
	
	// abs: Float -> Float
	r.Register(ns, "Num", "float", "abs", func(x float64) float64 {
		return math.Abs(x)
	})
	
	// fromInt: Int -> Float
	r.Register(ns, "Num", "float", "fromInt", func(x int) float64 {
		return float64(x)
	})
}

// Eq instance for Int
func (r *DictionaryRegistry) registerEqInt() {
	ns := "prelude"
	
	// eq: Int -> Int -> Bool
	r.Register(ns, "Eq", "int", "eq", func(x, y int) bool {
		return x == y
	})
	
	// neq: Int -> Int -> Bool
	r.Register(ns, "Eq", "int", "neq", func(x, y int) bool {
		return x != y
	})
}

// Eq instance for Float (law-compliant: NaN == NaN is true for reflexivity)
func (r *DictionaryRegistry) registerEqFloat() {
	ns := "prelude"
	
	// eq: Float -> Float -> Bool
	// IMPORTANT: This implementation makes NaN == NaN return true
	// to satisfy the reflexivity law of Eq type class
	r.Register(ns, "Eq", "float", "eq", func(x, y float64) bool {
		// Reflexive equality: NaN == NaN is true
		if math.IsNaN(x) && math.IsNaN(y) {
			return true
		}
		// Standard IEEE 754 equality for non-NaN values
		return x == y
	})
	
	// neq: Float -> Float -> Bool
	r.Register(ns, "Eq", "float", "neq", func(x, y float64) bool {
		// Consistent with our eq implementation
		if math.IsNaN(x) && math.IsNaN(y) {
			return false
		}
		return x != y
	})
}

// Eq instance for Bool
func (r *DictionaryRegistry) registerEqBool() {
	ns := "prelude"
	
	// eq: Bool -> Bool -> Bool
	r.Register(ns, "Eq", "bool", "eq", func(x, y bool) bool {
		return x == y
	})
	
	// neq: Bool -> Bool -> Bool
	r.Register(ns, "Eq", "bool", "neq", func(x, y bool) bool {
		return x != y
	})
}

// Eq instance for String
func (r *DictionaryRegistry) registerEqString() {
	ns := "prelude"
	
	// eq: String -> String -> Bool
	r.Register(ns, "Eq", "string", "eq", func(x, y string) bool {
		return x == y
	})
	
	// neq: String -> String -> Bool
	r.Register(ns, "Eq", "string", "neq", func(x, y string) bool {
		return x != y
	})
}

// Ord instance for Int
func (r *DictionaryRegistry) registerOrdInt() {
	ns := "prelude"
	
	// lt: Int -> Int -> Bool
	r.Register(ns, "Ord", "int", "lt", func(x, y int) bool {
		return x < y
	})
	
	// lte: Int -> Int -> Bool
	r.Register(ns, "Ord", "int", "lte", func(x, y int) bool {
		return x <= y
	})
	
	// gt: Int -> Int -> Bool
	r.Register(ns, "Ord", "int", "gt", func(x, y int) bool {
		return x > y
	})
	
	// gte: Int -> Int -> Bool
	r.Register(ns, "Ord", "int", "gte", func(x, y int) bool {
		return x >= y
	})
	
	// min: Int -> Int -> Int
	r.Register(ns, "Ord", "int", "min", func(x, y int) int {
		if x < y {
			return x
		}
		return y
	})
	
	// max: Int -> Int -> Int
	r.Register(ns, "Ord", "int", "max", func(x, y int) int {
		if x > y {
			return x
		}
		return y
	})
}

// Ord instance for Float (law-compliant: total ordering with NaN)
func (r *DictionaryRegistry) registerOrdFloat() {
	ns := "prelude"
	
	// For total ordering, we define: -Inf < finite < +Inf < NaN
	// This ensures all values are comparable and laws hold
	
	// compareFloat provides total ordering for floats
	compareFloat := func(x, y float64) int {
		// NaN is greatest
		xNaN := math.IsNaN(x)
		yNaN := math.IsNaN(y)
		if xNaN && yNaN {
			return 0 // NaN == NaN for total ordering
		}
		if xNaN {
			return 1 // x > y (NaN is greatest)
		}
		if yNaN {
			return -1 // x < y (NaN is greatest)
		}
		
		// Standard comparison for non-NaN
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
		return 0
	}
	
	// lt: Float -> Float -> Bool
	r.Register(ns, "Ord", "float", "lt", func(x, y float64) bool {
		return compareFloat(x, y) < 0
	})
	
	// lte: Float -> Float -> Bool
	r.Register(ns, "Ord", "float", "lte", func(x, y float64) bool {
		return compareFloat(x, y) <= 0
	})
	
	// gt: Float -> Float -> Bool
	r.Register(ns, "Ord", "float", "gt", func(x, y float64) bool {
		return compareFloat(x, y) > 0
	})
	
	// gte: Float -> Float -> Bool
	r.Register(ns, "Ord", "float", "gte", func(x, y float64) bool {
		return compareFloat(x, y) >= 0
	})
	
	// min: Float -> Float -> Float
	r.Register(ns, "Ord", "float", "min", func(x, y float64) float64 {
		if compareFloat(x, y) < 0 {
			return x
		}
		return y
	})
	
	// max: Float -> Float -> Float
	r.Register(ns, "Ord", "float", "max", func(x, y float64) float64 {
		if compareFloat(x, y) > 0 {
			return x
		}
		return y
	})
}

// Ord instance for String
func (r *DictionaryRegistry) registerOrdString() {
	ns := "prelude"
	
	// lt: String -> String -> Bool
	r.Register(ns, "Ord", "string", "lt", func(x, y string) bool {
		return x < y
	})
	
	// lte: String -> String -> Bool
	r.Register(ns, "Ord", "string", "lte", func(x, y string) bool {
		return x <= y
	})
	
	// gt: String -> String -> Bool
	r.Register(ns, "Ord", "string", "gt", func(x, y string) bool {
		return x > y
	})
	
	// gte: String -> String -> Bool
	r.Register(ns, "Ord", "string", "gte", func(x, y string) bool {
		return x >= y
	})
	
	// min: String -> String -> String
	r.Register(ns, "Ord", "string", "min", func(x, y string) string {
		if x < y {
			return x
		}
		return y
	})
	
	// max: String -> String -> String
	r.Register(ns, "Ord", "string", "max", func(x, y string) string {
		if x > y {
			return x
		}
		return y
	})
}

// ValidateRegistry checks that all required methods are present for each type class instance
func (r *DictionaryRegistry) ValidateRegistry() error {
	// Define required methods for each type class
	requiredMethods := map[string][]string{
		"Num": {"add", "sub", "mul", "div", "neg", "abs", "fromInt"},
		"Eq":  {"eq", "neq"},
		"Ord": {"lt", "lte", "gt", "gte", "min", "max"},
	}
	
	// Track which (class, type) pairs we've seen
	instances := make(map[string]map[string]bool)
	
	// Scan all registered dictionaries
	for key := range r.dictionaries {
		namespace, className, typeNF, method, err := ParseDictionaryKey(key)
		if err != nil {
			return fmt.Errorf("invalid dictionary key %s: %w", key, err)
		}
		
		// Skip non-prelude for now
		if namespace != "prelude" {
			continue
		}
		
		// Track this instance
		if instances[className] == nil {
			instances[className] = make(map[string]bool)
		}
		instanceKey := fmt.Sprintf("%s::%s", className, typeNF)
		instances[className][instanceKey] = true
		
		// Check if this is a valid method for the class
		validMethods, ok := requiredMethods[className]
		if !ok {
			continue // Unknown class, skip validation
		}
		
		found := false
		for _, m := range validMethods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid method %s for class %s", method, className)
		}
	}
	
	// Now verify each instance has all required methods
	for className, typeInstances := range instances {
		requiredList, ok := requiredMethods[className]
		if !ok {
			continue
		}
		
		for instanceKey := range typeInstances {
			for _, method := range requiredList {
				// Reconstruct the full key
				parts := []string{"prelude", instanceKey, method}
				key := fmt.Sprintf("%s::%s", parts[0], parts[1])
				if method != "" {
					key = fmt.Sprintf("%s::%s", key, method)
				}
				
				if _, exists := r.dictionaries[key]; !exists {
					return fmt.Errorf("missing method %s for instance %s", method, instanceKey)
				}
			}
		}
	}
	
	return nil
}