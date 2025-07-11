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

		// Chunk the file. `chunks` here still contains the data, which is fine for announcement.
		chunks, fileHash, err := file.ChunkFile(filePath)
		if err != nil {
			log.Fatalf("Failed to chunk file: %v", err)
		}
		log.Printf("File '%s' chunked successfully. File Hash: %s", filePath, fileHash)

		client := p2p.NewTrackerClient(cfg.TrackerURL, cfg.AuthToken)
		// Announce each chunk to the tracker
		for i := range chunks {
			if err := client.Announce(filePath, fileHash, len(chunks), i, chunks[i].Hash); err != nil {
				log.Printf("Failed to announce chunk %d: %v", i, err)
			}
		}
		log.Printf("Announced all %d chunks to tracker.", len(chunks))

		grpcPort, _ := cmd.Flags().GetString("port")

		// IMPORTANT: The gRPC server is now initialized with the file path and chunk metadata,
		// but it will read the actual chunk data from disk on demand.
		// Note: For a multi-file sharing peer, this `NewGRPCServer` call would need to
		// manage multiple shared files. For this demo, it assumes one file at a time.
		grpcServer := p2p.NewGRPCServer(filePath, fileHash, len(chunks), chunks)
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
