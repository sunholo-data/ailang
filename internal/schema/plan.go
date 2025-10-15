// Package schema provides the plan schema for proactive architecture validation.
package schema

import (
	"encoding/json"
	"fmt"
)

// Plan represents a structured architecture plan for validation and scaffolding
type Plan struct {
	Schema       string       `json:"schema"`                 // "ailang.plan.v1"
	Goal         string       `json:"goal"`                   // Human-readable goal
	Modules      []ModulePlan `json:"modules"`                // Module structure
	Types        []TypePlan   `json:"types"`                  // Type definitions
	Functions    []FuncPlan   `json:"functions"`              // Function signatures
	Effects      []string     `json:"effects,omitempty"`      // Effects used across modules
	Dependencies []string     `json:"dependencies,omitempty"` // External dependencies
}

// ModulePlan describes a module's structure
type ModulePlan struct {
	Path    string   `json:"path"`    // e.g., "myapp/core"
	Exports []string `json:"exports"` // Names of exported items
	Imports []string `json:"imports"` // Import paths (e.g., "std/io")
}

// TypePlan describes a type to be defined
type TypePlan struct {
	Name       string `json:"name"`       // Type name (e.g., "Option")
	Kind       string `json:"kind"`       // "adt", "record", "alias"
	Definition string `json:"definition"` // AILANG type syntax
	Module     string `json:"module"`     // Module path where type is defined
}

// FuncPlan describes a function signature
type FuncPlan struct {
	Name    string   `json:"name"`              // Function name
	Type    string   `json:"type"`              // Type signature (e.g., "int -> int")
	Effects []string `json:"effects,omitempty"` // Effects (e.g., ["IO", "FS"])
	Module  string   `json:"module"`            // Module path where function is defined
}

// NewPlan creates a new plan with the correct schema version
func NewPlan(goal string) *Plan {
	return &Plan{
		Schema:    PlanV1,
		Goal:      goal,
		Modules:   []ModulePlan{},
		Types:     []TypePlan{},
		Functions: []FuncPlan{},
		Effects:   []string{},
	}
}

// ToJSON converts the plan to deterministic JSON
func (p *Plan) ToJSON() ([]byte, error) {
	data, err := MarshalDeterministic(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plan: %w", err)
	}
	return FormatJSON(data)
}

// FromJSON loads a plan from JSON bytes
func PlanFromJSON(data []byte) (*Plan, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	// Validate schema version
	if p.Schema != PlanV1 {
		return nil, fmt.Errorf("unsupported plan schema: %s (expected %s)", p.Schema, PlanV1)
	}

	return &p, nil
}

// AddModule adds a module to the plan
func (p *Plan) AddModule(path string, exports, imports []string) {
	p.Modules = append(p.Modules, ModulePlan{
		Path:    path,
		Exports: exports,
		Imports: imports,
	})
}

// AddType adds a type definition to the plan
func (p *Plan) AddType(name, kind, definition, module string) {
	p.Types = append(p.Types, TypePlan{
		Name:       name,
		Kind:       kind,
		Definition: definition,
		Module:     module,
	})
}

// AddFunction adds a function signature to the plan
func (p *Plan) AddFunction(name, typeSignature, module string, effects []string) {
	p.Functions = append(p.Functions, FuncPlan{
		Name:    name,
		Type:    typeSignature,
		Effects: effects,
		Module:  module,
	})
}

// AddEffect adds an effect to the plan's effect list
func (p *Plan) AddEffect(effect string) {
	// Avoid duplicates
	for _, e := range p.Effects {
		if e == effect {
			return
		}
	}
	p.Effects = append(p.Effects, effect)
}
