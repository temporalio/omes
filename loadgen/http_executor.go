package loadgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HTTPExecutor runs load tests via HTTP calls to user-supplied starters.
type HTTPExecutor struct {
	Client *ClientStarter
	Worker *WorkerStarter
	Logger *zap.SugaredLogger
}

// Run implements the Executor interface, delegating to GenericExecutor.
func (e *HTTPExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	ge := &GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			return e.Client.Execute(ctx, run.Iteration)
		},
	}
	return ge.Run(ctx, info)
}

// InitRequest is sent to both client and worker starters on /init.
type InitRequest struct {
	RunID         string `json:"run_id"`
	TaskQueue     string `json:"task_queue"`
	ServerAddress string `json:"server_address"`
	Namespace     string `json:"namespace"`
}

// WorkerInitRequest extends InitRequest with worker-specific fields.
type WorkerInitRequest struct {
	InitRequest
	PromListenAddress string `json:"prom_listen_address,omitempty"`
}

// InitResponse is returned from /init endpoint.
type InitResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// WorkerInitResponse extends InitResponse with worker-specific fields.
type WorkerInitResponse struct {
	InitResponse
	WorkerPID int `json:"worker_pid,omitempty"`
}

// ExecuteRequest is sent to client starter on /execute.
type ExecuteRequest struct {
	Iteration int `json:"iteration"`
}

// ExecuteResponse is returned from /execute endpoint.
type ExecuteResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Traceback string `json:"traceback,omitempty"`
}

// ShutdownRequest is sent to starters on /shutdown.
type ShutdownRequest struct {
	DrainTimeoutMs int `json:"drain_timeout_ms,omitempty"`
}

// InfoResponse is returned from /info endpoint.
type InfoResponse struct {
	SDKLanguage    string `json:"sdk_language"`
	SDKVersion     string `json:"sdk_version"`
	StarterVersion string `json:"starter_version"`
	WorkerPID      int    `json:"worker_pid,omitempty"`
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}
}

// Retry executes fn with exponential backoff.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// starterBase provides common HTTP functionality for starters.
type starterBase struct {
	URL        string
	httpClient *http.Client
	logger     *zap.SugaredLogger
}

func newStarterBase(url string, logger *zap.SugaredLogger) starterBase {
	return starterBase{
		URL:        url,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		logger:     logger,
	}
}

func (b *starterBase) doRequest(ctx context.Context, method, path string, reqBody, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, b.URL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if respBody != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

func (b *starterBase) doPost(ctx context.Context, path string, reqBody, respBody any) error {
	return b.doRequest(ctx, http.MethodPost, path, reqBody, respBody)
}

func (b *starterBase) doGet(ctx context.Context, path string, respBody any) error {
	return b.doRequest(ctx, http.MethodGet, path, nil, respBody)
}

// ClientStarter is an HTTP client for calling client starter endpoints.
type ClientStarter struct {
	starterBase
}

// NewClientStarter creates a new ClientStarter.
func NewClientStarter(url string, logger *zap.SugaredLogger) *ClientStarter {
	return &ClientStarter{starterBase: newStarterBase(url, logger)}
}

// Init initializes the client starter.
func (c *ClientStarter) Init(ctx context.Context, req InitRequest) (*InitResponse, error) {
	var resp InitResponse
	if err := c.doPost(ctx, "/init", req, &resp); err != nil {
		return nil, fmt.Errorf("client init failed: %w", err)
	}
	if !resp.Success {
		return &resp, fmt.Errorf("client init returned error: %s", resp.Error)
	}
	return &resp, nil
}

// Execute calls /execute for a single iteration.
func (c *ClientStarter) Execute(ctx context.Context, iteration int) error {
	req := ExecuteRequest{Iteration: iteration}
	var resp ExecuteResponse
	if err := c.doPost(ctx, "/execute", req, &resp); err != nil {
		return fmt.Errorf("execute request failed: %w", err)
	}
	if !resp.Success {
		if resp.Traceback != "" {
			return fmt.Errorf("execute failed: %s\n%s", resp.Error, resp.Traceback)
		}
		return fmt.Errorf("execute failed: %s", resp.Error)
	}
	return nil
}

// Shutdown sends a shutdown request (best-effort).
func (c *ClientStarter) Shutdown(ctx context.Context) {
	req := ShutdownRequest{DrainTimeoutMs: 30000}
	var resp struct{ Status string }
	if err := c.doPost(ctx, "/shutdown", req, &resp); err != nil && c.logger != nil {
		c.logger.Warnf("Client shutdown request failed: %v", err)
	}
}

// Info retrieves SDK metadata.
func (c *ClientStarter) Info(ctx context.Context) (*InfoResponse, error) {
	var resp InfoResponse
	if err := c.doGet(ctx, "/info", &resp); err != nil {
		return nil, fmt.Errorf("info request failed: %w", err)
	}
	return &resp, nil
}

// WorkerStarter is an HTTP client for calling worker starter endpoints.
type WorkerStarter struct {
	starterBase
	WorkerPID int
}

// NewWorkerStarter creates a new WorkerStarter.
func NewWorkerStarter(url string, logger *zap.SugaredLogger) *WorkerStarter {
	return &WorkerStarter{starterBase: newStarterBase(url, logger)}
}

// Init initializes the worker starter.
func (w *WorkerStarter) Init(ctx context.Context, req WorkerInitRequest) (*WorkerInitResponse, error) {
	var resp WorkerInitResponse
	if err := w.doPost(ctx, "/init", req, &resp); err != nil {
		return nil, fmt.Errorf("worker init failed: %w", err)
	}
	if !resp.Success {
		return &resp, fmt.Errorf("worker init returned error: %s", resp.Error)
	}
	w.WorkerPID = resp.WorkerPID
	return &resp, nil
}

// Shutdown sends a shutdown request (best-effort).
func (w *WorkerStarter) Shutdown(ctx context.Context) {
	req := ShutdownRequest{DrainTimeoutMs: 30000}
	var resp struct{ Status string }
	if err := w.doPost(ctx, "/shutdown", req, &resp); err != nil && w.logger != nil {
		w.logger.Warnf("Worker shutdown request failed: %v", err)
	}
}

// Info retrieves SDK metadata.
func (w *WorkerStarter) Info(ctx context.Context) (*InfoResponse, error) {
	var resp InfoResponse
	if err := w.doGet(ctx, "/info", &resp); err != nil {
		return nil, fmt.Errorf("info request failed: %w", err)
	}
	return &resp, nil
}
