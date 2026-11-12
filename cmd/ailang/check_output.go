package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// checkJSONOutput represents the JSON output format for ailang check --json
type checkJSONOutput struct {
	File       string           `json:"file"`
	Passed     bool             `json:"passed"`
	ErrorCount int              `json:"error_count"`
	Errors     []checkJSONError `json:"errors"`
	// Warnings are non-blocking pipeline warnings, same shape as errors;
	// omitted when there are none (check_warnings.go).
	Warnings []checkJSONError `json:"warnings,omitempty"`
}

// checkJSONError represents a single error in JSON check output
type checkJSONError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// checkAgentFormat selects the compact, AI-agent-optimized one-line diagnostic
// rendering (M-AILANG-SEMANTIC-CONTEXT R1, --format=agent). Set once at command
// entry; read-only thereafter. The agent path reuses the same structured-error
// machinery as --json (outputCheckJSON is the single emit point), so it is just
// a different serialization of the same checkJSONOutput.
var checkAgentFormat bool

// outputCheckJSON writes a checkJSONOutput to stdout — as compact agent lines when
// --format=agent is active, otherwise as indented JSON.
func outputCheckJSON(out checkJSONOutput) {
	if checkAgentFormat {
		outputCheckAgent(out)
		return
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// outputCheckAgent renders diagnostics as dense, one-line-per-error text optimized
// for AI agent context windows (M-AILANG-SEMANTIC-CONTEXT R1): no ANSI, no boxes,
// no source-context dump — just "file:line:col CODE: message → suggestion". This is
// SEMANTIC distillation (the typed error + fix hint) rather than the full compiler
// output, so the agent loop spends tokens on meaning, not framing. On success it
// emits a single "ok: <file>" line so the agent gets a clear, parseable pass signal.
func outputCheckAgent(out checkJSONOutput) {
	if out.Passed {
		fmt.Printf("ok: %s\n", out.File)
	} else {
		for _, e := range out.Errors {
			fmt.Println(distillAgentLine(e, out.File))
		}
	}
	// Warnings follow the verdict, so a reader of the first line is unaffected.
	for _, w := range out.Warnings {
		fmt.Println(agentWarnLine(w, out.File))
	}
}

// agentLocRe extracts an embedded "at <file>:<line>:<col>: <rest>" location that
// many AILANG type/elaboration errors carry inside their message string (they are
// not yet structured ailerrors.Report values, so checkJSONError.Span is empty and
// Line==0). Pulling the location to the front and dropping the noisy
// "type error in <mod> (decl N): " framing is the semantic distillation R1 is about.
var agentLocRe = regexp.MustCompile(`\bat ([^\s:]+):(\d+):(\d+): (.*)$`)

// distillAgentLine renders one diagnostic as a single dense, token-lean line:
// "file:line:col CODE: message → suggestion". Prefers structured fields; falls back
// to parsing the location out of the message for not-yet-structured errors.
func distillAgentLine(e checkJSONError, fallbackFile string) string {
	file, line, col := e.File, e.Line, e.Column
	code, msg := e.Code, e.Message
	if line == 0 {
		if m := agentLocRe.FindStringSubmatch(msg); m != nil {
			file = m[1]
			line, _ = strconv.Atoi(m[2])
			col, _ = strconv.Atoi(m[3])
			msg = m[4]
			if code == "" || code == "ERROR" {
				if strings.HasPrefix(e.Message, "type error") {
					code = "TYPE_ERROR"
				}
			}
		}
	}
	if file == "" {
		file = fallbackFile
	}
	loc := file
	if line > 0 {
		loc = fmt.Sprintf("%s:%d:%d", file, line, col)
	}
	out := strings.TrimSpace(fmt.Sprintf("%s %s: %s", loc, code, msg))
	if e.Suggestion != "" {
		out += " → " + e.Suggestion
	}
	return out
}

// pipelineErrorsToCheckJSON converts a pipeline error to its JSON entries:
// one per error when the error carries several (every TC_MATCH_* error of a
// module, so an author can fix them in one pass), else one.
func pipelineErrorsToCheckJSON(err error, filename string) []checkJSONError {
	var many types.MatchErrors
	if errors.As(err, &many) && len(many) > 1 {
		out := make([]checkJSONError, len(many))
		for i, e := range many {
			out[i] = errorToCheckJSONError(e, filename)
		}
		return out
	}
	return []checkJSONError{errorToCheckJSONError(err, filename)}
}

// errorToCheckJSONError converts an error to a structured JSON error entry
func errorToCheckJSONError(err error, filename string) checkJSONError {
	if rep, ok := ailerrors.AsReport(err); ok {
		entry := checkJSONError{
			Code:    rep.Code,
			Message: rep.Message,
		}
		if rep.Span != nil {
			entry.File = rep.Span.Start.File
			entry.Line = rep.Span.Start.Line
			entry.Column = rep.Span.Start.Column
		}
		if rep.Fix != nil {
			entry.Suggestion = rep.Fix.Suggestion
		}
		return entry
	}
	// Fallback for non-Report errors
	return checkJSONError{
		Code:    jsonErrorCode(err.Error()),
		Message: err.Error(),
		File:    filename,
	}
}

// jsonErrorCode is the --json code of an error that is not a structured
// report: the nominal-identity code its message carries
// (M-V1-NOMINAL-TYPE-IDENTITY: TC_NOMINAL_001, TC_AMBIGUOUS_TYPE_001,
// TC_AMBIGUOUS_CTOR_001) or record unification (#2210/#2221: TC_REC_001, TC_REC_004,
// TC_REC_003, TC004), else the generic "ERROR". When several appear (a
// TC_AMBIGUOUS_CTOR_001 whose cause carries TC_NOMINAL_001) the first in the
// message is the error's own. Other inline TC_* codes keep "ERROR", as before,
// so existing consumers see no change.
func jsonErrorCode(msg string) string {
	best, at := "ERROR", -1
	for _, code := range append(types.NominalIdentityCodes(), types.RecordRowCodes()...) {
		if i := strings.Index(msg, code); i >= 0 && (at < 0 || i < at) {
			best, at = code, i
		}
	}
	return best
}
