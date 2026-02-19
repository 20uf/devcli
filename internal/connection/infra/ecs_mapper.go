package infra

import (
	"strings"

	"github.com/20uf/devcli/internal/connection/domain"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ECSMapper translates between AWS ECS API objects and domain entities.
// This is the anti-corruption layer that shields the domain from AWS SDK changes.
type ECSMapper struct{}

// NewECSMapper creates a new mapper instance.
func NewECSMapper() *ECSMapper {
	return &ECSMapper{}
}

// MapClusterARNToCluster extracts a cluster name from an ARN and returns a domain Cluster.
// ARN format: arn:aws:ecs:region:account-id:cluster/cluster-name
func (m *ECSMapper) MapClusterARNToCluster(arn string) (domain.Cluster, error) {
	name := extractNameFromARN(arn)
	return domain.NewCluster(name)
}

// MapServiceARNToService extracts a service name from an ARN and returns a domain Service.
// ARN format: arn:aws:ecs:region:account-id:service/cluster-name/service-name
func (m *ECSMapper) MapServiceARNToService(arn string) (domain.Service, error) {
	name := extractNameFromARN(arn)
	return domain.NewService(name)
}

// MapECSTaskToTask converts an AWS ECS Task to a domain Task entity.
// Extracts the task ID from the ARN and maps all containers.
func (m *ECSMapper) MapECSTaskToTask(ecsTask *types.Task) (domain.Task, error) {
	// Extract task ID from ARN
	taskID := extractNameFromARN(*ecsTask.TaskArn)

	// Map containers
	var containers []domain.Container
	if ecsTask.Containers != nil {
		for _, c := range ecsTask.Containers {
			if c.Name != nil && *c.Name != "" {
				container, err := domain.NewContainer(*c.Name)
				if err != nil {
					return domain.Task{}, err
				}
				containers = append(containers, container)
			}
		}
	}

	// Map task status
	status := domain.TaskStatusRunning // Default
	if ecsTask.LastStatus != nil {
		status = domain.TaskStatus(*ecsTask.LastStatus)
	}

	return domain.NewTask(taskID, containers, status), nil
}

// extractNameFromARN extracts the last segment from an ARN.
// Example: "arn:aws:ecs:region:account:resource/name" → "name"
func extractNameFromARN(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}
