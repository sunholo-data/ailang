package pkg

import "testing"

func TestIsAbsoluteCrossPlatform(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Unix absolute
		{"/etc/passwd", true},
		{"/", true},
		{"/foo/bar", true},

		// Windows drive letters (both separators)
		{`C:\Windows\System32`, true},
		{"C:/Windows/System32", true},
		{"D:\\foo", true},
		{"z:/foo", true}, // lower-case drive letter
		{"C:", true},

		// Windows UNC
		{`\\server\share`, true},
		{"//server/share", true},

		// Relative paths — must NOT match
		{"foo.txt", false},
		{"foo/bar.txt", false},
		{"./foo.txt", false},
		{"../foo.txt", false},
		{".", false},
		{"", false},

		// Edge case: single-letter file like "c" is relative, not a drive
		{"c", false},
		// "c:" alone IS a Windows drive reference — current dir on drive C
		{"c:", true},
		// Non-letter colon prefix — not a drive
		{"1:foo", false},
		{":foo", false},
	}
	for _, tc := range cases {
		got := IsAbsoluteCrossPlatform(tc.path)
		if got != tc.want {
			t.Errorf("IsAbsoluteCrossPlatform(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
