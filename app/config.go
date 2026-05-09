package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	MobileServerEnabled bool `json:"mobileServerEnabled"`
}

var (
	config Config
	configMu sync.Mutex
)

func loadConfig() {
	configMu.Lock()
	defer configMu.Unlock()

	dir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	path := filepath.Join(dir, ".gapi-config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		config = Config{MobileServerEnabled: false}
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	configMu.Lock()
	data, _ := json.Marshal(config)
	configMu.Unlock()

	dir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(dir, ".gapi-config.json"), data, 0644)
}

func IsMobileServerEnabled() bool {
	configMu.Lock()
	defer configMu.Unlock()
	return config.MobileServerEnabled
}

func SetMobileServerEnabled(enabled bool) {
	configMu.Lock()
	config.MobileServerEnabled = enabled
	configMu.Unlock()
	saveConfig()
}
