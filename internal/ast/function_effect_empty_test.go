package ast

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFuncTypeExplicitEmptyPresence(t *testing.T) {
	for _, tt := range []struct {
		name    string
		effects []EffectAnnotation
		present bool
	}{
		{"omitted", nil, false}, {"empty", []EffectAnnotation{}, true}, {"nonempty", []EffectAnnotation{{Name: "IO"}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fn := &FuncType{Params: []Type{&SimpleType{Name: "int"}}, Return: &SimpleType{Name: "int"}, Effects: tt.effects}
			if strings.Contains(fn.String(), "!") != tt.present {
				t.Fatalf("String lost effect presence: %s", fn.String())
			}
			for _, printed := range []string{Print(fn), Compact(fn)} {
				var data map[string]json.RawMessage
				if err := json.Unmarshal([]byte(printed), &data); err != nil {
					t.Fatal(err)
				}
				raw, ok := data["effects"]
				if ok != tt.present {
					t.Fatalf("JSON lost effect presence: %s", printed)
				}
				if tt.present && len(tt.effects) == 0 && string(raw) != "[]" {
					t.Fatalf("empty must be [], got %s", raw)
				}
			}
		})
	}
}
