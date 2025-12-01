package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

// handleSelectFolder opens a native folder picker dialog and returns the selected path
// GET /api/select-folder
// Cross-platform: uses osascript (macOS), zenity/kdialog (Linux), PowerShell (Windows)
func (s *Server) handleSelectFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path, err := openFolderPicker()
	if err != nil {
		// User cancelled or error occurred
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"cancelled": true,
			"path":      "",
			"error":     err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"cancelled": false,
		"path":      path,
	})
}

// openFolderPicker opens a native folder picker dialog and returns the selected path
// Cross-platform implementation:
// - macOS: osascript (AppleScript)
// - Linux: zenity (GTK) or kdialog (KDE)
// - Windows: PowerShell with FolderBrowserDialog
func openFolderPicker() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// macOS: Use osascript with AppleScript
		script := `tell application "System Events"
			activate
			set folderPath to POSIX path of (choose folder with prompt "Select workspace directory")
			return folderPath
		end tell`
		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("folder picker cancelled or failed: %w", err)
		}
		return strings.TrimSpace(string(output)), nil

	case "linux":
		// Linux: Try zenity first (GTK), fall back to kdialog (KDE)
		if _, err := exec.LookPath("zenity"); err == nil {
			cmd := exec.Command("zenity", "--file-selection", "--directory", "--title=Select workspace directory")
			output, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("folder picker cancelled or failed: %w", err)
			}
			return strings.TrimSpace(string(output)), nil
		}
		if _, err := exec.LookPath("kdialog"); err == nil {
			cmd := exec.Command("kdialog", "--getexistingdirectory", ".", "--title", "Select workspace directory")
			output, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("folder picker cancelled or failed: %w", err)
			}
			return strings.TrimSpace(string(output)), nil
		}
		return "", fmt.Errorf("no folder picker available (install zenity or kdialog)")

	case "windows":
		// Windows: Use PowerShell with FolderBrowserDialog
		script := `Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "Select workspace directory"
$dialog.ShowNewFolderButton = $true
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    Write-Output $dialog.SelectedPath
} else {
    exit 1
}`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("folder picker cancelled or failed: %w", err)
		}
		return strings.TrimSpace(string(output)), nil

	default:
		return "", fmt.Errorf("folder picker not supported on %s", runtime.GOOS)
	}
}
