package domain

import "time"

// TrackedDeployment represents a deployment that is being tracked in the dashboard.
// It's an Entity with identity and mutable state.
type TrackedDeployment struct {
	id          string
	runID       string
	workflow    Workflow
	branch      string
	status      RunStatus
	conclusion  RunConclusion
	startedAt   time.Time
	completedAt *time.Time
	repo        string
}

// NewTrackedDeployment creates a new tracked deployment.
func NewTrackedDeployment(runID string, workflow Workflow, branch string, repo string) TrackedDeployment {
	return TrackedDeployment{
		id:        runID, // Use run ID as identity
		runID:     runID,
		workflow:  workflow,
		branch:    branch,
		status:    RunStatusQueued,
		startedAt: time.Now(),
		repo:      repo,
	}
}

// ID returns the unique identifier.
func (td TrackedDeployment) ID() string {
	return td.id
}

// RunID returns the GitHub run ID.
func (td TrackedDeployment) RunID() string {
	return td.runID
}

// Workflow returns the workflow.
func (td TrackedDeployment) Workflow() Workflow {
	return td.workflow
}

// Branch returns the branch name.
func (td TrackedDeployment) Branch() string {
	return td.branch
}

// Status returns the current status.
func (td TrackedDeployment) Status() RunStatus {
	return td.status
}

// Conclusion returns the final conclusion (if completed).
func (td TrackedDeployment) Conclusion() RunConclusion {
	return td.conclusion
}

// StartedAt returns when tracking started.
func (td TrackedDeployment) StartedAt() time.Time {
	return td.startedAt
}

// CompletedAt returns when deployment completed (if completed).
func (td TrackedDeployment) CompletedAt() *time.Time {
	return td.completedAt
}

// Repo returns the GitHub repository URL.
func (td TrackedDeployment) Repo() string {
	return td.repo
}

// UpdateStatus updates the current status.
func (td *TrackedDeployment) UpdateStatus(status RunStatus) {
	td.status = status
}

// UpdateConclusion updates the conclusion and marks as completed.
func (td *TrackedDeployment) UpdateConclusion(conclusion RunConclusion) {
	td.conclusion = conclusion
	td.status = RunStatusCompleted
	now := time.Now()
	td.completedAt = &now
}

// IsActive checks if the deployment is still in progress.
func (td TrackedDeployment) IsActive() bool {
	return td.status == RunStatusInProgress || td.status == RunStatusQueued
}

// IsCompleted checks if the deployment has finished.
func (td TrackedDeployment) IsCompleted() bool {
	return td.status == RunStatusCompleted
}

// IsSuccess checks if the deployment succeeded.
func (td TrackedDeployment) IsSuccess() bool {
	return td.conclusion == RunConclusionSuccess
}

// IsFailed checks if the deployment failed.
func (td TrackedDeployment) IsFailed() bool {
	return td.conclusion == RunConclusionFailure
}

// IsCancelled checks if the deployment was cancelled.
func (td TrackedDeployment) IsCancelled() bool {
	return td.conclusion == RunConclusionCancelled
}

// Age returns how long the deployment has been tracked.
func (td TrackedDeployment) Age() time.Duration {
	if td.completedAt != nil {
		return td.completedAt.Sub(td.startedAt)
	}
	return time.Since(td.startedAt)
}

// ElapsedTime returns time elapsed since start (or duration if completed).
func (td TrackedDeployment) ElapsedTime() time.Duration {
	if td.completedAt != nil {
		return td.completedAt.Sub(td.startedAt)
	}
	return time.Since(td.startedAt)
}

// IsStale checks if the deployment is older than the given TTL.
// Useful for cleanup: deployments older than 7 days can be removed.
func (td TrackedDeployment) IsStale(maxAge time.Duration) bool {
	if td.completedAt != nil {
		return time.Since(*td.completedAt) > maxAge
	}
	return time.Since(td.startedAt) > maxAge
}

// String returns a human-readable representation.
func (td TrackedDeployment) String() string {
	return td.workflow.Name() + " on " + td.branch
}
