// Package module implements module loading and dependency resolution for AILANG.
package module

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// Module represents a parsed AILANG module
type Module struct {
	// Identity is the canonical module path (e.g., "std/list", "data/tree")
	Identity string

	// FilePath is the absolute path to the module file
	FilePath string

	// AST is the parsed module AST
	AST *ast.Module

	// Program is the full parsed program including the module
	Program *ast.Program

	// Dependencies are the modules this module imports
	Dependencies []string

	// Exports are the symbols exported by this module
	Exports map[string]ast.Node
}

// Loader handles module loading and dependency resolution
type Loader struct {
	// cache stores loaded modules by their identity
	cache map[string]*Module
	mu    sync.RWMutex

	// searchPaths are directories to search for modules
	searchPaths []string

	// stdlibPath is the path to the standard library
	stdlibPath string

	// currentFile is the file currently being loaded (for relative imports)
	currentFile string

	// loadStack tracks the current load chain for cycle detection
	loadStack []string
}

// NewLoader creates a new module loader
func NewLoader() *Loader {
	return &Loader{
		cache:       make(map[string]*Module),
		searchPaths: getDefaultSearchPaths(),
		stdlibPath:  getStdlibPath(),
		loadStack:   []string{},
	}
}

// getDefaultSearchPaths returns the default module search paths
func getDefaultSearchPaths() []string {
	paths := []string{
		".", // Current directory
	}

	// Add AILANG_PATH if set
	if ailangPath := os.Getenv("AILANG_PATH"); ailangPath != "" {
		paths = append(paths, strings.Split(ailangPath, string(os.PathListSeparator))...)
	}

	// Add home directory modules
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".ailang", "modules"))
	}

	return paths
}

// getStdlibPath returns the path to the standard library
func getStdlibPath() string {
	// Check environment variable
	if stdlib := os.Getenv("AILANG_STDLIB"); stdlib != "" {
		return stdlib
	}

	// Check relative to executable
	if exe, err := os.Executable(); err == nil {
		stdlib := filepath.Join(filepath.Dir(exe), "..", "stdlib")
		if info, err := os.Stat(stdlib); err == nil && info.IsDir() {
			return stdlib
		}
	}

	// Fallback to current directory
	return filepath.Join(".", "stdlib")
}

// Load loads a module by its import path
func (l *Loader) Load(importPath string) (*Module, error) {
	// Normalize the import path
	identity := l.normalizeModulePath(importPath)

	// Check cache
	if mod := l.getCached(identity); mod != nil {
		return mod, nil
	}

	// Check for circular dependency
	if err := l.checkCycle(identity); err != nil {
		return nil, err
	}

	// Add to load stack
	l.pushStack(identity)
	defer l.popStack()

	// Resolve the file path
	filePath, err := l.resolvePath(importPath)
	if err != nil {
		return nil, l.moduleNotFoundError(importPath, err)
	}

	// Parse the module file
	mod, err := l.parseModule(identity, filePath)
	if err != nil {
		return nil, err
	}

	// Load dependencies
	if err := l.loadDependencies(mod); err != nil {
		return nil, err
	}

	// Validate module
	if err := l.validateModule(mod); err != nil {
		return nil, err
	}

	// Cache the module
	l.cacheModule(mod)

	return mod, nil
}

// LoadFile loads a module from a specific file path
func (l *Loader) LoadFile(filePath string) (*Module, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Derive module identity from file path
	identity := l.deriveModuleIdentity(absPath)

	// Set current file for relative imports
	oldFile := l.currentFile
	l.currentFile = absPath
	defer func() { l.currentFile = oldFile }()

	// Check cache
	if mod := l.getCached(identity); mod != nil {
		return mod, nil
	}

	// Parse and load
	mod, err := l.parseModule(identity, absPath)
	if err != nil {
		return nil, err
	}

	// Load dependencies
	if err := l.loadDependencies(mod); err != nil {
		return nil, err
	}

	// Validate
	if err := l.validateModule(mod); err != nil {
		return nil, err
	}

	// Cache
	l.cacheModule(mod)

	return mod, nil
}

// parseModule parses a module file
func (l *Loader) parseModule(identity, filePath string) (*Module, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read module file: %w", err)
	}

	// Parse the file
	lex := lexer.New(string(content), filePath)
	p := parser.New(lex)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, l.parseError(filePath, p.Errors())
	}

	// Extract module declaration
	if program.Module == nil {
		// If no module declaration, create a default one
		program.Module = &ast.Module{
			Name:    identity,
			Exports: []string{},
			Imports: []*ast.Import{},
		}
	}

	// Validate module name matches expected identity
	// Skip validation if it's a default module (no explicit module declaration)
	if !l.isStdlib(identity) && program.Module.Name != identity {
		// If the module name was auto-generated (e.g. "Main"), use the expected identity
		if program.Module.Name == "Main" {
			program.Module.Name = l.expectedModuleName(filePath)
		} else {
			expectedName := l.expectedModuleName(filePath)
			if program.Module.Name != expectedName {
				return nil, l.moduleNameMismatchError(program.Module.Name, expectedName, filePath)
			}
		}
	}

	// Create module
	mod := &Module{
		Identity:     identity,
		FilePath:     filePath,
		AST:          program.Module,
		Program:      program,
		Dependencies: l.extractDependencies(program.Module),
		Exports:      l.extractExports(program),
	}

	return mod, nil
}

