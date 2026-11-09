package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

type statusServer struct {
	mux    *http.ServeMux
	server *http.Server
}

func (a *statusServer) start(address string, logger *zap.SugaredLogger) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return loadgen.NewUsageError("listening on --status-listen-address %q: %v", address, err)
	}
	a.server = &http.Server{Addr: listener.Addr().String(), Handler: a.mux}
	go func() {
		if err := a.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("Serving status: %v", err)
		}
	}()
	logger.Infof("Status server listening on %s", listener.Addr())
	return nil
}

func (a *statusServer) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

func (r *scenarioRunner) startStatusServer() error {
	if r.exportResultFile != "" || r.statusListenAddress != "" {
		r.counter = &loadgen.RunCounter{}
	}
	if r.statusListenAddress == "" {
		return nil
	}
	server := &statusServer{mux: http.NewServeMux()}
	server.mux.Handle("GET /status", r.statusHandler())
	if err := server.start(r.statusListenAddress, r.logger); err != nil {
		return err
	}
	r.status = server
	return nil
}

func (r *scenarioRunner) shutdownStatusServer() {
	if r.status != nil {
		if err := r.status.shutdown(); err != nil {
			r.logger.Errorf("Shutting down status server: %v", err)
		}
	}
}
