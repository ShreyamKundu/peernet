package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// configPath is a global variable to store the path to the config file,
// set by a persistent flag.
var configPath string

var rootCmd = &cobra.Command{
	Use:   "peernet",
	Short: "PeerNet is a P2P file sharing client",
	Long: `A command-line client for the PeerNet network.

Before you begin, register with a tracker:
  peernet register --tracker http://your-tracker.com --address your-ip:50051 --password your-pass

To manage multiple local peers, use the --config flag:
  peernet register --config ~/.peernet/peer1_config.yaml ...
  peernet share --config ~/.peernet/peer1_config.yaml ...
  peernet download --config ~/.peernet/peer2_config.yaml ...
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add a persistent flag for the config file path
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to the peer configuration file (default: ~/.peernet/config.yaml)")
}
