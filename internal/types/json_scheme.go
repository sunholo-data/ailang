package types

import "encoding/json"

// --- Scheme JSON (needed for Iface caching) ---

// MarshalScheme serializes a Scheme to JSON.
func MarshalScheme(s *Scheme) ([]byte, error) {
	if s == nil {
		return json.Marshal(nil)
	}

	typeBytes, err := json.Marshal(s.Type)
	if err != nil {
		return nil, err
	}

	type constraintJSON struct {
		Class   string          `json:"class"`
		Type    json.RawMessage `json:"type"`
		Literal bool            `json:"literal,omitempty"` // #2378
	}
	constraints := make([]constraintJSON, len(s.Constraints))
	for i, c := range s.Constraints {
		ct, err := json.Marshal(c.Type)
		if err != nil {
			return nil, err
		}
		constraints[i] = constraintJSON{Class: c.Class, Type: ct, Literal: c.Literal}
	}

	return json.Marshal(struct {
		TypeVars    []string         `json:"type_vars"`
		RowVars     []string         `json:"row_vars"`
		Constraints []constraintJSON `json:"constraints,omitempty"`
		Type        json.RawMessage  `json:"type"`
	}{
		TypeVars:    s.TypeVars,
		RowVars:     s.RowVars,
		Constraints: constraints,
		Type:        typeBytes,
	})
}

// UnmarshalScheme deserializes a Scheme from JSON.
func UnmarshalScheme(data []byte) (*Scheme, error) {
	if string(data) == "null" {
		return nil, nil
	}

	type constraintJSON struct {
		Class   string          `json:"class"`
		Type    json.RawMessage `json:"type"`
		Literal bool            `json:"literal,omitempty"` // #2378
	}
	var d struct {
		TypeVars    []string         `json:"type_vars"`
		RowVars     []string         `json:"row_vars"`
		Constraints []constraintJSON `json:"constraints,omitempty"`
		Type        json.RawMessage  `json:"type"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	t, err := UnmarshalType(d.Type)
	if err != nil {
		return nil, err
	}

	constraints := make([]Constraint, len(d.Constraints))
	for i, c := range d.Constraints {
		ct, err := UnmarshalType(c.Type)
		if err != nil {
			return nil, err
		}
		constraints[i] = Constraint{Class: c.Class, Type: ct, Literal: c.Literal}
	}

	return &Scheme{
		TypeVars:    d.TypeVars,
		RowVars:     d.RowVars,
		Constraints: constraints,
		Type:        t,
	}, nil
}
