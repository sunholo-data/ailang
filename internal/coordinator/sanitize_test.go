package coordinator

import "testing"

func TestSanitizeLog(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello world", "hello world"},
		{"strip LF", "hello\nworld", "helloworld"},
		{"strip CR", "hello\rworld", "helloworld"},
		{"strip CRLF", "hello\r\nworld", "helloworld"},
		{"strip tab", "hello\tworld", "helloworld"},
		{"strip NUL", "hello\x00world", "helloworld"},
		{"strip ESC", "hello\x1bworld", "helloworld"},
		{"forged log line", "task-1\n[ERROR] fake log entry", "task-1[ERROR] fake log entry"},
		{"unicode passthrough", "café résumé 中文 🚀", "café résumé 中文 🚀"},
		{"mixed", "id:1\nkey:2\tvalue:3", "id:1key:2value:3"},
		{"high ascii preserved", string([]byte{0x20, 0x7e, 0x7f}), string([]byte{0x20, 0x7e, 0x7f})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeLog(tc.in)
			if got != tc.want {
				t.Errorf("SanitizeLog(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
