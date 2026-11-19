package core

import "testing"

func TestDisplayFuncName(t *testing.T) {
	for in, want := range map[string]string{
		"inc":            "inc",
		"inc$numF":       "inc",
		"qi$numFI":       "qi",
		"f$numF$lambda3": "f$lambda3",
		"h$numF$s2":      "h",
		"h$s2":           "h",
		"go$s10$lambda4": "go$lambda4",
		"cost$sum":       "cost$sum",
		"f$lambda1":      "f$lambda1",
		"number":         "number",
	} {
		if got := DisplayFuncName(in); got != want {
			t.Errorf("DisplayFuncName(%q) = %q, want %q", in, got, want)
		}
	}
}
