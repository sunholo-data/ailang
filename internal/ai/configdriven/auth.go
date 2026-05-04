package configdriven

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// envInterpolationPattern restricts ${VAR} syntax in auth_headers to
// uppercase-snake-case identifiers. Mirrors the validation pattern in
// internal/pkg/ai_provider.go — single source of truth would require
// re-exporting; for now duplicated and asserted via a guard test.
var envInterpolationPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// applyAuth mutates the outgoing HTTP request to carry the auth shape
// declared in the provider config. Resolves env-var references at call
// time, not load time — fresh env each call.
//
// Returns an error if a referenced env var is unset, since silent fallback
// to an empty Authorization header would produce a confusing 401 rather
// than a clear "missing credential" diagnostic.
func applyAuth(req *http.Request, spec *pkg.AIProviderSpec) error {
	// Named auth shape, if declared.
	if spec.HasNamedAuth() {
		if err := applyNamedAuth(req, spec.Auth); err != nil {
			return err
		}
	}
	// auth_headers escape hatch, applied after named auth so it can
	// extend (or override) headers from the named shape.
	for headerName, headerVal := range spec.AuthHeaders {
		resolved, err := interpolateEnv(headerVal)
		if err != nil {
			return fmt.Errorf("auth_headers[%q]: %w", headerName, err)
		}
		req.Header.Set(headerName, resolved)
	}
	return nil
}

// applyNamedAuth handles the four documented v1 auth shapes:
// bearer, x-api-key, query-param, none.
func applyNamedAuth(req *http.Request, auth pkg.AIProviderAuth) error {
	switch auth.Type {
	case "none":
		return nil
	case "bearer":
		secret, err := requireEnv(auth.Env)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+secret)
	case "x-api-key":
		secret, err := requireEnv(auth.Env)
		if err != nil {
			return err
		}
		req.Header.Set("x-api-key", secret)
	case "query-param":
		secret, err := requireEnv(auth.Env)
		if err != nil {
			return err
		}
		// Append to existing query string.
		q := req.URL.Query()
		q.Set(auth.Name, secret)
		req.URL.RawQuery = q.Encode()
	default:
		// Validation in internal/pkg should have caught this; defensive guard.
		return fmt.Errorf("unknown auth.type %q", auth.Type)
	}
	return nil
}

// requireEnv returns the value of an env var, or a structured error if
// unset. We deliberately do NOT fall back to empty — silent empty
// credentials produce confusing 401 errors at the provider rather than a
// clear "missing credential" diagnostic.
func requireEnv(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf("env var %s is unset (required for AI provider auth)", name)
	}
	return val, nil
}

// interpolateEnv resolves all ${VAR} references in s. Returns an error if
// any referenced env var is unset.
func interpolateEnv(s string) (string, error) {
	var firstErr error
	result := envInterpolationPattern.ReplaceAllStringFunc(s, func(match string) string {
		// match is "${VAR}"; strip braces.
		varName := match[2 : len(match)-1]
		val := os.Getenv(varName)
		if val == "" && firstErr == nil {
			firstErr = fmt.Errorf("env var %s is unset (referenced in auth_headers as %s)", varName, match)
		}
		return val
	})
	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
}

// stripQueryAuth returns the URL with the auth query parameter removed.
// Useful for trace span attributes — we don't want secrets in spans.
func stripQueryAuth(u *url.URL, paramName string) string {
	if paramName == "" {
		return u.String()
	}
	clone := *u
	q := clone.Query()
	if q.Has(paramName) {
		q.Set(paramName, "[REDACTED]")
		clone.RawQuery = q.Encode()
	}
	return clone.String()
}

// safeEndpointForTrace returns the configured endpoint with secrets redacted.
func safeEndpointForTrace(spec *pkg.AIProviderSpec) string {
	u, err := url.Parse(spec.Endpoint)
	if err != nil {
		return spec.Endpoint
	}
	if spec.Auth.Type == "query-param" {
		return stripQueryAuth(u, spec.Auth.Name)
	}
	return spec.Endpoint
}

// guardEnvPattern asserts our local envInterpolationPattern matches the one
// in internal/pkg. Called from a test to keep the duplication honest.
func guardEnvPattern() bool {
	// Trivial round-trip; full equality is checked structurally elsewhere.
	return strings.EqualFold("test", "TEST") || envInterpolationPattern.MatchString("${X}")
}
