package project

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/temporalio/omes/loadgen"
	api "github.com/temporalio/omes/workers/go/projects/harness/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultClientHost               = "localhost"
	defaultClientReadyTimeout       = 3 * time.Second
	defaultClientReadyCheckInterval = 100 * time.Millisecond
)

// ProjectHandle is a gRPC client for calling project service endpoints.
type ProjectHandle struct {
	address   string
	taskQueue string
	conn      *grpc.ClientConn
	client    api.ProjectServiceClient
}

func NewProjectHandle(ctx context.Context, port int, req *api.InitRequest) (ProjectHandle, error) {
	address := fmt.Sprintf("%s:%d", defaultClientHost, port)
	c := ProjectHandle{address: address, taskQueue: req.GetTaskQueue()}

	deadline := time.Now().Add(defaultClientReadyTimeout)
	var err error
	for time.Now().Before(deadline) {
		conn, dialErr := net.Dial("tcp", c.address)
		if dialErr == nil {
			conn.Close()
			err = nil
			break
		}
		err = dialErr
		select {
		case <-ctx.Done():
			return ProjectHandle{}, ctx.Err()
		case <-time.After(defaultClientReadyCheckInterval):
		}
	}
	if err != nil {
		return ProjectHandle{}, fmt.Errorf("project server not ready after %v: %w", defaultClientReadyTimeout, err)
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

func (c *ProjectHandle) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.client = nil
	return err
}

func (c *ProjectHandle) init(ctx context.Context, req *api.InitRequest) error {
	_, err := c.client.Init(ctx, req)
	if err != nil {
		return fmt.Errorf("init request failed: %w", err)
	}
	return nil
}

// Execute calls ProjectService.Execute for a single iteration.
func (c *ProjectHandle) Execute(ctx context.Context, req *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if req.GetTaskQueue() == "" {
		req = &api.ExecuteRequest{
			Iteration: req.GetIteration(),
			TaskQueue: c.taskQueue,
			Payload:   req.GetPayload(),
		}
	}
	resp, err := c.client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	return resp, nil
}

// NewSteadyRateExecutor creates an executor that calls Execute once per iteration.
func NewSteadyRateExecutor(c *ProjectHandle) loadgen.Executor {
	return &loadgen.GenericExecutor{
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			_, err := c.Execute(ctx, &api.ExecuteRequest{
				Iteration: int64(run.Iteration),
				TaskQueue: c.taskQueue,
			})
			return err
		},
	}
}
