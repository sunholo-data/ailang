package smt

// verify.go — the `ailang verify` driver as a library: given a compiled
// program (Core + surface AST + loaded modules), encode every contracted
// function and solve it with Z3. The CLI (cmd/ailang/verify.go) only parses
// flags, calls Verify, and prints the report. Moved out of cmd/ailang by
// M-V1-SIMPLIFY-S2 M3 so an embedded or WASM caller can reach it.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/verification"
)

// VerifyResult holds the result of verifying a single function.
type VerifyResult struct {
	ReasonCode   verification.ReasonCode `json:"reason_code,omitempty"`
	Function     string                  `json:"function"`
	Status       string                  `json:"status"` // verified, counterexample, inconclusive, skipped, error, unknown, uncontracted
	Reason       string                  `json:"reason,omitempty"`
	Model        []ModelBinding          `json:"model,omitempty"`
	Rejections   []SMTRejectionReason    `json:"rejections,omitempty"`
	Duration     time.Duration           `json:"duration,omitempty"`
	SMTLib       string                  `json:"smtlib,omitempty"`
	BoundedDepth int                     `json:"bounded_depth,omitempty"`
}

// VerifyModule is one imported module as the verifier needs it: its surface
// AST (for type declarations and signatures) and its Core program (for
// cross-module callee inlining). Either may be nil.
type VerifyModule struct {
	CoreTI types.CoreTypeInfo
	File   *ast.File
	Core   *core.Program
}

// VerifyOptions configures one Verify run.
type VerifyOptions struct {
	CoreTI types.CoreTypeInfo
	// Timeout is the per-function Z3 timeout.
	Timeout time.Duration
	// RecursiveDepth is the global bounded-recursion unrolling depth
	// (0 disables). A per-function @verify(depth: N) overrides it.
	RecursiveDepth int
	// Verbose attaches the generated SMT-LIB to each solved result.
	Verbose bool
}

// VerifyReport is the outcome of Verify over one program.
type VerifyReport struct {
	verification.Evidence
	Results        []VerifyResult
	Verified       int
	Counterexample int
	// Inconclusive counts obligations attempted but undecided: a sat found
	// under bounded recursion unrolling (see solverVerifyResult). Not a
	// violation, not a proof.
	Inconclusive int
	Skipped      int
	Errors       int
	// Uncontracted counts exported functions that carry no contract at all;
	// they appear in Results with Status "uncontracted".
	Uncontracted int
}

// Obligations is the number of proof obligations the run examined: every
// result that is neither "uncontracted" nor a requires-only function. It
// equals CountObligations of the same program (INV: the verify loop reports
// every obligation CountObligations counts exactly once).
func (r *VerifyReport) Obligations() int {
	return TallyResults(r.Results).Obligations
}

// Tally projects the report onto the shared exit contract.
func (r *VerifyReport) Tally() verification.Tally {
	return TallyResults(r.Results)
}

// IsObligation reports whether a result row is a proof obligation. Two kinds
// of row are not: an exported function with no contract ("uncontracted"), and
// a requires-only function (skipped, reason_code missing_postcondition). The
// verifier assumes a precondition while proving the function's own ensures and
// does not check it at call sites, so a precondition alone gives it nothing to
// prove. Such a row stays in the report as evidence and still counts in the
// legacy `skipped` counter, but never in the exit tally.
func IsObligation(r VerifyResult) bool {
	if r.Status == "uncontracted" {
		return false
	}
	return !(r.Status == "skipped" && r.ReasonCode == verification.MissingPostcondition)
}

// TallyResults is the exit-contract tally of a solver run, computed from the
// result rows so every command derives it the same way (the CLI JSON report
// carries the same rows).
func TallyResults(results []VerifyResult) verification.Tally {
	t := verification.Tally{CheckPassed: true, SolverAvailable: true}
	for _, r := range results {
		if !IsObligation(r) {
			continue
		}
		t.Obligations++
		switch r.Status {
		case "verified":
			t.Verified++
		case "counterexample":
			t.Counterexample++
		case "inconclusive":
			t.Inconclusive++
		case "skipped":
			t.Skipped++
		default: // "error", "unknown"
			t.Errors++
		}
	}
	return t
}

// hasObligation is the ONE predicate for "this function carries a proof
// obligation": at least one ensures clause. CountObligations uses it; the
// verify loop reports every such function exactly once.
func hasObligation(meta *core.DeclMeta) bool {
	if meta == nil {
		return false
	}
	for _, c := range meta.Contracts {
		if c != nil && c.Kind == core.EnsuresKind {
			return true
		}
	}
	return false
}

