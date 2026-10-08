package testing

// Named tests compile ONCE per file (M-TEST-RUNNER-COMPILE-ONCE,
// design_docs/planned/v0_53_0/m-test-runner-compile-once.md).
//
// Each named test used to be its own pipeline compile: the stripped module
// plus that one body, as a transient root the compile cache never stores. On
// a real module that is one full compile per test (stapledons' protocol_test:
// 29 tests × ~7.3 s). prepareNamedTests instead compiles the stripped module
// with one entry per body,
//
//	pure func __namedtest_<k>() { <body k> }
//
// (the per-body VM path's form, minus its return annotation, since a body need
// not type as bool), and every test runs its entry on a fresh evaluator or VM
// over that one result.
//
// D1 (ratified 2026-10-06): if the batched compile fails, the run says so and
// every test takes the per-body path, so a body that does not type-check
// still fails alone, with today's message. The notice is never silent: a
// batch failure where every body then compiles alone is a harness bug, and
// the notice says to report it.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
)

// namedBatchPrefix names the batched entries; namedTestEntry (the per-body
// entry) shares the reserved "__namedtest_" prefix.
const namedBatchPrefix = "__namedtest_"

// namedBatch is the one compile every named test of a file runs against.
type namedBatch struct {
	entries map[ast.Expr]namedEntry      // keyed by the body's first expression
	props   map[*ast.Property]namedEntry // forall properties (property_batch.go)
	modules map[string]*loader.LoadedModule
	root    string
	lineMap []int // batched source line i+1 → user's line, for the stripped module

	// Bytecode image, built on the first --bytecode use. imgErr set means the
	// shared image could not be built and VM runs take the per-body path.
	img      *bytecode.BytecodeImage
	imgErr   error
	imgBuilt bool
}

type namedEntry struct {
	name   string
	checks []CheckInfo
	// Lines [first, last] of this entry in the batched source, and the test's
	// own line in the user's file, for mapping runtime-error positions (D5).
	first, last, testLine int
	// what reports a position inside the entry: "test body" or "property"
	label string
}

// BatchFailure records a file whose named tests could not share one compile.
type BatchFailure struct {
	Module string `json:"module"`
	Reason string `json:"reason"`
	// HarnessBug is set when every body then compiled on its own: the batch,
	// not the user's code, was at fault.
	HarnessBug bool `json:"harness_bug"`
	// userCause: the failure is explained without a harness bug (a reserved
	// name), so HarnessBug is never inferred.
	userCause bool
}

// Notice is the one-line stderr report for this failure.
func (f BatchFailure) Notice() string {
	msg := fmt.Sprintf("named tests in %s: could not share one compile (%s); compiled each test separately",
		f.Module, firstLine(f.Reason))
	if f.HarnessBug {
		msg += " — every body compiles on its own, so this is an ailang test harness bug; please report it"
	}
	return msg
}

