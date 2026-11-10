package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/gitexec"
)

// Ref modes (design doc §6.2).
const (
	refDefault  = "default"  // remote default-branch head (or ref.default): what breaks if we ship today
	refPinned   = "pinned"   // the manifest's pinned ref: reproducible
	refWorktree = "worktree" // the operator's checkout as it sits: debugging only, labelled DIRTY
)

// Source records where a corpus's files came from; every report row carries it.
type Source struct {
	Corpus   string     `json:"corpus"`
	Repo     string     `json:"repo"`
	Mode     string     `json:"mode"`
	Ref      string     `json:"ref"`
	SHA      string     `json:"sha"`
	State    string     `json:"state"` // "clean snapshot" or a DIRTY description
	Cached   bool       `json:"cached"`
	Root     string     `json:"-"` // materialised copy (scratch)
	Local    *LocalInfo `json:"local,omitempty"`
	Skipped  int        `json:"skipped_links,omitempty"`
	Freshest string     `json:"freshness,omitempty"` // local checkout vs the snapshot
}

// LocalInfo is the operator's checkout, read with --no-optional-locks so not even
// the index is refreshed. Never fetched, pulled, checked out or written.
type LocalInfo struct {
	Path      string `json:"path"`
	Branch    string `json:"branch"`
	HEAD      string `json:"head"`
	Porcelain int    `json:"uncommitted_paths"`
}

// Remote fetches snapshots of GitHub repos. The real one shells out to `gh`
// (the operator's own auth, so private repos work attended); tests stub it.
type Remote interface {
	ResolveRef(repo, ref string) (string, error) // ref "" = the default branch
	Tarball(repo, sha string, w io.Writer) error
}

type ghRemote struct{}

func ghOut(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func (ghRemote) ResolveRef(repo, ref string) (string, error) {
	if ref == "" {
		b, err := ghOut("api", "repos/"+repo, "--jq", ".default_branch")
		if err != nil {
			return "", err
		}
		ref = b
	}
	sha, err := ghOut("api", "repos/"+repo+"/commits/"+ref, "--jq", ".sha")
	if err != nil {
		return "", err
	}
	if len(sha) != 40 {
		return "", fmt.Errorf("gh: %s@%s resolved to %q, not a commit sha", repo, ref, sha)
	}
	return sha, nil
}

func (ghRemote) Tarball(repo, sha string, w io.Writer) error {
	cmd := exec.Command("gh", "api", "repos/"+repo+"/tarball/"+sha)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = w, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh tarball %s@%s: %v: %s", repo, sha, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Materialiser turns a corpus into a scratch copy. It never writes outside
// StateDir and the run directory, and it never runs a mutating git command.
type Materialiser struct {
	StateDir string // snapshot cache: <state>/snapshots/<owner>__<name>/<sha>.tar.gz
	SelfRoot string // the ailang checkout the tool runs from
	Offline  bool   // use the last fetched sha per (repo, ref); no network
	Remote   Remote
}

func repoSlug(repo string) string { return strings.ReplaceAll(repo, "/", "__") }

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

// Materialise copies corpus c (in ref mode mode) into dest and describes the source.
func (m *Materialiser) Materialise(c Corpus, mode, dest string) (Source, error) {
	src := Source{Corpus: c.Name, Repo: c.Repo, Mode: mode, Root: dest}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return src, err
	}
	local := expandHome(c.Local)
	if c.Repo == selfRepo {
		local = m.SelfRoot
	}
	if mode == refWorktree {
		if local == "" {
			return src, fmt.Errorf("corpus %s: REF=worktree needs a `local:` checkout", c.Name)
		}
		info, err := readLocal(local)
		if err != nil {
			return src, fmt.Errorf("corpus %s: unreadable checkout %s: %w", c.Name, local, err)
		}
		src.Local, src.Ref, src.SHA = info, info.Branch, info.HEAD
		src.State = fmt.Sprintf("DIRTY (worktree mode): %d uncommitted paths", info.Porcelain)
		roots := []string{""}
		if c.Repo == selfRepo {
			roots = selfPaths(c)
		}
		for _, r := range roots {
			if err := copyCheckout(filepath.Join(local, r), filepath.Join(dest, r), c.ExamplesManifest, local); err != nil {
				return src, fmt.Errorf("corpus %s: copy %s: %w", c.Name, local, err)
			}
		}
		return src, nil
	}
	ref := c.Ref.Default
	if mode == refPinned {
		if c.Ref.Pinned == "" {
			return src, fmt.Errorf("corpus %s: REF=pinned but the manifest has no ref.pinned", c.Name)
		}
		ref = c.Ref.Pinned
	}
	if c.Repo == selfRepo {
		return m.materialiseSelf(c, ref, src)
	}
	sha, cached, err := m.snapshot(c.Repo, ref)
	if err != nil {
		return src, fmt.Errorf("corpus %s: %w", c.Name, err)
	}
	src.Ref, src.SHA, src.Cached, src.State = refName(ref), sha, cached, "clean snapshot"
	tp := m.tarPath(c.Repo, sha)
	if src.Skipped, err = extractTarGz(tp, dest); err != nil {
		// A cached snapshot that does not extract would fail every later run;
		// drop it so the next online run fetches it again.
		rmErr := os.Remove(tp)
		return src, fmt.Errorf("corpus %s: snapshot %s: %w (cached tarball removed: %v; the next online run refetches it)", c.Name, sha, err, rmErr)
	}
	if local != "" {
		if info, err := readLocal(local); err == nil {
			src.Local = info
			src.Freshest = "local checkout differs from the snapshot"
			if info.HEAD == sha && info.Porcelain == 0 {
				src.Freshest = "local checkout = snapshot"
			}
		} else {
			src.Freshest = "local checkout not readable: " + err.Error()
		}
	}
	return src, nil
}

func refName(ref string) string {
	if ref == "" {
		return defaultBranchLabel
	}
	return ref
}

const defaultBranchLabel = "(remote default branch)"

// selfPaths are the top-level directories an in-repo corpus covers, taken from its
// include globs (the in-repo corpus must be narrowed: the repo is ~45k files).
func selfPaths(c Corpus) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range c.Include {
		seg := strings.SplitN(g, "/", 2)[0]
		if !strings.ContainsAny(seg, "*?") && !seen[seg] {
			seen[seg] = true
			out = append(out, seg)
		}
	}
	return out
}

