package protocol

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func validDescriptor(name string) ToolDescriptor {
	return ToolDescriptor{Name: name, Description: "description-" + name,
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`),
		OutputSchema: json.RawMessage(`{"type":"string"}`), Tags: []string{"tag-" + name}, Examples: []string{"example-" + name}}
}

func TestCallerSurfaceSortLookupAndDeepCopy(t *testing.T) {
	zeta := validDescriptor("zeta")
	alpha := validDescriptor("alpha")
	input := []ToolDescriptor{zeta, alpha}
	surface, err := CallerSurface(input)
	if err != nil {
		t.Fatal(err)
	}
	want := surface.All()
	if len(want) != 2 {
		t.Fatal("instrument failure: expected non-empty surface")
	}
	if want[0].Name != "alpha" || want[1].Name != "zeta" {
		t.Fatalf("surface order = [%s, %s]", want[0].Name, want[1].Name)
	}

	// Materialize the pre-mutation surface into bytes. Comparing two All() results
	// would be VACUOUS: under a shallow clone both alias the caller's storage, so
	// both change together and the comparison holds however broken the copy is
	// (measured — that exact mutation survived this test in its original form).
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	input[0].Name = "mutated"
	input[1].InputSchema[2] = 'X'
	input[1].OutputSchema[2] = 'X'
	input[1].Tags[0] = "mutated"
	input[1].Examples[0] = "mutated"
	gotJSON, err := json.Marshal(surface.All())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("surface changed after caller mutation:\n got %s\nwant %s", gotJSON, wantJSON)
	}

	// Mutating a returned slice must not reach the surface either. Compared against
	// a literal, not against another All() result, for the same reason.
	result := surface.All()
	result[0].Tags[0] = "result mutation"
	again, ok := surface.Lookup("alpha")
	if !ok || again.Tags[0] != "tag-alpha" {
		t.Fatalf("returned storage aliases surface: %#v", again)
	}
}

func TestCallerSurfaceRejectsUnsafeDescriptors(t *testing.T) {
	tests := []struct {
		name  string
		tools []ToolDescriptor
		want  string
	}{
		{"duplicate", []ToolDescriptor{validDescriptor("alpha"), validDescriptor("alpha")}, "duplicate"},
		{"nil input", []ToolDescriptor{{Name: "alpha"}}, "required"},
		{"scalar input", []ToolDescriptor{{Name: "alpha", InputSchema: json.RawMessage(`"scalar"`)}}, "invalid input"},
		{"non-object input", []ToolDescriptor{{Name: "alpha", InputSchema: json.RawMessage(`{"type":"array"}`)}}, "type object"},
		{"invalid input JSON", []ToolDescriptor{{Name: "alpha", InputSchema: json.RawMessage(`{`)}}, "invalid input"},
		{"invalid output JSON", []ToolDescriptor{{Name: "alpha", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{`)}}, "invalid output"},
		{"bad name", []ToolDescriptor{{Name: "bad name", InputSchema: json.RawMessage(`{"type":"object"}`)}}, "invalid"},
		{"long name", []ToolDescriptor{{Name: strings.Repeat("a", 65), InputSchema: json.RawMessage(`{"type":"object"}`)}}, "invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CallerSurface(test.tools); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CallerSurface() error = %v, want containing %q", err, test.want)
			}
		})
	}
}
