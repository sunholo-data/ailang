package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/types"
)

// verifyCommand implements the `ailang verify` CLI command.
// It performs static contract verification using SMT solving (Z3).
func verifyCommand() {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose", false, "Show generated SMT-LIB for each function")
	jsonFlag := fs.Bool("json", false, "Output results in JSON format")
	strictFlag := fs.Bool("strict", false, "Exit with error if any function cannot be verified")
	timeoutFlag := fs.Duration("timeout", 5*time.Second, "Per-function Z3 timeout")
	recursiveDepthFlag := fs.Int("verify-recursive-depth", 2, "Bounded recursion unrolling depth (1-10, 0 to disable)")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang verify [options] <file.ail>")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --verbose    Show generated SMT-LIB for each function")
		fmt.Println("  --json            Output results in JSON format")
		fmt.Println("  --strict          Exit with error if any function cannot be verified")
		fmt.Println("  --timeout         Per-function Z3 timeout (default: 5s)")
		fmt.Println("  --relax-modules   Relax MOD010 validation (allow module path mismatches)")
		fmt.Println()
		fmt.Println("Verifies requires/ensures contracts using Z3 SMT solver.")
		fmt.Println("Returns exit code 0 if all verifiable contracts are proven.")
		fmt.Println()
		fmt.Println("IFC labels (v0.16.0+):")
		fmt.Println("  Functions can use T<label> for tainted sources and T{not LABEL}")
		fmt.Println("  for sinks that refuse a given label. Declassify in the effect row")
		fmt.Println("  marks intentional label changes. See:")
		fmt.Println("    https://ailang.sunholo.com/docs/guides/ifc-labels")
		fmt.Println("    examples/runnable/contracts/inbox_injection_v2.ail")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Check Z3 availability upfront
	if !smt.Z3Available() {
		fmt.Fprintf(os.Stderr, "%s Z3 solver not found\n", red("Error:"))
		fmt.Fprintf(os.Stderr, "Install with: brew install z3 (macOS), apt install z3 (Linux), or choco install z3 / scoop install z3 (Windows)\n")
		fmt.Fprintf(os.Stderr, "Or download from https://github.com/Z3Prover/z3/releases and set AILANG_Z3_PATH.\n")
		os.Exit(1)
	}

	// Read and compile the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Suppress warnings in JSON mode so they don't pollute output
	if *jsonFlag {
		os.Setenv("AILANG_QUIET_WARNINGS", "1")
	}

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := *relaxModulesFlag
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	cfg := pipeline.Config{
		DryLink:      true, // Don't evaluate, just compile
		RelaxModules: relaxModulesEffective,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: compilation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	coreProg := result.Artifacts.Core
	surfaceAST := result.Artifacts.AST

	if coreProg == nil || coreProg.Meta == nil {
		fmt.Fprintf(os.Stderr, "%s: no functions with contracts found\n", yellow("Warning"))
		os.Exit(0)
	}

	// Extract ADT types, record type aliases, and inline record types from the Surface AST.
	adtResult := extractADTTypesWithRecords(surfaceAST)
	adtTypes := adtResult.ADTTypes
	adtRecordDecls := adtResult.RecordDecls
	recordAliases := adtResult.RecordAliases
	// Also extract from imported modules so cross-module types
	// (e.g., Block, XmlNode, TableCell) get declare-datatype in the Z3 output.
	if result.Modules != nil {
		for _, mod := range result.Modules {
			if mod.File != nil {
				modResult := extractADTTypesWithRecords(mod.File)
				for name, variants := range modResult.ADTTypes {
					if _, exists := adtTypes[name]; !exists {
						adtTypes[name] = variants
					}
				}
				adtRecordDecls = append(adtRecordDecls, modResult.RecordDecls...)
				for name, rec := range modResult.RecordAliases {
					if _, exists := recordAliases[name]; !exists {
						recordAliases[name] = rec
					}
				}
			}
		}
	}

	// Build Surface AST function lookup for param extraction
	surfaceFuncs := make(map[string]*ast.FuncDecl)
	for _, f := range surfaceAST.Funcs {
		surfaceFuncs[f.Name] = f
	}

	// Solver configuration
	solverCfg := smt.SolverConfig{
		Timeout: *timeoutFlag,
	}

	// Collect results
	var results []verifyResult
	var verified, counterexample, skipped, errCount int

	// Fixup: functions with ! {} (empty effects) are semantically pure
	// but the parser only sets IsPure for the explicit `pure` keyword.
	// For SMT verification, we treat empty-effect functions as pure.
	for funcName, meta := range coreProg.Meta {
		if fd, ok := surfaceFuncs[funcName]; ok {
			if fd.Effects != nil && len(fd.Effects) == 0 && !meta.IsPure {
				meta.IsPure = true
			}
		}
	}

	// Build surface params and return sorts for all functions (for cross-function call resolution)
	allSurfaceParams := make(map[string][]smt.FunctionParam)
	allSurfaceReturnSorts := make(map[string]string)
	for funcName, fd := range surfaceFuncs {
		var params []smt.FunctionParam
		for _, p := range fd.Params {
			paramType := convertASTTypeToType(p.Type)
			if paramType != nil {
				params = append(params, smt.FunctionParam{Name: p.Name, Type: paramType})
			}
		}
		allSurfaceParams[funcName] = params
		if fd.ReturnType != nil {
			allSurfaceReturnSorts[funcName] = astTypeToSMTSort(fd.ReturnType)
		}
	}

	// Build imported programs map and extend surface params/sorts from imported modules.
	importedPrograms := make(map[string]*core.Program)
	if result.Modules != nil {
		for modPath, mod := range result.Modules {
			if mod.Core != nil {
				importedPrograms[modPath] = mod.Core
				// Apply the same IsPure fixup as for same-module functions:
				// imported functions with ! {} (empty effects) are semantically
				// pure and must be marked so the callee resolver can inline them.
				if mod.File != nil {
					for _, fd := range mod.File.Funcs {
						if meta, ok := mod.Core.Meta[fd.Name]; ok {
							if fd.Effects != nil && len(fd.Effects) == 0 && !meta.IsPure {
								meta.IsPure = true
							}
						}
					}
				}
			}
			// Include imported module functions in surface params/sorts so the
			// callee resolver can build correct define-fun signatures for them.
			if mod.File != nil {
				for _, fd := range mod.File.Funcs {
					if _, exists := allSurfaceParams[fd.Name]; exists {
						continue // current module takes priority
					}
					var params []smt.FunctionParam
					for _, p := range fd.Params {
						if pt := convertASTTypeToType(p.Type); pt != nil {
							params = append(params, smt.FunctionParam{Name: p.Name, Type: pt})
						}
					}
					allSurfaceParams[fd.Name] = params
					if fd.ReturnType != nil {
						allSurfaceReturnSorts[fd.Name] = astTypeToSMTSort(fd.ReturnType)
					}
				}
			}
		}
	}

	encOpts := smt.EncodeFunctionOpts{
		Program:            coreProg,
		SurfaceParams:      allSurfaceParams,
		SurfaceReturnSorts: allSurfaceReturnSorts,
		ExtraDeclarations:  adtRecordDecls,
		RecordTypeAliases:  recordAliases,
		ImportedPrograms:   importedPrograms,
	}

	// Build the callee-sort gate inputs. The gate rejects a contracted function whose
	// cross-function callee has an unencodable signature *type* (e.g. Option[float], a
	// parametric ADT application) with a structured UNENCODABLE_TYPE reason, instead of
	// leaking an undeclared sort into the SMT script and crashing Z3. See
	// M-SMT-CALLEE-SORT-GATE.
	//
	// We inspect the surface AST types (not the flattened SMT sort strings): a mapped
	// sort like "Option" loses the [float] argument, and an imported parametric ADT such
	// as Option is registered in adtTypes yet cannot be declared as a usable monomorphic
	// sort — so a sort-string check wrongly passes it. The AST distinguishes a monomorphic
	// enum (SimpleType) from a parametric application (TypeApp with args) and a type var.
	calleeASTFuncs := make(map[string]*ast.FuncDecl)
	for name, fd := range surfaceFuncs {
		calleeASTFuncs[name] = fd
	}
	allTypeFiles := []*ast.File{surfaceAST}
	if result.Modules != nil {
		for _, mod := range result.Modules {
			if mod.File == nil {
				continue
			}
			allTypeFiles = append(allTypeFiles, mod.File)
			for _, fd := range mod.File.Funcs {
				if _, exists := calleeASTFuncs[fd.Name]; !exists {
					calleeASTFuncs[fd.Name] = fd
				}
			}
		}
	}
	// declarableADTs: monomorphic (non-parametric) ADT/record names that WILL be declared
	// as SMT sorts. Parametric ADTs (those whose TypeDecl carries type parameters) are
	// excluded — they cannot be monomorphized by the current encoder.
	declarableADTs := collectMonomorphicTypeNames(allTypeFiles)

	// Process each function with contracts
	for funcName, meta := range coreProg.Meta {
		if len(meta.Contracts) == 0 {
			continue
		}

		// Find the function body in Core decls
		body := findFunctionBody(coreProg, funcName)
		if body == nil {
			results = append(results, verifyResult{
				Function: funcName,
				Status:   "skipped",
				Reason:   "function body not found in Core AST",
			})
			skipped++
			continue
		}

		// Check if function has ensures clauses (nothing to verify without postconditions)
		hasEnsures := false
		for _, c := range meta.Contracts {
			if c.Kind == core.EnsuresKind {
				hasEnsures = true
				break
			}
		}
		if !hasEnsures {
			results = append(results, verifyResult{
				Function: funcName,
				Status:   "skipped",
				Reason:   "no ensures clause (nothing to verify)",
			})
			skipped++
			continue
		}

		// M4: Determine effective recursive depth for this function
		// Per-function @verify(depth: N) overrides global --verify-recursive-depth
		effectiveDepth := *recursiveDepthFlag
		if meta.VerifyDepth > 0 {
			effectiveDepth = meta.VerifyDepth
		}

		// Check if function is in the decidable SMT fragment
		encodable, rejections := smt.IsSMTEncodable(funcName, meta, body)
		// Callee-sort gate: reject cleanly if a cross-function callee has an
		// unencodable signature type (e.g. Option[float]) rather than leaking an
		// undeclared sort into the SMT script and crashing Z3. See M-SMT-CALLEE-SORT-GATE.
		if callee, badType := firstUnencodableCalleeType(funcName, body, coreProg, importedPrograms, calleeASTFuncs, declarableADTs); callee != "" {
			rejections = append(rejections, smt.SMTRejectionReason{
				Code:    smt.RejectUnencodable,
				Message: fmt.Sprintf("Function %q calls %q whose signature uses an unencodable type %q", funcName, callee, badType),
				Hint:    "Cross-function verification requires callee signatures over int/float/bool/string, records, or monomorphic enum ADTs. Parametric ADTs like Option/Result over primitives are not yet SMT-encodable.",
			})
			encodable = false
		}
		if !encodable {
			// If bounded recursion is enabled, filter out RejectRecursive
			if effectiveDepth > 0 {
				var filtered []smt.SMTRejectionReason
				for _, r := range rejections {
					if r.Code != smt.RejectRecursive {
						filtered = append(filtered, r)
					}
				}
				rejections = filtered
				encodable = len(rejections) == 0
			}
			if !encodable {
				reasons := make([]string, len(rejections))
				for i, r := range rejections {
					reasons[i] = r.Message
				}
				results = append(results, verifyResult{
					Function:   funcName,
					Status:     "skipped",
					Reason:     strings.Join(reasons, "; "),
					Rejections: rejections,
				})
				skipped++
				continue
			}
		}

		// Unwrap Lambda nodes to separate params from body
		params, innerBody := unwrapLambdaParams(funcName, surfaceFuncs, body)

		// Determine return sort from Surface AST
		returnSort := ""
		var returnType types.Type
		if fd, ok := surfaceFuncs[funcName]; ok && fd.ReturnType != nil {
			returnSort = astTypeToSMTSort(fd.ReturnType)
			returnType = convertASTTypeToType(fd.ReturnType)
		}

		// Build per-function encode options (return type, body, contracts for record discovery)
		funcEncOpts := encOpts
		funcEncOpts.ReturnType = returnType
		funcEncOpts.Body = innerBody
		funcEncOpts.Contracts = meta.Contracts
		funcEncOpts.RecursiveDepth = effectiveDepth

		// Demand-driven ADT filtering: only pass ADT types that this function actually
		// references via its params, return type, or body. This prevents cascade failures
		// where unrelated cross-module types (e.g., Json) poison functions that only use
		// primitive types (e.g., int → int).
		funcADTTypes := filterADTTypesForFunction(params, returnSort, innerBody, adtTypes)

		// Demand-driven record-alias and inline-record filtering: same principle.
		// Without these, every function gets the union of all aliases and inline
		// records from every imported module, causing cascade Z3 errors when an
		// unused alias references a sort that the ADT filter correctly dropped.
		// buildNeededSortSet must see the FULL alias map (not the filtered one)
		// so that aliases referenced only via inline-record bodies (e.g., TableCell
		// inside Record_headers_rows) are still detected and pulled into `needed`.
		needed := buildNeededSortSet(params, returnSort, innerBody, funcADTTypes, recordAliases, adtRecordDecls)
		funcEncOpts.ExtraDeclarations = filterExtraDeclarationsForFunction(adtRecordDecls, needed)
		// Filter aliases against the widened needed-set: includes both directly-
		// referenced aliases AND aliases pulled in by inline-record bodies.
		widenedAliases := make(map[string]*types.TRecord)
		for name, rec := range recordAliases {
			if needed[name] {
				widenedAliases[name] = rec
			}
		}
		funcEncOpts.RecordTypeAliases = widenedAliases

		// Encode function to SMT-LIB (with cross-function call support)
		encResult, err := smt.EncodeFunction(funcName, params, innerBody, returnSort, meta, funcADTTypes, funcEncOpts)
		if err != nil {
			// If the error is due to unresolvable cross-module types,
			// skip gracefully instead of reporting as error
			if errors.Is(err, smt.ErrUnresolvableTypes) {
				results = append(results, verifyResult{
					Function: funcName,
					Status:   "skipped",
					Reason:   fmt.Sprintf("Uses cross-module types not yet supported in Z3 encoding (%v)", err),
					Rejections: []smt.SMTRejectionReason{{
						Code:    smt.RejectUnencodable,
						Message: err.Error(),
						Hint:    "Cross-module record type aliases and recursive ADTs are not yet supported in Z3 verification",
					}},
				})
				skipped++
				continue
			}
			results = append(results, verifyResult{
				Function: funcName,
				Status:   "error",
				Reason:   fmt.Sprintf("encoding error: %v", err),
			})
			errCount++
			continue
		}

		// Solve with Z3
		solveResult, err := smt.Solve(encResult.SMTLib, solverCfg)
		if err != nil {
			results = append(results, verifyResult{
				Function: funcName,
				Status:   "error",
				Reason:   fmt.Sprintf("solver error: %v", err),
			})
			errCount++
			continue
		}

		vr := verifyResult{
			Function: funcName,
			Duration: solveResult.Duration,
		}
		// Mark bounded recursion depth if the function is recursive
		if effectiveDepth > 0 && smt.IsRecursiveFunc(innerBody, funcName) {
			vr.BoundedDepth = effectiveDepth
		}
		if *verboseFlag {
			vr.SMTLib = encResult.SMTLib
		}

		switch solveResult.Status {
		case smt.StatusVerified:
			vr.Status = "verified"
			verified++
		case smt.StatusCounterexample:
			vr.Status = "counterexample"
			vr.Model = solveResult.Model
			counterexample++
		case smt.StatusUnknown:
			vr.Status = "unknown"
			vr.Reason = solveResult.Error
			errCount++
		case smt.StatusError:
			vr.Status = "error"
			vr.Reason = solveResult.Error
			errCount++
		}

		results = append(results, vr)
	}

	// Output results
	if *jsonFlag {
		printVerifyJSON(results, filename, verified, counterexample, skipped, errCount)
	} else {
		printVerifyHuman(results, filename, verified, counterexample, skipped, errCount, *verboseFlag)
	}

	// Exit code logic
	if counterexample > 0 {
		os.Exit(1) // Contract violations found
	}
	if *strictFlag && (skipped > 0 || errCount > 0) {
		os.Exit(1) // Strict mode: non-verifiable functions are failures
	}
}

