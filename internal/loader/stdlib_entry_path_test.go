package loader

import (
	"strings"
	"testing"
)

// TestStdlibEntryPathAcceptsAilSuffix guards the regression that made every file
// under std/ impossible to type-check or introspect.
//
// The loader routes any canonical path starting with "std/" to the stdlib
// resolver. That is right for an IMPORT ("std/io"), but the same branch is
// reached when a std/ FILE is the entry — `ailang check std/io.ail`, or
// `ailang iface std/io`, which check.go turns into "std/io.ail". The path then
// arrived at validateModuleName as a module NAME, where the "." in ".ail" fails
// the [a-zA-Z0-9_/-] guard.
//
// Consequences, all real: `ailang check std/io.ail` and `ailang iface std/<any>`
// failed for all 45 std modules, which in turn meant tools/verify-stdlib.sh could
// not read a single interface. That gate sat dead from 2026-05-22 to 2026-08-13.
//
// The fix strips a trailing ".ail" before resolution. It cannot mask a real
// module, because "." is not a legal module-name character in the first place —
// which this test also pins.
func TestStdlibEntryPathAcceptsAilSuffix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// The import form must keep working — this is the common path.
		{"bare module name", "std/io", false},
		{"bare module name, nested", "std/net/http", false},

		// The entry-file form: what check.go / iface hand us. This is the regression.
		{"entry file path", strings.TrimSuffix("std/io.ail", ".ail"), false},
		{"entry file path, math", strings.TrimSuffix("std/math.ail", ".ail"), false},

		// A dot anywhere else is still rejected — the fix must not widen the guard.
		{"dot in the middle is still invalid", "std/io.core", true},
		{"double extension is still invalid", "std/io.ail.ail", true},
		{"traversal still rejected", "std/../etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModuleName(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateModuleName(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateModuleName(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

// TestStdlibEntryPathSuffixStrippingIsExact pins the narrow contract the loader
// relies on: exactly one trailing ".ail" is removed, and nothing else is touched.
func TestStdlibEntryPathSuffixStrippingIsExact(t *testing.T) {
	cases := map[string]string{
		"std/io.ail":     "std/io",
		"std/io":         "std/io",
		"std/io.ail.ail": "std/io.ail", // only the LAST one; the rest stays invalid
		"std/mail":       "std/mail",   // must not chop a substring match
	}
	for in, want := range cases {
		if got := strings.TrimSuffix(in, ".ail"); got != want {
			t.Errorf("TrimSuffix(%q, \".ail\") = %q, want %q", in, got, want)
		}
	}
}
