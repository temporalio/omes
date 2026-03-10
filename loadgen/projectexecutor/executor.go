package projectexecutor

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/metrics"
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
	// ebb-and-flow project executor (sine-wave backlog targeting)
	EbbAndFlowProjectExecutor = "ebb-and-flow"
	// saturation project executor (fills slots while staying below CPU ceiling)
	SaturationProjectExecutor = "saturation"
)

func NewProjectTestExecutor(executorType string, c *ProjectHandle, prometheusAddress string) (loadgen.Executor, error) {
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
	case EbbAndFlowProjectExecutor:
		return &ebbAndFlowProjectExecutor{client: c}, nil
	case SaturationProjectExecutor:
		if prometheusAddress == "" {
			return nil, fmt.Errorf("prometheus address required for saturation executor")
		}
		return &loadgen.SaturationExecutor{
			Execute: func(ctx context.Context, run *loadgen.Run) error {
				_, err := c.Execute(ctx, &api.ExecuteRequest{
					Iteration: int64(run.Iteration),
				})
				return err
			},
			Sample: func(ctx context.Context) (loadgen.SaturationSample, error) {
				results, err := metrics.QueryInstant(ctx, metrics.PromInstantQueryConfig{
					Address: prometheusAddress,
					Queries: []metrics.PromQuery{
						{Name: "cpu", Query: fmt.Sprintf(`sum(process_cpu_percent{job="%s"})`, metrics.JobWorkerProcess)},
						{Name: "slots_used", Query: fmt.Sprintf(`sum(temporal_worker_task_slots_used{job="%s"})`, metrics.JobWorkerApp)},
						{Name: "slots_available", Query: fmt.Sprintf(`sum(temporal_worker_task_slots_available{job="%s"})`, metrics.JobWorkerApp)},
					},
				})
				if err != nil {
					return loadgen.SaturationSample{}, err
				}
				return loadgen.SaturationSample{
					CPUPercentRaw:  results["cpu"],
					SlotsUsed:      results["slots_used"],
					SlotsAvailable: results["slots_available"],
				}, nil
			},
			Config: loadgen.DefaultSaturationConfig(),
		}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported --executor %q (supported: %s, %s, %s)",
			executorType,
			SteadyRateProjectExecutor,
			EbbAndFlowProjectExecutor,
			SaturationProjectExecutor,
		)
	}
}

// ProjectHandle is a gRPC client for calling project service endpoints.
type ProjectHandle struct {
	host    string
	port    int
	address string

	conn   *grpc.ClientConn
	client api.ProjectServiceClient
}

func NewProjectHandle(ctx context.Context, port int, req *api.InitRequest) (ProjectHandle, error) {
	address := fmt.Sprintf("%s:%d", defaultClientHost, port)
	c := ProjectHandle{
		host:    defaultClientHost,
		port:    port,
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
