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

// refusalClass names a measured construct class of a comment-unattached refusal
// (M-AILANG-FMT-INLINE-INTERIOR M3 §Corpus classification gate). Classification is
// REPORTING ONLY — it must not alter attachment behavior.
type refusalClass int

const (
	classLetChainInterior   refusalClass = iota // MUST be 0 after this sprint
	classNonLetEquationBody                     // deferred (3 files)
	classInlineTestsList                        // deferred (9 files)
	classNoEnclosingList                        // deferred footer/no-list (7 files)
	classOtherInterior                          // deferred other (12 files)
)

func (c refusalClass) String() string {
	switch c {
	case classLetChainInterior:
		return "let-chain-interior"
	case classNonLetEquationBody:
		return "non-let-equation-body"
	case classInlineTestsList:
		return "inline-tests-list"
	case classNoEnclosingList:
		return "no-enclosing-list"
	default:
		return "other-interior"
	}
}

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
	classCounts := map[refusalClass]int{}
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
					// M3: split the coarse counter into the measured construct class of
					// the ACTUAL refused comment so residual refusals are labeled truthfully.
					cls := classifyCorpusRefusal(path, data, prog.File)
					classCounts[cls]++
					if cls == classLetChainInterior {
						// A let-chain-interior residual means the fix did not cover a chain
						// comment — record the path so the gate failure is diagnosable.
						otherPaths = append(otherPaths, path+": LET-CHAIN-INTERIOR residual (should be 0)")
					}
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
	// M3: refusal-class split (truthful residual taxonomy). let-chain-interior MUST be 0.
	t.Logf("REFUSAL CLASSES (of %d comment-unattached): let-chain-interior=%d non-let-equation-body=%d inline-tests-list=%d no-enclosing-list=%d other-interior=%d",
		letInRefusal,
		classCounts[classLetChainInterior], classCounts[classNonLetEquationBody],
		classCounts[classInlineTestsList], classCounts[classNoEnclosingList],
		classCounts[classOtherInterior])
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

	// M-AILANG-FMT-INLINE-INTERIOR HEADLINE GATES:
	//   1. The let-chain-interior refusal class is fully eliminated (== 0). This is the
	//      construct class this sprint targeted; a non-zero count is a regression.
	//   2. Total comment-unattached refusals fall to the acceptance ceiling. M0/M1
	//      established that 27 of the 28 let-chain files are fully formattable; the 28th
	//      (records.ail) carries an ADDITIONAL deferred footer comment, so the achievable
	//      residual is 59-27 = 32 (not the plan's optimistic 31, which assumed all 28 had
	//      no coupled deferred comment). The ceiling is 32.
	const inlineInteriorRefusalCeiling = 32
	if classCounts[classLetChainInterior] != 0 {
		t.Fatalf("INLINE-INTERIOR GATE: let-chain-interior refusal class must be 0, got %d (see DEFECT logs) — a let-chain comment is still refusing",
			classCounts[classLetChainInterior])
	}
	if letInRefusal > inlineInteriorRefusalCeiling {
		t.Fatalf("INLINE-INTERIOR GATE: comment-unattached refusals = %d exceed the acceptance ceiling %d (baseline was 59)",
			letInRefusal, inlineInteriorRefusalCeiling)
	}

	// Record the interpolation refusal rate for the design doc's Verification Log.
	t.Logf("INTERPOLATION REFUSAL RATE: %d/%d parse-valid files (%.2f%%)",
		interpRefusal, parseValidN, 100*float64(interpRefusal)/float64(parseValidN))
	t.Logf("LET-IN-BODY REFUSAL RATE: %d/%d parse-valid files (%.2f%%)",
		letInRefusal, parseValidN, 100*float64(letInRefusal)/float64(parseValidN))
}

// classifyCorpusRefusal determines the measured construct class of a file's FIRST
// comment-unattached refusal (M3 §Corpus classification gate). It locates the refused
// comment byte by re-running attachment comment-by-comment, then classifies that byte's
// site. Reporting only — it does not alter attachment.
func classifyCorpusRefusal(path string, data []byte, file *ast.File) refusalClass {
	env, err := NewEnvelope(data)
	if err != nil {
		return classOtherInterior
	}
	a := &attacher{env: env, chainConsumed: map[*ast.Let]bool{}}
	a.collectLists(file)
	refByte := -1
	for _, c := range env.Comments() {
		if _, ok := a.attachOne(c); !ok {
			refByte = c.Start
			break
		}
	}
	if refByte < 0 {
		return classOtherInterior
	}

	// A comment bracketed by a let-chain root that STILL refuses is a let-chain-interior
	// residual (the class this sprint must eliminate). classifyRefusalSite reports the
	// tightest bracketing construct of the byte.
	shape, _ := classifyRefusalSite(env, file, refByte)
	if shape == shapeNestedLetBody {
		return classLetChainInterior
	}

	// Inline tests[...] list: the refused comment lies inside a `tests [ … ]` block.
	if commentInsideTestsList(env, refByte) {
		return classInlineTestsList
	}

	// Footer / no-enclosing-list: the refused byte is after the last top-level decl's
	// content (a trailing footer comment) — no registered list brackets it.
	if refByte > lastTopLevelEnd(a, file) {
		return classNoEnclosingList
	}

	// Non-let single-expression equation body vs other interior: if the refused byte is
	// inside a function body that is a single non-let expression, call it equation-body.
	if commentInSingleExprEquationBody(a, file, refByte) {
		return classNonLetEquationBody
	}
	return classOtherInterior
}

// commentInsideTestsList reports whether byteOff lies within a `tests [ … ]` bracketed
// region. The inline tests list is a string-built child list with no AST node, so it is
// detected textually: the nearest preceding code-level `[` is opened by a `tests` keyword.
func commentInsideTestsList(env *Envelope, byteOff int) bool {
	src := env.src
	depth := 0
	for j := byteOff - 1; j >= 0; j-- {
		if env.inStringSpan(j) {
			continue
		}
		switch src[j] {
		case ']':
			depth++
		case '[':
			if depth > 0 {
				depth--
				continue
			}
			// Found the opening `[` that encloses byteOff. Is it a `tests [`?
			k := j - 1
			for k >= 0 && (src[k] == ' ' || src[k] == '\t' || src[k] == '\n') {
				k--
			}
			return k >= 4 && src[k-4:k+1] == "tests"
		}
	}
	return false
}

// lastTopLevelEnd returns the max subtree-end byte across the file's top-level decls —
// a byte past this is a footer comment with no enclosing list.
func lastTopLevelEnd(a *attacher, file *ast.File) int {
	max := -1
	for _, d := range file.Decls {
		if e := a.subtreeEnd(d); e > max {
			max = e
		}
	}
	return max
}

// commentInSingleExprEquationBody reports whether byteOff is inside a FuncDecl whose
// body is a single non-let expression (the deferred non-let equation-body class).
func commentInSingleExprEquationBody(a *attacher, file *ast.File, byteOff int) bool {
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		blk, isBlk := fd.Body.(*ast.Block)
		if !isBlk || len(blk.Exprs) != 1 {
			continue
		}
		if _, isLet := blk.Exprs[0].(*ast.Let); isLet {
			continue // let chains are handled; this class is NON-let
		}
		min, err := a.env.MinAnchor(fd.Body)
		if err != nil {
			continue
		}
		end := a.subtreeEnd(fd.Body)
		if min <= byteOff && byteOff <= end {
			return true
		}
	}
	return false
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
