package argdecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestDecodeError_Error(t *testing.T) {
	err := &DecodeError{
		Expected: "int",
		Got:      "\"hello\"",
		Reason:   "expected JSON number for int type",
	}

	msg := err.Error()
	assert.Contains(t, msg, "ARG_DECODE_MISMATCH")
	assert.Contains(t, msg, "int")
	assert.Contains(t, msg, "\"hello\"")
	assert.Contains(t, msg, "expected JSON number for int type")
}

func TestDecodeJSON_Unit(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "null to unit",
			json:    "null",
			wantErr: false,
		},
		{
			name:    "non-null to unit fails",
			json:    "42",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitType := &types.TCon{Name: "Unit"}
			val, err := DecodeJSON(tt.json, unitType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				_, ok := val.(*eval.UnitValue)
				assert.True(t, ok, "Should decode to UnitValue")
			}
		})
	}
}

func TestDecodeJSON_Int(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected int
		wantErr  bool
	}{
		{
			name:     "positive integer",
			json:     "42",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "negative integer",
			json:     "-17",
			expected: -17,
			wantErr:  false,
		},
		{
			name:     "zero",
			json:     "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "string to int fails",
			json:    "\"42\"",
			wantErr: true,
		},
		{
			name:    "bool to int fails",
			json:    "true",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intType := &types.TCon{Name: "Int"}
			val, err := DecodeJSON(tt.json, intType)

			if tt.wantErr {
				assert.Error(t, err)
				var decodeErr *DecodeError
				if assert.ErrorAs(t, err, &decodeErr) {
					assert.Equal(t, "int", decodeErr.Expected)
				}
			} else {
				require.NoError(t, err)
				intVal, ok := val.(*eval.IntValue)
				require.True(t, ok, "Should decode to IntValue")
				assert.Equal(t, tt.expected, intVal.Value)
			}
		})
	}
}

func TestDecodeJSON_Float(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected float64
		wantErr  bool
	}{
		{
			name:     "float value",
			json:     "3.14",
			expected: 3.14,
			wantErr:  false,
		},
		{
			name:     "integer as float",
			json:     "42",
			expected: 42.0,
			wantErr:  false,
		},
		{
			name:     "negative float",
			json:     "-2.718",
			expected: -2.718,
			wantErr:  false,
		},
		{
			name:    "string to float fails",
			json:    "\"3.14\"",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			floatType := &types.TCon{Name: "Float"}
			val, err := DecodeJSON(tt.json, floatType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				floatVal, ok := val.(*eval.FloatValue)
				require.True(t, ok, "Should decode to FloatValue")
				assert.Equal(t, tt.expected, floatVal.Value)
			}
		})
	}
}

func TestDecodeJSON_String(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple string",
			json:     "\"hello\"",
			expected: "hello",
			wantErr:  false,
		},
		{
			name:     "empty string",
			json:     "\"\"",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "string with spaces",
			json:     "\"hello world\"",
			expected: "hello world",
			wantErr:  false,
		},
		{
			name:    "number to string fails",
			json:    "42",
			wantErr: true,
		},
		{
			name:    "bool to string fails",
			json:    "false",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringType := &types.TCon{Name: "String"}
			val, err := DecodeJSON(tt.json, stringType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				strVal, ok := val.(*eval.StringValue)
				require.True(t, ok, "Should decode to StringValue")
				assert.Equal(t, tt.expected, strVal.Value)
			}
		})
	}
}

func TestDecodeJSON_Bool(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "true",
			json:     "true",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "false",
			json:     "false",
			expected: false,
			wantErr:  false,
		},
		{
			name:    "number to bool fails",
			json:    "1",
			wantErr: true,
		},
		{
			name:    "string to bool fails",
			json:    "\"true\"",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boolType := &types.TCon{Name: "Bool"}
			val, err := DecodeJSON(tt.json, boolType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				boolVal, ok := val.(*eval.BoolValue)
				require.True(t, ok, "Should decode to BoolValue")
				assert.Equal(t, tt.expected, boolVal.Value)
			}
		})
	}
}

func TestDecodeJSON_List(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*testing.T, eval.Value)
	}{
		{
			name:    "list of ints",
			json:    "[1, 2, 3]",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				listVal, ok := val.(*eval.ListValue)
				require.True(t, ok)
				assert.Equal(t, 3, len(listVal.Elements))
				assert.Equal(t, 1, listVal.Elements[0].(*eval.IntValue).Value)
				assert.Equal(t, 2, listVal.Elements[1].(*eval.IntValue).Value)
				assert.Equal(t, 3, listVal.Elements[2].(*eval.IntValue).Value)
			},
		},
		{
			name:    "empty list",
			json:    "[]",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				listVal, ok := val.(*eval.ListValue)
				require.True(t, ok)
				assert.Equal(t, 0, len(listVal.Elements))
			},
		},
		{
			name:    "list of strings",
			json:    "[\"a\", \"b\", \"c\"]",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				listVal, ok := val.(*eval.ListValue)
				require.True(t, ok)
				assert.Equal(t, 3, len(listVal.Elements))
			},
		},
		{
			name:    "non-array to list fails",
			json:    "42",
			wantErr: true,
		},
		{
			name:    "mixed types fails",
			json:    "[1, \"two\", 3]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var listType types.Type
			if tt.name == "list of strings" {
				listType = &types.TList{Element: &types.TCon{Name: "String"}}
			} else {
				listType = &types.TList{Element: &types.TCon{Name: "Int"}}
			}

			val, err := DecodeJSON(tt.json, listType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, val)
				}
			}
		})
	}
}

