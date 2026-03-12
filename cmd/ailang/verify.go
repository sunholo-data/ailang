package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/smt"
	"github.com/sunholo/ailang/internal/types"
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

	// Extract ADT types from the Surface AST
	adtTypes := extractADTTypes(surfaceAST)

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

	encOpts := smt.EncodeFunctionOpts{
		Program:            coreProg,
		SurfaceParams:      allSurfaceParams,
		SurfaceReturnSorts: allSurfaceReturnSorts,
	}

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

		// Encode function to SMT-LIB (with cross-function call support)
		encResult, err := smt.EncodeFunction(funcName, params, innerBody, returnSort, meta, adtTypes, funcEncOpts)
		if err != nil {
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

// extractADTTypes extracts ADT type definitions from the Surface AST
// and converts them to SMT ADTVariant format for the encoder.
func extractADTTypes(file *ast.File) map[string][]smt.ADTVariant {
	adtTypes := make(map[string][]smt.ADTVariant)

	for _, decl := range file.Decls {
		typeDecl, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}
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
			}
			variants = append(variants, variant)
		}
		adtTypes[typeDecl.Name] = variants
	}

	return adtTypes
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
	default:
		return "Int" // Fallback for complex types
	}
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
	default:
		return nil
	}
}

// printVerifyHuman prints human-readable verification results.
func printVerifyHuman(results []verifyResult, filename string, verified, counterexample, skipped, errCount int, verbose bool) {
	total := verified + counterexample + skipped + errCount

	fmt.Printf("\n%s Verifying contracts in %s\n", cyan("→"), filename)
	if z3ver := smt.Z3Version(); z3ver != "" {
		fmt.Printf("  Solver: %s\n", z3ver)
	}
	fmt.Println()

	hasBounded := false
	for _, r := range results {
		switch r.Status {
		case "verified":
			if r.BoundedDepth > 0 {
				fmt.Printf("  %s %s  %s\n", green(fmt.Sprintf("✓ VERIFIED (bounded: depth %d)", r.BoundedDepth)), bold(r.Function), dim(r.Duration.String()))
				hasBounded = true
			} else {
				fmt.Printf("  %s %s  %s\n", green("✓ VERIFIED"), bold(r.Function), dim(r.Duration.String()))
			}
		case "counterexample":
			fmt.Printf("  %s %s\n", red("✗ VIOLATION"), bold(r.Function))
			if len(r.Model) > 0 {
				fmt.Printf("    Counterexample:\n")
				for _, b := range r.Model {
					fmt.Printf("      %s: %s = %s\n", b.Name, b.Sort, b.Value)
				}
			}
		case "skipped":
			fmt.Printf("  %s %s\n", yellow("⚠ SKIPPED"), bold(r.Function))
			if r.Reason != "" {
				fmt.Printf("    Reason: %s\n", r.Reason)
			}
			if len(r.Rejections) > 0 {
				for _, rej := range r.Rejections {
					if rej.Hint != "" {
						fmt.Printf("    Hint: %s\n", rej.Hint)
					}
				}
			}
		case "error", "unknown":
			fmt.Printf("  %s %s\n", red("! ERROR"), bold(r.Function))
			if r.Reason != "" {
				fmt.Printf("    %s\n", r.Reason)
			}
		}

		if verbose && r.SMTLib != "" {
			fmt.Printf("    SMT-LIB:\n")
			for _, line := range strings.Split(r.SMTLib, "\n") {
				if line != "" {
					fmt.Printf("      %s\n", line)
				}
			}
			fmt.Println()
		}
	}

	// Summary line
	fmt.Println()
	summary := fmt.Sprintf("  %d functions: ", total)
	parts := []string{}
	if verified > 0 {
		parts = append(parts, green(fmt.Sprintf("%d verified", verified)))
	}
	if counterexample > 0 {
		parts = append(parts, red(fmt.Sprintf("%d violations", counterexample)))
	}
	if skipped > 0 {
		parts = append(parts, yellow(fmt.Sprintf("%d skipped", skipped)))
	}
	if errCount > 0 {
		parts = append(parts, red(fmt.Sprintf("%d errors", errCount)))
	}
	if len(parts) == 0 {
		parts = append(parts, "no functions with contracts")
	}
	fmt.Printf("%s%s\n", summary, strings.Join(parts, ", "))

	if hasBounded {
		fmt.Printf("\n  %s \"bounded: depth N\" means the property was verified assuming at most N\n", dim("Note:"))
		fmt.Printf("  %s levels of recursion. This is sound but not a full inductive proof.\n", dim("      "))
	}
	fmt.Println()
}

// printVerifyJSON outputs verification results as JSON.
func printVerifyJSON(results []verifyResult, filename string, verified, counterexample, skipped, errCount int) {
	output := struct {
		File           string         `json:"file"`
		Verified       int            `json:"verified"`
		Counterexample int            `json:"counterexample"`
		Skipped        int            `json:"skipped"`
		Errors         int            `json:"errors"`
		Results        []verifyResult `json:"results"`
	}{
		File:           filename,
		Verified:       verified,
		Counterexample: counterexample,
		Skipped:        skipped,
		Errors:         errCount,
		Results:        results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: JSON encoding error: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
