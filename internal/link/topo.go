package link

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/loader"
)

// ModuleID represents a module identifier
type ModuleID string

// TopoSortFromRoot performs topological sorting starting from a single root module
// Input: root module ID and map of already-loaded modules with their import lists
// Output: sorted ModuleIDs in dependency order (dependencies first)
func (ml *ModuleLinker) TopoSortFromRoot(root string, loaded map[string]*loader.LoadedModule) ([]ModuleID, error) {
	// Store loaded modules for getDependencies
	ml.loadedModules = loaded

	// DEBUG: Show loaded modules
	// fmt.Printf("DEBUG TopoSort: root=%s, loaded modules: %v\n", root, func() []string {
	// 	var keys []string
	// 	for k := range loaded {
	// 		keys = append(keys, k)
	// 	}
	// 	return keys
	// }())

	// Use single root for topological sort
	roots := []ModuleID{ModuleID(root)}
	// Track visited nodes and current path for cycle detection
	visited := make(map[ModuleID]bool)
	inPath := make(map[ModuleID]bool)
	var sorted []ModuleID
	var cyclePath []ModuleID

	// DFS helper with cycle detection
	var dfs func(module ModuleID) error
	dfs = func(module ModuleID) error {
		// Already processed
		if visited[module] {
			return nil
		}

		// Cycle detected
		if inPath[module] {
			// Build the cycle for the error message: the path from the first
			// visit of module back to module. (This used to append the cycle
			// to the whole DFS path, so the message repeated it after the
			// path that led there.)
			var cycle []ModuleID
			var into ModuleID // the module whose import led into the cycle
			foundStart := false
			for i, m := range cyclePath {
				if m == module && !foundStart {
					foundStart = true
					if i > 0 {
						into = cyclePath[i-1]
					}
				}
				if foundStart {
					cycle = append(cycle, m)
				}
			}
			cycle = append(cycle, module) // Complete the cycle
			return &CycleError{
				Code:    "LDR002",
				Cycle:   cycle,
				Written: ml.cycleSpellings(into, cycle),
			}
		}

		// Mark as being processed
		inPath[module] = true
		cyclePath = append(cyclePath, module)

		// Get dependencies for this module
		deps, err := ml.getDependencies(module)
		if err != nil {
			return err
		}

		// Process dependencies first
		for _, dep := range deps {
			if err := dfs(dep); err != nil {
				return err
			}
		}

		// Mark as visited and add to sorted list
		visited[module] = true
		inPath[module] = false
		cyclePath = cyclePath[:len(cyclePath)-1]
		sorted = append(sorted, module)

		return nil
	}

	// Process all roots
	for _, root := range roots {
		if err := dfs(root); err != nil {
			return nil, err
		}
	}

	// DFS post-order already gives us dependency order (dependencies first)
	// No need to reverse!
	return sorted, nil
}

// getDependencies returns the module IDs that the given module depends on
func (ml *ModuleLinker) getDependencies(module ModuleID) ([]ModuleID, error) {
	// Get dependencies from the already-loaded module
	if loadedMod, ok := ml.loadedModules[string(module)]; ok {
		var deps []ModuleID
		for _, imp := range loadedMod.Imports {
			// Convert import path to canonical ModuleID
			deps = append(deps, ModuleID(loader.CanonicalModuleID(imp)))
		}
		return deps, nil
	}

	// Module should have been in the loaded map
	// Include a search trace for debugging
	var available []string
	for id := range ml.loadedModules {
		available = append(available, id)
	}
	return nil, fmt.Errorf("LDR001: module not found: %s\nAvailable modules: %v", module, available)
}

// cycleSpellings returns, for each module of a cycle, the spelling that
// imported it when the loader rewrote that spelling to the canonical identity
// ("" otherwise): for the first module, as the module that led into the cycle
// (into, "" for the entry) wrote it; for every other, as the previous one did.
func (ml *ModuleLinker) cycleSpellings(into ModuleID, cycle []ModuleID) []string {
	written := make([]string, len(cycle))
	if len(cycle) > 0 && into != "" {
		written[0] = ml.importSpelling(into, cycle[0])
	}
	for i := 1; i < len(cycle); i++ {
		written[i] = ml.importSpelling(cycle[i-1], cycle[i])
	}
	return written
}

// importSpelling is the WrittenPath of from's import of to, or "".
func (ml *ModuleLinker) importSpelling(from, to ModuleID) string {
	mod := ml.loadedModules[string(from)]
	if mod == nil || mod.File == nil {
		return ""
	}
	for _, imp := range mod.File.Imports {
		if imp != nil && loader.CanonicalModuleID(imp.Path) == string(to) {
			return imp.WrittenPath
		}
	}
	return ""
}

// CycleError represents a dependency cycle error
type CycleError struct {
	Code  string
	Cycle []ModuleID
	// Written[i] is the spelling that imported Cycle[i], when it differs from
	// the canonical module ID ("" otherwise; nil when unknown).
	Written []string
}

func (e *CycleError) Error() string {
	var path []string
	for i, m := range e.Cycle {
		written := ""
		if i < len(e.Written) {
			written = e.Written[i]
		}
		path = append(path, loader.DisplayModule(string(m), written))
	}
	return fmt.Sprintf("%s: dependency cycle detected: %s", e.Code, strings.Join(path, " -> "))
}

// GetSuggestion returns a fix suggestion for the cycle error
func (e *CycleError) GetSuggestion() string {
	if len(e.Cycle) < 2 {
		return ""
	}

	// Suggest breaking the cycle by extracting shared code
	return fmt.Sprintf("Consider extracting shared functionality from %s and %s into a separate module",
		e.Cycle[0], e.Cycle[1])
}
