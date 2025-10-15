package types

import (
	"fmt"
	"sort"
	"strings"
)

// Dict represents method implementations for a type class instance
// For now, we store method names - actual implementations will be in evaluator
type Dict = map[string]string // Method name -> implementation identifier

// ClassInstance represents a type class instance with its methods
type ClassInstance struct {
	ClassName string   // e.g., "Num", "Ord", "Eq"
	TypeHead  Type     // Monomorphic type for v1 (TInt, TFloat, etc.)
	Dict      Dict     // Method implementations
	Super     []string // Superclasses this instance provides (e.g., Ord provides Eq)
}

// InstanceEnv manages type class instances with coherence checking
type InstanceEnv struct {
	instances map[string]*ClassInstance // Key: "ClassName:NormalizedType"
	defaults  map[string]Type           // Default types for ambiguous literals
}

// NewInstanceEnv creates a new empty instance environment
func NewInstanceEnv() *InstanceEnv {
	return &InstanceEnv{
		instances: make(map[string]*ClassInstance),
		defaults:  make(map[string]Type),
	}
}

// Add adds an instance to the environment with coherence checking
func (env *InstanceEnv) Add(inst *ClassInstance) error {
	key := canonicalKey(inst.ClassName, inst.TypeHead)
	if _, exists := env.instances[key]; exists {
		return fmt.Errorf("overlapping instance: %s[%s]", inst.ClassName, inst.TypeHead)
	}
	env.instances[key] = inst
	return nil
}

// Lookup finds an instance, including superclass derivation
func (env *InstanceEnv) Lookup(class string, typ Type) (*ClassInstance, error) {
	// Direct lookup
	key := canonicalKey(class, typ)
	if inst, ok := env.instances[key]; ok {
		return inst, nil
	}

	// Superclass provision: Ord provides Eq
	if class == "Eq" {
		ordKey := canonicalKey("Ord", typ)
		if ordInst, ok := env.instances[ordKey]; ok {
			return deriveEqFromOrd(ordInst), nil
		}
	}

	return nil, &MissingInstanceError{
		Class: class,
		Type:  typ,
		Hint:  "Import std/prelude or define instance",
	}
}

// DefaultFor returns the default type for a class (for numeric literal defaulting)
func (env *InstanceEnv) DefaultFor(class string) Type {
	if def, ok := env.defaults[class]; ok {
		return def
	}
	return nil // Require annotation
}

// SetDefault sets the default type for a class
func (env *InstanceEnv) SetDefault(class string, typ Type) {
	env.defaults[class] = typ
}

// GetDefault gets the default type for a class
func (env *InstanceEnv) GetDefault(class string) Type {
	return env.defaults[class]
}

// canonicalKey creates a normalized key for instance lookup
// NOTE: InstanceEnv keys are class+type (no namespace, no method). Reuse NormalizeTypeName.
func canonicalKey(className string, typ Type) string {
	// Double-colon to match the rest of the system's visual convention for segments.
	return fmt.Sprintf("%s::%s", className, NormalizeTypeName(typ))
}

// deriveEqFromOrd creates an Eq instance from an Ord instance
// Uses the lawful definition: eq(x,y) = ¬lt(x,y) ∧ ¬lt(y,x)
func deriveEqFromOrd(ord *ClassInstance) *ClassInstance {
	// Create a derived Eq instance using Ord's methods
	return &ClassInstance{
		ClassName: "Eq",
		TypeHead:  ord.TypeHead,
		Dict: Dict{
			"eq":  fmt.Sprintf("derived_eq_from_ord_%s", NormalizeTypeName(ord.TypeHead)),
			"neq": fmt.Sprintf("derived_neq_from_ord_%s", NormalizeTypeName(ord.TypeHead)),
		},
	}
}

// MissingInstanceError represents a missing type class instance
type MissingInstanceError struct {
	Class string
	Type  Type
	Hint  string
}

func (e *MissingInstanceError) Error() string {
	msg := fmt.Sprintf("No instance for %s[%s] in scope", e.Class, e.Type)
	if e.Hint != "" {
		msg += ". " + e.Hint
	}
	return msg
}

// LoadBuiltinInstances creates the standard set of built-in instances
// These would normally come from "import std/prelude"
func LoadBuiltinInstances() *InstanceEnv {
	env := NewInstanceEnv()

	// Add all built-in instances
	for _, inst := range builtinInstances() {
		if err := env.Add(inst); err != nil {
			panic(fmt.Sprintf("Failed to add built-in instance: %v", err))
		}
	}

	// Set default types for numeric literals
	env.SetDefault("Num", TInt)
	env.SetDefault("Fractional", TFloat)

	return env
}

