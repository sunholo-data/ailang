package policytool

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
)

// The file operations. Each goes through the sandbox root handle; the
// policy's max_fs_transfer_bytes caps what a single call moves.

// maxTransfer is the byte ceiling for one read or write.
func (h *Host) maxTransfer() int64 {
	if h.res.MaxFSTransferBytes > 0 {
		return h.res.MaxFSTransferBytes
	}
	return 64 << 20 // trusted_host with no cap: still bounded for a tool call
}

// needRoot is the refusal for a file op under a policy without FS.
func (h *Host) needRoot() *Response {
	if h.root == nil {
		r := refuse("the policy admits no FS: file tools are not granted (allowed_caps has no FS / fs_sandbox is unset)")
		return &r
	}
	return nil
}

func (h *Host) read(req Request) Response {
	if r := h.needRoot(); r != nil {
		return *r
	}
	if req.Path == "" {
		return refuse("read: path is required")
	}
	f, err := h.root.Open(req.Path)
	if err != nil {
		return refuse("read %s: %v", req.Path, err)
	}
	defer f.Close()
	cap := h.maxTransfer()
	data, err := io.ReadAll(io.LimitReader(f, cap+1))
	if err != nil {
		return refuse("read %s: %v", req.Path, err)
	}
	if int64(len(data)) > cap {
		return refuse("read %s: file exceeds the %d-byte transfer cap", req.Path, cap)
	}
	return Response{OK: true, Content: string(data)}
}

// underGitDir reports whether a path has a `.git` component: the repository
// metadata is read-only to the agent (M6) — the confined git adapter trusts
// the clone's config, so the tools must not be a way to plant one.
func underGitDir(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Clean(path)), "/") {
		if seg == ".git" {
			return true
		}
	}
	return false
}

// denyWrite applies the policy's fs_deny_write to a tool path (relative to
// the sandbox root; an absolute in-root path loses the prefix).
func (h *Host) denyWrite(p string) string {
	if len(h.res.DenyWrite) == 0 || h.root == nil {
		return ""
	}
	rel, err := h.root.Rel(p)
	if err != nil {
		return "" // an escaping path is refused by the root handle anyway
	}
	return effects.MatchDenyWrite(h.res.DenyWrite, rel)
}

func (h *Host) write(req Request) Response {
	if r := h.needRoot(); r != nil {
		return *r
	}
	if req.Path == "" {
		return refuse("write: path is required")
	}
	if underGitDir(req.Path) {
		return refuse("write %s: .git/ is read-only to the lane's tools", req.Path)
	}
	if pat := h.denyWrite(req.Path); pat != "" {
		return refuse("write %s: matches fs_deny_write %q — read-only under this policy", req.Path, pat)
	}
	if int64(len(req.Content)) > h.maxTransfer() {
		return refuse("write %s: content exceeds the %d-byte transfer cap", req.Path, h.maxTransfer())
	}
	f, err := h.root.OpenFile(req.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return refuse("write %s: %v", req.Path, err)
	}
	_, werr := f.WriteString(req.Content)
	cerr := f.Close()
	if werr != nil {
		return refuse("write %s: %v", req.Path, werr)
	}
	if cerr != nil {
		return refuse("write %s: %v", req.Path, cerr)
	}
	return Response{OK: true}
}

// edit applies pi's expected-content contract: OldText must occur exactly
// once, and the file is rewritten inside the root.
func (h *Host) edit(req Request) Response {
	if r := h.needRoot(); r != nil {
		return *r
	}
	if req.Path == "" {
		return refuse("edit: path is required")
	}
	if req.OldText == "" {
		return refuse("edit %s: old_text is required (use write to create a file)", req.Path)
	}
	if underGitDir(req.Path) {
		return refuse("edit %s: .git/ is read-only to the lane's tools", req.Path)
	}
	if pat := h.denyWrite(req.Path); pat != "" {
		return refuse("edit %s: matches fs_deny_write %q — read-only under this policy", req.Path, pat)
	}
	cur := h.read(Request{Path: req.Path})
	if !cur.OK {
		return cur
	}
	n := strings.Count(cur.Content, req.OldText)
	switch {
	case n == 0:
		return refuse("edit %s: old_text not found — the file may have changed; read it again", req.Path)
	case n > 1:
		return refuse("edit %s: old_text occurs %d times; include more context so it is unique", req.Path, n)
	}
	next := strings.Replace(cur.Content, req.OldText, req.NewText, 1)
	if w := h.write(Request{Path: req.Path, Content: next}); !w.OK {
		return w
	}
	return Response{OK: true}
}

// String helpers shared by the schema code.
func joinStrings(v []string) string { return strings.Join(v, ", ") }
