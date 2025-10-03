package loader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// ModuleLoader loads and caches modules
type ModuleLoader struct {
	cache              map[string]*LoadedModule
	basePath           string // Base directory for relative imports
	warnedLegacyStdlib bool   // Track if we've warned about stdlib/std/* usage
}

// LoadedModule represents a loaded and parsed module
type LoadedModule struct {
	Path         string
	File         *ast.File
	Imports      []string                 // Module paths this module imports
	Exports      map[string]*ast.FuncDecl // Export table (for now, just functions)
	Types        map[string]*ast.TypeDecl // Exported type declarations
	Constructors map[string]string        // Constructor name -> Type name mapping
	Core         *core.Program            // Core representation (after elaboration)
	Iface        *iface.Iface             // Module interface (after type checking)
}

// NewModuleLoader creates a new module loader
func NewModuleLoader(basePath string) *ModuleLoader {
	return &ModuleLoader{
		cache:    make(map[string]*LoadedModule),
		basePath: basePath,
	}
}

// Preload adds a pre-loaded module to the cache
//
// This is used to inject modules that were already loaded and elaborated
// by the pipeline, avoiding redundant loading and elaboration.
//
// Parameters:
//   - path: The module path
//   - loaded: The LoadedModule with Core AST already populated
func (ml *ModuleLoader) Preload(path string, loaded *LoadedModule) {
	canonicalID := CanonicalModuleID(path)
	ml.cache[canonicalID] = loaded
}

// canonicalizeModulePath normalizes import paths and detects legacy patterns
//
// Returns the canonical path and a flag indicating if a legacy pattern was used.
// Legacy pattern: "stdlib/std/io" → canonical: "std/io" (legacy=true)
// Modern pattern: "std/io" → canonical: "std/io" (legacy=false)
func canonicalizeModulePath(path string) (string, bool) {
	legacy := false
	// Strip leading "./" or ".\" for cross-platform safety
	path = strings.TrimPrefix(strings.TrimPrefix(path, "./"), ".\\")

	// Detect and normalize legacy stdlib/std/* pattern
	if strings.HasPrefix(path, "stdlib/std/") {
		path = strings.TrimPrefix(path, "stdlib/")
		legacy = true
	}

	return path, legacy
}

// Load loads a module by path
func (ml *ModuleLoader) Load(path string) (*LoadedModule, error) {
	// Canonicalize the import path and check for legacy patterns
	canonPath, isLegacy := canonicalizeModulePath(path)

	// Emit one-time warning for legacy stdlib/std/* usage
	if isLegacy && !ml.warnedLegacyStdlib {
		fmt.Fprintf(os.Stderr, "Warning: import path 'stdlib/std/*' is deprecated; use 'std/*' instead\n")
		ml.warnedLegacyStdlib = true
	}

	// Use canonicalized path for all subsequent operations
	canonicalID := CanonicalModuleID(canonPath)

	// Check cache with canonical ID
	if loaded, ok := ml.cache[canonicalID]; ok {
		return loaded, nil
	}

	// Track search attempts for error reporting
	var searchTrace []string

	// Resolve path and track attempts
	fullPath := ""

	// Try relative path first
	if strings.HasPrefix(canonPath, "./") || strings.HasPrefix(canonPath, "../") {
		relPath := filepath.Join(ml.basePath, canonPath) + ".ail"
		searchTrace = append(searchTrace, "relative: "+relPath)
		fullPath = relPath
	} else if strings.HasPrefix(canonPath, "std/") {
		// Stdlib path - resolve from AILANG_STDLIB_PATH or default to "stdlib/"
		stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
		if stdlibPath == "" {
			stdlibPath = "stdlib"
		}
		stdPath := filepath.Join(stdlibPath, canonPath) + ".ail"
		searchTrace = append(searchTrace, "stdlib: "+stdPath)
		fullPath = stdPath
	} else if strings.HasSuffix(canonPath, ".ail") {
		// Absolute path
		searchTrace = append(searchTrace, "absolute: "+canonPath)
		fullPath = canonPath
	} else {
		// Project-relative - join with basePath for absolute resolution
		projPath := filepath.Join(ml.basePath, canonPath) + ".ail"
		searchTrace = append(searchTrace, "project: "+projPath)
		fullPath = projPath
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// Collect similar module suggestions
		similar := ml.suggestSimilar(path)
		report := newLDR001(canonicalID, searchTrace, similar, nil)
		return nil, errors.WrapReport(report)
	}

	// Parse file
	l := lexer.New(string(content), fullPath)
	p := parser.New(l)
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s: %v", path, p.Errors())
	}

	// Extract imports from the file
	imports := ml.extractImports(file)
	// DEBUG: Show imports
	if len(imports) > 0 {
		fmt.Printf("DEBUG loader: module %s imports %v\n", path, imports)
	}

	// Build export table
	exports := ml.buildExports(file)

	// Build types and constructors tables
	types, constructors := ml.buildTypes(file)

	// Note: Core elaboration is done by the runtime to avoid import cycles
	// (elaborate imports loader, so loader can't import elaborate)

	// Cache and return with canonical ID
	canonicalID = CanonicalModuleID(path)
	loaded := &LoadedModule{
		Path:         canonicalID, // Store canonical form
		File:         file,
		Imports:      imports,
		Exports:      exports,
		Types:        types,
		Constructors: constructors,
		Core:         nil, // Will be populated by runtime
	}
	ml.cache[canonicalID] = loaded

	return loaded, nil
}

