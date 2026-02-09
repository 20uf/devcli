package ecs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Client struct {
	ecs     *ecs.Client
	profile string
	region  string
}

func NewClient(profile, region string) (*Client, error) {
	var opts []func(*config.LoadOptions) error

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Client{
		ecs:     ecs.NewFromConfig(cfg),
		profile: profile,
		region:  region,
	}, nil
}

func (c *Client) ListClusters(ctx context.Context) ([]string, error) {
	var clusterArns []string
	paginator := ecs.NewListClustersPaginator(c.ecs, &ecs.ListClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		clusterArns = append(clusterArns, page.ClusterArns...)
	}

	names := make([]string, 0, len(clusterArns))
	for _, arn := range clusterArns {
		names = append(names, extractName(arn))
	}
	sort.Strings(names)

	return names, nil
}

func (c *Client) ListServices(ctx context.Context, cluster string) ([]string, error) {
	var serviceArns []string
	paginator := ecs.NewListServicesPaginator(c.ecs, &ecs.ListServicesInput{
		Cluster: aws.String(cluster),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		serviceArns = append(serviceArns, page.ServiceArns...)
	}

	names := make([]string, 0, len(serviceArns))
	for _, arn := range serviceArns {
		names = append(names, extractName(arn))
	}
	sort.Strings(names)

	return names, nil
}

func (c *Client) GetRunningTask(ctx context.Context, cluster, service string) (string, error) {
	resp, err := c.ecs.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(service),
		DesiredStatus: "RUNNING",
		MaxResults:    aws.Int32(1),
	})
	if err != nil {
		return "", err
	}

	if len(resp.TaskArns) == 0 {
		return "", fmt.Errorf("no running tasks for service %s", service)
	}

	return extractID(resp.TaskArns[0]), nil
}

func (c *Client) ListContainers(ctx context.Context, cluster, taskID string) ([]string, error) {
	resp, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   []string{taskID},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Tasks) == 0 {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	var names []string
	for _, container := range resp.Tasks[0].Containers {
		if container.Name != nil {
			names = append(names, *container.Name)
		}
	}
	sort.Strings(names)

	return names, nil
}

func (c *Client) ExecInteractive(ctx context.Context, cluster, taskID, container, command, profile string) error {
	args := []string{"ecs", "execute-command",
		"--cluster", cluster,
		"--task", taskID,
		"--container", container,
		"--command", command,
		"--interactive",
	}

	if profile != "" {
		args = append(args, "--profile", profile)
	}
	if c.region != "" {
		args = append(args, "--region", c.region)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// extractName returns the last segment after "/" in an ARN.
func extractName(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}

// extractID returns the task ID from a full task ARN.
func extractID(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}
