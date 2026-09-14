package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// M-PKG-MULTI-NAMESPACE-AUTH — `ailang pkg key` mints, lists and revokes
// scoped registry keys. Every subcommand needs the SUPERUSER key in
// AILANG_REGISTRY_API_KEY; a scoped key gets a 403 from the validator.

func pkgKeyCommand(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("Usage: ailang pkg key <command>")
		fmt.Println()
		fmt.Println("Manage scoped registry keys (requires the superuser AILANG_REGISTRY_API_KEY).")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  create --owner <name> --scope <glob> [--scope ...] [--note <text>]")
		fmt.Println("         Mint a key. The plaintext is printed ONCE and never stored.")
		fmt.Println("  list   Show key ids, owners, scopes and revocation state")
		fmt.Println("  revoke <id>")
		fmt.Println()
		fmt.Println("Scopes are globs over the full package name:")
		fmt.Println("  daneel/*            everything under the daneel namespace")
		fmt.Println("  sunholo/daneel_*    a prefix inside a shared namespace")
		fmt.Println("  sunholo/exact_pkg   one package")
		fmt.Println()
		fmt.Println("Reads need no key — the registry bucket is public.")
		return nil
	}
	switch args[0] {
	case "create":
		return pkgKeyCreate(args[1:])
	case "list":
		return pkgKeyList()
	case "revoke":
		if len(args) < 2 {
			return fmt.Errorf("usage: ailang pkg key revoke <id>")
		}
		return pkgKeyRevoke(args[1])
	default:
		return fmt.Errorf("unknown key command %q (create|list|revoke)", args[0])
	}
}

func pkgKeyCreate(args []string) error {
	fs := flag.NewFlagSet("pkg key create", flag.ContinueOnError)
	owner := fs.String("owner", "", "who this key is for (recorded as published_by)")
	note := fs.String("note", "", "free-text note")
	var scopes multiFlag
	fs.Var(&scopes, "scope", "package glob this key may write (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *owner == "" || len(scopes) == 0 {
		return fmt.Errorf("--owner and at least one --scope are required")
	}

	body, _ := json.Marshal(map[string]interface{}{"owner": *owner, "scopes": []string(scopes), "note": *note})
	resp, err := registryAdminRequest(http.MethodPost, "/admin/keys", body)
	if err != nil {
		return err
	}
	var out struct {
		ID     string   `json:"id"`
		Key    string   `json:"key"`
		Owner  string   `json:"owner"`
		Scopes []string `json:"scopes"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	fmt.Printf("%s Minted key for %s (scopes: %s)\n", green("✓"), out.Owner, strings.Join(out.Scopes, ", "))
	fmt.Printf("  id:  %s\n", out.ID)
	fmt.Printf("  key: %s\n", out.Key)
	fmt.Println()
	fmt.Println("  This is the only time the key is shown. Hand it over on a private channel;")
	fmt.Println("  the holder sets AILANG_REGISTRY_API_KEY=<key> and runs `ailang publish`.")
	return nil
}

func pkgKeyList() error {
	resp, err := registryAdminRequest(http.MethodGet, "/admin/keys", nil)
	if err != nil {
		return err
	}
	var out struct {
		Keys []struct {
			ID        string   `json:"id"`
			Owner     string   `json:"owner"`
			Scopes    []string `json:"scopes"`
			Note      string   `json:"note"`
			CreatedAt string   `json:"created_at"`
			RevokedAt string   `json:"revoked_at"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	if len(out.Keys) == 0 {
		fmt.Println("No scoped keys.")
		return nil
	}
	fmt.Printf("%-14s %-12s %-10s %s\n", "ID", "OWNER", "STATE", "SCOPES")
	for _, k := range out.Keys {
		state := "active"
		if k.RevokedAt != "" {
			state = "revoked"
		}
		line := fmt.Sprintf("%-14s %-12s %-10s %s", k.ID[:12], k.Owner, state, strings.Join(k.Scopes, ", "))
		if k.Note != "" {
			line += "  # " + k.Note
		}
		fmt.Println(line)
	}
	return nil
}

func pkgKeyRevoke(id string) error {
	if _, err := registryAdminRequest(http.MethodDelete, "/admin/keys/"+id, nil); err != nil {
		return err
	}
	fmt.Printf("%s Revoked key %s\n", green("✓"), id)
	return nil
}

func registryAdminRequest(method, path string, body []byte) ([]byte, error) {
	apiKey := os.Getenv("AILANG_REGISTRY_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("AILANG_REGISTRY_API_KEY not set (the superuser key is required for key management)")
	}
	req, err := http.NewRequest(method, registryValidatorURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry unavailable: %w", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var e map[string]string
		if json.Unmarshal(out, &e) == nil && e["error"] != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", e["error"], resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, out)
	}
	return out, nil
}
