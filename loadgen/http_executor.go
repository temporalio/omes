package loadgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/temporalio/omes/internal/utils"
)

// HTTPExecutor runs load tests via HTTP calls to user-supplied starters.
type HTTPExecutor struct {
	Client *ClientHandle
}

// Run implements the Executor interface, delegating to GenericExecutor.
func (e *HTTPExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	ge := &GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			return e.Client.Execute(ctx, run.Iteration, info.RunID)
		},
	}
	return ge.Run(ctx, info)
}

// ExecuteRequest is sent to client starter on /execute.
type ExecuteRequest struct {
	Iteration int    `json:"iteration"`
	RunID     string `json:"run_id"`
}

// ExecuteResponse is returned from /execute endpoint.
type ExecuteResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handlerBase provides common HTTP functionality for starters.
type handlerBase struct {
	URL        string
	httpClient *http.Client
}

func newHandlerBase(url string) handlerBase {
	return handlerBase{
		URL:        url,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// WaitForReady polls the /info endpoint until it returns 200 or the context is cancelled.
func (b *handlerBase) WaitForReady(ctx context.Context) error {
	return utils.WaitForReady(ctx, b.URL+"/info", 10*time.Second)
}

func (b *handlerBase) doRequest(ctx context.Context, method, path string, reqBody, respBody any) error {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if respBody != nil {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

func (b *handlerBase) doPost(ctx context.Context, path string, reqBody, respBody any) error {
	return b.doRequest(ctx, http.MethodPost, path, reqBody, respBody)
}

// ClientHandle is an HTTP client for calling client starter endpoints.
type ClientHandle struct {
	handlerBase
}

// NewClientHandle creates a new ClientHandle.
func NewClientHandle(url string) *ClientHandle {
	return &ClientHandle{handlerBase: newHandlerBase(url)}
}

// Execute calls /execute for a single iteration.
func (c *ClientHandle) Execute(ctx context.Context, iteration int, runID string) error {
	req := ExecuteRequest{Iteration: iteration, RunID: runID}
	var resp ExecuteResponse
	if err := c.doPost(ctx, "/execute", req, &resp); err != nil {
		return fmt.Errorf("execute request failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("execute failed: %s", resp.Error)
	}
	return nil
}
