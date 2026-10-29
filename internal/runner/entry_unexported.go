package runner

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// noExportsEntryError explains a run of a module that exports nothing. Entry
// resolution only happens for modules with exports, so such a run used to
// exit 0 with no output (#1797) — indistinguishable from success. Either way
// nothing runs, so the caller exits 1 with this message; it names the fix for
// a declared-but-unexported entry, and `ailang check` for a library module.
func noExportsEntryError(result *pipeline.Result, entry string) string {
	module := result.Interface.Module
	if mod := result.Modules[module]; mod != nil && mod.File != nil {
		for _, f := range mod.File.Funcs {
			if f.Name == entry && !f.IsExport {
				return fmt.Sprintf("entrypoint '%s' is declared in module %s but not exported; add 'export' (export func %s ...)",
					entry, module, entry)
			}
		}
	}
	return fmt.Sprintf("module %s exports no functions, so there is no entrypoint '%s' to run; add 'export func %s() ...', or use 'ailang check' to type-check a library module",
		module, entry, entry)
}
