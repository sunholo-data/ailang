package smt

// verify.go — the `ailang verify` driver as a library: given a compiled
// program (Core + surface AST + loaded modules), encode every contracted
// function and solve it with Z3. The CLI (cmd/ailang/verify.go) only parses
// flags, calls Verify, and prints the report. Moved out of cmd/ailang by
// M-V1-SIMPLIFY-S2 M3 so an embedded or WASM caller can reach it.

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// VerifyResult holds the result of verifying a single function.
type VerifyResult struct {
	Function     string               `json:"function"`
	Status       string               `json:"status"` // verified, counterexample, skipped, error, unknown, uncontracted
	Reason       string               `json:"reason,omitempty"`
	Model        []ModelBinding       `json:"model,omitempty"`
	Rejections   []SMTRejectionReason `json:"rejections,omitempty"`
	Duration     time.Duration        `json:"duration,omitempty"`
	SMTLib       string               `json:"smtlib,omitempty"`
	BoundedDepth int                  `json:"bounded_depth,omitempty"`
}

// VerifyModule is one imported module as the verifier needs it: its surface
// AST (for type declarations and signatures) and its Core program (for
// cross-module callee inlining). Either may be nil.
type VerifyModule struct {
	File *ast.File
	Core *core.Program
}

// VerifyOptions configures one Verify run.
type VerifyOptions struct {
	// Timeout is the per-function Z3 timeout.
	Timeout time.Duration
	// RecursiveDepth is the global bounded-recursion unrolling depth
	// (0 disables). A per-function @verify(depth: N) overrides it.
	RecursiveDepth int
	// Verbose attaches the generated SMT-LIB to each solved result.
	Verbose bool
	// UnsupportedConstructIsError reports a function whose body uses a
	// construct outside the decidable fragment (ErrUnsupportedConstruct, e.g.
	// a match on list patterns) as Status "error" — counted in Errors — instead
	// of the honest "skipped" (#757) that `ailang verify` reports. `ai-check`
	// sets it: its exit-code contract (cmd/ailang/ai_check_exit_test.go) has
	// always surfaced that case as a verifier error, and changing what the
	// eval harness banks as verify_errors is a ruling, not a refactor
	// (M-V1-SIMPLIFY-S4 M3A). Both modes name the construct in the reason.
	UnsupportedConstructIsError bool
}

// VerifyReport is the outcome of Verify over one program.
type VerifyReport struct {
	Results        []VerifyResult
	Verified       int
	Counterexample int
	Skipped        int
	Errors         int
	// Uncontracted counts exported functions that carry no contract at all;
	// they appear in Results with Status "uncontracted".
	Uncontracted int
}

