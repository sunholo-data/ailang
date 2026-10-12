package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func TestUnionDeclassifyRequirements(t *testing.T) {
	row := func(scope string) *types.Row {
		r := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Declassify": types.Unit()}}
		if scope != "" {
			r.Params = map[string]map[string]string{"Declassify": {"label": scope}}
		}
		return r
	}
	for _, scopes := range [][2]string{{"", "email"}, {"email", ""}, {"email", "secret"}, {"secret", "email"}} {
		merged := unionRequiredEffectRows(row(scopes[0]), row(scopes[1]))
		if types.SubsumeEffectRows(merged, row("email")) {
			t.Fatalf("narrowed %v: %#v", scopes, merged.Params)
		}
		if !types.SubsumeEffectRows(merged, row("")) {
			t.Fatalf("bare did not cover %v", scopes)
		}
	}
}

func TestBareDeclassifyMismatchRendersUnscoped(t *testing.T) {
	bare := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Declassify": types.Unit()}}
	scoped := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Declassify": types.Unit()},
		Params: map[string]map[string]string{"Declassify": {"label": "email"}}}
	err := formatLambdaEffectError("fixture:1:1", bare, scoped)
	want := "Declassify requires unscoped Declassify; declaration provides label=email"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("diagnostic %q does not contain %q", err, want)
	}
	if strings.Contains(err.Error(), "label=;") {
		t.Fatalf("diagnostic rendered an empty label value: %q", err)
	}
}
