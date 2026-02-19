package messaging

import (
	"encoding/json"
	"fmt"
)

// EnvelopeSlot names the 5 semantic embedding spaces in a message envelope.
const (
	SlotIntent     = "intent"     // What is being asked? Auto-computed from title + payload prefix.
	SlotCode       = "code"       // What code is affected? From file paths + code snippets.
	SlotContext    = "context"    // What was the sender working on? From recent files, errors, tools.
	SlotSkill      = "skill"      // What expertise is needed? From compiler phases, AST nodes, file patterns.
	SlotResolution = "resolution" // How was this resolved? From git diff + commit message (post-completion).
)

// AllSlots lists every valid envelope slot name.
var AllSlots = []string{SlotIntent, SlotCode, SlotContext, SlotSkill, SlotResolution}

// EnvelopeVector holds a single named embedding vector with its model metadata.
type EnvelopeVector struct {
	Vector    []float32 `json:"vector"`
	Model     string    `json:"model"`     // e.g. "ollama:nomic-embed-text"
	Dimension int       `json:"dimension"` // e.g. 768
}

// Envelope holds named embedding vectors for a message, each capturing a
// different aspect of the message's meaning.
//
// Slots are optional — most messages will only have "intent" (auto-computed).
// Other slots are populated explicitly via EnvelopeBuilder options or CLI flags.
type Envelope struct {
	Slots map[string]*EnvelopeVector `json:"slots"`
}

// NewEnvelope creates an empty envelope.
func NewEnvelope() *Envelope {
	return &Envelope{Slots: make(map[string]*EnvelopeVector)}
}

// IsEmpty returns true if the envelope has no populated slots.
func (e *Envelope) IsEmpty() bool {
	if e == nil {
		return true
	}
	return len(e.Slots) == 0
}

// Get returns the vector for the named slot, or nil if not present.
func (e *Envelope) Get(slot string) *EnvelopeVector {
	if e == nil || e.Slots == nil {
		return nil
	}
	return e.Slots[slot]
}

// GetVector returns just the float32 slice for the named slot, or nil.
func (e *Envelope) GetVector(slot string) []float32 {
	v := e.Get(slot)
	if v == nil {
		return nil
	}
	return v.Vector
}

// Set stores a vector in the named slot.
func (e *Envelope) Set(slot string, vector []float32, model string) {
	if e.Slots == nil {
		e.Slots = make(map[string]*EnvelopeVector)
	}
	e.Slots[slot] = &EnvelopeVector{
		Vector:    vector,
		Model:     model,
		Dimension: len(vector),
	}
}

// Merge copies non-nil slots from other into e without overwriting existing slots.
func (e *Envelope) Merge(other *Envelope) {
	if other == nil || other.Slots == nil {
		return
	}
	if e.Slots == nil {
		e.Slots = make(map[string]*EnvelopeVector)
	}
	for slot, vec := range other.Slots {
		if _, exists := e.Slots[slot]; !exists {
			e.Slots[slot] = vec
		}
	}
}

// MergeOverwrite copies all non-nil slots from other into e, overwriting existing slots.
func (e *Envelope) MergeOverwrite(other *Envelope) {
	if other == nil || other.Slots == nil {
		return
	}
	if e.Slots == nil {
		e.Slots = make(map[string]*EnvelopeVector)
	}
	for slot, vec := range other.Slots {
		e.Slots[slot] = vec
	}
}

// PopulatedSlots returns the names of slots that have vectors.
func (e *Envelope) PopulatedSlots() []string {
	if e == nil || e.Slots == nil {
		return nil
	}
	var names []string
	for slot := range e.Slots {
		names = append(names, slot)
	}
	return names
}

// ValidateSlot checks that a slot name is one of the 5 known slots.
func ValidateSlot(slot string) error {
	for _, s := range AllSlots {
		if s == slot {
			return nil
		}
	}
	return fmt.Errorf("unknown envelope slot %q: valid slots are %v", slot, AllSlots)
}

// ToJSON serializes the envelope to a JSON string for SQLite storage.
func (e *Envelope) ToJSON() string {
	if e == nil || e.IsEmpty() {
		return "{}"
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// EnvelopeFromJSON parses an envelope from a JSON string.
// Returns an empty envelope (not nil) for empty/invalid input.
func EnvelopeFromJSON(data string) *Envelope {
	if data == "" || data == "{}" {
		return NewEnvelope()
	}
	var env Envelope
	if err := json.Unmarshal([]byte(data), &env); err != nil {
		return NewEnvelope()
	}
	if env.Slots == nil {
		env.Slots = make(map[string]*EnvelopeVector)
	}
	return &env
}

// DimensionMatch checks if two envelope vectors have compatible dimensions.
// Returns an error if dimensions differ, nil if compatible or either is nil.
func DimensionMatch(a, b *EnvelopeVector) error {
	if a == nil || b == nil {
		return nil
	}
	if a.Dimension != b.Dimension {
		return fmt.Errorf("dimension mismatch: %d (%s) vs %d (%s)", a.Dimension, a.Model, b.Dimension, b.Model)
	}
	return nil
}
