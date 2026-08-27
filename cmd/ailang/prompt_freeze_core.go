package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

type registryPair struct{ Source, Mirror string }

type registryEntry struct {
	File, Hash, Description, Created string
	Tags                             []string
	Notes                            string
	Frozen                           *eval_harness.FrozenMarker
}

type orderedRegistry struct {
	SchemaVersion string
	VersionKeys   []string
	Versions      map[string]*registryEntry
	RawEntries    map[string]json.RawMessage
	Active        string
	Notes         []string
}

type registryJSON struct {
	SchemaVersion string                    `json:"schema_version"`
	Versions      map[string]*registryEntry `json:"versions"`
	Active        string                    `json:"active"`
	Notes         []string                  `json:"notes"`
}

func (e *registryEntry) UnmarshalJSON(data []byte) error {
	type wire struct {
		File, Hash, Description, Created string
		Tags                             []string
		Notes                            string
		Frozen                           *eval_harness.FrozenMarker
	}
	var w struct {
		File        string                     `json:"file"`
		Hash        string                     `json:"hash"`
		Description string                     `json:"description"`
		Created     string                     `json:"created"`
		Tags        []string                   `json:"tags"`
		Notes       string                     `json:"notes"`
		Frozen      *eval_harness.FrozenMarker `json:"frozen"`
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*e = registryEntry{w.File, w.Hash, w.Description, w.Created, w.Tags, w.Notes, w.Frozen}
	return nil
}

func loadOrderedRegistry(path string) (*orderedRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		SchemaVersion string          `json:"schema_version"`
		Versions      json.RawMessage `json:"versions"`
		Active        string          `json:"active"`
		Notes         []string        `json:"notes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var versions map[string]*registryEntry
	var rawEntries map[string]json.RawMessage
	if err := json.Unmarshal(raw.Versions, &versions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw.Versions, &rawEntries); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw.Versions))
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(versions))
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return &orderedRegistry{raw.SchemaVersion, keys, versions, rawEntries, raw.Active, raw.Notes}, nil
}

func writeOrderedRegistry(path string, r *orderedRegistry) error {
	var b bytes.Buffer
	b.WriteString("{\n  \"schema_version\": ")
	writeJSON(&b, r.SchemaVersion)
	b.WriteString(",\n  \"versions\": {")
	for i, k := range r.VersionKeys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n    ")
		writeJSON(&b, k)
		b.WriteString(": ")
		entryBytes, err := marshalEntryPreserving(r.RawEntries[k], r.Versions[k])
		if err != nil {
			return err
		}
		lines := strings.Split(string(entryBytes), "\n")
		b.WriteString(lines[0])
		for _, line := range lines[1:] {
			b.WriteString("\n    ")
			b.WriteString(line)
		}
	}
	b.WriteString("\n  },\n  \"active\": ")
	writeJSON(&b, r.Active)
	b.WriteString(",\n  \"notes\": ")
	n, _ := json.MarshalIndent(r.Notes, "", "  ")
	lines := strings.Split(string(n), "\n")
	b.WriteString(lines[0])
	for _, line := range lines[1:] {
		b.WriteString("\n  ")
		b.WriteString(line)
	}
	b.WriteString("\n}\n")
	return os.WriteFile(path, b.Bytes(), 0o644)
}

func marshalEntryPreserving(raw json.RawMessage, e *registryEntry) ([]byte, error) {
	if len(raw) > 0 {
		raw = normalizeRawEntry(raw)
		var original registryEntry
		if err := json.Unmarshal(raw, &original); err == nil {
			if original.Frozen != nil || e.Frozen == nil {
				if (original.Frozen == nil) == (e.Frozen == nil) {
					return raw, nil
				}
			} else {
				marker, err := json.MarshalIndent(e.Frozen, "", "  ")
				if err != nil {
					return nil, err
				}
				trimmed := bytes.TrimSpace(raw)
				base := bytes.TrimSpace(trimmed[:len(trimmed)-1])
				out := append(append(append([]byte{}, base...), []byte(",\n  \"frozen\": ")...), appendIndented(marker, "  ")...)
				return append(out, []byte("\n}")...), nil
			}
		}
	}
	return marshalEntry(e)
}

func normalizeRawEntry(raw json.RawMessage) json.RawMessage {
	lines := strings.Split(string(bytes.TrimSpace(raw)), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.TrimPrefix(lines[i], "    ")
	}
	return json.RawMessage(strings.Join(lines, "\n"))
}

func appendIndented(data []byte, prefix string) []byte {
	lines := strings.Split(string(data), "\n")
	var b strings.Builder
	b.WriteString(lines[0])
	for _, line := range lines[1:] {
		b.WriteByte('\n')
		b.WriteString(prefix)
		b.WriteString(line)
	}
	return []byte(b.String())
}

func writeJSON(b *bytes.Buffer, v string) { x, _ := json.Marshal(v); b.Write(x) }
func marshalEntry(e *registryEntry) ([]byte, error) {
	type ordered struct {
		File        string                     `json:"file"`
		Hash        string                     `json:"hash"`
		Description string                     `json:"description"`
		Created     string                     `json:"created"`
		Tags        []string                   `json:"tags"`
		Notes       string                     `json:"notes"`
		Frozen      *eval_harness.FrozenMarker `json:"frozen,omitempty"`
	}
	return json.MarshalIndent(ordered{e.File, e.Hash, e.Description, e.Created, e.Tags, e.Notes, e.Frozen}, "", "  ")
}

type corpusEvidence struct {
	Count   int
	Example string
}

func scanCorpus(repoRoot string) (map[string]corpusEvidence, error) {
	cmd := exec.Command("git", "ls-files", "eval_results/baselines/*.json")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	result := map[string]corpusEvidence{}
	for _, rel := range strings.Fields(string(out)) {
		data, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			return nil, err
		}
		var row struct {
			PromptVersion string `json:"prompt_version"`
		}
		if err := json.Unmarshal(data, &row); err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if row.PromptVersion != "" {
			ev := result[row.PromptVersion]
			ev.Count++
			if ev.Example == "" {
				ev.Example = rel
			}
			result[row.PromptVersion] = ev
		}
	}
	return result, nil
}

var corpusScanner = scanCorpus

func promptRegistryPair(root string) registryPair {
	return registryPair{filepath.Join(root, "prompts", "versions.json"), filepath.Join(root, "cmd", "ailang", "prompts", "versions.json")}
}
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func migrateRegistries(repoRoot, today string) (int, int, int, error) {
	pair := promptRegistryPair(repoRoot)
	r, err := loadOrderedRegistry(pair.Source)
	if err != nil {
		return 0, 0, 0, err
	}
	if r.Active == "latest" || r.Versions[r.Active] == nil {
		return 0, 0, 0, fmt.Errorf("active prompt %q is not a concrete registry key", r.Active)
	}
	evidence, err := corpusScanner(repoRoot)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, id := range r.VersionKeys {
		e := r.Versions[id]
		if id == r.Active || e.Frozen != nil {
			continue
		}
		if !eval_harness.IsHexSHA256(e.Hash) {
			return 0, 0, 0, fmt.Errorf("%s: recorded hash is not enforceable", id)
		}
		actual, xerr := fileHash(filepath.Join(repoRoot, e.File))
		if xerr != nil {
			return 0, 0, 0, xerr
		}
		if actual != e.Hash {
			return 0, 0, 0, fmt.Errorf("%s: file bytes do not match recorded hash", id)
		}
	}
	for _, id := range r.VersionKeys {
		e := r.Versions[id]
		if id == r.Active || e.Frozen != nil {
			continue
		}
		ev := evidence[id]
		reason := "legacy"
		if ev.Count > 0 {
			reason = "banked"
		}
		e.Frozen = &eval_harness.FrozenMarker{At: today, Reason: reason, EvidenceCount: ev.Count, EvidenceExample: ev.Example}
	}
	if err := writeOrderedRegistry(pair.Source, r); err != nil {
		return 0, 0, 0, err
	}
	if err := writeOrderedRegistry(pair.Mirror, r); err != nil {
		return 0, 0, 0, err
	}
	banked, legacy, mutable := 0, 0, 0
	for _, e := range r.Versions {
		if e.Frozen == nil {
			mutable++
		} else if e.Frozen.Reason == "banked" {
			banked++
		} else if e.Frozen.Reason == "legacy" {
			legacy++
		}
	}
	return banked, legacy, mutable, nil
}

func freezeVersion(repoRoot, versionID, today string) error {
	pair := promptRegistryPair(repoRoot)
	r, err := loadOrderedRegistry(pair.Source)
	if err != nil {
		return err
	}
	e := r.Versions[versionID]
	if e == nil {
		return fmt.Errorf("prompt version %q not found", versionID)
	}
	evidence, err := corpusScanner(repoRoot)
	if err != nil {
		return err
	}
	ev := evidence[versionID]
	if ev.Count == 0 {
		return fmt.Errorf("prompt version %q has no banked corpus evidence", versionID)
	}
	if !eval_harness.IsHexSHA256(e.Hash) {
		return fmt.Errorf("%s: recorded hash is not enforceable", versionID)
	}
	actual, err := fileHash(filepath.Join(repoRoot, e.File))
	if err != nil {
		return err
	}
	if actual != e.Hash {
		return fmt.Errorf("%s: file bytes do not match recorded hash", versionID)
	}
	if e.Frozen == nil {
		e.Frozen = &eval_harness.FrozenMarker{At: today, Reason: "banked", EvidenceCount: ev.Count, EvidenceExample: ev.Example}
	}
	if err := writeOrderedRegistry(pair.Source, r); err != nil {
		return err
	}
	return writeOrderedRegistry(pair.Mirror, r)
}

func checkRegistries(repoRoot string) ([]string, error) {
	pair := promptRegistryPair(repoRoot)
	source, err := loadOrderedRegistry(pair.Source)
	if err != nil {
		return nil, err
	}
	mirror, err := loadOrderedRegistry(pair.Mirror)
	if err != nil {
		return nil, err
	}
	evidence, err := corpusScanner(repoRoot)
	if err != nil {
		return nil, err
	}
	var v []string
	for id, ev := range evidence {
		if e := source.Versions[id]; e != nil && e.Frozen == nil {
			v = append(v, fmt.Sprintf("corpus-evidenced but not frozen: %s (%d citations, e.g. %s)", id, ev.Count, ev.Example))
		}
	}
	for _, id := range source.VersionKeys {
		e := source.Versions[id]
		if e.Frozen != nil {
			if !eval_harness.IsHexSHA256(e.Hash) {
				v = append(v, fmt.Sprintf("frozen version %s: recorded hash is not a 64-hex sha256 (unenforceable freeze)", id))
			} else if h, xerr := fileHash(filepath.Join(repoRoot, e.File)); xerr != nil {
				return nil, xerr
			} else if h != e.Hash {
				v = append(v, fmt.Sprintf("frozen version %s: file bytes do not match recorded hash", id))
			}
		}
		a, _ := json.Marshal(e)
		b, _ := json.Marshal(mirror.Versions[id])
		if !bytes.Equal(a, b) {
			v = append(v, fmt.Sprintf("cmd/ailang/prompts/versions.json: entry %s differs from source", id))
		}
	}
	return v, nil
}
