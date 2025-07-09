package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
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
			log.Fatalf("Registration failed with status: %s", resp.Status)
		}

		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		token := result["token"]

		cfg := &config.Config{
			TrackerURL: trackerURL,
			AuthToken:  token,
		}
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}

		fmt.Printf("✅ Successfully registered! Configuration saved.\n")
	},
}

func init() {
	registerCmd.Flags().String("tracker", "", "URL of the tracker server")
	registerCmd.Flags().String("address", "", "This peer's public IP and port (e.g., 123.45.67.89:50051)")
	registerCmd.Flags().String("password", "", "A password for your peer account")
	rootCmd.AddCommand(registerCmd)
}