// Verify runs static contract verification over every contracted function of
// coreProg. modules are the imported modules keyed by module path; their
// types and functions extend the encoding inputs. Z3 must be available
// (Z3Available) — the caller checks that up front so it can say how to
// install it.
func Verify(coreProg *core.Program, surfaceAST *ast.File, modules map[string]VerifyModule, opts VerifyOptions) *VerifyReport {
	report := &VerifyReport{}

	// Extract ADT types, record type aliases, and inline record types from the Surface AST.
	adtResult := ExtractADTTypesWithRecords(surfaceAST)
	adtTypes := adtResult.ADTTypes
	adtRecordDecls := adtResult.RecordDecls
	recordAliases := adtResult.RecordAliases
	// Also extract from imported modules so cross-module types
	// (e.g., Block, XmlNode, TableCell) get declare-datatype in the Z3 output.
	for _, mod := range modules {
		if mod.File != nil {
			modResult := ExtractADTTypesWithRecords(mod.File)
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

	// Build Surface AST function lookup for param extraction
	surfaceFuncs := make(map[string]*ast.FuncDecl)
	for _, f := range surfaceAST.Funcs {
		surfaceFuncs[f.Name] = f
	}

	// Solver configuration
	solverCfg := SolverConfig{
		Timeout: opts.Timeout,
	}

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
	allSurfaceParams := make(map[string][]FunctionParam)
	allSurfaceReturnSorts := make(map[string]string)
	for funcName, fd := range surfaceFuncs {
		var params []FunctionParam
		for _, p := range fd.Params {
			paramType := ConvertASTTypeToType(p.Type)
			if paramType != nil {
				params = append(params, FunctionParam{Name: p.Name, Type: paramType})
			}
		}
		allSurfaceParams[funcName] = params
		if fd.ReturnType != nil {
			allSurfaceReturnSorts[funcName] = ASTTypeToSMTSort(fd.ReturnType)
		}
	}

	// Build imported programs map and extend surface params/sorts from imported modules.
	importedPrograms := make(map[string]*core.Program)
	for modPath, mod := range modules {
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
				var params []FunctionParam
				for _, p := range fd.Params {
					if pt := ConvertASTTypeToType(p.Type); pt != nil {
						params = append(params, FunctionParam{Name: p.Name, Type: pt})
					}
				}
				allSurfaceParams[fd.Name] = params
				if fd.ReturnType != nil {
					allSurfaceReturnSorts[fd.Name] = ASTTypeToSMTSort(fd.ReturnType)
				}
			}
		}
	}

	encOpts := EncodeFunctionOpts{
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
	for _, mod := range modules {
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
	// declarableADTs: monomorphic (non-parametric) ADT/record names that WILL be declared
	// as SMT sorts. Parametric ADTs (those whose TypeDecl carries type parameters) are
	// excluded — they cannot be monomorphized by the current encoder.
	declarableADTs := CollectMonomorphicTypeNames(allTypeFiles)

	// Process each function with contracts
	for funcName, meta := range coreProg.Meta {
		if len(meta.Contracts) == 0 {
			continue
		}

		// Find the function body in Core decls
		body := FindFunctionBody(coreProg, funcName)
		if body == nil {
			report.Results = append(report.Results, VerifyResult{
				Function: funcName,
				Status:   "skipped",
				Reason:   "function body not found in Core AST",
			})
			report.Skipped++
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
			report.Results = append(report.Results, VerifyResult{
				Function: funcName,
				Status:   "skipped",
				Reason:   "no ensures clause (nothing to verify)",
			})
			report.Skipped++
			continue
		}

		// M4: Determine effective recursive depth for this function
		// Per-function @verify(depth: N) overrides global --verify-recursive-depth
		effectiveDepth := opts.RecursiveDepth
		if meta.VerifyDepth > 0 {
			effectiveDepth = meta.VerifyDepth
		}

		// Check if function is in the decidable SMT fragment
		encodable, rejections := IsSMTEncodable(funcName, meta, body)
		// Callee-sort gate: reject cleanly if a cross-function callee has an
		// unencodable signature type (e.g. Option[float]) rather than leaking an
		// undeclared sort into the SMT script and crashing Z3. See M-SMT-CALLEE-SORT-GATE.
		if callee, badType := FirstUnencodableCalleeType(funcName, body, coreProg, importedPrograms, calleeASTFuncs, declarableADTs); callee != "" {
			rejections = append(rejections, SMTRejectionReason{
				Code:    RejectUnencodable,
				Message: fmt.Sprintf("Function %q calls %q whose signature uses an unencodable type %q", funcName, callee, badType),
				Hint:    "Cross-function verification cannot encode parametric ADTs (Option/Result) or type variables in a callee signature. Callees returning records/enum ADTs are also not yet inlinable and will skip. Rewrite the callee to return a primitive, or inline its logic into the caller.",
			})
			encodable = false
		}
		if !encodable {
			// If bounded recursion is enabled, filter out RejectRecursive
			if effectiveDepth > 0 {
				var filtered []SMTRejectionReason
				for _, r := range rejections {
					if r.Code != RejectRecursive {
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
				report.Results = append(report.Results, VerifyResult{
					Function:   funcName,
					Status:     "skipped",
					Reason:     strings.Join(reasons, "; "),
					Rejections: rejections,
				})
				report.Skipped++
				continue
			}
		}

		// Unwrap Lambda nodes to separate params from body
		params, innerBody := UnwrapLambdaParams(funcName, surfaceFuncs, body)

		// Determine return sort from Surface AST
		returnSort := ""
		var returnType types.Type
		if fd, ok := surfaceFuncs[funcName]; ok && fd.ReturnType != nil {
			returnSort = ASTTypeToSMTSort(fd.ReturnType)
			returnType = ConvertASTTypeToType(fd.ReturnType)
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
		funcADTTypes, funcAliases, funcExtraDecls :=
			FilterSMTInputsForFunction(params, returnSort, innerBody, adtTypes, recordAliases, adtRecordDecls)
		funcEncOpts.RecordTypeAliases = funcAliases
		funcEncOpts.ExtraDeclarations = funcExtraDecls

		// Encode function to SMT-LIB (with cross-function call support)
		encResult, err := EncodeFunction(funcName, params, innerBody, returnSort, meta, funcADTTypes, funcEncOpts)
		if err != nil {
			// If the error is due to unresolvable cross-module types,
			// skip gracefully instead of reporting as error
			if errors.Is(err, ErrUnresolvableTypes) {
				report.Results = append(report.Results, UnresolvedTypeVerifyResult(funcName, err))
				report.Skipped++
				continue
			}
			// #757: an unsupported Core construct (e.g. match on list patterns)
			// is an honest capability boundary of the decidable fragment — skip
			// with the reason, never a hard ERROR leaking Go type names. Unless
			// the caller opted into the error lane (ai-check's contract), in
			// which case it falls through to the generic encoding error below.
			if errors.Is(err, ErrUnsupportedConstruct) && !opts.UnsupportedConstructIsError {
				report.Results = append(report.Results, VerifyResult{
					Function: funcName,
					Status:   "skipped",
					Reason:   fmt.Sprintf("unsupported construct in SMT encoding: %v", err),
					Rejections: []SMTRejectionReason{{
						Code:    RejectUnencodable,
						Message: fmt.Sprintf("Function %q uses a construct the SMT encoder cannot represent", funcName),
						Hint:    "Match on list patterns (x :: rest), string patterns, and similar unsupported constructs are outside the decodable fragment. Rewrite using length/element access on lists, or verify a version without the unsupported construct.",
					}},
				})
				report.Skipped++
				continue
			}
			report.Results = append(report.Results, VerifyResult{
				Function: funcName,
				Status:   "error",
				Reason:   fmt.Sprintf("encoding error: %v", err),
			})
			report.Errors++
			continue
		}

		// Solve with Z3
		solveResult, err := Solve(encResult.SMTLib, solverCfg)
		if err != nil {
			report.Results = append(report.Results, VerifyResult{
				Function: funcName,
				Status:   "error",
				Reason:   fmt.Sprintf("solver error: %v", err),
			})
			report.Errors++
			continue
		}

		vr := VerifyResult{
			Function: funcName,
			Duration: solveResult.Duration,
		}
		// Mark bounded recursion depth if the function is recursive
		if effectiveDepth > 0 && IsRecursiveFunc(innerBody, funcName) {
			vr.BoundedDepth = effectiveDepth
		}
		if opts.Verbose {
			vr.SMTLib = encResult.SMTLib
		}

		switch solveResult.Status {
		case StatusVerified:
			vr.Status = "verified"
			report.Verified++
		case StatusCounterexample:
			vr.Status = "counterexample"
			vr.Model = solveResult.Model
			report.Counterexample++
		case StatusUnknown:
			vr.Status = "unknown"
			vr.Reason = solveResult.Error
			report.Errors++
		case StatusError:
			vr.Status = "error"
			vr.Reason = solveResult.Error
			report.Errors++
		}

		report.Results = append(report.Results, vr)
	}

	// Account for exported functions that carry no contract at all.
	//
	// These never enter the loop above (`len(meta.Contracts) == 0` skips them),
	// so before this they were absent from the denominator entirely: a module
	// with 25 exports of which 11 had contracts reported "11 functions: 11
	// verified" and said nothing about the other 14. The teaching prompt asks
	// agents to maximise the surface area of verified code, which is a ratio
	// the tool has to be willing to print the bottom half of.
	for _, fd := range surfaceAST.Funcs {
		if !fd.IsExport {
			continue
		}
		meta, ok := coreProg.Meta[fd.Name]
		if ok && len(meta.Contracts) > 0 {
			continue
		}
		report.Results = append(report.Results, VerifyResult{
			Function: fd.Name,
			Status:   "uncontracted",
			Reason:   "no contract annotations (not a verification candidate)",
		})
		report.Uncontracted++
	}

	return report
}
