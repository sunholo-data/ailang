// Package streamcred is the host-side credential binding for Stream WebSocket
// dials (M-SERVEAPI-WS-BRIDGE D3).
//
// The operator binds a credential SOURCE to one upstream host:
//
//	ailang serve-api --stream-credential us-central1-aiplatform.googleapis.com=gcp-key-file:/secrets/daneel-sa.json
//
// and the binder adds "Authorization: Bearer <token>" to wss:// dials to that
// exact host, in Go, at dial time. The token never becomes an AILANG value,
// so it cannot reach a client frame, a trace or a log through program code.
//
// Every source is EXPLICIT (Mark, 2026-09-25): a named credentials file, the
// metadata server named as such, or a token file. There is no implicit
// Application Default Credentials lookup and no gcloud fallback, and a
// gcloud user credential file (type authorized_user) is refused — the relay
// must run as its own identity, not as whoever last ran gcloud on the box.
package streamcred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// cloudPlatformScope is the scope Vertex AI and other Google APIs accept.
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// refreshEarly refreshes a cached token this long before it expires, so a
// dial never presents a token about to lapse.
const refreshEarly = 5 * time.Minute

// Source produces the Authorization value for one dial.
type Source interface {
	// Token returns a bearer token. Errors must not contain the token.
	Token(ctx context.Context) (string, error)
	// Describe names the source for logs and errors (never a secret).
	Describe() string
}

// Binding binds one Source to one upstream host (and port).
type Binding struct {
	Host   string // lower-case hostname
	Port   string // "" = the wss default, 443
	Source Source
}

// Parse reads one --stream-credential value: HOST[:PORT]=SOURCE, where
// SOURCE is gcp-key-file:PATH, gcp-metadata, or bearer-file:PATH.
func Parse(spec string) (Binding, error) {
	hostPart, src, ok := strings.Cut(spec, "=")
	if !ok || hostPart == "" || src == "" {
		return Binding{}, fmt.Errorf("--stream-credential %q: want HOST=SOURCE (SOURCE: gcp-key-file:PATH | gcp-metadata | bearer-file:PATH)", spec)
	}
	if strings.Contains(hostPart, "/") {
		return Binding{}, fmt.Errorf("--stream-credential %q: HOST is a bare host[:port], not a URL", spec)
	}
	host, port := hostPart, ""
	if h, p, err := splitHostPort(hostPart); err == nil {
		host, port = h, p
	}
	source, err := parseSource(src)
	if err != nil {
		return Binding{}, fmt.Errorf("--stream-credential %q: %w", spec, err)
	}
	return Binding{Host: strings.ToLower(host), Port: port, Source: source}, nil
}

func splitHostPort(s string) (string, string, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", fmt.Errorf("no port")
	}
	return s[:i], s[i+1:], nil
}

func parseSource(src string) (Source, error) {
	kind, arg, _ := strings.Cut(src, ":")
	switch kind {
	case "gcp-key-file":
		if arg == "" {
			return nil, fmt.Errorf("gcp-key-file needs a path")
		}
		return newGCPKeyFile(arg)
	case "gcp-metadata":
		if arg != "" {
			return nil, fmt.Errorf("gcp-metadata takes no argument")
		}
		return &tokenSource{name: "gcp-metadata", ts: oauth2.ReuseTokenSourceWithExpiry(nil, google.ComputeTokenSource("", cloudPlatformScope), refreshEarly)}, nil
	case "bearer-file":
		if arg == "" {
			return nil, fmt.Errorf("bearer-file needs a path")
		}
		return &bearerFile{path: arg}, nil
	default:
		return nil, fmt.Errorf("unknown credential source %q (want gcp-key-file:PATH | gcp-metadata | bearer-file:PATH)", kind)
	}
}

// allowedKeyTypes are the Google credential file types a binding accepts:
// identities of their own. authorized_user (a gcloud user login) is not one.
var allowedKeyTypes = map[string]google.CredentialsType{
	string(google.ServiceAccount):             google.ServiceAccount,
	string(google.ImpersonatedServiceAccount): google.ImpersonatedServiceAccount,
	string(google.ExternalAccount):            google.ExternalAccount,
}

func newGCPKeyFile(path string) (Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gcp-key-file: %w", err)
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("gcp-key-file %s: not a Google credentials JSON file", path)
	}
	credType, ok := allowedKeyTypes[head.Type]
	if !ok {
		return nil, fmt.Errorf("gcp-key-file %s: credential type %q is not accepted (use a service_account, impersonated_service_account or external_account file; a gcloud user login is refused)", path, head.Type)
	}
	creds, err := google.CredentialsFromJSONWithType(context.Background(), data, credType, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("gcp-key-file %s: %w", path, err)
	}
	return &tokenSource{
		name: "gcp-key-file:" + path,
		ts:   oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, refreshEarly),
	}, nil
}

// tokenSource adapts a caching oauth2.TokenSource.
type tokenSource struct {
	name string
	ts   oauth2.TokenSource
}

func (t *tokenSource) Token(context.Context) (string, error) {
	tok, err := t.ts.Token()
	if err != nil {
		return "", fmt.Errorf("%s: %w", t.name, err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("%s: empty access token", t.name)
	}
	return tok.AccessToken, nil
}

func (t *tokenSource) Describe() string { return t.name }

// bearerFile reads a token some other tool keeps fresh, on every dial.
type bearerFile struct{ path string }

func (b *bearerFile) Token(context.Context) (string, error) {
	data, err := os.ReadFile(b.path)
	if err != nil {
		return "", fmt.Errorf("bearer-file %s: %w", b.path, err)
	}
	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("bearer-file %s: empty", b.path)
	}
	return tok, nil
}

func (b *bearerFile) Describe() string { return "bearer-file:" + b.path }

// Matches reports whether the binding applies to a wss:// URL: exact
// hostname (case-insensitive) and exact port, where a binding without a port
// means 443. The scheme is the caller's check too, but it is repeated here:
// a binding never applies to anything but wss.
func (b Binding) Matches(u *url.URL) bool {
	if u.Scheme != "wss" || !strings.EqualFold(u.Hostname(), b.Host) {
		return false
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	want := b.Port
	if want == "" {
		want = "443"
	}
	return port == want
}

// Binder turns bindings into the effects.StreamContext.Credentials hook.
// Two bindings for the same host:port are a configuration error.
func Binder(bindings []Binding) (func(u *url.URL) (map[string][]string, bool, error), error) {
	seen := map[string]bool{}
	for _, b := range bindings {
		key := b.Host + ":" + b.Port
		if seen[key] {
			return nil, fmt.Errorf("--stream-credential: %s is bound twice", strings.TrimSuffix(key, ":"))
		}
		seen[key] = true
	}
	return func(u *url.URL) (map[string][]string, bool, error) {
		for _, b := range bindings {
			if !b.Matches(u) {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			tok, err := b.Source.Token(ctx)
			if err != nil {
				return nil, true, err
			}
			return map[string][]string{"Authorization": {"Bearer " + tok}}, true, nil
		}
		return nil, false, nil
	}, nil
}

// Prefetch fetches each binding's token once, so a misconfigured source
// fails serve-api startup instead of the first dial, and so the first dial
// finds a cached token.
func Prefetch(ctx context.Context, bindings []Binding) error {
	for _, b := range bindings {
		if _, err := b.Source.Token(ctx); err != nil {
			return fmt.Errorf("--stream-credential %s: %w", b.Host, err)
		}
	}
	return nil
}
