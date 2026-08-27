package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkGitPromptFreezeInvariants(repoRoot string, source, mirror *orderedRegistry, violations []string) ([]string, error) {
	base, err := gitOutput(repoRoot, "merge-base", "HEAD", "origin/dev")
	if err != nil {
		return nil, fmt.Errorf("git merge-base HEAD origin/dev: %w", err)
	}
	baseRegistryBytes, err := gitBytes(repoRoot, "show", base+":prompts/versions.json")
	if err != nil {
		return nil, fmt.Errorf("git show merge-base prompt registry: %w", err)
	}
	baseRegistry, err := decodeRegistryEntries(baseRegistryBytes)
	if err != nil {
		return nil, fmt.Errorf("decode merge-base prompt registry: %w", err)
	}

	for id, baseEntry := range baseRegistry {
		if baseEntry.Frozen == nil {
			continue
		}
		current := source.Versions[id]
		if current == nil || current.File != baseEntry.File || current.Hash != baseEntry.Hash {
			violations = append(violations, fmt.Sprintf("frozen version %s: immutability violation in registry file/hash fields", id))
			continue
		}
		baseBytes, showErr := gitBytes(repoRoot, "show", base+":"+baseEntry.File)
		if showErr != nil {
			return nil, fmt.Errorf("git show merge-base prompt %s: %w", id, showErr)
		}
		currentBytes, readErr := os.ReadFile(filepath.Join(repoRoot, current.File))
		if readErr != nil {
			return nil, readErr
		}
		if !bytes.Equal(baseBytes, currentBytes) {
			violations = append(violations, fmt.Sprintf("frozen version %s: immutability violation in %s bytes", id, current.File))
		}
	}

	for _, id := range source.VersionKeys {
		entry := source.Versions[id]
		if entry.Frozen == nil {
			continue
		}
		mirrorEntry := mirror.Versions[id]
		if !sameFrozenRegistryEntry(entry, mirrorEntry) {
			// checkRegistries normally reports this first; retain this check here so
			// L3(d) remains complete when this helper is called independently.
			needle := fmt.Sprintf("cmd/ailang/prompts/versions.json: entry %s differs from source", id)
			if !containsViolation(violations, needle) {
				violations = append(violations, needle)
			}
		}
		mirrorPath := filepath.Join("cmd", "ailang", entry.File)
		sourceBytes, sourceErr := os.ReadFile(filepath.Join(repoRoot, entry.File))
		mirrorBytes, mirrorErr := os.ReadFile(filepath.Join(repoRoot, mirrorPath))
		if sourceErr != nil {
			return nil, sourceErr
		}
		if mirrorErr != nil || !bytes.Equal(sourceBytes, mirrorBytes) {
			violations = append(violations, fmt.Sprintf("frozen version %s: mirror bytes differ at %s", id, filepath.ToSlash(mirrorPath)))
		}
	}
	return violations, nil
}

func gitOutput(repoRoot string, args ...string) (string, error) {
	out, err := gitBytes(repoRoot, args...)
	return strings.TrimSpace(string(out)), err
}

func gitBytes(repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func decodeRegistryEntries(data []byte) (map[string]*registryEntry, error) {
	var registry struct {
		Versions map[string]*registryEntry `json:"versions"`
	}
	err := json.Unmarshal(data, &registry)
	return registry.Versions, err
}

func sameFrozenRegistryEntry(a, b *registryEntry) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.File == b.File && a.Hash == b.Hash && frozenMarkersEqual(a.Frozen, b.Frozen)
}

func frozenMarkersEqual(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}

func containsViolation(violations []string, want string) bool {
	for _, violation := range violations {
		if violation == want {
			return true
		}
	}
	return false
}