// verifyResult holds the result of verifying a single function.
type verifyResult struct {
	Function     string                   `json:"function"`
	Status       string                   `json:"status"` // verified, counterexample, skipped, error, unknown
	Reason       string                   `json:"reason,omitempty"`
	Model        []smt.ModelBinding       `json:"model,omitempty"`
	Rejections   []smt.SMTRejectionReason `json:"rejections,omitempty"`
	Duration     time.Duration            `json:"duration,omitempty"`
	SMTLib       string                   `json:"smtlib,omitempty"`
	BoundedDepth int                      `json:"bounded_depth,omitempty"`
}

// adtExtractionResult holds ADT types, inline record types from ADT constructor fields,
// and named record type aliases (type X = {fields}).
type adtExtractionResult struct {
	ADTTypes      map[string][]smt.ADTVariant
	RecordDecls   []string                  // SMT-LIB declare-datatype for inline record types in ADT fields
	RecordAliases map[string]*types.TRecord // Named record type aliases (e.g., "TableCell" → TRecord)
}

// extractADTTypesWithRecords extracts ADT type definitions and record type aliases
// from the Surface AST and converts them to SMT-compatible formats.
// Handles three cases:
//  1. Record type aliases (type TableCell = {text: string, ...}) → RecordAliases
//  2. ADTs (type Block = TextBlock(...) | ...) → ADTTypes
//  3. Inline record types in ADT constructor fields → RecordDecls
func extractADTTypesWithRecords(file *ast.File) adtExtractionResult {
	result := adtExtractionResult{
		ADTTypes:      make(map[string][]smt.ADTVariant),
		RecordAliases: make(map[string]*types.TRecord),
	}
	// Track record types found in ADT fields that need declaration
	recordsSeen := make(map[string]bool)

	for _, decl := range file.Decls {
		typeDecl, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}

		// Case 1: Record type alias (type TableCell = {text: string, colSpan: int, ...})
		if recType, ok := typeDecl.Definition.(*ast.RecordType); ok {
			collectNamedRecordAlias(typeDecl.Name, recType, &result, recordsSeen)
			continue
		}

		// Case 2: ADT (type Block = TextBlock(...) | TableBlock(...) | ...)
		algType, ok := typeDecl.Definition.(*ast.AlgebraicType)
		if !ok {
			continue
		}

		var variants []smt.ADTVariant
		for _, ctor := range algType.Constructors {
			variant := smt.ADTVariant{Name: ctor.Name}
			for _, field := range ctor.Fields {
				sortName := astTypeToSMTSort(field.Type)
				fieldName := field.Name
				if fieldName == "" {
					// Prefix with constructor name to ensure uniqueness across
					// all constructors in the datatype (Z3 requirement).
					fieldName = fmt.Sprintf("%s_%d", ctor.Name, len(variant.Fields))
				}
				variant.Fields = append(variant.Fields, smt.ADTField{
					Name: fieldName,
					Sort: sortName,
				})
				// If this field is an inline record type, collect its declaration
				if recType, ok := field.Type.(*ast.RecordType); ok {
					collectRecordDeclFromAST(recType, &result, recordsSeen)
				}
			}
			variants = append(variants, variant)
		}
		result.ADTTypes[typeDecl.Name] = variants
	}

	return result
}

