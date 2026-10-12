package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func verifyInputManifest(stage string, digests map[string]string) ([]string, error) {
	if _, present := digests["manifest.sha256"]; !present {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(stage, "manifest.sha256"))
	if err != nil {
		return nil, fmt.Errorf("manifest.sha256: %w", err)
	}
	seen := map[string]bool{}
	var verified []string
	for i, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if len(line) < 67 || line[64] != ' ' || (line[65] != ' ' && line[65] != '*') {
			return nil, fmt.Errorf("manifest.sha256 line %d: malformed checksum entry", i+1)
		}
		if _, err := hex.DecodeString(line[:64]); err != nil {
			return nil, fmt.Errorf("manifest.sha256 line %d: invalid digest", i+1)
		}
		name := line[66:]
		if name == "" || strings.HasSuffix(name, "/") {
			return nil, fmt.Errorf("manifest.sha256 line %d: missing file name", i+1)
		}
		if err := validateInputDataPath(name); err != nil {
			return nil, fmt.Errorf("manifest.sha256 line %d: %w", i+1, err)
		}
		if seen[name] {
			return nil, fmt.Errorf("manifest.sha256: duplicate entry %q", name)
		}
		seen[name] = true
		actual, exists := digests[name]
		if !exists {
			return nil, fmt.Errorf("manifest.sha256: missing entry file %q", name)
		}
		if !strings.EqualFold(actual, line[:64]) {
			return nil, fmt.Errorf("manifest.sha256: checksum mismatch for %q", name)
		}
		verified = append(verified, name)
	}
	return verified, nil
}