// resolvePath resolves a module path to a file path
func (ml *ModuleLoader) resolvePath(path string) string {
	// If path already ends with .ail, use it as-is (absolute)
	if strings.HasSuffix(path, ".ail") {
		return path
	}

	// Handle explicit relative imports (starts with ./ or ../)
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return filepath.Join(ml.basePath, path) + ".ail"
	}

	// Handle stdlib imports (always relative to stdlib root)
	if strings.HasPrefix(path, "std/") {
		// Resolve from AILANG_STDLIB_PATH env or default to "stdlib/"
		stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
		if stdlibPath == "" {
			stdlibPath = "stdlib"
		}
		return filepath.Join(stdlibPath, path) + ".ail"
	}

	// Default: treat as project-relative (join with basePath)
	// Example: "examples/v3_3/math/gcd" → "/abs/path/examples/v3_3/math/gcd.ail"
	return filepath.Join(ml.basePath, path) + ".ail"
}

// CanonicalModuleID returns the canonical module ID for a path
// Canonical form: repo-relative, forward slashes, no .ail extension
func CanonicalModuleID(p string) string {
	// Clean the path first
	p = filepath.Clean(p)

	// Remove .ail extension if present
	p = strings.TrimSuffix(p, ".ail")

	// Normalize to forward slashes (cross-platform)
	p = strings.ReplaceAll(p, "\\", "/")

	// Remove leading ./ if present
	p = strings.TrimPrefix(p, "./")

	// Remove leading / for absolute paths (make repo-relative)
	p = strings.TrimPrefix(p, "/")

	return p
}

// buildExports builds the export table for a module
func (ml *ModuleLoader) buildExports(file *ast.File) map[string]*ast.FuncDecl {
	exports := make(map[string]*ast.FuncDecl)

	// For now, just export all functions (since we don't have export declarations yet)
	// TODO: Once we have export declarations, use those
	for _, fn := range file.Funcs {
		// Export all public (non-underscore) functions
		if !strings.HasPrefix(fn.Name, "_") {
			exports[fn.Name] = fn
		}
	}

	return exports
}

// buildTypes extracts type declarations and constructors from a module
func (ml *ModuleLoader) buildTypes(file *ast.File) (map[string]*ast.TypeDecl, map[string]string) {
	types := make(map[string]*ast.TypeDecl)
	constructors := make(map[string]string) // ctor name -> type name

	// Check both Decls and Statements for type declarations
	allDecls := append(file.Decls, file.Statements...)
	for _, decl := range allDecls {
		if typeDecl, ok := decl.(*ast.TypeDecl); ok {
			// Only export if marked as exported
			if typeDecl.Exported {
				types[typeDecl.Name] = typeDecl

				// Extract constructors from algebraic types
				if algType, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
					for _, ctor := range algType.Constructors {
						constructors[ctor.Name] = typeDecl.Name
					}
				}
			}
		}
	}

	return types, constructors
}

