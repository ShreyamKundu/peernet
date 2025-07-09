package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "peernet",
	Short: "PeerNet is a P2P file sharing client",
	Long: `A command-line client for the PeerNet network.

Before you begin, register with a tracker:
  peernet register --tracker http://your-tracker.com --address your-ip:50051 --password your-pass
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
