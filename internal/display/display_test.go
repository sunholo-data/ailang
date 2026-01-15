package display

import (
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hello", 5, "hello"},
		{"hello", 4, "h..."},
		{"hello", 3, "..."},
		{"", 10, ""},
		{"hello", 0, "hello"},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"abc", "abc"},
		{"123456789012", "123456789012"},
		{"1234567890123456", "123456789012..."},
		{"a1b2c3d4-e5f6-g7h8-i9j0-k1l2m3n4o5p6", "a1b2c3d4-e5f..."},
	}

	for _, tt := range tests {
		got := TruncateID(tt.id)
		if got != tt.want {
			t.Errorf("TruncateID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text  string
		width int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello\nworld"},
		{"hello\nworld", 10, "hello\nworld"},
		{"a very long line that needs wrapping", 15, "a very long\nline that needs\nwrapping"},
	}

	for _, tt := range tests {
		got := WrapText(tt.text, tt.width)
		if got != tt.want {
			t.Errorf("WrapText(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		when time.Time
		want string
	}{
		{now, "just now"},
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-2 * time.Hour), "2h ago"},
		{now.Add(-3 * 24 * time.Hour), "3d ago"},
	}

	for _, tt := range tests {
		got := FormatAge(tt.when)
		if got != tt.want {
			t.Errorf("FormatAge(%v) = %q, want %q", tt.when, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   float64
		want string
	}{
		{0, "-"},
		{100, "100ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{30000, "30.0s"},
		{60000, "1.0m"},
		{120000, "2.0m"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestTaskStatusDisplay(t *testing.T) {
	tests := []struct {
		status string
		label  string
		icon   string
	}{
		{"pending", "Pending", "○"},
		{"running", "Running", "▶"},
		{"completed", "Completed", "✓"},
		{"failed", "Failed", "✗"},
		{"pending_approval", "Awaiting Approval", "⏳"},
		{"unknown", "unknown", "?"},
	}

	for _, tt := range tests {
		got := TaskStatusDisplay(tt.status)
		if got.Label != tt.label {
			t.Errorf("TaskStatusDisplay(%q).Label = %q, want %q", tt.status, got.Label, tt.label)
		}
		if got.Icon != tt.icon {
			t.Errorf("TaskStatusDisplay(%q).Icon = %q, want %q", tt.status, got.Icon, tt.icon)
		}
	}
}

func TestPrettyJSON(t *testing.T) {
	input := `{"name":"test","value":123}`
	got := PrettyJSON(input)
	if got == input {
		t.Error("PrettyJSON should add indentation")
	}
	if len(got) <= len(input) {
		t.Error("PrettyJSON output should be longer than compact input")
	}
}

func TestCompactJSON(t *testing.T) {
	input := `{
  "name": "test",
  "value": 123
}`
	want := `{"name":"test","value":123}`
	got := CompactJSON(input)
	if got != want {
		t.Errorf("CompactJSON() = %q, want %q", got, want)
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"foo": "bar"}`, true},
		{`[1, 2, 3]`, true},
		{`hello`, false},
		{``, false},
		{`  { "x": 1 }  `, true}, // whitespace is trimmed, then it's valid JSON
	}

	for _, tt := range tests {
		got := IsJSON(tt.input)
		if got != tt.want {
			t.Errorf("IsJSON(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFormatJSONValue(t *testing.T) {
	tests := []struct {
		val    interface{}
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{float64(42), 10, "42"},
		{float64(3.14), 10, "3.14"},
		{true, 10, "true"},
		{false, 10, "false"},
		{nil, 10, "null"},
	}

	for _, tt := range tests {
		got := FormatJSONValue(tt.val, tt.maxLen)
		if got != tt.want {
			t.Errorf("FormatJSONValue(%v, %d) = %q, want %q", tt.val, tt.maxLen, got, tt.want)
		}
	}
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"/Users/mark/dev/sunholo/ailang/file.go", 20, "/Users/m...file.go"},
		{"short", 10, "short"},
	}

	for _, tt := range tests {
		got := TruncateMiddle(tt.input, tt.maxLen)
		if len(got) > tt.maxLen {
			t.Errorf("TruncateMiddle(%q, %d) = %q (len %d), should be <= %d",
				tt.input, tt.maxLen, got, len(got), tt.maxLen)
		}
	}
}
