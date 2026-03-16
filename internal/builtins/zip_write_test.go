package builtins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// M-DOCPARSE-DX M4: ZIP write tests

func makeZipEntry(name, content string) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name":    &eval.StringValue{Value: name},
			"content": &eval.StringValue{Value: content},
		},
	}
}

func TestZipCreateArchive_SingleEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))

	entries := &eval.ListValue{
		Elements: []eval.Value{
			makeZipEntry("hello.txt", "Hello, World!"),
		},
	}

	result, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		entries,
	})
	require.NoError(t, err)
	tagged := result.(*eval.TaggedValue)
	assert.Equal(t, "Ok", tagged.CtorName)

	// Verify file exists
	_, err = os.Stat(zipPath)
	assert.NoError(t, err)

	// Read back entry
	readResult, err := zipReadEntryImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "hello.txt"},
	})
	require.NoError(t, err)
	readTagged := readResult.(*eval.TaggedValue)
	assert.Equal(t, "Ok", readTagged.CtorName)
	assert.Equal(t, "Hello, World!", readTagged.Fields[0].(*eval.StringValue).Value)
}

func TestZipCreateArchive_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "multi.zip")

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))

	entries := &eval.ListValue{
		Elements: []eval.Value{
			makeZipEntry("a.txt", "alpha"),
			makeZipEntry("b.txt", "beta"),
			makeZipEntry("sub/c.txt", "gamma"),
		},
	}

	result, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		entries,
	})
	require.NoError(t, err)
	tagged := result.(*eval.TaggedValue)
	assert.Equal(t, "Ok", tagged.CtorName)

	// List entries
	listResult, err := zipListEntriesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
	})
	require.NoError(t, err)
	listTagged := listResult.(*eval.TaggedValue)
	assert.Equal(t, "Ok", listTagged.CtorName)
	list := listTagged.Fields[0].(*eval.ListValue)
	assert.Equal(t, 3, len(list.Elements))

	// Read back each entry
	for _, tc := range []struct{ name, content string }{
		{"a.txt", "alpha"},
		{"b.txt", "beta"},
		{"sub/c.txt", "gamma"},
	} {
		readResult, err := zipReadEntryImpl(ctx, []eval.Value{
			&eval.StringValue{Value: zipPath},
			&eval.StringValue{Value: tc.name},
		})
		require.NoError(t, err)
		readTagged := readResult.(*eval.TaggedValue)
		require.Equal(t, "Ok", readTagged.CtorName, "entry %s", tc.name)
		assert.Equal(t, tc.content, readTagged.Fields[0].(*eval.StringValue).Value)
	}
}

func TestZipCreateArchive_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))

	entries := &eval.ListValue{Elements: []eval.Value{}}

	result, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		entries,
	})
	require.NoError(t, err)
	tagged := result.(*eval.TaggedValue)
	assert.Equal(t, "Ok", tagged.CtorName)

	// List should be empty
	listResult, err := zipListEntriesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
	})
	require.NoError(t, err)
	listTagged := listResult.(*eval.TaggedValue)
	assert.Equal(t, "Ok", listTagged.CtorName)
	list := listTagged.Fields[0].(*eval.ListValue)
	assert.Equal(t, 0, len(list.Elements))
}

func TestZipCreateArchive_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))

	entries := &eval.ListValue{
		Elements: []eval.Value{
			makeZipEntry("../escape.txt", "bad"),
		},
	}

	result, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		entries,
	})
	require.NoError(t, err)
	tagged := result.(*eval.TaggedValue)
	assert.Equal(t, "Err", tagged.CtorName)
	assert.Contains(t, tagged.Fields[0].(*eval.StringValue).Value, "path traversal")
}

func TestZipCreateArchive_Sandbox(t *testing.T) {
	sandbox := t.TempDir()

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))
	ctx.Env.Sandbox = sandbox

	entries := &eval.ListValue{
		Elements: []eval.Value{
			makeZipEntry("test.txt", "sandboxed"),
		},
	}

	result, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "out.zip"},
		entries,
	})
	require.NoError(t, err)
	tagged := result.(*eval.TaggedValue)
	assert.Equal(t, "Ok", tagged.CtorName)

	// Verify file is in sandbox
	_, err = os.Stat(filepath.Join(sandbox, "out.zip"))
	assert.NoError(t, err)
}

func TestZipCreateArchive_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "roundtrip.zip")

	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("FS"))

	// Create archive
	entries := &eval.ListValue{
		Elements: []eval.Value{
			makeZipEntry("doc.xml", "<root><item>test</item></root>"),
			makeZipEntry("meta.txt", "version=1.0"),
		},
	}

	createResult, err := zipCreateArchiveImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		entries,
	})
	require.NoError(t, err)
	assert.Equal(t, "Ok", createResult.(*eval.TaggedValue).CtorName)

	// Read back and verify content matches
	for _, tc := range []struct{ name, content string }{
		{"doc.xml", "<root><item>test</item></root>"},
		{"meta.txt", "version=1.0"},
	} {
		readResult, err := zipReadEntryImpl(ctx, []eval.Value{
			&eval.StringValue{Value: zipPath},
			&eval.StringValue{Value: tc.name},
		})
		require.NoError(t, err)
		readTagged := readResult.(*eval.TaggedValue)
		require.Equal(t, "Ok", readTagged.CtorName)
		assert.Equal(t, tc.content, readTagged.Fields[0].(*eval.StringValue).Value)
	}
}
