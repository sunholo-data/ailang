package activation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const maxFile = 64 * 1024

func digest(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func validateBinding(b []byte) error {
	if len(b) > maxFile || bytes.Contains(b, []byte("#")) {
		return errors.New("binding must be <=64 KiB without comments (records never retain credentials)")
	}
	var v struct {
		Version       int    `toml:"version"`
		StateDB       string `toml:"state_db"`
		WorkspaceRoot string `toml:"workspace_root"`
	}
	meta, err := toml.Decode(string(b), &v)
	if err != nil {
		return errors.New("invalid runtime binding TOML")
	}
	if len(meta.Undecoded()) > 0 || v.Version != 1 || !filepath.IsAbs(v.StateDB) || !filepath.IsAbs(v.WorkspaceRoot) || strings.ContainsAny(v.StateDB, "?#\x00") || strings.ContainsRune(v.WorkspaceRoot, '\x00') {
		return errors.New("binding must contain only version=1 and absolute state_db/workspace_root paths")
	}
	return nil
}
func readRegular(path string) ([]byte, os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("refusing non-regular file %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !os.SameFile(info, opened) {
		return nil, 0, fmt.Errorf("file changed while opening %s", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxFile*8+1))
	if err == nil && len(b) > maxFile*8 {
		err = fmt.Errorf("file too large: %s", path)
	}
	return b, info.Mode().Perm(), err
}
func capture(path string, installed []byte) (File, error) {
	var f File
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return f, errors.New("activation requires clean absolute file paths")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return f, err
	}
	path = filepath.Join(parent, filepath.Base(path))
	f = File{Path: path, Installed: bytes.Clone(installed), InstalledHash: digest(installed), PriorHash: digest(nil)}
	prior, mode, err := readRegular(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	f.Existed = true
	f.Prior = prior
	f.PriorMode = uint32(mode)
	f.PriorHash = digest(prior)
	return f, nil
}
func matchesPrior(f File) error {
	prior, err := restorable(f)
	if err != nil {
		return err
	}
	if !prior {
		return fmt.Errorf("target changed before installation: %s", f.Path)
	}
	return nil
}
func restorable(f File) (bool, error) {
	parent, err := filepath.EvalSymlinks(filepath.Dir(f.Path))
	if err != nil {
		return false, err
	}
	if parent != filepath.Dir(f.Path) {
		return false, fmt.Errorf("cleanup_pending: target parent changed: %s", f.Path)
	}
	b, mode, err := readRegular(f.Path)
	if os.IsNotExist(err) {
		if !f.Existed {
			return true, nil
		}
		return false, fmt.Errorf("cleanup_pending: owned file missing: %s", f.Path)
	}
	if err != nil {
		return false, err
	}
	hash := digest(b)
	if f.Existed && hash == f.PriorHash && uint32(mode) == f.PriorMode {
		return true, nil
	}
	if hash == f.InstalledHash && mode == 0600 {
		return false, nil
	}
	return false, fmt.Errorf("cleanup_pending: external file change at %s; preserve it and inspect", f.Path)
}
func (m *Manager) recordPath(id string) string { return filepath.Join(m.Dir, id+".json") }

// Inspect is read-only, does not create runtime state and remains usable after restore.
func (m *Manager) Inspect(id string) (r Record, err error) {
	if !filepath.IsAbs(m.Dir) || !identifier.MatchString(id) {
		return r, errors.New("invalid operation ID")
	}
	b, _, err := readRegular(m.recordPath(id))
	if err != nil {
		return r, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&r); err != nil {
		return r, err
	}
	if err = dec.Decode(new(any)); err != io.EOF {
		return r, errors.New("trailing activation record content")
	}
	if r.Version != 1 || r.OperationID != id || r.MissionID != "docs" || r.WorkItemID == "" {
		return r, errors.New("invalid activation record identity")
	}
	if r.Phase != "prepared" && r.Phase != "active" && r.Phase != "restoring" && r.Phase != "restored" {
		return r, errors.New("invalid activation phase")
	}
	for _, f := range []File{r.Marker, r.Binding} {
		if !filepath.IsAbs(f.Path) || filepath.Clean(f.Path) != f.Path || f.InstalledHash != digest(f.Installed) || f.PriorHash != digest(f.Prior) || !f.Existed && len(f.Prior) > 0 {
			return r, errors.New("invalid activation file evidence")
		}
	}
	if r.Marker.Existed || string(r.Marker.Installed) != "ailang mission activation "+id+"\n" || r.Marker.Path == r.Binding.Path {
		return r, errors.New("invalid marker ownership")
	}
	if err = validateBinding(r.Binding.Installed); err != nil {
		return r, err
	}
	if r.Binding.Existed {
		err = validateBinding(r.Binding.Prior)
	}
	return r, err
}
func (m *Manager) save(r Record) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(m.recordPath(r.OperationID), b, 0600)
}
func (m *Manager) owner() (string, error) {
	b, _, err := readRegular(filepath.Join(m.Dir, "active"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	id := string(b)
	if !filepath.IsAbs(m.Dir) || !identifier.MatchString(id) {
		return "", errors.New("invalid host ownership record; inspect manually")
	}
	return id, nil
}
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".activation-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
