package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// Run this exact harness on the baseline and fixed compiler against the SAME
// roots. Each row pins source bytes, parser failures and the full IFC diagnostics;
// unrelated HM failures cannot hide an IFC delta. The comparison is banked by
// sprint validation rather than treating every pre-existing leak as a test failure.
func TestIFCFunctionValueCorpusDelta(t *testing.T) {
	roots := config.IFCCorpus()
	if roots == "" {
		t.Skip("set AILANG_IFC_CORPUS to source roots for a pinned base/fixed comparison")
	}
	count := 0
	for i, root := range filepath.SplitList(roots) {
		before := count
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", ".claude", ".motoko", ".agent":
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".ail" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			name := fmt.Sprintf("root%d/%s", i, filepath.ToSlash(rel))
			sum := sha256.Sum256(source)
			p := parser.New(lexer.New(string(source), name))
			file := p.ParseFile()
			row := struct {
				Path, SourceHash string
				ParseErrors      []string
				Diagnostics      []*TypeCheckError
			}{Path: name, SourceHash: hex.EncodeToString(sum[:]), ParseErrors: nil}
			for _, parseErr := range p.Errors() {
				row.ParseErrors = append(row.ParseErrors, parseErr.Error())
			}
			if len(row.ParseErrors) == 0 {
				row.Diagnostics = CheckModuleIFC(file)
			}
			data, err := json.Marshal(row)
			if err != nil {
				return err
			}
			fmt.Printf("IFC_CORPUS\t%s\n", data)
			count++
			return nil
		})
		if err != nil {
			t.Fatalf("corpus root %q: %v", root, err)
		}
		if count == before {
			t.Fatalf("corpus root %q contained no AILANG files", root)
		}
	}
	if count == 0 {
		t.Fatal("corpus contained no AILANG files")
	}
	t.Logf("checked %d source snapshots", count)
}
