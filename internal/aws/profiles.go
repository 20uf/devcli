package aws

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// ErrNoConfigFile is returned when ~/.aws/config does not exist.
var ErrNoConfigFile = errors.New("AWS config file not found")

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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, ErrNoConfigFile
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
