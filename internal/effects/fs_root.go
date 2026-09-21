package effects

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/sunholo-data/ailang/internal/fileguard"
)

// M-EXECUTOR-POLICY-HARDENING M1 — root-anchored filesystem operations.
//
// When EffEnv.Sandbox is set, every FS operation goes through ONE confined
// root handle (internal/fileguard over os.Root) instead of a lexical
// resolveSandboxPath + os.*: `..`, symlinks that leave the root, absolute
// symlinks and check/use link swaps all fail inside the syscall. When no
// sandbox is configured the host backend is plain os.* — documented
// unsandboxed CLI behaviour is unchanged.
//
// Lifecycle: the holder is a pointer shared by every derived context
// (WithBudget, Clone), opened once per sandbox path on first use, and closed
// by the OWNER (runner, embed, serve) through CloseFSRoot. A context copy
// never closes it. A close followed by another operation reopens the same
// path — the holder is an owned cache, not a one-shot.

// fsBackend is the seam both backends satisfy. fileguard.Root implements it
// directly; hostFS adapts os.*.
type fsBackend interface {
	Open(path string) (*os.File, error)
	OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error)
	Stat(path string) (fs.FileInfo, error)
	Lstat(path string) (fs.FileInfo, error)
	Mkdir(path string, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	Rename(oldPath, newPath string) error
	ReadDir(path string) ([]fs.DirEntry, error)
}

// hostFS is the unconfined backend: the process's own view of the filesystem.
type hostFS struct{}

func (hostFS) Open(path string) (*os.File, error) { return os.Open(path) }
func (hostFS) OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag, perm)
}
func (hostFS) Stat(path string) (fs.FileInfo, error)        { return os.Stat(path) }
func (hostFS) Lstat(path string) (fs.FileInfo, error)       { return os.Lstat(path) }
func (hostFS) Mkdir(path string, perm fs.FileMode) error    { return os.Mkdir(path, perm) }
func (hostFS) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }
func (hostFS) Remove(path string) error                     { return os.Remove(path) }
func (hostFS) Rename(oldPath, newPath string) error         { return os.Rename(oldPath, newPath) }
func (hostFS) ReadDir(path string) ([]fs.DirEntry, error)   { return os.ReadDir(path) }

// fsRootHolder caches the open root for one sandbox path.
type fsRootHolder struct {
	mu   sync.Mutex
	dir  string
	root *fileguard.Root
}

// sandboxRoot returns the confined root for ctx.Env.Sandbox, opening it on
// first use. (nil, nil) when no sandbox is configured.
func (ctx *EffContext) sandboxRoot() (*fileguard.Root, error) {
	dir := ctx.Env.Sandbox
	if dir == "" {
		return nil, nil
	}
	if ctx.fsRoot == nil {
		// A context built as a literal (tests) rather than via NewEffContext.
		// Copies made after this point share the holder; copies made before
		// would open their own, which is still confined, just not shared.
		ctx.fsRoot = &fsRootHolder{}
	}
	h := ctx.fsRoot
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.root != nil && h.dir == dir {
		return h.root, nil
	}
	if h.root != nil {
		_ = h.root.Close()
		h.root = nil
	}
	r, err := fileguard.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("E_FS_SANDBOX_UNAVAILABLE: %w", err)
	}
	h.dir, h.root = dir, r
	return r, nil
}

// CloseFSRoot releases the shared sandbox root. The OWNER of the execution
// calls it once after every operation has stopped; it is idempotent and a
// no-op when no root was ever opened.
func (ctx *EffContext) CloseFSRoot() error {
	if ctx == nil || ctx.fsRoot == nil {
		return nil
	}
	h := ctx.fsRoot
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.root == nil {
		return nil
	}
	err := h.root.Close()
	h.root, h.dir = nil, ""
	return err
}

// fsBackendFor picks the backend for this context: the confined root when a
// sandbox is set, the host otherwise.
func (ctx *EffContext) fsBackendFor() (fsBackend, error) {
	r, err := ctx.sandboxRoot()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return hostFS{}, nil
	}
	return r, nil
}

// fsSandboxed reports whether a sandbox root is in force.
func (ctx *EffContext) fsSandboxed() bool { return ctx != nil && ctx.Env.Sandbox != "" }

// fsRejectProbe is the boolean-probe contract for a refused path: false,
// with the optional diagnostics (M-FS-SANDBOX-DIAGNOSTICS) when the refusal
// is a containment escape rather than an ordinary not-found.
func fsRejectProbe(ctx *EffContext, op, path string, err error) {
	if ctx.fsSandboxed() && fileguard.IsEscape(err) {
		logSandboxReject(ctx, op, path, "false")
	}
}

// fsCheckTransfer applies the per-transfer cap (Env.FSMaxBytes, from
// --fs-max-bytes or the policy's max_fs_transfer_bytes) to a write the way
// readCapped applies it to a read: refused before any byte reaches the file.
func (ctx *EffContext) fsCheckTransfer(path string, n int) error {
	if ctx == nil || ctx.Env.FSMaxBytes <= 0 || int64(n) <= ctx.Env.FSMaxBytes {
		return nil
	}
	return fmt.Errorf("E_FS_WRITE_TOO_LARGE: %s would receive %d bytes, cap %d (raise --fs-max-bytes or write it in parts)", path, n, ctx.Env.FSMaxBytes)
}

// writeAll opens path for truncating write through the backend and writes
// data with 0644 — os.WriteFile's exact semantics, through the root.
func fsWriteAll(b fsBackend, path string, data []byte) error {
	f, err := b.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

// fsAppendAll opens path for append-create through the backend and writes data.
func fsAppendAll(b fsBackend, path string, data []byte) error {
	f, err := b.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

// ---------------------------------------------------------------------------
// Exported file access for builtins that touch the host filesystem outside
// the FS op table (std/zip, std/gzip, std/tar, zip_xml). They MUST go through
// these rather than joining ctx.Env.Sandbox themselves: a joined path handed
// to os.* is the F1/F2 bug by another route.
// ---------------------------------------------------------------------------

// FSOpen opens path for reading through the context's filesystem backend.
func (ctx *EffContext) FSOpen(path string) (*os.File, error) {
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	return b.Open(path)
}

// FSCreate opens path for truncating write (0644) through the backend.
func (ctx *EffContext) FSCreate(path string) (*os.File, error) {
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	return b.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

// FSWriteFile writes data to path (0644) through the backend, under the
// transfer cap.
func (ctx *EffContext) FSWriteFile(path string, data []byte) error {
	if err := ctx.fsCheckTransfer(path, len(data)); err != nil {
		return err
	}
	b, err := ctx.fsBackendFor()
	if err != nil {
		return err
	}
	return fsWriteAll(b, path, data)
}

// FSMkdirAll creates path and its parents through the backend.
func (ctx *EffContext) FSMkdirAll(path string) error {
	b, err := ctx.fsBackendFor()
	if err != nil {
		return err
	}
	return b.MkdirAll(path, 0755)
}

// FSRemove removes a file, empty directory or link entry through the backend.
func (ctx *EffContext) FSRemove(path string) error {
	b, err := ctx.fsBackendFor()
	if err != nil {
		return err
	}
	return b.Remove(path)
}

// FSStat stats path through the backend (symlinks followed inside the root).
func (ctx *EffContext) FSStat(path string) (fs.FileInfo, error) {
	b, err := ctx.fsBackendFor()
	if err != nil {
		return nil, err
	}
	return b.Stat(path)
}
