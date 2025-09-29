package link

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/loader"
)


// ModuleLinker manages module interfaces and cross-module resolution
type ModuleLinker struct {
	ifaces        map[string]*iface.Iface              // Module interfaces by module path
	values        map[core.GlobalRef]eval.Value        // Cached evaluated exports
	loader        ModuleLoader                         // Interface to load modules
	loadedModules map[string]*loader.LoadedModule      // Modules loaded for TopoSort
	resolver      *Resolver                            // Cached resolver instance
}

// ModuleLoader is the interface for loading and evaluating modules
type ModuleLoader interface {
	LoadInterface(modulePath string) (*iface.Iface, error)
	EvaluateExport(ref core.GlobalRef) (eval.Value, error)
}

// NewModuleLinker creates a new module linker
func NewModuleLinker(loader ModuleLoader) *ModuleLinker {
	ml := &ModuleLinker{
		ifaces: make(map[string]*iface.Iface),
		values: make(map[core.GlobalRef]eval.Value),
		loader: loader,
	}
	ml.resolver = NewResolver(ml)
	return ml
}

// BuildGlobalEnv constructs the global environment for imports
func (ml *ModuleLinker) BuildGlobalEnv(imports []*ast.ImportDecl) (GlobalEnv, *LinkReport, error) {
	env := make(GlobalEnv)
	report := &LinkReport{
		ResolutionTrace: []string{},
		Suggestions:     []string{},
	}
	
	for _, imp := range imports {
		// Track resolution attempt
		report.ResolutionTrace = append(report.ResolutionTrace, 
			fmt.Sprintf("Resolving import: %s", imp.Path))
		
		// Load the interface for this module
		iface, err := ml.getOrLoadInterface(imp.Path)
		if err != nil {
			// Add suggestions for missing module
			suggestedModules := ml.suggestModules(imp.Path)
			for _, suggestion := range suggestedModules {
				report.Suggestions = append(report.Suggestions,
					fmt.Sprintf("Did you mean: %s?", suggestion))
			}
			return nil, report, fmt.Errorf("LDR001: module not found: %s", imp.Path)
		}
		
		// Process selective imports
		if len(imp.Symbols) > 0 {
			for _, sym := range imp.Symbols {
				report.ResolutionTrace = append(report.ResolutionTrace,
					fmt.Sprintf("  Looking for symbol: %s", sym))
				
				item, ok := iface.GetExport(sym)
				if !ok {
					// Suggest similar export names
					suggestedSymbols := ml.suggestExports(iface, sym)
					for _, suggestion := range suggestedSymbols {
						report.Suggestions = append(report.Suggestions,
							fmt.Sprintf("Symbol %s not found. Did you mean: %s?", sym, suggestion))
					}
					return nil, report, &ImportError{
						Code:    "IMP010",
						Message: fmt.Sprintf("symbol %s not exported from %s", sym, imp.Path),
						Module:  imp.Path,
						Symbol:  sym,
					}
				}
				
				// Check for conflicts
				if existing, exists := env[sym]; exists {
					return nil, report, &ImportConflictError{
						Code:    "IMP011",
						Message: fmt.Sprintf("conflicting import of %s", sym),
						Symbol:  sym,
						Modules: []string{existing.Ref.Module, imp.Path},
					}
				}
				
				report.ResolutionTrace = append(report.ResolutionTrace,
					fmt.Sprintf("  ✓ Resolved %s from %s", sym, imp.Path))
				
				env[sym] = &ImportedSym{
					Ref:    item.Ref,
					Type:   item.Type,
					Purity: item.Purity,
				}
			}
		} else {
			// Namespace imports not yet supported
			report.Suggestions = append(report.Suggestions,
				"Use selective import: import module/path (symbol1, symbol2)")
			return nil, report, &ImportError{
				Code:    "IMP012",
				Message: "namespace imports not yet supported",
				Module:  imp.Path,
			}
		}
	}
	
	return env, report, nil
}

// Resolver returns a GlobalResolver for the evaluator
func (ml *ModuleLinker) Resolver() *Resolver {
	return ml.resolver
}

// RegisterIface registers a module interface
func (ml *ModuleLinker) RegisterIface(iface *iface.Iface) {
	ml.ifaces[iface.Module] = iface
}

// GetIface retrieves a module interface by path
func (ml *ModuleLinker) GetIface(path string) *iface.Iface {
	return ml.ifaces[path]
}

// getOrLoadInterface retrieves or loads a module interface
func (ml *ModuleLinker) getOrLoadInterface(modulePath string) (*iface.Iface, error) {
	if iface, ok := ml.ifaces[modulePath]; ok {
		return iface, nil
	}
	
	iface, err := ml.loader.LoadInterface(modulePath)
	if err != nil {
		return nil, err
	}
	
	ml.ifaces[modulePath] = iface
	return iface, nil
}

// suggestModules suggests similar module names when a module is not found
func (ml *ModuleLinker) suggestModules(target string) []string {
	var suggestions []string
	var candidates []string
	
	// Collect all known module paths
	for path := range ml.ifaces {
		candidates = append(candidates, path)
	}
	
	// Sort by similarity (simple length difference for now)
	// TODO: Implement Levenshtein distance
	sort.Slice(candidates, func(i, j int) bool {
		diff1 := abs(len(candidates[i]) - len(target))
		diff2 := abs(len(candidates[j]) - len(target))
		return diff1 < diff2
	})
	
	// Return top 3 suggestions
	for i := 0; i < 3 && i < len(candidates); i++ {
		suggestions = append(suggestions, candidates[i])
	}
	
	return suggestions
}

// suggestExports suggests similar export names from a module interface
func (ml *ModuleLinker) suggestExports(iface *iface.Iface, target string) []string {
	var suggestions []string
	var exports []string
	
	// Collect all export names
	for name := range iface.Exports {
		exports = append(exports, name)
	}
	
	// Sort by similarity (simple prefix match for now)
	// TODO: Implement proper Levenshtein distance
	sort.Slice(exports, func(i, j int) bool {
		// Prefer exact prefix matches
		if strings.HasPrefix(exports[i], target) && !strings.HasPrefix(exports[j], target) {
			return true
		}
		if !strings.HasPrefix(exports[i], target) && strings.HasPrefix(exports[j], target) {
			return false
		}
		// Otherwise sort by length difference
		diff1 := abs(len(exports[i]) - len(target))
		diff2 := abs(len(exports[j]) - len(target))
		return diff1 < diff2
	})
	
	// Return top 3 suggestions
	for i := 0; i < 3 && i < len(exports); i++ {
		suggestions = append(suggestions, exports[i])
	}
	
	return suggestions
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ImportError represents an import-related error
type ImportError struct {
	Code    string
	Message string
	Module  string
	Symbol  string
}

func (e *ImportError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ImportConflictError represents a conflict between imports
type ImportConflictError struct {
	Code    string
	Message string
	Symbol  string
	Modules []string
}

func (e *ImportConflictError) Error() string {
	return fmt.Sprintf("%s: %s (from modules: %s)", e.Code, e.Message, strings.Join(e.Modules, ", "))
}