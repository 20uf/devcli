package domain

import "time"

// TaskStatus represents the lifecycle state of an ECS task.
type TaskStatus string

const (
	TaskStatusRunning     TaskStatus = "RUNNING"
	TaskStatusProvisioned TaskStatus = "PROVISIONED"
	TaskStatusPending     TaskStatus = "PENDING"
	TaskStatusActivating  TaskStatus = "ACTIVATING"
	TaskStatusStopping    TaskStatus = "STOPPING"
	TaskStatusDeprovisioned TaskStatus = "DEPROVISIONED"
	TaskStatusStopped     TaskStatus = "STOPPED"
)

// Task represents an ECS task instance (entity).
// A task has an identity (ID) and mutable state (containers, status).
type Task struct {
	id         string        // Unique identifier
	containers []Container   // Running containers in this task
	status     TaskStatus    // Current task status
	createdAt  time.Time     // When the task was created
}

// NewTask creates a new Task entity.
func NewTask(id string, containers []Container, status TaskStatus) Task {
	return Task{
		id:         id,
		containers: containers,
		status:     status,
		createdAt:  time.Now(),
	}
}

// ID returns the task's unique identifier.
func (t Task) ID() string {
	return t.id
}

// Containers returns the list of containers in this task.
func (t Task) Containers() []Container {
	return t.containers
}

// Status returns the task's current status.
func (t Task) Status() TaskStatus {
	return t.status
}

// CreatedAt returns when the task was created.
func (t Task) CreatedAt() time.Time {
	return t.createdAt
}

// IsRunning checks if the task is in RUNNING state.
func (t Task) IsRunning() bool {
	return t.status == TaskStatusRunning
}

// FindContainerByName finds a container by its name.
// Returns ErrNoContainerFound if not found.
func (t Task) FindContainerByName(name string) (Container, error) {
	for _, c := range t.containers {
		if c.Name() == name {
			return c, nil
		}
	}
	return Container{}, ErrNoContainerFound
}

// SelectContainer selects the best container from the task.
// Prefers containers with preferred names (php, app, web, api).
// Returns error if no containers are available.
func (t Task) SelectContainer() (Container, error) {
	if len(t.containers) == 0 {
		return Container{}, ErrNoContainerFound
	}

	// Try to find a preferred container
	for _, c := range t.containers {
		if c.IsPreferred() {
			return c, nil
		}
	}

	// Fall back to the first container
	return t.containers[0], nil
}
