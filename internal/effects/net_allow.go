package effects

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// #1558 — port-qualified allowlist entries.
//
// An allowlist entry (a policy's net_allow, --net-allow-domains,
// --stream-allow-domains) is `host` or `host:PORT`:
//
//	api.example.com        any port
//	api.example.com:8443   that port only
//	*.example.com:443      wildcard, that port only
//	127.0.0.1:7655         loopback, that port only (a port grant, below)
//	[::1]:7655             IPv6 literal with a port (brackets required)
//	::1                    bare IPv6 literal, any port
//
// A port-qualified LOOPBACK entry (localhost, 127.0.0.0/8, ::1) is itself a
// loopback grant for exactly that host and port: the program may reach the
// one local service the host named, and no other loopback port — without
// AllowLocalhost, which opens all of them. Redirect hops and Net[scope=public]
// frames never receive the grant.

// allowEntry is one parsed allowlist entry; port "" means any port.
type allowEntry struct {
	host string
	port string
}

// ParseNetAllowEntry parses one allowlist entry into its normalised host
// (lowercase, no trailing dot, IPv6 without brackets) and port ("" when the
// entry admits every port). The policy resolver uses it to refuse malformed
// net_allow entries at load time.
func ParseNetAllowEntry(raw string) (host, port string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", fmt.Errorf("empty entry")
	}
	if strings.Contains(s, "/") {
		return "", "", fmt.Errorf("%q is not a host or host:PORT (no scheme or path)", raw)
	}
	switch {
	case strings.HasPrefix(s, "["):
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return "", "", fmt.Errorf("%q has an unclosed '['", raw)
		}
		host = s[1:end]
		if ip := net.ParseIP(host); ip == nil || ip.To4() != nil {
			return "", "", fmt.Errorf("%q: brackets hold an IPv6 literal only", raw)
		}
		rest := s[end+1:]
		if rest != "" {
			if !strings.HasPrefix(rest, ":") {
				return "", "", fmt.Errorf("%q: want [IPv6]:PORT", raw)
			}
			port = rest[1:]
			if port == "" {
				return "", "", fmt.Errorf("%q: empty port", raw)
			}
		}
	case strings.Count(s, ":") == 1:
		i := strings.IndexByte(s, ':')
		host, port = s[:i], s[i+1:]
		if host == "" {
			return "", "", fmt.Errorf("%q: missing host", raw)
		}
		if port == "" {
			return "", "", fmt.Errorf("%q: empty port", raw)
		}
	case strings.Count(s, ":") > 1:
		// Unbracketed IPv6 literal: no port can be attached without brackets.
		if net.ParseIP(s) == nil {
			return "", "", fmt.Errorf("%q: an IPv6 literal with a port needs brackets ([::1]:PORT)", raw)
		}
		host = s
	default:
		host = s
	}
	if port != "" {
		n, perr := strconv.Atoi(port)
		if perr != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
			return "", "", fmt.Errorf("%q: port must be 1-65535", raw)
		}
	}
	host = normalizeHost(host)
	if host == "" {
		return "", "", fmt.Errorf("%q: missing host", raw)
	}
	return host, port, nil
}

// IsLoopbackEntryHost reports whether a parsed entry host names loopback:
// "localhost" or a loopback IP literal. The policy resolver uses it for the
// restricted-mode rule (loopback only port-qualified).
func IsLoopbackEntryHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// parseAllowList parses entries, dropping malformed ones: a malformed entry
// admits nothing (fail closed). Policy runs never get here with one — the
// resolver refuses them by name.
func parseAllowList(allowed []string) []allowEntry {
	out := make([]allowEntry, 0, len(allowed))
	for _, raw := range allowed {
		h, p, err := ParseNetAllowEntry(raw)
		if err != nil {
			continue
		}
		out = append(out, allowEntry{host: h, port: p})
	}
	return out
}

// isAllowedTarget: empty allowlist = every destination; otherwise some entry
// matches the host (exact or label-boundary wildcard, both sides normalised)
// and either names no port or names exactly this one.
func isAllowedTarget(hostname, port string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	hostname = normalizeHost(literalHost(hostname))
	for _, e := range parseAllowList(allowed) {
		if (e.port == "" || e.port == port) && matchDomain(hostname, e.host) {
			return true
		}
	}
	return false
}

// loopbackPortGranted reports whether a port-qualified loopback entry names
// exactly this host and port. Names match exactly, never by wildcard.
func loopbackPortGranted(hostname, port string, allowed []string) bool {
	hostname = normalizeHost(literalHost(hostname))
	if !IsLoopbackEntryHost(hostname) {
		return false
	}
	for _, e := range parseAllowList(allowed) {
		if e.port != "" && e.port == port && e.host == hostname {
			return true
		}
	}
	return false
}

// targetPort is the port a request to u dials: the explicit one, else the
// scheme's default.
func targetPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	return defaultPort(u.Scheme)
}

// forTarget is the policy for a request to u: when loopback is otherwise off
// and a port-qualified loopback entry names u's host and port, loopback is
// on for this target only. Redirect hops and Net[scope=public] never get it.
func (p destinationPolicy) forTarget(u *url.URL) destinationPolicy {
	if p.allowLocalhost || p.noLoopbackGrant || u == nil {
		return p
	}
	if loopbackPortGranted(u.Hostname(), targetPort(u), p.allowedDomains) {
		p.allowLocalhost = true
	}
	return p
}
