package types

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// Gob codec for TLabelled (M-TAINT-TYPES).
//
// Labels cannot ride the default gob path: the four Label implementations
// (labelBottom, labelConst, labelVar, labelJoin) are unexported value types with
// unexported fields, and gob refuses both. Registering them is therefore not
// enough — the encoder would find the type and then serialise nothing.
//
// Rather than exporting the label representation purely to satisfy a codec, this
// delegates to the tagged-union JSON codec that already exists for ifaces
// (MarshalLabel / UnmarshalLabel in label_json.go), so there is one definition of
// how a label serialises and gob and JSON cannot drift apart.
//
// Without this, every compile of an IFC-labelled module failed its cache write
// with "gob: type not registered for interface: types.TLabelled" and silently
// fell back to fresh compilation — reported as inbox_1788847828665_f0c5db58.

// labelledGob is the wire form: the inner type through the normal gob path, the
// label through the JSON codec.
type labelledGob struct {
	Inner Type
	Label []byte
}

// GobEncode implements gob.GobEncoder.
func (t *TLabelled) GobEncode() ([]byte, error) {
	labelBytes, err := MarshalLabel(t.L)
	if err != nil {
		return nil, fmt.Errorf("TLabelled: encoding label: %w", err)
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(labelledGob{Inner: t.Inner, Label: labelBytes}); err != nil {
		return nil, fmt.Errorf("TLabelled: encoding inner type: %w", err)
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (t *TLabelled) GobDecode(data []byte) error {
	var w labelledGob
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&w); err != nil {
		return fmt.Errorf("TLabelled: decoding inner type: %w", err)
	}
	label, err := UnmarshalLabel(w.Label)
	if err != nil {
		return fmt.Errorf("TLabelled: decoding label: %w", err)
	}
	t.Inner = w.Inner
	t.L = label
	return nil
}
