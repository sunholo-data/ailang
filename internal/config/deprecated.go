package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// EnvStrict, when set to 1 or true, makes every DeprecatedDefault an error —
// the v1.0.0 behaviour, available today so a plist or Cloud Run env can be
// proven complete before the release that refuses to start without it.
const EnvStrict = "AILANG_STRICT_CONFIG"

// ErrDeprecatedDefault is wrapped by the error DeprecatedDefault returns under
// AILANG_STRICT_CONFIG. Match it with errors.Is.
var ErrDeprecatedDefault = errors.New("deprecated default refused under " + EnvStrict + "=1")

// Strict reports whether AILANG_STRICT_CONFIG is on.
func Strict() bool {
	switch os.Getenv(EnvStrict) {
	case "1", "true", "TRUE", "True":
		return true
	}
	return false
}

// warnOutput is where the deprecation warning goes; tests swap it.
var warnOutput io.Writer = os.Stderr

var warned sync.Map // name -> struct{}

// DeprecatedDefault is the sanctioned way to keep a production default alive
// while it is retired (ruling D3, 2026-09-15). Call it only after the named
// variable has already resolved empty — it does not read the environment
// itself, so precedence stays where it belongs.
//
// It returns value and writes one warning per process per name to stderr:
//
//	<name> is unset; using deprecated default <value>. Set <name>. v1.0.0 will refuse to start without it.
//
// Under AILANG_STRICT_CONFIG=1 it returns "" and an error wrapping
// ErrDeprecatedDefault instead, so the failure v1.0.0 will produce can be
// rehearsed now.
func DeprecatedDefault(name, value string) (string, error) {
	if Strict() {
		return "", fmt.Errorf("%w: %s is unset (would have used %q)", ErrDeprecatedDefault, name, value)
	}
	if _, already := warned.LoadOrStore(name, struct{}{}); !already {
		fmt.Fprintf(warnOutput, "%s is unset; using deprecated default %s. Set %s. v1.0.0 will refuse to start without it.\n",
			name, value, name)
	}
	return value, nil
}

// resetWarningsForTest forgets which names have warned.
func resetWarningsForTest() {
	warned.Range(func(k, _ any) bool {
		warned.Delete(k)
		return true
	})
}
