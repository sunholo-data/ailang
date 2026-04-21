package compiler

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/vm"
)

// TestCompile_LineInfo_Populated verifies that the compiler stamps each
// emitted instruction with the source line of the statement that produced it.
func TestCompile_LineInfo_Populated(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "test",
				Exported: true,
				File:     "lineinfo.ail",
				Line:     3,
				Body: []stmt.Stmt{
					stmt.VarDecl{Name: "x", Value: stmt.LitInt{Value: 10}, Line: 4},
					stmt.VarDecl{Name: "y", Value: stmt.LitInt{Value: 0}, Line: 5},
				},
				Return: stmt.BinOp{
					Op:    stmt.OpDiv,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.VarRef{Name: "y"},
				},
			},
		},
	}

	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	proto := img.Prototypes[img.EntryPoint]

	if proto.File != "lineinfo.ail" {
		t.Errorf("proto.File = %q, want %q", proto.File, "lineinfo.ail")
	}
	if len(proto.LineInfo) != len(proto.Instructions) {
		t.Fatalf("LineInfo length %d != Instructions length %d",
			len(proto.LineInfo), len(proto.Instructions))
	}
	// At least one instruction should have line 4, one line 5, and the
	// trailing return (no line of its own) should fall back to the
	// FuncDecl's line (3).
	seen := map[int]bool{}
	for _, l := range proto.LineInfo {
		seen[l] = true
	}
	for _, want := range []int{3, 4, 5} {
		if !seen[want] {
			t.Errorf("expected LineInfo to contain line %d, got %v", want, proto.LineInfo)
		}
	}
}

// TestVM_DivByZero_ReportsLine verifies that a divide-by-zero at runtime
// surfaces a VMError with the source line of the offending statement.
func TestVM_DivByZero_ReportsLine(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "test",
				Exported: true,
				File:     "divzero.ail",
				Line:     1,
				Body: []stmt.Stmt{
					stmt.VarDecl{Name: "n", Value: stmt.LitInt{Value: 10}, Line: 2},
					stmt.VarDecl{Name: "d", Value: stmt.LitInt{Value: 0}, Line: 3},
					stmt.ReturnStmt{
						Value: stmt.BinOp{
							Op:    stmt.OpDiv,
							Left:  stmt.VarRef{Name: "n"},
							Right: stmt.VarRef{Name: "d"},
						},
						Line: 7,
					},
				},
			},
		},
	}

	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	machine := vm.NewVM(img)
	_, err = machine.Run(img.Prototypes[img.EntryPoint], nil)
	if err == nil {
		t.Fatalf("expected divide-by-zero error, got none")
	}

	var vmErr *vm.VMError
	if !errors.As(err, &vmErr) {
		t.Fatalf("expected *vm.VMError, got %T: %v", err, err)
	}
	if vmErr.Line != 7 {
		t.Errorf("VMError.Line = %d, want 7", vmErr.Line)
	}
	if vmErr.File != "divzero.ail" {
		t.Errorf("VMError.File = %q, want %q", vmErr.File, "divzero.ail")
	}
	if !strings.Contains(vmErr.Error(), "divzero.ail:7") {
		t.Errorf("error string %q should contain %q", vmErr.Error(), "divzero.ail:7")
	}
}

// TestVM_DivByZero_NoLineInfo verifies the error formatter degrades gracefully
// when File/Line are unavailable (e.g. hand-built test prototypes).
func TestVM_DivByZero_NoLineInfo(t *testing.T) {
	// Use the high-level helper which builds a FuncDecl with no File/Line.
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "test",
				Exported: true,
				Return: stmt.BinOp{
					Op:    stmt.OpDiv,
					Left:  stmt.LitInt{Value: 1},
					Right: stmt.LitInt{Value: 0},
				},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	_, err = vm.NewVM(img).Run(img.Prototypes[img.EntryPoint], nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var vmErr *vm.VMError
	if !errors.As(err, &vmErr) {
		t.Fatalf("expected *vm.VMError, got %T", err)
	}
	if vmErr.File != "" || vmErr.Line != 0 {
		t.Errorf("expected blank file/line, got file=%q line=%d", vmErr.File, vmErr.Line)
	}
	// Just make sure formatting doesn't panic and does not contain "at line 0".
	s := vmErr.Error()
	_ = bytecode.OpDiv // keep import
	if strings.Contains(s, "line 0") {
		t.Errorf("error string should not say 'line 0': %q", s)
	}
}
