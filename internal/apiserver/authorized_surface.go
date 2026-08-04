package apiserver

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ToolDescriptor is the internal, protocol-neutral form of a host tool.
// The public serveapi facade converts its descriptors to this type.
type ToolDescriptor struct {
	Name         string
	Description  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Tags         []string
	Examples     []string
}

// AuthorizedSurface is a validated, sorted, request-local tool surface.
// Its contents are detached from all caller-owned storage.
type AuthorizedSurface struct {
	tools  []ToolDescriptor
	byName map[string]int
}

// callerSurface copies and validates descriptors supplied by an embedding host.
func callerSurface(descriptors []ToolDescriptor) (*AuthorizedSurface, error) {
	tools := make([]ToolDescriptor, len(descriptors))
	for i, descriptor := range descriptors {
		tool := cloneToolDescriptor(descriptor)
		if err := validateToolDescriptor(tool); err != nil {
			return nil, fmt.Errorf("tool descriptor %d: %w", i, err)
		}
		tools[i] = tool
	}

	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	byName := make(map[string]int, len(tools))
	for i, tool := range tools {
		if _, exists := byName[tool.Name]; exists {
			return nil, fmt.Errorf("duplicate tool name %q", tool.Name)
		}
		byName[tool.Name] = i
	}
	return &AuthorizedSurface{tools: tools, byName: byName}, nil
}

func validateToolDescriptor(tool ToolDescriptor) error {
	if err := validateMCPName(tool.Name); err != nil {
		return err
	}
	if tool.InputSchema == nil {
		return fmt.Errorf("tool %q: input schema is required", tool.Name)
	}
	var input map[string]any
	if err := json.Unmarshal(tool.InputSchema, &input); err != nil {
		return fmt.Errorf("tool %q: invalid input schema: %w", tool.Name, err)
	}
	if input == nil || input["type"] != "object" {
		return fmt.Errorf("tool %q: input schema must have type object", tool.Name)
	}
	if tool.OutputSchema != nil {
		var output any
		if err := json.Unmarshal(tool.OutputSchema, &output); err != nil {
			return fmt.Errorf("tool %q: invalid output schema: %w", tool.Name, err)
		}
	}
	return nil
}

func cloneToolDescriptor(tool ToolDescriptor) ToolDescriptor {
	tool.InputSchema = append(json.RawMessage(nil), tool.InputSchema...)
	tool.OutputSchema = append(json.RawMessage(nil), tool.OutputSchema...)
	tool.Tags = append([]string(nil), tool.Tags...)
	tool.Examples = append([]string(nil), tool.Examples...)
	return tool
}

// Lookup returns a detached descriptor for name.
func (s *AuthorizedSurface) Lookup(name string) (ToolDescriptor, bool) {
	i, ok := s.byName[name]
	if !ok {
		return ToolDescriptor{}, false
	}
	return cloneToolDescriptor(s.tools[i]), true
}

// All returns a detached, sorted descriptor slice.
func (s *AuthorizedSurface) All() []ToolDescriptor {
	result := make([]ToolDescriptor, len(s.tools))
	for i, tool := range s.tools {
		result[i] = cloneToolDescriptor(tool)
	}
	return result
}
