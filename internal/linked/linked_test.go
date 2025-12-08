package linked

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

func TestNewLinker(t *testing.T) {
	linker := NewLinker()
	assert.NotNil(t, linker)
}

func TestLinker_Link_Nil(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	result, err := linker.Link(nil, dictReg)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestLinker_Link_Var(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	varExpr := &core.Var{
		CoreNode: core.CoreNode{},
		Name:     "x",
	}

	result, err := linker.Link(varExpr, dictReg)
	assert.NoError(t, err)
	assert.Equal(t, varExpr, result)
}

func TestLinker_Link_Lit(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	litExpr := &core.Lit{
		CoreNode: core.CoreNode{},
		Kind:     core.IntLit,
		Value:    42,
	}

	result, err := linker.Link(litExpr, dictReg)
	assert.NoError(t, err)
	assert.Equal(t, litExpr, result)
}

func TestLinker_Link_VarGlobal(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	globalExpr := &core.VarGlobal{
		CoreNode: core.CoreNode{},
		Ref: core.GlobalRef{
			Module: "std/io",
			Name:   "print",
		},
	}

	result, err := linker.Link(globalExpr, dictReg)
	assert.NoError(t, err)
	assert.Equal(t, globalExpr, result)
}

func TestLinker_Link_DictRef(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	dictRefExpr := &core.DictRef{
		CoreNode:  core.CoreNode{},
		ClassName: "Num",
		TypeName:  "Int",
	}

	result, err := linker.Link(dictRefExpr, dictReg)
	assert.NoError(t, err)
	// Should return the same DictRef (for now, until full linking is implemented)
	assert.Equal(t, dictRefExpr, result)
}

func TestLinker_Link_DictApp(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	dictAppExpr := &core.DictApp{
		CoreNode: core.CoreNode{},
		Dict: &core.DictRef{
			CoreNode:  core.CoreNode{},
			ClassName: "Num",
			TypeName:  "Int",
		},
		Method: "add",
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: 1},
			&core.Lit{Kind: core.IntLit, Value: 2},
		},
	}

	result, err := linker.Link(dictAppExpr, dictReg)
	assert.NoError(t, err)

	linkedApp, ok := result.(*core.DictApp)
	require.True(t, ok)
	assert.Equal(t, "add", linkedApp.Method)
	assert.Equal(t, 2, len(linkedApp.Args))
}

func TestLinker_Link_Let(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	letExpr := &core.Let{
		CoreNode: core.CoreNode{},
		Name:     "x",
		Value:    &core.Lit{Kind: core.IntLit, Value: 10},
		Body:     &core.Var{Name: "x"},
	}

	result, err := linker.Link(letExpr, dictReg)
	assert.NoError(t, err)

	linkedLet, ok := result.(*core.Let)
	require.True(t, ok)
	assert.Equal(t, "x", linkedLet.Name)
	assert.NotNil(t, linkedLet.Value)
	assert.NotNil(t, linkedLet.Body)
}

func TestLinker_Link_LetRec(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	letRecExpr := &core.LetRec{
		CoreNode: core.CoreNode{},
		Bindings: []core.RecBinding{
			{
				Name:  "f",
				Value: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
			},
			{
				Name:  "g",
				Value: &core.Lambda{Params: []string{"y"}, Body: &core.Var{Name: "y"}},
			},
		},
		Body: &core.Var{Name: "f"},
	}

	result, err := linker.Link(letRecExpr, dictReg)
	assert.NoError(t, err)

	linkedLetRec, ok := result.(*core.LetRec)
	require.True(t, ok)
	assert.Equal(t, 2, len(linkedLetRec.Bindings))
	assert.Equal(t, "f", linkedLetRec.Bindings[0].Name)
	assert.Equal(t, "g", linkedLetRec.Bindings[1].Name)
	assert.NotNil(t, linkedLetRec.Body)
}

func TestLinker_Link_Lambda(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	lambdaExpr := &core.Lambda{
		CoreNode: core.CoreNode{},
		Params:   []string{"x", "y"},
		Body:     &core.Var{Name: "x"},
	}

	result, err := linker.Link(lambdaExpr, dictReg)
	assert.NoError(t, err)

	linkedLambda, ok := result.(*core.Lambda)
	require.True(t, ok)
	assert.Equal(t, 2, len(linkedLambda.Params))
	assert.Equal(t, "x", linkedLambda.Params[0])
	assert.Equal(t, "y", linkedLambda.Params[1])
	assert.NotNil(t, linkedLambda.Body)
}

