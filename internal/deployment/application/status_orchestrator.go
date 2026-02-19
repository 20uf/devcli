package application

import (
	"context"
	"fmt"

	"github.com/20uf/devcli/internal/deployment/domain"
	"github.com/20uf/devcli/internal/deployment/infra"
)

// StatusOrchestrator is the application service for managing deployment tracking.
type StatusOrchestrator struct {
	tracker infra.TrackerRepository
	runs    domain.RunRepository
}

// NewStatusOrchestrator creates a new status orchestrator.
func NewStatusOrchestrator(tracker infra.TrackerRepository, runs domain.RunRepository) *StatusOrchestrator {
	return &StatusOrchestrator{
		tracker: tracker,
		runs:    runs,
	}
}

// ListTracked retrieves all tracked deployments with updated statuses from GitHub.
func (s *StatusOrchestrator) ListTracked(ctx context.Context) ([]domain.TrackedDeployment, error) {
	tracked, err := s.tracker.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracked deployments: %w", err)
	}

	// Refresh statuses from GitHub for active deployments
	for i, td := range tracked {
		if td.IsActive() {
			if run, err := s.runs.GetRun(ctx, td.RunID()); err == nil && run != nil {
				tracked[i].UpdateStatus(run.Status())
				if run.Conclusion() != "" {
					tracked[i].UpdateConclusion(run.Conclusion())
				}

				if err := s.tracker.Save(ctx, tracked[i]); err != nil {
					// Log but don't fail - we still want to show the run
					_ = err
				}
			}
		}
	}

	// Cleanup stale deployments (older than 7 days)
	_ = s.cleanupStale(ctx)

	return tracked, nil
}

// ListActive retrieves only active (in-progress or queued) deployments.
func (s *StatusOrchestrator) ListActive(ctx context.Context) ([]domain.TrackedDeployment, error) {
	return s.tracker.ListActive(ctx)
}

// TrackDeployment adds a new deployment to tracking.
func (s *StatusOrchestrator) TrackDeployment(ctx context.Context, runID string, workflow domain.Workflow, branch string, repo string) (domain.TrackedDeployment, error) {
	td := domain.NewTrackedDeployment(runID, workflow, branch, repo)

	if err := s.tracker.Save(ctx, td); err != nil {
		return domain.TrackedDeployment{}, fmt.Errorf("failed to track deployment: %w", err)
	}

	return td, nil
}

// GetTracked retrieves a specific tracked deployment.
func (s *StatusOrchestrator) GetTracked(ctx context.Context, id string) (*domain.TrackedDeployment, error) {
	td, err := s.tracker.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked deployment: %w", err)
	}

	if td != nil && td.IsActive() {
		// Refresh status from GitHub
		if run, err := s.runs.GetRun(ctx, td.RunID()); err == nil && run != nil {
			td.UpdateStatus(run.Status())
			if run.Conclusion() != "" {
				td.UpdateConclusion(run.Conclusion())
			}
			_ = s.tracker.Save(ctx, *td)
		}
	}

	return td, nil
}

// DismissTracked removes a deployment from tracking.
func (s *StatusOrchestrator) DismissTracked(ctx context.Context, id string) error {
	return s.tracker.Remove(ctx, id)
}

// GetRunLogs retrieves logs for a tracked deployment.
func (s *StatusOrchestrator) GetRunLogs(ctx context.Context, runID string) (string, error) {
	logs, err := s.runs.GetRunLogs(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs: %w", err)
	}
	return logs, nil
}

// cleanupStale removes deployments older than 7 days.
func (s *StatusOrchestrator) cleanupStale(ctx context.Context) error {
	const sevenDaysInSeconds = 7 * 24 * 60 * 60
	_, err := s.tracker.Cleanup(ctx, sevenDaysInSeconds)
	return err
}
