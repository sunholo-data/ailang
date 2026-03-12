package smt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ErrUnresolvableTypes indicates that the function references types
// (ADTs, record aliases) that cannot be fully resolved in Z3.
// Functions with this error should be SKIPPED, not reported as errors.
var ErrUnresolvableTypes = errors.New("unresolvable cross-module types")

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
	// RecordTypeAliases maps named record type aliases (e.g., "TableCell", "Point")
	// to their TRecord types. These are pre-registered during record type collection
	// so that named sorts take priority over anonymous Record_field1_field2 sorts.
	RecordTypeAliases map[string]*types.TRecord
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

	// Step 0: Pre-register named record type aliases (e.g., type TableCell = {text: string, ...}).
	// These must be registered BEFORE collectAndDeclareRecordTypes so that when anonymous
	// record literals with the same fields are discovered in function bodies, the named
	// sort takes priority (e.g., "TableCell" instead of "Record_colSpan_merged_rowSpan_text").
	//
	// Multi-pass: aliases may reference other aliases (e.g., ParsedDocument uses DocMetadata).
	// Iterate until no more progress, declaring aliases only when all their field sorts
	// are primitive (Int/Bool/String/Real), already declared, or other record aliases.
	// Aliases that reference ADT sorts (like Block) are skipped to avoid circular
	// dependencies — they'll be emitted only if needed by function params/return type.
	if len(opts) > 0 && opts[0].RecordTypeAliases != nil {
		remaining := make(map[string]*types.TRecord)
		for name, rec := range opts[0].RecordTypeAliases {
			remaining[name] = rec
		}
		for {
			progress := false
			for aliasName, rec := range remaining {
				if ctx.DeclaredTypes[aliasName] {
					delete(remaining, aliasName)
					progress = true
					continue
				}
				namedRec := &types.TRecord{Fields: rec.Fields, TypeName: aliasName}
				fieldSorts, err := MapRecordFields(namedRec)
				if err != nil {
					delete(remaining, aliasName)
					continue
				}
				// Only declare if all field sorts are primitives or already declared.
				// Do NOT count ADT types as resolvable — they may create circular
				// dependencies (e.g., ParsedDocument → Block → Record_blocks_kind → Block).
				if !allFieldsPrimitiveOrDeclared(fieldSorts, ctx) {
					continue // defer to next pass
				}
				fieldNames := SortedFieldNamesStr(fieldSorts)
				info := &RecordTypeInfo{
					SortName:   aliasName,
					CtorName:   RecordConstructorName(aliasName),
					FieldNames: fieldNames,
					FieldSorts: fieldSorts,
				}
				activeRecordTypes[aliasName] = info
				key := strings.Join(fieldNames, ",")
				activeFieldSetToSort[key] = aliasName
				decl := DeclareRecordDatatype(aliasName, fieldSorts)
				result.Declarations = append(result.Declarations, decl)
				ctx.DeclaredTypes[aliasName] = true
				delete(remaining, aliasName)
				progress = true
			}
			if !progress {
				break // no more aliases can be resolved
			}
		}
	}

	collectAndDeclareRecordTypes(params, returnSort, returnType, coreBody, contracts, ctx, result)

	// Step 0.5 + Step 1: Declare inline record types and ADT types.
	// Use multi-pass to handle dependencies: declarations are only emitted when
	// all sort references they contain are already declared or primitive.
	// This prevents cascading errors from recursive ADTs (e.g., Block → Record_blocks_kind → Block).
	{
		// Collect all pending declarations
		type pendingDecl struct {
			sortName string
			decl     string
		}
		var pending []pendingDecl

		// Add extra declarations (inline record types from ADT constructor fields)
		if len(opts) > 0 {
			for _, decl := range opts[0].ExtraDeclarations {
				sortName := extractSortNameFromDecl(decl)
				if sortName != "" && ctx.DeclaredTypes[sortName] {
					continue
				}
				pending = append(pending, pendingDecl{sortName: sortName, decl: decl})
			}
		}

		// Add ADT type declarations
		for typeName, variants := range adtTypes {
			if !ctx.DeclaredTypes[typeName] {
				decl := DeclareDatatype(typeName, variants)
				pending = append(pending, pendingDecl{sortName: typeName, decl: decl})
			}
		}

		// Multi-pass: emit declarations only when all referenced sorts are declared.
		for {
			progress := false
			var remaining []pendingDecl
			for _, pd := range pending {
				if ctx.DeclaredTypes[pd.sortName] {
					progress = true
					continue // already declared in a previous pass
				}
				if declReferencesUndeclaredSort(pd.decl, ctx) {
					remaining = append(remaining, pd) // defer
					continue
				}
				result.Declarations = append(result.Declarations, pd.decl)
				if pd.sortName != "" {
					ctx.DeclaredTypes[pd.sortName] = true
				}
				progress = true
			}
			if !progress || len(remaining) == 0 {
				break
			}
			pending = remaining
		}
		// If any declarations remain unresolved after multi-pass, we have circular
		// dependencies (mutual recursion) that require declare-datatypes (plural).
		// For now, emit them and let validateDeclarations catch ordering issues.
		// Sort: non-recursive first, then potentially recursive ones.
		for _, pd := range pending {
			if !ctx.DeclaredTypes[pd.sortName] {
				result.Declarations = append(result.Declarations, pd.decl)
				if pd.sortName != "" {
					ctx.DeclaredTypes[pd.sortName] = true
				}
			}
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

	// Step 5.5: Validate declarations — check for forward references to undeclared sorts.
	// If any declaration references a sort that's not primitive and not declared,
	// return ErrUnresolvableTypes so the caller can skip this function gracefully
	// instead of sending broken SMT-LIB to Z3.
	if err := validateDeclarations(result.Declarations, ctx); err != nil {
		return nil, err
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

// extractConstructorNames extracts constructor/accessor names from a declare-datatype declaration.
// Format: (declare-datatype Name ((Ctor1) (Ctor2 (field1 Type1) (field2 Type2))))
// Returns all constructor names AND field accessor names (which are NOT sort references).
func extractConstructorNames(decl string) map[string]bool {
	names := make(map[string]bool)
	// Find the variants section: everything after "(declare-datatype Name ("
	const prefix = "(declare-datatype "
	if !strings.HasPrefix(decl, prefix) {
		return names
	}
	rest := decl[len(prefix):]
	spaceIdx := strings.IndexByte(rest, ' ')
	if spaceIdx < 0 {
		return names
	}
	body := rest[spaceIdx+1:] // now starts with "((" for variants

	// Parse: find each constructor (first token after each top-level "(")
	depth := 0
	inToken := false
	tokenStart := 0
	isConstructor := false // true when depth indicates constructor name position
	fieldNameNext := false // true when we just saw "(" at field level

	for i := 0; i < len(body); i++ {
		ch := body[i]
		switch ch {
		case '(':
			depth++
			inToken = false
			// depth 2 = start of a variant: next token is constructor name
			if depth == 2 {
				isConstructor = true
			}
			// depth 3 = start of a field: next token is field accessor name
			if depth == 3 {
				fieldNameNext = true
			}
		case ')':
			if inToken {
				token := body[tokenStart:i]
				if isConstructor {
					names[token] = true
				}
				if fieldNameNext {
					names[token] = true
				}
				inToken = false
			}
			isConstructor = false
			fieldNameNext = false
			depth--
		case ' ', '\t', '\n':
			if inToken {
				token := body[tokenStart:i]
				if isConstructor {
					names[token] = true
					isConstructor = false
				}
				if fieldNameNext {
					names[token] = true
					fieldNameNext = false
				}
				inToken = false
			}
		default:
			if !inToken {
				inToken = true
				tokenStart = i
			}
		}
	}
	return names
}

// declReferencesUndeclaredSort checks if a single declare-datatype declaration
// references any sort that is not yet declared in the context.
func declReferencesUndeclaredSort(decl string, ctx *SMTContext) bool {
	if !strings.HasPrefix(decl, "(declare-datatype ") {
		return false
	}
	sortName := extractSortNameFromDecl(decl)
	constructors := extractConstructorNames(decl)
	refs := sortRefPattern.FindAllString(decl, -1)
	for _, ref := range refs {
		if primitiveSorts[ref] || ref == sortName || constructors[ref] {
			continue
		}
		if ctx.DeclaredTypes[ref] {
			continue
		}
		if strings.HasPrefix(ref, "mk_") {
			continue
		}
		return true // undeclared sort referenced
	}
	return false
}

// sortRefPattern matches sort names referenced inside SMT-LIB declarations.
// Captures sort names from (Seq SortName), (define-const result SortName ...),
// and (fieldName SortName) patterns.
var sortRefPattern = regexp.MustCompile(`\b([A-Z][A-Za-z_0-9]*)\b`)

// primitiveSorts are Z3 built-in sorts that don't need declaration.
var primitiveSorts = map[string]bool{
	"Int": true, "Bool": true, "String": true, "Real": true,
	"Seq": true, "ALL": true, "Array": true,
}

// validateDeclarations checks that all sort references in the assembled declarations
// resolve to either primitive sorts or declared sorts. Returns ErrUnresolvableTypes
// if any declaration references an undeclared non-primitive sort.
func validateDeclarations(decls []string, ctx *SMTContext) error {
	// Walk declarations IN ORDER, tracking which sorts have been declared so far.
	// This catches forward references that Z3 would reject.
	//
	// We seed with ctx.DeclaredTypes BUT exclude sorts that are being declared
	// in this batch — those must be checked positionally.
	declaredInBatch := make(map[string]bool)
	for _, decl := range decls {
		if name := extractSortNameFromDecl(decl); name != "" {
			declaredInBatch[name] = true
		}
	}
	declared := make(map[string]bool)
	for k, v := range ctx.DeclaredTypes {
		if v && !declaredInBatch[k] {
			declared[k] = true
		}
	}

	for _, decl := range decls {
		if !strings.HasPrefix(decl, "(declare-datatype ") {
			// Non-datatype declarations (declare-const, define-const, etc.) are fine
			continue
		}
		sortName := extractSortNameFromDecl(decl)
		constructors := extractConstructorNames(decl)
		refs := sortRefPattern.FindAllString(decl, -1)
		for _, ref := range refs {
			if primitiveSorts[ref] || ref == sortName || constructors[ref] {
				continue
			}
			if declared[ref] {
				continue
			}
			if strings.HasPrefix(ref, "mk_") {
				continue
			}
			return fmt.Errorf("%w: sort %q referenced in declaration of %q is not declared", ErrUnresolvableTypes, ref, sortName)
		}
		// Mark this sort as declared for subsequent declarations
		if sortName != "" {
			declared[sortName] = true
		}
	}
	return nil
}

// allFieldsPrimitiveOrDeclared checks whether all field sorts are either
// primitive (Int, Bool, String, Real), sequence types of primitives/declared sorts,
// or already declared in the context. Does NOT count pending ADT types as declared
// to avoid circular dependency chains.
func allFieldsPrimitiveOrDeclared(fieldSorts map[string]string, ctx *SMTContext) bool {
	for _, sort := range fieldSorts {
		if !isSortPrimitiveOrDeclared(sort, ctx) {
			return false
		}
	}
	return true
}

// isSortPrimitiveOrDeclared checks if a sort is primitive or already declared.
func isSortPrimitiveOrDeclared(sort string, ctx *SMTContext) bool {
	switch sort {
	case "Int", "Bool", "String", "Real":
		return true
	}
	if ctx.DeclaredTypes[sort] {
		return true
	}
	// Sequence type: (Seq X) — check inner sort
	if strings.HasPrefix(sort, "(Seq ") && strings.HasSuffix(sort, ")") {
		inner := sort[5 : len(sort)-1]
		return isSortPrimitiveOrDeclared(inner, ctx)
	}
	return false
}
