package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed all:editor_assets
var editorAssets embed.FS

func editorCommand() {
	if flag.NArg() < 2 {
		printEditorHelp()
		return
	}

	subcommand := flag.Arg(1)

	switch subcommand {
	case "install":
		if flag.NArg() < 3 {
			fmt.Fprintf(os.Stderr, "%s: missing editor name\n", red("Error"))
			fmt.Println("Usage: ailang editor install <vscode|vim|neovim>")
			os.Exit(1)
		}
		editor := flag.Arg(2)
		installEditor(editor)

	case "uninstall":
		if flag.NArg() < 3 {
			fmt.Fprintf(os.Stderr, "%s: missing editor name\n", red("Error"))
			fmt.Println("Usage: ailang editor uninstall <vscode|vim|neovim>")
			os.Exit(1)
		}
		editor := flag.Arg(2)
		uninstallEditor(editor)

	case "status":
		checkEditorStatus()

	case "help", "--help", "-h":
		printEditorHelp()

	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subcommand)
		printEditorHelp()
		os.Exit(1)
	}
}

func printEditorHelp() {
	fmt.Println("AILANG Editor Support")
	fmt.Println()
	fmt.Println("Usage: ailang editor <command> [editor]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install <editor>    Install syntax highlighting for an editor")
	fmt.Println("  uninstall <editor>  Remove AILANG support from an editor")
	fmt.Println("  status              Check installation status for all editors")
	fmt.Println()
	fmt.Println("Supported editors:")
	fmt.Println("  vscode    Visual Studio Code")
	fmt.Println("  vim       Vim (~/.vim/)")
	fmt.Println("  neovim    Neovim (~/.config/nvim/)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang editor install vscode")
	fmt.Println("  ailang editor status")
}

func installEditor(editor string) {
	switch strings.ToLower(editor) {
	case "vscode", "code":
		installVSCode()
	case "vim":
		installVim(false)
	case "neovim", "nvim":
		installVim(true)
	default:
		fmt.Fprintf(os.Stderr, "%s: unsupported editor '%s'\n", red("Error"), editor)
		fmt.Println("Supported: vscode, vim, neovim")
		os.Exit(1)
	}
}

func uninstallEditor(editor string) {
	switch strings.ToLower(editor) {
	case "vscode", "code":
		uninstallVSCode()
	case "vim":
		uninstallVim(false)
	case "neovim", "nvim":
		uninstallVim(true)
	default:
		fmt.Fprintf(os.Stderr, "%s: unsupported editor '%s'\n", red("Error"), editor)
		os.Exit(1)
	}
}

func installVim(isNeovim bool) {
	name := "Vim"
	var configDir string

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot determine home directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if isNeovim {
		name = "Neovim"
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			configDir = filepath.Join(home, ".config", "nvim")
		} else {
			configDir = filepath.Join(home, "AppData", "Local", "nvim")
		}
	} else {
		configDir = filepath.Join(home, ".vim")
	}

	fmt.Printf("%s Installing AILANG %s syntax...\n", cyan("→"), name)

	// Create directories
	dirs := []string{
		filepath.Join(configDir, "syntax"),
		filepath.Join(configDir, "ftdetect"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot create directory %s: %v\n", red("Error"), d, err)
			os.Exit(1)
		}
	}

	// Copy files
	files := []struct {
		src string
		dst string
	}{
		{"editor_assets/vim/syntax/ailang.vim", filepath.Join(configDir, "syntax", "ailang.vim")},
		{"editor_assets/vim/ftdetect/ailang.vim", filepath.Join(configDir, "ftdetect", "ailang.vim")},
	}

	for _, f := range files {
		content, err := editorAssets.ReadFile(f.src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot read embedded file %s: %v\n", red("Error"), f.src, err)
			os.Exit(1)
		}
		if err := os.WriteFile(f.dst, content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write file %s: %v\n", red("Error"), f.dst, err)
			os.Exit(1)
		}
	}

	fmt.Printf("%s %s syntax installed to %s\n", green("✓"), name, configDir)
}

func uninstallVim(isNeovim bool) {
	name := "Vim"
	var configDir string

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot determine home directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if isNeovim {
		name = "Neovim"
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			configDir = filepath.Join(home, ".config", "nvim")
		} else {
			configDir = filepath.Join(home, "AppData", "Local", "nvim")
		}
	} else {
		configDir = filepath.Join(home, ".vim")
	}

	files := []string{
		filepath.Join(configDir, "syntax", "ailang.vim"),
		filepath.Join(configDir, "ftdetect", "ailang.vim"),
	}

	removed := false
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			os.Remove(f)
			removed = true
		}
	}

	if removed {
		fmt.Printf("%s %s syntax files removed\n", green("✓"), name)
	} else {
		fmt.Printf("%s syntax not installed\n", name)
	}
}

func checkEditorStatus() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot determine home directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Println("AILANG Editor Status")
	fmt.Println()

	// VS Code. Ask the editor whether it will actually load the extension —
	// stat'ing the folder reports ✓ for an install VS Code has deregistered.
	if m, mErr := readVSCodeManifest(); mErr != nil {
		fmt.Printf("  %s VS Code: cannot read embedded extension manifest: %v\n", red("✗"), mErr)
	} else {
		st := vscodeExtensionStatus(home, m, exec.LookPath, execRunner)
		if st.Installed {
			version := st.Version
			if version == "" {
				version = "unknown version"
			}
			fmt.Printf("  %s VS Code: %s@%s (%s)\n", green("✓"), m.extID(), version, st.Detail)
		} else {
			fmt.Printf("  %s VS Code: %s\n", yellow("○"), st.Detail)
		}
	}

	// Vim
	vimSyntax := filepath.Join(home, ".vim", "syntax", "ailang.vim")
	if _, err := os.Stat(vimSyntax); err == nil {
		fmt.Printf("  %s Vim: %s\n", green("✓"), filepath.Join(home, ".vim"))
	} else {
		fmt.Printf("  %s Vim: not installed\n", yellow("○"))
	}

	// Neovim
	var nvimDir string
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		nvimDir = filepath.Join(home, ".config", "nvim")
	} else {
		nvimDir = filepath.Join(home, "AppData", "Local", "nvim")
	}
	nvimSyntax := filepath.Join(nvimDir, "syntax", "ailang.vim")
	if _, err := os.Stat(nvimSyntax); err == nil {
		fmt.Printf("  %s Neovim: %s\n", green("✓"), nvimDir)
	} else {
		fmt.Printf("  %s Neovim: not installed\n", yellow("○"))
	}

	fmt.Println()
	fmt.Println("Install with: ailang editor install <editor>")
}
