package vm

import (
	"io/fs"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/std"
)

// stdADTOrdinals maps "module.Type" to constructor ordinals in declaration
// order, read once from the embedded stdlib. The compiler assigns VM tags
// from the same declaration order, so an evaluator TaggedValue of any std
// ADT can cross the bridge. bytecode.StdADTTag is a hand-kept subset; a std
// type missing from it (std/env EnvError, #1864) used to fail the call.
var stdADTOrdinals = sync.OnceValue(func() map[string]map[string]int {
	out := map[string]map[string]int{}
	_ = fs.WalkDir(std.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".ail") {
			return nil
		}
		src, err := fs.ReadFile(std.FS, path)
		if err != nil {
			return nil
		}
		p := parser.New(lexer.New(string(src), path))
		file := p.ParseFile()
		if file == nil || file.Module == nil || len(p.Errors()) > 0 {
			return nil
		}
		for _, decl := range file.Decls {
			td, ok := decl.(*ast.TypeDecl)
			if !ok {
				continue
			}
			adt, ok := td.Definition.(*ast.AlgebraicType)
			if !ok {
				continue
			}
			ords := make(map[string]int, len(adt.Constructors))
			for i, c := range adt.Constructors {
				ords[c.Name] = i
			}
			out[file.Module.Path+"."+td.Name] = ords
		}
		return nil
	})
	return out
})

// stdADTTag returns the VM tag for a std ADT constructor: the hand-kept
// bytecode.StdADTTag first, then the declaration order in the embedded
// stdlib source.
func stdADTTag(modulePath, typeName, ctor string) (int, bool) {
	if tag, ok := bytecode.StdADTTag(modulePath, typeName, ctor); ok {
		return tag, true
	}
	if !strings.HasPrefix(modulePath, "std/") {
		return 0, false
	}
	tag, ok := stdADTOrdinals()[modulePath+"."+typeName][ctor]
	return tag, ok
}
