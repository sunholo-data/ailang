package apiserver

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/version"
)

// ServerIdentity is what a serve-api service calls itself (--server-info,
// M-MCP-OAUTH-CLIENT-MODELS D7). One value fills MCP serverInfo on both
// surfaces, the A2A card version and OpenAPI info.version, replacing the
// three hardcoded 0.8.1 constants every listed service used to report.
type ServerIdentity struct {
	Name    string // MCP serverInfo.name
	Title   string // MCP serverInfo.title; "" = not sent
	Version string // MCP serverInfo.version, A2A card version, OpenAPI info.version
	// Icon: the A2A card's iconUrl. A path ("/icon.png") is resolved against
	// the host the caller reached, so one flag serves every deployment and the
	// icon stays on the card's own host (Gemini Enterprise fetches it from
	// there); otherwise an https URL. "" = no icon.
	Icon string
}

// defaultServerName is the unbranded name `ailang mcp check` warns about.
const defaultServerName = "ailang-api"

// DefaultServerInfo is the identity with no --server-info: the name
// ailang-api and the running binary's version ("dev" for unreleased builds),
// without its leading "v".
func DefaultServerInfo() ServerIdentity {
	v := strings.TrimPrefix(version.Version, "v")
	if v == "" {
		v = "dev"
	}
	return ServerIdentity{Name: defaultServerName, Version: v}
}

// ParseServerInfo reads --server-info "name=parse,title=AILANG Parse,version=0.43.0,icon=/icon.png".
// Keys may be given in any order and any subset; a key left out keeps its
// default. Values cannot contain a comma or a control character, a key may
// appear once, and an unknown key is an error: a typo must not ship an
// unbranded listing silently.
func ParseServerInfo(s string) (ServerIdentity, error) {
	id := DefaultServerInfo()
	if strings.TrimSpace(s) == "" {
		return id, nil
	}
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		key, val, ok := strings.Cut(part, "=")
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if !ok || key == "" {
			return ServerIdentity{}, fmt.Errorf("--server-info: %q is not key=value (keys: name, title, version, icon)", part)
		}
		if seen[key] {
			return ServerIdentity{}, fmt.Errorf("--server-info: %s given twice", key)
		}
		seen[key] = true
		if val == "" {
			return ServerIdentity{}, fmt.Errorf("--server-info: %s is empty", key)
		}
		if strings.IndexFunc(val, unicode.IsControl) >= 0 {
			return ServerIdentity{}, fmt.Errorf("--server-info: %s contains a control character", key)
		}
		switch key {
		case "name":
			id.Name = val
		case "title":
			id.Title = val
		case "version":
			id.Version = val
		case "icon":
			if !strings.HasPrefix(val, "/") && !strings.HasPrefix(val, "https://") {
				return ServerIdentity{}, fmt.Errorf("--server-info: icon must be a path on this server (/icon.png) or an https URL, got %q", val)
			}
			id.Icon = val
		default:
			return ServerIdentity{}, fmt.Errorf("--server-info: unknown key %q (keys: name, title, version, icon)", key)
		}
	}
	return id, nil
}

// withDefaults fills each empty field from DefaultServerInfo, so an
// embedder's partial Config.ServerInfo keeps the fields it set.
func (id ServerIdentity) withDefaults() ServerIdentity {
	def := DefaultServerInfo()
	if id.Name == "" {
		id.Name = def.Name
	}
	if id.Version == "" {
		id.Version = def.Version
	}
	return id
}

// serverIdentity is the identity every surface reports.
func (s *Server) serverIdentity() ServerIdentity {
	return s.identity.withDefaults()
}
