package elaborate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

func elaborateSource(t *testing.T, src string) error {
	t.Helper()
	p := parser.New(lexer.New(src, "test/derive.ail"))
	file := p.ParseFile()
	require.Empty(t, p.Errors(), "unexpected parse errors: %v", p.Errors())

	elab := elaborate.NewElaboratorWithPath("test/derive")
	elab.AddBuiltinsToGlobalEnv()
	_, err := elab.ElaborateFile(file)
	return err
}

// TestDerivingJson_PolymorphicRejected verifies TST-DJ-03:
// polymorphic types with deriving (Json) fail cleanly with TC_DERIVE_JSON_POLY.
func TestDerivingJson_PolymorphicRejected(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "polymorphic record",
			src: `module test/derive
import std/json (Json)
export type Box[a] = { value: a } deriving (Json)
`,
		},
		{
			name: "polymorphic sum ADT with Eq and Json",
			src: `module test/derive
import std/json (Json)
type Tree[a] = Leaf(a) | Branch(Tree[a], Tree[a]) deriving (Eq, Json)
`,
		},
		{
			name: "polymorphic type without imports",
			src: `module test/derive
type Pair[a, b] = { fst: a, snd: b } deriving (Json)
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := elaborateSource(t, tc.src)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "TC_DERIVE_JSON_POLY")
		})
	}
}

func TestDerivingJson_MissingStdJsonImportRejected(t *testing.T) {
	src := `module test/derive
export type User = { name: string, age: int } deriving (Json)
`
	err := elaborateSource(t, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IMP_DERIVE_JSON")
	assert.Contains(t, err.Error(), "import std/json (Json)")
}

func TestDerivingJson_UnsupportedFieldTypeRejected(t *testing.T) {
	src := `module test/derive
import std/json (Json)
type CallbackHolder = { cb: int -> int } deriving (Json)
`
	err := elaborateSource(t, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TC_DERIVE_JSON_UNSUPPORTED")
}

func TestDerivingJson_SynthesizesExportedAndPrivateFunctions(t *testing.T) {
	src := `module test/derive
import std/json (Json)

export type PublicUser = { name: string, age: int, bio: Option[string] } deriving (Json)
type InternalTag = Empty | Label(string) deriving (Json)
`
	p := parser.New(lexer.New(src, "test/derive.ail"))
	file := p.ParseFile()
	require.Empty(t, p.Errors())

	elab := elaborate.NewElaboratorWithPath("test/derive")
	elab.AddBuiltinsToGlobalEnv()
	elab.RegisterConstructor("Json", "JNull", 0, true, 0)
	elab.RegisterConstructor("Option", "None", 0, true, 1)
	elab.RegisterConstructor("Option", "Some", 1, true, 1)
	elab.RegisterConstructor("Result", "Ok", 1, true, 2)
	elab.RegisterConstructor("Result", "Err", 1, true, 2)

	prog, err := elab.ElaborateFile(file)
	require.NoError(t, err)
	require.NotNil(t, prog)

	require.Contains(t, prog.Meta, "encodePublicUser")
	require.Contains(t, prog.Meta, "decodePublicUser")
	assert.True(t, prog.Meta["encodePublicUser"].IsExport)
	assert.True(t, prog.Meta["decodePublicUser"].IsExport)
	assert.True(t, prog.Meta["encodePublicUser"].IsPure)
	assert.True(t, prog.Meta["decodePublicUser"].IsPure)

	require.Contains(t, prog.Meta, "encodeInternalTag")
	require.Contains(t, prog.Meta, "decodeInternalTag")
	assert.False(t, prog.Meta["encodeInternalTag"].IsExport)
	assert.False(t, prog.Meta["decodeInternalTag"].IsExport)
}

func TestDerivingJson_CrossModuleLoaderExportPopulated(t *testing.T) {
	tmpDir := t.TempDir()
	modAPath := filepath.Join(tmpDir, "mod_a.ail")
	modASrc := `module mod_a
import std/json (Json)
export type Item = { id: int, title: string } deriving (Json)
`
	require.NoError(t, os.WriteFile(modAPath, []byte(modASrc), 0644))

	ml := loader.NewModuleLoader(tmpDir)
	loadedA, err := ml.Load("mod_a")
	require.NoError(t, err)

	scope := types.NewTypeScope("mod_a")
	scope.BindLocal("Item")
	scope.BindExplicit("Json", types.TypeRef{Module: "std/json", Name: "Json"})

	elab := elaborate.NewElaboratorWithPath("mod_a")
	elab.SetTypeScope(scope)
	elab.SetModuleLoader(ml)
	elab.AddBuiltinsToGlobalEnv()

	_, err = elab.ElaborateFile(loadedA.File)
	require.NoError(t, err)

	encDecl, err := ml.GetExport("mod_a", "encodeItem")
	require.NoError(t, err)
	require.NotNil(t, encDecl)
	decDecl, err := ml.GetExport("mod_a", "decodeItem")
	require.NoError(t, err)
	require.NotNil(t, decDecl)
}

func TestDerivingJson_DuplicateManualFunctionCollision(t *testing.T) {
	src := `module test/derive
import std/json (Json, JNull)

type Item = { id: int } deriving (Json)

pure func encodeItem(v: Item) -> Json = JNull
`
	err := elaborateSource(t, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MOD007")
	assert.Contains(t, err.Error(), "encodeItem")
}