// prepareNamedTests compiles every named test of the file in one pipeline
// run. It is a no-op when there is nothing to batch; on failure it records a
// BatchFailure and leaves the per-body path in charge.
func (e *Executor) prepareNamedTests(cases []TestCase, properties []PropertyCase) {
	e.batch, e.batchFailure, e.perBodyCompileFailures = nil, nil, 0
	if e.sourceFile == nil {
		return
	}
	src, err := os.ReadFile(e.modulePath)
	if err != nil {
		return // the per-body path reports the unreadable source per test
	}
	for _, f := range e.sourceFile.Funcs {
		if strings.HasPrefix(f.Name, namedBatchPrefix) {
			e.batchFailure = &BatchFailure{Module: e.modulePath, userCause: true,
				Reason: fmt.Sprintf("the module declares %s, a name reserved for the test harness", f.Name)}
			return
		}
	}

	base, lineMap := e.stripWithLineMap(string(src), e.sourceFile, nil)
	es := newEntrySource(base)
	entries := make(map[ast.Expr]namedEntry)
	for _, tc := range cases {
		if tc.IsInline || len(tc.Body) == 0 {
			continue
		}
		folded, checks := FoldTestBody(tc.Body)
		if folded == nil {
			continue // the per-body path reports it
		}
		name := fmt.Sprintf("%s%d", namedBatchPrefix, len(entries))
		ent := es.add(name, "", PrintAILANGSource(folded), tc.Location.Line)
		ent.checks = checks
		entries[tc.Body[0]] = ent
	}
	props := make(map[*ast.Property]namedEntry)
	for _, pc := range properties {
		if !isForall(pc.Property) {
			continue
		}
		name := fmt.Sprintf("%sprop_%d", namedBatchPrefix, len(props))
		ent := es.add(name, binderParams(pc.Property), PrintAILANGSource(pc.Property.Expr), pc.Property.Pos.Line)
		ent.label = "property"
		props[pc.Property] = ent
	}
	if len(entries)+len(props) == 0 {
		return
	}

	res, err := e.runNamedTestPipeline(es.String(), e.sourceFile.Module != nil)
	if err != nil {
		e.batchFailure = &BatchFailure{Module: e.modulePath, Reason: err.Error()}
		return
	}
	e.cacheModules(&res)
	e.batch = &namedBatch{entries: entries, props: props, modules: e.modules, root: e.rootModule, lineMap: lineMap}
	e.batchResult = res
}

// finishNamedTests completes the failure record once every test has run.
func (e *Executor) finishNamedTests() *BatchFailure {
	if e.batchFailure != nil && e.perBodyCompileFailures == 0 && !e.batchFailure.userCause {
		e.batchFailure.HarnessBug = true
	}
	return e.batchFailure
}

// batchedEntry returns the prepared entry for a named-test body, if any.
func (e *Executor) batchedEntry(bodyExprs []ast.Expr) (namedEntry, bool) {
	if e.batch == nil || len(bodyExprs) == 0 {
		return namedEntry{}, false
	}
	ent, ok := e.batch.entries[bodyExprs[0]]
	return ent, ok
}

// evaluateBatched runs one prepared entry with the same engine rules as the
// per-body path: the VM when --bytecode and it can, else the evaluator.
// handled is false when the shared bytecode image could not be built: one
// unlowerable body must not fail every test under --strict-bytecode, so the
// caller takes the per-body path, which isolates it as before.
func (e *Executor) evaluateBatched(ent namedEntry) (val eval.Value, handled bool, err error) {
	// Other harness paths (inline tests) replace the cached modules; every
	// batched entry evaluates against the batch's own compile.
	e.modules, e.rootModule = e.batch.modules, e.batch.root

	if e.bytecode && e.sharedImage() != nil {
		return nil, false, nil
	}
	val, err = e.runBatchedEntry(ent)
	if err != nil {
		err = &mappedError{msg: e.mapBatchPositions(err.Error(), ent), err: err}
	}
	return val, true, err
}

// mappedError carries the position-mapped message and keeps the original
// error reachable for errors.Is / errors.As.
type mappedError struct {
	msg string
	err error
}

func (m *mappedError) Error() string { return m.msg }
func (m *mappedError) Unwrap() error { return m.err }

var (
	// A position in the private temp file a batch is compiled in. The dir is
	// random per compile, so today's messages already differ run to run.
	batchTempPos = regexp.MustCompile(`/?[^\s(]*ailang-namedtest-\d+/[^\s:/]+\.ail:(\d+)(:\d+)?`)
	batchTempDir = regexp.MustCompile(`/?[^\s(]*ailang-namedtest-\d+/`)
)

// mapBatchPositions rewrites temp-file positions in a batched entry's error
// to the user's file (D5): a line of the stripped module maps through the
// strip line map, and a line inside this test's entry becomes the test's own
// line. The message then names a file the user can open, and is the same on
// every run.
func (e *Executor) mapBatchPositions(msg string, ent namedEntry) string {
	return e.mapPositions(msg, ent, e.batch.lineMap)
}

