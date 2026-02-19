package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var flagSetup bool

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate or install shell completion",
	Long: `Generate shell completion script.

By default, prints the completion script to stdout.
Use --setup to automatically install it for your shell.`,
	ValidArgs: []string{"bash", "zsh", "fish"},
	Args:      cobra.ExactArgs(1),
	RunE:      runCompletion,
}

func init() {
	completionCmd.Flags().BoolVar(&flagSetup, "setup", false, "Automatically install completion for your shell")
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	shell := args[0]

	if !flagSetup {
		return printCompletion(cmd, shell)
	}

	return setupCompletion(cmd, shell)
}

func printCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		fmt.Println("# Add this to your ~/.bashrc:")
		fmt.Println("#   eval \"$(devcli completion bash)\"")
		fmt.Println("#")
		fmt.Println("# Or save to a file:")
		fmt.Println("#   devcli completion bash > /etc/bash_completion.d/devcli")
		fmt.Println()
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		fmt.Println("# Add this to your ~/.zshrc:")
		fmt.Println("#   eval \"$(devcli completion zsh)\"")
		fmt.Println("#")
		fmt.Println("# Or install automatically:")
		fmt.Println("#   devcli completion zsh --setup")
		fmt.Println()
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		fmt.Println("# Save to fish completions directory:")
		fmt.Println("#   devcli completion fish > ~/.config/fish/completions/devcli.fish")
		fmt.Println("#")
		fmt.Println("# Or install automatically:")
		fmt.Println("#   devcli completion fish --setup")
		fmt.Println()
		return rootCmd.GenFishCompletion(os.Stdout, true)
	default:
		return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
	}
}

func setupCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return setupBash()
	case "zsh":
		return setupZsh()
	case "fish":
		return setupFish()
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func setupBash() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Try system-wide first, fallback to user
	completionDir := "/etc/bash_completion.d"
	completionFile := filepath.Join(completionDir, "devcli")

	if _, err := os.Stat(completionDir); os.IsNotExist(err) {
		// Fallback: add to .bashrc
		return addToRCFile(filepath.Join(home, ".bashrc"), "bash")
	}

	return writeCompletionFile(completionFile, "bash")
}

func setupZsh() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Check common zsh completion directories
	dirs := []string{
		filepath.Join(home, ".zsh/completions"),
		filepath.Join(home, ".zfunc"),
	}

	// Find or create a completion directory
	var completionDir string
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			completionDir = d
			break
		}
	}

	if completionDir == "" {
		completionDir = dirs[0]
		if err := os.MkdirAll(completionDir, 0755); err != nil {
			return fmt.Errorf("failed to create completion directory: %w", err)
		}
	}

	completionFile := filepath.Join(completionDir, "_devcli")

	// Check if file already exists
	if _, err := os.Stat(completionFile); err == nil {
		fmt.Printf("Completion file already exists: %s\n", completionFile)
		fmt.Printf("Overwrite? [y/N] ")
		var reply string
		if _, err := fmt.Scanln(&reply); err != nil {
			reply = "n"
		}
		if strings.ToLower(strings.TrimSpace(reply)) != "y" {
			fmt.Println("Skipped.")
			return nil
		}
	}

	// Write completion file
	f, err := os.Create(completionFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", completionFile, err)
	}
	defer f.Close() //nolint:errcheck

	if err := rootCmd.GenZshCompletion(f); err != nil {
		return err
	}

	// Ensure fpath includes the directory in .zshrc
	rcFile := filepath.Join(home, ".zshrc")
	fpathLine := fmt.Sprintf("fpath=(%s $fpath)", completionDir)
	initLine := "autoload -Uz compinit && compinit"

	if err := ensureLineInFile(rcFile, fpathLine, "fpath="); err != nil {
		return err
	}
	if err := ensureLineInFile(rcFile, initLine, "compinit"); err != nil {
		return err
	}

	fmt.Printf("Completion installed: %s\n", completionFile)
	fmt.Println("Restart your shell or run: source ~/.zshrc")
	return nil
}

func setupFish() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	completionDir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}

	completionFile := filepath.Join(completionDir, "devcli.fish")
	return writeCompletionFile(completionFile, "fish")
}

func writeCompletionFile(path, shell string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Completion file already exists: %s\n", path)
		fmt.Printf("Overwrite? [y/N] ")
		var reply string
		if _, err := fmt.Scanln(&reply); err != nil {
			reply = "n"
		}
		if strings.ToLower(strings.TrimSpace(reply)) != "y" {
			fmt.Println("Skipped.")
			return nil
		}
	}

	f, err := os.Create(path)
	if err != nil {
		// Try with sudo for system directories
		return writeCompletionWithSudo(path, shell)
	}
	defer f.Close() //nolint:errcheck

	switch shell {
	case "bash":
		if err := rootCmd.GenBashCompletion(f); err != nil {
			return err
		}
	case "fish":
		if err := rootCmd.GenFishCompletion(f, true); err != nil {
			return err
		}
	}

	fmt.Printf("Completion installed: %s\n", path)
	return nil
}

func writeCompletionWithSudo(path, shell string) error {
	fmt.Println("Writing to system directory requires sudo...")

	tmpFile, err := os.CreateTemp("", "devcli-completion-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	switch shell {
	case "bash":
		if err := rootCmd.GenBashCompletion(tmpFile); err != nil {
			tmpFile.Close() //nolint:errcheck
			return err
		}
	}
	tmpFile.Close() //nolint:errcheck

	c := exec.Command("sudo", "cp", tmpFile.Name(), path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Printf("Completion installed: %s\n", path)
	return nil
}

func addToRCFile(rcFile, shell string) error {
	line := fmt.Sprintf("eval \"$(devcli completion %s)\"", shell)
	if err := ensureLineInFile(rcFile, line, "devcli completion"); err != nil {
		return err
	}
	fmt.Printf("Added to %s: %s\n", rcFile, line)
	fmt.Println("Restart your shell or run: source " + rcFile)
	return nil
}

func ensureLineInFile(path, line, marker string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if a similar line already exists
	for _, existing := range strings.Split(string(content), "\n") {
		if strings.Contains(existing, marker) && !strings.HasPrefix(strings.TrimSpace(existing), "#") {
			return nil // Already configured
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	_, err = fmt.Fprintf(f, "\n# devcli shell completion\n%s\n", line)
	return err
}