// resolvePath resolves an import path to a file path
func (l *Loader) resolvePath(importPath string) (string, error) {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		if l.currentFile == "" {
			return "", fmt.Errorf("relative import '%s' with no current file", importPath)
		}
		dir := filepath.Dir(l.currentFile)
		path := filepath.Join(dir, importPath)
		if !strings.HasSuffix(path, ".ail") {
			path += ".ail"
		}
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
		return "", fmt.Errorf("module not found: %s", path)
	}

	// Handle stdlib imports
	if strings.HasPrefix(importPath, "std/") {
		path := filepath.Join(l.stdlibPath, strings.TrimPrefix(importPath, "std/"))
		if !strings.HasSuffix(path, ".ail") {
			path += ".ail"
		}
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
		return "", fmt.Errorf("stdlib module not found: %s", importPath)
	}

	// Search in search paths
	for _, searchPath := range l.searchPaths {
		path := filepath.Join(searchPath, importPath)
		if !strings.HasSuffix(path, ".ail") {
			path += ".ail"
		}
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
	}

	return "", fmt.Errorf("module not found in search paths: %s", importPath)
}

// loadDependencies loads all dependencies of a module
func (l *Loader) loadDependencies(mod *Module) error {
	for _, dep := range mod.Dependencies {
		if _, err := l.Load(dep); err != nil {
			return fmt.Errorf("failed to load dependency '%s': %w", dep, err)
		}
	}
	return nil
}

// validateModule validates a module for consistency
func (l *Loader) validateModule(mod *Module) error {
	// Check for duplicate exports
	seen := make(map[string]bool)
	for name := range mod.Exports {
		if seen[name] {
			return l.duplicateExportError(name, mod.Identity)
		}
		seen[name] = true
	}

	// Validate imports reference actual exports
	for _, imp := range mod.AST.Imports {
		depMod, err := l.Load(imp.Path)
		if err != nil {
			return err
		}

		// Check selective imports
		for _, item := range imp.Symbols {
			if _, ok := depMod.Exports[item]; !ok {
				return l.importNotExportedError(item, imp.Path, mod.Identity)
			}
		}
	}

	return nil
}

// Helper methods

func (l *Loader) getCached(identity string) *Module {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cache[identity]
}

func (l *Loader) cacheModule(mod *Module) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache[mod.Identity] = mod
}

func (l *Loader) checkCycle(identity string) error {
	for i, id := range l.loadStack {
		if id == identity {
			// Found a cycle
			cycle := append(l.loadStack[i:], identity)
			return l.circularDependencyError(cycle)
		}
	}
	return nil
}

func (l *Loader) pushStack(identity string) {
	l.loadStack = append(l.loadStack, identity)
}

func (l *Loader) popStack() {
	if len(l.loadStack) > 0 {
		l.loadStack = l.loadStack[:len(l.loadStack)-1]
	}
}

func (l *Loader) normalizeModulePath(path string) string {
	// Remove .ail extension if present
	path = strings.TrimSuffix(path, ".ail")
	// Normalize separators
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

func (l *Loader) deriveModuleIdentity(filePath string) string {
	// Remove .ail extension
	identity := strings.TrimSuffix(filepath.Base(filePath), ".ail")

	// For files in known directories, include the directory structure
	for _, searchPath := range l.searchPaths {
		if absSearch, err := filepath.Abs(searchPath); err == nil {
			if strings.HasPrefix(filePath, absSearch) {
				rel, _ := filepath.Rel(absSearch, filePath)
				identity = strings.TrimSuffix(rel, ".ail")
				identity = strings.ReplaceAll(identity, string(filepath.Separator), "/")
				break
			}
		}
	}

	return identity
}

func (l *Loader) expectedModuleName(filePath string) string {
	// The module name should match the relative path from the project root
	base := strings.TrimSuffix(filepath.Base(filePath), ".ail")
	return base
}

func (l *Loader) isStdlib(identity string) bool {
	return strings.HasPrefix(identity, "std/")
}

func (l *Loader) extractDependencies(mod *ast.Module) []string {
	deps := []string{}
	for _, imp := range mod.Imports {
		deps = append(deps, imp.Path)
	}
	return deps
}

func (l *Loader) extractExports(program *ast.Program) map[string]ast.Node {
	exports := make(map[string]ast.Node)

	// If explicit exports, use those
	if len(program.Module.Exports) > 0 {
		for _, name := range program.Module.Exports {
			// Find the declaration in module Decls
			for _, decl := range program.Module.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name == name {
						exports[name] = d
					}
				case *ast.Let:
					if d.Name == name {
						exports[name] = d
					}
				}
			}
		}
	} else {
		// Otherwise, export all top-level declarations
		for _, decl := range program.Module.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				exports[d.Name] = d
			case *ast.Let:
				exports[d.Name] = d
			}
		}
	}

	return exports
}

