package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ShreyamKundu/peernet/peer/config"
	"github.com/ShreyamKundu/peernet/peer/p2p"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download [file-hash]",
	Short: "Download a file from the network",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Pass the global configPath variable to Load
		cfg, err := config.Load(configPath)
		if err != nil || cfg.AuthToken == "" {
			log.Fatal("Configuration not found. Please run 'peernet register' first or specify --config.")
		}

		fileHash := args[0]
		outputDir, _ := cmd.Flags().GetString("output")
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		trackerClient := p2p.NewTrackerClient(cfg.TrackerURL, cfg.AuthToken)
		lookupResult, err := trackerClient.Lookup(fileHash)
		if err != nil {
			log.Fatalf("Failed to lookup file: %v", err)
		}
		if len(lookupResult.Chunks) == 0 {
			log.Fatalf("No peers found for file hash: %s", fileHash)
		}

		downloader := p2p.NewDownloader(trackerClient)
		outputPath := filepath.Join(outputDir, fileHash+".download")

		if err := downloader.DownloadFile(fileHash, lookupResult, outputPath); err != nil {
			log.Fatalf("Failed to download file: %v", err)
		}

		log.Printf("✅ File successfully downloaded to %s", outputPath)
	},
}

func init() {
	downloadCmd.Flags().StringP("output", "o", "./downloads", "Directory to save downloaded files")
	rootCmd.AddCommand(downloadCmd)
}
