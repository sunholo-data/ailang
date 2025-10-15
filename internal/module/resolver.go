// Package module provides path resolution utilities for AILANG modules.
package module

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Resolver handles module path resolution with platform-specific normalization
type Resolver struct {
	// projectRoot is the root directory of the current project
	projectRoot string

	// stdlibPath is the path to the standard library
	stdlibPath string

	// searchPaths are additional directories to search
	searchPaths []string

	// caseSensitive indicates if the filesystem is case-sensitive
	caseSensitive bool
}

// NewResolver creates a new path resolver
func NewResolver() *Resolver {
	return &Resolver{
		projectRoot:   findProjectRoot(),
		stdlibPath:    findStdlibPath(),
		searchPaths:   getSearchPaths(),
		caseSensitive: isFileSystemCaseSensitive(),
	}
}

// NormalizePath normalizes a file path for the current platform
func (r *Resolver) NormalizePath(path string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Clean the path (resolve . and ..)
	path = filepath.Clean(path)

	// Make absolute if relative
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to make path absolute: %w", err)
		}
		path = abs
	}

	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If file doesn't exist yet, just return cleaned path
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Handle case-insensitive filesystems
	if !r.caseSensitive {
		_ = r.normalizeCaseInsensitive(resolved) // Currently unused but preserved for future
	}

	return resolved, nil
}

// ResolveImport resolves an import path to a file path
func (r *Resolver) ResolveImport(importPath string, currentFile string) (string, error) {
	// Handle different import types
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		// Relative import
		return r.resolveRelativeImport(importPath, currentFile)
	} else if strings.HasPrefix(importPath, "std/") {
		// Standard library import
		return r.resolveStdlibImport(importPath)
	} else if strings.Contains(importPath, "/") {
		// Project module import (e.g., "data/structures")
		return r.resolveProjectImport(importPath)
	} else {
		// Local module import (e.g., "utils")
		return r.resolveLocalImport(importPath, currentFile)
	}
}

// resolveRelativeImport resolves a relative import path
func (r *Resolver) resolveRelativeImport(importPath, currentFile string) (string, error) {
	if currentFile == "" {
		return "", fmt.Errorf("relative import '%s' requires a current file context", importPath)
	}

	dir := filepath.Dir(currentFile)
	path := filepath.Join(dir, importPath)

	// Add .ail extension if not present
	if !strings.HasSuffix(path, ".ail") {
		path += ".ail"
	}

	// Normalize the path
	normalized, err := r.NormalizePath(path)
	if err != nil {
		return "", err
	}

	// Check if file exists
	if _, err := os.Stat(normalized); err != nil {
		return "", fmt.Errorf("module not found: %s", importPath)
	}

	return normalized, nil
}

// resolveStdlibImport resolves a standard library import
func (r *Resolver) resolveStdlibImport(importPath string) (string, error) {
	// Remove std/ prefix
	libPath := strings.TrimPrefix(importPath, "std/")

	// Build full path
	path := filepath.Join(r.stdlibPath, libPath)

	// Add .ail extension if not present
	if !strings.HasSuffix(path, ".ail") {
		path += ".ail"
	}

	// Normalize
	normalized, err := r.NormalizePath(path)
	if err != nil {
		return "", err
	}

	// Check existence
	if _, err := os.Stat(normalized); err != nil {
		return "", fmt.Errorf("stdlib module not found: %s", importPath)
	}

	return normalized, nil
}

// resolveProjectImport resolves a project module import
func (r *Resolver) resolveProjectImport(importPath string) (string, error) {
	// Try project root first
	path := filepath.Join(r.projectRoot, importPath)
	if !strings.HasSuffix(path, ".ail") {
		path += ".ail"
	}

	normalized, err := r.NormalizePath(path)
	if err == nil {
		if _, err := os.Stat(normalized); err == nil {
			return normalized, nil
		}
	}

	// Try search paths
	for _, searchPath := range r.searchPaths {
		path := filepath.Join(searchPath, importPath)
		if !strings.HasSuffix(path, ".ail") {
			path += ".ail"
		}

		normalized, err := r.NormalizePath(path)
		if err == nil {
			if _, err := os.Stat(normalized); err == nil {
				return normalized, nil
			}
		}
	}

	return "", fmt.Errorf("project module not found: %s", importPath)
}

// resolveLocalImport resolves a local module import
func (r *Resolver) resolveLocalImport(importPath, currentFile string) (string, error) {
	// Try relative to current file first
	if currentFile != "" {
		dir := filepath.Dir(currentFile)
		path := filepath.Join(dir, importPath)
		if !strings.HasSuffix(path, ".ail") {
			path += ".ail"
		}

		normalized, err := r.NormalizePath(path)
		if err == nil {
			if _, err := os.Stat(normalized); err == nil {
				return normalized, nil
			}
		}
	}

	// Try project root
	return r.resolveProjectImport(importPath)
}