// collectNamedRecordAlias registers a named record type alias for Z3 encoding.
// For example: type TableCell = {text: string, colSpan: int, rowSpan: int, merged: bool}
// becomes: (declare-datatype TableCell ((mk_TableCell (colSpan Int) (merged Bool) (rowSpan Int) (text String))))
func collectNamedRecordAlias(name string, recType *ast.RecordType, result *adtExtractionResult, seen map[string]bool) {
	if seen[name] {
		return
	}
	seen[name] = true

	typeFields := make(map[string]types.Type, len(recType.Fields))
	for _, f := range recType.Fields {
		ft := convertASTTypeToType(f.Type)
		if ft == nil {
			continue
		}
		typeFields[f.Name] = ft
	}
	if len(typeFields) > 0 {
		result.RecordAliases[name] = &types.TRecord{
			Fields:   typeFields,
			TypeName: name,
		}
	}

	// Also check if any field is itself an inline record type that needs declaration
	for _, f := range recType.Fields {
		if innerRec, ok := f.Type.(*ast.RecordType); ok {
			collectRecordDeclFromAST(innerRec, result, seen)
		}
	}
}

// collectRecordDeclFromAST generates a declare-datatype for a record type
// found in an ADT constructor field, so it's declared before the ADT.
func collectRecordDeclFromAST(recType *ast.RecordType, result *adtExtractionResult, seen map[string]bool) {
	rec := convertASTTypeToType(recType)
	if rec == nil {
		return
	}
	trec, ok := rec.(*types.TRecord)
	if !ok {
		return
	}
	sortName := smt.MapRecordSortName(trec)
	if seen[sortName] {
		return
	}
	seen[sortName] = true

	// Build field sorts
	fields := make(map[string]string)
	for name, fieldType := range trec.Fields {
		sort, err := smt.MapType(fieldType)
		if err != nil {
			continue // Skip unencodable fields
		}
		fields[name] = sort
	}
	if len(fields) > 0 {
		result.RecordDecls = append(result.RecordDecls, smt.DeclareRecordDatatype(sortName, fields))
	}
}

