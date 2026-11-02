package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/temporalio/omes/workers/go/harness"
	"go.temporal.io/api/serviceerror"
	ws "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	sdkworker "go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type privateCaptureCase struct {
	Namespace        string
	TaskQueuePrefix  string
	WorkflowIDPrefix string
}

type captureKey struct {
	Namespace  string `json:"namespace"`
	TaskQueue  string `json:"taskQueue"`
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	ActivityID string `json:"activityId"`
	Attempt    int32  `json:"attempt"`
}
type captureEvent struct {
	Sequence            uint64     `json:"sequence"`
	UTC                 string     `json:"utc"`
	ElapsedNS           int64      `json:"elapsedNS"`
	Kind                string     `json:"kind"`
	Key                 captureKey `json:"key"`
	Path                string     `json:"path,omitempty"`
	WorkerIdentity      string     `json:"workerIdentity,omitempty"`
	Method              string     `json:"method,omitempty"`
	Code                string     `json:"code,omitempty"`
	ErrorType           string     `json:"errorType,omitempty"`
	ContextState        string     `json:"contextState,omitempty"`
	CauseType           string     `json:"causeType,omitempty"`
	CauseCode           string     `json:"causeCode,omitempty"`
	DurationUS          int64      `json:"durationUS,omitempty"`
	HeartbeatTimeout    string     `json:"heartbeatTimeout,omitempty"`
	StartToCloseTimeout string     `json:"startToCloseTimeout,omitempty"`
	MaximumAttempts     int32      `json:"maximumAttempts,omitempty"`
}
type captureStats struct {
	Accepted   uint64 `json:"accepted"`
	Dropped    uint64 `json:"dropped"`
	Unmapped   uint64 `json:"unmapped"`
	Rejected   uint64 `json:"rejected"`
	PeakTokens int    `json:"peakTokens"`
	PeakKeys   int    `json:"peakKeys"`
	Closed     bool   `json:"closed"`
	Incomplete bool   `json:"incomplete"`
}
type privateCapture struct {
	interceptor.WorkerInterceptorBase
	mu                 sync.Mutex
	scope              privateCaptureCase
	queue              string
	logger             *zap.SugaredLogger
	done               chan struct{}
	stopTimer          *time.Timer
	tokens             map[string]captureKey
	keys               map[captureKey]bool
	events             chan captureEvent
	start              time.Time
	maxKeys, maxTokens int
	maxEvents          uint64
	stats              captureStats
	sequence           uint64
}

func newPrivateCapture(scope privateCaptureCase, maxKeys, maxTokens, maxEvents, depth int) *privateCapture {
	return &privateCapture{scope: scope, tokens: map[string]captureKey{}, keys: map[captureKey]bool{}, events: make(chan captureEvent, depth), start: time.Now(), maxKeys: maxKeys, maxTokens: maxTokens, maxEvents: uint64(maxEvents)}
}
func (c *privateCapture) bind(token []byte, key captureKey) bool {
	if len(token) == 0 || key.Namespace != c.scope.Namespace || !strings.HasPrefix(key.TaskQueue, c.scope.TaskQueuePrefix) || !strings.HasPrefix(key.WorkflowID, c.scope.WorkflowIDPrefix) || key.RunID == "" || key.ActivityID == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stats.Closed {
		return false
	}
	if c.queue != "" && key.TaskQueue != c.queue {
		c.stats.Rejected++
		return false
	}
	if len(token) > 4096 || len(key.WorkflowID) > 1024 || len(key.RunID) > 128 || len(key.ActivityID) > 256 || len(key.TaskQueue) > 256 {
		c.stats.Rejected++
		return false
	}
	if old, exists := c.tokens[string(token)]; exists && old != key {
		c.stats.Rejected++
		return false
	}
	if !c.keys[key] && len(c.keys) >= c.maxKeys {
		c.stats.Rejected++
		return false
	}
	if _, ok := c.tokens[string(token)]; !ok && len(c.tokens) >= c.maxTokens {
		c.stats.Rejected++
		return false
	}
	c.keys[key] = true
	c.tokens[string(token)] = key
	c.stats.PeakKeys = max(c.stats.PeakKeys, len(c.keys))
	c.stats.PeakTokens = max(c.stats.PeakTokens, len(c.tokens))
	return true
}
func (c *privateCapture) lookup(token []byte) (captureKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k, ok := c.tokens[string(token)]
	return k, ok
}
func (c *privateCapture) emit(e captureEvent) {
	e.UTC = time.Now().UTC().Format(time.RFC3339Nano)
	e.ElapsedNS = time.Since(c.start).Nanoseconds()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stats.Closed {
		return
	}
	c.sequence++
	e.Sequence = c.sequence
	if c.stats.Accepted >= c.maxEvents {
		c.stats.Dropped++
		return
	}
	select {
	case c.events <- e:
		c.stats.Accepted++
	default:
		c.stats.Dropped++
	}
}
func (c *privateCapture) unmapped() {
	c.mu.Lock()
	if !c.stats.Closed {
		c.stats.Unmapped++
	}
	c.mu.Unlock()
}
func (c *privateCapture) snapshot() captureStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.stats
	s.Incomplete = s.Dropped+s.Unmapped+s.Rejected > 0
	return s
}
func (c *privateCapture) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.stats.Closed {
		if c.stopTimer != nil {
			c.stopTimer.Stop()
		}
		c.stats.Closed = true
		clear(c.tokens)
		clear(c.keys)
		close(c.events)
	}
}
func captureCode(err error) string {
	if err == nil {
		return "OK"
	}
	return serviceerror.ToStatus(err).Code().String()
}
func captureContext(ctx context.Context, e captureEvent) captureEvent {
	if ctx.Err() == nil {
		e.ContextState = "live"
	} else if ctx.Err() == context.DeadlineExceeded {
		e.ContextState = "deadline"
	} else {
		e.ContextState = "canceled"
	}
	if cause := context.Cause(ctx); cause != nil {
		e.CauseType = fmt.Sprintf("%T", cause)
		e.CauseCode = captureCode(cause)
	}
	return e
}
func keyFromInfo(i activity.Info) captureKey {
	return captureKey{i.Namespace, i.TaskQueue, i.WorkflowExecution.ID, i.WorkflowExecution.RunID, i.ActivityID, i.Attempt}
}
func (c *privateCapture) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &captureInbound{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}, c: c}
}

