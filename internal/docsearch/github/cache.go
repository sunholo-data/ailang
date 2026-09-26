package github

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/docsearch"
)

const cacheTTL = time.Hour

type cacheEntry struct {
	CreatedAt time.Time                `json:"created_at"`
	Results   []docsearch.SearchResult `json:"results"`
	Stats     docsearch.SearchStats    `json:"stats"`
}

func cacheKey(repo, query, subdir string, limit int) string {
	digest := sha256.Sum256([]byte(repo + "\x00" + query + "\x00" + subdir + "\x00" + fmt.Sprint(limit)))
	return hex.EncodeToString(digest[:])
}

func cachePath(repo, query, subdir string, limit int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".ailang", "cache", "docsearch", "github", cacheKey(repo, query, subdir, limit)+".json"), nil
}

func readCache(repo, query, subdir string, limit int, now time.Time) ([]docsearch.SearchResult, docsearch.SearchStats, bool) {
	path, err := cachePath(repo, query, subdir, limit)
	if err != nil {
		return nil, docsearch.SearchStats{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, docsearch.SearchStats{}, false
	}
	var entry cacheEntry
	if json.Unmarshal(data, &entry) != nil || entry.CreatedAt.IsZero() || now.Sub(entry.CreatedAt) >= cacheTTL {
		return nil, docsearch.SearchStats{}, false
	}
	return entry.Results, entry.Stats, true
}

func writeCache(repo, query, subdir string, limit int, results []docsearch.SearchResult, stats docsearch.SearchStats, now time.Time) error {
	path, err := cachePath(repo, query, subdir, limit)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheEntry{CreatedAt: now, Results: results, Stats: stats})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Close()
	} else {
		tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
