package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaintenanceCheckpointFile is the canonical identity of one owned app file.
// Executable is portable: only the executable/non-executable distinction is kept.
type MaintenanceCheckpointFile struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}

// MaintenanceCheckpoint records actual source, including broken source and deletions.
// Root is a controller path and must never be included in the model prompt.
type MaintenanceCheckpoint struct {
	Root   string                      `json:"-"`
	Digest string                      `json:"digest"`
	Files  []MaintenanceCheckpointFile `json:"files"`
}

// MaintenanceCheckpointDigest hashes a sorted canonical file manifest, without newline.
func MaintenanceCheckpointDigest(files []MaintenanceCheckpointFile) (string, error) {
	ordered := append([]MaintenanceCheckpointFile{}, files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for i, f := range ordered {
		if err := maintenanceOwnedPath(f.Path); err != nil {
			return "", err
		}
		decoded, err := hex.DecodeString(f.SHA256)
		if err != nil || len(decoded) != sha256.Size || strings.ToLower(f.SHA256) != f.SHA256 {
			return "", fmt.Errorf("invalid file digest: %s", f.Path)
		}
		if i > 0 && ordered[i-1].Path == f.Path {
			return "", fmt.Errorf("duplicate owned path: %s", f.Path)
		}
	}
	data, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// CaptureMaintenanceCheckpoint copies existing declared regular files into a new
// controller-owned snapshot. Missing paths are omitted, preserving deletions. The
// caller must reap candidate processes before capture; this is not a live FS lock.
func CaptureMaintenanceCheckpoint(source, destination string, ownedPaths []string) (MaintenanceCheckpoint, error) {
	cp := MaintenanceCheckpoint{Files: []MaintenanceCheckpointFile{}}
	src, dst, err := maintenanceDisjointRoots(source, destination)
	if err != nil {
		return cp, err
	}
	paths := append([]string{}, ownedPaths...)
	sort.Strings(paths)
	for i, p := range paths {
		if err := maintenanceOwnedPath(p); err != nil {
			return cp, err
		}
		if i > 0 && paths[i-1] == p {
			return cp, fmt.Errorf("duplicate owned path: %s", p)
		}
	}
	if err := os.Mkdir(dst, 0700); err != nil {
		return cp, fmt.Errorf("reserve checkpoint: %w", err)
	}
	cp.Root = dst
	if err := os.Mkdir(filepath.Join(dst, "files"), 0700); err != nil {
		return cp, err
	}
	for _, p := range paths {
		data, mode, err := maintenanceReadRegular(src, p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return cp, err
		}
		f := maintenanceFileIdentity(p, data, mode)
		if err := maintenanceWriteNew(filepath.Join(dst, "files"), f, data); err != nil {
			return cp, err
		}
		cp.Files = append(cp.Files, f)
	}
	cp.Digest, err = MaintenanceCheckpointDigest(cp.Files)
	if err != nil {
		return cp, err
	}
	data, err := json.Marshal(cp)
	if err != nil {
		return cp, err
	}
	if err := maintenanceCreateFile(filepath.Join(dst, "checkpoint.json"), data, 0600); err != nil {
		return cp, err
	}
	return cp, nil
}

// RestoreMaintenanceCheckpoint verifies the immutable evidence and copies only
// its owned files into a fresh directory. No original starter is consulted.
func RestoreMaintenanceCheckpoint(cp MaintenanceCheckpoint, destination string) error {
	src, dst, err := maintenanceDisjointRoots(cp.Root, destination)
	if err != nil {
		return err
	}
	digest, err := MaintenanceCheckpointDigest(cp.Files)
	if err != nil {
		return err
	}
	if digest != cp.Digest {
		return fmt.Errorf("checkpoint manifest digest mismatch")
	}
	saved, _, err := maintenanceReadRegular(src, "checkpoint.json")
	if err != nil {
		return err
	}
	expected, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	if string(saved) != string(expected) {
		return fmt.Errorf("checkpoint evidence changed")
	}
	// Verify all content before materializing any candidate file.
	contents := make([][]byte, len(cp.Files))
	for i, f := range cp.Files {
		data, mode, err := maintenanceReadRegular(filepath.Join(src, "files"), f.Path)
		if err != nil {
			return err
		}
		if maintenanceFileIdentity(f.Path, data, mode) != f {
			return fmt.Errorf("checkpoint file changed: %s", f.Path)
		}
		contents[i] = data
	}
	if err := os.Mkdir(dst, 0700); err != nil {
		return fmt.Errorf("reserve workspace: %w", err)
	}
	for i, f := range cp.Files {
		if err := maintenanceWriteNew(dst, f, contents[i]); err != nil {
			return err
		}
	}
	return nil
}

func maintenanceOwnedPath(p string) error {
	if !utf8.ValidString(p) || p == "" || p == "." || path.Clean(p) != p || strings.Contains(p, "\\") || strings.ContainsRune(p, 0) || strings.Contains(p, ":") || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") {
		return fmt.Errorf("unsafe owned path: %q", p)
	}
	return nil
}

func maintenanceDisjointRoots(source, destination string) (string, string, error) {
	src, err := filepath.Abs(source)
	if err != nil {
		return "", "", err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("source root is not a regular directory")
	}
	src, err = filepath.EvalSymlinks(src)
	if err != nil {
		return "", "", err
	}
	dst, err := filepath.Abs(destination)
	if err != nil {
		return "", "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(dst))
	if err != nil {
		return "", "", err
	}
	dst = filepath.Join(parent, filepath.Base(dst))
	if maintenancePathWithin(src, dst) || maintenancePathWithin(dst, src) {
		return "", "", fmt.Errorf("workspace and checkpoint roots overlap")
	}
	return src, dst, nil
}

func maintenancePathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Lstat each component before reading: neither leaf nor ancestors may be links.
func maintenanceReadRegular(root, name string) ([]byte, os.FileMode, error) {
	if err := maintenanceOwnedPath(name); err != nil {
		return nil, 0, err
	}
	current := root
	for i, part := range strings.Split(name, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, 0, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, 0, fmt.Errorf("symlink in owned path: %s", name)
		}
		if i < len(strings.Split(name, "/"))-1 {
			if !info.IsDir() {
				return nil, 0, fmt.Errorf("non-directory ancestor: %s", name)
			}
		} else {
			if !info.Mode().IsRegular() {
				return nil, 0, fmt.Errorf("non-regular owned file: %s", name)
			}
			data, err := os.ReadFile(current)
			return data, info.Mode(), err
		}
	}
	return nil, 0, fmt.Errorf("empty owned path")
}

func maintenanceFileIdentity(name string, data []byte, mode os.FileMode) MaintenanceCheckpointFile {
	sum := sha256.Sum256(data)
	return MaintenanceCheckpointFile{Path: name, SHA256: hex.EncodeToString(sum[:]), Executable: mode&0111 != 0}
}

func maintenanceWriteNew(root string, f MaintenanceCheckpointFile, data []byte) error {
	target := filepath.Join(root, filepath.FromSlash(f.Path))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	mode := os.FileMode(0600)
	if f.Executable {
		mode = 0700
	}
	return maintenanceCreateFile(target, data, mode)
}

func maintenanceCreateFile(target string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
