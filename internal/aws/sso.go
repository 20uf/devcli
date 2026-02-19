package aws

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/20uf/devcli/internal/verbose"
	"gopkg.in/ini.v1"
)

// IsSSO returns true if the given profile uses SSO authentication.
func IsSSO(profile string) bool {
	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		configPath = filepath.Join(home, ".aws", "config")
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		return false
	}

	sectionName := "profile " + profile
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		// Try without "profile " prefix (for [default])
		section, err = cfg.GetSection(profile)
		if err != nil {
			return false
		}
	}

	return section.HasKey("sso_start_url") || section.HasKey("sso_session")
}

// EnsureSSOLogin checks if the SSO session is valid. If not, triggers aws sso login.
func EnsureSSOLogin(profile string) error {
	if !IsSSO(profile) {
		return nil
	}

	// Quick check: try sts get-caller-identity to see if session is valid
	check := verbose.Cmd(exec.Command("aws", "sts", "get-caller-identity", "--profile", profile))
	check.Stderr = nil
	check.Stdout = nil
	if err := check.Run(); err == nil {
		return nil
	}

	fmt.Printf("SSO session expired for profile %q, logging in...\n", profile)

	login := verbose.Cmd(exec.Command("aws", "sso", "login", "--profile", profile))
	login.Stdin = os.Stdin
	login.Stdout = os.Stdout
	login.Stderr = os.Stderr

	if err := login.Run(); err != nil {
		return fmt.Errorf("SSO login failed: %w", err)
	}

	// Verify login succeeded
	verify := verbose.Cmd(exec.Command("aws", "sts", "get-caller-identity", "--profile", profile))
	out, err := verify.Output()
	if err != nil {
		return fmt.Errorf("SSO login succeeded but credentials are still invalid")
	}

	_ = out
	fmt.Println("SSO login successful.")
	return nil
}

// FormatSSOError returns a user-friendly message for SSO-related errors.
func FormatSSOError(err error, profile string) string {
	msg := err.Error()
	if strings.Contains(msg, "SSO") || strings.Contains(msg, "sso") ||
		strings.Contains(msg, "expired") || strings.Contains(msg, "invalid") {
		return fmt.Sprintf("AWS SSO session expired. Run: aws sso login --profile %s", profile)
	}
	return msg
}

// ForceSSOLogin triggers SSO login unconditionally (skips the identity check).
func ForceSSOLogin(profile string) error {
	fmt.Printf("Refreshing SSO session for profile %q...\n", profile)

	login := verbose.Cmd(exec.Command("aws", "sso", "login", "--profile", profile))
	login.Stdin = os.Stdin
	login.Stdout = os.Stdout
	login.Stderr = os.Stderr

	if err := login.Run(); err != nil {
		return fmt.Errorf("SSO login failed: %w", err)
	}

	fmt.Println("SSO login successful.")
	return nil
}
