package eval_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	ailruntime "github.com/sunholo/ailang/internal/runtime"
)

// ============================================================================
// Phase 2A Benchmark Harness
// ============================================================================
//
// These benchmarks measure evaluator runtime performance vs native Go.
// Startup time (parsing, type checking, elaboration) is EXCLUDED.
//
// Run with: go test -bench=. -benchmem -count=5 ./internal/eval/ -run=^$ -timeout=600s
// Or:       make bench-phase2a

// findProjectRoot walks up from the current file to find the project root
// (directory containing go.mod).
func findProjectRoot() string {
	// Start from the directory of this test file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// precompiledModule holds a compiled module ready for repeated evaluation.
// The compilation (parse → typecheck → elaborate) is done once in setup.
type precompiledModule struct {
	rt   *ailruntime.ModuleRuntime
	inst *ailruntime.ModuleInstance
}

// compileModule compiles an AILANG source file through the full pipeline,
// returning a module runtime ready for repeated entrypoint calls.
// This is the SETUP phase — not measured in benchmarks.
// caps is a list of effect capabilities to grant (e.g., "IO").
func compileModule(filename string, caps ...string) (*precompiledModule, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	cfg := pipeline.Config{
		Mode:         pipeline.ModeCheck, // Compile only, don't evaluate
		RelaxModules: true,               // Benchmark files aren't at canonical module paths
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.RunWithContext(context.Background(), cfg, src)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}

	// Set up module runtime
	rt := ailruntime.NewModuleRuntime(".")

	// Set up effect context with requested capabilities
	if len(caps) > 0 {
		effCtx := effects.NewEffContext(nil)
		for _, cap := range caps {
			effCtx.Grant(effects.Capability{Name: cap})
		}
		// Suppress IO output during benchmarks by redirecting to devnull
		devnull, _ := os.Open(os.DevNull)
		if devnull != nil {
			effCtx.IOWriter = devnull
		}
		rt.GetEvaluator().SetEffContext(effCtx)
	}

	// Preload compiled modules
	if result.Modules != nil {
		for path, loaded := range result.Modules {
			rt.PreloadModule(path, loaded)
		}
	}

	// Load and evaluate module (initializes bindings)
	moduleName := ""
	if result.Interface != nil {
		moduleName = result.Interface.Module
	}
	if moduleName == "" {
		return nil, fmt.Errorf("no module name in compiled result")
	}

	inst, err := rt.LoadAndEvaluate(moduleName)
	if err != nil {
		return nil, fmt.Errorf("load module: %w", err)
	}

	return &precompiledModule{rt: rt, inst: inst}, nil
}

// callMain calls the "main" entrypoint on a precompiled module.
// This is what gets measured in the benchmark loop.
func (m *precompiledModule) callMain() (eval.Value, error) {
	return ailruntime.CallEntrypoint(m.rt, m.inst, "main", []eval.Value{&eval.UnitValue{}})
}

// benchmarkAILANGFile runs a benchmark for an AILANG source file.
// Compilation is excluded from timing; only evaluation is measured.
func benchmarkAILANGFile(b *testing.B, relPath string, caps ...string) {
	root := findProjectRoot()
	filename := filepath.Join(root, relPath)

	mod, err := compileModule(filename, caps...)
	if err != nil {
		b.Fatalf("compile %s: %v", relPath, err)
	}

	// Verify it works once
	val, err := mod.callMain()
	if err != nil {
		b.Fatalf("initial run %s: %v", relPath, err)
	}
	b.Logf("Result: %s", val.String())

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := mod.callMain()
		if err != nil {
			b.Fatalf("iteration %d: %v", i, err)
		}
	}
}

// benchmarkAILANGLatency runs a benchmark and reports p95 latency.
func benchmarkAILANGLatency(b *testing.B, relPath string, caps ...string) {
	root := findProjectRoot()
	filename := filepath.Join(root, relPath)

	mod, err := compileModule(filename, caps...)
	if err != nil {
		b.Fatalf("compile %s: %v", relPath, err)
	}

	// Warm up
	for i := 0; i < 3; i++ {
		if _, err := mod.callMain(); err != nil {
			b.Fatalf("warmup: %v", err)
		}
	}

	// Collect per-iteration timings
	latencies := make([]time.Duration, b.N)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := mod.callMain()
		latencies[i] = time.Since(start)
		if err != nil {
			b.Fatalf("iteration %d: %v", i, err)
		}
	}
	b.StopTimer()

	// Calculate and report p95
	if b.N > 0 {
		sort.Slice(latencies[:b.N], func(i, j int) bool { return latencies[i] < latencies[j] })
		p95idx := int(float64(b.N) * 0.95)
		if p95idx >= b.N {
			p95idx = b.N - 1
		}
		b.ReportMetric(float64(latencies[p95idx].Nanoseconds()), "p95-ns")
	}
}

