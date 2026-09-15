package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// The ONE loader for ~/.ailang/config.yaml (M-V1-SIMPLIFY-S3 M3, program
// §2.1). Before it there were three: messaging.LoadConfig (which ignored
// AILANG_CONFIG), coordinator.LoadCoordinatorConfig (which honoured it) and
// coordinator.getDefaultRepo (a hard-coded home path whose EVERY error became
// "sunholo-data/ailang"), plus eight places that re-read and re-parsed the
// file for one key. Each honoured a different path and failed a different
// way.
//
// The file is parsed ONCE per process per path and re-read only when its
// size or mtime changes — so a daemon that runs for days sees an edit the way
// it always did, without paying a parse per call. Sections decode into
// caller-owned types (File.Section), because the coordinator's section is a
// rich type with behaviour that must not move into a leaf.

// ErrConfigNotFound is wrapped when no config file exists at FilePath.
var ErrConfigNotFound = errors.New("config file not found")

// ErrConfigInvalid is wrapped when the file is not valid YAML, or a section
// does not decode into the type the caller asked for.
var ErrConfigInvalid = errors.New("config file invalid")

// File is one parsed config document.
type File struct {
	// Path the document was read from ("" for Parse).
	Path string
	raw  []byte
	root yaml.Node
}

type cachedFile struct {
	size  int64
	mtime time.Time
	file  *File
}

var fileCache struct {
	mu    sync.Mutex
	files map[string]cachedFile
}

// Load reads and parses the config file at FilePath — AILANG_CONFIG when
// set, else ~/.ailang/config.yaml — cached per process. A missing file is
// ErrConfigNotFound: most sections are optional, so callers decide whether
// that is an error for them (errors.Is).
func Load() (*File, error) {
	path := FilePath()
	if path == "" {
		return nil, fmt.Errorf("%w: %s is unset and the home directory is unresolvable", ErrConfigNotFound, EnvConfigFile)
	}
	return LoadFrom(path)
}

// LoadFrom is Load for an explicit path, with the same cache.
func LoadFrom(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("stat config file %s: %w", path, err)
	}
	fileCache.mu.Lock()
	defer fileCache.mu.Unlock()
	if c, ok := fileCache.files[path]; ok && c.size == info.Size() && c.mtime.Equal(info.ModTime()) {
		return c.file, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // the user's own config path
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	f, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	f.Path = path
	if fileCache.files == nil {
		fileCache.files = map[string]cachedFile{}
	}
	fileCache.files[path] = cachedFile{size: info.Size(), mtime: info.ModTime(), file: f}
	return f, nil
}

// Parse parses a config document held in memory — a candidate edit, or the
// cloud copy — with no cache. Path is "".
func Parse(data []byte) (*File, error) {
	f := &File{raw: data}
	if err := yaml.Unmarshal(data, &f.root); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
	}
	return f, nil
}

// section returns the mapping node for a top-level key, or nil.
func (f *File) section(key string) *yaml.Node {
	if f == nil {
		return nil
	}
	doc := &f.root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == key {
			return doc.Content[i+1]
		}
	}
	return nil
}

// Has reports whether the document has a top-level key with a non-null value.
func (f *File) Has(key string) bool {
	n := f.section(key)
	return n != nil && n.Tag != "!!null"
}

// Section decodes the top-level key into out. It returns false, nil when the
// key is absent or null — out is untouched — so a caller can apply its
// defaults. A present section that does not fit out is ErrConfigInvalid
// naming the key.
func (f *File) Section(key string, out any) (bool, error) {
	n := f.section(key)
	if n == nil || n.Tag == "!!null" {
		return false, nil
	}
	if err := n.Decode(out); err != nil {
		return true, fmt.Errorf("%w: section %q: %v", ErrConfigInvalid, key, err)
	}
	return true, nil
}

// UnknownKeys decodes the WHOLE document into out STRICTLY and returns every
// key that no field of a type named typeName reads, as "line N: key" with
// file-relative lines. YAML drops unknown keys silently, so a plausible
// setting can sit in the file doing nothing (`push_branch` on an agent,
// 2026-09-11). out is normally a wrapper naming one section, e.g.
// struct{ Coordinator CoordinatorConfig `yaml:"coordinator"` }; every sibling
// top-level block then reports as unknown to the wrapper, which is why the
// report is narrowed to typeName — a checker that cries wolf gets ignored. A
// genuine parse error is returned, not listed.
func (f *File) UnknownKeys(out any, typeName string) ([]string, error) {
	dec := yaml.NewDecoder(bytes.NewReader(f.raw))
	dec.KnownFields(true)
	var keys []string
	for {
		err := dec.Decode(out)
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		var te *yaml.TypeError
		if !errors.As(err, &te) {
			return nil, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
		}
		// yaml.v3 reports every unknown field in one TypeError:
		// "line 12: field push_branch not found in type coordinator.AgentConfig".
		for _, e := range te.Errors {
			if typeName != "" && !strings.Contains(e, "in type "+typeName) {
				continue
			}
			msg := e
			if i := strings.Index(msg, "field "); i >= 0 {
				if j := strings.Index(msg[i:], " not found"); j >= 0 {
					msg = msg[:i] + msg[i+len("field "):i+j]
				}
			}
			keys = append(keys, strings.TrimSpace(msg))
		}
		break
	}
	return keys, nil
}

// resetFileCacheForTest forgets every cached document.
func resetFileCacheForTest() {
	fileCache.mu.Lock()
	defer fileCache.mu.Unlock()
	fileCache.files = nil
}
