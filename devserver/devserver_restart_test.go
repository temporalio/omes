package devserver

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeProcesses struct {
	mu         sync.Mutex
	launches   int
	active     int
	maxActive  int
	failLaunch int
	output     *bytes.Buffer
}

func (f *fakeProcesses) launch(context.Context) (context.CancelFunc, chan error, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.launches++
	if f.launches == f.failLaunch {
		return nil, nil, errors.New("launch failed")
	}
	f.active++
	f.maxActive = max(f.maxActive, f.active)
	_, _ = f.output.WriteString("started\n")
	done := make(chan error, 1)
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			f.active--
			f.mu.Unlock()
			done <- context.Canceled
		})
	}, done, nil
}

func TestServerStopIgnoresCanceledCommandContext(t *testing.T) {
	workDir := t.TempDir()
	server := &Server{
		cancel:  func() {},
		done:    make(chan error, 1),
		workDir: workDir,
	}
	server.done <- context.Canceled

	require.NoError(t, server.Stop())
}

func newFakeServer(t *testing.T, processes *fakeProcesses) *Server {
	t.Helper()
	server := &Server{
		frontend: "127.0.0.1:7233",
		ports:    newPorts("127.0.0.1", [portCount]int{7233, 7234, 7243, 7235, 7236, 7237, 7238, 7239, 7240}),
		workDir:  t.TempDir(),
		launch:   processes.launch,
		ready:    func(context.Context) error { return nil },
	}
	require.NoError(t, server.startProcessLocked(t.Context()))
	return server
}

func TestServerRestartPreservesPortsAndOutput(t *testing.T) {
	var output bytes.Buffer
	processes := &fakeProcesses{output: &output}
	server := newFakeServer(t, processes)
	wantPorts := server.Ports()

	require.NoError(t, server.Restart(t.Context()))
	require.Equal(t, wantPorts, server.Ports())
	require.Equal(t, "started\nstarted\n", output.String())
	require.NoError(t, server.Stop())
}

func TestServerRepeatedRestarts(t *testing.T) {
	processes := &fakeProcesses{output: &bytes.Buffer{}}
	server := newFakeServer(t, processes)

	for range 5 {
		require.NoError(t, server.Restart(t.Context()))
	}
	require.NoError(t, server.Stop())
	require.Equal(t, 6, processes.launches)
	require.Zero(t, processes.active)
}

func TestServerRestartFailure(t *testing.T) {
	processes := &fakeProcesses{output: &bytes.Buffer{}, failLaunch: 2}
	server := newFakeServer(t, processes)

	err := server.Restart(t.Context())
	require.ErrorContains(t, err, "launch failed")
	require.Zero(t, processes.active)
	require.NoError(t, server.Stop())
}

func TestServerLifecycleCallsAreSerialized(t *testing.T) {
	processes := &fakeProcesses{output: &bytes.Buffer{}}
	server := newFakeServer(t, processes)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = server.Restart(t.Context())
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = server.Stop()
	}()
	wg.Wait()

	require.Equal(t, 1, processes.maxActive)
	require.Zero(t, processes.active)
}

func TestServerRestartCancellation(t *testing.T) {
	processes := &fakeProcesses{output: &bytes.Buffer{}}
	server := newFakeServer(t, processes)
	server.ready = func(ctx context.Context) error {
		<-ctx.Done()
		return context.Cause(ctx)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := server.Restart(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, processes.active)
	require.NoError(t, server.Stop())
}

func TestServerRejectsRestartAfterStop(t *testing.T) {
	processes := &fakeProcesses{output: &bytes.Buffer{}}
	server := newFakeServer(t, processes)

	require.NoError(t, server.Stop())
	require.NoError(t, server.Stop())
	require.ErrorContains(t, server.Restart(t.Context()), "stopped server")
	require.Equal(t, 1, processes.launches)
}