// builtinInstances returns all built-in type class instances
func builtinInstances() []*ClassInstance {
	return []*ClassInstance{
		// Num[Int]
		{
			ClassName: "Num",
			TypeHead:  TInt,
			Dict: Dict{
				"add": "builtin_num_int_add",
				"sub": "builtin_num_int_sub",
				"mul": "builtin_num_int_mul",
				"div": "builtin_num_int_div",
			},
		},

		// Num[Float]
		{
			ClassName: "Num",
			TypeHead:  TFloat,
			Dict: Dict{
				"add": "builtin_num_float_add",
				"sub": "builtin_num_float_sub",
				"mul": "builtin_num_float_mul",
				"div": "builtin_num_float_div",
			},
		},

		// Eq[Int]
		{
			ClassName: "Eq",
			TypeHead:  TInt,
			Dict: Dict{
				"eq":  "builtin_eq_int_eq",
				"neq": "builtin_eq_int_neq",
			},
		},

		// Eq[Float] - Lawful equivalence relation
		{
			ClassName: "Eq",
			TypeHead:  TFloat,
			Dict: Dict{
				"eq":  "builtin_eq_float_eq", // Lawful: NaN==NaN, -0==+0
				"neq": "builtin_eq_float_neq",
			},
		},

		// Eq[String]
		{
			ClassName: "Eq",
			TypeHead:  TString,
			Dict: Dict{
				"eq":  "builtin_eq_string_eq",
				"neq": "builtin_eq_string_neq",
			},
		},

		// Eq[Bool]
		{
			ClassName: "Eq",
			TypeHead:  TBool,
			Dict: Dict{
				"eq":  "builtin_eq_bool_eq",
				"neq": "builtin_eq_bool_neq",
			},
		},

		// Ord[Int]
		{
			ClassName: "Ord",
			TypeHead:  TInt,
			Super:     []string{"Eq"},
			Dict: Dict{
				"lt":  "builtin_ord_int_lt",
				"lte": "builtin_ord_int_lte",
				"gt":  "builtin_ord_int_gt",
				"gte": "builtin_ord_int_gte",
			},
		},

		// Ord[Float] - Total order with NaN greatest
		{
			ClassName: "Ord",
			TypeHead:  TFloat,
			Super:     []string{"Eq"},
			Dict: Dict{
				"lt":  "builtin_ord_float_lt", // NaN is greatest
				"lte": "builtin_ord_float_lte",
				"gt":  "builtin_ord_float_gt",
				"gte": "builtin_ord_float_gte",
			},
		},

		// Ord[String]
		{
			ClassName: "Ord",
			TypeHead:  TString,
			Super:     []string{"Eq"},
			Dict: Dict{
				"lt":  "builtin_ord_string_lt",
				"lte": "builtin_ord_string_lte",
				"gt":  "builtin_ord_string_gt",
				"gte": "builtin_ord_string_gte",
			},
		},

		// Show[Int]
		{
			ClassName: "Show",
			TypeHead:  TInt,
			Dict: Dict{
				"show": "builtin_show_int",
			},
		},

		// Show[Float]
		{
			ClassName: "Show",
			TypeHead:  TFloat,
			Dict: Dict{
				"show": "builtin_show_float",
			},
		},

		// Show[String]
		{
			ClassName: "Show",
			TypeHead:  TString,
			Dict: Dict{
				"show": "builtin_show_string",
			},
		},

		// Show[Bool]
		{
			ClassName: "Show",
			TypeHead:  TBool,
			Dict: Dict{
				"show": "builtin_show_bool",
			},
		},

		// Fractional[Float] - extends Num with fractional operations
		{
			ClassName: "Fractional",
			TypeHead:  TFloat,
			Dict: Dict{
				"divide":       "builtin_fractional_float_divide",
				"recip":        "builtin_fractional_float_recip",
				"fromRational": "builtin_fractional_float_fromRational",
			},
			Super: []string{"Num"}, // Fractional extends Num
		},
	}
}

// CanonicalDictParams creates deterministically ordered dictionary parameters
func CanonicalDictParams(constraints []ClassConstraint) []DictParam {
	// Sort by: ClassName, then Type.String()
	sort.Slice(constraints, func(i, j int) bool {
		if constraints[i].Class != constraints[j].Class {
			return constraints[i].Class < constraints[j].Class
		}
		return constraints[i].Type.String() < constraints[j].Type.String()
	})

	params := make([]DictParam, len(constraints))
	for i, c := range constraints {
		params[i] = DictParam{
			Name:      fmt.Sprintf("dict_%s_%s", c.Class, typeToName(c.Type)),
			ClassName: c.Class,
			Type:      c.Type,
		}
	}
	return params
}

// typeToName converts a type to a valid identifier name
func typeToName(t Type) string {
	s := t.String()
	// Replace non-identifier characters
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "->", "_to_")
	s = strings.ReplaceAll(s, "[", "_")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, ",", "_")
	return s
}

// DictParam represents a dictionary parameter in a qualified type
type DictParam struct {
	Name      string // e.g., "dict_Num_α"
	ClassName string // e.g., "Num"
	Type      Type   // e.g., TVar("α")
}
