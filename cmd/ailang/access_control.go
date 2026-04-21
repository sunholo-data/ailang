package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

func accessControlCommand(args []string) error {
	if len(args) == 0 {
		return accessControlHelp()
	}

	switch args[0] {
	case "add":
		return accessControlAdd(args[1:])
	case "remove":
		return accessControlRemove(args[1:])
	case "list":
		return accessControlList(args[1:])
	case "--help", "-h", "help":
		return accessControlHelp()
	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func accessControlHelp() error {
	fmt.Println("Usage: ailang access-control <command> [options]")
	fmt.Println("")
	fmt.Println("Manage user access control for the AILANG Dashboard")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  add      Add a user to a workspace with a role")
	fmt.Println("  remove   Remove a user from a workspace")
	fmt.Println("  list     List users in a workspace")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang access-control add --email m@sunholo.com --role Approver")
	fmt.Println("  ailang access-control add --email viewer@example.com --role Viewer --workspace myproject")
	fmt.Println("  ailang access-control list")
	fmt.Println("  ailang access-control remove --email viewer@example.com")
	fmt.Println("")
	fmt.Println("Roles:")
	fmt.Println("  Approver   Can approve/reject tasks (admin)")
	fmt.Println("  Viewer     Read-only access to dashboard")
	return nil
}

func accessControlAdd(args []string) error {
	var email, role, workspace, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
		case "--workspace", "-w":
			if i+1 < len(args) {
				workspace = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
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
	if workspace == "" {
		workspace = "default"
	}

	// Get Firebase project
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
		return fmt.Errorf("Firebase project not configured. Use --project flag, AILANG_FIREBASE_PROJECT env, or config file")
	}

	// Initialize Firestore
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase: %w", err)
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Firestore client: %w", err)
	}
	defer fsClient.Close()

	// Add user
	docID := fmt.Sprintf("email:%s:%s", email, workspace)
	now := time.Now()

	_, err = fsClient.Collection("access_control").Doc(docID).Set(ctx, map[string]interface{}{
		"email":        email,
		"workspace_id": workspace,
		"role":         role,
		"created_at":   now,
		"updated_at":   now,
	})
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	fmt.Printf("✓ Added %s as %s in workspace '%s'\n", email, role, workspace)
	return nil
}

func accessControlRemove(args []string) error {
	var email, workspace, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--workspace", "-w":
			if i+1 < len(args) {
				workspace = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	if email == "" {
		return fmt.Errorf("--email is required")
	}
	if workspace == "" {
		workspace = "default"
	}

	// Get Firebase project
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
		return fmt.Errorf("Firebase project not configured")
	}

	// Initialize Firestore
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase: %w", err)
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Firestore client: %w", err)
	}
	defer fsClient.Close()

	// Remove user
	docID := fmt.Sprintf("email:%s:%s", email, workspace)
	_, err = fsClient.Collection("access_control").Doc(docID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	fmt.Printf("✓ Removed %s from workspace '%s'\n", email, workspace)
	return nil
}

func accessControlList(args []string) error {
	var workspace, projectID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workspace", "-w":
			if i+1 < len(args) {
				workspace = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectID = args[i+1]
				i++
			}
		}
	}

	// Get Firebase project
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
		return fmt.Errorf("Firebase project not configured")
	}

	// Initialize Firestore
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase: %w", err)
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Firestore client: %w", err)
	}
	defer fsClient.Close()

	// List users
	var docs []*firestore.DocumentSnapshot
	if workspace != "" {
		iter := fsClient.Collection("access_control").Where("workspace_id", "==", workspace).Documents(ctx)
		docs, err = iter.GetAll()
	} else {
		iter := fsClient.Collection("access_control").Documents(ctx)
		docs, err = iter.GetAll()
	}
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(docs) == 0 {
		fmt.Println("No users configured")
		return nil
	}

	fmt.Printf("%-35s %-15s %-10s\n", "EMAIL", "WORKSPACE", "ROLE")
	fmt.Println(strings.Repeat("-", 60))
	for _, doc := range docs {
		data := doc.Data()
		email, _ := data["email"].(string)
		ws, _ := data["workspace_id"].(string)
		role, _ := data["role"].(string)
		if email == "" {
			// Legacy UID-based entry
			uid, _ := data["uid"].(string)
			email = fmt.Sprintf("(uid:%s)", uid)
		}
		fmt.Printf("%-35s %-15s %-10s\n", email, ws, role)
	}

	return nil
}