// CountObligations counts the proof obligations (functions with an ensures
// clause) of a compiled program without a solver. It is how a verification
// command knows, with Z3 missing, whether it has dropped anything
// (M-V1-VERIFICATION-HONESTY M1).
func CountObligations(coreProg *core.Program) int {
	if coreProg == nil {
		return 0
	}
	n := 0
	for _, meta := range coreProg.Meta {
		if hasObligation(meta) {
			n++
		}
	}
	return n
}

// Verify runs static contract verification over every contracted function of
// coreProg. modules are the imported modules keyed by module path; their
// types and functions extend the encoding inputs. Z3 must be available
// (Z3Available) — the caller checks that up front so it can say how to
// install it.
func Verify(coreProg *core.Program, surfaceAST *ast.File, modules map[string]VerifyModule, opts VerifyOptions) *VerifyReport {
	report := &VerifyReport{Evidence: verification.EncodedEvidence()}

	// Extract ADT types, record type aliases, and inline record types from the Surface AST.
	adtResult := ExtractADTTypesWithRecords(surfaceAST)
	adtTypes := adtResult.ADTTypes
	adtParams := adtResult.TypeParams
	ambiguous := make(map[string]bool)
	typeOwner := make(map[string]string)
	owner := surfaceAST.Path
	if surfaceAST.Module != nil {
		owner = surfaceAST.Module.Path
	}
	for name := range adtTypes {
		typeOwner[name] = owner
	}
	adtRecordDecls := adtResult.RecordDecls
	recordAliases := adtResult.RecordAliases
	// Also extract from imported modules so cross-module types
	// (e.g., Block, XmlNode, TableCell) get declare-datatype in the Z3 output.
	for _, mod := range modules {
		if mod.File != nil {
			modResult := ExtractADTTypesWithRecords(mod.File)
			modOwner := mod.File.Path
			if mod.File.Module != nil {
				modOwner = mod.File.Module.Path
			}
			for name, variants := range modResult.ADTTypes {
				if previous, exists := typeOwner[name]; exists && previous != modOwner {
					ambiguous[name] = true
					for _, v := range variants {
						ambiguous[v.Name] = true
					}
					for _, v := range adtTypes[name] {
						ambiguous[v.Name] = true
					}
				}
				if _, exists := adtTypes[name]; !exists {
					typeOwner[name] = modOwner
					adtTypes[name] = variants
					adtParams[name] = modResult.TypeParams[name]
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

	// Keep SMT admission separate from purity metadata. Key by metadata identity
	// so imported functions with the same name retain their own declaration.
	declarations := make(map[*core.DeclMeta]*ast.FuncDecl)
	for name, fd := range surfaceFuncs {
		if meta := coreProg.Meta[name]; meta != nil {
			declarations[meta] = fd
		}
	}

	// Build surface params and return sorts for all functions (for cross-function call resolution)
	allSurfaceParams := make(map[string][]FunctionParam)
	allSurfaceReturnSorts := make(map[string]string)
	for funcName, fd := range surfaceFuncs {
		allSurfaceParams[funcName] = SurfaceFunctionParams(fd)
		if fd.ReturnType != nil {
			allSurfaceReturnSorts[funcName] = ASTTypeToSMTSort(fd.ReturnType)
		}
	}

	// Build imported programs map and extend surface params/sorts from imported modules.
	importedPrograms := make(map[string]*core.Program)
	for modPath, mod := range modules {
		if mod.Core != nil {
			importedPrograms[modPath] = mod.Core
			if mod.File != nil {
				for _, fd := range mod.File.Funcs {
					if meta := mod.Core.Meta[fd.Name]; meta != nil {
						declarations[meta] = fd
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
				allSurfaceParams[fd.Name] = SurfaceFunctionParams(fd)
				if fd.ReturnType != nil {
					allSurfaceReturnSorts[fd.Name] = ASTTypeToSMTSort(fd.ReturnType)
				}
			}
		}
	}

	importedTI := make(map[string]types.CoreTypeInfo)
	for path, mod := range modules {
		importedTI[path] = mod.CoreTI
	}
	encOpts := EncodeFunctionOpts{
		CoreTI:              opts.CoreTI,
		ImportedCoreTI:      importedTI,
		ADTTypeParams:       adtParams,
		SurfaceDeclarations: declarations,
		Program:             coreProg,
		SurfaceParams:       allSurfaceParams,
		SurfaceReturnSorts:  allSurfaceReturnSorts,
		ExtraDeclarations:   adtRecordDecls,
		RecordTypeAliases:   recordAliases,
		ImportedPrograms:    importedPrograms,
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
		if meta == nil || len(meta.Contracts) == 0 {
			continue
		}

		// A requires-only function carries no proof obligation (hasObligation):
		// it is reported as evidence, with the reason code IsObligation keys on,
		// and never reaches the body or solver steps. Checked before the body
		// lookup so a missing body cannot turn it into an obligation skip.
		if !hasObligation(meta) {
			report.Results = append(report.Results, VerifyResult{
				Function:   funcName,
				Status:     "skipped",
				Reason:     "no ensures clause (nothing to verify)",
				ReasonCode: verification.MissingPostcondition,
			})
			report.Skipped++
			continue
		}

		// Find the function body in Core decls
		body := FindFunctionBody(coreProg, funcName)
		if body == nil {
			report.Results = append(report.Results, VerifyResult{
				Function:   funcName,
				Status:     "skipped",
				Reason:     "function body not found in Core AST",
				ReasonCode: verification.MissingBody,
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
		encodable, rejections := IsSMTEncodable(funcName, meta, body, surfaceFuncs[funcName])
		if opts.CoreTI != nil {
			filtered := rejections[:0]
			for _, r := range rejections {
				if r.Code != RejectDeepPatterns {
					filtered = append(filtered, r)
				}
			}
			rejections = filtered
			encodable = len(rejections) == 0
		}
		// Callee-sort gate: reject cleanly if a cross-function callee has an
		// unencodable signature type (e.g. Option[float]) rather than leaking an
		// undeclared sort into the SMT script and crashing Z3. See M-SMT-CALLEE-SORT-GATE.
		// Contract predicates are gated too: their callees are resolved as define-funs
		// (ResolveCalleesFromRoots), so an unencodable callee sort there would leak.
		if callee, badType := firstUnencodableCalleeInFunc(funcName, body, meta.Contracts, coreProg, importedPrograms, calleeASTFuncs, declarableADTs); callee != "" {
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
					ReasonCode: verification.UnsupportedEncoding,
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

		// Refuse unsupported/open signature sorts before body encoding or Z3.
		var signatureErr error
		for _, p := range params {
			if _, err := MapType(p.Type); err != nil {
				signatureErr = err
				break
			}
		}
		if returnType != nil {
			if sort, err := MapType(returnType); err != nil {
				signatureErr = err
			} else {
				returnSort = sort
			}
		}
		if signatureErr != nil {
			report.Results = append(report.Results, UnresolvedTypeVerifyResult(funcName, signatureErr))
			report.Skipped++
			continue
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
		extraRoots := make([]core.CoreExpr, 0, len(meta.Contracts))
		for _, c := range meta.Contracts {
			if c != nil {
				extraRoots = append(extraRoots, c.Expr)
			}
		}
		allRoots := append([]core.CoreExpr{innerBody}, extraRoots...)
		for _, root := range allRoots {
			for _, callee := range CollectCalleeNames(root, funcName, coreProg, importedPrograms) {
				calleeBody, _ := findFuncBodyInAnyProg(callee, coreProg, importedPrograms)
				extraRoots = append(extraRoots, calleeBody)
			}
		}
		seeds := buildNeededSortSet(params, returnSort, innerBody, adtTypes, recordAliases, adtRecordDecls, extraRoots...)
		var ambiguousName string
		ambiguousKeys := make([]string, 0, len(ambiguous))
		for name := range ambiguous {
			ambiguousKeys = append(ambiguousKeys, name)
		}
		sort.Strings(ambiguousKeys)
		for _, name := range ambiguousKeys {
			if seeds[name] {
				ambiguousName = name
				break
			}
		}
		if ambiguousName != "" {
			report.Results = append(report.Results, UnresolvedTypeVerifyResult(funcName, fmt.Errorf("ambiguous imported ADT identity %s", ambiguousName)))
			report.Skipped++
			continue
		}
		funcADTTypes, funcAliases, funcExtraDecls :=
			FilterSMTInputsForFunction(params, returnSort, innerBody, adtTypes, recordAliases, adtRecordDecls, extraRoots...)
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
			// with the reason, never a hard ERROR leaking Go type names. Every
			// command reports it this way: ai-check's former error lane was
			// retired with the shared exit contract (M-V1-VERIFICATION-HONESTY
			// M1), under which a skip passes loudly and --strict fails it.
			if errors.Is(err, ErrUnsupportedConstruct) {
				report.Results = append(report.Results, VerifyResult{
					Function:   funcName,
					Status:     "skipped",
					Reason:     fmt.Sprintf("unsupported construct in SMT encoding: %v", err),
					ReasonCode: verification.UnsupportedEncoding,
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
				Function:   funcName,
				Status:     "error",
				Reason:     fmt.Sprintf("encoding error: %v", err),
				ReasonCode: encodingErrorReason(err),
			})
			report.Errors++
			continue
		}

		// Solve with Z3
		solveResult, err := Solve(encResult.SMTLib, solverCfg)
		if err != nil {
			report.Results = append(report.Results, VerifyResult{
				Function:   funcName,
				Status:     "error",
				Reason:     fmt.Sprintf("solver error: %v", err),
				ReasonCode: verification.VerifierError,
			})
			report.Errors++
			continue
		}

		// The obligation is bounded-unrolled when the function recurses and
		// unrolling is on: its self-calls bottom out in an uninterpreted stub.
		boundedDepth := 0
		if effectiveDepth > 0 && IsRecursiveFunc(innerBody, funcName) {
			boundedDepth = effectiveDepth
		}
		vr := solverVerifyResult(funcName, solveResult, boundedDepth, opts.Verbose)
		if opts.Verbose {
			vr.SMTLib = encResult.SMTLib
		}
		report.tally(vr)
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
			Function:   fd.Name,
			Status:     "uncontracted",
			Reason:     "no contract annotations (not a verification candidate)",
			ReasonCode: verification.Uncontracted,
		})
		report.Uncontracted++
	}

	return report
}

// tally appends one result and moves exactly the counter its status names, so
// the counters partition Results (INV-1). An unrecognised status is a
// programming error in this file — it panics rather than silently dropping out
// of every counter (TestSolveOutcomeExhaustive pins the mapping).
func (r *VerifyReport) tally(vr VerifyResult) {
	switch vr.Status {
	case "verified":
		r.Verified++
	case "counterexample":
		r.Counterexample++
	case "inconclusive":
		r.Inconclusive++
	case "skipped":
		r.Skipped++
	case "error", "unknown":
		r.Errors++
	case "uncontracted":
		r.Uncontracted++
	default:
		panic(fmt.Sprintf("smt: verify status %q has no report counter", vr.Status))
	}
	r.Results = append(r.Results, vr)
}

// solverVerifyResult is the ONE place a solver outcome becomes a verify
// status; `ailang verify` and `ailang ai-check` both reach it through Verify.
// It attaches solver-owned reasons and does not classify a timeout from human
// diagnostic text.
//
// boundedDepth > 0 means the obligation was bounded-unrolled: the function's
// self-calls were expanded boundedDepth levels deep and bottom out in an
// UNINTERPRETED stub <name>_0 (unroll.go). That encoding is asymmetric:
//
//   - unsat is sound — the property holds under every interpretation of the
//     stub, the real function included — so it stays "verified" (with the
//     bounded caveat carried in BoundedDepth).
//   - sat is NOT a violation — Z3 may satisfy the negated postcondition by
//     inventing a value for the stub, an execution no real run can produce.
//     It is reported "inconclusive". The decision is structural (no model
//     inspection: a solver may omit interpretations it deems irrelevant, so
//     "no frontier symbol in the model" proves nothing). The declared cost is
//     that a GENUINE bug on a recursive function also reads inconclusive
//     until witness replay exists — "could not decide", never a false
//     violation. The model is withheld unless verbose: it is usually a
//     let-expression over the stub, not an input.
//
// M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE.
func solverVerifyResult(name string, result *SolverResult, boundedDepth int, verbose bool) VerifyResult {
	r := VerifyResult{Function: name, Status: result.Status.String(), Duration: result.Duration, ReasonCode: result.ReasonCode, BoundedDepth: boundedDepth}
	switch result.Status {
	case StatusCounterexample:
		if boundedDepth > 0 {
			r.Status = "inconclusive"
			r.ReasonCode = verification.BoundedInconclusive
			r.Reason = boundedInconclusiveReason(name, boundedDepth)
			if verbose {
				r.Model = result.Model
			}
			break
		}
		r.Model = result.Model
		r.ReasonCode = verification.Counterexample
	case StatusUnknown:
		r.Reason = result.Error
		if r.ReasonCode == "" {
			r.ReasonCode = verification.SolverUnknown
		}
	case StatusError:
		r.Reason = result.Error
		r.ReasonCode = verification.VerifierError
	}
	return r
}

// boundedInconclusiveReason explains an inconclusive outcome. The depth advice
// is deliberately conditional. When inputs are unbounded (any-length lists)
// raising the depth cannot decide the obligation: the candidate model just
// grows with it (measured: length 2 at depth 2, length 4 at depth 4), so
// unconditional depth-tuning advice is a treadmill. When a requires clause
// bounds the recursion (recursive_verify.ail: n <= 4) a depth past that bound
// DOES decide it (all three verify at depth 6), so "never" would be false too.
func boundedInconclusiveReason(name string, depth int) string {
	return fmt.Sprintf("bounded unrolling exhausted at depth %d: the solver's candidate model may depend on "+
		"the unrolling frontier (%s_0), so it is neither a proof nor a confirmed violation; "+
		"raising --verify-recursive-depth can decide this only if a requires clause bounds the recursion depth "+
		"of the inputs — for unbounded inputs it cannot", depth, name)
}

func encodingErrorReason(err error) verification.ReasonCode {
	if errors.Is(err, ErrUnsupportedConstruct) {
		return verification.UnsupportedEncoding
	}
	return verification.VerifierError
}
