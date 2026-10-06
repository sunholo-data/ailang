package testing

// Premise test for design_docs/planned/v0_53_0/m-test-runner-compile-once.md
// (V14/V15). It compiles every named test of a file ONCE, as one module with
// one `pure func __namedtest_k` entry per body, and compares each entry's
// outcome with today's per-body path (EvaluateNamedTestBodyExprs) on both
// engines. Delete it when the sprint lands the real batch and its tests.
//
// Opt-in: it compiles the fixture N+2 times and is a measurement, not a gate.
//   AILANG_NAMED_BATCH_PREMISE=1 go test ./internal/testing -run TestNamedBatchPremise -v
//   AILANG_NAMED_BATCH_PREMISE=/path/to/file_test.ail ...   # any other file

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/vm"
)

// tempRoot masks the per-compile temp dir; it is random on every run today.
var tempRoot = regexp.MustCompile(`\S*ailang-namedtest-\d+/`)

func outcome(v eval.Value, err error) string {
	if err != nil {
		return "ERR " + tempRoot.ReplaceAllString(err.Error(), "<tmp>/")
	}
	return "VAL " + v.String()
}

func TestNamedBatchPremise(t *testing.T) {
	in := os.Getenv("AILANG_NAMED_BATCH_PREMISE")
	if in == "" {
		t.Skip("opt-in measurement; set AILANG_NAMED_BATCH_PREMISE")
	}
	if in == "1" {
		in = "testdata/named_batch/mixed.ail"
	}
	src, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src), in))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatal(p.Errors())
	}
	suite := NewCollector(in).Collect(file)

	// Today's path, per body.
	today := NewExecutor(in)
	today.SetSourceFile(file)
	var named []TestCase
	var want []string
	t0 := time.Now()
	for _, tc := range suite.Tests {
		if tc.IsInline {
			continue
		}
		named = append(named, tc)
		want = append(want, outcome(today.EvaluateNamedTestBodyExprs(tc.Body)))
	}
	perBody := time.Since(t0)

	// Batched: one compile, one bytecode image.
	e := NewExecutor(in)
	e.SetSourceFile(file)
	var sb strings.Builder
	sb.WriteString(e.stripNonPureFunctions(string(src), file))
	checks := make([][]CheckInfo, len(named))
	for k, tc := range named {
		folded, c := FoldTestBody(tc.Body)
		checks[k] = c
		rt := "bool"
		if len(c) > 0 {
			rt = "int"
		}
		fmt.Fprintf(&sb, "\npure func __namedtest_%d() -> %s {\n  %s\n}\n", k, rt, PrintAILANGSource(folded))
	}
	t1 := time.Now()
	res, err := e.runNamedTestPipeline(sb.String(), file.Module != nil)
	if err != nil {
		t.Fatalf("batched compile: %v", err)
	}
	e.cacheModules(&res)
	img, err := runner.CompileBytecodeFromResult(res, "test")
	if err != nil {
		t.Fatalf("batched bytecode: %v", err)
	}
	compile := time.Since(t1)

	decode := func(k int, v eval.Value, err error) (eval.Value, error) {
		if err != nil || len(checks[k]) == 0 {
			return v, err
		}
		return decodeCheckSentinel(v, checks[k])
	}
	t2 := time.Now()
	for k, tc := range named {
		name := fmt.Sprintf("__namedtest_%d", k)
		ev := e.newHarnessEvaluator()
		fn, ok := ev.Env().Get(e.rootModule + "." + name)
		if !ok {
			t.Fatalf("%s not bound", name)
		}
		v, err := ev.CallValueN(fn, nil)
		if err != nil {
			err = fmt.Errorf("evaluation error: %w", err) // today's wrapper (EvaluateNamedTestBodyExprs)
		}
		gotEval := outcome(decode(k, v, err))

		proto := runner.FindEntryProto(img, name)
		if proto == nil {
			t.Fatalf("%s: no proto in the shared image", name)
		}
		var args []bytecode.Value
		if proto.NumParams == 1 {
			args = []bytecode.Value{bytecode.Unit()}
		}
		out, verr := vm.NewVM(img).Run(proto, args)
		var vv eval.Value
		if verr == nil {
			vv, verr = vm.BytecodeToEval(out)
		}
		gotVM := outcome(decode(k, vv, verr))

		t.Logf("%-24q today=%s", tc.Name, want[k])
		if gotEval != want[k] {
			t.Logf("%-24q batch/eval DIFFERS: %s", tc.Name, gotEval)
		}
		if gotVM != want[k] {
			t.Logf("%-24q batch/vm   DIFFERS: %s", tc.Name, gotVM)
		}
	}
	t.Logf("tests=%d per-body(today, evaluator)=%v batched compile+image=%v batched run(eval+vm)=%v",
		len(named), perBody, compile, time.Since(t2))
}