// ============================================================================
// Evaluator Benchmarks (AILANG through pipeline)
// ============================================================================

func BenchmarkEval_Fib30(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/fib30.ail")
}

func BenchmarkEval_ListMapFilter(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/list_map_filter.ail")
}

func BenchmarkEval_PatternMatch(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/pattern_match.ail")
}

func BenchmarkEval_ClosureCurried(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/closure_curried.ail")
}

func BenchmarkEval_StringPipeline(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/string_pipeline.ail")
}

func BenchmarkEval_GameStep(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/game_step.ail")
}

func BenchmarkEval_GameStep_Latency(b *testing.B) {
	benchmarkAILANGLatency(b, "benchmarks/runtime/game_step.ail")
}

func BenchmarkEval_CrossBoundary(b *testing.B) {
	benchmarkAILANGFile(b, "benchmarks/runtime/cross_boundary.ail", "IO")
}

func BenchmarkEval_CrossBoundary_Latency(b *testing.B) {
	benchmarkAILANGLatency(b, "benchmarks/runtime/cross_boundary.ail", "IO")
}

// ============================================================================
// Native Go Baselines
// ============================================================================
// These implement the same algorithms in pure Go for comparison.

// --- fib(30) ---

func nativeFib(n int) int {
	if n <= 1 {
		return n
	}
	return nativeFib(n-1) + nativeFib(n-2)
}

func BenchmarkNative_Fib30(b *testing.B) {
	// Verify correctness
	if got := nativeFib(30); got != 832040 {
		b.Fatalf("nativeFib(30) = %d, want 832040", got)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeFib(30)
	}
}

// --- map/filter over list ---

func nativeListMapFilter() int {
	// Build list [1..5000]
	xs := make([]int, 5000)
	for i := range xs {
		xs[i] = i + 1
	}
	// Filter evens
	evens := make([]int, 0, 2500)
	for _, x := range xs {
		if x%2 == 0 {
			evens = append(evens, x)
		}
	}
	// Square
	squared := make([]int, len(evens))
	for i, x := range evens {
		squared[i] = x * x
	}
	// Sum
	sum := 0
	for _, x := range squared {
		sum += x
	}
	return sum
}

func BenchmarkNative_ListMapFilter(b *testing.B) {
	expected := nativeListMapFilter()
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeListMapFilter()
	}
}

// --- pattern match (tree sum) ---

type nativeTree interface {
	isTree()
}
type nativeLeaf struct{ value int }
type nativeNode struct {
	left  nativeTree
	value int
	right nativeTree
}

func (nativeLeaf) isTree() {}
func (nativeNode) isTree() {}

func nativeTreeSum(t nativeTree) int {
	switch t := t.(type) {
	case nativeLeaf:
		return t.value
	case nativeNode:
		return nativeTreeSum(t.left) + t.value + nativeTreeSum(t.right)
	default:
		panic("unreachable")
	}
}

func nativeTreeCount(t nativeTree) int {
	switch t := t.(type) {
	case nativeLeaf:
		return 1
	case nativeNode:
		return 1 + nativeTreeCount(t.left) + nativeTreeCount(t.right)
	default:
		panic("unreachable")
	}
}

func nativeBuildTree(depth, seed int) nativeTree {
	if depth <= 0 {
		return nativeLeaf{seed}
	}
	return nativeNode{
		left:  nativeBuildTree(depth-1, seed*2),
		value: seed,
		right: nativeBuildTree(depth-1, seed*2+1),
	}
}

