package infra

import (
	"context"
	"fmt"
	"sort"

	"github.com/20uf/devcli/internal/connection/domain"
	"github.com/20uf/devcli/internal/verbose"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ECSClusterRepository implements domain.ClusterRepository using AWS ECS SDK.
type ECSClusterRepository struct {
	client *ecs.Client
	mapper *ECSMapper
}

// NewECSClusterRepository creates a new ECS cluster repository.
func NewECSClusterRepository(client *ecs.Client) *ECSClusterRepository {
	return &ECSClusterRepository{
		client: client,
		mapper: NewECSMapper(),
	}
}

// ListClusters fetches all ECS clusters from AWS and maps them to domain Clusters.
func (r *ECSClusterRepository) ListClusters(ctx context.Context) ([]domain.Cluster, error) {
	verbose.Log("ecs:ListClusters")

	var clusterArns []string
	paginator := ecs.NewListClustersPaginator(r.client, &ecs.ListClustersInput{})

	// Fetch all cluster ARNs
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}
		clusterArns = append(clusterArns, page.ClusterArns...)
	}

	if len(clusterArns) == 0 {
		return nil, domain.ErrNoClusterFound
	}

	// Map ARNs to domain Clusters
	var clusters []domain.Cluster
	for _, arn := range clusterArns {
		cluster, err := r.mapper.MapClusterARNToCluster(arn)
		if err != nil {
			continue // Skip invalid clusters
		}
		clusters = append(clusters, cluster)
	}

	// Sort by name
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name() < clusters[j].Name()
	})

	if len(clusters) == 0 {
		return nil, domain.ErrNoClusterFound
	}

	return clusters, nil
}

// ECSServiceRepository implements domain.ServiceRepository using AWS ECS SDK.
type ECSServiceRepository struct {
	client *ecs.Client
	mapper *ECSMapper
}

// NewECSServiceRepository creates a new ECS service repository.
func NewECSServiceRepository(client *ecs.Client) *ECSServiceRepository {
	return &ECSServiceRepository{
		client: client,
		mapper: NewECSMapper(),
	}
}

// ListServices fetches all services in a cluster from AWS and maps them to domain Services.
func (r *ECSServiceRepository) ListServices(ctx context.Context, cluster domain.Cluster) ([]domain.Service, error) {
	verbose.Log("ecs:ListServices cluster=%s", cluster.Name())

	var serviceArns []string
	paginator := ecs.NewListServicesPaginator(r.client, &ecs.ListServicesInput{
		Cluster: aws.String(cluster.Name()),
	})

	// Fetch all service ARNs
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}
		serviceArns = append(serviceArns, page.ServiceArns...)
	}

	if len(serviceArns) == 0 {
		return nil, domain.ErrNoServiceFound
	}

	// Map ARNs to domain Services
	var services []domain.Service
	for _, arn := range serviceArns {
		service, err := r.mapper.MapServiceARNToService(arn)
		if err != nil {
			continue // Skip invalid services
		}
		services = append(services, service)
	}

	// Sort by name
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name() < services[j].Name()
	})

	if len(services) == 0 {
		return nil, domain.ErrNoServiceFound
	}

	return services, nil
}

// ECSTaskRepository implements domain.TaskRepository using AWS ECS SDK.
type ECSTaskRepository struct {
	client *ecs.Client
	mapper *ECSMapper
}

// NewECSTaskRepository creates a new ECS task repository.
func NewECSTaskRepository(client *ecs.Client) *ECSTaskRepository {
	return &ECSTaskRepository{
		client: client,
		mapper: NewECSMapper(),
	}
}

// GetRunningTask fetches the first running task for a service from AWS and maps it to a domain Task.
func (r *ECSTaskRepository) GetRunningTask(ctx context.Context, cluster domain.Cluster, service domain.Service) (domain.Task, error) {
	verbose.Log("ecs:ListTasks cluster=%s service=%s status=RUNNING", cluster.Name(), service.Name())

	resp, err := r.client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:       aws.String(cluster.Name()),
		ServiceName:   aws.String(service.Name()),
		DesiredStatus: types.DesiredStatusRunning,
		MaxResults:    aws.Int32(1),
	})
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(resp.TaskArns) == 0 {
		return domain.Task{}, domain.ErrNoTaskFound
	}

	// Describe the task to get container information
	verbose.Log("ecs:DescribeTasks cluster=%s task=%s", cluster.Name(), resp.TaskArns[0])
	describeResp, err := r.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster.Name()),
		Tasks:   resp.TaskArns,
	})
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to describe tasks: %w", err)
	}

	if len(describeResp.Tasks) == 0 {
		return domain.Task{}, domain.ErrNoTaskFound
	}

	// Map the ECS task to domain Task
	return r.mapper.MapECSTaskToTask(&describeResp.Tasks[0])
}
