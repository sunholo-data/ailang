package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func pkgUnpublishCommand(args []string) error {
	flagSet := flag.NewFlagSet("unpublish", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")
	forceFlag := flagSet.Bool("force", false, "Skip confirmation prompt")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || flagSet.NArg() == 0 {
		fmt.Println("Usage: ailang unpublish <vendor/name@version> [--force]")
		fmt.Println()
		fmt.Println("Remove a package version from the AILANG registry.")
		fmt.Println("This deletes the tarball, metadata, and updates the index.")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  vendor/name@version  Package and version to remove (version required)")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --force    Skip confirmation prompt")
		fmt.Println()
		fmt.Println("Environment:")
		fmt.Println("  AILANG_REGISTRY_VALIDATOR  Validator service URL (required)")
		fmt.Println("  AILANG_REGISTRY_API_KEY    API key for authentication (required)")
		return nil
	}

	// Parse vendor/name@version
	arg := flagSet.Arg(0)
	atIdx := strings.LastIndex(arg, "@")
	if atIdx == -1 {
		return fmt.Errorf("version required: use vendor/name@version (e.g., sunholo/auth@0.1.0)")
	}
	name := arg[:atIdx]
	version := arg[atIdx+1:]

	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid package name: %s (must be vendor/name)", name)
	}
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Confirm with user
	if !*forceFlag {
		fmt.Printf("%s This will permanently remove %s@%s from the registry.\n", yellow("⚠"), name, version)
		fmt.Printf("  Existing lockfiles referencing this version will break.\n")
		fmt.Printf("  Type the package version to confirm: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != version {
			return fmt.Errorf("confirmation failed (expected %q, got %q)", version, input)
		}
	}

	// Call validator
	validatorURL := os.Getenv("AILANG_REGISTRY_VALIDATOR")
	if validatorURL == "" {
		return fmt.Errorf("AILANG_REGISTRY_VALIDATOR not set\nSet the Cloud Run validator URL: export AILANG_REGISTRY_VALIDATOR=https://ailang-registry-validator-XXXX.run.app")
	}

	apiKey := os.Getenv("AILANG_REGISTRY_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("AILANG_REGISTRY_API_KEY not set\nAPI key is required for unpublish")
	}

	fmt.Printf("  Removing %s@%s from registry...\n", name, version)

	url := fmt.Sprintf("%s/unpublish?name=%s&version=%s", validatorURL, name, version)
	client := &http.Client{Timeout: 2 * time.Minute}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if msg, ok := result["message"].(string); ok {
				fmt.Printf("%s %s\n", green("✓"), msg)
			} else {
				fmt.Printf("%s Removed %s@%s\n", green("✓"), name, version)
			}
			if remaining, ok := result["remaining_versions"].([]interface{}); ok && len(remaining) > 0 {
				versions := make([]string, len(remaining))
				for i, v := range remaining {
					versions[i] = fmt.Sprint(v)
				}
				fmt.Printf("  Remaining versions: %s\n", strings.Join(versions, ", "))
			}
		} else {
			fmt.Printf("%s Removed %s@%s\n", green("✓"), name, version)
		}
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("package %s@%s not found in registry", name, version)
	case http.StatusForbidden:
		return fmt.Errorf("not authorized (check AILANG_REGISTRY_API_KEY)")
	default:
		return fmt.Errorf("unpublish failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
}
