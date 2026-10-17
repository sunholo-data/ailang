package fileguard

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// Protection is matched against NAMES, but a write lands on a FILE. An
// in-root symlink gives a protected file a second name: with
// fs_deny_write = ["tests/**"], `lnk -> tests` made `lnk/new.py` writable
// and created tests/new.py (AILANG World, 2026-10-17, v0.54.11). The root
// handle cannot catch that — the link stays inside the root. CheckPath
// therefore matches the spelled name AND the name the kernel will reach.

// maxSymlinks bounds resolution (Linux's MAXSYMLINKS is 40).
const maxSymlinks = 40

// Resolve returns path's root-relative name with every in-root symlink
// followed, slash-separated. Components that do not exist yet (a file about
// to be created, its missing parents) are kept as spelled. With followFinal
// false the last component is not followed: a remove or rename acts on the
// link entry itself, never on its target. A symlink that leaves the root,
// or an absolute one, is an *EscapeError — the same verdict os.Root gives
// when the path is used.
func (r *Root) Resolve(p string, followFinal bool) (string, error) {
	name, err := r.Rel(p)
	if err != nil {
		return "", err
	}
	pending := splitRel(name)
	var out []string
	links := 0
	missing := false
	for len(pending) > 0 {
		c := pending[0]
		pending = pending[1:]
		switch c {
		case "", ".":
			continue
		case "..":
			if len(out) == 0 {
				return "", &EscapeError{Path: p, Root: r.dir}
			}
			out = out[:len(out)-1]
			continue
		}
		cur := path.Join(append(append([]string{}, out...), c)...)
		if missing || (!followFinal && len(pending) == 0) {
			out = append(out, c)
			continue
		}
		fi, err := r.root.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			missing = true
			out = append(out, c)
			continue
		}
		if err != nil {
			return "", r.wrap(p, err)
		}
		if fi.Mode()&fs.ModeSymlink == 0 {
			out = append(out, c)
			continue
		}
		if links++; links > maxSymlinks {
			return "", fmt.Errorf("resolve %s: too many levels of symbolic links", p)
		}
		target, err := r.root.Readlink(cur)
		if err != nil {
			return "", r.wrap(p, err)
		}
		if isAbsoluteCrossPlatform(target) {
			return "", &EscapeError{Path: p, Root: r.dir, Err: fmt.Errorf("absolute symlink %s -> %s", cur, target)}
		}
		pending = append(splitRel(target), pending...)
	}
	if len(out) == 0 {
		return ".", nil
	}
	return strings.Join(out, "/"), nil
}

func splitRel(p string) []string { return strings.Split(filepath.ToSlash(p), "/") }

// CheckPath applies p to path inside r: the name as spelled, then the name
// it resolves to through in-root symlinks, then — for a write to an existing
// regular file — its link count, because a hard link is a second name no
// path walk can find. move is the rename/remove variant (CheckMove, final
// component not followed; the ancestor rule only for a directory or a
// path that does not exist yet). A path that cannot be resolved is checked as
// spelled: every such path (escape, loop, permission) is refused again by
// the root handle when the operation runs.
func (r *Root) CheckPath(p Protection, name string, move bool) Violation {
	check := p.Check
	if move {
		check = p.CheckMove
		// Nothing lives beneath an existing non-directory (a file, or a
		// symlink entry, which a remove/rename moves as itself), so the
		// ancestor rule does not apply: under `**/*.test.ts` every directory
		// may hold a protected file, but helper.ail does not.
		if fi, err := r.Lstat(name); err == nil && !fi.IsDir() {
			check = p.Check
		}
	}
	spelled, err := r.Rel(name)
	if err != nil {
		spelled = name
	}
	if v := check(spelled); v.Protected() {
		return v
	}
	if !p.GitDir && len(p.DenyWrite) == 0 {
		return Violation{}
	}
	real, err := r.Resolve(name, !move)
	if err != nil {
		return Violation{}
	}
	if real != cleanRel(spelled) {
		if v := check(real); v.Protected() {
			v.Via = real
			return v
		}
	}
	if !move {
		if fi, err := r.root.Lstat(real); err == nil && fi.Mode().IsRegular() && linkCount(fi) > 1 {
			return Violation{HardLink: true, Via: real}
		}
	}
	return Violation{}
}
