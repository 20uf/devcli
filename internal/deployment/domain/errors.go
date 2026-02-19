package domain

import "errors"

// Domain-specific errors for the Deployment bounded context.
var (
	ErrNoWorkflowFound        = errors.New("no workflow found")
	ErrWorkflowNotFound       = errors.New("workflow file not found")
	ErrNoRunFound             = errors.New("no deployment run found")
	ErrInvalidWorkflow        = errors.New("workflow name is required")
	ErrInvalidInput           = errors.New("invalid input value")
	ErrInputTypeMismatch      = errors.New("input type mismatch")
	ErrInputValidationFailed  = errors.New("input validation failed")
	ErrMissingRequiredInput   = errors.New("missing required input")
	ErrRunNotTracking         = errors.New("run is not being tracked")
)
