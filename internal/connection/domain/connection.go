package domain

import (
	"errors"
	"time"
)

// Connection represents an intended connection to an ECS container (aggregate root).
// It encapsulates all information needed to connect to a container and execute a shell command.
// This is the entry point for the connection domain logic.
type Connection struct {
	id            string    // Unique identifier (e.g., UUID)
	cluster       Cluster   // Target cluster
	service       Service   // Target service
	task          Task      // Target task
	container     Container // Target container
	shellCommand  string    // Shell command to execute (e.g., "su -s /bin/sh www-data")
	createdAt     time.Time // When this connection was planned
	initiatedAt   *time.Time // When execution started (nil until executed)
}

// NewConnection creates a new Connection aggregate.
// Validates that all required fields are set.
func NewConnection(
	id string,
	cluster Cluster,
	service Service,
	task Task,
	container Container,
	shellCommand string,
) (Connection, error) {
	if id == "" {
		return Connection{}, errors.New("connection id is required")
	}
	if shellCommand == "" {
		return Connection{}, errors.New("shell command is required")
	}

	// Verify container exists in task
	if _, err := task.FindContainerByName(container.Name()); err != nil {
		return Connection{}, err
	}

	return Connection{
		id:           id,
		cluster:      cluster,
		service:      service,
		task:         task,
		container:    container,
		shellCommand: shellCommand,
		createdAt:    time.Now(),
	}, nil
}

// ID returns the connection identifier.
func (c Connection) ID() string {
	return c.id
}

// Cluster returns the target cluster.
func (c Connection) Cluster() Cluster {
	return c.cluster
}

// Service returns the target service.
func (c Connection) Service() Service {
	return c.service
}

// Task returns the target task.
func (c Connection) Task() Task {
	return c.task
}

// Container returns the target container.
func (c Connection) Container() Container {
	return c.container
}

// ShellCommand returns the shell command to execute.
func (c Connection) ShellCommand() string {
	return c.shellCommand
}

// CreatedAt returns when the connection was created.
func (c Connection) CreatedAt() time.Time {
	return c.createdAt
}

// IsInitiated checks if the connection has been executed.
func (c Connection) IsInitiated() bool {
	return c.initiatedAt != nil
}

// InitiatedAt returns when the connection was executed (or nil if not yet).
func (c Connection) InitiatedAt() *time.Time {
	return c.initiatedAt
}

// Initiate marks the connection as initiated (started execution).
func (c *Connection) Initiate() {
	now := time.Now()
	c.initiatedAt = &now
}

// String returns a human-readable representation of the connection.
// Format: cluster/service/container
func (c Connection) String() string {
	return c.cluster.Name() + "/" + c.service.Name() + "/" + c.container.Name()
}

// Label returns a displayable label for the connection with details.
// Format: "profile → cluster/service/container"
func (c Connection) Label(profile string) string {
	if profile != "" {
		return profile + " → " + c.String()
	}
	return c.String()
}
