package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CLIConfig struct {
	APIURL string `json:"api_url"`
	Token  string `json:"token"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prometheus", "config.json")
}

func loadConfig() (*CLIConfig, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CLIConfig{
				APIURL: "http://localhost:8080",
			}, nil
		}
		return nil, fmt.Errorf("config lesen: %w", err)
	}

	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parsen: %w", err)
	}
	return &cfg, nil
}

func saveConfig(cfg *CLIConfig) error {
	path := configPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("config-verzeichnis erstellen: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config serialisieren: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