// GetModuleIdentity derives a module identity from a file path
func (r *Resolver) GetModuleIdentity(filePath string) (string, error) {
	// Normalize the file path first
	normalized, err := r.NormalizePath(filePath)
	if err != nil {
		return "", err
	}

	// Remove .ail extension
	identity := strings.TrimSuffix(normalized, ".ail")

	// If in stdlib, use std/ prefix
	if strings.HasPrefix(normalized, r.stdlibPath) {
		rel, err := filepath.Rel(r.stdlibPath, identity)
		if err == nil {
			return "std/" + strings.ReplaceAll(rel, string(filepath.Separator), "/"), nil
		}
	}

	// If in project root, use relative path
	if strings.HasPrefix(normalized, r.projectRoot) {
		rel, err := filepath.Rel(r.projectRoot, identity)
		if err == nil {
			return strings.ReplaceAll(rel, string(filepath.Separator), "/"), nil
		}
	}

	// Otherwise, use base name
	return filepath.Base(identity), nil
}

// ValidateModuleName checks if a module name matches its expected name based on file path
func (r *Resolver) ValidateModuleName(declaredName, filePath string) error {
	expectedIdentity, err := r.GetModuleIdentity(filePath)
	if err != nil {
		return err
	}

	// For stdlib modules, allow some flexibility
	if strings.HasPrefix(expectedIdentity, "std/") {
		if declaredName == expectedIdentity || declaredName == filepath.Base(expectedIdentity) {
			return nil
		}
	}

	// For project modules, require exact match
	if declaredName != expectedIdentity && declaredName != filepath.Base(expectedIdentity) {
		return fmt.Errorf("module name '%s' doesn't match expected '%s' for file %s",
			declaredName, expectedIdentity, filePath)
	}

	return nil
}

// Helper functions

// findProjectRoot finds the project root directory
func findProjectRoot() string {
	// Look for markers like go.mod, .git, ailang.yaml
	markers := []string{"go.mod", ".git", "ailang.yaml", ".ailang"}

	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		// Check for markers
		for _, marker := range markers {
			path := filepath.Join(dir, marker)
			if _, err := os.Stat(path); err == nil {
				return dir
			}
		}

		// Move up
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Default to current directory
	pwd, _ := os.Getwd()
	return pwd
}

// findStdlibPath finds the standard library path
func findStdlibPath() string {
	// Check environment variable
	if stdlib := os.Getenv("AILANG_STDLIB"); stdlib != "" {
		return stdlib
	}

	// Check relative to executable
	if exe, err := os.Executable(); err == nil {
		// Try ../stdlib relative to executable
		stdlib := filepath.Join(filepath.Dir(exe), "..", "stdlib")
		if info, err := os.Stat(stdlib); err == nil && info.IsDir() {
			return stdlib
		}

		// Try stdlib in same directory as executable
		stdlib = filepath.Join(filepath.Dir(exe), "stdlib")
		if info, err := os.Stat(stdlib); err == nil && info.IsDir() {
			return stdlib
		}
	}

	// Check in project root
	projectRoot := findProjectRoot()
	stdlib := filepath.Join(projectRoot, "stdlib")
	if info, err := os.Stat(stdlib); err == nil && info.IsDir() {
		return stdlib
	}

	// Default to ./stdlib
	return filepath.Join(".", "stdlib")
}

// getSearchPaths returns additional search paths for modules
func getSearchPaths() []string {
	paths := []string{}

	// Add AILANG_PATH if set
	if ailangPath := os.Getenv("AILANG_PATH"); ailangPath != "" {
		for _, p := range strings.Split(ailangPath, string(os.PathListSeparator)) {
			if p != "" {
				paths = append(paths, p)
			}
		}
	}

	// Add user modules directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".ailang", "modules"))
	}

	// Add project root
	paths = append(paths, findProjectRoot())

	return paths
}

// isFileSystemCaseSensitive checks if the filesystem is case-sensitive
func isFileSystemCaseSensitive() bool {
	// On Windows and macOS, filesystems are typically case-insensitive
	// On Linux, they're typically case-sensitive
	switch runtime.GOOS {
	case "windows", "darwin":
		return false
	default:
		return true
	}
}

// normalizeCaseInsensitive normalizes the case of a path on case-insensitive systems
func (r *Resolver) normalizeCaseInsensitive(path string) string {
	// On case-insensitive systems, we could try to match the actual case
	// For now, just return the path as-is
	return path
}

// GetResolutionOrder returns the order in which paths will be searched
func (r *Resolver) GetResolutionOrder(importPath, currentFile string) []string {
	order := []string{}

	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		// Relative import
		if currentFile != "" {
			dir := filepath.Dir(currentFile)
			order = append(order, filepath.Join(dir, importPath))
		}
	} else if strings.HasPrefix(importPath, "std/") {
		// Stdlib
		libPath := strings.TrimPrefix(importPath, "std/")
		order = append(order, filepath.Join(r.stdlibPath, libPath))
	} else {
		// Project/local import
		order = append(order, filepath.Join(r.projectRoot, importPath))
		for _, searchPath := range r.searchPaths {
			order = append(order, filepath.Join(searchPath, importPath))
		}
	}

	// Add .ail extension variants
	extended := []string{}
	for _, path := range order {
		extended = append(extended, path)
		if !strings.HasSuffix(path, ".ail") {
			extended = append(extended, path+".ail")
		}
	}

	return extended
}