func TestLinker_Link_App(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	appExpr := &core.App{
		CoreNode: core.CoreNode{},
		Func:     &core.Var{Name: "f"},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: 1},
			&core.Lit{Kind: core.IntLit, Value: 2},
		},
	}

	result, err := linker.Link(appExpr, dictReg)
	assert.NoError(t, err)

	linkedApp, ok := result.(*core.App)
	require.True(t, ok)
	assert.NotNil(t, linkedApp.Func)
	assert.Equal(t, 2, len(linkedApp.Args))
}

func TestLinker_Link_BinOp(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	binOpExpr := &core.BinOp{
		CoreNode: core.CoreNode{},
		Op:       "+",
		Left:     &core.Lit{Kind: core.IntLit, Value: 1},
		Right:    &core.Lit{Kind: core.IntLit, Value: 2},
	}

	result, err := linker.Link(binOpExpr, dictReg)
	assert.NoError(t, err)

	linkedBinOp, ok := result.(*core.BinOp)
	require.True(t, ok)
	assert.Equal(t, "+", linkedBinOp.Op)
	assert.NotNil(t, linkedBinOp.Left)
	assert.NotNil(t, linkedBinOp.Right)
}

func TestLinker_Link_UnOp(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	unOpExpr := &core.UnOp{
		CoreNode: core.CoreNode{},
		Op:       "-",
		Operand:  &core.Lit{Kind: core.IntLit, Value: 42},
	}

	result, err := linker.Link(unOpExpr, dictReg)
	assert.NoError(t, err)

	linkedUnOp, ok := result.(*core.UnOp)
	require.True(t, ok)
	assert.Equal(t, "-", linkedUnOp.Op)
	assert.NotNil(t, linkedUnOp.Operand)
}

func TestLinker_Link_If(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	ifExpr := &core.If{
		CoreNode: core.CoreNode{},
		Cond:     &core.Lit{Kind: core.BoolLit, Value: true},
		Then:     &core.Lit{Kind: core.IntLit, Value: 1},
		Else:     &core.Lit{Kind: core.IntLit, Value: 2},
	}

	result, err := linker.Link(ifExpr, dictReg)
	assert.NoError(t, err)

	linkedIf, ok := result.(*core.If)
	require.True(t, ok)
	assert.NotNil(t, linkedIf.Cond)
	assert.NotNil(t, linkedIf.Then)
	assert.NotNil(t, linkedIf.Else)
}

func TestLinker_Link_Match(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	matchExpr := &core.Match{
		CoreNode:  core.CoreNode{},
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: 1},
				Body:    &core.Lit{Kind: core.StringLit, Value: "one"},
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    &core.Lit{Kind: core.StringLit, Value: "other"},
			},
		},
		Exhaustive: true,
	}

	result, err := linker.Link(matchExpr, dictReg)
	assert.NoError(t, err)

	linkedMatch, ok := result.(*core.Match)
	require.True(t, ok)
	assert.NotNil(t, linkedMatch.Scrutinee)
	assert.Equal(t, 2, len(linkedMatch.Arms))
	assert.True(t, linkedMatch.Exhaustive)
	assert.NotNil(t, linkedMatch.Arms[0].Body)
	assert.NotNil(t, linkedMatch.Arms[1].Body)
}

func TestLinker_Link_Record(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	recordExpr := &core.Record{
		CoreNode: core.CoreNode{},
		Fields: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: 10},
			"y": &core.Lit{Kind: core.IntLit, Value: 20},
		},
	}

	result, err := linker.Link(recordExpr, dictReg)
	assert.NoError(t, err)

	linkedRecord, ok := result.(*core.Record)
	require.True(t, ok)
	assert.Equal(t, 2, len(linkedRecord.Fields))
	assert.NotNil(t, linkedRecord.Fields["x"])
	assert.NotNil(t, linkedRecord.Fields["y"])
}

func TestLinker_Link_RecordAccess(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	recordAccessExpr := &core.RecordAccess{
		CoreNode: core.CoreNode{},
		Record:   &core.Var{Name: "r"},
		Field:    "x",
	}

	result, err := linker.Link(recordAccessExpr, dictReg)
	assert.NoError(t, err)

	linkedAccess, ok := result.(*core.RecordAccess)
	require.True(t, ok)
	assert.Equal(t, "x", linkedAccess.Field)
	assert.NotNil(t, linkedAccess.Record)
}