// astTypeToSMTSort converts an AST type annotation to an SMT-LIB sort name.
func astTypeToSMTSort(t ast.Type) string {
	switch ty := t.(type) {
	case *ast.SimpleType:
		switch ty.Name {
		case "int":
			return "Int"
		case "float":
			return "Real"
		case "bool":
			return "Bool"
		case "string":
			return "String"
		default:
			return ty.Name // ADT type name
		}
	case *ast.ListType:
		elemSort := astTypeToSMTSort(ty.Element)
		return fmt.Sprintf("(Seq %s)", elemSort)
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → (Seq Int)
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elemSort := astTypeToSMTSort(ty.Args[0])
			return fmt.Sprintf("(Seq %s)", elemSort)
		}
		return ty.Constructor // ADT type name
	case *ast.RecordType:
		rec := convertASTTypeToType(ty)
		if rec == nil {
			return "Int" // Fallback
		}
		trec, ok := rec.(*types.TRecord)
		if !ok {
			return "Int"
		}
		return smt.MapRecordSortName(trec)
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect SMT sorts.
		return astTypeToSMTSort(ty.Base)
	default:
		return "Int" // Fallback for complex types
	}
}

// collectMonomorphicTypeNames returns the set of type names (ADTs and record
// aliases) declared WITHOUT type parameters across the given files. These are the
// only user types that can be emitted as concrete SMT sorts. Parametric types
// (e.g. Option[a], Result[e,a]) are excluded: the encoder cannot monomorphize them,
// so a signature that mentions them must be gated rather than encoded.
func collectMonomorphicTypeNames(files []*ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, f := range files {
		if f == nil {
			continue
		}
		for _, decl := range f.Decls {
			if td, ok := decl.(*ast.TypeDecl); ok && len(td.TypeParams) == 0 {
				names[td.Name] = true
			}
		}
	}
	return names
}

