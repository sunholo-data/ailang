package runner

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// noExportsEntryError explains a run of a module that exports nothing. Entry
// resolution only happens for modules with exports, so such a run used to
// exit 0 with no output (#1797) — indistinguishable from success. A function
// with the entry's name that is declared but not exported is a hard error;
// any other export-less module returns "" and gets a stderr note from the
// caller, because example and test-only modules are run this way on purpose.
func noExportsEntryError(result *pipeline.Result, entry string) string {
	if result.Interface == nil {
		return ""
	}
	mod := result.Modules[result.Interface.Module]
	if mod == nil || mod.File == nil {
		return ""
	}
	for _, f := range mod.File.Funcs {
		if f.Name == entry && !f.IsExport {
			return fmt.Sprintf("entrypoint '%s' is declared in module %s but not exported; add 'export' (export func %s ...)",
				entry, result.Interface.Module, entry)
		}
	}
	return ""
}
