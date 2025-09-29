package link

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/loader"
)

// ModuleLinker manages module interfaces and cross-module resolution
type ModuleLinker struct {
	ifaces        map[string]*iface.Iface         // Module interfaces by module path
	values        map[core.GlobalRef]eval.Value   // Cached evaluated exports
	loader        ModuleLoader                    // Interface to load modules
	loadedModules map[string]*loader.LoadedModule // Modules loaded for TopoSort
	resolver      *Resolver                       // Cached resolver instance
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
func (ml *ModuleLinker) BuildGlobalEnv(imports []*ast.ImportDecl) (GlobalEnv, *LinkDiagnostics, error) {
	env := make(GlobalEnv)
	diag := &LinkDiagnostics{
		ResolutionTrace: []string{},
		Suggestions:     []string{},
	}

	for _, imp := range imports {
		// Track resolution attempt
		diag.ResolutionTrace = append(diag.ResolutionTrace,
			fmt.Sprintf("Resolving import: %s", imp.Path))

		// Load the interface for this module
		iface, err := ml.getOrLoadInterface(imp.Path)
		if err != nil {
			// Add suggestions for missing module
			suggestedModules := ml.suggestModules(imp.Path)
			for _, suggestion := range suggestedModules {
				diag.Suggestions = append(diag.Suggestions,
					fmt.Sprintf("Did you mean: %s?", suggestion))
			}
			return nil, diag, fmt.Errorf("LDR001: module not found: %s", imp.Path)
		}

		// Process selective imports
		if len(imp.Symbols) > 0 {
			for _, sym := range imp.Symbols {
				diag.ResolutionTrace = append(diag.ResolutionTrace,
					fmt.Sprintf("  Looking for symbol: %s", sym))

				item, ok := iface.GetExport(sym)
				if !ok {
					// Get available exports for error reporting
					var available []string
					for name := range iface.Exports {
						available = append(available, name)
					}

					// Build structured error report
					errReport := newIMP010(sym, imp.Path, available, diag.ResolutionTrace, nil)
					return nil, diag, errors.WrapReport(errReport)
				}

				// Check for conflicts
				if existing, exists := env[sym]; exists {
					providers := []string{existing.Ref.Module, imp.Path}
					errReport := newIMP011(sym, imp.Path, providers, nil)
					return nil, diag, errors.WrapReport(errReport)
				}

				diag.ResolutionTrace = append(diag.ResolutionTrace,
					fmt.Sprintf("  ✓ Resolved %s from %s", sym, imp.Path))

				env[sym] = &ImportedSym{
					Ref:    item.Ref,
					Type:   item.Type,
					Purity: item.Purity,
				}
			}
		} else {
			// Namespace imports not yet supported
			errReport := newIMP012(imp.Path, fmt.Sprintf("import %s", imp.Path), nil)
			return nil, diag, errors.WrapReport(errReport)
		}
	}

	return env, diag, nil
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

// TODO: Add similar export suggestions to IMP010 error report
// Commented out until we decide to use it (currently unused)
/*
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
*/

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
