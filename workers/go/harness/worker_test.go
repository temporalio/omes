package harness

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/temporalio/omes/clioptions"
	sdkclient "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
)

type fakeWorker struct {
	sdkworker.Worker

	run func(<-chan any) error
}

func (f fakeWorker) Run(stopCh <-chan any) error {
	if f.run == nil {
		return nil
	}
	return f.run(stopCh)
}

func TestRunWorkerCLIPassesSharedClientAndWorkerContext(t *testing.T) {
	sharedClient := clientStub{}
	var gotClients []sdkclient.Client
	var gotContexts []WorkerContext
	clientFactoryCalls := 0

	err := runWorkerCLI(
		func(client sdkclient.Client, context WorkerContext) sdkworker.Worker {
			gotClients = append(gotClients, client)
			gotContexts = append(gotContexts, context)
			return fakeWorker{}
		},
		func(ClientConfig) (sdkclient.Client, error) {
			clientFactoryCalls++
			return sharedClient, nil
		},
		[]string{
			"--task-queue", "omes",
			"--task-queue-suffix-index-start", "1",
			"--task-queue-suffix-index-end", "2",
			"--err-on-unimplemented=true",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if clientFactoryCalls != 1 {
		t.Fatalf("expected clientFactory once, got %d", clientFactoryCalls)
	}
	if len(gotClients) != 2 || gotClients[0] != sharedClient || gotClients[1] != sharedClient {
		t.Fatalf("expected both workers to receive shared client, got %#v", gotClients)
	}
	if len(gotContexts) != 2 {
		t.Fatalf("expected 2 worker contexts, got %d", len(gotContexts))
	}
	if gotContexts[0].TaskQueue != "omes-1" || gotContexts[1].TaskQueue != "omes-2" {
		t.Fatalf("unexpected task queues: %q, %q", gotContexts[0].TaskQueue, gotContexts[1].TaskQueue)
	}
	for _, context := range gotContexts {
		if !context.ErrOnUnimplemented {
			t.Fatal("expected err-on-unimplemented in worker context")
		}
	}
}

func TestRunWorkersStopsRemainingWorkersOnFailure(t *testing.T) {
	stopped := make(chan struct{}, 1)

	err := runWorkers([]sdkworker.Worker{
		fakeWorker{run: func(<-chan any) error {
			return context.Canceled
		}},
		fakeWorker{run: func(stopCh <-chan any) error {
			select {
			case <-stopCh:
				stopped <- struct{}{}
				return nil
			case <-time.After(time.Second):
				return context.DeadlineExceeded
			}
		}},
	}, make(chan struct{}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second * 2):
		t.Fatal("expected remaining worker to observe shutdown")
	}
}

func TestBuildWorkerOptionsPollerBehavior(t *testing.T) {
	for _, tc := range []struct {
		name             string
		argv             []string
		activityBehavior sdkworker.PollerBehavior
		workflowBehavior sdkworker.PollerBehavior
	}{
		{
			name: "no poller flags leaves both unset",
			argv: nil,
		},
		{
			// The autoscaling max must not also pin the initial poller count;
			// unset fields fall back to the SDK's own defaults.
			name: "activity autoscale max sets only the maximum",
			argv: []string{"--activity-poller-autoscale-max=20"},
			activityBehavior: sdkworker.NewPollerBehaviorAutoscaling(sdkworker.PollerBehaviorAutoscalingOptions{
				MaximumNumberOfPollers: 20,
			}),
		},
		{
			name: "workflow autoscale max sets only the maximum",
			argv: []string{"--workflow-poller-autoscale-max=10"},
			workflowBehavior: sdkworker.NewPollerBehaviorAutoscaling(sdkworker.PollerBehaviorAutoscalingOptions{
				MaximumNumberOfPollers: 10,
			}),
		},
		{
			name: "max concurrent pollers still pins a simple maximum",
			argv: []string{"--max-concurrent-activity-pollers=3"},
			activityBehavior: sdkworker.NewPollerBehaviorSimpleMaximum(sdkworker.PollerBehaviorSimpleMaximumOptions{
				MaximumNumberOfPollers: 3,
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var args clioptions.WorkerOptions
			flags := args.FlagSetWithPrefix("")
			if err := flags.Parse(tc.argv); err != nil {
				t.Fatalf("failed to parse %v: %v", tc.argv, err)
			}
			options, err := buildWorkerOptions(flags, args, "")
			if err != nil {
				t.Fatalf("buildWorkerOptions failed: %v", err)
			}
			if !reflect.DeepEqual(options.ActivityTaskPollerBehavior, tc.activityBehavior) {
				t.Errorf("activity poller behavior = %#v, want %#v",
					options.ActivityTaskPollerBehavior, tc.activityBehavior)
			}
			if !reflect.DeepEqual(options.WorkflowTaskPollerBehavior, tc.workflowBehavior) {
				t.Errorf("workflow poller behavior = %#v, want %#v",
					options.WorkflowTaskPollerBehavior, tc.workflowBehavior)
			}
		})
	}
}
