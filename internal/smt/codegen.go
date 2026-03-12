package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// FunctionParam describes a function parameter for SMT encoding.
type FunctionParam struct {
	Name string
	Type types.Type
}

// activeResolvedCallees holds the set of callee function names that have been
// resolved as define-fun during the current EncodeFunction call.
// This is a package-level variable set during encoding (sequential per function).
var activeResolvedCallees map[string]bool

// activeRecordTypes maps record sort names to their type info.
// Populated during EncodeFunction, used by encodeRecord/encodeRecordUpdate.
var activeRecordTypes map[string]*RecordTypeInfo

// activeFieldSetToSort maps a canonical field-set key to sort name,
// enabling record construction lookup by field names.
var activeFieldSetToSort map[string]string

// EncodeResult holds the generated SMT-LIB program for a function.
type EncodeResult struct {
	// SMTLib is the complete SMT-LIB program text.
	SMTLib string
	// Declarations are the type/constant declarations.
	Declarations []string
	// Assertions are the assert statements.
	Assertions []string
	// BodyExpr is the encoded function body expression.
	BodyExpr string
}

// EncodeFunctionOpts holds optional parameters for cross-function call resolution.
type EncodeFunctionOpts struct {
	// Program is the full Core program (needed for resolving cross-function calls).
	Program *core.Program
	// SurfaceParams maps function names to their parameter info.
	SurfaceParams map[string][]FunctionParam
	// SurfaceReturnSorts maps function names to their return sort strings.
	SurfaceReturnSorts map[string]string
	// ReturnType is the function's return type (used to discover record types in return position).
	ReturnType types.Type
	// Body is the function body expression (used to discover record types constructed in the body).
	Body core.CoreExpr
	// Contracts are the function's contracts (used to discover record types in ensures clauses).
	Contracts []*core.Contract
	// RecursiveDepth enables bounded recursion unrolling at the given depth (1-10).
	// When > 0 and the function is self-recursive, generates a define-fun chain
	// instead of rejecting. 0 means no unrolling (default behavior).
	RecursiveDepth int
	// HOFInlineDepth controls the unrolling depth for HOF specializations
	// (map, filter, foldl with literal lambda arguments). When > 0, inlinable
	// HOF calls are specialized into recursive functions and unrolled.
	// Default: 3 if not specified and HOF calls are detected.
	HOFInlineDepth int
	// ExtraDeclarations are additional SMT-LIB declarations to prepend
	// (e.g., record types found in ADT constructor fields that need
	// to be declared before the ADT itself).
	ExtraDeclarations []string
}

