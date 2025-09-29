package loader

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// ModuleLoader loads and caches modules
type ModuleLoader struct {
	cache    map[string]*LoadedModule
	basePath string // Base directory for relative imports
}

// LoadedModule represents a loaded and parsed module
type LoadedModule struct {
	Path     string
	File     *ast.File
	Imports  []string                 // Module paths this module imports
	Exports  map[string]*ast.FuncDecl // Export table (for now, just functions)
	Core     *core.Program            // Core representation (after elaboration)
	Iface    *iface.Iface             // Module interface (after type checking)
}

// NewModuleLoader creates a new module loader
func NewModuleLoader(basePath string) *ModuleLoader {
	return &ModuleLoader{
		cache:    make(map[string]*LoadedModule),
		basePath: basePath,
	}
}

// Load loads a module by path
func (ml *ModuleLoader) Load(path string) (*LoadedModule, error) {
	// Canonicalize the module ID
	canonicalID := CanonicalModuleID(path)
	
	// Check cache with canonical ID
	if loaded, ok := ml.cache[canonicalID]; ok {
		return loaded, nil
	}

	// Resolve path
	fullPath := ml.resolvePath(path)
	
	// Read file
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read module %s: %w", path, err)
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

	// Cache and return with canonical ID
	canonicalID = CanonicalModuleID(path)
	loaded := &LoadedModule{
		Path:    canonicalID,  // Store canonical form
		File:    file,
		Imports: imports,
		Exports: exports,
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
		// TODO: Resolve from AILANG_STDLIB_PATH env or default location
		return filepath.Join(ml.basePath, path) + ".ail"
	}

	// Default: treat as repo-relative (don't join with basePath!)
	// Example: "examples/v3_3/math/gcd" → "examples/v3_3/math/gcd.ail"
	return path + ".ail"
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

// GetExport retrieves an exported symbol from a module
func (ml *ModuleLoader) GetExport(modulePath, symbol string) (*ast.FuncDecl, error) {
	module, err := ml.Load(modulePath)
	if err != nil {
		return nil, err
	}
	
	decl, ok := module.Exports[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol %s not exported from %s", symbol, modulePath)
	}
	
	return decl, nil
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
	if strings.HasSuffix(canonical, ".ail") {
		canonical = canonical[:len(canonical)-4]
	}
	if strings.HasPrefix(canonical, ml.basePath) {
		canonical = strings.TrimPrefix(canonical, ml.basePath)
		canonical = strings.TrimPrefix(canonical, "/")
	}
	
	return canonical, nil
}