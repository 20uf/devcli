package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Run represents a tracked workflow run.
type Run struct {
	Repo       string    `json:"repo"`
	Workflow   string    `json:"workflow"`
	Branch     string    `json:"branch"`
	RunID      string    `json:"run_id"`
	Label      string    `json:"label"`
	Status     string    `json:"status"`     // queued, in_progress, completed
	Conclusion string    `json:"conclusion"` // success, failure, cancelled, ""
	StartedAt  time.Time `json:"started_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Store manages tracked workflow runs on disk.
type Store struct {
	Runs []Run  `json:"runs"`
	path string
}

// Load reads the tracker file from ~/.devcli/runs.json.
func Load() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".devcli")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "runs.json")
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

// Save writes the tracker to disk.
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Add records a new run to track.
func (s *Store) Add(repo, workflow, branch, runID, label string) {
	s.Runs = append(s.Runs, Run{
		Repo:      repo,
		Workflow:  workflow,
		Branch:    branch,
		RunID:     runID,
		Label:     label,
		Status:    "queued",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// Update sets the status/conclusion for a run.
func (s *Store) Update(runID, status, conclusion string) {
	for i := range s.Runs {
		if s.Runs[i].RunID == runID {
			s.Runs[i].Status = status
			s.Runs[i].Conclusion = conclusion
			s.Runs[i].UpdatedAt = time.Now()
			return
		}
	}
}

// Remove deletes a run from tracking.
func (s *Store) Remove(runID string) {
	for i := range s.Runs {
		if s.Runs[i].RunID == runID {
			s.Runs[i] = s.Runs[len(s.Runs)-1]
			s.Runs = s.Runs[:len(s.Runs)-1]
			return
		}
	}
}

// Active returns runs that are not completed.
func (s *Store) Active() []Run {
	var active []Run
	for _, r := range s.Runs {
		if r.Status != "completed" {
			active = append(active, r)
		}
	}
	return active
}

// All returns all tracked runs (active + recent completed).
func (s *Store) All() []Run {
	return s.Runs
}

// Cleanup removes completed runs older than 1 hour.
func (s *Store) Cleanup() {
	cutoff := time.Now().Add(-1 * time.Hour)
	var kept []Run
	for _, r := range s.Runs {
		if r.Status != "completed" || r.UpdatedAt.After(cutoff) {
			kept = append(kept, r)
		}
	}
	s.Runs = kept
}
