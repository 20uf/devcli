package domain

import "time"

// RunStatus represents the lifecycle state of a deployment run.
type RunStatus string

const (
	RunStatusQueued      RunStatus = "queued"
	RunStatusInProgress  RunStatus = "in_progress"
	RunStatusCompleted   RunStatus = "completed"
	RunStatusUnknown     RunStatus = "unknown"
)

// RunConclusion represents the final outcome of a completed run.
type RunConclusion string

const (
	RunConclusionSuccess   RunConclusion = "success"
	RunConclusionFailure   RunConclusion = "failure"
	RunConclusionCancelled RunConclusion = "cancelled"
	RunConclusionNeutral   RunConclusion = "neutral"
	RunConclusionSkipped   RunConclusion = "skipped"
	RunConclusionUnknown   RunConclusion = "unknown"
)

// Run represents a GitHub Actions workflow run (entity).
// A run has an identity (ID) and mutable state (status, conclusion, timestamps).
type Run struct {
	id          string         // Unique run ID from GitHub
	number      int            // Run number (e.g., #123)
	status      RunStatus      // Current status
	conclusion  RunConclusion  // Final outcome (if completed)
	branch      string         // Branch the workflow ran on
	createdAt   time.Time      // When the run was created
	startedAt   *time.Time     // When execution started
	completedAt *time.Time     // When execution completed
	url         string         // GitHub URL to the run
}

// NewRun creates a new Run entity.
func NewRun(id string, number int, status RunStatus, branch string, url string) Run {
	return Run{
		id:        id,
		number:    number,
		status:    status,
		branch:    branch,
		url:       url,
		createdAt: time.Now(),
	}
}

// ID returns the run's unique identifier.
func (r Run) ID() string {
	return r.id
}

// Number returns the run number.
func (r Run) Number() int {
	return r.number
}

// Status returns the current status.
func (r Run) Status() RunStatus {
	return r.status
}

// Conclusion returns the final conclusion (if completed).
func (r Run) Conclusion() RunConclusion {
	return r.conclusion
}

// Branch returns the branch the workflow ran on.
func (r Run) Branch() string {
	return r.branch
}

// URL returns the GitHub URL to the run.
func (r Run) URL() string {
	return r.url
}

// CreatedAt returns when the run was created.
func (r Run) CreatedAt() time.Time {
	return r.createdAt
}

// StartedAt returns when execution started (nil if not started).
func (r Run) StartedAt() *time.Time {
	return r.startedAt
}

// CompletedAt returns when execution completed (nil if not completed).
func (r Run) CompletedAt() *time.Time {
	return r.completedAt
}

// IsQueued checks if the run is waiting to execute.
func (r Run) IsQueued() bool {
	return r.status == RunStatusQueued
}

// IsInProgress checks if the run is currently executing.
func (r Run) IsInProgress() bool {
	return r.status == RunStatusInProgress
}

// IsCompleted checks if the run has finished.
func (r Run) IsCompleted() bool {
	return r.status == RunStatusCompleted
}

// IsFailed checks if the run failed.
func (r Run) IsFailed() bool {
	return r.conclusion == RunConclusionFailure
}

// IsSuccess checks if the run succeeded.
func (r Run) IsSuccess() bool {
	return r.conclusion == RunConclusionSuccess
}

// UpdateStatus updates the run's status.
// This is a mutable operation on the entity.
func (r *Run) UpdateStatus(status RunStatus) {
	r.status = status
	if status == RunStatusInProgress && r.startedAt == nil {
		now := time.Now()
		r.startedAt = &now
	}
}

// UpdateConclusion updates the run's conclusion.
func (r *Run) UpdateConclusion(conclusion RunConclusion) {
	r.conclusion = conclusion
	if r.status != RunStatusCompleted {
		r.status = RunStatusCompleted
	}
	if r.completedAt == nil {
		now := time.Now()
		r.completedAt = &now
	}
}

// Duration returns how long the run took (or has been running).
func (r Run) Duration() time.Duration {
	if r.startedAt == nil {
		return 0
	}

	end := r.completedAt
	if end == nil {
		now := time.Now()
		end = &now
	}

	return end.Sub(*r.startedAt)
}

// IsStale checks if the run is older than the given duration.
func (r Run) IsStale(maxAge time.Duration) bool {
	return time.Since(r.createdAt) > maxAge
}

// String returns a human-readable representation.
func (r Run) String() string {
	return "#" + string(rune(r.number))
}