func TestDecodeJSON_Record(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*testing.T, eval.Value)
	}{
		{
			name:    "simple record",
			json:    `{"x": 10, "y": 20}`,
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				recVal, ok := val.(*eval.RecordValue)
				require.True(t, ok)
				assert.Equal(t, 2, len(recVal.Fields))
				assert.Equal(t, 10, recVal.Fields["x"].(*eval.IntValue).Value)
				assert.Equal(t, 20, recVal.Fields["y"].(*eval.IntValue).Value)
			},
		},
		{
			name:    "record with string field",
			json:    `{"name": "Alice", "age": 30}`,
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				recVal, ok := val.(*eval.RecordValue)
				require.True(t, ok)
				assert.Equal(t, "Alice", recVal.Fields["name"].(*eval.StringValue).Value)
				assert.Equal(t, 30, recVal.Fields["age"].(*eval.IntValue).Value)
			},
		},
		{
			name:    "missing field fails",
			json:    `{"x": 10}`,
			wantErr: true,
		},
		{
			name:    "non-object to record fails",
			json:    "42",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordType *types.TRecord
			if tt.name == "record with string field" {
				recordType = &types.TRecord{
					Fields: map[string]types.Type{
						"name": &types.TCon{Name: "String"},
						"age":  &types.TCon{Name: "Int"},
					},
				}
			} else {
				recordType = &types.TRecord{
					Fields: map[string]types.Type{
						"x": &types.TCon{Name: "Int"},
						"y": &types.TCon{Name: "Int"},
					},
				}
			}

			val, err := DecodeJSON(tt.json, recordType)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.name == "missing field fails" {
					var decodeErr *DecodeError
					assert.ErrorAs(t, err, &decodeErr)
				}
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, val)
				}
			}
		})
	}
}

func TestDecodeJSON_TypeVariable(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*testing.T, eval.Value)
	}{
		{
			name:    "null to type var",
			json:    "null",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				_, ok := val.(*eval.UnitValue)
				assert.True(t, ok)
			},
		},
		{
			name:    "number to type var",
			json:    "42",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				intVal, ok := val.(*eval.IntValue)
				assert.True(t, ok)
				assert.Equal(t, 42, intVal.Value)
			},
		},
		{
			name:    "string to type var",
			json:    "\"hello\"",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				strVal, ok := val.(*eval.StringValue)
				assert.True(t, ok)
				assert.Equal(t, "hello", strVal.Value)
			},
		},
		{
			name:    "bool to type var",
			json:    "true",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				boolVal, ok := val.(*eval.BoolValue)
				assert.True(t, ok)
				assert.True(t, boolVal.Value)
			},
		},
		{
			name:    "array to type var",
			json:    "[1, 2, 3]",
			wantErr: false,
			check: func(t *testing.T, val eval.Value) {
				listVal, ok := val.(*eval.ListValue)
				assert.True(t, ok)
				assert.Equal(t, 3, len(listVal.Elements))
			},
		},
		{
			name:    "object to type var fails",
			json:    `{"x": 10}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeVar := &types.TVar2{Name: "a", Kind: types.Star}
			val, err := DecodeJSON(tt.json, typeVar)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, val)
				}
			}
		})
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	invalidJSON := "not valid json"
	val, err := DecodeJSON(invalidJSON, &types.TCon{Name: "Int"})
	assert.Error(t, err)
	assert.Nil(t, val)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestDecodeJSON_UnsupportedType(t *testing.T) {
	// Try to decode with an unsupported type constructor
	unsupportedType := &types.TCon{Name: "CustomType"}
	val, err := DecodeJSON("42", unsupportedType)
	assert.Error(t, err)
	assert.Nil(t, val)
	assert.Contains(t, err.Error(), "unsupported type constructor")
}

func TestDecodeJSON_NestedList(t *testing.T) {
	// List of lists
	listOfListsType := &types.TList{
		Element: &types.TList{
			Element: &types.TCon{Name: "Int"},
		},
	}

	val, err := DecodeJSON("[[1, 2], [3, 4]]", listOfListsType)
	require.NoError(t, err)

	listVal, ok := val.(*eval.ListValue)
	require.True(t, ok)
	assert.Equal(t, 2, len(listVal.Elements))

	innerList1, ok := listVal.Elements[0].(*eval.ListValue)
	require.True(t, ok)
	assert.Equal(t, 2, len(innerList1.Elements))
}

func TestDecodeJSON_RecordWithList(t *testing.T) {
	recordType := &types.TRecord{
		Fields: map[string]types.Type{
			"name":   &types.TCon{Name: "String"},
			"scores": &types.TList{Element: &types.TCon{Name: "Int"}},
		},
	}

	val, err := DecodeJSON(`{"name": "Alice", "scores": [95, 87, 92]}`, recordType)
	require.NoError(t, err)

	recVal, ok := val.(*eval.RecordValue)
	require.True(t, ok)
	assert.Equal(t, "Alice", recVal.Fields["name"].(*eval.StringValue).Value)

	scores, ok := recVal.Fields["scores"].(*eval.ListValue)
	require.True(t, ok)
	assert.Equal(t, 3, len(scores.Elements))
}

func TestDecodeJSON_AlternativeTypeNames(t *testing.T) {
	// Test lowercase type names
	tests := []struct {
		typeName string
		json     string
	}{
		{"int", "42"},
		{"float", "3.14"},
		{"string", "\"hello\""},
		{"bool", "true"},
		{"unit", "null"},
		{"()", "null"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			typ := &types.TCon{Name: tt.typeName}
			val, err := DecodeJSON(tt.json, typ)
			assert.NoError(t, err)
			assert.NotNil(t, val)
		})
	}
}