// astTypeEncodable reports whether an AST type can appear in a callee signature
// that the SMT encoder can handle. Primitives, lists/sequences, records, and
// monomorphic ADTs (declared in `declarable`) are encodable. A parametric ADT
// application (TypeApp with args, e.g. Option[float]) or a bare type variable is
// NOT — those are exactly the shapes that leak an undeclared sort into Z3.
// A nil type (no annotation) is treated as encodable — we only gate on what we can see.
func astTypeEncodable(t ast.Type, declarable map[string]bool) bool {
	switch ty := t.(type) {
	case nil:
		return true
	case *ast.SimpleType:
		switch ty.Name {
		case "int", "float", "bool", "string":
			return true
		default:
			return declarable[ty.Name] // monomorphic user ADT/record
		}
	case *ast.ListType:
		return astTypeEncodable(ty.Element, declarable)
	case *ast.TypeApp:
		// list[T] is the one encodable type application (maps to (Seq T)).
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			return astTypeEncodable(ty.Args[0], declarable)
		}
		return false // parametric ADT application (Option[float], Result[e,a], ...)
	case *ast.RecordType:
		return true // records map to declared record datatypes
	case *ast.LabelledType:
		return astTypeEncodable(ty.Base, declarable)
	case *ast.TypeVar:
		return false // unbound type variable in a signature
	default:
		return false // tuples, function types, and other shapes are not encodable
	}
}

