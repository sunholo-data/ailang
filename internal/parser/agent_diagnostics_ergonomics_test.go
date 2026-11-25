package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestM1_IfThenBranchBlockDiagnostic(t *testing.T) {
	src := `module m
export pure func f(c: bool) -> int {
  if c then
    let x = 1;
    x + 1
  else
    0
}
`
	p := New(lexer.New(src, "test.ail"))
	_ = p.ParseFile()
	if len(p.Errors()) == 0 {
		t.Fatal("expected PAR_IF_BRANCH_BLOCK error, got none")
	}
	first := p.Errors()[0].Error()
	if !strings.Contains(first, "PAR_IF_BRANCH_BLOCK") {
		t.Fatalf("expected PAR_IF_BRANCH_BLOCK, got: %s", first)
	}
}

func TestM3_LocalFuncDiagnostic(t *testing.T) {
	src := `module m
export pure func outer(x: int) -> int {
  func helper(n: int) -> int {
    n + 1
  }
  helper(x)
}
`
	p := New(lexer.New(src, "test.ail"))
	_ = p.ParseFile()
	if len(p.Errors()) == 0 {
		t.Fatal("expected PAR_LOCAL_FUNC error, got none")
	}
	first := p.Errors()[0].Error()
	if !strings.Contains(first, "PAR_LOCAL_FUNC") || !strings.Contains(first, "helper") {
		t.Fatalf("expected PAR_LOCAL_FUNC mentioning helper, got: %s", first)
	}
}

func TestM3_HashCommentDiagnostic(t *testing.T) {
	src := `module m
# Helper comment in Python style
export pure func f(x: int) -> int = x + 1
`
	p := New(lexer.New(src, "test.ail"))
	_ = p.ParseFile()
	if len(p.Errors()) == 0 {
		t.Fatal("expected PAR_HASH_COMMENT error, got none")
	}
	first := p.Errors()[0].Error()
	if !strings.Contains(first, "PAR_HASH_COMMENT") || !strings.Contains(first, "--") {
		t.Fatalf("expected PAR_HASH_COMMENT suggesting '--', got: %s", first)
	}
}

func TestM5a_PipeOperatorParsesLeftAssociative(t *testing.T) {
	src := `module m
export pure func f(x: int) -> int {
  let y = x + 1 |> (\n. n * 2) |> g;
  y
}
`
	p := New(lexer.New(src, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors for pipe operator: %v", p.Errors())
	}
	if file == nil || len(file.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %#v", file)
	}
}

func TestM4_ContextualContractKeywordsAsRecordFields(t *testing.T) {
	src := `module m
type Requirement = { name: string, lo: int, hi: int }
type PackageDefinition = { name: string, requires: [Requirement], ensures: bool, invariant: int }

export pure func makePkg(n: string) -> PackageDefinition =
  { name: n, requires: [], ensures: true, invariant: 0 }

export pure func bumpPkg(p: PackageDefinition) -> PackageDefinition =
  { p | invariant: p.invariant + 1, requires: p.requires }

export pure func countReqs(p: PackageDefinition) -> int {
  let { requires: rs, ... } = p
  length(rs)
}
`
	p := New(lexer.New(src, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if file == nil || len(file.Funcs) != 3 {
		t.Fatalf("expected 3 funcs, got %#v", file)
	}
}