func (m *Materialiser) materialiseSelf(c Corpus, ref string, src Source) (Source, error) {
	paths := selfPaths(c)
	if len(paths) == 0 {
		return src, fmt.Errorf("corpus %s: an in-repo corpus needs include globs with a literal first segment", c.Name)
	}
	if ref == "" {
		ref = "HEAD"
	}
	info, err := readLocal(m.SelfRoot)
	if err != nil {
		return src, fmt.Errorf("corpus %s: %w", c.Name, err)
	}
	sha, err := gitOut(m.SelfRoot, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return src, fmt.Errorf("corpus %s: %w", c.Name, err)
	}
	dirty, err := gitOut(m.SelfRoot, append([]string{"status", "--porcelain", "--"}, paths...)...)
	if err != nil {
		return src, fmt.Errorf("corpus %s: %w", c.Name, err)
	}
	src.Local, src.Ref, src.SHA = info, ref, sha
	src.State = "clean snapshot (git archive of " + ref + ")"
	if n := countLines(dirty); n > 0 {
		src.State = fmt.Sprintf("snapshot of %s; NOTE %d uncommitted paths under %s are NOT in it", ref, n, strings.Join(paths, ","))
	}
	cmd := gitexec.Command(append([]string{"--no-optional-locks", "-C", m.SelfRoot, "archive", "--format=tar", sha, "--"}, paths...)...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return src, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return src, fmt.Errorf("corpus %s: git archive: %w", c.Name, err)
	}
	skipped, xerr := extractTar(out, src.Root, 0)
	if werr := cmd.Wait(); werr != nil {
		return src, fmt.Errorf("corpus %s: git archive: %v: %s", c.Name, werr, stderr.String())
	}
	src.Skipped = skipped
	return src, xerr
}

func (m *Materialiser) tarPath(repo, sha string) string {
	return filepath.Join(m.StateDir, "snapshots", repoSlug(repo), sha+".tar.gz")
}

// snapshot makes sure <sha>.tar.gz of repo@ref is in the cache and returns its sha.
// Online it resolves the ref and records it as the last sha for (repo, ref);
// offline it uses that record and says so through the cached flag.
func (m *Materialiser) snapshot(repo, ref string) (string, bool, error) {
	dir := filepath.Join(m.StateDir, "snapshots", repoSlug(repo))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	label := ref
	if label == "" {
		label = "default-branch"
	}
	lastFile := filepath.Join(dir, "last-"+strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(label))
	if m.Offline {
		b, err := os.ReadFile(lastFile)
		if err != nil {
			return "", false, fmt.Errorf("offline and no earlier fetch of %s@%s: %w", repo, refName(ref), err)
		}
		sha := strings.TrimSpace(string(b))
		if _, err := os.Stat(m.tarPath(repo, sha)); err != nil {
			return "", false, fmt.Errorf("offline: cached snapshot %s missing: %w", sha, err)
		}
		return sha, true, nil
	}
	sha, err := m.Remote.ResolveRef(repo, ref)
	if err != nil {
		return "", false, err
	}
	tp := m.tarPath(repo, sha)
	cached := true
	if _, err := os.Stat(tp); err != nil {
		cached = false
		tmp, err := os.CreateTemp(dir, ".fetch-*")
		if err != nil {
			return "", false, err
		}
		ferr := m.Remote.Tarball(repo, sha, tmp)
		cerr := tmp.Close()
		if ferr != nil || cerr != nil {
			os.Remove(tmp.Name())
			return "", false, errors.Join(ferr, cerr)
		}
		// Only a tarball that reads to the end (within the size caps) is cached.
		if err := verifyTarGz(tmp.Name()); err != nil {
			os.Remove(tmp.Name())
			return "", false, fmt.Errorf("downloaded snapshot of %s@%s is not a readable tar.gz: %w", repo, sha, err)
		}
		if err := os.Rename(tmp.Name(), tp); err != nil {
			return "", false, err
		}
	}
	return sha, cached, os.WriteFile(lastFile, []byte(sha+"\n"), 0o644)
}

