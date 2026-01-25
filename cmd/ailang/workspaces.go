package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/server/auth"
)

func workspacesCommand(args []string) error {
	if len(args) == 0 {
		return workspacesHelp()
	}

	switch args[0] {
	case "list":
		return workspacesList(args[1:])
	case "add", "create":
		return workspacesAdd(args[1:])
	case "grant":
		return workspacesGrant(args[1:])
	case "revoke":
		return workspacesRevoke(args[1:])
	case "set-public":
		return workspacesSetPublic(args[1:])
	case "show":
		return workspacesShow(args[1:])
	case "--help", "-h", "help":
		return workspacesHelp()
	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func workspacesHelp() error {
	fmt.Println("Usage: ailang workspaces <command> [options]")
	fmt.Println("")
	fmt.Println("Manage workspaces and access control for the AILANG Dashboard")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list        List all workspaces (or accessible workspaces for a user)")
	fmt.Println("  add         Create a new workspace")
	fmt.Println("  show        Show workspace details")
	fmt.Println("  grant       Grant user access to a workspace")
	fmt.Println("  revoke      Revoke user access from a workspace")
	fmt.Println("  set-public  Toggle public visibility of a workspace")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang workspaces list")
	fmt.Println("  ailang workspaces list --email m@sunholo.com")
	fmt.Println("  ailang workspaces add --id sunholo-data/ailang --name \"AILANG Project\" --public")
	fmt.Println("  ailang workspaces grant --id sunholo-data/ailang --email dev@example.com --role Approver")
	fmt.Println("  ailang workspaces revoke --id sunholo-data/ailang --email dev@example.com")
	fmt.Println("  ailang workspaces set-public --id sunholo-data/ailang --public=true")
	fmt.Println("")
	fmt.Println("Workspace IDs:")
	fmt.Println("  Use GitHub repo format: owner/repo (e.g., sunholo-data/ailang)")
	fmt.Println("")
	fmt.Println("Roles:")
	fmt.Println("  Approver   Can approve/reject tasks (admin access)")
	fmt.Println("  Viewer     Read-only access to workspace data")
	return nil
}

func workspacesList(args []string) error {
	var email, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()
	workspacesConfig := coordinator.LoadWorkspacesConfig()
	ws := auth.NewWorkspaceService(fsClient, workspacesConfig)

	accessible, err := ws.ListAccessibleWorkspaces(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(accessible) == 0 {
		if email != "" {
			fmt.Printf("No accessible workspaces for %s\n", email)
		} else {
			fmt.Println("No public workspaces configured")
		}
		return nil
	}

	fmt.Printf("%-30s %-20s %-10s %-8s\n", "ID", "NAME", "ROLE", "PUBLIC")
	fmt.Println(strings.Repeat("-", 70))
	for _, w := range accessible {
		publicStr := "no"
		if w.IsPublic {
			publicStr = "yes"
		}
		role := w.Role
		if role == "" {
			role = "(public)"
		}
		name := w.Name
		if name == "" {
			name = "-"
		}
		fmt.Printf("%-30s %-20s %-10s %-8s\n", w.ID, name, role, publicStr)
	}

	return nil
}

func workspacesAdd(args []string) error {
	var id, name, projectID string
	var isPublic bool

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--public":
			isPublic = true
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if id == "" {
		return fmt.Errorf("--id is required (e.g., sunholo-data/ailang)")
	}
	if name == "" {
		name = id // Default name to ID
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()

	// Encode workspace ID for Firestore document ID (replaces "/" with "__")
	docID := auth.EncodeDocID(id)

	// Check if workspace already exists
	doc, err := fsClient.Collection("workspaces").Doc(docID).Get(ctx)
	if err == nil && doc.Exists() {
		return fmt.Errorf("workspace '%s' already exists", id)
	}

	// Create workspace
	now := time.Now()
	_, err = fsClient.Collection("workspaces").Doc(docID).Set(ctx, map[string]interface{}{
		"id":          id, // Store original ID with "/"
		"name":        name,
		"github_repo": id, // Assume ID is the GitHub repo
		"is_public":   isPublic,
		"created_at":  now,
		"created_by":  "cli",
	})
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	publicStr := ""
	if isPublic {
		publicStr = " (public)"
	}
	fmt.Printf("✓ Created workspace '%s'%s\n", id, publicStr)
	return nil
}

func workspacesGrant(args []string) error {
	var id, email, role, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--role", "-r":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if id == "" {
		return fmt.Errorf("--id is required")
	}
	if email == "" {
		return fmt.Errorf("--email is required")
	}
	if role == "" {
		return fmt.Errorf("--role is required (Approver or Viewer)")
	}
	if role != "Approver" && role != "Viewer" {
		return fmt.Errorf("invalid role: %s (must be 'Approver' or 'Viewer')", role)
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()
	workspacesConfig := coordinator.LoadWorkspacesConfig()
	ws := auth.NewWorkspaceService(fsClient, workspacesConfig)

	err = ws.GrantAccess(ctx, email, id, role, "cli")
	if err != nil {
		return fmt.Errorf("failed to grant access: %w", err)
	}

	fmt.Printf("✓ Granted %s access to '%s' as %s\n", email, id, role)
	return nil
}

func workspacesRevoke(args []string) error {
	var id, email, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if id == "" {
		return fmt.Errorf("--id is required")
	}
	if email == "" {
		return fmt.Errorf("--email is required")
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()
	workspacesConfig := coordinator.LoadWorkspacesConfig()
	ws := auth.NewWorkspaceService(fsClient, workspacesConfig)

	err = ws.RevokeAccess(ctx, email, id)
	if err != nil {
		return fmt.Errorf("failed to revoke access: %w", err)
	}

	fmt.Printf("✓ Revoked %s access from '%s'\n", email, id)
	return nil
}

func workspacesSetPublic(args []string) error {
	var id, projectID string
	var isPublic bool
	publicSet := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--public":
			isPublic = true
			publicSet = true
		case "--public=true":
			isPublic = true
			publicSet = true
		case "--public=false":
			isPublic = false
			publicSet = true
		case "--private":
			isPublic = false
			publicSet = true
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if id == "" {
		return fmt.Errorf("--id is required")
	}
	if !publicSet {
		return fmt.Errorf("--public or --private is required")
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()
	workspacesConfig := coordinator.LoadWorkspacesConfig()
	ws := auth.NewWorkspaceService(fsClient, workspacesConfig)

	err = ws.SetPublic(ctx, id, isPublic)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	if isPublic {
		fmt.Printf("✓ Workspace '%s' is now public\n", id)
	} else {
		fmt.Printf("✓ Workspace '%s' is now private\n", id)
	}
	return nil
}

func workspacesShow(args []string) error {
	var id, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if id == "" {
		return fmt.Errorf("--id is required")
	}

	fsClient, cancel, err := getFirestoreClient(projectID)
	if err != nil {
		return err
	}
	defer fsClient.Close()
	defer cancel()

	ctx := context.Background()
	workspacesConfig := coordinator.LoadWorkspacesConfig()
	ws := auth.NewWorkspaceService(fsClient, workspacesConfig)

	workspace, err := ws.GetWorkspace(ctx, id)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	fmt.Printf("Workspace: %s\n", workspace.ID)
	fmt.Printf("Name:      %s\n", workspace.Name)
	fmt.Printf("Public:    %v\n", workspace.IsPublic)
	fmt.Printf("GitHub:    %s\n", workspace.GitHubRepo)
	fmt.Printf("Created:   %s\n", workspace.CreatedAt.Format(time.RFC3339))

	return nil
}

// getFirestoreClient creates a Firestore client with the configured project
func getFirestoreClient(projectID string) (*firestore.Client, context.CancelFunc, error) {
	if projectID == "" {
		projectID = os.Getenv("AILANG_FIREBASE_PROJECT")
	}
	if projectID == "" {
		cfg := coordinator.LoadFirebaseConfig()
		if cfg != nil {
			projectID = cfg.ProjectID
		}
	}
	if projectID == "" {
		return nil, nil, fmt.Errorf("Firebase project not configured. Use --project flag, AILANG_FIREBASE_PROJECT env, or config file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to initialize Firebase: %w", err)
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to get Firestore client: %w", err)
	}

	return fsClient, cancel, nil
}
