package protocol

import (
	"encoding/json"
	"testing"
)

func TestResolveToolHints(t *testing.T) {
	cases := []struct {
		name     string
		words    []string
		declared bool
		pure     bool
		want     string // JSON of the resolved annotations; "null" = none
		wantErr  bool
	}{
		{"pure undeclared is read-only closed-world", nil, false, true,
			`{"readOnlyHint":true,"destructiveHint":false,"idempotentHint":false,"openWorldHint":false}`, false},
		{"effectful undeclared is not guessed", nil, false, false, `null`, false},
		{"readOnly states destructiveHint false", []string{"readOnly", "openWorld"}, true, false,
			`{"readOnlyHint":true,"destructiveHint":false,"idempotentHint":false,"openWorldHint":true}`, false},
		{"absent destructive means additive", []string{"openWorld"}, true, false,
			`{"readOnlyHint":false,"destructiveHint":false,"idempotentHint":false,"openWorldHint":true}`, false},
		{"destructive idempotent closed-world", []string{"destructive", "idempotent"}, true, false,
			`{"readOnlyHint":false,"destructiveHint":true,"idempotentHint":true,"openWorldHint":false}`, false},
		{"declared overrides pure default", []string{"destructive"}, true, true,
			`{"readOnlyHint":false,"destructiveHint":true,"idempotentHint":false,"openWorldHint":false}`, false},
		{"unknown word", []string{"readonly"}, true, false, ``, true},
		{"duplicate word", []string{"readOnly", "readOnly"}, true, false, ``, true},
		{"contradiction", []string{"readOnly", "destructive"}, true, false, ``, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveToolHints(c.words, c.declared, c.pure)
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(got)
			if string(b) != c.want {
				t.Errorf("got %s\nwant %s", b, c.want)
			}
		})
	}
}

// CallerSurface clones descriptors; the hint pointers must not alias the
// caller's, or a host mutating its descriptor after registration would change
// what clients were told.
func TestCallerSurfaceClonesAnnotations(t *testing.T) {
	hints, _ := ResolveToolHints([]string{"destructive"}, true, false)
	in := ToolDescriptor{Name: "t", Title: "T", InputSchema: json.RawMessage(`{"type":"object"}`), Annotations: hints}
	surface, err := CallerSurface([]ToolDescriptor{in})
	if err != nil {
		t.Fatal(err)
	}
	*hints.DestructiveHint = false
	hints.ReadOnlyHint = true
	got, _ := surface.Lookup("t")
	if got.Title != "T" || got.Annotations == nil || got.Annotations.ReadOnlyHint || !*got.Annotations.DestructiveHint {
		t.Errorf("surface annotations aliased the caller's: %+v", got.Annotations)
	}
}