// GetExport retrieves an exported symbol from a module
// Returns (nil, nil) if the symbol is a type or constructor (not a function)
func (ml *ModuleLoader) GetExport(modulePath, symbol string) (*ast.FuncDecl, error) {
	module, err := ml.Load(modulePath)
	if err != nil {
		// If Load() returned a LoaderReport, pass it through
		return nil, err
	}

	// Check if it's a function export
	decl, ok := module.Exports[symbol]
	if ok {
		return decl, nil
	}

	// Check if it's a type name - return nil (not an error, just not a function)
	if _, isType := module.Types[symbol]; isType {
		return nil, nil
	}

	// Check if it's a constructor - return nil (not an error, just not a function)
	if _, isCtor := module.Constructors[symbol]; isCtor {
		return nil, nil
	}

	// Symbol not found at all - build error report
	var available []string
	for name := range module.Exports {
		available = append(available, name)
	}
	for name := range module.Types {
		available = append(available, name+" (type)")
	}
	for name := range module.Constructors {
		available = append(available, name+" (ctor)")
	}
	sort.Strings(available)

	// Return structured error report (wrapped)
	errReport := newIMP010Loader(symbol, modulePath, available, nil)
	return nil, errors.WrapReport(errReport)
}

// newIMP010Loader creates an IMP010 error report (symbol not exported)
// Similar to link.newIMP010 but for the loader context
func newIMP010Loader(symbol, modID string, available []string, span *ast.Span) *errors.Report {
	sortedAvailable := make([]string, len(available))
	copy(sortedAvailable, available)
	sort.Strings(sortedAvailable)

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP010",
		Phase:   "loader",
		Message: fmt.Sprintf("symbol '%s' not exported by '%s'", symbol, modID),
		Span:    span,
		Data: map[string]any{
			"available_exports": sortedAvailable,
			"module_id":         modID,
			"symbol":            symbol,
		},
		Fix: &errors.Fix{
			Suggestion: fmt.Sprintf("Check exports in %s. Available: %s",
				modID, strings.Join(sortedAvailable[:min(3, len(sortedAvailable))], ", ")),
			Confidence: 0.85,
		},
	}
}

// LoadAll loads a module and all its transitive dependencies
func (ml *ModuleLoader) LoadAll(roots []string) (map[string]*LoadedModule, error) {
	modules := make(map[string]*LoadedModule)
	visited := make(map[string]bool)
	var searchTrace []string

	// DFS to load all dependencies
	var loadDeps func(path string) error
	loadDeps = func(path string) error {
		// Skip if already visited
		if visited[path] {
			return nil
		}
		visited[path] = true

		// Track search attempt
		searchTrace = append(searchTrace, fmt.Sprintf("Loading module: %s", path))

		// Load the module
		module, err := ml.Load(path)
		if err != nil {
			// Include search trace in error
			return fmt.Errorf("failed to load %s (search trace: %v): %w",
				path, searchTrace, err)
		}
		// Store with canonical ID (module.Path), not input path
		modules[module.Path] = module

		// Load its dependencies
		for _, dep := range module.Imports {
			searchTrace = append(searchTrace, fmt.Sprintf("  -> dependency: %s", dep))
			if err := loadDeps(dep); err != nil {
				return err
			}
		}

		return nil
	}

	// Load all root modules and their dependencies
	for _, root := range roots {
		if err := loadDeps(root); err != nil {
			return nil, err
		}
	}

	return modules, nil
}

// extractImports extracts module paths from import declarations
func (ml *ModuleLoader) extractImports(file *ast.File) []string {
	var imports []string
	for _, imp := range file.Imports {
		imports = append(imports, imp.Path)
	}
	return imports
}

// LoadInterface loads just the interface of a module (for the linker)
func (ml *ModuleLoader) LoadInterface(modulePath string) (*iface.Iface, error) {
	module, err := ml.Load(modulePath)
	if err != nil {
		return nil, err
	}

	// If the interface is already built, return it
	if module.Iface != nil {
		return module.Iface, nil
	}

	// Otherwise, we need to build it (requires type checking)
	// This will be done by the pipeline
	return nil, fmt.Errorf("interface not yet built for module %s", modulePath)
}

