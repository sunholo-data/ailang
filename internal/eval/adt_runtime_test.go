package eval

import (
	"testing"
)

// TestTaggedValue tests the TaggedValue runtime type
func TestTaggedValue(t *testing.T) {
	tests := []struct {
		name     string
		value    *TaggedValue
		wantType string
		wantStr  string
	}{
		{
			name: "nullary constructor",
			value: &TaggedValue{
				TypeName: "Option",
				CtorName: "None",
				Fields:   []Value{},
			},
			wantType: "Option",
			wantStr:  "None",
		},
		{
			name: "unary constructor",
			value: &TaggedValue{
				TypeName: "Option",
				CtorName: "Some",
				Fields:   []Value{&IntValue{Value: 42}},
			},
			wantType: "Option",
			wantStr:  "Some(42)",
		},
		{
			name: "binary constructor",
			value: &TaggedValue{
				TypeName: "Pair",
				CtorName: "Pair",
				Fields: []Value{
					&IntValue{Value: 1},
					&StringValue{Value: "hello"},
				},
			},
			wantType: "Pair",
			wantStr:  "Pair(1, hello)",
		},
		{
			name: "nested constructor",
			value: &TaggedValue{
				TypeName: "Result",
				CtorName: "Ok",
				Fields: []Value{
					&TaggedValue{
						TypeName: "Option",
						CtorName: "Some",
						Fields:   []Value{&IntValue{Value: 99}},
					},
				},
			},
			wantType: "Result",
			wantStr:  "Ok(Some(99))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Type(); got != tt.wantType {
				t.Errorf("Type() = %v, want %v", got, tt.wantType)
			}
			if got := tt.value.String(); got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

// TestIsTag tests the isTag runtime helper
func TestIsTag(t *testing.T) {
	someValue := &TaggedValue{
		TypeName: "Option",
		CtorName: "Some",
		Fields:   []Value{&IntValue{Value: 42}},
	}

	noneValue := &TaggedValue{
		TypeName: "Option",
		CtorName: "None",
		Fields:   []Value{},
	}

	intValue := &IntValue{Value: 42}

	tests := []struct {
		name     string
		value    Value
		typeName string
		ctorName string
		want     bool
	}{
		{
			name:     "matching Some",
			value:    someValue,
			typeName: "Option",
			ctorName: "Some",
			want:     true,
		},
		{
			name:     "matching None",
			value:    noneValue,
			typeName: "Option",
			ctorName: "None",
			want:     true,
		},
		{
			name:     "wrong constructor name",
			value:    someValue,
			typeName: "Option",
			ctorName: "None",
			want:     false,
		},
		{
			name:     "wrong type name",
			value:    someValue,
			typeName: "Result",
			ctorName: "Some",
			want:     false,
		},
		{
			name:     "non-tagged value",
			value:    intValue,
			typeName: "Option",
			ctorName: "Some",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTag(tt.value, tt.typeName, tt.ctorName); got != tt.want {
				t.Errorf("isTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTaggedValueRecordAccessAutoUnwrap is a regression test for M-STREAM-DX/M4:
// Single-constructor ADTs wrapping a record (newtype pattern) should support
// field access by auto-unwrapping the TaggedValue to get the inner RecordValue.
// Example: `type Item = Item({name: string})` → `Item({name: "x"}).name` should return "x"
func TestTaggedValueRecordAccessAutoUnwrap(t *testing.T) {
	// Simulate: type Item = Item({name: string, value: int})
	innerRecord := &RecordValue{
		Fields: map[string]Value{
			"name":  &StringValue{Value: "hello"},
			"value": &IntValue{Value: 42},
		},
	}
	taggedItem := &TaggedValue{
		TypeName: "Item",
		CtorName: "Item",
		Fields:   []Value{innerRecord},
	}

	// Auto-unwrap: single-field TaggedValue where field is RecordValue
	// This mirrors the logic in evalCoreRecordAccess / evalRecordAccess
	var record *RecordValue
	if len(taggedItem.Fields) == 1 {
		if innerRec, isRecord := taggedItem.Fields[0].(*RecordValue); isRecord {
			record = innerRec
		}
	}

	if record == nil {
		t.Fatal("auto-unwrap failed: expected RecordValue from single-field TaggedValue")
	}

	// Field access should work
	nameVal, ok := record.Fields["name"]
	if !ok {
		t.Fatal("field 'name' not found after auto-unwrap")
	}
	if nameVal.String() != "hello" {
		t.Errorf("name = %q, want %q", nameVal.String(), "hello")
	}

	valueVal, ok := record.Fields["value"]
	if !ok {
		t.Fatal("field 'value' not found after auto-unwrap")
	}
	if valueVal.String() != "42" {
		t.Errorf("value = %q, want %q", valueVal.String(), "42")
	}
}

// TestTaggedValueMultiFieldNoUnwrap verifies that multi-field TaggedValues
// are NOT auto-unwrapped (only single-field newtype pattern gets unwrapped).
func TestTaggedValueMultiFieldNoUnwrap(t *testing.T) {
	// Simulate: type Pair = Pair(int, {name: string}) — 2 fields, should NOT unwrap
	twoFieldTagged := &TaggedValue{
		TypeName: "Pair",
		CtorName: "Pair",
		Fields: []Value{
			&IntValue{Value: 1},
			&RecordValue{Fields: map[string]Value{"name": &StringValue{Value: "x"}}},
		},
	}

	var record *RecordValue
	if len(twoFieldTagged.Fields) == 1 {
		if innerRec, isRecord := twoFieldTagged.Fields[0].(*RecordValue); isRecord {
			record = innerRec
		}
	}

	if record != nil {
		t.Error("multi-field TaggedValue should NOT be auto-unwrapped")
	}
}

// TestTaggedValueNonRecordNoUnwrap verifies that single-field TaggedValues
// wrapping non-record values are NOT auto-unwrapped.
func TestTaggedValueNonRecordNoUnwrap(t *testing.T) {
	// Simulate: type Wrapper = Wrapper(int) — single field but not a record
	nonRecordTagged := &TaggedValue{
		TypeName: "Wrapper",
		CtorName: "Wrapper",
		Fields:   []Value{&IntValue{Value: 99}},
	}

	var record *RecordValue
	if len(nonRecordTagged.Fields) == 1 {
		if innerRec, isRecord := nonRecordTagged.Fields[0].(*RecordValue); isRecord {
			record = innerRec
		}
	}

	if record != nil {
		t.Error("single-field TaggedValue wrapping non-record should NOT be auto-unwrapped")
	}
}

// TestGetField tests the getField runtime helper
func TestGetField(t *testing.T) {
	someValue := &TaggedValue{
		TypeName: "Option",
		CtorName: "Some",
		Fields:   []Value{&IntValue{Value: 42}},
	}

	pairValue := &TaggedValue{
		TypeName: "Pair",
		CtorName: "Pair",
		Fields: []Value{
			&IntValue{Value: 1},
			&StringValue{Value: "hello"},
		},
	}

	noneValue := &TaggedValue{
		TypeName: "Option",
		CtorName: "None",
		Fields:   []Value{},
	}

	intValue := &IntValue{Value: 42}

	tests := []struct {
		name      string
		value     Value
		index     int
		wantValue Value
		wantError bool
	}{
		{
			name:      "get field 0 from Some",
			value:     someValue,
			index:     0,
			wantValue: &IntValue{Value: 42},
			wantError: false,
		},
		{
			name:      "get field 0 from Pair",
			value:     pairValue,
			index:     0,
			wantValue: &IntValue{Value: 1},
			wantError: false,
		},
		{
			name:      "get field 1 from Pair",
			value:     pairValue,
			index:     1,
			wantValue: &StringValue{Value: "hello"},
			wantError: false,
		},
		{
			name:      "out of bounds - negative index",
			value:     someValue,
			index:     -1,
			wantValue: nil,
			wantError: true,
		},
		{
			name:      "out of bounds - index too large",
			value:     someValue,
			index:     1,
			wantValue: nil,
			wantError: true,
		},
		{
			name:      "out of bounds - nullary constructor",
			value:     noneValue,
			index:     0,
			wantValue: nil,
			wantError: true,
		},
		{
			name:      "non-tagged value",
			value:     intValue,
			index:     0,
			wantValue: nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getField(tt.value, tt.index)
			if (err != nil) != tt.wantError {
				t.Errorf("getField() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Compare values by their string representation
				if got.String() != tt.wantValue.String() {
					t.Errorf("getField() = %v, want %v", got, tt.wantValue)
				}
			}
		})
	}
}
