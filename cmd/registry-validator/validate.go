package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// runAilangCheck runs ailang check on the package directory.
// Uses `ailang check --package .` for packages with ailang.toml (resolves dependencies
// and cross-module types), falls back to per-file checks for bare packages.
func runAilangCheck(dir string) (bool, string) {
	// If ailang.toml exists, use package-level check (resolves deps + cross-module types)
	if fileExists(filepath.Join(dir, "ailang.toml")) {
		// Step 1: Rewrite any remaining path deps to registry versions.
		// The publish command now rewrites path deps client-side, but older clients
		// may still send tarballs with path deps.
		manifest, err := pkg.LoadManifest(dir)
		if err == nil {
			hasPathDeps := false
			for _, dep := range manifest.Dependencies {
				if dep.Path != "" {
					hasPathDeps = true
					break
				}
			}
			if hasPathDeps {
				if err := rewritePathDepsToRegistry(dir, manifest); err != nil {
					log.Printf("Warning: failed to rewrite path deps: %v", err)
				}
			}
		}

		// Step 2: Resolve deps and generate lock file directly via Go API.
		// This downloads all transitive deps to cache and writes ailang.lock.
		// We call the resolver directly instead of shelling out to avoid issues
		// with ailang install modifying ailang.toml or ailang lock silently failing.
		resolveManifest, err := pkg.LoadManifest(dir)
		if err != nil {
			return false, fmt.Sprintf("Failed to load manifest for dependency resolution: %v", err)
		}

		if len(resolveManifest.Dependencies) > 0 {
			resolved, err := pkg.ResolveDependencies(resolveManifest, dir)
			if err != nil {
				return false, fmt.Sprintf("Dependency resolution failed: %v", err)
			}

			locked := make([]pkg.LockedPackage, len(resolved))
			for i, r := range resolved {
				locked[i] = pkg.LockedPackage(r)
			}

			lf := pkg.NewLockFile(locked, "registry-validator")
			if err := lf.Save(dir); err != nil {
				return false, fmt.Sprintf("Failed to write lock file: %v", err)
			}
			log.Printf("Resolved %d dependencies, lock file written", len(resolved))
		} else {
			// No deps — write empty lock file for consistency
			lf := pkg.NewLockFile(nil, "registry-validator")
			if err := lf.Save(dir); err != nil {
				return false, fmt.Sprintf("Failed to write lock file: %v", err)
			}
		}

		// Step 3: Run package-level type check
		cmd := exec.Command("ailang", "check", "--package", ".")
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, string(output)
		}
		return true, ""
	}

	// Fallback: check individual files (for packages without ailang.toml)
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})

	for _, f := range files {
		cmd := exec.Command("ailang", "check", f)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, string(output)
		}
	}
	return true, ""
}

// runAilangVerify runs ailang verify and parses results.
func runAilangVerify(dir string) (verified, total, skipped int) {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})

	for _, f := range files {
		cmd := exec.Command("ailang", "verify", "--json", f)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			skipped++
			continue
		}
		// Parse JSON output to count verified/counterexample/skipped
		var results []struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(output, &results) == nil {
			for _, r := range results {
				total++
				switch r.Status {
				case "verified":
					verified++
				case "skipped", "timeout", "error":
					skipped++
				}
			}
		}
	}
	return
}

// rewritePathDepsToRegistry reads ailang.toml, replaces path dependencies with
// registry version strings (latest version from registry index), and writes back.
// This enables the validator to resolve deps that were local path deps during development.
func rewritePathDepsToRegistry(dir string, manifest *pkg.PackageManifest) error {
	tomlPath := filepath.Join(dir, "ailang.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("read ailang.toml: %w", err)
	}
	content := string(data)

	for depName, dep := range manifest.Dependencies {
		if dep.Path == "" {
			continue // Not a path dep — skip
		}
		// Replace the path dep line with a version dep.
		// Look up the latest version from registry index.
		version := lookupLatestVersion(depName)
		if version == "" {
			log.Printf("Warning: dependency %s not found in registry, skipping", depName)
			continue
		}

		// Replace various path dep formats:
		//   "vendor/name" = { path = "..." }
		//   "vendor/name" = { path = "../something" }
		// with:
		//   "vendor/name" = "version"
		old := fmt.Sprintf(`"%s" = { path = "%s" }`, depName, dep.Path)
		new := fmt.Sprintf(`"%s" = "%s"`, depName, version)
		if strings.Contains(content, old) {
			content = strings.Replace(content, old, new, 1)
			log.Printf("Rewrote dep %s: path -> registry %s", depName, version)
		} else {
			// Try without quotes around path value
			old2 := fmt.Sprintf(`"%s" = {path = "%s"}`, depName, dep.Path)
			if strings.Contains(content, old2) {
				content = strings.Replace(content, old2, new, 1)
				log.Printf("Rewrote dep %s: path -> registry %s", depName, version)
			} else {
				log.Printf("Warning: could not find path dep pattern for %s in ailang.toml", depName)
			}
		}
	}

	return os.WriteFile(tomlPath, []byte(content), 0644)
}

// lookupLatestVersion fetches the registry index and finds the latest version of a package.
func lookupLatestVersion(pkgName string) string {
	registryURL := os.Getenv("AILANG_REGISTRY")
	if registryURL == "" {
		registryURL = "https://storage.googleapis.com/ailang-registry"
	}
	indexURL := registryURL + "/index.json"

	resp, err := http.Get(indexURL)
	if err != nil {
		log.Printf("Warning: failed to fetch registry index: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var index struct {
		Packages []struct {
			Name   string `json:"name"`
			Latest string `json:"latest"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return ""
	}
	for _, p := range index.Packages {
		if p.Name == pkgName {
			return p.Latest
		}
	}
	return ""
}

func getAilangVersion() string {
	cmd := exec.Command("ailang", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getMetaString(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMetaStringSlice(meta map[string]interface{}, key string) []string {
	if meta == nil {
		return nil
	}
	v, ok := meta[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func jsonError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
