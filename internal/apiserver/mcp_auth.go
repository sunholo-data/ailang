package apiserver

import (
	"fmt"
	"sort"
	"strings"
)

// M-SERVEAPI-DIRECTORY-READY: registration rules for @mcp_auth,
// @mcp_token_verifier and @mcp_secret. Every rule here is an author or
// configuration bug, refused at registration with an ERROR (the posture of an
// invalid @mcp_name): a gated tool that cannot be authorized must not be served
// open, and a "secret" that would still be advertised is not a secret.

const (
	mcpAuthOAuth2 = "oauth2"
	mcpAuthNoAuth = "noauth"
)

// tokenVerifier locates the server's one @mcp_token_verifier function.
type tokenVerifier struct {
	modPath string
	name    string
}

// resolveTokenVerifier finds the single (string) -> bool verifier across the
// loaded modules. It returns (nil, nil) when there is none, and an error when
// there are several or the one found has the wrong signature: either way no
// @mcp_auth("oauth2") tool can be registered.
func resolveTokenVerifier(modules map[string]*ModuleInfo) (*tokenVerifier, error) {
	var found []tokenVerifier
	var badSig []string
	for modPath, info := range modules {
		for _, e := range info.Exports {
			if !e.IsTokenVerifier {
				continue
			}
			found = append(found, tokenVerifier{modPath: modPath, name: e.Name})
			if !e.VerifierSigOK {
				badSig = append(badSig, modPath+"/"+e.Name)
			}
		}
	}
	switch {
	case len(found) == 0:
		return nil, nil
	case len(found) > 1:
		names := make([]string, len(found))
		for i, v := range found {
			names[i] = v.modPath + "/" + v.name
		}
		sort.Strings(names)
		return nil, fmt.Errorf("@mcp_token_verifier: %d verifiers (%s); a server has exactly one", len(found), strings.Join(names, ", "))
	case len(badSig) > 0:
		return nil, fmt.Errorf("@mcp_token_verifier %s: signature must be (token: string) -> bool ! {...}", badSig[0])
	}
	return &found[0], nil
}

// validateMCPAuth checks one export's @mcp_auth and @mcp_secret against the
// server configuration. verifierErr is resolveTokenVerifier's error, if any.
func validateMCPAuth(e ExportInfo, issuer string, verifier *tokenVerifier, verifierErr error) error {
	switch e.MCPAuth {
	case "", mcpAuthNoAuth:
	case mcpAuthOAuth2:
		if issuer == "" {
			return fmt.Errorf("@mcp_auth(%q) needs --oauth-issuer (the authorization server named in the protected-resource metadata)", mcpAuthOAuth2)
		}
		if verifierErr != nil {
			return fmt.Errorf("@mcp_auth(%q): %v", mcpAuthOAuth2, verifierErr)
		}
		if verifier == nil {
			return fmt.Errorf("@mcp_auth(%q) needs one @mcp_token_verifier function (token: string) -> bool", mcpAuthOAuth2)
		}
	default:
		return fmt.Errorf("@mcp_auth(%q): unknown scheme; supported: %q, %q", e.MCPAuth, mcpAuthOAuth2, mcpAuthNoAuth)
	}
	return validateSecretParams(e)
}

// validateSecretParams: each @mcp_secret name must be a declared param that is
// also @optional, because the listed surface drops it and binds its zero value
// through the @optional path. _headers is bound by the server and is never a
// client-supplied secret.
func validateSecretParams(e ExportInfo) error {
	optional := make(map[string]bool, len(e.Optional))
	for _, n := range e.Optional {
		optional[n] = true
	}
	for _, name := range e.MCPSecret {
		isParam := false
		for _, p := range e.ParamNames {
			if p == name {
				isParam = true
				break
			}
		}
		switch {
		case name == headersParam:
			return fmt.Errorf("@mcp_secret(%q): _headers is bound from the request, not supplied by the client", name)
		case !isParam:
			return fmt.Errorf("@mcp_secret(%q): no such parameter (params: %s)", name, strings.Join(e.ParamNames, ", "))
		case !optional[name]:
			return fmt.Errorf("@mcp_secret(%q): must also be @optional(%q) so the listed surface can bind its zero value", name, name)
		}
	}
	return nil
}
