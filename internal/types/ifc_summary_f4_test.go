package types_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

const f4SummaryJSON = `{"params":[{"name":"input","label":{"constants":[],"variables":[]},"forbidden":["email","secret"],"applied":{"constants":[],"variables":[]}}],"return":{"constants":[],"variables":[]},"has_return_label":false,"intrinsic":{"constants":[],"variables":[]},"declassify":[]}`

func TestIFCSummaryF4CanonicalShape(t *testing.T) {
	var s types.ImportedIFCSig
	if err := json.Unmarshal([]byte(f4SummaryJSON), &s); err != nil {
		t.Fatalf("valid F4 metadata rejected: %v", err)
	}
	raw, err := json.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var before, after any
	if err := json.Unmarshal([]byte(f4SummaryJSON), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("lossy F4 roundtrip: %s", raw)
	}
}

func TestIFCSummaryF4RejectsMalformedParam(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing forbidden", func(p map[string]any) { delete(p, "forbidden") }},
		{"null forbidden", func(p map[string]any) { p["forbidden"] = nil }},
		{"old string forbidden", func(p map[string]any) { p["forbidden"] = "secret" }},
		{"duplicate forbidden", func(p map[string]any) { p["forbidden"] = []string{"email", "email"} }},
		{"unsorted forbidden", func(p map[string]any) { p["forbidden"] = []string{"secret", "email"} }},
		{"reserved forbidden", func(p map[string]any) { p["forbidden"] = []string{"*"} }},
		{"missing applied", func(p map[string]any) { delete(p, "applied") }},
		{"null applied", func(p map[string]any) { p["applied"] = nil }},
		{"missing applied constituent", func(p map[string]any) { p["applied"] = map[string]any{"constants": []string{}} }},
		{"duplicate applied", func(p map[string]any) {
			p["applied"] = map[string]any{"constants": []string{"email", "email"}, "variables": []string{}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(f4SummaryJSON), &raw); err != nil {
				t.Fatal(err)
			}
			tc.mutate(raw["params"].([]any)[0].(map[string]any))
			data, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			var s types.ImportedIFCSig
			if err := json.Unmarshal(data, &s); err == nil {
				t.Fatalf("malformed metadata accepted: %s", data)
			}
		})
	}
}
