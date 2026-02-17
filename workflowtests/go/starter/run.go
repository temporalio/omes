package starter

import (
	"fmt"
	"os"
)

// Run is the entry point for workflow tests - mirrors omes_starter.run() in Python
// and run({client, worker}) in TypeScript.
//
// Usage: starter.Run(clientMain, workerMain)
// CLI: ./program client --task-queue ... OR ./program worker --task-queue ...
func Run(client ExecuteFunc, worker ConfigureWorkerFunc) {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: <program> [client|worker] ...")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...) // Remove subcommand for flag parsing

	switch cmd {
	case "client":
		RunClient(client)
	case "worker":
		RunWorker(worker)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
