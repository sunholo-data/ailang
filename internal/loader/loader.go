package loader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/std"
)

// ModuleLoader loads and caches modules
type ModuleLoader struct {
	cache              map[string]*LoadedModule
	basePath           string              // Base directory for relative imports
	strictSyntaxMode   bool                // When true, syntactic sugar is not allowed
	stdlibResolver     *StdlibResolver     // Stdlib path resolver (initialized lazily)
	pkgLoader          PackageResolver     // Optional package loader for pkg/ imports
	modulePrefixMap    map[string][]string // module_prefix → package names (e.g., "docparse" → ["sunholo/ailang_parse", "sunholo/docparse"])
	currentPackageName string              // <vendor>/<name> of the package being compiled, when known. Enables bare-canonical self-imports (e.g., "import sunholo/linkedin/types" from within sunholo/linkedin).
}

// PackageResolver resolves package imports to source file paths.
// Implemented by pkg.PackageLoader.
type PackageResolver interface {
	ResolveImport(importPath string) (string, error)
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
	CoreTI       interface{}              // Type info for Core expressions (types.CoreTypeInfo, interface{} to avoid import cycle)
}

// NewModuleLoader creates a new module loader
func NewModuleLoader(basePath string) *ModuleLoader {
	return &ModuleLoader{
		cache:            make(map[string]*LoadedModule),
		basePath:         basePath,
		strictSyntaxMode: false, // Default: allow syntactic sugar
	}
}

// SetStrictSyntaxMode enables or disables strict syntax mode
func (ml *ModuleLoader) SetStrictSyntaxMode(strict bool) {
	ml.strictSyntaxMode = strict
}

// SetPackageResolver sets the resolver for pkg/ imports.
func (ml *ModuleLoader) SetPackageResolver(resolver PackageResolver) {
	ml.pkgLoader = resolver
}

// SetModulePrefixMap sets the module_prefix → package name mapping.
// Input is pkgName → prefix (as built by the pipeline); this method inverts it
// to prefix → []pkgName for fast lookup during bare import resolution.
// Multiple packages may share a prefix (e.g., "sunholo/docparse" and
// "sunholo/ailang_parse" both use module_prefix="docparse"). All are stored
// so the resolver can try each until one succeeds.
func (ml *ModuleLoader) SetModulePrefixMap(prefixMap map[string]string) {
	ml.modulePrefixMap = make(map[string][]string)
	for pkgName, prefix := range prefixMap {
		ml.modulePrefixMap[prefix] = append(ml.modulePrefixMap[prefix], pkgName)
	}
}

