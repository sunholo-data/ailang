package iteration

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Binding is machine-local placement, not model or workflow policy.
type Binding struct {
	Version       int    `toml:"version"`
	StateDB       string `toml:"state_db"`
	WorkspaceRoot string `toml:"workspace_root"`
}

// LoadBinding reads existing configuration without creating any state.
func LoadBinding(path string) (Binding, error) {
	var b Binding
	f, err := os.Open(path)
	if err != nil {
		return b, fmt.Errorf("configure %s with version=1 and absolute state_db/workspace_root paths: %w", path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil {
		return b, err
	}
	if len(data) > 64*1024 {
		return b, fmt.Errorf("runtime binding exceeds 64 KiB")
	}
	meta, err := toml.Decode(string(data), &b)
	if err != nil {
		return b, err
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return b, fmt.Errorf("unknown runtime binding key %s", unknown[0])
	}
	if b.Version != 1 || !filepath.IsAbs(b.StateDB) || !filepath.IsAbs(b.WorkspaceRoot) {
		return b, fmt.Errorf("runtime binding requires version=1 and absolute state_db/workspace_root paths")
	}
	if strings.ContainsAny(b.StateDB, "?#\x00") || strings.ContainsRune(b.WorkspaceRoot, '\x00') {
		return b, fmt.Errorf("runtime binding paths cannot contain NUL; state_db also cannot contain ? or #")
	}
	return b, nil
}
