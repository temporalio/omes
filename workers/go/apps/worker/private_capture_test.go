package worker

import (
	"context"
	"crypto/tls"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/workers/go/harness"
	common "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	ws "go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestPrivateCaptureBounds(t *testing.T) {
	scope := privateCaptureCase{Namespace: "ns", TaskQueuePrefix: "tq", WorkflowIDPrefix: "wf"}
	key := captureKey{Namespace: "ns", TaskQueue: "tq", WorkflowID: "wf", RunID: "run"}
	key.ActivityID = "a"
	key.Attempt = 1
	t.Run("EventCapAndOverflowAreIncomplete", func(t *testing.T) {
		c := newPrivateCapture(scope, 2, 2, 2, 4)
		for range 3 {
			c.emit(captureEvent{Kind: "bounded", Key: key})
		}
		s := c.snapshot()
		if s.Accepted != 2 || s.Dropped != 1 || !s.Incomplete {
			t.Fatalf("cap marker: %+v", s)
		}
		c.close()
		q := newPrivateCapture(scope, 2, 2, 8, 1)
		q.emit(captureEvent{Kind: "queued", Key: key})
		q.emit(captureEvent{Kind: "overflow", Key: key})
		s = q.snapshot()
		if s.Accepted != 1 || s.Dropped != 1 || !s.Incomplete {
			t.Fatalf("queue marker: %+v", s)
		}
		q.close()
	})
	t.Run("IdentityStateBoundAndConcurrentClose", func(t *testing.T) {
		c := newPrivateCapture(scope, 1, 1, 8, 8)
		if !c.bind([]byte{1}, key) {
			t.Fatal("first binding")
		}
		other := key
		other.Attempt = 2
		if c.bind([]byte{2}, other) {
			t.Fatal("bound exceeded")
		}
		if c.snapshot().PeakTokens != 1 || !c.snapshot().Incomplete {
			t.Fatal("state marker")
		}
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 50 {
					c.emit(captureEvent{Kind: "concurrent", Key: key})
				}
			}()
		}
		c.close()
		wg.Wait()
		c.mu.Lock()
		n := len(c.tokens) + len(c.keys)
		c.mu.Unlock()
		if n != 0 || !c.snapshot().Closed {
			t.Fatal("state not purged")
		}
	})
	t.Run("InvokerContextReplyErrorPreserved", func(t *testing.T) {
		c := newPrivateCapture(scope, 2, 2, 16, 16)
		defer c.close()
		token := []byte{3}
		c.bind(token, key)
		req := &ws.RecordActivityTaskHeartbeatRequest{TaskToken: token}
		before := proto.Clone(req)
		reply := &ws.RecordActivityTaskHeartbeatResponse{}
		want := serviceerror.NewDeadlineExceeded("unit sentinel")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		got := c.rpc(ctx, "/temporal.api.workflowservice.v1.WorkflowService/RecordActivityTaskHeartbeat", req, reply, nil, func(in context.Context, _ string, r, o any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			calls++
			if in != ctx || r != req || o != reply {
				t.Fatal("invoker arguments changed")
			}
			if !c.mu.TryLock() {
				t.Fatal("lock across invoker")
			}
			c.mu.Unlock()
			reply.CancelRequested = true
			return want
		})
		if got != want || calls != 1 || ctx.Err() != nil || !reply.CancelRequested || !proto.Equal(req, before) {
			t.Fatal("observer mutated protocol/context")
		}
	})
}

