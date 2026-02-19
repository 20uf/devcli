package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/20uf/devcli/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// If version is "dev" (go run), try to read from VERSION file
	if version == "dev" {
		if v, err := readVersionFile(); err == nil && v != "" {
			version = v
		}
	}

	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}

// readVersionFile tries to read version from VERSION file
func readVersionFile() (string, error) {
	// Try multiple paths: current dir, parent, home dir
	paths := []string{
		"VERSION",
		"../VERSION",
		filepath.Join(os.Getenv("HOME"), "Projects/devcli/VERSION"),
	}

	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			version := strings.TrimSpace(string(data))
			if version != "" {
				return version, nil
			}
		}
	}

	return "", fmt.Errorf("VERSION file not found")
}
