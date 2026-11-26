package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/format"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TestDerivingJson_ParseAndFormatRoundTrip verifies TST-DJ-01:
// deriving (Json) and deriving (Eq, Json) parse on records and ADTs,
// and ailang fmt round-trips them idempotently.
func TestDerivingJson_ParseAndFormatRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantTypeName string
		wantExported bool
		wantDeriving []ast.DeriveKind
	}{
		{
			name:         "record deriving Json",
			src:          "module test/dj\n\nexport type Config = { host: string, port: int } deriving (Json)\n",
			wantTypeName: "Config",
			wantExported: true,
			wantDeriving: []ast.DeriveKind{ast.DeriveJson},
		},
		{
			name:         "record deriving Eq and Json",
			src:          "module test/dj\n\ntype Point = { x: int, y: int } deriving (Eq, Json)\n",
			wantTypeName: "Point",
			wantExported: false,
			wantDeriving: []ast.DeriveKind{ast.DeriveEq, ast.DeriveJson},
		},
		{
			name:         "sum ADT deriving Json and Eq",
			src:          "module test/dj\n\nexport type Shape = Point | Circle(float) | Named(label: string, size: int) deriving (Json, Eq)\n",
			wantTypeName: "Shape",
			wantExported: true,
			wantDeriving: []ast.DeriveKind{ast.DeriveJson, ast.DeriveEq},
		},
		{
			name:         "leading-pipe ADT deriving Json",
			src:          "module test/dj\n\ntype Status =\n  | Idle\n  | Running(int) deriving (Json)\n",
			wantTypeName: "Status",
			wantExported: false,
			wantDeriving: []ast.DeriveKind{ast.DeriveJson},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.New(lexer.New(tc.src, "test.ail"))
			file := p.ParseFile()
			require.Empty(t, p.Errors(), "unexpected parse errors: %v", p.Errors())
			require.Len(t, file.Statements, 1)

			td, ok := file.Statements[0].(*ast.TypeDecl)
			require.True(t, ok, "expected *ast.TypeDecl, got %T", file.Statements[0])
			assert.Equal(t, tc.wantTypeName, td.Name)
			assert.Equal(t, tc.wantExported, td.Exported)
			assert.Equal(t, tc.wantDeriving, td.Deriving)

			formattedBytes, err := format.Source(&ast.Program{File: file}, format.Options{})
			require.NoError(t, err, "format.Source failed")
			formatted := string(formattedBytes)

			// Re-parse formatted output and verify identical Deriving slice + idempotence
			p2 := parser.New(lexer.New(formatted, "test.ail"))
			file2 := p2.ParseFile()
			require.Empty(t, p2.Errors(), "re-parse of formatted source failed: %v\nformatted:\n%s", p2.Errors(), formatted)
			require.Len(t, file2.Statements, 1)
			td2, ok := file2.Statements[0].(*ast.TypeDecl)
			require.True(t, ok)
			assert.Equal(t, tc.wantDeriving, td2.Deriving)

			formatted2Bytes, err := format.Source(&ast.Program{File: file2}, format.Options{})
			require.NoError(t, err)
			assert.Equal(t, formatted, string(formatted2Bytes), "formatter must be idempotent")
		})
	}
}

func TestDerivingUnsupportedStillRejected(t *testing.T) {
	src := "module test/dj\ntype Foo = { x: int } deriving (Ord)\n"
	p := parser.New(lexer.New(src, "test.ail"))
	_ = p.ParseFile()
	require.NotEmpty(t, p.Errors())
	assert.Contains(t, p.Errors()[0].Error(), "Ord")
	assert.Contains(t, p.Errors()[0].Error(), "Json")
}