func TestPrivateCaptureDisabledAndOptions(t *testing.T) {
	app, finish := privateCaptureApp(harness.App{Worker: buildWorker, ClientFactory: buildClient}, privateCaptureCase{})
	finish()
	if reflect.ValueOf(app.Worker).Pointer() != reflect.ValueOf(buildWorker).Pointer() || reflect.ValueOf(app.ClientFactory).Pointer() != reflect.ValueOf(buildClient).Pointer() {
		t.Fatal("disabled app altered")
	}
	c := newPrivateCapture(privateCaptureCase{"ns", "tq", "owned-"}, 8, 16, 512, 64)
	options := harness.BuildSDKClientOptions(harness.ClientConfig{Namespace: "ns", TLS: &tls.Config{ServerName: "ordinary.invalid"}, APIKey: "unit-only-credential"})
	options.DataConverter = clioptions.OmesDataConverter()
	changed := c.clientOptions(options, zap.NewNop().Sugar())
	defer c.finish()
	if len(changed.ConnectionOptions.DialOptions) != len(options.ConnectionOptions.DialOptions)+1 {
		t.Fatal("missing transport hook")
	}
	changed.ConnectionOptions.DialOptions = options.ConnectionOptions.DialOptions
	if reflect.ValueOf(changed.Credentials).Pointer() != reflect.ValueOf(options.Credentials).Pointer() {
		t.Fatal("credentials callback replaced")
	}
	a := reflect.ValueOf(changed.Credentials).Call([]reflect.Value{reflect.ValueOf(context.Background())})
	b := reflect.ValueOf(options.Credentials).Call([]reflect.Value{reflect.ValueOf(context.Background())})
	if a[0].String() != b[0].String() || !a[1].IsNil() || !b[1].IsNil() {
		t.Fatal("credentials behavior changed")
	}
	changed.Credentials = nil
	options.Credentials = nil
	if !reflect.DeepEqual(changed, options) {
		t.Fatal("client option changed")
	}
	wo := sdkworker.Options{MaxConcurrentActivityExecutionSize: 8, MaxHeartbeatThrottleInterval: time.Second, BuildID: "ordinary-build", UseBuildIDForVersioning: true}
	modified := c.workerOptions("tq-1", wo)
	if len(modified.Interceptors) != len(wo.Interceptors)+1 {
		t.Fatal("missing SDK hook")
	}
	modified.Interceptors = wo.Interceptors
	if !reflect.DeepEqual(modified, wo) {
		t.Fatal("worker option changed")
	}
	if !reflect.DeepEqual(c.workerOptions("unrelated", wo), wo) {
		t.Fatal("unrelated task queue modified")
	}
	c.workerOptions("tq-2", wo)
	if !c.snapshot().Incomplete || !c.snapshot().Closed {
		t.Fatal("multiple queues silently misattributed")
	}
}
func TestPrivateCaptureIncrementalDynamicTuples(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	notify := make(chan struct{}, 16)
	logger := zap.New(core, zap.Hooks(func(zapcore.Entry) error {
		select {
		case notify <- struct{}{}:
		default:
		}
		return nil
	})).Sugar()
	c := newPrivateCapture(privateCaptureCase{"ns", "tq", "owned-"}, 2, 2, 3, 8)
	c.clientOptions(sdkclient.Options{Namespace: "ns"}, logger)
	c.workerOptions("tq-1", sdkworker.Options{})
	for _, run := range []string{"run1", "run2"} {
		c.received(&ws.PollActivityTaskQueueResponse{WorkflowNamespace: "ns", WorkflowExecution: &common.WorkflowExecution{WorkflowId: "owned-wf", RunId: run}, ActivityId: "healthy", ActivityType: &common.ActivityType{Name: "heartbeat"}, Attempt: 1, TaskToken: []byte("unlogged-token-" + run), Input: &common.Payloads{Payloads: []*common.Payload{{Data: []byte("unlogged-payload")}}}}, "eager", "worker")
	}
	deadline := time.After(time.Second)
	for observed.FilterMessage("private_activity_capture_event").Len() < 2 {
		select {
		case <-notify:
		case <-deadline:
			t.Fatal("no incremental output before close")
		}
	}
	if c.snapshot().Closed {
		t.Fatal("capture closed before incremental output")
	}
	key := captureKey{"ns", "tq-1", "owned-wf", "run1", "healthy", 1}
	if c.bind([]byte{3}, captureKey{"ns", "tq-1", "foreign", "run", "healthy", 1}) {
		t.Fatal("foreign admission")
	}
	c.emit(captureEvent{Kind: "within-cap", Key: key})
	c.emit(captureEvent{Kind: "over-cap", Key: key})
	c.unmapped()
	c.finish()
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("writer retained")
	}
	ends := observed.FilterMessage("private_activity_capture_end").All()
	if len(ends) != 1 || !c.snapshot().Incomplete || c.snapshot().Dropped != 1 || c.snapshot().Unmapped != 1 || c.snapshot().PeakKeys != 2 {
		t.Fatalf("missing end/gap marker: %+v", c.snapshot())
	}
	entries := observed.FilterMessage("private_activity_capture_event").All()
	runs := map[string]bool{}
	for _, e := range entries {
		encoded, err := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()).EncodeEntry(e.Entry, e.Context)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(encoded.String(), "unlogged-") {
			t.Fatal("payload or token emitted")
		}
		encoded.Free()
		event := e.ContextMap()["event"].(captureEvent)
		if event.Key.WorkflowID != "owned-wf" {
			t.Fatal("wrong identity")
		}
		runs[event.Key.RunID] = true
	}
	if !runs["run1"] || !runs["run2"] {
		t.Fatal("run identity collapsed")
	}
}
func TestPrivateCaptureSlowLogDoesNotBlockHooks(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	core, _ := observer.New(zapcore.InfoLevel)
	logger := zap.New(core, zap.Hooks(func(e zapcore.Entry) error {
		if e.Message == "private_activity_capture_event" {
			select {
			case <-entered:
			default:
				close(entered)
			}
			<-release
		}
		return nil
	})).Sugar()
	c := newPrivateCapture(privateCaptureCase{"ns", "tq", "owned-"}, 1, 1, 4, 1)
	c.clientOptions(sdkclient.Options{Namespace: "ns"}, logger)
	c.emit(captureEvent{Kind: "first"})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("writer not entered")
	}
	emitted := make(chan struct{})
	go func() {
		for range 1000 {
			c.emit(captureEvent{Kind: "bounded"})
		}
		close(emitted)
	}()
	select {
	case <-emitted:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("hook blocked on logger")
	}
	c.finish()
	if !c.snapshot().Incomplete {
		t.Fatal("overflow unmarked")
	}
	close(release)
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("writer not cleaned up after logger released")
	}
}

