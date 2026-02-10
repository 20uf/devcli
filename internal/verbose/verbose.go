package verbose

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	enabled bool

	debugStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22D3EE")).Bold(true)
)

// Enable turns verbose logging on.
func Enable() { enabled = true }

// IsEnabled returns whether verbose mode is active.
func IsEnabled() bool { return enabled }

// Cmd logs the command being executed and returns it unchanged.
func Cmd(cmd *exec.Cmd) *exec.Cmd {
	if !enabled {
		return cmd
	}
	args := strings.Join(cmd.Args, " ")
	fmt.Printf("%s %s\n", labelStyle.Render("[exec]"), debugStyle.Render(args))
	return cmd
}

// Log prints a debug message when verbose mode is active.
func Log(format string, a ...any) {
	if !enabled {
		return
	}
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", labelStyle.Render("[debug]"), debugStyle.Render(msg))
}
