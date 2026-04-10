package types

import (
	"encoding/json"
	"fmt"
)

// M-INCREMENTAL-TYPECHECK: Tagged-union JSON serialization for types.Type and Kind.
// Enables disk-caching of compiled module artifacts (CoreTypeInfo, Iface, etc.)
// so unchanged modules can skip elaboration→typecheck→monomorph→lower.
//
// All Type implementations get MarshalJSON/UnmarshalJSON so nested types
// serialize correctly via standard json.Marshal.

// typeJSON is the tagged-union envelope for serializing types.Type.
type typeJSON struct {
	Tag  string          `json:"tag"`
	Data json.RawMessage `json:"data"`
}

// kindJSON is the tagged-union envelope for serializing Kind.
type kindJSON struct {
	Tag  string          `json:"tag"`
	Data json.RawMessage `json:"data,omitempty"`
}

// --- MarshalJSON implementations for each Type ---

func (t *TCon) MarshalJSON() ([]byte, error) {
	raw, _ := json.Marshal(struct{ Name string }{t.Name})
	return json.Marshal(typeJSON{Tag: "tcon", Data: raw})
}

func (t *TVar) MarshalJSON() ([]byte, error) {
	raw, _ := json.Marshal(struct{ Name string }{t.Name})
	return json.Marshal(typeJSON{Tag: "tvar", Data: raw})
}

func (t *TVar2) MarshalJSON() ([]byte, error) {
	kindBytes, err := MarshalKind(t.Kind)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(struct {
		Name string          `json:"name"`
		Kind json.RawMessage `json:"kind"`
	}{t.Name, kindBytes})
	return json.Marshal(typeJSON{Tag: "tvar2", Data: raw})
}

func (t *RowVar) MarshalJSON() ([]byte, error) {
	kindBytes, err := MarshalKind(t.Kind)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(struct {
		Name string          `json:"name"`
		Kind json.RawMessage `json:"kind"`
	}{t.Name, kindBytes})
	return json.Marshal(typeJSON{Tag: "rowvar", Data: raw})
}

func (t *TList) MarshalJSON() ([]byte, error) {
	elemBytes, err := json.Marshal(t.Element)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(struct {
		Element json.RawMessage `json:"element"`
	}{elemBytes})
	return json.Marshal(typeJSON{Tag: "tlist", Data: raw})
}

func (t *TArray) MarshalJSON() ([]byte, error) {
	elemBytes, err := json.Marshal(t.Element)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(struct {
		Element json.RawMessage `json:"element"`
	}{elemBytes})
	return json.Marshal(typeJSON{Tag: "tarray", Data: raw})
}

func (t *TMap) MarshalJSON() ([]byte, error) {
	keyBytes, err := json.Marshal(t.Key)
	if err != nil {
		return nil, err
	}
	valBytes, err := json.Marshal(t.Value)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(struct {
		Key   json.RawMessage `json:"key"`
		Value json.RawMessage `json:"value"`
	}{keyBytes, valBytes})
	return json.Marshal(typeJSON{Tag: "tmap", Data: raw})
}

func (t *TTuple) MarshalJSON() ([]byte, error) {
	elems := make([]json.RawMessage, len(t.Elements))
	for i, e := range t.Elements {
		raw, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		elems[i] = raw
	}
	raw, _ := json.Marshal(struct {
		Elements []json.RawMessage `json:"elements"`
	}{elems})
	return json.Marshal(typeJSON{Tag: "ttuple", Data: raw})
}

func (t *TFunc2) MarshalJSON() ([]byte, error) {
	params := make([]json.RawMessage, len(t.Params))
	for i, p := range t.Params {
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		params[i] = raw
	}
	retBytes, err := json.Marshal(t.Return)
	if err != nil {
		return nil, err
	}
	var effBytes json.RawMessage
	if t.EffectRow != nil {
		effBytes, err = json.Marshal(t.EffectRow)
		if err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(struct {
		Params    []json.RawMessage `json:"params"`
		EffectRow json.RawMessage   `json:"effect_row,omitempty"`
		Return    json.RawMessage   `json:"return"`
	}{params, effBytes, retBytes})
	return json.Marshal(typeJSON{Tag: "tfunc2", Data: raw})
}

func (t *TRecord) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage)
	for name, ft := range t.Fields {
		raw, err := json.Marshal(ft)
		if err != nil {
			return nil, err
		}
		fields[name] = raw
	}
	var rowBytes json.RawMessage
	if t.Row != nil {
		var err error
		rowBytes, err = json.Marshal(t.Row)
		if err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(struct {
		Fields   map[string]json.RawMessage `json:"fields"`
		Row      json.RawMessage            `json:"row,omitempty"`
		TypeName string                     `json:"type_name,omitempty"`
	}{fields, rowBytes, t.TypeName})
	return json.Marshal(typeJSON{Tag: "trecord", Data: raw})
}