type captureFactoryWorker struct{ sdkworker.Worker }

func TestPrivateCaptureFactoryAndFrozenOwnership(t *testing.T) {
	want := &captureFactoryWorker{}
	base := sdkworker.Options{MaxConcurrentActivityExecutionSize: 8, MaxHeartbeatThrottleInterval: time.Second}
	calls := 0
	app, finish := privateCaptureApp(harness.App{Worker: func(client sdkclient.Client, in harness.WorkerContext) sdkworker.Worker {
		calls++
		if client != nil || in.TaskQueue != "tq-1" || len(in.WorkerOptions.Interceptors) != 1 {
			t.Fatal("factory wiring changed")
		}
		in.WorkerOptions.Interceptors = nil
		if !reflect.DeepEqual(in.WorkerOptions, base) {
			t.Fatal("factory options changed")
		}
		return want
	}}, privateCaptureCase{"ns", "tq", "owned-"})
	defer finish()
	if got := app.Worker(nil, harness.WorkerContext{TaskQueue: "tq-1", WorkerOptions: base}); got != want || calls != 1 || len(base.Interceptors) != 0 {
		t.Fatal("factory effect changed")
	}
	c := newPrivateCapture(privateCaptureCase{"ns", "tq", "owned-"}, 2, 2, 8, 8)
	defer c.close()
	key := captureKey{"ns", "tq-1", "owned-wf", "run1", "a", 1}
	if !c.bind([]byte{1}, key) {
		t.Fatal("initial identity")
	}
	changed := key
	changed.RunID = "run2"
	if c.bind([]byte{1}, changed) {
		t.Fatal("opaque token rebound to foreign run")
	}
	if got, _ := c.lookup([]byte{1}); got != key || !c.snapshot().Incomplete {
		t.Fatal("frozen binding not preserved")
	}
}
