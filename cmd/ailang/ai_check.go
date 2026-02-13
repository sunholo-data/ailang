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
)

// aiCheckOutput is the unified JSON output for ai-check
type aiCheckOutput struct {
	File   string          `json:"file"`
	Check  aiCheckSection  `json:"check"`
	Verify aiVerifySection `json:"verify"`
}

// aiCheckSection is the type-check portion of ai-check output
type aiCheckSection struct {
	Passed     bool             `json:"passed"`
	ErrorCount int              `json:"error_count"`
	Errors     []checkJSONError `json:"errors"`
}

// aiVerifySection is the contract verification portion of ai-check output
type aiVerifySection struct {
	Available      bool           `json:"available"`
	Verified       int            `json:"verified"`
	Counterexample int            `json:"counterexample"`
	Skipped        int            `json:"skipped"`
	Errors         int            `json:"errors"`
	Results        []verifyResult `json:"results"`
}

// aiCheckCommand implements the `ailang ai-check` CLI command.
// It runs type checking + contract verification in a single invocation
// with unified JSON output designed for AI/machine consumption.
func aiCheckCommand() {
	fs := flag.NewFlagSet("ai-check", flag.ExitOnError)
	timeoutFlag := fs.Duration("timeout", 5*time.Second, "Per-function Z3 timeout")
	recursiveDepthFlag := fs.Int("verify-recursive-depth", 2, "Bounded recursion unrolling depth (1-10, 0 to disable)")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang ai-check [options] <file.ail>")
		fmt.Println()
		fmt.Println("Runs type checking + contract verification in one pass.")
		fmt.Println("Always outputs JSON (designed for AI/machine consumption).")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --timeout                 Per-function Z3 timeout (default: 5s)")
		fmt.Println("  --verify-recursive-depth  Bounded recursion depth (default: 2)")
		fmt.Println("  --relax-modules           Relax MOD010 validation")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Suppress warnings so they don't pollute JSON output
	os.Setenv("AILANG_QUIET_WARNINGS", "1")

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		outputAICheck(aiCheckOutput{
			File: filename,
			Check: aiCheckSection{
				Passed:     false,
				ErrorCount: 1,
				Errors:     []checkJSONError{{Code: "IO_ERROR", Message: fmt.Sprintf("cannot read file: %v", err), File: filename}},
			},
			Verify: aiVerifySection{Available: false, Results: []verifyResult{}},
		})
		os.Exit(1)
	}

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := *relaxModulesFlag
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	// Run pipeline ONCE (both check and verify use the same compilation result)
	cfg := pipeline.Config{
		DryLink:      true,
		RelaxModules: relaxModulesEffective,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, pipelineErr := pipeline.Run(cfg, src)

	// Build check section
	checkSection := aiCheckSection{
		Passed:     true,
		ErrorCount: 0,
		Errors:     []checkJSONError{},
	}

	if pipelineErr != nil {
		checkSection.Passed = false
		checkSection.ErrorCount = 1
		checkSection.Errors = []checkJSONError{errorToCheckJSONError(pipelineErr, filename)}
	} else if len(result.Errors) > 0 {
		checkSection.Passed = false
		checkSection.ErrorCount = len(result.Errors)
		jsonErrors := make([]checkJSONError, len(result.Errors))
		for i, e := range result.Errors {
			jsonErrors[i] = errorToCheckJSONError(e, filename)
		}
		checkSection.Errors = jsonErrors
	}

	// Build verify section
	verifySection := aiVerifySection{
		Available: false,
		Results:   []verifyResult{},
	}

	// Only attempt verification if check passed and Z3 is available
	if checkSection.Passed && smt.Z3Available() {
		verifySection.Available = true

		coreProg := result.Artifacts.Core
		surfaceAST := result.Artifacts.AST

		if coreProg != nil && coreProg.Meta != nil && surfaceAST != nil {
			verifySection = runVerification(coreProg, surfaceAST, *timeoutFlag, *recursiveDepthFlag)
		}
	}

	output := aiCheckOutput{
		File:   filename,
		Check:  checkSection,
		Verify: verifySection,
	}

	outputAICheck(output)

	// Exit code: 1 if check failed or contract violations found
	if !checkSection.Passed || verifySection.Counterexample > 0 {
		os.Exit(1)
	}
}

