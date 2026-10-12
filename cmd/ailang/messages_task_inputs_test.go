package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTaskInputsFile(t *testing.T) {
	if inputs, err := loadTaskInputsFile(""); err != nil || len(inputs) != 0 {
		t.Fatalf("empty: %v %v", inputs, err)
	}
	filename := filepath.Join(t.TempDir(), "inputs.json")
	for _, tt := range []struct {
		raw   string
		valid bool
	}{{`[{"repo":"a/b","ref":"main","path":"poster.png"}]`, true}, {`[{"repo":"a/b","ref":"main","path":"../secret"}]`, false}, {`{}`, false}, {`[] []`, false}} {
		if err := os.WriteFile(filename, []byte(tt.raw), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := loadTaskInputsFile(filename)
		if (err == nil) != tt.valid {
			t.Fatalf("%q: %v", tt.raw, err)
		}
	}
	if _, err := loadTaskInputsFile(filename + "missing"); err == nil {
		t.Fatal("accepted missing file")
	}
}
