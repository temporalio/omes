package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "dev",
		Short: "Development tools for Omes",
	}

	rootCmd.AddCommand(buildProtoCmd())
	rootCmd.AddCommand(buildWorkerImageCmd())
	rootCmd.AddCommand(buildPushWorkerImageCmd())
	rootCmd.AddCommand(buildCliImageCmd())
	rootCmd.AddCommand(buildPushCliImageCmd())
	rootCmd.AddCommand(testCmd())
	rootCmd.AddCommand(cleanCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(checkCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
