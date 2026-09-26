package builtins

// ============================================================================
// std/io builtins — Go codegen specs
// ============================================================================

func registerIOCodegenSpecs() {
	setSpec("_io_println", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Println", Signature: "func Println(v interface{}) interface{}",
			Body: `fmt.Println(Show(v))
	return struct{}{}`,
		},
		Imports:      []string{"fmt"},
		StdlibName:   "println",
		StdlibModule: "std/io",
	})
	setSpec("_io_print", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Print", Signature: "func Print(v interface{}) interface{}",
			Body: `fmt.Print(Show(v))
	return struct{}{}`,
		},
		Imports:      []string{"fmt"},
		StdlibName:   "print",
		StdlibModule: "std/io",
	})

	// Effect stubs — these panic with clear messages
	for _, spec := range []struct {
		name, module, stdlib, funcName, msg string
		numArgs                             int
	}{
		{"_fs_readFile", "std/fs", "readFile", "ReadFile", "FS", 1},
		{"_fs_writeFile", "std/fs", "writeFile", "WriteFile", "FS", 2},
		{"_fs_exists", "std/fs", "fileExists", "FileExists", "FS", 1},
		{"_fs_readFileBytes", "std/fs", "readFileBytes", "ReadFileBytes", "FS", 1},
		{"_fs_mkdir", "std/fs", "mkdir", "Mkdir", "FS", 1},
		{"_fs_mkdirAll", "std/fs", "mkdirAll", "MkdirAll", "FS", 1},
		{"_fs_isDir", "std/fs", "isDir", "IsDir", "FS", 1},
		{"_fs_isFile", "std/fs", "isFile", "IsFile", "FS", 1},
		{"_fs_removeFile", "std/fs", "removeFile", "RemoveFile", "FS", 1},
		{"_fs_rename", "std/fs", "renameFile", "RenameFile", "FS", 2},
		{"_zip_readEntry", "std/zip", "readEntry", "ReadEntry", "zip", 0},
		{"_zip_readEntryBytes", "std/zip", "readEntryBytes", "ReadEntryBytes", "zip", 0},
		{"_zip_listEntries", "std/zip", "listEntries", "ListEntries", "zip", 0},
		{"_zip_createArchive", "std/zip", "createArchive", "CreateArchive", "zip", 0},
		{"_env_getEnvOr", "std/env", "getEnvOr", "GetEnvOr", "Env", 2},
		{"_env_getArgs", "std/env", "getArgs", "GetArgs", "Env", 0},
		{"_env_getEnv", "std/env", "getEnv", "GetEnv", "Env", 1},
		{"_ai_call", "std/ai", "call", "Call", "AI", 0},
		{"_ai_callJson", "std/ai", "callJson", "CallJson", "AI", 0},
		{"_ai_callJsonSimple", "std/ai", "callJsonSimple", "CallJsonSimple", "AI", 0},
	} {
		funcName := spec.funcName
		msg := spec.msg
		registerIfMissing(spec.name, spec.numArgs, false, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: ` + msg + ` effect not available in compiled mode - provide a handler")`,
			},
			StdlibName:   spec.stdlib,
			StdlibModule: spec.module,
		})
	}
}
