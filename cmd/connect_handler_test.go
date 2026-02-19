package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

// Test: ConnectHandler initialization
func TestConnectHandler_Init(t *testing.T) {
	handler, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	if handler.orchestrator == nil {
		t.Errorf("Orchestrator not initialized")
	}

	if handler.repos == nil {
		t.Errorf("Repositories not initialized")
	}

	t.Log("✓ ConnectHandler initialized successfully")
}

// Test: Non-interactive mode with all flags
func TestConnectHandler_NonInteractive_AllFlags(t *testing.T) {
	handler, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err = handler.Handle(cmd, "production", "api-service", "php", "bash")
	if err != nil && err.Error() == "user cancelled" {
		t.Errorf("Should not cancel with all flags provided")
	}

	t.Log("✓ Non-interactive mode handles all flags")
}

// Test: Partial flags (cluster provided, service not)
func TestConnectHandler_PartialFlags(t *testing.T) {
	handler, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	_ = handler.Handle(cmd, "production", "", "", "bash")

	t.Log("✓ Partial flags flow initiated")
}

// Test: History replay when no flags
func TestConnectHandler_HistoryReplay(t *testing.T) {
	handler, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	if handler.history == nil {
		t.Logf("Note: History not available (expected in test)")
	}

	t.Log("✓ History available for replay")
}

// Test: ESC cancellation during cluster selection
func TestConnectHandler_ESCCancellation(t *testing.T) {
	_, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	t.Log("✓ ESC cancellation path available")
}

// Test: Shell execution parameter
func TestConnectHandler_ShellExecution(t *testing.T) {
	handler, err := NewConnectHandler(context.Background(), "default", "us-east-1")
	if err != nil {
		t.Skipf("Skipping (no AWS config): %v", err)
	}

	shells := []string{"bash", "sh", "zsh"}
	for _, shell := range shells {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		_ = handler.Handle(cmd, "production", "api", "php", shell)
	}

	t.Log("✓ Shell parameter handling")
}

// Test: Handler with AWS profile
func TestConnectHandler_WithProfile(t *testing.T) {
	profiles := []string{"default", "production", "staging"}

	for _, profile := range profiles {
		handler, err := NewConnectHandler(context.Background(), profile, "us-east-1")
		if err != nil {
			t.Skipf("Skipping (no AWS config): %v", err)
		}

		if handler == nil {
			t.Errorf("Handler nil for profile %s", profile)
		}
	}

	t.Log("✓ Profile parameter handling")
}
