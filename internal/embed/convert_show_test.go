package embed

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ToGo is the converter serve-api, MCP and A2A use for an entry's result (the
// API-level check is TestFunctionCall/list_of_strings_is_JSON in
// internal/apiserver). It builds structured JSON
// rather than show text, so M-V1-SHOW-TOTAL does not change results: nested
// strings stay JSON strings (not `"\"a\""`), floats stay numbers, and nothing
// is elided. Only text a program builds itself with show changes.
func TestResultJSONDoesNotGoThroughShow(t *testing.T) {
	long := make([]eval.Value, 60)
	for i := range long {
		long[i] = &eval.IntValue{Value: i}
	}
	v := &eval.RecordValue{Fields: map[string]eval.Value{
		"names": &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "a"}, &eval.StringValue{Value: "b, c"}}},
		"big":   &eval.FloatValue{Value: 1e21},
		"long":  &eval.ListValue{Elements: long},
		"shown": &eval.StringValue{Value: eval.Show(&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "x"}}})},
	}}
	g, err := ToGo(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]interface{}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	names, _ := back["names"].([]interface{})
	if len(names) != 2 || names[0] != "a" || names[1] != "b, c" {
		t.Fatalf("names = %#v (%s)", back["names"], raw)
	}
	if back["big"] != 1e21 {
		t.Fatalf("big = %#v", back["big"])
	}
	if l, _ := back["long"].([]interface{}); len(l) != 60 {
		t.Fatalf("long has %d elements", len(l))
	}
	// A string the program built with show is carried as data, verbatim.
	if back["shown"] != `["x"]` {
		t.Fatalf("shown = %#v", back["shown"])
	}
}
