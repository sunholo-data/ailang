package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenErrorCodes_AllCodesPresent(t *testing.T) {
	// Run against the real codes.go
	codesPath := filepath.Join("..", "..", "internal", "errors", "codes.go")
	records, err := parseErrorCodes(codesPath)
	if err != nil {
		t.Fatalf("parseErrorCodes: %v", err)
	}

	// Read the source file and verify every constant appears in the output
	src, err := os.ReadFile(codesPath)
	if err != nil {
		t.Fatalf("reading codes.go: %v", err)
	}

	// Build a map from the parsed records
	byCode := make(map[string]ErrorRecord, len(records))
	for _, r := range records {
		byCode[r.Code] = r
	}

	// Count constants defined in the file
	constantCount := 0
	lines := strings.Split(string(src), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for lines like: PAR001 = "PAR001"
		if len(trimmed) > 5 && !strings.HasPrefix(trimmed, "//") {
			if idx := strings.Index(trimmed, ` = "`); idx > 0 {
				name := strings.TrimSpace(trimmed[:idx])
				// Validate it looks like an error code (letters + digits)
				if isErrorCodeName(name) {
					constantCount++
					if _, found := byCode[name]; !found {
						t.Errorf("constant %s defined in codes.go but missing from output", name)
					}
				}
			}
		}
	}

	if constantCount == 0 {
		t.Error("no constants found in codes.go — check parser")
	}
	if len(records) < constantCount {
		t.Errorf("output has %d records but codes.go has %d constants", len(records), constantCount)
	}
}

func TestGenErrorCodes_SchemaValid(t *testing.T) {
	codesPath := filepath.Join("..", "..", "internal", "errors", "codes.go")
	records, err := parseErrorCodes(codesPath)
	if err != nil {
		t.Fatalf("parseErrorCodes: %v", err)
	}

	for _, r := range records {
		if r.Code == "" {
			t.Errorf("record has empty Code: %+v", r)
		}
		if r.Category == "" {
			t.Errorf("record %s has empty Category", r.Code)
		}
		if r.Summary == "" {
			t.Errorf("record %s has empty Summary", r.Code)
		}
		if r.FixHint == "" {
			t.Errorf("record %s has empty FixHint", r.Code)
		}
	}
}

func TestGenErrorCodes_JSONRoundtrip(t *testing.T) {
	codesPath := filepath.Join("..", "..", "internal", "errors", "codes.go")
	records, err := parseErrorCodes(codesPath)
	if err != nil {
		t.Fatalf("parseErrorCodes: %v", err)
	}

	out := ErrorCodesOutput{
		SchemaVersion: "v1",
		Records:       records,
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded ErrorCodesOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.SchemaVersion != "v1" {
		t.Errorf("schema_version: got %s, want v1", decoded.SchemaVersion)
	}
	if len(decoded.Records) != len(records) {
		t.Errorf("records count: got %d, want %d", len(decoded.Records), len(records))
	}
}

func isErrorCodeName(s string) bool {
	if len(s) < 4 || len(s) > 8 {
		return false
	}
	// Must start with 2-4 uppercase letters then 2-4 digits
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i < 2 || i > 4 {
		return false
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	return j == len(s) && (j-i) >= 2
}