// ConfigureStdlibResolver configures the stdlib resolver with CLI flags
// Call this before loading any stdlib modules
func (ml *ModuleLoader) ConfigureStdlibResolver(cliPath string, traceEnabled, strictMode bool) {
	ml.stdlibResolver = NewStdlibResolver(cliPath, traceEnabled, strictMode)
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

// DeleteCached removes a module from the loader cache, forcing re-load on next access.
// This is used by hot reload to invalidate stale modules.
func (ml *ModuleLoader) DeleteCached(modulePath string) {
	canonicalID := CanonicalModuleID(modulePath)
	delete(ml.cache, canonicalID)
}

// canonicalizeModulePath normalizes import paths
//
// Returns the canonical path.
// Modern pattern: "std/io" → canonical: "std/io"
func canonicalizeModulePath(path string) string {
	// Strip leading "./" or ".\" for cross-platform safety
	path = strings.TrimPrefix(strings.TrimPrefix(path, "./"), ".\\")
	return path
}

// Load loads a module by path
func (ml *ModuleLoader) Load(path string) (*LoadedModule, error) {
	// Canonicalize the import path
	canonPath := canonicalizeModulePath(path)

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
	var content []byte // Pre-loaded content (from embedded stdlib fallback)

	// Try relative path first
	if strings.HasPrefix(canonPath, "./") || strings.HasPrefix(canonPath, "../") {
		relPath := filepath.Join(ml.basePath, canonPath) + ".ail"
		searchTrace = append(searchTrace, "relative: "+relPath)
		fullPath = relPath
	} else if strings.HasPrefix(canonPath, "pkg/") {
		// External package import — resolve via PackageLoader
		if ml.pkgLoader == nil {
			return nil, fmt.Errorf("package import %q requires ailang.toml and ailang.lock; run 'ailang init package' and 'ailang lock'", canonPath)
		}
		// Strip pkg/ prefix for the package resolver
		pkgImportPath := strings.TrimPrefix(canonPath, "pkg/")
		resolvedPath, err := ml.pkgLoader.ResolveImport(pkgImportPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve package import %q: %w", canonPath, err)
		}
		fullPath = resolvedPath
		searchTrace = append(searchTrace, "package: "+resolvedPath)
	} else if strings.HasPrefix(canonPath, "std/") {
		// Standard library path - use StdlibResolver
		// Initialize resolver lazily if not configured
		if ml.stdlibResolver == nil {
			ml.stdlibResolver = NewStdlibResolver("", false, false)
		}

		resolvedPath, err := ml.stdlibResolver.ResolveStdlib(canonPath)
		if err != nil {
			// Filesystem resolution failed — try embedded stdlib fallback.
			// The stdlib .ail files are compiled into the binary via std/embed.go,
			// so the binary is self-contained even without a filesystem std/ directory.
			moduleName := strings.TrimPrefix(canonPath, "std/")
			embFile := moduleName + ".ail"
			if embContent, embErr := std.FS.ReadFile(embFile); embErr == nil {
				content = embContent
				fullPath = "<embedded>/std/" + embFile
				searchTrace = append(searchTrace, "embedded: "+fullPath)
				if ml.stdlibResolver.traceEnabled {
					fmt.Fprintf(os.Stderr, "[trace-loader] Filesystem stdlib not found, using embedded fallback: %s\n", fullPath)
				}
			} else {
				// Both filesystem and embedded failed — return original error
				return nil, err
			}
		} else {
			fullPath = resolvedPath
			searchTrace = append(searchTrace, "std: "+resolvedPath)
		}
	} else if strings.HasSuffix(canonPath, ".ail") {
		// Absolute path
		searchTrace = append(searchTrace, "absolute: "+canonPath)
		fullPath = canonPath
	} else if ml.pkgLoader != nil && ml.currentPackageName != "" && pathMatchesPackagePrefix(canonPath, ml.currentPackageName) {
		// Bare canonical self-import: `import sunholo/linkedin/types` from
		// within the `sunholo/linkedin` package. The natural author form —
		// module declarations use this same path. Route through the package
		// resolver's self-reference path so it resolves to the sibling file.
		resolvedPath, err := ml.pkgLoader.ResolveImport(canonPath)
		if err != nil {
			searchTrace = append(searchTrace, "self("+ml.currentPackageName+"): failed: "+err.Error())
			// Don't fall through — if the prefix matches but resolution fails,
			// the author's intent was clear and the surfaced error (e.g. "not
			// in exports list") is more useful than a project-relative miss.
			report := newLDR001(canonicalID, searchTrace, ml.suggestSimilar(path), nil)
			return nil, errors.WrapReport(report)
		}
		fullPath = resolvedPath
		searchTrace = append(searchTrace, "self("+ml.currentPackageName+"): "+resolvedPath)
	} else if ml.pkgLoader != nil && ml.modulePrefixMap != nil {
		// Try module_prefix resolution: bare imports like "docparse/types/document"
		// may be intra-package imports where "docparse" is a module_prefix.
		// Remap to canonical pkg/ path and resolve via the package loader.
		// Multiple packages may share a prefix, so try each until one resolves.
		resolved := false
		firstSeg := canonPath
		if idx := strings.Index(canonPath, "/"); idx >= 0 {
			firstSeg = canonPath[:idx]
		}
		if pkgNames, ok := ml.modulePrefixMap[firstSeg]; ok {
			for _, pkgName := range pkgNames {
				// Remap: "docparse/types/document" → "sunholo/ailang_parse/types/document"
				canonImport := pkgName + strings.TrimPrefix(canonPath, firstSeg)
				resolvedPath, err := ml.pkgLoader.ResolveImport(canonImport)
				if err == nil {
					fullPath = resolvedPath
					searchTrace = append(searchTrace, "prefix("+firstSeg+"→"+pkgName+"): "+resolvedPath)
					resolved = true
					break
				}
				searchTrace = append(searchTrace, "prefix("+firstSeg+"→"+pkgName+"): failed: "+err.Error())
			}
		}
		if !resolved {
			// Fall through to project-relative
			projPath := filepath.Join(ml.basePath, canonPath) + ".ail"
			searchTrace = append(searchTrace, "project: "+projPath)
			fullPath = projPath
		}
	} else {
		// Project-relative - join with basePath for absolute resolution
		projPath := filepath.Join(ml.basePath, canonPath) + ".ail"
		searchTrace = append(searchTrace, "project: "+projPath)
		fullPath = projPath
	}

	// Read file (skip if content already loaded from embedded stdlib)
	if content == nil {
		var err error
		content, err = os.ReadFile(fullPath)
		if err != nil {
			// Collect similar module suggestions
			similar := ml.suggestSimilar(path)
			report := newLDR001(canonicalID, searchTrace, similar, nil)
			return nil, errors.WrapReport(report)
		}
	}

	// Parse file
	l := lexer.New(string(content), fullPath)
	p := parser.New(l)
	p.SetStrictSyntaxMode(ml.strictSyntaxMode)
	file := p.ParseFile()
	// Record the resolved source path on the AST so downstream consumers
	// (e.g., apiserver route filtering, MOD011 collision detection) can
	// distinguish "same file loaded under two canonical IDs" from "two
	// different files claiming the same module header". The parser itself
	// doesn't set this field.
	if file != nil {
		file.Path = fullPath
	}
	if len(p.Errors()) > 0 {
		// Format each error individually to preserve custom .Error() methods
		// (e.g., ParserError with suggestions)
		var errorMsgs []string
		for _, err := range p.Errors() {
			errorMsgs = append(errorMsgs, err.Error())
		}
		return nil, fmt.Errorf("parse errors in %s:\n%s", path, strings.Join(errorMsgs, "\n\n"))
	}

	// Extract imports from the file
	imports := ml.extractImports(file)
	// DEBUG: Show imports (commented out - pollutes output for benchmarks)
	//if len(imports) > 0 {
	//	fmt.Printf("DEBUG loader: module %s imports %v\n", path, imports)
	//}

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

	// Handle standard library imports (always relative to std root)
	if strings.HasPrefix(path, "std/") {
		// Resolve from AILANG_STDLIB_PATH env or default to current directory
		stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
		if stdlibPath == "" {
			stdlibPath = "." // std/ is at repository root
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

	// The CLI renders only "CODE: Message", so the actionable hint must live in Message to reach
	// an agent reading `ailang check` output. Same hint as the linker path. M-AGENT-STUCK-FIXES M2.
	hint := importhint.IMP010(symbol, modID)
	suggestion := fmt.Sprintf("Check exports in %s. Available: %s",
		modID, strings.Join(sortedAvailable[:min(3, len(sortedAvailable))], ", "))
	confidence := 0.85
	if hint != "" {
		suggestion = strings.TrimPrefix(hint, " — ")
		confidence = 0.95
	}

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP010",
		Phase:   "loader",
		Message: fmt.Sprintf("symbol '%s' not exported by '%s'%s", symbol, modID, hint),
		Span:    span,
		Data: map[string]any{
			"available_exports": sortedAvailable,
			"module_id":         modID,
			"symbol":            symbol,
		},
		Fix: &errors.Fix{
			Suggestion: suggestion,
			Confidence: confidence,
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
			// Search trace is internal debugging info — only show with DEBUG_LOADER=1
			if os.Getenv("DEBUG_LOADER") == "1" {
				fmt.Fprintf(os.Stderr, "[debug-loader] search trace: %v\n", searchTrace)
			}
			return fmt.Errorf("failed to load %s: %w", path, err)
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

// extractImports extracts module paths from import declarations.
// Relative imports (./...) are normalized to canonical pkg/ paths using the
// current module's declaration. The AST's imp.Path is updated in-place so all
// downstream pipeline stages see the canonical identity (not the relative syntax).
func (ml *ModuleLoader) extractImports(file *ast.File) []string {
	var imports []string
	for _, imp := range file.Imports {
		// Normalize relative imports: ./plan → pkg/vendor/name/plan
		// Update imp.Path in-place so pipeline import resolution uses canonical paths
		if imp.IsRelative && file.Module != nil {
			canonical := NormalizeRelativeImport(file.Module.Path, imp.RelativePath)
			if canonical != "" {
				imp.Path = "pkg/" + canonical
				// Also mark as package import so the loader routes correctly
				imp.IsPackage = true
				parts := strings.SplitN(canonical, "/", 3)
				if len(parts) >= 2 {
					imp.PackageName = parts[0] + "/" + parts[1]
				}
			}
		}
		imports = append(imports, imp.Path)
	}
	return imports
}

// NormalizeRelativeImport resolves a relative import against the current module's
// canonical path. This is module-space resolution, not filesystem resolution.
//
// If current module is "sunholo/billing_entitlements/entitlement" and relative is "plan",
// the result is "sunholo/billing_entitlements/plan".
//
// If relative contains "/" (e.g., "sub/bar"), it resolves to the same prefix:
// "sunholo/billing_entitlements/sub/bar".
func NormalizeRelativeImport(currentModulePath, relativePath string) string {
	// Strip last segment (current module name) to get the prefix
	lastSlash := strings.LastIndex(currentModulePath, "/")
	if lastSlash == -1 {
		// Single-segment module path — relative resolves to same level
		return relativePath
	}
	prefix := currentModulePath[:lastSlash]
	return prefix + "/" + relativePath
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

// IsTempPath returns true if the given path is in a temporary directory.
// This is used for relaxed module matching - files in temp directories
// are allowed to have mismatched module declarations.
//
// Detection is conservative: if uncertain, returns false (doesn't auto-relax).
//
// Patterns detected:
//   - os.TempDir() prefix (cross-platform)
//   - /tmp/ prefix (Unix)
//   - /var/folders/ prefix (macOS)
//   - Windows %TEMP% prefix
//   - Canonical paths starting with "tmp/" (after CanonicalModuleID strips leading /)
func IsTempPath(path string) bool {
	// First check for canonical paths that were originally in /tmp/
	// CanonicalModuleID strips leading "/" so /tmp/foo becomes tmp/foo
	if strings.HasPrefix(path, "tmp/") || path == "tmp" {
		return true
	}

	// Check for /var/folders/ canonical paths (macOS)
	if strings.HasPrefix(path, "var/folders/") {
		return true
	}

	// Normalize path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't resolve the path, be conservative
		return false
	}

	// Check os.TempDir() first (cross-platform)
	tempDir := os.TempDir()
	if strings.HasPrefix(absPath, tempDir) {
		return true
	}

	// Platform-specific patterns
	if runtime.GOOS == "windows" {
		// Windows: check %TEMP% and %TMP% environment variables
		if temp := os.Getenv("TEMP"); temp != "" {
			if strings.HasPrefix(absPath, temp) {
				return true
			}
		}
		if tmp := os.Getenv("TMP"); tmp != "" {
			if strings.HasPrefix(absPath, tmp) {
				return true
			}
		}
	} else {
		// Unix-like systems
		// Check /tmp/ prefix
		if strings.HasPrefix(absPath, "/tmp/") || absPath == "/tmp" {
			return true
		}

		// Check /var/folders/ prefix (macOS temp directories)
		if strings.HasPrefix(absPath, "/var/folders/") {
			return true
		}
	}

	return false
}
