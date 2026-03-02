package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Container struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
}

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	// If DOCKER_HOST is not set, try common Docker Desktop socket paths
	if os.Getenv("DOCKER_HOST") == "" {
		socketPaths := []string{
			"/var/run/docker.sock",
			os.Getenv("HOME") + "/.docker/desktop/docker.sock",
		}
		for _, path := range socketPaths {
			if _, err := os.Stat(path); err == nil {
				opts = append(opts, client.WithHost("unix://"+path))
				break
			}
		}
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

// ListContainers returns all containers (running and stopped)
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]Container, 0, len(containers))
	for _, cont := range containers {
		name := ""
		if len(cont.Names) > 0 {
			name = cont.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		result = append(result, Container{
			ID:     cont.ID[:12],
			Name:   name,
			Image:  cont.Image,
			Status: cont.Status,
			State:  cont.State,
		})
	}

	return result, nil
}

// StartContainer starts a container by ID
func (c *Client) StartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", id, err)
	}
	return nil
}

// StopContainer stops a container by ID
func (c *Client) StopContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", id, err)
	}
	return nil
}
