package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ShreyamKundu/peernet/peer/config"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register this peer with a tracker and save credentials",
	Run: func(cmd *cobra.Command, args []string) {
		trackerURL, _ := cmd.Flags().GetString("tracker")
		address, _ := cmd.Flags().GetString("address")
		password, _ := cmd.Flags().GetString("password")

		if trackerURL == "" || address == "" || password == "" {
			log.Fatal("Must provide --tracker, --address, and --password")
		}

		payload := map[string]string{"address": address, "password": password}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(trackerURL+"/api/v1/peers/register", "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Fatalf("Failed to register with tracker: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			responseBody, _ := io.ReadAll(resp.Body)
			log.Fatalf("Registration failed with status: %s, body: %s", resp.Status, string(responseBody))
		}

		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		token := result["token"]

		cfg := &config.Config{
			TrackerURL: trackerURL,
			AuthToken:  token,
		}
		// Pass the global configPath variable to Save
		if err := cfg.Save(configPath); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}

		fmt.Printf("✅ Successfully registered! Configuration saved to %s.\n", configPathOrDefault(configPath))
	},
}

func init() {
	registerCmd.Flags().String("tracker", "", "URL of the tracker server")
	registerCmd.Flags().String("address", "", "This peer's public IP and port (e.g., 123.45.67.89:50051)")
	registerCmd.Flags().String("password", "", "A password for your peer account")
	rootCmd.AddCommand(registerCmd)
}

// Helper to display the correct config path in messages
func configPathOrDefault(path string) string {
	if path != "" {
		return path
	}
	defaultPath, _ := config.DefaultConfigFilePath() // Assuming this function exists and works
	return defaultPath
}
