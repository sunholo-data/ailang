package ollama

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// parseHost turns an endpoint string into the base URL the ollama SDK client
// takes, with exactly the normalisation the SDK's own envconfig.Host applies
// to OLLAMA_HOST: an optional scheme (http by default; the bare "ollama.com"
// means https on 443), a host that may be an IP, a hostname or empty
// (127.0.0.1), a port that defaults per scheme (11434, or 80 / 443 when the
// scheme is explicit) and is replaced by that default when out of range, and
// an optional path. host_test.go pins it against the SDK for every shape the
// rig and the plists have used, so the client built from an explicit
// endpoint is the one ClientFromEnvironment would have built.
func parseHost(s string) *url.URL {
	defaultPort := "11434"

	s = strings.TrimSpace(s)
	scheme, hostport, ok := strings.Cut(s, "://")
	switch {
	case !ok:
		scheme, hostport = "http", s
		if s == "ollama.com" {
			scheme, hostport = "https", "ollama.com:443"
		}
	case scheme == "http":
		defaultPort = "80"
	case scheme == "https":
		defaultPort = "443"
	}

	hostport, path, _ := strings.Cut(hostport, "/")
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host, port = "127.0.0.1", defaultPort
		if ip := net.ParseIP(strings.Trim(hostport, "[]")); ip != nil {
			host = ip.String()
		} else if hostport != "" {
			host = hostport
		}
	}

	if n, err := strconv.ParseInt(port, 10, 32); err != nil || n > 65535 || n < 0 {
		port = defaultPort
	}

	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
		Path:   path,
	}
}
