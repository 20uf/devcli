package aws

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Dependency struct {
	Name         string
	Check        string
	InstallURL   string
	InstallMac   string
	InstallLinux string
}

var requiredDeps = []Dependency{
	{
		Name:         "aws",
		Check:        "aws",
		InstallURL:   "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
		InstallMac:   "brew install awscli",
		InstallLinux: "curl \"https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip\" -o /tmp/awscliv2.zip && unzip -o /tmp/awscliv2.zip -d /tmp && sudo /tmp/aws/install && rm -rf /tmp/aws /tmp/awscliv2.zip",
	},
	{
		Name:         "session-manager-plugin",
		Check:        "session-manager-plugin",
		InstallURL:   "https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
		InstallMac:   "brew install --cask session-manager-plugin",
		InstallLinux: "curl \"https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb\" -o /tmp/session-manager-plugin.deb && sudo dpkg -i /tmp/session-manager-plugin.deb && rm /tmp/session-manager-plugin.deb",
	},
}

// CheckDependencies verifies that all required CLI tools are installed.
// If missing, offers to install them automatically on supported platforms.
func CheckDependencies() error {
	var missing []Dependency

	for _, dep := range requiredDeps {
		if _, err := exec.LookPath(dep.Check); err != nil {
			missing = append(missing, dep)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	platform := runtime.GOOS
	canAutoInstall := platform == "darwin" || platform == "linux"

	for _, dep := range missing {
		fmt.Printf("Missing dependency: %s\n", dep.Name)

		if !canAutoInstall {
			fmt.Printf("  Install manually: %s\n", dep.InstallURL)
			continue
		}

		installCmd := dep.InstallLinux
		if platform == "darwin" {
			installCmd = dep.InstallMac
		}

		fmt.Printf("  Install command: %s\n", installCmd)
		fmt.Printf("Install %s now? [y/N] ", dep.Name)

		var reply string
		fmt.Scanln(&reply)
		reply = strings.TrimSpace(strings.ToLower(reply))

		if reply != "y" && reply != "yes" {
			return fmt.Errorf("missing required dependency: %s\n  Install: %s", dep.Name, dep.InstallURL)
		}

		fmt.Printf("Installing %s...\n", dep.Name)
		cmd := exec.Command("sh", "-c", installCmd)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w\n  Install manually: %s", dep.Name, err, dep.InstallURL)
		}

		// Verify installation
		if _, err := exec.LookPath(dep.Check); err != nil {
			return fmt.Errorf("%s installed but not found in PATH. Restart your shell and try again", dep.Name)
		}

		fmt.Printf("%s installed successfully.\n", dep.Name)
	}

	return nil
}