// EvaluateExport evaluates a specific export from a module
func (ml *ModuleLoader) EvaluateExport(ref core.GlobalRef) (eval.Value, error) {
	_, err := ml.Load(ref.Module)
	if err != nil {
		return nil, err
	}

	// This requires the module to be compiled and evaluated
	// The pipeline will handle this
	return nil, fmt.Errorf("export evaluation not yet implemented for %s.%s", ref.Module, ref.Name)
}

// NormalizeContent normalizes file content (CRLF, BOM, etc.)
func (ml *ModuleLoader) NormalizeContent(content []byte) []byte {
	// Remove BOM if present
	if bytes.HasPrefix(content, []byte{0xEF, 0xBB, 0xBF}) {
		content = content[3:]
	}

	// Normalize line endings (CRLF -> LF)
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	content = bytes.ReplaceAll(content, []byte("\r"), []byte("\n"))

	return content
}

// CanonicalPath returns the canonical path for a module
func (ml *ModuleLoader) CanonicalPath(path string) (string, error) {
	// Resolve to absolute path
	fullPath := ml.resolvePath(path)

	// Get canonical path (resolves symlinks, etc.)
	canonical, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// If file doesn't exist yet, just clean the path
		canonical = filepath.Clean(fullPath)
	}

	// Convert back to module path format
	// Remove .ail extension and base path
	canonical = strings.TrimSuffix(canonical, ".ail")
	if strings.HasPrefix(canonical, ml.basePath) {
		canonical = strings.TrimPrefix(canonical, ml.basePath)
		canonical = strings.TrimPrefix(canonical, "/")
	}

	return canonical, nil
}

// newLDR001 creates an error report for module not found
// Data fields: module_id, search_trace[], similar[] (optional)
func newLDR001(modID string, searchTrace, similar []string, span *ast.Span) *errors.Report {
	// Ensure deterministic ordering
	sortedTrace := make([]string, len(searchTrace))
	copy(sortedTrace, searchTrace)
	sort.Strings(sortedTrace)

	sortedSimilar := make([]string, len(similar))
	copy(sortedSimilar, similar)
	sort.Strings(sortedSimilar)

	data := map[string]any{
		"module_id":    modID,
		"search_trace": sortedTrace,
	}

	// Only add similar if non-empty
	if len(sortedSimilar) > 0 {
		data["similar"] = sortedSimilar
	}

	suggestion := fmt.Sprintf("Check module path '%s' exists", modID)
	if len(sortedSimilar) > 0 {
		suggestion = fmt.Sprintf("Module not found. Similar modules: %s", strings.Join(sortedSimilar[:min(3, len(sortedSimilar))], ", "))
	}

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "LDR001",
		Phase:   "loader",
		Message: fmt.Sprintf("module not found: %s", modID),
		Span:    span,
		Data:    data,
		Fix: &errors.Fix{
			Suggestion: suggestion,
			Confidence: 0.85,
		},
	}
}

// suggestSimilar finds similar module names based on simple heuristic
func (ml *ModuleLoader) suggestSimilar(want string) []string {
	// Collect all cached module paths
	var all []string
	for cached := range ml.cache {
		all = append(all, cached)
	}

	// Find modules containing any part of the wanted path
	var hits []string
	base := filepath.Base(want)

	for _, s := range all {
		// Check if the cached path contains the base name
		if strings.Contains(s, base) {
			hits = append(hits, s)
			continue
		}
		// Check if any path component matches
		wantParts := strings.Split(want, "/")
		sParts := strings.Split(s, "/")
		for _, wp := range wantParts {
			for _, sp := range sParts {
				if wp == sp && wp != "" {
					hits = append(hits, s)
					break
				}
			}
		}
	}

	// Remove duplicates and sort
	seen := make(map[string]bool)
	var unique []string
	for _, h := range hits {
		if !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}

	sort.Strings(unique)

	// Return top 5
	if len(unique) > 5 {
		return unique[:5]
	}
	return unique
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