// Error constructors

func (l *Loader) moduleNotFoundError(path string, err error) error {
	trace := l.buildResolutionTrace()
	return &ModuleError{
		Code:    errors.LDR001,
		Message: fmt.Sprintf("Module not found: %s", path),
		Path:    path,
		Trace:   trace,
		Cause:   err,
	}
}

func (l *Loader) circularDependencyError(cycle []string) error {
	return &ModuleError{
		Code:    errors.LDR002,
		Message: "Circular module dependency detected",
		Cycle:   cycle,
		Trace:   l.buildResolutionTrace(),
	}
}

func (l *Loader) moduleNameMismatchError(actual, expected, path string) error {
	return &ModuleError{
		Code:    errors.MOD001,
		Message: fmt.Sprintf("Module name '%s' doesn't match expected '%s' for file %s", actual, expected, path),
		Path:    path,
	}
}

func (l *Loader) duplicateExportError(name, module string) error {
	return &ModuleError{
		Code:    errors.MOD004,
		Message: fmt.Sprintf("Duplicate export '%s' in module %s", name, module),
		Path:    module,
	}
}

func (l *Loader) importNotExportedError(item, fromModule, inModule string) error {
	return &ModuleError{
		Code:    errors.LDR004,
		Message: fmt.Sprintf("Import '%s' not exported by module %s (imported in %s)", item, fromModule, inModule),
		Path:    inModule,
	}
}

func (l *Loader) parseError(path string, errs []error) error {
	// Convert first parse error to module error
	if len(errs) > 0 {
		return &ModuleError{
			Code:    errors.PAR001,
			Message: fmt.Sprintf("Parse error in %s: %v", path, errs[0]),
			Path:    path,
			Cause:   errs[0],
		}
	}
	return fmt.Errorf("parse error in %s", path)
}

func (l *Loader) buildResolutionTrace() []string {
	trace := []string{}
	for i, id := range l.loadStack {
		indent := strings.Repeat("  ", i)
		if i == 0 {
			trace = append(trace, fmt.Sprintf("Resolving %s", id))
		} else {
			trace = append(trace, fmt.Sprintf("%s-> import %s", indent, id))
		}
	}
	return trace
}

// ModuleError represents a module loading error with structured information
type ModuleError struct {
	Code    string   // Error code (e.g., LDR001)
	Message string   // Human-readable message
	Path    string   // Module path that caused the error
	Cycle   []string // For circular dependencies
	Trace   []string // Resolution trace
	Cause   error    // Underlying error
}

func (e *ModuleError) Error() string {
	return e.Message
}

// GetDependencyGraph returns the full dependency graph
func (l *Loader) GetDependencyGraph() map[string][]string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	graph := make(map[string][]string)
	for id, mod := range l.cache {
		graph[id] = mod.Dependencies
	}
	return graph
}

// TopologicalSort returns modules in dependency order
func (l *Loader) TopologicalSort() ([]string, error) {
	graph := l.GetDependencyGraph()

	// Kahn's algorithm for topological sort
	// We need a reverse graph for proper topological sorting
	// If A depends on B, we want B to come before A
	reverseGraph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all nodes
	for node := range graph {
		reverseGraph[node] = []string{}
		inDegree[node] = 0
	}

	// Build reverse graph and count in-degrees
	for node, deps := range graph {
		for _, dep := range deps {
			// dep is depended on by node
			if _, exists := reverseGraph[dep]; !exists {
				reverseGraph[dep] = []string{}
				inDegree[dep] = 0
			}
			reverseGraph[dep] = append(reverseGraph[dep], node)
		}
		// node has deps.length dependencies
		inDegree[node] = len(deps)
	}

	// Find nodes with no incoming edges
	queue := []string{}
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Process nodes that depend on this one
		for _, dependent := range reverseGraph[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for cycles
	if len(result) != len(graph) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// Writer interface for dumping module information
func (l *Loader) DumpModules(w io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	fmt.Fprintf(w, "Loaded Modules:\n")
	for id, mod := range l.cache {
		fmt.Fprintf(w, "  %s:\n", id)
		fmt.Fprintf(w, "    File: %s\n", mod.FilePath)
		fmt.Fprintf(w, "    Dependencies: %v\n", mod.Dependencies)
		fmt.Fprintf(w, "    Exports: %v\n", l.getExportNames(mod))
	}
}

func (l *Loader) getExportNames(mod *Module) []string {
	names := []string{}
	for name := range mod.Exports {
		names = append(names, name)
	}
	return names
}