// describeASTType renders an AST type for diagnostic messages.
func describeASTType(t ast.Type) string {
	switch ty := t.(type) {
	case nil:
		return "?"
	case *ast.SimpleType:
		return ty.Name
	case *ast.ListType:
		return "[" + describeASTType(ty.Element) + "]"
	case *ast.TypeApp:
		parts := make([]string, len(ty.Args))
		for i, a := range ty.Args {
			parts[i] = describeASTType(a)
		}
		return ty.Constructor + "[" + strings.Join(parts, ", ") + "]"
	case *ast.RecordType:
		return "{...}"
	case *ast.LabelledType:
		return describeASTType(ty.Base)
	case *ast.TypeVar:
		return ty.Name
	default:
		return fmt.Sprintf("%T", t)
	}
}

// firstUnencodableCalleeType walks the cross-function call graph reachable from a
// contracted function's body and returns the first callee whose signature (a
// parameter type or the return type) is not SMT-encodable — e.g. a parametric ADT
// like Option[float]. Returns (calleeName, renderedType), or ("","") if all clean.
//
// This closes a gap in the fragment gate: the smt-side checks only inspect
// $builtin/stdlib call names, never the signature TYPES of a user cross-function
// callee. Without this, such a callee leaks an undeclared sort into the SMT script
// and Z3 hard-errors instead of the verifier skipping with UNENCODABLE_TYPE.
func firstUnencodableCalleeType(
	funcName string,
	body core.CoreExpr,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
	declarable map[string]bool,
) (string, string) {
	for _, name := range smt.CollectCalleeNames(body, funcName, prog, imported) {
		fd, ok := calleeASTFuncs[name]
		if !ok {
			continue
		}
		if !astTypeEncodable(fd.ReturnType, declarable) {
			return name, describeASTType(fd.ReturnType)
		}
		for _, p := range fd.Params {
			if !astTypeEncodable(p.Type, declarable) {
				return name, describeASTType(p.Type)
			}
		}
	}
	return "", ""
}

