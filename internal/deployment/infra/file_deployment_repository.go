package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// FileDeploymentRepository implements DeploymentRepository using local file storage.
type FileDeploymentRepository struct {
	storePath string
}

// NewFileDeploymentRepository creates a new file-based deployment repository.
func NewFileDeploymentRepository(storePath string) *FileDeploymentRepository {
	return &FileDeploymentRepository{
		storePath: storePath,
	}
}

// deploymentRecord is the serializable format for Deployment.
type deploymentRecord struct {
	ID        string            `json:"id"`
	Workflow  string            `json:"workflow"`
	Branch    string            `json:"branch"`
	Inputs    map[string]string `json:"inputs"`
	Timestamp string            `json:"timestamp"`
	RunID     string            `json:"run_id,omitempty"`
	Status    string            `json:"status,omitempty"`
}

// Save persists a deployment record.
func (r *FileDeploymentRepository) Save(ctx context.Context, deployment domain.Deployment) error {
	// Create store directory if needed
	if err := os.MkdirAll(r.storePath, 0755); err != nil {
		return fmt.Errorf("failed to create deployment store: %w", err)
	}

	// Convert deployment to record
	record := deploymentRecord{
		ID:       deployment.ID(),
		Workflow: deployment.Workflow().Name(),
		Branch:   deployment.Branch(),
		Inputs:   r.inputsToMap(deployment.Inputs()),
	}

	if deployment.HasRun() {
		record.RunID = deployment.Run().ID()
		record.Status = string(deployment.Run().Status())
	}

	// Save to file
	filePath := filepath.Join(r.storePath, deployment.ID()+".json")
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	return nil
}

// FindByID retrieves a deployment by its ID.
func (r *FileDeploymentRepository) FindByID(ctx context.Context, id string) (*domain.Deployment, error) {
	filePath := filepath.Join(r.storePath, id+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read deployment: %w", err)
	}

	var record deploymentRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	// Reconstruct deployment from record
	workflow, _ := domain.NewWorkflow(record.Workflow)
	deployment, _ := domain.NewDeployment(record.ID, workflow, record.Branch, "")

	// Restore inputs
	for key, value := range record.Inputs {
		// This is simplified - in practice would need input type info
		input, _ := domain.NewInput(key, domain.InputTypeString, value, false)
		// Would need to set value in the input
		_ = input
	}

	return &deployment, nil
}

// FindRecent retrieves recent deployments.
func (r *FileDeploymentRepository) FindRecent(ctx context.Context, limit int) ([]domain.Deployment, error) {
	entries, err := os.ReadDir(r.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Deployment{}, nil
		}
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var deployments []domain.Deployment
	for i, entry := range entries {
		if i >= limit {
			break
		}

		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(r.storePath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var record deploymentRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		workflow, _ := domain.NewWorkflow(record.Workflow)
		deployment, _ := domain.NewDeployment(record.ID, workflow, record.Branch, "")
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// Helper: Convert inputs slice to map
func (r *FileDeploymentRepository) inputsToMap(inputs []domain.Input) map[string]string {
	result := make(map[string]string)
	for _, input := range inputs {
		result[input.Key()] = input.Value()
	}
	return result
}
