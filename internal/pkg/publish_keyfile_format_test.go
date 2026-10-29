package pkg

import (
	"strings"
	"testing"
)

func TestPublishKeyFile_Format(t *testing.T) {
	for _, tt := range []struct {
		name, input string
		valid       bool
	}{
		{"trim", " \tdummy-file-key\n", true}, {"empty", " \n\t", false},
		{"multiline", "dummy-first\ndummy-second", false}, {"CR", "dummy\rkey", false},
		{"NUL", "dummy\x00key", false}, {"TAB", "dummy\tkey", false}, {"DEL", "dummy\x7fkey", false},
		{"limit", strings.Repeat("x", 16*1024), true}, {"too-large", strings.Repeat("x", 16*1024+1), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key, err := parsePublishKey([]byte(tt.input))
			if (err == nil) != tt.valid {
				t.Fatalf("format result valid=%v err=%v", tt.valid, err)
			}
			if tt.name == "trim" && key != "dummy-file-key" {
				t.Fatal("trimmed value changed")
			}
			if err != nil && (strings.Contains(err.Error(), "dummy") || key != "") {
				t.Fatal("invalid credential exposed")
			}
		})
	}
}