func BenchmarkNative_PatternMatch(b *testing.B) {
	tree := nativeBuildTree(12, 1)
	expected := nativeTreeSum(tree) + nativeTreeCount(tree)
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	var sink int
	for i := 0; i < b.N; i++ {
		tree := nativeBuildTree(12, 1)
		sink = nativeTreeSum(tree) + nativeTreeCount(tree)
	}
	_ = sink
}

// --- closure/curried HOFs ---

func nativeClosureCurried() int {
	add := func(x int) func(int) int {
		return func(y int) int { return x + y }
	}
	mul := func(x int) func(int) int {
		return func(y int) int { return x * y }
	}
	compose := func(f, g func(int) int) func(int) int {
		return func(x int) int { return f(g(x)) }
	}

	// Build [1..1000]
	xs := make([]int, 1000)
	for i := range xs {
		xs[i] = i + 1
	}

	transform := compose(add(7), mul(3))

	sum := 0
	for _, x := range xs {
		sum += transform(x)
	}
	return sum
}

func BenchmarkNative_ClosureCurried(b *testing.B) {
	expected := nativeClosureCurried()
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeClosureCurried()
	}
}

// --- string pipeline ---

func nativeStringPipeline() int {
	processLine := func(line string) string {
		if strings.HasPrefix(line, "X-") {
			return strings.ToUpper(line)
		} else if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			trimmed := make([]string, len(parts))
			for i, p := range parts {
				trimmed[i] = strings.TrimSpace(p)
			}
			return strings.Join(trimmed, ": ")
		}
		return strings.ToLower(line)
	}

	// Build 500 lines
	lines := make([]string, 500)
	for i := range lines {
		n := i + 1
		switch n % 3 {
		case 0:
			lines[i] = fmt.Sprintf("X-Custom-Header: value%d", n)
		case 1:
			lines[i] = "Content-Type : text/plain"
		default:
			lines[i] = fmt.Sprintf("some plain text line %d", n)
		}
	}

	totalLen := 0
	for _, line := range lines {
		totalLen += len(processLine(line))
	}
	return totalLen
}

func BenchmarkNative_StringPipeline(b *testing.B) {
	expected := nativeStringPipeline()
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeStringPipeline()
	}
}

// --- game step ---

type nativeEntity struct {
	x, y, vx, vy int
}

func nativeStepEntity(e nativeEntity) nativeEntity {
	nx := e.x + e.vx
	ny := e.y + e.vy
	nvx := e.vx
	nvy := e.vy
	if nx < 0 || nx > 800 {
		nvx = -nvx
	}
	if ny < 0 || ny > 600 {
		nvy = -nvy
	}
	fx := nx
	if fx < 0 {
		fx = 0
	} else if fx > 800 {
		fx = 800
	}
	fy := ny
	if fy < 0 {
		fy = 0
	} else if fy > 600 {
		fy = 600
	}
	return nativeEntity{fx, fy, nvx, nvy}
}

func nativeGameStep() int {
	// Build 50 entities
	entities := make([]nativeEntity, 50)
	for i := range entities {
		n := i + 1
		entities[i] = nativeEntity{
			x:  (n * 10) % 800,
			y:  (n * 7) % 600,
			vx: (n % 5) - 2,
			vy: (n % 3) - 1,
		}
	}

	// Run 200 frames
	for frame := 0; frame < 200; frame++ {
		for i := range entities {
			entities[i] = nativeStepEntity(entities[i])
		}
	}

	// Sum X positions
	sum := 0
	for _, e := range entities {
		sum += e.x
	}
	return sum
}

func BenchmarkNative_GameStep(b *testing.B) {
	expected := nativeGameStep()
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeGameStep()
	}
}

// --- cross-boundary (pure calling effectful) ---

// compute matches the AILANG cross_boundary.ail: base case returns 1 (not n)
func nativeCompute(n int) int {
	if n <= 1 {
		return 1
	}
	return nativeCompute(n-1) + nativeCompute(n-2)
}

func nativeCrossBoundary() int {
	// compute(15) called 100 times with accumulator
	total := 0
	for i := 0; i < 100; i++ {
		total += nativeCompute(15)
	}
	return total
}

func BenchmarkNative_CrossBoundary(b *testing.B) {
	expected := nativeCrossBoundary()
	b.Logf("Result: %d", expected)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		nativeCrossBoundary()
	}
}

// ============================================================================
// Summary helper (run as a test, not benchmark)
// ============================================================================
