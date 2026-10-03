package pkg

import "testing"

func TestIsNamedTestBodyFile(t *testing.T) {
	for name, want := range map[string]bool{
		"_namedtest_body_607890236.ail": true,
		"_namedtest_body_1.ail":         true,
		"_namedtest_body_.ail":          false,
		"_namedtest_body_readme.ail":    false,
		"_namedtest_body_12.md":         false,
		"x_test.ail":                    false,
		"_smoke.ail":                    false,
	} {
		if got := IsNamedTestBodyFile(name); got != want {
			t.Errorf("IsNamedTestBodyFile(%q) = %v, want %v", name, got, want)
		}
	}
}
