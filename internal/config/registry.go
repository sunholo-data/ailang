package config

import (
	"fmt"
	"os"
	"sort"
	"sync"
)

// Var is one environment variable AILANG reads, as the reference page and
// the getters see it. Every getter in this package resolves its name from an
// Env* constant that appears in exactly one Var, and a getter with a default
// reads that default from the Var (getOr) — so docs/docs/reference/env-vars.md,
// generated from Registry by `make docs-env`, cannot drift from the code.
type Var struct {
	// Name is the variable, e.g. AILANG_TRACE.
	Name string
	// Default is what the getter returns when the variable is unset or empty,
	// spelled the way an operator would set it ("500", "off", "1"). Empty
	// means "unset": the feature is off, or the value comes from a fallback
	// the Doc names (a flag, the config file, another variable).
	Default string
	// Where is the area the variable configures; the reference page groups
	// by it. One of the area names below.
	Where string
	// Doc is one sentence: what the variable does and what its values mean.
	Doc string
}

// The areas. One per file in this package; the generated page uses them as
// section headings in this order.
const (
	AreaCloud       = "Cloud identity"
	AreaStorage     = "Storage plane"
	AreaCompiler    = "Compiler and runtime"
	AreaPaths       = "Paths"
	AreaTrace       = "Tracing"
	AreaTelemetry   = "Telemetry export"
	AreaCoordinator = "Coordinator daemon"
	AreaJob         = "Cloud Run job (execute-job)"
	AreaServer      = "Dashboard server"
	AreaExecutor    = "Executors"
	AreaProviders   = "Provider credentials"
	AreaAI          = "AI clients"
	AreaOllama      = "Ollama"
	AreaEmbed       = "Embeddings"
	AreaEval        = "Eval harness"
	AreaMission     = "Mission loop"
	AreaRig         = "Rig lock"
	AreaPubSub      = "Pub/Sub"
	AreaRegistry    = "Package registry"
	AreaModels      = "Model registry"
	AreaMCP         = "Prompt and MCP"
	AreaMicroRAG    = "MicroRAG"
	AreaAPIServer   = "serve-api"
	AreaNotify      = "Notifications"
)

// Areas lists the areas in the order the generated page presents them.
var Areas = []string{
	AreaCloud, AreaStorage, AreaCompiler, AreaPaths, AreaTrace, AreaTelemetry,
	AreaCoordinator, AreaJob, AreaServer, AreaExecutor, AreaProviders, AreaAI,
	AreaOllama, AreaEmbed, AreaEval, AreaMission, AreaRig, AreaPubSub, AreaRegistry,
	AreaModels, AreaMCP, AreaMicroRAG, AreaAPIServer, AreaNotify,
}

// Registry is every environment variable this package reads, grouped by
// area. registry_test.go asserts that every Env* constant in the package
// appears here exactly once and that every entry names an Env* constant.
var Registry = concat(
	cloudVars, storageVars, compilerVars, pathVars, traceVars, telemetryVars,
	coordinatorVars, jobVars, serverVars, executorVars, providerVars, aiVars,
	ollamaVars, embedVars, evalVars, missionVars, rigVars, pubsubVars, registryVars,
	modelVars, mcpVars, microragVars, apiserverVars, notifyVars,
)

func concat(groups ...[]Var) []Var {
	var out []Var
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

var registryIndex struct {
	once   sync.Once
	byName map[string]Var
}

func indexRegistry() map[string]Var {
	registryIndex.once.Do(func() {
		m := make(map[string]Var, len(Registry))
		for _, v := range Registry {
			m[v.Name] = v
		}
		registryIndex.byName = m
	})
	return registryIndex.byName
}

// Lookup returns the Registry entry for name.
func Lookup(name string) (Var, bool) {
	v, ok := indexRegistry()[name]
	return v, ok
}

// Names returns every registered name, sorted.
func Names() []string {
	out := make([]string, 0, len(Registry))
	for _, v := range Registry {
		out = append(out, v.Name)
	}
	sort.Strings(out)
	return out
}

// get is the one raw read for a registered name: exactly os.Getenv.
func get(name string) string { return os.Getenv(name) }

// isSet reports whether name is present in the environment, empty or not.
func isSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

// getOr returns the value of name, or its Registry default when the variable
// is unset or empty. The default comes from the Registry so the generated
// reference and the getter cannot disagree; an unregistered name is a
// programming error and panics, which registry_test.go rules out.
func getOr(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultOf(name)
}

// defaultOf returns the registered default for name; see getOr.
func defaultOf(name string) string {
	entry, ok := Lookup(name)
	if !ok {
		panic(fmt.Sprintf("config: %s is read with a default but not in Registry", name))
	}
	return entry.Default
}

// Raw reads a variable whose NAME is chosen at runtime — a `--api-key-env`
// flag, the credential variable an AI provider declares for itself, an OTLP
// endpoint key iterated from a list. It exists so those sites do not read
// os.Getenv themselves; every statically-named variable has a typed getter
// and belongs in the Registry instead.
func Raw(name string) string { return os.Getenv(name) }

// RawSet is Raw's presence check: true when name is in the environment even
// if its value is empty.
func RawSet(name string) bool { return isSet(name) }