// EncodeFunction generates a complete SMT-LIB program for verifying a function's contracts.
//
// The encoding follows the standard pattern for bounded verification:
//  1. Declare types (datatypes for ADTs)
//  2. Declare symbolic variables (function parameters)
//  3. Assert preconditions (requires clauses)
//  4. Define function body as "result"
//  5. Assert negation of postconditions (ensures clauses)
//  6. Check satisfiability
//
// If the result is "unsat", all postconditions hold for all inputs satisfying preconditions.
// If "sat", the model provides a counterexample.
func EncodeFunction(
	funcName string,
	params []FunctionParam,
	body core.CoreExpr,
	returnSort string,
	meta *core.DeclMeta,
	adtTypes map[string][]ADTVariant,
	opts ...EncodeFunctionOpts,
) (*EncodeResult, error) {
	ctx := NewSMTContext()
	result := &EncodeResult{}

	// Resolve cross-function call dependencies
	var calleeDefs []CalleeDef
	if len(opts) > 0 && opts[0].Program != nil {
		var err error
		calleeDefs, err = ResolveCallees(
			funcName, body, opts[0].Program,
			opts[0].SurfaceParams, opts[0].SurfaceReturnSorts, adtTypes,
		)
		if err != nil {
			return nil, fmt.Errorf("resolving cross-function calls: %w", err)
		}
		// Register resolved callees so encodeApp can recognize them
		for _, def := range calleeDefs {
			ctx.ResolvedCallees[def.Name] = true
		}
	}

	// Set active resolved callees for encodeApp to use
	activeResolvedCallees = ctx.ResolvedCallees
	defer func() { activeResolvedCallees = nil }()

	// Collect and declare record types from function parameters and return type
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	var returnType types.Type
	var coreBody core.CoreExpr
	var contracts []*core.Contract
	if len(opts) > 0 {
		returnType = opts[0].ReturnType
		coreBody = opts[0].Body
		contracts = opts[0].Contracts
	}
	collectAndDeclareRecordTypes(params, returnSort, returnType, coreBody, contracts, ctx, result)

	// Step 0.5: Add extra declarations (e.g., record types from ADT constructor fields)
	// These must come before ADT declarations since ADTs may reference these sorts.
	// Skip any that were already declared by collectAndDeclareRecordTypes.
	if len(opts) > 0 {
		for _, decl := range opts[0].ExtraDeclarations {
			// Extract sort name from "(declare-datatype SortName ...)"
			// to check if already declared
			sortName := extractSortNameFromDecl(decl)
			if sortName != "" && ctx.DeclaredTypes[sortName] {
				continue // Already declared by record type discovery
			}
			result.Declarations = append(result.Declarations, decl)
			if sortName != "" {
				ctx.DeclaredTypes[sortName] = true
			}
		}
	}

	// Step 1: Declare ADT types
	for typeName, variants := range adtTypes {
		if !ctx.DeclaredTypes[typeName] {
			decl := DeclareDatatype(typeName, variants)
			result.Declarations = append(result.Declarations, decl)
			ctx.DeclaredTypes[typeName] = true
		}
	}

	// Step 1.5: Add callee define-fun declarations (for cross-function calls)
	for _, def := range calleeDefs {
		result.Declarations = append(result.Declarations, def.SMTLib)
	}

	// Step 1.55: HOF inlining — specialize map/filter/foldl with literal lambdas
	var hofInlinedKinds []string
	hofDepth := 0
	if len(opts) > 0 {
		hofDepth = opts[0].HOFInlineDepth
	}
	if hofDepth == 0 {
		hofDepth = 3 // default depth for HOF specializations
	}
	inlineResult := InlineHOFCalls(funcName, body, hofDepth)
	if inlineResult != nil {
		body = inlineResult.NewBody
		for _, spec := range inlineResult.Specializations {
			result.Declarations = append(result.Declarations, spec.Declarations...)
			// Register specialized functions as resolved callees so encodeApp recognizes them
			ctx.ResolvedCallees[spec.TopLevelName] = true
			switch spec.Kind {
			case HOFMap:
				hofInlinedKinds = append(hofInlinedKinds, "map")
			case HOFFilter:
				hofInlinedKinds = append(hofInlinedKinds, "filter")
			case HOFFoldl:
				hofInlinedKinds = append(hofInlinedKinds, "foldl")
			}
		}
	}
	// Suppress "unused" warning for hofInlinedKinds (used for labeling output)
	_ = hofInlinedKinds

	// Step 1.6: Bounded recursion unrolling (if enabled)
	var unrollTopName string
	recursiveDepth := 0
	if len(opts) > 0 {
		recursiveDepth = opts[0].RecursiveDepth
	}
	if recursiveDepth > 0 && isRecursive(body, funcName) {
		unrollResult, err := UnrollRecursiveFunction(UnrollConfig{
			FuncName:   funcName,
			Params:     params,
			Body:       body,
			ReturnSort: returnSort,
			Depth:      recursiveDepth,
		})
		if err != nil {
			return nil, fmt.Errorf("unrolling recursive function: %w", err)
		}
		result.Declarations = append(result.Declarations, unrollResult.Declarations...)
		unrollTopName = unrollResult.TopLevelName
	}

	// Step 2: Declare symbolic variables for function parameters
	for _, p := range params {
		sort, err := MapType(p.Type)
		if err != nil {
			return nil, fmt.Errorf("cannot encode parameter %q: %w", p.Name, err)
		}
		decl := DeclareConst(p.Name, sort)
		result.Declarations = append(result.Declarations, decl)
		ctx.Variables[p.Name] = sort
	}

	// Step 3: Assert preconditions
	for _, contract := range meta.Contracts {
		if contract.Kind != core.RequiresKind {
			continue
		}
		if contract.Expr == nil {
			continue
		}
		encoded, err := EncodeExpr(contract.Expr)
		if err != nil {
			return nil, fmt.Errorf("cannot encode requires clause: %w", err)
		}
		result.Assertions = append(result.Assertions, Assert(encoded))
	}

	// Step 4: Encode function body and bind to "result"
	var bodyExpr string
	if unrollTopName != "" {
		// Bounded recursion: use the top-level unrolled function
		var paramNames []string
		for _, p := range params {
			paramNames = append(paramNames, p.Name)
		}
		bodyExpr = fmt.Sprintf("(%s %s)", unrollTopName, strings.Join(paramNames, " "))
	} else {
		var err error
		bodyExpr, err = EncodeExpr(body)
		if err != nil {
			return nil, fmt.Errorf("cannot encode function body: %w", err)
		}
	}
	result.BodyExpr = bodyExpr

	// Determine result type
	resultSort := returnSort
	if resultSort == "" {
		resultSort = inferResultSort(params, body, ctx, adtTypes)
	}
	resultDecl := fmt.Sprintf("(define-const result %s %s)", resultSort, bodyExpr)
	result.Declarations = append(result.Declarations, resultDecl)

	// Step 5: Assert negation of postconditions (ensures clauses)
	for _, contract := range meta.Contracts {
		if contract.Kind != core.EnsuresKind {
			continue
		}
		if contract.Expr == nil {
			continue
		}
		encoded, err := EncodeExpr(contract.Expr)
		if err != nil {
			return nil, fmt.Errorf("cannot encode ensures clause: %w", err)
		}
		result.Assertions = append(result.Assertions, AssertNot(encoded))
	}

	// Step 6: Build complete SMT-LIB program
	var lines []string
	lines = append(lines, fmt.Sprintf("; Verification of %s", funcName))
	lines = append(lines, "(set-logic ALL)")
	lines = append(lines, "")
	lines = append(lines, result.Declarations...)
	lines = append(lines, "")
	lines = append(lines, result.Assertions...)
	lines = append(lines, "")
	lines = append(lines, CheckSat())
	lines = append(lines, GetModel())

	result.SMTLib = strings.Join(lines, "\n")
	return result, nil
}

// extractSortNameFromDecl extracts the sort name from a "(declare-datatype SortName ...)" string.
func extractSortNameFromDecl(decl string) string {
	const prefix = "(declare-datatype "
	if !strings.HasPrefix(decl, prefix) {
		return ""
	}
	rest := decl[len(prefix):]
	idx := strings.IndexByte(rest, ' ')
	if idx < 0 {
		return ""
	}
	return rest[:idx]
}