type captureInbound struct {
	interceptor.ActivityInboundInterceptorBase
	c *privateCapture
}

func (i *captureInbound) Init(next interceptor.ActivityOutboundInterceptor) error {
	return i.Next.Init(&captureOutbound{ActivityOutboundInterceptorBase: interceptor.ActivityOutboundInterceptorBase{Next: next}, c: i.c})
}
func (i *captureInbound) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	info := activity.GetInfo(ctx)
	key := keyFromInfo(info)
	selected := captureActivityType(info.ActivityType.Name) && i.c.bind(info.TaskToken, key)
	if selected {
		e := captureContext(ctx, captureEvent{Kind: "sdk_execute_entry", Key: key, HeartbeatTimeout: info.HeartbeatTimeout.String(), StartToCloseTimeout: info.StartToCloseTimeout.String()})
		if info.RetryPolicy != nil {
			e.MaximumAttempts = info.RetryPolicy.MaximumAttempts
		}
		i.c.emit(e)
	}
	out, err := i.Next.ExecuteActivity(ctx, in)
	if selected {
		i.c.emit(captureContext(ctx, captureEvent{Kind: "sdk_execute_return", Key: key, Code: captureCode(err), ErrorType: fmt.Sprintf("%T", err)}))
	}
	return out, err
}

type captureOutbound struct {
	interceptor.ActivityOutboundInterceptorBase
	c *privateCapture
}

func (i *captureOutbound) RecordHeartbeat(ctx context.Context, details ...any) {
	info := activity.GetInfo(ctx)
	key := keyFromInfo(info)
	selected := captureActivityType(info.ActivityType.Name) && i.c.bind(info.TaskToken, key)
	if selected {
		i.c.emit(captureContext(ctx, captureEvent{Kind: "user_heartbeat_invocation", Key: key}))
	}
	i.Next.RecordHeartbeat(ctx, details...)
}
func (c *privateCapture) received(p *ws.PollActivityTaskQueueResponse, path, identity string) {
	if p == nil || len(p.TaskToken) == 0 {
		return
	}
	// Eager responses omit the task queue; a second worker queue closes capture.
	c.mu.Lock()
	queue := c.queue
	c.mu.Unlock()
	key := captureKey{p.WorkflowNamespace, queue, p.GetWorkflowExecution().GetWorkflowId(), p.GetWorkflowExecution().GetRunId(), p.ActivityId, p.Attempt}
	if !captureActivityType(p.GetActivityType().GetName()) {
		return
	}
	if c.bind(p.TaskToken, key) {
		c.emit(captureEvent{Kind: "task_response_received", Key: key, Path: path, WorkerIdentity: identity})
	}
}
func (c *privateCapture) rpc(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	var token []byte
	var identity string
	switch q := req.(type) {
	case *ws.RecordActivityTaskHeartbeatRequest:
		token = q.TaskToken
		identity = q.Identity
	case *ws.RespondActivityTaskCompletedRequest:
		token = q.TaskToken
		identity = q.Identity
	case *ws.RespondActivityTaskFailedRequest:
		token = q.TaskToken
		identity = q.Identity
	case *ws.RespondActivityTaskCanceledRequest:
		token = q.TaskToken
		identity = q.Identity
	}
	key, selected := c.lookup(token)
	start := time.Now()
	if selected {
		c.emit(captureContext(ctx, captureEvent{Kind: "rpc_invocation", Key: key, Method: method, WorkerIdentity: identity}))
	}
	err := invoke(ctx, method, req, reply, cc, opts...)

	if selected {
		c.emit(captureContext(ctx, captureEvent{Kind: "rpc_result", Key: key, Method: method, WorkerIdentity: identity, Code: captureCode(err), ErrorType: fmt.Sprintf("%T", err), DurationUS: time.Since(start).Microseconds()}))
	} else if len(token) > 0 {
		c.unmapped()
	}
	if err == nil {
		switch q := req.(type) {
		case *ws.PollActivityTaskQueueRequest:
			if q.Namespace == c.scope.Namespace && q.GetTaskQueue().GetName() == c.captureQueue() {
				if p, ok := reply.(*ws.PollActivityTaskQueueResponse); ok {
					c.received(p, "poll", q.Identity)
				}
			}
		case *ws.RespondWorkflowTaskCompletedRequest:
			if p, ok := reply.(*ws.RespondWorkflowTaskCompletedResponse); ok {
				for _, task := range p.ActivityTasks {
					c.received(task, "eager", q.Identity)
				}
			}
		}
	} else {
		if _, ok := req.(*ws.PollActivityTaskQueueRequest); ok {
			c.unmapped()
		}
	}
	return err
}

