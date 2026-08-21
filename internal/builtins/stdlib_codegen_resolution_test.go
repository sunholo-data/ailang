package builtins

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	stdlibModuleDeclaration = regexp.MustCompile(`(?m)^module\s+(std/[A-Za-z0-9_]+)\s*$`)
	stdlibExportedFunction  = regexp.MustCompile(`(?m)^export\s+(?:pure\s+)?func\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

type crossModuleCodegenExemption struct {
	builtin string
	reason  string
}

// stdlibBuiltinFamilyPrefix independently ties each exporting module to the
// builtin family that implements it. These spellings come from the registered
// builtin names (not from mechanically transforming the module name); notably,
// std/string is implemented by the _str_ family.
var stdlibBuiltinFamilyPrefix = map[string]string{
	"std/ai":      "_ai_",
	"std/bytes":   "_bytes_",
	"std/clock":   "_clock_",
	"std/debug":   "_debug_",
	"std/env":     "_env_",
	"std/fs":      "_fs_",
	"std/io":      "_io_",
	"std/json":    "_json_",
	"std/list":    "_list_",
	"std/math":    "_math_",
	"std/net":     "_net_",
	"std/option":  "_option_",
	"std/process": "_process_",
	"std/result":  "_result_",
	"std/stream":  "_stream_",
	"std/string":  "_str_",
	"std/xml":     "_xml_",
	"std/zip":     "_zip_",
}

// These exports deliberately use an implementation whose builtin family name
// differs from the exporting module. The qualified claim remains the exporting
// module; this table pins the exceptional implementation identity and rationale.
var crossModuleCodegenExemptions = map[StdlibKey]crossModuleCodegenExemption{
	{"std/bytes", "fromString"}:           {"_str_fromString", "bytes conversion is implemented by the shared string/bytes bridge"},
	{"std/bytes", "toString"}:             {"_str_toString", "bytes conversion is implemented by the shared string/bytes bridge"},
	{"std/fs", "listDir"}:                 {"_io_listDir", "directory listing uses the shared IO effect helper"},
	{"std/list", "concat"}:                {"concat_List", "list concatenation uses the polymorphic core concatenation helper"},
	{"std/list", "maximumFloat"}:          {"_math_maximumFloat", "list reduction delegates pairwise comparison to the math helper"},
	{"std/list", "maximumInt"}:            {"_math_maximumInt", "list reduction delegates pairwise comparison to the math helper"},
	{"std/list", "maximumString"}:         {"_math_maximumString", "list reduction delegates pairwise comparison to the comparison helper"},
	{"std/list", "minimumFloat"}:          {"_math_minimumFloat", "list reduction delegates pairwise comparison to the math helper"},
	{"std/list", "minimumInt"}:            {"_math_minimumInt", "list reduction delegates pairwise comparison to the math helper"},
	{"std/list", "minimumString"}:         {"_math_minimumString", "list reduction delegates pairwise comparison to the comparison helper"},
	{"std/math", "floatToInt"}:            {"_float_to_int", "numeric conversion uses the conversion builtin family"},
	{"std/math", "intToFloat"}:            {"_int_to_float", "numeric conversion uses the conversion builtin family"},
	{"std/stream", "asyncExecProcess"}:    {"_process_asyncExec", "stream source creation is implemented by the process bridge"},
	{"std/stream", "asyncReadStdinLines"}: {"_process_asyncReadStdinLines", "stdin streaming is implemented by the process bridge"},
	{"std/string", "floatToStr"}:          {"_string_floatToStr", "legacy numeric formatting builtin predates the canonical string family spelling"},
	{"std/string", "intToStr"}:            {"_string_intToStr", "legacy numeric formatting builtin predates the canonical string family spelling"},
	{"std/string", "stringToFloat"}:       {"_stringToFloat", "legacy numeric parsing builtin predates the canonical string family spelling"},
	{"std/string", "stringToInt"}:         {"_stringToInt", "legacy numeric parsing builtin predates the canonical string family spelling"},
}

func TestStdlibCodegenResolutionIsModuleQualified(t *testing.T) {
	// Blind spot (declared, not closed by this structural gate): a codegen spec
	// can claim a genuine export of its own module, use that module's expected
	// builtin-family prefix, and still implement the wrong semantics. For example,
	// a fabricated _list_evilZipWith spec claiming std/list.zipWith and returning
	// xs passes every check below, while interpreted zipWith(+, [1,2,3],
	// [10,20,30]) produces [11,22,33] and compiled code produces [1,2,3]. Closing
	// this requires a source-level delegation/substitution invariant.
	paths, err := filepath.Glob(filepath.Join(stdlibSourceDir, "*.ail"))
	if err != nil {
		t.Fatalf("instrument failure: enumerate stdlib sources: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("instrument failure")
	}
	if len(StdlibIndex) == 0 {
		t.Fatal("instrument failure")
	}
	if len(stdlibBuiltinFamilyPrefix) == 0 {
		t.Fatal("instrument failure")
	}

	claims := make(map[StdlibKey][]string)
	for builtinName, meta := range Registry {
		if meta.GoCodegen == nil || meta.GoCodegen.StdlibName == "" {
			continue
		}
		spec := meta.GoCodegen
		key := StdlibKey{Module: spec.StdlibModule, Name: spec.StdlibName}
		claims[key] = append(claims[key], builtinName)
	}
	for key, names := range claims {
		if key.Module == "" {
			t.Errorf("qualified claim %q has an empty StdlibModule", key.Name)
		}
		if len(names) > 1 {
			sort.Strings(names)
			t.Errorf("duplicate qualified claim %s.%s: %v", key.Module, key.Name, names)
		}
	}

	qualifiedEntriesChecked := 0
	for key, builtinName := range StdlibIndex {
		prefix, ok := stdlibBuiltinFamilyPrefix[key.Module]
		if !ok || prefix == "" {
			t.Errorf("qualified claim %s.%s uses builtin %q but module has no builtin family prefix", key.Module, key.Name, builtinName)
			continue
		}
		qualifiedEntriesChecked++
		if strings.HasPrefix(builtinName, prefix) {
			continue
		}
		exemption, exempt := crossModuleCodegenExemptions[key]
		if !exempt || exemption.builtin != builtinName || exemption.reason == "" {
			t.Errorf("qualified claim %s.%s uses builtin %q outside expected family %q without a matching cross-module exemption", key.Module, key.Name, builtinName, prefix)
		}
	}
	if qualifiedEntriesChecked == 0 {
		t.Fatal("instrument failure")
	}

	exportCount := 0
	exported := make(map[StdlibKey]bool)
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		moduleMatch := stdlibModuleDeclaration.FindSubmatch(body)
		if moduleMatch == nil {
			t.Errorf("%s has no module declaration", path)
			continue
		}
		module := string(moduleMatch[1])
		for _, match := range stdlibExportedFunction.FindAllSubmatch(body, -1) {
			exportCount++
			name := string(match[1])
			exported[StdlibKey{Module: module, Name: name}] = true
			spec := GetCodegenSpecByStdlibName(module, name)
			if spec == nil {
				continue // Missing codegen is allowed and fails loudly during compilation.
			}
		}
	}
	if exportCount == 0 {
		t.Fatal("instrument failure")
	}

	for key, exemption := range crossModuleCodegenExemptions {
		if !exported[key] {
			t.Errorf("cross-module exemption %s.%s has no corresponding stdlib export", key.Module, key.Name)
		}
		if exemption.reason == "" {
			t.Errorf("cross-module exemption %s.%s has no reason", key.Module, key.Name)
		}
		if got := StdlibIndex[key]; got != exemption.builtin {
			t.Errorf("cross-module exemption %s.%s resolved to %q, want %q", key.Module, key.Name, got, exemption.builtin)
		}
	}

	t.Logf("scan summary: %d stdlib files, %d exported functions, %d qualified index entries", len(paths), exportCount, len(StdlibIndex))
}
