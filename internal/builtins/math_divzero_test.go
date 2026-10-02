package builtins

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
)

// #1449: div_Int and mod_Int report a zero divisor as RT001, the code the VM
// and the Num[int] dictionary use, not the unregistered "RT_DIV0".
func TestIntDivZero_Builtins(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	for name, op := range map[string]string{"div_Int": ailerrors.OpDivision, "mod_Int": ailerrors.OpModulo} {
		spec, ok := GetSpec(name)
		require.True(t, ok, name)
		_, err := spec.Impl(ctx.EffContext, []eval.Value{testctx.MakeInt(7), testctx.MakeInt(0)})
		var dz *ailerrors.DivByZeroError
		require.True(t, errors.As(err, &dz), "%s(7, 0) error = %v, want *DivByZeroError", name, err)
		require.Equal(t, op, dz.Op, name)

		v, err := spec.Impl(ctx.EffContext, []eval.Value{testctx.MakeInt(7), testctx.MakeInt(2)})
		require.NoError(t, err, name)
		want := map[string]int{"div_Int": 3, "mod_Int": 1}[name]
		require.Equal(t, want, testctx.GetInt(v), name)
	}
}
