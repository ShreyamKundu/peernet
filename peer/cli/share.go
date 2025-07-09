package cli

import (
	"log"
	"os"

	"github.com/ShreyamKundu/peernet/peer/config"
	"github.com/ShreyamKundu/peernet/peer/file"
	"github.com/ShreyamKundu/peernet/peer/p2p"
	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share [file-path]",
	Short: "Share a file on the network",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil || cfg.AuthToken == "" {
			log.Fatal("Configuration not found. Please run 'peernet register' first.")
		}

		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("File does not exist: %s", filePath)
		}

		chunks, fileHash, err := file.ChunkFile(filePath)
		if err != nil {
			log.Fatalf("Failed to chunk file: %v", err)
		}
		log.Printf("File '%s' chunked successfully. File Hash: %s", filePath, fileHash)

		client := p2p.NewTrackerClient(cfg.TrackerURL, cfg.AuthToken)
		for i := range chunks {
			if err := client.Announce(filePath, fileHash, len(chunks), i); err != nil {
				log.Printf("Failed to announce chunk %d: %v", i, err)
			}
		}
		log.Printf("Announced all %d chunks to tracker.", len(chunks))

		grpcPort, _ := cmd.Flags().GetString("port")
		grpcServer := p2p.NewGRPCServer(filePath, chunks)
		log.Printf("Starting gRPC server to serve file chunks on port %s...", grpcPort)
		if err := grpcServer.Start(grpcPort); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	},
}

func init() {
	shareCmd.Flags().StringP("port", "p", "50051", "Port for this peer to listen for requests")
	rootCmd.AddCommand(shareCmd)
}