// findFunctionBody finds the body expression of a named function in the Core program.
func findFunctionBody(prog *core.Program, funcName string) core.CoreExpr {
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			for _, binding := range d.Bindings {
				if binding.Name == funcName {
					return binding.Value
				}
			}
		case *core.Let:
			if d.Name == funcName {
				return d.Value
			}
		}
	}
	return nil
}

// unwrapLambdaParams unwraps Lambda nodes from the Core body,
// extracting parameters with types from the Surface AST and returning the inner body.
func unwrapLambdaParams(
	funcName string,
	surfaceFuncs map[string]*ast.FuncDecl,
	body core.CoreExpr,
) ([]smt.FunctionParam, core.CoreExpr) {
	// Unwrap Lambda nodes to get param names and inner body
	var coreParamNames []string
	innerBody := body
	for {
		lam, ok := innerBody.(*core.Lambda)
		if !ok {
			break
		}
		coreParamNames = append(coreParamNames, lam.Params...)
		innerBody = lam.Body
	}

	// Build params using Surface AST types (most reliable)
	if fd, ok := surfaceFuncs[funcName]; ok && len(fd.Params) > 0 {
		params := make([]smt.FunctionParam, 0, len(fd.Params))
		for _, p := range fd.Params {
			// Skip unit params from zero-arg function desugaring (func f() → func f(_: ()))
			if isUnitParam(p) {
				continue
			}
			paramType := convertASTTypeToType(p.Type)
			if paramType != nil {
				params = append(params, smt.FunctionParam{
					Name: p.Name,
					Type: paramType,
				})
			}
		}
		return params, innerBody
	}

	// Fallback: use Core param names with Int default (skip unit/dummy params)
	params := make([]smt.FunctionParam, 0, len(coreParamNames))
	for _, name := range coreParamNames {
		if name == "_" {
			continue
		}
		params = append(params, smt.FunctionParam{
			Name: name,
			Type: &types.TCon{Name: "int"},
		})
	}
	return params, innerBody
}

