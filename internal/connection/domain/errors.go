package domain

import "errors"

// Domain-specific errors for the Connection bounded context.
var (
	ErrNoClusterFound    = errors.New("no ECS cluster found")
	ErrNoServiceFound    = errors.New("no service found in cluster")
	ErrNoTaskFound       = errors.New("no running task found")
	ErrNoContainerFound  = errors.New("no container found in task")
	ErrInvalidCluster    = errors.New("cluster name is required")
	ErrInvalidService    = errors.New("service name is required")
	ErrInvalidContainer  = errors.New("container name is required")
)