func (t *TRecordOpen) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage)
	for name, ft := range t.Fields {
		raw, err := json.Marshal(ft)
		if err != nil {
			return nil, err
		}
		fields[name] = raw
	}
	var rowBytes json.RawMessage
	if t.Row != nil {
		var err error
		rowBytes, err = json.Marshal(t.Row)
		if err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(struct {
		Fields map[string]json.RawMessage `json:"fields"`
		Row    json.RawMessage            `json:"row,omitempty"`
	}{fields, rowBytes})
	return json.Marshal(typeJSON{Tag: "trecord_open", Data: raw})
}

func (t *TRecord2) MarshalJSON() ([]byte, error) {
	var rowBytes json.RawMessage
	if t.Row != nil {
		var err error
		rowBytes, err = json.Marshal(t.Row)
		if err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(struct {
		Row json.RawMessage `json:"row,omitempty"`
	}{rowBytes})
	return json.Marshal(typeJSON{Tag: "trecord2", Data: raw})
}

func (t *TApp) MarshalJSON() ([]byte, error) {
	constrBytes, err := json.Marshal(t.Constructor)
	if err != nil {
		return nil, err
	}
	args := make([]json.RawMessage, len(t.Args))
	for i, a := range t.Args {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		args[i] = raw
	}
	raw, _ := json.Marshal(struct {
		Constructor json.RawMessage   `json:"constructor"`
		Args        []json.RawMessage `json:"args"`
	}{constrBytes, args})
	return json.Marshal(typeJSON{Tag: "tapp", Data: raw})
}

func (r *Row) MarshalJSON() ([]byte, error) {
	kindBytes, err := MarshalKind(r.Kind)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]json.RawMessage)
	for name, t := range r.Labels {
		raw, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		labels[name] = raw
	}
	var tailBytes json.RawMessage
	if r.Tail != nil {
		tailBytes, err = json.Marshal(r.Tail)
		if err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(struct {
		Kind       json.RawMessage            `json:"kind"`
		Labels     map[string]json.RawMessage `json:"labels"`
		Tail       json.RawMessage            `json:"tail,omitempty"`
		Budgets    map[string]*int            `json:"budgets,omitempty"`
		MinBudgets map[string]*int            `json:"min_budgets,omitempty"`
	}{kindBytes, labels, tailBytes, r.Budgets, r.MinBudgets})
	return json.Marshal(typeJSON{Tag: "row", Data: raw})
}

// --- UnmarshalType: dispatcher for all types ---

