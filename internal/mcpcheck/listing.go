package mcpcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Listing hygiene (M-MCP-OAUTH-CLIENT-MODELS M3c). Both checks only WARN:
// neither directory documents a rule, but reviewers see the serverInfo
// name, and the module inventory is not for a public listing.

// unbrandedServerName is serve-api's default serverInfo.name.
const unbrandedServerName = "ailang-api"

// metaModulesURI is serve-api's module-inventory resource. From v0.57.2 the
// listed surface (/mcp/connect/) no longer carries it.
const metaModulesURI = "ailang://meta/modules"

// checkServerInfo reads the initialize result's serverInfo.
func checkServerInfo(initResult json.RawMessage) Finding {
	f := Finding{Check: "server-info", Status: Warn}
	var res struct {
		ServerInfo struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	_ = json.Unmarshal(initResult, &res)
	si := res.ServerInfo
	switch si.Name {
	case "":
		f.Message = "initialize returned no serverInfo.name; reviewers see an unnamed server"
	case unbrandedServerName:
		f.Message = fmt.Sprintf("serverInfo is the unbranded serve-api default (%s %s); reviewers see it: set serve-api --server-info \"name=...,title=...,version=...\"", si.Name, si.Version)
	default:
		f.Status = Pass
		label := si.Name
		if si.Title != "" {
			label += " (" + si.Title + ")"
		}
		f.Message = strings.TrimSpace("serverInfo: " + label + " " + si.Version)
	}
	return f
}

// checkListedResources WARNs when the checked surface lists the module
// inventory (serve-api up to v0.57.1 listed it on /mcp/connect/, with
// @noexpose helpers and absolute paths). A server that does not answer
// resources/list has none to judge.
func (s *session) checkListedResources(ctx context.Context) Finding {
	f := Finding{Check: "listed-resources", Status: Pass}
	st, _, res, err := s.rpc(ctx, "resources/list", map[string]any{})
	if err != nil || st != http.StatusOK {
		f.Status = Skip
		if err != nil {
			f.Message = fmt.Sprintf("resources/list not answered: %v", err)
		} else {
			f.Message = fmt.Sprintf("resources/list not answered: HTTP %d", st)
		}
		return f
	}
	var list struct {
		Resources []struct {
			URI string `json:"uri"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(res, &list); err != nil {
		f.Status = Skip
		f.Message = fmt.Sprintf("resources/list: %v", err)
		return f
	}
	for _, r := range list.Resources {
		if r.URI == metaModulesURI {
			f.Status = Warn
			f.Message = metaModulesURI + " is listed: the server's module inventory is on a directory surface (serve-api <= v0.57.1 leaks hidden exports there; upgrade)"
			return f
		}
	}
	f.Message = fmt.Sprintf("%d resource(s) listed, no module inventory", len(list.Resources))
	return f
}
