package pipeline

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// BenchmarkValidateCoreTypeInfo_SmallProgram benchmarks validation on a small program (~10 nodes)
func BenchmarkValidateCoreTypeInfo_SmallProgram(b *testing.B) {
	prog, coreTI := makeProgram(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCoreTypeInfo(prog, coreTI)
	}
}

// BenchmarkValidateCoreTypeInfo_MediumProgram benchmarks validation on a medium program (~100 nodes)
func BenchmarkValidateCoreTypeInfo_MediumProgram(b *testing.B) {
	prog, coreTI := makeProgram(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCoreTypeInfo(prog, coreTI)
	}
}

// BenchmarkValidateCoreTypeInfo_LargeProgram benchmarks validation on a large program (~1000 nodes)
func BenchmarkValidateCoreTypeInfo_LargeProgram(b *testing.B) {
	prog, coreTI := makeProgram(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCoreTypeInfo(prog, coreTI)
	}
}

// makeProgram creates a synthetic program with N let-bindings
// Structure: let x1 = 1 in let x2 = 2 in ... let xN = N in xN
func makeProgram(n int) (*core.Program, types.CoreTypeInfo) {
	coreTI := types.NewCoreTypeInfo()
	nodeID := uint64(1)

	// Build nested lets from inside out
	var body core.CoreExpr

	// Innermost: variable reference
	varNode := &core.Var{
		CoreNode: core.CoreNode{NodeID: nodeID},
		Name:     "x0",
	}
	coreTI.Set(varNode.ID(), &types.TCon{Name: "Int"})
	nodeID++
	body = varNode

	// Wrap in N let-bindings
	for i := 0; i < n; i++ {
		litNode := &core.Lit{
			CoreNode: core.CoreNode{NodeID: nodeID},
			Kind:     core.IntLit,
			Value:    i,
		}
		coreTI.Set(litNode.ID(), &types.TCon{Name: "Int"})
		nodeID++

		letNode := &core.Let{
			CoreNode: core.CoreNode{NodeID: nodeID},
			Name:     "x" + string(rune('0'+i%10)),
			Value:    litNode,
			Body:     body,
		}
		coreTI.Set(letNode.ID(), &types.TCon{Name: "Int"})
		nodeID++

		body = letNode
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{body},
	}

	return prog, coreTI
}

// BenchmarkValidateCoreTypeInfo_DeepNesting benchmarks validation on deeply nested expressions
// This tests worst-case recursion depth
func BenchmarkValidateCoreTypeInfo_DeepNesting(b *testing.B) {
	// Create deeply nested lets: 500 levels deep
	prog, coreTI := makeProgram(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCoreTypeInfo(prog, coreTI)
	}
}

// BenchmarkValidateCoreTypeInfo_WideTree benchmarks validation on wide expressions
// This tests branch factor (many children per node)
func BenchmarkValidateCoreTypeInfo_WideTree(b *testing.B) {
	coreTI := types.NewCoreTypeInfo()
	nodeID := uint64(1)

	// Create a record with 100 fields
	fields := make(map[string]core.CoreExpr)
	for i := 0; i < 100; i++ {
		litNode := &core.Lit{
			CoreNode: core.CoreNode{NodeID: nodeID},
			Kind:     core.IntLit,
			Value:    i,
		}
		coreTI.Set(litNode.ID(), &types.TCon{Name: "Int"})
		nodeID++

		fields["field"+string(rune('0'+i%10))] = litNode
	}

	recordNode := &core.Record{
		CoreNode: core.CoreNode{NodeID: nodeID},
		Fields:   fields,
	}
	coreTI.Set(recordNode.ID(), &types.TCon{Name: "Record"})

	prog := &core.Program{
		Decls: []core.CoreExpr{recordNode},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCoreTypeInfo(prog, coreTI)
	}
}