// Extraction caps: a corpus is source code, so anything near these is a wrong or
// hostile archive, and filling the disk is not a measurement.
var (
	maxEntryBytes int64 = 64 << 20 // one file
	maxTotalBytes int64 = 4 << 30  // one archive
)

func extractTarGz(path, dest string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	return extractTar(gz, dest, 1)
}

// verifyTarGz reads a tar.gz to the end without writing anything.
func verifyTarGz(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := checkSize(h, &total); err != nil {
			return err
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return err
		}
	}
}

func checkSize(h *tar.Header, total *int64) error {
	if h.Typeflag != tar.TypeReg {
		return nil
	}
	if h.Size > maxEntryBytes {
		return fmt.Errorf("archive entry %q is %d bytes, over the %d-byte per-file cap", h.Name, h.Size, maxEntryBytes)
	}
	*total += h.Size
	if *total > maxTotalBytes {
		return fmt.Errorf("archive exceeds the %d-byte total cap at %q", maxTotalBytes, h.Name)
	}
	return nil
}

// extractTar unpacks regular files and directories under dest, dropping strip
// leading path components. Links are skipped (counted), never followed, and an
// entry that would land outside dest, or over the size caps, is an error.
func extractTar(r io.Reader, dest string, strip int) (int, error) {
	tr := tar.NewReader(r)
	skipped := 0
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return skipped, nil
		}
		if err != nil {
			return skipped, err
		}
		parts := strings.Split(strings.TrimPrefix(h.Name, "./"), "/")
		if len(parts) <= strip {
			continue
		}
		rel := filepath.Clean(filepath.Join(parts[strip:]...))
		if rel == "." {
			continue
		}
		// filepath.IsLocal is the platform-aware test: it rejects absolute paths,
		// any ".." escape in the host separator (a "../" prefix check misses `..\`
		// on Windows), and reserved device names. The Rel check below is the same
		// guarantee stated on the joined path, so a future edit to the lines above
		// cannot silently reopen the hole.
		if !filepath.IsLocal(rel) {
			return skipped, fmt.Errorf("unsafe path in archive: %q", h.Name)
		}
		target := filepath.Join(dest, rel)
		if within, err := filepath.Rel(dest, target); err != nil || !filepath.IsLocal(within) {
			return skipped, fmt.Errorf("unsafe path in archive: %q", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return skipped, err
			}
		case tar.TypeReg:
			if err := checkSize(h, &total); err != nil {
				return skipped, err
			}
			if err := writeFile(target, tr); err != nil {
				return skipped, err
			}
		case tar.TypeSymlink, tar.TypeLink:
			skipped++
		}
	}
}

func writeFile(target string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// copyCheckout copies what `check` reads (.ail, ailang.toml, ailang.lock and the
// corpus's examples manifest) from a working tree into dest. The checkout is only read.
func copyCheckout(from, dest, manifestRel, checkoutRoot string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != from && prunedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		relToRoot, _ := filepath.Rel(checkoutRoot, p)
		name := d.Name()
		if !strings.HasSuffix(name, ".ail") && name != "ailang.toml" && name != "ailang.lock" &&
			(manifestRel == "" || filepath.ToSlash(relToRoot) != manifestRel) {
			return nil
		}
		rel, _ := filepath.Rel(from, p)
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		return writeFile(filepath.Join(dest, rel), in)
	})
}

func gitOut(dir string, args ...string) (string, error) {
	cmd := gitexec.Command(append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %v: %s", strings.Join(args, " "), dir, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func countLines(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

// readLocal reads a checkout's branch, HEAD and uncommitted-path count, read-only.
func readLocal(dir string) (*LocalInfo, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	head, err := gitOut(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	branch, err := gitOut(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	st, err := gitOut(dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	return &LocalInfo{Path: dir, Branch: branch, HEAD: head, Porcelain: countLines(st)}, nil
}