// runVerification runs contract verification on compiled artifacts.
// Extracted from verifyCommand() for reuse by ai-check.
func runVerification(coreProg *core.Program, surfaceAST *ast.File, timeout time.Duration, recursiveDepth int) aiVerifySection {
	section := aiVerifySection{
		Available: true,
		Results:   []verifyResult{},
	}

	// Extract ADT types from the Surface AST
	adtTypes := extractADTTypes(surfaceAST)

	// Build Surface AST function lookup
	surfaceFuncs := make(map[string]*ast.FuncDecl)
	for _, f := range surfaceAST.Funcs {
		surfaceFuncs[f.Name] = f
	}

	// Solver configuration
	solverCfg := smt.SolverConfig{
		Timeout: timeout,
	}

	// Fixup: functions with ! {} (empty effects) are semantically pure
	for funcName, meta := range coreProg.Meta {
		if fd, ok := surfaceFuncs[funcName]; ok {
			if fd.Effects != nil && len(fd.Effects) == 0 && !meta.IsPure {
				meta.IsPure = true
			}
		}
	}

	// Build surface params and return sorts for all functions
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

		body := findFunctionBody(coreProg, funcName)
		if body == nil {
			section.Results = append(section.Results, verifyResult{
				Function: funcName, Status: "skipped",
				Reason: "function body not found in Core AST",
			})
			section.Skipped++
			continue
		}

		hasEnsures := false
		for _, c := range meta.Contracts {
			if c.Kind == core.EnsuresKind {
				hasEnsures = true
				break
			}
		}
		if !hasEnsures {
			section.Results = append(section.Results, verifyResult{
				Function: funcName, Status: "skipped",
				Reason: "no ensures clause (nothing to verify)",
			})
			section.Skipped++
			continue
		}

		encodable, rejections := smt.IsSMTEncodable(funcName, meta, body)
		if !encodable && recursiveDepth > 0 {
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
			section.Results = append(section.Results, verifyResult{
				Function: funcName, Status: "skipped",
				Reason: strings.Join(reasons, "; "), Rejections: rejections,
			})
			section.Skipped++
			continue
		}

		params, innerBody := unwrapLambdaParams(funcName, surfaceFuncs, body)

		returnSort := ""
		funcEncOpts := encOpts
		if fd, ok := surfaceFuncs[funcName]; ok && fd.ReturnType != nil {
			returnSort = astTypeToSMTSort(fd.ReturnType)
			funcEncOpts.ReturnType = convertASTTypeToType(fd.ReturnType)
		}
		funcEncOpts.Body = innerBody
		funcEncOpts.Contracts = meta.Contracts
		funcEncOpts.RecursiveDepth = recursiveDepth

		encResult, err := smt.EncodeFunction(funcName, params, innerBody, returnSort, meta, adtTypes, funcEncOpts)
		if err != nil {
			section.Results = append(section.Results, verifyResult{
				Function: funcName, Status: "error",
				Reason: fmt.Sprintf("encoding error: %v", err),
			})
			section.Errors++
			continue
		}

		solveResult, err := smt.Solve(encResult.SMTLib, solverCfg)
		if err != nil {
			section.Results = append(section.Results, verifyResult{
				Function: funcName, Status: "error",
				Reason: fmt.Sprintf("solver error: %v", err),
			})
			section.Errors++
			continue
		}

		vr := verifyResult{
			Function: funcName,
			Duration: solveResult.Duration,
		}
		if recursiveDepth > 0 && smt.IsRecursiveFunc(innerBody, funcName) {
			vr.BoundedDepth = recursiveDepth
		}

		switch solveResult.Status {
		case smt.StatusVerified:
			vr.Status = "verified"
			section.Verified++
		case smt.StatusCounterexample:
			vr.Status = "counterexample"
			vr.Model = solveResult.Model
			section.Counterexample++
		case smt.StatusUnknown:
			vr.Status = "unknown"
			vr.Reason = solveResult.Error
			section.Errors++
		case smt.StatusError:
			vr.Status = "error"
			vr.Reason = solveResult.Error
			section.Errors++
		}

		section.Results = append(section.Results, vr)
	}

	return section
}

// outputAICheck writes the unified ai-check JSON output
func outputAICheck(out aiCheckOutput) {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