func TestLinker_Link_List(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	listExpr := &core.List{
		CoreNode: core.CoreNode{},
		Elements: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: 1},
			&core.Lit{Kind: core.IntLit, Value: 2},
			&core.Lit{Kind: core.IntLit, Value: 3},
		},
	}

	result, err := linker.Link(listExpr, dictReg)
	assert.NoError(t, err)

	linkedList, ok := result.(*core.List)
	require.True(t, ok)
	assert.Equal(t, 3, len(linkedList.Elements))
}

func TestLinker_Link_Intrinsic(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	intrinsicExpr := &core.Intrinsic{
		CoreNode: core.CoreNode{},
		Op:       core.OpAdd,
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: 1},
			&core.Lit{Kind: core.IntLit, Value: 2},
		},
	}

	result, err := linker.Link(intrinsicExpr, dictReg)
	assert.NoError(t, err)

	linkedIntrinsic, ok := result.(*core.Intrinsic)
	require.True(t, ok)
	assert.Equal(t, core.OpAdd, linkedIntrinsic.Op)
	assert.Equal(t, 2, len(linkedIntrinsic.Args))
}

func TestLinker_Link_DictAbs(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	dictAbsExpr := &core.DictAbs{
		CoreNode: core.CoreNode{},
		Params: []core.DictParam{
			{Name: "d", ClassName: "Num", Type: "Int"},
		},
		Body: &core.Var{Name: "x"},
	}

	result, err := linker.Link(dictAbsExpr, dictReg)
	assert.NoError(t, err)
	// DictAbs should pass through unchanged
	assert.Equal(t, dictAbsExpr, result)
}

func TestLinker_Link_NestedExpressions(t *testing.T) {
	linker := NewLinker()
	dictReg := types.NewDictionaryRegistry()

	// Test nested expressions: let x = (1 + 2) in if x > 0 then x else 0
	nestedExpr := &core.Let{
		CoreNode: core.CoreNode{},
		Name:     "x",
		Value: &core.BinOp{
			Op:    "+",
			Left:  &core.Lit{Kind: core.IntLit, Value: 1},
			Right: &core.Lit{Kind: core.IntLit, Value: 2},
		},
		Body: &core.If{
			Cond: &core.BinOp{
				Op:    ">",
				Left:  &core.Var{Name: "x"},
				Right: &core.Lit{Kind: core.IntLit, Value: 0},
			},
			Then: &core.Var{Name: "x"},
			Else: &core.Lit{Kind: core.IntLit, Value: 0},
		},
	}

	result, err := linker.Link(nestedExpr, dictReg)
	assert.NoError(t, err)

	linkedLet, ok := result.(*core.Let)
	require.True(t, ok)
	assert.Equal(t, "x", linkedLet.Name)

	// Check the nested BinOp was linked
	_, ok = linkedLet.Value.(*core.BinOp)
	assert.True(t, ok)

	// Check the nested If was linked
	linkedIf, ok := linkedLet.Body.(*core.If)
	require.True(t, ok)
	assert.NotNil(t, linkedIf.Cond)
}

func TestLinkExprs(t *testing.T) {
	dictReg := types.NewDictionaryRegistry()

	exprs := []core.CoreExpr{
		&core.Lit{Kind: core.IntLit, Value: 1},
		&core.Lit{Kind: core.IntLit, Value: 2},
		&core.Lit{Kind: core.IntLit, Value: 3},
	}

	result := linkExprs(exprs, dictReg)
	assert.Equal(t, 3, len(result))
	for i, expr := range result {
		lit, ok := expr.(*core.Lit)
		assert.True(t, ok)
		assert.Equal(t, i+1, lit.Value.(int))
	}
}

func TestLinkExprs_Empty(t *testing.T) {
	dictReg := types.NewDictionaryRegistry()

	result := linkExprs([]core.CoreExpr{}, dictReg)
	// Empty slice returns nil slice (not allocated)
	assert.Equal(t, 0, len(result))
}

func TestLinkExprs_WithNested(t *testing.T) {
	dictReg := types.NewDictionaryRegistry()

	exprs := []core.CoreExpr{
		&core.BinOp{
			Op:    "+",
			Left:  &core.Lit{Kind: core.IntLit, Value: 1},
			Right: &core.Lit{Kind: core.IntLit, Value: 2},
		},
		&core.Var{Name: "x"},
	}

	result := linkExprs(exprs, dictReg)
	assert.Equal(t, 2, len(result))

	binOp, ok := result[0].(*core.BinOp)
	assert.True(t, ok)
	assert.Equal(t, "+", binOp.Op)

	varExpr, ok := result[1].(*core.Var)
	assert.True(t, ok)
	assert.Equal(t, "x", varExpr.Name)
}
