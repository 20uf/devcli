package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// FileTrackerRepository implements TrackerRepository using file-based storage.
type FileTrackerRepository struct {
	storePath string
}

// NewFileTrackerRepository creates a new file-based tracker repository.
func NewFileTrackerRepository(storePath string) *FileTrackerRepository {
	return &FileTrackerRepository{
		storePath: storePath,
	}
}

// trackedRecord is the serializable format for TrackedDeployment.
type trackedRecord struct {
	ID          string `json:"id"`
	RunID       string `json:"run_id"`
	Workflow    string `json:"workflow"`
	Branch      string `json:"branch"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	StartedAt   int64  `json:"started_at"`
	CompletedAt *int64 `json:"completed_at,omitempty"`
	Repo        string `json:"repo"`
}

// Save persists a tracked deployment.
func (r *FileTrackerRepository) Save(ctx context.Context, tracked domain.TrackedDeployment) error {
	if err := os.MkdirAll(r.storePath, 0755); err != nil {
		return fmt.Errorf("failed to create tracker store: %w", err)
	}

	record := trackedRecord{
		ID:        tracked.ID(),
		RunID:     tracked.RunID(),
		Workflow:  tracked.Workflow().Name(),
		Branch:    tracked.Branch(),
		Status:    string(tracked.Status()),
		Conclusion: string(tracked.Conclusion()),
		StartedAt: tracked.StartedAt().Unix(),
		Repo:      tracked.Repo(),
	}

	if tracked.CompletedAt() != nil {
		completedUnix := tracked.CompletedAt().Unix()
		record.CompletedAt = &completedUnix
	}

	filePath := filepath.Join(r.storePath, tracked.ID()+".json")
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracked deployment: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save tracked deployment: %w", err)
	}

	return nil
}

// List retrieves all tracked deployments.
func (r *FileTrackerRepository) List(ctx context.Context) ([]domain.TrackedDeployment, error) {
	entries, err := os.ReadDir(r.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.TrackedDeployment{}, nil
		}
		return nil, fmt.Errorf("failed to list tracked deployments: %w", err)
	}

	var tracked []domain.TrackedDeployment
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		td, err := r.loadFromFile(filepath.Join(r.storePath, entry.Name()))
		if err != nil {
			continue
		}

		tracked = append(tracked, *td)
	}

	return tracked, nil
}

// GetByID retrieves a specific tracked deployment.
func (r *FileTrackerRepository) GetByID(ctx context.Context, id string) (*domain.TrackedDeployment, error) {
	filePath := filepath.Join(r.storePath, id+".json")
	return r.loadFromFile(filePath)
}

// Remove removes a tracked deployment.
func (r *FileTrackerRepository) Remove(ctx context.Context, id string) error {
	filePath := filepath.Join(r.storePath, id+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove tracked deployment: %w", err)
	}
	return nil
}

// ListActive retrieves only active deployments (queued or in-progress).
func (r *FileTrackerRepository) ListActive(ctx context.Context) ([]domain.TrackedDeployment, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var active []domain.TrackedDeployment
	for _, td := range all {
		if td.IsActive() {
			active = append(active, td)
		}
	}

	return active, nil
}

// Cleanup removes stale deployments.
func (r *FileTrackerRepository) Cleanup(ctx context.Context, maxAgeSecs int64) (removed int, err error) {
	all, err := r.List(ctx)
	if err != nil {
		return 0, err
	}

	maxAge := time.Duration(maxAgeSecs) * time.Second
	count := 0

	for _, td := range all {
		if td.IsStale(maxAge) {
			if err := r.Remove(ctx, td.ID()); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// loadFromFile reconstructs a TrackedDeployment from JSON file.
func (r *FileTrackerRepository) loadFromFile(filePath string) (*domain.TrackedDeployment, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read tracked deployment: %w", err)
	}

	var record trackedRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracked deployment: %w", err)
	}

	workflow, err := domain.NewWorkflow(record.Workflow)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow in tracked deployment: %w", err)
	}

	td := domain.NewTrackedDeployment(record.RunID, workflow, record.Branch, record.Repo)

	status := domain.RunStatus(record.Status)
	td.UpdateStatus(status)

	if record.Conclusion != "" {
		conclusion := domain.RunConclusion(record.Conclusion)
		td.UpdateConclusion(conclusion)
	}

	return &td, nil
}
