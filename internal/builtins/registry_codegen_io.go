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
		Imports:    []string{"fmt"},
		StdlibName: "println",
	})
	setSpec("_io_print", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Print", Signature: "func Print(v interface{}) interface{}",
			Body: `fmt.Print(Show(v))
	return struct{}{}`,
		},
		Imports:    []string{"fmt"},
		StdlibName: "print",
	})

	// Effect stubs — these panic with clear messages
	for _, spec := range []struct {
		name, stdlib, funcName, msg string
		numArgs                     int
	}{
		{"_fs_readFile", "readFile", "ReadFile", "FS", 1},
		{"_fs_writeFile", "writeFile", "WriteFile", "FS", 2},
		{"_fs_exists", "fileExists", "FileExists", "FS", 1},
		{"_fs_readFileBytes", "readFileBytes", "ReadFileBytes", "FS", 1},
		{"_fs_mkdir", "mkdir", "Mkdir", "FS", 1},
		{"_fs_mkdirAll", "mkdirAll", "MkdirAll", "FS", 1},
		{"_fs_isDir", "isDir", "IsDir", "FS", 1},
		{"_fs_isFile", "isFile", "IsFile", "FS", 1},
		{"_fs_removeFile", "removeFile", "RemoveFile", "FS", 1},
		{"_zip_readEntry", "readEntry", "ReadEntry", "zip", 0},
		{"_zip_readEntryBytes", "readEntryBytes", "ReadEntryBytes", "zip", 0},
		{"_zip_listEntries", "listEntries", "ListEntries", "zip", 0},
		{"_zip_createArchive", "createArchive", "CreateArchive", "zip", 0},
		{"_env_getEnvOr", "getEnvOr", "GetEnvOr", "Env", 2},
		{"_env_getArgs", "getArgs", "GetArgs", "Env", 0},
		{"_env_getEnv", "getEnv", "GetEnv", "Env", 1},
		{"_ai_call", "call", "Call", "AI", 0},
		{"_ai_callJson", "callJson", "CallJson", "AI", 0},
		{"_ai_callJsonSimple", "callJsonSimple", "CallJsonSimple", "AI", 0},
	} {
		funcName := spec.funcName
		msg := spec.msg
		registerIfMissing(spec.name, spec.numArgs, false, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: ` + msg + ` effect not available in compiled mode - provide a handler")`,
			},
			StdlibName: spec.stdlib,
		})
	}
}
