package projectexecutor

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/projecttests/go/harness/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultClientHost = "localhost"
	// Default timeout waiting for project service to be ready.
	defaultClientReadyTimeout = 3 * time.Second
	// Interval between readiness checks.
	defaultClientReadyCheckInterval = 100 * time.Millisecond

	// steady-rate project executor (wrapper over generic executor)
	SteadyRateProjectExecutor = "steady-rate"
)

func NewProjectTestExecutor(executorType string, c *ProjectHandle) (loadgen.Executor, error) {
	switch executorType {
	case SteadyRateProjectExecutor:
		ge := &loadgen.GenericExecutor{
			Execute: func(ctx context.Context, run *loadgen.Run) error {
				_, err := c.Execute(ctx, &api.ExecuteRequest{
					Iteration: int64(run.Iteration),
				})
				return err
			},
		}
		return ge, nil
	default:
		return nil, fmt.Errorf("unsupported --executor %q (supported: %s)", executorType, SteadyRateProjectExecutor)
	}
}

// ProjectHandle is a gRPC client for calling project service endpoints.
type ProjectHandle struct {
	address string
	conn    *grpc.ClientConn
	client  api.ProjectServiceClient
}

func NewProjectHandle(ctx context.Context, port int, req *api.InitRequest) (ProjectHandle, error) {
	address := fmt.Sprintf("%s:%d", defaultClientHost, port)
	c := ProjectHandle{
		address: address,
	}

	readyCheck := func() error {
		conn, err := net.Dial("tcp", c.address)
		if err == nil {
			return conn.Close()
		}
		return err
	}
	err := utils.WaitUntil(ctx, readyCheck, defaultClientReadyTimeout, defaultClientReadyCheckInterval)
	if err != nil {
		return ProjectHandle{}, err
	}

	conn, err := grpc.NewClient(
		c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return ProjectHandle{}, fmt.Errorf("failed to connect project service %s: %w", c.address, err)
	}

	c.conn = conn
	c.client = api.NewProjectServiceClient(conn)

	if err := c.init(ctx, req); err != nil {
		return ProjectHandle{}, fmt.Errorf("project init failed: %w", err)
	}
	return c, nil
}

// Close closes the underlying gRPC connection if one was created.
func (c *ProjectHandle) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.client = nil
	return err
}

// init calls ProjectService.Init once to provide run-scoped context.
func (c *ProjectHandle) init(ctx context.Context, req *api.InitRequest) error {
	if req == nil {
		return fmt.Errorf("init request is required")
	}
	if c.client == nil {
		return fmt.Errorf("project client not connected; call Connect first")
	}
	_, err := c.client.Init(ctx, req)
	if err != nil {
		return fmt.Errorf("init request failed: %w", err)
	}
	return nil
}

// Execute calls ProjectService.Execute for a single iteration.
func (c *ProjectHandle) Execute(ctx context.Context, req *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("execute request is required")
	}
	if c.client == nil {
		return nil, fmt.Errorf("project client not connected; call Connect first")
	}
	resp, err := c.client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	return resp, nil
}
