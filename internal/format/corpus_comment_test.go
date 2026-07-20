package format

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// corpus_comment_test.go is the M3 corpus gate: it sweeps every PARSE-VALID
// examples/**/*.ail file through the comment-preserving formatter and asserts:
//   - 0 envelope/attachment errors (fail-closed defects), EXCEPT the fail-closed
//     interpolation carve-out and the enumerated let-in-body limit, which are
//     COUNTED and reported, not silently tolerated;
//   - structural round-trip Parse(fmt(x)) == Parse(x) for every file that formats;
//   - marker preservation: every comment survives exactly once.
// It also records the interpolation-comment refusal rate (BINDING CONSTRAINT 2
// evidence gate).

func TestCorpusCommentGate(t *testing.T) {
	var (
		parseValidN   int
		formattedN    int
		interpRefusal int
		letInRefusal  int
		otherRefusal  int
		roundTripFail int
		preExistingRT int
		markerFail    int
	)
	var otherPaths []string
	var preExistingPaths []string

	walkAilExamples(t, func(path string, data []byte) {
		p := parser.New(lexer.New(string(data), path))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			return // non-parsing: OUT of the gate (exit 3 by design)
		}
		parseValidN++

		out, err := SourceWithComments(prog, data, Options{})
		if err != nil {
			if ee, ok := err.(*EnvelopeError); ok {
				switch ee.Kind {
				case "interp-comment":
					interpRefusal++
					return
				case "comment-unattached":
					letInRefusal++
					return
				}
			}
			otherRefusal++
			if len(otherPaths) < 30 {
				otherPaths = append(otherPaths, path+": "+err.Error())
			}
			return
		}
		formattedN++

		// Marker preservation (count-based): the number of scanned comments in the
		// output must equal the number in the input. Per-comment substring counting
		// is unreliable (a bare `--` comment is a substring of every `-- foo`), so we
		// compare comment COUNTS via the same scanner on input and output.
		inComments, _ := lexer.CollectComments(data)
		outComments, _ := lexer.CollectComments(out)
		if len(outComments) != len(inComments) {
			markerFail++
			if len(otherPaths) < 30 {
				otherPaths = append(otherPaths, path+": comment count changed "+
					itoa(len(inComments))+" -> "+itoa(len(outComments)))
			}
		}

		// Structural round-trip.
		rp := parser.New(lexer.New(string(out), path))
		reprog := rp.Parse()
		rtBroken := len(rp.Errors()) > 0 || reprog == nil || reprog.File == nil
		if !rtBroken {
			if diff := cmp.Diff(prog.File, reprog.File, ignorePosSpan); diff != "" {
				rtBroken = true
			}
		}
		if rtBroken {
			// Classify: is this a PRE-EXISTING Phase-1 printer bug (the comment-free
			// skeleton also fails round-trip), or a Phase-2 comment REGRESSION?
			if phase1AlsoBreaks(prog.File, path) {
				preExistingRT++
				if len(preExistingPaths) < 40 {
					preExistingPaths = append(preExistingPaths, path)
				}
			} else {
				roundTripFail++
				if len(otherPaths) < 30 {
					otherPaths = append(otherPaths, path+": PHASE-2 round-trip REGRESSION (comment-free skeleton round-trips, commented does not)")
				}
			}
		}
	})

	if parseValidN == 0 {
		t.Skip("no parse-valid corpus files")
	}
	t.Logf("CORPUS COMMENT GATE: parse-valid=%d formatted=%d | interp-refusal=%d let-in-refusal=%d other-refusal=%d PHASE2-rt-regression=%d preexisting-Phase1-rt-bug=%d marker-fail=%d",
		parseValidN, formattedN, interpRefusal, letInRefusal, otherRefusal, roundTripFail, preExistingRT, markerFail)
	for _, p := range otherPaths {
		t.Logf("  DEFECT: %s", p)
	}
	for _, p := range preExistingPaths {
		t.Logf("  PRE-EXISTING Phase-1 printer bug (round-trip fails comment-free too, caught fail-closed by fmt.go): %s", p)
	}

	// HARD GATE: zero Phase-2 defects AND zero pre-existing Phase-1 round-trip bugs.
	// interp-refusal and let-in-refusal are the enumerated evidence-gated fail-closed
	// carve-outs. preExistingRT WAS a logged-and-tolerated exception (the contract
	// corpus's Phase-1 printer bugs); m-fmt-properties-printer-roundtrip drove it to
	// zero (contract-clause emission + parser clobber fix + the two adjacent
	// paren-separator / @verify-annotation printer fixes), so it is now a HARD
	// failure — any future non-zero count means a printer round-trip regression
	// silently regrew the exception class.
	if otherRefusal != 0 || roundTripFail != 0 || markerFail != 0 || preExistingRT != 0 {
		t.Fatalf("corpus comment gate FAILED: other-refusal=%d PHASE2-rt-regression=%d marker-fail=%d preexisting-Phase1-rt-bug=%d (see DEFECT logs; preExistingRT must stay 0 after m-fmt-properties-printer-roundtrip)",
			otherRefusal, roundTripFail, markerFail, preExistingRT)
	}
	// Record the interpolation refusal rate for the design doc's Verification Log.
	t.Logf("INTERPOLATION REFUSAL RATE: %d/%d parse-valid files (%.2f%%)",
		interpRefusal, parseValidN, 100*float64(interpRefusal)/float64(parseValidN))
	t.Logf("LET-IN-BODY REFUSAL RATE: %d/%d parse-valid files (%.2f%%)",
		letInRefusal, parseValidN, 100*float64(letInRefusal)/float64(parseValidN))
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// phase1AlsoBreaks reports whether the COMMENT-FREE Phase-1 Source output of the
// same parsed file also fails structural round-trip — i.e. the round-trip failure
// is a pre-existing printer bug (independent of comments), not a Phase-2 comment
// regression. It formats file via Phase-1 Source and re-parses.
func phase1AlsoBreaks(file *ast.File, path string) bool {
	out, err := Source(&ast.Program{File: file}, Options{})
	if err != nil {
		return true // Phase-1 can't even print it → not a Phase-2 comment regression
	}
	rp := parser.New(lexer.New(string(out), path))
	reprog := rp.Parse()
	if len(rp.Errors()) > 0 || reprog == nil || reprog.File == nil {
		return true
	}
	return cmp.Diff(file, reprog.File, ignorePosSpan) != ""
}
