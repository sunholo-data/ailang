package core

import "testing"

func TestDisplayFuncName(t *testing.T) {
	for in, want := range map[string]string{
		"inc":            "inc",
		"inc$numF":       "inc",
		"qi$numFI":       "qi",
		"f$numF$lambda3": "f$lambda3",
		"h$numF$s2":      "h$s2",
		"f$lambda1":      "f$lambda1",
		"number":         "number",
	} {
		if got := DisplayFuncName(in); got != want {
			t.Errorf("DisplayFuncName(%q) = %q, want %q", in, got, want)
		}
	}
}
