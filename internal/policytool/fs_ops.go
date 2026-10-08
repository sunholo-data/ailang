package policytool

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/fileguard"
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
		return refuse("read: path is required (use \".\" to list the sandbox root)")
	}
	if fi, err := h.root.Stat(req.Path); err == nil && fi.IsDir() {
		return h.listDir(req.Path)
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

// maxListEntries bounds one directory listing.
const maxListEntries = 2000

// listDir answers a read of a DIRECTORY with its entries, one per line, a
// trailing "/" on subdirectories. The lane has no shell and no other listing
// tool, and `ailang tree` needs an ailang.toml at the root, so in a monorepo an
// agent could not see the repo's layout: measured 2026-10-08, motoko spent all
// 300 steps of a 3-line fix writing its own directory lister after
// `read packages` was refused. `.git` is omitted (read-only to the agent and
// never useful to list).
func (h *Host) listDir(path string) Response {
	entries, err := h.root.ReadDir(path)
	if err != nil {
		return refuse("read %s: %v", path, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var b strings.Builder
	n := 0
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		if n == maxListEntries {
			fmt.Fprintf(&b, "... (listing truncated at %d entries)\n", maxListEntries)
			break
		}
		b.WriteString(e.Name())
		if e.IsDir() {
			b.WriteString("/")
		}
		b.WriteString("\n")
		n++
	}
	return Response{OK: true, Content: fmt.Sprintf("directory %s (%d entries):\n%s", path, n, b.String())}
}

// protected is the ONE write gate every write the tool endpoint performs
// passes — the write and edit ops, `fmt --write`, and the files other CLI
// ops are known to write (cliSchema.writes). It applies the shared
// fileguard.Protection the FS effect uses: `.git` is always read-only to the
// agent (M6 — the confined git adapter trusts the clone's config, so the
// tools must not be a way to plant one), and fs_deny_write (M7) protects the
// operator's paths. Both are case- and normalization-folded (#1559).
// It returns a refusal reason, or "" when the write may proceed.
func (h *Host) protected(op, p string) string {
	rel, err := h.root.Rel(p)
	if err != nil {
		// An escaping path is refused by the root handle anyway; check the
		// lexical spelling so the refusal still names a protected component.
		rel = p
	}
	v := fileguard.Protection{GitDir: true, DenyWrite: h.res.DenyWrite}.Check(rel)
	switch {
	case v.GitDir:
		return fmt.Sprintf("%s %s: .git/ is read-only to the lane's tools", op, p)
	case v.Pattern != "":
		return fmt.Sprintf("%s %s: matches fs_deny_write %q — read-only under this policy", op, p, v.Pattern)
	}
	return ""
}

func (h *Host) write(req Request) Response {
	if r := h.needRoot(); r != nil {
		return *r
	}
	if req.Path == "" {
		return refuse("write: path is required")
	}
	if why := h.protected("write", req.Path); why != "" {
		return refuse("%s", why)
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
	if why := h.protected("edit", req.Path); why != "" {
		return refuse("%s", why)
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
