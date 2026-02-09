package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const maxEntries = 50

type Entry struct {
	Command   string    `json:"command"`
	Label     string    `json:"label"`
	Args      []string  `json:"args"`
	Timestamp time.Time `json:"timestamp"`
}

type Store struct {
	Entries []Entry `json:"entries"`
	path    string
}

// Load reads the history file from ~/.devcli/history.json.
func Load() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".devcli")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "history.json")
	store := &Store{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, store); err != nil {
		return store, nil
	}

	return store, nil
}

// Save writes the history to disk.
func (s *Store) Save() error {
	// Keep only the last N entries
	if len(s.Entries) > maxEntries {
		s.Entries = s.Entries[len(s.Entries)-maxEntries:]
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Add records a new command execution.
func (s *Store) Add(command, label string, args []string) {
	s.Entries = append(s.Entries, Entry{
		Command:   command,
		Label:     label,
		Args:      args,
		Timestamp: time.Now(),
	})
}

// Labels returns display labels for the last N entries (most recent first).
func (s *Store) Labels(command string) []string {
	var labels []string
	seen := make(map[string]bool)

	for i := len(s.Entries) - 1; i >= 0; i-- {
		e := s.Entries[i]
		if command != "" && e.Command != command {
			continue
		}
		if seen[e.Label] {
			continue
		}
		seen[e.Label] = true
		labels = append(labels, fmt.Sprintf("%s (%s)", e.Label, e.Timestamp.Format("02 Jan 15:04")))
	}

	return labels
}

// FindByLabel returns the entry matching the given label prefix.
func (s *Store) FindByLabel(command, labelPrefix string) *Entry {
	for i := len(s.Entries) - 1; i >= 0; i-- {
		e := s.Entries[i]
		if command != "" && e.Command != command {
			continue
		}
		if len(labelPrefix) > 0 && len(e.Label) >= len(labelPrefix) && e.Label[:len(labelPrefix)] == labelPrefix {
			return &s.Entries[i]
		}
	}
	return nil
}