func (e *Executor) mapPositions(msg string, ent namedEntry, lineMap []int) string {
	msg = batchTempPos.ReplaceAllStringFunc(msg, func(m string) string {
		sub := batchTempPos.FindStringSubmatch(m)
		line, _ := strconv.Atoi(sub[1])
		switch {
		case line >= 1 && line <= len(lineMap):
			return fmt.Sprintf("%s:%d%s", e.modulePath, lineMap[line-1], sub[2])
		case line >= ent.first && line <= ent.last:
			return fmt.Sprintf("%s:%d (%s)", e.modulePath, ent.testLine, ent.label)
		}
		return m
	})
	return batchTempDir.ReplaceAllString(msg, filepath.Dir(e.modulePath)+"/")
}

func (e *Executor) runBatchedEntry(ent namedEntry) (eval.Value, error) {
	if e.bytecode {
		val, vmErr := e.runVMEntry(e.batch.img, findBatchedProto(e.batch.img, ent.name))
		switch {
		case vmErr == nil:
			e.engine.VMBodies++
			return decodeEntry(val, ent)
		case e.strictBytecode:
			e.engine.StrictFailures++
			return nil, fmt.Errorf("--strict-bytecode: %w", vmErr)
		default:
			e.engine.noteFallback(vmErr)
		}
	}

	ev := e.newHarnessEvaluator()
	fn, ok := ev.Env().Get(e.batch.root + "." + ent.name)
	if !ok {
		return nil, fmt.Errorf("named test entry %s is not bound in the test harness", ent.name)
	}
	val, err := ev.CallValueN(fn, nil)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}
	return decodeEntry(val, ent)
}

func decodeEntry(val eval.Value, ent namedEntry) (eval.Value, error) {
	if len(ent.checks) > 0 {
		return decodeCheckSentinel(val, ent.checks)
	}
	return val, nil
}

// sharedImage builds the batch's bytecode image on first use and returns the
// build error, if any. A failure is reported once, as a batch failure, so the
// slower per-body VM path that follows is never silent.
func (e *Executor) sharedImage() error {
	b := e.batch
	if !b.imgBuilt {
		b.imgBuilt = true
		b.img, b.imgErr = compileTestImage(e.batchResult)
		if b.imgErr != nil && e.batchFailure == nil {
			e.batchFailure = &BatchFailure{Module: e.modulePath, userCause: true,
				Reason: "shared bytecode image: " + b.imgErr.Error()}
		}
	}
	return b.imgErr
}

// findBatchedProto matches an entry exactly or as "<module>.<entry>".
// runner.FindEntryProto also accepts a "_<entry>" suffix, which is ambiguous
// among numbered entries.
func findBatchedProto(img *bytecode.BytecodeImage, name string) *bytecode.FuncPrototype {
	for _, p := range img.Prototypes {
		if p.Name == name || strings.HasSuffix(p.Name, "."+name) {
			return p
		}
	}
	return nil
}

// entrySource builds a batched source: the stripped module, then one entry
// function per test, recording each entry's line span for position mapping.
type entrySource struct{ sb strings.Builder }

func newEntrySource(base string) *entrySource {
	es := &entrySource{}
	es.sb.WriteString(base)
	es.sb.WriteString("\n")
	return es
}

// add appends `pure func <name>(<params>) { <body> }`. There is no return
// annotation: a named-test body is not required to TYPE as bool (the runner
// fails `{ 1.5 }` "expected bool result"; `-> bool` would instead fail the
// whole batch, caught by TestTestCommandBytecodeFlags), and a property's
// result is checked the same way.
func (es *entrySource) add(name, params, body string, testLine int) namedEntry {
	first := strings.Count(es.sb.String(), "\n") + 1
	fmt.Fprintf(&es.sb, "\npure func %s(%s) {\n  %s\n}\n", name, params, body)
	return namedEntry{name: name, first: first, last: strings.Count(es.sb.String(), "\n"),
		testLine: testLine, label: "test body"}
}

func (es *entrySource) String() string { return es.sb.String() }
