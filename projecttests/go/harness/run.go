package harness

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/temporalio/omes/projecttests/go/harness/api"
	"google.golang.org/grpc"
)

const defaultHost = "0.0.0.0"

func New() *Harness {
	return &Harness{}
}

func (h *Harness) RegisterWorker(workerFunc WorkerFunc) {
	if h.workerFunc != nil {
		log.Fatalf("worker already registered")
	}
	h.workerFunc = workerFunc
}

func (h *Harness) OnInit(fn InitFunc) {
	if h.initFunc != nil {
		log.Fatalf("init handler already registered")
	}
	h.initFunc = fn
}

func (h *Harness) OnExecute(fn ExecuteFunc) {
	if h.executeFunc != nil {
		log.Fatalf("execute handler already registered")
	}
	h.executeFunc = fn
}

func (h *Harness) Run() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: <program> [project-server|worker] ...")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...) // Remove subcommand for flag parsing

	switch cmd {
	case "project-server":
		h.startGrpcServer()
	case "worker":
		h.StartWorker()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func (h *Harness) startGrpcServer() {
	if h.executeFunc == nil {
		log.Fatalf("Attempted to start server, but no execute handler was registered")
	}

	fs := flag.NewFlagSet("client", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP port")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", defaultHost, *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	api.RegisterProjectServiceServer(grpcServer, h)
	if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Fatalf("gRPC server failed: %v", err)
	}
}
