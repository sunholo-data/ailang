package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestShowPreview_Builtin(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	list := testctx.MakeList([]eval.Value{testctx.MakeInt(1), testctx.MakeInt(2), testctx.MakeInt(3)})

	result, err := showPreviewImpl(ctx.EffContext, []eval.Value{list, testctx.MakeInt(5)})
	require.NoError(t, err)
	assert.Equal(t, "[1, 2…(+4 chars)", testctx.GetString(result))

	result, err = showPreviewImpl(ctx.EffContext, []eval.Value{testctx.MakeString("abc"), testctx.MakeInt(10)})
	require.NoError(t, err)
	assert.Equal(t, `"abc"`, testctx.GetString(result), "a preview quotes a top-level string")

	_, err = showPreviewImpl(ctx.EffContext, []eval.Value{list, testctx.MakeString("5")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "_show_preview: arg 1 (maxChars)")

	spec, ok := GetSpec("_show_preview")
	require.True(t, ok, "_show_preview should be registered")
	assert.Equal(t, "std/string", spec.Module)
	assert.Equal(t, 2, spec.NumArgs)
	assert.True(t, spec.IsPure)
}
