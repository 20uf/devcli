package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/20uf/devcli/internal/connection/domain"
)

// HistoryAdapterRepository implements domain.ConnectionRepository using the legacy history.Store.
// This adapter bridges the old history system with the new domain model.
// It stores connection metadata as JSON in the history entry args.
type HistoryAdapterRepository struct {
	historyPath string
}

// NewHistoryAdapterRepository creates a new history adapter for connections.
func NewHistoryAdapterRepository(historyPath string) *HistoryAdapterRepository {
	return &HistoryAdapterRepository{
		historyPath: historyPath,
	}
}

// connectionMetadata is stored in history.Entry.Args for serialization.
type connectionMetadata struct {
	Cluster      string `json:"cluster"`
	Service      string `json:"service"`
	Container    string `json:"container"`
	ShellCommand string `json:"shell_command"`
	Profile      string `json:"profile"`
}

// Save persists a connection to history as a "connect" command entry.
func (r *HistoryAdapterRepository) Save(ctx context.Context, conn domain.Connection) error {
	// For now, we just log it (real implementation would use history.Store)
	// This is a placeholder that shows the pattern
	metadata := connectionMetadata{
		Cluster:      conn.Cluster().Name(),
		Service:      conn.Service().Name(),
		Container:    conn.Container().Name(),
		ShellCommand: conn.ShellCommand(),
	}

	data, _ := json.Marshal(metadata)
	_ = data // Placeholder for history storage

	return nil
}

// FindByLabel retrieves a connection by its display label.
// Label format: "profile → cluster/service/container"
func (r *HistoryAdapterRepository) FindByLabel(ctx context.Context, label string) (*domain.Connection, error) {
	// Parse label: "profile → cluster/service/container"
	parts := strings.Split(label, " → ")
	if len(parts) != 2 {
		return nil, nil
	}

	profile := parts[0]
	resourcePath := parts[1]

	segments := strings.Split(resourcePath, "/")
	if len(segments) != 3 {
		return nil, nil
	}

	clusterName, serviceName, containerName := segments[0], segments[1], segments[2]

	// Reconstruct connection from label
	cluster, err := domain.NewCluster(clusterName)
	if err != nil {
		return nil, err
	}

	service, err := domain.NewService(serviceName)
	if err != nil {
		return nil, err
	}

	container, err := domain.NewContainer(containerName)
	if err != nil {
		return nil, err
	}

	// Minimal task reconstruction
	task := domain.NewTask(
		uuid.New().String(),
		[]domain.Container{container},
		domain.TaskStatusRunning,
	)

	conn, err := domain.NewConnection(
		fmt.Sprintf("conn-%s", profile),
		cluster,
		service,
		task,
		container,
		"su -s /bin/sh www-data", // Default shell
	)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

// FindRecent retrieves recent connections (placeholder for now).
func (r *HistoryAdapterRepository) FindRecent(ctx context.Context, limit int) ([]domain.Connection, error) {
	// Placeholder: real implementation would read from history.Store
	return []domain.Connection{}, nil
}

// IntegrationHelper provides utilities for connecting old history to new domain.
type IntegrationHelper struct {
	historyEntryCommand string // "connect", "deploy", etc.
	historyEntryLabel   string  // Display label from history
	historyEntryArgs    []string // Args from history
}

// NewIntegrationHelper creates a helper for bridging history and domain.
func NewIntegrationHelper(command, label string, args []string) *IntegrationHelper {
	return &IntegrationHelper{
		historyEntryCommand: command,
		historyEntryLabel:   label,
		historyEntryArgs:    args,
	}
}

// ParseConnectionArgs parses the args array from history into a connection request.
// Expected format: ["--profile", "dev", "--cluster", "prod", "--service", "api", "--container", "php"]
func (h *IntegrationHelper) ParseConnectionArgs() (profile, cluster, service, container, shell string) {
	for i := 0; i < len(h.historyEntryArgs)-1; i += 2 {
		key := h.historyEntryArgs[i]
		val := h.historyEntryArgs[i+1]

		switch key {
		case "--profile":
			profile = val
		case "--cluster":
			cluster = val
		case "--service":
			service = val
		case "--container":
			container = val
		case "--shell":
			shell = val
		}
	}

	if shell == "" {
		shell = "su -s /bin/sh www-data"
	}

	return
}

// HistoryEntry mirrors the history.Entry structure for conversion.
type HistoryEntry struct {
	Command   string    `json:"command"`
	Label     string    `json:"label"`
	Args      []string  `json:"args"`
	Timestamp time.Time `json:"timestamp"`
}

// ConnectionFromHistoryEntry reconstructs a Connection from a history Entry.
// This enables replaying old connections with the new domain model.
func ConnectionFromHistoryEntry(entry *HistoryEntry) (*domain.Connection, error) {
	helper := NewIntegrationHelper(entry.Command, entry.Label, entry.Args)
	profile, clusterName, serviceName, containerName, shell := helper.ParseConnectionArgs()

	// Validate and construct domain objects
	cluster, err := domain.NewCluster(clusterName)
	if err != nil {
		return nil, err
	}

	service, err := domain.NewService(serviceName)
	if err != nil {
		return nil, err
	}

	container, err := domain.NewContainer(containerName)
	if err != nil {
		return nil, err
	}

	// Reconstruct task (minimal, just for the container)
	task := domain.NewTask(
		uuid.New().String(),
		[]domain.Container{container},
		domain.TaskStatusRunning,
	)

	// Create connection
	conn, err := domain.NewConnection(
		fmt.Sprintf("conn-%s-%d", profile, entry.Timestamp.Unix()),
		cluster,
		service,
		task,
		container,
		shell,
	)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}
