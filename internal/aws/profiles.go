package aws

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// ListProfiles returns all profile names from ~/.aws/config.
func ListProfiles() ([]string, error) {
	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, ".aws", "config")
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, err
	}

	var profiles []string
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "DEFAULT" || name == "default" {
			continue
		}
		// AWS config uses "profile xxx" for named profiles
		name = strings.TrimPrefix(name, "profile ")
		profiles = append(profiles, name)
	}

	sort.Strings(profiles)
	return profiles, nil
}
