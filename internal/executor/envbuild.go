package executor

import (
	"runtime"
	"sort"
	"strings"
)

// envSet is the canonical child-environment builder (M-EXECUTOR-ENV-HARDENING
// D4). Every key is set through set/unset, so a key exists at most once and
// the value a child sees is the one the LAST layer wrote — by construction,
// not by relying on os/exec's last-wins de-duplication of Cmd.Env.
//
// Layers are applied in a fixed order by BuildEnvironment:
//
//	inherited (host) → harness-injected → validated Task.ExtraEnv → executor-required
//
// so the documented precedence is ExtraEnv > harness-injected > inherited, with
// the executor's own required set (e.g. motoko's MODEL, MOTOKO_CONFIG) last.
// Names ExtraEnv may not set at all are refused by ValidateExtraEnv, which is
// what keeps "ExtraEnv wins" from meaning "a benchmark can rewrite the trace
// context or the program policy".
type envSet struct {
	order []string          // folded keys, first-insertion order
	names map[string]string // folded key -> name as written
	vals  map[string]string // folded key -> value
}

func newEnvSet() *envSet {
	return &envSet{names: map[string]string{}, vals: map[string]string{}}
}

// foldEnvKey is the identity used for de-duplication. Windows environment
// names are case-insensitive (os/exec folds them there too); everywhere else
// PATH and Path are different variables.
func foldEnvKey(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(name)
	}
	return name
}

// splitEnvEntry splits "NAME=value". Windows keeps per-drive working
// directories as "=C:=C:\dir" entries whose name starts with '='; the first
// '=' after position 0 is the separator. ok is false for an entry with no
// separator or an empty name.
func splitEnvEntry(kv string) (name, value string, ok bool) {
	start := 0
	if strings.HasPrefix(kv, "=") {
		start = 1
	}
	i := strings.IndexByte(kv[start:], '=')
	if i < 0 {
		return "", "", false
	}
	i += start
	if i == 0 {
		return "", "", false
	}
	return kv[:i], kv[i+1:], true
}

func (e *envSet) set(name, value string) {
	k := foldEnvKey(name)
	if _, exists := e.vals[k]; !exists {
		e.order = append(e.order, k)
	}
	e.names[k] = name
	e.vals[k] = value
}

func (e *envSet) unset(name string) {
	k := foldEnvKey(name)
	if _, exists := e.vals[k]; !exists {
		return
	}
	delete(e.vals, k)
	delete(e.names, k)
	for i, o := range e.order {
		if o == k {
			e.order = append(e.order[:i], e.order[i+1:]...)
			break
		}
	}
}

// setEntries applies a slice of "NAME=value" entries in order (later wins).
// Malformed entries are skipped: they cannot be represented in a child env.
func (e *envSet) setEntries(entries []string) {
	for _, kv := range entries {
		if name, value, ok := splitEnvEntry(kv); ok {
			e.set(name, value)
		}
	}
}

// setMapSorted applies a map in sorted-name order so the result does not
// depend on Go's randomized map iteration (A1).
func (e *envSet) setMapSorted(m map[string]string) {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		e.set(k, m[k])
	}
}

// environ renders the set as a Cmd.Env slice: one entry per key, in
// first-insertion order.
func (e *envSet) environ() []string {
	out := make([]string, 0, len(e.order))
	for _, k := range e.order {
		out = append(out, e.names[k]+"="+e.vals[k])
	}
	return out
}