// UnmarshalType deserializes a types.Type from JSON.
func UnmarshalType(data []byte) (Type, error) {
	if string(data) == "null" {
		return nil, nil
	}

	var envelope typeJSON
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("UnmarshalType: %w", err)
	}

	switch envelope.Tag {
	case "tcon":
		var d struct{ Name string }
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		return &TCon{Name: d.Name}, nil

	case "tvar":
		var d struct{ Name string }
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		return &TVar{Name: d.Name}, nil

	case "tvar2":
		var d struct {
			Name string          `json:"name"`
			Kind json.RawMessage `json:"kind"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		k, err := UnmarshalKind(d.Kind)
		if err != nil {
			return nil, err
		}
		return &TVar2{Name: d.Name, Kind: k}, nil

	case "rowvar":
		var d struct {
			Name string          `json:"name"`
			Kind json.RawMessage `json:"kind"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		k, err := UnmarshalKind(d.Kind)
		if err != nil {
			return nil, err
		}
		return &RowVar{Name: d.Name, Kind: k}, nil

	case "tlist":
		var d struct {
			Element json.RawMessage `json:"element"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		elem, err := UnmarshalType(d.Element)
		if err != nil {
			return nil, err
		}
		return &TList{Element: elem}, nil

	case "tarray":
		var d struct {
			Element json.RawMessage `json:"element"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		elem, err := UnmarshalType(d.Element)
		if err != nil {
			return nil, err
		}
		return &TArray{Element: elem}, nil

	case "tmap":
		var d struct {
			Key   json.RawMessage `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		key, err := UnmarshalType(d.Key)
		if err != nil {
			return nil, err
		}
		val, err := UnmarshalType(d.Value)
		if err != nil {
			return nil, err
		}
		return &TMap{Key: key, Value: val}, nil

	case "ttuple":
		var d struct {
			Elements []json.RawMessage `json:"elements"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		elems := make([]Type, len(d.Elements))
		for i, raw := range d.Elements {
			t, err := UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			elems[i] = t
		}
		return &TTuple{Elements: elems}, nil

	case "tfunc2":
		var d struct {
			Params    []json.RawMessage `json:"params"`
			EffectRow json.RawMessage   `json:"effect_row,omitempty"`
			Return    json.RawMessage   `json:"return"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		params := make([]Type, len(d.Params))
		for i, raw := range d.Params {
			t, err := UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			params[i] = t
		}
		ret, err := UnmarshalType(d.Return)
		if err != nil {
			return nil, err
		}
		var effectRow *Row
		if len(d.EffectRow) > 0 && string(d.EffectRow) != "null" {
			eff, err := UnmarshalType(d.EffectRow)
			if err != nil {
				return nil, err
			}
			if row, ok := eff.(*Row); ok {
				effectRow = row
			}
		}
		return &TFunc2{Params: params, EffectRow: effectRow, Return: ret}, nil

	case "trecord":
		var d struct {
			Fields   map[string]json.RawMessage `json:"fields"`
			Row      json.RawMessage            `json:"row,omitempty"`
			TypeName string                     `json:"type_name,omitempty"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		fields := make(map[string]Type)
		for name, raw := range d.Fields {
			t, err := UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			fields[name] = t
		}
		var row Type
		if len(d.Row) > 0 && string(d.Row) != "null" {
			var err error
			row, err = UnmarshalType(d.Row)
			if err != nil {
				return nil, err
			}
		}
		return &TRecord{Fields: fields, Row: row, TypeName: d.TypeName}, nil

	case "trecord_open":
		var d struct {
			Fields map[string]json.RawMessage `json:"fields"`
			Row    json.RawMessage            `json:"row,omitempty"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		fields := make(map[string]Type)
		for name, raw := range d.Fields {
			t, err := UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			fields[name] = t
		}
		var row Type
		if len(d.Row) > 0 && string(d.Row) != "null" {
			var err error
			row, err = UnmarshalType(d.Row)
			if err != nil {
				return nil, err
			}
		}
		return &TRecordOpen{Fields: fields, Row: row}, nil

	case "trecord2":
		var d struct {
			Row json.RawMessage `json:"row,omitempty"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		var row *Row
		if len(d.Row) > 0 && string(d.Row) != "null" {
			t, err := UnmarshalType(d.Row)
			if err != nil {
				return nil, err
			}
			if r, ok := t.(*Row); ok {
				row = r
			}
		}
		return &TRecord2{Row: row}, nil

	case "tapp":
		var d struct {
			Constructor json.RawMessage   `json:"constructor"`
			Args        []json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal(envelope.Data, &d); err != nil {
			return nil, err
		}
		constr, err := UnmarshalType(d.Constructor)
		if err != nil {
			return nil, err
		}
		args := make([]Type, len(d.Args))
		for i, raw := range d.Args {
			t, err := UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			args[i] = t
		}
		return &TApp{Constructor: constr, Args: args}, nil

	case "row":
		return unmarshalRow(envelope.Data)

	default:
		return nil, fmt.Errorf("UnmarshalType: unknown tag %q", envelope.Tag)
	}
}

// unmarshalRow deserializes a Row from its data payload.
func unmarshalRow(data []byte) (*Row, error) {
	var d struct {
		Kind       json.RawMessage            `json:"kind"`
		Labels     map[string]json.RawMessage `json:"labels"`
		Tail       json.RawMessage            `json:"tail,omitempty"`
		Budgets    map[string]*int            `json:"budgets,omitempty"`
		MinBudgets map[string]*int            `json:"min_budgets,omitempty"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	k, err := UnmarshalKind(d.Kind)
	if err != nil {
		return nil, err
	}

	labels := make(map[string]Type)
	for name, raw := range d.Labels {
		t, err := UnmarshalType(raw)
		if err != nil {
			return nil, err
		}
		labels[name] = t
	}

	var tail *RowVar
	if len(d.Tail) > 0 && string(d.Tail) != "null" {
		t, err := UnmarshalType(d.Tail)
		if err != nil {
			return nil, err
		}
		if rv, ok := t.(*RowVar); ok {
			tail = rv
		}
	}

	return &Row{
		Kind:       k,
		Labels:     labels,
		Tail:       tail,
		Budgets:    d.Budgets,
		MinBudgets: d.MinBudgets,
	}, nil
}

// --- Kind JSON ---

// MarshalKind serializes a Kind to JSON.
func MarshalKind(k Kind) ([]byte, error) {
	if k == nil {
		return json.Marshal(nil)
	}

	var kj kindJSON
	switch kk := k.(type) {
	case KStar:
		kj = kindJSON{Tag: "star"}
	case KEffect:
		kj = kindJSON{Tag: "effect"}
	case KRecord:
		kj = kindJSON{Tag: "record"}
	case KRow:
		inner, err := MarshalKind(kk.ElemKind)
		if err != nil {
			return nil, err
		}
		kj = kindJSON{Tag: "row", Data: inner}
	default:
		return nil, fmt.Errorf("MarshalKind: unsupported kind %T", k)
	}
	return json.Marshal(kj)
}

// UnmarshalKind deserializes a Kind from JSON.
func UnmarshalKind(data []byte) (Kind, error) {
	if string(data) == "null" {
		return nil, nil
	}

	var kj kindJSON
	if err := json.Unmarshal(data, &kj); err != nil {
		return nil, err
	}

	switch kj.Tag {
	case "star":
		return KStar{}, nil
	case "effect":
		return KEffect{}, nil
	case "record":
		return KRecord{}, nil
	case "row":
		inner, err := UnmarshalKind(kj.Data)
		if err != nil {
			return nil, err
		}
		return KRow{ElemKind: inner}, nil
	default:
		return nil, fmt.Errorf("UnmarshalKind: unknown tag %q", kj.Tag)
	}
}

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
		Class string          `json:"class"`
		Type  json.RawMessage `json:"type"`
	}
	constraints := make([]constraintJSON, len(s.Constraints))
	for i, c := range s.Constraints {
		ct, err := json.Marshal(c.Type)
		if err != nil {
			return nil, err
		}
		constraints[i] = constraintJSON{Class: c.Class, Type: ct}
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
		Class string          `json:"class"`
		Type  json.RawMessage `json:"type"`
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
		constraints[i] = Constraint{Class: c.Class, Type: ct}
	}

	return &Scheme{
		TypeVars:    d.TypeVars,
		RowVars:     d.RowVars,
		Constraints: constraints,
		Type:        t,
	}, nil
}

// --- CoreTypeInfo JSON ---

// MarshalCoreTypeInfo serializes a CoreTypeInfo (map[uint64]Type) to JSON.
func MarshalCoreTypeInfo(cti CoreTypeInfo) ([]byte, error) {
	m := make(map[string]json.RawMessage)
	for id, t := range cti {
		raw, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("MarshalCoreTypeInfo(node %d): %w", id, err)
		}
		m[fmt.Sprintf("%d", id)] = raw
	}
	return json.Marshal(m)
}

// UnmarshalCoreTypeInfo deserializes a CoreTypeInfo from JSON.
func UnmarshalCoreTypeInfo(data []byte) (CoreTypeInfo, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	cti := make(CoreTypeInfo, len(m))
	for key, raw := range m {
		var id uint64
		if _, err := fmt.Sscanf(key, "%d", &id); err != nil {
			return nil, fmt.Errorf("UnmarshalCoreTypeInfo: invalid key %q: %w", key, err)
		}
		t, err := UnmarshalType(raw)
		if err != nil {
			return nil, fmt.Errorf("UnmarshalCoreTypeInfo(node %s): %w", key, err)
		}
		cti[id] = t
	}
	return cti, nil
}
