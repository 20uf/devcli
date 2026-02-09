package aws

import (
	"fmt"
	"os/exec"
	"strings"
)

type Dependency struct {
	Name    string
	Check   string
	Install string
}

var requiredDeps = []Dependency{
	{
		Name:    "aws",
		Check:   "aws",
		Install: "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
	},
	{
		Name:    "session-manager-plugin",
		Check:   "session-manager-plugin",
		Install: "https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
	},
}

// CheckDependencies verifies that all required CLI tools are installed.
func CheckDependencies() error {
	var missing []string

	for _, dep := range requiredDeps {
		if _, err := exec.LookPath(dep.Check); err != nil {
			missing = append(missing, fmt.Sprintf("  - %s: %s", dep.Name, dep.Install))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required dependencies:\n%s", strings.Join(missing, "\n"))
	}

	return nil
}
