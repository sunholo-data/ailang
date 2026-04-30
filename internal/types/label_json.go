package types

import (
	"encoding/json"
	"fmt"
)

// labelJSON is the tagged-union envelope for serialising a Label.
type labelJSON struct {
	Tag  string          `json:"tag"`
	Data json.RawMessage `json:"data,omitempty"`
}

// MarshalLabel serialises a Label to JSON. nil is serialised as ⊥ for
// backwards compatibility with ifaces that omit the field.
func MarshalLabel(l Label) ([]byte, error) {
	if l == nil {
		return json.Marshal(labelJSON{Tag: "bottom"})
	}
	switch v := l.(type) {
	case labelBottom:
		return json.Marshal(labelJSON{Tag: "bottom"})
	case labelConst:
		raw, _ := json.Marshal(struct {
			Name string `json:"name"`
		}{v.name})
		return json.Marshal(labelJSON{Tag: "const", Data: raw})
	case labelVar:
		raw, _ := json.Marshal(struct {
			Name string `json:"name"`
		}{v.name})
		return json.Marshal(labelJSON{Tag: "var", Data: raw})
	case labelJoin:
		parts := make([]json.RawMessage, len(v.parts))
		for i, p := range v.parts {
			raw, err := MarshalLabel(p)
			if err != nil {
				return nil, err
			}
			parts[i] = raw
		}
		raw, _ := json.Marshal(struct {
			Parts []json.RawMessage `json:"parts"`
		}{parts})
		return json.Marshal(labelJSON{Tag: "join", Data: raw})
	default:
		return nil, fmt.Errorf("MarshalLabel: unsupported label type %T", l)
	}
}

// UnmarshalLabel deserialises a Label from JSON. Empty/null/missing data
// returns ⊥ for backwards compatibility with pre-label ifaces.
func UnmarshalLabel(data []byte) (Label, error) {
	if len(data) == 0 || string(data) == "null" {
		return LabelBottom(), nil
	}
	var env labelJSON
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("UnmarshalLabel: %w", err)
	}
	switch env.Tag {
	case "", "bottom":
		return LabelBottom(), nil
	case "const":
		var d struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return nil, err
		}
		return LabelConst(d.Name), nil
	case "var":
		var d struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return nil, err
		}
		return LabelVar(d.Name), nil
	case "join":
		var d struct {
			Parts []json.RawMessage `json:"parts"`
		}
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return nil, err
		}
		result := LabelBottom()
		for _, raw := range d.Parts {
			p, err := UnmarshalLabel(raw)
			if err != nil {
				return nil, err
			}
			result = LabelJoin(result, p)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("UnmarshalLabel: unknown tag %q", env.Tag)
	}
}
