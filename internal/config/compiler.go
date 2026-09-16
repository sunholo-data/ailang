package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Compiler and runtime knobs (M-V1-SIMPLIFY-S4 M2). The DEBUG_* switches
// stay where they are by rule; these are the operator-facing ones.
const (
	EnvNoPrelude         = "AILANG_NO_PRELUDE"
	EnvRecordsV2         = "AILANG_RECORDS_V2"
	EnvDTree             = "AILANG_DTREE"
	EnvNoCache           = "AILANG_NO_CACHE"
	EnvRelaxModules      = "AILANG_RELAX_MODULES"
	EnvSeed              = "AILANG_SEED"
	EnvTZ                = "TZ"
	EnvLang              = "LANG"
	EnvFSSandbox         = "AILANG_FS_SANDBOX"
	EnvFSSandboxDebug    = "AILANG_FS_SANDBOX_DEBUG"
	EnvFSMaxBytes        = "AILANG_FS_MAX_BYTES"
	EnvRedactEnv         = "AILANG_REDACT_ENV"
	EnvMetrics           = "AILANG_METRICS"
	EnvMetricsDebug      = "AILANG_METRICS_DEBUG"
	EnvMetricsVerbose    = "AILANG_METRICS_VERBOSE"
	EnvHubURL            = "AILANG_HUB_URL"
	EnvDumpSMT           = "AILANG_DUMP_SMT"
	EnvNoVersionWarnings = "AILANG_NO_VERSION_WARNINGS"
	EnvQuietWarnings     = "AILANG_QUIET_WARNINGS"
	EnvGOGC              = "GOGC"
)

var compilerVars = []Var{
	{EnvNoPrelude, "0", AreaCompiler, "1 starts the type checker with an empty instance environment instead of auto-importing std/prelude's Eq, Ord, Num and Show instances."},
	{EnvRecordsV2, "0", AreaCompiler, "1 enables the records-v2 type-checker path."},
	{EnvDTree, "0", AreaCompiler, "1 compiles match expressions through the experimental decision-tree compiler (no guards, no list/record/tuple patterns)."},
	{EnvNoCache, "0", AreaCompiler, "1 disables the compile cache for this run."},
	{EnvRelaxModules, "", AreaCompiler, "1, true or yes relaxes module checking, the same as --relax-modules; the flag and the variable are OR-ed."},
	{EnvSeed, "", AreaCompiler, "Integer seed for the effect runtime; when present, seeded mode may draw from Rand."},
	{EnvTZ, "UTC", AreaCompiler, "Time zone the Clock effect reports."},
	{EnvLang, "C", AreaCompiler, "Locale the effect runtime reports."},
	{EnvFSSandbox, "", AreaCompiler, "Directory the FS effect is confined to; empty means no sandbox."},
	{EnvFSSandboxDebug, "0", AreaCompiler, "1 logs every sandbox rejection to stderr."},
	{EnvFSMaxBytes, "", AreaCompiler, "Cap on every FS read (a byte count or K/M/G/T-suffixed size); unset or 0 is unbounded, --fs-max-bytes overrides it, and a malformed value is an error. serve-api uses its upload cap instead."},
	{EnvRedactEnv, "on", AreaCompiler, "off disables redaction of sensitive environment values in traces and errors."},
	{EnvMetrics, "0", AreaCompiler, "1 collects pipeline phase timings and memory for each compile."},
	{EnvMetricsDebug, "0", AreaCompiler, "1 prints the raw phase-timing map to stderr."},
	{EnvMetricsVerbose, "0", AreaCompiler, "1 prints a metrics summary to stderr at the end of the run."},
	{EnvHubURL, "", AreaCompiler, "Collaboration-hub URL the metrics collector POSTs to; unset sends nothing."},
	{EnvDumpSMT, "", AreaCompiler, "Set to anything to keep the SMT-LIB file handed to Z3 instead of deleting it."},
	{EnvNoVersionWarnings, "", AreaCompiler, "Set to anything to suppress the stdlib version-mismatch warning."},
	{EnvQuietWarnings, "", AreaCompiler, "Set by the CLI in JSON and quiet modes to suppress the stdlib version warning."},
	{EnvGOGC, "", AreaCompiler, "Go's GC percent; when unset the run and exec commands raise it to 500 for a faster compile."},
}

// NoPrelude reports AILANG_NO_PRELUDE=1.
func NoPrelude() bool { return getOr(EnvNoPrelude) == "1" }

// RecordsV2 reports AILANG_RECORDS_V2=1.
func RecordsV2() bool { return getOr(EnvRecordsV2) == "1" }

// DTree reports AILANG_DTREE=1.
func DTree() bool { return getOr(EnvDTree) == "1" }

// NoCache reports AILANG_NO_CACHE=1.
func NoCache() bool { return getOr(EnvNoCache) == "1" }

// RelaxModules reports whether AILANG_RELAX_MODULES is 1, true or yes
// (case-insensitive). Callers OR it with their --relax-modules flag.
func RelaxModules() bool {
	switch strings.ToLower(get(EnvRelaxModules)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// Seed returns AILANG_SEED parsed as int64 and whether it was set to a valid
// integer. An unparseable value counts as unset, as it always did.
func Seed() (int64, bool) {
	s := get(EnvSeed)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// TZ returns TZ, default UTC.
func TZ() string { return getOr(EnvTZ) }

// Locale returns LANG, default C.
func Locale() string { return getOr(EnvLang) }

// FSSandbox returns AILANG_FS_SANDBOX, "" when no sandbox is set.
func FSSandbox() string { return get(EnvFSSandbox) }

// FSMaxBytes returns the AILANG_FS_MAX_BYTES cap in bytes and whether it was
// set; a malformed value is an error (a safety cap never falls back).
func FSMaxBytes() (int64, bool, error) {
	raw := strings.TrimSpace(get(EnvFSMaxBytes))
	if raw == "" {
		return 0, false, nil
	}
	n, err := ParseByteSize(raw)
	if err != nil {
		return 0, true, fmt.Errorf("%s: %w", EnvFSMaxBytes, err)
	}
	return n, true, nil
}

// FSSandboxDebug reports AILANG_FS_SANDBOX_DEBUG=1.
func FSSandboxDebug() bool { return getOr(EnvFSSandboxDebug) == "1" }

// RedactEnv reports whether env-value redaction is on: anything but
// AILANG_REDACT_ENV=off.
func RedactEnv() bool { return getOr(EnvRedactEnv) != "off" }

// Metrics reports AILANG_METRICS=1.
func Metrics() bool { return getOr(EnvMetrics) == "1" }

// MetricsDebug reports AILANG_METRICS_DEBUG=1.
func MetricsDebug() bool { return getOr(EnvMetricsDebug) == "1" }

// MetricsVerbose reports AILANG_METRICS_VERBOSE=1.
func MetricsVerbose() bool { return getOr(EnvMetricsVerbose) == "1" }

// HubURL returns AILANG_HUB_URL, "" when unset.
func HubURL() string { return get(EnvHubURL) }

// DumpSMT reports whether AILANG_DUMP_SMT is set.
func DumpSMT() bool { return get(EnvDumpSMT) != "" }

// StdlibVersionWarningsSuppressed reports whether either
// AILANG_NO_VERSION_WARNINGS or AILANG_QUIET_WARNINGS is set.
func StdlibVersionWarningsSuppressed() bool {
	return get(EnvNoVersionWarnings) != "" || get(EnvQuietWarnings) != ""
}

// GOGCSet reports whether the operator set GOGC, in which case the CLI
// leaves the GC percent alone.
func GOGCSet() bool { return get(EnvGOGC) != "" }
