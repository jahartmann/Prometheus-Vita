package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagAPIURL string
	flagToken  string
	flagOutput string
	apiClient  *APIClient
)

func getClient() *APIClient {
	if apiClient == nil {
		apiClient = NewAPIClient(flagAPIURL, flagToken)
	}
	return apiClient
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "prometheus",
		Short: "Prometheus CLI - Proxmox Cluster Management",
		Long:  "Kommandozeilen-Tool zur Verwaltung von Proxmox-Clustern ueber die Prometheus API.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig()
			if err != nil {
				return
			}
			if flagAPIURL == "" {
				flagAPIURL = cfg.APIURL
			}
			if flagToken == "" {
				flagToken = cfg.Token
			}
			if flagAPIURL == "" {
				flagAPIURL = "http://localhost:8080"
			}
			// Reset client so it picks up new flags
			apiClient = nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "API-Server URL (Standard: aus config)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "API-Token (Standard: aus config)")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "table", "Ausgabeformat: table oder json")

	rootCmd.AddCommand(nodesCmd())
	rootCmd.AddCommand(vmsCmd())
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(driftCmd())
	rootCmd.AddCommand(updatesCmd())
	rootCmd.AddCommand(recommendationsCmd())
	rootCmd.AddCommand(sshKeysCmd())
	rootCmd.AddCommand(newConfigCmd())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Version anzeigen",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("prometheus-cli v0.1.0")
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