func captureActivityType(name string) bool {
	return name == "heartbeat" || name == "retryable_error" || name == "timeout"
}
func (c *privateCapture) captureQueue() string { c.mu.Lock(); defer c.mu.Unlock(); return c.queue }
func (c *privateCapture) workerOptions(queue string, options sdkworker.Options) sdkworker.Options {
	c.mu.Lock()
	if c.queue != "" && c.queue != queue {
		c.stats.Rejected++
		c.mu.Unlock()
		c.close()
		return options
	}
	c.queue = queue
	c.mu.Unlock()
	if !strings.HasPrefix(queue, c.scope.TaskQueuePrefix) {
		return options
	}
	options.Interceptors = append(append([]interceptor.WorkerInterceptor{}, options.Interceptors...), c)
	return options
}
func (c *privateCapture) clientOptions(options sdkclient.Options, logger *zap.SugaredLogger) sdkclient.Options {
	if options.Namespace != c.scope.Namespace {
		return options
	}
	c.logger = logger
	if logger == nil {
		c.logger = zap.NewNop().Sugar()
	}
	c.done = make(chan struct{})
	c.stopTimer = time.AfterFunc(10*time.Minute, c.close)
	go c.writeEvents()
	options.ConnectionOptions.DialOptions = append(append([]grpc.DialOption{}, options.ConnectionOptions.DialOptions...), grpc.WithChainUnaryInterceptor(c.rpc))
	return options
}
func (c *privateCapture) writeEvents() {
	defer close(c.done)
	c.logger.Infow("private_activity_capture_start", "namespace", c.scope.Namespace, "taskQueuePrefix", c.scope.TaskQueuePrefix, "workflowIDPrefix", c.scope.WorkflowIDPrefix, "window", "10m", "maxEvents", c.maxEvents, "maxKeys", c.maxKeys, "maxTokens", c.maxTokens)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case e, ok := <-c.events:
			if !ok {
				c.logger.Infow("private_activity_capture_end", "stats", c.snapshot())
				return
			}
			c.logger.Infow("private_activity_capture_event", "event", e)
		case <-tick.C:
			stats := c.snapshot()
			if stats.Closed {
				tick.Stop()
				continue
			}
			c.logger.Infow("private_activity_capture_status", "stats", stats)
		}
	}
}
func (c *privateCapture) finish() {
	c.close()
	if c.done != nil {
		select {
		case <-c.done:
		case <-time.After(250 * time.Millisecond):
		}
	}
}
func privateCaptureApp(app harness.App, selection privateCaptureCase) (harness.App, func()) {
	if selection.Namespace == "" || selection.TaskQueuePrefix == "" || selection.WorkflowIDPrefix == "" {
		return app, func() {}
	}
	c := newPrivateCapture(selection, 8, 16, 512, 64)
	app.ClientFactory = func(config harness.ClientConfig) (sdkclient.Client, error) { return buildClientWithCapture(config, c) }
	original := app.Worker
	app.Worker = func(client sdkclient.Client, context harness.WorkerContext) sdkworker.Worker {
		context.WorkerOptions = c.workerOptions(context.TaskQueue, context.WorkerOptions)
		return original(client, context)
	}
	return app, c.finish
}
