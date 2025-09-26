package link

import (
	"fmt"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Linker resolves dictionary references to concrete implementations
type Linker struct {
	registry     *types.DictionaryRegistry
	errors       []error
	warnings     []string
	dryRun       bool
	resolvedRefs map[string]bool // Track resolved references for idempotency
}

// NewLinker creates a new linker without a registry (for REPL)
func NewLinker() *Linker {
	return &Linker{
		registry:     types.NewDictionaryRegistry(),
		errors:       nil,
		warnings:     nil,
		dryRun:       false,
		resolvedRefs: make(map[string]bool),
	}
}

// NewLinkerWithRegistry creates a new linker with the given registry
func NewLinkerWithRegistry(registry *types.DictionaryRegistry) *Linker {
	return &Linker{
		registry:     registry,
		errors:       nil,
		warnings:     nil,
		dryRun:       false,
		resolvedRefs: make(map[string]bool),
	}
}

// AddDictionary adds a dictionary to the linker (for REPL)
func (l *Linker) AddDictionary(key string, dict core.DictValue) {
	// Register each method in the dictionary
	for method, impl := range dict.Methods {
		l.registry.Register("prelude", dict.TypeClass, dict.Type, method, impl)
	}
}

// DryRun performs a dry run to find required instances
func (l *Linker) DryRun(expr core.CoreExpr) []string {
	// Simplified version - would walk expr tree to find DictRef nodes
	return []string{}
}

// Link links a single expression (simplified for REPL)
func (l *Linker) Link(expr core.CoreExpr) (core.CoreExpr, error) {
	// Simplified version - in practice would transform DictRef nodes
	return expr, nil
}

// LinkOptions configures the linking process
type LinkOptions struct {
	DryRun  bool   // If true, only check for errors without modifying
	Verbose bool   // If true, print detailed linking information
	Namespace string // Default namespace for lookups (usually "prelude")
}

// LinkProgram resolves all dictionary references in a Core program
func (l *Linker) LinkProgram(prog *core.Program, opts LinkOptions) (*core.Program, error) {
	l.dryRun = opts.DryRun
	l.errors = nil
	l.warnings = nil
	
	if opts.Namespace == "" {
		opts.Namespace = "prelude"
	}
	
	// Validate the registry first
	if err := l.registry.ValidateRegistry(); err != nil {
		return nil, fmt.Errorf("invalid dictionary registry: %w", err)
	}
	
	// Walk the program and collect all DictRef nodes
	dictRefs := l.collectDictRefs(prog)
	
	// Check that all references can be resolved
	for _, ref := range dictRefs {
		baseKey := l.makeDictKey(opts.Namespace, ref)
		
		// Check if already resolved (idempotency)
		if l.resolvedRefs[baseKey] {
			continue
		}
		
		// Verify that at least one method exists for this class/type
		// We check for the "add" method as a proxy for the Num class
		// In a full implementation, we'd have a better way to check dictionary existence
		var methodToCheck string
		switch ref.ClassName {
		case "Num":
			methodToCheck = "add"
		case "Eq":
			methodToCheck = "eq"
		case "Ord":
			methodToCheck = "lt"
		default:
			methodToCheck = ""
		}
		
		if methodToCheck != "" {
			testKey := fmt.Sprintf("%s::%s", baseKey, methodToCheck)
			if _, ok := l.registry.Lookup(testKey); ok {
				l.resolvedRefs[baseKey] = true
				continue
			}
		}
		
		l.errors = append(l.errors, fmt.Errorf(
			"unresolved dictionary: %s::%s for type %s at position %v",
			ref.ClassName, ref.TypeName, ref.TypeName, ref.Span()))
	}
	
	// If we have errors, return them
	if len(l.errors) > 0 {
		return nil, l.combineErrors()
	}
	
	// If dry run, we're done
	if l.dryRun {
		return prog, nil
	}
	
	// In a real implementation, we would transform DictRef nodes
	// to include the actual implementation references.
	// For now, we just return the program as-is since the evaluator
	// will look up implementations at runtime.
	return prog, nil
}

// collectDictRefs walks the program and collects all DictRef nodes
func (l *Linker) collectDictRefs(prog *core.Program) []*core.DictRef {
	var refs []*core.DictRef
	for _, decl := range prog.Decls {
		refs = append(refs, l.collectFromExpr(decl)...)
	}
	return refs
}

// collectFromExpr recursively collects DictRef nodes from an expression
func (l *Linker) collectFromExpr(expr core.CoreExpr) []*core.DictRef {
	var refs []*core.DictRef
	
	if expr == nil {
		return refs
	}
	
	switch e := expr.(type) {
	case *core.DictRef:
		refs = append(refs, e)
		
	case *core.Let:
		refs = append(refs, l.collectFromExpr(e.Value)...)
		refs = append(refs, l.collectFromExpr(e.Body)...)
		
	case *core.LetRec:
		for _, binding := range e.Bindings {
			refs = append(refs, l.collectFromExpr(binding.Value)...)
		}
		refs = append(refs, l.collectFromExpr(e.Body)...)
		
	case *core.Lambda:
		refs = append(refs, l.collectFromExpr(e.Body)...)
		
	case *core.App:
		refs = append(refs, l.collectFromExpr(e.Func)...)
		for _, arg := range e.Args {
			refs = append(refs, l.collectFromExpr(arg)...)
		}
		
	case *core.If:
		refs = append(refs, l.collectFromExpr(e.Cond)...)
		refs = append(refs, l.collectFromExpr(e.Then)...)
		refs = append(refs, l.collectFromExpr(e.Else)...)
		
	case *core.Match:
		refs = append(refs, l.collectFromExpr(e.Scrutinee)...)
		for _, arm := range e.Arms {
			refs = append(refs, l.collectFromExpr(arm.Body)...)
		}
		
	case *core.BinOp:
		refs = append(refs, l.collectFromExpr(e.Left)...)
		refs = append(refs, l.collectFromExpr(e.Right)...)
		
	case *core.UnOp:
		refs = append(refs, l.collectFromExpr(e.Operand)...)
		
	case *core.Record:
		for _, field := range e.Fields {
			refs = append(refs, l.collectFromExpr(field)...)
		}
		
	case *core.RecordAccess:
		refs = append(refs, l.collectFromExpr(e.Record)...)
		
	case *core.List:
		for _, elem := range e.Elements {
			refs = append(refs, l.collectFromExpr(elem)...)
		}
		
	case *core.DictAbs:
		refs = append(refs, l.collectFromExpr(e.Body)...)
		
	case *core.DictApp:
		refs = append(refs, l.collectFromExpr(e.Dict)...)
		for _, arg := range e.Args {
			refs = append(refs, l.collectFromExpr(arg)...)
		}
	}
	
	return refs
}

// makeDictKey constructs a dictionary key for a DictRef
func (l *Linker) makeDictKey(namespace string, ref *core.DictRef) string {
	// DictRef only has class and type, no method
	// This is for dictionary references, not method lookups
	return fmt.Sprintf("%s::%s::%s", namespace, ref.ClassName, ref.TypeName)
}

// combineErrors combines all collected errors into a single error
func (l *Linker) combineErrors() error {
	if len(l.errors) == 0 {
		return nil
	}
	if len(l.errors) == 1 {
		return l.errors[0]
	}
	
	msg := fmt.Sprintf("%d linking errors:\n", len(l.errors))
	for i, err := range l.errors {
		msg += fmt.Sprintf("  %d. %v\n", i+1, err)
	}
	return fmt.Errorf("%s", msg)
}

// GetErrors returns all linking errors
func (l *Linker) GetErrors() []error {
	return l.errors
}

// GetWarnings returns all linking warnings
func (l *Linker) GetWarnings() []string {
	return l.warnings
}

// LinkResult contains the result of linking
type LinkResult struct {
	Program  *core.Program
	Errors   []error
	Warnings []string
	Resolved map[string]bool // Dictionary keys that were resolved
}

// LinkWithResult performs linking and returns detailed results
func (l *Linker) LinkWithResult(prog *core.Program, opts LinkOptions) LinkResult {
	linked, err := l.LinkProgram(prog, opts)
	
	result := LinkResult{
		Program:  linked,
		Errors:   l.errors,
		Warnings: l.warnings,
		Resolved: l.resolvedRefs,
	}
	
	if err != nil && len(result.Errors) == 0 {
		result.Errors = append(result.Errors, err)
	}
	
	return result
}

// VerifyIdempotence checks that linking is idempotent
func VerifyIdempotence(prog *core.Program, registry *types.DictionaryRegistry) error {
	linker1 := NewLinkerWithRegistry(registry)
	linker2 := NewLinkerWithRegistry(registry)
	
	opts := LinkOptions{
		DryRun:    false,
		Namespace: "prelude",
	}
	
	// First link
	prog1, err := linker1.LinkProgram(prog, opts)
	if err != nil {
		return fmt.Errorf("first link failed: %w", err)
	}
	
	// Second link (should be identity)
	_, err = linker2.LinkProgram(prog1, opts)
	if err != nil {
		return fmt.Errorf("second link failed: %w", err)
	}
	
	// Check that the resolved references are the same
	// In a full implementation, we'd compare prog2 with prog1
	if len(linker1.resolvedRefs) != len(linker2.resolvedRefs) {
		return fmt.Errorf("linking is not idempotent: different number of resolved references")
	}
	
	// In a real implementation, we'd do deep structural comparison
	// For now, we just check that we resolved the same references
	for key := range linker1.resolvedRefs {
		if !linker2.resolvedRefs[key] {
			return fmt.Errorf("linking is not idempotent: reference %s not resolved in second pass", key)
		}
	}
	
	return nil
}