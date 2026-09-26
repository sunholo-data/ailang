// Package fileguard is the one confined-filesystem abstraction the runtime,
// the CLI and the agent tools share (M-EXECUTOR-POLICY-HARDENING M1, D1).
//
// A Root wraps os.Root: every operation is performed by the kernel relative
// to an open directory descriptor, so a path that traverses out of the root
// (`..`, an absolute path, a symlink whose target leaves the root, a link
// swapped in between check and use) fails inside the syscall rather than
// after a lexical check. There is deliberately no second path parser here:
// Rel only turns an absolute path that is lexically inside the root into a
// relative name for compatibility; the root handle is always the final
// authority.
//
// Symlink contract (inherited from os.Root, documented as a migration in the
// CHANGELOG): relative symlinks that stay inside the root resolve; absolute
// symlinks are refused even when they point back inside.
//
// Lifecycle: one owner opens the root at execution start, shares the handle
// with derived contexts, and closes it once when every operation has
// stopped. Callers must not reopen a root from an agent-writable pathname
// per operation.
package fileguard

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EscapeError is returned when a path would leave the root — by lexical
// position (absolute path outside the root) or by the kernel's verdict
// (traversal or symlink out of the open directory).
type EscapeError struct {
	Path string
	Root string
	Err  error
}

func (e *EscapeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("path %q escapes sandbox %q: %v", e.Path, e.Root, e.Err)
	}
	return fmt.Sprintf("path %q escapes sandbox %q", e.Path, e.Root)
}

func (e *EscapeError) Unwrap() error { return e.Err }

// ErrUnsupportedPlatform is returned by Open where os.Root cannot confine
// (GOOS=js: documented TOCTOU in symlink validation). Restricted execution
// refuses there instead of falling back to a lexical check.
var ErrUnsupportedPlatform = errors.New("fileguard: confined filesystem is not supported on this platform")

// IsEscape reports whether err is a containment refusal: either our typed
// EscapeError or os.Root's own "path escapes from parent" verdict (which the
// os package does not export as a sentinel).
func IsEscape(err error) bool {
	if err == nil {
		return false
	}
	var ee *EscapeError
	if errors.As(err, &ee) {
		return true
	}
	return strings.Contains(err.Error(), "path escapes from parent")
}

// Root is an open, confined directory.
type Root struct {
	dir  string
	root *os.Root
}

// Open opens dir as a confined root. dir must exist and be a directory.
func Open(dir string) (*Root, error) {
	if !supported {
		return nil, ErrUnsupportedPlatform
	}
	if dir == "" {
		return nil, errors.New("fileguard: empty root directory")
	}
	clean := filepath.Clean(dir)
	r, err := os.OpenRoot(clean)
	if err != nil {
		return nil, fmt.Errorf("fileguard: open sandbox root %q: %w", dir, err)
	}
	return &Root{dir: clean, root: r}, nil
}

// Dir is the directory the root was opened on (cleaned, as given).
func (r *Root) Dir() string { return r.dir }

// Close releases the directory handle. Safe to call more than once.
func (r *Root) Close() error {
	if r == nil || r.root == nil {
		return nil
	}
	err := r.root.Close()
	r.root = nil
	return err
}

// Rel converts a program-supplied path into a name for the root handle.
//
//   - relative path → returned cleaned (`..` is left in; the kernel decides)
//   - absolute path lexically inside the root → the relative remainder
//   - absolute path outside → *EscapeError
//
// "Absolute" is judged for every mainstream host (a leading `/` or `\`, a
// drive letter, a UNC prefix) so a program saying fs.exists("/etc/passwd")
// is refused identically on every platform.
func (r *Root) Rel(path string) (string, error) {
	if path == "" {
		return ".", nil
	}
	if !isAbsoluteCrossPlatform(path) {
		return filepath.Clean(path), nil
	}
	clean := filepath.Clean(path)
	if clean == r.dir {
		return ".", nil
	}
	prefix := r.dir + string(filepath.Separator)
	if !strings.HasPrefix(clean, prefix) {
		return "", &EscapeError{Path: path, Root: r.dir}
	}
	return strings.TrimPrefix(clean, prefix), nil
}

// wrap turns os.Root's escape verdict into a typed EscapeError and leaves
// every other error (not found, permission, not a directory) as-is.
func (r *Root) wrap(path string, err error) error {
	if err == nil {
		return nil
	}
	if IsEscape(err) {
		var ee *EscapeError
		if errors.As(err, &ee) {
			return err
		}
		return &EscapeError{Path: path, Root: r.dir, Err: err}
	}
	return err
}

// Open opens a file for reading.
func (r *Root) Open(path string) (*os.File, error) {
	name, err := r.Rel(path)
	if err != nil {
		return nil, err
	}
	f, err := r.root.Open(name)
	return f, r.wrap(path, err)
}

// OpenFile opens a file with the given flags and permission.
func (r *Root) OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error) {
	name, err := r.Rel(path)
	if err != nil {
		return nil, err
	}
	f, err := r.root.OpenFile(name, flag, perm)
	return f, r.wrap(path, err)
}

// Stat follows symlinks (inside the root only).
func (r *Root) Stat(path string) (fs.FileInfo, error) {
	name, err := r.Rel(path)
	if err != nil {
		return nil, err
	}
	fi, err := r.root.Stat(name)
	return fi, r.wrap(path, err)
}

// Lstat does not follow the final symlink.
func (r *Root) Lstat(path string) (fs.FileInfo, error) {
	name, err := r.Rel(path)
	if err != nil {
		return nil, err
	}
	fi, err := r.root.Lstat(name)
	return fi, r.wrap(path, err)
}

// Mkdir creates one directory; the parent must exist.
func (r *Root) Mkdir(path string, perm fs.FileMode) error {
	name, err := r.Rel(path)
	if err != nil {
		return err
	}
	return r.wrap(path, r.root.Mkdir(name, perm))
}

// MkdirAll creates a directory and its missing parents.
func (r *Root) MkdirAll(path string, perm fs.FileMode) error {
	name, err := r.Rel(path)
	if err != nil {
		return err
	}
	return r.wrap(path, r.root.MkdirAll(name, perm))
}

// Remove removes a file, an empty directory, or a symlink ENTRY (the link
// itself, never its target).
func (r *Root) Remove(path string) error {
	name, err := r.Rel(path)
	if err != nil {
		return err
	}
	return r.wrap(path, r.root.Remove(name))
}

// Rename renames within the root; both operands are confined.
func (r *Root) Rename(oldPath, newPath string) error {
	oldName, err := r.Rel(oldPath)
	if err != nil {
		return err
	}
	newName, err := r.Rel(newPath)
	if err != nil {
		return err
	}
	if err := r.root.Rename(oldName, newName); err != nil {
		if IsEscape(err) {
			// os.Root does not say which operand escaped; name both.
			return &EscapeError{Path: oldPath + " -> " + newPath, Root: r.dir, Err: err}
		}
		return err
	}
	return nil
}

// ReadDir lists a directory sorted by name, through the root handle.
func (r *Root) ReadDir(path string) ([]fs.DirEntry, error) {
	f, err := r.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// isAbsoluteCrossPlatform mirrors internal/effects.isAbsoluteCrossPlatform and
// pkg.IsAbsoluteCrossPlatform; this package is a leaf and cannot import either.
func isAbsoluteCrossPlatform(path string) bool {
	if path == "" {
		return false
	}
	if path[0] == '/' || path[0] == '\\' {
		return true
	}
	if len(path) >= 2 && path[1] == ':' {
		c := path[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return strings.HasPrefix(path, "//")
}
