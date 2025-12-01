package golang

import "testing"

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"helloWorld", "HelloWorld"},
		{"hello", "Hello"},
		{"_private", "Private"},
		{"", ""},
		{"a", "A"},
		{"snake_case_name", "SnakeCaseName"},
		{"__double__underscore__", "DoubleUnderscore"},
		{"ALLCAPS", "ALLCAPS"},
		{"mixedCase_and_snake", "MixedCaseAndSnake"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "helloWorld"},
		{"HelloWorld", "helloWorld"},
		{"hello", "hello"},
		{"", ""},
		{"A", "a"},
		{"snake_case_name", "snakeCaseName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToGoTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tree", "Tree"},
		{"FrameInput", "FrameInput"},
		{"tree", "Tree"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoTypeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToGoFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"frame_input", "FrameInput"},
		{"x", "X"},
		{"position", "Position"},
		{"draw_cmds", "DrawCmds"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoFieldName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToGoFuncName(t *testing.T) {
	tests := []struct {
		input    string
		exported bool
		expected string
	}{
		{"step", true, "Step"},
		{"step", false, "step"},
		{"init_world", true, "InitWorld"},
		{"init_world", false, "initWorld"},
		{"factorial", true, "Factorial"},
		{"factorial", false, "factorial"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoFuncName(tt.input, tt.exported)
			if result != tt.expected {
				t.Errorf("ToGoFuncName(%q, %v) = %q, want %q", tt.input, tt.exported, result, tt.expected)
			}
		})
	}
}

func TestToKindConstName(t *testing.T) {
	tests := []struct {
		typeName    string
		variantName string
		expected    string
	}{
		{"Tree", "Leaf", "TreeKindLeaf"},
		{"Tree", "Node", "TreeKindNode"},
		{"DrawCmd", "Sprite", "DrawCmdKindSprite"},
		{"option", "some", "OptionKindSome"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName+"_"+tt.variantName, func(t *testing.T) {
			result := ToKindConstName(tt.typeName, tt.variantName)
			if result != tt.expected {
				t.Errorf("ToKindConstName(%q, %q) = %q, want %q", tt.typeName, tt.variantName, result, tt.expected)
			}
		})
	}
}

func TestToKindTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tree", "TreeKind"},
		{"DrawCmd", "DrawCmdKind"},
		{"option", "OptionKind"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToKindTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("ToKindTypeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToVariantStructName(t *testing.T) {
	tests := []struct {
		typeName    string
		variantName string
		expected    string
	}{
		{"Tree", "Leaf", "TreeLeaf"},
		{"Tree", "Node", "TreeNode"},
		{"option", "some", "OptionSome"},
		{"option", "none", "OptionNone"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName+"_"+tt.variantName, func(t *testing.T) {
			result := ToVariantStructName(tt.typeName, tt.variantName)
			if result != tt.expected {
				t.Errorf("ToVariantStructName(%q, %q) = %q, want %q", tt.typeName, tt.variantName, result, tt.expected)
			}
		})
	}
}

func TestSanitizeGoIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"123abc", "_123abc"},
		{"hello-world", "hello_world"},
		{"hello.world", "hello_world"},
		{"", "_"},
		{"valid_name", "valid_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeGoIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeGoIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsGoKeyword(t *testing.T) {
	keywords := []string{"break", "case", "chan", "const", "continue", "default",
		"defer", "else", "fallthrough", "for", "func", "go", "goto", "if",
		"import", "interface", "map", "package", "range", "return", "select",
		"struct", "switch", "type", "var"}

	for _, kw := range keywords {
		if !IsGoKeyword(kw) {
			t.Errorf("IsGoKeyword(%q) = false, want true", kw)
		}
	}

	nonKeywords := []string{"hello", "world", "foo", "bar", "Tree", "Node"}
	for _, nkw := range nonKeywords {
		if IsGoKeyword(nkw) {
			t.Errorf("IsGoKeyword(%q) = true, want false", nkw)
		}
	}
}

func TestEscapeKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"type", "type_"},
		{"func", "func_"},
		{"map", "map_"},
		{"hello", "hello"},
		{"Tree", "Tree"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeKeyword(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeKeyword(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
