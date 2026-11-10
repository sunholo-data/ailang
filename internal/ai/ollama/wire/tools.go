package wire

import "encoding/json"

// The tool types reproduce the SDK's decoding of a JSON Schema, which is
// LOSSY and the server sees the loss: keywords outside these structs
// ("additionalProperties", "minimum", "default", "$schema", ...) are dropped,
// the struct field order is imposed, "type" may be a string or an array, and
// property maps keep document order while free-form values ("items", "$defs",
// object enum members) are re-encoded by encoding/json with sorted keys.
// testdata/wire/chat.json in the parent package pins every one of those
// effects; keep it that way rather than "fixing" the loss here, because the
// models were evaluated against these exact bytes.

// Tools is the tool list of a ChatRequest.
type Tools []Tool

// Tool is one advertised tool.
type Tool struct {
	Type     string       `json:"type"`
	Items    any          `json:"items,omitempty"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is a tool's name, description and parameter schema.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

// ToolFunctionParameters is the top level of a tool's parameter schema.
type ToolFunctionParameters struct {
	Type       string             `json:"type"`
	Defs       any                `json:"$defs,omitempty"`
	Items      any                `json:"items,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Properties *ToolPropertiesMap `json:"properties"`
}

// ToolProperty is one property schema.
type ToolProperty struct {
	AnyOf       []ToolProperty     `json:"anyOf,omitempty"`
	Type        PropertyType       `json:"type,omitempty"`
	Items       any                `json:"items,omitempty"`
	Description string             `json:"description,omitempty"`
	Enum        []any              `json:"enum,omitempty"`
	Properties  *ToolPropertiesMap `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
}

// PropertyType is a JSON Schema "type": one name marshals as a string,
// several as an array.
type PropertyType []string

// UnmarshalJSON accepts a string or an array of strings.
func (pt *PropertyType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*pt = []string{s}
		return nil
	}
	var a []string
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*pt = a
	return nil
}

// MarshalJSON writes a single type as a string, otherwise an array.
func (pt PropertyType) MarshalJSON() ([]byte, error) {
	if len(pt) == 1 {
		return json.Marshal(pt[0])
	}
	return json.Marshal([]string(pt))
}

// ToolPropertiesMap holds property schemas in document order.
type ToolPropertiesMap struct {
	m *orderedMap[ToolProperty]
}

// MarshalJSON writes the properties in document order.
func (t ToolPropertiesMap) MarshalJSON() ([]byte, error) {
	if t.m == nil {
		return []byte("null"), nil
	}
	return t.m.MarshalJSON()
}

// UnmarshalJSON reads properties, keeping document order.
func (t *ToolPropertiesMap) UnmarshalJSON(data []byte) error {
	t.m = &orderedMap[ToolProperty]{}
	return t.m.UnmarshalJSON(data)
}

// ToolCall is a tool invocation, in a reply or re-sent in history.
type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction names the tool and carries its arguments.
type ToolCallFunction struct {
	Index     int                       `json:"index"`
	Name      string                    `json:"name"`
	Arguments ToolCallFunctionArguments `json:"arguments"`
}

// ToolCallFunctionArguments holds tool-call arguments in document order. The
// zero value marshals as {}; arguments decoded from JSON null marshal as null.
type ToolCallFunctionArguments struct {
	m *orderedMap[any]
}

// String is the arguments as compact JSON ({} when unset), the form callers
// hand on as ai.ToolCall.Arguments.
func (t *ToolCallFunctionArguments) String() string {
	if t == nil || t.m == nil {
		return "{}"
	}
	b, _ := json.Marshal(t.m)
	return string(b)
}

// MarshalJSON writes the arguments in document order.
func (t ToolCallFunctionArguments) MarshalJSON() ([]byte, error) {
	if t.m == nil {
		return []byte("{}"), nil
	}
	return t.m.MarshalJSON()
}

// UnmarshalJSON reads arguments, keeping document order.
func (t *ToolCallFunctionArguments) UnmarshalJSON(data []byte) error {
	t.m = &orderedMap[any]{}
	return t.m.UnmarshalJSON(data)
}
