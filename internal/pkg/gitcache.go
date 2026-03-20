package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitCache manages cached git clones for git-based dependencies.
// Clones are stored in ~/.ailang/cache/git/<url-hash>/.
type GitCache struct {
	baseDir string
}

// NewGitCache creates a cache rooted at ~/.ailang/cache/git/.
func NewGitCache() (*GitCache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	dir := filepath.Join(home, ".ailang", "cache", "git")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create git cache dir: %w", err)
	}
	return &GitCache{baseDir: dir}, nil
}

// Resolve clones or fetches a git repo and checks out the specified tag or rev.
// Returns the local path to the package (including subdir) and the resolved commit hash.
func (gc *GitCache) Resolve(gitURL, tag, rev, subdir string) (localPath string, resolvedRev string, err error) {
	cacheDir := gc.CacheDir(gitURL)

	if rev != "" {
		// Exact commit — clone if needed, checkout
		if err := gc.ensureClone(gitURL, cacheDir, ""); err != nil {
			return "", "", err
		}
		if err := gc.checkout(cacheDir, rev); err != nil {
			return "", "", fmt.Errorf("failed to checkout rev %s: %w", rev, err)
		}
		resolvedRev = rev
	} else if tag != "" {
		// Tag — clone with tag, or fetch and checkout if cached
		if err := gc.ensureClone(gitURL, cacheDir, tag); err != nil {
			return "", "", err
		}
		// Resolve tag to commit hash
		resolved, err := gc.revParse(cacheDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve tag %s: %w", tag, err)
		}
		resolvedRev = resolved
	} else {
		return "", "", fmt.Errorf("git dependency must specify tag or rev")
	}

	// Build final path with optional subdir
	localPath = cacheDir
	if subdir != "" {
		localPath = filepath.Join(cacheDir, subdir)
	}

	// Verify the path exists
	if _, err := os.Stat(localPath); err != nil {
		return "", "", fmt.Errorf("subdir %q not found in git repo %s", subdir, gitURL)
	}

	return localPath, resolvedRev, nil
}

// CacheDir returns the deterministic cache directory for a git URL.
func (gc *GitCache) CacheDir(gitURL string) string {
	h := sha256.Sum256([]byte(gitURL))
	return filepath.Join(gc.baseDir, hex.EncodeToString(h[:])[:16])
}

// ensureClone clones the repo if not cached, or fetches updates if cached.
func (gc *GitCache) ensureClone(gitURL, cacheDir, branch string) error {
	if _, err := os.Stat(filepath.Join(cacheDir, ".git")); err == nil {
		// Already cloned — fetch updates
		return gc.fetch(cacheDir, branch)
	}

	// Fresh clone
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "--branch", branch, "--depth", "1")
	}
	args = append(args, gitURL, cacheDir)

	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed for %s: %w", gitURL, err)
	}
	return nil
}

// fetch updates the cached repo.
func (gc *GitCache) fetch(cacheDir, branch string) error {
	if branch != "" {
		// Fetch the specific tag/branch and checkout
		cmd := exec.Command("git", "fetch", "--depth", "1", "origin", "tag", branch, "--force")
		cmd.Dir = cacheDir
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Tag fetch failed — try as branch
			cmd2 := exec.Command("git", "fetch", "--depth", "1", "origin", branch)
			cmd2.Dir = cacheDir
			cmd2.Stderr = os.Stderr
			if err2 := cmd2.Run(); err2 != nil {
				return fmt.Errorf("git fetch failed for %s: %w", branch, err)
			}
		}
		return gc.checkout(cacheDir, branch)
	}
	return nil
}

// checkout checks out a specific ref.
func (gc *GitCache) checkout(cacheDir, ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = cacheDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s failed: %w", ref, err)
	}
	return nil
}

// revParse returns the current HEAD commit hash.
func (gc *GitCache) revParse(cacheDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = cacheDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