// isUnitParam returns true if the param is a unit parameter from zero-arg desugaring.
// The parser desugars `func f()` to `func f(_: ())`, producing a param named "_" with unit type.
func isUnitParam(p *ast.Param) bool {
	if p.Name != "_" {
		return false
	}
	if st, ok := p.Type.(*ast.SimpleType); ok {
		return st.Name == "()"
	}
	return false
}

// convertASTTypeToType converts an AST type annotation to a types.Type.
func convertASTTypeToType(t ast.Type) types.Type {
	if t == nil {
		return nil
	}
	switch ty := t.(type) {
	case *ast.SimpleType:
		return &types.TCon{Name: ty.Name}
	case *ast.ListType:
		elem := convertASTTypeToType(ty.Element)
		if elem == nil {
			return nil
		}
		return &types.TList{Element: elem}
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → TList{Element: int}
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elem := convertASTTypeToType(ty.Args[0])
			if elem == nil {
				return nil
			}
			return &types.TList{Element: elem}
		}
		// Other TypeApps (e.g., Option[int]) → TApp
		if len(ty.Args) > 0 {
			args := make([]types.Type, len(ty.Args))
			for i, a := range ty.Args {
				at := convertASTTypeToType(a)
				if at == nil {
					return nil
				}
				args[i] = at
			}
			return &types.TApp{
				Constructor: &types.TCon{Name: ty.Constructor},
				Args:        args,
			}
		}
		return &types.TCon{Name: ty.Constructor}
	case *ast.RecordType:
		fields := make(map[string]types.Type, len(ty.Fields))
		for _, f := range ty.Fields {
			ft := convertASTTypeToType(f.Type)
			if ft == nil {
				return nil
			}
			fields[f.Name] = ft
		}
		return &types.TRecord{Fields: fields}
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect type structure.
		return convertASTTypeToType(ty.Base)
	default:
		return nil
	}
}
