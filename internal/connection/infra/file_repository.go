package infra

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/20uf/devcli/internal/connection/domain"
)

// FileConnectionRepository implements domain.ConnectionRepository using JSON files.
// It stores connections in ~/.devcli/connections.json for replay functionality.
type FileConnectionRepository struct {
	filePath string
}

// NewFileConnectionRepository creates a new file-based connection repository.
func NewFileConnectionRepository() (*FileConnectionRepository, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".devcli")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &FileConnectionRepository{
		filePath: filepath.Join(dir, "connections.json"),
	}, nil
}

// connectionRecord is the serializable representation of a connection.
type connectionRecord struct {
	ID           string `json:"id"`
	Cluster      string `json:"cluster"`
	Service      string `json:"service"`
	Container    string `json:"container"`
	ShellCommand string `json:"shell_command"`
	Label        string `json:"label"` // For display/search
	Profile      string `json:"profile"`
	CreatedAt    string `json:"created_at"`
}

// Save persists a connection record to disk.
func (r *FileConnectionRepository) Save(ctx context.Context, conn domain.Connection) error {
	// Read existing records
	records, err := r.loadRecords()
	if err != nil {
		records = []connectionRecord{}
	}

	// Create new record
	record := connectionRecord{
		ID:           conn.ID(),
		Cluster:      conn.Cluster().Name(),
		Service:      conn.Service().Name(),
		Container:    conn.Container().Name(),
		ShellCommand: conn.ShellCommand(),
		CreatedAt:    conn.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	// Append and keep only last 50 entries
	records = append(records, record)
	if len(records) > 50 {
		records = records[len(records)-50:]
	}

	// Write back to disk
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

// FindByLabel retrieves a connection by its label.
func (r *FileConnectionRepository) FindByLabel(ctx context.Context, label string) (*domain.Connection, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, nil
	}

	// Search backwards (most recent first)
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Label == label {
			return r.recordToConnection(records[i])
		}
	}

	return nil, nil
}

// FindRecent retrieves the N most recent connections.
func (r *FileConnectionRepository) FindRecent(ctx context.Context, limit int) ([]domain.Connection, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, nil
	}

	// Return last N records (most recent first)
	var result []domain.Connection
	start := len(records) - limit
	if start < 0 {
		start = 0
	}

	for i := len(records) - 1; i >= start; i-- {
		conn, err := r.recordToConnection(records[i])
		if err == nil && conn != nil {
			result = append(result, *conn)
		}
	}

	return result, nil
}

// loadRecords reads the connection records from disk.
func (r *FileConnectionRepository) loadRecords() ([]connectionRecord, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []connectionRecord{}, nil
		}
		return nil, err
	}

	var records []connectionRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// recordToConnection converts a stored record back to a domain Connection.
func (r *FileConnectionRepository) recordToConnection(record connectionRecord) (*domain.Connection, error) {
	cluster, err := domain.NewCluster(record.Cluster)
	if err != nil {
		return nil, err
	}

	service, err := domain.NewService(record.Service)
	if err != nil {
		return nil, err
	}

	container, err := domain.NewContainer(record.Container)
	if err != nil {
		return nil, err
	}

	// Reconstruct a minimal task with the container
	task := domain.NewTask(record.ID, []domain.Container{container}, domain.TaskStatusRunning)

	conn, err := domain.NewConnection(
		record.ID,
		cluster,
		service,
		task,
		container,
		record.ShellCommand,
	)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}
