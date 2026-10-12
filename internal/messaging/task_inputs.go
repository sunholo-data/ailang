package messaging

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode"
)

// TaskInput declares data fetched by the cloud job parent before execution. Refs #1600.
// Repo is an exact owner/repo grant key; Ref must name a branch or tag.
// Empty Path selects the whole tree; empty Dest selects excluded .incoming/<n>/.
// SHA256 pins a single file. Directory roots may instead contain manifest.sha256.
type TaskInput struct {
	Repo   string `json:"repo"`
	Ref    string `json:"ref"`
	Path   string `json:"path"`
	Dest   string `json:"dest,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

const MaxTaskInputs = 16

var inputRepoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// ValidateInputRepo validates the exact key used by trusted registry grants.
func ValidateInputRepo(repo string) error {
	if !inputRepoPattern.MatchString(repo) {
		return fmt.Errorf("repo %q must be an exact owner/repo", repo)
	}
	for _, part := range strings.Split(repo, "/") {
		if part == "." || part == ".." {
			return fmt.Errorf("invalid repo %q", repo)
		}
	}
	return nil
}

// ValidateInputPath rejects traversal and platform-dependent path syntax.
// A trailing slash is accepted for directory destinations.
func ValidateInputPath(value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, "\\:\x00\r\n") || strings.HasPrefix(value, "/") {
		return fmt.Errorf("path %q must be workspace-relative", value)
	}
	clean := strings.TrimSuffix(value, "/")
	if clean == "" || clean == "." || path.Clean(clean) != clean {
		return fmt.Errorf("path %q must be clean and relative", value)
	}
	for _, p := range strings.Split(clean, "/") {
		if p == ".." || p == "." {
			return fmt.Errorf("path %q contains traversal", value)
		}
	}
	return nil
}

func validateInputRef(ref string) error {
	if ref == "" || ref == "@" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, " ~^:?*[\\") || strings.Contains(ref, "..") || strings.Contains(ref, "@{") || strings.HasSuffix(ref, ".") {
		return fmt.Errorf("ref %q must be a branch or tag", ref)
	}
	if len(ref) == 40 || len(ref) == 64 {
		if _, err := hex.DecodeString(ref); err == nil {
			return fmt.Errorf("raw commit SHA refs are unsupported: %q", ref)
		}
	}
	for _, r := range ref {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("invalid ref %q", ref)
		}
	}
	for _, p := range strings.Split(ref, "/") {
		if p == "" || strings.HasPrefix(p, ".") || strings.HasSuffix(p, ".lock") {
			return fmt.Errorf("invalid ref %q", ref)
		}
	}
	return nil
}

func (input TaskInput) Validate() error {
	if err := ValidateInputRepo(input.Repo); err != nil {
		return err
	}
	if err := validateInputRef(input.Ref); err != nil {
		return err
	}
	if err := ValidateInputPath(input.Path); err != nil {
		return fmt.Errorf("path: %w", err)
	}
	if err := ValidateInputPath(input.Dest); err != nil {
		return fmt.Errorf("dest: %w", err)
	}
	if input.SHA256 != "" {
		if input.Path == "" {
			return fmt.Errorf("sha256 requires a single-file path")
		}
		if _, err := hex.DecodeString(input.SHA256); err != nil || len(input.SHA256) != 64 {
			return fmt.Errorf("sha256 must be 64 hexadecimal digits")
		}
	}
	return nil
}

func ValidateTaskInputs(inputs []TaskInput) error {
	if len(inputs) > MaxTaskInputs {
		return fmt.Errorf("inputs: maximum %d entries", MaxTaskInputs)
	}
	for i, input := range inputs {
		if err := input.Validate(); err != nil {
			return fmt.Errorf("inputs[%d]: %w", i, err)
		}
	}
	return nil
}

// DecodeTaskInputs rejects unknown fields, trailing JSON, and malformed records.
// Empty/null input is the legacy zero-input value.
func DecodeTaskInputs(data []byte) ([]TaskInput, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var inputs []TaskInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inputs); err != nil {
		return nil, fmt.Errorf("inputs: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("inputs: trailing JSON")
	}
	if err := ValidateTaskInputs(inputs); err != nil {
		return nil, err
	}
	return inputs, nil
}
