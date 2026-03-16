package effects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/eval"
)

// M-DOCPARSE-DX M3: listDir tests

func TestFsListDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "beta.txt"), []byte("b"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gamma"), 0755))

	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	result, err := fsListDir(ctx, []eval.Value{
		&eval.StringValue{Value: dir},
	})
	require.NoError(t, err)

	list := result.(*eval.ListValue)
	assert.Equal(t, 3, len(list.Elements))
	assert.Equal(t, "alpha.txt", list.Elements[0].(*eval.StringValue).Value)
	assert.Equal(t, "beta.txt", list.Elements[1].(*eval.StringValue).Value)
	assert.Equal(t, "gamma", list.Elements[2].(*eval.StringValue).Value)
}

func TestFsListDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	result, err := fsListDir(ctx, []eval.Value{
		&eval.StringValue{Value: dir},
	})
	require.NoError(t, err)

	list := result.(*eval.ListValue)
	assert.Equal(t, 0, len(list.Elements))
}

func TestFsListDir_NonexistentDir(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))

	_, err := fsListDir(ctx, []eval.Value{
		&eval.StringValue{Value: "/nonexistent/path/xyz"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listDir")
}

func TestFsListDir_Sandbox(t *testing.T) {
	sandbox := t.TempDir()
	subdir := filepath.Join(sandbox, "data")
	require.NoError(t, os.Mkdir(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("x"), 0644))

	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("FS"))
	ctx.Env.Sandbox = sandbox

	result, err := fsListDir(ctx, []eval.Value{
		&eval.StringValue{Value: "data"},
	})
	require.NoError(t, err)

	list := result.(*eval.ListValue)
	assert.Equal(t, 1, len(list.Elements))
	assert.Equal(t, "file.txt", list.Elements[0].(*eval.StringValue).Value)
}
