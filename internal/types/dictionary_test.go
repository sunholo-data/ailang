package types

import (
	"testing"
)

func TestResolvedConstraint(t *testing.T) {
	tests := []struct {
		name      string
		nodeID    uint64
		className string
		typ       Type
		method    string
	}{
		{
			name:      "arithmetic operator",
			nodeID:    1,
			className: "Num",
			typ:       TInt,
			method:    "add",
		},
		{
			name:      "comparison operator",
			nodeID:    2,
			className: "Ord",
			typ:       TFloat,
			method:    "lt",
		},
		{
			name:      "equality operator",
			nodeID:    3,
			className: "Eq",
			typ:       TBool,
			method:    "eq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &ResolvedConstraint{
				NodeID:    tt.nodeID,
				ClassName: tt.className,
				Type:      tt.typ,
				Method:    tt.method,
			}

			if rc.NodeID != tt.nodeID {
				t.Errorf("NodeID = %v, want %v", rc.NodeID, tt.nodeID)
			}
			if rc.ClassName != tt.className {
				t.Errorf("ClassName = %v, want %v", rc.ClassName, tt.className)
			}
			if !rc.Type.Equals(tt.typ) {
				t.Errorf("Type = %v, want %v", rc.Type, tt.typ)
			}
			if rc.Method != tt.method {
				t.Errorf("Method = %v, want %v", rc.Method, tt.method)
			}
		})
	}
}

func TestOperatorMethod(t *testing.T) {
	tests := []struct {
		op     string
		method string
	}{
		{"+", "add"},
		{"-", "sub"},
		{"*", "mul"},
		{"/", "div"},
		{"==", "eq"},
		{"!=", "neq"},
		{"<", "lt"},
		{"<=", "lte"},
		{">", "gt"},
		{">=", "gte"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			got := OperatorMethod(tt.op, false)
			if got != tt.method {
				t.Errorf("operatorMethod(%q) = %v, want %v", tt.op, got, tt.method)
			}
		})
	}
}

func TestNormalizeTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected string
	}{
		{
			name:     "int type",
			typ:      TInt,
			expected: "Int",
		},
		{
			name:     "float type",
			typ:      TFloat,
			expected: "Float",
		},
		{
			name:     "string type",
			typ:      TString,
			expected: "String",
		},
		{
			name:     "bool type",
			typ:      TBool,
			expected: "Bool",
		},
		{
			name:     "type constructor",
			typ:      &TCon{Name: "Maybe"},
			expected: "Maybe",
		},
		{
			name: "type application",
			typ: &TApp{
				Constructor: &TCon{Name: "List"},
				Args:        []Type{TInt},
			},
			expected: "List<Int>",
		},
		{
			name: "nested type application",
			typ: &TApp{
				Constructor: &TCon{Name: "Maybe"},
				Args: []Type{
					&TApp{
						Constructor: &TCon{Name: "List"},
						Args:        []Type{TString},
					},
				},
			},
			expected: "Maybe<List<String>>",
		},
		{
			name:     "type variable",
			typ:      &TVar{Name: "a"},
			expected: "_a",
		},
		{
			name: "function type",
			typ: &TFunc{
				Params: []Type{TInt, TInt},
				Return: TInt,
			},
			expected: "Func<Int,Int->Int>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTypeName(tt.typ)
			if got != tt.expected {
				t.Errorf("NormalizeTypeName(%v) = %v, want %v", tt.typ, got, tt.expected)
			}
		})
	}
}

func TestMakeDictionaryKey(t *testing.T) {
	tests := []struct {
		module    string
		className string
		typeName  string
		method    string
		expected  string
	}{
		{
			module:    "prelude",
			className: "Num",
			typeName:  "int",
			method:    "add",
			expected:  "prelude::Num::Int::add",
		},
		{
			module:    "std",
			className: "Ord",
			typeName:  "float",
			method:    "lt",
			expected:  "std::Ord::Float::lt",
		},
		{
			module:    "user",
			className: "Show",
			typeName:  "[int]",
			method:    "show",
			expected:  "user::Show::List<Int>::show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Create a type from the string
			var typ Type
			switch tt.typeName {
			case "int":
				typ = &TCon{Name: "int"}
			case "float":
				typ = &TCon{Name: "float"}
			case "[int]":
				typ = &TList{Element: &TCon{Name: "int"}}
			default:
				typ = &TCon{Name: tt.typeName}
			}
			got := MakeDictionaryKey(tt.module, tt.className, typ, tt.method)
			if got != tt.expected {
				t.Errorf("MakeDictionaryKey(%q, %q, %v, %q) = %v, want %v",
					tt.module, tt.className, typ, tt.method, got, tt.expected)
			}
		})
	}
}

func TestParseDictionaryKey(t *testing.T) {
	tests := []struct {
		key       string
		module    string
		className string
		typeName  string
		method    string
		wantErr   bool
	}{
		{
			key:       "prelude::Num::Int::add",
			module:    "prelude",
			className: "Num",
			typeName:  "Int",
			method:    "add",
			wantErr:   false,
		},
		{
			key:       "std::Ord::Float::lt",
			module:    "std",
			className: "Ord",
			typeName:  "Float",
			method:    "lt",
			wantErr:   false,
		},
		{
			key:     "invalid::key",
			wantErr: true,
		},
		{
			key:     "too::many::parts::in::this::key::extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			module, className, typeName, method, err := ParseDictionaryKey(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDictionaryKey(%q) expected error, got nil", tt.key)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDictionaryKey(%q) unexpected error: %v", tt.key, err)
				return
			}

			if module != tt.module {
				t.Errorf("module = %v, want %v", module, tt.module)
			}
			if className != tt.className {
				t.Errorf("className = %v, want %v", className, tt.className)
			}
			if typeName != tt.typeName {
				t.Errorf("typeName = %v, want %v", typeName, tt.typeName)
			}
			if method != tt.method {
				t.Errorf("method = %v, want %v", method, tt.method)
			}
		})
	}
}
